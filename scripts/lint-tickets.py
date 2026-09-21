#!/usr/bin/env python3
"""Check the YAML frontmatter on every ticket. The convention is the
`ticket-workflow` skill; `tickets/README.md` orients.

No third-party dependencies on purpose: this repo has no CI, no Makefile and no
virtualenv story for tooling, so a linter that needs `pip install` is a linter
nobody runs. The frontmatter grammar is deliberately restricted to `key: value`
and `key: [a, b]` so that hand-parsing it here is safe rather than fragile.

    scripts/lint-tickets.py            # check, exit 1 on any error
    scripts/lint-tickets.py --report   # print the counts
    scripts/lint-tickets.py --update-status   # write them into tickets/STATUS.md

tickets/STATUS.md may be read by the customer as well as by us, so the
generated block is tables and counts with no working notes: the notes for an
agent are here and in the ticket-workflow and ticket-triage skills. Above the
BEGIN GENERATED marker is a short hand-written preamble; keep it to what the
page is and who each section is for. Dated findings go in the ticket's
working folder, never in STATUS.md, and stale_prose_ids() rejects a preamble
that names an archived or non-existent ticket. Rerun --update-status after
any ticket change.
"""

import re
import subprocess
import sys
import time
from collections import Counter, defaultdict
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
TICKETS = REPO / "tickets"
# 01-open/     PS####-slug.md             filed, unclaimed
# 02-working/  PS####-slug/PS####-slug.md claimed; security review and log notes
#                                         live beside it; stays here through
#                                         merge and UAT deploy until VALIDATED
# 03-done/     PS####-slug/               validated; branch deleted; awaiting compaction
# 04-archive/  PS####-slug.md             compacted, one file; engineering-log/ beside it
#
# The stage name is the logical key everywhere in this script and in the
# ticket's _lifecycle; DIR maps it to the numbered directory.
LIFECYCLE = ("open", "working", "done", "archive")
DIR = {"open": "01-open", "working": "02-working", "done": "03-done", "archive": "04-archive"}
FOLDER_STAGES = ("working", "done")

# The integration branch. It is also what UAT deploys from, and is advanced
# only with Brandon's approval; claims, merges and every count here are
# measured against origin/<INTEGRATION>.
INTEGRATION = "main"
ORIGIN = f"origin/{INTEGRATION}"

STATUS = ("ready", "blocked", "waiting-on-customer", "in-review", "needs-triage")
PRIORITY = ("high", "medium", "low", "investigate")
DOMAIN = (
    "api", "web", "calc", "data", "seed", "deploy", "docs", "security",
    "tooling", "infrastructure",
)

# Present on every ticket. Empty is allowed except where noted below.
FIELDS = (
    "id", "title", "status", "blocked_by", "waiting_on", "priority",
    "domain", "raised", "raised_by", "asked", "claimed",
    "merged", "deployed", "validated",
)
REQUIRED_NONEMPTY = ("id", "title", "status", "domain", "raised")

# Allowed, but not required -- adding these to FIELDS would make every one of
# the existing tickets "missing a field".
OPTIONAL_FIELDS = ("recipient",)

# Who actually answers an Ask-Customer question. This is what makes the list
# sendable: a batch is one message to one person, not a drip.
#
# The domains of this app do not map to people at the customer the way a
# business subject area would, so there is no domain -> recipient table: a
# question goes to "Customer" unless `recipient:` names a role there (IT,
# Operations, Stakeholder).
RECIPIENT = ("Customer", "Stakeholder", "IT", "Operations")


def recipient_of(ticket):
    return ticket.get("recipient") or "Customer"


ID_RE = re.compile(r"^PS\d{4}$")


def num(tid):
    """The numeric part of an id. 'PS1020' -> 1020."""
    return int(tid[2:])


def is_ask(tid):
    """Ask-Customer tickets are the 1000+ range. Every id carries the same
    prefix, so the number is what tells the two ranges apart -- anything
    keying off the prefix here is a bug."""
    return num(tid) >= 1000


def parse_frontmatter(text, errors, name):
    """Parse the restricted subset. Returns {} and records an error on anything else."""
    if not text.startswith("---\n"):
        errors.append(f"{name}: no YAML frontmatter (must open with '---')")
        return {}
    end = text.find("\n---\n", 3)
    if end == -1:
        errors.append(f"{name}: frontmatter is not closed with '---'")
        return {}

    fields = {}
    for lineno, line in enumerate(text[4:end].split("\n"), start=2):
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        if ":" not in line:
            errors.append(f"{name}:{lineno}: not 'key: value' -> {line!r}")
            continue
        key, _, raw = line.partition(":")
        key, raw = key.strip(), raw.strip()
        if key in fields:
            errors.append(f"{name}:{lineno}: duplicate key {key!r}")
        if raw.startswith("["):
            if not raw.endswith("]"):
                errors.append(f"{name}:{lineno}: unterminated list -> {raw!r}")
                continue
            fields[key] = [v.strip() for v in raw[1:-1].split(",") if v.strip()]
        else:
            fields[key] = raw
    return fields


def check(path, lifecycle, all_ids, errors):
    name = str(path.relative_to(REPO))
    fields = parse_frontmatter(path.read_text(), errors, name)
    if not fields:
        return None

    unknown = set(fields) - set(FIELDS) - set(OPTIONAL_FIELDS)
    if unknown:
        errors.append(f"{name}: unknown field(s) {sorted(unknown)}")
    for field in FIELDS:
        if field not in fields:
            errors.append(f"{name}: missing field {field!r}")
    for field in REQUIRED_NONEMPTY:
        if not fields.get(field):
            errors.append(f"{name}: {field!r} must not be empty")

    # The id is the whole point of the header: it is what makes a collision
    # visible when two sessions take the same number.
    tid = fields.get("id", "")
    if tid and not ID_RE.match(tid):
        errors.append(f"{name}: id {tid!r} is not PS followed by four digits")
    if tid and not path.name.startswith(tid + "-"):
        errors.append(f"{name}: id {tid!r} does not match the filename")

    status = fields.get("status", "")
    if status and status not in STATUS:
        errors.append(f"{name}: status {status!r} not in {list(STATUS)}")
    priority = fields.get("priority", "")
    if priority and priority not in PRIORITY:
        errors.append(f"{name}: priority {priority!r} not in {list(PRIORITY)}")
    domain = fields.get("domain", "")
    if domain and domain not in DOMAIN:
        errors.append(f"{name}: domain {domain!r} not in {list(DOMAIN)}")
    who = fields.get("recipient", "")
    if who and who not in RECIPIENT:
        errors.append(f"{name}: recipient {who!r} not in {list(RECIPIENT)}")
    # Guarded on ID_RE: a malformed id is already reported above, and num()
    # would raise on it rather than add a second message.
    if who and ID_RE.match(tid) and not is_ask(tid):
        errors.append(f"{name}: recipient is for Ask-Customer tickets, not engineering")

    blocked_by = fields.get("blocked_by") or []
    if isinstance(blocked_by, str):
        errors.append(f"{name}: blocked_by must be a list, e.g. [PS1003]")
        blocked_by = []
    if blocked_by and status != "blocked":
        errors.append(f"{name}: blocked_by is set but status is {status!r}, not 'blocked'")
    if status == "blocked" and not blocked_by and not fields.get("waiting_on"):
        errors.append(f"{name}: status is 'blocked' with no blocked_by and no waiting_on")

    # Lifecycle lives in the directory; the header records the dates that
    # earned each move. `validated` is what separates 02-working/ from 03-done/ --
    # a ticket waits in 02-working/ through merge and UAT deploy until its
    # check has passed.
    if lifecycle in FOLDER_STAGES and not fields.get("claimed"):
        errors.append(f"{name}: in {DIR[lifecycle]}/ but 'claimed' is empty")
    if lifecycle == "open" and fields.get("claimed"):
        errors.append(f"{name}: 'claimed' is set but the ticket is in {DIR['open']}/")
    if lifecycle in ("done", "archive") and not fields.get("validated"):
        errors.append(f"{name}: in {DIR[lifecycle]}/ but 'validated' is empty")
    if lifecycle in ("open", "working") and fields.get("validated"):
        errors.append(f"{name}: 'validated' is set but the ticket is in {DIR[lifecycle]}/")

    fields["_lifecycle"] = lifecycle
    fields["_path"] = name
    all_ids[tid].append(name)
    return fields


def git(*args):
    """Run a git command, or return None if it fails for any reason."""
    try:
        done = subprocess.run(
            ("git",) + args, cwd=REPO, capture_output=True, text=True, timeout=10
        )
    except (OSError, subprocess.SubprocessError):
        return None
    return done.stdout.strip() if done.returncode == 0 else None


def remote_branches():
    """Branch names on the remote, and whether the answer is authoritative.

    `git branch -r` lists remote-tracking refs in *this clone*, and a plain
    `git fetch` never removes the ref for a branch someone deleted elsewhere --
    only `--prune` does. A check that reports phantom problems teaches people
    to ignore it, so ask the remote directly and say so when we cannot.
    """
    out = git("ls-remote", "--heads", "origin")
    if out is not None:
        return {line.split("refs/heads/")[-1] for line in out.splitlines() if line}, True
    local = git("branch", "-r") or ""
    return {l.strip().split("/", 1)[-1] for l in local.splitlines()}, False


BRANCH_ID = re.compile(r"\bPS\d{4}\b")


def branch_ticket_id(name):
    """The ticket id a branch names, or None if it names none.

    Branches are `TKT-<id>-<description>`. Match the id itself rather than the
    whole shape, so a branch with an odd description still reports.
    """
    if not name.startswith("TKT-"):
        return None
    found = BRANCH_ID.search(name)
    return found.group(0) if found else None


def merged_into_main(branch):
    """True if origin/<branch> is fully contained in origin/main."""
    return git("merge-base", "--is-ancestor", f"origin/{branch}", ORIGIN) is not None


def sync_warnings(claim_state):
    """A number is only claimed once origin has it.

    Agents run on several machines, so a commit sitting in one local clone is
    invisible to all the others: a ticket committed and not pushed is not filed,
    and a number claimed and not pushed is not taken. Warn rather than fail:
    commit -> lint -> push is a legitimate order and this has to stay readable
    in the middle of it.
    """
    if git("rev-parse", "--verify", ORIGIN) is None:
        return [f"no {ORIGIN} ref -- cannot tell whether these numbers are taken"]

    warnings = []
    unpushed = git("rev-list", "--count", f"{ORIGIN}..HEAD", "--", "tickets/")
    if unpushed and unpushed != "0":
        warnings.append(
            f"{unpushed} commit(s) touching tickets/ are not on {ORIGIN}. Until "
            "you push, those numbers are NOT claimed and another machine can take them."
        )

    branches, authoritative = remote_branches()
    if not authoritative:
        warnings.append(
            "could not reach origin -- branch checks below use local "
            "remote-tracking refs, which may name branches that no longer exist."
        )
    for name in sorted(branches):
        tid = branch_ticket_id(name)
        if tid is None:
            continue
        where = claim_state.get(tid)
        if where is None:
            continue  # the ticket is long gone, or not filed yet
        if where == "open":
            warnings.append(
                f"branch {name} exists but {tid} is still in {DIR['open']}/ -- work in "
                "flight with no claim, which is how numbers get taken twice."
            )
        elif where in ("done", "archive"):
            # The session that validated the ticket deletes its own branch as
            # part of the move to 03-done/. Nobody else does: a remote branch is
            # how Brandon sees which workspaces are on which tickets, and an
            # agent touches only main and its own TKT- branch.
            merged = f" (merged into {ORIGIN})" if authoritative and merged_into_main(name) else ""
            warnings.append(
                f"{tid} is in {DIR[where]}/ but {name} is still on origin{merged} -- "
                f"its owner deletes it at validation; no other session may."
            )
    branched = {branch_ticket_id(n) for n in branches}
    for tid, where in claim_state.items():
        if where == "working" and tid not in branched:
            warnings.append(f"{tid} is claimed in {DIR['working']}/ but has no branch on origin.")

    behind = git("rev-list", "--count", f"HEAD..{ORIGIN}")
    if behind and behind != "0":
        warnings.append(
            f"local branch is {behind} commit(s) behind {ORIGIN} -- merge before "
            "trusting any count or next-free number above."
        )

    fetch_head = REPO / ".git" / "FETCH_HEAD"
    if not fetch_head.exists():
        warnings.append("no record of a fetch in this clone -- run `git fetch` first.")
    else:
        hours = (time.time() - fetch_head.stat().st_mtime) / 3600
        if hours > 1:
            warnings.append(
                f"origin was last fetched {hours:.0f}h ago -- run `git fetch`; the "
                "comparisons above are only as fresh as that."
            )
    return warnings


def ids_in_use(tickets):
    """Every ticket number that is taken.

    Some tickets may be documented as legacy `doc/PS####-slug/` folders with
    no ticket file, left over from before `tickets/` existed. Those folders
    are migrated into tickets/ by the compact-tickets skill; until every one
    is gone, a folder still counts so a number is never handed out twice.
    """
    return {t["id"] for t in tickets} | {
        d.name.split("-")[0] for d in (REPO / "doc").glob("PS*")
        if d.is_dir() and ID_RE.match(d.name.split("-")[0])
    }


def next_free(in_use):
    """(engineering, ask-customer) -- the next unused number in each range."""
    taken = [num(i) for i in in_use if not is_ask(i)]
    asked = [num(i) for i in in_use if is_ask(i)]
    return max(taken, default=0) + 1, max(asked, default=1000) + 1


def main():
    errors, all_ids, tickets = [], defaultdict(list), []

    for lifecycle in LIFECYCLE:
        directory = TICKETS / DIR[lifecycle]
        if not directory.is_dir():
            errors.append(f"tickets/{DIR[lifecycle]}/ is missing")
            continue
        if lifecycle in FOLDER_STAGES:
            # One folder per ticket, the ticket file named like the folder.
            # Anything else in the folder is the ticket's working papers.
            for stray in sorted(directory.glob("PS*.md")):
                errors.append(f"tickets/{DIR[lifecycle]}/{stray.name}: {DIR[lifecycle]}/ holds "
                              f"folders, not files -- move it into {stray.stem}/")
            for folder in sorted(d for d in directory.iterdir() if d.is_dir()):
                path = folder / (folder.name + ".md")
                if not path.is_file():
                    errors.append(f"tickets/{DIR[lifecycle]}/{folder.name}/: no "
                                  f"{folder.name}.md inside it")
                    continue
                got = check(path, lifecycle, all_ids, errors)
                if got:
                    tickets.append(got)
            continue
        for path in sorted(directory.glob("*.md")):
            if path.name in ("README.md", "STATUS.md"):
                continue
            got = check(path, lifecycle, all_ids, errors)
            if got:
                tickets.append(got)

    # A legacy doc/PS*/ folder makes a number taken (see ids_in_use) but is NOT
    # a competing claim on it: the duplicate check is about ticket FILES in two
    # lifecycle folders, which is the collision the claim lock exists to make
    # visible.
    for tid, where in sorted(all_ids.items()):
        if len(where) > 1:
            errors.append(f"id {tid!r} is claimed by {len(where)} places: {', '.join(where)}")

    known = ids_in_use(tickets)
    for ticket in tickets:
        for dep in ticket.get("blocked_by") or []:
            if dep not in known:
                errors.append(f"{ticket['_path']}: blocked_by {dep!r} does not exist")

    errors += stale_prose_ids(tickets)

    if "--report" in sys.argv:
        report(tickets)
    if "--update-status" in sys.argv and not errors:
        update_status(tickets)

    for warning in sync_warnings({t['id']: t['_lifecycle'] for t in tickets}):
        print(f"  sync: {warning}", file=sys.stderr)

    if errors:
        print(f"\n{len(errors)} problem(s):", file=sys.stderr)
        for error in errors:
            print(f"  {error}", file=sys.stderr)
        return 1
    print(f"{len(tickets)} tickets, no problems.")
    return 0


BEGIN = "<!-- BEGIN GENERATED -- scripts/lint-tickets.py --update-status -->"
END = "<!-- END GENERATED -->"

PREAMBLE = """# Ticket status

Where every ticket stands, generated from the ticket headers. *Questions for
the customer* is for the customer; the engineering sections are for whoever
picks up the next piece of work. Do not edit below the marker.
"""


def unblocks(live):
    """Reverse `blocked_by`: {blocker id -> [ids it would free]}.

    `blocked_by` already records which customer questions gate engineering
    work; reversed, it says which questions pay for themselves the moment they
    are answered. Derived rather than asserted, so unlike a hand-written
    shortlist it cannot go stale.
    """
    index = defaultdict(list)
    for t in live:
        for dep in t.get("blocked_by") or []:
            index[dep].append(t["id"])
    return index


def is_blocked(t):
    return bool(t.get("blocked_by")) or t.get("status") == "blocked"


def waiting_on(t):
    if t.get("blocked_by"):
        return ", ".join(t["blocked_by"])
    return t.get("waiting_on") or "unstated"


def status_block(tickets, stamp):
    """Answer the only two questions asked of the ticket set:

        1. which tickets go in front of the client
        2. which can be started right now

    Everything else is reference. Ordered accordingly.
    """
    live = [t for t in tickets if t["_lifecycle"] not in ("archive", "done")]
    ask = [t for t in live if is_ask(t["id"])]
    eng = [t for t in tickets if not is_ask(t["id"]) and t["_lifecycle"] != "archive"]
    order = {p: i for i, p in enumerate(PRIORITY)}
    by_priority = lambda t: (order.get(t.get("priority"), 99), t["id"])
    freed = unblocks(live)
    out = [BEGIN, "", f"*Generated {stamp} by `scripts/lint-tickets.py --update-status`.*", ""]

    # ---------------------------------------------------------- 1. the client
    out += [f"## Questions for the customer — {len(ask)} open", ""]

    payers = sorted((t for t in ask if freed.get(t["id"])),
                    key=lambda t: (-len(freed[t["id"]]), t["id"]))
    if payers:
        out += [f"### When answered, will unlock engineering work ({len(payers)})", "",
                "| Ask | Question | Frees | Recipient | Asked |", "|---|---|---|---|---|"]
        for t in payers:
            out.append(f"| **{t['id']}** | {t['title']} | {', '.join(freed[t['id']])} "
                       f"| {recipient_of(t)} | {t.get('asked') or '—'} |")
        out.append("")

    chases = [t for t in ask if t.get("asked")]
    if chases:
        out += [f"### Asked, awaiting an answer ({len(chases)})", "",
                "| Ask | Question | Recipient | Asked |", "|---|---|---|---|"]
        for t in sorted(chases, key=lambda t: t["id"]):
            out.append(f"| **{t['id']}** | {t['title']} | {recipient_of(t)} "
                       f"| {t['asked']} |")
        out.append("")

    rest = [t for t in ask if not freed.get(t["id"]) and not t.get("asked")]
    if rest:
        out += [f"### Not yet asked, by who can answer ({len(rest)})", ""]
        grouped = defaultdict(list)
        for t in rest:
            grouped[recipient_of(t)].append(t)
        for who in RECIPIENT:
            group = sorted(grouped.get(who, []), key=lambda t: t["id"])
            if not group:
                continue
            out += [f"**{who} — {len(group)}**", "", "| Ask | Question |", "|---|---|"]
            for t in group:
                out.append(f"| **{t['id']}** | {t['title']} |")
            out.append("")

    # ------------------------------------------------------------ 2. the work
    # 02-working/ is a claim, and a claim is what stops two sessions taking the
    # same ticket. Offering a claimed ticket as "ready to start" would hand
    # someone the collision the lock exists to prevent.
    working = [t for t in eng if t["_lifecycle"] == "working"]
    claimed = sorted((t for t in working if not t.get("merged")), key=by_priority)
    awaiting = sorted((t for t in working if t.get("merged")), key=by_priority)
    done = sorted((t for t in eng if t["_lifecycle"] == "done"), key=lambda t: t["id"])
    ready = sorted((t for t in eng if t["_lifecycle"] == "open" and not is_blocked(t)),
                   key=by_priority)
    out += [f"## Engineering — ready to start ({len(ready)})", "",
            "| Ticket | Title | Priority | Domain |", "|---|---|---|---|"]
    for t in ready:
        out.append(f"| **{t['id']}** | {t['title']} | {t.get('priority') or '—'} "
                   f"| {t['domain']} |")
    out.append("")

    if claimed:
        out += [f"## Engineering — in progress ({len(claimed)})", "",
                "| Ticket | Title | Claimed |", "|---|---|---|"]
        for t in claimed:
            out.append(f"| **{t['id']}** | {t['title']} | {t.get('claimed') or '—'} |")
        out.append("")

    if awaiting:
        out += [f"## Engineering — merged, awaiting validation ({len(awaiting)})", "",
                "| Ticket | Title | Merged | Deployed |", "|---|---|---|---|"]
        for t in awaiting:
            out.append(f"| **{t['id']}** | {t['title']} | {t['merged']} "
                       f"| {t.get('deployed') or '—'} |")
        out.append("")

    if done:
        out += [f"## Engineering — validated, awaiting archive ({len(done)})", "",
                "| Ticket | Title | Validated |", "|---|---|---|"]
        for t in done:
            out.append(f"| **{t['id']}** | {t['title']} | {t['validated']} |")
        out.append("")

    stuck = sorted((t for t in eng
                    if t["_lifecycle"] == "open" and is_blocked(t)), key=by_priority)
    if stuck:
        out += [f"## Engineering — blocked ({len(stuck)})", "",
                "| Ticket | Title | Priority | Waiting on |", "|---|---|---|---|"]
        for t in stuck:
            out.append(f"| **{t['id']}** | {t['title']} | {t.get('priority') or '—'} "
                       f"| {waiting_on(t)} |")
        out.append("")

    # ------------------------------------------------------------ 3. reference
    out += ["## Open tickets by domain", "", "| Domain | Open |", "|---|---|"]
    for domain, count in Counter(t["domain"] for t in live).most_common():
        out.append(f"| {domain} | {count} |")

    eng_next, ask_next = next_free(ids_in_use(tickets))
    out += ["", f"**Next free number:** `PS{eng_next:04d}` (engineering), "
            f"`PS{ask_next}` (Ask Customer).", "", END]
    return "\n".join(out)


def update_status(tickets):
    path = TICKETS / "STATUS.md"
    text = path.read_text() if path.is_file() else PREAMBLE
    stamp = time.strftime("%Y-%m-%d %H:%M %Z")
    block = status_block(tickets, stamp)
    if BEGIN in text and END in text:
        head, rest = text.split(BEGIN, 1)
        text = head + block + rest.split(END, 1)[1]
    else:
        text = text.rstrip() + "\n\n" + block + "\n"
    path.write_text(text)
    print(f"tickets/STATUS.md updated ({stamp})")


def report(tickets):
    """The same two answers STATUS.md gives, for a terminal."""
    live = [t for t in tickets if t["_lifecycle"] not in ("archive", "done")]
    ask = [t for t in live if is_ask(t["id"])]
    eng = [t for t in tickets if not is_ask(t["id"]) and t["_lifecycle"] != "archive"]
    order = {p: i for i, p in enumerate(PRIORITY)}
    by_priority = lambda t: (order.get(t.get("priority"), 99), t["id"])
    freed = unblocks(live)

    payers = sorted((t for t in ask if freed.get(t["id"])),
                    key=lambda t: (-len(freed[t["id"]]), t["id"]))
    print(f"\n## Questions for the customer — {len(ask)} open, {len(payers)} unlock work\n")
    for t in payers:
        print(f"  {t['id']}  frees {', '.join(freed[t['id']]):<8} "
              f"{recipient_of(t):<11} {t['title'][:52]}")
    chases = [t for t in ask if t.get("asked")]
    if chases:
        print(f"\n  asked and still waiting: "
              f"{', '.join(sorted(t['id'] for t in chases))}")
    rest = defaultdict(list)
    for t in ask:
        if not freed.get(t["id"]) and not t.get("asked"):
            rest[recipient_of(t)].append(t["id"])
    for who in RECIPIENT:
        if rest.get(who):
            print(f"\n  {who} ({len(rest[who])}): {', '.join(sorted(rest[who]))}")

    ready = sorted((t for t in eng if t["_lifecycle"] == "open" and not is_blocked(t)),
                   key=by_priority)
    print(f"\n## Ready to start — {len(ready)}\n")
    for t in ready:
        print(f"  {t['id']}  {t.get('priority') or '-':<11} {t['title'][:64]}")

    working = [t for t in eng if t["_lifecycle"] == "working"]
    claimed = sorted((t for t in working if not t.get("merged")), key=by_priority)
    if claimed:
        print(f"\n## In progress — {len(claimed)} (claimed, do not start)\n")
        for t in claimed:
            print(f"  {t['id']}  {t.get('claimed') or '-'}")
    awaiting = sorted((t for t in working if t.get("merged")), key=by_priority)
    if awaiting:
        print(f"\n## Merged, awaiting validation — {len(awaiting)}\n")
        for t in awaiting:
            print(f"  {t['id']}  merged {t['merged']:<9} deployed {t.get('deployed') or '-':<11} {t['title'][:40]}")
    done = sorted((t for t in eng if t["_lifecycle"] == "done"), key=lambda t: t["id"])
    if done:
        print(f"\n## Done, awaiting compaction — {len(done)}\n")
        for t in done:
            print(f"  {t['id']}  validated {t['validated']}  {t['title'][:48]}")

    stuck = sorted((t for t in eng
                    if t["_lifecycle"] == "open" and is_blocked(t)), key=by_priority)
    if stuck:
        print(f"\n## Blocked — {len(stuck)}\n")
        for t in stuck:
            print(f"  {t['id']}  waiting on {waiting_on(t):<14} {t['title'][:48]}")

    print("\n## By domain\n")
    for domain, count in Counter(t["domain"] for t in live).most_common():
        print(f"  {domain:<16} {count}")

    eng_next, ask_next = next_free(ids_in_use(tickets))
    print(f"\nNext free PS number: PS{eng_next:04d}")
    print(f"Next free Ask Customer number: PS{ask_next}\n")


def stale_prose_ids(tickets):
    """Reject a hand-written STATUS.md section that names a dead ticket.

    Judgement is worth keeping in that file. A reference to a ticket that no
    longer exists, or has been archived, is not judgement -- it is rot, and it
    is the one kind a script can catch. History belongs in
    tickets/04-archive/engineering-log/.
    """
    path = TICKETS / "STATUS.md"
    if not path.is_file():
        return []
    prose = path.read_text().split(BEGIN, 1)[0]
    live = {t["id"] for t in tickets if t["_lifecycle"] != "archive"}
    archived = {t["id"] for t in tickets if t["_lifecycle"] == "archive"}
    errors = []
    for tid in sorted(set(re.findall(r"\bPS\d{4}\b", prose))):
        if tid in live:
            continue
        where = "is archived" if tid in archived else "does not exist"
        errors.append(
            f"tickets/STATUS.md: hand-written prose names {tid}, which {where}. "
            "Drop the reference -- history goes in tickets/04-archive/engineering-log/."
        )
    return errors


if __name__ == "__main__":
    sys.exit(main())
