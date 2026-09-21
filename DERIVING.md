# Deriving a new app from ProtoStack

ProtoStack is a template. A new app starts as a copy of it, gets its own
name, its own secrets and an empty ticket queue, and from then on has no
reason to say "proto" anywhere. This is the ordered list; `bin/000-rebrand`
does the mechanical half.

Identity lives in **`etc/app.env`** (`APP_SLUG`, `APP_DISPLAY_NAME`). Every
`bin/` script derives binary, database, service and role names from it.
Files that cannot source it — the flat API env files, SQL, `go.mod`, the
web app, `infra/` — are rewritten by the rebrand script.

## 1. Clone and rename

```bash
git clone <protostack-remote> acme && cd acme
git remote set-url origin <the new app's remote>
bin/000-rebrand acme "Acme Portal" --ticket-prefix AC       # --module github.com/you/acme-api to set the Go module path
git status && git diff --stat                               # review; it is one commit
git commit -am "Derive acme from ProtoStack"
```

What it changed: `etc/app.env`; `proto-api`/`proto-migrate` → `acme-*`;
`loc_proto_*`, `uat_proto_db`, `proto_app_svc_account`, `proto_ddl`,
`proto_ci`, `<env>_proto_user` → `acme` equivalents; `Proto App` → `Acme
Portal` (web title, login page, `APP_NAME` in every API env, systemd
`Description`); `uat-proto`/`prd-proto` file names and contents under
`infra/`; nginx upstream and site file names; the Go module path and every
import; `PS0023` → `AC0023` in skills, `CLAUDE.md` and the ticket linter.
"ProtoStack" itself is left alone — it is where the app came from.

What it did not change, on purpose: the migrator's advisory-lock key
(`internal/migrate/migrate.go`, `"PROTO"`) — one migrator at a time per
Postgres cluster is the safe default for every derived app.

## 2. Secrets and passwords — nothing forces this, so do it now

| Where | What | Note |
|---|---|---|
| `src/api-srv/sql/roles.sql` | `reallygoodPSQLpassword` for `acme_app_svc_account` | also in `etc/e2e-api-srv.env`, `etc/loc-api-srv.env`, `etc/loc-test.env` (×3), `infra/uat/etc/uat-acme-test.env` (×3) — keep them equal |
| `src/api-srv/sql/fixtures/015-initialdata.sql` | `admin` / `admin123!`, `basic` / `basic123!` | the pytest and Playwright suites log in with these; change the suites' fixtures with them |
| `etc/loc-test.env` | docker `postgres` superuser password | must match the container |
| `etc/loc-api-srv.env`, `etc/e2e-api-srv.env` | `JWT_SECRET`, `TOTP_ENCRYPTION_KEY` (32 bytes hex) | local-only, but make them yours |
| `etc/uat-api-srv.env.example`, `infra/*/etc/*-api.env` | `SERVERONLY` placeholders | stay placeholders in git; real values only on the host |

## 3. Hosts

- **Presales** (`bin/600`/`610`): `etc/uat-deploy.env` — `UAT_HOST` (ssh
  alias), `PG_CONTAINER`. Service and paths are already derived from the
  slug. Host prep: `doc/320-presales-uat-deploy.md`.
- **Post-sales** (`infra/`): rename `infra/uat/` → `infra/<real-uat-host>/`
  and `infra/prd/` → `infra/<real-prd-host>/`, then fill the `SERVERONLY`
  values on the host per `infra/README.md`. Moving a presales host onto
  this path later is `doc/330-presales-to-postsales-migration.md`.

## 4. The example entity

`example` is the worked entity the others are copied from (`doc/810`,
PS0010). Keep it until the first real entity exists, then delete
`src/api-srv/internal/{model,repository,handler}/example_*`, its routes in
`cmd/server/main.go`, `src/web-app/pages/example-*`, its store, schema,
routes and e2e spec, and drop the table in a migration. Do not edit the
initial migration.

## 5. Tickets and agent guidance

- `tickets/`: ProtoStack's history stays with ProtoStack. Empty
  `01-open/`, `02-working/`, `03-done/`, `04-archive/` (keep the
  folders), then `python3 scripts/lint-tickets.py --update-status`.
- `CLAUDE.md`: the first paragraph and the Layout list describe
  ProtoStack; make them describe this app. The Conventions, Tickets and
  Branches sections carry over as they are.
- `README.md`: title, first paragraph, and drop this file's mention if you
  do not want derivation instructions in the derived app. `DERIVING.md`
  itself can go.

## 6. Local database and first run

```bash
docker exec loc-postgres psql -U postgres -c "CREATE DATABASE loc_acme_db"   # dev DB, once (doc/050)
bin/510-reset-test-db                                                      # loc_acme_test_db
bin/200-run-tests all                                                      # go + playwright + pytest, all green
bin/215-run-api-srv && bin/220-run-web-app                                 # http://localhost:5173
```

## 7. Check

```bash
# pr[o]to: spelled so bin/000-rebrand leaves this line alone
grep -rn -i 'pr[o]to\|wbmartin' bin etc src infra scripts .claude CLAUDE.md README.md doc \
  | grep -v 'ProtoStack\|protocol\|X-Forwarded-Proto\|lockKey'
```

Nothing. If something prints, it is either a new file that hard-coded a
name (put it in `etc/app.env` or `constants/app.ts` instead) or a pattern
`bin/000-rebrand` missed — add it there so the next derivation gets it.
