# Project Commands

One row per `bin/` script; a new script means a new row here. Numbers:
`0xx` setup, `1xx` build, `2xx` run, `3xx` tests, `4xx` certs, `5xx`
database, `6xx` deploy. Details follow in the sections below.

| Script | What it does |
|--------|--------------|
| `bin/000-rebrand <slug> "<Name>"` | Rename the app when deriving from this template (`DERIVING.md`) |
| `bin/100-build-api-server` | Build the API and migrator binaries into `dist/` |
| `bin/110-build-web-app` | `vite build` and a timestamped tar.gz in `dist/` |
| `bin/205-migrate-db [dbname]` | Create/migrate the developer database (`loc_proto_db`) |
| `bin/215-run-api-srv` | Build and run the API on :8080 (foreground or background) |
| `bin/216-stop-local-stack` | Stop this checkout's API and Vite |
| `bin/220-run-web-app` | Vite dev server on :5173, proxying `/api` to :8080 |
| `bin/230-run-web-app-e2e` | Playwright suite on its own API (:8082) and Vite (:5182) |
| `bin/300-run-tests [mode]` | The test gate: infra, docs, go, govulncheck, unit, e2e, py |
| `bin/305-update-grants-fixture` | Rewrite the Go/web grants mirror fixture |
| `bin/330-test-load` | Load test a private API on the e2e database |
| `bin/400-make-local-dev-certs [host]` | Self-signed cert in `etc/crt/` for testing `ENABLE_TLS=true` |
| `bin/510-reset-test-db [dbname]` | DROP + rebuild a scratch `_test_db` / `_e2e_db` |
| `bin/515-reseed-db [dbname]` | TRUNCATE + re-seed a scratch database |
| `bin/600-bundle-uat` | Presales: build, dump, bundle, scp to UAT, delete the local bundle |
| `bin/605-survey-host <ssh-host>` | Read-only post-sales host survey |
| `bin/610-deploy-uat` | Presales: shipped in the bundle, run on the UAT host with sudo |
| `bin/620-export-tenant-data <out-dir>` | Scrubbed dump of the local database for a host |

## Setup and Build

```
./bin/000-rebrand <slug> "<Display Name>" [--module <go module path>]
./bin/100-build-api-server
./bin/400-make-local-dev-certs [host]   # default localhost; FORCE=1 overwrites
```
- `000`: the mechanical half of `DERIVING.md`. Reads the current identity
  from `etc/app.env`, rewrites every tracked text file and the infra file
  names that carry the slug, writes `etc/app.env`, prints the steps left to
  a human. Refuses a dirty tree.
- `100`: `dist/<slug>-api` and `dist/<slug>-migrate`. `bin/215` runs it
  first, so a separate build is only for checking it compiles.
- `400`: writes `etc/crt/api-srv-fullchain.pem` and
  `etc/crt/api-srv-privkey.pem` (gitignored, 365 days); set
  `ENABLE_TLS=true` in `etc/loc-api-srv.env` to use them. Dev is plain
  HTTP otherwise; real hosts use certbot (doc/310-api-srv-cert-renewal.md).

## API Server Management

To run the api server:
```
./bin/215-run-api-srv
```
- **Interactive mode** (default): Shows live logs, blocks until server exits
- **Daemon mode** (background): Runs in background, exits after startup message
- **Detection**: Automatically detects based on terminal output redirection
- **Force interactive**: `./bin/215-run-api-srv --interactive` or `./bin/215-run-api-srv -i`
- **Force daemon**: `./bin/215-run-api-srv --daemon` or `./bin/215-run-api-srv -d`

To check if api server is running (the port is `SERVER_PORT` in
`etc/loc-api-srv.env`):
```
curl -s http://localhost:8080/health
```

To view server logs (when running in background):
```
tail -f tmp/api-srv/server.log
```

To stop the api server, `./bin/216-stop-local-stack` (below), or
`kill $(cat tmp/api-srv/server.pid)` for a background run. `bin/215` (on
restart) and `bin/216` match this checkout's binary plus config path, so
another checkout's API and `bin/300`'s private instance on 8082 keep
running. `bin/216` stops only this checkout's vite listening on :5173, so
Playwright's vite on :5182 (`bin/230`) survives.

## Local Stack

To stop the api server and the web app dev server:
```
./bin/216-stop-local-stack
```

To run the web app (Vite on :5173, proxying /api to :8080):
```
./bin/220-run-web-app
```

To build and package the web app for deployment:
```
./bin/110-build-web-app
```
- Builds the web app using `vite build`
- Creates a timestamped tar.gz archive in `dist/` (e.g., `<slug>-web_20251221_074106.tar.gz`)
- Archive contains all built files ready for deployment
- Copy the archive to your web server and extract with `tar -xzf filename.tar.gz`

## Testing Commands

The one command, on a laptop or a build host:
```
./bin/300-run-tests            # all: infra + docs + go + govulncheck + unit + e2e + py (load is not part of all)
./bin/300-run-tests infra      # infra/<host>/bin: bash -n, shared copies identical across hosts
./bin/300-run-tests docs       # scripts/check-doc-paths.sh: backticked repo paths in docs exist
./bin/300-run-tests go         # gofmt, go vet, staticcheck, doc/003 current, go test (run before every commit)
./bin/300-run-tests unit       # eslint, vitest (src/web-app)
./bin/300-run-tests e2e        # Playwright only
./bin/300-run-tests py         # pytest HTTP suite against a private API instance
./bin/300-run-tests load       # load test on its own private API (see below); not part of `all`
```
- Sources `etc/loc-test.env` (override the file with `TEST_ENV=...`):
  scratch database names, `TEST_PG_URL_BASE` for the migrator,
  `TEST_DATABASE_URL` for the Go repository suite, `E2E_API_ENV`
  for the API instance the e2e/py suites start, `TEST_LOG_DIR`.
- `docs`: `scripts/check-doc-paths.sh` reads every backticked token in
  `CLAUDE.md`, `DERIVING.md`, the READMEs and `doc/*.md`, keeps the
  ones that look like repo paths, and resolves each by prefix
  (`bin/215`), against a package root (`internal/...`,
  `bin/uat-proto-build` under `infra/uat/`) or by basename.
  A miss prints `MISS <doc>:<line> <token>` and fails the run;
  so does an entry in `scripts/doc-paths-allow.txt` its document no longer
  names. Skips URLs, absolute paths, `<placeholder>`s and gitignored
  runtime dirs. The `docs=` count is path tokens checked.
- `go`: first, in each Go module (`src/api-srv`, `src/load-test`):
  `gofmt -l .` (any file listed fails; fix with `gofmt -w`), `go vet ./...`
  and `staticcheck ./...`. Checks only, nothing is rewritten; zero
  findings is the gate. A staticcheck finding that is wrong for a line
  gets `//lint:ignore <check> <reason>` above it, never a blanket config.
  `U1000` (unused) skips exported names. staticcheck is pinned
  (`STATICCHECK_VERSION` in the script) and installed once into
  `tmp/tools/bin/` (needs proxy.golang.org the first time; v0.8.1 builds
  with Go 1.26, which `go install` fetches unless `GOTOOLCHAIN=local`).
  Then rebuilds `loc_proto_test_db` via `bin/510` and runs
  `go test -v ./...` in each module with `TEST_DATABASE_URL` exported
  (one `go-test.log`; the `go=` count covers both), with
  `-coverprofile`: per module `tmp/test-reports/go-coverage-<module>.out`
  and an HTML page, total printed; a report, no threshold.
  A repository suite that cannot reach the database SKIPS (see
  `internal/repository/testdb_test.go`); the runner greps for that skip and
  fails the run, so a green run always includes the integration suites.
  Any other `--- SKIP` fails it too, except a test named `TestLive...`
  (pytest: `SKIPPED` except `test_live_...`; vitest and Playwright: any
  skipped test). Rules: `tests/README.md`, "Writing tests".
- `all` only: after the Go suites, `govulncheck ./...` in `src/api-srv`
  and `src/load-test`. The scanner is pinned (`GOVULNCHECK_VERSION` in
  the script) and installed once into `tmp/tools/bin/`. Any vulnerability
  the code calls, in a module or in the standard library of the toolchain
  `go.mod` selects (its `toolchain` line), fails the run; so does being
  unable to install it or reach vuln.go.dev. Not in `go` mode, which
  stays offline. Logs: `govulncheck-<module>.log`.
- `unit`: `npx eslint . --max-warnings 0` in `src/web-app` (same as
  `npm run lint`; check only, any error or warning fails; log
  `eslint.log`), then `vitest run` against `src/web-app/tests`.
- `e2e`: rebuilds `loc_proto_e2e_db` via `bin/510`, installs Chromium if
  needed, runs `bin/230-run-web-app-e2e`.
- `py`: rebuilds `loc_proto_e2e_db`, builds the API and starts it on
  `127.0.0.1:8082` from a copy of `etc/e2e-api-srv.env` with the developer
  rate limits (the pytest suite asserts a 429), bootstraps the pytest venv
  at `tests/.venv` itself, runs the suite, stops the API.
- `load`: delegates to `bin/330-test-load` (see below).
- No test script writes to the source tree: pytest's HTML report goes to
  `tmp/test-reports/`; logs land in `tmp/test-logs/` (`go-test.log`,
  `playwright.log`, `pytest.log`, `py-api.log`); pinned tools in
  `tmp/tools/bin/`. Every mode ends by comparing
  `git status --porcelain --untracked-files=all` with its value at the
  start and fails on any difference, so a dirty tree may still run the
  gate but a run may not dirty it. `npm run test:coverage` writes
  `src/web-app/coverage/`, which is gitignored. Ends with a summary line,
  e.g. `[INFO] tests passed (all): go=105 unit=42 e2e=3 py=63`.
- Requires a Postgres at `PGHOST`:`PGPORT` (doc/060-dev-env-start.md);
  nothing else -- the e2e API config has `DATABASE_URL` inline.

To run the Playwright end-to-end suite directly:
```
./bin/230-run-web-app-e2e            # or: cd src/web-app && npm run test:e2e
./bin/230-run-web-app-e2e --headed   # any playwright args pass through
```
- `src/web-app/playwright.config.ts` starts its own stack: the Go API on
  :8082 from `etc/e2e-api-srv.env` and Vite on :5182 proxying to it, so a
  developer's 8080/5173 keep running. Global setup runs `bin/515`, signs in
  once as the fixture user (`admin` / `admin123!`) and saves the cookie to
  `tests/e2e/.auth/` (gitignored); specs start already signed in. The
  reseed is skipped when `bin/300` already rebuilt the database
  (`E2E_DB_SEEDED=1`), so a `bin/300-run-tests` run reseeds only once.
- Specs are `src/web-app/tests/e2e/*.spec.ts`; vitest owns `*.test.ts`.
- `route-sweep.spec.ts` opens every route as `admin`, `basic` and
  signed out, and fails on an error heading, a near-blank page, a
  console error or an unexpected 4xx/5xx.

To load test the API (starts and stops its own private instance, like
`py` mode; never touches the developer's 8080 or its database):
```
./bin/330-test-load                                   # 60s, 100 workers
./bin/330-test-load --iterations 1 --concurrency 1    # smoke run
```
- Rebuilds `loc_proto_e2e_db`, starts the Go API on `127.0.0.1:8082` from
  `etc/e2e-api-srv.env`, and refuses to start if something already
  answers there. Builds `src/load-test` (its own Go module) and the API
  into `tmp/test-logs/`; the report goes to `tmp/test-reports/`.
- Each worker loops login, create, update, get, delete, logout on an
  example row; every row it creates it deletes. The report holds only
  what the client measured (no server CPU/memory).
- Logs in as `TEST_PERSONA_USER` from that file (`basic`: tenant 2,
  MFA-exempt, not `admin`, so the per-user rate limiter runs on every
  request). The binary takes `--user` and `--env <file>`, reads
  `TEST_PERSONA_PASSWORD` from the file (there is no `--password`; argv
  shows in `ps` and shell history), and refuses a login outside tenant 2.
- The URL and the `--env` file are the script's own, from the test env
  file: `bin/330` refuses `--base-url` and `--env` (any spelling) on its
  command line, so the load cannot be re-aimed at the developer's 8080.
  Extra flags pass through (`--duration`, `--concurrency`, `--iterations`,
  `--output`).

## Go/Web Mirror Fixture

```
./bin/305-update-grants-fixture    # rewrite model/testdata/grants.json
```

- The web app mirrors Go by hand (`src/web-app/constants/grants.ts`,
  `src/web-app/constants/api.ts`); `src/web-app/tests/contract.test.ts`
  fails on drift. Grants go through a fixture:
  `TestGrantsFixtureIsCurrent` fails when
  `src/api-srv/internal/model/testdata/grants.json` differs from
  `AllKnownGrants`, and vitest replays the same file against
  `grants.ts` both ways. Routes: every `API_ENDPOINTS` path must be
  registered in `src/api-srv/cmd/server/main.go`.
- After a grant change: run `bin/305`, mirror the change in `grants.ts`,
  commit both. `bin/305` is the only writer; the gate never rewrites
  the fixture.

## Developer Database

```
./bin/205-migrate-db [dbname]      # default: the database etc/loc-api-srv.env names (loc_proto_db)
```
- Creates the database if missing, `sql/roles.sql`, `cmd/migrate up`,
  then `status`. A database it creates also gets
  `sql/fixtures/local-seed.sql` (tenant 1); an existing one gets
  migrations only. Connects as the postgres role in `etc/loc-test.env`
  (the app role is DML only). Refuses `_test_db` / `_e2e_db` names, and
  never writes `doc/003-schema-reference.sql`.

## Scratch Databases

```
./bin/510-reset-test-db [dbname]   # default loc_proto_test_db; DROP + rebuild
./bin/515-reseed-db [dbname]       # default loc_proto_e2e_db; TRUNCATE + re-seed without dropping
```
- `510`: DROP + CREATE, `sql/roles.sql`, schema through `cmd/migrate up`,
  then `sql/fixtures/test-seed.sql` (tenants 2 and 3) and
  `sql/fixtures/test-tenant-lock.sql` (`CHECK (security_context_id <> 1)`
  on every tenant table). Refuses any name not ending in
  `_test_db` or `_e2e_db`. Uses psql against whatever serves
  `PGHOST`:`PGPORT`, reached over TCP (`PSQL` overrides); the migrator
  connects with `TEST_PG_URL_BASE`. This also covers what used to be the
  e2e-only reset: `bin/510-reset-test-db loc_proto_e2e_db`.
- `510` also rewrites `doc/003-schema-reference.sql` right after `up`,
  before any fixture (`SCHEMA_REFERENCE_FILE` moves it; `bin/300` points
  it at `tmp/test-logs/` and fails the `go` step when the two differ).
  After adding a migration: run `510`, commit the reference with it.
- `510` sources `etc/loc-test.env` when `TEST_PG_URL_BASE` is unset.
- `515` sources `etc/loc-test.env` when `TEST_PG_URL_BASE` is unset, and
  falls back to `510` if the database doesn't exist yet. It re-runs
  `test-seed.sql` and the lock after the TRUNCATE (which keeps the lock).
- `515` is what Playwright's global setup calls: dropping the database
  under a running API leaves its pool pointing at nothing.
- Users after a reset: `admin` / `admin123!`, `basic` / `basic123!`, `mfa` / `basic123!` (the only one that is not MFA-exempt; it enrols a TOTP on first sign-in), and `switcher` / `basic123!` (homed in tenant 2 with a group in 3: the login that tests `/changeSecurityContext`).

## Presales UAT Deploy

```
./bin/600-bundle-uat [--no-scp] [--no-db]   # build api + migrator + web, dump local db, scp bundle
sudo ./deploy.sh [--no-db] [--no-restart]   # bin/610-deploy-uat, shipped inside the bundle
```
- Settings come from `etc/uat-deploy.env` (`APP_NAME`, `UAT_HOST`,
  `LOCAL_ENV_FILE`, `APP_ROOT`, `SERVICE`, `SVC_USER`, `DB_APP_ROLE`, `DB_NAME`,
  `UAT_DOMAIN`); `PRESALES_ENV=<file>` swaps the file. No inline defaults -- a
  missing key stops the run. The database address comes from
  `etc/loc-test.env`, as everywhere else: `600` and `620` run `psql`,
  `pg_dump` and `pg_restore` against `PGHOST`:`PGPORT` over TCP.
- The first deploy installs the nginx site from `etc/uat-nginx-site.conf.template`
  for `UAT_DOMAIN` and prints the `certbot --nginx` line (doc/320).
- The bundle holds a full DB dump: a successful scp deletes the local copy;
  a failed scp or `--no-scp` keeps it. Earlier bundles left in `dist/` are
  listed with the `rm` to run, never deleted automatically (doc/320).
- See doc/320-presales-uat-deploy.md. Run `./bin/300-run-tests` first.
- Post-sales pipeline placeholders: `infra/{uat,prd}/` (see `infra/README.md`).

## Post-sales Host Survey and Data Export

```
./bin/605-survey-host <ssh-host>                    # read-only host record -> tmp/survey/
./bin/620-export-tenant-data <out-dir> [api-env]    # scrubbed local db dump, outside the repo
```
- `605` runs one probe on the host under sudo (the password once):
  users, tools, `/srv/{uat,prd}-proto`, units, Postgres roles, databases
  and owners, nginx, fail2ban, cron, ports. Env files as key names only;
  units, nginx sites and crontab pass a best-effort secret mask. Changes
  nothing. Names from `etc/app.env` and `etc/uat-deploy.env`. Run before
  the first post-sales step on a host and keep the file (doc/330 0.1,
  doc/331 0.1).
- `620` reads the database `LOCAL_ENV_FILE` (`etc/uat-deploy.env`) names,
  or the one in `[api-env]`; refuses it unless its `schema_migration`
  matches the checkout's migrations exactly; copies it to a throwaway
  `<db>_export`; runs `src/api-srv/sql/export-scrub.sql` there (refuses
  tenant-2 rows and the fixture logins; every password, TOTP seed and MFA
  exemption cleared, live logins `invited`, open invite links revoked,
  issue reports removed);
  writes `<slug>-data-<stamp>.dump` with `.sha256` and `.manifest` to
  `<out-dir>`, which must be outside the checkout. The export is customer
  data. The first sign-in on the host is an invite link issued there
  by `proto-migrate invite <userName>` (doc/330 A9, doc/900).
