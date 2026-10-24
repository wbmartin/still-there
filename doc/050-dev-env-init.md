# Dev environment: first-time setup

Day-to-day start is doc/060-dev-env-start.md; this page is the one-time
part.

## What runs in dev

Three processes, no nginx:

- Postgres on 127.0.0.1:5432 -- whatever serves that address (`PGHOST`,
  `PGPORT` in `etc/loc-test.env`); the `loc-postgres` container below is
  one way to have one.
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
- The Postgres client on the host: `bin/205`, `bin/510` and `bin/515` run
  `psql` against `PGHOST`:`PGPORT`. Install it even when the server
  itself is a container -- no `bin/` script reaches a database any other
  way. Docker only if a container is how you serve that address (below).
- Python 3 with `venv`, for the pytest suite; `bin/300-run-tests` builds
  `tests/.venv` itself. Playwright's Chromium is installed by the same
  script on first use.
- `openssl` only for `bin/400-make-local-dev-certs` (testing the API's TLS
  path; dev is plain HTTP).

## Create the database

Requires a Postgres listening on 127.0.0.1:5432 whose `postgres` role can
create databases, and nothing cares what serves it: a native server, the
`loc-postgres` container below, anything else. The address and the password
are `PGHOST`/`PGPORT`/`PGPASSWORD` in `etc/loc-test.env`, and every script
that reaches a database uses them. The schema is applied by the migrator,
never by hand.

**One Postgres per machine, shared by every app derived from this
template.** Each app's databases and its app role carry the app's slug
(`loc_acme_db`, `acme_app`), so two apps cannot collide; the cluster-wide
`db_reader`/`db_writer` roles are created idempotently for exactly that
reason (`src/api-srv/sql/roles.sql`). So if something already listens on
127.0.0.1:5432, use it -- there is nothing to install and nothing to
rename. `loc-postgres` names the machine's server, not this app, which is
why `bin/000-rebrand` leaves it alone (`DERIVING.md` section 1).

Nothing listening there yet? Create the container once (use the
`PGPASSWORD` value from `etc/loc-test.env`; on later days
`docker start loc-postgres`, see doc/060-dev-env-start.md):

```bash
docker run -d --name loc-postgres -e POSTGRES_PASSWORD='<PGPASSWORD>' \
  -p 127.0.0.1:5432:5432 postgres:17
```

Fastest path -- a scratch database, rebuilt from nothing every time:

```bash
bin/510-reset-test-db                    # -> loc_stillthere_test_db
bin/510-reset-test-db loc_stillthere_e2e_db # any *_test_db / *_e2e_db name
```

That runs, in order: `CREATE DATABASE`, `src/api-srv/sql/roles.sql` (cluster
roles, idempotent), `go run ./cmd/migrate ... up` (the embedded
`sql/migrations/`), then `sql/fixtures/test-seed.sql` (seed groups,
`admin`/`admin123!`, `basic`/`basic123!`, `mfa`/`basic123!` (not MFA-exempt),
`switcher`/`basic123!` (groups in tenants 2 and 3), two example rows) and
`sql/fixtures/test-tenant-lock.sql`. A scratch database is in tenants 2 and 3
only and refuses tenant 1 (doc/900), so it is for the test suites, not for
browsing as a developer. See the header of that script for `PGHOST` /
`PGPORT` / `PSQL` / `PGPASSWORD` / `TEST_PG_URL_BASE`.

Your own database -- the `loc_stillthere_db` that `etc/loc-api-srv.env`'s
`DATABASE_URL` points at, or a name you pass -- and, after every pull that
brings a migration, the same command again:

```bash
bin/205-migrate-db                       # -> loc_stillthere_db
```

It creates the database if missing, runs `roles.sql` and `migrate up`, and
seeds a database it just created with `sql/fixtures/local-seed.sql` (the
same logins, in tenant 1).

Manual path, the same steps by hand:

```bash
psql -h 127.0.0.1 -U postgres -d postgres -c "CREATE DATABASE loc_stillthere_db;"
psql -h 127.0.0.1 -U postgres -d postgres -v ON_ERROR_STOP=1 -f src/api-srv/sql/roles.sql
cd src/api-srv
go run ./cmd/migrate -url "postgres://postgres:<pw>@127.0.0.1:5432/loc_stillthere_db" up
#   or, when the env file carries DATABASE_URL for a role that owns the schema:
#   go run ./cmd/migrate -config ../../etc/loc-api-srv.env up
psql -h 127.0.0.1 -U postgres -d loc_stillthere_db -v ON_ERROR_STOP=1 -f sql/fixtures/local-seed.sql
go run ./cmd/migrate -url "postgres://postgres:<pw>@127.0.0.1:5432/loc_stillthere_db" status
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
(`infra/prd-wbm/etc/nginx_stillthere.teamtaggle.com`, doc/320). The
template has no file upload or download; a derived app that adds one
decides then whether dev needs more than the Vite proxy.

## A second machine (Linux)

The `bin/` scripts are bash and run unchanged on Linux. Differences from a
Mac:

- Docker Engine instead of Docker Desktop, if a container is how you serve
  Postgres; add yourself to the `docker` group, then start it as in
  doc/060. No `bin/` script runs `docker` itself.
- Debian/Ubuntu ship `python3` without `venv`: install `python3-venv`
  before the first `bin/300-run-tests`.
- Playwright's Chromium needs system libraries the first time:
  `cd src/web-app && npx playwright install-deps chromium` (sudo).
