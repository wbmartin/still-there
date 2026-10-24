-- One-time database setup on a deploy host (doc/340-db-host-setup.md).
-- As the postgres superuser, the four variables on stdin AHEAD of this file,
-- never as `psql -v` (argv shows in ps). printf is a shell builtin, so the
-- passwords reach no process's argv:
--
--   { printf '\\set db %s\n'     prd_stillthere_db
--     printf '\\set ddl_pw %s\n' "$DDL_PW"
--     printf '\\set app_pw %s\n' "$APP_PW"
--     printf '\\set ci_pw %s\n'  "$CI_PW"
--     cat stillthere-host-db.sql; } | sudo -u postgres psql -X
--
-- Passwords are hex (openssl rand -hex 32). ci_pw is empty on prd: no CI
-- role (prd runs no tests; a build host that did would set it).
--
-- Three roles, three jobs:
--   stillthere_ddl  owns the database, the public schema and every object in it;
--              only stillthere-migrate and the one-time data load act as it
--              (etc/<env>-stillthere-migrate.env)
--   stillthere_app  owns nothing; db_reader + db_writer, DML only; the API
--              (etc/<env>-stillthere-api.env)
--   stillthere_ci   a build host only, never prd: CREATEDB for the test run's
--              scratch databases; no CONNECT on the real one
--
-- Idempotent: a role or database that exists is kept, and each role's
-- password and attributes are set to what this run is given, so a role that
-- already exists takes the password in the env file.
-- A database of this name that stillthere_ddl does not own is refused: rename
-- the old one aside first. Run it again with the same values and
-- nothing changes. The server logs the passwords only if log_statement is
-- ddl or all (the default is none).
\set ON_ERROR_STOP on

SELECT :{?db} AND :{?ddl_pw} AND :{?app_pw} AND :{?ci_pw} AS vars_set \gset
\if :vars_set
\else
DO $$ BEGIN RAISE EXCEPTION
  'set db, ddl_pw, app_pw and ci_pw on stdin first (header, doc/340)'; END $$;
\endif

SELECT :'ddl_pw' ~ '^[0-9a-f]{32,}$' AND :'app_pw' ~ '^[0-9a-f]{32,}$'
   AND (:'ci_pw' = '' OR :'ci_pw' ~ '^[0-9a-f]{32,}$') AS pw_ok \gset
\if :pw_ok
\else
DO $$ BEGIN RAISE EXCEPTION
  'passwords must be hex, 32+ characters (openssl rand -hex 32)'; END $$;
\endif

-- Refuse before changing anything if the database exists under another
-- owner: it is renamed aside, never taken over.
SELECT coalesce((SELECT datdba::regrole::text = 'stillthere_ddl'
                   FROM pg_database WHERE datname = :'db'), true) AS owner_ok
\gset
\if :owner_ok
\else
DO $$ BEGIN RAISE EXCEPTION
  'database exists, not owned by stillthere_ddl: rename it aside (doc/340)';
END $$;
\endif

-- Roles are cluster-wide. db_reader / db_writer may already exist from
-- another app on the same cluster.
SELECT format('CREATE ROLE %I', r) FROM unnest(ARRAY[
    'db_reader', 'db_writer', 'stillthere_ddl', 'stillthere_app']) AS r
 WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = r) \gexec
SELECT 'CREATE ROLE stillthere_ci'
 WHERE :'ci_pw' <> ''
   AND NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'stillthere_ci') \gexec

ALTER ROLE stillthere_ddl LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  PASSWORD :'ddl_pw';
ALTER ROLE stillthere_app LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  PASSWORD :'app_pw';
SELECT format('ALTER ROLE stillthere_ci LOGIN NOSUPERUSER CREATEDB NOCREATEROLE '
              'PASSWORD %L', :'ci_pw')
 WHERE :'ci_pw' <> '' \gexec

SELECT format('GRANT %I TO stillthere_app', g)
  FROM unnest(ARRAY['db_reader', 'db_writer']) AS g
 WHERE NOT pg_has_role('stillthere_app', g, 'MEMBER') \gexec

SELECT format('CREATE DATABASE %I OWNER stillthere_ddl', :'db')
 WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'db') \gexec

-- Nothing connects to the real database by accident; the CI role cannot.
SELECT format('REVOKE CONNECT ON DATABASE %I FROM PUBLIC', :'db')
\gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO stillthere_ddl, stillthere_app', :'db')
\gexec

\connect :db
-- stillthere_ddl owns public on every Postgres version (15+ gives it to
-- pg_database_owner, 14 and older to postgres with CREATE for PUBLIC), so
-- nobody else may create there. Everything stillthere_ddl creates from now on,
-- the restored dump and every later migration, is readable by db_reader and
-- writable by db_writer without a GRANT in each migration.
ALTER SCHEMA public OWNER TO stillthere_ddl;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO db_reader, db_writer;
ALTER DEFAULT PRIVILEGES FOR ROLE stillthere_ddl IN SCHEMA public
  GRANT SELECT ON TABLES TO db_reader;
ALTER DEFAULT PRIVILEGES FOR ROLE stillthere_ddl IN SCHEMA public
  GRANT ALL ON TABLES TO db_writer;
ALTER DEFAULT PRIVILEGES FOR ROLE stillthere_ddl IN SCHEMA public
  GRANT ALL ON SEQUENCES TO db_writer;

-- Expect: the database owned by stillthere_ddl; CONNECT for stillthere_ddl and
-- stillthere_app only (=Tc/... for PUBLIC is gone).
SELECT datname, datdba::regrole AS owner, datacl
  FROM pg_database WHERE datname = :'db';
