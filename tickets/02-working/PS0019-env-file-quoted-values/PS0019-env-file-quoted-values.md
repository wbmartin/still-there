---
id: PS0019
title: ReadEnvFile keeps surrounding quotes; APP_NAME reaches the TOTP issuer with quotes
status: ready
blocked_by:
waiting_on:
priority: medium
domain: api
raised: 2026-09-21
raised_by: PS0006 triage of doc/000-TODO.md
asked:
claimed: 2026-09-21 by TKT-PS0019-env-file-quoted-values
merged:
deployed:
validated:
---
# PS0019 — Env file parser and quoted values

**Source:** `doc/000-TODO.md` ("the hand-rolled .env parser is fragile …
`joho/godotenv` would be more robust"), triaged 2026-09-21 by PS0006.
Brandon: "I think it is dumb, but I keep hearing this complaint" — filed
with both options; decide at pickup.

## Bug

`app_lib.ReadEnvFile` (`config.go`) splits on the first `=` and trims
whitespace, nothing else. Every committed env file has
`APP_NAME="Proto App"`, so `Config.AppName` is the string `"Proto App"`
*including the quotes*, and `MFASetupHandler` passes it to
`generateTOTPUri` as the issuer — authenticator apps show `"Proto App"`.
Verified 2026-09-21.

## Options

1. **Strip matching surrounding quotes** in `ReadEnvFile` (`"…"` or
   `'…'`), ~5 lines, no dependency, keeps the `doc/905` "simple env file
   format" contract. `cmd/migrate` shares the parser and benefits.
2. **`github.com/joho/godotenv`** — the de-facto Go env-file parser
   (there is no stdlib one). Handles quotes, `export`, trailing `#`
   comments, `\n` escapes, `${VAR}` expansion. One dependency; expansion
   is a behaviour we do not want in a file that holds secrets, so it would
   be `godotenv.Read` with expansion avoided, or `godotenv.Unmarshal`.

Recommendation: 1. The only real-world complaint is quotes, and the
parser is 25 lines that `cmd/migrate` also uses.

## Acceptance

- `APP_NAME="Proto App"` → `Config.AppName == "Proto App"`; a unit test
  covers `"…"`, `'…'`, and an unquoted value with an inner `=`.
- TOTP issuer shows without quotes on a fresh MFA enrolment.

## Deploy needs

Go binary. Existing enrolments keep their old issuer string; harmless.
