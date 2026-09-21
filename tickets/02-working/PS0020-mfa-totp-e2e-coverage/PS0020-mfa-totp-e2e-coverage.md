---
id: PS0020
title: TOTP MFA has no automated coverage — non-exempt fixture user and a Playwright enrol/verify test
status: in-review
blocked_by:
waiting_on:
priority: low
domain: security
raised: 2026-09-21
raised_by: PS0017 investigation
asked:
claimed: 2026-09-21 by TKT-PS0020-mfa-totp-e2e-coverage
merged:
deployed:
validated:
---
# PS0020 — Cover the TOTP second factor end to end

**Source:** PS0017 (passkey) investigation, 2026-09-21. Brandon: file it.

## What

The temp-token → `/mfaSetup` → `/mfaVerify` flow is the only second
factor and nothing exercises it: every fixture user is `mfa_exempt`,
`tests/conftest.py:42-47` fails a test the moment a temp token appears,
`tests/test_auth_flow.py:110-112` (MFA lockout) is a stub, and no Go
handler test covers `auth_handler.go` / `user_mfa_handler.go`. Any
change to that flow — PS0017 when it is lifted, or a bug fix — lands
untested. Local enrolments also cannot be reused across machines because
the seed is encrypted under each environment's `TOTP_ENCRYPTION_KEY`
(PS0003).

## Do

1. **Fixture:** a third user `mfa` (`mfa_exempt = false`) in
   `sql/fixtures/local-seed.sql`, seeded with **no** seed (`totp_seed =
   ''`, `mfa_setup = false`) so enrolment is part of the test and no
   encrypted blob has to match a key.
2. **pytest** (`tests/test_auth_flow.py`): log in as `mfa` → temp token
   → `POST /mfaSetup {tempToken}` returns a secret → compute a code with
   `pyotp` (add to `tests/requirements.txt`) → `POST /mfaVerify` → JWT
   cookie + CSRF → an authenticated call works; wrong code → 401 and the
   lockout stub becomes real (`FAILED_LOGIN_ATTEMPT_THRESHOLD` from
   `etc/loc-test.env`). Keep `conftest.py`'s "test users must be exempt"
   guard for the ordinary clients; the MFA test drives `/auth` itself.
3. **Playwright** (`tests/e2e/mfa.spec.ts`, signed-out): login as `mfa`
   → `/mfa-setup` shows the QR + secret → derive the code in the test
   (`otplib` dev dependency, or read the secret from the page and
   compute) → verify → dashboard. Reset the user's seed at the end
   (`POST /admin/resetUserMFA` as admin) so the run is repeatable; or
   have `bin/515-reseed-db` cover it.
4. **Go:** a check-sequence test for `MFAVerifyHandler` proving the
   temp-token path skips JWT and the JWT path requires it, and that a
   bad code calls `incrementFailedAttempt`.

## Root cause

The fixtures had only exempt users because the test client could not
complete a second factor; nothing then exercised the temp-token path,
so two defects in it went unnoticed (below).

## What changed and why

- **Fixture user `mfa`** (`basic123!`, Basic User group,
  `mfa_exempt = false`, no seed): `local-seed.sql` now takes `mfa_exempt`
  per row. Enrolment happens inside the tests, so no encrypted seed has
  to match an environment's `TOTP_ENCRYPTION_KEY`. `doc/050`, `doc/902`
  list the user.
- **Two defects found by the new Go test and fixed** in
  `user_mfa_handler.go`:
  1. On the temp-token path neither `MFASetupHandler` nor
     `MFAVerifyHandler` called `CheckAllChecksComplete()` after
     `SkipJWT()`. A request the rate limiter had already answered 429
     kept executing — the user lookup, the code check, the counters, a
     second (discarded) response. Now both return there.
  2. `/mfaVerify` "expired" the temp token with `KV.Set(key, "", "0")`;
     `"0"` is not a valid TTL, the error was unread, and the token stayed
     valid for its full 10 minutes — replayable with any fresh code to
     mint another session. Now `KV.Del`, error logged.
- **MFA handlers take `UserRepositoryInterface`** instead of the concrete
  repository (interface gained `UpdateUserMFASetup`; mock too) so they
  can be tested with a mock like every other handler.
- **Go tests** `user_mfa_handler_test.go` (9 cases): unknown temp token
  → 401 without a lookup; rate limit exceeded → 429 and the repository
  is not reached (failed before the fix); wrong code → 401, counter
  incremented, no cookie, token kept; valid code on first enrolment →
  `mfa_setup` written, HttpOnly cookie + CSRF issued, token burnt
  (failed before the fix); locked user → 429; JWT path: no cookie →
  401, session but never set up → 400.
- **pytest `test_zz_mfa_totp.py`** (`pyotp` added to
  `tests/requirements.txt`): password → temp token (no cookie, no
  grants) → `/mfaSetup` → wrong code 401 → right code → cookie + CSRF →
  `/me` works → **replaying the temp token is refused** → second sign-in
  (`mfaAlreadySetup: true`) → admin `resetUserMFA` → a new secret.
  Lockout case: `threshold` wrong codes → 429, then `/auth` itself is
  429. It runs last because the lockout also locks the shared IP.
  `conftest.py`'s exempt-only guard now also looks inside `user`, where
  the temp token actually sits; the `test_auth_flow.py` stub is gone.
- **Playwright `mfa.spec.ts`** (signed-out): login as `mfa` →
  `/mfa-setup` shows QR + secret → wrong code stays → right code →
  dashboard → logout → sign in again with no QR → dashboard → admin API
  session resets the user. `fixtures/totp.ts` is RFC 6238 in 30 lines
  on `node:crypto`; no dependency.

## Files modified

`src/api-srv/sql/fixtures/local-seed.sql`, `internal/handler/
user_mfa_handler{,_test}.go` (test new), `internal/handler/
user_handler{,_test}.go` (interface + mock), `tests/{conftest,
test_auth_flow}.py`, `tests/test_zz_mfa_totp.py` (new),
`tests/requirements.txt`, `src/web-app/tests/e2e/mfa.spec.ts` (new),
`tests/e2e/fixtures/totp.ts` (new), `doc/050-dev-env-init.md`,
`doc/902-project-commands.md`.

## Endpoints introduced or modified

None new. `POST /mfaSetup` and `POST /mfaVerify` (anonymous-by-temp-token
or JWT; unchanged contract) now stop when a security check has failed
and consume the temp token on success.

## Checks

`go build`, `go vet`, `tsc`, `eslint` clean; `bin/300-run-tests all`:
go=189 unit=28 e2e=8 py=67.

## Acceptance

- `bin/300-run-tests all` runs an enrol → verify → authenticated-call
  path in pytest and Playwright, and a lockout case in pytest.
- `test_auth_flow.py:110-112` is no longer a stub.
- PS0017 condition (2) is met.

## Deploy needs

Go binary (the two handler fixes). Fixture user is local/e2e seed data only; UAT is deployed from a dump, so `mfa` will exist there too — harmless, it has Basic grants and no seed.
