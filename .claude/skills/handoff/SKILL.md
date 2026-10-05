---
name: handoff
description: Reset the session cleanly — merge origin/main into the working branch, record what is still outstanding in tasks/todo.md, commit and push everything, then emit a hand-off prompt for the next agent ending in "ready to hand off" or "Questions before you hand off". Use when wrapping up a session, resetting context, handing work to another agent or machine, or when asked to hand off, close out, or write a next-session prompt.
---

# Handing the session off

Five moves, in order: **merge `origin/main` in, write down what is left,
commit and push, check nothing is stranded, then emit the prompt.**

The point is that the next agent — on another machine, in another clone, with
none of this context — can pick the work up from `origin/main` alone.

## Why the order is the whole point

**`origin/main` is the source of truth. Local `main` is not.** A commit that
exists only in this clone is invisible to every other session, so **work that is
not pushed has not been handed off** — it has been hidden. That is why the push
comes before the prompt: a prompt describing commits nobody else can see is
worse than no prompt, because it reads as if the work is there.

**The prompt is a message, not a file.** A hand-off written to a file and
committed is stale by the time the next reader finds it — the work it points
at has moved on, and *the copy someone reads next is the stale one*. A prompt
is true for about one session. Durable facts belong in `tasks/todo.md`, the
commit messages or the docs; the prompt only carries what is needed to resume.

## 1. Merge `origin/main` into the branch

```bash
git branch --show-current      # what you are on decides the rest
git fetch origin
```

**On a `feature/*` branch** — merge `origin/main` in and resolve *here*:

```bash
git merge origin/main          # conflicts get resolved on the branch, never on main
```

Expect this to be real work sometimes. `tasks/todo.md` is append-heavy and
several sessions run at once, so two items landing at the same place is the
normal conflict, not a surprise. Resolve by **keeping both**, in date order.

**Re-verify after the merge if `main` moved.** A branch that passed before the
merge is not evidence about the merge. If nothing you depend on changed, say so
and show it:

```bash
git diff --stat <base> origin/main -- src/ bin/ scripts/ infra/ tests/
```

**On `main`** — just `git pull`. There is nothing to merge.

**Do not merge the branch into `main`.** Only staff advance `main`, and
landing a branch needs their approval as a separate act. A hand-off leaves the
branch on `origin`, ready; it does not land it.

## 2. Write down what is outstanding

Everything left goes in **`tasks/todo.md`**:

| The work | Goes in |
|---|---|
| Engineering work still to do | `tasks/todo.md`, as a queue item |
| A decision only staff can take | `tasks/todo.md` §0 |
| What happened, with evidence | the commit message, or the doc the work touched |

**Never append a dated session log to `tasks/todo.md`.** A queue buried in
narrative stops being a queue. The queue holds what is *left*; the commits
hold what *happened*.

Each outstanding item needs three things or it is not written down:

- **Where it stands** — not "in progress". Which half is done, which is not.
- **The number**, if there is one. "The total is off" ages badly; "12 rows
  over 3 tables" tells the next reader whether it moved.
- **The next physical action.** A file to open, a query to run, a person to ask.

**Verify before you write it.** If it takes a query or a test run to know, run it.

**Then prune §0.** Re-read `tasks/todo.md` §0 whole, not only the lines you
added. An entry that is now true (the migration ran, the decision was made)
gets checked off with the reason and the date (`**Closed <date>:** <why>`,
e.g. "applied on dev, `bin/205-migrate-db` clean"). An entry marked
unverified, or one you cannot tell is still true, gets verified now or
struck — never carried forward unchanged for the next session to trip over.

## 3. Commit everything and push

```bash
git status --short             # read this; do not blind-add
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

## 4. Check nothing is stranded

Four questions, each of which costs a session when skipped:

- **Is everything pushed?** `git status` clean and `git log origin/<branch>..HEAD`
  empty.
- **Is a branch being left mid-work by a session that is ending?** If the work
  is stopping rather than continuing, say so in the prompt so the next agent
  knows the branch is theirs to take, not to work around.
- **Is anything waiting on staff?** Name the act, not just "review": a pull
  request into `main` to approve and merge, a check on UAT, a deploy or
  release, a host step (`tasks/todo.md` **Host work**), or a decision. It is
  the thing that will otherwise sit for days.
- **Is anything waiting on the customer?** A question written down in
  `tasks/todo.md` but never actually asked.

## 5. Emit the prompt

**In the response, in a fenced block, so it can be copied whole.** Never write
it to a file — see the note above.

Keep it to what a cold start needs. `tasks/todo.md` holds the detail; the
prompt points at it rather than restating it, because the restatement is the
copy that goes stale.

````
```
## Hand-off — <subject>

**Branch:** `feature/...` (pushed, at <sha>) — or `main`
**Read first:** `tasks/todo.md` <section>
**State:** merged with origin/main at <sha>; deployed to UAT: <sha or no>;
<what is verified vs claimed>

### What is done
- ... with the numbers, and how each was verified

### What is next, in order
1. The next physical action — a file, a query, a person

### What will bite you
- The traps specific to this work, not the ones in CLAUDE.md

### Do not
- Merge to main without asking staff — several branches may be in flight
- <anything else specific to this work>
```
````

### The two endings

The last line of the response — **outside the fenced block**, so the prompt
pastes clean — is exactly one of:

**`ready to hand off`** — everything is pushed, the outstanding work is written
down where the next agent will find it, and nothing in the prompt is a guess.

**`Questions before you hand off`** — followed by the questions, numbered. Use
this when a real fork was hit and answering it wrong would waste the next
session: work that cannot be verified from this machine, an item that could
belong to two different branches, a branch that may be someone else's, a
decision only staff can take. **Do not use it to ask permission to do the
hand-off.**

Prefer the honest one. A hand-off that says *ready* while something is unpushed
or unverified is the failure this whole skill exists to prevent.

## Never in the prompt

- **Credentials.** Nothing from `etc/*.env` or `.env`. A hand-off prompt gets
  pasted into other sessions and other machines.
- **Deploy steps to run.** What a deploy needs (binary, web build, which
  `src/api-srv/sql/migrations/` files) goes in `tasks/todo.md`; the prompt
  says only that it is waiting on staff.
- **Anything you did not verify.** Write `unknown`. The next agent can act on
  unknown; it cannot act on a confident wrong number.
