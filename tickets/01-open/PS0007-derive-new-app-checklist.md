---
id: PS0007
title: Single rename point and a "deriving a new app" checklist
status: blocked
blocked_by: [PS0003, PS0004]
waiting_on:
priority: high
domain: docs
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
merged:
deployed:
validated:
---
# PS0007 — There is no "how to derive a new app from this" path

**Source:** template review 2026-09-21. Blocked on PS0003 and PS0004
because what survives (vault? which binary name? which role model?)
decides what the checklist names.

## Problem

The single biggest extensibility gap. `infra/README.md` has a good
"Deriving an app from this" section (rename `infra/uat/` → `infra/<host>/`,
sed `proto` → `<app>`) but it covers the infra mirror only. Nothing covers
the app itself. The identity "proto" is spread across ~54 files with no
single place to change it.

### Where the identity lives today (grep counts, 2026-09-21)

| String | Lines / files | Notes |
|---|---|---|
| `proto` (ci) | 310 / 54 | includes env prefix `PROTO_*` (7 distinct vars: `PROTO_API_ENV`, `PROTO_E2E_*`, `PROTO_PG_URL_BASE`, `PROTO_TEST_DATABASE_URL`, `PROTO_TEST_DB`, `PROTO_TEST_ENV`, `PROTO_TEST_LOG_DIR`); binaries `proto-api` 69, `proto-migrate` 43, `proto-go-app` 13; roles `proto_ddl` 8, `proto_app_svc_account` 11 |
| `uat-proto\|prd-proto` | 100 / 34 | infra/ |
| `loc_starter` | 18 / 8 | db names in `bin/510/513/514`, `etc/*.env`, vault-configs |
| `Proto App` | 6 / 6 | `APP_NAME` in every api env |
| `loc-postgres` | 7 / 5 | container name |
| `wbmartin` | 1 | `bin/010` GitHub URL |

Plus, in code and config that a sed will not reach cleanly:

- Go module path `github.com/wbmartin/starter-api` (`src/api-srv/go.mod`).
- JWT `Issuer: "proto-go-app"` hardcoded at `internal/app_lib/auth.go:129`
  instead of `Config.AppName`.
- Migrator advisory-lock key `0x50524f544f` ("PROTO") at
  `internal/migrate/migrate.go:35` — derived apps on a shared cluster
  would contend on the same lock (arguably desirable; decide).
- KV mode literal `"shrmpl"` (`config.go:326`, `kv_client.go:49`).
- `repository/testdb_test.go:33` reads `PROTO_TEST_DATABASE_URL`.
- Web: `index.html:5,7` (`/vite.svg`, title `web-app`), `pages/login.tsx:70`
  ("Login to Proto"), `package.json` name `web-app`, `public/vite.svg`,
  `App.css` (untouched Vite scaffold, not imported anywhere — delete),
  `.env.production` entirely commented out, no `base` in `vite.config.ts`.
- Passwords a derived app must change and nothing forces it:
  `sql/fixtures/015-initialdata.sql` admin `admin123!`,
  `sql/roles.sql:36` `reallygoodPSQLpassword` (also in `etc/e2e-api-srv.env`,
  `etc/loc-test.env` ×3, `infra/uat/etc/uat-proto-test.env` ×3).
- `scripts/lint-tickets.py` `PS` id prefix regex — one constant would do.

## What to do

1. **One `APP_SLUG`** (and `APP_NAME`) in a single sourced file —
   `etc/app.env` or similar — read by every `bin/` script for binary
   names, service names, db names, container name, env-var prefix
   (`${APP_SLUG^^}_*`). The Go side reads `APP_NAME` from config already;
   make the JWT issuer use it.
2. **`doc/000-deriving-a-new-app.md`** (or `DERIVING.md` at root): the
   ordered checklist — clone, set `APP_SLUG`, `go mod edit -module`,
   rename `infra/<host>/`, web title/logo/`constants/app.ts`, rotate the
   three passwords + `JWT_SECRET` + `TOTP_ENCRYPTION_KEY`, delete
   `features/example` (after PS0012) or the `example_*` files, delete
   `doc/history/` (PS0006), reset `tickets/` to empty + `STATUS.md`,
   update `CLAUDE.md`'s Layout paragraph. Generalise what
   `infra/README.md` already does.
3. Optional: `bin/000-rebrand <slug>` that applies the sed and prints the
   manual steps. Only worth it if the checklist has more than ~5 sed-able
   lines after step 1.
4. `web-app/constants/app.ts` with `APP_NAME`, `APP_TITLE`; delete
   `App.css`, `public/vite.svg`.

## Acceptance

- Following the doc on a fresh clone yields an app with no `proto`,
  `starter`, `wbmartin`, `Proto App` string outside `doc/history/`.
- `grep -rn 'proto' bin etc src --include=* -il | wc -l` after the
  rebrand is 0 (or only the migrate lock key, if kept on purpose).

## Deploy needs

None for the doc. Renames touch UAT if the binary/env names change (PS0004).
