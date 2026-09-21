# PS0020 — Security Considerations

Branch `TKT-PS0020-mfa-totp-e2e-coverage`. Profiles: **4. Auth &
permission logic** (two fixes on the MFA verify path), **3. Non-
authenticated endpoint** (`/mfaSetup`, `/mfaVerify` on the temp-token
path), **8. Config / fixtures** (a non-exempt seed user). No new endpoint,
dependency in the product (only `pyotp` in the test venv), or config.

## 4. Auth & permission logic — the two fixes

1. **Checks now terminate the temp-token path.** Before: `SkipJWT()` was
   called but `CheckAllChecksComplete()` never was on that branch, so a
   request the rate limiter or user-block check had rejected (response
   already written, `blocked = true`) carried on into the handler body.
   Effect was bounded — the client saw the 429 and a later
   `WriteHeader` is superfluous — but the TOTP comparison and counters
   still ran on a request that should have been dead, and any future
   side effect added to that path would have escaped the limiter. Now
   both handlers return at that point. `TestMFAVerifyHandler_TempTokenPath/
   rate_limit_exceeded` proves the repository is not reached; it failed
   on the old code.
2. **Temp token is now single-use.** Before: `KV.Set(key, "", "0")` was
   an invalid expiration; the error was discarded, so the token issued
   by `/auth` stayed valid for its 10-minute TTL after a successful
   verify. An attacker holding the token (it travels in the `/auth`
   response body and then in two request bodies) and able to produce a
   current code could mint additional sessions. Now `KV.Del`. Proven in
   Go (`token burnt`) and in pytest (replay → 401). Rating: medium
   before, closed now — exploitation still needed a valid TOTP code.
- **Backward compatibility:** contracts unchanged; the web client already
  discards the temp token after verify.
- **Fail closed:** unchanged for the JWT path; the temp path now fails
  closed on limiter/block outcomes too.
- **Personas:** tests cover the exempt admin/basic (unchanged), the new
  non-exempt `mfa` user through enrol, verify, replay, lockout and admin
  reset, and a signed-out browser through the real screens.

## 3. Anonymous-by-temp-token endpoints

- The reason they are public is unchanged and documented in the handler:
  there is no session before the second factor. Protection is the
  bearer temp token (32 random bytes, 10 min, now single-use), the IP
  rate limit, and the failed-attempt lockout on wrong codes — the
  lockout is now exercised (7 wrong codes → 429, and `/auth` for that
  user/IP is 429 too).
- Enumeration: unchanged — an unknown token is a flat 401.

## 8. Fixture user `mfa`

- Local/e2e seed only, but the presales deploy ships a dump of the local
  DB, so it will exist on UAT: Basic User group (no admin grants), MFA
  required, password `basic123!` like `basic`. Same exposure class as
  the existing seeded users, which `local-seed.sql` already tells the
  operator to change for any non-dev install.
- Tests reset the user (`/admin/resetUserMFA`) and reseed (`bin/515`) so
  no enrolled secret is left behind for the next run.

## What did I miss?

- `resetUserMFA` still leaves `mfa_setup = true` (PS0021 §2); the tests
  pass because the temp-token verify path ignores that flag, which is
  the behaviour PS0021 should decide on.
- The pytest lockout deliberately locks the shared IP for 5 minutes at
  the end of the run; the API process is per-run, so the next run starts
  clean. A developer running `bin/300-run-tests py` twice within 5 min
  against a long-lived API would see 429s — the suite starts its own
  API, so not in practice.
- `pyotp` is a test-venv dependency only; nothing ships.
