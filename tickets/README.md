# Tickets

Everything this app is doing, waiting on, or has finished, one file per
item. Two kinds:

| Number | What it is | Who acts |
|---|---|---|
| `PS0001`–`PS0999` | Engineering work on the app — the Go API, the web app, a calculation, a seed, a document | us |
| `PS1000` and up | **A question for the customer** — something only the stakeholder or the business can answer | **the customer** |

**`tickets/STATUS.md` is where every ticket stands.** Its first section,
*Questions for the customer*, lists the open questions grouped by who can
answer them, with the ones that unlock engineering work at the top. Each
question is one file in `01-open/`, named `PS1NNN-ASK-CUSTOMER-…`, with the
background, the numbers, and what we will do with the answer. When it is
answered the file moves to `04-archive/` with the answer recorded in it, so
the archive is also the record of what the customer has told us.

## Where a ticket lives

The folder is the state. A ticket moves forward through the four and is
never deleted.

```
01-open/       filed, nobody working on it            PS####-slug.md
02-working/    claimed and in progress, through        PS####-slug/     (the ticket, its
               merge and UAT deploy until validated                      security review, its papers)
03-done/       validated; waiting to be archived      PS####-slug/
04-archive/    finished, one file per ticket           PS####-slug.md
04-archive/engineering-log/   dated working notes, one entry per file
```

Questions for the customer skip the middle two: `01-open/` until answered,
then `04-archive/`.

Every ticket opens with a short header — id, title, status, what it is
blocked by, who it is waiting on, and the dates it was raised, asked,
merged, deployed and validated. `STATUS.md` is generated from those headers,
so it is current whenever the header is.

## For the engineering side

The working procedure is the **`ticket-workflow`** skill, and it is the only
copy: taking a number, claiming, branching, the checks, validating on UAT,
the header fields. `ticket-triage` health-checks the set, `compact-tickets`
folds `03-done/` (and any legacy `doc/PS*/` folders) into `04-archive/`. The
global `pr-ticket-docs` and `security-considerations` skills write into the
ticket's `02-working/` folder, never into `doc/`.

```bash
python3 scripts/lint-tickets.py                   # is every header well-formed?
python3 scripts/lint-tickets.py --report          # counts and the next free number
python3 scripts/lint-tickets.py --update-status   # regenerate STATUS.md
```

Four rules carry the weight: claim on `origin/main` and push *before* you
work, because the push is the lock; a branch lands on `main` only with
Brandon's approval, and `main` (what UAT deploys from) is Brandon's alone;
an agent touches `main` and its own branch and never another's; a ticket is
not done until validated on UAT, and never deleted.
