# Developer Handover: minIDM

## Stack Summary
- **Languages**: Go (Backend), TypeScript (Frontend).
- **Database**: PostgreSQL (Migrations via `goose` in `services/api/db/migrations`).
- **UI**: Tailwind v4 + Base UI (`@base-ui/react`). Component styles follow the `radix-mira` Shadcn/ui theme with OKLCH colors. Do **not** use `npx shadcn add` — it generates Radix-based components. Write new UI primitives manually using `@base-ui/react` (see `web/src/components/ui/` for examples).
- **Build**: `task` (Go-Task). Run `task --list` for available commands.

## Architecture
- **Unified Binary**: In production, the Go server serves the React app from `internal/app/static.go`.
- **Feature Folders**: Go code is organized by feature (e.g., `internal/identity`, `internal/oauth2`) to keep domain logic isolated.
- **RBAC**: Resource-Action model. Actions (`read`, `write`, `delete`) on Resources (`identity`, `role`, `session`, `oauth2_client`).
- **Dev Proxy**: In development, Vite proxies `/api` → `http://localhost:8080`, making all requests same-origin so cookies work without CORS changes.
- **CQRS Pattern**: Each feature has `handler.go` (HTTP wiring + JSON) + per-command/query files. Business rules live in command handlers, never in HTTP handlers.

## Current State (as of 2026-06-17)

### Completed
- **Identity registration & listing** — full CQRS-based flow with Argon2id password hashing.
- **RBAC schema** — `roles`, `resources`, `actions`, `permissions`, `identity_roles` tables (migrations `002` and `004`). Seeded with `admin` (all permissions) and `viewer` (read-only) roles. First registered identity becomes admin; all subsequent get viewer.
- **Session infrastructure** — `sessions` table (migration `003`). Login creates a session row; logout deletes it.
- **Cookie-based sessions** — Login sets an `HttpOnly; SameSite=Strict` cookie named `session`. Set `SECURE_COOKIES=true` env var in production (requires HTTPS). Bearer token / localStorage approach has been fully removed.
- **Authorization middleware** — `internal/rbac/middleware.go`:
  - `Authenticate`: reads `session` cookie, validates against DB, injects identity UUID into request context.
  - `Require(resource, action)`: checks identity has the named permission via `IdentityHasPermission` query.
- **`GET /api/me`** — auth-only endpoint (no permission check) the frontend uses to verify session validity on page load.
- **Role management (CRUD)** — `RolesPage` at `/roles`. Create, list, edit, and delete custom roles. Built-in roles are protected from deletion.
- **Permission management (Matrix)** — `RolePermissionsPage` at `/roles/:id/permissions`. A matrix UI to toggle Resource × Action permissions for a role.
- **Identity Role Assignment** — `IdentityRolesPage` at `/identities/:id/roles`. Shows assigned roles with remove buttons; dropdown to assign available roles.
- **CQRS Refactoring** — The `rbac` and `session` packages have been refactored into clean Command/Query handlers (e.g., `roles.go`, `permissions.go`, `logout.go`). Business logic like "built-in role protection" moved from the HTTP layer into Command handlers.
- **App Navigation** — `AppNav` component provides easy switching between Identities, Roles, and OAuth2 Clients.
- **Auth context** — `web/src/context/auth.tsx`. Calls `/api/me` on mount to determine initial auth state. Exposes `setAuthenticated(bool)` so pages update auth state after login/logout/401 without touching localStorage.
- **Connection pooling** — switched `server.go` from `pgx.Connect` to `pgxpool.New`. Critical fix: concurrent requests (e.g. `Promise.all`) on a single `pgx.Conn` caused sporadic 401s.

### Identity Detail Page (completed 2026-05-08)
Single-identity admin view at `/identities/:id`. Fetches identity details, assigned roles, and active sessions in parallel.

#### New Backend
| File | Responsibility |
|------|----------------|
| `internal/identity/get_identity.go` | `GetIdentityHandler` — wraps `GetIdentityByID` query |
| `internal/identity/list_identity_sessions.go` | `ListIdentitySessionsHandler` — wraps `ListActiveSessionsByIdentityID` query |

New endpoints (both protected: `identity:read`):
- `GET /api/identities/{id}` — returns identity sans `pw_hash` (subject ID, email, enabled, timestamps)
- `GET /api/identities/{id}/sessions` — returns `[{handle, created_at, expires_at}]` for non-expired sessions (token never exposed; handle = first 8 hex chars of SHA-256)

New SQL query in `db/queries/sessions.sql`: `ListActiveSessionsByIdentityID` — filters `expires_at > NOW()` for a given `identity_id`.

#### New Frontend
| File | Change |
|------|--------|
| `web/src/pages/IdentityDetailPage.tsx` | New page — details card, roles card (with "Manage Roles" link), active sessions card |
| `web/src/api.ts` | Added `IdentitySession` type, `getIdentity`, `getIdentitySessions` |
| `web/src/App.tsx` | Added `/identities/:id` protected route (more specific than `/identities/:id/roles`) |
| `web/src/pages/DashboardPage.tsx` | Identity row action changed from "Manage Roles" → "View" linking to detail page |

### Session Revocation (completed 2026-05-08)
Admins can revoke individual active sessions from the Identity Detail page.

#### New Backend
| File | Responsibility |
|------|----------------|
| `internal/identity/revoke_identity_session.go` | `RevokeIdentitySessionHandler` — lists sessions for the identity, matches by handle, calls `DeleteSession` |

New endpoint (protected: `identity:write`):
- `DELETE /api/identities/{id}/sessions/{handle}` — revokes the session matching the opaque handle. Handle = first 8 hex chars of `SHA-256(token)`. Token is never round-tripped.

#### `sessionHandle` helper
`internal/identity/revoke_identity_session.go` exports `sessionHandle(token string) string` (unexported, shared with `list_identity_sessions.go` via same package). The handle is safe to expose — it cannot be used to reconstruct the token.

#### Frontend
Revoke button added to the active sessions table on `IdentityDetailPage`. Calls `DELETE /api/identities/:id/sessions/:handle`. Row is removed from the UI on success.

New API function: `revokeIdentitySession(identityId, handle)` in `web/src/api.ts`.

### OAuth2/OIDC Provider (completed 2026-05-07)
Full authorization code flow with PKCE (S256 only), RS256 JWT signing, and OIDC discovery. Package: `internal/oauth2/`.

#### DB Migrations Added
| Migration | Purpose |
|-----------|---------|
| `005_create_oauth2_clients.sql` | Registered client apps — `client_id` (public), `client_secret_hash` (Argon2id), `redirect_uris TEXT[]`, `scopes TEXT[]` |
| `006_create_oauth2_authorization_codes.sql` | Short-lived (5 min), single-use auth codes. `used BOOL`, `code_challenge`, `code_challenge_method` |
| `007_create_oauth2_tokens.sql` | Token tracking via `jti TEXT UNIQUE` (access) and `refresh_token_hash TEXT` (refresh). `revoked BOOL` |
| `008_seed_oauth2_rbac.sql` | Inserts `oauth2_client` resource; grants admin role all permissions on it |

#### Package Structure: `internal/oauth2/`
| File | Responsibility |
|------|----------------|
| `keys.go` | `LoadOrGenerateRSAKey(path)` — loads PEM or generates 2048-bit RSA; `KeyID(key)` — SHA-256 fingerprint as hex kid |
| `crypto.go` | Argon2id for client secrets; SHA-256 for refresh tokens; S256 PKCE `ComputeCodeChallenge`; `GenerateClientID/Secret/AuthCode/RefreshToken` |
| `clients.go` | CQRS: `CreateClientHandler` (generates client_id + secret server-side), `ListClientsHandler`, `GetClientHandler`, `UpdateClientHandler`, `DeleteClientHandler` |
| `authorize.go` | `GET /oauth2/authorize` — validates PKCE+client+scope, checks session cookie. Unauthenticated → redirect to `/login?next=<original URL>` |
| `token.go` | `POST /oauth2/token` — `authorization_code` grant (validates code + PKCE verifier + client secret) and `refresh_token` grant (rotation: revoke old, issue new). Emits RS256 JWTs |
| `userinfo.go` | `GET /oauth2/userinfo` — parses + validates Bearer JWT, checks JTI against `oauth2_tokens` (revocation), returns `{sub, email}` |
| `discovery.go` | `GET /.well-known/openid-configuration` — static OIDC discovery JSON |
| `jwks.go` | `GET /oauth2/jwks.json` — RSA public key as JWKS (RFC 7517). Encodes modulus `n` and exponent `e` as base64url |
| `handler.go` | `Register(mux, q, key, issuer, protectRead, protectWrite)` wires all routes. Admin API never returns `client_secret_hash` |

#### Endpoints
| Method | Path | Auth | Notes |
|--------|------|------|-------|
| `GET` | `/.well-known/openid-configuration` | Public | OIDC discovery |
| `GET` | `/oauth2/jwks.json` | Public | RSA public key |
| `GET` | `/oauth2/authorize` | Session cookie (self-checked) | Redirects unauthenticated users to `/login?next=` |
| `POST` | `/oauth2/token` | Client credentials in body | `grant_type`: `authorization_code` or `refresh_token` |
| `GET` | `/oauth2/userinfo` | Bearer JWT | Returns `sub`, `email` |
| `GET/POST` | `/api/oauth2/clients` | RBAC `oauth2_client:read/write` | List / create |
| `GET/PATCH/DELETE` | `/api/oauth2/clients/{id}` | RBAC `oauth2_client:read/write` | Get / update / delete |

#### Key Design Decisions
- **PKCE is mandatory** — `code_challenge_method=S256` required. `plain` rejected.
- **Tokens**: access token = RS256 JWT with `jti`. `jti` stored in `oauth2_tokens` for revocation lookup. Refresh token = opaque random, stored as SHA-256 hash.
- **`id_token`** = same JWT as `access_token` (payload satisfies OIDC id_token spec for simple deployments). Included in token response when `openid` scope is requested.
- **Refresh token rotation**: on every refresh grant, old token row is set `revoked=TRUE` and a new token row is created.
- **Client secret**: generated server-side (32 random bytes, base64url), hashed Argon2id, returned plaintext once on creation only.
- **Consent**: auto-approved — all scopes the client is allowed for are immediately granted.
- **Authorization code TTL**: 5 minutes, single-use via DB `used` flag.
- **Access token TTL**: 1 hour.
- **Refresh token TTL**: 30 days (stored in `expires_at` column).

#### New Environment Variables
| Variable | Default | Purpose |
|----------|---------|---------|
| `OAUTH2_KEY_PATH` | `oauth2_signing.key` | PEM RSA private key file. Auto-generated on first run if missing |
| `OAUTH2_ISSUER` | `http://localhost:8080` | `iss` claim in JWTs + discovery doc. Must be externally reachable in prod |

#### `NewHandler` signature changed
`router.go` now: `NewHandler(queries *db.Queries, signingKey *rsa.PrivateKey, issuer string) http.Handler`

#### Frontend Changes
| File | Change |
|------|--------|
| `web/src/pages/OAuthClientsPage.tsx` | New page — register/edit/delete clients, scopes checkboxes, one-per-line redirect URIs, modal showing client secret once |
| `web/src/api.ts` | Added `OAuthClient`, `CreateOAuthClientResult` types + 5 API functions |
| `web/src/App.tsx` | Added `/oauth2/clients` protected route |
| `web/src/components/app-nav.tsx` | Added "OAuth2 Clients" nav link |
| `web/src/pages/AuthPage.tsx` | Now reads `?next=` query param; redirects there after successful login |

### Known Architecture Notes
- `internal/rbac/handler.go` contains all role and permission management API routes.
- `internal/app/router.go` is the single place where all routes and protection chains are assembled.
- `web/src/api.ts` — all API functions. No token management; cookies are handled transparently by the browser.
- `golang-jwt/jwt/v5` is a direct dependency (promoted from indirect during OAuth2 implementation).
- The CORS middleware in `router.go` adds `Authorization` to `Access-Control-Allow-Headers` (needed for `userinfo` Bearer token calls from browser clients).
- `GET /api/identities/{id}` and `GET /api/identities/{id}/sessions` are registered in `internal/identity/handler.go` (not `rbac/handler.go`). The identity package now has its own `parseUUID` helper.

### Audit Logging (completed 2026-05-08)
Tracks administrative actions across identities, roles, permissions, sessions, and OAuth2 clients. Viewable at `/audit-logs` (admin only).

#### DB Migrations Added
| Migration | Purpose |
|-----------|---------|
| `009_create_audit_logs.sql` | `audit_logs` table — `id`, `actor_id UUID` (nullable FK → identities), `action TEXT`, `resource_type TEXT`, `resource_id TEXT` (nullable), `details JSONB` (nullable), `created_at TIMESTAMPTZ` |
| `010_seed_audit_rbac.sql` | Inserts `audit_log` resource; grants admin `read` permission on it |

#### Package: `internal/audit/`
| File | Responsibility |
|------|----------------|
| `auditor.go` | `Auditor` — `Log(ctx, actorID, action, resourceType, resourceID, details)`. Fire-and-forget; errors silently dropped. `UUIDStr(pgtype.UUID) string` helper |
| `list_events.go` | `ListEventsHandler` — wraps `ListAuditLogs` query with limit/offset |
| `handler.go` | `Register(mux, q, protectRead) *Auditor` — wires `GET /api/audit-logs`, returns the shared `Auditor` |

#### Endpoint
- `GET /api/audit-logs?limit=N&offset=N` — protected by `audit_log:read`. Returns `[]AuditLog` ordered by `created_at DESC`.

#### Events Logged
| Action | Trigger |
|--------|---------|
| `session.login` | Successful login |
| `session.logout` | Logout |
| `identity.register` | New identity registered (no actor) |
| `identity.session.revoke` | Admin revokes a session |
| `role.create` / `.update` / `.delete` | Role CRUD |
| `role.permission.add` / `.remove` | Permission matrix changes |
| `identity.role.assign` / `.remove` | Role assignment changes |
| `oauth2_client.create` / `.update` / `.delete` | OAuth2 client CRUD |

#### Integration Pattern
`audit.Register(...)` is called first in `router.go` and returns the `*Auditor`. The auditor is then passed as a dependency to `session.Register`, `identity.Register`, `rbac.RegisterRoleRoutes`, and `oauth2.Register`. Each package calls `a.auditor.Log(...)` after successful write operations. `session/logout.go` was updated to look up the session before deletion so the identity ID is available for the log.

#### Frontend
| File | Change |
|------|--------|
| `web/src/pages/AuditLogsPage.tsx` | New page — table of recent events with timestamp, action, resource type, resource ID, actor, and details |
| `web/src/api.ts` | Added `AuditLog` type + `listAuditLogs` function |
| `web/src/App.tsx` | Added `/audit-logs` protected route |
| `web/src/components/app-nav.tsx` | Added "Audit Log" nav link |

#### Known Architecture Notes
- `internal/identity/handler.go` now imports `minIDM/internal/rbac` (for `IdentityFromContext`) and `minIDM/internal/audit`. No import cycle: rbac does not import identity.
- `internal/oauth2/handler.go` now imports `minIDM/internal/rbac` and `minIDM/internal/audit`.
- `session/logout.go` now calls `GetSessionByToken` before `DeleteSession` to capture the identity ID.

### OAuth2 Consent Screen + Auto-Consent (completed 2026-05-08)
Adds an explicit consent step to the authorization code flow. Admins can enable per-client auto-consent to skip the screen for trusted first-party apps.

#### DB Migration
| Migration | Purpose |
|-----------|---------|
| `011_add_auto_consent.sql` | `ALTER TABLE oauth2_clients ADD COLUMN auto_consent BOOLEAN NOT NULL DEFAULT FALSE` |

#### New Endpoints
| Method | Path | Auth | Notes |
|--------|------|------|-------|
| `GET` | `/api/oauth2/client-info?client_id=X` | Public | Returns `{name, description, scopes, auto_consent}` for the consent page display |
| `POST` | `/api/oauth2/consent` | Session cookie | Re-validates all authorize params, issues auth code, returns `{redirect_url}` |

#### Updated Authorize Flow (`internal/oauth2/authorize.go`)
After full validation (client, redirect_uri, PKCE, scopes, session), if `client.AutoConsent = false` the handler redirects to `/oauth2/consent?{all validated params}` instead of issuing the code immediately. Auto-consent clients proceed as before.

#### New Files
| File | Responsibility |
|------|----------------|
| `internal/oauth2/consent.go` | `ConsentHandler` — re-validates all params + session, issues auth code, returns JSON `{redirect_url}` |
| `internal/oauth2/client_info.go` | `ClientInfoHandler` — public endpoint returning non-sensitive client display info |

#### `oauth2.Register` signature change
Now accepts an additional `authenticate func(http.Handler) http.Handler` param (used to gate `POST /api/oauth2/consent`).

#### Frontend
| File | Change |
|------|--------|
| `web/src/pages/ConsentPage.tsx` | New page at `/oauth2/consent` — fetches client info, lists requested scopes with human-readable labels, Approve (→ POST consent) / Deny (→ redirect with `access_denied`) |
| `web/src/pages/OAuthClientsPage.tsx` | Edit form gains vertical "Scopes" + "Settings" checkbox sections with descriptions; Auto-consent toggle; display row shows blue `auto-consent` badge |
| `web/src/api.ts` | Added `auto_consent` to `OAuthClient`, `UpdateOAuthClientData`; added `ClientInfo`, `ConsentParams` types; `getClientInfo`, `approveConsent` functions |
| `web/src/App.tsx` | Added `/oauth2/consent` route (no ProtectedRoute wrapper — page handles auth itself) |

#### Key Design Decisions
- **No pending-consent DB table**: validated params are passed as query params to the React consent page and re-validated on `POST /api/oauth2/consent`. Safe because PKCE ensures the code is only useful to whoever holds the code_verifier.
- **Auto-consent default is `false`**: all existing and new clients require explicit consent unless an admin enables the flag.
- **Consent POST is session-gated**: `rbac.Authenticate` middleware is applied; unauthenticated POSTs get 401.

### OAuth2 Nonce Support (completed 2026-05-08)
`nonce` parameter flows from the authorization request through to the `id_token` JWT claim, satisfying OIDC libraries that validate nonce (e.g. PKCE + nonce combined).

#### DB Migration
| Migration | Purpose |
|-----------|---------|
| `012_add_nonce_to_auth_codes.sql` | `ALTER TABLE oauth2_authorization_codes ADD COLUMN nonce TEXT` (nullable) |

#### Changes
- **`authorize.go`**: reads `nonce` query param; passes to consent redirect URL if present; passes to `CreateAuthorizationCode` as `pgtype.Text`
- **`consent.go`**: reads `nonce` from JSON body; passes to `CreateAuthorizationCode`
- **`token.go`**: `tokenClaims` gains `Nonce string \`json:"nonce,omitempty"\``; `issueAndRespond` accepts `nonce string` param; sets `claims.Nonce` when `openid` scope is present and nonce is non-empty; refresh token grant passes `""` (nonce is not round-tripped on refresh)
- **`ConsentPage.tsx`**: reads `nonce` URL param, forwards it in the `approveConsent` POST body
- **`api.ts`**: `ConsentParams` gains `nonce?: string`

#### Key Design Decisions
- Nonce is only included in `id_token` when the `openid` scope is requested (per OIDC spec).
- Refresh token grant does not re-include the original nonce — nonce is a one-time binding per auth code exchange.
- Nonce is stored as a nullable `TEXT` column so existing auth codes are unaffected by the migration.

### Client Secret Rotation (completed 2026-05-19)
Admin endpoint to regenerate a client's secret without invalidating existing tokens.

New endpoint (protected: `oauth2_client:write`):
- `POST /api/oauth2/clients/{id}/rotate-secret` — returns `{ client_secret }` once. Returns 422 for public clients.

| File | Responsibility |
|------|----------------|
| `internal/oauth2/rotate_secret.go` | `RotateSecretHandler` — fetches client, returns `ErrPublicClient` if no hash, generates new secret, re-hashes Argon2id, updates DB |

Key design: `ErrPublicClient` sentinel in `clients.go`; does not revoke existing tokens (access tokens verified by RSA key, not client secret). Frontend hides "Rotate Secret" button for public clients.

### Token Introspection (RFC 7662) (completed 2026-05-19)
Lets resource servers validate access tokens without parsing JWTs themselves.

- `POST /oauth2/introspect` — form-encoded: `token`, `client_id`, `client_secret`. Returns `{active: true, ...claims}` or `{active: false}`.

| File | Responsibility |
|------|----------------|
| `internal/oauth2/introspect.go` | `IntrospectHandler` — authenticates client via `authenticateClient()`, parses JWT, checks JTI revocation |

Key design: client auth failures return 401 (not `active:false`) per RFC 7662 §2.1. Response includes `jti`, `iss`, `sub`, `client_id`, `scope`, `token_type`, `exp`, `iat`, `aud`. `Cache-Control: no-store` set per RFC.

### Token Revocation (RFC 7009) (completed 2026-05-19)
Lets clients explicitly invalidate tokens.

- `POST /oauth2/revoke` — form-encoded: `token`, `token_type_hint`, `client_id`, `client_secret`. Always returns 200.

| File | Responsibility |
|------|----------------|
| `internal/oauth2/revoke.go` | `RevokeHandler` — authenticates client, tries access token (JTI lookup), then refresh token (hash lookup), marks `revoked=TRUE` |

Key design: always returns 200 per RFC 7009 §2.2. Public clients authenticate with `client_id` only.

### Token Admin UI (completed 2026-05-19)
Admins can list, inspect, and revoke active OAuth2 tokens from the admin portal.

#### New Backend
| File | Responsibility |
|------|----------------|
| `internal/oauth2/list_tokens.go` | `ListTokensHandler` — lists all non-revoked, non-expired tokens |
| `internal/oauth2/inspect_token.go` | `InspectTokenHandler` — parses and validates a raw JWT, returns decoded claims + DB status |

New endpoints (protected: `oauth2_client:read/write`):
- `GET /api/oauth2/tokens` — list active tokens
- `DELETE /api/oauth2/tokens/{id}` — admin-revoke a token by DB UUID (inline in `handler.go`)
- `POST /api/oauth2/tokens/inspect` — decode + validate a raw token string

#### Frontend
| File | Change |
|------|--------|
| `web/src/pages/TokensPage.tsx` | New page — list table with revoke buttons; raw token inspect form showing decoded claims |
| `web/src/App.tsx` | Added `/oauth2/tokens` protected route |
| `web/src/components/app-nav.tsx` | Added "Tokens" nav link |

### Identity Enable/Disable (completed 2026-05-19)
Admins can enable or disable an identity from the Identity Detail page.

| File | Responsibility |
|------|----------------|
| `internal/identity/set_enabled.go` | `SetEnabledHandler` — updates `is_enabled` flag, returns updated row |

New endpoint (protected: `identity:write`):
- `PATCH /api/identities/{id}/enabled` — body: `{ enabled: bool }`. Returns `{ id, is_enabled, updated_at }`. Logs `identity.enable` or `identity.disable` audit event.

Disabled identities cannot log in — session creation checks `is_enabled`. Frontend: toggle switch on `IdentityDetailPage`.

### Identity Password Reset (completed 2026-05-19)
Admins can reset any identity's password from the Identity Detail page.

| File | Responsibility |
|------|----------------|
| `internal/identity/reset_password.go` | `ResetPasswordHandler` — validates new password (min 8 chars), re-hashes Argon2id, updates `pw_hash` |

New endpoint (protected: `identity:write`):
- `POST /api/identities/{id}/reset-password` — body: `{ password: string }`. Returns 204. Logs `identity.password.reset` audit event.

Frontend: "Reset Password" form on `IdentityDetailPage`.

### Audit Log Improvements (completed 2026-05-19)
Upgraded from a simple paginated list to a filterable view with actor email and total count for pagination.

#### DB / SQL Changes
Queries in `db/queries/audit.sql` replaced with filtered variants:
- `ListAuditLogsFiltered` — LEFT JOINs `identities` for `actor_email`; filter params `$1::text` (resource_type), `$2::text` (action LIKE prefix), `$3::uuid` (actor_id), `$4::timestamptz` (since), `$5::timestamptz` (until)
- `CountAuditLogsFiltered` — same filter params, returns `int64`
- `ListDistinctAuditResourceTypes` — distinct `resource_type` values for dropdown

Note: sqlc generates unnamed `$N::type` casts as `Column1`–`Column5` in the params struct.

#### Backend Changes
- `list_events.go` → `ListEventsFilter` struct; `ListEventsResult{Total int64, Rows []db.ListAuditLogsFilteredRow}`; action prefix appended with `%` for LIKE
- `handler.go` → `GET /api/audit-logs/resource-types` endpoint added; all filter params parsed from query string (RFC3339 since/until); response is `{"total": N, "logs": [...]}` (not bare array); default page size 50

#### Frontend Changes
| File | Change |
|------|--------|
| `web/src/pages/AuditLogsPage.tsx` | Filter bar (resource type dropdown, action prefix input, actor UUID input, since/until date pickers), pagination controls, actor email display |
| `web/src/api.ts` | `AuditLog` gains `actor_email: string \| null`; `listAuditLogs` returns `AuditLogsResponse{total, logs}`; added `listAuditResourceTypes()` |

### Public Client Support (completed 2026-05-19)
Clients without a client secret for native/mobile apps (RFC 8252). PKCE remains mandatory for all clients.

#### DB Migration
`013_public_client_support.sql` — `ALTER TABLE oauth2_clients ALTER COLUMN client_secret_hash DROP NOT NULL`

Identification: `client_secret_hash IS NULL` → public client. No separate boolean column needed.

#### Backend Changes
| File | Change |
|------|--------|
| `internal/oauth2/crypto.go` | Added `authenticateClient(ctx, q, clientID, clientSecret)` helper — accepts `client_id` only for public clients |
| `internal/oauth2/token.go` | Replaced inline client auth with `authenticateClient()` |
| `internal/oauth2/introspect.go` | Replaced inline client auth with `authenticateClient()` |
| `internal/oauth2/revoke.go` | Replaced inline client auth with `authenticateClient()` |
| `internal/oauth2/clients.go` | `CreateClientCommand.IsPublic bool`; `ErrPublicClient` sentinel |
| `internal/oauth2/handler.go` | `clientResponse.IsPublic bool`; `CreateClient` passes `is_public`; `RotateSecret` returns 422 for public clients |

#### Frontend Changes
- Create form gains public client checkbox with description.
- Amber `public` badge in client list for public clients.
- "Rotate Secret" button hidden for public clients.
- `OAuthClient.is_public` and `CreateOAuthClientData.is_public` in `api.ts`.

### Docker Compose + Production Binary (completed 2026-05-19)
Production deployment via Docker Compose with a standalone embedded migration runner.

#### `compose.yaml`
Three services:
1. **`db`** — PostgreSQL 18 with `pg_isready` health check
2. **`migrate`** — `entrypoint: ["/app/migrate"]`; `depends_on: db: condition: service_healthy`; `restart: "no"`
3. **`app`** — port 8080; `depends_on: migrate: condition: service_completed_successfully`; named volume `app_keys` at `/app/keys/`

Named volumes: `idm_database` (Postgres data), `app_keys` (RSA signing key persists across restarts).

**Important**: use `entrypoint:` in compose, NOT `command:`. The Dockerfile sets `ENTRYPOINT ["/app/minIDM"]`; `command:` is appended as args to that entrypoint, not a replacement.

#### Standalone Migrate Binary (`services/api/cmd/migrate/main.go`)
Embeds all migrations via `embed.FS` and runs `goose.Up` using the Go API. Completely independent of dev mode.

```go
// services/api/db/migrations/embed.go
package migrations
import "embed"
//go:embed *.sql
var SQL embed.FS
```

Dev mode (`task dev`) continues using Testcontainers + goose CLI via `exec.Command` — unaffected.

#### Dockerfile
Backend build stage compiles both `/app/minIDM` (server) and `/app/migrate` (migrator). Both copied to distroless final image.

### Dev Proxy and AuthPage Fix (completed 2026-05-19)

#### Vite Proxy (`vite.config.js`)
Added `/oauth2` and `/.well-known` proxy rules so OAuth2 protocol endpoints route through the Vite dev server:
```javascript
proxy: {
  '/api':         'http://localhost:8080',
  '/oauth2':      'http://localhost:8080',
  '/.well-known': 'http://localhost:8080',
}
```
Required so that `GET /oauth2/authorize` redirects (which use relative URLs like `/login?next=...`) resolve on port 5173 where the SPA runs, not port 8080.

#### AuthPage (`web/src/pages/AuthPage.tsx`)
Post-login redirect uses `window.location.href = next` instead of React Router `navigate(next)`. `navigate()` is SPA-only and cannot trigger backend routes or full-page navigation.

### OIDC Test Client (`~/Projects/oidc-test-client`) (completed 2026-05-19)
Reference test application for end-to-end OIDC flow testing against minIDM.

**Stack**: Node.js, Express, TypeScript (ESM/NodeNext), `express-session`

Tokens stored server-side in Express session (in-memory); browser only sees the opaque `connect.sid` cookie.

| Route | Purpose |
|-------|---------|
| `/` | Home — shows token state and user info |
| `/login` | Initiates PKCE flow; redirects to `/oauth2/authorize` |
| `/callback` | Exchanges auth code for tokens; stores in session |
| `/userinfo` | Calls userinfo endpoint with Bearer token |
| `/introspect` | Calls introspect endpoint |
| `/refresh` | Refreshes access token; stores rotated refresh token |
| `/revoke` | Revokes refresh token |
| `/logout` | Clears session |

**Discovery URL rewriting**: The client rewrites all endpoints from the discovery document to replace the server's internal issuer (e.g., `http://localhost:8080`) with the configured `ISSUER` (`http://localhost:5173`), routing everything through Vite's proxy so `HttpOnly` session cookies flow without CORS issues.

**SSO note**: `localhost` cookies are port-agnostic. The minIDM session cookie (set at port 5173/8080 for `localhost`) is automatically sent to the test client on port 3000, enabling SSO — the second app never prompts for login if the user is already authenticated in minIDM.

#### Configuration (`.env`)
```
ISSUER=http://localhost:5173
CLIENT_ID=<from minIDM>
CLIENT_SECRET=<from minIDM>   # omit for public clients
REDIRECT_URI=http://localhost:3000/callback
PORT=3000
```

### Security Hardening — Critical Fixes (completed 2026-05-30)

#### Session token hashing (migration `016`)
`sessions.token` was stored as plaintext — a DB breach would have exposed every live session. Fixed in migration `016_hash_session_tokens.sql`:
- `ALTER TABLE sessions RENAME COLUMN token TO token_hash` — self-documenting schema
- `014_hash_session_tokens.sql` (remote, runs first) truncates all pre-existing plaintext sessions
- `login.go` stores `SHA-256(token)` via `hashSessionToken()`; cookie still carries plaintext
- `logout.go` and `rbac/middleware.go` hash the cookie value before every DB lookup/delete
- `identity/revoke_identity_session.go`: `sessionHandle()` simplified — stored value is the full 64-char hex SHA-256, so handle = `tokenHash[:8]` (no re-hash needed)
- sqlc regenerated: `Session.Token` → `Session.TokenHash`, `CreateSessionParams.Token` → `CreateSessionParams.TokenHash`

#### Disabled identity enforcement
Disabled accounts could previously log in and exchange OAuth2 tokens:
- `session/login.go` — returns `ErrAccountNotActive` (→ 403 `account_not_active`) before password verification
- `oauth2/token.go` — checks `identity.IsEnabled` in both `authorization_code` and `refresh_token` grant handlers; returns `access_denied 403`
- `oauth2/userinfo.go` — checks `identity.IsEnabled`; returns `invalid_token 401`
- `rbac/middleware.go` — `Authenticate` fetches the identity after session lookup and returns 401 if disabled

#### DOKS deployment
- `k8s/deployment.yaml` — Deployment + Service, Postgres StatefulSet, Traefik IngressRoutes (HTTP→HTTPS + TLS)
- `k8s/migrate-job.yaml` — one-off goose migration Job (triggered via GitHub Actions `migrate.yml`)
- `k8s/secrets.example.sh`, `k8s/traefik-values.yaml` — cluster setup helpers
- `Dockerfile` — `migrator` stage (goose + migration files) pushed as `minidm-migrate:latest`
- `.github/workflows/ci.yml` — PR type-check + build
- `.github/workflows/deploy.yml` — push-to-main: build both images, kubectl rollout, auto-rollback on failure
- `.github/workflows/migrate.yml` — manual `workflow_dispatch` to apply migration Job and stream logs
- `DEPLOY.md` — step-by-step DOKS setup guide

### Per-Client OAuth2 Roles & Groups (completed 2026-06-17)
Per-client flat roles and flat groups as an assignment mechanism. Roles are emitted as a `roles` JWT claim in issued access/ID tokens.

#### Design Decisions
- **Flat roles** — no role hierarchy. Each client defines its own named roles (`oauth2_client_roles`).
- **Flat groups** — assignment mechanism only, no nesting. Groups assign multiple identities to multiple roles in bulk (`oauth2_client_groups`, `client_group_roles`).
- **`roles` claim** — always included in the token when non-empty; no scope gate (matches Azure AD / Keycloak defaults). Absence of roles = claim omitted.
- **Effective roles** = direct assignments UNION via-group assignments (`GetEffectiveClientRolesForIdentity` SQL query using `DISTINCT … UNION`).

#### DB Migration
`017_client_roles_groups.sql` — five tables: `oauth2_client_roles`, `identity_client_roles`, `oauth2_client_groups`, `identity_client_groups`, `client_group_roles`.

#### sqlc Note
`GetEffectiveClientRolesForIdentity` uses `$1::uuid` / `$2::uuid` casts, which sqlc generates as `Column1`/`Column2` in the params struct (not named params). Other queries use plain `$N` and get named params.

#### New Backend Files
| File | Responsibility |
|------|----------------|
| `services/api/db/migrations/017_client_roles_groups.sql` | Schema migration |
| `services/api/db/queries/client_roles.sql` | All role/group CRUD + membership + effective-roles queries |
| `services/api/db/sqlc/client_roles.sql.go` | Generated sqlc code |
| `services/api/internal/oauth2/client_roles.go` | Handlers: list/create/update/delete roles; assign/remove identities; assign/remove groups |
| `services/api/internal/oauth2/client_groups.go` | Handlers: list/create/update/delete groups; add/remove members; add/remove roles |
| `services/api/internal/identity/list_client_roles.go` | `ListIdentityClientRoles`, `RemoveIdentityClientRole`, `ListIdentityClientGroups`, `RemoveIdentityClientGroup` — identity-centric view |

#### Backend Changes
- `internal/oauth2/token.go` — `tokenClaims` gains `Roles []string \`json:"roles,omitempty"\``; `issueAndRespond` calls `GetEffectiveClientRolesForIdentity` and sets claim when non-empty.
- `internal/oauth2/handler.go` — 22 new routes under `/api/oauth2/clients/{id}/roles` and `/api/oauth2/clients/{id}/groups`.
- `internal/identity/handler.go` — 4 new routes: `GET/DELETE /api/identities/{id}/client-roles`, `GET/DELETE /api/identities/{id}/client-groups`.

#### New Frontend
| File | Change |
|------|--------|
| `web/src/pages/ClientDetailPage.tsx` | New page at `/oauth2/clients/:id` — Roles tab + Groups tab, each with create card and expandable rows (members/roles columns) |
| `web/src/pages/IdentityDetailPage.tsx` | Added "Client Roles" card — shows direct role assignments (blue badge) and group memberships (purple badge), both removable |
| `web/src/pages/OAuthClientsPage.tsx` | Added "Roles & Groups" link per client row linking to `ClientDetailPage` |
| `web/src/api.ts` | Added `ClientRole`, `ClientGroup`, `RoleMember`, `RoleGroup`, `ClientRoleAssignment`, `ClientGroupMembership` types + ~20 API functions |
| `web/src/App.tsx` | Added `/oauth2/clients/:id` protected route |

### Dialog-Based Create Forms (completed 2026-06-17)
Replaced inline create-form cards with modal dialogs on three listing pages to reduce visual clutter.

#### New UI Component
`web/src/components/ui/dialog.tsx` — Base UI `@base-ui/react/dialog` wrapper. Exports: `Dialog`, `DialogTrigger`, `DialogClose`, `DialogContent`, `DialogHeader`, `DialogTitle`, `DialogDescription`. Uses `data-open:` / `data-closed:` Tailwind v4 animation attributes (same pattern as `select.tsx`).

#### Pages Updated
| Page | Change |
|------|--------|
| `DashboardPage` | "Create Identity" card removed; "New Identity" button in Identity Registry card header opens dialog |
| `RolesPage` | "Create Role" card removed; "New Role" button in Roles card header opens dialog |
| `OAuthClientsPage` | "Register OAuth2 Client" card removed; "New Client" button in card header opens create dialog. Inline row edit form also moved to a separate edit dialog |

## Next Steps for the Next AI

### Features
1. **Self-service password reset** — email-based reset link flow (requires email delivery). Currently only admins can reset passwords via `POST /api/identities/{id}/reset-password`.
2. **Refresh token revocation in user-facing UI** — users can see and revoke their own active sessions from a "My Account" page (currently only admins can do this for other identities).
3. **Dynamic client registration** (RFC 7591) — allow clients to self-register rather than requiring admin creation.
4. **Scope-based consent** — currently consent grants all client scopes; could show per-scope checkboxes and issue codes only for approved scopes.

### Security Hardening (remaining — do these before shipping to production)

#### High — missing security controls

8. **Rate limiting on authentication endpoints**: `POST /api/login`, `POST /api/register`, and `POST /oauth2/token` have no rate limiting — all three are brute-forceable. Use `golang.org/x/time/rate` to apply per-IP token-bucket limiting in `session/handler.go` and `oauth2/handler.go`. Suggested limits: login 5 req/min, register 3 req/min, token 10 req/min. Return `429 Too Many Requests` with `Retry-After`.

9. **Gate open registration** (`internal/identity/handler.go:Register`): `POST /api/register` is a fully public endpoint — anyone can self-register and becomes `viewer`. For a production IDM this is almost always wrong. Implement an admin-issued invitation flow: admin calls `POST /api/invitations` → gets a short-lived signed token → invitee POSTs to `/api/register` with the token. Alternatively, add a `REGISTRATION_ENABLED` env var (default `false`) that must be explicitly set to allow open registration.

10. **Add security response headers** (`internal/app/router.go`): The server emits no security headers. Add a `securityHeadersMiddleware` in `router.go` (applied before `corsMiddleware`) that sets:
    - `X-Content-Type-Options: nosniff`
    - `X-Frame-Options: DENY`
    - `Referrer-Policy: strict-origin-when-cross-origin`
    - `Permissions-Policy: geolocation=(), camera=(), microphone=()`
    - `Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'` (tighten after verifying no inline scripts)
    - `Strict-Transport-Security: max-age=63072000; includeSubDomains` (only when `SECURE_COOKIES=true`)

11. **Lock down CORS** (`internal/app/router.go:corsMiddleware`): `Access-Control-Allow-Origin: *` is hardcoded. Add an `ALLOWED_ORIGINS` env var (comma-separated). In `corsMiddleware`, reflect the request `Origin` header only if it appears in the allowlist, otherwise omit the CORS header entirely. The wildcard prevents cookies from being sent cross-origin anyway (browser blocks it), but locking it down is defense in depth and required for `credentials: 'include'` flows.

#### Medium — hardening improvements

12. **Password max-length guard** (`internal/identity/handler.go:Register`): The only password check is `len(req.Password) < 8`. Argon2id has no input length limit — a request with a 1 MB password string will consume significant CPU. Add `if len(req.Password) > 128 { http.Error(..., 422) }` before the `< 8` check. 128 chars is a reasonable upper bound for any legitimate user.

13. **Scope re-validation on refresh token grant** (`internal/oauth2/token.go:handleRefreshToken`): When a refresh token is exchanged, `stored.Scopes` are carried forward to the new access token without checking whether the client still has those scopes. If an admin narrows a client's `scopes` after token issuance, old refresh tokens continue producing over-privileged access tokens until they expire (30 days). Fix: after fetching the stored token, intersect `stored.Scopes` with `client.Scopes` and use the intersection for the new token.

14. **Track failed login attempts / account lockout** (`internal/session/login.go`): There is no mechanism to detect or respond to credential-stuffing. Add a `login_attempts` table (`identity_id`, `attempted_at`, `success BOOL`) and lock the account (or impose a delay) after N failures within a window. Alternatively, use an in-memory per-email token bucket as a simpler first step. Log failures to the audit log under a new `session.login_failed` action.

#### Low — compliance and operational

15. **Audit log stderr fallback** (`internal/audit/auditor.go`): `_ = a.q.CreateAuditLog(...)` silently drops all write failures. For a compliance-critical IDM, at minimum log failures to `log.Printf("audit: failed to write event: %v", err)` so they appear in container logs.

16. **PII in audit log details**: `session.login` logs `{"email": req.Email}` and `identity.register` logs the same. Under GDPR/CCPA the email address in an audit detail field requires the same retention controls as the identity itself. Consider logging only the identity UUID and omitting the email from `details`, since the email is already recoverable from the identity record.

17. **`state` parameter enforcement in authorize flow** (`internal/oauth2/authorize.go`): `state` is read but not required. OIDC clients that omit `state` lose their CSRF protection for the redirect. Log a warning, or (better) return `invalid_request` if `state` is absent, to enforce best practice for all downstream clients.

## CQRS Pattern (for new features)
Every new feature should follow this shape:

```
internal/<feature>/
  handler.go         ← Register(mux, ...) + HTTP methods (JSON encode/decode only)
  <command_name>.go  ← XxxHandler + XxxCommand struct + Handle(ctx, cmd) → (result, error)
  <query_name>.go    ← XxxHandler + Handle(ctx, ...) → (result, error)
```

Rules:
- HTTP handlers only: parse → call handler → encode. No SQL, no business logic.
- Business rules (protection, validation, generation) go in command handlers.
- Use `pgtype.UUID` for all UUIDs; parse with `id.Scan(stringValue)`.
- Errors from command handlers bubble up; HTTP handler maps them to status codes.

## Sqlc Workflow
1. Write migration SQL in `db/migrations/<NNN>_<name>.sql` (use `-- +goose Up` / `-- +goose Down`)
2. Write queries in `db/queries/<feature>.sql` with `-- name: XxxFn :one/:many/:exec`
3. Run `task gen` → runs `sqlc generate` + `task db:migrate`
4. Generated code in `db/sqlc/` — never edit manually

Key sqlc type mappings (pgx/v5):
| SQL | Go |
|-----|----|
| `UUID` | `pgtype.UUID` |
| `TEXT` nullable | `pgtype.Text` |
| `TEXT NOT NULL` | `string` |
| `TEXT[]` | `[]string` |
| `BOOLEAN NOT NULL` | `bool` |
| `TIMESTAMPTZ NOT NULL` | `pgtype.Timestamptz` |
| `TIMESTAMPTZ` nullable | `pgtype.Timestamptz` |
