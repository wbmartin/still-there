---
id: PS0002
title: Security and correctness hygiene found in the 2026-09-21 template review
status: in-review
blocked_by:
waiting_on:
priority: high
domain: security
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0002-security-correctness-hygiene
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
   Two limiters exist. `CheckRateLimit` (`internal/app_lib/app_helper.go:288`)
   keys anonymous callers by client IP (honours `X-Forwarded-For`); used
   by 2 handlers. `CheckRateLimitAndUserBlock` (`:352`) keys them as the
   literal `RPM*anon*<minute>` (`:389`); used by 27. Callers that reach
   it with no claims: `MFASetupHandler` and `MFAVerifyHandler`
   (`user_mfa_handler.go:298, 490` — the temp-token step between
   password and MFA), plus refresh/change-password/logoff when the JWT
   check already failed. With `MAX_UNAUTH_REQ_PER_MIN=20` every
   not-yet-authenticated user shares 20 req/min on the MFA step:
   twenty concurrent logins or one hostile client → 429 for everyone.
   **Fix (agreed with Brandon 2026-09-21):** when claims are nil,
   `CheckRateLimitAndUserBlock` uses the same IP key as `CheckRateLimit`,
   via one shared `clientIP(r)` helper in app_lib that also replaces the
   copy at `handler/auth_handler.go:24-42`. Both functions stay; no call
   site changes. The `"admin"` exemption from rate limiting is
   **deliberate — leave it.**
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

## Root cause or requirements

Ten independent defects from the 2026-09-21 review; no design decision
in any of them. Items 1-4 are API bugs, 5-6 web bugs, 7-10 tracked
artifacts that should never have been committed.

## What changed and why

Branch `TKT-PS0002-security-correctness-hygiene`, three commits, all ten items.

| # | Change | Why |
|---|---|---|
| 1 | `AppUserModel.TOTPSeed` tagged `json:"-"` (as `PwHash`). New `TestRetrieveUserList_OmitsSecrets` marshals the exact `DataWrapper` `/userList` writes and asserts `totpSeed`/`pwHash` absent. | READ_USER holders were getting every user's encrypted seed. Nothing decodes `totpSeed` from JSON; the seed only moves via `UpdateUserTOTPSeed`. |
| 2 | `DeleteAppUserGroup` returns `groupID` instead of `RowsAffected()`. | Every delete answered `userGroupId: 1`. |
| 3 | New `app_lib.ClientIP(r)` (first valid `X-Forwarded-For` entry, else `RemoteAddr` host). `CheckRateLimit` and `CheckRateLimitAndUserBlock` both key anonymous callers by it; `handler.getClientIP` deleted, its two callers (login lockout, MFA verify) use the shared one. `"admin"` exemption untouched. | 27 handlers shared one `RPM*anon*<minute>` bucket, so the MFA step 429'd for everyone at 20 req/min. |
| 4 | Removed `fmt.Printf("DEBUG: ...")` from `RetrieveExample`. | stdout noise on a request path. |
| 5 | `generateStrongPassword` uses `crypto.getRandomValues` with rejection sampling (`limit = 256 - 256 % 70`). | `Math.random` is not a CSPRNG; plain modulo would bias the first 46 characters. |
| 6 | `use-auth-guard` compares `location.pathname` to `` `/${ROUTES.LOGIN}` ``. | `ROUTES.LOGIN` is `"login"` so the TBD-on-login check never matched. Systemic fix is PS0013. |
| 7 | `git rm --cached` the vault secrets file; `.example` committed with placeholders; `etc/vault-configs/*` ignored except `*.example`. | Live DATABASE_URL password, JWT_SECRET, TOTP key in git. **Still in history — see Security_Considerations.** |
| 8 | `git rm -r --cached tests/reports`; `tests/reports/` ignored. `bin/310`/`320` still write there (unchanged; PS0014). | 680 KB of generated HTML; every run dirtied the tree. |
| 9 | Root `package.json` + `package-lock.json` deleted; `/node_modules/` ignored. | Stray `npm i` at root; `zustand` already in `src/web-app`. |
| 10 | `.pytest_cache/` ignored (matches root and `tests/`). | Present on disk, untracked noise. |

## Files modified

- `.gitignore`
- `etc/vault-configs/dev_proto-api-server-secrets-json_550e8400-….example` (new); the non-`.example` file untracked
- `package.json`, `package-lock.json` (deleted); `tests/reports/*` (17 files untracked)
- `src/api-srv/internal/app_lib/app_helper.go`
- `src/api-srv/internal/handler/auth_handler.go`, `user_mfa_handler.go`, `example_handler.go`, `user_handler_test.go`
- `src/api-srv/internal/model/app_user_model.go`
- `src/api-srv/internal/repository/app_user_group_repository.go`
- `src/web-app/hooks/use-auth-guard.ts`, `src/web-app/utils/password-utils.ts`

## Endpoints introduced or modified

No new endpoints, no path or method changes. Response-shape changes:

| Endpoint | Auth | Change | Data |
|---|---|---|---|
| `GET /userList` | JWT + CSRF + READ_USER | `totpSeed` field removed from each user object | credentials (encrypted TOTP seed) — no longer returned |
| `DELETE /userGroup` | JWT + CSRF + grant | `userGroupId` now the deleted id, not `1` | none |
| every handler using `CheckRateLimitAndUserBlock` with no valid JWT (MFA setup/verify temp-token step, failed refresh/change-password/logoff) | anonymous | 429 threshold now per client IP instead of one global bucket | none |

Environment: local now → UAT on Brandon's deploy.

## Acceptance (met 2026-09-21)

- `go build ./... && go vet ./... && go test ./...` clean in `src/api-srv`.
- `npx tsc --noEmit -p tsconfig.app.json` and eslint clean in `src/web-app`.
- `bin/200-run-tests all`: go=106 e2e=3 py=63, all passed; tree clean after the run.
- `git ls-files | grep -E 'tests/reports|vault-configs|^package'` → only the `.example`.
- `TestRetrieveUserList_OmitsSecrets` proves `totpSeed` absent from the `/userList` body.

## Deploy needs

- Go binary rebuild (items 1-4).
- Web build (items 5-6).
- No migration.
- Server-side: nothing — the vault file is read from the server's own `etc/vault-configs/`, untouched by this change. **Rotate** the three secrets that were in git history (see Security_Considerations).

## Direction (Brandon, 2026-09-21)

"Just looks like fixes." Proceed as written; no decisions pending.
