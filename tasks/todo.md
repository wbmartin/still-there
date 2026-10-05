# To do

The GRC backport (B001-B099) is done and merged to `main` (2026-10-05).
Its decision log, doc/700-grc-backport-feedback.md, was deleted at the
merge; read it with `git show e4d50fe:doc/700-grc-backport-feedback.md`.
Skipped: B002, B005, B015, B049, B050, B057, B067, B068, B079, B084,
B092. B092 is a live bug left as is: a wrong old password on "change my
password" answers 401, so the web signs the user out.

## 0. Waiting on staff

- **Dev DB:** run `bin/205-migrate-db` to apply 0004/0005 (B034, B035)
  before the next `bin/215-run-api-srv` or `bin/600-bundle-uat`; the API
  exits at start otherwise.
- **Before a real export (B071):** the export scrub refuses a live login
  named `admin`: rename or delete the seeded dev admin first. (The A9
  first-login link is now `proto-migrate invite`, B099.)
- **After GRC RG0044 (repo reset) runs:** update doc/350 (B082, marked
  "Not yet run").
- **Attribution note:** commits 8dd505f..388c0ae (B085-B090) say "Claude
  Opus 5.5" but were written by Sonnet; not rewritten (pushed).
- **Host steps** moved to Host work below (C007).
- **2026-10-03:** Staff agreed with every recommendation: B046 rename
  and B058 no-override confirmed; B054 one-way, B062 15 default, B066
  DENIED, B069 no zod bump kept; B063 StickyFilters removed; B093-B099
  done.
- **Follow-up:** the switcher context-switch pytest still mutates the seeded
  `switcher` user (B043).
- **Closed (C013): unknown file `radley-uat-builder-token`**, seen
  untracked in the repo root during the B033 run, then gone. Never
  committed. Its name matches the radley-grc tracker's ("Radley House")
  builder-agent token file, which GRC gitignored in d364daa/d202ac6, so
  that tooling most likely wrote it while running in this checkout.
  `.gitignore` now ignores `/*-token` at the root.

## Host work

The one list of hand steps on a host. ProtoStack has no hosts today:
H1-H5 apply only if a presales UAT deploy is ever done, H6-H12 only to
a post-sales host installed before the change named (a host installed
from doc/330 or doc/331 at the current `main` already has it).

Each entry: host and change, command, check, rollback. Agents add
entries and never run or tick them; only a staff member's report ticks one.
Commands run as root on the host named, `hostname` first; a variable set
in an earlier block is written `${VAR:?}` inside each command. Checks
print presence and lengths, never a secret value.

### Presales UAT (only if a UAT deploy is ever done)

Set first, as root on the UAT host: `E=/srv/uat-proto/etc/api-srv.env`.
Deploys never rewrite that file. The API ignores keys it does not know,
so an added key is safe ahead of the deploy that needs it.

- [ ] **H1. UAT, `api-srv.env`: add the four keys the API requires.**
  `LOG_LEVEL=info` (B031), `SERVER_BIND_ADDRESS=127.0.0.1` (B016, B061),
  `INVITE_TTL_HOURS=72` (B035), `ALLOW_MFA_EXEMPT_VIA_API=false` (B097).
  Without them the new API will not start, and `bin/610-deploy-uat`
  refuses a file missing the second or fourth. Before the deploy:
  ```bash
  hostname
  cp -p "${E:?}" /root/api-srv.env.H1.before
  for kv in LOG_LEVEL=info SERVER_BIND_ADDRESS=127.0.0.1 \
      INVITE_TTL_HOURS=72 ALLOW_MFA_EXEMPT_VIA_API=false; do
    grep -q "^${kv%%=*}=" "${E:?}" || echo "$kv" >> "${E:?}"; done
  ```
  Check (non-secret keys, so the lines print):
  `grep -E '^(LOG_LEVEL|SERVER_BIND_ADDRESS|INVITE_TTL_HOURS|ALLOW_MFA_EXEMPT_VIA_API)=' "${E:?}"`
  shows each key once; then the deploy's `/health` poll passes.
  Rollback: `cat /root/api-srv.env.H1.before > "${E:?}"` (keeps owner
  and mode), only if a value is wrong.
- [ ] **H2. UAT, `api-srv.env`: `JWT_SESSION_EXPIRE_HOURS=12`** (B016).
  ```bash
  hostname
  cp -p "${E:?}" /root/api-srv.env.H2.before
  sed 's/^JWT_SESSION_EXPIRE_HOURS=.*/JWT_SESSION_EXPIRE_HOURS=12/' "${E:?}" > /root/api-srv.env.H2.new && cat /root/api-srv.env.H2.new > "${E:?}" && rm /root/api-srv.env.H2.new
  systemctl restart uat-proto-api
  ```
  Check: `grep '^JWT_SESSION_EXPIRE_HOURS=' "${E:?}"` prints `=12`;
  `systemctl is-active uat-proto-api` prints `active`.
  Rollback: `cat /root/api-srv.env.H2.before > "${E:?}" && systemctl restart uat-proto-api`.
- [ ] **H3. UAT, `api-srv.env`: rotate the DB password off the repo
  literal** (B027). First `cp -p "${E:?}" /root/api-srv.env.H3.before`,
  then the block in doc/320-presales-uat-deploy.md, "A host set up before
  610 generated secrets" (rewrites `DATABASE_URL`, then
  `deploy.sh --no-db` sets the role to it).
  Check:
  ```bash
  hostname
  grep -c ':reallygoodPSQLpassword@' "${E:?}"    # 0
  awk '/^DATABASE_URL=/ { r = substr($0, index($0, "//") + 2)
    u = substr(r, 1, index(r, "@") - 1)
    print "db password length " length(substr(u, index(u, ":") + 1)) }' "${E:?}"    # 64
  ```
  and the deploy's `/health` poll passes. Rollback:
  `cat /root/api-srv.env.H3.before > "${E:?}"`, then `deploy.sh --no-db`
  again. The `.before` copy holds the old password: `rm` it once the
  check passes.
- [ ] **H4. Laptop then UAT: set `UAT_DOMAIN`** (B072). Until it is set,
  presales deploys install no nginx site. In the repo, set
  `UAT_DOMAIN=<real name>` in `etc/uat-deploy.env` and commit; the next
  deploy installs the site. Then on UAT, with `D` set to that name:
  ```bash
  hostname
  ls -l "/etc/nginx/sites-enabled/${D:?}"
  nginx -t
  sleep 2; curl -sI "http://${D:?}/" | head -1
  ```
  Check: the link exists, `nginx -t` ok, an HTTP status line; then run
  the `certbot --nginx -d` line the deploy printed. Rollback:
  `rm "/etc/nginx/sites-enabled/${D:?}" "/srv/uat-proto/etc/nginx-presales_${D:?}" && nginx -t && systemctl reload nginx`,
  and revert the `etc/uat-deploy.env` line.
- [ ] **H5. Every deployed host: grant 9020 (reset another user's MFA)
  to each real tenant's admin group** (B010). No migration seeds groups,
  so a deployed admin cannot reset MFA until it is added. In the app, as
  an admin: User Groups, the admin group, add `RESET_MFA_TOKEN`, save.
  Check: that group's admins see the MFA reset on a user's edit page.
  Rollback: remove the grant the same way.

### Post-sales hosts (when hosts exist)

UAT copies from its build checkout (`/build-stg/proto`, at the deployed
commit). Prod has no checkout: it takes `infra/prd/` through the
`git archive` block in doc/331-prd-postsales-install.md, "Every later
release", or one file at a time through infra/README.md, "Moving one
host file, hash-checked". Run in this order.

- [ ] **H6. UAT and prod: the `svc-proto` user** (B074). The API unit
  in H7 runs as it, so it comes first.
  ```bash
  hostname
  id svc-proto || useradd -r -s /usr/sbin/nologin -M svc-proto
  ```
  Check: `id svc-proto` prints a uid. Rollback: `userdel svc-proto`
  (only after H7's rollback).
- [ ] **H7. UAT and prod: recopy `bin/` and the API unit,
  `daemon-reload`** (B045, B046, B074 sandbox). On UAT:
  ```bash
  hostname
  C=/build-stg/proto/infra/uat; A=/srv/uat-proto
  tar -C "${A:?}" -czf /root/uat-proto-H7.before.tgz bin etc/uat-proto-api.service
  cp "${C:?}"/bin/* "${A:?}/bin/" && chmod 0755 "${A:?}"/bin/* && chown shrmpl-hook:shrmpl-hook "${A:?}"/bin/*
  cp "${C:?}/etc/uat-proto-api.service" "${A:?}/etc/"
  systemctl daemon-reload && systemctl restart uat-proto-api
  ```
  Prod: the doc/331 "Every later release" block, then the same unit copy
  and reload with `prd` names. Check: `diff -r "${C:?}/bin" "${A:?}/bin"`
  silent; `systemctl show -p User uat-proto-api` prints
  `User=svc-proto`; `sleep 2; systemctl is-active uat-proto-api`
  prints `active`; `systemd-analyze security uat-proto-api` scores
  about 1.7. Rollback:
  `tar -C "${A:?}" -xzf /root/uat-proto-H7.before.tgz && systemctl daemon-reload && systemctl restart uat-proto-api`.
- [ ] **H8. UAT build host: hook timeout, client tools, Go** (B047,
  B040). `HOOK_TIMEOUT=3600` in
  `/srv/shrmpl/etc/hooks/build-proto_<id>.env` (one queued build waits
  behind one running); `apt-get install postgresql-client` (the build's
  preflight needs `psql` and `pg_dump`).
  Check: `grep '^HOOK_TIMEOUT=' /srv/shrmpl/etc/hooks/build-proto_*.env`
  prints `3600`; `command -v psql pg_dump` prints two paths; the first
  build reaches the Go gate (it needs network once for staticcheck, and
  Go 1.26 where `GOTOOLCHAIN=local`). Rollback: the old `HOOK_TIMEOUT`
  value back; the package can stay.
- [ ] **H9. UAT and prod: the nginx site** (B048: keepalive upstream,
  plain port-80 redirect, IPv6 on 443). Render it with the real site name
  as doc/330 A8 (UAT) or doc/331 A8 (prod) does, after
  `cp -p` of the installed file to `/root/nginx_<site>.before`; reload
  only if `nginx -t` passes. Check, with `S` set to the site name:
  ```bash
  hostname
  nginx -t && systemctl reload nginx
  sleep 2
  curl -6 -sI "https://${S:?}/api/health" | head -1
  curl -sI "http://${S:?}/" | grep -i '^location: https://'
  ```
  An HTTP status over IPv6, and port 80 redirects. Rollback: copy the
  `.before` back, `nginx -t && systemctl reload nginx`.
- [ ] **H10. UAT build and prod promote, together, then one red UAT
  run** (B058: promote needs `sha-<commit>-passed`). Both scripts arrive
  with H7; run H7 on both hosts in the same window, since a new promote
  refuses builds an old UAT never tagged. Then push one deliberately
  failing commit to `BRANCH_TO_CHECKOUT`. Check: Slack *tests FAILED,
  don't promote*; `prd-proto-promote-uat-and-deploy <that sha>` refuses;
  the next green push is tagged `-passed` and promotes. Rollback: push
  a revert of the red commit.
- [ ] **H11. Monitoring host: UAT's monitors move off UAT** (B074). Copy
  `infra/prd/shrmpl-etc/nack-monitors/uat-proto-*` and
  `infra/prd/shrmpl-etc/pulsecheck-endpoints/uat-proto-*` to the prod
  host's `/srv/shrmpl/etc/` (codes renamed as infra/README.md, One-time
  host setup step 7, says), then remove the same files from UAT's `/srv/shrmpl/etc/`.
  Check: `ls /srv/shrmpl/etc/*/uat-proto-*` lists them on prod and
  nothing on UAT; stopping UAT's API alerts from prod. Rollback: move
  them back.
- [ ] **H12. UAT build host: the e2e harness env file passes the new
  config checks** (C003, a84ba5c). Since C003 the API refuses to start
  unless `JWT_SECRET` has 32+ characters and `TOTP_ENCRYPTION_KEY` is 64
  hex, and the e2e harness runs it from
  `/srv/uat-proto/etc/uat-proto-e2e-api.env`. The old repo copy's TOTP
  key was not hex. Check first:
  ```bash
  hostname
  awk '{ k = substr($0, 1, index($0, "=") - 1); v = substr($0, index($0, "=") + 1) }
    k == "JWT_SECRET" { print k " length " length(v) "  (need >= 32)" }
    k == "TOTP_ENCRYPTION_KEY" { print k " length " length(v) (v ~ /^[0-9a-fA-F]+$/ ? " hex" : " NOT HEX") "  (need 64 hex)" }
  ' /srv/uat-proto/etc/uat-proto-e2e-api.env
  ```
  If the TOTP line fails, take the repo's throwaway key (not a secret)
  from the checkout; the file keeps its own `DATABASE_URL`:
  ```bash
  hostname
  F=/srv/uat-proto/etc/uat-proto-e2e-api.env; R=/build-stg/proto/infra/uat/etc/uat-proto-e2e-api.env
  cp -p "${F:?}" /root/uat-proto-e2e-api.env.H12.before
  L=$(grep '^TOTP_ENCRYPTION_KEY=' "${R:?}")
  ( umask 077; L=${L:?} awk '/^TOTP_ENCRYPTION_KEY=/ { $0 = ENVIRON["L"] } { print }' "${F:?}" > "${F:?}.new" ) && cat "${F:?}.new" > "${F:?}" && rm "${F:?}.new"
  ```
  A short `JWT_SECRET` gets the same treatment from the repo's line.
  Check: the awk above prints `length 64 hex` and a length of 32 or
  more; the next build's e2e step passes. Rollback:
  `cat /root/uat-proto-e2e-api.env.H12.before > "${F:?}"` (with C003
  reverted, or the API will not start).
