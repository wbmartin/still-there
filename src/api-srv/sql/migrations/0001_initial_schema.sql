-- Initial database schema for the ProtoStack starter API server: the core
-- tables for authentication, authorization, and the example entity.
--
-- Conventions (doc/900-database-guidelines.md):
--   - Singular table names, "<entity>_id" SERIAL primary keys.
--   - No foreign keys; referential integrity is managed in the application layer.
--   - No triggers and no DEFAULT clauses on application data. create_ts,
--     update_ts and delete_ts are epoch-seconds NUMERIC(17,7) supplied by the
--     Go handlers on every INSERT/UPDATE.
--   - Soft deletes via delete_ts. Rows are never removed.
--   - Optimistic locking via update_ts on every table.
--   - security_context_id is the tenant discriminator.
--
-- Migration 0001. Applied by cmd/migrate to an EXISTING, EMPTY database whose
-- cluster already has the roles from sql/roles.sql. Pure SQL: no CREATE
-- DATABASE, no psql meta-commands, no role creation. A database built by hand
-- from the old sql/010-initialcreate.sql is brought under the migrator with
-- `migrate stamp 1`, not by running this.
--
-- Never edit this file once it has been applied anywhere: the migrator stores
-- its SHA-256 and `migrate verify` fails on drift. Schema changes are new
-- files, migrations/0002_<name>.sql and so on.

-- =============================================================================
-- APPCORE: authentication and authorization
-- =============================================================================

-- User groups table (per security context)
CREATE TABLE app_user_group (
    app_user_group_id SERIAL PRIMARY KEY,
    group_name VARCHAR(255) NOT NULL,
    user_grant_list INTEGER[] NOT NULL,
    security_context_id INTEGER NOT NULL,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER,
    UNIQUE (group_name, security_context_id)
);

-- Application users table
CREATE TABLE app_user (
    app_user_id SERIAL PRIMARY KEY,
    user_name VARCHAR(50) NOT NULL UNIQUE,
    pw_hash VARCHAR(125) NOT NULL,
    first_name VARCHAR(200) NOT NULL,
    last_name VARCHAR(200) NOT NULL,
    email VARCHAR(200) NOT NULL,
    security_context_id INTEGER NOT NULL,
    app_user_group_membership_list INTEGER[],
    mfa_exempt BOOLEAN NOT NULL,
    totp_seed VARCHAR(255) NOT NULL,
    mfa_setup BOOLEAN NOT NULL,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);

-- =============================================================================
-- APPLICATION: the example entity
-- =============================================================================

CREATE TABLE example (
    example_id SERIAL PRIMARY KEY,
    example_text TEXT NOT NULL,
    example_int INTEGER,
    example_float FLOAT,
    example_int_array INTEGER[],
    example_float_array FLOAT[],
    security_context_id INTEGER NOT NULL,
    create_ts NUMERIC(17,7) NOT NULL,
    create_user_id INTEGER NOT NULL,
    update_ts NUMERIC(17,7) NOT NULL,
    update_user_id INTEGER NOT NULL,
    delete_ts NUMERIC(17,7),
    delete_user_id INTEGER
);

-- =============================================================================
-- INDEXES
-- =============================================================================

CREATE INDEX idx_app_user_username ON app_user(user_name);
CREATE INDEX idx_app_user_security_context ON app_user(security_context_id);
CREATE INDEX idx_app_user_security_context_update_ts ON app_user(security_context_id, update_ts);
CREATE INDEX idx_app_user_id_update_ts ON app_user(app_user_id, update_ts);

CREATE INDEX idx_app_user_group_security_context ON app_user_group(security_context_id);
CREATE INDEX idx_app_user_group_security_context_update_ts ON app_user_group(security_context_id, update_ts);
CREATE INDEX idx_app_user_group_id_update_ts ON app_user_group(app_user_group_id, update_ts);

CREATE INDEX idx_example_security_context ON example(security_context_id);
CREATE INDEX idx_example_delete_user_id ON example(delete_user_id);
CREATE INDEX idx_example_security_context_update_ts ON example(security_context_id, update_ts);
CREATE INDEX idx_example_id_update_ts ON example(example_id, update_ts);

-- =============================================================================
-- GRANTS
-- Roles come from sql/roles.sql. Schema-level grants are per database, so
-- they belong here, after the tables they cover exist.
-- =============================================================================

GRANT USAGE ON SCHEMA public TO db_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO db_reader;

GRANT USAGE ON SCHEMA public TO db_writer;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO db_writer;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO db_writer;
