# Project Commands

One row per `bin/` script; a new script means a new row here. Numbers:
`0xx` setup, `1xx` build, `2xx` run, `3xx` tests, `4xx` certs, `5xx`
database, `6xx` deploy. Details follow in the sections below.

| Script | What it does |
|--------|--------------|
| `bin/000-rebrand <slug> "<Name>"` | Rename the app when deriving from this template (`DERIVING.md`) |
| `bin/100-build-api-server` | Build the API and migrator binaries into `dist/` |
| `bin/110-build-web-app` | `vite build` and a timestamped tar.gz in `dist/` |
| `bin/120-build-pi-agent` | Cross-compile the Pi agent into `dist/` |
| `bin/205-migrate-db [dbname]` | Create/migrate the developer database (`loc_stillthere_db`) |
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
| `bin/600-deploy-prd [--no-upload \| --check]` | Build a release, scp it to `prd-wbm`, print the staff's one sudo command; `--check` reads what is live |
| `bin/630-deploy-pi-agent <ssh-target> [agent.json]` | Build and install the Pi agent as a systemd service |

## Setup and Build

```
./bin/000-rebrand <slug> "<Display Name>" [--module <go module path>]
./bin/100-build-api-server
./bin/120-build-pi-agent                # PI_GOARCH=arm64 for 64-bit Pi OS
./bin/630-deploy-pi-agent pi@<pi-host> [agent.json]
./bin/400-make-local-dev-certs [host]   # default localhost; FORCE=1 overwrites
```
- `000`: the mechanical half of `DERIVING.md`. Reads the current identity
  from `etc/app.env`, rewrites every tracked text file and the infra file
  names that carry the slug, writes `etc/app.env`, prints the steps left to
  a human. Refuses a dirty tree.
- `100`: `dist/<slug>-api` and `dist/<slug>-migrate`. `bin/215` runs it
  first, so a separate build is only for checking it compiles.
- `120`: `dist/<slug>-agent`, static, `GOARCH=arm GOARM=7` by default
  (32-bit Raspberry Pi OS).
- `630`: builds with `120`, copies the binary, unit and
  `src/pi-agent/deploy/install.sh` to the Pi and runs the install with
  sudo: user `stillthere` (group `bluetooth`), `/usr/local/bin/stillthere-agent`,
  `/etc/stillthere/agent.json` (0600; replaced only when one is passed),
  service `stillthere-agent`. Logs: `journalctl -u stillthere-agent`.
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

The one command, on a laptop:
```
./bin/300-run-tests            # all: infra + docs + go + govulncheck + unit + e2e + py (load is not part of all)
./bin/300-run-tests infra      # infra/<host>/bin: bash -n, no defaults, required keys set; install/rollback on a fake root
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
  `bin/prd-stillthere-install` under `infra/prd-wbm/`) or by basename.
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
  Then rebuilds `loc_stillthere_test_db` via `bin/510` and runs
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
- `e2e`: rebuilds `loc_stillthere_e2e_db` via `bin/510`, installs Chromium if
  needed, runs `bin/230-run-web-app-e2e`.
- `py`: rebuilds `loc_stillthere_e2e_db`, builds the API and starts it on
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
- Rebuilds `loc_stillthere_e2e_db`, starts the Go API on `127.0.0.1:8082` from
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
./bin/205-migrate-db [dbname]      # default: the database etc/loc-api-srv.env names (loc_stillthere_db)
```
- Creates the database if missing, `sql/roles.sql`, `cmd/migrate up`,
  then `status`. A database it creates also gets
  `sql/fixtures/local-seed.sql` (tenant 1); an existing one gets
  migrations only. Connects as the postgres role in `etc/loc-test.env`
  (the app role is DML only). Refuses `_test_db` / `_e2e_db` names, and
  never writes `doc/003-schema-reference.sql`.

## Scratch Databases

```
./bin/510-reset-test-db [dbname]   # default loc_stillthere_test_db; DROP + rebuild
./bin/515-reseed-db [dbname]       # default loc_stillthere_e2e_db; TRUNCATE + re-seed without dropping
```
- `510`: DROP + CREATE, `sql/roles.sql`, schema through `cmd/migrate up`,
  then `sql/fixtures/test-seed.sql` (tenants 2 and 3) and
  `sql/fixtures/test-tenant-lock.sql` (`CHECK (security_context_id <> 1)`
  on every tenant table). Refuses any name not ending in
  `_test_db` or `_e2e_db`. Uses psql against whatever serves
  `PGHOST`:`PGPORT`, reached over TCP (`PSQL` overrides); the migrator
  connects with `TEST_PG_URL_BASE`. This also covers what used to be the
  e2e-only reset: `bin/510-reset-test-db loc_stillthere_e2e_db`.
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

## Deploy to prd

```
./bin/300-run-tests all                  # first, on the commit that ships
./bin/600-deploy-prd                     # build, bundle, scp to prd-wbm, print the sudo command
./bin/600-deploy-prd --no-upload         # build and bundle only: dist/<bundle>/
./bin/600-deploy-prd --check             # read-only: live revision, unit state, /api/health
```
- Settings come from `etc/prd-deploy.env` (`PRD_SSH_HOST`, `APP_ROOT`,
  `SITE_URL`), after `etc/app.env`. No inline defaults: a missing key
  stops the run.
- Refuses a dirty tree (`git status --porcelain`, untracked files
  included), and checks again after the build: the bundle's
  `stillthere-api.revision` names the commit it was built from.
- Builds `api-srv` and `stillthere-migrate` (linux/amd64, `CGO_ENABLED=0`,
  `-trimpath`), the vite bundle as `web-pub.tar.gz`, the revision file
  and `SHA256SUMS`, in `dist/<YYYYmmddTHHMMSSZ>-<commit>/`; uploads it to
  `<APP_ROOT>/incoming/<bundle>/` as the staff login (the `stillthere-deploy`
  group, no sudo), under a `.partial` name renamed when whole; deletes
  the local copy once uploaded (kept on failure and with `--no-upload`).
- Prints `ssh -t prd-wbm sudo /srv/prd-stillthere/bin/prd-stillthere-install <bundle>`
  for the staff member: the host half (`infra/prd-wbm/bin/`, see
  `infra/README.md`) checks the bundle, migrates, swaps with `.prev`
  kept, restarts and checks health, rolling itself back when the new
  release is not healthy. `prd-stillthere-rollback` swaps back by hand.
