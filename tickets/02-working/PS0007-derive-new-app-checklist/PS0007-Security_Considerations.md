# PS0007 — Security Considerations

Branch `TKT-PS0007-derive-new-app-checklist`. Profiles: **8** (config,
scripts, deploy) and a sliver of **4** (JWT issuer).

## 4. JWT issuer

`iss` is now `Config.AppName` instead of the compiled-in `proto-go-app`.
Nothing in `ValidateJWT` checks `iss`, so this is a label change: tokens
minted before and after the deploy both validate. If a derived app ever
adds issuer validation it should compare against the same config value.

## 8. Config, scripts, secrets

**`bin/000-rebrand` rewrites every tracked text file.** Its guards: refuses
on a dirty tree (the result is one reviewable diff); operates only on
`git ls-files` (never `node_modules`, `dist`, `.git`, untracked local
files such as a real `etc/uat-api-srv.env`); skips `tickets/`; ordered,
word-bounded patterns so `ProtoStack`, `protocol` and `X-Forwarded-Proto`
survive; then `go build` + `tsc`. Reviewed output from two dry runs in a
worktree. It edits `infra/*/etc/*.env` — the *redacted* copies; a host's
filled files are not in git and are not touched, which is why
`DERIVING.md` step 3 says the host side is manual.

**Secrets are the point of `DERIVING.md` §2.** Nothing in the template
forces a derived app to change `admin123!`, `reallygoodPSQLpassword`, or
the local `JWT_SECRET`/`TOTP_ENCRYPTION_KEY`; the rebrand deliberately does
not generate new ones (a script that silently rotates passwords across
six files and the fixtures the test suites log in with would break the
first `bin/200` run and hide what changed). The checklist lists every file
each value lives in so the rotation is one deliberate pass. This remains
the derived app's responsibility.

**Local database rename.** `ALTER DATABASE loc_starter_db RENAME TO
loc_proto_db` on Brandon's docker Postgres; two scratch DBs dropped. No
data lost; the dev DB's contents are unchanged under the new name.

**Harness variables renamed** (`PROTO_*` → `TEST_*`/`E2E_*`). Same values,
same files, same `root:shrmpl-hook 0640` guidance on the infra copy.
`TEST_DATABASE_URL` still carries the local superuser password inside
`etc/loc-test.env`, as before.

**Web scaffold files deleted** (`App.css`, `vite.svg`): dead assets, no
exposure change. `package.json` name change has no runtime effect.

## What did I miss?

- The rebrand's bare `\bproto\b` pass also hits prose in `doc/` — wanted:
  the docs describe the app by its names. `ProtoStack` (the template's own
  name, provenance) is intentionally untouched.
- `--ticket-prefix` rewrites `PS0023`-style ids in skills and `CLAUDE.md`
  but not in `tickets/`; the linter is skipped that run and says why.
- `bin/216` still `pkill -f vite` for any vite on the machine — unchanged
  from 211, noted in PS0003.
