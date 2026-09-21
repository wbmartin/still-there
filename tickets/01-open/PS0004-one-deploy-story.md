---
id: PS0004
title: One deploy story — make presales 600/610 the degenerate case of the infra/ pipeline
status: ready
blocked_by:
waiting_on:
priority: high
domain: deploy
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
merged:
deployed:
validated:
---
# PS0004 — Two deploy stories with incompatible contracts

**Source:** template review 2026-09-21. **Decision ticket.**

## Background

PS0001 brought two deploy paths into the template. `infra/README.md`
presents them as stages of one journey ("presales uses 600/610; post-sales
fills in infra/<host>"). In practice they have different contracts, so
moving a host from presales to post-sales is a migration, not a fill-in.

| | Presales `bin/600-bundle-uat` + `bin/610-deploy-uat` | Post-sales `infra/uat/bin/uat-proto-*` |
|---|---|---|
| Build | laptop, static linux/amd64 | on the server, triggered by shrmpl-hook webhook |
| Artifact transport | scp bundle to `/tmp` | oras registry |
| API binary name | `api-srv` | `proto-api` |
| Migrator | `proto-migrate` (verify only, with `--no-db`) | `proto-migrate verify → pending → dump → up` |
| DB roles | `proto_app_svc_account` owner + DML (`src/api-srv/sql/roles.sql`) | `proto_ddl` owner, `<env>_proto_user` DML, `proto_ci` (`sql/ddl-role-host-setup.sql`) |
| Migrator config | the API env file passed as `-config` (works only because one role) | dedicated `*-migrate.env` with the DDL role |
| Env file on host | `etc/api-srv.env` | `etc/uat-proto-api.env` |
| systemd unit | generated on first run, `User=` from `SVC_USER` (www-data) | shipped from `infra/uat/etc/uat-proto-api.service`, `User=proto`, restart via root-owned `.path` unit |
| Install layout | `$APP_ROOT/libexec/api-srv` | `libexec/proto-api` |
| DB refresh | `pg_dump` from laptop, wholesale restore | none (migrations only) |

`bin/100-build-api-server` emits a third name, `dist/proto-go-app`.

Also found:

- **`bin/600` fails by default in this repo.** `etc/uat-deploy.env` sets
  `LOCAL_ENV_FILE=etc/loc-api-srv.env`, which has no `DATABASE_URL` line
  (secrets come from the vault), so step 4 dies with "cannot determine the
  local database" — after ~a minute of building Go + web. Fails-fast
  ordering and a vault-free local env (PS0003) both fix this.
- `bin/600`'s header says "no inline defaults" but it generates
  `000-roles.sql` with inline role names and `CREATE USER
  proto_app_svc_account` if `roles.sql` is absent (it never is).
- Role names are inline in `bin/610` (3 places), not env-driven.
- `infra/prd/` is `infra/uat/` plus promote + backup: 17 files of
  duplication for a pipeline the README says "is not wired up or exercised
  today". `infra/` totals 34 files / ~1,277 lines and hardcodes
  `shrmpl-hook` user, `oras`, `jq`, Slack, `/build-stg/proto`,
  `/srv/<env>-proto`, unix user `proto`, port 8088, `HOOKID` filename
  convention. It is coherent and env-driven, so it has value — but a
  derived app must sed ~100 `uat-proto|prd-proto` occurrences.
- `infra/README.md` "One-time host setup" is already half a runbook and
  overlaps `doc/320-presales-uat-deploy.md`.

## Options

**A. Unify (recommended).** Same host layout, same role model, same binary
names, same env file names, same unit file for both paths. `bin/610`
becomes "install this bundle the way `uat-proto-install-api` would", so
post-sales only *adds* the webhook/registry front end. Pick one role model
(the infra three-role one is the safer production shape; the presales
one-role is simpler — decide). Parameterise role names and the app slug
from one env file (ties to PS0007).

**B. Keep both, document the migration.** Cheapest now; every derived app
pays the migration later.

**C. Drop `infra/` from the template**, keep it as a design note in
`doc/`, stage it in the derived app that first goes post-sales.

Independent of A/B/C: drop `infra/prd/` until a UAT is real (re-derive it
then), and fix the `bin/600` fail-by-default.

## Acceptance (once decided)

- `bin/600 && bin/610` works on a fresh clone with the vault-free local
  env, and fails before building when the local DB cannot be determined.
- One binary name, one env file name, one role model across `bin/`,
  `infra/`, `etc/*.example`, `doc/320`, `infra/README.md`.
- `grep -rn 'proto_app_svc_account\|proto_ddl' bin infra etc src/api-srv/sql`
  shows the chosen model only.

## Deploy needs

Changes the UAT host layout if A. Coordinate with Brandon's next UAT deploy.

## Related

PS0003 (vault-free local env), PS0007 (single rename point), PS0015 (sql/
role files).
