# Web App

React 19 / TypeScript / Vite / Mantine frontend.

## Running

```bash
bin/220-run-web-app   # from the repo root; vite dev server on :5173,
                       # proxying /api to VITE_API_PROXY_URL
                       # (.env.development: the dev API, :8080)
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

The Vite dev server proxies `/api/*` to `VITE_API_PROXY_URL`, stripping
the `/api` prefix before forwarding. `.env.development` sets it to the dev
API (`SERVER_PORT` in `etc/loc-api-srv.env`); `playwright.config.ts`
passes the e2e API's instead. There is no default: `vite.config.ts` fails
`vite serve` without it, so `npm run dev:prod` needs it in the
environment. A build never uses it, and `.env.production` sets nothing:
nginx serves the bundle and proxies `/api` on the same origin. Any
`VITE_*`-prefixed variable is available in the browser.

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
- `schemas/` — zod form schemas (`loginSchema` and the monitoring forms;
  the template's entity edit pages validate inline)
- `hooks/` — `use-auth-guard.ts` and similar
- `constants/` — `api.ts` (endpoint paths), `grants.ts`, `routes.ts`,
  `nav.ts` (the navbar's pages, which the command palette lists too)
- `help/` — the help module (`doc/151-help-module.md`): the framework
  (panel, search, markdown, tours, palette, inline `Term` / `FieldHelp` /
  `HelpLink` / `Nudge`) and, in `help/content/`, the help itself: every
  route needs a page entry there (`tests/help-content.test.ts`)
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
`--app-page-header-height`. No template list page has a filter row yet; a
sticky filter row is added with the first page that has one (GRC pattern),
pinned at `calc(var(--app-shell-header-offset, 0px) +
var(--app-page-header-height, 0px))` just below the title.

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

Rules for any store:

- **A request body's keys are the Go `json` tags**, spelled exactly.
  `decodeJSONBody` (`src/api-srv/internal/handler/common.go`) drops a key
  it does not know, so `userId` posted where the handler reads
  `appUserId` arrives as a zero and the call fails every time (B013). The
  store test is what catches it: it stubs `fetch` and asserts each
  request's URL and body against the tags (`tests/examples-store.test.ts`,
  `tests/app-user-store.test.ts`).
- **The store throws; the page tells the user.** A store does not toast
  or swallow a failure (`readExampleById` turns only the 404/400 it
  expects into `null`). The example pages catch, call `reportError`
  (console only) and show the user something: `notifications.show` for a
  failed save, delete or edit-page load, and the list page renders the
  store's `error` in an `<Alert />`.
- **Zustand `set` checks keys only for a bare object literal.**
  `set({ itemLst: [] })` is a type error; the updater form
  `set((state) => ({ ... }))` and a literal cast `as Partial<Store>` (which
  `create-entity-store.ts` needs, being generic) are not checked, so a
  misspelled key there is silently a new field. Prefer the bare literal;
  where a cast is unavoidable, a test reads the field back
  (`tests/entity-store-cache.test.ts`).

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
| `tests/examples-store.test.ts` | `tests/widget-store.test.ts` — vitest (jsdom, as every test in `tests/`), `fetch` stubbed, each write's URL and body against the Go `json` tags |

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
  the load-once ref, the load effect), then the
  `//====^ HOOK LINE ^====` banner, then the early returns for
  `CHECKING` / not-`YES` / loading / not loaded, then handlers and JSX.
  Nothing below the banner may call a hook.
- the load effect runs only when `authState` is `"YES"` (`useAuthGuard`
  answers `"DENIED"`, not `"YES"`, to a user without the page's grants,
  so a refused page asks the API nothing), and an edit page fetches by id
  through the store — never by searching `itemList`, which is empty on a
  deep link. A list page loads once (a `hasFetched` ref). An edit page
  loads once **per route id**: the router reuses the page when only
  `?widgetId=` changes, so a `requestedId` ref records the id the form was
  loaded for, a different id reloads it (`new` resets it), and a response
  for an id the page has since left is dropped.
- an edit page never renders its form for a record it did not load. The
  load records `{ id, found }`; until it matches the route id the page
  shows a `Loader`, and an id that is not a number, came back empty
  (`null` from the store) or failed shows the title and an `<Alert />`
  instead of a blank form whose Save would write a record nobody read.
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
