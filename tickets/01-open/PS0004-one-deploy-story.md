---
id: PS0004
title: Document the presales → post-sales deploy migration; fix bin/600 fail-fast
status: ready
blocked_by:
waiting_on:
priority: high
domain: docs
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
merged:
deployed:
validated:
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

## Acceptance

- The migration doc exists and `infra/README.md` + `doc/320` link it.
- `bin/600` with a bad `LOCAL_ENV_FILE` fails in < 2 s.
- `grep -n proto_app_svc_account bin/610` → 0 (reads a variable).
- `bin/600 && bin/610` still works end-to-end on UAT (Brandon runs).

## Deploy needs

None for the docs. The `bin/610` change is exercised on the next UAT
deploy.
