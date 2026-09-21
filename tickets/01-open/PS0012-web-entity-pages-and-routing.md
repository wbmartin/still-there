---
id: PS0012
title: Web — shared entity list/edit scaffolding, route registry, route-level auth guard, feature folders
status: ready
blocked_by:
waiting_on:
priority: high
domain: web
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
merged:
deployed:
validated:
---
# PS0012 — Two generations of web code side by side

**Source:** template review 2026-09-21. Baseline: `tsc` clean; `eslint` 0
errors / 3 warnings (react-hooks/exhaustive-deps on `form` in the three
edit pages).

## Problem

PS0001's `stores/create-entity-store.ts` factory brought every store onto
one pattern (`examples-store`, `app-user-store`, `app-user-group-store`).
The pages never got the same treatment:

- **List pages are ~200-line copy-pastes.** `pages/example-list-page.tsx`,
  `app-user-list-page.tsx:12-221`, `app-user-group-list-page.tsx`: identical
  `hasFetched` ref + `useEffect`, identical sort switch
  (`example-list-page.tsx:48-82` vs `app-user-list-page.tsx:53-81`),
  identical pagination, identical delete-click/confirm/cancel, identical
  sortable-header markup.
- **Edit pages copy the same shell.** `example-edit-page.tsx:23-225`,
  `app-user-edit-page.tsx:31-378`, `app-user-group-edit-page.tsx`: same
  `id/isNew/numericId` parsing from `?xId=new`, same `hasFetched`, same
  CHECKING / !YES / loading triple-return, the same success/error
  `notifications.show` block six times, same DeleteConfirmModal wiring.
  `pages/` is 10 files / 2,133 lines; the six entity pages share ~70%.
- **Adding an entity is ~7 hand edits with no registry:** `types/x-type.ts`,
  `stores/x-store.ts`, two pages, `constants/api.ts`, `constants/routes.ts`,
  `constants/grants.ts`, two route entries in `App.tsx:76-139` (each
  repeating `errorElement: <ErrorPage />`), and a nav link in
  `App.tsx:161-165` — a second hand-maintained list.
- **Auth is a per-page hook.** `hooks/use-auth-guard.ts` is called at the
  top of 8 pages; each then re-renders the CHECKING/!YES states. A page
  that forgets the hook is silently public. `AppLayout` (`App.tsx:141-179`)
  already knows `authState`.
- **ROUTES have no leading slash** (`constants/routes.ts`), so every
  consumer does `` `/${ROUTES.X}` `` (`App.tsx:147, 161-165`,
  `use-auth-guard.ts:29, 51`, `login.tsx:42, 55`, every page). The comment
  at `routes.ts:4` says "append ?id or ?new" but pages use `?appUserId=`,
  `?exampleId=`, `?appUserGroupId=`; `EXAMPLE_EDIT` is `example` while the
  others are `xxxEdit`.
- **`app-user-group-edit-page.tsx:67-87` never fetches the group** — it
  reads `getState().itemList` on mount, so a deep link/refresh to
  `?appUserGroupId=N` yields an empty form, and the effect ignores
  `authState`. `examples-store` and `app-user-store` have `readXById`;
  the group store has none.
- `useAuthStore.getState().grantList` read during render
  (`app-user-edit-page.tsx:63`) — no re-render on change. `setTimeout(…, 0)`
  hacks to dodge set-state-in-effect (`app-user-edit-page.tsx:93, 101`,
  `app-user-group-edit-page.tsx:76`, `use-auth-guard.ts:16`).
- `Container size={1280} px={32} py={32} mx="auto"` repeated ~20×.
- `//====^ HOOK LINE ^====` / `FUNCTION LINE` banners in every page and
  `auth-store.ts:46, 99, 149`.
- Three error surfaces: `App.tsx:34-44` `ErrorPage` (plain HTML),
  `components/ErrorBoundary.tsx` (Mantine), per-page `<Alert color="red">`.

## Target

```
src/web-app/
  lib/            api.ts, cache-utils.ts, create-entity-store.ts, page-shell.tsx,
                  entity-list.tsx (or useEntityList + <SortableTable columns>),
                  use-entity-edit-page.ts, notify.ts
  routes/         registry.ts: [{ path, element, label, navGrant, guard }] → router AND nav derive from it
                  require-auth.tsx: <RequireAuth grants=[...]> route wrapper
  features/
    example/      list-page.tsx, edit-page.tsx, store.ts, types.ts, schema.ts
    users/        …
    groups/       …
    auth/         login, mfa-onboarding, preferences, change-password modal
  constants/      app.ts (APP_NAME…), api.ts, grants.ts (generated — PS0013)
```

- `ROUTES` become absolute paths with `editFor(id)` helpers; one id
  convention (`/example/:id` or a single `?id=`).
- `readById` added to the entity-store factory config so all three edit
  pages load the same way.
- `ErrorBoundary` is the router `errorElement`; delete `ErrorPage`.
- A derived app copies `features/example/` to add an entity and deletes
  it when done (PS0007 checklist).

## Acceptance

- `features/example/` list + edit pages are each < 80 lines.
- Adding a throwaway entity touches: one feature folder + one registry
  line. Verify once on a scratch branch.
- No page imports `use-auth-guard`; the guard is in the registry.
- Deep link to `/…?appUserGroupId=N` loads the group.
- `tsc`, `eslint` clean; `bin/200 e2e` green; vitest green.
- Playwright: one CRUD spec per entity using the existing
  `tests/e2e/fixtures/api-client.ts` (built in PS0001, currently unused).

## Deploy needs

Web build only.

## Related

PS0013 (hygiene, constants generation), PS0007 (derivation checklist).
