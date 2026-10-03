# Project Commands

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

To check if api server is running:
```
curl -k https://local-atticus-snipper.teamtaggle.com:8443/health
```

To view server logs (when running in background):
```
tail -f tmp/server.log
```

To stop the api server:
```
pkill -f "dist/proto-api" || kill $(cat tmp/server.pid 2>/dev/null) 2>/dev/null || true
```

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
./bin/300-run-tests            # all: go + govulncheck + unit + e2e + py (load is not part of all)
./bin/300-run-tests go         # go vet + go test (seconds; run before every commit)
./bin/300-run-tests unit       # vitest (src/web-app/tests)
./bin/300-run-tests e2e        # Playwright only
./bin/300-run-tests py         # pytest HTTP suite against a private API instance
./bin/300-run-tests load       # load test against a running API (see below); not part of `all`
```
- Sources `etc/loc-test.env` (override the file with `TEST_ENV=...`):
  scratch database names, `TEST_PG_URL_BASE` for the migrator,
  `TEST_DATABASE_URL` for the Go repository suite, `E2E_API_ENV`
  for the API instance the e2e/py suites start, `TEST_LOG_DIR`.
- `go`: rebuilds `loc_proto_test_db` via `bin/510`, runs `go vet ./...` and
  `go test -v ./...` with `TEST_DATABASE_URL` exported. A repository
  suite that cannot reach the database SKIPS (see
  `internal/repository/testdb_test.go`); the runner greps for that skip and
  fails the run, so a green run always includes the integration suites.
- `all` only: after the Go suites, `govulncheck ./...` in `src/api-srv`
  and `src/load-test`. The scanner is pinned (`GOVULNCHECK_VERSION` in
  the script) and installed once into `tmp/tools/bin/`. Any vulnerability
  the code calls, in a module or in the standard library of the toolchain
  `go.mod` selects (its `toolchain` line), fails the run; so does being
  unable to install it or reach vuln.go.dev. Not in `go` mode, which
  stays offline. Logs: `govulncheck-<module>.log`.
- `unit`: `npm run test:unit` (vitest) against `src/web-app/tests`.
- `e2e`: rebuilds `loc_proto_e2e_db` via `bin/510`, installs Chromium if
  needed, runs `bin/230-run-web-app-e2e`.
- `py`: rebuilds `loc_proto_e2e_db`, builds the API and starts it on
  `127.0.0.1:8082` from a copy of `etc/e2e-api-srv.env` with the developer
  rate limits (the pytest suite asserts a 429), bootstraps the pytest venv
  at `tests/.venv` itself, runs the suite, stops the API.
- `load`: delegates to `bin/330-test-load` (see below).
- No test script writes to the source tree any more: pytest's HTML report
  goes to `tmp/test-reports/`; logs land in `tmp/test-logs/` (`go-test.log`,
  `playwright.log`, `pytest.log`, `py-api.log`). Ends with a summary line,
  e.g. `[INFO] tests passed (all): go=105 unit=42 e2e=3 py=63`.
- Requires the `loc-postgres` container (doc/060-dev-env-start.md); nothing
  else -- the e2e API config has `DATABASE_URL` inline.

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

To load test the api server against a server you started (reads
`TEST_BASE_URL` from `etc/loc-api-srv.env`, or `LOAD_TEST_ENV=<file>` for
another):
```
./bin/330-test-load
./bin/330-test-load --user admin --password 'admin123!'   # defaults shown
./bin/330-test-load --insecure                             # opt-in TLS skip
```
- Builds `src/load-test` (its own Go module) and writes its report to
  `tmp/test-reports/`.

## Scratch Databases

```
./bin/510-reset-test-db [dbname]   # default loc_proto_test_db; DROP + rebuild
./bin/515-reseed-db [dbname]       # default loc_proto_e2e_db; TRUNCATE + re-seed without dropping
```
- `510`: DROP + CREATE, `sql/roles.sql`, schema through `cmd/migrate up`,
  then `sql/fixtures/test-seed.sql` (tenants 2 and 3) and
  `sql/fixtures/test-tenant-lock.sql` (`CHECK (security_context_id <> 1)`
  on every tenant table). Refuses any name not ending in
  `_test_db` or `_e2e_db`. Uses psql inside the `loc-postgres` container
  when it is running (`PG_CONTAINER`), a native `psql` otherwise (`PSQL`
  overrides); the migrator connects with `TEST_PG_URL_BASE`. This also
  covers what used to be the e2e-only reset: `bin/510-reset-test-db
  loc_proto_e2e_db`.
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
- Settings come from `etc/uat-deploy.env` (`APP_NAME`, `UAT_HOST`, `PG_CONTAINER`,
  `LOCAL_ENV_FILE`, `APP_ROOT`, `SERVICE`, `SVC_USER`); `PRESALES_ENV=<file>`
  swaps the file. No inline defaults -- a missing key stops the run.
- See doc/320-presales-uat-deploy.md. Run `./bin/300-run-tests` first.
- Post-sales pipeline placeholders: `infra/{uat,prd}/` (see `infra/README.md`).
