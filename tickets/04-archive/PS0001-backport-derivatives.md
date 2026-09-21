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
merged: 2026-09-21 c73e026 (PR #3)
deployed:
validated: 2026-09-21
---
# PS0001 — Backport infra, ticketing and tooling from derivative apps

ProtoStack is the template eastern-rebate, atticus-meeting and others
derive from; those apps had built infrastructure and process that
belonged back in the template. Eight commits brought it back genericized:
a versioned SQL migrator, a presales UAT deploy path, the ticket workflow
and lint tooling, `infra/<host>` mirrors for the post-sales pipeline, and
a run of generic API/web/test fixes found while running the derivatives
in production.

## What changed
- `tickets/{01-open,02-working,03-done,04-archive}`, `scripts/lint-tickets.py`, skills `ticket-workflow`/`ticket-triage`/`compact-tickets`/`handoff`, `CLAUDE.md`.
- `sql/` → `src/api-srv/sql/` (go:embed): `roles.sql`, `migrations/0001_initial_schema.sql`, `fixtures/015-initialdata.sql`, `ddl-role-host-setup.sql`; `cmd/migrate` (status/pending/verify/up/stamp), `internal/migrate` (checksum drift check, advisory lock, refuses out-of-order).
- `bin/600-bundle-uat` + `bin/610-deploy-uat` (presales quick-deploy: local build + pg_dump → scp → systemd swap → `/health` poll), `presales-deploy` skill, `etc/uat-api-srv.env.example`, `doc/320`.
- Web app: `ApiUnreachableError`/`SessionExpiredError` split in `utils/api.ts`; auth-store signs out only on 401/403, not on network/5xx; `stores/create-entity-store.ts` factory replaces three hand-written stores; `ErrorBoundary` no longer posts to an unauthenticated endpoint; `vitest.config.ts` + 28 tests.
- API server: `SERVER_BIND_ADDRESS`, optional `LOG_RECEIVER_HOST_PORT`, `KV.Del`, CSRF-mismatch log no longer prints the expected token, `http.MaxBytesReader` on pre-auth bodies, unique-violation (23505) → 409, `repository/scoped_query.go` centralises the security-context WHERE clause.
- `infra/uat/`, `infra/prd/` placeholders (35 files, `SERVERONLY`/`example.com` throughout) for the post-sales pipeline.
- `bin/200-run-tests [all|go|e2e|py]`, Playwright smoke on 8082/5182, `connectTestDB(t)` helper.

## Rules that survived
- Schema changes reach a database only through the versioned migrator (`cmd/migrate`), never hand-run DDL.
- `proto_ddl` owns the schema and runs migrations; `proto_app_svc_account` keeps DML-only grants (`db_reader`/`db_writer`).
- Every `infra/<host>` script derives its values from `etc/*.env` with no inline defaults — a missing key dies by name rather than silently deploying with a guess.
- Pre-auth request bodies are capped (login/context-switch/MFA 4 KiB, error-report 8 KiB) before lockout/rate-limit checks run.
- A skipped integration suite counts as a test failure, not a pass.

## Figures at close
| | date |
|---|---:|
| `bin/200-run-tests all` | go=105 e2e=3 py=63 passed |
| `npx vitest run` | 28/28 |
| lint/typecheck/vet | 0 errors, clean |
| `bin/600-bundle-uat --no-scp --no-db` bundle size | 7.2 MB |

## Endpoints
- `POST /auth` — anonymous, body capped 4 KiB.
- `POST /security-context` — JWT, body capped 4 KiB.
- `POST /mfa/setup`, `POST /mfa/verify` — JWT, body capped 4 KiB.
- `POST /error-report` — anonymous, body capped 8 KiB; web app no longer calls it.
- `POST/PUT /user`, `POST/PUT /app-user-group` — JWT, unique violation now 409 `DUPLICATE_ERROR` (was 500).

## Security
Net security improvement: the migrator separates DDL ownership from the runtime role, pre-auth bodies are now bounded, a CSRF-token log leak is closed, and the web app no longer spams an anonymous error-report endpoint. Residuals are confined to local-dev/presales convenience — well-known local-only Postgres passwords remain in `etc/loc-test.env`/`etc/e2e-api-srv.env`/`roles.sql` (pre-existing pattern, must be rotated on real hosts), `bin/610-deploy-uat` is a root-run DB-replacing script acceptable only for presales UAT with no customer data, and `proto_ddl`/`proto_ci` host setup is a manual runbook step by design. `.gitignore` now excludes Playwright's saved auth cookie from ever being committed.

## Left open
- `infra/uat`/`infra/prd` are unexercised placeholders; review again when a derived app fills them in.
- Graduation from the presales full-DB-dump deploy to the post-sales migration pipeline is its own ticket (PS0004).

## Sources
- git: `c73e026` (PR #3)
