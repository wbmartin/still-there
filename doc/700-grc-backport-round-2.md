# GRC backport, round 2: candidates

A decision list of generic improvements made in radley-grc (GRC) since the
last backport that ProtoStack (PROTO) lacks. Nothing here is applied. It is
not a plan: staff mark each **Decision**, and the work comes out of that.

- **PROTO:** `main` @ 3727be2 (round 1, B001-B099, merged and closed).
- **GRC:** branch uat @ 4ffa9e9. Range reviewed: bedf958..uat, 102 commits,
2026-10-02 to 2026-10-05. bedf958 is where round 1 stopped (its log:
`git show e4d50fe:doc/700-grc-backport-feedback.md`). GRC's main and
cleanup branches hold nothing outside uat.
- **Method:** four read-only passes over a scratch clone (Go API, web app,
infra and scripts, docs and process). The sibling checkout was not touched.
Every PROTO claim was checked against the file at 3727be2.
- **Date:** 2026-10-05.

**How the GRC-specific commits were removed.** The range has three kinds of
commit:

1. **RG0043 rounds 3-6** (8f36989..0b4d056). GRC's code review findings
47-124, each with a fix. This is where nearly all the generic work is.
2. **RG0045 host work** (83d9282..d8ac326). UAT and prod steps. The
procedure is generic; the host values are not.
3. **House work items #55-#93** (f1366cf..efc7b82), plus the move from
tickets to the Radley House tracker (5b6c300, f2ebccf). All are GRC
product features (evidence, risks, SoD, escalation, notifications,
nightly pass, legacy files) except #55, invite e-mail (C028).

The full exclusion list is at the end. The diff is heavy on docs and infra,
as expected, but rounds 3-4 also hold small security fixes to code PROTO
shares.

**How to read.** C-numbers are new; B-numbers refer to round 1. GRC paths
are written plain (they do not exist here). GRC src/internal/X is PROTO
src/api-srv/internal/X; GRC infra/uat-unified001 and
infra/prd-unified002 are PROTO `infra/uat` and `infra/prd`. "RG0043 #n"
is item n of section 11 (Settled) in GRC's review ticket. Sizes: S is under
an hour, M is a half day.

## Start here

C001-C010 are done (PR #6, merged 2026-10-05) and sit under
**Settled** at the bottom, numbers unchanged. Review picks up at C011.

## Decision table


| Item                                                    | Area   | Rec   | Size |
| ------------------------------------------------------- | ------ | ----- | ---- |
| C011 Dev env-file comments and dead keys                | infra  | pull  | S    |
| C012 `bin/216` stops vite by port; `bin/330` pins URL   | infra  | adapt | S    |
| C013 `.gitignore` and tracker-secret rule               | infra  | pull  | S    |
| C014 Retire `doc/310` (host key copied to laptop)       | infra  | adapt | S    |
| C015 `http.Server` timeouts                             | go     | pull  | S    |
| C016 Sliding-window rate limit                          | go     | pull  | S    |
| C017 Review skill: stable ids, Settled list, rounds     | docs   | adapt | M    |
| C018 Review skill: new checklist lines                  | docs   | adapt | S    |
| C019 Handoff skill corrections                          | docs   | adapt | S    |
| C020 Customer's people by id, never by name             | docs   | adapt | S    |
| C021 `doc/905` lists every env var read                 | docs   | adapt | S    |
| C022 `doc/904` matches the Python in the tree           | docs   | adapt | S    |
| C023 README tree and doc index gaps                     | docs   | pull  | S    |
| C024 Tag `template-2026-10-05` after this round         | docs   | adapt | S    |
| C025 Config struct: drop json tags                      | go     | pull  | S    |
| C026 User list query stops reading `totp_seed`          | go     | pull  | S    |
| C027 One password policy                                | go     | adapt | S    |
| C028 E-mail the invite link when mail is configured     | go+web | adapt | M    |
| C029 Router error page uses the error boundary's screen | web    | adapt | S    |
| C030 `useAuthGuard` dependencies                        | web    | adapt | S    |
| C031 `fooList` naming residue                           | web    | adapt | S    |
| C032 README rules: row keys, `rel="noreferrer"`         | web    | adapt | S    |
| C033 Placeholder favicon                                | web    | adapt | S    |
| C034 App DB role narrowed to S/I/U (was B050)           | infra  | skip  | M    |
| C035 DB passwords off argv (was B050)                   | infra  | skip  | S    |


C034 and C035 were skipped in round 1. GRC has fixed both since, so they are
listed in case staff want to reopen them.

# P2: infra and operations

### C011 — Dev env-file comments and dead keys

- **GRC source:** 4231421, 350bb3a; RG0043 #74, #89, #93.
- **GRC did:**
  - Removed every comment naming the dev API as the test target.
  - Rewrote `.env.development` to point at the dev API's `SERVER_PORT`.
  - Made `.env.production` say production sets nothing (nginx proxies
  `/api` on the same origin).
  - Dropped `localhost:3000` from CORS.
- **PROTO differs:**
  - `etc/loc-api-srv.env` carries `TEST_BASE_URL=http://localhost:8080`
  with "bin/330-test-load reads this". That is false: `bin/330-test-load`
  reads it from `etc/e2e-api-srv.env`, and nothing reads the dev copy.
  - `src/web-app/.env.development` points at that dead key.
  - `src/web-app/.env.production` still has the `api.yourdomain.com`
  placeholder, although B093 made `vite.config.ts` need it only for
  `vite serve`.
  - CORS allows port 3000, which nothing serves.
- **Recommend:** pull. Drop the dead key and the port 3000 origin, rewrite
both `.env.*` comments, and re-run `TestLoadConfig_RepoEnvFiles`.
- **Size:** S
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025; also
  `src/web-app/README.md` (API Proxy), which named a default and
  `.env.production` for `VITE_API_PROXY_URL`.

### C012 — Stopping dev leaves a test run alone; the load test cannot be re-aimed

- **GRC source:** 4231421 (RG0043 #94); 1803424 (#109).
- **GRC did:** The stop script finds the web app by its port
(`lsof -ti tcp:5173 -sTCP:LISTEN`), so a running test stack survives.
The load-test script refuses a `--base-url` on its command line.
- **PROTO differs:**
  - `bin/216-stop-local-stack` scopes the API correctly but kills every
  vite under the checkout. Its own comment says Playwright's vite on 5182
  matches too, so stopping dev kills a running e2e web server.
  - `bin/330-test-load` appends `"$@"` after its own `--base-url`, and
  Go's flag package takes the last value, so the load can be re-aimed at
  the dev API.
- **Recommend:** adapt. Kill vite by port 5173 in bin/216. Refuse
`--base-url`, `-base-url` and `--env` in bin/330.
- **Size:** S
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025. Adapted:
bin/216 kills a pid on 5173 only if its argv is also this checkout's vite,
so another checkout's dev vite on 5173 survives (GRC kills any listener);
bin/330 also refuses `--env`/`-env` and runs the check before the DB rebuild.

### C013 — `.gitignore` and the tracker-secret rule

- **GRC source:** 461c5b9, d364daa, d202ac6, 4231421 (RG0043 #99); GRC
CLAUDE.md "the repo carries no tracker URL, project id, token or API
call".
- **GRC did:**
  - Ignored .claude/settings.local.json (per-machine Claude Code
  permissions) and the tracker's root credential files, by exact name.
  - Removed ignore lines that matched nothing.
  - Wrote a rule: tracker coordinates live in a user-level skill or a
  gitignored root env file, never in the repo.
- **PROTO differs:**
  - `git check-ignore .claude/settings.local.json` matches nothing, so the
  file could be committed.
  - `.gitignore` line 2 (`examples`) matches nothing.
  - `tasks/todo.md` carries an unexplained untracked
  `radley-uat-builder-token` that "appeared and vanished". Its name
  matches GRC's tracker builder-agent token file, so it was most likely
  written by the same tooling.
- **Recommend:** pull.
  1. Add `/.claude/settings.local.json` and drop `examples`.
  2. Add the tracker-secret line to `DERIVING.md` §5.
  3. Close the todo item with that explanation, and ignore the name or a
  narrow `/*-token` pattern.
- **Size:** S
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025. Ignores
  root-only `/*-token`, not the exact name: the name is a derived app's,
  and no tracked file matches.

### C014 — Retire `doc/310`: it copies a host's TLS private key to a laptop

- **GRC source:** 4231421; RG0043 #87.
- **GRC did:** Deleted its cert "renewal" doc and the unused TLS keys.
- **PROTO differs:** `doc/310-api-srv-cert-renewal.md` is genericised, but
its procedure still copies a UAT host's Let's Encrypt privkey.pem to the
laptop's etc/crt directory. `bin/400-make-local-dev-certs` already makes local
TLS (self-signed, mode 600), and the hosts renew their own
(`infra/uat/bin/uat-proto-renew-web-certs`). PROTO's TLS keys in
`etc/loc-api-srv.env` are live (bin/400 writes those paths), so they stay.
- **Recommend:** adapt (staff's call). Delete doc/310 and point the four
references (CLAUDE.md, README, bin/400 header, doc/902) at bin/400 for
local TLS and at the host renew scripts. A real host key on a workstation
is not a practice to hand the next app.
- **Size:** S
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### C015 — `http.Server` timeouts

- **GRC source:** 8f36989; RG0043 #47.
- **GRC did:** Set `ReadHeaderTimeout` 10 s, `ReadTimeout` 2 min,
`WriteTimeout` 5 min (matching nginx `proxy_read_timeout`) and
`IdleTimeout` 2 min.
- **PROTO differs:** `src/api-srv/cmd/server/main.go` builds the server
with `Addr` and `Handler` only, so a client trickling headers holds a
connection forever. Round 1 listed this as "open in both".
- **Recommend:** pull the four fields and the comment. The values already
match PROTO's nginx. This matters most where a derived app runs without
nginx.
- **Size:** S
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025. Fields
set in a new `newHTTPServer` so a test checks them and that
`WriteTimeout` equals nginx's `proxy_read_timeout`.

### C016 — Sliding-window rate limit

- **GRC source:** 3d44381; RG0043 #59.
- **GRC did:** Added `rateWindow`: this minute's bucket plus the previous
minute's, weighted by overlap. Buckets are keyed by absolute minute
(`Unix()/60`) with a 2-minute TTL, and an unreadable bucket fails
closed. It has a boundary-burst test.
- **PROTO differs:** `src/api-srv/internal/security/app_helper.go` keys on
`time.Now().Minute()` (minute of the hour) with a 1-minute TTL, so a
client gets twice the limit across a minute boundary. Round 1 listed
this as "open in both".
- **Recommend:** pull `rateWindow` and its test into both checks, keeping
PROTO's `ip:` anonymous key (B059). Low risk, since the lockout is the
real brake on guessing; the change makes the key mean what it says.
- **Size:** S
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025. PROTO's
KV has no batch, so both checks share `rateWindow.incrementAndCount`
(INCR then GET); PROTO's `Get` answers "" for an absent key.

# P2: docs and process

### C017 — Review skill: stable ids, a Settled list, rounds with one gate

- **GRC source:** RG0043 header and section 11 preamble; abc1d8e, d951ec0,
9120ee0, db21a8e, 150274d, 0b4d056.
- **GRC did:** Ran its 124-finding review as six fix rounds. Each round
was a PR, then the gate, then the staff UAT check, then the items moved
to Settled with a Status line (disposition, who ruled, date, commit).
The bookkeeping is where it broke:
  - the sections were renumbered after every round, so "(r2)" means two
  different rounds;
  - 58 code comments cite `RG0043 X.Y` numbers that no longer resolve;
  - round 6 was settled before its UAT check, unlike rounds 3-5;
  - one item bundled 12 sub-findings, and one of them could not be found
  again;
  - the counts table went stale inside the ticket;
  - squash-merging each round forced a merge-back commit every time.
- **PROTO differs:** `.claude/skills/stale-and-insecure-review/SKILL.md`
covers the review only. The report goes in the reply, and "outstanding
findings go in `tasks/todo.md`". The skill has no ids, no Status format,
no Settled list and no rule for fix rounds.
- **Recommend:** adapt. Add a "Triage and settling" section:
  1. Each finding gets an id once (F001…) that is never renumbered;
  commits and comments cite it.
  2. After triage, write the findings to one committed file. Each gets a
  Status line, and settled items move to `## Settled` in id order.
  3. One defect per item; a sweep becomes F041a, F041b.
  4. One settle gate for every round: merged, gate green, staff check.
  5. A long-lived review branch takes merge commits, or each round gets a
  fresh branch.
  6. Keep a frozen at-review severity table and a live open count, updated
  in the commit that settles items.
- **Size:** M
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025; "staff"
  for the human who triages and checks; the Report section
  now numbers findings and gives an area × severity table, and
  `tasks/todo.md` points at the review file instead of holding findings.

### C018 — Review skill: checklist lines from GRC's run

- **GRC source:** RG0043 section 7 (Checked and clean), 8 (Not covered),

#105, #106, #111-#124.

- **GRC did:** Recorded counted assertions PROTO's checklist does not ask
for, and found a class of docs and process defects in round 6.
- **PROTO differs:** The checklist in the skill covers check order, tenant
predicate, migrations, getenv, innerHTML, grants mirror, storage,
pipefail, SERVERONLY, synthetic data and doc paths. It has none of the
lines below.
- **Recommend:** adapt. Add:
  - **Go:**
    - crypto and session: argon2id params, constant-time compare,
    `crypto/rand` for every token, invite tokens stored hashed, JWT
    method checked, cookie flags, CORS an exact allow-list;
    - 0 SQL built by concatenating input;
    - 0 hard `DELETE` in app code;
    - every public or login-layer write caps its body.
  - **Infra:**
    - env keys in `etc/*.env` equal the keys `config.go` reads, plus
    `TEST_*`;
    - 0 passwords in argv;
    - every root paste block guards each variable inside the command
    (C001);
    - no `curl | sh`.
  - **Docs:**
    - every `bin/` script has a doc/902 row (cheap to add to
    `bin/300-run-tests docs`);
    - every doc is linked from the README index;
    - no undated counts in prose;
    - every paste block runs from where it says;
    - a historical doc says so in its header.
  - **Tasks:**
    - finished §0 items are checked off with a reason;
    - nothing carried as "unverified" across sessions;
    - a plan doc's status matches the code and cites no branch missing
    from origin;
    - people by id (C020).
  - **Data:** no export or bundle left in `dist/` or `tmp/`.
  - **Not covered:** add a prompt for generated files and docs that are
  historical by label.
- **Size:** S
- **Decision:** \[ \] pull / \[X \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025; Tasks
  is its own checklist group, the dist/tmp line sits under Data and
  docs; the doc/902 row check is not yet in `bin/300-run-tests docs`.

### C019 — Handoff skill corrections

- **GRC source:** f2ebccf, 5b6c300 (handoff diff); RG0043 #116.
- **GRC did:**
  - Widened the re-verify pathspec.
  - Made "waiting on Brandon" name the act (PR, UAT check, release, host
  step, decision).
  - Put deploy state on the State line.
  - Cleaned stale todo entries by checking each one off with a reason.
- **PROTO differs:**
  - `.claude/skills/handoff/SKILL.md` re-verifies with
  `-- src/ sql/ tests/`, but PROTO has no top-level `sql/`, and `bin/`,
  `scripts/` and `infra/` are not watched.
  - The waiting list names only merge, presales deploy and decision.
  - Nothing says to prune §0.
- **Recommend:** adapt.
  - Use the pathspec `src/ bin/ scripts/ infra/ tests/`.
  - Add "a host step (Host work, C007)" to the waiting list.
  - Add "deployed to UAT: sha or no" to the State line.
  - In step 2: re-read §0, check off what is now true with reason and
  date, and verify or strike anything marked unverified.
  - Make sure all references to brandon are updated to "staff" so other people can use this

  Leave out GRC's uat → main flow (B057).
- **Size:** S
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025. "Host
  work" is the `tasks/todo.md` section name. Other skills have no
  "Brandon"; CLAUDE.md, `tasks/todo.md`, doc/330 and an app_helper.go
  comment did, and say "staff" too since (staff ruling: throughout).

### C020 — The customer's people by id, never by name

- **GRC source:** RG0043 #116, #124: `tasks/todo.md` named four customer
employees. They were replaced by person ids; only a repository reset
clears the history.
- **PROTO differs:** The CLAUDE.md "Customer data never in git" bullet
covers exports and fixtures, but not names in notes, tasks, docs or
commit messages, which is where GRC leaked them.
- **Recommend:** adapt. Extend that bullet: "that includes the customer's
people: `tasks/`, docs and commit messages refer to them by person id".
Add the line to the review checklist (C018).
- **Size:** S
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025; "by
  person id or role", matching clause in `DERIVING.md` §2; review
  checklist line aligned. No customer names found in `tasks/`, `doc/`
  or the last 200 commit messages.

### C021 — `doc/905` lists every environment variable anything reads

- **GRC source:** 1803424; RG0043 #122.
- **GRC did:** Replaced "the only env var is X" with every variable read,
each with its reader, and stated that no shipped binary reads one.
- **PROTO differs:** `doc/905-config-guidance.md` says the only variable
the Go code reads is `TEST_DATABASE_URL`. That is true for Go, but the
harnesses also read `TEST_API_ENV`, `TEST_BASE_URL` (`tests/conftest.py`),
`E2E_API_ENV`, `E2E_USER_NAME`, `E2E_PASSWORD`, `CI`
(`src/web-app/playwright.config.ts`), `E2E_DB_SEEDED`, and
`<SERVICE>_LIVE` (`bin/300-run-tests`).
- **Recommend:** adapt. Write one bullet: no shipped binary reads an
environment variable. Then list the test-only ones with their reader and
setter.
- **Size:** S
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025. List
built from the tree; also covers `TEST_ENV` and the other `bin/` overrides,
`VITE_API_PROXY_URL`, and the libpq variables pgx falls back to.

### C022 — `doc/904` says what Python the tree actually uses

- **GRC source:** 1803424; RG0043 #122.
- **GRC did:** Added a first line: what the tree's Python uses today, and
that the polars notes apply only when a script needs dataframes.
- **PROTO differs:** `doc/904-python-guidance.md` prescribes polars and
psycopg. PROTO's only Python is the pytest suite (`tests/requirements.txt`:
pytest, requests, pytest-html, pyotp). The README index says "Polars,
PEP8, psycopg".
- **Recommend:** adapt. Add the same kind of first line and fix the README
description, so an agent in a derived app does not pull in polars for a
ten-line script.
- **Size:** S
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025. CLAUDE.md's
"(pytest and scripts)" still fits; left as is.

### C023 — README tree and doc index gaps

- **GRC source:** 1803424, 49224e7; RG0043 #111, #122 (GRC's README tree
omitted `infra/`, the runbooks and 6xx).
- **PROTO differs:** It is clean of counts, personal paths and misordered
paste blocks (B051, B075, B086, B090). The remaining gaps:
  - the README tree omits `scripts/`, `tasks/` and `src/load-test/`;
  - the doc index omits `doc/003-schema-reference.sql`.
- **Recommend:** pull. Three tree lines and one index row.
- **Size:** S
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025. Also
indexed `doc/700-grc-backport-round-2.md` itself (temporary), the only
other `doc/` file missing from the index.

### C024 — Tag a template sync point after this round

- **GRC source:** GRC's CLAUDE.md "Template lineage" block and its todo,
which expects a new `template-` tag from PROTO after the carry-back.
- **PROTO differs:** `DERIVING.md` §8 is commit-based and reviews
`<last-sync>..template/main` between `template-YYYY-MM-DD` tags. The only
tag is `template-2026-09-21`, so nothing marks round 1's merge (3727be2).
GRC's lineage block still filters on the template's tickets archive,
which PROTO removed (B091), so it now returns nothing.
- **Recommend:** adapt (staff action). Tag `template-2026-10-05`, at
3727be2 or after this round lands, and push it. Tell GRC to replace its
lineage block with `DERIVING.md` §8's.
- **Size:** S
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

# P3: Go API template quality

### C025 — Config struct: drop the json tags

- **GRC source:** b8d9b30; RG0043 #56.
- **GRC did:** Removed every `json:"..."` tag from `config.Config`. Nothing
marshals it, so the tags only made an accidental `json.Marshal(config)`
print `jwt_secret` under a tidy name.
- **PROTO differs:** `src/api-srv/internal/config/config.go` tags every
field, secrets included.
- **Recommend:** pull. Delete the tags, or tag the secrets `json:"-"` if a
guard is preferred.
- **Size:** S
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-c011-c025; tags
  deleted (no `json:"-"`); a comment above the struct says it is never
  serialized.

### C026 — The user list query stops reading `totp_seed`

- **GRC source:** 8f36989; RG0043 #52.
- **GRC did:** Dropped `totp_seed` from the list SELECT, with a repository
test that a listed user's seed is nil. The single-user read keeps it.
- **PROTO differs:** `RetrieveUserList` in
`src/api-srv/internal/repository/user_repository.go` selects and scans
  it. It is not on the wire (`json:"-"`), and no list caller uses it: MFA
  reads it through `RetrieveUserByUserName`.
- **Recommend:** pull. A list query that cannot hold the encrypted second
factor cannot leak it.
- **Size:** S
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### C027 — One password policy

- **GRC source:** 3d44381, 92bfd1f; RG0043 #60.
- **GRC did:** Added `auth.ValidatePassword(userName, password, mfaExempt)`.
The minimum is 8 runes, or 15 for an MFA-exempt login (NIST 800-63B
single-factor). The password never equals the user name. There are no
composition rules. Every route that sets a password calls it.
- **PROTO differs:** "At least 8" is written five times (user create,
change own, change other, invite redeem, `/userSelfCreate`). Four of them
count bytes. Nothing checks the password against the user name.
- **Recommend:** adapt. Add the function and its test, and call it at all
five sites. Staff's call: take the 15-for-exempt rule, or only 8 plus
not-the-user-name? If 15 is taken, pytest fixtures that create exempt
logins need 15-character passwords (B097), and the web password
generator should go to 16.
- **Size:** S
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### C028 — E-mail the invite link when the server has mail configured

- **GRC source:** 95c63c1 (#55).
- **GRC did:**
  - **Mail package.** A new `internal/mail`: an SMTP sender (implicit TLS
  on 465, STARTTLS otherwise, TLS 1.2 minimum, timeouts), an RFC 5322
  composer and an invite template.
  - **Config.** An optional key group, validated at load: `MAIL_SMTP_HOST`
  (empty means off), port, username and password (both or neither),
  `MAIL_FROM`, and `APP_PUBLIC_URL` (https).
  - **Invites.** Invite create and reissue send `<APP_PUBLIC_URL>/invite#<token>`.
  The answer gains `emailed` and `emailNote`. A failed send keeps the
  invite and the copy box.
  - **Web app.** The invite modal shows the note.
- **PROTO differs:** It has no mail capability. Invites (B035) return the
link for the admin to copy, and
`src/web-app/components/invite-link-modal.tsx` says no e-mail is sent.
- **Recommend:** adapt, as one change, Go and web together. Bring over
`internal/mail`, the key group (folded into PROTO's single `LoadConfig`,
with empty keys in every repo env file and doc/905), the handler hook,
the two model fields and the modal note. Two changes from GRC's version:
  - Look up the address through PROTO's user repository.
  - Refuse to send at all without TLS. GRC refuses only to authenticate,
  so with no username it sends the one-time link in the clear.

  Every derived app will want invite e-mail, and the transport is
  provider-agnostic.
- **Size:** M
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

# P3: web app template quality

### C029 — The router's error page uses the error boundary's screen

- **GRC source:** 350bb3a; RG0043 #72, #74, #81.
- **GRC did:** Extracted `ErrorScreen` from the error boundary and
rendered it from the router's `ErrorPage` via `useRouteError()`, so both
failure paths look the same.
- **PROTO differs:** `ErrorPage` in `src/web-app/app.tsx` is a bare
unstyled `<h1>` with no way back and no `reportError`. It is the
`errorElement` on every route, so it catches every page crash first, and
the polished `components/error-boundary.tsx` only sees errors outside
the router. The boundary's link is the last literal route
(`href="/dashboard"`). PROTO already shows the message only, never the
stack.
- **Recommend:** adapt.
  - Export `ErrorScreen` and render it from `ErrorPage`.
  - Call `reportError` there.
  - Use `ROUTES.DASHBOARD` for the link.
  - Update the e2e route sweep's error heading.
  - Do not take GRC's admin-role gate.
- **Size:** S
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### C030 — `useAuthGuard` effect dependencies

- **GRC source:** 350bb3a; RG0043 #75.
- **GRC did:** Keyed the effect on `requiredGrants.join(",")`, listed the
real dependencies, and removed the `exhaustive-deps` suppression.
- **PROTO differs:** `src/web-app/hooks/use-auth-guard.ts` has the same
suppression, with a dependency on the array. All 8 callers pass a fresh
literal, so the effect re-runs every render, and a denied page repeats
its toast and `navigate` until the redirect lands.
- **Recommend:** adapt. Port the string key and the real dependencies.
Leave the four documented `form` suppressions alone. The hook is copied
into every page of every derived app.
- **Size:** S
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### C031 — `fooList` naming residue

- **GRC source:** 34b7f33; RG0043 #70 (239 renames in GRC).
- **PROTO differs:** It is mostly compliant (B038). Three plurals remain:
`availableSecurityContexts`, `selectedUsers` and the hook parameter
`requiredGrants`. Mantine `data` arrays are named `*Options`.
- **Recommend:** adapt. Rename the three. Decide whether `*Options` is a
list (`*OptionList`) and say so in one CLAUDE.md line, so the next app
does not argue it.
- **Size:** S
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### C032 — README rules for row keys and new-tab links

- **GRC source:** 350bb3a; RG0043 #83, #84.
- **GRC did:** Added a `rowKey` helper (a Symbol-keyed counter for id-less
rows) and replaced 18 `key={i}`, 7 of them on editable lists where an
index key handed a deleted row's input state to its neighbour. Added
`rel="noreferrer"` to every `target="_blank"`.
- **PROTO differs:** It is clean today: no index keys, no `_blank` links.
But the web README's house patterns say nothing about either.
- **Recommend:** adapt, as rules only.
  - A mapped row is keyed on its id, never the index. An id-less editable
  sub-list uses a symbol key: copy GRC's helper and its test when the
  first one appears.
  - `target="_blank"` always carries `rel="noreferrer"`.

  Do not ship the helper unused.
- **Size:** S
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### C033 — Placeholder favicon

- **GRC source:** 350bb3a; RG0043 #65.
- **GRC did:** Added a temporary `public/favicon.svg`, linked from
`index.html`.
- **PROTO differs:** There is no `public/` directory and no icon link, so
`/favicon.ico` falls through nginx to `index.html` and the tab shows the
browser default.
- **Recommend:** adapt, optional. Add a neutral letter-mark SVG and a
`DERIVING.md` row saying the derived app replaces it. Skip if each app
should bring its own.
- **Size:** S
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

# Skipped in round 1, fixed in GRC since

### C034 — The app DB role gets SELECT/INSERT/UPDATE only

- **GRC source:** 1803424; RG0043 #102; RG0045 C2, C4.
- **GRC did:** The app role holds its own grants: `SELECT, INSERT, UPDATE`
on tables and `USAGE, SELECT` on sequences. It has no DELETE or TRUNCATE
(nothing is hard-deleted). The host SQL stays idempotent and ends with a
one-row grant summary. doc/340 gained a "convert an existing host"
section with a check and a rollback. It is applied on UAT and prod.
- **PROTO differs:** It was skipped in B050, when GRC had no fix. PROTO
grants `ALL` to `db_writer` in two places:
  - in `infra/sql/proto-host-db.sql`;
  - in the migrations themselves (`0001_initial_schema.sql`,
  `0005_user_invite.sql`).

  `doc/340-db-host-setup.md` calls this "DML only", but `ALL` adds
  TRUNCATE, REFERENCES and TRIGGER. The API never hard-deletes.
- **Recommend:** skip, per B050. If reopened (M):
  - a new migration narrowing `db_writer`;
  - the same change in the host SQL default privileges;
  - a `doc/900-database-guidelines.md` rule for new tables;
  - a test that TRUNCATE as the app role is refused;
  - GRC's grant-summary query as the host SQL's last line;
  - correcting doc/340's "DML only" either way.
- **Size:** M
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### C035 — DB passwords off the command line

- **GRC source:** 1803424; RG0043 #106; RG0045 C1, C3.
- **GRC did:** A `libpq_env <url>` function, run in a subshell, turns a
postgres URL into `PG*` variables, so `pg_dump` and `psql` take no URL
argument. Runbooks pass passwords through `PGPASSWORD` or a mode-600
file on stdin.
- **PROTO differs:** Skipped in B050. Still open:
  - both `*-proto-install-api` scripts run `pg_dump` with a password URL;
  - on UAT, every build's `bin/510-reset-test-db` puts the CI role's
  password in argv through `TEST_PG_URL_BASE`.

  The host SQL already takes passwords on stdin (B026).
- **Recommend:** skip, per B050. If reopened (S), copy `libpq_env` into
both install scripts, keeping them byte-identical (`bin/300-run-tests infra` checks this), and build `TEST_PG_URL_BASE` without the password,
since `PGPASSWORD` is already exported.
- **Size:** S
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

# Checked: already in PROTO

GRC fixed these in this range, but PROTO already has the same or better.
Nothing to bring.


| GRC change                                                                 | PROTO already has it   |
| -------------------------------------------------------------------------- | ---------------------- |
| #49 fail-closed TOTP counter                                               | B008                   |
| #50 raw JSON decode errors echoed                                          | B011, `decodeJSONBody` |
| strict mode, paths and binary names in `bin/` (#88, #95, #96, #101)        | B095, B050, B070       |
| test harness refuses any API but the test one (#89, #100, #109)            | B018, stronger         |
| build uses the checkout's test runner; hash-checked host moves             | B058, B082             |
| one X-Frame-Options per response (RG0045 A4)                               | B022                   |
| fresh repository runbook (RG0044)                                          | B082, doc/350          |
| exports never in the repo; 6xx in doc/902 (#86, #91, #92)                  | B071, B075             |
| host env comments, placeholder rules (#97)                                 | B026                   |
| threat model "as built / not built" (#118-#120)                            | B077, doc/110          |
| doc counts, personal paths, paste-block order (#111, #112, #122)           | B051, B086, B090       |
| web: schemas README, test docs, api.ts mirror, POC comments (#64, #66-#68) | PS0013, B037, B044     |
| web: tenant-switch identity refresh, error stack hidden (#78, #81)         | already                |
| ticket skills and linter removed (#90, 5b6c300)                            | B091                   |


# Excluded as GRC-specific

- **RG0043 GRC findings:**
  - #54 and #55 were deferred in GRC, so there is no code. CORS 200 on

  OPTIONS is still open in both and can be its own item.
  - #57 is the login-layer audit chain (skipped as B067).
  - #58 `/auditRead`, #62, #63, #69 (GRC types and pages split), #77 and

  #79 (`use-draft`), #80 and #82 (will not fix), #113-#115 and #117 (GRC
  tickets and plan docs), #121.
  - The requirements text in #118-#120.
- **RG0043 #103, #104, #107, #108:** backup holds etc and keys, one
registry credential, monitor ids in filenames, tracked dev secrets.
GRC will not fix them; skipped in B050.
- **RG0043 #98:** the one-time fail2ban migration. PROTO keeps it for
fresh hosts; deleting it after use is in C008.
- **RG0043 #110:** Python one-line imports. PROTO has none.
- **RG0045 host values:** GRC keys, hostnames, audit-chain counts and SQL
hashes. The reusable shapes are in C007.
- **House #59-#93 feature commits** (f1366cf, c3cafe9, aaa4226, 192590e,
261a511, 6ae6c3d, d6f9b3c, fa3f6ae, 4ea503f, d222858, ac09374, ef5cd5a,
0211d0e, 616434c, 39720b0, 3f251bd, 9b5a45f, 5f179ce, efc7b82):
  - #87 seed `--reset` refused: PROTO's migrate has no seed command.
  - #88 matrix tests: tied to GRC's permission matrix file; PROTO has its

  grant tests and the B039 route contract.
  - #92 nightly pass: PROTO has no worker.
  - #93 phone header: GRC's domain header controls; PROTO's header is

  burger, title and menu only.
- **Tracker move:** Radley House items, its skill and procedure, the
tracker token gitignores, and GRC's Round-N todo layout (PROTO's flat §0
is better). The generic part is in C013.
- **Branch flow feature → uat → main:** PROTO stays main-only (B057). The
release-safety part is in C010.
- **GRC renames and counts:** the doc/331 rename (PROTO chose
`doc/331-prd-postsales-install.md`), GRC doc/105 and doc/810 counts, GRC
doc/902 rows, the GRC CLAUDE.md example names, and the R2 and Entra text.

# Settled

Done items, moved here in their original order with their original
numbers. All ten merged to `main` in PR #6 (merge 854e31f). Each keeps its
Status line.


| Item                                                             | Decision | Commit  |
| ---------------------------------------------------------------- | -------- | ------- |
| C001 Runbook paste blocks guard each variable inside the command | done     | 4ba97d9 |
| C002 MFA onboarding drops `/mfaVerify`'s CSRF token              | done     | b21bec6 |
| C003 Config refuses a bad JWT secret or TOTP key at load         | done     | a84ba5c |
| C004 `/mfaSetup` and `/mfaVerify` accept a temp token only       | done     | a0be2e4 |
| C005 A failed tenant-check lookup answers 500, not 403           | done     | 4016c17 |
| C006 An admin cannot delete their own login                      | done     | b634ffd |
| C007 One list of pending host work; a guarded host-change block  | done     | ab16fe5 |
| C008 How a derived app retires the presales path                 | done     | dfea118 |
| C009 `bin/600` leaves full DB-dump bundles in `dist/`            | done     | 068fc16 |
| C010 Release branches advance by merge commit, checked first     | done     | 3e5bf31 |


### C001 — Runbook paste blocks guard each variable inside the command

- **GRC source:** 1803424; RG0043 #105.
- **GRC did:** Every use of a variable set in an earlier block became
`${APP:?}` inside the command itself (23 blocks across doc/330 and
doc/331). A guard line on top does not work in an interactive root shell:
a failed `${VAR:?}` aborts only that line, and the next pasted line runs
with the variable empty.
- **PROTO differs:** It uses the guard-line form GRC rejected:
`: "${APP_ROOT:?}"`, then plain `"$APP_ROOT/..."`. There are 51 guard
lines across `doc/330-uat-postsales-cutover.md`,
`doc/331-prd-postsales-install.md`, `doc/340-db-host-setup.md`,
`doc/350-repository-reset.md` and `infra/README.md`. The failure was
reproduced in `bash -i` and `zsh -i`. Where it would damage a host:
  - doc/330 near line 296: `chown -R ... "$APP_ROOT"/{bin,libexec,...}`
  would chown `/bin`;
  - doc/330 near line 657: `mv "$APP_ROOT/etc" ...` would move `/etc`.

  The rule is also stated wrongly in `infra/README.md` (around line 187)
  and in the doc/330 Lessons.
- **Recommend:** pull.
  1. In doc/330, 331, 340 and 350, write every earlier-block variable as
  `${VAR:?}` inside each destructive or path-building command (`rm`,
  `mv`, `cp`, `chown`, `chmod`, `mkdir`, `tar -C`, `sed -i`). The top
  guard line can stay.
  2. Correct the two rule statements and say why.
- **Size:** M
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-round-2. Beyond
Recommend: a guard inside `$(...)` stops only the substitution, so the
`{ ...; } | psql` groups (doc/330 A5 and rollback, doc/331, doc/340)
open with their own `:` guard, and the 0.2 uuid fetches are assigned
first and used as `${NACK_ID:?}` etc. in `git mv`.

### C002 — MFA onboarding drops `/mfaVerify`'s CSRF token

- **GRC source:** 350bb3a; RG0043 #76.
- **GRC did:** The onboarding page calls `setCsrfToken(data.csrfToken)`
before it sets the auth state, as the `/auth` path does.
- **PROTO differs:** It has the same bug.
`src/web-app/pages/mfa-onboarding-page.tsx` sets `authState: "YES"`
(line 90) but never stores the token the API returns. Every other path
that mints a session stores it (`src/web-app/stores/auth-store.ts`). The
first write after MFA gets a 403, a `/refresh` and a retry (B004's
resync hides it).
- **Recommend:** pull. Call `setCsrfToken` inside `if (data.user)`. Add a
vitest case asserting the next write carries `X-CSRF-Token`.
- **Size:** S
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-round-2

### C003 — Config refuses a bad JWT secret or TOTP key at load

- **GRC source:** 8f36989; RG0043 #48. Round 1 B032 already did the
integer and duration keys.
- **GRC did:** `LoadConfig` refuses a `JWT_SECRET` shorter than 32
characters and a `TOTP_ENCRYPTION_KEY` that is not 64 hex characters
(AES-256). GRC rotated its short dev secret to pass.
- **PROTO differs:** Both keys are checked for presence only
(`src/api-srv/internal/config/config.go`, around lines 239-248), while
`internal/auth/totp.go` hex-decodes the key for `aes.NewCipher`. A bad
key loads, then answers 500 at the first enrolment or login. It is live
in `infra/uat/etc/uat-proto-e2e-api.env`:
`TOTP_ENCRYPTION_KEY=e2e-only-totp-key-not-for-deployment` (not hex), so
MFA on the post-sales UAT e2e harness would fail.
- **Recommend:** adapt.
  1. Add the two checks.
  2. Give the e2e file a 64-hex key.
  3. `TestLoadConfig_RepoEnvFiles` loads the redacted SERVERONLY copies
  and the `__TOTP_KEY__` template too, so exempt those placeholders or
  assert them separately.
- **Size:** S
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-round-2. The
repo-file test asserts the redacted copies and the template hold
placeholders, then appends stand-ins; harness files load as written.

### C004 — `/mfaSetup` and `/mfaVerify` accept a temp token only

- **GRC source:** 3d44381; RG0043 #61.
- **GRC did:** Removed the session (JWT) branch from both handlers. It ran
after `SkipCSRF()`, so a signed-in write skipped CSRF, and nothing called
  it. A missing temp token answers 401.
- **PROTO differs:** Both branches are still there
(`src/api-srv/internal/handler/user_mfa_handler.go`; B094 added a block
check to them). No caller uses them: the web app and every pytest send
`tempToken`.
- **Recommend:** pull. Delete both session branches and answer 401
without a temp token. In the same commit:
  - update the two rows in `route_check_contract_test.go`;
  - update B094's session subtests;
  - update the doc/105 route table;
  - update `doc/110-security-architecture.md` line 130, which cites "the
  session path of `/mfaSetup` and `/mfaVerify`".
- **Size:** S
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-round-2. Beyond
GRC: the checks now run before the body is read (the body no
longer picks a path), and B094's block subtests became 401 cases.

### C005 — A failed tenant-check lookup answers 500, not 403

- **GRC source:** 8f36989; RG0043 #51.
- **GRC did:** The security-context helpers return `(false, nil)` on a
mismatch and an error only when the lookup fails. Callers answer 500
(`SEC_CTX_CHECK`) on an error and 403 on `!valid`.
- **PROTO differs:** The helpers return an error for a mismatch, and seven
callers fold `!valid || err != nil` into 403
(`src/api-srv/internal/handler/user_handler.go`,
`user_invite_handler.go`). A DB outage reads as "access denied", and
since B006 a 403 no longer signs the user out, so nobody notices.
- **Recommend:** pull GRC's split as is, and update
`user_handler_test.go`. Derived apps copy this handler shape for every
tenant check.
- **Size:** S
- **Decision:** \[X \] pull / \[ \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-round-2

### C006 — An admin cannot delete their own login

- **GRC source:** 8f36989; RG0043 #53.
- **GRC did:** `DELETE /user` with the caller's own id answers 400
(`DEL_SELF`), like the existing self-block refusal.
- **PROTO differs:** `DeleteUser` in `user_handler.go` has no self check,
while `user_block_handler.go` refuses self-block (`BLK_SELF`). The last
admin can delete themselves.
- **Recommend:** pull. Four lines and a Go test. Optionally hide Delete on
the user's own row in the web app.
- **Size:** S
- **Decision:** \[X \] pull / \[ \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-round-2. The
optional web change was taken: Delete is hidden on the caller's own
login (list row and edit page), matched by user name as Block is.

### C007 — One list of pending host work; a guarded host-change block

- **GRC source:** RG0045 (83d9282, e3e02d3, 4d9c265, 177182e, 11b0f29 and
more); GRC CLAUDE.md rule.
- **GRC did:**
  - **One list.** All server work not yet done lives in one list. Each line
  gives the host, the command, the check that proves it, and the
  rollback. Agents add lines but never run or tick them; only staff's
  report ticks one.
  - **It caught a real miss.** A read-only re-check found that a UAT nginx
  step from the first release had never been run.
  - **Reusable shapes:**
    - `hostname` as the first line of every block;
    - host SQL hash-checked against `git show origin/main:`;
    - backup first, then stop if `echo "backup exit $?"` is not 0;
    - then apply, verify, and keep a one-statement rollback;
    - before a release that tightens config validation, an env-file
    pre-check that prints key presence and lengths, never values;
    - `sleep 2` between `systemctl reload nginx` and the `curl` check
    (the old workers answer otherwise).
- **PROTO differs:**
  - `infra/README.md` has the release checklist and the hash-checked file
  move (B082). It has no `hostname` line, no host-SQL variant, no config
  pre-check and no host-work rule.
  - `tasks/todo.md` §0 mixes staff decisions with host env edits
  (`LOG_LEVEL`, `SERVER_BIND_ADDRESS`, `INVITE_TTL_HOURS`,
  `ALLOW_MFA_EXEMPT_VIA_API`) that have no check or rollback. A promote
  with any of them missing installs an API that refuses to start.
- **Recommend:** adapt.
  1. Add a `## Host work` section to `tasks/todo.md` (host, change,
  command, check, rollback) and move the host steps out of §0.
  2. Add a CLAUDE.md Conventions bullet: a change that needs a hand step on
  a host adds a line there; agents never run or tick it.
  3. In the `infra/README.md` Release checklist, add:
     - a pre-flight item: a new or tightened required key means check each
     host's env file first, printing lengths only;
     - `hostname` first;
     - the host-SQL block (backup, stop, apply, verify, rollback);
     - `sleep 2` after a reload.
- **Size:** S
- **Decision:** \[ \] pull / \[X \] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-round-2. Host work
in `tasks/todo.md` (H1-H12) gives each step a host, command, check and
rollback, plus the C003 e2e env-file step; the host-SQL block in
`infra/README.md` splits backup and apply into two pastes, so a failed
backup stops the paste rather than relying on a comment.

### C008 — How a derived app retires the presales path

- **GRC source:** 4231421, 9400622; RG0043 #85, #86, #90, #91, #98.
- **GRC did:** After cutover it deleted:
  - the presales bundle and deploy scripts, the two `etc/uat-*` templates
  and the hand-deploy web tarball script;
  - the presales-only skills and their doc/902 rows;
  - the one-time fail2ban migration script, already applied;
  - the local bundles in `dist/`.

  doc/330's opening line records the retirement.
- **PROTO differs:** It is a template and keeps presales, which is right,
but no document says how a derived app retires it. Section B5 of
`doc/330-uat-postsales-cutover.md` only says 610 now refuses, and C only
drops the presales DB. Deleting by hand has traps:
  - `bin/620-export-tenant-data` and `bin/605-survey-host` both source
  `etc/uat-deploy.env`;
  - `scripts/doc-paths-allow.txt` allows doc/320 paths;
  - `doc/110-security-architecture.md` rows 2.5, 3.1 and 3.5 describe
  presales controls;
  - CLAUDE.md, `DERIVING.md` §3, the `infra/README.md` banner and the
  presales-deploy and handoff skills all mention it.
- **Recommend:** adapt, docs only (delete nothing in PROTO). Add section
"D. Retire the presales path in this app's repo" to doc/330. It lists:
  - the files to delete;
  - the keys to move out of `etc/uat-deploy.env` first;
  - the docs to reword and the allow lines to drop;
  - deleting `migrate-to-srv-config` once every host has run it.

  Point to it from `DERIVING.md` §3.
- **Size:** S
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-round-2. Also
lists two traps found in the tree: the Go env-file test counts the
template (fails at 5), and `bin/110-build-web-app` becomes optional dead
weight.

### C009 — `bin/600` leaves full DB-dump bundles in `dist/`

- **GRC source:** RG0043 #91, #123 (a presales bundle holding an unscrubbed
real DB sat in the ignored `dist/`).
- **GRC did:** Deleted the bundles. Its export script names any leftover
in `dist/` and prints the `rm`.
- **PROTO differs:** `bin/620-export-tenant-data` already refuses an output
directory inside the checkout, which is stronger than GRC. But
`bin/600-bundle-uat` writes a tarball holding a full `pg_dump` to
`dist/` and never removes it after the scp, so bundles pile up: GRC's
exact case.
- **Recommend:** adapt. Delete the tarball after a successful scp. For a
`--no-scp` build, list the earlier bundles found and print the `rm -f`.
- **Size:** S
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-round-2. Beyond
Recommend: a failed scp keeps the bundle and prints the retry; a failed
build removes its staging dir (EXIT trap); the earlier-bundle list
prints on every run, not only `--no-scp`.

### C010 — Release branches advance by merge commit, checked first

- **GRC source:** f2ebccf; RG0043 "Worked 2026-10-03"; RG0045 A2, B2.
- **GRC did:** Moved to feature → uat → main. A release is a uat → main PR
with a merge commit, never a squash, and
`git merge-tree --write-tree origin/main origin/uat` must exit 0 first.
One squash left the branches conflicting in 19 files, and fixing it took
a synthetic release branch with both parents.
- **PROTO differs:** It is main-only (B057 skipped the two-branch model).
`doc/331-prd-postsales-install.md` B6 discusses the release-branch choice
but not how that branch is advanced.
- **Recommend:** adapt, docs only. Add two sentences to doc/331 B6 and a
pre-flight item to `infra/README.md`: if an app moves an integration
branch into a release branch, use merge commits only, and run the
`git merge-tree` check before the PR. Add GRC's read-only grep that the
build hook's branch and `BRANCH_TO_CHECKOUT` agree.
- **Size:** S
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip
- **Status:** done 2026-10-05 on feature/grc-backport-round-2. The
branch/hook grep went into `infra/README.md` one-time setup step 7
(where the hook branch is set), with the prod promote-hook variant.

