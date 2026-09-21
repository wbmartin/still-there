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
merged:
deployed:
validated:
---
# PS0015 — `sql/` and `cmd/` carry names from earlier eras

**Source:** template review 2026-09-21.
**Decision (Brandon, 2026-09-21):** the role model is **three roles —
app, ci, ddl** — and some of them are set up manually as part of server
setup. That resolves what this ticket was blocked on.

## `src/api-srv/sql/`

`embed.go` is good — it says what is and is not embedded. The rest:

- `ddl-role-host-setup.sql` is 100% commented-out prose: a document, not
  SQL. → `doc/9xx-db-host-setup.md` (or fold into `infra/README.md`'s
  one-time host setup).
- `roles.sql` today creates `db_reader`/`db_writer`/`proto_app_svc_account`
  (owner + DML, plaintext password `:36`) and `bin/600` ships it to UAT
  as `000-roles.sql`. Target: the three named roles. `ddl` (owner, runs
  migrations) and `ci` (CREATEDB) are created **manually at host setup**
  and documented, never by a shipped script; `app` (DML only) is what
  `roles.sql` creates idempotently for local dev and what the presales
  bundle may create on first install. Table ownership belongs to `ddl`,
  so the presales path's "app owns everything" is a documented
  divergence the PS0004 runbook fixes at migration time (REASSIGN OWNED).
  Decide the role *names* once (today: `proto_ddl`, `proto_ci`,
  `proto_app_svc_account` / `<env>_proto_user` — two names for the app
  role) and derive from `APP_SLUG` (PS0007).
- `fixtures/015-initialdata.sql` keeps a legacy `015-` number from the
  deleted `010-initialcreate.sql` era; admin password `admin123!`. →
  `fixtures/local-seed.sql`; the derivation checklist (PS0007) says to
  change the password.
- `cmd/migrate` `pending` counts orphan rows as applied — fine, note it.
  Add `migrate new <name>` that stamps the next `NNNN_` filename: the one
  growth path every derived app hits first.
- Migrator advisory-lock key `0x50524f544f` ("PROTO",
  `internal/migrate/migrate.go:35`): derived apps on one cluster share
  it. Decide whether that is desired; if not, derive from `APP_SLUG`
  (PS0007).

## `src/api-srv/cmd/`

- `cmd/load-test` (1,130 lines, inline HTML report template,
  `InsecureSkipVerify`, hardcoded `admin/admin123!` at `client.go:62-63`)
  is the largest package in the tree and the noisiest under lint, sitting
  beside `server` and `migrate`. → `tools/loadtest/` or its own module,
  or delete and have `bin/330` call k6/hey. (PS0014 also lists it; assign
  to one.)
- `cmd/server` takes a positional config path; `cmd/migrate` takes
  `-config`/`-url`. → same flag set (PS0011).

## `internal/model/`

DTOs + constants + one orphan: `security_model.go` mixes request/response
shapes for six endpoints; `grants.go` is really an authz catalog;
`user_grant_model.go` has no table. → `model/` = persisted rows;
`handler/dto.go` = wire shapes; grants → `internal/auth` (PS0008/PS0010).

## What changed and why

Branch `TKT-PS0015-sql-and-cmd-shape`, 2 commits. Items already landed
elsewhere: `cmd/load-test` out of `cmd/` (PS0014), lock key kept on
purpose (PS0007), `cmd/server` flag parity → PS0011, `internal/model/`
split → PS0010 (parked). Done here:

| Item | Change |
|---|---|
| `ddl-role-host-setup.sql` | deleted; it was a document. Now **`doc/340-db-host-setup.md`**: the three-role table (who owns, who connects, where the credential lives), the same SQL in real fenced blocks, and step 3 (bring under the migrator). `infra/README.md`, `doc/050`, `roles.sql`, the two `*-migrate.env` headers point at it. `embed.go` comment updated. |
| One name for the app role | `proto_app_svc_account` (local/presales) and `<env>_proto_user` (post-sales) were the same job under two names. Now **`<slug>_app`** everywhere: `roles.sql`, every API env file (loc, e2e, uat example, `infra/{uat,prd}`), `uat-deploy.env` (`DB_APP_ROLE=${APP_SLUG}_app`), `doc/320`, `doc/340`, `DERIVING.md`, the presales skill, `bin/000-rebrand` (one rule: `\b<slug>_(app\|ddl\|ci)\b`; re-proven in a worktree). `<slug>_ddl` and `<slug>_ci` stay as they were. |
| `doc/330` simplified by it | the migration no longer creates a second app role, never changes `DATABASE_URL`'s user, and has no "drop the presales role" step — `proto_app` keeps its name and password on both paths and merely stops owning the schema. |
| `fixtures/015-initialdata.sql` → `fixtures/local-seed.sql` | the `015-` number was from the deleted `010-initialcreate.sql` era. Ten references updated (`bin/510`, `515`, docs, READMEs, playwright config comment, the file's own header). |
| `migrate new <name>` | `go run ./cmd/migrate new add_widgets` → `sql/migrations/0002_add_widgets.sql` with the conventions header; numbers from the files on disk (so an uncommitted tree counts), validates `[a-z][a-z0-9_]*`, `-dir` for another directory, no database needed. Proven: two consecutive `new`s give 0002 and 0003; a bad name is refused. |

**Local side effect:** Brandon's docker role renamed
(`ALTER ROLE proto_app_svc_account RENAME TO proto_app`, password re-set
to the roles.sql value); `loc_proto_db` works unchanged.

## Files modified

- `src/api-srv/cmd/migrate/main.go`, `src/api-srv/sql/embed.go`, `sql/roles.sql`, `sql/fixtures/local-seed.sql` (from `015-initialdata.sql`); deleted `sql/ddl-role-host-setup.sql`
- new `doc/340-db-host-setup.md`; `doc/050`, `105`, `320`, `330`, `810`, `902`, `README.md`, `DERIVING.md`, `src/api-srv/README.md`
- `etc/app.env`, `loc-api-srv.env`, `e2e-api-srv.env`, `uat-api-srv.env.example`, `uat-deploy.env`; `infra/README.md`, `infra/{uat,prd}/etc/*-api.env`, `*-migrate.env`
- `bin/000-rebrand`, `bin/510-reset-test-db`, `bin/515-reseed-db`, `src/web-app/playwright.config.ts` (comment), `.claude/skills/presales-deploy/SKILL.md`

## Endpoints introduced or modified

None.

## Acceptance (met 2026-09-21)

- [x] `sql/` contains only SQL that something executes: `migrations/` (embedded, `cmd/migrate`), `roles.sql` (`bin/510`, `bin/610`), `fixtures/local-seed.sql` (`bin/510`, `515`), `embed.go`.
- [x] `go run ./cmd/migrate new add_widgets` creates `migrations/0002_add_widgets.sql`.
- [x] `cmd/` holds only shipped binaries: `server`, `migrate`.
- [x] `bin/300-run-tests all` go=163 unit=28 e2e=3 py=63 after the role rename.

## Deploy needs

Go binary (migrate). **UAT: the app role is renamed.** The presales host's
`etc/api-srv.env` `DATABASE_URL` says `proto_app_svc_account`; before or
with the next deploy run `ALTER ROLE proto_app_svc_account RENAME TO
proto_app` there and edit the URL (the bundled `000-roles.sql` would
otherwise create a second, empty `proto_app`, and `610` would hand the
restored database to it). One line in `doc/320`'s host prep covers new
hosts.

## Waiting for

The next UAT deploy, with the role rename above done by hand first.

## Direction (Brandon, 2026-09-21)

Three roles: app, ci, ddl. ddl and ci are manual host setup. Claimable.
`cmd/load-test` relocation is PS0014's.
