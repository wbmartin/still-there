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
-- (proto_ci) are host setup, not a local default: see doc/340-db-host-setup.md.

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

-- The app role: DML only. Its name derives from APP_SLUG; bin/000-rebrand
-- rewrites it. Change the password on any non-local install (DERIVING.md).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_roles WHERE rolname = 'proto_app'
    ) THEN
        CREATE USER proto_app
            WITH PASSWORD 'reallygoodPSQLpassword';
    END IF;
END
$$;

-- Granted only when missing: on a build host bin/510 runs this file as the
-- CI role, which may not GRANT roles (Postgres 16 wants ADMIN OPTION even
-- for a membership that already exists).
DO $$
BEGIN
    IF NOT pg_has_role('proto_app', 'db_reader', 'MEMBER') THEN
        GRANT db_reader TO proto_app;
    END IF;
    IF NOT pg_has_role('proto_app', 'db_writer', 'MEMBER') THEN
        GRANT db_writer TO proto_app;
    END IF;
END
$$;
