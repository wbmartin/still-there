---
id: PS0007
title: Single rename point and a "deriving a new app" checklist
status: in-review
blocked_by:
waiting_on:
priority: high
domain: docs
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0007-derive-new-app-checklist
merged: 9d74ba9
deployed: none
validated: 2026-09-21
---
# PS0007 — There is no "how to derive a new app from this" path

Template review found the identity "proto" spread across ~54 files (310
lines) with no single place to change it, and no documented path for
turning the template into a real app. Delivered a single sourced
identity file, a rebrand script, and an ordered checklist covering
everything a sed cannot reach cleanly (Go module path, hardcoded JWT
issuer, migrator lock key, passwords, web branding).

## What changed
- `etc/app.env` (`APP_SLUG`, `APP_DISPLAY_NAME`): sourced by every
  `bin/` script that derives binary/service/db/container names; JWT
  issuer now reads `Config.AppName` instead of a hardcoded string.
- `bin/000-rebrand <slug> "<Name>" [--module] [--ticket-prefix]`: refuses
  on a dirty tree, ordered word-bounded perl substitutions over
  `git ls-files` only, `go mod edit` + import rewrite, `git mv` of every
  slug-bearing `infra/` file, then `go build` + `tsc` + lint. Idempotent.
- `DERIVING.md` at repo root: clone/rebrand, a table of every file each
  secret lives in (passwords, `JWT_SECRET`, `TOTP_ENCRYPTION_KEY`), both
  deploy paths' names, example-entity/tickets/CLAUDE.md reset, first run.
- `PROTO_*` test-harness env vars renamed once to neutral `TEST_*`/`E2E_*`
  names (Go tests read these literally, so they can't be slug-derived —
  the one departure from the ticket's literal ask).
- `src/web-app/constants/app.ts` (`APP_NAME`, `APP_TITLE`); dead
  `App.css`/`public/vite.svg` deleted.
- Migrator advisory-lock key (`"PROTO"`) kept as a hardcoded constant on
  purpose — one migrator per cluster is the safe default for derived apps
  sharing infrastructure.

## Rules that survived
- One `APP_SLUG`/`APP_DISPLAY_NAME` source (`etc/app.env`) is the only
  place app identity is set; every script and config derives from it.
- A rebrand script only touches `git ls-files`, refuses on a dirty tree,
  and never rotates secrets — secret rotation is the derived app's
  deliberate, documented, manual step (`DERIVING.md`).
- Harness env-var names that Go tests read literally are not
  slug-derived; give them one neutral rename instead.

## Figures at close
| | date |
|---|---:|
| identity occurrences before fix | 310 lines / 54 files |
| rebrand proof run (`acme`, worktree) | 85 files changed, 30 renamed |
| rebrand second run | no-op (idempotent) |
| bin/200-run-tests after rename | go=105 e2e=3 py=63 |
| post-rebrand grep for leftover identity | only the deliberate migrator lock key |

## Endpoints
None. JWT `iss` claim changed from literal `proto-go-app` to
`Config.AppName`; nothing validates `iss`, so existing sessions are
unaffected.

## Security
`bin/000-rebrand` rewrites every tracked text file, so its guards matter:
dirty-tree refusal (one reviewable diff), `git ls-files` scope only
(never `node_modules`/`dist`/untracked host env files), word-bounded
patterns verified not to touch `ProtoStack`/`protocol`/
`X-Forwarded-Proto`, `tickets/` skipped. It deliberately does not rotate
`admin123!`, `reallygoodPSQLpassword`, `JWT_SECRET`, or
`TOTP_ENCRYPTION_KEY` — a script that silently rotated six files' worth
of secrets would break the derived app's first test run and hide what
changed, so `DERIVING.md` lists every file each secret lives in and
rotation stays a deliberate manual pass.

## Left open
- The rebrand's bare `\bproto\b` pass also rewrites prose in `doc/`
  (acceptable — docs should describe the app by its real name).
- `--ticket-prefix` doesn't rewrite ids already inside `tickets/`
  (linter skipped on that proof run, by design).
- `bin/216`'s `pkill -f vite` still kills any vite process on the
  machine — pre-existing, noted under PS0003.

## Sources
- `tickets/04-archive/engineering-log/260921_1603-TKT-PS0007-derive-new-app-checklist.md`
- git: `9d74ba9`
