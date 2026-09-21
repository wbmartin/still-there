# Database approach

This section describes how I'd like an AI to think and respond for database design and change decisions.

- When creating the database tables, please use "entity_name_id" pattern to name primary keys where the table is "entity_name".
- Table names should be singular names of the entity.
- We will only use one database per project, don't worry about abstracting for multiple database implementations. Unless otherwise sepcified, the database will be Postgres.
- All tables should have a "create_ts", "update_ts", and "delete_ts" of type numeric(17,7) for time and a "create_user_id", "update_user_id", "delete_user_id" of type integer.
- Each table should also have a "security_context_id" field of type integer to enable multi-tenant implementations.
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
- CRITICAL: For ALL SQL queries, the migrations in `src/api-srv/sql/migrations/` are the exclusive and absolute source for table and column names. Do NOT infer, assume, or 'hallucinate' any column names not defined there for the respective table.
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
  reset proves the full chain applies cleanly.

```
proto-migrate (-config <env file> | -url <url>) status     applied / pending / drifted / orphaned, per version
proto-migrate ... pending                                  count of pending files, for scripts
proto-migrate ... verify                                   fail if the database disagrees with the files
proto-migrate ... up                                       apply every pending file, in order
proto-migrate ... stamp <version>                          record up to <version> as applied WITHOUT running
```

`-config` reads only `DATABASE_URL` from a KEY=VALUE file -- the migrator's
own file with the DDL role's credentials, never the API's.
