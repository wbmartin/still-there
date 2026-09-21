---
id: PS0015
title: sql/ and cmd/ shape — role files, fixture naming, migrate `new`, load-test out of cmd/
status: ready
blocked_by: 
waiting_on:
priority: medium
domain: api
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
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

## Acceptance

- `sql/` contains only files that are SQL and are executed by something.
- `go run ./cmd/migrate -config … new add_widgets` creates
  `migrations/0002_add_widgets.sql`.
- `cmd/` holds only shipped binaries.

## Deploy needs

Go binary (migrate). If `roles.sql` moves, `bin/600` and `bin/610` follow
in the same PR.

## Direction (Brandon, 2026-09-21)

Three roles: app, ci, ddl. ddl and ci are manual host setup. Claimable.
`cmd/load-test` relocation is PS0014's.
