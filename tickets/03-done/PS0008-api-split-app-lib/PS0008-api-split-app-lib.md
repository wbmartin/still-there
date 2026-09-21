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

**Source:** template review 2026-09-21.
**Decision (Brandon, 2026-09-21):** "KV and log as distinct interfaces
should go away." Combined with PS0003 (sunset shrmpl kv/log): the split
does **not** create `internal/kv` and `internal/logx` packages with
interfaces. The in-process store is one concrete type; logging is
`log/slog`. Blocked on PS0003 so the move happens once, on the smaller
tree.

## Problem

`src/api-srv/internal/app_lib/` holds everything that is not a handler,
model or repository:

| File | Contents |
|---|---|
| `config.go` | env-file parser, `Config` struct (VaultClient gone with PS0003) |
| `auth.go` | JWT sign/verify (HS256), Argon2id password hashing |
| `app_helper.go` | per-request security check orchestrator (`AppHelper`), two rate-limit implementations, user-block check |
| `kv_memory.go` | in-process KV store (the only one after PS0003) |
| `rbac.go` | grant checks |
| `security_headers.go` | middleware |
| `services.go` | `AppServices` DI struct — exposes `DBPool` to handlers (no handler uses it) |

`golangci-lint` flags the package name three times (revive: underscore).
`handler/user_mfa_handler.go:33-133` owns TOTP secret generation and
AES-GCM seed encryption that belong beside the other crypto.

## Target (proposal — names are Brandon's call)

```
internal/config/    env loader, Config struct
internal/auth/      jwt.go, password.go (argon2), totp.go (from user_mfa_handler), rbac.go, claims
internal/security/  AppHelper as-is (explicit per-handler checks are by design — PS0009), rate limit, user block, client IP, security headers
internal/cache/     the in-process store (concrete struct, no interface)
internal/app/       AppServices without DBPool
```

## What to do

- Move files; fix imports; **behaviour identical**. `go build`, `go vet`,
  `go test ./...` green at every commit. Pure mechanical PR.
- Drop `DBPool` from `AppServices`.
- Logging: after PS0003 the call sites use whatever thin wrapper PS0003
  chose; no separate package unless the wrapper needs one.
- Do not fix logic here — PS0009/PS0010/PS0011 own that and are on hold.
  Moving first makes their eventual diffs readable.

## What changed and why

Branch `TKT-PS0008-api-split-app-lib`, one commit. Behaviour identical;
`go build`, `go vet`, `go test ./...`, `bin/200-run-tests all` (go=105
e2e=3 py=63) green. Names follow the ticket's proposal, with one extra
package because the logger needs a home that neither `app` nor `auth`
can be without a cycle.

| Package | From | Contents |
|---|---|---|
| `internal/config` | `config.go` | `ReadEnvFile`, `LoadConfig`, `Config` |
| `internal/cache` | `kv.go`, `kv_test.go` | `KV`, `NewKV` — concrete, no interface |
| `internal/logging` | `logging.go`, `logging_test.go` | `Logger`, `NewLogger` — concrete, no interface |
| `internal/auth` | `auth.go` → `jwt.go` + `password.go`; `rbac.go`; **new `totp.go`** | `Claims`, `GenerateJWT`, `ValidateJWT`; `HashPassword`, `CheckPasswordHash`; `RBACManager`; `GenerateTOTPSecret`, `ValidateTOTPSecret`, `GenerateTOTPURI`, `EncryptTOTPSeed`, `DecryptTOTPSeed`, `ValidateTOTPCode` — moved out of `handler/user_mfa_handler.go` (lines 33-133 plus the bare `totp.Validate` call), exported, unchanged |
| `internal/security` | `app_helper.go`, `security_headers.go` | `AppHelper` as-is (explicit per-handler checks stay — PS0009), rate limit, user block, `ClientIP`, `HeadersMiddleware`, `CORSMiddleware` |
| `internal/app` | `services.go` | `Services{KV, Config, Logger}` — `DBPool` dropped (no handler used it) |

Two renames forced by revive's stutter rule on the new package names,
both mechanical: `app_lib.AppServices` → `app.Services` (20 sites),
`SecurityHeadersMiddleware` → `security.HeadersMiddleware` (1 site).

Dependency graph, no cycles: `handler` → `security`, `app`, `auth`,
`cache`, `config`, `logging`; `security` → `app`, `auth`, `logging`,
`model`; `app` → `cache`, `config`, `logging`; `auth` → `logging`,
`model`.

`golangci-lint`: 53 → 50 findings (revive 29 → 26). The three
package-name findings are gone; nothing new; the remaining 50 are
pre-existing and PS0011's.

## Files modified

- moved (git detects 9 renames): `internal/app_lib/{config,kv,kv_test,logging,logging_test,auth,rbac,app_helper,security_headers,services}.go` → the packages above
- new: `internal/auth/password.go` (split from `auth.go`), `internal/auth/totp.go`
- `cmd/server/main.go`, `cmd/migrate/main.go`, every `internal/handler/*.go` (import + qualifier changes only; `user_mfa_handler.go` also loses the 100 lines now in `auth/totp.go`)
- `src/api-srv/README.md`, `doc/105-api-srv-tech-spec.md` (paths)

## Endpoints introduced or modified

None. No config-key changes.

## Acceptance (met 2026-09-21)

- [x] No `internal/app_lib`. No revive package-name findings.
- [x] No Go interface whose only implementation is the store or the logger (`grep -rn "interface {" internal` → the two repository interfaces in `user_handler.go` only).
- [x] `grep -rn 'aes\|totp' internal/handler` → only `auth.*TOTP*` calls, field names and message strings.

## Deploy needs

Go binary. No config-key changes.

## Waiting for

The next UAT deploy.
