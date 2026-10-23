# Database host setup — the three roles

One-time, per deployed host, as the `postgres` superuser. Not a migration,
not embedded, not run by `bin/510`. The SQL is one idempotent file,
`infra/sql/stillthere-host-db.sql`, run from the runbook that needs it:
`doc/330-uat-postsales-cutover.md` (UAT) and
`doc/331-prd-postsales-install.md` (prod). It creates the cluster roles
itself, so a host does not need `src/api-srv/sql/roles.sql` first.

Three roles, three jobs (names derive from `APP_SLUG`, `etc/app.env`):

| Role | Owns | Connects as it | Where the credential lives |
|---|---|---|---|
| `stillthere_ddl` | the database, the `public` schema, every table and sequence | **only** `stillthere-migrate` (from `*-install-api`) and the one-time data load | `etc/<env>-stillthere-migrate.env` |
| `stillthere_app` | nothing — `db_reader` + `db_writer`, DML only | the API | `etc/<env>-stillthere-api.env` |
| `stillthere_ci` | only the scratch `*_test_db` / `*_e2e_db` databases it creates (`CREATEDB`) | the test harness on the build host | `etc/uat-stillthere-test.env`, `etc/uat-stillthere-e2e-api.env` (UAT only) |

Postgres lets only an object's owner (or a superuser) `ALTER` it. The
API's role must therefore never own the schema — a compromised API then
cannot change it — and the migrator's role must own all of it. `REVOKE
CONNECT ... FROM PUBLIC` keeps the CI role, and any other app's role on a
shared cluster, off the real database.

`roles.sql` creates `db_reader`, `db_writer` and `stillthere_app` for local dev
and the presales bundle (`bin/610` runs it as `000-roles.sql`, with the
template's literal password). The presales path lets `stillthere_app` own the
database because it restores a laptop dump wholesale; a host leaves that
model by `doc/330-uat-postsales-cutover.md`, which renames the presales
database aside rather than moving its ownership. Locally none of this is
needed: `bin/510` runs the migrator as `postgres`, which owns what it
creates.

## Run it

The runbook's env-file step has already generated each password into its
env file (`openssl rand -hex 32`). This block reads them back and hands
them to psql as `\set` lines on stdin, ahead of the SQL file. Nothing goes
on a command line: `printf` is a shell builtin, and `psql -v name=value`
would show the password in `ps`. The SQL file is opened by root and piped
in, because `postgres` cannot read under `/root`.

Set `ENV` (`uat` or `prd`), `APP_ROOT` and `SQL` (the path to
`infra/sql/stillthere-host-db.sql` on the host) as the runbook's variables
block does; on prod the test env file is absent, so the CI password is
empty and no CI role is made.

```bash
getv()  { sed -n "s/^$2=//p" "$1" 2>/dev/null | head -1; }
urlpw() { sed -n 's#^DATABASE_URL=postgres://[^:]*:\([^@]*\)@.*#\1#p' "$1" | head -1; }
urldb() { sed -nE 's#^DATABASE_URL=postgres://[^/]+/([^?]+).*#\1#p' "$1" | head -1; }
{ : "${ENV:?}" "${APP_ROOT:?}" "${SQL:?}"
  printf '\\set db %s\n'     "$(urldb "${APP_ROOT:?}/etc/${ENV:?}-stillthere-migrate.env")"
  printf '\\set ddl_pw %s\n' "$(urlpw "${APP_ROOT:?}/etc/${ENV:?}-stillthere-migrate.env")"
  printf '\\set app_pw %s\n' "$(urlpw "${APP_ROOT:?}/etc/${ENV:?}-stillthere-api.env")"
  printf '\\set ci_pw %s\n'  "$(getv "${APP_ROOT:?}/etc/${ENV:?}-stillthere-test.env" 'export PGPASSWORD')"
  cat "${SQL:?}"
} | sudo -u postgres psql -X
```

The file stops before changing anything if a variable is missing, a
password is not hex, or a database of that name exists and `stillthere_ddl`
does not own it (a presales database: rename it aside, `doc/330` A1). A
role that already exists is kept and takes the password and attributes
given here; run it again with the same env files and nothing changes.

The last thing it prints is the database with its owner and ACL: owner
`stillthere_ddl`, and `CONNECT` (`c`) for `stillthere_ddl` and `stillthere_app` only —
`=T/stillthere_ddl` (temporary tables for `PUBLIC`) is Postgres's default and
stays.

## Load the data as the DDL role

The dump the runbooks load is made `--no-owner --no-privileges`.
`pg_restore --role=stillthere_ddl` creates every object as `stillthere_ddl`, so it
owns all of it, and the default privileges the SQL set give `db_reader`
and `db_writer` their grants as each table is created: no ownership loop,
no `GRANT` pass. `postgres` runs it and reads the file from stdin, so no
password is needed and none is on a command line:

```bash
: "${ENV:?}" "${APP_ROOT:?}" "${DUMP:?}"
DB=$(sed -nE 's#^DATABASE_URL=postgres://[^/]+/([^?]+).*#\1#p' "${APP_ROOT:?}/etc/${ENV:?}-stillthere-migrate.env")
sudo -u postgres pg_restore --no-owner --no-privileges --role=stillthere_ddl --exit-on-error -d "${DB:?}" < "${DUMP:?}"
```

`pg_restore` must be the same Postgres major version as the `pg_dump` that
made the file, or newer; an older one stops with `unsupported version`.
The dump carries `schema_migration`, so the migrator knows where the
schema stands.

A host that starts **empty** instead skips the load: the first deploy's
`*-install-api` runs `stillthere-migrate up` as `stillthere_ddl`. A database whose
schema already **is** `0001` but has no `schema_migration` table is
recorded, not re-run, as the hook user:

```bash
: "${ENV:?}" "${APP_ROOT:?}"
sudo -u shrmpl-hook "${APP_ROOT:?}/libexec/stillthere-migrate" -config "${APP_ROOT:?}/etc/${ENV:?}-stillthere-migrate.env" stamp 1
sudo -u shrmpl-hook "${APP_ROOT:?}/libexec/stillthere-migrate" -config "${APP_ROOT:?}/etc/${ENV:?}-stillthere-migrate.env" status
```

New migrations start with `go run ./cmd/migrate new <name>` in
`src/api-srv`, which creates the next `sql/migrations/NNNN_<name>.sql`.

## Verify

Ownership, then what each role may and may not do. The passwords go to
psql in `PGPASSWORD`, the environment, not on its command line:

```bash
: "${ENV:?}" "${APP_ROOT:?}"
urlpw() { sed -n 's#^DATABASE_URL=postgres://[^:]*:\([^@]*\)@.*#\1#p' "$1" | head -1; }
DB=$(sed -nE 's#^DATABASE_URL=postgres://[^/]+/([^?]+).*#\1#p' "${APP_ROOT:?}/etc/${ENV:?}-stillthere-migrate.env")
sudo -u postgres psql -X -d "${DB:?}" -c "SELECT tableowner, count(*) FROM pg_tables WHERE schemaname = 'public' GROUP BY 1"
PGPASSWORD=$(urlpw "${APP_ROOT:?}/etc/${ENV:?}-stillthere-api.env") psql -X -h 127.0.0.1 -U stillthere_app -d "${DB:?}" -c 'SELECT count(*) FROM app_user' -c 'CREATE TABLE nope (i int)'
```

Expect one owner row, `stillthere_ddl | <table count>`; then the user count;
then `ERROR:  permission denied for schema public` — the API's role cannot
create (or alter, or drop) anything. On UAT, also the CI role:

```bash
: "${APP_ROOT:?}"
DB=$(sed -nE 's#^DATABASE_URL=postgres://[^/]+/([^?]+).*#\1#p' "${APP_ROOT:?}/etc/uat-stillthere-migrate.env")
PGPASSWORD=$(sed -n 's/^export PGPASSWORD=//p' "${APP_ROOT:?}/etc/uat-stillthere-test.env") psql -X -h 127.0.0.1 -U stillthere_ci -d "${DB:?}" -c 'SELECT 1'
```

Expect `FATAL:  permission denied for database`. After the first deploy,
`*-install-api` prints `verify: database matches the embedded migrations`
as `stillthere_ddl`.
