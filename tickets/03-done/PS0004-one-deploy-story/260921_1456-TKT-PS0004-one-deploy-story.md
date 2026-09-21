# 2026-09-21 — PS0004 one deploy story

Brandon: it's a migration, document it, no code unification. Wrote
`doc/330` as an 8-step runbook with state-after and rollback; the
interesting part is ownership — `REASSIGN OWNED` refuses on the public
schema and the database, so per-object `ALTER … OWNER` (as
`ddl-role-host-setup.sql` already says) plus a separate `ALTER DATABASE`.
Drop the old role only after verify; it is the rollback login.

Script side: `bin/600` now preflights everything the dump needs before
`go build` — `pg_isready` turned out useless (ignores a missing database),
so it is a real `psql -c 'select 1'`. `bin/610` takes `DB_APP_ROLE` from
the bundle; first attempt used `:'role'` inside the `DO $$` block, which
psql silently does not expand — `set_config`/`current_setting` instead,
proven on `loc-postgres`.

UAT end-to-end not run: ProtoStack is not being deployed to UAT right now.
Both PS0002 and PS0004 wait on the next real deploy.
