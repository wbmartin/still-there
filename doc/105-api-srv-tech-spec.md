# API Server Technical Specification

## Overview

This document specifies the ProtoStack-ReactGo API server, a production-ready Golang application server providing secure, multi-tenant CRUD operations with JWT-based authentication, RBAC authorization, and comprehensive security features.

### Architecture

- **Language**: Go 1.25.4 minimum (`go` line); built with go1.25.14
  (`toolchain` line, fetched by `GOTOOLCHAIN=auto`)
- **Framework**: Standard library with pgx for PostgreSQL
- **Architecture**: n-tier
  - **Presentation Layer**: HTTP handlers with JSON APIs
  - **Application Layer**: Business logic and security checks
  - **Domain Layer**: Models and business rules
  - **Infrastructure Layer**: Database repositories and external services
- **External Dependencies**:
  - PostgreSQL database

### Project Structure

```
src/api-srv/
├── cmd/server/main.go              # Application entry point
├── internal/
│   ├── app/services.go             # Services: config, store, logger for every handler
│   ├── auth/                       # jwt.go, password.go (argon2), totp.go, rbac.go
│   ├── cache/kv.go                 # In-process key-value store
│   ├── config/config.go            # Env-file loader, Config
│   ├── logging/logging.go          # slog wrapper, code= attribute
│   ├── security/                   # app_helper.go (per-handler checks), security_headers.go
│   ├── handler/                    # HTTP handlers
│   │   ├── auth_handler.go         # Authentication endpoints
│   │   ├── common.go               # Shared handler utilities
│   │   ├── example_handler.go      # Example CRUD operations
│   │   ├── health_handler.go       # Health checks
│   │   ├── app_user_group_handler.go   # User group CRUD operations
│   │   └── user_handler.go         # User CRUD operations
│   ├── model/                      # Domain models
│   │   ├── app_user_model.go       # User entities
│   │   ├── example_model.go        # Example entities
│   │   ├── example_model_test.go   # Example model tests
│   │   ├── grants.go               # Grant definitions
│   │   ├── security_model.go       # Security-related models
│   │   ├── user_grant_model.go     # RBAC models
│   │   ├── user_grant_model_test.go # User grant model tests
│   │   └── app_user_group_model.go     # User group entities
│   └── repository/                 # Data access layer
│       ├── errors.go               # Sentinel errors, ClientMessageError
│       ├── example_repository.go   # Example data operations
│       ├── example_repository_test.go # Example repository tests
│       ├── app_user_group_repository.go # User group data operations
│       ├── app_user_group_repository_test.go # User group repository tests
│       ├── user_repository.go      # User data operations
│       └── user_repository_test.go # User repository tests
├── .golangci.yml                   # Go linting configuration
├── go.mod
├── go.sum
└── README.md
```

## Database Design

### Schema Guidelines

- **Primary Keys**: `entity_name_id` pattern (e.g., `example_id`, `app_user_id`)
- **Table Names**: Singular entity names
- **Timestamps**: `numeric(17,7)` for high-precision timestamps
- **Audit Fields**: All tables include:
  - `create_ts`, `create_user_id`
  - `update_ts`, `update_user_id` (optimistic locking)
  - `delete_ts`, `delete_user_id` (soft deletes)
- **Multi-tenancy**: `security_context` integer field on all tables
- **Naming**: Snake_case column names
- **Relationships**: No foreign keys; managed in application layer
- **Defaults/Triggers**: Prohibited; all logic in application code

## Database
initial/examples in src/api-srv/sql/migrations/0001_initial_schema.sql (DDL) and src/api-srv/sql/fixtures/local-seed.sql (dev DML, tenant 1) / test-seed.sql (test DML, tenants 2 and 3)

## API Design

### Endpoints

| Method | Path | Auth | CSRF | RBAC Grant | Description |
|--------|------|------|------|------------|-------------|
| GET | /health | No | No | N/A | Basic health check |
| POST | /auth | No | No | N/A | Username/password authentication |
| GET | /refresh | Yes | No | N/A | Refresh JWT token |
| POST | /changeSecurityContext | Yes | No | N/A | Switch security context |
| POST | /logoff | Yes | No | N/A | Clear authentication cookie |
| GET | /me | Yes | No | LOGON | The caller's own user row (profile + preferences) |
| PUT | /me | Yes | Yes | LOGON | Update own firstName/lastName/email/preferenceJsonb, optimistic lock |
| GET | /example | Yes | No | GET_EXAMPLE | Retrieve single example |
| GET | /exampleList | Yes | No | GET_EXAMPLE | Retrieve multiple examples |
| POST | /example | Yes | Yes | ADD_EXAMPLE | Create new example |
| POST | /exampleList | Yes | Yes | ADD_EXAMPLE | Bulk create examples |
| PUT | /example | Yes | Yes | CHG_EXAMPLE | Update example |
| PUT | /exampleList | Yes | Yes | CHG_EXAMPLE | Bulk update examples |
| DELETE | /example | Yes | Yes | DEL_EXAMPLE | Soft delete example |
| DELETE | /exampleList | Yes | Yes | DEL_EXAMPLE | Bulk soft delete examples |
| GET | /userList | Yes | No | GET_USER | Retrieve user list |
| POST | /user | Yes | Yes | ADD_USER | Create new user |
| PUT | /user | Yes | Yes | CHG_USER | Update user |
| DELETE | /user | Yes | Yes | DEL_USER | Delete user |
| POST | /admin/resetUserMFA | Yes | Yes | RESET_MFA_TOKEN | Clear a user's TOTP seed (9020: its own grant, not CHG_USER) |
| POST | /addUserListToGroup | Yes | Yes | CHG_USER_GROUP | Add users to group |
| POST | /removeUserListFromGroup | Yes | Yes | CHG_USER_GROUP | Remove users from group |
| GET | /grantList | Yes | No | GET_USER | Retrieve available grants |
| GET | /userGroupList | Yes | No | GET_USER_GROUP | Retrieve user group list |
| POST | /userGroup | Yes | Yes | ADD_USER_GROUP | Create user group |
| PUT | /userGroup | Yes | Yes | CHG_USER_GROUP | Update user group |
| DELETE | /userGroup | Yes | Yes | DEL_USER_GROUP | Delete user group |

### Request/Response Format

- **Content-Type**: `application/json`
- **Field Naming**: camelCase in JSON, snake_case in database
- **Timestamps**: Unix nanoseconds as float64 (17,7 precision)
- **Error Responses**: Standard HTTP status codes with JSON error messages

### Authentication Flow

1. **Initial Auth** (`/auth`):
   - Input: `{userName: string, password: string}`
   - Verify credentials against `app_user` table
   - Every failure answers the same 401 "Invalid credentials": an unknown
     name runs argon2 against a dummy hash (`auth.CheckPasswordHashOrDummy`)
     so timing does not reveal which names exist, and it counts toward the
     per-IP and per-username lockout exactly like a wrong password. A lookup
     error other than not-found is an outage: 500 "Login is temporarily
     unavailable", not counted. So is a lockout counter that cannot be read
     or written (here and on `/mfaVerify`'s wrong code): the lockout fails
     closed rather than silently stop counting
   - A completed login deletes the per-username counter
     (`loginfail-user-<name>`): at `/auth` for an MFA-exempt user, at
     `/mfaVerify` with a temp token otherwise, never after the password step
     alone. The per-IP counter is not cleared (one valid account must not
     reset an address's count); it and any lock run out their windows
   - If `mfa_exempt = true`: Return full JWT
   - If `mfa_exempt = false`: Store temp token in the in-process KV store,
     return temp token and `mfaEnrolled` — true when a seed is stored and
     verified (`mfa_setup`), so the web app sends the user to verify a code
     (`/mfa-verify`), not to enrol (`/mfa-setup`)

2. **MFA enrolment** (`/mfaSetup`, temp token or session):
   - Input: `{tempToken: string}` (or a session cookie)
   - Generates a TOTP secret if the user has none, stores it AES-GCM
     encrypted under `TOTP_ENCRYPTION_KEY`, and returns `secret` and
     `qrCodeImage`. A seed stored but not yet verified is shown again
   - An **enrolled** user is refused 409 `MFA_ALREADY_SETUP` with neither:
     the stored secret is never shown again, since a temp token takes only
     the password

3. **MFA verify** (`/mfaVerify`):
   - Input: `{tempToken: string, totpCode: string}`
   - Verify temp token from the in-process KV store, then the TOTP code
     (30 s step, one step of skew either side)
   - A code is good once (RFC 6238 section 5.2): the step it matched must be
     above `TOTP_LAST_STEP*<user name>`, which is compared and raised in one
     atomic KV call (`SetIfGreater`) and lives 3 min, longer than the 90 s
     window. A replayed or older step gets the wrong-code answer (401
     "Invalid TOTP code") and counts toward the lockout. A store error is
     500 `TOTP_STEP_STORE` (fail closed). `/admin/resetUserMFA` deletes the
     key so a re-enrolled secret starts clean
   - No seed, or the empty seed an admin reset leaves, is 400
     `TOTP_SECRET_MISSING` ("Please restart MFA setup")
   - Return full JWT with cookie; the temp token is single use

4. **JWT Structure**:
   ```json
   {
     "iss": "Proto App",
     "sub": "username",
     "exp": 1678886400,
     "nbf": 1678882800,
     "iat": 1678882800,
     "jti": "1678882800123456789",
     "uid": 123,
     "ses": "session-id",
     "sctx": 1,
     "grants": [1001, 1002, 1003, 8001, 9001]
   }
   ```
   `jti` is the issue time in Unix ns (`iat` is whole seconds, too coarse
   to order a token against a revocation). `ses` is the session id: minted
   at sign-in and on a context switch, kept by `/refresh`, and its last 3
   characters are the CSRF token.

## Security

### Authentication
- **JWT**: RS256 signed tokens with session management
- **Password Hashing**: Argon2id with time=3, memory=64MB, threads=4, key length=32
- **Session Management**: Session IDs stored in JWT; `CheckJWT` refuses a revoked session (see Session Revocation)

### Session Revocation
A JWT is otherwise good until it expires, so `CheckJWT` also reads two
in-process KV keys on every request (`internal/security/session_revocation.go`):

- `SESSION_REVOKED*<session id>` — one session ended. Written by `/logoff`,
  and by `/changeSecurityContext` for the session it replaces.
- `REVOKED_BEFORE*<user name>` = `<unix ns>|<kept session id>` — every token
  of that login issued earlier is refused, except the kept session. Written
  by `/changeOwnPassword` (keeps the caller's session: the person who changed
  the password stays signed in), and with nothing kept by
  `/changeOtherUserPassword`, `/admin/resetUserMFA`, `PUT /user` (name,
  groups, MFA exemption; the prior name), `DELETE /user`,
  `/addUserListToGroup`, `/removeUserListFromGroup`, and `PUT /appUserGroup`
  (every member of the group, any tenant).

A revoked token gets 401; a KV read error gets 500 (fail closed). A failed
revocation after a committed write is answered 500 `SES_REVOKE`, never
hidden. Entries live `JWT_SESSION_EXPIRE_HOURS`, the longest a token can.

**Limit: a restart forgets revocations.** The KV is process memory and
revocations have no durable row, so after an API restart a token revoked
before it is good again until it expires. Accepted for one app server; a
second server or a need to survive restarts means a shared store (see
`internal/cache/kv.go`).

### Session cookie
One helper writes it, `setAuthCookie` in `internal/handler/common.go`, so
login (`/auth`, MFA-exempt), `/mfaVerify`, `/refresh` and
`/changeSecurityContext` cannot drift apart: `auth`, `Path=/`, `HttpOnly`,
`Secure`, `SameSite=Strict`.

**The JWT's exp (`JWT_SESSION_EXPIRE_HOURS`) is the only session
lifetime.** Every issuing path writes a browser-session cookie with no
`Max-Age` (`authCookieSessionMaxAge`); it lasts until the browser closes or
the server refuses the token. Only `/logoff` sets one
(`authCookieExpireNow`, sent as `Max-Age=0`) to expire it. A cookie
`Max-Age` shorter than the token would drop a session the server still
accepts: the web app calls `/refresh` on every page load, so one reload
would cut the session to that `Max-Age`.

### Request Body Limits
Go puts no bound on `r.Body`, so every handler that reads one caps it
first, through `decodeJSONBody` (or `readBody` for raw bytes) in
`internal/handler/common.go`. Over the cap the read stops and the answer is
**413** `{"error": "Request body too large"}` (logged `BODY_TOO_LARGE`); a
malformed body is **400** `{"error": "Invalid request body"}`. The decoder's
own text goes to the log only.

| Cap | Routes |
|---|---|
| 4 KiB (`maxAuthBodyBytes`) | `/auth`, `/mfaSetup`, `/mfaVerify`, `/changeSecurityContext`, `/userSelfCreate` (read before or without a session) |
| 8 KiB | `/error-report` |
| 12 KiB (4 KiB + `maxPreferenceBytes`) | `PUT /me` |
| 64 KiB (`maxLoginLayerBodyBytes`) | `POST`/`PUT /user`, `POST`/`PUT /appUserGroup`, `/addUserListToGroup`, `/removeUserListFromGroup`, `/changeOwnPassword`, `/changeOtherUserPassword`, `/admin/resetUserMFA` |
| 1 MiB (`maxEntityBodyBytes`) | `example_handler.go` writes, bulk lists included |

Deletes take their ids in the query string and read no body. The entity cap
is per entity: a derived app copies it with the handler and raises it where
a real bulk payload needs more. The cap is applied after the handler's
security checks except where the body is read before them (the 4 KiB
routes), so a signed-in caller can make the server hold at most the cap.

On servers nginx's `client_max_body_size` (20M in
`infra/*/etc/nginx_*`) is the outer bound and answers its own 413 before
the API sees the request; the per-route caps above are the inner bound, and
the only one when the API is reached without nginx (local dev).

### Authorization (RBAC)
- **Grants**: Context-specific permissions in JWT (only current context's grants)
- **User Groups**: Collections of grants assigned to users per security context
- **Context Switching**: Users can switch between security contexts, JWT updated accordingly

### Security Checks (Per Request)
1. **JWT Validation**: Verify token signature and claims
2. **Rate Limiting**:
   - Authenticated: 200 requests/minute per user
   - Unauthenticated: 20 requests/minute per IP
   - "IP" here and in the login lockout is `security.ClientIP`. With
     `TRUSTED_PROXY_COUNT=0` (local dev) it is `RemoteAddr` and
     `X-Forwarded-For` is ignored. With N (1 behind the host's nginx, which
     sets `X-Forwarded-For $proxy_add_x_forwarded_for`) it is the entry N
     hops from the right of `X-Forwarded-For`: the right-hand entries are
     the ones the trusted proxies appended, anything to their left is
     client-supplied and ignored. Ports are stripped. Behind a proxy this is
     sound only if the API is reachable through that proxy alone (bind it
     to loopback)
3. **User Blocking**: Check the in-process KV store for blocked users
   (session revocation is part of step 1, JWT validation)
4. **CSRF Protection**: Compare `X-CSRF-Token` header with last 3 chars of session ID (safe methods — GET, HEAD, OPTIONS, TRACE — are exempt)
5. **RBAC**: Verify user has required grant for target security context

**Each handler runs these itself, explicitly, in this order, in its own
body** (`internal/security/app_helper.go`). There is no middleware and no
route wrapper, by decision: the policy for an endpoint is read in
the handler, not inferred from a chain. The shape every entity handler has:

```go
app := security.NewAppHelper(w, r, h.appsrv, "RetrieveExample")
app.CheckJWT()
app.CheckRateLimitAndUserBlock()
app.CheckCSRF()
app.CheckAuthorization(model.GrantExampleRead)
if !app.CheckAllChecksComplete() {
    return
}
```

`CheckAllChecksComplete` is the contract: it returns false — and the
handler returns — if any check failed *or* if any of the five was neither
checked nor skipped. So an endpoint that needs a different set writes the
difference where the auditor reads it, with `Skip*` calls:

- `health_handler.go` — five `Skip*` calls: a public liveness probe checks
  nothing, and says so.
- `auth_handler.go` `AuthHandler` (login) — five `Skip*` calls: no JWT
  exists yet, and brute force is answered by the per-IP + per-username
  lockout (7 failures / 5 min) that runs before the password check and
  counts unknown names as well as wrong passwords.
- `RefreshHandler`, `LogoffHandler` — `CheckJWT`, rate limit and block, then
  `SkipCSRF` / `SkipAuthorization` with a one-line reason each.
- `UserSelfCreateHandler`, `error_report_handler.go` — `SkipJWT` +
  `CheckRateLimit` (the anonymous, IP-keyed limiter), everything else skipped.
- `MFASetupHandler`, `MFAVerifyHandler` — rate limit and block, `SkipCSRF`,
  `SkipAuthorization`; then either `SkipJWT` (a temp token from the
  password step stands in) or `CheckJWT`, chosen in the handler.

`example_handler_test.go` proves the sequence for the copy-me handler with
a mocked repository: no cookie → 401, DELETE without CSRF → 403, missing
grant → 403, blocked user → 403, and the repository is not called until
every check has passed.

### Security Headers (All Responses)
- **Content-Security-Policy**: Restricts resource loading to prevent XSS
- **Strict-Transport-Security**: Forces HTTPS connections (1 year + subdomains + preload)
- **X-Frame-Options**: Prevents clickjacking attacks
- **X-Content-Type-Options**: Prevents MIME type sniffing
- **Referrer-Policy**: Controls referrer information leakage
- **Permissions-Policy**: Restricts browser features (camera, microphone, geolocation, etc.)
- **X-Permitted-Cross-Domain-Policies**: Restricts Flash cross-domain requests
- **Cache-Control**: Prevents caching of sensitive API responses

nginx adds none of these on `/api/`; it sets its own four (nosniff,
frame, referrer, HSTS) on the web locations only, so each API header
arrives once.

### Multi-tenancy
- **Security Context**: Integer identifier for tenant isolation
- **Data Segregation**: All queries filter by `security_context`
- **Grant Isolation**: Permissions scoped to security contexts, JWT contains only current context grants
- **Context Switching**: `/changeSecurityContext` endpoint allows users to switch contexts
- **Home tenant**: a user's `app_user` row stays in its home tenant when a
  switch moves the JWT to another. `/me` therefore reads the row by the
  JWT's subject (`user_name` is unique installation-wide), refuses it with
  401 if its id is not the JWT's `uid`, and `PUT /me` writes it scoped by
  the row's own tenant, not the session's.

## Configuration

### Environment Variables (.env format)
Environment files follow the naming convention `loc-{service}.env` for local development configurations.

### Loading
- Command-line argument specifies config file path
- Flat `KEY=VALUE` file, no environment-variable lookups; grammar and the
  required-key policy in `doc/905-config-guidance.md`
- Every required key is checked for presence and shape at load; start-up
  fails with an error naming the key

## Logging

### Structured Logging
- **Library**: `Logger` (`internal/logging/logging.go`), a thin wrapper over
  `log/slog`'s text handler
- **Format**: logfmt, one line per call, written to stderr; systemd captures
  it into journald. No remote receiver.
- **Levels**: DEBUG, INFO, WARN, ERROR, via `Debug/Info/Warn/Error(code,
  message, kv...)`
- **Threshold**: `LOG_LEVEL` (required: `debug`, `info`, `warn` or `error`)
  sets the lowest level written. Per-request flow lines and health polls are
  DEBUG, so deployed hosts (`info`) skip them; local dev runs `debug`.
- **Context**: Username, user ID, request details passed as key/value pairs
- **Code**: the `code` argument is emitted as a `code=` attribute on every
  line, e.g. `level=INFO source=example_handler.go:88 msg="..." code=EX_CR_START service=proto-api username=alice`

### Log Codes
- 12-character codes for consistent categorization, uniqueness
- Examples: STRT (start), AUTH (authentication), DBFL (database failure)

### Username Tracking
- Extracted from JWT claims for authenticated requests
- Defaults to "PreAuth" for unauthenticated endpoints
- Included in all error responses and logs

## Error Handling

### HTTP Status Codes
- **200**: Success
- **201**: Created
- **400**: Bad Request (validation errors)
- **401**: Unauthorized (invalid/missing auth)
- **403**: Forbidden (insufficient permissions)
- **404**: Not Found (missing, soft-deleted, or another tenant's row)
- **409**: Conflict (stale `updateTs`, duplicate name, row still in use)
- **413**: Request body over the route's cap (see Request Body Limits)
- **417**: Expectation Failed (a bulk update whose rows did not all match)
- **429**: Too Many Requests (rate limited)
- **500**: Internal Server Error

### Repository Errors
A repository answers a failure the caller can act on with a sentinel from
`internal/repository/errors.go`, bare or wrapped with `%w`; never with a
bare `fmt.Errorf` whose text a handler would match. A handler passes any
repository error to `mapRepoError(app, w, err, failMsg, failCode)`
(`internal/handler/common.go`), which picks the status:

| Sentinel | Status | Client message |
|---|---|---|
| `ErrValidation` | 400 | "The request is not valid" |
| `ErrNotFound`, `ErrDeleted` | 404 | "Not found" |
| `ErrConflict` | 409 | "The record was modified by another user" |
| `ErrDuplicate` | 409 | "A record with that name already exists" |
| `ErrInUse` | 409 | "The record is still in use" |
| `ErrExpectationFailed` | 417 | "One or more records could not be updated due to concurrent modifications" |
| anything else | 500 | the handler's `failMsg`; the error text is logged only |

A soft-deleted row answers 404 like a missing one, so a caller cannot tell
that an id once existed. Where the detail is the useful half of the answer
(which rule a user name broke, how many members hold a group open) the
repository returns `clientMessage(sentinel, ...)`, a `ClientMessageError`:
`errors.Is` finds the sentinel and the message replaces the generic one.
Handlers that branch on an error for control flow (login's not-found, the
self-signup name check) use `errors.Is`, never the error's text.

### Optimistic Locking
- Client sends `updateTs` in requests
- Server compares with database value in WHERE clause
- On a zero-row write: query to tell a stale lock (`ErrConflict`, 409)
  from a missing or soft-deleted row (`ErrNotFound`/`ErrDeleted`, 404)
- Every write carries the tenant (`security_context_id`) in its WHERE
  clause, and so does the check that explains a zero-row write. An id that
  lives in another tenant therefore answers 404, never 409: a 409 would
  tell the caller the row exists. For the login-layer tables this is
  `zeroRowOutcome` (`internal/repository/scope.go`). Two lookups are
  unscoped by design: `RetrieveUserByUserName` (login has no tenant yet)
  and `GetGrantsForGroups` (a login may hold another tenant's group; that
  is how it gets a second security context).

### Database Resilience
- **No application retry**: a repository calls pgx once and returns its
  error. pgxpool drops a broken connection and dials a new one for the
  next request; a failed request answers 500 and the client retries.
  `pgconn.SafeToRetry` was considered and left out: a retry would sit on
  the request path for errors the pool already recovers from.
- **Timeout Handling**: Context-based cancellation

## Code Conventions

### Naming
- **Variables**: camelCase, descriptive names (e.g., `userName`, `securityContext`)
- **Lists**: Suffix with "List" (e.g., `userList`, `grantIdList`), don't use plural forms
- **Functions**: PascalCase, action-oriented (e.g., `GetUserByUserName`)
- **Files**: snake_case, descriptive (e.g., `auth_handler.go`)

### Structure
- **Imports**: Grouped (standard library, third-party, internal)
- **Error Handling**: Early returns, wrapped errors
- **DRY Principle**: Shared utilities in `common.go` and the `internal/{app,auth,cache,config,logging,security}` packages
- **Single Responsibility**: Focused functions and files

### Database Queries
- **Explicit Fields**: No wildcards in SELECT statements
- **Prepared Statements**: Parameterized queries
- **Multi-line**: Each field on separate line for readability

## Testing

### Unit Tests
- **Coverage**: Repository layer with pgx mocking
- **Framework**: testify/suite
- **Focus**: Business logic, error conditions

### Integration Tests
- **Python-based**: pytest with requests
- **Coverage**: Comprehensive API testing including:
  - Security context isolation and multi-tenancy
  - Error scenarios (401, 403, 404, 409, 429, 400)
  - Bulk operations performance and correctness
  - Authentication and authorization flows
  - Optimistic locking and conflict resolution
  - Rate limiting and CSRF protection
- **Environment**: Requires running server and dependencies

### Test Data
- **Isolation**: Separate test database/schema
- **Cleanup**: Proper teardown between tests
- **Fixtures**: Reusable test data setup

## Deployment

### Build Process
- **Commands**: `go build ./...`, `go test ./...`
- **Configuration**: Environment-specific config files

### Runtime Requirements
- **PostgreSQL**: Database server

### Operational Considerations
- **Health Checks**: `/health` endpoint for load balancers
- **Metrics**: Basic request counting (future enhancement)
- **Scaling**: Stateless design supports horizontal scaling
- **Security**: No sensitive data in logs, secure cookie settings

## Development Workflow

### Local Development
1. Build the database: `bin/510-reset-test-db`
2. Run server: `bin/215-run-api-srv`
3. Run tests: `go test ./...`

### Code Quality
- **Formatting**: `go fmt`
- **Linting**: golangci-lint with `.golangci.yml` configuration
- **Testing**: Run `go test ./...` to validate changes
- **Documentation**: Inline comments, README updates

## Future Enhancements

### High Priority
- TOTP implementation for MFA
- Input validation middleware
- OpenAPI/Swagger documentation

### Medium Priority
- Performance monitoring
- Caching layer
- Configuration hot-reloading
- Advanced rate limiting

### Low Priority
- Docker containerization
- CI/CD pipeline
- Log aggregation
- Feature flags

## Recent Implementations

### Completed Features
- **Comprehensive Integration Tests**: 43 test cases covering authentication, authorization, CRUD operations, error scenarios, and security features
- **User Management**: Full CRUD operations for users with proper RBAC controls
- **User Group Management**: Group-based permission assignment and management
- **Security Context Switching**: Multi-tenant context switching with JWT updates
- **Enhanced Error Handling**: Specific error codes and improved logging
- **Database Optimization**: Strategic indexing for update_ts fields and optimistic locking
- **Code Quality**: golangci-lint configuration and comprehensive test coverage

This specification captures the current implementation and serves as a foundation for future development and maintenance of the ProtoStack-ReactGo API server.
