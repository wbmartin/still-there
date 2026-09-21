---
id: PS0005
title: One agent rulebook — fold doc/9xx guidance into CLAUDE.md, fix or retire opencode.json
status: in-review
blocked_by:
waiting_on:
priority: high
domain: docs
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0005-agent-guidance-consolidation
merged:
deployed:
validated:
---
# PS0005 — Two overlapping agent-guidance systems

**Source:** template review 2026-09-21.

## Problem

Two rulebooks exist and neither knows about the other:

1. **`CLAUDE.md` + `.claude/skills/*`** — what a Claude Code session
   actually reads. Layout, tickets, branches, procedure. Actively
   maintained (PS0001).
2. **`doc/9xx-*.md` + `opencode.json`** — database rules (no FKs, no
   triggers, `security_context_id` on every table), naming conventions
   (`List` suffix, verbose names, camelCase), git preferences, Python/Go/
   TypeScript style, and `903-roles-and-goals.md` ("act as a senior
   engineer…" — duplicates the spirit of the global CLAUDE.md "Core
   Principles"). These carry real, load-bearing rules but `CLAUDE.md`
   never references them, so an agent sees them only when a human points.

`opencode.json` `instructions` lists `doc/907-security-guidelines.md` and
`doc/908-api-guidelines.md`. **Neither exists.** The real files are
`doc/907-golang-guidance.md` and `doc/908-typescript-guidance.md`. So even
the opencode path is broken, and its intent (security/API guidelines) is
unmet — no such docs exist anywhere. There is no `.opencode/` directory;
it is unclear whether opencode is still a used host.

## What to do

1. Decide: is opencode still an agent host here? If no, delete
   `opencode.json`. If yes, fix the two names and keep it pointing at the
   same list `CLAUDE.md` uses.
2. Fold or link: either (a) add a "Conventions" section to `CLAUDE.md`
   that references `doc/900`, `901`, `905`, `906`, `907`, `908` by name and
   one-line purpose, or (b) fold the short ones (`901`, `906`, `907`,
   `908` are each a few lines) into `CLAUDE.md` directly and keep only
   `900-database-guidelines.md` and `905-config-guidance.md` as docs.
   Recommendation: (b) — fewer files, one place agents read.
3. Retire `doc/903-roles-and-goals.md` (the global `~/.claude/CLAUDE.md`
   covers it) and `doc/904-python-guidance.md` unless pytest survives
   PS0014 (then keep, and reference it from `tests/README.md`).
4. `doc/902-project-commands.md` is the command reference `README.md`
   points at; it belongs with the bin/ tooling ticket (PS0014) for content
   but its *location* is decided here. Note it names
   `bin/410-regenerate-vault-certs` (actual:
   `410-regenerate-vault-mtls-certs`).
5. Make sure the `ticket-workflow` skill's check step no longer says
   `bin/300-*` (matches nothing; real runners are `200`, `310/320/330`).

## What changed and why

Per the direction below, verbatim.

| Item | Done |
|---|---|
| `opencode.json` | deleted |
| `CLAUDE.md` | new **Conventions** section between Layout and Tickets: a three-line pointer to `doc/900` / `905` / `904`, then six one-rule bullets — schema (no FKs/triggers/DEFAULTs, `security_context_id`, `<table>_id`, snake_case), explicit security checks (the PS0009 decision, marked non-negotiable), `List` naming + verbose names, 80-col Go/TS, flat env-file config, git-diff-not-memory for PR summaries. +150 words; CLAUDE.md 377 → 527 words. |
| `doc/901`, `906`, `907`, `908` | deleted — each is now one of those bullets |
| `doc/903` | deleted — `~/.claude/CLAUDE.md` Core Principles covers it |
| `doc/904` | kept (PS0014 keeps pytest); `tests/README.md` now points at it, and its stale prerequisites (TLS, "KV store, logging" services — PS0003 fallout) are replaced with how `bin/200-run-tests py` actually runs it |
| `doc/900`, `905`, `902` | stay as files. `900` had two stale lines fixed: the column is `security_context_id` (was `security_context`), and the "exclusive source for column names" pointed at another app's `/atticus-web-snipper/doc/003-…sql` — now `src/api-srv/sql/migrations/`. Item 4's `bin/410` name in `902` is moot: PS0003 removed that block. |
| `README.md` | doc-index rows for the five deleted files removed |
| `ticket-workflow` skill | check step says `bin/200-run-tests all` instead of `bin/300-*` |

## Files modified

- `CLAUDE.md`, `README.md`, `tests/README.md`, `.claude/skills/ticket-workflow/SKILL.md`
- `doc/900-database-guidelines.md`
- deleted: `opencode.json`, `doc/901-naming-preferences.md`, `doc/903-roles-and-goals.md`, `doc/906-git.md`, `doc/907-golang-guidance.md`, `doc/908-typescript-guidance.md`

## Endpoints introduced or modified

None.

## Acceptance (met 2026-09-21)

- [x] `opencode.json` is gone.
- [x] `CLAUDE.md` is the single entry point: the six rules are in it; `900`/`905`/`904` are linked by path.
- [x] `grep -rln '903-roles\|904-python' .` → only `tickets/` (this ticket's own text) and the deliberate `904` pointers in `CLAUDE.md`, `README.md`, `tests/README.md`.
- [x] `grep -rn 'bin/300' .claude` → nothing.

## Deploy needs

None. `deployed: none`.

## Related

PS0006 (template-vs-history split decides what else lives in `doc/`).

## Direction (Brandon, 2026-09-21)

Folding into `CLAUDE.md` is fine **but do not bloat context.** So:

- `CLAUDE.md` gains a *Conventions* section of ≤ 12 lines: the rules that
  bite on every change (no FKs/triggers, `security_context_id` on every
  table, `List` suffix + camelCase, 80-col Go/TS line breaks, git
  preferences) each as one line, plus a path pointer for the long form.
- `doc/900-database-guidelines.md` and `doc/905-config-guidance.md` stay
  as files and are pointed at, not inlined.
- `901`, `906`, `907`, `908` are short enough to become those single
  lines and be deleted.
- `903`, `904` retired as proposed (904 only if PS0014 keeps pytest —
  it does, see PS0014).
- `opencode.json`: **delete** (Brandon, 2026-09-21).
- Add one conventions line that records the PS0009 decision so no agent
  proposes it again: *"Security checks are called explicitly in each
  handler (JWT → rate limit → CSRF → authorization →
  `CheckAllChecksComplete`); exceptions are written in the handler, not
  hidden in middleware. Non-negotiable."*
- Alternative if the section still feels heavy: a `conventions` skill in
  `.claude/skills/` that loads on demand. Prefer the CLAUDE.md lines
  first; measure the token cost before adding a skill.
