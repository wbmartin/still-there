# Web App

React 19 / TypeScript / Vite / Mantine frontend.

## Running

```bash
bin/220-run-web-app   # from the repo root; vite dev server on :5173,
                       # proxying /api to VITE_API_PROXY_URL (default
                       # http://localhost:8080)
```

or, from `src/web-app`:

```bash
npm run dev          # development mode (.env.development)
npm run dev:prod     # production mode (.env.production)
```

### Building

```bash
npm run build        # development build
npm run build:prod   # production build
```

## API Proxy

The Vite dev server proxies `/api/*` to `VITE_API_PROXY_URL` (default
`http://localhost:8080`; see `vite.config.ts`), stripping the `/api`
prefix before forwarding. Set `VITE_API_PROXY_URL` in `.env.development`
/ `.env.production`; any `VITE_*`-prefixed variable is available in the
browser.

## Project Structure

- `pages/` — route components; entity pages are named
  `<entity>-list-page.tsx` / `<entity>-edit-page.tsx`
- `components/` — shared UI components (modals, error boundary)
- `stores/` — Zustand state: one store per entity plus
  `create-entity-store.ts` (see below) and `auth-store.ts`
- `schemas/` — zod form schemas (today only `loginSchema`; edit pages validate inline)
- `hooks/` — `use-auth-guard.ts` and similar
- `constants/` — `api.ts` (endpoint paths), `grants.ts`, `routes.ts`
- `types/` — TypeScript model types
- `utils/` — fetch wrapper, cache helpers
- `tests/e2e/` — Playwright specs (config: `playwright.config.ts`; runs
  its own dev pair on ports 5182/8082, separate from the ordinary dev
  stack)

## Entity Stores

`stores/create-entity-store.ts` factors out the CRUD/cache pattern every
entity store follows (e.g. `stores/examples-store.ts`): each store holds
`itemList` plus `maxUpdateTs`, the highest `updateTs` seen on a full list
read. `readList({ incremental: true })` sends `?afterUpdateTs=<maxUpdateTs>`
and merges the response into the cache instead of replacing it; a row
that comes back with `deleteTs` set is a tombstone and is removed rather
than stored. `maxUpdateTs` only advances on a full (non-incremental)
read — mutations never touch it. Every store registers a cache clearer so
sign-out and security-context switches cannot leave another tenant's rows
in memory.

Entity pages (`example-list-page.tsx`, `example-edit-page.tsx`, and the
`appUser`/`appUserGroup` equivalents) are deliberate copy-paste
scaffolding — a new entity is made by copying the `example-*` files.
There is no shared list/edit factory, and that is by design; ticket
PS0012 owns any change to that.

## Dependency cadence

Caret ranges. Any ticket that touches the web app runs `npm update
--legacy-peer-deps` (in-range bumps; plain `npm update` trips an arborist
bug in npm 10 on this peer set) and `npx depcheck` (no unused or undeclared
packages). Major versions
(Mantine, ESLint, Vitest, Vite plugins) are their own ticket, one at a time.
