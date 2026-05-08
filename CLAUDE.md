# Developer Handover: minIDM

## Stack Summary
- **Languages**: Go (Backend), TypeScript (Frontend).
- **Database**: PostgreSQL (Migrations via `goose` in `services/api/db/migrations`).
- **UI**: Tailwind v4 + Shadcn/ui. Style is `radix-mira` with OKLCH colors.
- **Build**: `task` (Go-Task). Run `task --list` for available commands.

## Architecture
- **Unified Binary**: In production, the Go server serves the React app from `internal/app/static.go`.
- **Feature Folders**: Go code is organized by feature (e.g., `internal/identity`, `internal/oauth2`) to keep domain logic isolated.
- **RBAC**: Resource-Action model. Actions (`read`, `write`, `delete`) on Resources (`identity`, `role`, `session`, `oauth2_client`).
- **Dev Proxy**: In development, Vite proxies `/api` → `http://localhost:8080`, making all requests same-origin so cookies work without CORS changes.
- **CQRS Pattern**: Each feature has `handler.go` (HTTP wiring + JSON) + per-command/query files. Business rules live in command handlers, never in HTTP handlers.

## Current State (as of 2026-05-08)

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
- `GET /api/identities/{id}/sessions` — returns `[{created_at, expires_at}]` for non-expired sessions (token never exposed)

New SQL query in `db/queries/sessions.sql`: `ListActiveSessionsByIdentityID` — filters `expires_at > NOW()` for a given `identity_id`.

#### New Frontend
| File | Change |
|------|--------|
| `web/src/pages/IdentityDetailPage.tsx` | New page — details card, roles card (with "Manage Roles" link), active sessions card |
| `web/src/api.ts` | Added `IdentitySession` type, `getIdentity`, `getIdentitySessions` |
| `web/src/App.tsx` | Added `/identities/:id` protected route (more specific than `/identities/:id/roles`) |
| `web/src/pages/DashboardPage.tsx` | Identity row action changed from "Manage Roles" → "View" linking to detail page |

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

## Next Steps for the Next AI
1. **Session listing & revocation**: Allow an admin to revoke individual active sessions from the Identity Detail page. Requires a `DELETE /api/identities/{id}/sessions/{token_hash}` endpoint (or similar). The token itself must not be round-tripped; expose a short opaque handle (e.g. first 8 chars of SHA-256) for identification.
3. **Audit Logging**: Implement a system to track changes to identities, roles, and permissions.
4. **OAuth2 consent screen**: Add a React page shown during `/oauth2/authorize` listing requested scopes for user approval. Currently auto-approves. Requires storing pending request state (either in DB or a signed cookie) before displaying the consent UI.
5. **OAuth2 nonce support**: Add `nonce TEXT` column to `oauth2_authorization_codes`; pass it through to the `id_token` JWT `nonce` claim. Required by OIDC libraries that validate nonce (e.g. PKCE + nonce combined).
6. **Client secret rotation**: `POST /api/oauth2/clients/{id}/rotate-secret` — re-generate, re-hash, return once. Does not affect active tokens (those use JWTs verified by the key, not the secret).
7. **Token introspection** (RFC 7662): `POST /oauth2/introspect` — lets resource servers validate an access token without parsing JWTs themselves. Useful for non-JWT-aware services.
8. **Token revocation** (RFC 7009): `POST /oauth2/revoke` — lets clients explicitly invalidate a refresh or access token.

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
