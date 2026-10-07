---
name: radley-house
description: Work this repo's tickets in the Radley House tracker — read the backlog, file a work item, claim it before starting, post progress, hand back to a human and pick work back up. Use when asked about tickets, work items, the backlog, the House, "#123", what to work on next, or when a change needs a question answered before it can go on. Covers the API calls and the conventions; coordinates and tokens come from the untracked env file.
---

# The Radley House tracker

The House is the system of record for this repo's tickets. A ticket is a
**work item** with a House-assigned number, cited as `#123`.

**This file names no House URL, project id or token.** Every one of those
is a per-repo value and lives in `etc/.radley-house`, which is gitignored
and never committed. That is what lets this skill ship in the template and
be inherited by each derived app unchanged — the app supplies its own
coordinates. A derived app that has not written its own `etc/.radley-house`
has no tracker yet; see `DERIVING.md` section 2.

## Load the coordinates

Every call needs these. Load them from the env file, never from a literal:

```bash
set -a && . ./etc/.radley-house && set +a
H="X-Build-Token: $BUILD_TOKEN"
B="$HOUSE_API_URL/api/projects/$HOUSE_PROJECT_ID"
```

`BUILD_TOKEN` is the project-scoped token and is what every call below
uses. Never echo a token, never paste one into a file, a commit, a
work-item field or a message. If `etc/.radley-house` is missing, stop and
say so — do not guess a URL or an id.

A `401` or `403` means the token is stale or wrong for this project. Say
so and stop; a fresh token is a staff action.

## Read before you write

Lists come back priority-sorted, high to low.

```bash
curl -s -H "$H" "$B/external/work-items"                   # the backlog
curl -s -H "$H" "$B/external/work-items?tag=needs-agent"   # handed back to an agent
curl -s -H "$H" "$B/external/work-items/123"               # one item
```

**Read an item's history before resuming it.** It carries the comments,
questions, answers, progress and attachment activity that explain where
the work stands.

## File a work item

The House assigns the number on creation — **never invent or reserve
one**. The response carries it.

```bash
curl -s -X POST -H "$H" -H "Content-Type: application/json" \
  "$B/external/work-items" \
  -d '{"title":"Short outcome","detail":"Context and approach",
       "acceptance":"Observable finish line","kind":"engineering",
       "priority":"medium"}'
```

A new discovery defaults to `parked`. Set `ready` only when it is
genuinely actionable.

The values the API accepts, as it reports them on a rejected field:

| field | values |
|---|---|
| `kind` | `decision`, `engineering`, `idea`, `investigation` |
| `priority` | `high`, `investigate`, `low`, `medium` |
| `status` | `in_progress`, `parked`, `ready`, `review`, `validating` |

`done` and `dropped` are **not** in that list — the API itself refuses
them, which is how "only staff close an item" is enforced.

## Claim, then work, then post progress

Claim before starting, so no one else picks it up. Branches in this repo
are `feature/<slug>` (`CLAUDE.md`, Branches):

```bash
curl -s -X POST -H "$H" -H "Content-Type: application/json" \
  "$B/external/work-items/123/claim" -d '{"branch":"feature/my-slug"}'

curl -s -X POST -H "$H" -H "Content-Type: application/json" \
  "$B/external/work-items/123/events" \
  -d '{"event":"progress","note":"What changed and how it was checked",
       "reference":"<pr url>","status":"review"}'
```

Release a claim you are not going to finish:

```bash
curl -s -X POST -H "$H" "$B/external/work-items/123/release"
```

## Edit a field, or leave a comment

`PATCH` changes only the fields you send; anything omitted is unchanged. A
`note` on its own is a comment.

```bash
curl -s -X PATCH -H "$H" -H "Content-Type: application/json" \
  "$B/external/work-items/123" \
  -d '{"acceptance":"Covered by an API test","waiting_on":null,
       "note":"Clarified after review"}'
```

`blocked_by: [122]` takes work-item numbers **in this project only**.

## Status, and who may close

The flow: `parked` → `ready` → `in_progress` → `review` → `validating`
→ `done`, or `dropped`.

Move finished agent work to `review` or `validating`, and stop there.
**Only staff close an item `done` or `dropped`** — the API rejects both
from this token, so never try, and never tick a staff-owned item on
someone's behalf.

## Human handoff: exactly one tag

Use **one** of these when the next move belongs to someone else. The
House rejects an item carrying both.

| tag | meaning | how to set it |
|---|---|---|
| `needs-human` | an agent is blocked on a question | record a `question` event holding the question, then `PATCH` the **full resulting tag list** — add `needs-human`, remove `needs-agent` |
| `needs-agent` | the human answered and handed it back | remove it when you resume the work |

Staff's "Answer & send to agent" swaps `needs-human` for `needs-agent`
and records the answer in history, so the answer is in the item — read it
there rather than asking again.

## Attachments

Private to staff and project builders; they never reach a client. 10 MB
cap.

```bash
curl -s -X POST -H "$H" "$B/external/work-items/123/attachments" \
  -F "file=@evidence.png"
curl -s -H "$H" "$B/external/work-items/123/attachments"
curl -s -H "$H" -o evidence.png \
  "$B/external/work-items/123/attachments/<attachment-id>/download"
```

Evidence belongs on the item, not in the repo: a screenshot, a timing, a
log, an API response.

## What stays out of the House, and out of the repo

- **No secret in a work item.** Not in `title`, `detail`, `acceptance`,
  a `note` or an attachment.
- **No customer name** anywhere — person id or role, per `CLAUDE.md`.
  A work item is as permanent as a commit.
- **No House coordinate in a tracked file.** Values live only in
  `etc/.radley-house`.
- Host work keeps its line in `tasks/todo.md` (host, command, check,
  rollback) whether or not it also has a work item — agents never run or
  tick one.
