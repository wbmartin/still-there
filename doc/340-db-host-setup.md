# Database host setup — the three roles

One-time, per deployed host, by hand, as a superuser. Not embedded, not
run by `bin/510`, not a migration. Do it after `src/api-srv/sql/roles.sql`
and before the first `migrate up` (or `migrate stamp`) on that database.

Three roles, three jobs (names derive from `APP_SLUG`, `etc/app.env`):

| Role | Owns | Connects as it | Where the credential lives |
|---|---|---|---|
| `proto_ddl` | every table and sequence; `CREATE` on `public` | **only** `proto-migrate` | `etc/<env>-proto-migrate.env` |
| `proto_app` | nothing — `db_reader` + `db_writer`, DML only | the API | `etc/<env>-proto-api.env` |
| `proto_ci` | only scratch `*_test_db` / `*_e2e_db` databases (`CREATEDB`) | the test harness on a build host | `etc/uat-proto-test.env` (UAT only) |

Postgres lets only an object's owner (or a superuser) `ALTER` it. The
API's role must therefore never own the schema — a compromised API then
cannot change it — and the migrator's role must own all of it.

`roles.sql` creates `db_reader`, `db_writer` and `proto_app` idempotently
for local dev and the presales bundle (`bin/610` runs it as
`000-roles.sql`). The presales path lets `proto_app` own the database
because it restores a laptop dump wholesale; moving such a host onto this
model is `doc/330-presales-to-postsales-migration.md`. Locally none of
this is needed: `bin/510` runs the migrator as `postgres`, which owns
what it creates.

## 1. The roles — as a superuser, on the cluster (any database)

```sql
CREATE ROLE proto_ddl LOGIN PASSWORD '<generate>';
CREATE ROLE proto_ci  LOGIN PASSWORD '<generate>' CREATEDB;   -- UAT only
GRANT CONNECT ON DATABASE uat_proto_db TO proto_ddl;

-- Optional but recommended: nothing connects to the real database by
-- accident. Grant CONNECT back only to the roles that need it.
REVOKE CONNECT ON DATABASE uat_proto_db FROM PUBLIC;
GRANT  CONNECT ON DATABASE uat_proto_db TO proto_app;
```

## 2. Ownership and default privileges — inside the database (`\c uat_proto_db`)

Per object, **not** `REASSIGN OWNED BY postgres TO proto_ddl`: REASSIGN
also tries to move the `public` schema and the database itself, which
Postgres refuses ("required by the database system"). Indexes follow
their tables. Reversible by running the same block with the roles
swapped. On a brand-new database with no tables yet the loops are no-ops.

```sql
DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT tablename AS name FROM pg_tables WHERE schemaname = 'public' LOOP
    EXECUTE format('ALTER TABLE public.%I OWNER TO proto_ddl', r.name);
  END LOOP;
  FOR r IN SELECT sequencename AS name FROM pg_sequences WHERE schemaname = 'public' LOOP
    EXECUTE format('ALTER SEQUENCE public.%I OWNER TO proto_ddl', r.name);
  END LOOP;
END $$;

GRANT CREATE ON SCHEMA public TO proto_ddl;

-- Anything proto_ddl creates from now on is readable/writable by the app
-- without each migration having to remember the GRANT.
ALTER DEFAULT PRIVILEGES FOR ROLE proto_ddl IN SCHEMA public GRANT SELECT ON TABLES    TO db_reader;
ALTER DEFAULT PRIVILEGES FOR ROLE proto_ddl IN SCHEMA public GRANT ALL    ON TABLES    TO db_writer;
ALTER DEFAULT PRIVILEGES FOR ROLE proto_ddl IN SCHEMA public GRANT ALL    ON SEQUENCES TO db_writer;

-- Expect one row: proto_ddl | <table count>
SELECT tableowner, count(*) FROM pg_tables WHERE schemaname = 'public' GROUP BY 1;
```

## 3. Bring the database under the migrator — as `proto_ddl`

```bash
proto-migrate -config etc/uat-proto-migrate.env up        # new, empty database
proto-migrate -config etc/uat-proto-migrate.env stamp 1   # existing schema that already IS 0001: record, do not run
proto-migrate -config etc/uat-proto-migrate.env status    # expect: 0001  initial_schema   applied <now> by proto_ddl
```

New migrations start with `go run ./cmd/migrate new <name>` in
`src/api-srv`, which creates the next `sql/migrations/NNNN_<name>.sql`.
