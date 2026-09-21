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

**Source:** template review 2026-09-21.
**Decision (Brandon, 2026-09-21):** "We need to clean all this up, maybe
archive to an engineering log." So the history goes to
`tickets/04-archive/engineering-log/` as dated entries (the existing
convention: one file per entry, `YYMMDD_hhmm-<slug>.md`, never
compacted), not to a `doc/history/` folder. `doc/` becomes
template-only.

## History to archive

| File | Becomes | Note |
|---|---|---|
| `doc/110-api-server-code-review.md` | `engineering-log/251224_0000-api-server-code-review.md` | flagged AppHelper (→ PS0009) and TOTP seed handling (→ PS0002); still useful as record |
| `doc/215-react-web-code-review.md` | `engineering-log/2512xx_0000-react-web-code-review.md` | take the date from `git log --follow` |
| `doc/120-passkey-plan.md` | `engineering-log/<date>-passkey-plan.md` | no passkey code exists; if it is still wanted, also file a PS ticket that points at the entry |
| `doc/001-file-storage-implementation-prompt.md` | `engineering-log/<date>-file-storage-prompt.md` | same: ticket if wanted |
| `doc/archive/100-Golang-app-server.md`, `101/102-appserverquestions.md`, `210-React-web-app-idea.md` | `engineering-log/2511xx_0000-<slug>.md` | orphaned (never linked); dates from git |
| `doc/000-TODO.md` | **triage, then delete** | each live item → a PS ticket or dropped in the commit message; resolved items (e.g. "hardcoded API proxy URL in vite.config.ts" — already `VITE_API_PROXY_URL`) noted as such; the slog/vault arguments are now PS0003 |
| `doc/811-non-entity-notes.md` | `engineering-log/…` | shrmpl client notes; PS0003 sunsets the clients |
| `doc/850-mtls-troubleshooting.md` | `engineering-log/…` **if** PS0003 sunsets the vault; otherwise renumber to `3xx` ops | |

`doc/archive/` is then empty — delete the folder; the name collided with
`tickets/04-archive/` anyway. Each archived entry gets a two-line header
("Archived from `doc/NNN-…` on 2026-09-21 by PS0006; point-in-time, not
maintained.") so a reader knows its status.

## Broken or stale references to fix (verified 2026-09-21)

| File | Says | Should be |
|---|---|---|
| `README.md` project-structure | root `sql/010-initialcreate.sql`, `sql/015-initialdata.sql` | `src/api-srv/sql/migrations/0001_initial_schema.sql`, `src/api-srv/sql/fixtures/015-initialdata.sql`; add `tickets/`, `infra/`, `scripts/`, `.claude/`; drop the shrmpl lines per PS0003 |
| `src/api-srv/README.md` | a different app: `note` entity, `/auth2`, RS256 (code is HS256 `auth.go:145`), snake_case JSON (real camelCase), `?ids=` (real `exampleIdList`), `bin/010-download-shrmpl.sh`, `sql/010-…`, `go run cmd/server/main.go` with no config arg (`main.go:20` requires one), Go 1.21+ (go.mod 1.25.4), no `TOTP_ENCRYPTION_KEY`/`FAILED_LOGIN_*` | regenerate from `cmd/server/main.go` routes + `config.go` keys, or cut to a pointer at `doc/105` |
| `src/web-app/README.md` | omits hooks/, schemas/, tests/, e2e, the entity-store pattern | fix paths now; add a paragraph stating that pages are deliberate copy-paste scaffolding (PS0012) |
| `tests/README.md` | needs TLS + kv/log services, `TEST_BASE_URL=https://local-atticus-snipper.teamtaggle.com:8443`, dir `integration-tests/`, report `integration_test_report.html` | reality: `http://localhost:8080`, `tests/`, `bin/200 py` on 8082, in-process KV — rewrite (PS0014 keeps pytest) |
| `doc/060-dev-env-start.md` | `./bin/010-download-shrmpl.sh` | gone with PS0003 |
| `doc/810-create-new-entity-type-notes.md` | `sql/010-initialcreate.sql`, `sql/update_admin_grants.sql`, unprefixed `internal/model/…`, `schema biz`/`appcore`, `group_global_writer`, `ReadEntityList` (code uses `Retrieve*`) | fix the paths now; leave `<!-- rewrite: PS0010 -->` for the content |
| `doc/105-api-srv-tech-spec.md` L83, 307-308 | `sql/010-inititial-create.sql` (misspelled), `sql/015-initialdata.sql` | `src/api-srv/sql/…` |
| `doc/902-project-commands.md` | `bin/410-regenerate-vault-certs` | `bin/410-regenerate-vault-mtls-certs`, or gone with the vault |

`doc/050-dev-env-init.md`'s mention of old `sql/010` is deliberate
historical framing, not a broken link.

## What to do

1. Triage `000-TODO.md` with Brandon — **he wants to start this in the
   next round**; the agent opens the session by walking the file with
   him, files tickets, deletes it.
2. `git mv` the history files into the engineering log with dated names
   and the two-line header; delete `doc/archive/`.
3. Fix every path in the table; leave rewrite markers where content is
   owned elsewhere.
4. Regenerate the README.md documentation index; it should list only
   what is in `doc/`.
5. Run `python3 scripts/lint-tickets.py` (it walks `04-archive/`).

## What changed and why

Branch `TKT-PS0006-doc-template-vs-history`, 3 commits, plus one ticket
commit on `main`.

**1. `000-TODO.md` triage (walked with Brandon 2026-09-21).** Every item
verified against code first.

| Item | Outcome |
|---|---|
| JSONB + UUID on `example` | **PS0016** (low) |
| Passkey second factor (`doc/120`) | **PS0017** (low), points at the archived plan |
| Profile page: preferences + avatar | **PS0018** (low), avatar scoped out pending storage |
| `.env` parser and quoted values | **PS0019** (medium) — verified bug: `APP_NAME="Proto App"` reaches the TOTP issuer with literal quotes; both fixes (strip quotes / `godotenv`) recorded, Brandon decides at pickup |
| User-list page number in URL | folded into **PS0012** |
| HTTP-level handler unit tests | folded into **PS0014** |
| Monitoring service, nginx file serving, https-vs-http load test | **dropped** (Brandon) |
| "Thought to be complete" block | all nine confirmed done in code; deleted |
| Routes / inline styles / forms library | already PS0013 / PS0012 |
| Starting prompt, code-review prompts | preserved in this ticket's log entry |

**2. History → `tickets/04-archive/engineering-log/`**, ten files, dated
from `git log --follow`, each with the two-line "Archived from … by
PS0006" header (replacing PS0003's note on 811/850): `110`, `215`, `120`,
`001`, `archive/100`, `101`, `102`, `210`, `811`, `850`. `doc/archive/`
deleted. `doc/` is now template-only (11 files).

**3. Paths.** `README.md` structure block rewritten to match `ls`
(`src/api-srv/sql/`, `infra/`, `tickets/`, `scripts/`, `.claude/`);
doc index lists exactly `ls doc/*.md` plus `infra/README.md`, with a
pointer to the engineering log. `doc/105` line 81 and every `doc/810`
path fixed (`Read*` → `Retrieve*`, no `biz`/`appcore`, migrations not
`update_*.sql`); `810` carries `<!-- rewrite: PS0010 -->`. `doc/060`
and `doc/902` were already fixed by PS0003.

**4. Package READMEs regenerated from code** (subagent, verified by diffing
against `main.go` routes, `config.go` keys, `ls bin`, `ls src/web-app`):
`src/api-srv/README.md` (was a different app), `src/web-app/README.md`
(real directories, the `createEntityStore` pattern, the copy-paste
scaffolding paragraph for PS0012), `tests/README.md` (plain http, `tests/`,
`bin/200 py`, real fixtures and files).

**Found, not fixed (→ PS0014 note):** `tests/conftest.py`
`TestClient.authenticate()` posts to `/auth2`, which does not exist; dead
because the fixture users are `mfa_exempt`.

## Files modified

- `README.md`, `src/api-srv/README.md`, `src/web-app/README.md`, `tests/README.md`
- `doc/105-api-srv-tech-spec.md`, `doc/810-create-new-entity-type-notes.md`
- deleted `doc/000-TODO.md`, `doc/archive/`; moved 10 files into `tickets/04-archive/engineering-log/`
- on `main`: `tickets/01-open/PS0016`–`PS0019` (new), `PS0012`, `PS0014` (notes)

## Endpoints introduced or modified

None.

## Acceptance (met 2026-09-21)

- [x] `grep -rn 'sql/0\|download-shrmpl.sh\|regenerate-vault-certs\|doc/archive' --include=*.md .` → only `tickets/` (ticket text) and `doc/050`'s deliberate historical line.
- [x] `doc/000-TODO.md`, `doc/archive/` gone.
- [x] README structure block matches `ls`; doc index == `ls doc/*.md`.
- [x] `python3 scripts/lint-tickets.py` clean (19 tickets).

## Deploy needs

None. `deployed: none`.
