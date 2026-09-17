# Deploying minIDM to Digital Ocean Kubernetes (DOKS)

## Overview

The deployment uses:
- An **in-cluster container registry** (from [doks-infra](https://github.com/jjones028/doks-infra)) to store the app and migrator images
- A **DOKS cluster** running the app
- A **DigitalOcean Managed PostgreSQL** cluster (`apps-postgres`), shared with recipe-keeper as a
  separate database — not an in-cluster operator. (This originally ran on an in-cluster
  CloudNativePG `Cluster`, which has since been retired; see "Create the Database" below.)
- **Traefik** as the ingress controller with automatic Let's Encrypt TLS
- **GitHub Actions** to build, push, and roll out on every push to `main`. Needs a valid
  `DIGITALOCEAN_ACCESS_TOKEN` repo secret (Kubernetes: Read scope) — this can silently expire
  with no obvious symptom besides deploys quietly not landing; check
  `gh run list --workflow=deploy.yml` if a push doesn't seem to have taken effect.

---

## 1. Prerequisites

Install and authenticate the DO CLI:

```bash
brew install doctl helm kubectl
doctl auth init   # paste your DO personal access token
```

---

## 2. Bootstrap the Cluster and Registry

Before deploying minIDM, complete the cluster setup in the [doks-infra](https://github.com/jjones028/doks-infra) repo. That covers DOKS cluster provisioning, Traefik, and the in-cluster container registry (`task bootstrap` + `task registry`). Come back here once everything is running and your DNS A records for `auth.yourdomain.com` and `registry.yourdomain.com` point at the Traefik load balancer IP.

---

## 3. Create the Registry Pull Secret

The in-cluster registry uses htpasswd authentication. Create a pull secret so Kubernetes can pull images:

```bash
kubectl create secret docker-registry registry-credentials \
  --docker-server=registry.yourdomain.com \
  --docker-username=<htpasswd-username> \
  --docker-password=<htpasswd-password>
```

---

## 4. Create the Database

minIDM uses a DigitalOcean Managed PostgreSQL cluster rather than an in-cluster database
operator. If recipe-keeper is already deployed here, its `apps-postgres` cluster already exists
— just add a database and user to it:

```bash
CLUSTER_ID=$(doctl databases list --format ID,Name --no-header | awk '/apps-postgres/{print $1}')
doctl databases db create "$CLUSTER_ID" minidm
doctl databases user create "$CLUSTER_ID" minidm_app
```

`user create` prints a generated password — note it. Then set that user as owner of the database
(the default owner is the cluster's admin user, more privilege than the app needs), using `psql`
from any pod that can reach the cluster:

```bash
psql "$ADMIN_URI" -c "ALTER DATABASE minidm OWNER TO minidm_app;"
```

(`$ADMIN_URI` is the cluster's admin connection string — `doctl databases connection <cluster-id>`
prints the host/port; get the admin password from the DO control panel or the cluster's original
`doctl databases create` output.)

If recipe-keeper is *not* already deployed here, create the cluster first:

```bash
doctl databases create apps-postgres --engine pg --version 18 --region nyc3 \
  --size db-s-1vcpu-1gb --num-nodes 1 --wait
```

then run the `db create`/`user create`/`ALTER DATABASE` steps above.

---

## 5. Configure Manifests

In `k8s/deployment.yaml`, replace every occurrence of:
- `auth.yourdomain.com` → your actual domain

If using Spaces backup, also replace:
- `your-backup-bucket` → your Spaces bucket name
- `nyc3.digitaloceanspaces.com` → your region endpoint if different

---

## 6. Create Kubernetes Secrets

Edit `k8s/secrets.example.sh` with real values, then run it:

```bash
# Generates oauth2_signing.key in the current directory, then creates the app secret.
bash k8s/secrets.example.sh
```

> **Keep `oauth2_signing.key` backed up.** Losing it invalidates all active JWTs.

`minidm-secrets` only needs `oauth2-issuer`. Separately, create the database connection secret
using the user/password/database from step 4:

```bash
HOST=<apps-postgres host, from doctl databases connection>
PORT=25060
kubectl create secret generic minidm-db-app \
  --from-literal=uri="postgresql://minidm_app:<password>@${HOST}:${PORT}/minidm?sslmode=require"
```

`k8s/deployment.yaml` and `k8s/migrate-job.yaml` both read `DATABASE_URL` from this secret's
`uri` key.

---

## 7. Apply Manifests

```bash
kubectl apply -f k8s/deployment.yaml
```

Verify everything comes up:

```bash
kubectl get pods -w
```

---

## 8. Run Database Migrations

Migrations are not run automatically on deploy. Run them once after the initial deploy and again whenever new migration files are added.

**Via GitHub Actions (recommended):**

Go to **Actions → Migrate → Run workflow** in your GitHub repo.

**Manually:**

```bash
kubectl delete job minidm-migrate --ignore-not-found
kubectl apply -f k8s/migrate-job.yaml
kubectl wait --for=condition=complete job/minidm-migrate --timeout=180s
kubectl logs -l job-name=minidm-migrate
```

---

## 9. Bootstrap the First Admin

Set the bootstrap env vars directly on the running deployment, restart once, then remove them:

```bash
kubectl set env deployment/minidm \
  BOOTSTRAP_ADMIN_EMAIL=you@example.com \
  BOOTSTRAP_ADMIN_PASSWORD=your_secure_password

# Wait for rollout, then check logs to confirm the account was created
kubectl rollout status deployment/minidm

# Remove bootstrap vars — they only need to run once
kubectl set env deployment/minidm \
  BOOTSTRAP_ADMIN_EMAIL- \
  BOOTSTRAP_ADMIN_PASSWORD-
```

---

## 10. Set Up GitHub Actions

Add these secrets to your GitHub repo (**Settings → Secrets and variables → Actions**):

| Secret | Value |
|--------|-------|
| `DIGITALOCEAN_ACCESS_TOKEN` | DO personal access token, **Kubernetes: Read** scope (fine-grained tokens) — the only DO API call this workflow makes is fetching the kubeconfig |
| `CLUSTER_NAME` | DOKS cluster name (e.g. `minidm-cluster`) |
| `REGISTRY_HOST` | Registry hostname (e.g. `registry.jjones.dev`) |
| `REGISTRY_USERNAME` | htpasswd username |
| `REGISTRY_PASSWORD` | htpasswd password |

Push to `main` — the **Deploy** workflow builds both images and rolls out the new version. Rollback is automatic if the rollout times out.

---

## Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | Postgres connection string. In DOKS, sourced from the `minidm-db-app` secret (manually created, step 4/6 — DigitalOcean Managed PostgreSQL). |
| `SECURE_COOKIES` | Prod | `false` | Set `true` when serving over HTTPS |
| `OAUTH2_ISSUER` | Yes | `http://localhost:8080` | Externally reachable base URL (used in JWT `iss` claim) |
| `OAUTH2_KEY_PATH` | No | `oauth2_signing.key` | Path to RSA private key (mounted from k8s secret) |
| `BOOTSTRAP_ADMIN_EMAIL` | First run | — | Seeds the first admin identity |
| `BOOTSTRAP_ADMIN_PASSWORD` | First run | — | Required when `BOOTSTRAP_ADMIN_EMAIL` is set |
