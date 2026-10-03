---
name: stale-and-insecure-review
description: Read-only whole-tree review for stale artifacts and insecure practice — four parallel passes (Go API, web app, infra/scripts/tests/config, docs) plus a git-history secret scan, findings graded H/M/L with path:line evidence, a checklist of the template's invariants phrased as counted assertions, and a report that must end with "Checked and clean" and "Not covered". Use when asked to "review the tree", "audit the codebase", look for stale or insecure code, check the template's invariants still hold, or run a periodic code review (not a per-PR review — that is the security-considerations skill).
---

# Stale-and-insecure review

A whole-tree review, not a PR review. It answers two questions: does the
tree still honour the rules in `CLAUDE.md`, and what has gone stale (dead
code, docs describing what is no longer there, scripts nobody runs)? The
per-change questions belong to the global `security-considerations` skill.

The output is a report. **The review changes nothing**: no edits, no
fixes, no commits. A finding becomes work only when Brandon picks it up.

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

**Data and docs**

- [ ] Customer data never in git: fixtures, seeds and `examples/` are
  synthetic; no export or file derived from one in any commit.
- [ ] Nothing fabricated reaches a screen, report or log: no placeholder
  counts, invented metrics or defaulted-in missing values.
- [ ] Every backticked repo path in the docs exists.
  *(gate: `scripts/check-doc-paths.sh`; check
  `scripts/doc-paths-allow.txt` entries still have true reasons.)*
- [ ] Docs describe the tree as it is (routes, env keys, scripts,
  skills); nothing names a removed feature as live.

## Report

Deliver the report in the reply, in markdown, with these sections in this
order. **A report without "Not covered" is incomplete.**

1. **Summary** — commit reviewed (`git rev-parse HEAD`), date, gate
   result and counts, finding counts by severity.
2. **Findings** — grouped H, M, L. Each: one-line title, `path:line`,
   what is wrong, why it matters, the smallest fix.
3. **Checked and clean** — every checklist line that held, with its count
   ("all N `NewAppHelper` sites ...", "0 `os.Getenv`", "N commits
   scanned, 0 secrets"). This is what lets the next reviewer skip an area.
4. **Not covered** — what was not read or could not be checked: live host
   state, files skipped, areas read only by grep sweep, anything a pass
   ran out of time on.
5. **Suggested batches** — findings grouped into changes that can land
   independently (one area, one reviewer), H first.

If Brandon wants the outstanding findings kept, they go in
`tasks/todo.md`; the report itself is not committed.
