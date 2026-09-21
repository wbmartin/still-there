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
claimed:
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

## Acceptance

- Every file `opencode.json` references exists, or `opencode.json` is gone.
- `CLAUDE.md` is the single entry point: every rule an agent must follow
  is in it or linked from it by path.
- `grep -rln '903-roles\|904-python' .` returns nothing unless kept on
  purpose.

## Deploy needs

None (docs only). `deployed: none`.

## Related

PS0006 (template-vs-history split decides what else lives in `doc/`).
