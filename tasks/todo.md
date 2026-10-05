# To do

The GRC backport (B001-B099) is done and merged to `main` (2026-10-05).
Its decision log, doc/700-grc-backport-feedback.md, was deleted at the
merge; read it with `git show e4d50fe:doc/700-grc-backport-feedback.md`.
Skipped: B002, B005, B015, B049, B050, B057, B067, B068, B079, B084,
B092. B092 is a live bug left as is: a wrong old password on "change my
password" answers 401, so the web signs the user out.

## 0. Waiting on Brandon

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
- **No UAT deploys planned for ProtoStack (template only); apply these
  only if a UAT deploy is ever done:**
  - Existing UAT host `api-srv.env` hand edits (deploys never overwrite
    that file): add `LOG_LEVEL=info` (B031: required, the API will not
    start without it); add `SERVER_BIND_ADDRESS=127.0.0.1` (B016; required
    since B061, the API and `bin/610-deploy-uat` refuse a file without
    it); add `INVITE_TTL_HOURS=72` (B035, required key); add
    `ALLOW_MFA_EXEMPT_VIA_API=false` (B097: required, the API and
    `bin/610-deploy-uat` refuse a file without it); set
    `JWT_SESSION_EXPIRE_HOURS=12` (B016); rotate the DB password off the
    repo literal (B027, step in `doc/320-presales-uat-deploy.md`).
  - `etc/uat-deploy.env`: set `UAT_DOMAIN` (B072). Until set, presales
    deploys install no nginx site.
- **Grant 9020 (reset another user's MFA) by hand** on every real tenant's
  admin group (B010). No migration seeds groups, so a deployed admin loses
  MFA reset until it is added.
- **2026-10-03:** Brandon agreed with every recommendation: B046 rename
  and B058 no-override confirmed; B054 one-way, B062 15 default, B066
  DENIED, B069 no zod bump kept; B063 StickyFilters removed; B093-B099
  done.
- **Post-sales host steps, when hosts exist:** recopy infra `bin/` and
  `*-proto-api.service`, `daemon-reload` (B045/B046); `HOOK_TIMEOUT=3600`
  and postgresql-client on the UAT build host (B047); recopy the nginx
  site, `nginx -t`, reload, check `curl -6` (B048); recopy UAT build and
  prd promote together, then one deliberately red UAT run (B058); B074:
  `svc-proto` user, sandboxed unit, UAT monitors moved to prd shrmpl-etc. The first
  Go gate run needs network for staticcheck; Go 1.26 where
  `GOTOOLCHAIN=local` (B040).
- **Follow-up:** the switcher context-switch pytest still mutates the seeded
  `switcher` user (B043).
- **Unknown file `radley-uat-builder-token`** appeared untracked in the repo
  root during the B033 test run and then vanished. Never committed
  (`git log --all -- radley-uat-builder-token` is empty). Check what wrote it.
