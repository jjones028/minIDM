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
- **Dev Proxy**: Vite dev server proxies `/api` → `http://localhost:8080`. This makes browser requests same-origin so `HttpOnly` cookies flow without CORS credential complexity.

## Current State (as of 2026-05-08, updated 2026-05-08)

### Backend
- **Identity**: Registration (Argon2id hashing), listing, detail retrieval, active session listing + revocation, bootstrap admin creation. First identity → `admin` role; subsequent → `viewer` role.
  - `GET /api/identities/{id}` — identity details (omits `pw_hash`)
  - `GET /api/identities/{id}/sessions` — active sessions `[{handle, created_at, expires_at}]` (token never exposed; handle = first 8 hex chars of SHA-256)
  - `DELETE /api/identities/{id}/sessions/{handle}` — revoke a session by opaque handle (`identity:write`)
  - New queries: `GetIdentityByID`, `ListActiveSessionsByIdentityID`, `DeleteSession`
  - `internal/identity/revoke_identity_session.go`: `RevokeIdentitySessionHandler` — matches handle → deletes session row
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
| `GET /.well-known/openid-configuration` | OIDC discovery document |
| `GET /oauth2/jwks.json` | RSA-2048 public key as JWKS |
| `GET /oauth2/authorize` | Checks session cookie itself; redirects unauthenticated to `/login?next=<url>` |
| `POST /oauth2/token` | Form-encoded; `grant_type=authorization_code` or `refresh_token`; client authenticates via body params |
| `GET /oauth2/userinfo` | Bearer token (RS256 JWT); validates signature + JTI revocation |

#### Admin API (RBAC: `oauth2_client` resource)
`GET/POST /api/oauth2/clients` and `GET/PATCH/DELETE /api/oauth2/clients/{id}`. The `client_secret_hash` field is never exposed; plaintext secret is returned only on creation inside `{ client, client_secret }`.

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
- **Routing** (`web/src/App.tsx`): `ProtectedRoute` + routes for `/`, `/identities/:id`, `/identities/:id/roles`, `/roles`, `/roles/:id/permissions`, `/oauth2/clients`.
- **API client** (`web/src/api.ts`): Axios with `baseURL: '/api'`. Includes `getIdentity`, `getIdentitySessions`, and OAuth2 client CRUD functions.
- **Navigation** (`web/src/components/app-nav.tsx`): Identities | Roles | OAuth2 Clients.
- **AuthPage** (`web/src/pages/AuthPage.tsx`): Reads `?next=` query param; redirects there after login (supports OAuth2 authorize flow).
- **IdentityDetailPage** (`web/src/pages/IdentityDetailPage.tsx`): Shows subject ID, enabled/disabled status, assigned roles (with "Manage Roles" link), and active sessions. Reached via "View" button from the dashboard.
- **OAuthClientsPage** (`web/src/pages/OAuthClientsPage.tsx`): Create/list/edit/delete clients. Shows `client_id` with copy button. Displays client secret in a modal on creation ("shown once").
- **Shadcn components installed**: `button`, `card`, `input`, `table`, `select`. Note: no `badge` component — use inline `<span>` with Tailwind classes for status indicators.

## Critical Files
| File | Purpose |
|------|---------|
| `services/api/internal/app/router.go` | All route registration and middleware chains |
| `services/api/internal/app/server.go` | Startup: DB pool, RSA key load, issuer config |
| `services/api/internal/identity/handler.go` | Identity API — list, get, sessions. Has its own `parseUUID` helper |
| `services/api/internal/rbac/middleware.go` | `Authenticate` + `Require` middleware |
| `services/api/internal/rbac/handler.go` | Role and Permission management API (also owns identity role routes) |
| `services/api/internal/session/handler.go` | Login / logout cookie logic |
| `services/api/internal/oauth2/` | Full OAuth2/OIDC provider package |
| `services/api/db/migrations/` | `001`–`008` (identity, rbac, sessions, builtin, oauth2 ×4) |
| `services/api/db/queries/sessions.sql` | Includes `ListActiveSessionsByIdentityID` |
| `services/api/db/queries/oauth2.sql` | Sqlc source for OAuth2 queries |
| `web/src/context/auth.tsx` | React auth state |
| `web/src/api.ts` | All API calls + TypeScript types |
| `web/src/components/app-nav.tsx` | Main application navigation |
| `web/vite.config.js` | Vite proxy (`/api` → `:8080`) |

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

#### Endpoint
`GET /api/audit-logs?limit=N&offset=N` — protected `audit_log:read`. JSON array ordered by `created_at DESC`.

#### Events
`session.login`, `session.logout`, `identity.register`, `identity.session.revoke`, `role.create/update/delete`, `role.permission.add/remove`, `identity.role.assign/remove`, `oauth2_client.create/update/delete`.

#### Wiring
`audit.Register` is called first in `router.go`; the returned `*Auditor` is passed to every other `Register` function. `session/logout.go` updated to look up session before delete to capture actor ID.

#### Frontend
- `web/src/pages/AuditLogsPage.tsx` — table: timestamp, action badge, resource type, resource ID, actor, details
- `web/src/api.ts` — `AuditLog` type + `listAuditLogs(limit, offset)`
- `web/src/App.tsx` — `/audit-logs` protected route
- `web/src/components/app-nav.tsx` — "Audit Log" nav link

## Next Steps
1. **OAuth2 consent screen**: Show a React UI during the authorize flow listing requested scopes for user approval (currently auto-approved).
3. **OAuth2 nonce support**: Add `nonce TEXT` to `oauth2_authorization_codes`; include in `id_token` JWT. Required by some OIDC client libraries.
4. **Client secret rotation**: `POST /api/oauth2/clients/{id}/rotate-secret`.
5. **Token introspection** (RFC 7662): `POST /oauth2/introspect`.
6. **Token revocation** (RFC 7009): `POST /oauth2/revoke`.
