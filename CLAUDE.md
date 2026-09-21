# ProtoStack

The generic starter template other apps derive from: a Go stdlib
`net/http` + `pgx` API in `src/api-srv`, a React 19 / TypeScript / Vite /
Mantine web app in `src/web-app`. Nothing here is customer- or
app-specific — anything that becomes so belongs in the derived app, not
here.

## Layout

- `bin/` — numbered scripts: `1xx` build, `2xx` run/test-runner, `3xx`
  tests, `4xx` certs, `5xx` db, `6xx` deploy.
- `etc/` — environment files. `uat-deploy.env` drives the presales
  `bin/600`/`610` pair (no inline defaults in the scripts).
- `infra/<host>/` — mirror of `/srv/<env>-proto/{bin,etc}` on a server.
  Scripts deploy verbatim (no secrets, no per-host values); every value
  comes from `etc/*.env`, whose repo copies are redacted `SERVERONLY`.
  Placeholder until post-sales; presales uses `bin/600`/`610`.
- `src/api-srv/sql/` — `migrations/NNNN_name.sql`, embedded in
  `cmd/migrate`. This is the **only** way a schema change reaches a
  database; never hand-run DDL. Also `roles.sql` and `fixtures/`.
- `doc/` — reference docs.
- `tickets/` — every piece of engineering work and every open question for
  the customer, one file per item. See below.

## Conventions

The rules that bite on every change. Long form: `doc/900-database-guidelines.md`
(schema), `doc/905-config-guidance.md` (env files), `doc/904-python-guidance.md`
(pytest and scripts).

- Schema: no foreign keys, no triggers, no `DEFAULT`s for application data
  (`create_ts`/`update_ts`/`delete_ts` are set in Go); every table carries
  `security_context_id`; primary key is `<table>_id`; snake_case.
- Security checks are called explicitly in each handler (JWT → rate limit →
  CSRF → authorization → `CheckAllChecksComplete`); exceptions are written in
  the handler, not hidden in middleware. Non-negotiable.
- A list is `fooList` / `foo_list`, never a plural — variables, functions
  (`getUserList`), endpoints (`/exampleList`). Verbose names over terse
  (`tagCache`, not `tc`); a variable holding an `ExampleRepository` is
  `exampleRepository` (Go receivers may be one letter).
- Go and TypeScript: break lines around 80 columns.
- Config is a flat `KEY=VALUE` file passed to the binary; no environment-
  variable lookups.
- Git: summarize a PR from `git diff`, not memory, in markdown with headings.

## Tickets

Every piece of work is a ticket in `tickets/`. Ids are `PS` plus four
digits: `PS0000`–`PS0999` is engineering work, `PS1000`+ is a question for
the customer/stakeholder (`PS1001-ASK-CUSTOMER-slug.md`). The folder is the
state: `01-open/` → `02-working/PS0023-slug/` → `03-done/` → `04-archive/`.

**The `ticket-workflow` skill is the only copy of the procedure** — taking a
number, claiming, branching as `TKT-PS0023-slug`, the checks, asking
Brandon before the merge, validating on UAT. Load it when you pick a
ticket up. `ticket-triage` health-checks the set; `compact-tickets` folds
`03-done/` (and any legacy `doc/PS*/` folders) into `04-archive/`; `handoff`
closes a session.

```bash
python3 scripts/lint-tickets.py                   # before every commit touching tickets/
python3 scripts/lint-tickets.py --report          # next free number
python3 scripts/lint-tickets.py --update-status   # regenerate tickets/STATUS.md
```

## Branches

`main` is the default branch. Feature branches are `TKT-PS####-slug`;
pull requests target `main`, and only Brandon advances `main` — agents
never merge their own branch into it without his approval, and never
touch any branch but `main` and their own.

**Ticket papers live in the ticket folder, never in `doc/PS*/`.** The
global `pr-ticket-docs` skill's Description *is* the ticket body
(`tickets/02-working/PS0023-slug/PS0023-slug.md`), and the global
`security-considerations` skill writes `PS0023-Security_Considerations.md`
beside it. Both are required before asking for the merge, and
`security-considerations` must be complete before pushing to `main`.
