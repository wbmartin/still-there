---
id: PS0004
title: Document the presales → post-sales deploy migration; fix bin/600 fail-fast
status: in-review
blocked_by:
waiting_on:
priority: high
domain: docs
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0004-one-deploy-story
merged: ef91f35
deployed: none
validated: 2026-09-21
---
# PS0004 — Two deploy paths; the move between them is a migration

**Source:** template review 2026-09-21.
**Decision (Brandon, 2026-09-21):** "Agree it's a migration, one we need
to plan for and should be addressed in documents." So: **no code
unification now.** Both paths stay. This ticket writes the plan and the
runbook, and makes the small script fixes that do not change either
contract.

## The two contracts (as found)

| | Presales `bin/600-bundle-uat` + `bin/610-deploy-uat` | Post-sales `infra/uat/bin/uat-proto-*` |
|---|---|---|
| Build | laptop, static linux/amd64 | on the server, triggered by shrmpl-hook webhook |
| Artifact transport | scp bundle to `/tmp` | oras registry |
| API binary name | `api-srv` | `proto-api` |
| Migrator | `proto-migrate` (verify only, with `--no-db`) | `proto-migrate verify → pending → dump → up` |
| DB roles | `proto_app_svc_account` owner + DML (`src/api-srv/sql/roles.sql`) | `proto_ddl` owner, `<env>_proto_user` DML, `proto_ci` (`sql/ddl-role-host-setup.sql`) |
| Migrator config | the API env passed as `-config` | dedicated `*-migrate.env` with the DDL role |
| Env file on host | `etc/api-srv.env` | `etc/uat-proto-api.env` |
| systemd unit | generated on first run, `User=` from `SVC_USER` | shipped from `infra/uat/etc/*.service`, `User=proto`, root-owned `.path` restart unit |
| Install layout | `$APP_ROOT/libexec/api-srv` | `libexec/proto-api` |
| DB refresh | `pg_dump` from laptop, wholesale restore | migrations only |

`bin/100-build-api-server` emits a third name, `dist/proto-go-app`.
`infra/prd/` is `infra/uat/` + promote + backup (17 files); the README
says the pipeline "is not wired up or exercised today".

## Deliverables

1. **`doc/3xx-presales-to-postsales-migration.md`** — the runbook for a
   host that was deployed with 600/610 and is moving to the infra
   pipeline. Ordered steps with the state before/after each: stop the
   presales unit; create the three roles and reassign ownership from
   `proto_app_svc_account` (exact `ALTER … OWNER TO`, `REASSIGN OWNED`);
   rename `libexec/api-srv` → `libexec/proto-api` or symlink; rename the
   env file and diff its keys against `infra/uat/etc/uat-proto-api.env`;
   replace the generated unit with the shipped one + `.path` unit;
   install shrmpl-hook + oras; first webhook build; verify; roll back
   path. Reference `infra/README.md`'s one-time host setup rather than
   repeat it.
2. **A "which path am I on" table** at the top of `doc/320-presales-uat-deploy.md`
   and `infra/README.md` pointing at each other and at the migration
   doc, so a derived app knows the two are different and where the seam
   is.
3. **Script fixes that change no contract:**
   - `bin/600`: check the local DB / `LOCAL_ENV_FILE` **before** building
     (today it builds Go + web for ~a minute, then dies with "cannot
     determine the local database" because `etc/loc-api-srv.env` has no
     `DATABASE_URL` — PS0003 also fixes the env side).
   - `bin/600`: remove the fallback branch that generates `000-roles.sql`
     with inline role names when `roles.sql` is absent (it never is;
     CLAUDE.md makes it mandatory). Header says "no inline defaults".
   - `bin/610`: the three inline `proto_app_svc_account` occurrences read
     from `etc/uat-deploy.env` instead.
4. **PS0007 input:** list every name that differs between the paths so
   the derivation checklist can say "rename in both".

## Not doing (recorded so nobody re-opens it by accident)

Unifying binary names, role models, unit files or env names across the
two paths; deleting `infra/prd/`. Revisit when a derived app actually
makes the migration and the runbook has been exercised once.

## What changed and why

Branch `TKT-PS0004-one-deploy-story`. Docs + two script fixes; neither
deploy contract changes.

| Deliverable | Done |
|---|---|
| 1. Runbook | `doc/330-presales-to-postsales-migration.md` — 8 ordered steps with state-after each (snapshot; stop unit; roles + per-object ownership to `proto_ddl` with `ALTER DEFAULT PRIVILEGES`; `stamp`/`status` under the migrator; env rename + key diff + API login switched to `<env>_proto_user`; users/dirs/shipped units + path unit; webhook/oras/first hand-run build; verify) and a rollback that works up to step 8. References `infra/README.md` one-time setup rather than repeating it. Explicitly: no `api-srv → proto-api` symlink (install-api's atomic `mv` would replace the link); never run 600/610 against a migrated host. |
| 2. Path table | "Which path am I on?" at the top of `doc/320` and `infra/README.md`, each pointing at the other and at `doc/330`; the one-liner test is `ls libexec` → `api-srv` vs `proto-api`. |
| 3a. `bin/600` fail-fast | New preflight block after arg parsing, before `go build`: `LOCAL_DB` from `LOCAL_ENV_FILE`, `roles.sql` present, `DB_APP_ROLE` named in it, and a real `psql -c 'select 1'` to the container/database (`pg_isready` was tried first — it ignores a missing database). Measured: 0.02–0.2 s to the error. |
| 3b. `bin/600` roles fallback | Deleted; `cp roles.sql` unconditionally. `LOCAL_ENV_FILE` comment in `etc/uat-deploy.env` no longer mentions the password. |
| 3c. `bin/610` role | `DB_APP_ROLE` is a new required key in `etc/uat-deploy.env` (`proto_app_svc_account`), shipped in the bundle's `deploy.env`, overridable by the host's `etc/deploy.env` like the other four. The `DO $$` ownership block gets it via `set_config`/`current_setting` because psql does not expand `:vars` inside `$$` bodies (proven against `loc-postgres`). `grep proto_app_svc_account bin/610` → 0. |
| 4. PS0007 input | "Names that differ" table in `doc/330` — 18 rows; binary, env, role, unit, user, restart, transport, DB name. Includes `bin/100`'s third name `dist/proto-go-app`. |

## Files modified

- `bin/600-bundle-uat`, `bin/610-deploy-uat`, `etc/uat-deploy.env`
- `doc/330-presales-to-postsales-migration.md` (new)
- `doc/320-presales-uat-deploy.md`, `infra/README.md`

## Endpoints introduced or modified

None.

## Acceptance

- [x] `doc/330` exists; `infra/README.md` and `doc/320` link it.
- [x] `bin/600` with a bad `LOCAL_ENV_FILE` fails in < 2 s (measured 0.2 s; missing DB 0.1 s).
- [x] `grep -n proto_app_svc_account bin/610` → 0.
- [ ] `bin/600 && bin/610` end-to-end on UAT — **Brandon runs; not now** (ProtoStack is not being published to UAT at present, 2026-09-21).

## Waiting for

The next real `bin/600`/`bin/610` run against UAT, to exercise `DB_APP_ROLE`
through the bundle and the `set_config` ownership block on a server. Until
then the change is proven locally only (script syntax, fail-fast timing,
the psql pattern against `loc-postgres`).

## Deploy needs

None for the docs. The `bin/610` change is exercised on the next UAT
deploy: the bundle now carries `DB_APP_ROLE`; a host with its own
`$APP_ROOT/etc/deploy.env` needs nothing added (the bundled value applies
unless overridden).
