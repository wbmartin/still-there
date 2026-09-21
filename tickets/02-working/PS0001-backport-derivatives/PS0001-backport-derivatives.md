---
id: PS0001
title: Backport infra, ticketing and tooling from derivative apps
status: ready
blocked_by:
waiting_on:
priority: high
domain: infrastructure
raised: 2026-09-21
raised_by: Brandon
asked:
claimed: 2026-09-21 by TKT-PS0001-backport-derivatives
merged:
deployed:
validated:
---
# PS0001 — Backport infra, ticketing and tooling from derivative apps

**Ticket:** PS0001  **Branch:** `TKT-PS0001-backport-derivatives` → `main`  **Type:** infrastructure / tooling
**Security review:** `PS0001-Security_Considerations.md` (beside this file)

## What

ProtoStack is the template that eastern-rebate, atticus-meeting and others
derive from. Those apps built infrastructure and process that belongs in the
template. This ticket brings it back, genericized, in eight commits.

## Requirements

- Schema changes reach a database only through a versioned migrator (atticus-meeting TKT-MA0005).
- A lightweight presales deploy: build locally, ship binaries + local DB to UAT, one sudo command (eastern-rebate).
- A ticket workflow with lint tooling (eastern-rebate).
- Generic API/web/test fixes found while running the derivatives in production (atticus-meeting).
- Keep Mantine; keep shrmpl kv/log/vault optional; keep pytest.

## What changed and why

| Commit | Area | Change |
|---|---|---|
| `ffd3115` | Ticketing | `tickets/{01-open,02-working,03-done,04-archive}`, `scripts/lint-tickets.py` (`PS####`, `PS1000+` = customer questions), skills `ticket-workflow` / `ticket-triage` / `compact-tickets` / `handoff`, `CLAUDE.md`. |
| `e6aeb0e` | Migrator | `sql/` → `src/api-srv/sql/` (go:embed). `010-initialcreate.sql` split: `roles.sql` (idempotent `db_reader`/`db_writer`/`proto_app_svc_account`), `migrations/0001_initial_schema.sql` (pure DDL; grants now run *after* the tables exist — in the old file they ran before and were a no-op), `fixtures/015-initialdata.sql` (idempotent). `ddl-role-host-setup.sql` documents the one-time `proto_ddl` (owner, runs migrations) / `proto_ci` (CREATEDB) host setup. `cmd/migrate` (`status|pending|verify|up|stamp`), `internal/migrate` (schema_migration table, SHA-256 drift check, per-file tx, advisory lock, refuses out-of-order), `app_lib.ReadEnvFile`. `bin/510-reset-test-db`, `bin/100` builds `proto-migrate`. |
| `5fac67d` | Presales deploy | `bin/600-bundle-uat` (static linux/amd64 api-srv + proto-migrate, vite bundle, pg_dump via `docker exec`, roles.sql, MANIFEST → scp to `$UAT_HOST:/tmp`), `bin/610-deploy-uat` (ships as `deploy.sh`; installs systemd unit on first run, stop → drop/restore DB → swap binary/web keeping `.prev` → start → `/health` poll; `--no-db` runs `proto-migrate verify`), `presales-deploy` skill, `etc/uat-api-srv.env.example` (`SERVERONLY` redaction), `doc/320-presales-uat-deploy.md`. |
| `8559400` | Web app | `utils/api.ts`: `ApiUnreachableError` vs `SessionExpiredError` + `setSessionExpiredListener`. `auth-store`: refresh signs out only on 401/403; network/5xx → `authState: "UNREACHABLE"` (Mantine alert + retry) — fixes users bounced to login during an nginx 502. `stores/create-entity-store.ts` factory replaces three hand-written stores (public API now `itemList`/`readList`/`createItem`/`updateItem`/`deleteItem`; 9 call sites adapted; `updateItem` now throws on failure instead of silently succeeding). `cache-utils` self-registering clearers. `ErrorBoundary`: try-again without reload, no unauthenticated `POST /error-report`, message only. `vitest.config.ts` + 28 tests. 10 pre-existing lint errors fixed. |
| `eb6452c` | API server | `SERVER_BIND_ADDRESS` (bind 127.0.0.1 behind nginx); `LOG_RECEIVER_HOST_PORT` optional → stdout logger; `KV.Del`; CSRF-mismatch log no longer prints the expected token; `http.MaxBytesReader` on bodies read before auth/rate-limit (login 4K, context switch 4K, MFA 4K, error-report 8K); unique violations (23505) → `ErrDuplicate` → 409 on user/group create+update; `repository/scoped_query.go` centralises the security-context WHERE clause (`example`, `app_user_group` lists; SQL unchanged). |
| `72c1db6` | Infra mirrors | `infra/uat/` and `infra/prd/` placeholders for the post-sales pipeline (atticus TKT-MA0005 model): host `bin/` scripts deploy verbatim, every per-host value in `infra/<host>/etc/*.env` (repo copies `SERVERONLY`-redacted), systemd unit + root-owned restart `.path` unit, nginx, shrmpl-hook env, `infra/README.md`. Presales `600/610` now read `etc/uat-deploy.env` (local, no secrets) and a bundled `deploy.env`; no inline defaults, missing keys die by name. |
| `ac9c8eb` | Test infra | `bin/200-run-tests [all|go|e2e|py]` — a skipped integration suite is a failure; `connectTestDB(t)` helper (fixes a pre-existing `go vet` failure in the group repo test); Playwright smoke on ports 8082/5182 (`playwright.config.ts`, `tests/e2e/`, `bin/230-run-web-app-e2e`, `bin/513`/`514` scratch resets, `etc/e2e-api-srv.env`, `etc/loc-test.env`); `bin/320` no longer relies on a stale venv `activate`; `bin/220` repo-relative; `.gitignore` generic entries; `doc/902` refreshed. |

Deliberately not ported: atticus's Tailwind swap, Node-cutover shims (`legacy_pw_hash`, bcrypt), realtime/websocket, team tenancy, webhook/oras registry pipeline, `azure-pipelines.yml`; eastern-rebate's named contacts and post-pytest SQL cleanup ritual.

## Files modified

`git diff --stat 0c2130a..HEAD` — 100+ files. Highlights: `CLAUDE.md`, `tickets/**`, `scripts/lint-tickets.py`, `.claude/skills/**`, `src/api-srv/{cmd/migrate,internal/migrate,sql/**}`, `src/api-srv/internal/{app_lib,handler,repository}/*.go`, `src/web-app/{stores,utils,components,pages,tests}/**`, `bin/{100,200,220,230,320,510,513,514,600,610}-*`, `etc/{loc-test,e2e-api-srv}.env`, `etc/uat-api-srv.env.example`, `doc/{050,320,900,902}-*.md`, `.gitignore`, `infra/**`, `etc/uat-deploy.env`.

## Endpoints

No new endpoints. Modified behaviour, all local environment:

| Endpoint | Auth | Change | Data |
|---|---|---|---|
| `POST /auth` (login) | anonymous by design | body capped 4 KiB | credentials |
| `POST /security-context` | JWT | body capped 4 KiB | none |
| `POST /mfa/setup`, `POST /mfa/verify` | JWT | body capped 4 KiB | TOTP codes |
| `POST /error-report` | anonymous | body capped 8 KiB; web app no longer calls it | client error text |
| `POST/PUT /user`, `POST/PUT /app-user-group` | JWT | unique violation → 409 `DUPLICATE_ERROR` (was 500) | none new |

## Deploy needs

- Rebuild both binaries (`bin/100-build-api-server` → `proto-go-app`, `proto-migrate`).
- Existing databases: run `roles.sql` if roles are missing, then `proto-migrate -config <env> stamp 1` once to adopt them; new databases: `bin/510-reset-test-db` or `migrate up`.
- Hosts that will use the migrator: one-time `ddl-role-host-setup.sql` (create `proto_ddl`, move ownership) — presales `610-deploy-uat` restores full dumps and does not need it.
- Local dev: `etc/loc-test.env` drives `bin/200-run-tests`; docker `loc-postgres` expected.

## Verification

`bin/200-run-tests all` on the integrated branch: **go=105 e2e=3 py=63** passed; `npx vitest run` 28/28; `tsc -b`, `npm run lint` (0 errors), `go vet ./...`, `gofmt -l` clean; `scripts/lint-tickets.py` clean; `bin/600-bundle-uat --no-scp --no-db` produced a 7.2 MB bundle with `api-srv`, `proto-migrate`, `web-pub.tar.gz`, `deploy.sh`, `MANIFEST`. `510-reset-test-db` + `migrate status|verify|pending` exercised against docker Postgres 17 via both the docker and native psql paths.
