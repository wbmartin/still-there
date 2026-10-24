# Database host setup — the roles and the database

One-time, on the prd host (`prd-wbm`, shared with gsd-notes), as the
`postgres` superuser. Not a migration,
not embedded, not run by `bin/510`. The SQL is one idempotent file,
`infra/sql/stillthere-host-db.sql`, run from the install runbook (doc/320,
first install). It creates the cluster roles itself, so the host does
not need `src/api-srv/sql/roles.sql` first.

Two roles, two jobs (names derive from `APP_SLUG`, `etc/app.env`):

| Role | Owns | Connects as it | Where the credential lives |
|---|---|---|---|
| `stillthere_ddl` | the database, the `public` schema, every table and sequence | **only** `stillthere-migrate`, run by `prd-stillthere-install` as root | `etc/prd-stillthere-migrate.env` (root:root 0600) |
| `stillthere_app` | nothing — `db_reader` + `db_writer`, DML only | the API, as `svc-stillthere` | `etc/prd-stillthere-api.env` (root:svc-stillthere 0640) |

The file can also make a third, `stillthere_ci` (`CREATEDB`, for a build host
that runs the test gate against scratch databases). Prod runs no tests,
so it is given an empty CI password and makes no CI role.

Postgres lets only an object's owner (or a superuser) `ALTER` it. The
API's role must therefore never own the schema — a compromised API then
cannot change it — and the migrator's role must own all of it. The API's
user cannot read the migrator's env file either. `REVOKE CONNECT ...
FROM PUBLIC` keeps any other role on the cluster off the database.

The cluster is shared with gsd-notes: `db_reader` and `db_writer` are
cluster-wide and gsd's `gsd_app` is in them too, so the file keeps them
as they are and never drops them. What separates the two apps is
`CONNECT`: only `stillthere_ddl` and `stillthere_app` may connect to
`prd_stillthere_db`.

`roles.sql` creates `db_reader`, `db_writer` and `stillthere_app` for local
dev, with the dev password. Locally none of this file is needed:
`bin/510` runs the migrator as `postgres`, which owns what it creates.

## Run it

The runbook's env-file step has already generated each password into its
env file (`openssl rand -hex 32`). This block reads them back and hands
them to psql as `\set` lines on stdin, ahead of the SQL file. Nothing goes
on a command line: `printf` is a shell builtin, and `psql -v name=value`
would show the password in `ps`. The SQL file is opened by root and piped
in, so `postgres` needs no read access to it. In the root shell of
doc/320 step 3, after its step 5 unpacked the SQL file to
`/tmp/stillthere-infra/infra/sql/` (the block's rules: doc/320, How the
command blocks work). The output holds no password.

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  urlpw() { sed -n 's#^DATABASE_URL=postgres://[^:]*:\([^@]*\)@.*#\1#p' "$1" | head -1; }
  urldb() { sed -nE 's#^DATABASE_URL=postgres://[^/]+/([^?]+).*#\1#p' "$1" | head -1; }
  { printf '\\set db %s\n'     "$(urldb /srv/prd-stillthere/etc/prd-stillthere-migrate.env)"
    printf '\\set ddl_pw %s\n' "$(urlpw /srv/prd-stillthere/etc/prd-stillthere-migrate.env)"
    printf '\\set app_pw %s\n' "$(urlpw /srv/prd-stillthere/etc/prd-stillthere-api.env)"
    printf '\\set ci_pw %s\n'  ''
    cat /tmp/stillthere-infra/infra/sql/stillthere-host-db.sql
  } | sudo -u postgres psql -X
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

The file stops before changing anything if a variable is missing, a
password is not hex, or a database of that name exists and `stillthere_ddl`
does not own it. A role that already exists is kept and takes the
password and attributes given here; run it again with the same env files
and nothing changes.

The last thing it prints is the database with its owner and ACL: owner
`stillthere_ddl`, and `CONNECT` (`c`) for `stillthere_ddl` and `stillthere_app` only —
`=T/stillthere_ddl` (temporary tables for `PUBLIC`) is Postgres's default and
stays.

## The schema

The database starts empty. The first `prd-stillthere-install` runs
`stillthere-migrate up` as `stillthere_ddl`, and the default privileges the SQL set
give `db_reader` and `db_writer` their grants as each table is created:
no `GRANT` in any migration. The first login is then made with
`stillthere-migrate bootstrap-admin` (doc/320, first admin).

New migrations start with `go run ./cmd/migrate new <name>` in
`src/api-srv`, which creates the next `sql/migrations/NNNN_<name>.sql`.

## Verify

Ownership, then what each role may and may not do. The password goes to
psql in `PGPASSWORD`, the environment, not on its command line. In a
root shell (doc/320 step 3), after the first install:

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  urlpw() { sed -n 's#^DATABASE_URL=postgres://[^:]*:\([^@]*\)@.*#\1#p' "$1" | head -1; }
  sudo -u postgres psql -X -d prd_stillthere_db -c "SELECT tableowner, count(*) FROM pg_tables WHERE schemaname = 'public' GROUP BY 1"
  PGPASSWORD=$(urlpw /srv/prd-stillthere/etc/prd-stillthere-api.env) psql -X -h 127.0.0.1 -U stillthere_app -d prd_stillthere_db -c 'SELECT count(*) FROM app_user' -c 'CREATE TABLE nope (i int)'
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Expect one owner row, `stillthere_ddl | <table count>`; then the user count;
then `ERROR:  permission denied for schema public` — the API's role
cannot create (or alter, or drop) anything. The install prints
`verify: database matches the embedded migrations` as `stillthere_ddl` on every
release.
