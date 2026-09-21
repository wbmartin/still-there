-- Cluster-level roles. Run ONCE per Postgres server, as a superuser, before
-- the first migration: roles are cluster-wide, not per database, so they are
-- not a migration and the migrator's DDL role does not need CREATEROLE.
-- Idempotent -- safe to re-run; on a shared local server db_reader/db_writer
-- may already exist from another ProtoStack-derived app.
--
--   psql -d postgres -v ON_ERROR_STOP=1 -f src/api-srv/sql/roles.sql
--
-- bin/510-reset-test-db runs it before `migrate up` on a scratch database.
-- The service account below is the local default and is NOT what any
-- deployed environment connects as; it exists so a fresh local database has
-- a DML role to hand. The DDL owner role (proto_ddl) and the CI role
-- (proto_ci) are host setup, not a local default: see ddl-role-host-setup.sql.

-- Roles are CLUSTER-WIDE in Postgres, not per-database. On a shared local
-- server db_reader/db_writer may already exist from another app, so create
-- them idempotently rather than failing.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'db_reader') THEN
        CREATE ROLE db_reader;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'db_writer') THEN
        CREATE ROLE db_writer;
    END IF;
END
$$;

-- app svc account must be unique to each app, change for non-local installs
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_roles WHERE rolname = 'proto_app_svc_account'
    ) THEN
        CREATE USER proto_app_svc_account
            WITH PASSWORD 'reallygoodPSQLpassword';
    END IF;
END
$$;

GRANT db_reader TO proto_app_svc_account;
GRANT db_writer TO proto_app_svc_account;
