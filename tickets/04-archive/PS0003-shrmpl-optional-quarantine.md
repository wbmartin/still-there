---
id: PS0003
title: Sunset the shrmpl kv, log and vault clients and all their local tooling
status: in-review
blocked_by:
waiting_on:
priority: high
domain: tooling
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0003-shrmpl-optional-quarantine
merged: b514d5c
deployed: none
validated: 2026-09-21
---
# PS0003 — Sunset shrmpl kv/log/vault; in-process KV and slog are the template

Brandon (2026-09-21): "Sunset the shrmpl server binaries for logging and
KV. We will wire those in if there is ever a time," and the vault goes
away too. `KV_BACKEND=memory` had already been the default everywhere
since 2026-06-15, so six commits removed the pluggable interfaces and the
external TCP clients entirely, leaving one concrete in-process `KV` store
and `log/slog`.

## What changed
- `kv_client.go` (409-line TCP client) deleted; `kv_memory.go` → `kv.go`, type `MemoryKV` → `KV`. The `Batch("INCR k 1min","GET k")` string-command wire-protocol emulation is gone; callers use `Incr`/`Get` directly.
- `logging_client.go` (custom TCP wire format) deleted; `Logger` now wraps `log/slog` to stderr, code as a `code=` structured attribute, `runtime.Callers(skip+1)` reproduces the old caller-frame arithmetic (proven by test).
- Vault removed: `VaultClient`, the mTLS fetch in `LoadConfig`, the hard-coded `/Users/brandon/...fullchain.pem`, five `VAULT_*` keys, `bin/410-regenerate-vault-mtls-certs`, `etc/vault-configs/`, 9 `etc/crt/shrmpl_vault_*.pem`.
- Four "KV outage → fail open" branches (`CheckRateLimit`, `CheckRateLimitAndUserBlock`, `CheckUserBlocked`, `AuthHandler` lockout) now fail closed → 500, since the only possible store error on an in-process store is a bug.
- `POST /auth` rejects empty or >50-char usernames (VARCHAR(50)) with the same 401 as a bad password, before any store/DB access — a longer name would otherwise have hit the store's 100-char key cap.
- `bin/010-download-shrmpl`, `210-run-shrmpl-app-infra`, `410-regenerate-vault-mtls-certs` deleted; `211` → `216-stop-local-stack`.
- `etc/loc-api-srv.env` and `etc/e2e-api-srv.env` made self-contained (inline `DATABASE_URL`/`JWT_SECRET`/64-hex `TOTP_ENCRYPTION_KEY`, no `/Users/` paths, no `VAULT_*`).
- `internal/repository` and `internal/migrate` integration tests no longer fall back to `etc/loc-api-srv.env` (the developer's live DB) when `PROTO_TEST_DATABASE_URL` is unset — they now skip, naming `bin/200-run-tests go`.

## Rules that survived
- No pluggable KV interface, no pluggable logger — one concrete in-process store and stdlib logging (Brandon, PS0008 decision carried here).
- A store error on an in-process KV is a bug, not an outage — every rate-limit/lockout/block check fails closed, not open.
- The env file is the secret on every host now that the vault is gone — `root:proto 0640`, same convention `infra/` already used.
- Incr honours its TTL only on create, so repeated attempts cannot extend an attacker's own lockout window (now documented in `kv.go`).
- Integration test suites that need a real DB never fall back silently to a developer's live database — missing config means skip, not "use whatever is configured."

## Figures at close
| | date |
|---|---:|
| `bin/200-run-tests all` | go=105 e2e=3 py=63, all passed |
| `config.go` line count | 376 → 200 |
| `grep -rn shrmpl src bin etc doc README.md CLAUDE.md` | only `doc/archive/`, `infra/` (shrmpl-hook, a different product) |
| `grep -rn /Users/ src bin etc` | 0 |

## Endpoints
- `POST /auth` — anonymous, empty or >50-char `userName` → 401 `INVALID_CREDS` immediately.
- Every handler on `CheckRateLimit`/`CheckRateLimitAndUserBlock`/`CheckUserBlocked` — any, store error now 500 (was: request allowed through).

## Security
The removed shrmpl-kv/log/vault clients closed real attack surface: the API no longer opens outbound TCP to `KV_HOST_PORT`/`LOG_RECEIVER_HOST_PORT`, no longer speaks an unauthenticated line protocol with key names interpolated unescaped into commands, and no longer ships every log line — including usernames and client IPs — over plaintext TCP. Fail-open became fail-closed on all four rate-limit/lockout/block paths, so a bug can no longer silently disable them. slog's text handler quotes values, closing a log-injection path a hostile username could hit via the old `fmt.Fprintf`. One caveat restated in `kv.go`: two or more API processes behind a load balancer now make every limit strictly per-process — a future ticket, not a config flag, which is the point of this one. Local MFA enrolments under the old dev vault key break once and need `POST /admin/resetUserMFA` or a `bin/510` rebuild; the old dev vault-config values remain in git history per PS0002's still-open rotation.

## Left open
- Secret rotation (JWT_SECRET, TOTP_ENCRYPTION_KEY, DATABASE_URL password) — carried from PS0002, still outstanding.
- Shared (cross-process) rate-limit/lockout store — future ticket, deliberately not this one.
- Validation waits on the next UAT deploy alongside PS0002/PS0004.
- Package split after this consolidation is PS0008; the derivation checklist naming what survives is PS0007.

## Sources
- `tickets/04-archive/engineering-log/260921_1519-TKT-PS0003-shrmpl-optional-quarantine.md`
- git: `b514d5c`
