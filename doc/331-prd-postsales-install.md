# Prod install — local data → a fresh prod host

The runbook that installs the app on its prod host and loads it from your
local database: a fresh install, with no `/srv/prd-proto`, no database and
no roles yet. It follows `doc/330-uat-postsales-cutover.md`, which
rehearsed every step on UAT, and it loads **the same dump** UAT loaded,
normally the same day: one dump, one freeze, from local. Nothing is taken
from UAT but the registry build A7 promotes.

**Prod has no checkout and never builds.** The binaries and the web bundle
come from the registry: `prd-proto-promote-uat` retags the digests UAT
built and tested as `:prd`, and `prd-proto-install-api` /
`prd-proto-publish-web-client` pull them. Everything else the host needs
from the repo (`bin/`, env templates, units, nginx site, monitors,
fail2ban stanza, `infra/sql/proto-host-db.sql`) arrives as a `git archive`
of `infra/prd` and `infra/sql` at a pushed commit, so what is installed is
always a commit, never local edits.

This shape ran on real hosts in a derived app on 2026-10-01; this generic
form has not run as written yet. Tick each step off as it runs and add
what surprised you to **Lessons** at the bottom (and to `doc/330`'s).

## What changes

| | Local (today) | Prod (after) |
|---|---|---|
| System of record | the local database | `prd_proto_db` on prod; local becomes a stale dev copy |
| Binaries | built locally | UAT's registry digests retagged `:prd`; `libexec/proto-api`, `proto-migrate`, `proto-api.revision`, shrmpl-hook-owned |
| Host files | — | a `git archive` of `infra/prd` + `infra/sql` (no checkout) |
| Deploy | — | `prd-proto-promote-uat-and-deploy`: by hand, or from a prod webhook (B6) |
| Env | `etc/loc-api-srv.env` | `etc/prd-proto-{api,migrate,deploy,backup}.env` |
| DB | `postgres` owns it | owned by `proto_ddl`; the API is `proto_app` (DML); no CI role |
| Backups | none | nightly `prd-proto-backup-all` → `REMOTE_SERVER`, plus `install-api`'s pre-migrate dump |
| Monitoring | none | nackmon (backup, cert renewal), pulsecheck (api, web), the `[prd-proto]` jail |

`infra/prd` is the template's name for this host's mirror; a derived app
has renamed it — use that name wherever this runbook says `infra/prd`.

## 0. Before the install

Nothing below touches prod's other apps. Block conventions as in `doc/330`
§0: **local blocks** (zsh, repo root) carry no inline `#` comments and
read `PRD_SSH` and `MON_SSH` (the host that runs nackmon and pulsecheck;
it may be prod itself), which you set once in that terminal; **host
blocks** run in bash as root.

**0.1 Have to hand**

- `doc/330` run, and its dump still here: `ls dist/*-data-*.dump`.
- The freeze has held since that dump: nobody has edited the local data.
  If someone has, make a new dump (`doc/330` 0.4, `scp` to prod instead)
  and load that.
- DNS: the prod site name resolves to the host. Certbot's HTTP challenge
  in A8 needs it.
- The registry login (UAT's), the Slack webhook URL, the certbot e-mail,
  and the backup destination: an ssh alias in root's `~/.ssh/config` on
  prod (`REMOTE_SERVER`) and a directory there (`REMOTE_DIR`).
- The tools, on prod (prints nothing when all are there; no go, no node —
  prod never builds):

```bash
for t in git oras jq curl psql pg_dump pg_restore certbot nginx fail2ban-client openssl; do command -v "$t" >/dev/null || echo "missing: $t"; done
id shrmpl-hook >/dev/null || echo "missing: the shrmpl-hook user"
```

**0.2 Local: the ids.** Two nackmon check-in codes (certs, backup) and the
promote hook's id, generated on prod and named into the repo first, so the
archive says what the host will have:

```bash
: "${PRD_SSH:?}"
N=infra/prd/shrmpl-etc/nack-monitors
H=infra/prd/shrmpl-etc/hooks
git mv "$N/prd-proto-renew-web-certs_NACKID.env" "$N/prd-proto-renew-web-certs_$(ssh "$PRD_SSH" cat /proc/sys/kernel/random/uuid).env"
git mv "$N/prd-proto-backup-all_NACKID.env" "$N/prd-proto-backup-all_$(ssh "$PRD_SSH" cat /proc/sys/kernel/random/uuid).env"
git mv "$H/prd-promote-proto_HOOKID.env" "$H/prd-promote-proto_$(ssh "$PRD_SSH" cat /proc/sys/kernel/random/uuid).env"
ls $N $H
git commit -m "prd-proto: nackmon check-in ids and the promote hook id"
git push
```

**0.3 Local: package the host files, ship them with the dump.** The
archive is cut from the pushed commit (`origin/main` in the template),
never the working tree; the `git log` line shows 0.2's commit:

```bash
: "${PRD_SSH:?}"
source etc/app.env
DUMP=$(ls -t dist/$APP_SLUG-data-*.dump | head -1)
echo "$DUMP"
git fetch origin
git log --oneline -1 origin/main
git archive --format=tar.gz -o dist/$APP_SLUG-infra-prd.tgz origin/main infra/prd infra/sql
scp "$DUMP" "$DUMP.sha256" dist/$APP_SLUG-infra-prd.tgz "$PRD_SSH:/tmp/"
```

## A. Install — nothing of ours is serving yet

One pass, in order, in **one root shell** on prod (`sudo -i`). If the
shell is lost, paste this block again and carry on: every later value is
read back from a file, and the env-file step only fills what still says
`SERVERONLY`. The helpers are `doc/330`'s.

```bash
ENV=prd; APP_ROOT=/srv/prd-proto; G=/root/proto-infra
I=$G/infra/prd; SQL=$G/infra/sql/proto-host-db.sql
getv()  { sed -n "s/^$2=//p" "$1" 2>/dev/null | head -1; }
urlpw() { sed -n 's#^DATABASE_URL=postgres://[^:]*:\([^@]*\)@.*#\1#p' "$1" | head -1; }
urldb() { sed -nE 's#^DATABASE_URL=postgres://[^/]+/([^?]+).*#\1#p' "$1" | head -1; }
idof()  { f=$(ls "$I/shrmpl-etc/$1/$2"_*.env 2>/dev/null | head -1); f=${f##*/"$2"_}; echo "${f%.env}"; }
fill()  { [ -n "$3" ] || { echo "[skip] $2: no value, fill it by hand"; return; }
  ( umask 077; K="$2" V="$3" awk 'index($0, ENVIRON["K"] "=") == 1 && /SERVERONLY/ { print ENVIRON["K"] "=" ENVIRON["V"]; next } { print }' "$1" > "$1.new" ) && cat "$1.new" > "$1" && rm "$1.new"; }
fillpw() { ( umask 077; R="$2" V="$3" awk '{ s = ENVIRON["R"] ":SERVERONLY@"; i = index($0, s) } i { $0 = substr($0, 1, i - 1) ENVIRON["R"] ":" ENVIRON["V"] "@" substr($0, i + length(s)) } { print }' "$1" > "$1.new" ) && cat "$1.new" > "$1" && rm "$1.new"; }
```

### A1. Unpack the host files

And check they are the commit you meant:

```bash
: "${G:?}"
mkdir -p "$G" && tar -xzf /tmp/proto-infra-prd.tgz -C "$G"
gzip -dc /tmp/proto-infra-prd.tgz | git get-tar-commit-id
idof nack-monitors prd-proto-renew-web-certs; idof nack-monitors prd-proto-backup-all; idof hooks prd-promote-proto
```

Expect 0.3's commit and three uuids (0.2), not `NACKID` / `HOOKID`.

### A2. The service user

```bash
id proto >/dev/null 2>&1 || useradd -r -s /usr/sbin/nologin -M -d /nonexistent proto
```

B074 (pending) renames it `svc-proto`.

### A3. Directories, scripts, env files

The app root is `root:shrmpl-hook 1775` (`doc/330` A4 says why). An env
file is copied only when the host has none yet, so a re-run keeps the
filled ones.

```bash
: "${APP_ROOT:?}" "${I:?}"
mkdir -p "$APP_ROOT"/{bin,etc,libexec,stg,backup,crt,log,web-pub}
chown root:shrmpl-hook "$APP_ROOT" && chmod 1775 "$APP_ROOT"
chown -R shrmpl-hook:shrmpl-hook "$APP_ROOT"/{bin,libexec,stg,backup,web-pub}
chown -R www-data:www-data "$APP_ROOT/log"
cp "$I"/bin/* "$APP_ROOT/bin/" && chmod 0755 "$APP_ROOT"/bin/* && chown shrmpl-hook:shrmpl-hook "$APP_ROOT"/bin/*
cp "$I/etc/prd-proto-api.service" "$I/etc/prd-proto-restart.path" "$I/etc/prd-proto-restart.service" "$APP_ROOT/etc/"
for f in "$I"/etc/prd-proto-*.env; do [ -e "$APP_ROOT/etc/${f##*/}" ] || cp "$f" "$APP_ROOT/etc/"; done
chown root:proto "$APP_ROOT/etc/prd-proto-api.env" && chmod 0640 "$APP_ROOT/etc/prd-proto-api.env"
for f in deploy migrate; do chown root:shrmpl-hook "$APP_ROOT/etc/prd-proto-$f.env" && chmod 0640 "$APP_ROOT/etc/prd-proto-$f.env"; done
chown root:root "$APP_ROOT/etc/prd-proto-backup.env" && chmod 0600 "$APP_ROOT/etc/prd-proto-backup.env"
stat -c '%U:%G %a' "$APP_ROOT"
```

Expect `root:shrmpl-hook 1775`.

### A4. Fill the env files

Prod's own secrets, generated here, never UAT's (A5 reads the passwords
back from the files):

```bash
: "${APP_ROOT:?}"
E=$APP_ROOT/etc
fillpw "$E/prd-proto-api.env" proto_app "$(openssl rand -hex 32)"
fillpw "$E/prd-proto-migrate.env" proto_ddl "$(openssl rand -hex 32)"
fill "$E/prd-proto-api.env" JWT_SECRET "$(openssl rand -hex 32)"
fill "$E/prd-proto-api.env" TOTP_ENCRYPTION_KEY "$(openssl rand -hex 32)"
echo "nackmon ids: certs $(idof nack-monitors prd-proto-renew-web-certs), backup $(idof nack-monitors prd-proto-backup-all)"
grep -n '^[^#].*SERVERONLY' "$E"/prd-proto-*.env
```

What the grep lists is filled by hand: in `prd-proto-deploy.env` the three
`CONTAINER_REGISTRY_*` (UAT's), `SLACK_WEBHOOK` (the whole URL),
`CERTBOT_EMAIL`, and `CERT_HEALTHCHECK_URL` (nackmon's check-in URL ending
`?code=` and the certs id); in `prd-proto-backup.env` `REMOTE_SERVER`,
`REMOTE_DIR` and `HEALTHCHECK_URL` (the backup id):

```bash
vi /srv/prd-proto/etc/prd-proto-deploy.env /srv/prd-proto/etc/prd-proto-backup.env
```

```bash
: "${APP_ROOT:?}"
grep -n '^[^#].*SERVERONLY' "$APP_ROOT"/etc/prd-proto-*.env || echo "all filled"
ssh -o BatchMode=yes "$(getv "$APP_ROOT/etc/prd-proto-backup.env" REMOTE_SERVER)" "ls -ld $(getv "$APP_ROOT/etc/prd-proto-backup.env" REMOTE_DIR)"
```

`all filled` is the only acceptable answer, and the backup destination
lists its directory. The new TOTP key is why A5 resets MFA: the local
enrolments were encrypted under the local key.

### A5. Nackmon monitors

On the monitoring host, before anything checks in (A8's certificate and
B2's backup both end by calling their check-in URL). If prod runs nackmon
itself, stage the files from the archive: `cp
"$I"/shrmpl-etc/nack-monitors/prd-proto-*.env /tmp/`. Otherwise, locally:

```bash
: "${MON_SSH:?}"
scp infra/prd/shrmpl-etc/nack-monitors/prd-proto-*.env "$MON_SSH:/tmp/"
```

On the monitoring host, as root (`REF` as in `doc/330` 0.3). The block
ends in `vi`: set `NACK_SLACK_WEBHOOK` in both files:

```bash
M=/srv/shrmpl/etc/nack-monitors
REF=$(ls "$M"/*.env | grep -v /prd-proto- | head -1); echo "REF=$REF"
for f in /tmp/prd-proto-*_*.env; do install -o "$(stat -c %U "$REF")" -g "$(stat -c %G "$REF")" -m "$(stat -c %a "$REF")" "$f" "$M/" && rm "$f"; done
ls "$M" | grep prd-proto-
vi "$M"/prd-proto-*.env
```

```bash
grep -c SERVERONLY /srv/shrmpl/etc/nack-monitors/prd-proto-*.env
systemctl restart shrmpl-nackmon && systemctl is-active shrmpl-nackmon
```

Expect `0` for each file, then `active`.

### A6. Roles, the database, the data

`doc/340-db-host-setup.md`, as on UAT; prod has no test env file, so the
CI password is empty and no CI role is made. The SQL goes to psql on
stdin, passwords as `\set` lines ahead of it:

```bash
: "${ENV:?}" "${APP_ROOT:?}" "${SQL:?}"
{ printf '\\set db %s\n'     "$(urldb "$APP_ROOT/etc/$ENV-proto-migrate.env")"
  printf '\\set ddl_pw %s\n' "$(urlpw "$APP_ROOT/etc/$ENV-proto-migrate.env")"
  printf '\\set app_pw %s\n' "$(urlpw "$APP_ROOT/etc/$ENV-proto-api.env")"
  printf '\\set ci_pw %s\n'  "$(getv "$APP_ROOT/etc/$ENV-proto-test.env" 'export PGPASSWORD')"
  cat "$SQL"
} | sudo -u postgres psql -X
```

Expect a last row `prd_proto_db | proto_ddl |
{=T/proto_ddl,proto_ddl=CTc/proto_ddl,proto_app=c/proto_ddl}`. Then the
dump, checked:

```bash
DUMP=$(ls -t /tmp/proto-data-*.dump | head -1); echo "DUMP=$DUMP"
(cd /tmp && sha256sum -c "${DUMP##*/}.sha256")
```

Expect `OK`. Restore it as `proto_ddl`, reset MFA, check the owner, and
the negative check:

```bash
: "${APP_ROOT:?}" "${DUMP:?}"
DB=$(urldb "$APP_ROOT/etc/prd-proto-migrate.env")
sudo -u postgres pg_restore --no-owner --no-privileges --role=proto_ddl --exit-on-error -d "$DB" < "$DUMP"
sudo -u postgres psql -X -d "$DB" -c "UPDATE app_user SET totp_seed = '', mfa_setup = FALSE WHERE mfa_setup"
sudo -u postgres psql -X -d "$DB" -c "SELECT tableowner, count(*) FROM pg_tables WHERE schemaname = 'public' GROUP BY 1"
PGPASSWORD=$(urlpw "$APP_ROOT/etc/prd-proto-api.env") psql -X -h 127.0.0.1 -U proto_app -d "$DB" -c 'SELECT count(*) FROM app_user' -c 'CREATE TABLE nope (i int)'
```

Expect one owner row `proto_ddl | <n>` (as on UAT), the user count, and
`ERROR:  permission denied for schema public`.

### A7. Units, then the first deploy

Do not poke the path unit by hand first (`doc/330` A6 says why).

```bash
: "${APP_ROOT:?}"
for u in prd-proto-api.service prd-proto-restart.path prd-proto-restart.service; do ln -sf "$APP_ROOT/etc/$u" "/etc/systemd/system/$u"; done
systemctl daemon-reload
systemctl enable --now prd-proto-restart.path
systemctl enable prd-proto-api
```

The first deploy promotes whatever `:uat` is now and installs it — the
command every later release runs. UAT's last build must have deployed with
its tests green:

```bash
sudo -u shrmpl-hook -H /srv/prd-proto/bin/prd-proto-promote-uat-and-deploy
```

Expect the promote (api and web revisions agree, retagged `:prd`), then
`verify: database matches the embedded migrations` and `migrate: nothing
pending` (or, with a migration newer than the dump, a
`backup/pre-migrate-*.dump` and `up`), `prd-proto-api healthy` and the
revision, the web publish, and Slack *PROD deployed*.

### A8. Certificates, then the nginx site

`nginx -t` before every reload: prod's nginx serves the other apps on the
host too, and a failed first issue through certbot's nginx plugin can
leave it down. No site names this host yet; certbot answers the challenge
from a temporary server block of its own. First a `--dry-run` against
Let's Encrypt's staging server, which issues nothing and counts against no
rate limit:

```bash
: "${APP_ROOT:?}"
SITE=$(getv "$APP_ROOT/etc/prd-proto-deploy.env" SITE_NAME); EMAIL=$(getv "$APP_ROOT/etc/prd-proto-deploy.env" CERTBOT_EMAIL); echo "$SITE $EMAIL"
nginx -t
certbot certonly --nginx -d "$SITE" --email "$EMAIL" --agree-tos -n --dry-run
nginx -t && systemctl is-active nginx
```

Stop on any failure. Expect `The dry run was successful.`, then `syntax is
ok` and `active`. Then the real certificate, through the renewal script
cron will run (it copies `fullchain.pem` root:www-data 0640 and
`privkey.pem` root:root 0600 into `crt/<site>/`, reloads nginx, and checks
in with nackmon):

```bash
: "${APP_ROOT:?}" "${SITE:?}"
"$APP_ROOT/bin/prd-proto-renew-web-certs"
ls -l "$APP_ROOT/crt/$SITE"
```

Then the site, from the repo's template with the real name. A failing
`nginx -t` takes the link back out, so nothing broken is ever loaded:

```bash
: "${APP_ROOT:?}" "${I:?}" "${SITE:?}"
SRC=$(ls "$I"/etc/nginx_* | head -1); REPO_SITE=${SRC##*/nginx_}
sed "s/${REPO_SITE//./\\.}/$SITE/g" "$SRC" > "$APP_ROOT/etc/nginx_$SITE"
ln -sf "$APP_ROOT/etc/nginx_$SITE" "/etc/nginx/sites-enabled/$SITE"
if nginx -t; then systemctl reload nginx && echo "reloaded"; else rm "/etc/nginx/sites-enabled/$SITE"; echo "nginx -t FAILED: link removed, nginx untouched"; fi
curl -fsS "https://$SITE/api/health"
curl -fsS -o /dev/null -w '%{http_code}\n' "https://$SITE"
```

Expect `reloaded`, the health JSON, and `200`.

### A9. First login

Open the site and sign in with a login from the local data; a user who had
MFA enrols again.

**Prod is live and is now the system of record.** The freeze ends; work
happens on prod.

## B. Wire it up

Same root shell (or paste A's variables block again).

### B1. Uptime monitors

On the monitoring host; only now, since before A8 both would page. As
`doc/330` B2 with `prd-proto-api` / `prd-proto-web`: stage the files in
`/tmp` (`cp "$I"/shrmpl-etc/pulsecheck-endpoints/prd-proto-*.env /tmp/` on
prod if it is the monitoring host, else `scp` them from local), then on
the monitoring host, with `SITE` set to prod's `SITE_NAME`:

```bash
: "${SITE:?}"
D=/srv/shrmpl/etc/pulsecheck-endpoints
REF=$(ls "$D"/*.env | grep -v /prd-proto- | head -1); echo "REF=$REF"
for f in prd-proto-api prd-proto-web; do install -o "$(stat -c %U "$REF")" -g "$(stat -c %G "$REF")" -m "$(stat -c %a "$REF")" "/tmp/$f.env" "$D/$f.env" && rm "/tmp/$f.env"; sed -i "s#^PULSE_URL=https://[^/]*#PULSE_URL=https://$SITE#" "$D/$f.env"; done
grep -h '^PULSE_URL=' "$D"/prd-proto-*.env
vi "$D/prd-proto-api.env" "$D/prd-proto-web.env"
```

```bash
grep -c SERVERONLY /srv/shrmpl/etc/pulsecheck-endpoints/prd-proto-*.env
systemctl restart shrmpl-pulsecheck && systemctl is-active shrmpl-pulsecheck
journalctl -u shrmpl-pulsecheck --since "-5 min" | grep prd-proto
```

A monitor changes in the repo first, then is copied in the same way
(`doc/330` B2).

### B2. Backup, once by hand

Then confirm what it wrote and shipped. The dump is listed by `pg_restore
--list` (as root: the backup directory is root-only 0700):

```bash
: "${APP_ROOT:?}"
B=$APP_ROOT/etc/prd-proto-backup.env
"$APP_ROOT/bin/prd-proto-backup-all"
ls -l "$(getv "$B" BACKUP_ROOT)/$(getv "$B" BACKUP_ENV)"
pg_restore --list "$(ls -t "$(getv "$B" BACKUP_ROOT)/$(getv "$B" BACKUP_ENV)"/*-PG-*.dump | head -1)" | head -20
ssh "$(getv "$B" REMOTE_SERVER)" "ls -l $(getv "$B" REMOTE_DIR)" | grep -i prd-proto-backup
```

Expect `Backup complete`; three files here and on `REMOTE_SERVER`:
`PRD-PROTO-BACKUP-PG-<host>-prd_proto_db-<Day>.dump`,
`...-PG-<host>-Globals-<Day>.sql.gz`, `...-FS-<host>-<Day>.tar.gz`; and a
listing that names the tables. The FS tarball carries `etc/`, the TOTP key
the database is useless without, so the destination holds prod secrets.

### B3. Cron (root)

Matching each monitor's `NACK_CRON` (`30 7 * * *` backup, `30 3 3 * *`
certs in the template; move them clear of the other apps' jobs on the
host, in the repo first). Appends only if absent:

```bash
mkdir -p /var/log/srv
if crontab -l 2>/dev/null | grep -q prd-proto-; then echo "prd-proto lines already in root's crontab"; else
  { crontab -l 2>/dev/null; cat <<'CRON'
30 7 * * * /srv/prd-proto/bin/prd-proto-backup-all > /var/log/srv/prd-proto-backup-all-`date +\%Y-\%m-\%dT\%H\%M\%S`.log 2>&1
30 3 3 * * /srv/prd-proto/bin/prd-proto-renew-web-certs > /var/log/srv/prd-proto-renew-web-certs-`date +\%Y-\%m-\%dT\%H\%M\%S`.log 2>&1
CRON
  } | crontab -
fi
crontab -l | grep prd-proto-
```

### B4. fail2ban

As `doc/330` B4, for the `[prd-proto]` jail (`migrate-to-srv-config` first
if `/etc/fail2ban/jail.local` is still a real file):

```bash
: "${I:?}" "${APP_ROOT:?}"
J=/srv/fail2ban/etc/jail.local
[ -e /srv/fail2ban/etc/nginx-404.conf ] || cp "$I/fail2ban/etc/nginx-404.conf" /srv/fail2ban/etc/
if grep -q '^\[prd-proto\]' "$J"; then echo "[prd-proto] already in $J"; else
  cp -p "$J" "$J.bak-$(date +%Y%m%d%H%M%S)"
  [ -z "$(tail -c1 "$J")" ] || echo >> "$J"
  sed -n '/^\[prd-proto\]/,$p' "$I/fail2ban/etc/jail.local" >> "$J"
fi
tail -12 "$J"
ls -l "$APP_ROOT/log/nginx_access.log"
fail2ban-client -t
fail2ban-regex "$APP_ROOT/log/nginx_access.log" /etc/fail2ban/filter.d/nginx-404.conf '"(GET|POST|HEAD) /api/'
fail2ban-client reload && fail2ban-client status prd-proto
```

If `-t` fails, copy the `.bak-*` file back, `fail2ban-client reload`, and
fix the stanza in the repo first.

### B5. Local

The local database is now a development copy that goes stale. **Never load
a local dump over prod**: this runbook only ever creates the database.
Prod is restored from its own backups.

### B6. The prod webhook, if you want one — last

`infra/prd/shrmpl-etc/hooks/prd-promote-proto_*.env` makes a push to one
branch run `prd-proto-promote-uat-and-deploy` here: no build, no checkout.
Which branch is your release model's choice, so `HOOK_ALLOWED_BRANCHES` is
`SERVERONLY` in the repo; set `RELEASE_BRANCH` in this shell to it. Know
what a push promotes: whatever `:uat` is **at that moment**, not the
commit pushed. If the release branch is also the branch UAT builds
(`BRANCH_TO_CHECKOUT`), every push promotes the build before it, while UAT
is still building this one. Without a hook, every release is the by-hand
command in **Every later release**.

```bash
: "${I:?}" "${RELEASE_BRANCH:?}"
H=/srv/shrmpl/etc/hooks
HOOKID=$(idof hooks prd-promote-proto); echo "HOOKID=$HOOKID"
SRC=$I/shrmpl-etc/hooks/prd-promote-proto_$HOOKID.env; DST=$H/prd-promote-proto_$HOOKID.env
REF=$(ls "$H"/*.env | grep -v /prd-promote-proto_ | head -1); echo "REF=$REF"
[ -e "$DST" ] || install -o "$(stat -c %U "$REF")" -g "$(stat -c %G "$REF")" -m "$(stat -c %a "$REF")" "$SRC" "$DST"
fill "$DST" HOOK_SECRET "$(openssl rand -hex 32)"
fill "$DST" HOOK_SLACK_WEBHOOK "$(getv "$REF" HOOK_SLACK_WEBHOOK)"
fill "$DST" HOOK_ALLOWED_BRANCHES "$RELEASE_BRANCH"
sed -i "s#/hook/HOOKID#/hook/$HOOKID#" "$DST"
echo "$(grep -c -e '^[^#].*SERVERONLY' -e HOOKID "$DST") placeholders left"
systemctl restart shrmpl-hook && systemctl is-active shrmpl-hook
echo "secret: $(getv "$DST" HOOK_SECRET)"
```

GitHub → the repository → Settings → Webhooks → Add (a second webhook
beside UAT's): payload URL with this `HOOKID` (the pattern in the other
hooks' `HOOK_URL` comments), content type `application/json`, the secret,
*just the push event*. GitHub's delivery log shows the ping answered; the
first real run is the next push to `RELEASE_BRANCH`.

## C. A day later

- The nightly backup ran and checked in (`ls /var/log/srv/ | grep
  prd-proto`, no nackmon alert).
- Clean up: the dump is customer data.

```bash
rm -f /tmp/proto-data-*.dump /tmp/proto-data-*.dump.sha256 /tmp/proto-infra-prd.tgz
```

Keep `/root/proto-infra` until the next infra change replaces it. Locally,
delete `dist/*-data-*.dump*` and `dist/*-infra-prd.tgz`.

## Every later release

A push to `BRANCH_TO_CHECKOUT` builds UAT, gated by the tests, and deploys
it there. Once UAT has that build and it has been looked at, promote it,
on prod:

```bash
sudo -u shrmpl-hook -H /srv/prd-proto/bin/prd-proto-promote-uat-and-deploy
```

That promotes what `:uat` is now; to promote a specific UAT build, append
its full commit as the one argument. A release that carries a migration
changes prod's schema, and `install-api` dumps the database to `backup/`
first. Rolling code back across a migration is a restore of that dump, not
a re-promotion: `proto-migrate verify` refuses an older binary on a newer
schema.

**A change under `infra/prd/`** (a `bin/` script, a unit, the nginx site)
does not ride the registry: prod has no checkout to pull it. Push it, then
ship and copy it the way 0.3 and A1/A3 did. Locally:

```bash
: "${PRD_SSH:?}"
source etc/app.env
git fetch origin
git archive --format=tar.gz -o dist/$APP_SLUG-infra-prd.tgz origin/main infra/prd infra/sql
scp dist/$APP_SLUG-infra-prd.tgz "$PRD_SSH:/tmp/"
```

On prod, as root:

```bash
G=/root/proto-infra; I=/root/proto-infra/infra/prd; APP_ROOT=/srv/prd-proto
gzip -dc /tmp/proto-infra-prd.tgz | git get-tar-commit-id
rm -rf "$G" && mkdir -p "$G" && tar -xzf /tmp/proto-infra-prd.tgz -C "$G" && rm /tmp/proto-infra-prd.tgz
cp "$I"/bin/* "$APP_ROOT/bin/" && chmod 0755 "$APP_ROOT"/bin/* && chown shrmpl-hook:shrmpl-hook "$APP_ROOT"/bin/*
for f in "$I"/etc/*; do case $f in *.env|*/nginx_*) continue;; esac; cmp -s "$f" "$APP_ROOT/etc/${f##*/}" || echo "differs: ${f##*/}"; done
SITE=$(sed -n 's/^SITE_NAME=//p' "$APP_ROOT/etc/prd-proto-deploy.env"); SRC=$(ls "$I"/etc/nginx_* | head -1); REPO_SITE=${SRC##*/nginx_}
sed "s/${REPO_SITE//./\\.}/$SITE/g" "$SRC" | cmp -s - "$APP_ROOT/etc/nginx_$SITE" || echo "differs: nginx_$SITE"
```

A unit listed as `differs` is copied over with `cp`, then
`systemctl daemon-reload`. The site is compared as rendered with the real
name; if it differs, render it again as in A8 and reload only if
`nginx -t` passes. Env files are never overwritten from the
archive: a new key is added to the host's file by hand (its value from the
repo template, or a secret generated there).

## Rollback (before users work on prod)

Until anyone but you has entered data on prod, local is still whole: undo
the install and lift the freeze. After that there is no going back to
local; prod is restored from its own backups. In a root shell with A's
variables:

```bash
: "${APP_ROOT:?}"
SITE=$(getv "$APP_ROOT/etc/prd-proto-deploy.env" SITE_NAME); DB=$(urldb "$APP_ROOT/etc/prd-proto-migrate.env")
systemctl disable --now prd-proto-api prd-proto-restart.path
rm -f /etc/systemd/system/prd-proto-api.service /etc/systemd/system/prd-proto-restart.path /etc/systemd/system/prd-proto-restart.service
systemctl daemon-reload
rm -f "/etc/nginx/sites-enabled/$SITE" && nginx -t && systemctl reload nginx
rm -f /srv/shrmpl/etc/hooks/prd-promote-proto_*.env && systemctl restart shrmpl-hook
crontab -l | grep -v prd-proto- | crontab -
sed -i '/^\[prd-proto\]/,$d' /srv/fail2ban/etc/jail.local && fail2ban-client -t && fail2ban-client reload
sudo -u postgres psql -X -v ON_ERROR_STOP=1 -c "ALTER DATABASE \"$DB\" RENAME TO \"${DB}_failed\""
mv "$APP_ROOT" /root/prd-proto-failed
```

Then remove the `prd-proto-*` monitors on the monitoring host (restart its
nackmon and pulsecheck) and the prod webhook on GitHub. Keep
`/root/prd-proto-failed` and the `_failed` database for the post-mortem;
drop both a week later.

## Lessons

What the first runs taught, carried here as steps above. Add yours.

- Prod has no checkout, so the host files arrive as a `git archive` of a
  pushed commit, never the working tree: what is installed is always a
  commit, named by `git get-tar-commit-id`.
- Prod loads the dump UAT loaded, the same day: one dump, one freeze, from
  local, and nothing read from UAT's database. `install-api`'s verify / up
  reconciles the schema with whatever build `:uat` is.
- The first certificate is a `--dry-run` first, with `nginx -t` before
  every reload: a failed first issue through certbot's nginx plugin has
  taken a shared nginx down. The site link comes back out if `nginx -t`
  fails.
- `sudo -u postgres psql -f /root/...` is `Permission denied`: the file is
  opened as `postgres`, and `/root` is 0700. The rest of that block ran
  on without the roles. Root pipes the file in; a block whose next step
  depends on the last is split so its output is read first.
- A `<placeholder>` in a host block is a bash syntax error (`<` and `>`
  are redirections); names come from the files themselves.
- Local blocks failed in zsh on an inline comment: `#` ran as a command
  and an apostrophe swallowed the lines after it. Local blocks carry no
  inline comments.
- Ids are generated before the archive is made and committed with it: the
  monitoring host may be the host being installed, and its monitor files
  ship with the rest.
- `fill` only replaces `SERVERONLY`, so a re-run never regenerates a secret
  already in use; passwords reach psql as `\set` lines on stdin, never
  `psql -v` (argv, visible in `ps`).
- A prod webhook promotes what `:uat` is when the push lands, not the
  commit pushed. On the branch UAT also builds, each push promotes the
  previous build.
- The nightly backup's tarball carries `etc/` (the TOTP key), so its
  directory is root-only 0700 and the remote destination holds prod
  secrets (B020).
