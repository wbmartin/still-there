---
id: PS0010
title: Make the example entity a lean, copyable template — error mapping, scoped queries, retry removal, doc/810 rewrite
status: needs-triage
blocked_by: 
waiting_on:
priority: high
domain: api
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
merged:
deployed:
validated:
---
# PS0010 — Adding an entity costs ~900 lines of Go and the recipe is stale

**Source:** template review 2026-09-21. Pairs with PS0009 (which removes
the security-check ceremony); this ticket owns everything else per entity.

## Problem

### Ceremony per handler method
Each of the 8 `ExampleHandler` methods repeats: claims extraction, parse,
call, then a 3-4 way `errors.Is` switch mapping repo errors to HTTP
status/code/message (`example_handler.go:76-108, 284-332, 405-450`). A
15-line validation block is duplicated verbatim at `:236-266`.
`writeJSON` (`handler/common.go:14`) and `app.WriteJSON` are used
interchangeably in the same file (`:112` vs `:191, 653`).

### Error contracts drift between repositories
- Example maps to `ErrNotFound / ErrDeleted / ErrConflict` (3-state,
  `repository/errors.go`).
- Group has no `ErrDeleted`; "in use" is detected by
  `strings.Contains(err.Error(), "cannot delete group")`
  (`app_user_group_handler.go:319`).
- User returns `fmt.Errorf("user not found")` (`user_repository.go:57`) and
  `"user %d not found, already has group, or concurrent update"` (`:512`).
- Group `Update`/`Delete` existence checks omit `security_context_id`
  (`app_user_group_repository.go:191, 264`) — tenant scoping gap.
- `DeleteExampleList` ignores RowsAffected (silent partial delete,
  `example_repository.go:1332-1345`) while `UpdateExampleList` errors.

### Older and newer query patterns side by side
- `user_repository.go:299-321` hand-rolls WHERE with `= ANY($2)`;
  `app_user_group_repository.go:33` uses `= ANY($1)`; example uses
  `buildScopedWhere` (`scoped_query.go`, PS0001). `withLiveParent` /
  `buildParentScopedWhere` (`scoped_query.go:54-96`) have no callers.
- `user_handler.go:250-253` swallows `afterUpdateTs` parse errors
  (`GetFloatFromQuery` → 0) where example uses `GetOptionalFloatFromQuery`.
- `app_user_group_handler.go:52` uses raw `http.Error` mid-file.
- Example wraps only `Query` in retry (`:1152`); group/user wrap
  Query+Scan (`app_user_group_repository.go:89-115`); no `tx.Begin` path
  is wrapped.

### Retry helper is inert
`repository/retry.go:22-33` matches `database/sql` sentinels (never
produced by pgx) and exact error strings (pgx wraps, so never equal) ⇒ it
never retries. If it did, `InitialInterval: 1s` is on a request path.
Delete `ExecuteWithRetry` and the `cenkalti/backoff` dependency (pgxpool
reconnects), or use `pgconn.SafeToRetry`.

### `db:"..."` tags are never read
Every model carries `db:` tags (~50) but all repositories scan
positionally (`example_repository.go:963-968` etc.). Either adopt
`pgx.RowToStructByName` (drops ~100 lines of scan lists per entity) or
delete the tags so newcomers don't assume a mapper exists.

### Grant catalog defined four times
`model/grants.go`: constants, `AllKnownGrants`, `UserGrants`,
`ExampleGrants`, `GrantIDToShortCodeMap`. `UserGrants`/`ExampleGrants` are
unused; `AllKnownGrants` is used only by its own test. The web
`constants/grants.ts` hardcodes the same ids with no test linking them
(PS0013). `RetrieveGrantList` iterates a map ⇒ nondeterministic order and
requires `GrantUserCreate` (`user_handler.go:304, 316`).

### The recipe
`doc/810-create-new-entity-type-notes.md` references `sql/010-initialcreate.sql`,
`schema biz`/`appcore`, `group_global_writer`, `ReadEntityList` (code uses
`Retrieve*`). It is the document a derived app reaches for first.

## Target

- `handler/common.go`: `mapRepoError(err) (status int, code, msg string)`;
  one `WriteJSON`; one optional-query-param helper.
- `repository/errors.go`: full sentinel set (`ErrNotFound`, `ErrDeleted`,
  `ErrConflict`, `ErrDuplicate`, `ErrInUse`); every repo returns only
  these; every existence check is tenant-scoped via `buildScopedWhere`.
- One `[]Grant{ID, Code, Group}` slice; derive maps; export as JSON for the
  web (PS0013 consumes).
- `retry.go` + backoff dep gone.
- Decide `RowToStructByName` vs no tags; apply to all three entities.
- `doc/810` rewritten against the result: SQL migration
  `migrations/0002_<entity>.sql` → model → repository → handler →
  `Routes()` → web (`features/<entity>/`, PS0012). Target: a new entity is
  ~200 lines of Go, not ~900.

## Acceptance

- `example_*` files are the shortest of the three entities, not the longest.
- `grep -rn 'strings.Contains(err.Error()' internal` → nothing.
- `go.mod` has no `cenkalti/backoff`.
- Following `doc/810` verbatim produces a working entity; verify by doing
  it once on a throwaway branch.
- `bin/200 all` green.

## Deploy needs

Go binary. No migration unless the throwaway-entity check leaves one (it
must not).

## Direction (Brandon, 2026-09-21)

"It's anticipated more complicated scenarios; discuss after PS0009."
PS0009 is now decided (explicit per-handler checks stay, by design), so
strike the "ceremony per handler method" security-check portion of this
ticket entirely; the error-mapping switch and the rest are what is left
to discuss. **Discussion still pending; do not claim.** Some of what reads as ceremony (per-method error switches,
positional scans, separate existence checks) is headroom for entities
that will not be simple CRUD. The discussion should sort each item into
*headroom to keep* vs *drift to fix*; the review's candidates for
"drift regardless": group existence checks missing
`security_context_id`, `strings.Contains` error detection, the inert
retry helper, `DeleteExampleList` ignoring RowsAffected, the stale
`doc/810`.
