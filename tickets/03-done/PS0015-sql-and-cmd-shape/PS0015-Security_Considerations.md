# PS0015 — Security Considerations

Branch `TKT-PS0015-sql-and-cmd-shape`. Profiles **6** (data layer —
roles, migrations) and **8** (config/host setup). No endpoint changes.

## 6. Roles and migrations

**The role model is now written in one place** (`doc/340`) as real SQL,
not a commented-out file nobody could run: `<slug>_ddl` owns the schema
and only the migrator connects as it; `<slug>_app` is DML-only and is
what the API connects as; `<slug>_ci` can only create scratch databases.
The security property — a compromised API cannot alter the schema —
depends on step 2 of that doc being run on the host; nothing in the
repo can enforce it, so the doc says exactly what to run and what the
verifying query should print.

**One app role name.** Renaming `proto_app_svc_account` / `<env>_proto_user`
to `proto_app` changes no privilege: same `db_reader` + `db_writer`
membership, same DML-only intent. It removes the migration-day step
where a second app role was created and `DATABASE_URL` re-pointed, which
was also the step most likely to leave the old owner role lying around
with a live password. The presales password (`reallygoodPSQLpassword`)
is unchanged and still local-only; `DERIVING.md` §2 lists every file it
lives in.

**UAT hand step.** The live presales host must rename its role before
the next `bin/610` run, or the bundled `000-roles.sql` creates a fresh
`proto_app` and the restore hands the database to a role with the
roles.sql password rather than the host's. Written into the ticket's
Deploy needs and the Waiting-for note so it is not skipped.

**`migrate new`** writes only into the source tree (`-dir`, default
`sql/migrations`), never to a database; it takes no connection string.
Name validated to `[a-z][a-z0-9_]*`, so it cannot be steered into
another path.

## 8. Config

`fixtures/local-seed.sql` still seeds `admin`/`admin123!` and
`basic`/`basic123!` — unchanged, local-only, and the first item on the
derivation checklist. `bin/600` still refuses to bundle unless
`DB_APP_ROLE` is a role `roles.sql` creates (now `proto_app`).

## What did I miss?

- `doc/340` step 1 has the operator type passwords in a `psql` heredoc;
  same note as PS0004 — generate them into the env files first.
- Nothing here changes the migrator's advisory-lock decision (PS0007).
