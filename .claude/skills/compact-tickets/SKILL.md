---
name: compact-tickets
description: Fold every validated ticket folder in tickets/03-done/ into one compacted file in tickets/04-archive/ — what changed, the rules that survived, the figures at close, the endpoints it touched, the customer's answer verbatim — moving its dated log entries to 04-archive/engineering-log/ and deleting the folder. Also normalises legacy archived tickets and doc/PS*/ folders to the same form. Use when STATUS.md shows tickets under "validated, awaiting archive", when asked to compact, archive or tidy tickets, or when closed-ticket history is costing tokens.
---

# Compacting tickets

A closed ticket is read many times and written once. Compaction makes the
read cheap: one file, the conclusions only, the narrative left in git
history and in the log entries. **Lossy in the tree, lossless in git.**

Runs in a batch — one session reads N folders — on `main` directly, as
ticket metadata. It moves nothing between `01-open/` and `02-working/`,
touches no branch.

## What goes in, what comes out

```
tickets/03-done/PS0090-slug/
  PS0090-slug.md                        the ticket: what, root cause, what changed, files, endpoints, deploy needs
  PS0090-Security_Considerations.md     the per-PR security review
  260909_1130-TKT-PS0090-slug.md        dated narrative
  <anything else>                       trackers, evidence
        │
        ▼
tickets/04-archive/PS0090-slug.md          ONE file, the shape below
tickets/04-archive/engineering-log/260909_1130-TKT-PS0090-slug.md   git mv, unchanged
```

The folder is deleted. The security review, the description sections and
the evidence do not survive as files; what they *established* does.

## The archive file

```markdown
---
id: PS0090
title: <unchanged>
status: <unchanged>
blocked_by:
waiting_on:
priority: <unchanged>
domain: <unchanged>
raised: 2026-09-08
raised_by: PS0069
asked:
claimed: 2026-09-09 by TKT-PS0090-slug
merged: c5a9589
deployed: 2026-09-09
validated: 2026-09-10
---
# PS0090 — <short title>

One paragraph: the problem in the words of whoever raised it, and what
was delivered. No history of attempts.

## What changed
- files, tables, endpoints, sql migrations, tests — one line each, present tense

## Rules that survived
- each learning stated as a rule, one line, with the number that proves it
- this is the section CLAUDE.md and the skills cite instead of restating

## Figures at close
| | date |
|---|---:|
| every number the ticket measured | value |

## Endpoints                (only if the ticket introduced or changed one)
- `METHOD /path` — auth state, data classification, one line each

## Security
One paragraph, only if Security_Considerations said more than "none".

## Customer's answer          (PS1xxx only)
Verbatim, with who and when.

## Left open
- pointers to the tickets or todo items that carry what this did not finish

## Sources
- `tickets/04-archive/engineering-log/<entries>`
- git: `<first sha>` … `<merge sha>`
```

Keep the frontmatter exactly — the linter requires `validated:` in
`04-archive/` and rejects any field outside its vocabulary. Aim for 3–6 KB.

## Procedure, per ticket

1. `git checkout main && git pull`. Read the whole folder and every
   `04-archive/engineering-log/*-TKT-PS####-*` entry that already exists for
   it.
2. Write `tickets/04-archive/PS####-slug.md` in the shape above. Every
   figure copied from the folder, none invented; every rule stated once.
3. `git mv` each `YYMMDD_hhmm-TKT-PS####-*.md` in the folder to
   `tickets/04-archive/engineering-log/`. Any other dated narrative in the
   folder that is worth keeping becomes one such entry, named for the date
   it happened; the rest is history and stays in git.
4. `git rm -r tickets/03-done/PS####-slug/`.
5. `grep -rn 'tickets/03-done/PS####\|doc/PS####' --exclude-dir=.git .` —
   rewrite any path that pointed into the folder to the archive file.
6. `python3 scripts/lint-tickets.py --update-status`.

One commit per batch, listing the ids: `Compact PS0090, PS0096, PS0098`. Push.

## Legacy shapes

Two older forms exist and normalise to the same output:

- **A `doc/PS####-slug/` folder with no ticket** — from before `tickets/`
  existed. It holds `PS####-Description.md` and
  `PS####-Security_Considerations.md`, written by the global `pr-ticket-docs`
  and `security-considerations` skills. Create the archive file from the
  folder; `raised:` and `validated:` come from the folder's git history
  (`git log --format=%ad --date=short -- doc/PS####-slug | tail -1` and
  `| head -1`); `merged:` is the merge commit into `main`. Delete the folder.
- **A ticket file with a `doc/PS####-slug/` folder beside it** — a ticket
  migrated into `tickets/` while its folder was left in `doc/`. Treat the
  file plus the folder as the input; rewrite the file in place; delete the
  folder.

The linter counts a `doc/PS*/` folder as a taken number until it is gone,
so a legacy folder is never a free number — but it is not a ticket either,
and it should not outlive its compaction.

## What not to do

- **Do not compact anything in `02-working/`.** Unvalidated is not done.
- **Do not rewrite a log entry.** They move; their text is untouched.
- **Do not summarise the customer's answer.** It is quoted, with who and
  when.
- **Do not delete a branch.** That happened at validation, by the owner.
- **Do not invent a figure.** A number not in the folder is not in the
  archive file; write "not measured" if the gap matters.
