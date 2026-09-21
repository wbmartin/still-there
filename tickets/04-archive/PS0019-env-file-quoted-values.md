---
id: PS0019
title: ReadEnvFile keeps surrounding quotes; APP_NAME reaches the TOTP issuer with quotes
status: in-review
blocked_by:
waiting_on:
priority: medium
domain: api
raised: 2026-09-21
raised_by: PS0006 triage of doc/000-TODO.md
asked:
claimed: 2026-09-21 by TKT-PS0019-env-file-quoted-values
merged: 67bb62c
deployed: none
validated: 2026-09-21
---
# PS0019 — Env file parser and quoted values

`app_lib.ReadEnvFile` split each line on the first `=` and trimmed
whitespace only, so `APP_NAME="Proto App"` (every committed env file)
left `Config.AppName` as the string `"Proto App"` including the quotes,
which `MFASetupHandler` then passed straight into the TOTP issuer —
authenticator apps showed `"Proto App"`. Fixed by stripping one matching
pair of surrounding quotes in the parser rather than adopting
`joho/godotenv`, keeping the "simple env file format" contract in
`doc/905` and avoiding `${VAR}` expansion in a file that holds secrets.

## What changed
- `config.ReadEnvFile` (`src/api-srv/internal/config/config.go`): each
  value passed through `unquote` — one matching pair of surrounding
  `"…"` or `'…'` removed; an unmatched or lone quote is left as content.
  No `export`, no `${VAR}` expansion, no escapes added.
- `cmd/migrate` shares the same parser and inherits the fix.
- `internal/config/config_test.go` (new): double/single quotes, unquoted
  value, URL with `?`, inner `=`, empty, `""`, unterminated quote, padded
  whitespace, comments skipped.

## Rules that survived
- The env-file parser strips one matching pair of surrounding quotes and
  nothing else — no variable expansion, no escapes — because a file that
  holds secrets should not gain an expansion foot-gun for the sake of a
  cosmetic quoting complaint.
- Every consumer of a shared low-level parser (here, both the API server
  and `cmd/migrate`) must be checked for what an env-value bug quietly
  fed them before considering the fix complete.

## Figures at close
| | date |
|---|---:|
| go tests | 106 (2026-09-21) |
| e2e (Playwright) | 3 (2026-09-21) |
| pytest | 63 (2026-09-21) |

## Security
Config-parsing change touching every secret the app loads. Verified
byte-identical parsed values for `DATABASE_URL`, `JWT_SECRET`,
`TOTP_ENCRYPTION_KEY` before and after, since none are quoted in any
committed env file or `SERVERONLY` placeholder; a host file that had
quoted a secret previously fed the literal quotes into it and now gets
the intended value, so the UAT env file needed a
`grep '^JWT_SECRET="'` check before deploy (if quoted there, existing
sessions would invalidate on deploy). No new parsing surface: only a
matching outer quote pair is stripped.

## Left open
- None — deploy needs a Go binary rebuild only; existing enrolments keep
  their old (quoted) issuer string, which is harmless.

## Sources
- `tickets/04-archive/engineering-log/260921_1627-TKT-PS0019-env-file-quoted-values.md`
- git: `67bb62c`
