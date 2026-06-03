# Deploying minIDM to Digital Ocean Kubernetes (DOKS)

## Overview

The deployment uses:
- A **DO Container Registry** to store the app and migrator images
- A **DOKS cluster** running the app
- **CloudNativePG (CNPG)** as the PostgreSQL operator — managed cluster with optional Spaces backup
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

## 2b. Alternative: In-Cluster Container Registry

Use this instead of the DO Container Registry if you want to avoid the registry subscription cost or keep images fully self-hosted.

Two storage backends are available — pick one:

| | Option A: Block Storage (PVC) | Option B: DigitalOcean Spaces |
|---|---|---|
| Storage | DO block volume attached to a node | S3-compatible object storage bucket |
| Replicas | 1 (volume is `ReadWriteOnce`) | Many (stateless pod, storage is remote) |
| Node failure | Pod must reschedule to same AZ | Any node, any AZ |
| Backups | Manual DO volume snapshots | Built-in Spaces versioning / lifecycle |
| Best for | Simple single-node setups | Production or multi-node clusters |

---

### Option A: Block Storage (PVC)

Create `k8s/registry.yaml`:

```yaml
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: registry-pvc
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 10Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: registry
spec:
  replicas: 1
  selector:
    matchLabels:
      app: registry
  template:
    metadata:
      labels:
        app: registry
    spec:
      containers:
        - name: registry
          image: registry:2
          ports:
            - containerPort: 5000
          env:
            - name: REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY
              value: /var/lib/registry
            # Optional: enable htpasswd auth (see below)
            # - name: REGISTRY_AUTH
            #   value: htpasswd
            # - name: REGISTRY_AUTH_HTPASSWD_REALM
            #   value: Registry
            # - name: REGISTRY_AUTH_HTPASSWD_PATH
            #   value: /auth/htpasswd
          volumeMounts:
            - name: storage
              mountPath: /var/lib/registry
            # - name: auth
            #   mountPath: /auth
      volumes:
        - name: storage
          persistentVolumeClaim:
            claimName: registry-pvc
        # - name: auth
        #   secret:
        #     secretName: registry-htpasswd
---
apiVersion: v1
kind: Service
metadata:
  name: registry
spec:
  selector:
    app: registry
  ports:
    - port: 5000
      targetPort: 5000
---
# Expose the registry externally via Traefik so GitHub Actions can push to it.
# Replace registry.yourdomain.com with your actual subdomain.
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: registry
spec:
  entryPoints: [websecure]
  routes:
    - match: Host(`registry.yourdomain.com`)
      kind: Rule
      services:
        - name: registry
          port: 5000
  tls:
    certResolver: letsencrypt
```

Apply it and point a DNS A record (`registry.yourdomain.com`) at the Traefik load balancer IP:

```bash
kubectl apply -f k8s/registry.yaml
```

---

### Option B: DigitalOcean Spaces

`registry:2` has a built-in S3 storage driver. Image layers are stored as objects in a Spaces bucket — the registry pod is stateless and can run on any node.

**1. Create a Spaces bucket**

In the DO console: **Spaces → Create Space**. Choose the same region as your cluster (e.g. `nyc3`). Note the bucket name and region endpoint (e.g. `nyc3.digitaloceanspaces.com`).

**2. Create Spaces access keys**

In the DO console: **API → Spaces Keys → Generate New Key**. Save the access key and secret — they are shown only once.

**3. Store the credentials as a Kubernetes Secret**

```bash
kubectl create secret generic registry-spaces-credentials \
  --from-literal=access-key=YOUR_SPACES_ACCESS_KEY \
  --from-literal=secret-key=YOUR_SPACES_SECRET_KEY
```

**4. Create `k8s/registry.yaml`**

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: registry
spec:
  replicas: 1
  selector:
    matchLabels:
      app: registry
  template:
    metadata:
      labels:
        app: registry
    spec:
      containers:
        - name: registry
          image: registry:2
          ports:
            - containerPort: 5000
          env:
            - name: REGISTRY_STORAGE
              value: s3
            - name: REGISTRY_STORAGE_S3_REGIONENDPOINT
              value: https://nyc3.digitaloceanspaces.com   # change region if needed
            - name: REGISTRY_STORAGE_S3_REGION
              value: us-east-1   # registry:2 requires a value; DO ignores it
            - name: REGISTRY_STORAGE_S3_BUCKET
              value: your-bucket-name
            - name: REGISTRY_STORAGE_S3_SECURE
              value: "true"
            - name: REGISTRY_STORAGE_S3_V4AUTH
              value: "true"
            - name: REGISTRY_STORAGE_S3_ACCESSKEY
              valueFrom:
                secretKeyRef:
                  name: registry-spaces-credentials
                  key: access-key
            - name: REGISTRY_STORAGE_S3_SECRETKEY
              valueFrom:
                secretKeyRef:
                  name: registry-spaces-credentials
                  key: secret-key
            # Optional: enable htpasswd auth (see below)
            # - name: REGISTRY_AUTH
            #   value: htpasswd
            # - name: REGISTRY_AUTH_HTPASSWD_REALM
            #   value: Registry
            # - name: REGISTRY_AUTH_HTPASSWD_PATH
            #   value: /auth/htpasswd
          # volumeMounts:
          #   - name: auth
          #     mountPath: /auth
      # volumes:
      #   - name: auth
      #     secret:
      #       secretName: registry-htpasswd
---
apiVersion: v1
kind: Service
metadata:
  name: registry
spec:
  selector:
    app: registry
  ports:
    - port: 5000
      targetPort: 5000
---
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: registry
spec:
  entryPoints: [websecure]
  routes:
    - match: Host(`registry.yourdomain.com`)
      kind: Rule
      services:
        - name: registry
          port: 5000
  tls:
    certResolver: letsencrypt
```

Apply it and point a DNS A record (`registry.yourdomain.com`) at the Traefik load balancer IP:

```bash
kubectl apply -f k8s/registry.yaml
```

---

### Optional: enable htpasswd authentication (both options)

```bash
# Generate credentials (replace user/password)
htpasswd -Bbn registryuser secretpassword > htpasswd

kubectl create secret generic registry-htpasswd \
  --from-file=htpasswd=htpasswd
```

Then uncomment the `REGISTRY_AUTH` env vars and the `auth` volume/volumeMount in `registry.yaml` and re-apply.

### Configure GitHub Actions (both options)

Add these secrets to your repo in addition to the ones in step 10:

| Secret | Value |
|--------|-------|
| `REGISTRY_HOST` | `registry.yourdomain.com` |
| `REGISTRY_USERNAME` | htpasswd username (omit if auth disabled) |
| `REGISTRY_PASSWORD` | htpasswd password (omit if auth disabled) |

Update your workflow's build-and-push step to log in with these credentials instead of `doctl registry login`:

```yaml
- name: Log in to in-cluster registry
  uses: docker/login-action@v3
  with:
    registry: ${{ secrets.REGISTRY_HOST }}
    username: ${{ secrets.REGISTRY_USERNAME }}
    password: ${{ secrets.REGISTRY_PASSWORD }}

- name: Build and push
  uses: docker/build-push-action@v5
  with:
    push: true
    tags: ${{ secrets.REGISTRY_HOST }}/minidm:${{ github.sha }}
```

Update image references in `k8s/deployment.yaml` and `k8s/migrate-job.yaml`:

```yaml
image: registry.yourdomain.com/minidm:latest
```

The cluster pulls from the in-cluster registry via its ClusterIP service (no image pull secret needed when auth is disabled). If htpasswd auth is enabled, create an `imagePullSecret` from the same credentials and reference it in your pod specs.

---

## 3. Bootstrap the Cluster

Before deploying minIDM, complete the cluster setup in the [doks-infra](https://github.com/jjones028/doks-infra) repo. That covers DOKS cluster provisioning, Traefik, and CloudNativePG. Come back here once `task bootstrap` has run successfully and your DNS A record for `auth.yourdomain.com` points at the Traefik load balancer IP.

Allow the cluster to pull from the DO Container Registry (skip if using an in-cluster registry):

```bash
doctl registry kubernetes-manifest | kubectl apply -f -
```

---

## 5. Configure Manifests

In `k8s/deployment.yaml`, replace every occurrence of:
- `YOUR_REGISTRY` → your DO container registry name (or `registry.yourdomain.com` if using in-cluster registry)
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

`minidm-secrets` now only needs `oauth2-issuer` — the `database-url` / `db-password` keys are no longer required. CloudNativePG generates the `minidm-pg-app` secret automatically with a `uri` key that the app reads directly.

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
| `DATABASE_URL` | Yes | — | Postgres connection string. In DOKS, sourced automatically from the `minidm-pg-app` secret generated by CloudNativePG (`uri` key). |
| `SECURE_COOKIES` | Prod | `false` | Set `true` when serving over HTTPS |
| `OAUTH2_ISSUER` | Yes | `http://localhost:8080` | Externally reachable base URL (used in JWT `iss` claim) |
| `OAUTH2_KEY_PATH` | No | `oauth2_signing.key` | Path to RSA private key (mounted from k8s secret) |
| `BOOTSTRAP_ADMIN_EMAIL` | First run | — | Seeds the first admin identity |
| `BOOTSTRAP_ADMIN_PASSWORD` | First run | — | Required when `BOOTSTRAP_ADMIN_EMAIL` is set |
