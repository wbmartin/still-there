---
id: PS0012
title: Web — fix the entity-page bugs; document that pages are deliberate scaffolding
status: in-review
blocked_by:
waiting_on:
priority: medium
domain: web
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0012-web-entity-pages-and-routing
merged: f76f1fb
deployed: none
validated: 2026-09-21
---
# PS0012 — Entity pages: scaffolding by design, plus a few real bugs

A template review proposed shared list/edit factories, a route registry,
route-level auth and feature folders; Brandon rejected all of it as
premature optimization — the copy-paste entity pages are deliberate
scaffolding for a derived app to copy and edit in place. The ticket fixed
only the real bugs and documented the scaffolding contract so no future
agent "refactors" it.

## What changed
- `stores/app-user-group-store.ts`: added `readAppUserGroupById` (GET
  `/appUserGroupList?appUserGroupIdList=N`, merged into cache) — the group
  edit page now fetches by id instead of reading a possibly-empty store
  cache, fixing an empty form on a cold deep link or refresh.
- `pages/app-user-group-edit-page.tsx`, `pages/example-edit-page.tsx`: load
  effects gated on `authState === "YES"`, `setTimeout` dodge removed.
- `pages/app-user-edit-page.tsx`: `grantList` read via hook selector
  instead of `getState()` during render.
- `constants/routes.ts`: route values made absolute; 40 call sites drop the
  `` `/${ROUTES.X}` `` wrapper — fixes the auth-guard login-path comparison
  that never matched before this.
- `hooks/use-auth-guard.ts`: `CHECKING` derived from `authState` instead of
  separate `isValidating` state set inside the effect.
- List pages: `page`, `sort`, `dir` moved into the URL via
  `useSearchParams`; inline `style={{}}` replaced with Mantine props or a
  `.sortable-column` class where possible.
- Lint brought to zero (was 4 errors / 3 warnings on main from PS0013's
  dependency bumps): redundant `as` casts removed, `hasFetched` is a ref,
  three `exhaustive-deps` warnings carry a scoped disable with a reason.
- `tests/e2e/entity-pages.spec.ts` (new): deep link to a group loads
  correctly cold and after reload; sort persists in the URL.
- `src/web-app/README.md`: new *Scaffolding* section — the four files to
  copy, the five by-hand additions, the house pattern (hook order, HOOK
  LINE banners, `hasFetched`, load-by-id, URL state, absolute routes).
  `doc/810` now points at it instead of restating stale file names.

## Rules that survived
- Entity pages are deliberate copy-paste scaffolding, not shared
  components — documented in `src/web-app/README.md`, not left as tribal
  knowledge (Brandon's decision, 2026-09-21).
- `ROUTES` constants are absolute paths; consumers use them directly, no
  `` `/${ROUTES.X}` `` wrapper.
- An edit page's load effect gates on `authState === "YES"` and fetches by
  id — never reads the store cache directly on mount.
- List-page pagination and sort state live in the URL, not component
  state, so refresh or a shared link doesn't lose them.

## Figures at close
| | date |
|---|---:|
| eslint problems | 0 (was 4 errors / 3 warnings) |
| go tests | 163 |
| vitest unit tests | 28 |
| e2e tests | 5 |
| pytest | 63 |

## Endpoints
None introduced. The web app now also calls the existing
`GET /appUserGroupList?appUserGroupIdList=N` (JWT, READ_USER_GROUP grant,
security-context scoped).

## Security
Profiles 1 (UI feature) and 4 (auth & permission logic). No new data
surfaced beyond what the page already rendered — the group fetch runs
through the full existing handler chain and is tenant-scoped, so a deep
link to another tenant's id returns an empty form rather than data. The
auth-guard change is a like-for-like refactor of the loading-flag
derivation; no redirect or grant decision changed, and the guard still
fails closed on any state other than `YES`. `grantList` via selector fixes
a stale-render UI convenience only — the server enforces the grant
independently.

## Left open
- Six files under `pages/`/`components/` still use inline `style={{}}` —
  noted for the next page revisit.
- `error-boundary.tsx`'s `wordBreak` has no Mantine prop equivalent, left
  as inline style.

## Sources
- `tickets/04-archive/engineering-log/260921_1735-TKT-PS0012-web-entity-pages-and-routing.md`
- git: `f76f1fb`
