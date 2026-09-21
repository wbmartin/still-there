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
merged:
deployed:
validated:
---
# PS0003 — Sunset shrmpl kv/log; in-process KV and slog are the template

**Source:** template review 2026-09-21.
**Decision (Brandon, 2026-09-21):** "Sunset the shrmpl server binaries for
logging and KV. We will wire those in if there is ever a time." And
(PS0008): "KV and log as distinct interfaces should go away." So: no
pluggable KV interface, no pluggable logger — one concrete in-process
KV store and `log/slog`.

**Vault (Brandon, 2026-09-21): goes away too.** Nothing pending; claimable.

## Background

Commit `32c4490` (2026-06-15) made `KV_BACKEND=memory` the default
everywhere; `LOG_RECEIVER_HOST_PORT` is optional and commented out in
e2e/infra envs. The template still carries the full local stack.

## Remove

| Area | Files |
|---|---|
| Go | `internal/app_lib/kv_client.go` (409 lines, external TCP KV), the `KV` interface and `KV_BACKEND` config key — `kv_memory.go` becomes the concrete store (rename to `kv.go` or `cache.go`); `internal/app_lib/logging_client.go` (custom TCP wire format) → `log/slog` to stdout/stderr; `LOG_RECEIVER_HOST_PORT` config key; the `"shrmpl"` mode literal (`config.go:326`) |
| bin | `010-download-shrmpl`, `210-run-shrmpl-app-infra`, `211-stop-shrmpl-app-infra` (also does `pkill -f vite`) |
| etc | `loc-shrmpl-kv-srv.env`, `loc-shrmpl-log-srv.env`; `bin/tools/*` (gitignored binaries — note in the doc that they can be deleted locally); `.gitignore` entry `bin/tools/*` |
| docs | `doc/050`, `doc/060`, `doc/105`, `doc/902`, `README.md` drop 010/210/211 from the start-up path; `doc/811-non-entity-notes.md` (shrmpl client notes) → PS0006's history archive |
| vault | `VaultClient` in `config.go:42-141`, the five `VAULT_*` keys, `bin/410-regenerate-vault-mtls-certs`, `etc/loc-shrmpl-vault-srv.env`, `etc/vault-configs/` (PS0002 untracks it first), 9 `etc/crt/shrmpl_vault_*.pem` (gitignored), `doc/850-mtls-troubleshooting.md` → history |

`infra/*/shrmpl-etc/` is **shrmpl-hook**, the webhook receiver for the
post-sales pipeline — a different product. Not in scope; PS0004 documents
it.

## Keep / change

- The ~300 `app.Info("EX_CR_START", ...)` call sites: the current logger
  drops the `code` argument (sends literal `"0000"`, omits it from the
  stderr line). With slog, make the code a structured attribute
  (`slog.Info(msg, "code", code, ...)`) so the codes become searchable
  rather than deleting them. Decide the wrapper signature so the call
  sites do not all change (a thin `app.Info(code, msg, kv...)` over slog
  is fine).
- Account-lockout and rate-limit paths use the KV; they keep working on
  the in-process store. The "fail open when KV unavailable" branch
  (commit 2026-06-15) becomes dead and goes.
- `etc/loc-api-srv.env` becomes self-contained: inline `DATABASE_URL`,
  `JWT_SECRET`, `TOTP_ENCRYPTION_KEY` like `etc/e2e-api-srv.env`; drop the
  10 absolute `/Users/brandon/...` paths, `TLS_NAME=local-atticus-snipper.teamtaggle.com`,
  `VAULT_SERVER=loc-proto.cnfg.li:7474`. This also fixes `bin/600`
  failing by default (PS0004 background).
- `config.go:70` hardcoded
  `/Users/brandon/Code/ProtoStack-ReactGo/etc/crt/shrmpl_vault_server_fullchain.pem`
  goes with the vault. Secrets for every environment come from the env
  file (`SERVERONLY`-redacted in the repo, as infra/ already does).

## What changed and why

Branch `TKT-PS0003-shrmpl-optional-quarantine`, six commits. The template
now has one in-process store and stdlib logging; nothing external to run
locally.

| Area | Change |
|---|---|
| KV | `kv_client.go` (TCP client + `ThisAppKVInterface`) deleted. `kv_memory.go` → `kv.go`, type `MemoryKV` → `KV`, `NewKV()`. The string-command `Batch("INCR k 1min","GET k")` emulation of the wire protocol is gone with it; the four callers (`CheckRateLimit`, `CheckRateLimitAndUserBlock`, `isAccountLocked`, `incrementFailedAttempt`) use `Incr`/`Get` directly. `AppServices.KV *KV`. |
| Logging | `logging_client.go` deleted. `logging.go`: `Logger` over `log/slog` text handler → stderr, `AddSource` trimmed to `file.go:NN`, level DEBUG (as before). Same `Debug/Info/Warn/Error(code, msg, kv...)` and `*WithCallerSkip` surface, same skip arithmetic, so none of the ~300 call sites moved; the code is now a `code=` attribute. `AppServices.Logger *Logger`; `Close()` gone. Two tests prove `code=`, source line, and wrapper attribution. |
| Fail open → closed | The store is in-process, so an error is a bug: rate limit, user block and lockout checks now log ERROR and answer 500 instead of letting the request through. `MFAVerifyHandler` already did. |
| `/auth` | Rejects empty or >50-char usernames (VARCHAR(50)) with the same 401 as a bad password, before any store or DB access — an oversize name would otherwise hit the store's 100-char key cap. |
| Config | `VaultClient`, the vault fetch in `LoadConfig`, the hard-coded `/Users/brandon/…fullchain.pem`, and the `LOG_RECEIVER_HOST_PORT` / `KV_HOST_PORT` / `KV_BACKEND` keys are gone (`config.go` 376 → 200 lines). `ReadEnvFile` unchanged. |
| Tests | `internal/repository` and `internal/migrate` no longer fall back to `etc/loc-api-srv.env` when `PROTO_TEST_DATABASE_URL` is unset — that is the developer's live DB, and the fallback was only inert because the vault made `LoadConfig` fail. They skip, naming `bin/200-run-tests go`. |
| `etc/` | `loc-api-srv.env` self-contained: inline `DATABASE_URL` (roles.sql default), local-only `JWT_SECRET`, 64-hex `TOTP_ENCRYPTION_KEY`, relative TLS paths, no `/Users/`, no `TLS_NAME`, no `VAULT_*`. `e2e-api-srv.env` likewise (its TOTP key was not hex and would have failed MFA setup). `uat-api-srv.env.example`, `infra/{uat,prd}/etc/*-api.env`, `infra/uat/etc/uat-proto-e2e-api.env` lose the three keys. Deleted `loc-shrmpl-{kv,log,vault}-srv.env`, `vault-configs/`. |
| `bin/` | Deleted `010`, `210`, `410`. `211` → `216-stop-local-stack` (API + vite only). `.gitignore` drops `bin/tools/*` and the `vault-configs` lines. |
| Docs | `README.md`, `src/api-srv/README.md`, `doc/050`, `060`, `105`, `901`, `902`, `000-TODO` updated; `doc/811`, `doc/850` → `doc/archive/` with a header note. |

## Files modified

- `src/api-srv/cmd/server/main.go`
- `src/api-srv/internal/app_lib/`: `kv.go` (was `kv_memory.go`), `kv_test.go`, `logging.go` (new), `logging_test.go` (new), `config.go`, `services.go`, `rbac.go`, `app_helper.go`; deleted `kv_client.go`, `logging_client.go`
- `src/api-srv/internal/handler/`: `auth_handler.go`, `common.go`
- `src/api-srv/internal/repository/testdb_test.go`, `internal/migrate/migrate_test.go`
- `src/api-srv/README.md`, `README.md`, `doc/000-TODO.md`, `doc/050-dev-env-init.md`, `doc/060-dev-env-start.md`, `doc/105-api-srv-tech-spec.md`, `doc/901-naming-preferences.md`, `doc/902-project-commands.md`, `doc/archive/811-non-entity-notes.md`, `doc/archive/850-mtls-troubleshooting.md` (moved)
- `etc/loc-api-srv.env`, `etc/e2e-api-srv.env`, `etc/uat-api-srv.env.example`, `infra/uat/etc/uat-proto-api.env`, `infra/uat/etc/uat-proto-e2e-api.env`, `infra/prd/etc/prd-proto-api.env`
- `bin/216-stop-local-stack` (was `211`); deleted `bin/010-download-shrmpl`, `bin/210-run-shrmpl-app-infra`, `bin/410-regenerate-vault-mtls-certs`, `etc/loc-shrmpl-*-srv.env`, `etc/vault-configs/*.example`
- `.gitignore`

## Endpoints introduced or modified

No new endpoints. Behaviour changes on existing ones:

| Endpoint | Auth | Change |
|---|---|---|
| `POST /auth` | anonymous | empty or >50-char `userName` → 401 `INVALID_CREDS` immediately |
| every handler using `CheckRateLimit` / `CheckRateLimitAndUserBlock` / `CheckUserBlocked` | any | store error → 500 (was: request allowed through) |

Environment: local → UAT on the next deploy.

## Acceptance (met 2026-09-21)

- [x] Fresh-clone path: `bin/100`, `bin/215`, `bin/220`, `bin/200-run-tests all` run with no shrmpl binary and no edits — `bin/200 all`: go=105 e2e=3 py=63 (pytest covers lockout, rate limit, user block).
- [x] `grep -rn shrmpl src bin etc doc README.md CLAUDE.md` → only `doc/archive/`, `infra/` (`shrmpl-hook`) and the deploy docs' `shrmpl-hook` mentions.
- [x] `grep -rn /Users/ src bin etc` → nothing.
- [x] `bin/215` output carries the code: `level=INFO source=auth_handler.go:97 msg="Auth request started" service=proto-go-app code=AUTH_REQ_STRT username=preauth`.
- [x] `go build ./... && go vet ./... && go test ./...` clean; `go test` without the runner skips the integration suites instead of touching the dev DB.

## Waiting for

The next UAT deploy (with PS0002, PS0004). The live UAT env file loses
`KV_BACKEND` / `LOG_RECEIVER_HOST_PORT` (harmless if left; the loader
ignores unknown keys).

## Deploy needs

Go binary. UAT `etc/api-srv.env`: remove `KV_BACKEND`, `KV_HOST_PORT`,
`LOG_RECEIVER_HOST_PORT`, any `VAULT_*` (all optional to remove). No
migration, no web build.

**Local developers:** delete `bin/tools/`, `etc/vault-configs/`,
`etc/crt/shrmpl_vault_*.pem` if present (this session did). MFA seeds in
`loc_starter_db` were encrypted under the old dev key and will not
decrypt: `POST /admin/resetUserMFA` or `bin/510`.

## Related

PS0002 (vault-configs out of git), PS0008 (package split after this),
PS0007 (checklist names what survives).
