# Deploying minIDM to Digital Ocean Kubernetes (DOKS)

## Overview

The deployment uses:
- A **DO Container Registry** to store the app and migrator images
- A **DOKS cluster** running the app + PostgreSQL StatefulSet
- **Traefik** as the ingress controller with automatic Let's Encrypt TLS
- **GitHub Actions** to build, push, and roll out on every push to `main`

---

## 1. Prerequisites

Install and authenticate the DO CLI:

```bash
brew install doctl helm kubectl
doctl auth init   # paste your DO personal access token
```

---

## 2. Create a Container Registry

```bash
doctl registry create minidm-registry --subscription-tier starter
```

Note the registry name — you'll use it as the `REGISTRY_NAME` secret.

---

## 3. Provision a DOKS Cluster

```bash
doctl kubernetes cluster create minidm-cluster \
  --region nyc3 \
  --node-pool "name=default;size=s-2vcpu-4gb;count=2"
```

Save the kubeconfig:

```bash
doctl kubernetes cluster kubeconfig save minidm-cluster
```

Allow the cluster to pull from your registry:

```bash
doctl registry kubernetes-manifest | kubectl apply -f -
```

---

## 4. Install Traefik

```bash
helm repo add traefik https://helm.traefik.io/traefik
helm repo update

helm install traefik traefik/traefik \
  --namespace traefik \
  --create-namespace \
  -f k8s/traefik-values.yaml
```

Edit `k8s/traefik-values.yaml` first — replace the `email` with yours.

Get the load balancer IP Traefik provisions:

```bash
kubectl get svc -n traefik traefik -w
```

Point your DNS A record (`auth.yourdomain.com`) at that IP.

---

## 5. Configure Manifests

In `k8s/deployment.yaml`, replace every occurrence of:
- `YOUR_REGISTRY` → your DO container registry name
- `auth.yourdomain.com` → your actual domain

In `k8s/traefik-values.yaml`, replace `you@yourdomain.com` with your email.

---

## 6. Create Kubernetes Secrets

Edit `k8s/secrets.example.sh` with real values, then run it:

```bash
# Generates oauth2_signing.key in the current directory, then creates both secrets.
bash k8s/secrets.example.sh
```

> **Keep `oauth2_signing.key` backed up.** Losing it invalidates all active JWTs.

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
sed "s/YOUR_REGISTRY/your-registry-name/g" k8s/migrate-job.yaml | kubectl apply -f -
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
| `DIGITALOCEAN_ACCESS_TOKEN` | DO personal access token |
| `REGISTRY_NAME` | Container registry name (e.g. `minidm-registry`) |
| `CLUSTER_NAME` | DOKS cluster name (e.g. `minidm-cluster`) |

Push to `main` — the **Deploy** workflow builds both images and rolls out the new version. Rollback is automatic if the rollout times out.

---

## Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | Postgres connection string |
| `SECURE_COOKIES` | Prod | `false` | Set `true` when serving over HTTPS |
| `OAUTH2_ISSUER` | Yes | `http://localhost:8080` | Externally reachable base URL (used in JWT `iss` claim) |
| `OAUTH2_KEY_PATH` | No | `oauth2_signing.key` | Path to RSA private key (mounted from k8s secret) |
| `BOOTSTRAP_ADMIN_EMAIL` | First run | — | Seeds the first admin identity |
| `BOOTSTRAP_ADMIN_PASSWORD` | First run | — | Required when `BOOTSTRAP_ADMIN_EMAIL` is set |
