-- One-time database setup for the SMS relay (src/sms-relay) on prd-wbm,
-- the same shape as stillthere-host-db.sql (doc/340-db-host-setup.md).
-- As the postgres superuser, the three variables on stdin AHEAD of this
-- file, never as `psql -v` (argv shows in ps):
--
--   { printf '\\set db %s\n'     prd_smsrelay_db
--     printf '\\set ddl_pw %s\n' "$DDL_PW"
--     printf '\\set app_pw %s\n' "$APP_PW"
--     cat smsrelay-host-db.sql; } | sudo -u postgres psql -X
--
-- Passwords are hex (openssl rand -hex 32). Two roles, two jobs:
--   smsrelay_ddl  owns the database, the public schema and every object in
--                 it; only `sms-relay migrate` acts as it
--                 (etc/prd-smsrelay-migrate.env)
--   smsrelay_app  owns nothing; SELECT, INSERT, UPDATE, DELETE on the
--                 tables, by default privileges; the relay
--                 (etc/prd-smsrelay.env)
-- The relay is split to its own repository later, so it is granted
-- directly and uses none of the cluster's shared db_reader / db_writer.
--
-- Idempotent: a role or database that exists is kept, and each role's
-- password and attributes are set to what this run is given. A database of
-- this name that smsrelay_ddl does not own is refused.
\set ON_ERROR_STOP on

SELECT :{?db} AND :{?ddl_pw} AND :{?app_pw} AS vars_set \gset
\if :vars_set
\else
DO $$ BEGIN RAISE EXCEPTION
  'set db, ddl_pw and app_pw on stdin first (header)'; END $$;
\endif

SELECT :'ddl_pw' ~ '^[0-9a-f]{32,}$' AND :'app_pw' ~ '^[0-9a-f]{32,}$'
  AS pw_ok \gset
\if :pw_ok
\else
DO $$ BEGIN RAISE EXCEPTION
  'passwords must be hex, 32+ characters (openssl rand -hex 32)'; END $$;
\endif

SELECT coalesce((SELECT datdba::regrole::text = 'smsrelay_ddl'
                   FROM pg_database WHERE datname = :'db'), true) AS owner_ok
\gset
\if :owner_ok
\else
DO $$ BEGIN RAISE EXCEPTION
  'database exists, not owned by smsrelay_ddl: rename it aside'; END $$;
\endif

SELECT format('CREATE ROLE %I', r)
  FROM unnest(ARRAY['smsrelay_ddl', 'smsrelay_app']) AS r
 WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = r) \gexec

ALTER ROLE smsrelay_ddl LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  PASSWORD :'ddl_pw';
ALTER ROLE smsrelay_app LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  PASSWORD :'app_pw';

SELECT format('CREATE DATABASE %I OWNER smsrelay_ddl', :'db')
 WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'db') \gexec

-- Only the relay's two roles connect.
SELECT format('REVOKE CONNECT ON DATABASE %I FROM PUBLIC', :'db')
\gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO smsrelay_ddl, smsrelay_app', :'db')
\gexec

\connect :db
ALTER SCHEMA public OWNER TO smsrelay_ddl;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO smsrelay_app;
ALTER DEFAULT PRIVILEGES FOR ROLE smsrelay_ddl IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO smsrelay_app;
ALTER DEFAULT PRIVILEGES FOR ROLE smsrelay_ddl IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO smsrelay_app;

-- Expect: owner smsrelay_ddl; CONNECT for smsrelay_ddl and smsrelay_app
-- only.
SELECT datname, datdba::regrole AS owner, datacl
  FROM pg_database WHERE datname = :'db';
