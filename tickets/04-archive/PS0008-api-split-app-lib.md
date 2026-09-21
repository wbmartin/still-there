---
id: PS0008
title: Split internal/app_lib into packages named by role (no KV/log interfaces)
status: in-review
blocked_by:
waiting_on:
priority: medium
domain: api
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0008-api-split-app-lib
merged: 5ce23c6
deployed: none
validated: 2026-09-21
---
# PS0008 — `app_lib` is a grab-bag named by history

`src/api-srv/internal/app_lib/` held config, auth, the security-check
orchestrator, the in-process KV store, RBAC, and security headers in one
undifferentiated package (`golangci-lint` flagged the name three times).
Brandon: KV and log stay concrete types, not interfaces. Split into six
role-named packages, pure mechanical move, behaviour unchanged.

## What changed
- `internal/app_lib` → `internal/config`, `internal/cache` (concrete
  `KV`, no interface), `internal/logging` (concrete `Logger`, no
  interface), `internal/auth` (JWT, argon2id password hashing, RBAC,
  and new `totp.go`), `internal/security` (`AppHelper` as-is, rate
  limit, user block, `ClientIP`, headers/CORS middleware),
  `internal/app` (`Services{KV, Config, Logger}` — `DBPool` dropped).
- TOTP secret generation, otpauth URI, AES-GCM seed encrypt/decrypt, and
  code validation moved out of `handler/user_mfa_handler.go` into
  `internal/auth/totp.go`, exported. Handler package no longer imports
  any crypto package directly.
- Two forced renames (revive stutter rule): `app_lib.AppServices` →
  `app.Services` (20 call sites), `SecurityHeadersMiddleware` →
  `security.HeadersMiddleware` (1 call site).
- `DBPool` removed from the DI struct — no handler used it; repositories
  are now the only path to the database.

## Rules that survived
- No Go interface exists whose only implementation is the KV store or
  the logger — both stay concrete types by design.
- All cryptography (JWT, password hashing, TOTP, AES-GCM) lives in one
  package (`internal/auth`), reviewable in one place.
- A pure package-move ticket proves itself with `go build`/`vet`/`test`
  green at every commit and a stable or falling lint count, not a
  behaviour argument.

## Figures at close
| | date |
|---|---:|
| golangci-lint findings | 53 → 50 (revive 29 → 26; 3 package-name findings gone) |
| bin/200-run-tests | go=105 e2e=3 py=63 (unchanged) |
| files renamed (git-detected) | 9 |
| new files | `internal/auth/password.go`, `internal/auth/totp.go` |

## Endpoints
None. No config-key changes.

## Security
Mechanical move with no behaviour change (git detects 9 renames; the
only new code is a one-line `ValidateTOTPCode` wrapper and six newly
exported TOTP functions). Consolidating all crypto into `internal/auth`
means JWT, argon2id, and TOTP/AES-GCM review starts in one directory
instead of being split across handler and lib code. `AppHelper` (the
explicit per-handler check chain) moved unchanged into
`internal/security` — ordering and fail-closed behaviour from
PS0002/PS0003 untouched, pytest 401/403/429/lockout coverage stayed
green. `DBPool` leaving `Services` closes an unused path from handlers to
the raw database connection.

## Left open
- `security.ClientIP` still trusts the first `X-Forwarded-For` entry —
  pre-existing PS0002 finding, awaiting Brandon's call.
- Remaining 50 lint findings (receiver naming, `userIds`, context-as-
  argument) are PS0011's, left untouched on purpose.

## Sources
- `tickets/04-archive/engineering-log/260921_1610-TKT-PS0008-api-split-app-lib.md`
- git: `5ce23c6`
