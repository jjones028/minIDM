# Developer Handover: minIDM

## Stack Summary
- **Languages**: Go (Backend), TypeScript (Frontend).
- **Database**: PostgreSQL (Migrations via `goose` in `services/api/db/migrations`).
- **UI**: Tailwind v4 + Shadcn/ui. Style is `radix-mira` with OKLCH colors.
- **Build**: `task` (Go-Task). Run `task --list` for available commands.

## Architecture
- **Unified Binary**: In production, the Go server serves the React app from `internal/app/static.go`.
- **Feature Folders**: Go code is organized by feature (e.g., `internal/identity`) to keep domain logic isolated.
- **RBAC**: Resource-Action model. Actions (e.g., `read`, `write`, `delete`) are performed on Resources (e.g., `identity`, `role`, `session`).
- **Dev Proxy**: In development, Vite proxies `/api` → `http://localhost:8080`, making all requests same-origin so cookies work without CORS changes.

## Current State (as of 2026-05-04)

### Completed
- **Identity registration & listing** — full CQRS-based flow with Argon2id password hashing.
- **RBAC schema** — `roles`, `resources`, `actions`, `permissions`, `identity_roles` tables (migration `002_create_rbac.sql`). Seeded with `admin` (all permissions) and `viewer` (read-only) roles. First registered identity becomes admin; all subsequent get viewer.
- **Session infrastructure** — `sessions` table (migration `003_create_sessions.sql`). Login creates a session row; logout deletes it.
- **Cookie-based sessions** — Login sets an `HttpOnly; SameSite=Strict` cookie named `session`. Set `SECURE_COOKIES=true` env var in production (requires HTTPS). Bearer token / localStorage approach has been fully removed.
- **Authorization middleware** — `internal/rbac/middleware.go`:
  - `Authenticate`: reads `session` cookie, validates against DB, injects identity UUID into request context.
  - `Require(resource, action)`: checks identity has the named permission via `IdentityHasPermission` query.
- **`GET /api/me`** — auth-only endpoint (no permission check) the frontend uses to verify session validity on page load.
- **Role management UI** — `IdentityRolesPage` at `/identities/:id/roles`. Shows assigned roles with remove buttons; dropdown (shadcn `Select` with `position="popper"`) to assign available roles.
- **Navigation** — Dashboard identity table has a "Manage Roles" button per row.
- **Auth context** — `web/src/context/auth.tsx`. Calls `/api/me` on mount to determine initial auth state. Exposes `setAuthenticated(bool)` so pages update auth state after login/logout/401 without touching localStorage.
- **Connection pooling** — switched `server.go` from `pgx.Connect` (single connection, not goroutine-safe) to `pgxpool.New`. Critical fix: concurrent requests (e.g. `Promise.all`) on a single `pgx.Conn` caused sporadic 401s.

### Known Architecture Notes
- `internal/rbac/handler.go` contains the role management API routes (`GET /api/roles`, `GET|POST|DELETE /api/identities/{id}/roles/{roleId}`).
- `internal/app/router.go` is the single place where all routes and protection chains are assembled.
- `web/src/api.ts` — all API functions. No token management; cookies are handled transparently by the browser.

## Next Steps for the Next AI
1. **OAuth2/OIDC Provider Integration**: Begin implementing `internal/oauth2` features to support external clients and OIDC flows (`clients` table, authorization endpoint, token endpoint, PKCE).
2. **Role management for roles themselves**: Currently roles are seeded only. Add UI and API to create/edit/delete roles and assign permissions to them.
3. **Identity detail page**: A single-identity view showing subject ID, enabled status, assigned roles, and active sessions.
4. **Session listing & revocation**: Allow an admin to view and revoke active sessions for any identity.
