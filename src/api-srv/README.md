# API Server

Go stdlib `net/http` + `pgx` API server. Local dev runs plain HTTP
(`ENABLE_TLS=false`); TLS is available via `ENABLE_TLS`/`TLS_CERT_PATH`/
`TLS_KEY_PATH` for deployed environments.

## Quick Start

1. Build the local database (from the repo root):
   ```bash
   bin/510-reset-test-db
   ```
2. Run the server:
   ```bash
   bin/215-run-api-srv
   ```

The binary takes the env file path as its single argument
(`bin/215-run-api-srv` passes `etc/loc-api-srv.env`); run it with no
argument and it exits with "Configuration file path is required".

## Configuration

`etc/loc-api-srv.env` is self-contained and needs no edits for local dev.
Keys read by `internal/config/config.go` `LoadConfig`; a missing required
key or a bad value stops start-up with an error naming the key. File
grammar (quoting, comments, no `$VAR`) is in `doc/905-config-guidance.md`.

Required:

- `DATABASE_URL`, `JWT_SECRET`, `SERVER_NAME`, `APP_NAME`
- `SERVER_PORT` — positive integer
- `SERVER_BIND_ADDRESS` — the interface to listen on: `127.0.0.1` behind
  nginx, in dev and in the test harness; `0.0.0.0` for every interface
- `ENABLE_TLS` — boolean; when true, `TLS_CERT_PATH` and `TLS_KEY_PATH` are
  required
- `TOTP_ENCRYPTION_KEY` — 32 bytes as hex (AES-256)
- `JWT_SESSION_EXPIRE_HOURS`, `MAX_USER_REQ_PER_MIN`,
  `MAX_UNAUTH_REQ_PER_MIN`, `FAILED_LOGIN_ATTEMPT_THRESHOLD`,
  `INVITE_TTL_HOURS` (how long an invite link works) — positive integers
- `FAILED_LOGIN_ATTEMPT_WINDOW_DURATION`, `FAILED_LOGIN_LOCKOUT_DURATION` —
  KV durations above 0: `30s`, `5min`, `1h`
- `LOG_LEVEL` — one of `debug`, `info`, `warn`, `error`; lines below it are
  dropped. Per-request lines (`APP_HELPER_INIT`, `RLP0`) and health polls
  are debug; deployed files set `info`, local dev `debug`

Optional:

- `TRUSTED_PROXY_COUNT` — non-negative integer, default 0. How
  many proxies append to `X-Forwarded-For`; the client IP (rate limit and
  login lockout keys) is the entry that many hops from the right. 0 ignores
  the header (local dev, Playwright); 1 behind the host's nginx; the pytest
  copy `bin/300-run-tests` builds sets 1
- `CORS_ALLOWED_ORIGINS` — comma-separated; empty means no CORS

`TEST_BASE_URL` is not part of `Config` — the server ignores it. It is read
by `bin/330-test-load` and exported by the test runners for the pytest
suite in `tests/`.

## API Endpoints

Every route `cmd/server/main.go` registers. The full table (checks,
grant per route, request bodies) is in `doc/105-api-srv-tech-spec.md`
(Endpoints); `internal/handler/route_check_contract_test.go` fails when a
route is added without a row there.

All JSON is camelCase. List endpoints take `?<entity>IdList=1,2,3`
(comma-separated) and `?afterUpdateTs=<ts>` for incremental reads. A
password is always sent as `password`; group membership bodies are
`{userIdList, appUserGroupId, updateTsList}`.

### Authentication
- `POST /auth` — username/password login
- `GET /refresh` — refresh session
- `POST /changeSecurityContext`
- `POST /changeOwnPassword`
- `GET /me`, `PUT /me` — the caller's own row (profile + preferences;
  preferences travel only here, never on `/userList`)
- `POST /logoff`
- `POST /userSelfCreate` — public self-registration
- `POST /mfaSetup`, `POST /mfaVerify` — `/auth`'s `mfaEnrolled` sends an
  enrolled user to verify; `/mfaSetup` refuses them 409 and never shows the
  stored secret again
- `POST /admin/resetUserMFA`
- `POST /admin/blockUser`, `POST /admin/unblockUser` — body `{userName}`;
  block sets `account_status = blocked` (`/auth` refuses it), ends the
  login's sessions and sets the `BLOCKED*` KV flag, reloaded at start
- `POST /userInvite`, `POST /userInviteReissue`, `POST /userInviteRevoke`
  — an invited login (no password, cannot sign in) and its one-time link;
  the token is answered once, only its SHA-256 stored
- `POST /inviteCheck`, `POST /inviteRedeem` — public: who a link is for;
  set the password from it (login becomes active). Rate limit plus the
  `/auth` lockout per address and per token; one 404 for any dead link

### Example (CRUD)
- `GET /example?exampleId={id}`
- `POST /example`
- `PUT /example`
- `DELETE /example?exampleId={id}&updateTs={ts}`
- `GET /exampleList?exampleIdList=1,2,3&afterUpdateTs=<ts>`
- `POST /exampleList`, `PUT /exampleList` — all-or-nothing transaction
- `DELETE /exampleList?exampleIdList=1,2&updateTs=<ts1>,<ts2>`

### Users
- `GET /userList?appUserIdList=1,2,3&afterUpdateTs=<ts>`
- `POST /user`, `PUT /user`, `DELETE /user`
- `POST /changeOtherUserPassword`
- `POST /addUserListToGroup`, `POST /removeUserListFromGroup`
- `GET /grantList`

### User groups
- `GET /appUserGroupList`
- `POST /appUserGroup`, `PUT /appUserGroup`, `DELETE /appUserGroup`

### Other
- `GET /health` — no auth
- `POST /error-report` — signed-in (LOGON); logs the caller's client error

## Security

Auth is an HS256 JWT carried in an `auth` cookie
(`internal/auth/jwt.go`); claims hold user ID, session ID, security
context, and a grants list. Passwords are Argon2id.

There is no security middleware chain — each handler calls its checks
explicitly, in order, via the `AppHelper` in
`internal/security/app_helper.go`:

1. `CheckJWT` — parse and validate the cookie
2. `CheckRateLimitAndUserBlock` — per-minute rate limit (key
   `RPM*<user name>*<minute>`, or `RPM*ip:<client IP>*<minute>` when
   anonymous) and the `BLOCKED*` flag that
   `/admin/blockUser` sets, backed by the in-process KV
3. `CheckCSRF` — `X-CSRF-Token` header must match the last 3 characters of
   the session ID; skipped for GET/HEAD/OPTIONS/TRACE. A mismatch is logged
   without either token
4. `CheckAuthorization(grant)` — caller must hold the named grant for
   their current security context
5. `CheckAllChecksComplete` — every handler calls this before doing any
   work

**Fail closed.** `CheckAllChecksComplete` answers 500 when any of the five
was neither checked nor skipped, and a KV error inside a check (session
revocation, rate counter, block flag) is 500, never a pass. The login
lockout counters and the TOTP replay key follow the same rule.

**Exceptions** are written in the handler with `Skip*` calls, each with a
reason: `/health` and `/auth` skip all five; `/refresh`, `/logoff` and
`/changeSecurityContext` skip CSRF and the grant; the public
`/userSelfCreate`, `/inviteCheck`, `/inviteRedeem` run only the
per-address rate limit (the invite routes then the lockout); `/mfaSetup` and `/mfaVerify` accept a temp token
in place of a session. The table with the reason for each is in doc/105
(Security Checks, Exceptions). The `admin` subject skips the rate limit
and the block flag (`app_helper.go`), stated there too.

**Lockout.** `/auth` counts failures per client IP and per user name
(unknown names included, against a dummy hash so timing does not tell);
`FAILED_LOGIN_ATTEMPT_THRESHOLD` failures within
`FAILED_LOGIN_ATTEMPT_WINDOW_DURATION` lock both for
`FAILED_LOGIN_LOCKOUT_DURATION` (429). A wrong TOTP code and a dead
invite link count toward the same lockout.

**MFA.** A login that is not `mfaExempt` gets a temp token from `/auth`,
not a session. `/mfaSetup` enrols (TOTP secret stored AES-GCM encrypted
under `TOTP_ENCRYPTION_KEY`; refused 409 once enrolled), `/mfaVerify`
checks the code (each code good once) and issues the session. No
response carries `pw_hash` or `totp_seed`: both are `json:"-"` in
`internal/model/app_user_model.go`.

A handler reads its body through `decodeJSONBody` / `readBody`
(`internal/handler/common.go`), which caps it before the first byte is
decoded: 413 over the cap, 400 when malformed. Caps per route are in
`doc/105-api-srv-tech-spec.md` (Request Body Limits).

Entities are `example`, `appUser`, `appUserGroup`. Grants are centralized
in `internal/model/grants.go` (e.g. `GrantExampleRead`, `GrantUserRead`,
`GrantUserGroupRead`).

## Database

Go 1.25 (see `go.mod`). Schema changes are migrations only:
`sql/migrations/NNNN_name.sql`, embedded and applied by `cmd/migrate` —
never hand-run DDL. Also `sql/roles.sql` (the role the migrator and app
connect as) and `sql/fixtures/` (seed data, e.g. `local-seed.sql`).

## Testing

`bin/300-run-tests go` runs gofmt, `go vet`, staticcheck and
`go test -v ./...` with `TEST_DATABASE_URL` naming the scratch database
(doc/902). Repository suites connect through `connectTestDB`
(`internal/repository/testdb_test.go`), which refuses a non-scratch
database and skips without one; the gate fails any skip, so a green run
includes them. They act as `testTenant` (2) and prove isolation against
`otherTenant` (3). How a test proves what it claims (negative
assertions, OS probes, anchored detectors, no skips, `TestLive` tests):
`tests/README.md`, "Writing tests".

## Dependencies

No change runs a blanket `go get -u ./...`. A change with a security
reason to touch modules works in this order, in each module
(`src/api-srv`, `src/load-test`):

1. `govulncheck ./...`; a called vulnerability is the target
   (`bin/300-run-tests all` runs it and fails on one).
2. `go get <module>@<fixed version>` for the module it names, then
   `go mod tidy`; commit `go.mod` and `go.sum` with it.
3. Go toolchain and other module upgrades are their own change.

The web app follows the same order (`src/web-app/README.md`,
"Dependency cadence").

## In-Process Infrastructure

No external services: `DATABASE_URL`, `JWT_SECRET`, and
`TOTP_ENCRYPTION_KEY` are inline in the env file.

- **`Logger`** (`internal/logging/logging.go`): thin wrapper over
  `log/slog`, writing one logfmt line per call to stderr with a `code=`
  attribute.
- **`KV`** (`internal/cache/kv.go`): in-process key-value store for
  rate limits, login lockouts, MFA temp tokens and TOTP replay keys,
  session revocations, and user blocks. Process-local — restarting the
  server clears it (a revoked token is good again until it expires; block
  flags are reloaded from the database at start). So the API runs as one
  process: a second one needs a shared store (Redis or similar) first.

## Architecture

```
cmd/server/       # entry point, routing (main.go)
cmd/migrate/       # schema migrator, embeds sql/migrations
internal/app/      # Services: the config, store and logger every handler gets
internal/auth/     # JWT, argon2 passwords, TOTP + seed encryption, RBAC
internal/cache/    # in-process key-value store (rate limits, lockouts, MFA tokens)
internal/config/   # env-file loader, Config
internal/logging/  # slog wrapper with the code= attribute
internal/security/ # AppHelper (explicit per-handler checks), client IP, headers/CORS
internal/handler/  # HTTP handlers, one file per entity/concern
internal/model/    # models, grants, DTOs
internal/repository/  # data access; every write and its zero-row check carry the tenant
sql/migrations/    # schema, applied by cmd/migrate
sql/roles.sql       # database roles
sql/fixtures/       # seed data
```

Repository errors are sentinels (`internal/repository/errors.go`:
`ErrNotFound`, `ErrDeleted`, `ErrConflict`, `ErrDuplicate`, `ErrInUse`,
`ErrValidation`, `ErrExpectationFailed`), returned bare or wrapped with
`%w`. Handlers answer them with `mapRepoError` (`internal/handler/common.go`):
400 / 404 / 409 / 417, anything else 500 with the error text logged only. A
soft-deleted row answers 404. Repositories call pgx once; there is no retry
wrapper (pgxpool reconnects on its own).

See `doc/105-api-srv-tech-spec.md` for more depth.
