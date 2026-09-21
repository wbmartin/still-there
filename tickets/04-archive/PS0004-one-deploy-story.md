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

The presales quick-deploy (`bin/600`/`bin/610`) and the post-sales infra
pipeline (`infra/uat/bin/uat-proto-*`) differ in binary name, artifact
transport, DB role model, env file, and systemd unit. Brandon (2026-09-21):
"Agree it's a migration, one we need to plan for and should be addressed
in documents" — no code unification. This ticket wrote the runbook and
made two script fixes that change neither contract.

## What changed
- `doc/330-presales-to-postsales-migration.md` — 8-step ordered runbook (snapshot, stop unit, per-object `ALTER … OWNER` to `proto_ddl` with `ALTER DEFAULT PRIVILEGES` since `REASSIGN OWNED` refuses on public schema/database, `stamp`/`status` under the migrator, env rename + key diff, shipped units + path unit, webhook/oras/first build, verify) with a rollback that works through step 8; old role dropped only after verify since it is the rollback login.
- "Which path am I on?" table added to the top of `doc/320-presales-uat-deploy.md` and `infra/README.md`, each pointing at the other and at `doc/330`.
- `bin/600` preflight moved before `go build`: checks `LOCAL_ENV_FILE`'s `DATABASE_URL`, `roles.sql` presence, `DB_APP_ROLE` named in it, and a real `psql -c 'select 1'` (`pg_isready` was tried first — it ignores a missing database).
- `bin/600`'s fallback that generated `000-roles.sql` with inline role names when `roles.sql` was absent — deleted; `roles.sql` is copied unconditionally.
- `bin/610`'s three inline `proto_app_svc_account` occurrences replaced by `DB_APP_ROLE`, a new required key in `etc/uat-deploy.env`, shipped in the bundle's `deploy.env`; the `DO $$` ownership block reads it via `set_config`/`current_setting` because psql does not expand `:vars` inside `$$` bodies (first attempt used `:'role'` and silently failed to expand — proven against `loc-postgres`).

## Rules that survived
- The two deploy paths (presales bundle-and-scp vs. post-sales webhook-and-migrate) stay separate; do not unify binary names, role models, unit files, or env names across them.
- Never run `bin/610` against a host already migrated to the post-sales pipeline — it would `DROP DATABASE` and hand ownership back.
- A preflight check runs before the expensive build step, not after — `bin/600` now fails in under a second instead of after a full Go+web build.
- No inline defaults in deploy scripts; a missing config key dies by name (`roles.sql` fallback removed as a violation of this).

## Figures at close
| | date |
|---|---:|
| `bin/600` fail-fast on bad `LOCAL_ENV_FILE` | < 2 s (measured 0.2 s; missing DB 0.1 s) |
| `grep -n proto_app_svc_account bin/610` | 0 |
| `doc/330` names-that-differ table | 18 rows |
| end-to-end `bin/600 && bin/610` on UAT | not run — ProtoStack not deployed to UAT at close |

## Endpoints
None.

## Security
No endpoints, UI, or data-layer code touched — infrastructure/config only. Removing the `bin/600` roles fallback means the bundle no longer carries a generated SQL file with an inline DB password; it now ships only the committed `roles.sql` placeholder, already in git and used only if the role doesn't yet exist. `DB_APP_ROLE` is a role name, not a credential, and is committed in `etc/uat-deploy.env` on purpose. The role name reaches SQL in two places: `CREATE DATABASE ... OWNER $DB_APP_ROLE` (bash interpolation from root-owned files, same treatment `$DB_NAME` already had) and the `DO $$` ownership loop (passed as a psql variable, then `format('%I')`-quoted — stricter than the `$DB_NAME` sink). The `doc/330` runbook moves a host to least-privilege (API stops connecting as schema owner), tells the operator to generate new passwords for the three new roles, `REVOKE CONNECT ... FROM PUBLIC`, and keep `proto_app_svc_account` until verification passes as the rollback login.

## Left open
- End-to-end `bin/600`/`bin/610` run on UAT to exercise `DB_APP_ROLE` and the `set_config` ownership block on a real server — waits on the next real deploy.
- Root-shell password entry during `doc/330` step 3a can land in root's shell history — not mitigated in this ticket; flagged for a future doc line if wanted.
- `doc/330`'s names-that-differ table feeds PS0007 (derivation checklist).

## Sources
- `tickets/04-archive/engineering-log/260921_1456-TKT-PS0004-one-deploy-story.md`
- git: `ef91f35`
