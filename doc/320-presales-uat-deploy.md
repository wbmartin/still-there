# Presales UAT deploy

## Purpose

A pre-agreement quick deploy for standing a demo/UAT instance up fast: local build +
local Postgres dump replace the UAT server wholesale. There is no webhook, no build
registry, no migrator-driven schema step — the local dev DB dump *is* the UAT DB after
each deploy.

This trades rigor for speed and is meant for the presales phase, before there's a
signed agreement to justify building the full-fat pipeline. Once that exists, the
post-agreement upgrade adds a webhook/registry-driven deploy and a real migrator step
(`proto-migrate`) instead of restoring a dump.

## Server layout

```
/srv/uat-<app>/
  libexec/
    api-srv           # installed by 610-deploy-uat
    proto-migrate      # installed if built into the bundle (used for --no-db verify only)
  web-pub/             # nginx serves this directly
  etc/
    api-srv.env        # real secrets — created once by hand, never shipped in the bundle
    uat-<app>-api.service
  log/                 # nginx only; API logs to journald
```

`<app>` is `$APP_NAME` (default `proto`); the systemd unit is `uat-${APP_NAME}-api`.

## One-time host prep

1. Create `/srv/uat-<app>/{libexec,web-pub,etc,log}`.
2. Copy `etc/uat-api-srv.env.example` to `/srv/uat-<app>/etc/api-srv.env` and replace
   every `SERVERONLY` placeholder with a real value (DB password, `JWT_SECRET`,
   `TOTP_ENCRYPTION_KEY`). This file is never overwritten by a deploy.
3. Point nginx at `web-pub/` for static files and proxy `/api/` to
   `127.0.0.1:$SERVER_PORT/`, stripping the `/api` prefix.
4. Make sure a `proto_app_svc_account`-owned Postgres instance is reachable at the
   `DATABASE_URL` in that env file — `bin/610-deploy-uat` creates the role and database
   itself on first deploy (see `000-roles.sql` in the bundle).

## The two commands

```bash
# 1. Local machine — build + bundle + ship
bin/600-bundle-uat                 # build, dump local DB, scp to $UAT_HOST:/tmp/
bin/600-bundle-uat --no-scp        # build only, skip the scp
bin/600-bundle-uat --no-db         # skip the local DB dump (server DB is kept)

# 2. Server — extract + deploy (needs sudo)
cd /tmp && tar xzf <app>_bundle_YYYYMMDD_hhmm.tar.gz
sudo ./<app>_bundle_YYYYMMDD_hhmm/deploy.sh              # replaces DB + binary + web
sudo ./<app>_bundle_YYYYMMDD_hhmm/deploy.sh --no-db      # keep the server's current DB
```

`610-deploy-uat` stops the service, optionally drops/recreates/restores the DB from
`db.sql`, installs `api-srv` (and `proto-migrate`, if it was built into the bundle) into
`libexec/`, replaces `web-pub/`, starts the service, and polls `/health` for 15s.

Because a normal deploy restores the whole local dump, the `schema_migration` table (if
present) travels with it — there is no separate migrate-apply step. When `--no-db` is
used instead, the server's existing DB might be ahead of or behind this build, so the
script runs `proto-migrate -config api-srv.env verify` after installing the binary as a
sanity check (warns, does not fail, if the migrator isn't in the bundle).

## Rollback

Both the binary and the web bundle are kept as `.prev` alongside the live copy
(`libexec/api-srv.prev`, `libexec/proto-migrate.prev`, `web-pub.prev`) after any deploy
that replaces them. To roll back by hand:

```bash
sudo systemctl stop uat-<app>-api
sudo mv /srv/uat-<app>/libexec/api-srv.prev /srv/uat-<app>/libexec/api-srv
sudo rm -rf /srv/uat-<app>/web-pub && sudo mv /srv/uat-<app>/web-pub.prev /srv/uat-<app>/web-pub
sudo systemctl start uat-<app>-api
```

There is no DB rollback beyond re-running a deploy from an older bundle's `db.sql`.

## Post-agreement upgrade

This presales path is intentionally disposable. Once there's a signed agreement, the
next step up is a webhook/registry-driven deploy pipeline with a real `proto-migrate`
apply step against a persistent UAT database, instead of restoring a fresh dump each
time.
