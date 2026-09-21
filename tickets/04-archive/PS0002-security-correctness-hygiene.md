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
merged: ea5f4f0
deployed: none
validated: 2026-09-21
---
# PS0002 — Security and correctness hygiene

Ten independent defects surfaced by a four-pass read-only review of the
template on 2026-09-21 (api-srv, web-app, bin/etc/infra/tests, docs);
none needed a design decision, so they were fixed together in three
commits.

## What changed
- `AppUserModel.TOTPSeed` tagged `json:"-"`; `GET /userList` no longer returns encrypted TOTP seeds. `TestRetrieveUserList_OmitsSecrets` asserts absence.
- `DeleteAppUserGroup` returns the real `groupID` instead of `RowsAffected()` (was always `1`).
- New `app_lib.ClientIP(r)` shared by `CheckRateLimit` and `CheckRateLimitAndUserBlock`; the latter's 27 handlers moved off one global `RPM*anon*<minute>` bucket to per-IP keys. `handler.getClientIP` deleted.
- Removed a `fmt.Printf("DEBUG: ...")` on the `RetrieveExample` request path.
- `generateStrongPassword` (web) moved from `Math.random` to `crypto.getRandomValues` with rejection sampling (70-char alphabet, no modulo bias).
- `use-auth-guard` TBD branch now compares `location.pathname` to `` `/${ROUTES.LOGIN}` `` (was comparing to `"login"`, never matched).
- `git rm --cached` on `etc/vault-configs/dev_proto-api-server-secrets-json_...` (live DATABASE_URL password, JWT_SECRET, TOTP key); `.example` committed; folder gitignored except `*.example`.
- `git rm -r --cached tests/reports` (17 files, 680 KB); root `package.json`/`package-lock.json` (stray `zustand` dupe) deleted; `.gitignore` gains `node_modules/`, `.pytest_cache/`.

## Rules that survived
- A list endpoint returns only what the caller needs — an encrypted secret field is not exempt just because it is ciphertext.
- Anonymous rate-limit buckets are keyed by client IP, never a single shared literal, so one hostile client cannot 429 every not-yet-authenticated user.
- The `"admin"` exemption from rate limiting is deliberate — leave it (Brandon, 2026-09-21).
- Nothing that looks like a secrets file (`etc/vault-configs/`, `tests/reports/`, stray root `package.json`) gets tracked; gitignore it and ship a `.example`.

## Figures at close
| | date |
|---|---:|
| `go build/vet/test` (api-srv) | clean |
| `tsc --noEmit` + eslint (web-app) | clean |
| `bin/200-run-tests all` | go=106 e2e=3 py=63, all passed |
| `git ls-files \| grep -E 'tests/reports\|vault-configs\|^package'` | only the `.example` |

## Endpoints
- `GET /userList` — JWT+CSRF+READ_USER, `totpSeed` field removed from each user object.
- `DELETE /userGroup` — JWT+CSRF+grant, `userGroupId` now the deleted id, not `1`.
- Every handler on `CheckRateLimitAndUserBlock` reached with no valid JWT (MFA setup/verify temp-token step, failed refresh/change-password/logoff) — anonymous, 429 threshold now per client IP instead of one global bucket.

## Security
Net improvement, but three live secrets (DATABASE_URL password, JWT_SECRET, TOTP_ENCRYPTION_KEY) were tracked in `etc/vault-configs/` and remain in git history even after untracking — rotation of all three (plus optional `git filter-repo` history purge) is required outside this PR and is urgent if UAT/prd ever reused any of the `dev_` values. Rate-limit keying is now per-IP rather than one shared bucket, but `ClientIP` still trusts the first `X-Forwarded-For` entry while both nginx configs *append* the real peer IP (`$proxy_add_x_forwarded_for`) — a client can already pick its own bucket per request and evade IP-keyed limits and per-IP login lockout (per-username lockout is unaffected); this was pre-existing on two other handlers and is now extended to 27, not newly introduced. Recommended follow-up: key on the *last* XFF entry or have nginx overwrite rather than append — deferred as a scope decision for Brandon.

## Left open
- X-Forwarded-For trust (first-entry vs last-entry) — one-line follow-up for Brandon, not done here because it changes lockout semantics beyond agreed scope.
- Secret rotation (JWT_SECRET, TOTP_ENCRYPTION_KEY, DATABASE_URL password) and optional history purge — outside this PR.
- Where `tests/reports/` should write instead of the gitignored default is PS0014.
- Validation waits on the next UAT deploy of ProtoStack itself (not scheduled at close).

## Sources
- git: `ea5f4f0`
