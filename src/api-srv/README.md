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
Keys read by `internal/config/config.go` `LoadConfig`:

- `DATABASE_URL`, `JWT_SECRET`, `SERVER_NAME`, `APP_NAME` — required
- `SERVER_PORT` — required, integer
- `ENABLE_TLS` — required, boolean; when true, `TLS_CERT_PATH` and
  `TLS_KEY_PATH` are required
- `TOTP_ENCRYPTION_KEY` — required, 32 bytes as hex (AES-256)
- `FAILED_LOGIN_ATTEMPT_THRESHOLD` — required, integer
- `FAILED_LOGIN_ATTEMPT_WINDOW_DURATION`, `FAILED_LOGIN_LOCKOUT_DURATION` —
  required
- `JWT_SESSION_EXPIRE_HOURS`, `MAX_USER_REQ_PER_MIN`,
  `MAX_UNAUTH_REQ_PER_MIN` — optional, integer
- `SERVER_BIND_ADDRESS` — optional; empty listens on every interface
- `CORS_ALLOWED_ORIGINS` — optional, comma-separated; empty means no CORS

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
- `POST /mfaSetup`, `POST /mfaVerify`
- `POST /admin/resetUserMFA`

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
- `POST /error-report` — no auth

## Security

Auth is an HS256 JWT carried in an `auth` cookie
(`internal/auth/jwt.go`); claims hold user ID, session ID, security
context, and a grants list. Passwords are Argon2id.

There is no security middleware chain — each handler calls its checks
explicitly, in order, via the `AppHelper` in
`internal/security/app_helper.go`:

1. `CheckJWT` — parse and validate the cookie
2. `CheckRateLimitAndUserBlock` — per-user/unauthenticated rate limits and
   admin blocks, backed by the in-process KV
3. `CheckCSRF` — `X-CSRF-Token` header must match the last 3 characters of
   the session ID; skipped for GET/HEAD/OPTIONS/TRACE
4. `CheckAuthorization(grant)` — caller must hold the named grant for
   their current security context
5. `CheckAllChecksComplete` — every handler calls this before doing any
   work

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
internal/repository/  # data access
sql/migrations/    # schema, applied by cmd/migrate
sql/roles.sql       # database roles
sql/fixtures/       # seed data
```

See `doc/105-api-srv-tech-spec.md` for more depth.
