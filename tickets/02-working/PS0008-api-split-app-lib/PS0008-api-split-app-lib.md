---
id: PS0008
title: Split internal/app_lib into packages named by role (no KV/log interfaces)
status: blocked
blocked_by: [PS0003]
waiting_on:
priority: medium
domain: api
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0008-api-split-app-lib
merged:
deployed:
validated:
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

## Acceptance

- No `internal/app_lib`. No revive package-name findings.
- No Go interface whose only implementation is the in-process store or
  the logger.
- `grep -rn 'aes\|totp' internal/handler` → only calls into `internal/auth`.

## Deploy needs

Go binary. No config-key changes.
