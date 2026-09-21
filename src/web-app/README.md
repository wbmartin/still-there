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

## Scaffolding

Entity pages are deliberate copy-paste scaffolding. Each page is
self-contained and reads top to bottom; there is no shared list/edit
factory, route registry, route-level auth wrapper or feature folder, and
that is by design (PS0012). A derived app copies the `example-*` files and
edits in place. If it grows past ~6 entities and wants the factoring, that
is the derived app's call, not the template's.

To add an entity `widget`, copy and rename:

| Copy | To |
|---|---|
| `types/example-type.ts` | `types/widget-type.ts` — the model, `defaultWidgetFormData` |
| `stores/examples-store.ts` | `stores/widget-store.ts` — `createEntityStore` + `readWidgetById` |
| `pages/example-list-page.tsx` | `pages/widget-list-page.tsx` |
| `pages/example-edit-page.tsx` | `pages/widget-edit-page.tsx` |

Then add, by hand, in each of:

- `constants/api.ts` — `WIDGET_LIST: "/widgetList"`, `WIDGET: "/widget"`
  (mirrors `src/api-srv/cmd/server/main.go`; no generator)
- `constants/grants.ts` — the four `*_WIDGET` grant ids from `roles.sql`
- `constants/routes.ts` — `WIDGET_LIST: "/widgetList"`,
  `WIDGET_EDIT: "/widget"` (absolute paths; the edit page takes
  `?widgetId=N` or `?widgetId=new`)
- `app.tsx` — two entries in `createBrowserRouter` and a `NavLink`
- `tests/e2e/` — a deep-link case like `entity-pages.spec.ts`

The API half is `doc/810-create-new-entity-type-notes.md`.

House pattern inside a page, in this order:

- hooks first (`useSearchParams`, `useAuthGuard`, the store, `useForm`,
  a `hasFetched` ref, the load effect), then the
  `//====^ HOOK LINE ^====` banner, then the early returns for
  `CHECKING` / not-`YES` / loading, then handlers and JSX. Nothing below
  the banner may call a hook.
- the load effect runs once (`hasFetched` ref), only when `authState`
  is `"YES"`, and fetches by id through the store — never by searching
  `itemList`, which is empty on a deep link.
- `form` from `useForm` is a new object each render, so the load effect
  carries a scoped `react-hooks/exhaustive-deps` disable with that reason.
- list state that should survive a refresh or a shared link (page, sort)
  lives in the URL via `useSearchParams`, not in `useState`.
- `ROUTES` values are absolute; `navigate(ROUTES.X)`, never
  `` `/${ROUTES.X}` ``.

## Dependency cadence

Caret ranges. Any ticket that touches the web app runs `npm update
--legacy-peer-deps` (in-range bumps; plain `npm update` trips an arborist
bug in npm 10 on this peer set) and `npx depcheck` (no unused or undeclared
packages). Major versions
(Mantine, ESLint, Vitest, Vite plugins) are their own ticket, one at a time.
