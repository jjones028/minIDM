# Project Context: minIDM

## Technical Stack
- **Backend**: Go 1.26.1+, `pgx/v5` (PostgreSQL driver), `sqlc` (Type-safe SQL), `goose` (Migrations).
- **Frontend**: React 19, Vite, Tailwind CSS v4, Radix UI.
- **Design System**: Shadcn/ui (`radix-mira` style) with Amber (`b1tMbvTAQ`) accent palette.
- **Tooling**: `go-task` (Orchestration), Docker (Testcontainers for Dev DB).

## Architecture
- **Pattern**: Backend-for-Frontend (BFF). The API is specifically designed to serve the primary Web UI.
- **Deployment Model**: Single Atomic Binary. The React frontend is built and embedded into the Go binary using `embed` for production.
- **Backend Structure**: Feature-based CQRS. Core logic resides in `internal/feature_name`.
- **RBAC Model**: Advanced **Roles > Resources > Actions** mapping (Resource-Action intersection).

## Current State
- **Backend**: 
  - Identity registration with Argon2id hashing.
  - Identity listing.
  - Development mode with automated PostgreSQL lifecycle via Testcontainers.
  - Feature-based CQRS structure established.
- **Frontend**: 
  - Themed with `radix-mira` and Amber highlights.
  - Functional registration and listing UI.
  - `ThemeProvider` implemented with system-sync and `d` key toggle for dark mode.
- **Build**: 
  - Unified `Taskfile.yml` for dev, build, and code generation.

## Next Steps
1. **RBAC Implementation**: 
   - Define SQL schema for `roles`, `resources`, `actions`, and `permissions`.
   - Implement Go middleware for resource-level access control.
2. **Session Management**: Move from simple API calls to a secure, cookie-based session model suitable for a BFF.
3. **OAuth2/OIDC Provider**: 
   - Implement authorization server logic as an internal feature.
   - Add `clients` and `tokens` management.
