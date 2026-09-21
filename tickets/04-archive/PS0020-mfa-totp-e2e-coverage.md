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
merged: 6f426e9
deployed: none
validated: 2026-09-21
---
# PS0020 — Cover the TOTP second factor end to end

Every fixture user was `mfa_exempt`, so the temp-token → `/mfaSetup` →
`/mfaVerify` flow — the app's only second factor — had no Go, pytest or
Playwright coverage and the MFA-lockout test was a stub. A non-exempt
fixture user seeded with no TOTP secret (enrolment happens inside the
tests, so no encrypted seed has to match an environment's
`TOTP_ENCRYPTION_KEY`) plus new coverage at all three layers found and
fixed two real defects in the temp-token verify path.

## What changed
- Fixture user `mfa` (`basic123!`, Basic User group, `mfa_exempt =
  false`, no seed) added to `local-seed.sql`; `local-seed.sql` now takes
  `mfa_exempt` per row.
- **Fix 1:** neither `MFASetupHandler` nor `MFAVerifyHandler` called
  `CheckAllChecksComplete()` after `SkipJWT()` on the temp-token path, so
  a request the rate limiter had already rejected (429 written) kept
  executing the user lookup, code check and counters. Both now return at
  that point.
- **Fix 2:** `/mfaVerify` "expired" the temp token with
  `KV.Set(key, "", "0")` — `"0"` is not a valid TTL, the error was
  discarded, and the token stayed valid for its full 10-minute TTL after
  a successful verify, replayable to mint further sessions. Changed to
  `KV.Del`, error logged.
- MFA handlers switched to `UserRepositoryInterface` so they can be unit
  tested with a mock.
- Go: `user_mfa_handler_test.go` (9 cases, including the two that failed
  on the old code). pytest: `test_zz_mfa_totp.py` (`pyotp` added to
  `tests/requirements.txt`) — enrol, wrong code, right code, replay
  refused, admin reset, lockout (threshold wrong codes → 429, then
  `/auth` itself 429). Playwright: `mfa.spec.ts` with a 30-line RFC 6238
  helper (`fixtures/totp.ts`) on `node:crypto`, no dependency.

## Rules that survived
- A handler branch that calls `SkipJWT()` must still call
  `CheckAllChecksComplete()` before proceeding — skipping one check does
  not exempt a request from the others that already ran.
- A single-use token must be invalidated with a delete, not a set to an
  invalid/zero TTL; a discarded error from that call can leave a
  "consumed" credential live for its full original lifetime.
- A coverage ticket that adds the first real test of a code path should
  be expected to find a live defect, not just prove the happy path.
- A test-only fixture user for a flow the ordinary test client cannot
  complete (here, MFA) should seed with no secret and enrol inside the
  test, rather than an encrypted seed that has to match a
  per-environment key.

## Figures at close
| | date |
|---|---:|
| go tests | 189 (2026-09-21) |
| unit (vitest) | 28 (2026-09-21) |
| e2e (Playwright) | 8 (2026-09-21) |
| pytest | 67 (2026-09-21) |

## Endpoints
- `POST /mfaSetup`, `POST /mfaVerify` — no contract change; both now
  terminate on a failed security check on the temp-token path, and
  `/mfaVerify` consumes (deletes) the temp token on success instead of
  leaving it live.

## Security
Two fixes on the auth/permission path, rated medium before and closed
now. The temp-token path previously let rate-limit/user-block rejections
fall through to the TOTP comparison and counters (bounded impact — the
client still saw the 429 — but any future side effect on that path would
have escaped the limiter). More significantly, the temp token issued by
`/auth` was not actually invalidated after a successful verify and
remained valid — and replayable — for its full 10-minute TTL; exploiting
it still required a valid current TOTP code. Both are proven by tests
that failed against the pre-fix code. The new fixture user `mfa` carries
only Basic grants and no admin access, the same exposure class as the
other seeded users that `local-seed.sql` already tells operators to
change for any non-dev install; since the presales deploy ships a dump
of the local database, `mfa` will exist on UAT too, harmlessly.

## Left open
- `resetUserMFA` still leaves `mfa_setup = true` after a reset (PS0021
  §2); the current tests pass because the temp-token verify path ignores
  that flag — PS0021 should decide the intended behaviour.
- PS0017 (passkey second factor) condition (2), that TOTP be tested, is
  now met.

## Sources
- `tickets/04-archive/engineering-log/260921_1920-TKT-PS0020-mfa-totp-e2e-coverage.md`
- git: `6f426e9`
