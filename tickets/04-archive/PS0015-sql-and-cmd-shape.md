---
id: PS0015
title: sql/ and cmd/ shape — role files, fixture naming, migrate `new`, load-test out of cmd/
status: in-review
blocked_by:
waiting_on:
priority: medium
domain: api
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0015-sql-and-cmd-shape
merged: 1fa0001
deployed: none
validated: 2026-09-21
---
# PS0015 — `sql/` and `cmd/` carry names from earlier eras

A template review found a commented-out "SQL" file that was really prose,
two names for the same app database role across deploy paths, a stale
fixture filename carrying a deleted migration's number, and no way to
scaffold a new migration file by hand. Brandon settled the role model —
three roles (app, ci, ddl), with ci and ddl created manually at host
setup — which unblocked the ticket.

## What changed
- `sql/ddl-role-host-setup.sql` deleted; replaced by
  `doc/340-db-host-setup.md` (real fenced SQL, the three-role table, step
  3 to bring the schema under the migrator); `infra/README.md`, `doc/050`,
  `roles.sql`, both `*-migrate.env` headers, and `embed.go`'s comment
  point at it.
- App role renamed to one name everywhere: `<slug>_app` (was
  `proto_app_svc_account` locally, `<env>_proto_user` post-sales) —
  `roles.sql`, every API env file, `infra/{uat,prd}`, `uat-deploy.env`
  (`DB_APP_ROLE=${APP_SLUG}_app`), `doc/320`, `doc/340`, `DERIVING.md`,
  the presales-deploy skill, `bin/000-rebrand`. `<slug>_ddl`/`<slug>_ci`
  names unchanged.
- `doc/330` simplified: the migration no longer creates a second app
  role, never repoints `DATABASE_URL`'s user, and drops the "drop the
  presales role" step.
- `sql/fixtures/015-initialdata.sql` → `sql/fixtures/local-seed.sql`; ten
  references updated (`bin/510`, `bin/515`, docs, READMEs, playwright
  config comment, file header).
- `cmd/migrate new <name>` (new): stamps the next `NNNN_name.sql`
  migration file from what's on disk (counts uncommitted files too),
  validates `[a-z][a-z0-9_]*`, takes `-dir` for another directory, needs
  no database connection.
- Local side effect: Brandon's dev-docker role renamed (`ALTER ROLE
  proto_app_svc_account RENAME TO proto_app`, password re-set to the
  `roles.sql` value).

## Rules that survived
- Three database roles: `<slug>_ddl` (owner, runs migrations), `<slug>_ci`
  (CREATEDB), `<slug>_app` (DML only) — `ddl` and `ci` are created
  manually at host setup, never by a shipped script; `app` is what
  `roles.sql` creates idempotently.
- One name for the app role across every deploy path (`<slug>_app`),
  derived from `APP_SLUG` — no more "two names, one job."
- `sql/` contains only SQL something executes: `migrations/`,
  `roles.sql`, `fixtures/local-seed.sql`, `embed.go`. A document that
  happens to be `.sql` goes in `doc/`.
- `cmd/` holds only shipped binaries (`server`, `migrate`).
- `migrate new` writes only to the source tree, never touches a
  database, and validates its name argument.

## Figures at close
| | date |
|---|---:|
| go tests | 163 |
| unit tests | 28 |
| e2e tests | 3 |
| pytest | 63 |

## Endpoints
None.

## Security
Profiles 6 (data layer — roles, migrations) and 8 (config/host setup).
The role model is now written as real, runnable SQL in `doc/340` rather
than a commented-out file nobody could execute; the security property —
a compromised API cannot alter the schema — depends on host-setup step 2
actually being run, which the repo can't enforce but the doc now states
exactly, including the verifying query. Renaming the app role to one name
changes no privilege (same `db_reader`+`db_writer` membership, still
DML-only) and removes a migration-day step that risked leaving an old
owner role with a live password lying around. The presales password is
unchanged and still local-only. `migrate new` takes no connection string
and validates its filename argument, so it can't be steered outside
`sql/migrations`.

## Left open
- Live presales UAT host still needs the role rename (`ALTER ROLE
  proto_app_svc_account RENAME TO proto_app`) run by hand before or with
  the next `bin/610` deploy, or the bundled `roles.sql` creates a second
  empty `proto_app`.
- `internal/model/` split (DTOs vs. persisted rows vs. grants catalog)
  parked, tracked by PS0010.
- `cmd/server`/`cmd/migrate` flag parity tracked by PS0011.
- Migrator advisory-lock key kept shared on purpose (PS0007 decision).

## Sources
- `tickets/04-archive/engineering-log/260921_1657-TKT-PS0015-sql-and-cmd-shape.md`
- git: `1fa0001`
