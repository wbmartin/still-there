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

## What changed and why

Option 1 (Brandon, 2026-09-21: "start 19" on the recommendation).
`config.ReadEnvFile` now runs each value through `unquote`: one pair of
matching surrounding `"…"` or `'…'` is removed; an unmatched or lone
quote is content. Nothing else changes — no `export`, no `${VAR}`
expansion, no escapes; `doc/905`'s "simple env file format" stands.
`cmd/migrate` shares the parser and inherits the fix.

`internal/config/config_test.go` (new) covers double/single quotes, an
unquoted value, a URL with `?`, an inner `=`, empty, `""`, an
unterminated quote, padded whitespace, and that comments are skipped.

Proven end to end: `LoadConfig("etc/loc-api-srv.env").AppName` is
`Proto App`; `auth.GenerateTOTPURI` now yields `…&issuer=Proto+App`
(was `%22Proto+App%22`).

## Files modified

- `src/api-srv/internal/config/config.go`
- `src/api-srv/internal/config/config_test.go` (new)

## Endpoints introduced or modified

None. `POST /mfaSetup` returns an otpauth URI whose issuer no longer
carries quotes; the JWT `iss` claim (`Config.AppName`, PS0007) likewise.

## Acceptance (met 2026-09-21)

- [x] `APP_NAME="Proto App"` → `Config.AppName == "Proto App"`; unit test covers `"…"`, `'…'`, unquoted with inner `=`.
- [x] TOTP issuer without quotes on a fresh enrolment — URI proven above; `bin/200-run-tests all` go=106 e2e=3 py=63.

## Deploy needs

Go binary. Existing enrolments keep their old issuer string; harmless.

## Waiting for

The next UAT deploy.
