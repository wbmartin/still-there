---
name: ticket-workflow
description: The ProtoStack ticket procedure — take a free number, claim it on origin/main as a folder in tickets/02-working/, branch as TKT-<id>-<description>, run the checks, ask Brandon before merging to main, land it yourself, keep it in 02-working/ until validated on UAT, then move the folder to 03-done/ and delete your branch. Also the frontmatter vocabulary and the two number ranges. Use when starting, creating, numbering, claiming, releasing, merging, validating or finishing a ticket.
---

# Working a ticket

```
01-open/PS0023-slug.md                  filed, unclaimed
02-working/PS0023-slug/PS0023-slug.md   claimed → merged to main → deployed to UAT → VALIDATED, all here
03-done/PS0023-slug/                    validated; your branch deleted; awaits compaction
04-archive/PS0023-slug.md               compacted to one file by the compact-tickets skill
04-archive/engineering-log/             dated entries, one file each, never compacted
```

**`origin/main` is the source of truth; local `main` is not.** A ticket
committed and not pushed is not filed, and a number claimed and not pushed
is not taken. `git fetch` before you take a number, and push in the same
breath as you commit.

## Branches: yours, `main`, nothing else

**`main` is the default and integration branch, and it is also what the UAT
server deploys from. Landing a `TKT-` branch on `main` needs Brandon's
approval.**

**An agent touches `main` and its own `TKT-` branch. It never checks out,
merges, rebases, pulls or deletes any other branch.** Remote branches are
how Brandon sees which workspaces are on which tickets. A human can override
this for a specific branch; that is a person's instruction, never an agent's
call.

## 1. Take a number

```bash
git fetch origin
python3 scripts/lint-tickets.py --report     # prints the next free number
```

Use the linter, not a count: it reads all four lifecycle folders and any
legacy `doc/PS*/` folders. Ids are `PS` plus four digits, zero-padded.

| Range | Who acts | Filename |
|---|---|---|
| `PS0000`–`PS0999` | us — engineering work | `PS0023-hyphenated-description.md` |
| **`PS1000`+** | **the customer/stakeholder** | `PS1001-ASK-CUSTOMER-hyphenated-description.md` |

**One question per file in the PS1000 range.** PS1xxx tickets skip
02-working/ and 03-done/: they go from `01-open/` to `04-archive/` when the
answer arrives, with the answer verbatim and `validated:` set.

## 2. Claim it on `origin/main`, branch, push both

The claim is a folder. The push is the lock.

```bash
git checkout main && git pull
mkdir tickets/02-working/PS0023-slug
git mv tickets/01-open/PS0023-slug.md tickets/02-working/PS0023-slug/
# set  claimed: 2026-09-15 by TKT-PS0023-slug  in the frontmatter
python3 scripts/lint-tickets.py
git commit -am "Claim PS0023" && git push          # rejected? someone beat you to it
git checkout -b TKT-PS0023-slug && git push -u origin TKT-PS0023-slug
```

Branch and folder share the id and slug. Pushing the branch in the same
step as the claim is what keeps the linter's `sync:` check quiet.

**Ticket metadata goes direct to `main`** — creating, claiming, a figure
correction in a ticket body, the move to 03-done/, archiving. **Everything
else goes on the branch**: Go, React, `src/api-srv/sql/migrations/NNNN_*.sql`,
tests, docs, config, scripts, and the ticket's working papers.

## 3. Work in the folder

Everything about a live ticket lives in `tickets/02-working/PS0023-slug/`:

| File | Written by | When |
|---|---|---|
| `PS0023-slug.md` | you | the ticket; keep its body current — it is the PR description |
| `PS0023-Security_Considerations.md` | the global `security-considerations` skill | before asking for the merge — required |
| `YYMMDD_hhmm-TKT-PS0023-slug.md` | you | dated narrative — the engineering-log entry, written here, moved to `04-archive/engineering-log/` at compaction |
| anything else | you | trackers, measurements, evidence |

The ticket body is what the global `pr-ticket-docs` skill's "Description"
becomes, so before the merge it must carry these sections: **What**,
**Root cause or requirements**, **What changed and why**, **Files modified**,
**Endpoints introduced or modified** (auth state, data classification), and
**Deploy needs** (Go binary? web build? which `src/api-srv/sql/migrations/`
files?).

No `doc/PS*/` folder. No entry written straight into `04-archive/`.

## 4. Check, ask before the merge, then do the whole merge

Finish the work and prove it locally — the checks are part of what Brandon
approves:

```bash
cd src/api-srv && go build ./...
cd src/web-app && npx tsc --noEmit -p tsconfig.app.json
bin/300-run-tests all             # rebuilds the scratch DBs, then go + playwright + pytest
```

Write **Deploy needs** in the ticket body, push the branch, ask. When he says yes:

```bash
git fetch origin
git checkout TKT-PS0023-slug
git merge origin/main            # resolve HERE, never on main; re-verify after
git checkout main && git pull
git merge --no-ff TKT-PS0023-slug
git push                          # rejected? pull, merge on the branch, re-verify, push
```

Then set `merged: <sha>` in the ticket and push to `main`. The ticket stays
in `02-working/`. STATUS.md lists it under *merged, awaiting validation*.

Merging to `main` deploys nothing by itself. Brandon runs the UAT deploy
(`bin/600-bundle-uat`, `bin/610-deploy-uat`) from `main`; you verify from
here afterwards.

## 5. Validate, move to 03-done/, delete your branch

**Validated** means the change is on UAT and its check passes — the test
suite green against UAT, or the verification the ticket body states — or,
for a ticket with nothing to deploy, that it is merged. A ticket that needs
a UAT deploy, a data load, or a human waits in `02-working/` with a note in
its body saying what it is waiting for. That is the intended state, not a
cold claim.

When it is proved, in one commit to `main`:

```bash
# set  deployed: 2026-09-20   (or  deployed: none)
# set  validated: 2026-09-20
git mv tickets/02-working/PS0023-slug tickets/03-done/
python3 scripts/lint-tickets.py --update-status
git commit -am "PS0023 validated" && git push
git push origin --delete TKT-PS0023-slug        # your branch, only now, only you
```

Compaction to `04-archive/` is a separate, batched step — the
`compact-tickets` skill. Do not archive by hand.

## A stale claim is not sacred

A `02-working/` folder with no `merged:` and no commit for several days
belongs to a session that is gone. `git log -1 --format='%an %ar' -- tickets/02-working/<folder>`,
say so in the commit message, and move the folder's ticket file back to
`01-open/` (clear `claimed:`, delete the folder). A folder *with* `merged:`
set is waiting for validation, not cold — read its body.

## Frontmatter

`scripts/lint-tickets.py` checks it; run it before every commit that touches
`tickets/`.

```yaml
---
id: PS0023               # matches filename and folder
title: Vendor group edit hard-codes the period year
status: ready            # ready | blocked | waiting-on-customer | in-review | needs-triage
blocked_by: [PS1003]     # one direction only; "blocks" is derived
waiting_on: Stakeholder
priority: medium         # high | medium | low | investigate
domain: web              # api web calc data seed deploy docs security tooling infrastructure
raised: 2026-09-15
raised_by: Brandon       # a person, or the ticket/run that surfaced it
asked:                   # PS1000-range: the date it was put to the customer
claimed:                 # date and branch, on the move to 02-working/
merged:                  # sha, once on origin/main
deployed:                # date it reached UAT, or `none` for a ticket with nothing to deploy
validated:               # date, on the move to 03-done/ (or 04-archive/ for PS1xxx)
---
```

Optional, PS1000-range only: `recipient: Customer | Stakeholder | IT |
Operations` when the question goes to one specific role rather than the
customer generally.

**The directory owns lifecycle; the header records the dates that earned
each move.** `01-open/` with `status: blocked` is coherent. The grammar is
`key: value` and `key: [a, b]` only — the linter hand-parses it so that it
needs no `pip install`.
