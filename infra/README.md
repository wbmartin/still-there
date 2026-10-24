# infra/ — the prd host's mirror

One host, `prd-wbm`, no UAT, and one deploy: build on the laptop, scp the
bundle, install it on the host with one sudo command. The laptop half is
`bin/600-deploy-prd` (settings `etc/prd-deploy.env`); the host half is
here. The first install, every later release and rollback are
`doc/320-prd-wbm-deploy.md`; the database setup is
`doc/340-db-host-setup.md`. The host itself was built for gsd-notes (its
doc/360).

**The host is shared with gsd-notes**: its site, its API on 8088, its
database, fail2ban with its `jail.local` and the sshd jail, and certbot.
Everything here adds beside those and replaces nothing host-wide: this
app's root `/srv/prd-stillthere`, user `svc-stillthere`, group
`stillthere-deploy`, database `prd_stillthere_db`, API on
`127.0.0.1:8090`, two nginx sites of its own, a fail2ban jail in
`/etc/fail2ban/jail.d/` with its own filter.

| Repo dir | Host path | What |
|---|---|---|
| `infra/prd-wbm/bin/`, `etc/` | `/srv/prd-stillthere/bin/`, `/srv/prd-stillthere/etc/` | install and rollback scripts; env files, unit, the two nginx sites, the ACME bootstrap site, the fail2ban jail and filter |
| `infra/sql/` | — | `stillthere-host-db.sql`: the Postgres roles and the database (`doc/340-db-host-setup.md`) |

The two sites, so they can be separated later:

| Site file | Name | Serves |
|---|---|---|
| `nginx_stillthere.teamtaggle.com` | `stillthere.teamtaggle.com` | the web app, and `/api/` proxied to `127.0.0.1:8090` |
| `nginx_webhook.teamtaggle.com` | `webhook.teamtaggle.com` | only `POST /telnyx-inbound`, proxied to the API's `/smsInbound` (64k body cap); everything else 404 |

`acme_stillthere-bootstrap.conf` is a port-80-only site enabled once, for
the first certificates, and removed (doc/320, step 12).

## The convention

`infra/<host>/{bin,etc}` mirrors `/srv/<env>-stillthere/{bin,etc}` on that host,
file for file. **The repo is the source of truth for structure; the
filled file lives only on the host.**

- `bin/` scripts are **deployed verbatim**: copied to the host, mode
  0755, nothing edited there. They carry no secrets and no per-host
  values. Each one derives `$SRV_ROOT` from its own location, takes
  `APP_ENV="$(basename "$SRV_ROOT")"`, sources its
  `etc/$APP_ENV-*.env`, and dies naming the key if a required one is
  unset. No `${VAR:-default}` fallbacks. The slug is the part of
  `APP_ENV` after `<env>-`, and the binary and revision names derive
  from it. `bin/300-run-tests infra`
  checks all three, then runs both scripts on a fake app root under
  `tmp/` (`scripts/check-prd-install.sh`: first install, re-run, second
  release, damaged bundle, refused verify, rollback and back, an
  unhealthy release rolling itself back, an interrupted install
  resuming).
- **Names come from the app root**, `/srv/<env>-<slug>`: env files,
  unit and scripts carry the `prd-stillthere` prefix; the binaries are
  `api-srv` and `<slug>-migrate`.
- `etc/*.env` holds everything per-host or secret. The repo copy is
  **redacted**: every secret is spelled `SERVERONLY` and is generated on
  the host (`openssl rand -hex 32`), never on the laptop and never in git.
  `prd-stillthere-deploy.env` has no secret, so its repo copy is the host file.
  A deploy never rewrites an env file: a new or tightened key is added to
  the host's file by hand before the release that needs it.
- `etc/prd-stillthere-api.service`, the two nginx sites and the fail2ban
  jail and filter are copied to the host's `etc/` and symlinked into
  `/etc/systemd/system/`, `/etc/nginx/sites-enabled/` and
  `/etc/fail2ban/{jail.d,filter.d}/`, so each has one copy on the host.
  The unit runs the API as `svc-stillthere` in a systemd sandbox: the
  whole filesystem is read-only to it (`ProtectSystem=strict`; it logs to
  the journal and keeps its state in Postgres); no capabilities, IP and
  Unix sockets only, `@system-service` syscalls.
  `systemd-analyze security prd-stillthere-api` on the host scores it.
- nginx logs to `/var/log/nginx/prd-stillthere_{access,error}.log` and
  `/var/log/nginx/prd-stillthere-webhook_{access,error}.log`, which
  Debian's nginx logrotate already rotates. TLS is certbot's own lineage
  under `/etc/letsencrypt/live/<name>/`, one certificate per name, issued
  and renewed by webroot (`/var/www/letsencrypt`) so nginx never stops;
  no script of ours copies certificates.

| Script | Sources | Required keys |
|---|---|---|
| `prd-stillthere-install <bundle>` | `etc/prd-stillthere-deploy.env`; `etc/prd-stillthere-migrate.env` for `stillthere-migrate` and the database name | `SERVICE`, `HEALTH_URL`, `HEALTH_WAIT_SECONDS`; `DATABASE_URL` |
| `prd-stillthere-rollback` | the same | the same |

Both run as root (`sudo`), by staff only: sudo on the host needs the
staff member's password, so an agent never runs them.

## Layout on the host

```
/srv/prd-stillthere/                       root:root 0755
├── bin/        prd-stillthere-install, prd-stillthere-rollback       root 0755
├── etc/        prd-stillthere-api.env                  root:svc-stillthere 0640
│               prd-stillthere-migrate.env              root:root 0600
│               prd-stillthere-deploy.env, unit, nginx
│               sites, fail2ban jail and filter         root:root 0644
├── libexec/    api-srv, stillthere-migrate, stillthere-api.revision,
│               web-pub.sha256 (each + .prev)           root 0755 / 0644
├── web-pub/    the live web bundle nginx serves        root, a+rX
├── web-pub.prev/                                        root, a+rX
├── incoming/   bundles uploaded from the laptop        root:stillthere-deploy 2770
├── stage/      the install's checked copy of a bundle  root:root 0700
└── backup/     pre-migrate-<bundle>.dump               root:root 0700
```

**Uploads without sudo.** `incoming/` belongs to the group `stillthere-deploy`,
mode 2770: a staff login in that group scp's a bundle there as itself,
and the setgid bit keeps every upload in the group. Nothing runs from
`incoming/`: the install copies the bundle into root's `stage/`, checks
`SHA256SUMS` there and uses only that copy, so a group member cannot
swap a file between the check and its use. What goes live in `libexec/`
and `web-pub/` is root's, read-only to everyone else. Old bundles stay
in `incoming/` until the staff member removes them.

**The bundle**, as `bin/600-deploy-prd` builds it, in
`incoming/<YYYYmmddTHHMMSSZ>-<commit>/`:

| File | What |
|---|---|
| `api-srv` | `cmd/server`, linux/amd64, `CGO_ENABLED=0`, `-trimpath` |
| `stillthere-migrate` | `cmd/migrate`, the same |
| `web-pub.tar.gz` | the vite production build |
| `stillthere-api.revision` | `commit=` (full sha; the laptop refuses a dirty tree), `branch=`, `built_at=`, `go=`, `node=` |
| `SHA256SUMS` | the four above; the install refuses anything else listed or missing |

The install appends `bundle=` and `installed_at=` to the revision file
it puts in `libexec/`; `bin/600-deploy-prd --check` reads it back.

**Migrations are forward-only.** The install runs the incoming
`stillthere-migrate` before anything live changes, dumps the database to
`backup/` when a migration is pending, then applies it. A rollback puts
back the previous code and web bundle, never the schema.

## Deriving an app from this

`bin/000-rebrand <slug>` does the `stillthere` → `<app>` rename here too (file
names, contents, roles, nginx upstream). What is left for a human:
rename `infra/prd-wbm/` to the real host, give the nginx sites and
`CORS_ALLOWED_ORIGINS` the real public names, pick a free API port, and
set `etc/prd-deploy.env`.
