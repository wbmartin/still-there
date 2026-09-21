---
name: handoff
description: Reset the session cleanly — merge origin/main into the working branch, record what is still outstanding in the ticket or tasks/todo.md, commit and push everything, then emit a hand-off prompt for the next agent ending in "ready to hand off" or "Questions before you hand off". Use when wrapping up a session, resetting context, handing work to another agent or machine, or when asked to hand off, close out, or write a next-session prompt.
---

# Handing the session off

Five moves, in order: **merge `origin/main` in, write down what is left,
commit and push, check nothing is stranded, then emit the prompt.**

The point is that the next agent — on another machine, in another clone, with
none of this context — can pick the work up from `origin/main` alone.

## Why the order is the whole point

**`origin/main` is the source of truth. Local `main` is not.** A commit that
exists only in this clone is invisible to every other session, so **work that is
not pushed has not been handed off** — it has been hidden. That is the same rule
the ticket workflow rests on, and it is why the push comes before the prompt: a
prompt describing commits nobody else can see is worse than no prompt, because
it reads as if the work is there.

**The prompt is a message, not a file.** A hand-off written to a file and
committed is stale by the time the next reader finds it — the ticket it points
at has moved on, and *the copy someone reads next is the stale one*. A prompt
is true for about one session. Durable facts belong in the ticket folder or in
its log entry; the prompt only carries what is needed to resume.

## 1. Merge `origin/main` into the branch

```bash
git branch --show-current      # what you are on decides the rest
git fetch origin
```

**On a `TKT-*` branch** — merge `origin/main` in and resolve *here*:

```bash
git merge origin/main          # conflicts get resolved on the branch, never on main
```

Expect this to be real work sometimes. `tasks/todo.md` is append-heavy and
several sessions run at once, so two items landing at the same place is the
normal conflict, not a surprise. Resolve by **keeping both**, in date order.
The engineering log is a directory, one file per entry, so two sessions
writing two entries touch two different paths and there is nothing to resolve.

**Re-verify after the merge if `main` moved.** A branch that passed before the
merge is not evidence about the merge. If nothing you depend on changed, say so
and show it (`git diff --stat <base> origin/main -- src/ sql/ tests/`).

**On `main`** — just `git pull`. There is nothing to merge, and a hand-off
from `main` usually means the session was doing ticket metadata or triage.

**Do not merge the branch into `main`.** Landing a ticket needs Brandon's
approval and is a separate act — see the `ticket-workflow` skill. A hand-off
leaves the branch on `origin`, ready; it does not land it.

## 2. Write down what is outstanding

One item, one home. Route by who can act on it, then **write it where that
person will look**:

| The work | Goes in |
|---|---|
| A ticket is claimed in `tickets/02-working/` | **the ticket body**, as a dated `## Worked <date>` section |
| No ticket — loose engineering work | `tasks/todo.md`, as a queue item |
| A decision only Brandon can take | `tasks/todo.md` §0 |
| What happened, with evidence | a dated `YYMMDD_hhmm-TKT-PS####-slug.md` in the ticket's `02-working/` folder (moved to `04-archive/engineering-log/` at compaction); `TKT-NONE` entries go straight to `04-archive/engineering-log/` |

**Never append a dated session log to `tasks/todo.md`.** A queue buried in
narrative stops being a queue. The queue holds what is *left*; the log holds
what *happened*.

Each outstanding item needs three things or it is not written down:

- **Where it stands** — not "in progress". Which half is done, which is not.
- **The number**, if there is one. "The total is off" ages badly; "12 rows
  over 3 tables" tells the next reader whether it moved.
- **The next physical action.** A file to open, a query to run, a person to ask.

**Verify before you write it.** If it takes a query or a test run to know, run it.

## 3. Commit everything and push

```bash
git status --short             # read this; do not blind-add
python3 scripts/lint-tickets.py
git add -A
git commit
git push
```

**Read `git status` before adding.** A session leaves debris that must not land:
`tmp/`, downloaded exports, virtualenvs, scratch SQL, `__pycache__`, `dist/`
builds. Anything genuinely temporary belongs in the scratchpad directory, not
the repo.

**Push in the same breath as the commit.** If the push is rejected, pull, merge,
re-verify and push again — that loop is the work, not an error.

**Ticket lifecycle moves are not part of a hand-off.** Do not `git mv` a
ticket between `01-open/`, `02-working/`, `03-done/` and `04-archive/` to
tidy up. That move *is* the claim lock, and doing it on a branch hides the
very claim it looks like it is making.

## 4. Check nothing is stranded

Five questions, each of which costs a session when skipped:

- **Is everything pushed?** `git status` clean and `git log origin/<branch>..HEAD`
  empty.
- **Does the linter pass?** Nothing runs it for you — there is no CI here.
- **Is a claimed ticket still claimed by a session that is ending?** If the work
  is stopping rather than continuing, say so in the prompt so the next agent
  knows the claim is theirs to take, not to work around.
- **Is anything waiting on Brandon?** A merge approval, a UAT deploy
  (`bin/600-bundle-uat`, `bin/610-deploy-uat`), a decision. Name it; it is
  the thing that will otherwise sit for days.
- **Is anything waiting on the customer?** A `PS1000`-range ticket with an
  empty `asked:` is written down but never asked.

## 5. Emit the prompt

**In the response, in a fenced block, so it can be copied whole.** Never write
it to a file — see the note above.

Keep it to what a cold start needs. The ticket folder holds the detail;
the prompt points at it rather than restating it, because the restatement is
the copy that goes stale.

````
```
## Hand-off — <ticket id or subject>

**Branch:** `TKT-...` (pushed, at <sha>) — or `main`
**Ticket:** `tickets/02-working/PS00NN-.../PS00NN-....md` — read this first
**State:** merged with origin/main at <sha>; <what is verified vs claimed>

### What is done
- ... with the numbers, and how each was verified

### What is next, in order
1. The next physical action — a file, a query, a person

### What will bite you
- The traps specific to this work, not the ones in CLAUDE.md

### Do not
- Merge to main without asking Brandon — several branches may be in flight
- <anything else specific to this work>
```
````

### The two endings

The last line of the response — **outside the fenced block**, so the prompt
pastes clean — is exactly one of:

**`ready to hand off`** — everything is pushed, the linter passes, the
outstanding work is written down where the next agent will find it, and nothing
in the prompt is a guess.

**`Questions before you hand off`** — followed by the questions, numbered. Use
this when a real fork was hit and answering it wrong would waste the next
session: work that cannot be verified from this machine, an item that could
belong to two different tickets, a claim that may be someone else's, a decision
only Brandon can take. **Do not use it to ask permission to do the hand-off.**

Prefer the honest one. A hand-off that says *ready* while something is unpushed
or unverified is the failure this whole skill exists to prevent.

## Never in the prompt

- **Credentials.** Nothing from `etc/*.env` or `.env`. A hand-off prompt gets
  pasted into other sessions and other machines.
- **Deploy steps to run.** What a deploy needs (binary, web build, which
  `src/api-srv/sql/migrations/` files) goes in the ticket body; the prompt
  says only that it is waiting for Brandon.
- **Anything you did not verify.** Write `unknown`. The next agent can act on
  unknown; it cannot act on a confident wrong number.
