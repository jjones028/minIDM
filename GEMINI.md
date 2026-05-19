# Project Context: minIDM

## Technical Stack
- **Backend**: Go 1.26.1+, `pgxpool/v5` (PostgreSQL connection pool), `sqlc` (type-safe SQL), `goose` (migrations).
- **Frontend**: React 19, Vite, Tailwind CSS v4, Radix UI / Shadcn/ui.
- **Design System**: Shadcn/ui (`radix-mira` style) with OKLCH colors. `web/src/index.css` is the source of truth for theming.
- **Tooling**: `go-task` (orchestration), Docker/Testcontainers (dev DB lifecycle via `cmd/dev/main.go`).

## Architecture
- **Pattern**: Backend-for-Frontend (BFF). The API is specifically designed to serve the primary Web UI.
- **Deployment Model**: Single atomic binary. The React frontend is built and embedded into the Go binary via `//go:embed` in `internal/app/static.go`.
- **Backend Structure**: Feature-based CQRS. Core logic lives in `internal/<feature_name>/` (e.g. `internal/identity/`, `internal/rbac/`, `internal/session/`, `internal/oauth2/`). Each feature follows a strict pattern:
  - `handler.go`: HTTP routing and JSON marshalling only.
  - `<command>.go` / `<query>.go`: Isolated logic handlers that execute against `db.Queries`.
  - Business rules (e.g., protecting built-in roles, PKCE validation) are enforced in Command handlers, not HTTP handlers.
- **RBAC Model**: Roles → Resources × Actions. The `IdentityHasPermission` SQL query drives all authorization checks.
- **Dev Proxy**: Vite dev server proxies `/api`, `/oauth2`, and `/.well-known` → `http://localhost:8080`. All three are required: OAuth2 authorize redirects use relative URLs that must resolve on the Vite port (5173).

## Current State (as of 2026-05-19)

### Backend
- **Identity**: Registration (Argon2id hashing), listing, detail retrieval, enable/disable, admin password reset, active session listing + revocation. First identity → `admin` role; subsequent → `viewer` role.
  - `GET /api/identities/{id}` — identity details (omits `pw_hash`)
  - `GET /api/identities/{id}/sessions` — active sessions `[{handle, created_at, expires_at}]` (token never exposed; handle = first 8 hex chars of SHA-256)
  - `DELETE /api/identities/{id}/sessions/{handle}` — revoke a session by opaque handle (`identity:write`)
  - `PATCH /api/identities/{id}/enabled` — enable/disable identity (`identity:write`); disabled identities cannot log in
  - `POST /api/identities/{id}/reset-password` — admin password reset (`identity:write`); re-hashes Argon2id
  - Key files: `get_identity.go`, `list_identity_sessions.go`, `revoke_identity_session.go`, `set_enabled.go`, `reset_password.go`
- **RBAC**: Full schema — `roles`, `resources`, `actions`, `permissions`, `identity_roles`. Middleware in `internal/rbac/middleware.go`:
  - `Authenticate` reads the `session` HTTP cookie and validates it against the `sessions` table.
  - `Require(resource, action)` checks the identity has the named permission.
- **Sessions**: Cookie-based. Login → `Set-Cookie: session=<token>; HttpOnly; SameSite=Strict`. `SECURE_COOKIES=true` env var enables `Secure` flag.
- **`GET /api/me`**: Auth-only endpoint. Returns `{ id }` for the current identity.
- **Connection pool**: `server.go` uses `pgxpool.New`.
- **Routes**: `internal/app/router.go`. `NewHandler(queries, signingKey, issuer)` now takes RSA key and issuer for OAuth2.

### OAuth2/OIDC Provider (completed 2026-05-07)
Package `internal/oauth2/`. Full authorization code flow with mandatory PKCE (S256), RS256 JWT signing, OIDC discovery, and client management UI.

#### DB Schema (migrations 005–008)
```
oauth2_clients
  id UUID PK, client_id TEXT UNIQUE, client_secret_hash TEXT,
  name TEXT, description TEXT, redirect_uris TEXT[], scopes TEXT[],
  is_enabled BOOL, created_at, updated_at

oauth2_authorization_codes
  code TEXT PK, client_id TEXT→oauth2_clients, identity_id UUID→identities,
  redirect_uri TEXT, scopes TEXT[], code_challenge TEXT,
  code_challenge_method TEXT DEFAULT 'S256', expires_at TIMESTAMPTZ,
  used BOOL DEFAULT FALSE

oauth2_tokens
  id UUID PK, client_id TEXT, identity_id UUID,
  jti TEXT UNIQUE, refresh_token_hash TEXT,
  scopes TEXT[], expires_at TIMESTAMPTZ, revoked BOOL DEFAULT FALSE
```
Migration 008 seeds `oauth2_client` resource and grants admin full permissions.

#### Public Endpoints (no RBAC)
| Endpoint | Notes |
|----------|-------|
| `GET /.well-known/openid-configuration` | OIDC discovery document (includes introspection + revocation endpoints) |
| `GET /oauth2/jwks.json` | RSA-2048 public key as JWKS |
| `GET /oauth2/authorize` | Checks session cookie itself; redirects unauthenticated to `/login?next=<url>` |
| `POST /oauth2/token` | Form-encoded; `grant_type=authorization_code` or `refresh_token` |
| `POST /oauth2/introspect` | RFC 7662; requires client auth; returns `{active: true/false, ...claims}` |
| `POST /oauth2/revoke` | RFC 7009; requires client auth; always returns 200 |
| `GET /oauth2/userinfo` | Bearer token (RS256 JWT); validates signature + JTI revocation |

#### Admin API (RBAC: `oauth2_client` resource)
`GET/POST /api/oauth2/clients` and `GET/PATCH/DELETE /api/oauth2/clients/{id}`. The `client_secret_hash` field is never exposed; plaintext secret returned only on creation in `{ client, client_secret }`.

- `POST /api/oauth2/clients/{id}/rotate-secret` — re-generate + re-hash secret; returns `{ client_secret }` once. 422 for public clients.
- `GET /api/oauth2/tokens` — list all active (non-revoked, non-expired) tokens
- `DELETE /api/oauth2/tokens/{id}` — admin-revoke a token by DB UUID
- `POST /api/oauth2/tokens/inspect` — decode + validate a raw token string; returns claims + DB status

#### Public Client Support
`client_secret_hash IS NULL` → public client (for native/mobile apps, RFC 8252). Migration `013_public_client_support.sql` makes the column nullable. `authenticateClient()` helper in `crypto.go` accepts `client_id` only for public clients. PKCE remains mandatory for all clients.

#### Token Design
- **Access token**: RS256-signed JWT. Claims: `sub` (identity.subject_id), `iss`, `aud` (client_id), `exp`, `iat`, `jti` (random UUID), `scope`, `email` (if requested).
- **id_token**: same JWT as access token (satisfies OIDC requirements for simple deployments).
- **Refresh token**: opaque random (32 bytes base64url), stored as SHA-256 hex hash.
- **Revocation**: `jti` stored in `oauth2_tokens.jti`. Userinfo endpoint does DB lookup to confirm not revoked.
- **Refresh rotation**: old `oauth2_tokens` row marked `revoked=TRUE`; new row created.

#### New Environment Variables
| Variable | Default | Purpose |
|----------|---------|---------|
| `OAUTH2_KEY_PATH` | `oauth2_signing.key` | RSA private key PEM. Auto-generated if file does not exist |
| `OAUTH2_ISSUER` | `http://localhost:8080` | JWT `iss` claim and discovery base URL |

### Frontend
- **Auth context** (`web/src/context/auth.tsx`): Calls `GET /api/me` on mount to establish initial auth state.
- **Routing** (`web/src/App.tsx`): `ProtectedRoute` + routes for `/`, `/identities/:id`, `/identities/:id/roles`, `/roles`, `/roles/:id/permissions`, `/oauth2/clients`, `/oauth2/tokens`, `/audit-logs`.
- **API client** (`web/src/api.ts`): Axios with `baseURL: '/api'`. All API calls centralized here with TypeScript types.
- **Navigation** (`web/src/components/app-nav.tsx`): Identities | Roles | OAuth2 Clients | Tokens | Audit Log.
- **AuthPage** (`web/src/pages/AuthPage.tsx`): Reads `?next=` query param; uses `window.location.href` (not React Router `navigate`) so backend OAuth2 redirects work correctly.
- **IdentityDetailPage** (`web/src/pages/IdentityDetailPage.tsx`): Subject ID, enable/disable toggle, password reset form, assigned roles, active sessions with revoke buttons.
- **OAuthClientsPage** (`web/src/pages/OAuthClientsPage.tsx`): Create/list/edit/delete clients. Public client checkbox. Amber `public` badge. Blue `auto-consent` badge. Secret shown once in modal.
- **TokensPage** (`web/src/pages/TokensPage.tsx`): List active tokens with admin revoke; raw token inspect form showing decoded claims.
- **AuditLogsPage** (`web/src/pages/AuditLogsPage.tsx`): Filterable (resource type, action prefix, actor UUID, date range), paginated, shows actor email.
- **Shadcn components installed**: `button`, `card`, `input`, `table`, `select`. Note: no `badge` component — use inline `<span>` with Tailwind classes for status indicators.

## Critical Files
| File | Purpose |
|------|---------|
| `services/api/internal/app/router.go` | All route registration and middleware chains |
| `services/api/internal/app/server.go` | Startup: DB pool, RSA key load, issuer config |
| `services/api/internal/identity/handler.go` | Identity API — all endpoints; has its own `parseUUID` helper |
| `services/api/internal/rbac/middleware.go` | `Authenticate` + `Require` middleware |
| `services/api/internal/rbac/handler.go` | Role and Permission management API (also owns identity role routes) |
| `services/api/internal/session/handler.go` | Login / logout cookie logic |
| `services/api/internal/oauth2/` | Full OAuth2/OIDC provider package (authorize, token, introspect, revoke, userinfo, JWKS, discovery, clients, tokens) |
| `services/api/internal/audit/` | Audit logging package |
| `services/api/db/migrations/` | `001`–`013` (identity, rbac, sessions, builtin, oauth2 ×4, audit ×2, auto_consent, nonce, public_client) |
| `services/api/db/migrations/embed.go` | `//go:embed *.sql` for production migrate binary |
| `services/api/cmd/migrate/main.go` | Standalone migration binary (used in Docker) |
| `services/api/db/queries/audit.sql` | Filtered audit log queries with LEFT JOIN for actor email |
| `web/src/context/auth.tsx` | React auth state |
| `web/src/api.ts` | All API calls + TypeScript types |
| `web/src/components/app-nav.tsx` | Main application navigation |
| `web/vite.config.js` | Vite proxy (`/api`, `/oauth2`, `/.well-known` → `:8080`) |
| `compose.yaml` | Production Docker Compose (db + migrate + app) |

## CQRS Pattern (follow this for all new features)
```
internal/<feature>/
  handler.go          ← Register(mux, ...) wires routes; HTTP methods parse+encode only
  <command_name>.go   ← XxxHandler + XxxCommand + Handle(ctx, cmd) → (result, error)
  <query_name>.go     ← XxxHandler + Handle(ctx, ...) → (result, error)
```
Business rules always in command handlers. HTTP handlers never contain SQL or domain logic.

## Sqlc Workflow
1. Add migration: `db/migrations/<NNN>_<name>.sql` with `-- +goose Up` / `-- +goose Down`
2. Add queries: `db/queries/<feature>.sql` with `-- name: FnName :one|:many|:exec`
3. Run: `task gen` (sqlc generate + goose up)
4. Edit `db/sqlc/` — **never** (auto-generated)

Type mappings (pgx/v5 driver):
- `UUID` → `pgtype.UUID` | `TEXT NOT NULL` → `string` | `TEXT` nullable → `pgtype.Text`
- `TEXT[]` → `[]string` | `BOOL NOT NULL` → `bool` | `TIMESTAMPTZ NOT NULL` → `pgtype.Timestamptz`

### Audit Logging (completed 2026-05-08)
Package `internal/audit/`. Tracks administrative events system-wide.

#### DB Schema (migrations 009–010)
```
audit_logs
  id UUID PK, actor_id UUID→identities (nullable, NULL=system),
  action TEXT, resource_type TEXT, resource_id TEXT (nullable),
  details JSONB (nullable), created_at TIMESTAMPTZ
```
Migration 010 seeds `audit_log` resource and grants admin `read` permission.

#### Package Structure
| File | Responsibility |
|------|----------------|
| `auditor.go` | `Auditor.Log(ctx, actorID, action, resourceType, resourceID, details)` — fire-and-forget DB insert. `UUIDStr(pgtype.UUID)` helper |
| `list_events.go` | `ListEventsHandler` — `ListAuditLogs` with limit/offset |
| `handler.go` | `Register(mux, q, protectRead) *Auditor` — wires `GET /api/audit-logs`, returns shared `Auditor` |

#### Endpoints
- `GET /api/audit-logs?resource_type=X&action=Y&actor_id=Z&since=T&until=T&limit=N&offset=N` — protected `audit_log:read`. Returns `{"total": N, "logs": [...]}`. All filters optional; default page size 50.
- `GET /api/audit-logs/resource-types` — protected `audit_log:read`. Returns distinct resource type values for the filter dropdown.

`ListAuditLogsFiltered` LEFT JOINs `identities` to return `actor_email`. sqlc generates unnamed `$N::type` params as `Column1`–`Column5`.

#### Events
`session.login`, `session.logout`, `identity.register`, `identity.session.revoke`, `identity.enable`, `identity.disable`, `identity.password.reset`, `role.create/update/delete`, `role.permission.add/remove`, `identity.role.assign/remove`, `oauth2_client.create/update/delete/rotate_secret`, `oauth2_token.revoke`.

#### Wiring
`audit.Register` is called first in `router.go`; the returned `*Auditor` is passed to every other `Register` function. `session/logout.go` updated to look up session before delete to capture actor ID.

#### Frontend
- `web/src/pages/AuditLogsPage.tsx` — filter bar (resource type dropdown, action prefix, actor UUID, since/until), paginated table, actor email display
- `web/src/api.ts` — `AuditLog{actor_email}`, `AuditLogsResponse{total, logs}`, `AuditLogsFilter`, `listAuditLogs(filter)`, `listAuditResourceTypes()`
- `web/src/App.tsx` — `/audit-logs` protected route
- `web/src/components/app-nav.tsx` — "Audit Log" nav link

### OAuth2 Consent Screen + Auto-Consent (completed 2026-05-08)

#### DB (migration 011)
`ALTER TABLE oauth2_clients ADD COLUMN auto_consent BOOLEAN NOT NULL DEFAULT FALSE`

#### New Endpoints
| Method | Path | Auth | Notes |
|--------|------|------|-------|
| `GET` | `/api/oauth2/client-info?client_id=X` | Public | `{name, description, scopes, auto_consent}` for consent page |
| `POST` | `/api/oauth2/consent` | Session cookie | Re-validates params, issues auth code, returns `{redirect_url}` |

#### Flow
`GET /oauth2/authorize` → after full validation, if `auto_consent=false` → redirect to `/oauth2/consent?{params}`. React page shows client name + requested scopes. Approve → `POST /api/oauth2/consent` → server re-validates + issues code → `{redirect_url}` → `window.location.href`. Deny → redirect to `redirect_uri?error=access_denied`.

#### New Files
- `internal/oauth2/consent.go` — `ConsentHandler`: re-validates all params + session, issues code
- `internal/oauth2/client_info.go` — `ClientInfoHandler`: public display info for consent page

#### `oauth2.Register` signature
Now accepts `authenticate func(http.Handler) http.Handler` as final param (gates `POST /api/oauth2/consent`).

#### Frontend
- `web/src/pages/ConsentPage.tsx` — consent UI at `/oauth2/consent`; Approve/Deny; no ProtectedRoute (handles 401 itself)
- `web/src/pages/OAuthClientsPage.tsx` — edit form: vertical scope checkboxes with descriptions; Auto-consent toggle under Settings section with warning copy; blue `auto-consent` badge in display row
- `web/src/api.ts` — `auto_consent` on `OAuthClient`/`UpdateOAuthClientData`; `ClientInfo`, `ConsentParams`, `getClientInfo`, `approveConsent`

#### Design Notes
- No pending-consent DB table: params are re-validated on POST (safe — PKCE binds the code to the verifier holder).
- `auto_consent` defaults to `false` for all clients.

### OAuth2 Nonce Support (completed 2026-05-08)
`nonce` flows from the authorize request → auth code → `id_token` JWT claim, satisfying OIDC libraries that validate nonce.

**Migration**: `012_add_nonce_to_auth_codes.sql` — `ALTER TABLE oauth2_authorization_codes ADD COLUMN nonce TEXT` (nullable).

**Backend changes**:
- `authorize.go`: reads `nonce` query param; included in consent redirect URL if present; stored in `CreateAuthorizationCode` as `pgtype.Text`
- `consent.go`: reads `nonce` from JSON body; passed to `CreateAuthorizationCode`
- `token.go`: `tokenClaims` gains `Nonce string \`json:"nonce,omitempty"\``; `issueAndRespond(... nonce string)` sets claim when `openid` in scopes; refresh grant passes `""` (nonce not round-tripped)

**Frontend changes**:
- `ConsentPage.tsx`: reads `nonce` from URL params; forwards in `approveConsent` POST body
- `api.ts`: `ConsentParams` gains `nonce?: string`

**Design decisions**: nonce only stamped in `id_token` (not access token) when `openid` scope present; nullable column leaves existing auth codes unaffected.

### Production Deployment (completed 2026-05-19)

#### Docker Compose (`compose.yaml`)
Three services: `db` (PostgreSQL 18 with `pg_isready` health check), `migrate` (standalone migration runner, `restart: "no"`), `app` (the unified binary, depends on migrate completing).

```bash
docker compose up -d   # start all services
docker compose down    # stop (data preserved in named volumes)
```

Named volumes: `idm_database` (Postgres), `app_keys` (RSA signing key — persists across restarts).

**Critical**: compose must use `entrypoint: ["/app/migrate"]`, NOT `command:`. The Dockerfile sets `ENTRYPOINT ["/app/minIDM"]`; `command:` would be appended as args to that entrypoint.

#### Standalone Migrate Binary
`services/api/cmd/migrate/main.go` embeds all migrations via `//go:embed *.sql` in `db/migrations/embed.go` and runs `goose.Up` via Go API. Independent of dev mode — `task dev` still uses Testcontainers + goose CLI.

#### Taskfile Fix
`build:api` task uses `cp -r ../../web/dist/. internal/app/dist/` (note the trailing `.`) to avoid Windows coreutils glob failures when copying the directory contents.

### OIDC Test Client (`~/Projects/oidc-test-client`) (completed 2026-05-19)
Reference Node.js/Express test application for end-to-end OIDC testing. Implements full authorization code flow with PKCE. Tokens stored server-side in Express session; browser only sees opaque `connect.sid` cookie.

Rewrites discovery endpoints from the server's internal issuer to the configured `ISSUER` so all OAuth2 calls route through Vite's proxy in dev mode.

Configure with `.env`: `ISSUER=http://localhost:5173`, `CLIENT_ID`, `CLIENT_SECRET`, `REDIRECT_URI=http://localhost:3000/callback`.

## Next Steps
1. **Self-service password reset** — email-based reset link flow (currently only admins can reset via API).
2. **User account page** — let authenticated users see/revoke their own sessions and tokens (currently admin-only).
3. **Dynamic client registration** (RFC 7591) — self-registration without requiring admin creation.
4. **Scope-based consent** — per-scope consent checkboxes rather than all-or-nothing approval.
