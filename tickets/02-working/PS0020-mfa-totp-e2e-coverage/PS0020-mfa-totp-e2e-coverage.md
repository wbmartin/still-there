---
id: PS0020
title: TOTP MFA has no automated coverage — non-exempt fixture user and a Playwright enrol/verify test
status: ready
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

## Acceptance

- `bin/300-run-tests all` runs an enrol → verify → authenticated-call
  path in pytest and Playwright, and a lockout case in pytest.
- `test_auth_flow.py:110-112` is no longer a stub.
- PS0017 condition (2) is met.

## Deploy needs

None (tests and fixtures only; the fixture user is local/e2e seed data).
