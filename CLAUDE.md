# Developer Handover: my-idm

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

## Next Steps for the Next AI
1. **RBAC Schema**: Build out the tables for the Resource-Action model established in the design discussion.
2. **Middleware**: Create an authorization middleware in `internal/app` that leverages the RBAC tables.
3. **OAuth2 Service**: Start implementing `internal/oauth2` to turn the IDM into a full OIDC Provider.
4. **BFF Security**: Implement HttpOnly cookie sessions to replace any placeholder token logic.
