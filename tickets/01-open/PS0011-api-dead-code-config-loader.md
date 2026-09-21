---
id: PS0011
title: API dead code, duplicated helpers, table-driven config loader, lint baseline
status: blocked
blocked_by:
waiting_on: Brandon
priority: low
domain: api
raised: 2026-09-21
raised_by: template review 2026-09-21
asked:
claimed:
merged:
deployed:
validated:
---
# PS0011 — API leftovers from earlier iterations

**Source:** template review 2026-09-21. Baseline: `go vet`, `go build`,
`go test ./...` clean; `golangci-lint run` (repo `.golangci.yml`) reports
59 issues: revive 31, lll 11, errcheck 5, goconst 4, goimports 3, unused 3,
ineffassign 2. `grep TODO|FIXME|XXX|HACK` → 0, but `rbac.go:66` carries
"WBM 2025.12.11 I don't think is used".

Best done after PS0008/PS0009/PS0010 so the diffs do not collide; pure
deletions can go any time.

## Dead code (delete)

| Where | What |
|---|---|
| `repository/scoped_query.go:54-96` | `withLiveParent`, `buildParentScopedWhere` — no callers |
| `handler/auth_handler.go:130` | `resetFailedAttempts` |
| `app_lib/rbac.go:66-83` | `HasAnyGrantInContext`, `ValidateGrantID` (self-flagged) |
| `app_lib/app_helper.go:249-258` | `GetContext`, `CanProceed` |
| `model/security_model.go` | `Auth2RequestModel`, `LogoffRequest`, `BlockUserRequest`, `ResetUserPasswordRequest`, `AddUserRequest`, `ChangeCurrentUserPasswordRequest`, `ChangeSecurityContextResponseWrapper` — defined only |
| `model/user_grant_model.go` (+ its test) | no `user_grant` table exists; snake_case JSON tags. Keep only if a table is planned |
| `model/grants.go` | `UserGrants`, `ExampleGrants` (PS0010 owns the catalog reshape) |
| `user_repository.go:67` | orphan doc comment |
| `app_lib/services.go` | `DBPool` field (PS0008) |

## Duplicated helpers (collapse to one)

- `CurrentTimestamp()` in `handler/common.go:154` **and**
  `repository/user_repository.go:18` with different formulas; inline
  `float64(time.Now().UnixNano())/1e9` at `user_repository.go:505`. →
  one `Now()` in a shared package.
- `getClientIP` in `app_helper.go` and `auth_handler.go:24-42` (PS0002/
  PS0009).
- `"*KEY NOT FOUND*"` sentinel string ×6 — a constant or a typed error.
- `formatTOTPLabel` compiles a regex per call (`user_mfa_handler.go:66`).
- `min` at `logging_client.go:292` shadows the Go 1.21+ builtin.
- `error_report_handler.go:78` unchecked `w.Write`.
- `LogoffHandler` comment says "No rate limiting" beside a rate-limit call
  (`auth_handler.go:724`).
- Receiver names mix `sh`/`ah`.

## Config loader

`app_lib/config.go:200-375` parses each key by hand, then validates in a
second block; required-ness for `FAILED_LOGIN_*` is checked twice
(`:283-293` and `:364-374`). Adding a key touches three places.
`doc/000-TODO.md` already flags the hand-rolled `.env` parser.
→ Table-driven spec `{key, required, kind, dest}` or struct tags; one
pass; one error listing every missing key by name (the `bin/` scripts
already follow "missing keys die by name"). Both binaries must accept the
same flag (`cmd/server/main.go:20-26` takes positional `os.Args[1]`;
`cmd/migrate/main.go:33-36` takes `-config`/`-url`).

## go.mod

All direct deps are imported. Heavy for purpose: `cenkalti/backoff`
(PS0010 removes), `skip2/go-qrcode` + `boombuler/barcode` for one QR at
`user_mfa_handler.go:436` — consider returning the otpauth URI and
rendering in the web app.

## Acceptance

- `golangci-lint run` ≤ 10 issues, all `lll`, or the config tightened
  with reasons.
- `unused` and `ineffassign` at 0.
- Both binaries: `-config <file>`.
- Tests for the config loader (there are none today for `config.go`).

## Deploy needs

Go binary. If the server binary's CLI changes from positional to
`-config`, update `bin/215`, `bin/610`, `infra/*/etc/*.service` in the
same PR.

## Direction (Brandon, 2026-09-21)

**Hold.** "It may be hard to separate leftovers from anticipated
scenarios." Do not claim until PS0010 is settled (PS0009 is decided); then walk the
dead-code table with Brandon item by item before deleting. The
`security_model.go` DTOs and `user_grant_model.go` in particular may be
placeholders for planned endpoints/tables.

## Deferred (Brandon, 2026-09-21)

**Deferred.** Brandon wants to wait on this. Do not claim, do not walk the
items, until he re-opens it. `status: blocked`, `waiting_on: Brandon` — blocked on his decision, not
on another ticket.
