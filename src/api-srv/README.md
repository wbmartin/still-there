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

- `SERVER_BIND_ADDRESS` — empty listens on every interface
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

All JSON is camelCase. List endpoints take `?<entity>IdList=1,2,3`
(comma-separated) and `?afterUpdateTs=<ts>` for incremental reads.

### Authentication
- `POST /auth` — username/password login
- `GET /refresh` — refresh session
- `POST /changeSecurityContext`
- `POST /changeOwnPassword`
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
2. `CheckRateLimitAndUserBlock` — per-user/unauthenticated rate limits and
   admin blocks (`/admin/blockUser`), backed by the in-process KV
3. `CheckCSRF` — `X-CSRF-Token` header must match the last 3 characters of
   the session ID; skipped for GET/HEAD/OPTIONS/TRACE
4. `CheckAuthorization(grant)` — caller must hold the named grant for
   their current security context
5. `CheckAllChecksComplete` — every handler calls this before doing any
   work

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

## In-Process Infrastructure

No external services: `DATABASE_URL`, `JWT_SECRET`, and
`TOTP_ENCRYPTION_KEY` are inline in the env file.

- **`Logger`** (`internal/logging/logging.go`): thin wrapper over
  `log/slog`, writing one logfmt line per call to stderr with a `code=`
  attribute.
- **`KV`** (`internal/cache/kv.go`): in-process key-value store for
  rate limits, login lockouts, MFA temp tokens, and user blocks.
  Process-local — restarting the server clears it. Graduate to a shared
  store (Redis or similar) once the app runs behind a load balancer with
  2+ API processes.

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
