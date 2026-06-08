# minIDM — Web Frontend

React 19 + Vite + Tailwind CSS v4 admin portal for minIDM.

## Stack

- **UI primitives**: [`@base-ui/react`](https://base-ui.com) — unstyled, accessible components
- **Styling**: Tailwind CSS v4 with OKLCH color tokens (`src/index.css`)
- **Icons**: `lucide-react`
- **HTTP**: `axios`
- **Routing**: `react-router-dom` v6

## Development

```bash
npm install
npm run dev     # dev server on http://localhost:5173 (proxies /api, /oauth2, /.well-known → :8080)
npm run build   # production build into dist/ (embedded by Go server)
```

The Vite dev server proxies all backend routes to `http://localhost:8080`. Start the Go backend first (`task dev` from the repo root runs both together).

## Adding UI Components

Do **not** use `npx shadcn add` — it generates Radix UI-based code. Write new components manually using `@base-ui/react` primitives, following the existing components in `src/components/ui/` as patterns.

## Structure

```
src/
  api.ts              ← all API calls + TypeScript types
  App.tsx             ← routes
  context/auth.tsx    ← auth state (calls /api/me on mount)
  components/
    ui/               ← Base UI-backed primitives (button, card, input, select, table)
    app-nav.tsx       ← top navigation
  pages/              ← one file per route
```
