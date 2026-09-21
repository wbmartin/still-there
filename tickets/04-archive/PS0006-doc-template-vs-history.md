---
id: PS0006
title: Move ProtoStack history out of doc/ into the engineering log; fix every broken path
status: ready
blocked_by:
waiting_on:
priority: high
domain: docs
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed: 2026-09-21 by TKT-PS0006-doc-template-vs-history
merged: 5a38afa
deployed: none
validated: 2026-09-21
---
# PS0006 — doc/ mixes template content with ProtoStack's own history

Template review found `doc/` full of ProtoStack's own build history (code
reviews, a passkey plan, a stale TODO) mixed in with template reference
docs, plus ~15 broken paths pointing at pre-repo-reorg locations
(`sql/010-…`, `download-shrmpl.sh`, a `note` entity that never existed).
Brandon: "We need to clean all this up, maybe archive to an engineering
log." History moved to `tickets/04-archive/engineering-log/` as dated
entries; `doc/` is now template-only.

## What changed
- `doc/000-TODO.md` triaged with Brandon: 4 new tickets (PS0016–PS0019),
  2 items folded into PS0012/PS0014, 3 dropped, 9 "thought complete"
  items confirmed in code and deleted.
- 10 history files moved into `tickets/04-archive/engineering-log/` with
  dated names (from `git log --follow`) and a two-line "archived,
  point-in-time" header; `doc/archive/` deleted.
- `README.md`, `src/api-srv/README.md`, `src/web-app/README.md`,
  `tests/README.md` regenerated from the actual code (`main.go` routes,
  `config.go` keys, `ls`), replacing content that described a different
  app (wrong entity name, wrong auth routes, wrong crypto, wrong paths).
- Every stale path in `doc/105`, `doc/810` fixed; `doc/810` carries a
  `<!-- rewrite: PS0010 -->` marker for its content (paths only fixed
  here).

## Rules that survived
- `doc/` holds template reference material only; ProtoStack's own build
  narrative lives in `tickets/04-archive/engineering-log/`, dated, never
  compacted.
- An archived doc gets a two-line header stating it is point-in-time, so
  a reader does not mistake a since-fixed finding for an open one.
- Package READMEs are regenerated from code (routes, config keys, tree),
  not maintained by hand — this ticket found them describing a different
  app.

## Figures at close
| | date |
|---|---:|
| doc/ file count after cleanup | 11 |
| history files archived | 10 |
| new tickets filed from TODO triage | 4 (PS0016–PS0019) |
| lint-tickets.py | clean, 19 tickets |

## Security
Docs-only change; no code, config or endpoints touched. Archived reviews
still name real findings (TOTP-seed exposure, fixed by PS0002; AppHelper
pattern, kept by design per PS0009) — the point-in-time header prevents
misreading them as open. Triage surfaced one security-adjacent bug
(PS0019: `APP_NAME="Proto App"` reaches the TOTP issuer with literal
quotes, cosmetic) and one dead code path (`tests/conftest.py` posts to a
nonexistent `/auth2`, noted for PS0014). No secrets or `/Users/` paths
remain in `doc/` or the regenerated READMEs.

## Left open
- PS0016 (JSONB + UUID on example), PS0017 (passkey second factor),
  PS0018 (profile page), PS0019 (.env quoted-value parser bug) — filed
  from the TODO triage.
- `doc/810` content rewrite — PS0010.
- `tests/conftest.py` dead `/auth2` branch — PS0014 note.

## Sources
- `tickets/04-archive/engineering-log/260921_1542-TKT-PS0006-doc-template-vs-history.md`
- git: `5a38afa`
