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
- `components/` — shared UI components (modals, error boundary) and
  shared leaves: `sort-th.tsx` (a sortable list-page `<th>` whose label is a
  button, with `aria-sort`) and `help-tip.tsx` (`LabelWithHelp` puts field
  help behind an info icon; use it as an input's `label` instead of
  `description`)
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

## Page Header and Filter Row

`AppLayout` (`app.tsx`) renders every page inside
`<div className="app-page">`. The page root's child that is, or holds, the
page's one `<h1>` (a `Title order={1}` alone or in a `Group` with its
buttons) stays pinned under the shell header while the page scrolls
(`index.css`). `hooks/use-pinned-page-header.ts` measures that block into
`--app-page-header-height`, and `components/sticky-filters.tsx` uses it: a
list page that adds a filter row wraps it in `<StickyFilters>` so the row
pins just below the title. No template list page has a filter row yet.

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

## API Errors

Every call goes through `apiJson` in `utils/api.ts`; it is the only
export that touches the network (the raw request function is private, and
ESLint's `no-restricted-globals` keeps `fetch` out of app code). A 2xx
comes back parsed; no caller reads `response.ok`. A failure becomes one
of:

| Thrown | When | Who reacts |
|---|---|---|
| `ApiUnreachableError` | `fetch` itself failed (API down, connection dropped), or a **5xx** or **429** (nginx's 502 while the API restarts, the per-minute rate limit, an account lockout); `status` is set for the last two, with the server's message | `stores/api-status-store.ts` raises one banner above the page ("still signed in"), cleared by the next 2xx; the page-load `/refresh` retries at 1/2/4 s, then sets `authState` `"UNREACHABLE"` |
| `SessionExpiredError` | **401** on a call that was not `anonymous` — `CheckJWT`'s answer to a missing, invalid, expired or revoked `auth` cookie | `stores/auth-store.ts`, subscribed to `session-expired`: the one sign-out path, and only from an established session |
| `SessionChangedError` | a write refused for CSRF whose `/refresh` shows another user or tenant (another tab) | `utils/api.ts` has already moved the tab to that identity |
| `ApiError` | everything else: 400, **403**, 404, 409; `status` set, the server's message (`SessionExpiredError` is a subclass) | the calling store or page; `isApiStatus(err, 404)` picks out one it expects |

`utils/api.ts` announces each call's outcome through `subscribeApiOutcome`
(`ok` on a 2xx, `unreachable`, `session-expired`); the api status store
and the auth store are its two subscribers. The auth store signs out only
from `authState` `"YES"`, so the 401 a signed-out page load gets is not
toasted as "Session ended". The CSRF resync's own `/refresh` is a single
attempt, neither retried nor announced.

A **403 is not an expired session**. The API answers 403 for a missing
grant, a blocked account, a CSRF token `utils/api.ts` could not resync
(it retries a `"code": "CSRF"` refusal once after `/refresh`) and a
cross-tenant refusal; signing in again fixes none of them, so signing out
on one would send a signed-in user to the login page for the first call
their role may not make, behind the wrong reason ("Session ended").

`anonymous: true` marks the calls made without a session (`/auth`,
`/mfaSetup`, `/mfaVerify`, `/inviteCheck`, `/inviteRedeem`,
`/userSelfCreate`) or leaving one (`/logoff`): no `X-CSRF-Token` header
is sent, and a 401 means a wrong password or code (or a session already
gone) and must not touch the session.
The page-load `/refresh` (`authenticateWithAuthToken`) is separate: there
a 401 or 403 (the block check) sets `authState` `"NO"`, an unreachable API
is retried at 1, 2 and 4 s, and any other failure sets `"UNREACHABLE"`
(the full-page notice with "Try again").

## Scaffolding

Entity pages are deliberate copy-paste scaffolding. Each page is
self-contained and reads top to bottom; there is no shared list/edit
factory, route registry, route-level auth wrapper or feature folder, and
that is by design. A derived app copies the `example-*` files and
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
- `constants/grants.ts` — the four `*_WIDGET` grant ids from
  `src/api-srv/internal/model/grants.go`, after
  `bin/305-update-grants-fixture`; `tests/contract.test.ts` fails if the
  two disagree
- `constants/routes.ts` — `WIDGET_LIST: "/widgetList"`,
  `WIDGET_EDIT: "/widget"` (absolute paths; the edit page takes
  `?widgetId=N` or `?widgetId=new`)
- `app.tsx` — two entries in `createBrowserRouter` and a `NavLink`
- `tests/e2e/` — a deep-link case like `entity-pages.spec.ts`, and
  two rows in `route-sweep.spec.ts` (`?widgetId=new` and a fixture id,
  with where admin, basic and a signed-out visitor land); its first
  test fails on a route it does not sweep

The API half is `doc/810-create-new-entity-type-notes.md`.

House pattern inside a page, in this order:

- hooks first (`useSearchParams`, `useAuthGuard`, the store, `useForm`,
  a `hasFetched` ref, the load effect), then the
  `//====^ HOOK LINE ^====` banner, then the early returns for
  `CHECKING` / not-`YES` / loading, then handlers and JSX. Nothing below
  the banner may call a hook.
- the load effect runs once (`hasFetched` ref), only when `authState`
  is `"YES"` (`useAuthGuard` answers `"DENIED"`, not `"YES"`, to a
  user without the page's grants, so a refused page asks the API
  nothing), and fetches by id through the store — never by searching
  `itemList`, which is empty on a deep link.
- `form` from `useForm` is a new object each render, so the load effect
  carries a scoped `react-hooks/exhaustive-deps` disable with that reason.
- list state that should survive a refresh or a shared link (page, sort)
  lives in the URL via `useSearchParams`, not in `useState`. Each list
  page writes it through its own merge-style `setListParams` (a sort or a
  page change keeps every other parameter), reads `dir` strictly
  (anything but `desc` is `asc`), and clamps `?page=` to the last page.
  Sortable headers are `SortTh`, handed the page's own `sortColumn`,
  `sortDirection` and `handleSort`; the sort itself stays in the page.
- `ROUTES` values are absolute; `navigate(ROUTES.X)`, never
  `` `/${ROUTES.X}` ``.

## Dependency cadence

Caret ranges. Every change that touches the web app runs `npx depcheck`
(no unused or undeclared packages); it is cheap and never bumps anything.
No change runs a blanket `npm update`: an in-range bump of the linter
plugins ships new lint rules, so an unrelated diff picks up lint fallout
and lockfile churn its reviewer did not ask for.

A change with a security reason to touch dependencies works in this order:

1. `npx depcheck`; remove unused packages first. Deleting a dead
   dependency is often the whole fix.
2. `npm audit --omit=dev`; production advisories are the target.
3. `npm update --legacy-peer-deps <advised package>` for the packages the
   advisories name, nothing else (plain `npm update` trips an arborist bug
   in npm 10 on this peer set). Commit `package-lock.json` with it.
4. Dev-only advisories and toolchain bumps (ESLint, typescript-eslint,
   Vite, Vitest) are their own change, with the lint fallout in the same
   diff. Major versions (Mantine, ESLint, Vitest, Vite plugins) are their
   own change too, one at a time.

The Go side follows the same order (`src/api-srv/README.md`,
"Dependencies").
