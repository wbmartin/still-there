---
id: PS0009
title: Explicit per-handler security checks are by design — record it; fix the exceptions' loose ends; make ExampleHandler mockable
status: in-review
blocked_by:
waiting_on:
priority: medium
domain: api
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0009-api-protected-middleware
merged: 0985e19
deployed: none
validated: 2026-09-21
---
# PS0009 — The check sequence in each handler is deliberate

Template review proposed replacing the explicit per-handler
`AppHelper` check sequence with `Protected(grant)` middleware. Brandon,
verbatim: "It is deliberate and non-negotiable. I will not sacrifice the
clarity of the code and visual auditability for a convention of
middleware." This ticket records that decision, cleans up the
exceptions' loose ends, and makes the copy-me `ExampleHandler` mockable
and tested — no endpoint behaviour changed.

## What changed
- `doc/105-api-srv-tech-spec.md` gets a "Security Checks" section: the
  fixed order (`CheckJWT` → rate limit → `CheckCSRF` →
  `CheckAuthorization` → `CheckAllChecksComplete`), the
  `CheckAllChecksComplete` fail-closed contract, and every exception with
  its reason. A header comment atop `internal/security/app_helper.go`
  states the same in twelve lines.
- `AuthHandler`, `RefreshHandler`, `ChangeSecurityContextHandler` lose an
  unused `appUserGroupRepo` parameter.
- `setAuthCookie(w, token, maxAge)` replaces five duplicated cookie
  literals (login, MFA verify, refresh, change-context, logoff);
  surfaced that login/MFA set a session cookie while refresh/change-
  context set `MaxAge: 3600` against a 72-hour JWT (pre-existing,
  unchanged, noted for PS0011).
- `APP_HELPER_INIT` log line demoted from INFO to DEBUG.
- **Found and fixed:** the login handler (`AuthHandler`) built an
  `AppHelper` but never ran the contract — no `Skip*` calls, no
  `CheckAllChecksComplete` — the one implicit exception the design
  exists to forbid. Now writes five `Skip*` calls with reasons (no JWT
  yet; brute force covered by per-IP + per-username lockout, stricter
  than the per-minute limiter) plus the gate call. Zero behaviour change.
- `ExampleRepositoryInterface` (8 methods) added; `ExampleHandler` takes
  it. New `example_handler_test.go`: `TestExampleHandler_CheckSequence`,
  6 cases through `httptest` with a panicking mock, proving the
  repository is unreached until every check passes (401 no-cookie, 403
  no-CSRF on DELETE, 200 GET without CSRF since safe methods are exempt,
  403 missing grant, 403 blocked user, 200 all-pass).

## Rules that survived
- Every handler calls its security checks explicitly, in order, in its
  own body — no middleware, no route wrapper, no `Skip*` removal. Any
  future proposal to centralize this should read this ticket and stop.
- `CheckAllChecksComplete` fails closed if a check failed **or was
  forgotten** — a new handler that omits a check returns 500, it does
  not silently run unprotected. This is the property middleware cannot
  give, and it's why the design stands.
- An endpoint needing a different check set writes that difference in
  its own handler with `Skip*` calls and a stated reason — never
  implicitly by skipping the contract entirely.
- `ExampleHandler` (the copy-me handler) stays on a repository interface
  so every new entity handler inherits a testable check chain.

## Figures at close
| | date |
|---|---:|
| go build/vet/test | green |
| bin/200-run-tests | go=113 e2e=3 py=63 |
| new handler test cases | 6 (TestExampleHandler_CheckSequence) |

## Endpoints
None changed in behaviour. `POST /auth` now passes explicitly through
`CheckAllChecksComplete` with every check skipped — identical responses
to before.

## Security
This ticket records the security architecture rather than changing it;
all code changes are refactors with byte-identical responses (pytest 63
+ Playwright 3 green throughout). The load-bearing finding: login's
security exception was previously implicit (the contract was never
invoked), which is exactly the failure mode `CheckAllChecksComplete`
exists to prevent for every other handler — now made explicit with no
behaviour change. `setAuthCookie` consolidates `HttpOnly`/`Secure`/
`SameSite=Strict`/`Path=/` into one place instead of five, values
unchanged. `ExampleRepositoryInterface` exists only to make the check
chain testable without a database; no production path changes.

## Left open
- The check-chain test covers `RetrieveExample`/`DeleteExample`
  explicitly; the other six example endpoints share the same six lines
  but aren't individually asserted — cheap follow-up if wanted.
- `ClientIP` still trusts the first `X-Forwarded-For` entry — pre-
  existing PS0002 finding, awaiting Brandon's call.

## Sources
- `tickets/04-archive/engineering-log/260921_1634-TKT-PS0009-api-protected-middleware.md`
- git: `0985e19`
