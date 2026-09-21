---
id: PS0007
title: Single rename point and a "deriving a new app" checklist
status: in-review
blocked_by:
waiting_on:
priority: high
domain: docs
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0007-derive-new-app-checklist
merged: 9d74ba9
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

## What changed and why

Branch `TKT-PS0007-derive-new-app-checklist`, 5 commits. All four items
in "What to do", with one practical reading (below).

| Item | Done |
|---|---|
| 1. Single rename point | **`etc/app.env`** (`APP_SLUG`, `APP_DISPLAY_NAME`). Sourced by `bin/100`, `215`, `216`, `200`, `510`, `513`, `514`, `600`; `etc/loc-test.env` and `etc/uat-deploy.env` are written in terms of `$APP_SLUG`. Derived names: `dist/<slug>-api`, `<slug>-migrate` (local *and* the presales bundle — `610` installs `$APP_NAME-migrate`), `loc_<slug>_{test,e2e}_db`, `uat-<slug>-api`, `/srv/uat-<slug>`, `<slug>_app_svc_account`. JWT issuer is now `Config.AppName` (`GenerateJWT` takes it). `scripts/lint-tickets.py` has one `ID_PREFIX`. |
| — `PROTO_*` env prefix | The ticket asked for `${APP_SLUG^^}_*`; Go tests read these names literally, so a slug-derived prefix is not practical. Renamed once to neutral harness names instead — `TEST_ENV`, `TEST_DB`, `E2E_DB`, `TEST_PG_URL_BASE`, `TEST_DATABASE_URL`, `E2E_API_ENV`, `TEST_LOG_DIR`, `TEST_API_ENV` — across bin, etc, Go tests, playwright config, infra, docs. A derived app never touches them. |
| 2. Checklist | **`DERIVING.md`** at the root: clone + rebrand, secrets/passwords table (every file each one lives in), hosts (both deploy paths), the example entity, tickets/CLAUDE.md/README reset, local DB + first run, the final grep. |
| 3. `bin/000-rebrand` | Built — there were ~20 sed-able tokens, well over the "~5" threshold. `bin/000-rebrand <slug> "<Name>" [--module] [--ticket-prefix]`: refuses on a dirty tree, reads the current identity from `app.env`, ordered perl substitutions over every tracked text file (specific tokens first, bare `\bproto\b` last; `ProtoStack`, `protocol`, `X-Forwarded-Proto` untouched), `go mod edit` + import rewrite, `git mv` of every `infra/` file carrying the slug (incl. nginx sites), rewrites `app.env`, then `go build` + `tsc` + lint, and prints the human steps. Idempotent. |
| 4. Web | `constants/app.ts` (`APP_NAME`, `APP_TITLE`); `login.tsx` uses it; `index.html` title; `package.json` name `proto-web-app`; `App.css` and `public/vite.svg` deleted. |
| Kept on purpose | Migrator advisory-lock key `"PROTO"` — one migrator per cluster is the safe default; documented in `migrate.go` and `DERIVING.md`. |

**Proven** in a throwaway worktree: `bin/000-rebrand acme "Acme Portal"
--ticket-prefix AC` → 85 files changed, 30 renamed, `go build` + `tsc`
clean, module `acme-api`, prefix `AC` in skills/CLAUDE.md/linter, and the
acceptance grep over `bin etc src infra scripts .claude doc README
CLAUDE.md` finds only the lock key. Second run: "nothing to do".

**Side effects on this laptop:** local databases renamed to the slug
(`loc_starter_db` → `loc_proto_db` via `ALTER DATABASE`; the old scratch
`_test_db`/`_e2e_db` dropped, `bin/200` rebuilds them). `SERVER_NAME` in
the local/e2e env files is `proto-api`/`proto-api-e2e`.

## Files modified

- new: `etc/app.env`, `bin/000-rebrand`, `DERIVING.md`, `src/web-app/constants/app.ts`
- `bin/100`, `200`, `215`, `216`, `230`, `310`, `320`, `510`, `513`, `514`, `600`, `610`
- `etc/loc-test.env`, `loc-api-srv.env`, `e2e-api-srv.env`, `uat-api-srv.env.example`, `uat-deploy.env`
- `src/api-srv/internal/app_lib/auth.go`, `internal/handler/auth_handler.go`, `internal/migrate/migrate.go`, `migrate_test.go`, `internal/repository/testdb_test.go`
- `src/web-app/index.html`, `package.json`, `package-lock.json`, `pages/login.tsx`, `playwright.config.ts`; deleted `App.css`, `public/vite.svg`
- `infra/README.md`, `infra/uat/etc/uat-proto-test.env`, `infra/uat/bin/uat-proto-run-tests`
- `scripts/lint-tickets.py`, `CLAUDE.md`, `README.md`, `.claude/skills/presales-deploy/SKILL.md`, `doc/050`, `060`, `105`, `320`, `330`, `902`, `tests/README.md`

## Endpoints introduced or modified

None. JWT `iss` claim is now `APP_NAME` (`"Proto App"`) instead of the
literal `proto-go-app`; nothing validates the issuer, so existing sessions
keep working.

## Acceptance (met 2026-09-21)

- [x] Following `DERIVING.md` on a fresh clone yields no `proto`, `starter`,
  `wbmartin`, `Proto App` outside `tickets/` — proven with `acme` in a
  worktree; only the deliberate lock key remains.
- [x] `bin/200-run-tests all` on the template after the renames: go=105
  e2e=3 py=63. `bin/215` builds and runs `dist/proto-api`; `bin/216` stops it.

## Deploy needs

Go binary (JWT issuer). The presales bundle now ships the migrator as
`proto-migrate` still (slug is `proto`), so UAT is unchanged; `deploy.env`
carries the same keys. Next UAT deploy with PS0002/03/04.

## Waiting for

The next UAT deploy.

## Direction (Brandon, 2026-09-21)

Approved as written. Still blocked on PS0003 (now decided: shrmpl kv/log
sunset — vault pending) and PS0004 (now a documentation ticket, so the
binary/role names stay as they are today; the checklist documents both
deploy paths' names until they converge).
