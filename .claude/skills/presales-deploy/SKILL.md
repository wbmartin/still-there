---
name: presales-deploy
description: Ship this app to its UAT server (presales quick-deploy path — build the Go API + web app from local, dump the local Postgres, bundle, scp, hand the operator the one sudo command, then verify). Use when asked to "run a build cycle", "deploy to uat", "ship to uat", "health check the server", or whether UAT is what's expected. See doc/320-presales-uat-deploy.md for the full write-up.
---

# UAT build cycle (presales quick-deploy)

Two scripts do the work; this skill is the procedure around them.

| | |
|---|---|
| `bin/600-bundle-uat` | local: build `api-srv` (static linux/amd64) + `proto-migrate` (if `src/api-srv/cmd/migrate` exists) + `web-pub.tar.gz` + `db.sql` (pg_dump run *inside* the `$PG_CONTAINER` container) → `dist/${APP_NAME}_bundle_YYYYMMDD_hhmm.tar.gz` → scp to `$UAT_HOST:/tmp/` |
| `bin/610-deploy-uat` | server, shipped in the bundle as `deploy.sh`: stop `$SERVICE` → drop/recreate the UAT DB from `db.sql` → replace `$APP_ROOT/libexec/api-srv` (+ `proto-migrate`) → replace `$APP_ROOT/web-pub` → start → poll `/health` |

Defaults (override via env): `APP_NAME=proto`, `UAT_HOST=uat-unified001`,
`PG_CONTAINER=loc-postgres`, `APP_ROOT=/srv/uat-${APP_NAME}`, `SERVICE=uat-${APP_NAME}-api`,
`SVC_USER=www-data`.

**The operator runs the server step.** `sudo` on the UAT host needs a password, so Claude
never runs `deploy.sh`; it hands over the exact command and verifies afterwards.

## 1. Build + ship (Claude)

```bash
cd <repo>                                      # this app's repo
git checkout -q dev                            # UAT builds come from dev unless told otherwise
bin/600-bundle-uat 2>&1 | tail -15
ssh -o BatchMode=yes "$UAT_HOST" 'cd /tmp && f=$(ls -t '"$APP_NAME"'_bundle_*.tar.gz | head -1) && tar xzf "$f" && cat "${f%.tar.gz}/MANIFEST"'
```

The script prints `app=… branch=… commit=…` first — confirm it says `dev`. Untracked
files do not affect the build.

Flags: `--no-scp` (build only), `--no-db` (skip the dump).

## 2. Hand-off (one line, nothing else needed)

```
sudo /tmp/${APP_NAME}_bundle_YYYYMMDD_hhmm/deploy.sh
```

Add `--no-db` when the operator has been editing data on UAT and wants to keep the
server DB. Report the branch/commit and note anything odd in the build (dump size jump
= local data changed; `(dirty)` in MANIFEST = uncommitted source).

## 3. Verify after the operator says "done"

```bash
ssh -o BatchMode=yes "$UAT_HOST" '
systemctl is-active '"$SERVICE"'; ps -o pid,user,etime,cmd -C api-srv | tail -1
stat -c "%s %y %n" '"$APP_ROOT"'/libexec/api-srv
URL=$(sed -nE "s/^DATABASE_URL=(.*)/\1/p" '"$APP_ROOT"'/etc/api-srv.env)
psql "$URL" -Atc "select (select count(*) from information_schema.tables where table_schema='"'"'public'"'"') tables,(select count(*) from app_user) app_users,(select count(*) from app_user_group) app_user_groups,(select count(*) from example) examples,(select pg_size_pretty(pg_database_size(current_database())))"
curl -s -m 3 http://127.0.0.1:${SERVER_PORT:-8088}/health; echo
find '"$APP_ROOT"'/web-pub -name "._*" | wc -l'
```

Pass = service `active` as `$SVC_USER`, binary mtime ≈ deploy time, `/health` returns
200, DB counts plausible, `0` junk files. Report as a short table. Claude cannot log in
to the UI (no typing passwords) — the UI eyeball is the operator's.

## Server facts

- App root `$APP_ROOT` (`/srv/uat-${APP_NAME}`) — `libexec/api-srv` (+ `libexec/proto-migrate`
  if built), `web-pub/` (owned by `$SVC_USER`), `etc/api-srv.env` (DB URL, `SERVER_PORT`;
  **never overwritten by deploy**), `etc/uat-${APP_NAME}-api.service` (symlinked into
  `/etc/systemd/system`), `log/` (nginx only; API logs to journald —
  `sudo journalctl -u uat-${APP_NAME}-api -f`).
- DB role `proto_app_svc_account` owns the database; `db_reader`/`db_writer` grant roles.
  `000-roles.sql` in the bundle is idempotent (repo's `src/api-srv/sql/roles.sql` if
  present, else generated from the local env's `DATABASE_URL` password).
- Previous binaries/web kept as `.prev` (`api-srv.prev`, `proto-migrate.prev`,
  `web-pub.prev`) for a quick manual rollback — see doc/320-presales-uat-deploy.md.
- A normal deploy restores the whole local dump, so `schema_migration` (if present)
  travels with it — no migrate-apply step runs. `--no-db` runs
  `proto-migrate -config api-srv.env verify` instead, as a sanity check only.

## Known gotchas

- `main` may be far behind `dev`; a main build can roll UAT back. Always confirm the
  printed `branch=`.
- Working directory resets between turns — always `cd` into the repo explicitly.
- macOS tar adds `._*` files; the scripts set `COPYFILE_DISABLE=1 --no-xattrs` and deploy
  strips any leftovers. If `web-pub` ever shows `._index.html`, someone tarred by hand.
- The local dev env (`etc/loc-api-srv.env`) normally pulls secrets from vault, so it has
  no inline `DATABASE_URL` by default — the DB-dump step in `600-bundle-uat` needs one to
  find the local DB name/password. Either add an inline `DATABASE_URL` line locally, or
  set `LOCAL_DB=`/`DB_PASS=` overrides, or use `--no-db`.
- The dump is whatever is in the local dev DB — check row counts if totals look off.
