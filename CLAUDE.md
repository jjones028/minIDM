# Developer Handover: minIDM

## Stack Summary
- **Languages**: Go (Backend), TypeScript (Frontend).
- **Database**: PostgreSQL (Migrations via `goose` in `services/api/db/migrations`).
- **UI**: Tailwind v4 + Shadcn/ui. Style is `radix-mira` with OKLCH colors.
- **Build**: `task` (Go-Task). Run `task --list` for available commands.

## Architecture
- **Unified Binary**: In production, the Go server serves the React app from `internal/app/static.go`.
- **Feature Folders**: Go code is organized by feature (e.g., `internal/identity`) to keep domain logic isolated.
- **RBAC**: Moving toward a Resource-Action model. Actions (e.g., `read`) are performed on Resources (e.g., `identity`).

## Current State
- The project has a working Identity registration flow.
- The UI is fully themed and supports dark mode (Shortcut: `d`).
- The build pipeline is established; `task build` produces a self-contained deployment artifact.
- `web/src/index.css` is the source of truth for the project's visual theme.
- Foundation for CQRS-based feature development in Go is active.

## Next Steps for the Next AI
1. **RBAC Schema Implementation**: Define SQL tables and relationships for `roles`, `resources`, `actions`, and `permissions` based on the design pattern.
2. **Authorization Middleware**: Develop the Go middleware to intercept requests and perform resource-level access control.
3. **Session Infrastructure**: Replace current token handling with a secure, cookie-based session management system.
4. **OAuth2/OIDC Provider Integration**: Begin implementing `internal/oauth2` features to support external clients and OIDC flows.
