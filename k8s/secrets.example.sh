# Run these commands once before the first deploy.
# Fill in real values — never commit this file with secrets in it.

# ── 1. Generate the RSA signing key ───────────────────────────────────────────
# The same key must persist across all pod restarts; generate it once and keep
# the file safe (back it up separately from your cluster).
openssl genrsa -out oauth2_signing.key 2048

# ── 2. Application secrets ────────────────────────────────────────────────────
kubectl create secret generic minidm-secrets \
  --from-literal=db-password='your_strong_password' \
  --from-literal=database-url='postgres://minidm:your_strong_password@postgres-service:5432/minidm?sslmode=disable' \
  --from-literal=oauth2-issuer='https://auth.yourdomain.com'

# ── 3. OAuth2 RSA signing key ─────────────────────────────────────────────────
kubectl create secret generic minidm-oauth2-key \
  --from-file=oauth2_signing.key=./oauth2_signing.key
