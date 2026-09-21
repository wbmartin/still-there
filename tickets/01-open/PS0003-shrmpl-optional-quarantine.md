---
id: PS0003
title: Sunset the shrmpl kv, log and vault clients and all their local tooling
status: ready
blocked_by:
waiting_on:
priority: high
domain: tooling
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
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

## Acceptance

- Fresh clone: `bin/100`, `bin/215`, `bin/220`, `bin/200 all` run with no
  shrmpl binary and no absolute-path edits.
- `grep -rn 'shrmpl' src bin etc doc README.md CLAUDE.md` → only
  `infra/` (shrmpl-hook) and PS0004's migration doc.
- `grep -rn '/Users/' src bin etc` → nothing.
- Log lines carry the code: pick one and find it in `bin/215` output.
- pytest suite green (it exercises lockout and rate limiting).

## Deploy needs

Go binary. UAT env file loses `KV_BACKEND`, `LOG_RECEIVER_HOST_PORT`
and `VAULT_*` — update `etc/uat-api-srv.env.example`,
`infra/*/etc/*-api.env` redacted copies, and the live UAT env at deploy.

## Related

PS0002 (vault-configs out of git), PS0008 (package split after this),
PS0007 (checklist names what survives).
