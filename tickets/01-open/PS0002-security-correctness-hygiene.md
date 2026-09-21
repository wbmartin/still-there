---
id: PS0002
title: Security and correctness hygiene found in the 2026-09-21 template review
status: ready
blocked_by:
waiting_on:
priority: high
domain: security
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
merged:
deployed:
validated:
---
# PS0002 — Security and correctness hygiene

**Source:** read-only review of the template on 2026-09-21 (four subagent
passes: api-srv, web-app, bin/etc/infra/tests, docs). This ticket collects
the findings that are plain bugs or plain leaks. None needs a design
decision, so it goes first. Every item below was spot-checked against
source by the reviewing session.

## What

Fix the concrete defects; remove tracked artifacts that should never have
been committed. Small, independent commits are fine.

## Items

### API (`src/api-srv/`)

1. **`GET /userList` returns encrypted TOTP seeds.**
   `internal/model/app_user_model.go:15` — `TOTPSeed *string` is tagged
   `json:"totpSeed,omitempty"`. `internal/repository/user_repository.go`
   (~302-306, 345) selects `totp_seed` into it and
   `internal/handler/user_handler.go` (~218-292) marshals the model as-is.
   Any holder of READ_USER gets every user's ciphertext. Fix: `json:"-"`
   as `PwHash` already is, or a list DTO without the field. Add a handler
   test that asserts the field is absent.
2. **`DeleteAppUserGroup` returns RowsAffected as the id.**
   `internal/repository/app_user_group_repository.go:285`
   `return int(result.RowsAffected()), nil` → the handler
   (`app_user_group_handler.go` ~348) answers `"userGroupId": 1` for every
   delete. Return `groupID`.
3. **Anonymous rate limit keys all unauthenticated traffic on one bucket.**
   `internal/app_lib/app_helper.go:389` builds
   `RPM*anon*<minute>` — every anonymous client shares the limit. The
   sibling `CheckRateLimit` (~282-350) keys by IP correctly. Also:
   `getClientIP` is duplicated in `handler/auth_handler.go:24-42`; the
   `"admin"` username is hard-exempt from rate limiting (~300, 358).
   Fix: one IP extractor in app_lib, one rate-limit path, no username
   exemption (or make it a config key).
4. **`example_handler.go:64-68` has a `fmt.Printf("DEBUG: ...")`** on a
   request path. Remove.

### Web (`src/web-app/`)

5. **`utils/password-utils.ts` generates passwords with `Math.random`.**
   Use `crypto.getRandomValues`.
6. **`hooks/use-auth-guard.ts:15`** compares `location.pathname` to
   `ROUTES.LOGIN` (`"login"`, no leading slash — `constants/routes.ts:14`)
   so the check never matches. Minimal fix here (compare to `/login`);
   the systemic ROUTES fix is PS0013.

### Tracked artifacts (repo root)

7. **`etc/vault-configs/dev_proto-api-server-secrets-json_550e8400-...`**
   is tracked and holds a DATABASE_URL password, `JWT_SECRET` and a TOTP
   encryption key. `git rm --cached`, add `etc/vault-configs/` to
   `.gitignore`, commit a `.example`. (Whether the vault stays at all is
   PS0003; the file must leave git regardless.)
8. **`tests/reports/` — 17 generated HTML/`.out` files (680 KB, Dec 2025)**
   are tracked and `bin/310`/`bin/320` keep writing new ones there, so
   every test run dirties the tree. `git rm -r --cached tests/reports`,
   gitignore it. Where reports should go instead is PS0014; for now
   `tmp/test-logs/` (already gitignored) is fine.
9. **Root `package.json` + `package-lock.json`** contain only `zustand`,
   which `src/web-app/package.json` already declares. Stray `npm i` in the
   wrong directory. Delete both; add root `node_modules/` to `.gitignore`.
10. `.gitignore` also misses `tests/.pytest_cache/` and root
    `.pytest_cache/` (both present on disk).

## Acceptance

- `go build ./... && go vet ./... && go test ./...` clean in `src/api-srv`.
- `npx tsc --noEmit -p tsconfig.app.json` clean in `src/web-app`.
- `bin/200-run-tests all` green.
- `git ls-files | grep -E 'tests/reports|vault-configs|^package'` returns
  only the `.example`.
- A handler test proves `totpSeed` is not in the `/userList` body.

## Deploy needs

Go binary rebuild (items 1-4). No migration. No web build unless 5-6 ship
with it.
