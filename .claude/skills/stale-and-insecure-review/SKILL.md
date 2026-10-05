---
name: stale-and-insecure-review
description: Read-only whole-tree review for stale artifacts and insecure practice — four parallel passes (Go API, web app, infra/scripts/tests/config, docs) plus a git-history secret scan, findings graded H/M/L with path:line evidence, a checklist of the template's invariants phrased as counted assertions, and a report that must end with "Checked and clean" and "Not covered". Use when asked to "review the tree", "audit the codebase", look for stale or insecure code, check the template's invariants still hold, run a periodic code review, or triage and settle a review's findings in fix rounds (not a per-PR review — that is the security-considerations skill).
---

# Stale-and-insecure review

A whole-tree review, not a PR review. It answers two questions: does the
tree still honour the rules in `CLAUDE.md`, and what has gone stale (dead
code, docs describing what is no longer there, scripts nobody runs)? The
per-change questions belong to the global `security-considerations` skill.

The output is a report. **The review changes nothing**: no edits, no
fixes, no commits. A finding becomes work only when staff triage it
(see "Triage and settling").

## Rules for the reviewer

- **Read-only.** Fixing while reviewing hides what was found and mixes a
  review with a change nobody asked for.
- **Every finding cites `path:line` that the reviewer read.** A grep hit
  is a lead, not a finding, until the line is read in context. Nothing is
  reported from memory or from another review.
- **Show only what was verified** (`CLAUDE.md`): a count in the report is
  a count of lines confirmed, not an estimate. If a pass could not finish
  an area, it goes under "Not covered", not under "clean".
- **Never paste a secret.** A secret found by the scan is reported by
  path, commit and kind (`JWT_SECRET value in <sha>:<path>:<line>`), never
  by value.
- **Names come from `etc/app.env`.** Write `$APP_SLUG`-derived names, not
  this template's own slug, so the method reads the same in a derived app.

## Severity

| | Meaning |
|---|---|
| **H** | Exploitable, leaks a tenant's or a user's data, a secret in git, or broken on a host. |
| **M** | A latent bug, a rule broken in a way that has not bitten yet, or drift that misleads (a doc, comment or test that says something false). |
| **L** | Hygiene: dead code, stale names, a missing reason, style the gate does not catch. |

## Method

Run the four passes as parallel subagents (one area each, read-only), and
the secret scan alongside. Each pass returns findings with severity and
`path:line`, its checklist lines with the counts it confirmed, and what it
did not read.

1. **Go API — `src/api-srv`.** Handlers, `internal/security`,
   `internal/repository`, `sql/migrations`, `sql/roles.sql`,
   `sql/fixtures`, `cmd/`.
2. **Web app — `src/web-app`.** `utils/`, `stores/`, `pages/`,
   `constants/`, `tests/` (vitest and Playwright).
3. **Infra, scripts, tests, config — `bin/`, `scripts/`, `infra/`,
   `etc/`, `tests/`** (pytest).
4. **Docs — `CLAUDE.md`, `DERIVING.md`, the READMEs, `doc/`, `tasks/`,
   `.claude/skills/`.** A doc that describes a route, file, env key or
   behaviour that is not in the tree is an M.
5. **Secret scan over all refs.** `git log -p --all` restricted to
   `etc/ infra/ bin/ tests/ src/ scripts/ .claude/`, searched for private
   key blocks, `password=`/`secret=`/`token=` values, connection strings
   with a password other than `SERVERONLY`, cloud key prefixes, and real
   e-mail domains in fixtures. State the commit count scanned.

Before the passes, run `bin/300-run-tests all` once (read-only: it leaves
`git status` as it found it) and record the result and counts. A red gate
is the first finding.

## Invariant checklist

Each line is an assertion the report answers with a count ("all N ...",
"0 ...") or a finding. Lines marked *(gate)* are enforced by
`bin/300-run-tests`; for those, a green gate is the evidence, and the
reviewer checks only that the check still covers its scope (its canary or
floor is current, the list it walks is complete).

**Go API**

- [ ] Every `NewAppHelper` site in `internal/handler` runs JWT → rate
  limit + user block → CSRF → authorization → `CheckAllChecksComplete`
  in that order, in the handler, never middleware; every `Skip*` has a
  reason. *(gate: `internal/handler/route_check_contract_test.go`, which
  fails on a route in `cmd/server/main.go` without a row and on a skip
  without a reason.)* Review: the rows say what the handlers do.
- [ ] Every repository read and write carries the `security_context_id`
  predicate from the caller's JWT, never from the request body; list reads
  go through `buildScopedWhere` (`internal/repository/scoped_query.go`).
- [ ] A handler that takes `securityContextId` in a body overwrites it
  from the session.
- [ ] Migrations: no `REFERENCES`/foreign key, no trigger, no `DEFAULT` on
  application data; every table has `security_context_id` and a
  `<table>_id` primary key; no existing migration edited since it landed
  (`git log` per file).
- [ ] 0 `os.Getenv`/`os.LookupEnv` outside tests; config comes only from
  the flat `KEY=VALUE` file.
- [ ] Passwords hash with argon2id at the parameters in
  `internal/auth/password.go`; every secret compare (password hash, TOTP
  code) uses `subtle.ConstantTimeCompare`; 0 `math/rand` imports in
  `src/api-srv`, so every token, salt and id comes from `crypto/rand`.
- [ ] Invite tokens are stored hashed, never raw; the JWT parser refuses
  any signing method but the one it issues (`internal/auth/jwt.go`).
- [ ] Every `http.Cookie` the API sets is `HttpOnly`, `Secure` and
  `SameSite=Strict`; CORS answers only an origin listed in
  `CORS_ALLOWED_ORIGINS`, compared exactly (no wildcard, no suffix
  match).
- [ ] 0 SQL built from input: every value binds as `$N`; a
  `fmt.Sprintf` or `+` in a query joins only constants and column names
  written in code (`grep -rn 'Sprintf\|" +' internal/repository`, each
  hit read).
- [ ] 0 hard `DELETE` in app code (`grep -rni 'DELETE FROM'` over
  non-test `.go` files); rows are soft-deleted through `delete_ts` and
  `delete_user_id`.
- [ ] Every handler reads its body through `decodeJSONBody` or
  `readBody` (`internal/handler/common.go`), which cap it: 0
  `json.NewDecoder(r.Body)` or `io.ReadAll(r.Body)` anywhere else, and
  every public or login-layer write uses one of the small caps.
- [ ] `gofmt`, `go vet`, `staticcheck`, `govulncheck` clean; every
  `//lint:ignore` has a reason that is still true. *(gate)*
- [ ] Lists are `fooList`, never plurals (variables, functions, routes).

**Tenancy and tests**

- [ ] Tenant 2 is the test tenant: fixtures and tests use 2 (and 3 for
  isolation), never 1; `sql/fixtures/test-tenant-lock.sql` covers every
  tenant table, including any added since it was written.

**Web app**

- [ ] 0 `innerHTML`, `dangerouslySetInnerHTML`, `eval(`,
  `new Function(` in `src/web-app` (excluding `node_modules`).
- [ ] Raw `fetch` only in `utils/api.ts` (and the e2e harness);
  `console.*` only in `utils/report-error.ts` (and test harnesses).
  *(gate: eslint `no-restricted-globals`, `no-console`.)*
- [ ] `constants/grants.ts` equals `internal/model/grants.go`, and every
  `constants/api.ts` path is registered in `cmd/server/main.go`.
  *(gate: `src/web-app/tests/contract.test.ts` and the Go fixture test;
  check the route canary still matches the route count.)*
- [ ] No token, password or TOTP secret written to `localStorage`,
  `sessionStorage` or a log.

**Infra, scripts, config**

- [ ] Every bash script under `bin/`, `scripts/` and `infra/*/bin/` sets
  `set -euo pipefail`.
- [ ] Every secret in a committed `infra/**/*.env` is `SERVERONLY`; no
  per-host value is inlined in a script. Shared host scripts are
  byte-identical across hosts. *(gate: `bin/300-run-tests infra`.)*
- [ ] `etc/*.env` in git carry no real secret (local dev values only).
- [ ] Env keys match the code: every key in an API env file
  (`etc/*-api-srv.env*`, `infra/*/etc/*-api.env`) is one
  `internal/config/config.go` reads, or a `TEST_*` key the test harness
  reads; every key `config.go` requires is in each. A key nothing reads
  is an M: the reader trusts it does something.
- [ ] 0 passwords in argv: no password, or URL carrying one, is a
  command-line argument (`psql`, `pg_dump`, `migrate -url`), where `ps`
  and shell history show it; it travels in `PG*` variables or a
  mode-600 file. Grep `bin/`, `infra/*/bin/` and the runbooks for those
  tools and read where each URL comes from.
- [ ] Every root paste block writes each variable set in an earlier
  block as `${VAR:?}` inside every destructive or path-building command
  that uses it (`rm`, `mv`, `cp`, `chown`, `chmod`, `mkdir`, `tar -C`,
  `sed -i`), in the runbooks and `infra/README.md`. A `: "${VAR:?}"`
  line on top does not count (`infra/README.md` "Release checklist"
  says why).
- [ ] 0 `curl | sh` (or `wget ... | bash`) in scripts, runbooks or docs.

**Data and docs**

- [ ] Customer data never in git: fixtures, seeds and `examples/` are
  synthetic; no export or file derived from one in any commit.
- [ ] 0 exports or bundles left in `dist/` or `tmp/` (gitignored is not
  gone): `find dist tmp -type f` for dumps, archives and spreadsheets.
  The schema-only output the gate writes under `tmp/test-logs/` is not
  an export.
- [ ] Nothing fabricated reaches a screen, report or log: no placeholder
  counts, invented metrics or defaulted-in missing values.
- [ ] Every backticked repo path in the docs exists.
  *(gate: `scripts/check-doc-paths.sh`; check
  `scripts/doc-paths-allow.txt` entries still have true reasons.)*
- [ ] Docs describe the tree as it is (routes, env keys, scripts,
  skills); nothing names a removed feature as live.
- [ ] Every `bin/` script has a row in `doc/902-project-commands.md`
  (`for s in bin/*; do grep -q "^|.*${s#bin/}" doc/902-project-commands.md
  || echo "$s"; done` prints nothing).
- [ ] Every file in `doc/` is linked from the README doc index.
- [ ] 0 undated counts in prose: a count of tests, routes, findings or
  lines carries the date or commit it was taken at, or gives the command
  that produces it.
- [ ] Every paste block runs from where its lead-in says (host, user,
  directory): its relative paths resolve from that directory, and a host
  block does not assume a checkout.
- [ ] A historical doc (a decision log, a finished plan, a superseded
  runbook) says so in its header; every other doc reads as current.

**Tasks**

- [ ] Every finished item in `tasks/todo.md` §0 is checked off with a
  reason and date, or removed; 0 done items still read as open.
- [ ] 0 items carried as "unverified" across sessions: each is verified
  or struck.
- [ ] A plan doc's status matches the code, and every branch it cites
  exists on origin (`git ls-remote --heads origin <branch>`).
- [ ] The customer's people appear by person id, never by name, in
  `tasks/`, docs and commit messages (`git log --all --format=%B`).

## Report

Deliver the report in the reply, in markdown, with these sections in this
order. **A report without "Not covered" is incomplete.**

1. **Summary** — commit reviewed (`git rev-parse HEAD`), date, gate
   result and counts, finding counts as an area × severity table (the
   four passes plus the secret scan as rows, H/M/L as columns).
2. **Findings** — grouped H, M, L. Each: id, `[area]`, one-line title,
   `path:line`, what is wrong, why it matters, the smallest fix. Number
   them `F001`, `F002`… in report order, once (see "Triage and
   settling").
3. **Checked and clean** — every checklist line that held, with its count
   ("all N `NewAppHelper` sites ...", "0 `os.Getenv`", "N commits
   scanned, 0 secrets"). This is what lets the next reviewer skip an area.
4. **Not covered** — what was not read or could not be checked: live host
   state, files skipped, areas read only by grep sweep, anything a pass
   ran out of time on. Name two kinds explicitly: generated files
   (`doc/003-schema-reference.sql`, `model/testdata/grants.json`,
   lockfiles), and whether each was read or only trusted to its
   generator and gate; and docs historical by label, each named, with
   whether it was read or skipped because of the label.
5. **Suggested batches** — findings, by id, grouped into changes that
   can land independently (one area, one reviewer), H first.

The reply is the whole output of the review. It is committed only once
staff triage it, as below.

## Triage and settling

After the review, staff triage the report and fix the findings in
rounds. The ids and the review file are what let a code comment written
in round 1 still resolve after round 6, so keep them stable.

**Ids.** Each finding gets its id once, at report time, and keeps it for
life: through triage, every round, and the move to Settled. Number
across the whole report, never per section, so moving an item changes no
other id. Commits and code comments cite the file and id
(`review-2026-10-01 F041`).

**One defect per item.** A finding that bundles a sweep (the same defect
at many sites, or several defects under one title) splits into
sub-items `F041a`, `F041b`…, each with its own `path:line` and Status.
The parent settles when its last sub-item does.

**The review file.** Once staff have triaged, write the findings to one
committed file, `tasks/review-YYYY-MM-DD.md` (the review date), with
each item's text as reported. Its sections, in order:

1. `## Counts` — the at-review area × severity table from the Summary,
   frozen (never edited after it is written), then a live line
   `**Open:** N (H a, M b, L c). **Settled:** S.` Update the live line
   in the same commit that settles items.
2. `## Open` — items still being worked, in id order.
3. `## Settled` — items with a disposition, in id order.

Every triaged item carries one Status line:

```
**Status:** <disposition> — <who ruled>, <date>, <commit or PR>
```

Dispositions: `open (round N)`, `done`, `will not fix`, `by design`,
`deferred` (name where it went), `decided elsewhere` (name where). Every
disposition but `open` settles the item: move it, text and id
unchanged, to `## Settled`.

**Rounds.** Each round is one PR fixing a set of open items. Every round
passes the same settle gate, in this order:

1. the PR is merged;
2. `bin/300-run-tests all` is green on the merged branch;
3. staff check the change, on UAT when there is one.

Settle a round's items only after step 3, even when staff approve the
merge earlier: the approval is not the check.

**Branches.** A long-lived review branch merges each round PR with a
merge commit, so the branch fast-forwards to the target. Otherwise cut a
fresh branch from the target for each round. Squash-merging rounds into
a branch that lives on forces a merge-back after every round.

**`tasks/todo.md`** carries one queue line pointing at the review file
while its open count is above zero, never the findings themselves.
