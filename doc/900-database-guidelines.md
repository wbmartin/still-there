# Database approach

This section describes how I'd like an AI to think and respond for database design and change decisions.

- When creating the database tables, please use "entity_name_id" pattern to name primary keys where the table is "entity_name".
- Table names should be singular names of the entity.
- We will only use one database per project, don't worry about abstracting for multiple database implementations. Unless otherwise sepcified, the database will be Postgres.
- All tables should have a "create_ts", "update_ts", and "delete_ts" of type numeric(17,7) for time and a "create_user_id", "update_user_id", "delete_user_id" of type integer.
- Each table should also have a "security_context" field of type integer to enable multi-tenant implementations.
- Don't create foreign keys / Postgres references within the database - managed in the application layer similar to how nosql is managed.
- Don't create triggers for updating dates within the database - managed by the application layer.
- Don't plan for a database migration tool to manage schema changes - managed manually through sql files.
- Database fields should be snake_case.
- When generating SQL in a .sql file it is okay to consolidate multiple fields into single lines, but each keyword (e.g. select, from, where)should start a new line.
- When generating SQL seed data (data.sql) that uses common calculated values like a timestamp:
  - Define the common values once in a single Common Table Expression (CTE) at the beginning of the script. (e.g., WITH common_vars AS (SELECT NOW() as ts, 0 as system_id)).
  - Use the INSERT INTO ... SELECT ... FROM common_vars_cte, (VALUES ...) AS data(...) format for inserting the data. This allows selecting values from both the CTE and the VALUES clause containing the row data.
- Strictly prohibit the use of database DEFAULT clauses, triggers, or SQL functions for generating or managing any application data fields like create_ts or update_ts. All such values must be calculated and set within the application code before executing the SQL INSERT or UPDATE statement.
- The database schema should define the create_ts, update_ts, and delete_ts columns (e.g., numeric(17,7) NOT NULL), but responsibility for providing their values lies entirely with the API application handlers.
- CRITICAL: For ALL SQL queries, you MUST use /atticus-web-snipper/doc/003-ai-postgres-schema-reference.sql as the exclusive and absolute source for all table and column names. Do NOT infer, assume, or 'hallucinate' any column names not explicitly defined in this file for the respective table.
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