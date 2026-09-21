# 2026-09-21 — PS0007 single rename point

`etc/app.env` + `bin/000-rebrand` + `DERIVING.md`. The one departure from
the ticket text: `PROTO_*` harness variables cannot be slug-derived (Go
tests read them literally), so they were renamed once to neutral
`TEST_*`/`E2E_*` names and are now out of the derived app's way.

The rebrand is ordered perl substitutions with the bare slug last and
word boundaries doing the safety work; two things needed explicit rules
because `_` is a word character: `_proto_user`/`_proto_db` and
`proto_api_server`. Proved in a worktree with `acme`: 85 files, 30
renames, builds, only the migrator lock key left — which is kept because
one migrator per cluster is the right default. Second run is a no-op.

Renamed Brandon's `loc_starter_db` to `loc_proto_db` so the template's
own names follow the slug; flagged in the ticket.
