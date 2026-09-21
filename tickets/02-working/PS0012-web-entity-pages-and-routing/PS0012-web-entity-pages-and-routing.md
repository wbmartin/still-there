---
id: PS0012
title: Web — fix the entity-page bugs; document that pages are deliberate scaffolding
status: ready
blocked_by:
waiting_on:
priority: medium
domain: web
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0012-web-entity-pages-and-routing
merged:
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

## Acceptance

- Deep link to `/…?appUserGroupId=N` loads the group after refresh.
- `tsc` and `eslint` clean with zero warnings.
- `bin/200 e2e` and vitest green.
- README section exists and lists the files to copy.

## Deploy needs

Web build only.
