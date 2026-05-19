# minIDM

A self-hosted Identity Management system and OAuth2/OIDC provider built with Go and React.

## Features

- **Identity management** — register, list, view, enable/disable, and admin password reset
- **Cookie-based sessions** — `HttpOnly; SameSite=Strict` cookies; no tokens in localStorage
- **RBAC** — resource × action permission model; built-in `admin` and `viewer` roles
- **OAuth2/OIDC provider** — authorization code flow with mandatory PKCE (S256), RS256 JWTs, OIDC discovery, JWKS endpoint
- **Public client support** — native/mobile apps (RFC 8252); no client secret required, PKCE enforced
- **Token introspection** (RFC 7662) and **token revocation** (RFC 7009)
- **Consent screen** — explicit per-authorization consent with optional per-client auto-consent
- **Audit log** — filterable, paginated audit trail with actor email for all administrative actions
- **Admin portal** — full React UI for managing identities, roles, OAuth2 clients, tokens, and audit logs
- **Production-ready** — Docker Compose deployment with health-checked migrations and persistent key storage

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26+, `pgxpool/v5`, `sqlc`, `goose`, `golang-jwt/jwt` |
| Frontend | React 19, Vite, Tailwind CSS v4, Shadcn/ui (`radix-mira`) |
| Database | PostgreSQL |
| Build | [go-task](https://taskfile.dev/) |

## Quick Start

### Development Mode (Recommended)

Requires Docker or Podman Desktop running.

```bash
# Install dependencies
cd services/api && go mod download && cd ../..
cd web && npm install && cd ..

# Start everything (backend + frontend hot reload + ephemeral DB via Testcontainers)
task dev
```

The API runs at `http://localhost:8080`; the frontend (with hot reload) at `http://localhost:5173`. The first registered identity automatically becomes admin.

### Production (Docker Compose)

```bash
# Build the unified binary + migrate binary
task build

# Start database, run migrations, then start the app
docker compose up -d
```

The app is available at `http://localhost:8080`. Set `SECURE_COOKIES=true` and update `OAUTH2_ISSUER` to your public URL before deploying to production.

### Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `DATABASE_URL` | `postgres://user:pass@localhost:5432/idm?sslmode=disable` | PostgreSQL connection string |
| `OAUTH2_KEY_PATH` | `oauth2_signing.key` | RSA private key PEM; auto-generated on first run |
| `OAUTH2_ISSUER` | `http://localhost:8080` | JWT `iss` claim and OIDC discovery base URL |
| `SECURE_COOKIES` | `false` | Set `true` in production (requires HTTPS) |

## Project Structure

```
services/api/
  cmd/
    server/       ← main entry point (unified binary serving API + SPA)
    migrate/      ← standalone migration binary (used in Docker)
    dev/          ← development runner (Testcontainers + hot reload)
  internal/
    app/          ← router, server startup, static file embedding
    identity/     ← identity CRUD, sessions, enable/disable, password reset
    rbac/         ← roles, permissions, middleware
    session/      ← login/logout cookie handling
    oauth2/       ← full OAuth2/OIDC provider
    audit/        ← audit log write + query
  db/
    migrations/   ← goose SQL migrations (001–013)
    queries/      ← sqlc query sources
    sqlc/         ← generated Go code (never edit)

web/
  src/
    pages/        ← React page components
    components/   ← shared UI components
    context/      ← auth context
    api.ts        ← all API calls + TypeScript types
```

## API Overview

### Identity
| Method | Path | Auth | Notes |
|--------|------|------|-------|
| `POST` | `/api/register` | Public | Register new identity |
| `GET` | `/api/me` | Session | Current identity info |
| `GET` | `/api/identities` | `identity:read` | List all |
| `GET` | `/api/identities/{id}` | `identity:read` | Detail |
| `PATCH` | `/api/identities/{id}/enabled` | `identity:write` | Enable/disable |
| `POST` | `/api/identities/{id}/reset-password` | `identity:write` | Admin password reset |
| `GET` | `/api/identities/{id}/sessions` | `identity:read` | Active sessions |
| `DELETE` | `/api/identities/{id}/sessions/{handle}` | `identity:write` | Revoke session |

### OAuth2/OIDC
| Method | Path | Auth | Notes |
|--------|------|------|-------|
| `GET` | `/.well-known/openid-configuration` | Public | OIDC discovery |
| `GET` | `/oauth2/jwks.json` | Public | RSA public key |
| `GET` | `/oauth2/authorize` | Session cookie | Redirect-based; unauthenticated → `/login?next=` |
| `POST` | `/oauth2/token` | Client credentials | `authorization_code` or `refresh_token` grant |
| `POST` | `/oauth2/introspect` | Client credentials | RFC 7662 |
| `POST` | `/oauth2/revoke` | Client credentials | RFC 7009 |
| `GET` | `/oauth2/userinfo` | Bearer JWT | Returns `sub`, `email` |
| `GET/POST` | `/api/oauth2/clients` | `oauth2_client:read/write` | List / create |
| `GET/PATCH/DELETE` | `/api/oauth2/clients/{id}` | `oauth2_client:read/write` | Get / update / delete |
| `POST` | `/api/oauth2/clients/{id}/rotate-secret` | `oauth2_client:write` | Rotate secret (confidential clients only) |
| `GET` | `/api/oauth2/tokens` | `oauth2_client:read` | List active tokens |
| `DELETE` | `/api/oauth2/tokens/{id}` | `oauth2_client:write` | Admin-revoke token |
| `POST` | `/api/oauth2/tokens/inspect` | `oauth2_client:read` | Decode + validate raw token |

### Audit
| Method | Path | Auth | Notes |
|--------|------|------|-------|
| `GET` | `/api/audit-logs` | `audit_log:read` | Filterable, paginated |
| `GET` | `/api/audit-logs/resource-types` | `audit_log:read` | Distinct resource types |

## Development Tasks

```bash
task --list          # list all available tasks
task dev             # start dev mode (backend + frontend)
task gen             # regenerate sqlc code + run migrations
task build           # build production binary + frontend
task web:build       # build frontend only
task build:api       # build Go binaries only
```

## Database Migrations

Migrations live in `services/api/db/migrations/` and use [goose](https://github.com/pressly/goose) with `-- +goose Up` / `-- +goose Down` annotations.

In development, `task gen` runs migrations automatically. In production, the `migrate` container (or binary) runs them before the app starts.

To add a new migration:
1. Create `services/api/db/migrations/<NNN>_<name>.sql`
2. Add SQL queries to `services/api/db/queries/<feature>.sql`
3. Run `task gen`
