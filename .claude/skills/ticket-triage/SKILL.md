---
name: ticket-triage
description: Health-check and triage the ProtoStack ticket set — run the linter, find duplicate questions and missing blocked_by links, catch stale claims and archive drift, then propose what to ask the customer next and what engineering work is actually unblocked. Use when asked to triage tickets, check ticket health, prioritise the backlog, decide what to work on next, or work out which questions to put to the customer.
---

# Ticket triage

`scripts/lint-tickets.py` answers *is the metadata well-formed* and *what to
ask and what to start*. This skill answers only what is left: are two of
these the same question, has this claim gone cold, is a figure stale.

**The deliverable is an updated `tickets/STATUS.md`, not an essay.** Run the
linter with `--update-status` and the standings are already correct — the
customer questions ranked by what they unblock, the rest grouped by who
answers them, and the engineering work split into ready, in progress and
blocked. Do not restate any of that back to the reader; it is in the file,
and repeating it is what makes a triage run unreadable.

Report only what the generator cannot know, as a short list. **Propose
changes; do not apply them unless asked.**

## Hard constraint, read this first

**Never `git mv` a ticket between `01-open/`, `02-working/`, `03-done/` and
`04-archive/`.** That move is the claim lock, and it only works when a
person pushes it to `main` before starting work. Proposing a move is the
job; making one silently breaks the only mechanism that stops two sessions
taking the same ticket.

## 1. Lint first — errors are blockers

```bash
git fetch origin
python3 scripts/lint-tickets.py --report --update-status
```

`--update-status` rewrites the generated block at the bottom of
`tickets/STATUS.md` — open counts per range, the engineering table with
priority and blocked-on state, tickets per domain, the next free number in
each range, and the timestamp of this run. **Do this on every run**, so the
file always carries the date it was last true. Everything outside the
`BEGIN GENERATED` markers is hand-written judgement; leave it alone.

`tickets/STATUS.md` is where the standings live. `tickets/README.md`
orients, and the working procedure is the `ticket-workflow` skill — this
skill proposes, that one is what a session follows when it picks a ticket up.

If it reports problems, fix or surface those before triaging. Do not triage
a set whose metadata is broken; the counts will be wrong in ways that are
not obvious. `--report` also gives the next free number in each range — use
it rather than counting.

## 2. Duplicate questions, not duplicate numbers

The linter catches a number claimed twice. It cannot see two tickets asking
the same thing in different words. Read the frontmatter `title` plus the
first paragraph of each open ticket, grouped by `domain` — near-duplicates
almost always share one.

**The trap: a question/doing pair is not a duplicate.** A `PS1xxx` ticket
asks the customer whether a figure is basis A or basis B; a `PS0xxx` ticket
changes the calculation once that is answered. Two tickets, correctly, with
the `PS0xxx` one carrying `blocked_by: [PS1xxx]`.

The test — *would one answer close both?* If yes, propose archiving one
with a pointer to the other. If no, it is a pair, and the engineering
ticket should carry `blocked_by`.

**Missing `blocked_by` links are the more common finding than true
duplicates**, and they matter more: they are what makes step 4 computable.
Every engineering ticket that cannot start until the customer answers
something should say so.

## 3. Stale

Concrete checks, cheapest first:

```bash
# A claim with no commits against it -- the session that took it is gone.
for d in tickets/02-working/PS*/; do
  printf "%-60s %s\n" "$(basename $d)" "$(git log -1 --format='%ar by %an' -- $d)"
done
```

Read only. Never check out, fetch into, merge or delete a branch that is
not your own — `ls-remote` is what the linter uses, and it is enough.

- **Cold claim** — in `02-working/` with **no `merged:`** and no commit for
  several days. Propose moving it back to `01-open/`, per the
  `ticket-workflow` skill. A folder *with* `merged:` set is awaiting
  validation, not cold: read its body for what it waits on (a UAT deploy, a
  data load, a human) and leave it.
- **Unclaimed work** — a `TKT-*` branch on `origin` whose ticket is
  still in `01-open/`. The linter reports this as a `sync:` warning in both
  directions (branch without a claim, claim without a branch), so read those
  rather than working it out by hand. It is the invisible-work failure the
  claim scheme exists to prevent.
- **Archive drift** — a ticket described as closed in a log entry,
  `tickets/STATUS.md`, or its own body, but still sitting in `01-open/`.
- **Validated but not moved** — a `02-working/` folder whose body or log
  entry says the UAT check passed, still not in `03-done/`. Propose the
  move; the owner makes it.
- **Lingering branches** — the linter warns when a `03-done/` or archived
  ticket's branch is still on origin. Report it to the owner; **never delete
  it yourself**, and never propose that another session does.
- **Compaction backlog** — the count under *validated, awaiting archive* in
  STATUS.md. Above a handful, propose a `compact-tickets` run.
- **Chases** — a 1000-range ticket with `asked:` set weeks ago and no
  answer. List these separately from unasked ones; the action is different.
- **Stale figures** — a ticket whose prose quotes a number that has since
  moved (a total, a count, a threshold). This is the class of defect that
  reaches a client. When a ticket cites a figure, spot-check it against the
  database or the app.

## 4. Prioritise — two different jobs

### Engineering range: order by what unblocks what

Walk `blocked_by`. A ticket blocked on a customer question **nobody has
asked** is not actionable — report it as blocked rather than ranking it
high, and note which question would unblock it. Within the actionable set,
order by `priority`, then by how many other tickets each one unblocks.

### 1000 range: the ranking is generated — add only what it cannot see

**Do not derive this by hand.** The linter reverses `blocked_by` and puts
the questions that unblock engineering work at the top of *Questions for
the customer*, with what each frees and who answers it; the rest are
grouped by recipient — the customer generally, or the role `recipient:`
names. Read it, do not recompute it.

Empty `asked:` across the customer range is deliberate — Brandon paces
these so they do not all land on one contact at once. **Never propose
asking them all.**

The generator ranks by what a question unblocks. Two things it cannot
weigh, and they are the only reason to name a specific batch:

1. **Cheapness to the recipient.** A yes/no beats an investigation, and one
   heavy question with two light ones gets a faster reply than three heavy
   ones.
2. **Whether the answer expires.** Anything resting on a guess that is
   already in a figure the customer will see is worth more now than later.

If those change the order the generator produced, say so in one line each.
If they do not, say nothing and let *Questions for the customer* stand.

## 5. Report

**`tickets/STATUS.md` is the report.** Regenerate it and say so. Then add
only findings that are not in it, as a short list — no history, no recent
closes, no narrative of what the run did:

- **Blockers** — anything the linter rejected.
- **Duplicates and missing links** — proposed archives, proposed
  `blocked_by`. A missing link is worth more than a duplicate: it is what
  makes *Questions for the customer* rank correctly, and an unlinked
  blocker ranks as knowledge.
- **Stale** — cold claims, unclaimed branches, archive drift, validated but
  not moved, lingering branches, chases.
- **Stale figures** — a ticket quoting a number the data has moved past.
- **Compaction backlog** — how many sit in `03-done/`.
- **Batch adjustment** — only if cheapness or an expiring answer reorders
  what the generator produced.

If a section has nothing, omit it. A triage run with three findings should
produce three lines, not three pages.

Then stop and let the user choose. If they accept changes, apply them by
editing frontmatter (`priority:`, `blocked_by:`, `asked:`) and commit — new
tickets are committed **and pushed** immediately, and triage edits go in
one commit describing what moved and why, and rerun with `--update-status`
so the generated block reflects the change. `origin/main` is the source of
truth: an unpushed commit claims nothing, and the linter's `sync:` warnings
say when you are in that state. Proposed lifecycle moves stay proposals.
