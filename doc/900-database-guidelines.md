# Database approach

This section describes how I'd like an AI to think and respond for database design and change decisions.

- When creating the database tables, please use "entity_name_id" pattern to name primary keys where the table is "entity_name".
- Table names should be singular names of the entity.
- We will only use one database per project, don't worry about abstracting for multiple database implementations. Unless otherwise sepcified, the database will be Postgres.
- All tables should have a "create_ts", "update_ts", and "delete_ts" of type numeric(17,7) for time and a "create_user_id", "update_user_id", "delete_user_id" of type integer.
- Each table should also have a "security_context_id" field of type integer to enable multi-tenant implementations.
- Tenant numbering is fixed: **1 is the first real tenant, 2 is the test tenant.**
  2 is reserved — every automated suite (Go integration, pytest, Playwright)
  runs in it, and no real tenant is ever assigned it. A row's tenant is taken
  from the authenticated caller's JWT claim, never from a client-supplied
  field; handlers overwrite `security_context_id` on the way in. Tests
  therefore select their tenant by the seeded user they authenticate as, and a
  test that must prove isolation uses 2 and 3 as the two sides.
- Test databases (`*_test_db`, `*_e2e_db`) hold no tenant 1. `bin/510`
  seeds `sql/fixtures/test-seed.sql` (logins and groups 1-3 in 2, groups 4-5
  in 3), then `sql/fixtures/test-tenant-lock.sql` adds
  `CHECK (security_context_id <> 1)` to every table with that column, read
  from the catalog so a new table is covered with no edit. A test that writes
  tenant 1 fails at the database. A developer's own database takes
  `local-seed.sql` (tenant 1) and is never locked.
- Every UPDATE/DELETE matches on the caller's tenant as well as the id, and
  the follow-up check that explains a zero-row write ("absent" 404 vs
  "stale `update_ts`" 409) uses the same tenant predicate. A row in another
  tenant reads as absent, so an id is never confirmed across tenants. Two
  named lookups are exempt: `RetrieveUserByUserName` and
  `GetGrantsForGroups` (see doc/105, Optimistic Locking).
- Isolation is enforced in application code only: no row-level security,
  and the API role reads and writes every tenant. A new tenant table ships
  with a cross-tenant negative test at the repository layer (read, update,
  delete of another tenant's row answer not-found); the threat model and
  the per-table list are in `doc/110-security-architecture.md`.
- Don't create foreign keys / Postgres references within the database - managed in the application layer similar to how nosql is managed.
- Don't create triggers for updating dates within the database - managed by the application layer.
- Schema changes are versioned SQL files applied by the project's own migrator (`src/api-srv/cmd/migrate`); see "Schema changes" below. No third-party migration framework.
- Database fields should be snake_case.
- When generating SQL in a .sql file it is okay to consolidate multiple fields into single lines, but each keyword (e.g. select, from, where)should start a new line.
- When generating SQL seed data (data.sql) that uses common calculated values like a timestamp:
  - Define the common values once in a single Common Table Expression (CTE) at the beginning of the script. (e.g., WITH common_vars AS (SELECT NOW() as ts, 0 as system_id)).
  - Use the INSERT INTO ... SELECT ... FROM common_vars_cte, (VALUES ...) AS data(...) format for inserting the data. This allows selecting values from both the CTE and the VALUES clause containing the row data.
- Strictly prohibit the use of database DEFAULT clauses, triggers, or SQL functions for generating or managing any application data fields like create_ts or update_ts. All such values must be calculated and set within the application code before executing the SQL INSERT or UPDATE statement.
- The database schema should define the create_ts, update_ts, and delete_ts columns (e.g., numeric(17,7) NOT NULL), but responsibility for providing their values lies entirely with the API application handlers.
- CRITICAL: For ALL SQL queries, `doc/003-schema-reference.sql` is the source for table and column names: the schema every migration in `src/api-srv/sql/migrations/` adds up to, generated (never hand-edited) so nobody has to replay `ALTER`s in their head. Do NOT infer, assume, or 'hallucinate' any column names not listed there for the respective table. Read it; change the schema only with a new migration.
- DO NOT add a delete_ts column to any query if it is not in the schema.

# Database related code approach

- Do not use wild cards in SQL queries. Always write every field name explicitly.
- Try to have one field name per line. use the 3 quotes strategy to cover multiple lines
- when modifying or creating tables, always add the following commands to the sql migration files.

```
GRANT USAGE ON SCHEMA biz TO group_global_writer;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA biz TO group_global_writer;
GRANT ALL ON ALL SEQUENCES IN SCHEMA biz TO group_global_writer;

GRANT USAGE ON SCHEMA appcore TO group_global_writer;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA appcore TO group_global_writer;
GRANT ALL ON ALL SEQUENCES IN SCHEMA appcore TO group_global_writer;
```

# Schema changes

The schema lives in `src/api-srv/sql/migrations/NNNN_name.sql`, embedded into
the `proto-migrate` binary (`cmd/migrate`) and applied in ascending version
order, forward only. The migrator records each file's SHA-256 in
`schema_migration` in the same transaction as the file itself.

- A change is a NEW file: the next `NNNN`, a short snake_case name, pure SQL
  (no `CREATE DATABASE`, no `\c`, no role creation -- roles are cluster-wide
  and live in `sql/roles.sql`). Include the GRANTs for any new table/sequence
  unless the host's default privileges already cover it.
- Never edit a migration that has been applied anywhere. Its checksum is
  stored; `migrate verify` (and `up`, which verifies first) refuses to proceed
  on drift. A wrong migration is fixed by the next one.
- Seed/test data is a fixture (`sql/fixtures/`), not a migration.
- `bin/510-reset-test-db` runs `migrate up` on an empty database, so every
  reset proves the full chain applies cleanly. It then writes
  `doc/003-schema-reference.sql` (`proto-migrate schema-reference`) from
  that database, and `bin/300-run-tests go` fails when the committed copy
  differs: a migration lands with its regenerated reference.
- A developer's own database: `bin/205-migrate-db` (creates it and seeds
  tenant 1 the first time, migrations only after that).

```
proto-migrate (-config <env file> | -url <url>) status     applied / pending / drifted / orphaned, per version
proto-migrate ... pending                                  count of pending files, for scripts
proto-migrate ... verify                                   fail if the database disagrees with the files
proto-migrate ... up                                       apply every pending file, in order
proto-migrate ... stamp <version>                          record up to <version> as applied WITHOUT running
proto-migrate ... schema-reference                         print the schema as CREATE TABLE/INDEX text (doc/003)
proto-migrate ... -base-url <origin> [-invite-ttl-hours N] invite <userName>
                                                           first sign-in link for an invited login (doc/330 A9)
```

`-config` reads only `DATABASE_URL` from a KEY=VALUE file -- the migrator's
own file with the DDL role's credentials, never the API's.
