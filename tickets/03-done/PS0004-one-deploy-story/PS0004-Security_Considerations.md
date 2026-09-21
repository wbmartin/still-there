# PS0004 — Security Considerations

Branch `TKT-PS0004-one-deploy-story`. Profiles: **8** (infrastructure /
config / deploy scripts). No endpoints, no UI, no data-layer code.

## 8. Infrastructure and config

**Secrets.** Removing the `bin/600` roles fallback removes the one place
that read the DB password out of `LOCAL_ENV_FILE` and wrote it into a
generated SQL file inside the bundle. The bundle now carries only the
committed `src/api-srv/sql/roles.sql` — whose `reallygoodPSQLpassword`
placeholder is the pre-existing local default, already in git, and is only
used if the role does not exist yet on the server (idempotent `IF NOT
EXISTS`). Unchanged by this ticket; `doc/320` already tells the operator
the host's `api-srv.env` holds the real credential. Net: the bundle carries
one less secret than before.

`DB_APP_ROLE` is a role *name*, not a credential; it is committed in
`etc/uat-deploy.env` on purpose like the other four keys.

**Injection surface in `bin/610`.** The role name now reaches SQL from a
variable instead of a literal. Two sinks:

- `CREATE DATABASE $DB_NAME OWNER $DB_APP_ROLE` — bash interpolation into a
  `-c` string, same treatment `$DB_NAME` already had. Both values come from
  root-owned files on the host (`deploy.env` from the bundle the operator
  just extracted as root, or `$APP_ROOT/etc/deploy.env`), not from the
  network. Anyone who can write them is already root.
- the `DO $$` ownership loop — passed as a psql variable (`-v role=`),
  quoted with `:'role'` into `set_config`, and then through `format('%I')`,
  so it is identifier-quoted at the point of use. Stricter than the
  `$DB_NAME` sink.

`bin/600`'s `grep -q "$DB_APP_ROLE" "$ROLES_SQL"` is a sanity check, not a
security control: it confirms the operator's env names a role the bundled
SQL will create.

**New preflight connects to the local container** with `psql -U postgres`
inside `docker exec` — same trust as the `pg_dump` that follows; no new
credential, no new network path.

**The runbook (`doc/330`)** moves a host to the least-privilege model the
post-sales pipeline was designed for: the API stops connecting as the
schema owner. It tells the operator to generate new passwords for
`proto_ddl` / `<env>_proto_user` / `proto_ci`, to `REVOKE CONNECT … FROM
PUBLIC`, and to drop `proto_app_svc_account` only after verification
(step 8) so it remains the rollback login until then. It warns that
running `bin/610` against a migrated host would `DROP DATABASE` and hand
ownership back — a destructive, privilege-widening action — and says to
remove the host from `etc/uat-deploy.env`.

The step-1 dump lands in `/root` (mode inherited from root's umask);
`doc/330` says to run as root. Contains the full database; same
sensitivity as the `backup/` dir `infra/README.md` already uses.

## What did I miss?

- `doc/330` step 3a writes passwords in a heredoc typed at a root shell —
  they will land in root's shell history if the operator pastes real
  values. Acceptable for a one-time runbook; noted in-doc? **No** — added
  nothing; an operator following `infra/README.md` conventions generates
  and stores them in the `etc/*.env` files first. Flag if you want a line.
- `--no-db` skips the whole preflight, including `roles.sql` presence.
  Correct: `--no-db` bundles no `000-roles.sql` and `610 --no-db` never
  reads it.
- Nothing in this ticket touches `infra/*/bin` scripts, nginx, or the
  webhook secret.
