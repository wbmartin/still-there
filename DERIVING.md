# Deriving a new app from ProtoStack

ProtoStack is a template. A new app starts as a copy of it, gets its own
name and its own secrets, and from then on has no
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
bin/000-rebrand acme "Acme Portal"                          # --module github.com/you/acme-api to set the Go module path
git status && git diff --stat                               # review; it is one commit
git commit -am "Derive acme from ProtoStack"
```

What it changed: `etc/app.env`; `proto-api`/`proto-migrate` → `acme-*`;
`loc_proto_*`, `uat_proto_db`, `proto_app`, `proto_ddl`,
`proto_ci`, `proto_app` → `acme` equivalents; `Proto App` → `Acme
Portal` (web title, login page, `APP_NAME` in every API env, systemd
`Description`); `uat-proto`/`prd-proto` file names and contents under
`infra/`; nginx upstream and site file names; the Go module path and every
import.
"ProtoStack" itself is left alone — it is where the app came from.

What it did not change, on purpose: the migrator's advisory-lock key
(`internal/migrate/migrate.go`, `"PROTO"`) — one migrator at a time per
Postgres cluster is the safe default for every derived app.

## 2. Secrets and passwords — nothing forces this, so do it now

| Where | What | Note |
|---|---|---|
| `src/api-srv/sql/roles.sql` | `reallygoodPSQLpassword` for `acme_app` | also in `etc/e2e-api-srv.env`, `etc/loc-api-srv.env`, `etc/loc-test.env` (×3), `infra/uat/etc/uat-acme-test.env` (×3) — keep them equal |
| `src/api-srv/sql/fixtures/test-seed.sql`, `local-seed.sql` | `admin` / `admin123!`, `basic` / `basic123!` | the pytest, Playwright and load suites log in with these (test seed, tenant 2; the load test reads `TEST_PERSONA_USER`/`TEST_PERSONA_PASSWORD` from `etc/e2e-api-srv.env`); the dev seed has the same logins in tenant 1; change the suites' fixtures with them, and the fixture-login list `src/api-srv/sql/export-scrub.sql` refuses |
| `etc/loc-test.env` | docker `postgres` superuser password | must match the container |
| `etc/loc-api-srv.env`, `etc/e2e-api-srv.env` | `JWT_SECRET`, `TOTP_ENCRYPTION_KEY` (32 bytes hex) | local-only, but make them yours |
| `etc/uat-api-srv.env.template`, `infra/*/etc/*-api.env` | `__SECRET__` / `SERVERONLY` placeholders | stay placeholders in git; real values are generated on the host (presales by `610`, post-sales by doc/330 / doc/331) |

**Customer data never enters git.** A tenant's source data (exports,
spreadsheets, import sets) and anything derived from it, including mock
data built from real records, lives in a directory outside the repository,
named by a key in the app's env file. The key is the derived app's to add;
ProtoStack has no import path. Fixtures, seeds and mock data are synthetic.
Deleting a committed file does not remove it: the first derived app
committed a tenant's export and needed a new repository, new webhooks, a
new deploy key and a re-clone of every checkout to get it out of history.
The rule has to hold before the first commit of real data.

## 3. Hosts

- **Presales** (`bin/600`/`610`): `etc/uat-deploy.env` — `UAT_HOST` (ssh
  alias), `PG_CONTAINER`. Service and paths are already derived from the
  slug. Host prep: `doc/320-presales-uat-deploy.md`.
- **Post-sales** (`infra/`): rename `infra/uat/` → `infra/<real-uat-host>/`
  and `infra/prd/` → `infra/<real-prd-host>/`, then fill the `SERVERONLY`
  values on the host per `infra/README.md`. Moving the presales UAT host
  onto this path is `doc/330-uat-postsales-cutover.md`; a fresh prod host
  is `doc/331-prd-postsales-install.md`. Both say `infra/uat` /
  `infra/prd`: use the renamed directories.

## 4. The example entity

`example` is the worked entity the others are copied from (`doc/810`).
Keep it until the first real entity exists, then delete
`src/api-srv/internal/{model,repository,handler}/example_*`, its routes in
`cmd/server/main.go`, `src/web-app/pages/example-*`, its store, schema,
routes and e2e spec, and drop the table in a migration. Do not edit the
initial migration.

## 5. Agent guidance

- `CLAUDE.md`: the first paragraph and the Layout list describe
  ProtoStack; make them describe this app. The Conventions and Branches
  sections carry over as they are.
- `README.md`: title, first paragraph, and drop this file's mention if you
  do not want derivation instructions in the derived app. `DERIVING.md`
  itself can go.

## 6. Local database and first run

```bash
docker exec loc-postgres psql -U postgres -c "CREATE DATABASE loc_acme_db"   # dev DB, once (doc/050)
bin/510-reset-test-db                                                      # loc_acme_test_db
bin/300-run-tests all                                                      # go + playwright + pytest, all green
bin/215-run-api-srv && bin/220-run-web-app                                 # http://localhost:5173
```

## 7. Check

```bash
# pr[o]to: spelled so bin/000-rebrand leaves this line alone
grep -rn -i 'pr[o]to\|wbmartin' bin etc src infra .claude CLAUDE.md README.md doc \
  | grep -v 'ProtoStack\|protocol\|X-Forwarded-Proto\|lockKey'
```

Nothing. If something prints, it is either a new file that hard-coded a
name (put it in `etc/app.env` or `constants/app.ts` instead) or a pattern
`bin/000-rebrand` missed — add it there so the next derivation gets it.

## 8. Pulling template changes into a derived app, later

ProtoStack keeps moving after you derive. The template tags `main` at each
sync point: `template-YYYY-MM-DD`. A derived app reviews the commits since
the last tag it synced to:

```bash
# in the derived app
git remote add template <protostack-remote>            # once
git fetch template --tags
git log --oneline template-<last-sync>..template/main  # what changed since
git show <sha>                                         # one change in full
```

For each commit decide pull / adapt / skip; where you pull, cherry-pick or
re-apply it. Names differ (`bin/000-rebrand` rewrote them), so a
cherry-pick of anything touching `etc/`, `infra/`, SQL role names or the
Go module path needs a rebrand pass by hand. Skip template-only changes
(`DERIVING.md`, `bin/000-rebrand`).

When you are done, record the template tag you synced to in your own
`CLAUDE.md`, so the next sync starts from there.
