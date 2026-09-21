---
id: PS0005
title: One agent rulebook — fold doc/9xx guidance into CLAUDE.md, fix or retire opencode.json
status: ready
blocked_by:
waiting_on:
priority: high
domain: docs
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0005-agent-guidance-consolidation
merged: 3d7384c
deployed: none
validated: 2026-09-21
---
# PS0005 — Two overlapping agent-guidance systems

Two rulebooks existed and neither knew about the other: `CLAUDE.md` +
`.claude/skills/*` (what a Claude Code session actually reads) and
`doc/9xx-*.md` + `opencode.json` (load-bearing schema/naming/style rules
an agent only saw if a human pointed at them, plus a broken pointer —
`opencode.json` named two `doc/907`/`908` files that had different real
names). Folded the short docs into `CLAUDE.md`, deleted `opencode.json`,
kept the two long-form docs as files.

## What changed
- `opencode.json` deleted (Brandon, 2026-09-21: not a used agent host here).
- `CLAUDE.md` gains a **Conventions** section (≤12 lines) between Layout and Tickets: a pointer to `doc/900`/`905`/`904`, then six one-rule bullets — schema (no FKs/triggers/DEFAULTs, `security_context_id`, `<table>_id`, snake_case), explicit security checks (non-negotiable, PS0009 decision), `List` naming + verbose names, 80-col Go/TS, flat env-file config, git-diff-not-memory for PR summaries.
- `doc/901-naming-preferences.md`, `doc/903-roles-and-goals.md`, `doc/906-git.md`, `doc/907-golang-guidance.md`, `doc/908-typescript-guidance.md` deleted — each folded into a `CLAUDE.md` bullet, or (903) already covered by the global `~/.claude/CLAUDE.md`.
- `doc/900-database-guidelines.md` kept as a file; two stale lines fixed — the column is `security_context_id` (was `security_context`), and the schema-reference pointer that named another app's file (`/atticus-web-snipper/...`) now points at `src/api-srv/sql/migrations/`.
- `doc/904-python-guidance.md` kept (PS0014 keeps pytest); `tests/README.md`'s stale pre-PS0003 prerequisites (TLS, "KV store, logging" services) replaced with how `bin/200-run-tests py` actually runs it.
- `README.md` doc-index rows for the five deleted files removed; `ticket-workflow` skill's check step changed from `bin/300-*` (matched nothing) to `bin/200-run-tests all`.

## Rules that survived
- One agent rulebook: `CLAUDE.md` is the entry point; long-form docs (`900`, `905`, `904`) are linked by path, not duplicated.
- The explicit-security-checks rule is recorded where every session reads it: "Security checks are called explicitly in each handler ... not hidden in middleware. Non-negotiable."
- A schema-reference pointer names this repo's own migrations, never another product's schema file.
- Fold short guidance into `CLAUDE.md`; keep only long-form docs as separate files — measure the token cost before adding more.

## Figures at close
| | date |
|---|---:|
| `CLAUDE.md` word count | 377 → 527 (+150 words) |
| `grep -rln '903-roles\|904-python' .` | only `tickets/` (this ticket's own text) and deliberate `904` pointers |
| `grep -rn 'bin/300' .claude` | 0 |
| `opencode.json` | gone |

## Endpoints
None.

## Security
No code, config, or endpoints touched — docs and agent guidance only. The value here is where two things now live: the explicit-security-checks rule (the control that stops an agent from "helpfully" moving JWT/CSRF/authorization into middleware, where an exception path would be invisible at the handler) is now in `CLAUDE.md`, read on every session start rather than in a doc nobody pointed agents at. And `doc/900` no longer points an agent at another application's schema file, which could otherwise have pulled column names and data-model assumptions from a different product.

## Left open
- `CLAUDE.md`'s +150 words is measured, not guessed, against Brandon's "≤12 lines" ceiling (six bullets); revisit if it grows further.
- What else belongs in `doc/` vs. history is decided by PS0006 (template-vs-history split).

## Sources
- `tickets/04-archive/engineering-log/260921_1524-TKT-PS0005-agent-guidance-consolidation.md`
- git: `3d7384c`
