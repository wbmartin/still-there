# To do

Branch `backport-grc` (off `main` at `c7da74d`). The work list is
`doc/700-grc-backport-feedback.md`: every item has a decision, an expanded
write-up (what is wrong, what GRC changed, the fix, the open question) and,
where Brandon answered, a `Brandon: ...` line that overrides the
recommendation. Done items sit under `# Completed` at the end of that file.

## 0. Waiting on Brandon

- **B093 undecided.** `bin/600-bundle-uat` stops at the Vite build unless
  `VITE_API_PROXY_URL` is set (`src/web-app/vite.config.ts:13-15` throws in
  production mode). Mark pull/adapt/skip in doc/700.
- **Existing UAT host `api-srv.env` needs hand edits before the next
  deploy** (deploys never overwrite that file): add `LOG_LEVEL=info`
  (B031: required now, the API will not start without it), add
  `SERVER_BIND_ADDRESS=127.0.0.1` (B016; required since B061, the API and
  `bin/610-deploy-uat` refuse a file without it) and set
  `JWT_SESSION_EXPIRE_HOURS=12` (B016), rotate the DB password off the repo literal (B027, step in
  `doc/320-presales-uat-deploy.md`).
- **Grant 9020 (reset another user's MFA) by hand** on every real tenant's
  admin group (B010). No migration seeds groups, so a deployed admin loses
  MFA reset until it is added.
- **Unknown file `radley-uat-builder-token`** appeared untracked in the repo
  root during the B033 test run and then vanished. Never committed
  (`git log --all -- radley-uat-builder-token` is empty). Check what wrote it.
- **Merge `backport-grc` into `main`** when Brandon approves. Not before.

## 1. Queue (doc/700 items not done, not skipped), in order

Next: **B034**. Then B035-B048, B051-B056, B058-B066, B069-B078,
B080-B083, B085-B090. 50 items (pull or adapt; Brandon treats them alike).
Skipped, do not do: B002, B005, B015, B049, B050, B057, B067, B068, B079,
B084, B092.

Known partial overlaps:
- B088: the two `etc/uat-api-srv.env.template` defects were fixed by B016;
  only the `.gitignore` part remains.
- B037 must keep the B004 CSRF resync inside the now-private `apiRequest`.
- B086's expansion was corrected by B075's: the script exists in Brandon's
  external loc-postgres repo; the defect is the doc depending on it.

## 2. How each item is worked (the procedure every item so far used)

1. Read the item in doc/700, its Expanded block and any `Brandon:` line.
2. Implement only that item, plus what it says must land with it. Repo
   rules in `CLAUDE.md` apply (explicit security checks per handler, never
   edit an existing migration, ~80 columns, `fooList` names).
3. Add the tests the fix steps name; update tests that pinned the bug.
4. `bin/300-run-tests all` fully green before completing; counts must not
   drop without an explanation. Last full run at `f3eb739`:
   go=386 unit=42 e2e=12 py=81, govulncheck clean. `gofmt -l` empty.
   Infra/script items are also verified in a throwaway `ubuntu:24.04`
   container (shellcheck, `nginx -t`, `fail2ban-regex`, real permission
   runs), showing old failing and new passing.
5. In doc/700, move the item block under `# Completed`, retitle it
   `### B0NN — COMPLETED — <title>`, add a `- **Completed:**` line (what
   changed, deviations, test counts).
6. One commit per item, subject `B0NN: <summary>`.

Known flaky: `tests/test_z_error_scenarios.py::test_rate_limiting_returns_429`
failed once (404s, no 429) during B020 and passed on every run since.
