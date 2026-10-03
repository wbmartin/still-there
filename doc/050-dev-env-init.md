# Dev environment: first-time setup

Day-to-day start is doc/060-dev-env-start.md; this page is the one-time
part.

## What runs in dev

Three processes, no nginx:

- Postgres in the `loc-postgres` container on 127.0.0.1:5432 (the name
  the `bin/` scripts look for).
- The Go API, `bin/215-run-api-srv`: plain HTTP on 127.0.0.1:8080
  (`SERVER_BIND_ADDRESS`, `SERVER_PORT`, `ENABLE_TLS=false` in
  `etc/loc-api-srv.env`). Sessions, rate limits, lockouts and user blocks
  live in its in-process KV store (`internal/cache/kv.go`), so there is no
  Redis to run. It logs to stderr, or to `tmp/api-srv/server.log` when
  `bin/215` runs it in the background.
- Vite, `bin/220-run-web-app`, on :5173: serves the web app and proxies
  `/api` to the API with the prefix stripped (`src/web-app/vite.config.ts`;
  target `VITE_API_PROXY_URL` in `src/web-app/.env.development`).

## Tools

- Go: `src/api-srv/go.mod` names the version; its `toolchain` line is
  fetched by `go` when the local one is older.
- Node.js 22 (Vite 7 needs 20.19+ or 22.12+) with npm. Once, and after a
  pull that changes `package-lock.json`: `cd src/web-app && npm ci`.
- Docker, for `loc-postgres`. `psql` on the host is optional: the scripts
  run it inside the container when it is up.
- Python 3 with `venv`, for the pytest suite; `bin/300-run-tests` builds
  `tests/.venv` itself. Playwright's Chromium is installed by the same
  script on first use.
- `openssl` only for `bin/400-make-local-dev-certs` (testing the API's TLS
  path; dev is plain HTTP).

## Create the database

Requires a Postgres listening on 127.0.0.1:5432 whose `postgres` role can
create databases (the `loc-postgres` container from doc/060-dev-env-start.md,
or a native server). The schema is applied by the migrator, never by hand.

Fastest path -- a scratch database, rebuilt from nothing every time:

```bash
bin/510-reset-test-db                    # -> loc_proto_test_db
bin/510-reset-test-db loc_proto_e2e_db # any *_test_db / *_e2e_db name
```

That runs, in order: `CREATE DATABASE`, `src/api-srv/sql/roles.sql` (cluster
roles, idempotent), `go run ./cmd/migrate ... up` (the embedded
`sql/migrations/`), then `sql/fixtures/test-seed.sql` (seed groups,
`admin`/`admin123!`, `basic`/`basic123!`, `mfa`/`basic123!` (not MFA-exempt),
`switcher`/`basic123!` (groups in tenants 2 and 3), two example rows) and
`sql/fixtures/test-tenant-lock.sql`. A scratch database is in tenants 2 and 3
only and refuses tenant 1 (doc/900), so it is for the test suites, not for
browsing as a developer. See the header of that script for `PG_CONTAINER` /
`PSQL` / `PGPASSWORD` / `TEST_PG_URL_BASE`.

Your own database -- the `loc_proto_db` that `etc/loc-api-srv.env`'s
`DATABASE_URL` points at, or a name you pass -- and, after every pull that
brings a migration, the same command again:

```bash
bin/205-migrate-db                       # -> loc_proto_db
```

It creates the database if missing, runs `roles.sql` and `migrate up`, and
seeds a database it just created with `sql/fixtures/local-seed.sql` (the
same logins, in tenant 1).

Manual path, the same steps by hand:

```bash
psql -h 127.0.0.1 -U postgres -d postgres -c "CREATE DATABASE loc_proto_db;"
psql -h 127.0.0.1 -U postgres -d postgres -v ON_ERROR_STOP=1 -f src/api-srv/sql/roles.sql
cd src/api-srv
go run ./cmd/migrate -url "postgres://postgres:<pw>@127.0.0.1:5432/loc_proto_db" up
#   or, when the env file carries DATABASE_URL for a role that owns the schema:
#   go run ./cmd/migrate -config ../../etc/loc-api-srv.env up
psql -h 127.0.0.1 -U postgres -d loc_proto_db -v ON_ERROR_STOP=1 -f sql/fixtures/local-seed.sql
go run ./cmd/migrate -url "postgres://postgres:<pw>@127.0.0.1:5432/loc_proto_db" status
#   expect: one "applied" line per file in sql/migrations/
```

An existing database built by hand from the old (since deleted)
sql/010-initialcreate.sql already has the 0001 schema: bring it under the
migrator with
`go run ./cmd/migrate -url ... stamp 1` instead of `up`.

For a deployed environment, the DDL owner role and default privileges are a
one-time host step: `doc/340-db-host-setup.md`.

## Files and the web app

There is no nginx in the dev path: Vite serves the web app and proxies
`/api`, so the browser sees one origin. nginx exists only on deployed hosts
(`etc/uat-nginx-site.conf.template`, doc/320-presales-uat-deploy.md). The
template has no file upload or download; a derived app that adds one
decides then whether dev needs more than the Vite proxy.

## A second machine (Linux)

The `bin/` scripts are bash and run unchanged on Linux. Differences from a
Mac:

- Docker Engine instead of Docker Desktop; add yourself to the `docker`
  group (or the scripts' `docker exec` fails), then start the container as
  in doc/060.
- Debian/Ubuntu ship `python3` without `venv`: install `python3-venv`
  before the first `bin/300-run-tests`.
- Playwright's Chromium needs system libraries the first time:
  `cd src/web-app && npx playwright install-deps chromium` (sudo).
