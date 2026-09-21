---
id: PS0008
title: Split internal/app_lib into packages named by role
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
# PS0008 — `app_lib` is a grab-bag named by history

**Source:** template review 2026-09-21. Do after PS0003 (removing the
shrmpl clients shrinks what has to move) but not blocked on it.

## Problem

`src/api-srv/internal/app_lib/` holds everything that was not a handler,
model or repository:

| File | Contents |
|---|---|
| `config.go` | env-file parser (200 lines, hand-rolled), `Config` struct, **and** `VaultClient` (mTLS secrets fetch, lines 42-141) |
| `auth.go` | JWT sign/verify (HS256), Argon2id password hashing |
| `app_helper.go` | the per-request security check state machine (`AppHelper`, five `*Checked` booleans), two rate-limit implementations, user-block check, `GetContext`/`CanProceed` (unused) |
| `kv_client.go` / `kv_memory.go` | the `KV` interface, external shrmpl TCP client, in-process implementation |
| `logging_client.go` | custom TCP log client + stdout fallback |
| `rbac.go` | grant checks (two functions self-flagged "I don't think is used", :66-83) |
| `security_headers.go` | middleware |
| `services.go` | `AppServices` DI struct — exposes `DBPool` to handlers (no handler uses it; 2 refs = def + main) |

`golangci-lint` flags the package name three times (revive: underscore).
The handler package also owns crypto it should not: `user_mfa_handler.go:33-133`
generates TOTP secrets, AES-GCM-encrypts/decrypts seeds and builds otpauth
URIs.

## Target

```
internal/config/    env loader (table-driven — PS0011), Config struct; vault only if PS0003 keeps it
internal/auth/      jwt.go, password.go (argon2), totp.go (from user_mfa_handler), rbac.go, claims
internal/httpx/     AppHelper → middleware (PS0009), security headers, rate limit, client IP
internal/kv/        KV interface, memory impl (shrmpl impl only if PS0003 keeps it)
internal/logx/      slog setup (or the shrmpl client if kept)
internal/app/       AppServices without DBPool
```

Names are a proposal; the point is one package per role and no package
whose name is a historical accident.

## What to do

- Move files; fix imports; keep behaviour identical. `go build`, `go vet`,
  `go test ./...` stay green at every commit.
- Drop `DBPool` from `AppServices` so handlers can only reach the DB
  through repositories.
- Do not fix logic here — PS0009 (middleware), PS0010 (entity template),
  PS0011 (dead code, config loader) own the behaviour changes. Moving
  first makes those diffs readable.

## Acceptance

- No `internal/app_lib`. `golangci-lint run` no longer reports revive
  package-name findings.
- `grep -rn 'aes\|totp' internal/handler` → only calls into
  `internal/auth`.

## Deploy needs

Go binary. No migration, no config-key changes.
