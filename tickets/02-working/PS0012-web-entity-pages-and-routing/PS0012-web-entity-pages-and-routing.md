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
deployed:
validated:
---
# PS0012 — Entity pages: scaffolding by design, plus a few real bugs

**Source:** template review 2026-09-21.
**Decision (Brandon, 2026-09-21):** "Premature optimization — these are
designed to be scaffolding." The review proposed shared list/edit
factories, a route registry, route-level auth and feature folders. **Not
doing any of that.** The copy-paste pages are the point: a derived app
copies `example-list-page.tsx` / `example-edit-page.tsx` and edits in
place, and can read the whole page top to bottom.

This ticket keeps only what is a bug or would mislead the next agent.

## Fix

1. **`pages/app-user-group-edit-page.tsx:67-87` never fetches the group.**
   It reads `useAppUserGroupStore.getState().itemList` on mount, so a deep
   link or refresh to `?appUserGroupId=N` yields an empty form, and the
   effect ignores `authState` (fires before auth resolves).
   `examples-store` has `readExampleById` and `app-user-store` has
   `readUserById`; add the equivalent to the group store (or to the
   `create-entity-store` factory config, since `idParam` is already
   there) and load the same way the other two edit pages do.
2. **`hooks/use-auth-guard.ts:15`** compares `location.pathname` to
   `ROUTES.LOGIN` (`"login"`, no leading slash) — never matches. PS0002
   patches the comparison; here, decide whether `ROUTES` values should
   carry the leading slash (every consumer does `` `/${ROUTES.X}` ``:
   `App.tsx:147, 161-165`, `use-auth-guard.ts:29, 51`, `login.tsx:42, 55`,
   every page). Making them absolute is a mechanical change that removes
   a class of bug without changing the scaffolding shape. Recommended.
3. **`useAuthStore.getState().grantList` read during render**
   (`app-user-edit-page.tsx:63`) — no re-render on grant change. Use the
   hook selector.
4. `setTimeout(…, 0)` to dodge set-state-in-effect
   (`app-user-edit-page.tsx:93, 101`, `app-user-group-edit-page.tsx:76`,
   `use-auth-guard.ts:16`) — fix only if the fix is local and obvious;
   otherwise leave with a comment.
5. The three `react-hooks/exhaustive-deps` warnings on `form` in the edit
   pages (the only eslint output) — silence with the documented pattern
   or a scoped disable with a reason.

## Document (so no future agent "refactors" the scaffolding)

- `src/web-app/README.md`: a short *Scaffolding* section stating that
  entity pages are intentionally self-contained copies; to add an
  entity, copy the two `example-*` pages, the store, the type, and add
  the route + nav entries in `App.tsx` and the constants — list the
  exact files (this doubles as the web half of `doc/810`, PS0010).
- Note in the same section that the `//====^ HOOK LINE ^====` banners
  and the `hasFetched` ref are the house pattern.

## Not doing (recorded)

`EntityListPage<T>`, `useEntityEditPage`, `routes/registry.ts`,
`<RequireAuth>`, `features/<entity>/` folders, `page-shell`, ErrorPage/
ErrorBoundary consolidation. If a derived app grows past ~6 entities
and wants them, that is the derived app's call.

## From the PS0006 triage of doc/000-TODO.md (2026-09-21)

- The user-list page keeps `page` in `useState` although it already
  imports `useSearchParams`; a refresh or a shared link loses the page.
  Put the page (and any filter) in the URL. Same pattern for every list
  page this ticket touches.
- Six files under `pages/` and `components/` still use `style={{…}}`
  instead of Mantine props/layout components; clean as each page is
  revisited.

## What

Bug fixes and one README section on the web entity pages, keeping the
copy-paste scaffolding shape (decision above). No factoring.

## Root cause

- `app-user-group-edit-page` searched the store's `itemList` on mount
  instead of fetching by id, so a deep link or refresh had nothing to find.
  The other two edit pages fetch by id through the store.
- `ROUTES` values had no leading slash and every consumer re-added it;
  the one place that forgot (the auth-guard comparison) never matched.
- `useAuthStore.getState()` in render bypasses the subscription.
- `setTimeout(…, 0)` was papering over `react-hooks/set-state-in-effect`.
- `page` in `useState` on all three list pages, lost on refresh.
- The in-range dep bumps (PS0013) brought a stricter typescript-eslint:
  two `no-unnecessary-type-assertion` errors and two
  `set-state-in-effect` errors were live on `main`.

## What changed and why

- `stores/app-user-group-store.ts`: `readAppUserGroupById` — GET
  `/appUserGroupList?appUserGroupIdList=N`, merge into the cache (no GET
  `/appUserGroup` exists; same shape as `readUserById`).
- `pages/app-user-group-edit-page.tsx`: load effect now waits for
  `authState === "YES"`, fetches by id, shows a loader, no `setTimeout`.
- `pages/example-edit-page.tsx`: same `authState` gate so the template
  page is the best copy.
- `pages/app-user-edit-page.tsx`: `grantList` via hook selector; the
  `setTimeout` dodges removed (`userLoading` starts false for `new`).
- `constants/routes.ts`: values are absolute. All 40 consumers dropped
  the `` `/${ROUTES.X}` `` wrapper (`navigate(ROUTES.X)`); `app.tsx`
  child routes take the absolute paths unchanged.
- `hooks/use-auth-guard.ts`: `CHECKING` is derived (`TBD` off the login
  page) instead of `isValidating` state set inside the effect.
- List pages: `page` read from and written to `?page=`; sorting drops
  `page` (resets to 1). `Table.Th` sort headers use a `.sortable-column`
  class (`index.css`); other `style={{}}` sites became Mantine props
  (`h`, `maw`, `Paper withBorder`, `Image w/h/fit`). Left:
  `error-boundary.tsx` `wordBreak` — no Mantine prop for it.
- Lint to zero: `mfa-onboarding-page` `hasFetched` is a ref (house
  pattern); two redundant `as` casts removed; the three edit-page
  `exhaustive-deps` warnings on `form` carry a scoped disable with the
  reason (`form` is a new object every render; `hasFetched` makes the
  effect one-shot).
- `tests/e2e/entity-pages.spec.ts`: deep link to group 2 loads "Admin"
  cold and after `reload()`; sort lands in the URL. Verified the first
  case fails against the old page.
- `src/web-app/README.md` *Scaffolding*: the four files to copy, the
  five by-hand additions, the in-page house pattern (hook order, HOOK
  LINE, `hasFetched`, load by id, URL state, absolute `ROUTES`).
  `doc/810` WEB-APP section now points at it (it said routes have no
  leading slash and named files that no longer exist).

## Files modified

`doc/810-create-new-entity-type-notes.md`, `src/web-app/README.md`,
`app.tsx`, `constants/routes.ts`, `hooks/use-auth-guard.ts`, `index.css`,
`pages/{app-user-edit,app-user-group-edit,app-user-group-list,
app-user-list,example-edit,example-list,login,mfa-onboarding,
user-preferences}-page.tsx`, `stores/{app-user-group-store,auth-store,
create-entity-store}.ts`, `tests/e2e/entity-pages.spec.ts` (new).

## Endpoints introduced or modified

None. The web app now also calls the existing authenticated
`GET /appUserGroupList?appUserGroupIdList=N` (JWT, READ_USER_GROUP grant;
returns group name + grant ids, no PII).

## Checks

`go build`, `tsc`, `eslint` (0 problems, was 4 errors / 3 warnings on
main), `bin/300-run-tests all`: go=163 unit=28 e2e=5 py=63.

## Acceptance

- Deep link to `/…?appUserGroupId=N` loads the group after refresh.
- `tsc` and `eslint` clean with zero warnings.
- `bin/200 e2e` and vitest green.
- README section exists and lists the files to copy.

## Deploy needs

Web build only.
