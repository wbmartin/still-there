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

**Source:** template review 2026-09-21, which proposed replacing the
`AppHelper` check sequence with `Protected(grant)` middleware.

**Decision (Brandon, 2026-09-21), verbatim:** "It is deliberate and
non-negotiable. I will not sacrifice the clarity of the code and visual
auditability for a convention of middleware. The files you listed are
exceptions, which also happen on middleware, but this puts them in a
fixed place."

So: every handler calls its checks explicitly, in order, in its own body
(`CheckJWT` → rate limit → `CheckCSRF` → `CheckAuthorization` →
`CheckAllChecksComplete`), and an endpoint that needs a different set
(login, self-register, MFA setup, health, error-report) writes that
difference in the handler where an auditor reads it. `doc/110`'s
"critical" finding on this is closed as *by design*. No middleware, no
route wrapper, no `Skip*` removal. Any future agent proposing otherwise
should read this ticket and stop.

## What remains (small)

1. **Record the principle** where agents read it: PS0005 adds the
   conventions line to `CLAUDE.md`; this ticket adds a short
   "Security checks" paragraph to `doc/105-api-srv-tech-spec.md` (the
   order, the `CheckAllChecksComplete` contract, how an exception is
   written — using `health_handler.go`'s five `Skip*` calls and the login
   handler as the two worked examples). Optionally a one-line comment at
   the top of `app_helper.go` pointing at it.
2. **The exceptions' loose ends** (do not change what they check):
   - `auth_handler.go:148, 409, 539` take `appUserGroupRepo` params that
     are unused — drop them.
   - The auth cookie struct is built four times (`auth_handler.go:368,
     499, 677, 740`) — one `setAuthCookie(w, token)` helper in the
     handler package.
   - `app_helper.go:52` logs INFO "AppHelper instantiated" on every
     request — drop or demote to debug.
   - `getClientIP` is duplicated (`app_helper.go`, `auth_handler.go:24-42`)
     — one copy (PS0002 item 3 touches the same code; coordinate).
3. **`ExampleHandler` on an interface.** `example_handler.go:15-18` takes
   `*repository.ExampleRepository` concretely; `UserHandler`
   (`user_handler.go:17-104`) defines `UserRepositoryInterface` and is
   the only mockable, tested handler. Give the example — the copy-me —
   the same shape and a `example_handler_test.go` modelled on
   `user_handler_test.go` that exercises the check sequence (e.g. a
   request without JWT gets 401 before the repo is touched). This is
   what makes the explicit sequence *testable* per handler, which is the
   design's own strength. Brandon's unit-test preference (PS0014):
   separate the mockable logic from the HTTP function that calls it, so
   the logic is tested in Go without `httptest` and the handler stays a
   readable sequence of checks + calls.

## Explicitly not in scope (needs a separate conversation if ever)

Struct-receiver vs closure handler style; per-handler `Routes(mux)` vs
the flat table in `cmd/server/main.go`; path naming conventions. The
review noted these; Brandon has not asked for them.

## What changed and why

Branch `TKT-PS0009-api-protected-middleware`, one commit. No endpoint
changes behaviour; pytest (63) and Playwright (3) are the proof.

**1. The principle, recorded.** `doc/105` "Security Checks" now states
the rule, shows the six-line shape, explains the `CheckAllChecksComplete`
contract and lists every exception with its reason (health, login,
refresh/logoff, self-create/error-report, MFA setup/verify). A header
comment atop `internal/security/app_helper.go` says the same in twelve
lines and points at the doc and this ticket. `CLAUDE.md` has the
one-liner (PS0005).

**2. Loose ends.**
- `AuthHandler`, `RefreshHandler`, `ChangeSecurityContextHandler` lose the
  unused `appUserGroupRepo` parameter; `main.go` updated.
- `setAuthCookie(w, jwt, maxAge)` replaces five identical cookie literals
  (login, MFA verify, refresh, change-context, logoff). Values preserved
  exactly — which makes visible that login/MFA set a session cookie while
  refresh/change-context set `MaxAge: 3600` against a 72-hour JWT. Not
  changed here (behaviour), noted for PS0011.
- `APP_HELPER_INIT` demoted from INFO to DEBUG (kept: it carries
  `handler=`, and dropping it would orphan `helperLocation`).
- `getClientIP` was already unified by PS0002.
- **Found:** `AuthHandler` (login) built an `AppHelper` but never ran the
  contract — no `Skip*`, no `CheckAllChecksComplete` — so its exception
  was implicit, the one thing the design forbids. Now written out: five
  `Skip*` with the reason (no JWT yet; brute force is the per-IP +
  per-username lockout, stricter than the per-minute limiter) and the
  `CheckAllChecksComplete` gate. Zero behaviour change.

**3. `ExampleHandler` is mockable and its sequence is tested.**
`ExampleRepositoryInterface` (eight methods, satisfied by
`*repository.ExampleRepository`); `NewExampleHandler` takes it.
`example_handler_test.go`: `MockExampleRepository` (testify), a real
`app.Services` with the in-process store and a buffer logger (cheap after
PS0003), and `TestExampleHandler_CheckSequence` — six cases through
`httptest`: no cookie → 401; DELETE without CSRF → 403; GET without CSRF
→ 200 (safe methods exempt, by design — documented); missing grant → 403;
blocked user → 403; all pass → repository called with the claims'
context → 200. The mock panics on any unexpected call, so "repository not
reached until every check passes" is asserted, not assumed. This is the
HTTP-level test the design's strength calls for; the ticket's "test logic
without httptest" preference applies to the handler *body* (PS0014), not
to proving the check chain.

## Files modified

- `src/api-srv/internal/handler/example_handler.go`, `example_handler_test.go` (new), `auth_handler.go`, `user_mfa_handler.go`
- `src/api-srv/internal/security/app_helper.go`, `src/api-srv/cmd/server/main.go`
- `doc/105-api-srv-tech-spec.md`

## Endpoints introduced or modified

None. `POST /auth` now passes through `CheckAllChecksComplete` with every
check skipped — same responses as before.

## Acceptance (met 2026-09-21)

- [x] `doc/105` paragraph; `CLAUDE.md` line (PS0005).
- [x] `go build`, `go vet`, `go test ./...` green; `TestExampleHandler_CheckSequence` (6 cases) passes; `bin/200-run-tests all` go=113 e2e=3 py=63.
- [x] No behavioural change — pytest green.

## Deploy needs

Go binary. No config change.

## Waiting for

The next UAT deploy.
