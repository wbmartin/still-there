---
id: PS0006
title: Split doc/ into template content vs ProtoStack history; fix every broken path
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
# PS0006 — doc/ mixes template content with ProtoStack's own history

**Source:** template review 2026-09-21.

## Problem

A derived app clones this repo and inherits Brandon's TODO list, two
Dec-2025 code-review snapshots, two unimplemented feature plans and an
orphaned archive, with nothing marking them as history. Separately, the
docs that *are* template content have stale paths from before the
`sql/ → src/api-srv/sql/` move (PS0001).

### History living in the live tree

| File | What it is | Status |
|---|---|---|
| `doc/000-TODO.md` | unstructured second backlog, no PS numbers, never closed | contains resolved items (e.g. "hardcoded API proxy URL in vite.config.ts" — already reads `VITE_API_PROXY_URL`); also argues for slog and against vault (PS0003) |
| `doc/110-api-server-code-review.md` | point-in-time review, 2025-12-24 | flagged AppHelper as critical (still open → PS0009); flagged TOTP seed handling (→ PS0002) |
| `doc/215-react-web-code-review.md` | point-in-time review, Dec 2025 | partly stale |
| `doc/120-passkey-plan.md` | WebAuthn plan | no passkey code exists |
| `doc/001-file-storage-implementation-prompt.md` | an OpenCode CLI prompt | no file-download handler exists; misfiled in the 00x setup band |
| `doc/archive/{100,101,102,210}-*.md` | pre-ticket-system specs | never linked from anywhere (`grep -rn doc/archive` → nothing); name collides with `tickets/04-archive/` |

### Broken or stale references (verified)

| File | Says | Should be |
|---|---|---|
| `README.md` project-structure | root `sql/010-initialcreate.sql`, `sql/015-initialdata.sql` | `src/api-srv/sql/migrations/0001_initial_schema.sql`, `src/api-srv/sql/fixtures/015-initialdata.sql`; also add `tickets/`, `infra/`, `scripts/`, `.claude/` |
| `src/api-srv/README.md` | describes a different app: `note` entity, `/auth2`, RS256 (code is HS256, `auth.go:145`), snake_case JSON (real: camelCase), `?ids=` (real: `exampleIdList`), `bin/010-download-shrmpl.sh`, `sql/010-…`, `go run cmd/server/main.go` with no config arg (`main.go:20` requires one), Go 1.21+ (go.mod says 1.25.4), no `KV_BACKEND`/`TOTP_ENCRYPTION_KEY`/`FAILED_LOGIN_*` | regenerate from `cmd/server/main.go` routes + `config.go` keys, or delete and point at `doc/105` |
| `src/web-app/README.md` | omits hooks/, schemas/, tests/, e2e, the entity-store pattern | rewrite after PS0012 |
| `tests/README.md` | needs TLS + kv/log services, `TEST_BASE_URL=https://local-atticus-snipper.teamtaggle.com:8443`, dir `integration-tests/`, report `integration_test_report.html` | reality: `http://localhost:8080`, `tests/`, `bin/200 py` on 8082, memory KV — rewrite after PS0014 decides pytest's fate |
| `doc/060-dev-env-start.md` | `./bin/010-download-shrmpl.sh` | no `.sh` |
| `doc/810-create-new-entity-type-notes.md` | `sql/010-initialcreate.sql`, `sql/update_admin_grants.sql`, unprefixed `internal/model/…`, `schema biz`/`appcore`, `group_global_writer`, `ReadEntityList` (code uses `Retrieve*`) | rewrite against `example_*` + `migrations/0002_*.sql` — content owned by PS0010, location by this ticket |
| `doc/105-api-srv-tech-spec.md` L83, 307-308 | `sql/010-inititial-create.sql` (misspelled), `sql/015-initialdata.sql` | `src/api-srv/sql/…` |
| `doc/902-project-commands.md` | `bin/410-regenerate-vault-certs` | `bin/410-regenerate-vault-mtls-certs` |
| `doc/850-mtls-troubleshooting.md` | in the 8xx how-to band | ops (3xx) — or leaves with the vault (PS0003) |

`doc/050-dev-env-init.md`'s mention of the old `sql/010` is deliberate
historical framing, not a broken link.

## Proposed target

```
doc/            template-generic only: 050, 060, 105, 810, 811, 900, 905 (+ what PS0005 keeps),
                310, 320, 902, and a new 000-deriving-a-new-app.md (PS0007)
doc/history/    (or delete outright) 000-TODO.md, 110, 215, 120, 001, archive/*
tickets/        the ONE backlog — anything in 000-TODO.md still worth doing becomes a ticket
```

Rename `doc/archive/` so it no longer collides with `tickets/04-archive/`
in meaning.

## What to do

1. Triage `doc/000-TODO.md`: each live item → a ticket or dropped; then
   delete the file.
2. Move or delete the history set. If kept, one `doc/history/README.md`
   line: "ProtoStack's own engineering record; not template content."
3. Fix every path in the table above. Where the correct content depends
   on another ticket (810 → PS0010, tests/README → PS0014, web README →
   PS0012), fix the paths now and leave a `<!-- rewrite: PSxxxx -->` marker.
4. Regenerate the README.md documentation index to match.

## Acceptance

- A script or grep over every `*.md` for `sql/0`, `.sh`, `bin/4`,
  `doc/archive` finds no stale reference.
- `doc/000-TODO.md` gone; its live items are tickets.
- README.md structure block matches `ls`.

## Deploy needs

None. `deployed: none`.
