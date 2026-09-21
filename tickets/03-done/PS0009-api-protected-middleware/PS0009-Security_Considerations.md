# PS0009 — Security Considerations

Branch `TKT-PS0009-api-protected-middleware`. Profile **4** (auth &
permission logic). This ticket *records* the security architecture
rather than changing it; the code changes are refactors with identical
responses.

## The design, now written down

Explicit per-handler checks in a fixed order, gated by
`CheckAllChecksComplete`, which fails closed if a check failed **or was
forgotten**. That second half is the property middleware cannot give: a
new handler that omits a check does not silently run unprotected — it
returns 500 until its author states every check or skip. `doc/105` and
the `app_helper.go` header now say so, and `example_handler_test.go`
demonstrates it for the handler every new entity is copied from.

## Login's exception was implicit — now explicit

`AuthHandler` never called the contract. Nothing was unprotected — its
controls are the body cap, the username bound (PS0003), the per-IP +
per-username lockout, and argon2 verification — but an auditor reading
the handler could not tell that "no rate limit" was a decision. It now
reads: five `Skip*` with the reason, then `CheckAllChecksComplete`. Same
responses. The reason is recorded because it is a real trade-off:
`/auth` has no per-minute limiter; lockout (7 fails / 5 min per IP and
per username) is the brute-force control, stricter for a single source
and, unlike the limiter, keyed by the target account as well.

## Refactors with security surface

- `setAuthCookie`: `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/` are
  now in one place instead of five. Values byte-identical; the test suite
  logs in, refreshes and logs off through it. It also exposed that login
  issues a session cookie while refresh sets `MaxAge: 3600` against a
  72-hour JWT — pre-existing, unchanged, noted for PS0011.
- `ExampleRepositoryInterface`: an interface for the handler's dependency,
  not for the store or the logger (PS0003/PS0008 rule). It exists so the
  check chain can be tested with no database; no production path changes.
- Unused `appUserGroupRepo` params removed: less reach, nothing lost.
- `APP_HELPER_INIT` at DEBUG: it logs the handler name only.

## What did I miss?

- The test proves the chain for `RetrieveExample` and `DeleteExample`;
  the other six example endpoints have the same six lines but are not
  individually asserted. A table test over all eight is a cheap follow-up
  if wanted.
- `ClientIP` still trusts the first `X-Forwarded-For` entry (PS0002
  finding, awaiting Brandon).
