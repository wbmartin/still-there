---
id: PS0003
title: Decide shrmpl's place in the template and quarantine or remove it
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
# PS0003 — shrmpl: optional since June, still ~30% of the tooling surface

**Source:** template review 2026-09-21. **Decision ticket** — Brandon
decides the scope; the agent then executes.

## Background

Commit `32c4490` (2026-06-15, "Add in-process KV backend (default),
replacing the external shrmpl-kv dependency") made `KV_BACKEND=memory` the
default in every env file (loc, e2e, uat example, infra uat/prd).
`LOG_RECEIVER_HOST_PORT` is optional (unset = stdout) and is commented out
in e2e/infra envs. The vault is contacted only when all five `VAULT_*` keys
are set, and only `etc/loc-api-srv.env` sets them. UAT/infra envs set
`DATABASE_URL` directly.

Yet the template still carries the full shrmpl local stack and three Go
clients. A derived app inherits all of it and has to learn what is live.

## What is shrmpl-only today

| Area | Files |
|---|---|
| bin | `010-download-shrmpl` (mac-only binaries from `raw.githubusercontent.com/wbmartin/shrmpl-srv-preview`, no checksum, list omits `shrmpl-vault-cli` which is on disk), `210-run-shrmpl-app-infra`, `211-stop-shrmpl-app-infra`, `410-regenerate-vault-mtls-certs` |
| etc | `loc-shrmpl-kv-srv.env`, `loc-shrmpl-log-srv.env`, `loc-shrmpl-vault-srv.env`, `vault-configs/*` (tracked secrets — removal from git is PS0002), 9 of the 11 `etc/crt/*.pem` (all `shrmpl_vault_*`), `bin/tools/*` (5 binaries, ~11 MB, gitignored) |
| Go | `internal/app_lib/kv_client.go` (409 lines, external TCP KV), `internal/app_lib/logging_client.go` (custom TCP wire format), `VaultClient` in `internal/app_lib/config.go:42-141` |
| docs | `doc/850-mtls-troubleshooting.md`, and `doc/050`, `doc/060`, `doc/105`, `doc/902`, `README.md` all still present 010/210/410 as the start-up path |
| infra | `infra/*/shrmpl-etc/` is the **shrmpl-hook** webhook receiver — a different shrmpl product, still load-bearing for the post-sales pipeline (PS0004). Not in scope here. |

## Defects inside the shrmpl code paths (fix or delete, not leave)

- `config.go:70` hardcodes
  `/Users/brandon/Code/ProtoStack-ReactGo/etc/crt/shrmpl_vault_server_fullchain.pem`
  in Go source. If the vault stays, this becomes a config key.
- `logging_client.go` `log(level, code, ...)` never uses `code`: sends the
  literal `"0000"` to shrmpl-log (`:131`) and omits it from the stderr line
  (`:148`). ~300 `app.Info("EX_CR_START", ...)` call sites carry dead
  codes. Either output the code or drop the code parameter.
- `logging_client.go:113-124` dials the receiver with a 5 s timeout
  **while holding `l.mu`** whenever the client is nil and a host is
  configured. Receiver down ⇒ every request serializes on a 5 s dial per
  log line.
- `bin/211` runs `pkill -f vite` and `pkill -f proto-go-app` — kills every
  Vite dev server on the machine.
- `etc/loc-*.env` files carry 10 absolute `/Users/brandon/...` paths (two
  spelled `/Users/Brandon`, working only because macOS is
  case-insensitive), `TLS_NAME=local-atticus-snipper.teamtaggle.com` and
  `VAULT_SERVER=loc-proto.cnfg.li:7474`.

## Options

**A. Remove.** Delete the shrmpl kv/log/vault clients, scripts, envs,
certs, docs. Keep the `KV` interface + memory impl (leave a slot for a
Redis impl). Replace the logger with `log/slog` to stdout (journald
captures it on the servers). Local dev secrets inline in
`etc/loc-api-srv.env` like `etc/e2e-api-srv.env` already does. Simplest
template; PS0001 said "keep shrmpl optional", so this reverses that.

**B. Quarantine.** Move the local shrmpl stack to `optional/shrmpl/`
(scripts, envs, certs, one README) and the Go clients behind a build tag
or into `internal/kv/shrmpl`, `internal/logx/shrmpl`. `loc-api-srv.env`
becomes vault-free by default. Docs demote 010/210/410 to an appendix.
Keeps the option without the cost on the happy path.

**C. Keep as-is** and only fix the defects above.

Recommendation from the review: **B for kv, A for log and vault.** Nothing
deployed uses vault or shrmpl-log; the KV interface is worth keeping.

## Acceptance (once decided)

- A fresh clone runs `bin/100`, `bin/215`, `bin/220`, `bin/200 all` with no
  shrmpl binary, no vault, no absolute path edits.
- No `/Users/` string anywhere under `src/`, `bin/`, `etc/`.
- `doc/050`, `doc/060`, `README.md` describe the vault-free path first.
- If A: `go.mod` loses nothing shrmpl-specific (there is no shrmpl module;
  the clients are hand-written), `internal/app_lib` shrinks by ~700 lines.

## Deploy needs

Go binary; env-file changes on UAT if keys are renamed. No migration.

## Related

PS0002 (vault-configs out of git), PS0007 (rename checklist depends on what
survives), PS0008 (`app_lib` split is easier after this).
