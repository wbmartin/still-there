# To do

Branch `backport-grc` (off `main` at `c7da74d`). The work list is
`doc/700-grc-backport-feedback.md`: every item has a decision, an expanded
write-up (what is wrong, what GRC changed, the fix, the open question) and,
where Brandon answered, a `Brandon: ...` line that overrides the
recommendation. Done items sit under `# Completed` at the end of that file.

## 0. Waiting on Brandon

- **Decide B096-B098** (pull/adapt/skip in doc/700, under "Bugs found in
  PROTO along the way"). B096: grant id numbering convention (new ids only; shipped ids pinned by
  `testdata/grants.json`). B097: `mfa_exempt` accepted by the API from any
  user admin (refuse it, or report exempt logins on a host). B098: `example`
  has no cross-tenant repository test (tenant 2 vs 3).
- **Dev DB:** run `bin/205-migrate-db` to apply 0004/0005 (B034, B035)
  before the next `bin/215-run-api-srv` or `bin/600-bundle-uat`; the API
  exits at start otherwise.
- **`etc/uat-deploy.env`:** set `UAT_DOMAIN` (B072). Until set, presales
  deploys install no nginx site.
- **Decide B071:** add a `migrate invite <user>` subcommand to replace the
  hand psql INSERT in doc/330 A9? The export scrub refuses a live login
  named `admin`: rename or delete the seeded dev admin before a real
  export.
- **Optional reviews:** B062 rows-per-page [10,15,25,50,100] default 15 vs
  GRC [10,20,25,50,100] default 25; B063 added unused
  `components/sticky-filters.tsx` (delete it and its test to defer); B066
  changed shared `use-auth-guard` to return DENIED; B069 zod 4.1.13 not
  bumped to 4.6.5 (no advisory).
- **After GRC RG0044 (repo reset) runs:** update doc/350 (B082, marked
  "Not yet run").
- **Attribution note:** commits 8dd505f..388c0ae (B085-B090) say "Claude
  Opus 5.5" but were written by Sonnet; not rewritten (pushed).
- **Existing UAT host `api-srv.env` needs hand edits before the next
  deploy** (deploys never overwrite that file): add `LOG_LEVEL=info`
  (B031: required, the API will not start without it); add
  `SERVER_BIND_ADDRESS=127.0.0.1` (B016; required since B061, the API and
  `bin/610-deploy-uat` refuse a file without it); add
  `INVITE_TTL_HOURS=72` (B035, required key); set
  `JWT_SESSION_EXPIRE_HOURS=12` (B016); rotate the DB password off the repo
  literal (B027, step in `doc/320-presales-uat-deploy.md`).
- **Grant 9020 (reset another user's MFA) by hand** on every real tenant's
  admin group (B010). No migration seeds groups, so a deployed admin loses
  MFA reset until it is added.
- **Confirm B046 scope:** Brandon's "yes" was taken to also cover renaming
  post-sales `libexec/proto-api` -> `libexec/api-srv` (revertable on its
  own).
- **Confirm B058:** prd promote refuses unless the `sha-<rev>-passed` tag
  matches, with no override.
- **Optional, B054:** the route drift check is one-way (web paths must exist
  in Go), per the recommendation. Say if it should be two-way.
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
- **Merge `backport-grc` into `main`** when Brandon approves. Not before.

## 1. Queue (doc/700 items not done, not skipped), in order

The decided queue is empty: B062-B066, B069-B078, B080-B083, B085-B088, B090,
B093, B094 and B095 are done (B089 folded into B038, B091 removed the ticket
system). Remaining work is only items awaiting Brandon's decision: B096, B097, B098. Once one is decided pull or adapt, work it with
the §2 procedure.
Skipped, do not do: B002, B005, B015, B049, B050, B057, B067, B068, B079,
B084, B092. B067 (audit log) is skipped, so B035 stripped GRC's audit
calls; keep audit writes out of later items too.

## 2. How each item is worked (the procedure every item so far used)

1. Read the item in doc/700, its Expanded block and any `Brandon:` line.
2. Implement only that item, plus what it says must land with it. Repo
   rules in `CLAUDE.md` apply (explicit security checks per handler, never
   edit an existing migration, ~80 columns, `fooList` names).
3. Add the tests the fix steps name; update tests that pinned the bug.
4. `bin/300-run-tests all` fully green before completing; counts must not
   drop without an explanation. Last full run at `388c0ae`:
   infra=15 docs=705 go=708 unit=105 e2e=70 py=103, govulncheck clean,
   `gofmt -l` empty. The gate has `infra` and `docs` modes, skip-fails,
   and a clean-tree check: the run fails if it changes `git status`; do
   not edit files while it runs.
   Infra/script items are also verified in a throwaway `ubuntu:24.04`
   container (shellcheck, `nginx -t`, `fail2ban-regex`, real permission
   runs), showing old failing and new passing.
5. In doc/700, move the item block under `# Completed`, retitle it
   `### B0NN — COMPLETED — <title>`, add a `- **Completed:**` line (what
   changed, deviations, test counts).
6. One commit per item, subject `B0NN: <summary>`.

Known flaky: `tests/test_z_error_scenarios.py::test_rate_limiting_returns_429`
failed once (404s, no 429) during B020 and passed on every run since.
