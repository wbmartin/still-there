# Guidance for config files

- use the simple env file format
- don't worry about environment variables. The config file has the value you
  need
- one flat `KEY=VALUE` file per host, passed to the binary as its first
  argument. The API (`cmd/server`) loads it with `config.LoadConfig`;
  `etc/loc-api-srv.env` is the dev one. `cmd/migrate` reads only
  `DATABASE_URL` from a file of the same shape (`-config`, e.g.
  `infra/<host>/etc/*-migrate.env`)
- a key the API needs is **present in the file**. `config.LoadConfig`
  refuses to start, naming the key, when one is missing rather than guessing
  a value: `DATABASE_URL`, `JWT_SECRET`, `TOTP_ENCRYPTION_KEY`,
  `SERVER_NAME`, `APP_NAME`, `SERVER_PORT`, `SERVER_BIND_ADDRESS`,
  `ENABLE_TLS`, `JWT_SESSION_EXPIRE_HOURS`, the two `MAX_*_REQ_PER_MIN`
  keys, the three
  `FAILED_LOGIN_*` keys, `INVITE_TTL_HOURS`, `LOG_LEVEL`,
  `ALLOW_MFA_EXEMPT_VIA_API`, and the TLS paths when
  `ENABLE_TLS=true`. A silent default is how a server ends up with every JWT
  born expired (0 hours) or every request refused 429 (a limit of 0), or an
  API meant to sit behind nginx answering on every interface. It
  also checks shape at load, not at first use: the port, hours, limits and
  threshold are whole numbers above 0, the two `FAILED_LOGIN_*_DURATION`
  keys are KV durations above 0 (`30s`, `5min`, `1h`), `LOG_LEVEL` is
  `debug`, `info`, `warn` or `error`, `ALLOW_MFA_EXEMPT_VIA_API` is
  exactly `true` or `false` (`true` only in the test harness file),
  `JWT_SECRET` is at least 32 characters, and `TOTP_ENCRYPTION_KEY` is
  exactly 64 hex characters (32 bytes, the AES-256 key the MFA seeds are
  encrypted under; `openssl rand -hex 32` makes either), `SMS_PROVIDER` is
  `log` or `telnyx` (with its four `TELNYX_*` keys) and
  `WATCHDOG_TICK_SECONDS` is at least 5. The optional keys
  (`TRUSTED_PROXY_COUNT`, `CORS_ALLOWED_ORIGINS`, the issue-report
  module's `ISSUE_REPORTS_ENABLED` and `ISSUE_RELAY_*`,
  `doc/150-issue-report-module.md`, the help module's `HELP_ENABLED`,
  `doc/151-help-module.md`, and the SMS relay's `SMS_RELAY_URL`,
  `SMS_RELAY_TOKEN` (a secret) and `SMS_RELAY_QUEUE`, all three or none,
  `src/sms-relay/README.md`) and the whole list are in `src/api-srv/README.md`.
  The relay itself (`src/sms-relay`) reads its own file of the same
  format, every key required
- the install (`infra/prd-wbm/bin/prd-stillthere-install`) never rewrites a
  host's env file, so a key that became required after the file was
  written must be added by hand before the new binary starts
  (`doc/320-prd-wbm-deploy.md`)
- **quoting**: the parser (`config.ReadEnvFile`) splits each line on the
  **first** `=`, trims whitespace from the key and the value, and then strips
  **exactly one** matching pair of surrounding quotes, `"…"` or `'…'`. That
  is the whole grammar, and it is how a value with spaces is written
  (`APP_NAME="Still There"`). Everything else in the value is literal:
  - no `export ` prefix (it becomes part of the key, so the key is not found)
  - no `${VAR}` or `$VAR` expansion, and no `\n` / `\"` escapes: this file
    holds secrets, and a password containing `$`, `\` or `#` has to reach the
    binary exactly as written
  - no inline comments: a `#` after the value is part of the value. Only a
    line whose first non-space character is `#` is a comment
  - a line with no `=` is skipped silently
  - a value quoted at one end only, or opening and closing with different
    quote characters, keeps those characters (`APP_NAME="Proto` gives the
    value `"Proto`, quote included; it is not an error); `""` is the empty
    string, and `""x""` strips one pair to `"x"`
  - a repeated key takes the last value in the file
- **before starting a binary built after PS0019 (`2b4f0e4`) against an env
  file written before it**, grep the host file for a quoted value: it used to
  be read *with* its quotes, so it changes value under the current parser.

  ```bash
  grep -n '="' /path/to/host.env     # double-quoted values
  grep -n "='" /path/to/host.env     # single-quoted values
  ```

  `JWT_SECRET` and `TOTP_ENCRYPTION_KEY` are the two that hurt: a quoted
  `JWT_SECRET` invalidates every live session, and a quoted
  `TOTP_ENCRYPTION_KEY` stops existing TOTP seeds from decrypting. Either
  unquote the value **and** keep the bytes the old binary saw (if the quotes
  really were part of the secret, double them: `JWT_SECRET=""abc""` parses
  as `"abc"`) or accept the re-key. `APP_NAME` is the one quoted value in
  `etc/loc-api-srv.env`, and losing its quotes is the point of the change
- **no shipped binary reads an environment variable.** Nothing in
  `cmd/server`, `cmd/migrate` or the `internal/` packages they import calls
  `os.Getenv`, `os.LookupEnv` or `os.Environ`; every value comes from the
  config file or a flag. Two libraries underneath do read their own, and
  neither is app configuration: pgx fills a connection setting the URL
  leaves out from the libpq variables (`PGHOST`, `PGUSER`, `PGPASSWORD`,
  `PGPASSFILE`, `PGSSLMODE` and the rest; a value in the URL always wins),
  and the Go runtime reads `GOMAXPROCS`, `GODEBUG` and `TZ`. So write every
  connection setting the API relies on, `sslmode` included, into
  `DATABASE_URL` rather than leave it to the service's environment
- every variable the tests and harnesses read, with its reader and who sets
  it. None is set on a server running the API:

  | Variable | Read by | Set by |
  | --- | --- | --- |
  | `TEST_DATABASE_URL` | the database-backed Go tests: `internal/repository/testdb_test.go`, `internal/migrate/migrate_test.go`, `cmd/migrate/invite_test.go`, `cmd/migrate/bootstrap_admin_test.go`. Without it they skip, so a bare `go test` never writes to a database a config file happened to name | `etc/loc-test.env` (a host: its own `TEST_ENV` file); `bin/300-run-tests go` passes it to `go test` |
  | `TEST_API_ENV` | `tests/conftest.py`: the private API's config file, read for its limits and checked to name a scratch database | `bin/300-run-tests py` (the copy of `etc/e2e-api-srv.env` it writes to the log directory) |
  | `TEST_BASE_URL` | `tests/conftest.py` and the `tests/test_*.py` modules | `bin/300-run-tests py`, from the `TEST_BASE_URL` key in that config file |
  | `E2E_API_ENV` | `src/web-app/playwright.config.ts` (default `etc/e2e-api-srv.env`); `bin/230-run-web-app-e2e`, `bin/300-run-tests`, `bin/330-test-load` | `etc/loc-test.env`; bin/230 and bin/300 make it absolute and export it |
  | `E2E_USER_NAME`, `E2E_PASSWORD` | `src/web-app/playwright.config.ts`, for global setup and the specs | nothing in the tree. The defaults, `admin` / `admin123!`, are the synthetic fixture login from `sql/fixtures/test-seed.sql` (`DERIVING.md`), not a secret; a host whose fixtures differ sets both |
  | `E2E_DB_SEEDED` | `src/web-app/tests/e2e/global-setup.ts`: set means skip the reseed | `bin/300-run-tests e2e`, which has just rebuilt the database |
  | `CI` | `src/web-app/playwright.config.ts`: forbid `.only`, two retries, never reuse a running server, list reporter | set by nothing in the tree; unset on a laptop (playwright reads it if a caller sets it) |
  | `<SERVICE>_LIVE` | a live test (`TestLive<Service>...`, `test_live_<service>_...`); the template has none | by hand only (`tests/README.md`, "Writing tests"); nothing in the gate or on a host sets one |

- the `bin/` scripts take a few overrides from the caller's environment;
  all but the last two default from a file, so a plain run needs none:

  | Variable | Read by | Set by |
  | --- | --- | --- |
  | `TEST_ENV` | `bin/300-run-tests`, `bin/330-test-load`: which harness file to source | default `etc/loc-test.env` |
  | `TEST_PG_URL_BASE`, `PGPASSWORD`, `PSQL` | `bin/205-migrate-db`, `bin/510-reset-test-db`, `bin/515-reseed-db` (each sources `etc/loc-test.env` only when `TEST_PG_URL_BASE` is unset) | `etc/loc-test.env`, which also exports `PGUSER`, `PGHOST`, `PGPORT` for psql; `PSQL` by hand |
  | `TEST_DB`, `E2E_DB`, `TEST_LOG_DIR` | `bin/300-run-tests`, `bin/330-test-load`, `bin/515-reseed-db` | `etc/loc-test.env` |
  | `SCHEMA_REFERENCE_FILE` | `bin/510-reset-test-db` (default `doc/003-schema-reference.sql`) | `bin/300-run-tests`, to a file in its log directory |
  | `FORCE` | `bin/400-make-local-dev-certs`: `FORCE=1` overwrites the pair | by hand |

- the web app: `src/web-app/vite.config.ts` reads `VITE_API_PROXY_URL`
  through `loadEnv`, and only for `vite serve` (the `/api` proxy target).
  `src/web-app/.env.development` sets it for the dev server and
  `src/web-app/playwright.config.ts` passes its own for the e2e stack.
  No `VITE_*` variable reaches the bundle: nothing under `src/web-app`
  reads import.meta.env, so `vite build` bakes no environment value into
  the browser code
