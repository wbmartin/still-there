---
id: PS0009
title: Explicit per-handler security checks are by design — record it; fix the exceptions' loose ends; make ExampleHandler mockable
status: ready
blocked_by:
waiting_on:
priority: medium
domain: api
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
merged:
deployed:
validated:
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

## Acceptance

- `doc/105` has the paragraph; `CLAUDE.md` has the line (PS0005).
- `go build`, `go vet`, `go test ./...` green; new example handler test
  passes; pytest suite green.
- No behavioural change to any endpoint (pytest is the proof).

## Deploy needs

Go binary. No config change.
