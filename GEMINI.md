# Project Context: minIDM

## Technical Stack
- **Backend**: Go 1.26.1+, `pgxpool/v5` (PostgreSQL connection pool), `sqlc` (type-safe SQL), `goose` (migrations).
- **Frontend**: React 19, Vite, Tailwind CSS v4, Radix UI / Shadcn/ui.
- **Design System**: Shadcn/ui (`radix-mira` style) with OKLCH colors. `web/src/index.css` is the source of truth for theming.
- **Tooling**: `go-task` (orchestration), Docker/Testcontainers (dev DB lifecycle via `cmd/dev/main.go`).

## Architecture
- **Pattern**: Backend-for-Frontend (BFF). The API is specifically designed to serve the primary Web UI.
- **Deployment Model**: Single atomic binary. The React frontend is built and embedded into the Go binary via `//go:embed` in `internal/app/static.go`.
- **Backend Structure**: Feature-based CQRS. Core logic lives in `internal/<feature_name>/` (e.g. `internal/identity/`, `internal/rbac/`, `internal/session/`). Each feature follows a strict pattern:
  - `handler.go`: HTTP routing and JSON Marshalling.
  - `<command>.go` / `<query>.go`: Isolated logic handlers (e.g., `roles.go`, `logout.go`) that execute against `db.Queries`.
  - Business rules (e.g., protecting built-in roles) are enforced in Command handlers, not HTTP handlers.
- **RBAC Model**: Roles → Resources × Actions. The `IdentityHasPermission` SQL query drives all authorization checks.
- **Dev Proxy**: Vite dev server proxies `/api` → `http://localhost:8080`. This makes browser requests same-origin so `HttpOnly` cookies flow without CORS credential complexity.

## Current State (as of 2026-05-04)

### Backend
- **Identity**: Registration (Argon2id hashing), listing, bootstrap admin creation. First identity → `admin` role; subsequent → `viewer` role.
- **RBAC**: Full schema — `roles`, `resources`, `actions`, `permissions`, `identity_roles`. Middleware in `internal/rbac/middleware.go`:
  - `Authenticate` reads the `session` HTTP cookie and validates it against the `sessions` table.
  - `Require(resource, action)` checks the identity has the named permission.
  - `RegisterRoleRoutes` exposes:
    - `GET|POST /api/roles`: List and create roles.
    - `PATCH|DELETE /api/roles/{id}`: Update or delete roles (built-in roles are protected).
    - `GET|POST|DELETE /api/roles/{id}/permissions`: Manage role permissions (resource x action).
    - `GET /api/resources`, `GET /api/actions`: Metadata for permission matrix.
    - `GET|POST /api/identities/{id}/roles`: List and assign roles to identities.
    - `DELETE /api/identities/{id}/roles/{roleId}`: Remove role from identity.
- **Sessions**: Cookie-based. Login → `Set-Cookie: session=<token>; HttpOnly; SameSite=Strict`. Logout → clears cookie and deletes DB row. `SECURE_COOKIES=true` env var enables the `Secure` flag (requires HTTPS, set this in production).
- **`GET /api/me`**: Auth-only endpoint (no permission check). Returns `{ id }` for the current identity. Used by the frontend to check session validity on page load.
- **Connection pool**: `server.go` uses `pgxpool.New` — required because Go's HTTP server handles requests on separate goroutines and `pgx.Conn` is not goroutine-safe.
- **Routes assembled in**: `internal/app/router.go`. All protection chains are built there.

### Frontend
- **Auth context** (`web/src/context/auth.tsx`): Calls `GET /api/me` on mount to establish initial auth state. Exposes `setAuthenticated(bool)` — pages call this after login (true), logout (false), or receiving a 401 (false). No localStorage token management; cookies are transparent.
- **Routing** (`web/src/App.tsx`): `ProtectedRoute` waits for `checked` (initial `/api/me` complete) before rendering or redirecting.
- **API client** (`web/src/api.ts`): Axios instance with `baseURL: '/api'`. No interceptors; cookies are sent automatically.
- **Navigation** (`web/src/components/app-nav.tsx`): Centralized navigation between Identities and Roles.
- **Identity dashboard** (`web/src/pages/DashboardPage.tsx`): Lists identities, creates identities, "Manage Roles" button per row.
- **Role management page** (`web/src/pages/IdentityRolesPage.tsx`): At `/identities/:id/roles`. Shows assigned roles with remove buttons; `Select` (Shadcn, `position="popper"`) + clear button (×) to assign unassigned roles.
- **Role CRUD page** (`web/src/pages/RolesPage.tsx`): Create, list, edit, and delete roles. Links to permissions management.
- **Permission Matrix** (`web/src/pages/RolePermissionsPage.tsx`): Toggle-based grid for managing Resource × Action permissions per role.
- **Shadcn components installed**: `button`, `card`, `input`, `table`, `select`.

## Critical Files
| File | Purpose |
|------|---------|
| `services/api/internal/app/router.go` | All route registration and middleware chains |
| `services/api/internal/rbac/middleware.go` | `Authenticate` + `Require` middleware |
| `services/api/internal/rbac/handler.go` | Role and Permission management API handlers |
| `services/api/internal/session/handler.go` | Login / logout cookie logic |
| `services/api/db/migrations/` | `001` identities, `002` RBAC, `003` sessions, `004` builtin roles |
| `services/api/db/queries/rbac.sql` | Source SQL for sqlc-generated RBAC queries |
| `web/src/context/auth.tsx` | React auth state (replaces localStorage tokens) |
| `web/src/api.ts` | All API calls |
| `web/src/components/app-nav.tsx` | Main application navigation |
| `web/vite.config.js` | Vite proxy (`/api` → `:8080`) |

## Next Steps
1. **OAuth2/OIDC Provider**: Implement `internal/oauth2` — `clients` table, authorization endpoint, token endpoint, PKCE support.
2. **Identity detail page**: Single-identity view — subject ID, enabled status, roles, active sessions.
3. **Session management**: Admin view to list and revoke active sessions per identity.
4. **Audit Logging**: Implement a system to track changes to identities, roles, and permissions.
