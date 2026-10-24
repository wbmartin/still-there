# Deploy to prd-wbm — first install, every release, rollback

One host, `prd-wbm` (Debian 13), no UAT. It serves two names for this
app: `stillthere.teamtaggle.com` (the web app and its API) and
`webhook.teamtaggle.com` (only Telnyx's inbound-text webhook). A release
is built on the laptop from `main`, uploaded by scp, and installed by
the staff member with one sudo command: `bin/600-deploy-prd` on the
laptop, then `infra/prd-wbm/bin/prd-stillthere-install` on the host. The
host layout, the bundle and the ownership design are in
`infra/README.md`; the database roles in `doc/340-db-host-setup.md`.

**The host is shared with gsd-notes.** It already runs gsd's site, its
API on 8088, its Postgres database, certbot and (once gsd installs it)
fail2ban with its `jail.local`. Nothing below removes or edits any of
that: this app adds its own app root, user, group, database, API port
(8090), two nginx site files, two certificates and one fail2ban jail
file. The host was built by gsd-notes' doc/360.

Who runs what: sudo on the host needs the staff member's password, so
**every host block below is the staff member's**. An agent prepares,
uploads and checks, and never runs a host step.

## How the command blocks work

Each block is labelled with the machine it runs on and fenced by
`: '#===================================='` lines, which print nothing
but show in the terminal where the paste starts and stops. Every block
is one guarded compound command, so a paste into the wrong window runs
nothing:

- **PRD-WBM** blocks run in the root shell step 3 opens (`ssh -t prd-wbm
  sudo -i`). Each checks `hostname -s` is `prd-wbm` and the uid is 0, or
  prints `WRONG HOST OR NOT ROOT: nothing ran`.
- **LAPTOP** blocks run from the repo root of a clean checkout of
  `main`. Each checks it is not on `prd-wbm` and that
  `bin/600-deploy-prd` is in the current directory, or prints
  `WRONG HOST: nothing ran`.

No block has an inline `#` comment. Paths in `rm`, `mv`, `install`,
`ln`, `chmod` and redirects are spelled out. A block marked **do NOT
paste the output** prints a secret: keep its output out of chat, tickets
and notes.

## First install

The steps match `tasks/todo.md`, Host work. Each has its check; the
rollback is the step undone, written where it is more than the obvious
reverse.

### 1. DNS (staff, at the DNS provider; then laptop)

Add two A records pointing at prd-wbm's public IPv4 (the address
`gsd.diligentpursuits.com` already points at): `stillthere.teamtaggle.com`
and `webhook.teamtaggle.com`. No AAAA records: the host's firewall
rules for 80 and 443 are IPv4 only (gsd-notes doc/360). Then:

LAPTOP

```bash
: '#===================================='
if [ "$(hostname -s)" != prd-wbm ] && [ -x bin/600-deploy-prd ]; then
  dig +short A stillthere.teamtaggle.com
  dig +short A webhook.teamtaggle.com
  dig +short A gsd.diligentpursuits.com
else echo "WRONG HOST: nothing ran"; fi
: '#===================================='
```

Check: the three lines print the same address. A new record can take
minutes to an hour to show; certificates (step 12) need it. Rollback:
delete the two records.

### 2. The host files, from a commit (laptop)

From a clean checkout of `main`, never the working tree, so what lands
is in git. The archive goes with its SHA-256, checked on the host before
it is unpacked (step 5).

LAPTOP

```bash
: '#===================================='
if [ "$(hostname -s)" != prd-wbm ] && [ -x bin/600-deploy-prd ]; then
  git status --short
  git log -1 --oneline
  mkdir -p dist
  git archive --format=tar -o dist/stillthere-infra.tar HEAD infra/prd-wbm infra/sql
  (cd dist && shasum -a 256 stillthere-infra.tar > stillthere-infra.tar.sha256 && cat stillthere-infra.tar.sha256)
  scp dist/stillthere-infra.tar dist/stillthere-infra.tar.sha256 prd-wbm:/tmp/
else echo "WRONG HOST: nothing ran"; fi
: '#===================================='
```

Check: `git status` printed nothing, the log line is the commit to
install, and the scp finished silently. Note the hash it printed; step 5
prints it again on the host.

### 3. A root shell on the host

LAPTOP

```bash
: '#===================================='
if [ "$(hostname -s)" != prd-wbm ]; then ssh -t prd-wbm sudo -i; else echo "WRONG HOST: nothing ran"; fi
: '#===================================='
```

sudo asks for the staff member's password. Every PRD-WBM block below
runs in this shell. `exit` leaves it.

### 4. Preflight: what the shared host already has (read-only)

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  echo "--- port 8090 (must print nothing)"; ss -ltn | grep ':8090 '
  echo "--- app root, user, group (must not exist)"; ls -ld /srv/prd-stillthere; id svc-stillthere; getent group stillthere-deploy
  echo "--- database (must print 0)"; sudo -u postgres psql -XAtc "SELECT count(*) FROM pg_database WHERE datname = 'prd_stillthere_db'"
  echo "--- webroot"; ls -ld /var/www/letsencrypt
  echo "--- certbot authenticators"; grep -H '^authenticator' /etc/letsencrypt/renewal/*.conf
  echo "--- renewal hooks"; ls -l /etc/letsencrypt/renewal-hooks/pre /etc/letsencrypt/renewal-hooks/post /etc/letsencrypt/renewal-hooks/deploy
  echo "--- nginx sites"; ls -l /etc/nginx/sites-enabled/
  echo "--- fail2ban"; fail2ban-client ping
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check:

- Nothing on 8090; `/srv/prd-stillthere`, `svc-stillthere` and
  `stillthere-deploy` absent (`No such file`, `no such user`, nothing);
  the database count `0`; `/var/www/letsencrypt` exists.
- **Certificates.** gsd's certificate must already renew by webroot:
  `authenticator = webroot` for `gsd.diligentpursuits.com`, no `nginx`
  hook under `pre/` or `post/`, and a `deploy/nginx` hook that reloads
  nginx. Hooks in those directories run for every certificate on the
  host, so gsd's old standalone hooks (stop nginx before a renewal) would
  make this app's webroot renewals fail, and without the deploy hook a
  renewed certificate is not picked up. If it still says `standalone`,
  stop here and have gsd-notes doc/320, Certificates, "Optional: switch
  to webroot", done first.
- The sites list shows gsd's site. It must not change in any step below
  except by the files this runbook adds.
- `fail2ban-client ping` answers `pong` if fail2ban runs (step 18 needs
  it); an error means it is not installed yet, which blocks nothing else.

### 5. Unpack the host files, after checking them

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  cat /tmp/stillthere-infra.tar.sha256
  (cd /tmp && sha256sum -c stillthere-infra.tar.sha256) \
  && rm -rf /tmp/stillthere-infra \
  && mkdir -m 0700 /tmp/stillthere-infra \
  && tar -x -C /tmp/stillthere-infra -f /tmp/stillthere-infra.tar \
  && find /tmp/stillthere-infra -type f | sort
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: the hash is the one step 2 printed on the laptop,
`stillthere-infra.tar: OK`, then the listing: the two scripts under
`infra/prd-wbm/bin/`, twelve files under `infra/prd-wbm/etc/` (three
are the SMS relay's, installed by SMS relay below) and the two files
under `infra/sql/`. A failed check unpacks nothing:
run step 2 again.

### 6. The service user and the deploy group

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ] && [ -n "$SUDO_USER" ]; then
  id svc-stillthere > /dev/null 2>&1 || useradd --system --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin svc-stillthere
  getent group stillthere-deploy > /dev/null || groupadd --system stillthere-deploy
  usermod -aG stillthere-deploy "$SUDO_USER"
  id svc-stillthere
  getent group stillthere-deploy
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `svc-stillthere` exists with its own group; `stillthere-deploy`
lists the staff login. The login sees the new group from its next login
(step 14). Rollback: `gpasswd -d <staff login> stillthere-deploy;
groupdel stillthere-deploy; userdel svc-stillthere`.

### 7. Directories

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  install -d -o root -g root -m 0755 /srv/prd-stillthere /srv/prd-stillthere/bin /srv/prd-stillthere/etc /srv/prd-stillthere/libexec
  install -d -o root -g root -m 0700 /srv/prd-stillthere/backup /srv/prd-stillthere/stage
  install -d -o root -g stillthere-deploy -m 2770 /srv/prd-stillthere/incoming
  stat -c '%U:%G %a %n' /srv/prd-stillthere /srv/prd-stillthere/*
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `root:root 755` for the root, `bin`, `etc`, `libexec`;
`root:root 700` for `backup` and `stage`; `root:stillthere-deploy 2770
incoming`. The API writes no files, so there is no `files/`. Rollback:
`rm -rf /srv/prd-stillthere`.

### 8. Scripts and host files

First install only: this overwrites the env files, so on a running host
it would throw away the secrets step 9 fills (a later change to a host
file: Every later release, below).

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  I=/tmp/stillthere-infra/infra/prd-wbm
  install -o root -g root -m 0755 "$I/bin/prd-stillthere-install" "$I/bin/prd-stillthere-rollback" /srv/prd-stillthere/bin/
  install -o root -g root -m 0644 "$I/etc/prd-stillthere-deploy.env" "$I/etc/prd-stillthere-api.service" "$I/etc/nginx_stillthere.teamtaggle.com" "$I/etc/nginx_webhook.teamtaggle.com" "$I/etc/acme_stillthere-bootstrap.conf" "$I/etc/fail2ban-jail_prd-stillthere.conf" "$I/etc/fail2ban-filter_prd-stillthere-404.conf" /srv/prd-stillthere/etc/
  install -o root -g svc-stillthere -m 0640 "$I/etc/prd-stillthere-api.env" /srv/prd-stillthere/etc/
  install -o root -g root -m 0600 "$I/etc/prd-stillthere-migrate.env" /srv/prd-stillthere/etc/
  ls -l /srv/prd-stillthere/bin /srv/prd-stillthere/etc
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: the modes and owners above (`infra/README.md`, Layout on the
host). The API env file already carries the monitoring keys
`SMS_PROVIDER=log` and `WATCHDOG_TICK_SECONDS=30`: texts go to the log
until step 17.

### 9. Fill the secrets, on the host

Each `SERVERONLY` that is set (not commented) gets a fresh
`openssl rand -hex 32`, made here and written straight into the file:
the value goes through the environment of one `awk`, never a command
line, and never reaches the laptop or git. `umask 077` keeps the
temporary file private; `cat >` keeps each file's owner and mode. Run
again, it changes nothing (no `SERVERONLY` is left to replace). The
output holds no secret.

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  ( umask 077
    APP_PW=$(openssl rand -hex 32) JWT=$(openssl rand -hex 32) TOTP=$(openssl rand -hex 32) awk '
      /^DATABASE_URL=/ && !a { sub(/SERVERONLY/, ENVIRON["APP_PW"]); a = 1 }
      /^JWT_SECRET=/ && !j { sub(/SERVERONLY/, ENVIRON["JWT"]); j = 1 }
      /^TOTP_ENCRYPTION_KEY=/ && !t { sub(/SERVERONLY/, ENVIRON["TOTP"]); t = 1 }
      { print }' /srv/prd-stillthere/etc/prd-stillthere-api.env > /srv/prd-stillthere/etc/prd-stillthere-api.env.new \
    && cat /srv/prd-stillthere/etc/prd-stillthere-api.env.new > /srv/prd-stillthere/etc/prd-stillthere-api.env \
    && rm /srv/prd-stillthere/etc/prd-stillthere-api.env.new
    DDL_PW=$(openssl rand -hex 32) awk '
      /^DATABASE_URL=/ && !d { sub(/SERVERONLY/, ENVIRON["DDL_PW"]); d = 1 }
      { print }' /srv/prd-stillthere/etc/prd-stillthere-migrate.env > /srv/prd-stillthere/etc/prd-stillthere-migrate.env.new \
    && cat /srv/prd-stillthere/etc/prd-stillthere-migrate.env.new > /srv/prd-stillthere/etc/prd-stillthere-migrate.env \
    && rm /srv/prd-stillthere/etc/prd-stillthere-migrate.env.new )
  grep -n '^[^#]*SERVERONLY' /srv/prd-stillthere/etc/*.env; echo "unfilled: $?"
  awk -F= '/^(JWT_SECRET|TOTP_ENCRYPTION_KEY)=/ { print $1, length($2) }' /srv/prd-stillthere/etc/prd-stillthere-api.env
  ls -l /srv/prd-stillthere/etc/*.env
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `unfilled: 1` (grep found nothing), both secrets length 64, and
the env files still `root:svc-stillthere 0640` (api), `root:root 0600`
(migrate), `root:root 0644` (deploy). `TOTP_ENCRYPTION_KEY` encrypts
every MFA seed: it goes into every backup with the database, and a
database restored under another key loses every MFA enrolment.
Rollback: step 8 again, then this step.

### 10. Roles and the database

`doc/340-db-host-setup.md`, Run it: one PRD-WBM block. It reads the
passwords step 9 wrote and makes `stillthere_ddl`, `stillthere_app` and
`prd_stillthere_db`; `db_reader` and `db_writer` already exist (gsd made
them) and are kept. Check: the last row is `prd_stillthere_db |
stillthere_ddl` with `CONNECT` for `stillthere_ddl` and
`stillthere_app` only. Rollback (the database is still empty), as root:
`sudo -u postgres psql -X -c 'DROP DATABASE prd_stillthere_db' -c 'DROP ROLE stillthere_app' -c 'DROP ROLE stillthere_ddl'`
(never `db_reader` or `db_writer`: gsd's roles use them).

### 11. The unit

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  ln -s /srv/prd-stillthere/etc/prd-stillthere-api.service /etc/systemd/system/prd-stillthere-api.service
  systemctl daemon-reload
  systemctl enable prd-stillthere-api
  systemctl is-enabled prd-stillthere-api; systemctl is-active prd-stillthere-api
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `enabled`, then `inactive`. Do not start it: there is no binary
until the first install, which starts it. Rollback:
`systemctl disable prd-stillthere-api; rm /etc/systemd/system/prd-stillthere-api.service; systemctl daemon-reload`.

### 12. Certificates, by webroot

Needs step 1's records resolving and step 4's certificate check passed.
The two real sites name certificate files that do not exist yet, so
`nginx -t` would refuse them; a port-80-only bootstrap site
(`acme_stillthere-bootstrap.conf`) answers the HTTP-01 challenge for
both names from `/var/www/letsencrypt` while certbot issues one
certificate per name, then goes. nginx keeps running throughout, so gsd
never goes down (certbot's standalone mode would stop it).

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  ln -s /srv/prd-stillthere/etc/acme_stillthere-bootstrap.conf /etc/nginx/sites-enabled/stillthere-acme-bootstrap
  nginx -t && systemctl reload nginx \
  && certbot certonly --non-interactive --webroot -w /var/www/letsencrypt --cert-name stillthere.teamtaggle.com -d stillthere.teamtaggle.com \
  && certbot certonly --non-interactive --webroot -w /var/www/letsencrypt --cert-name webhook.teamtaggle.com -d webhook.teamtaggle.com
  rm /etc/nginx/sites-enabled/stillthere-acme-bootstrap
  nginx -t && systemctl reload nginx
  ls -l /etc/letsencrypt/live/
  grep -H '^authenticator' /etc/letsencrypt/renewal/stillthere.teamtaggle.com.conf /etc/letsencrypt/renewal/webhook.teamtaggle.com.conf
  certbot renew --dry-run --cert-name stillthere.teamtaggle.com
  certbot renew --dry-run --cert-name webhook.teamtaggle.com
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: both `certonly` runs say `Successfully received certificate`;
`/etc/letsencrypt/live/` lists both names beside gsd's; both renewal
files say `authenticator = webroot`; both dry runs succeed with nginx
running. certbot's timer renews them from now on, and the deploy hook
step 4 checked reloads nginx. A certbot failure still removes the
bootstrap site: fix the cause (usually DNS) and run the block again.
Rollback: `certbot delete --cert-name stillthere.teamtaggle.com` and the
same for `webhook.teamtaggle.com`.

### 13. The two nginx sites

Added beside gsd's site; `default` and gsd's file are not touched.

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  ln -s /srv/prd-stillthere/etc/nginx_stillthere.teamtaggle.com /etc/nginx/sites-enabled/stillthere.teamtaggle.com
  ln -s /srv/prd-stillthere/etc/nginx_webhook.teamtaggle.com /etc/nginx/sites-enabled/webhook.teamtaggle.com
  nginx -t && systemctl reload nginx
  sleep 2
  curl -sSI http://stillthere.teamtaggle.com/ | head -3
  curl -sS -o /dev/null -w 'site %{http_code} %{ssl_verify_result}\n' https://stillthere.teamtaggle.com/
  curl -sS -o /dev/null -w 'webhook / %{http_code}\n' https://webhook.teamtaggle.com/
  curl -sS -o /dev/null -w 'webhook GET %{http_code}\n' https://webhook.teamtaggle.com/telnyx-inbound
  curl -sS -o /dev/null -w 'webhook POST %{http_code}\n' -X POST -d '{}' https://webhook.teamtaggle.com/telnyx-inbound
  curl -sS -o /dev/null -w 'gsd %{http_code}\n' https://gsd.diligentpursuits.com/api/health
  nginx -T 2>/dev/null | grep -E 'client_max_body_size'
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `nginx -t` says `test is successful`; the http call answers
`301` to `https://stillthere.teamtaggle.com/`; `site 404 0` (no web
bundle yet, and a verified certificate: `0`); `webhook / 404`;
`webhook GET 405`; `webhook POST 502` (the SMS relay is not running yet);
`gsd 200`; among the body caps, `20M` (this site) and `64k` (the
webhook) beside gsd's own. If `nginx -t` fails, remove the two links
before anything else reloads nginx. Rollback:
`rm /etc/nginx/sites-enabled/stillthere.teamtaggle.com /etc/nginx/sites-enabled/webhook.teamtaggle.com; nginx -t && systemctl reload nginx`.

### 14. The first release (laptop, then the staff member)

The staff member logs out of every ssh session to the host (the root
shell too: `exit`) and back in, so the login carries `stillthere-deploy`.
Then, on the laptop, from a clean checkout of `main` with the gate green
on it:

LAPTOP

```bash
: '#===================================='
if [ "$(hostname -s)" != prd-wbm ] && [ -x bin/600-deploy-prd ]; then
  ssh prd-wbm id
  bin/300-run-tests all && bin/600-deploy-prd
else echo "WRONG HOST: nothing ran"; fi
: '#===================================='
```

`id` lists `stillthere-deploy`. The script ends with one line to run,
for example
`ssh -t prd-wbm sudo /srv/prd-stillthere/bin/prd-stillthere-install 20261024T201500Z-0123456789ab`.
The staff member pastes that line on the laptop. Expect `checksums
match`, `N migration(s) pending: dumping prd_stillthere_db to ...` (a
dump of the empty database), one `applied NNNN_...` line per migration,
three `swapped` lines, then `prd-stillthere-api healthy` and the
revision. On a first install there is no `.prev`, so an unhealthy start
stops with the `journalctl` line and no rollback; fix the cause (usually
an env key) and run the same line again. Then:

LAPTOP

```bash
: '#===================================='
if [ "$(hostname -s)" != prd-wbm ] && [ -x bin/600-deploy-prd ]; then
  bin/600-deploy-prd --check
else echo "WRONG HOST: nothing ran"; fi
: '#===================================='
```

Check: the revision, `prd-stillthere-api: active`, `"status":"ok"` and
`live commit is this checkout's HEAD`; it exits 0. Step 3 again for a
root shell.

### 15. First admin

The database has no login. `stillthere-migrate bootstrap-admin` makes
the tenant-1 Admin group with every grant the API checks and one invited
login in it, and prints its sign-in link. Before pasting, replace
`REPLACE_LOGIN` and `REPLACE_EMAIL` in the block with the login name
(letters, digits, `_` and `-`) and the email; the block refuses to run
with them unchanged.

PRD-WBM — **prints the sign-in link: do NOT paste the output**

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  LOGIN=REPLACE_LOGIN; EMAIL=REPLACE_EMAIL
  if [ "$LOGIN" = REPLACE_LOGIN ] || [ "$EMAIL" = REPLACE_EMAIL ]; then echo "EDIT LOGIN AND EMAIL FIRST: nothing ran"; else
    /srv/prd-stillthere/libexec/stillthere-migrate -config /srv/prd-stillthere/etc/prd-stillthere-migrate.env -base-url https://stillthere.teamtaggle.com -invite-ttl-hours 24 bootstrap-admin "$LOGIN" "$EMAIL"
  fi
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: one `[INFO] bootstrap-admin: ... in group Admin with every grant`
line and the link, `https://stillthere.teamtaggle.com/invite#<64 hex>`.
Open the link (it is the only copy of the token), set a password, enrol
TOTP, sign in. A lost or expired link: the same block with
`invite "$LOGIN"` in place of `bootstrap-admin "$LOGIN" "$EMAIL"` issues
a new one and revokes the old. Run `bootstrap-admin` again after a
success and it refuses (`already has 1 live login`). Everyone after the
first is invited from the web app's users page.

### 16. Smoke checks

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  curl -fsS https://stillthere.teamtaggle.com/api/health; echo
  systemctl show prd-stillthere-api -p User
  journalctl -u prd-stillthere-api --no-pager | grep WATCHDOG_MODE | tail -1
  systemd-analyze security prd-stillthere-api | tail -1
  ss -ltnp | grep -E ':(80|443|8088|8090|5432) '
  curl -fsS https://gsd.diligentpursuits.com/api/health; echo
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `{"data":{"status":"ok"}}`; `User=svc-stillthere`; the
`WATCHDOG_MODE` line says `SMS_PROVIDER=log`; the security score is in
the `OK` band or better; 8088 (gsd), 8090 and 5432 listen on loopback
only; gsd's health still `ok`. In the browser, signed in as the admin:
add a contact and a device.

### 17. Telnyx: live texts and the inbound webhook (optional)

Until this step the API writes texts to its log (`SMS_PROVIDER=log`).
Live texts need the Telnyx account, number and toll-free verification
(`tasks/todo.md`, Waiting on staff). Replies reach the API only through
the SMS relay (SMS relay, below): install it first or alongside, with
the same `TELNYX_PUBLIC_KEY`.

**In the Telnyx portal** (staff, browser):

1. On the messaging profile the number belongs to, set the inbound
   webhook URL to `https://webhook.teamtaggle.com/telnyx-inbound`
   (API v2 webhooks; no failover URL).
2. Collect the four values: an API key (`TELNYX_API_KEY`, secret), the
   number in E.164 (`TELNYX_FROM_NUMBER`, `+1...`), the messaging
   profile id (`TELNYX_MESSAGING_PROFILE_ID`) and the account's webhook
   public key (`TELNYX_PUBLIC_KEY`, Keys & Credentials, Public Key).

**On the host**, the four keys go only in
`/srv/prd-stillthere/etc/prd-stillthere-api.env`: never in the repo copy
(`infra/prd-wbm/etc/prd-stillthere-api.env` has them commented as
`SERVERONLY`), never in chat. This block opens the file in `vi` and has
no closing fence, because `vi` takes the terminal. In it: set
`SMS_PROVIDER=telnyx`, and uncomment the four `TELNYX_` lines with the
values typed in place of `SERVERONLY`.

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then cp -p /srv/prd-stillthere/etc/prd-stillthere-api.env /root/prd-stillthere-api.env.before-telnyx && vi /srv/prd-stillthere/etc/prd-stillthere-api.env; else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
```

Then restart and check; the output shows lengths, not values:

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  ls -l /srv/prd-stillthere/etc/prd-stillthere-api.env
  awk -F= '/^SMS_PROVIDER=/ { print } /^TELNYX_[A-Z_]+=/ { print $1, length($2) " chars" }' /srv/prd-stillthere/etc/prd-stillthere-api.env
  systemctl restart prd-stillthere-api
  sleep 3
  systemctl is-active prd-stillthere-api
  journalctl -u prd-stillthere-api --no-pager -n 100 | grep WATCHDOG_MODE | tail -1
  curl -sS -o /dev/null -w 'webhook POST %{http_code}\n' -X POST -d '{}' https://webhook.teamtaggle.com/telnyx-inbound
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: the file is still `root svc-stillthere` `-rw-r-----`;
`SMS_PROVIDER=telnyx` and four `TELNYX_` lines, none `10 chars`
(`SERVERONLY`); `active`; `WATCHDOG_MODE ... SMS_PROVIDER=telnyx`;
`webhook POST 401` (an unsigned call: the SMS relay refuses its signature, so
the route reaches it). Then text the number from a phone: the Telnyx
portal's message log shows the webhook delivered with `200`. An API
that will not start names the key in `journalctl -u prd-stillthere-api
-n 20`. Rollback:
`cp -p /root/prd-stillthere-api.env.before-telnyx /srv/prd-stillthere/etc/prd-stillthere-api.env; systemctl restart prd-stillthere-api`.
Afterwards delete `/root/prd-stillthere-api.env.before-telnyx` (it holds
the other secrets).

### 18. fail2ban: this app's jail (optional)

Needs fail2ban running (step 4: `pong`). Installing fail2ban, its
`jail.local` and the sshd jail is gsd-notes' step (its doc/320,
"Optional: fail2ban"), not this one. This adds one jail,
`prd-stillthere`, over both sites' access logs, with its own filter
(`/api/` and `POST /telnyx-inbound` ignored: their 4xx are the API
answering real callers).

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  if fail2ban-client ping > /dev/null 2>&1; then
    ln -s /srv/prd-stillthere/etc/fail2ban-filter_prd-stillthere-404.conf /etc/fail2ban/filter.d/prd-stillthere-404.conf
    ln -s /srv/prd-stillthere/etc/fail2ban-jail_prd-stillthere.conf /etc/fail2ban/jail.d/prd-stillthere.conf
    fail2ban-client -t && fail2ban-client reload
    sleep 2
    fail2ban-client status
    fail2ban-client status prd-stillthere
  else echo "fail2ban is not running: gsd-notes doc/320, Optional: fail2ban, first. Nothing ran"; fi
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `-t` reports the config OK; `status` lists `prd-stillthere`
beside gsd's jails; the jail's file list names both access logs.
Rollback:
`rm /etc/fail2ban/jail.d/prd-stillthere.conf /etc/fail2ban/filter.d/prd-stillthere-404.conf; fail2ban-client reload`.

### 19. Clean up

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  rm -rf /tmp/stillthere-infra
  rm -f /tmp/stillthere-infra.tar /tmp/stillthere-infra.tar.sha256
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

LAPTOP

```bash
: '#===================================='
if [ "$(hostname -s)" != prd-wbm ] && [ -x bin/600-deploy-prd ]; then
  rm -f dist/stillthere-infra.tar dist/stillthere-infra.tar.sha256
else echo "WRONG HOST: nothing ran"; fi
: '#===================================='
```

## SMS relay

The relay (`src/sms-relay/README.md`) takes Telnyx's inbound webhook for
every app on this host and holds each text in its own database until
the app reads it. Its own user `svc-smsrelay`, root `/srv/prd-smsrelay`,
database `prd_smsrelay_db` and loopback port 8095; the webhook site
(`nginx_webhook.teamtaggle.com`, still in this app's `etc/`) proxies
`/telnyx-inbound` and `/telnyx-inbound/<queue>` to it, and the API reads
queue `stillthere` with `SMS_RELAY_*`. Once, after First install. The
relay needs the account's webhook public key, so it goes in with or
after step 17. Telnyx's URL stays
`https://webhook.teamtaggle.com/telnyx-inbound` (queue `stillthere`).

Blocks as in How the command blocks work.

### R1. Build and upload (laptop)

From a clean checkout of `main`. The binary and the host files go with
one SHA-256 file, checked on the host before anything is unpacked.

LAPTOP

```bash
: '#===================================='
if [ "$(hostname -s)" != prd-wbm ] && [ -x bin/140-build-sms-relay ]; then
  git status --short
  git log -1 --oneline
  bin/140-build-sms-relay
  git archive --format=tar -o dist/smsrelay-infra.tar HEAD infra/prd-wbm infra/sql
  (cd dist && shasum -a 256 sms-relay smsrelay-infra.tar > smsrelay.sha256 && cat smsrelay.sha256)
  scp dist/sms-relay dist/smsrelay-infra.tar dist/smsrelay.sha256 prd-wbm:/tmp/
else echo "WRONG HOST: nothing ran"; fi
: '#===================================='
```

Check: `git status` printed nothing, the log line is the commit to
install, two hashes, and the scp finished silently.

### R2. A root shell on the host

LAPTOP

```bash
: '#===================================='
if [ "$(hostname -s)" != prd-wbm ]; then ssh -t prd-wbm sudo -i; else echo "WRONG HOST: nothing ran"; fi
: '#===================================='
```

Every PRD-WBM block below runs in this shell.

### R3. Check, unpack, preflight

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  cat /tmp/smsrelay.sha256
  (cd /tmp && sha256sum -c smsrelay.sha256) \
  && rm -rf /tmp/smsrelay-infra \
  && mkdir -m 0700 /tmp/smsrelay-infra \
  && tar -x -C /tmp/smsrelay-infra -f /tmp/smsrelay-infra.tar \
  && ls /tmp/smsrelay-infra/infra/prd-wbm/etc/prd-smsrelay.service /tmp/smsrelay-infra/infra/prd-wbm/etc/prd-smsrelay.env /tmp/smsrelay-infra/infra/prd-wbm/etc/prd-smsrelay-migrate.env /tmp/smsrelay-infra/infra/sql/smsrelay-host-db.sql
  echo "--- port 8095 (must print nothing)"; ss -ltn | grep ':8095 '
  echo "--- user and root (must not exist)"; id svc-smsrelay; ls -ld /srv/prd-smsrelay
  echo "--- database (must print 0)"; sudo -u postgres psql -XAtc "SELECT count(*) FROM pg_database WHERE datname = 'prd_smsrelay_db'"
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: the hashes are R1's, both `OK`, the four files listed; nothing on
8095, `no such user`, `No such file`, `0`. A failed hash unpacks
nothing: run R1 again.

### R4. User, directories, files

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  id svc-smsrelay > /dev/null 2>&1 || useradd --system --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin svc-smsrelay
  install -d -o root -g root -m 0755 /srv/prd-smsrelay /srv/prd-smsrelay/etc /srv/prd-smsrelay/libexec
  install -o root -g root -m 0755 /tmp/sms-relay /srv/prd-smsrelay/libexec/sms-relay
  install -o root -g root -m 0644 /tmp/smsrelay-infra/infra/prd-wbm/etc/prd-smsrelay.service /srv/prd-smsrelay/etc/prd-smsrelay.service
  install -o root -g svc-smsrelay -m 0640 /tmp/smsrelay-infra/infra/prd-wbm/etc/prd-smsrelay.env /srv/prd-smsrelay/etc/prd-smsrelay.env
  install -o root -g root -m 0600 /tmp/smsrelay-infra/infra/prd-wbm/etc/prd-smsrelay-migrate.env /srv/prd-smsrelay/etc/prd-smsrelay-migrate.env
  id svc-smsrelay
  ls -l /srv/prd-smsrelay/etc /srv/prd-smsrelay/libexec
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `svc-smsrelay` with its own group; `prd-smsrelay.env`
`root svc-smsrelay -rw-r-----`, `prd-smsrelay-migrate.env` `root root
-rw-------`, the unit `-rw-r--r--`, `sms-relay` `-rwxr-xr-x`. First
install only: it overwrites the env files R5 fills.

### R5. Secrets and the consumer token

Two database passwords and the token Still There reads its queue with,
each `openssl rand -hex 32` made here and passed through the environment
of one `awk`, never a command line. The relay's file gets only the
token's SHA-256; the token itself goes into the API's file as
`SMS_RELAY_TOKEN`, with `SMS_RELAY_URL` and `SMS_RELAY_QUEUE` (any
earlier `SMS_RELAY_` lines there are replaced). The API's file is copied
to `/root` first. A second run changes nothing. The output holds no
secret.

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  ( umask 077
    APP_PW=$(openssl rand -hex 32) awk '/^DATABASE_URL=/ && !a { sub(/SERVERONLY/, ENVIRON["APP_PW"]); a = 1 } { print }' /srv/prd-smsrelay/etc/prd-smsrelay.env > /srv/prd-smsrelay/etc/prd-smsrelay.env.new \
    && cat /srv/prd-smsrelay/etc/prd-smsrelay.env.new > /srv/prd-smsrelay/etc/prd-smsrelay.env \
    && rm /srv/prd-smsrelay/etc/prd-smsrelay.env.new
    DDL_PW=$(openssl rand -hex 32) awk '/^DATABASE_URL=/ && !d { sub(/SERVERONLY/, ENVIRON["DDL_PW"]); d = 1 } { print }' /srv/prd-smsrelay/etc/prd-smsrelay-migrate.env > /srv/prd-smsrelay/etc/prd-smsrelay-migrate.env.new \
    && cat /srv/prd-smsrelay/etc/prd-smsrelay-migrate.env.new > /srv/prd-smsrelay/etc/prd-smsrelay-migrate.env \
    && rm /srv/prd-smsrelay/etc/prd-smsrelay-migrate.env.new
    if grep -q '^CONSUMER_TOKEN_LIST=stillthere:SERVERONLY$' /srv/prd-smsrelay/etc/prd-smsrelay.env; then
      cp -p /srv/prd-stillthere/etc/prd-stillthere-api.env /root/prd-stillthere-api.env.before-smsrelay
      TOKEN=$(openssl rand -hex 32)
      HASH=$(printf %s "$TOKEN" | sha256sum | cut -d' ' -f1)
      HASH="$HASH" awk '/^CONSUMER_TOKEN_LIST=stillthere:SERVERONLY$/ { sub(/SERVERONLY/, ENVIRON["HASH"]) } { print }' /srv/prd-smsrelay/etc/prd-smsrelay.env > /srv/prd-smsrelay/etc/prd-smsrelay.env.new \
      && cat /srv/prd-smsrelay/etc/prd-smsrelay.env.new > /srv/prd-smsrelay/etc/prd-smsrelay.env \
      && rm /srv/prd-smsrelay/etc/prd-smsrelay.env.new
      TOKEN="$TOKEN" awk '!/^#? ?SMS_RELAY_(URL|TOKEN|QUEUE)=/ { print } END { print "SMS_RELAY_URL=http://127.0.0.1:8095"; print "SMS_RELAY_TOKEN=" ENVIRON["TOKEN"]; print "SMS_RELAY_QUEUE=stillthere" }' /srv/prd-stillthere/etc/prd-stillthere-api.env > /srv/prd-stillthere/etc/prd-stillthere-api.env.new \
      && cat /srv/prd-stillthere/etc/prd-stillthere-api.env.new > /srv/prd-stillthere/etc/prd-stillthere-api.env \
      && rm /srv/prd-stillthere/etc/prd-stillthere-api.env.new
    fi )
  grep -n '^[^#]*SERVERONLY' /srv/prd-smsrelay/etc/*.env
  awk -F= '/^(CONSUMER_TOKEN_LIST|SMS_RELAY_[A-Z]+)=/ { print FILENAME, $1, length($2) }' /srv/prd-smsrelay/etc/prd-smsrelay.env /srv/prd-stillthere/etc/prd-stillthere-api.env
  ls -l /srv/prd-smsrelay/etc/*.env /srv/prd-stillthere/etc/prd-stillthere-api.env
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: the only `SERVERONLY` left is `TELNYX_PUBLIC_KEY` (R6);
`CONSUMER_TOKEN_LIST 75`, `SMS_RELAY_URL 21`, `SMS_RELAY_TOKEN 64`,
`SMS_RELAY_QUEUE 10`; the modes as in R4, and the API's file still
`root svc-stillthere -rw-r-----`. Rollback: R4 again, then this step;
for the API's file,
`cp -p /root/prd-stillthere-api.env.before-smsrelay /srv/prd-stillthere/etc/prd-stillthere-api.env`.

### R6. The Telnyx public key

The relay checks every webhook with the account's public key and will
not start without one. Copied from the API's file when step 17 put it
there:

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  K=$(sed -n 's/^TELNYX_PUBLIC_KEY=//p' /srv/prd-stillthere/etc/prd-stillthere-api.env | head -1)
  if [ -n "$K" ] && [ "$K" != SERVERONLY ]; then
    ( umask 077
      K="$K" awk '/^TELNYX_PUBLIC_KEY=SERVERONLY$/ { $0 = "TELNYX_PUBLIC_KEY=" ENVIRON["K"] } { print }' /srv/prd-smsrelay/etc/prd-smsrelay.env > /srv/prd-smsrelay/etc/prd-smsrelay.env.new \
      && cat /srv/prd-smsrelay/etc/prd-smsrelay.env.new > /srv/prd-smsrelay/etc/prd-smsrelay.env \
      && rm /srv/prd-smsrelay/etc/prd-smsrelay.env.new )
  else echo "no TELNYX_PUBLIC_KEY in the API file: set it by hand (next block)"; fi
  awk '/^TELNYX_PUBLIC_KEY=/ { print "TELNYX_PUBLIC_KEY", length($0) - 18 " chars" }' /srv/prd-smsrelay/etc/prd-smsrelay.env
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `TELNYX_PUBLIC_KEY 44 chars`. Only if it printed `no
TELNYX_PUBLIC_KEY` (or `10 chars`): this block opens the file in `vi`
and has no closing fence, because `vi` takes the terminal; type the
portal's key (Keys & Credentials, Public Key) in place of `SERVERONLY`.

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then vi /srv/prd-smsrelay/etc/prd-smsrelay.env; else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
```

### R7. Roles, the database, the schema

As doc/340, Run it, for the relay's file: `smsrelay_ddl` owns the
database, `smsrelay_app` (the relay) has DML only; neither is in
`db_reader` or `db_writer`. Then the schema, as `smsrelay_ddl`.

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  urlpw() { sed -n 's#^DATABASE_URL=postgres://[^:]*:\([^@]*\)@.*#\1#p' "$1" | head -1; }
  urldb() { sed -nE 's#^DATABASE_URL=postgres://[^/]+/([^?]+).*#\1#p' "$1" | head -1; }
  { printf '\\set db %s\n'     "$(urldb /srv/prd-smsrelay/etc/prd-smsrelay-migrate.env)"
    printf '\\set ddl_pw %s\n' "$(urlpw /srv/prd-smsrelay/etc/prd-smsrelay-migrate.env)"
    printf '\\set app_pw %s\n' "$(urlpw /srv/prd-smsrelay/etc/prd-smsrelay.env)"
    cat /tmp/smsrelay-infra/infra/sql/smsrelay-host-db.sql
  } | sudo -u postgres psql -X \
  && /srv/prd-smsrelay/libexec/sms-relay migrate /srv/prd-smsrelay/etc/prd-smsrelay-migrate.env
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: the last row `prd_smsrelay_db | smsrelay_ddl` with `CONNECT`
(`c`) for `smsrelay_ddl` and `smsrelay_app` only, then `migrate: 1
applied; schema current`. Rollback:
`sudo -u postgres psql -X -c 'DROP DATABASE prd_smsrelay_db' -c 'DROP ROLE smsrelay_app' -c 'DROP ROLE smsrelay_ddl'`.

### R8. The unit

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  ln -s /srv/prd-smsrelay/etc/prd-smsrelay.service /etc/systemd/system/prd-smsrelay.service
  systemctl daemon-reload
  systemctl enable --now prd-smsrelay
  sleep 3
  systemctl is-active prd-smsrelay
  curl -sS http://127.0.0.1:8095/health; echo
  journalctl -u prd-smsrelay --no-pager -n 5
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `active`, `{"status":"ok"}`, a `listening` line with
`127.0.0.1:8095` and `queueTokenCount 1`. A relay that will not start
names the key or the pending migration in the journal. Rollback:
`systemctl disable --now prd-smsrelay; rm /etc/systemd/system/prd-smsrelay.service; systemctl daemon-reload`.

### R9. The webhook site points at the relay

The site file is this app's (`/srv/prd-stillthere/etc/`); the old copy
goes to `/root` first and comes back if `nginx -t` fails.

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  cp -p /srv/prd-stillthere/etc/nginx_webhook.teamtaggle.com /root/nginx_webhook.teamtaggle.com.before-smsrelay
  install -o root -g root -m 0644 /tmp/smsrelay-infra/infra/prd-wbm/etc/nginx_webhook.teamtaggle.com /srv/prd-stillthere/etc/nginx_webhook.teamtaggle.com
  if nginx -t; then systemctl reload nginx; else cp -p /root/nginx_webhook.teamtaggle.com.before-smsrelay /srv/prd-stillthere/etc/nginx_webhook.teamtaggle.com; echo "nginx -t failed: old site restored, nothing reloaded"; fi
  sleep 2
  curl -sS -o /dev/null -w 'GET %{http_code}\n' https://webhook.teamtaggle.com/telnyx-inbound
  curl -sS -o /dev/null -w 'POST bare %{http_code}\n' -X POST -d '{}' https://webhook.teamtaggle.com/telnyx-inbound
  curl -sS -o /dev/null -w 'POST queue %{http_code}\n' -X POST -d '{}' https://webhook.teamtaggle.com/telnyx-inbound/stillthere
  curl -sS -o /dev/null -w 'POST bad queue %{http_code}\n' -X POST -d '{}' https://webhook.teamtaggle.com/telnyx-inbound/Bad_Name
  curl -sS -o /dev/null -w 'queue API %{http_code}\n' https://webhook.teamtaggle.com/queue/stillthere/peek
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `test is successful`; `GET 405`, `POST bare 401`, `POST queue
401` (unsigned: the relay refuses them, so both routes reach it),
`POST bad queue 404` and `queue API 404` (nginx, never the relay).
Rollback:
`cp -p /root/nginx_webhook.teamtaggle.com.before-smsrelay /srv/prd-stillthere/etc/nginx_webhook.teamtaggle.com; nginx -t && systemctl reload nginx`
(texts then go to the API's `/smsInbound` again).

### R10. Still There reads its queue

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  systemctl restart prd-stillthere-api
  sleep 3
  systemctl is-active prd-stillthere-api
  journalctl -u prd-stillthere-api --no-pager -n 100 | grep -E 'SMS_RELAY_(MODE|RECEIVE)' | tail -3
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Check: `active`; `SMS_RELAY_MODE ... queue stillthere` and no
`SMS_RELAY_RECEIVE` error. Then text `HELP` to the number from a phone:
within `WATCHDOG_TICK_SECONDS` the reply arrives, and
`journalctl -u prd-stillthere-api -n 50` shows `SMS_RELAY_DONE`.
Rollback: R5's copy of the API's file back, then
`systemctl restart prd-stillthere-api`.

### R11. Clean up

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  rm -rf /tmp/smsrelay-infra
  rm -f /tmp/sms-relay /tmp/smsrelay-infra.tar /tmp/smsrelay.sha256
  rm -f /root/prd-stillthere-api.env.before-smsrelay /root/nginx_webhook.teamtaggle.com.before-smsrelay
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

LAPTOP

```bash
: '#===================================='
if [ "$(hostname -s)" != prd-wbm ] && [ -x bin/140-build-sms-relay ]; then
  rm -f dist/smsrelay-infra.tar dist/smsrelay.sha256
else echo "WRONG HOST: nothing ran"; fi
: '#===================================='
```

`/root/prd-stillthere-api.env.before-smsrelay` holds the API's secrets:
delete it once R10 checks out. A later relay release: R1, R3, then
`install` the binary as in R4, `sms-relay migrate` as in R7 and
`systemctl restart prd-smsrelay`; env files are never rewritten.

## Every later release

From a clean checkout of `main` (it reaches `main` only by the PR from
`dev`, after the full gate: `CLAUDE.md`, Branches):

LAPTOP

```bash
: '#===================================='
if [ "$(hostname -s)" != prd-wbm ] && [ -x bin/600-deploy-prd ]; then
  git status --short
  git log -1 --oneline
  bin/300-run-tests all && bin/600-deploy-prd
else echo "WRONG HOST: nothing ran"; fi
: '#===================================='
```

The staff member pastes the printed
`ssh -t prd-wbm sudo /srv/prd-stillthere/bin/prd-stillthere-install <bundle>`
line, then runs the `--check` block from step 14. What the install
does, in order: checks the bundle in a root-only copy (`sha256sum -c`);
runs the incoming `stillthere-migrate verify` (a refusal stops it before
anything live changes); dumps the database to
`backup/pre-migrate-<bundle>.dump` when a migration is pending, then
`up`; swaps `api-srv`, `stillthere-migrate`, `web-pub/` and the revision
file, the live ones kept as `.prev`; restarts the unit; waits up to
`HEALTH_WAIT_SECONDS` for `/health`. Not healthy: it runs
`prd-stillthere-rollback` itself and exits 1. Running the same line
again for a bundle that is live changes nothing; for one interrupted
part-way (`libexec/install.pending`) it finishes the swap without
rotating `.prev` twice.

- **Migrations are forward-only.** Before shipping one, read it
  (`git diff --stat <live commit>..HEAD -- src/api-srv/sql/migrations`,
  the live commit from `--check`) and decide whether the previous code
  still runs on the new schema; if not, a rollback is the dump restore
  below.
- **A new required env key** ships with the release that needs it: add it
  to `/srv/prd-stillthere/etc/prd-stillthere-api.env` by hand first (a
  deploy never rewrites an env file), or the new API refuses to start
  and the install rolls back. Record it in `tasks/todo.md`, Host work.
- **A changed host file** (a script, the unit, a site, the deploy env,
  the jail): steps 2, 3 and 5 for the commit, then for that one file a
  copy of the live one to `/root/`, a `diff` against the new one, its
  `install` line from step 8 (never the env-file lines), and the reload:
  `systemctl daemon-reload` for the unit, `nginx -t && systemctl reload
  nginx` for a site, `fail2ban-client reload` for the jail. Back out with
  the copy. Then step 19.
- **Old bundles** stay in `incoming/` (the staff login can delete its own
  uploads without sudo); pre-migrate dumps stay in `backup/` until
  removed by hand.

## Rollback

LAPTOP

```bash
: '#===================================='
if [ "$(hostname -s)" != prd-wbm ] && [ -x bin/600-deploy-prd ]; then
  ssh -t prd-wbm sudo /srv/prd-stillthere/bin/prd-stillthere-rollback
  bin/600-deploy-prd --check
else echo "WRONG HOST: nothing ran"; fi
: '#===================================='
```

It swaps the live set and `.prev` (binaries, web bundle, revision),
restarts and checks health; a second run swaps back. It never touches
the schema. It runs the now-live (older) `stillthere-migrate verify` and
warns when the database is ahead of that code: the older binary is then
running on a schema it was not built for. If it is unhealthy there, the
way back is the pre-migrate dump of the release that migrated, which
loses every write since that release went live. In a root shell (step
3), list the dumps first:

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  ls -lt /srv/prd-stillthere/backup/
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Then restore the one taken before the release that migrated: replace
`REPLACE_DUMP` with its file name (`pre-migrate-<bundle>.dump`) before
pasting; the block refuses to run with it unchanged or missing.

PRD-WBM

```bash
: '#===================================='
if [ "$(hostname -s)" = prd-wbm ] && [ "$(id -u)" = 0 ]; then
  DUMP_NAME=REPLACE_DUMP
  if [ "$DUMP_NAME" = REPLACE_DUMP ] || [ ! -f "/srv/prd-stillthere/backup/$DUMP_NAME" ]; then echo "SET DUMP_NAME TO A FILE IN backup/ FIRST: nothing ran"; else
    systemctl stop prd-stillthere-api
    sudo -u postgres pg_restore --clean --if-exists --no-owner --role=stillthere_ddl --single-transaction --exit-on-error -d prd_stillthere_db < "/srv/prd-stillthere/backup/$DUMP_NAME"
    systemctl start prd-stillthere-api
    sleep 3
    systemctl is-active prd-stillthere-api
  fi
else echo "WRONG HOST OR NOT ROOT: nothing ran"; fi
: '#===================================='
```

Then the `--check` block from step 14. gsd-notes rehearsed this restore
(twice over the same database, the `--clean` case) on prd-wbm on
2026-10-17; this app's restore has not been rehearsed yet.

## Certificates

One certificate per name (`stillthere.teamtaggle.com`,
`webhook.teamtaggle.com`), both issued and renewed by webroot from
`/var/www/letsencrypt`: each site's port-80 block serves
`/.well-known/acme-challenge/` from there and redirects everything else
to https. certbot's timer renews them with gsd's; the host-wide deploy
hook reloads nginx after a renewal. Step 4 is the check that the host is
set up for this; `certbot renew --dry-run` repeats it at any time.

## Backups

Backups: not set up yet (gsd-notes doc/380 is the pattern). Until then
the only copies are the pre-migrate dumps in
`/srv/prd-stillthere/backup/` on the host itself.
