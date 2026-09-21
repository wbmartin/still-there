# Create the database

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
`sql/migrations/`), then `sql/fixtures/015-initialdata.sql` (seed groups,
`admin`/`admin123!`, `basic`/`basic123!`, two example rows). See the header of
that script for `PG_CONTAINER` / `PSQL` / `PGPASSWORD` / `TEST_PG_URL_BASE`.

Manual path, for a database with a name of your choosing (e.g. the
`loc_proto_db` that `etc/loc-api-srv.env`'s `DATABASE_URL` points at):

```bash
psql -h 127.0.0.1 -U postgres -d postgres -c "CREATE DATABASE loc_proto_db;"
psql -h 127.0.0.1 -U postgres -d postgres -v ON_ERROR_STOP=1 -f src/api-srv/sql/roles.sql
cd src/api-srv
go run ./cmd/migrate -url "postgres://postgres:<pw>@127.0.0.1:5432/loc_proto_db" up
#   or, when the env file carries DATABASE_URL for a role that owns the schema:
#   go run ./cmd/migrate -config ../../etc/loc-api-srv.env up
psql -h 127.0.0.1 -U postgres -d loc_proto_db -v ON_ERROR_STOP=1 -f sql/fixtures/015-initialdata.sql
go run ./cmd/migrate -url "postgres://postgres:<pw>@127.0.0.1:5432/loc_proto_db" status
#   expect: 0001  initial_schema   applied <now> by postgres
```

An existing database built by hand from the old `sql/010-initialcreate.sql`
already has the 0001 schema: bring it under the migrator with
`go run ./cmd/migrate -url ... stamp 1` instead of `up`.

For a deployed environment, the DDL owner role and default privileges are a
one-time host step: `src/api-srv/sql/ddl-role-host-setup.sql`.


# Nginx setup
wil mayneed to have a container running nginx.  This will depend on how we solve the file upload/download problem
