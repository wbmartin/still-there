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
pkill -f "proto-go-app" || kill $(cat tmp/server.pid 2>/dev/null) 2>/dev/null || true
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
./bin/222-build-web-app
```
- Builds the web app using `npm run build`
- Creates a timestamped tar.gz archive (e.g., `web-app_20251221_074106.tar.gz`)
- Archive contains all built files ready for deployment
- Copy the archive to your web server and extract with `tar -xzf filename.tar.gz`

## Testing Commands

The one command, on a laptop or a build host:
```
./bin/200-run-tests          # all: go + e2e + py
./bin/200-run-tests go       # go vet + go test (seconds; run before every commit)
./bin/200-run-tests e2e      # Playwright only
./bin/200-run-tests py       # pytest HTTP suite against a private API instance
```
- Sources `etc/loc-test.env` (override the file with `TEST_ENV=...`):
  scratch database names, `TEST_PG_URL_BASE` for the migrator,
  `TEST_DATABASE_URL` for the Go repository suite, `E2E_API_ENV`
  for the API instance the e2e/py suites start, `TEST_LOG_DIR`.
- `go`: rebuilds `loc_starter_test_db` via `bin/510`, runs `go vet ./...` and
  `go test -v ./...` with `TEST_DATABASE_URL` exported. A repository
  suite that cannot reach the database SKIPS (see
  `internal/repository/testdb_test.go`); the runner greps for that skip and
  fails the run, so a green run always includes the integration suites.
- `e2e`: rebuilds `loc_starter_e2e_db` via `bin/513`, installs Chromium if
  needed, runs `bin/230-run-web-app-e2e`.
- `py`: rebuilds `loc_starter_e2e_db`, builds the API and starts it on
  `127.0.0.1:8082` from a copy of `etc/e2e-api-srv.env` with the developer
  rate limits (the pytest suite asserts a 429), runs
  `bin/320-test-web-service --skip-lint`, stops the API.
- Logs land in `tmp/test-logs/` (`go-test.log`, `playwright.log`,
  `pytest.log`, `py-api.log`). Ends with a summary line, e.g.
  `[INFO] tests passed (all): go=105 e2e=3 py=63`.
- Requires the `loc-postgres` container (doc/060-dev-env-start.md); nothing
  else -- the e2e API config has `DATABASE_URL` inline.

To run the Playwright end-to-end suite directly:
```
./bin/230-run-web-app-e2e            # or: cd src/web-app && npm run test:e2e
./bin/230-run-web-app-e2e --headed   # any playwright args pass through
```
- `src/web-app/playwright.config.ts` starts its own stack: the Go API on
  :8082 from `etc/e2e-api-srv.env` and Vite on :5182 proxying to it, so a
  developer's 8080/5173 keep running. Global setup runs `bin/514`, signs in
  once as the fixture user (`admin` / `admin123!`) and saves the cookie to
  `tests/e2e/.auth/` (gitignored); specs start already signed in.
- Specs are `src/web-app/tests/e2e/*.spec.ts`; vitest owns `*.test.ts`.

To test the api server (repository + model unit tests, with coverage):
```
./bin/310-test-repo-model
```
- Connects via `TEST_DATABASE_URL` when set, otherwise through
  `etc/loc-api-srv.env`. Prefer `./bin/200-run-tests go`.

To test the api server (pytest HTTP suite against a server you started):
```
./bin/320-test-web-service                                   # server from etc/loc-api-srv.env
TEST_API_ENV=etc/e2e-api-srv.env ./bin/320-test-web-service # another server's config
```

To load test the api server:
```
./bin/330-test-load
```

## Scratch Databases

```
./bin/510-reset-test-db [dbname]   # default loc_starter_test_db
./bin/513-reset-e2e-db             # loc_starter_e2e_db (thin wrapper on 510)
./bin/514-reset-e2e-fixtures       # TRUNCATE + re-seed e2e db without dropping it
```
- `510`: DROP + CREATE, `sql/roles.sql`, schema through `cmd/migrate up`,
  then `sql/fixtures/015-initialdata.sql`. Refuses any name not ending in
  `_test_db` or `_e2e_db`. Uses psql inside the `loc-postgres` container
  when it is running (`PG_CONTAINER`), a native `psql` otherwise (`PSQL`
  overrides); the migrator connects with `TEST_PG_URL_BASE`.
- `513`/`514` source `etc/loc-test.env` when `TEST_PG_URL_BASE` is unset.
- `514` is what Playwright's global setup calls: dropping the database under
  a running API leaves its pool pointing at nothing.
- Users after a reset: `admin` / `admin123!` and `basic` / `basic123!`.

## Presales UAT Deploy

```
./bin/600-bundle-uat [--no-scp] [--no-db]   # build api + migrator + web, dump local db, scp bundle
sudo ./deploy.sh [--no-db] [--no-restart]   # bin/610-deploy-uat, shipped inside the bundle
```
- Settings come from `etc/uat-deploy.env` (`APP_NAME`, `UAT_HOST`, `PG_CONTAINER`,
  `LOCAL_ENV_FILE`, `APP_ROOT`, `SERVICE`, `SVC_USER`); `PRESALES_ENV=<file>`
  swaps the file. No inline defaults -- a missing key stops the run.
- See doc/320-presales-uat-deploy.md. Run `./bin/200-run-tests` first.
- Post-sales pipeline placeholders: `infra/{uat,prd}/` (see `infra/README.md`).
