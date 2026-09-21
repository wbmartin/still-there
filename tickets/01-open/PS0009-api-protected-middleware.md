---
id: PS0009
title: Replace the AppHelper check state machine with Protected(grant) middleware; one handler style; route registry
status: needs-triage
blocked_by:
waiting_on:
priority: high
domain: api
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
merged:
deployed:
validated:
---
# PS0009 — Security checks are a stateful orchestrator, not middleware

**Source:** template review 2026-09-21. `doc/110-api-server-code-review.md`
(2025-12-24) called this critical; it is still open.

## Problem

`internal/app_lib/app_helper.go:21-35` — every request constructs an
`AppHelper` carrying five `*Checked` booleans (JWT, rate limit, CSRF,
authorization, …) and `Skip*` methods. Handlers call the checks in
sequence and then `CheckAllChecksComplete`. Consequences:

- `HealthHandler` must call five `Skip*` methods to say "public"
  (`handler/health_handler.go:13-17`).
- The auth handlers are closures (`auth_handler.go`, `user_mfa_handler.go`:
  `AuthHandler`, `RefreshHandler`, `MFASetupHandler`, `UserSelfCreateHandler`
  …) that build an AppHelper but **never call `CheckAllChecksComplete`**
  (`auth_handler.go:159`), so the invariant is unenforceable for exactly
  the endpoints that matter most.
- Every entity handler method repeats the same four check lines + the
  complete check (`example_handler.go:76-108, 284-332, 405-450`, and the
  same in user/group handlers). That is most of the ~900 lines of
  per-entity ceremony (rest is PS0010).
- `app_helper.go:52` logs INFO "AppHelper instantiated" on every request.
- `auth_handler.go:148, 409, 539` take `appUserGroupRepo` params that are
  unused. The auth cookie struct is built four times (`:368, 499, 677,
  740`).

Two handler construction styles coexist: struct receivers
(`ExampleHandler`, `UserHandler`, `AppUserGroupHandler`) vs closures (auth,
MFA). Closures cannot share helpers like a `setAuthCookie`.

The route table (`cmd/server/main.go:127-207`) is 30 hand-wired
`mux.HandleFunc` lines mixing both styles, constructs
`adminResetMFAHandler` mid-list (`:184`), and mixes path conventions:
`/example`, `/exampleList`, `/admin/resetUserMFA`, `/error-report`.

Only `UserHandler` is built on interfaces (`user_handler.go:17-104`
`UserRepositoryInterface`, `AppUserGroupRepositoryInterface`) and so is
the only mockable, tested handler. `ExampleHandler` — the copy-me — takes
`*repository.ExampleRepository` concretely (`example_handler.go:15-18`).

## Target

- `Protected(grant Grant) func(http.HandlerFunc) http.HandlerFunc` (or
  `app.Secure(...)`) that runs JWT → rate limit → CSRF → authorization and
  puts claims on the context. Public routes are simply unwrapped. No
  booleans, no `CheckAllChecksComplete`, nothing to forget.
- Public-but-rate-limited (login, self-register, error-report) is a
  second wrapper, `RateLimited(...)`.
- One handler style: struct with receiver methods, constructed from
  interfaces, each exposing `Routes(mux *http.ServeMux)`. `main.go`
  becomes `for _, h := range handlers { h.Routes(mux) }`.
- One path convention (decide: `/example` + `/exampleList` as today, or
  `/examples` REST-ish; keep the web `constants/api.ts` in step — PS0013).
- `setAuthCookie(w, token)` helper; drop unused params.

## What to do

1. Write the middleware with tests (pure functions over `httptest`; there
   are currently **no** tests for `app_helper.go`, `auth.go`, `rbac.go`).
2. Convert handlers one at a time, starting with `ExampleHandler` (it is
   the template) and adding the interface + a mocked test as
   `user_handler_test.go` does.
3. Convert the auth/MFA closures to a struct.
4. Delete `AppHelper`, the `Skip*` methods, the health-handler skips.

## Acceptance

- `grep -rn 'CheckAllChecksComplete\|Skip[A-Z]' internal` → nothing.
- Every route in `main.go` is either wrapped in `Protected`/`RateLimited`
  or is on an explicit public list in one place.
- pytest suite (`bin/200 py`) green — it is the black-box check on
  auth/CSRF/rate-limit/isolation behaviour.
- Handler tests exist for example, user, group, auth.

## Deploy needs

Go binary. Behaviour should be identical from the client's view; if the
path convention changes, web build + `constants/api.ts` in the same PR.

## Related

PS0008 (do the package move first or together), PS0010, PS0002 item 3
(anon rate-limit key — fix there, fold here).

## Direction (Brandon, 2026-09-21)

**Not aligned. Discuss in depth before any work.** Do not claim.

Points to settle in that discussion (agent's read of where the
disagreement may lie — Brandon to confirm or correct):

1. Is the explicit per-handler check sequence (`CheckJWT`, `CheckRate…`,
   `CheckCSRF`, `CheckAuthorization`, `CheckAllChecksComplete`) a
   deliberate design — checks visible in the handler body, varied per
   endpoint, greppable — rather than an accident? If so, the ticket's
   framing ("state machine instead of middleware") is wrong and the
   real defects are narrower: the closures that never call
   `CheckAllChecksComplete`, the unused params, the four cookie builds.
2. Which endpoints legitimately need a *different* check set (login: no
   JWT, rate-limited by IP; MFA setup: JWT but no grant; health: nothing;
   error-report: none but body cap)? A middleware chain must express
   each of these as clearly as the current code does or it is a loss.
3. Handler style: is struct-vs-closure a real problem or cosmetic? The
   cost is helper sharing (`setAuthCookie`) and mockability.
4. Route table: is a `Routes(mux)` per handler wanted, or is one flat
   list in `main.go` preferred *because* it is the one place to read
   every route?
5. The `ExampleHandler`-on-interfaces point stands on its own (it is
   the copy-me and is the only untestable handler); could be split out.

PS0010 is blocked behind this.
