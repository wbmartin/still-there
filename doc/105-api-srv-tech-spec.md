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
├── cmd/server/main.go              # Entry point; registers every route
├── cmd/migrate/                    # Schema migrator (embeds sql/migrations)
├── internal/
│   ├── app/services.go             # Services: config, KV, logger for every handler
│   ├── auth/                       # jwt.go, password.go (argon2id), totp.go, rbac.go, token.go (invite tokens)
│   ├── cache/kv.go                 # In-process key-value store
│   ├── config/config.go            # Env-file loader, Config
│   ├── logging/logging.go          # slog wrapper, code= attribute
│   ├── migrate/                    # Migration runner used by cmd/migrate
│   ├── security/                   # app_helper.go (per-handler checks, ClientIP),
│   │                               # session_revocation.go, security_headers.go
│   ├── handler/                    # HTTP handlers, one file per concern
│   │   ├── auth_handler.go         # /auth, /refresh, /changeSecurityContext, /logoff, /changeOwnPassword
│   │   ├── user_mfa_handler.go     # /userSelfCreate, /mfaSetup, /mfaVerify, /admin/resetUserMFA
│   │   ├── user_handler.go         # /user, /userList, /changeOtherUserPassword, group membership, /grantList
│   │   ├── user_block_handler.go   # /admin/blockUser, /admin/unblockUser
│   │   ├── user_invite_handler.go  # /userInvite*, /inviteCheck, /inviteRedeem
│   │   ├── app_user_group_handler.go # /appUserGroup, /appUserGroupList
│   │   ├── profile_handler.go      # GET/PUT /me
│   │   ├── example_handler.go      # /example, /exampleList (the copy-me entity)
│   │   ├── error_report_handler.go # /error-report
│   │   ├── health_handler.go       # /health
│   │   ├── common.go               # Body caps, cookie writer, mapRepoError, query helpers
│   │   └── route_check_contract_test.go # Every route's checks (see Security Checks)
│   ├── model/                      # Wire and row models; grants.go (grant ids and short codes)
│   └── repository/                 # Data access; errors.go (sentinels), scope.go (tenant-scoped zero-row checks)
├── sql/                            # migrations/, fixtures/, roles.sql
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
- **Multi-tenancy**: `security_context_id` integer column on all tables
- **Naming**: Snake_case column names
- **Relationships**: No foreign keys; managed in application layer
- **Defaults/Triggers**: Prohibited; all logic in application code

## Database
Schema: `src/api-srv/sql/migrations/NNNN_name.sql` (0001 to 0005 today),
applied by `cmd/migrate` only. Seed rows: `src/api-srv/sql/fixtures/`
`local-seed.sql` (dev, tenant 1) and `test-seed.sql` (tests, tenants 2 and
3; `test-tenant-lock.sql` refuses tenant 1 in a test database).

## API Design

### Endpoints

Every route `cmd/server/main.go` registers, in its order. The checks
columns are the prose form of `routeContractList` in
`internal/handler/route_check_contract_test.go`, which fails when a route
is added without a row and proves each row's checks answer before any
repository call; change the two together. Grants are the short codes of
`internal/model/grants.go` (id in brackets).

Columns: **JWT** `CheckJWT` (a session cookie). **Rate** the per-minute
limit: *user+block* is `CheckRateLimitAndUserBlock` (per user name, plus
the `BLOCKED*` flag); *address* is the anonymous, per-IP limit; *lockout*
is the login lockout instead. **CSRF** `CheckCSRF` (never on GET).
**Grant** `CheckAuthorization`. A dash is a check the handler skips with a
`Skip*` call; each is explained under Security Checks, Exceptions.

| Method | Path | JWT | Rate | CSRF | Grant | Description |
|---|---|---|---|---|---|---|
| GET | /health | - | - | - | - | Liveness: `{"data":{"status":"ok"}}` |
| POST | /auth | - | lockout | - | - | User name + password; full session or MFA temp token |
| GET | /refresh | Yes | user+block | (GET) | - | New JWT for a live session, same session id |
| POST | /changeSecurityContext | Yes | user+block | - | - | Switch tenant; the login must hold a group there |
| POST | /changeOwnPassword | Yes | user+block | Yes | CHG_OWN_PW (8001) | Change own password; other sessions revoked |
| GET | /me | Yes | user+block | (GET) | LOGON (1) | The caller's own row (profile + preferences; preferences appear only here, not on `/userList`) |
| PUT | /me | Yes | user+block | Yes | LOGON (1) | Own firstName/lastName/email/preferenceJsonb, optimistic lock |
| POST | /logoff | Yes | user+block | - | - | End the session and expire the cookie |
| GET | /example | Yes | user+block | (GET) | READ_EXAMPLE (1001) | One example |
| POST | /example | Yes | user+block | Yes | CREATE_EXAMPLE (1002) | Create |
| PUT | /example | Yes | user+block | Yes | UPDATE_EXAMPLE (1003) | Update, optimistic lock |
| DELETE | /example | Yes | user+block | Yes | DELETE_EXAMPLE (1004) | Soft delete |
| GET | /exampleList | Yes | user+block | (GET) | READ_EXAMPLE (1001) | All, by id list, or changed since a time |
| POST | /exampleList | Yes | user+block | Yes | CREATE_EXAMPLE (1002) | Bulk create, one transaction |
| PUT | /exampleList | Yes | user+block | Yes | UPDATE_EXAMPLE (1003) | Bulk update, one transaction |
| DELETE | /exampleList | Yes | user+block | Yes | DELETE_EXAMPLE (1004) | Bulk soft delete |
| POST | /userSelfCreate | - | address | - | - | Public self-registration (see Exceptions) |
| POST | /inviteCheck | - | address, then lockout | - | - | Who a live invite link is for (public) |
| POST | /inviteRedeem | - | address, then lockout | - | - | Set the password from a live link (public) |
| POST | /mfaSetup | temp token or Yes | address | - | - | Enrol: TOTP secret and QR code |
| POST | /mfaVerify | temp token or Yes | address; lockout | - | - | Check a TOTP code; completes an MFA login |
| GET | /userList | Yes | user+block | (GET) | READ_USER (9002) | Logins of the caller's tenant |
| POST | /user | Yes | user+block | Yes | CREATE_USER (9001) | Create a login with a password |
| PUT | /user | Yes | user+block | Yes | UPDATE_USER (9003) | Update a login |
| DELETE | /user | Yes | user+block | Yes | DELETE_USER (9004) | Soft delete a login |
| POST | /changeOtherUserPassword | Yes | user+block | Yes | CHANGE_USER_PW (9009) | Set another login's password |
| POST | /addUserListToGroup | Yes | user+block | Yes | UPDATE_USER_GROUP (9012) | Add logins to a group |
| POST | /removeUserListFromGroup | Yes | user+block | Yes | UPDATE_USER_GROUP (9012) | Remove logins from a group |
| POST | /userInvite | Yes | user+block | Yes | CREATE_USER (9001) | Invited login (no password) and its one-time link |
| POST | /userInviteReissue | Yes | user+block | Yes | UPDATE_USER (9003) | New link; revokes the live one |
| POST | /userInviteRevoke | Yes | user+block | Yes | UPDATE_USER (9003) | Stop an invited login's live links |
| POST | /admin/resetUserMFA | Yes | user+block | Yes | RESET_MFA_TOKEN (9020) | Clear a login's TOTP seed |
| POST | /admin/blockUser | Yes | user+block | Yes | BLOCK_USER (9008) | Block a login of the caller's tenant |
| POST | /admin/unblockUser | Yes | user+block | Yes | BLOCK_USER (9008) | Unblock it |
| GET | /grantList | Yes | user+block | (GET) | CREATE_USER (9001) | Every grant id and short code |
| GET | /appUserGroupList | Yes | user+block | (GET) | READ_USER_GROUP (9010) | Groups of the caller's tenant |
| POST | /appUserGroup | Yes | user+block | Yes | CREATE_USER_GROUP (9011) | Create a group |
| PUT | /appUserGroup | Yes | user+block | Yes | UPDATE_USER_GROUP (9012) | Update a group (name, grant list) |
| DELETE | /appUserGroup | Yes | user+block | Yes | DELETE_USER_GROUP (9013) | Soft delete a group (409 while in use) |
| POST | /error-report | Yes | user+block | Yes | LOGON (1) | Log a client error report |

`ADD_USER_TO_GROUP` (9014) and `REMOVE_USER_FROM_GROUP` (9015) are defined
in `grants.go` but no route checks them: group membership is checked with
`UPDATE_USER_GROUP`.

### Request/Response Format

- **Content-Type**: `application/json`
- **Field Naming**: camelCase in JSON, snake_case in database
- **Timestamps**: seconds since the Unix epoch as a float64 with a
  fractional part (`model.Now()`, `numeric(17,7)` in the database)
- **Success body**: `{"data": ...}`. The session routes (`/auth`,
  `/mfaVerify`, `/refresh`, `/changeSecurityContext`) answer
  `{"user": {...}, "csrfToken": "..."}` instead.
- **Error body**: `{"error": "<message>"}`; a CSRF refusal adds
  `"code": "CSRF"`, which the web client keys its token resync on. See
  Error Handling.
- **Never serialized**: `PwHash` and `TOTPSeed` are `json:"-"` in
  `internal/model/app_user_model.go`, so no response carries a password
  hash or a TOTP seed.
- **Query strings**: list reads take `?<entity>IdList=1,2,3` and
  `?afterUpdateTs=<ts>`; deletes take the id and `updateTs` (lists:
  comma-separated) and read no body.

Request bodies, where a route takes one (the field names the handler
decodes). The decoder ignores an unknown key and leaves a missing one
zero, so a client still sending an old name (`pwHash`, `userIds`) is
refused 400 by the field's own validation, not by the decoder:

| Route | Body |
|---|---|
| `/auth` | `{userName, password}` |
| `/mfaSetup` | `{tempToken}`, or no token and a session cookie |
| `/mfaVerify` | `{tempToken, totpCode}`, or `{totpCode}` and a session cookie |
| `/changeSecurityContext` | `{securityContext}` |
| `/changeOwnPassword` | `{oldPassword, newPassword}` |
| `PUT /me` | `{firstName, lastName, email, preferenceJsonb, updateTs}` |
| `/userSelfCreate` | `{username, password}` (lower-case `username`) |
| `POST /user` | `{userName, password, firstName, lastName, email, appUserGroupMembershipList, mfaExempt}` |
| `PUT /user` | the user row as `GET /userList` returns it, with `updateTs` |
| `/changeOtherUserPassword` | `{appUserId, newPassword}` |
| `/addUserListToGroup`, `/removeUserListFromGroup` | `{userIdList, appUserGroupId, updateTsList}` |
| `/userInvite` | `POST /user`'s body without `password` |
| `/userInviteReissue`, `/userInviteRevoke` | `{appUserId}` |
| `/inviteCheck`, `/inviteRedeem` | `{token}`, `{token, password}` |
| `/admin/resetUserMFA` | `{userId}` |
| `/admin/blockUser`, `/admin/unblockUser` | `{userName}` |
| `POST`/`PUT /appUserGroup` | the group row (`groupName`, `userGrantList`; `appUserGroupId` and `updateTs` on PUT) |
| `/error-report` | any text; the first 2000 characters are logged |

The password key is `password` everywhere a password is set
(`pwHash` is gone, B038); list keys are `fooList`, never plural.

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
   - If `mfa_exempt = false`: Store a temp token in the in-process KV store
     (`TEMP_AUTH_SUCCESS_<token>` = user name, 10 min), return it with no
     grants and `mfaEnrolled` — true when a seed is stored and
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
   - There is no `/auth2` route; `/mfaVerify` is the second step

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
   HS256, signed with `JWT_SECRET` (`internal/auth/jwt.go`); `iss` is the
   `APP_NAME` config value. `jti` is the issue time in Unix ns (`iat` is whole seconds, too coarse
   to order a token against a revocation). `ses` is the session id: minted
   at sign-in and on a context switch, kept by `/refresh`, and its last 3
   characters are the CSRF token.

## Security

### Authentication
- **JWT**: HS256 signed with `JWT_SECRET`, carried in the `auth` cookie
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
  `/changeOtherUserPassword`, `/admin/resetUserMFA`, `/admin/blockUser`,
  `PUT /user` (name,
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

### User Blocking
`POST /admin/blockUser` / `/admin/unblockUser`, body `{"userName": ...}`,
grant `BLOCK_USER` (9008), `internal/handler/user_block_handler.go`.
`app_user.account_status` is `active`, `blocked` or `invited` (migration
`0004`, NOT NULL, no DEFAULT; `CreateUser` writes `active`, `/userInvite`
`invited`). Block:

- sets the row `blocked`; `/auth` refuses any login that is not `active`
  with the same counted 401 as a wrong password;
- revokes every session of the login (`REVOKED_BEFORE*`, above): a live
  token gets 401 and the web app signs out; unblock does not revive it;
- sets `BLOCKED*<user name>` for `JWT_SESSION_EXPIRE_HOURS`, which
  `CheckRateLimitAndUserBlock` answers 403. The server reloads every
  blocked login's flag at start, so after a restart (revocations gone) a
  pre-block token gets 403 "User account is blocked" per call until it
  expires; the web app does not sign out on 403.

Unblock sets the row `active` and clears the flag. Refusals: own login 400,
another tenant's or unknown login 404, wrong starting state 409, stale row
409. The `admin` subject is exempt from the flag check (as from the rate
limit); its row still stops it signing in.

### Invite Links
An admin onboards a person without handling their password (B035,
`internal/handler/user_invite_handler.go`). It is a second admin path
beside `POST /user` (admin sets a password), and both sit beside public
self-registration (`POST /userSelfCreate`); a derived app keeps the paths
it wants.

- `POST /userInvite` (ADD_USER, body as `POST /user` without a password)
  creates the login with `account_status = invited` and an empty
  `pw_hash`, and one `user_invite` row (migration `0005`), in one
  transaction. The answer carries `inviteToken` (32 random bytes, hex)
  once; only its SHA-256 is stored. The link is `<origin>/invite#<token>`:
  the fragment never reaches a server log or `Referer`.
- `POST /userInviteReissue` / `/userInviteRevoke` (CHG_USER, body
  `{appUserId}`): reissue revokes every live link and issues a new one;
  revoke with no live link is 409. A login that is not invited, deleted or
  of another tenant is 404.
- `POST /inviteCheck` / `/inviteRedeem` (public, body `{token}` /
  `{token, password}`, 4 KiB cap): `SkipJWT`, `CheckRateLimit` (per
  address), `SkipUserBlock`, `SkipCSRF`, `SkipAuthorization`, each stated
  in the handler. Then the `/auth` lockout: the per-address key /auth
  counts plus a per-token key (`invite:<16 hex of the hash>`); 429 when
  locked. Every unusable link (malformed, unknown, expired, redeemed,
  revoked, login deleted or not invited) is one counted 404 with one
  message. Redeem checks the password floor (8), then the token, then
  runs Argon2id, then locks invite and user `FOR UPDATE` and sets the
  password, `active`, `redeemed_ts` and `redeemed_source_ip`.
- An invited login cannot sign in (`/auth` refuses anything not
  `active`), and `/changeOtherUserPassword` answers it 409. Links expire
  after `INVITE_TTL_HOURS` (required). No e-mail is sent: the admin copies
  the link from the web app, which shows it once.

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
| 4 KiB (`maxAuthBodyBytes`) | `/auth`, `/mfaSetup`, `/mfaVerify`, `/changeSecurityContext`, `/userSelfCreate`, `/inviteCheck`, `/inviteRedeem` (read before or without a session) |
| 8 KiB (`maxErrorReportBodyBytes`) | `/error-report` (signed-in; logs the first 2000 characters) |
| 12 KiB (4 KiB + `maxPreferenceBytes`) | `PUT /me` |
| 64 KiB (`maxLoginLayerBodyBytes`) | `POST`/`PUT /user`, `POST`/`PUT /appUserGroup`, `/addUserListToGroup`, `/removeUserListFromGroup`, `/changeOwnPassword`, `/changeOtherUserPassword`, `/admin/resetUserMFA`, `/admin/blockUser`, `/admin/unblockUser`, `/userInvite`, `/userInviteReissue`, `/userInviteRevoke` |
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
1. **JWT Validation** (`CheckJWT`): the `auth` cookie's signature and
   claims, then session revocation (see Session Revocation). No cookie or
   a bad token is 401
2. **Rate Limiting**: a fixed per-minute window counted in the KV
   - Signed in: `MAX_USER_REQ_PER_MIN` per user name; key
     `RPM*<user name>*<minute>`
   - Anonymous: `MAX_UNAUTH_REQ_PER_MIN` per address; key
     `RPM*ip:<client IP>*<minute>` (`anonRateIdentifier`). The `ip:`
     prefix keeps anonymous buckets apart from user names, so a user named
     like an address never shares that address's bucket
   - Over the limit is 429 "Rate limit exceeded"
   - "IP" here and in the login lockout is `security.ClientIP`. With
     `TRUSTED_PROXY_COUNT=0` (local dev) it is `RemoteAddr` and
     `X-Forwarded-For` is ignored. With N (1 behind the host's nginx, which
     sets `X-Forwarded-For $proxy_add_x_forwarded_for`) it is the entry N
     hops from the right of `X-Forwarded-For`: the right-hand entries are
     the ones the trusted proxies appended, anything to their left is
     client-supplied and ignored. Ports are stripped. Behind a proxy this is
     sound only if the API is reachable through that proxy alone (bind it
     to loopback)
3. **User Blocking**: the `BLOCKED*<user name>` KV flag answers 403 "User
   account is blocked" (session revocation is part of step 1). See User
   Blocking
4. **CSRF Protection**: Compare `X-CSRF-Token` header with last 3 chars of
   session ID (safe methods — GET, HEAD, OPTIONS, TRACE — are exempt);
   missing or wrong is 403 with `"code": "CSRF"`. A mismatch is logged
   (`CSM0`) without either token: both are slices of a session ID
5. **RBAC**: Verify user has required grant for the JWT's security
   context; 403 `Permission denied (grant "<SHORT_CODE>" required)`

**The `admin` subject skips steps 2 and 3.** `CheckRateLimit` and
`CheckRateLimitAndUserBlock` return early for a JWT whose subject is the
literal `admin` (the fixture login). That login has no rate limit and its
`BLOCKED*` flag is never read; blocking it still stops it signing in
(`account_status`). Stated as it is; flagged for a security look in
`doc/700-grc-backport-feedback.md` (Appendix: open in both repos).

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

#### The fail-closed contract

`CheckAllChecksComplete` is the contract: it returns false — and the
handler returns — if any check failed *or* if any of the five was neither
checked nor skipped. An unchecked check answers 500 "<name> check not
completed", never a success. Every check that reads the KV fails closed
too: an error reading revocations, the rate counter or the block flag is
500 ("Session check failed", "Rate limit check failed", "User block check
failed"), not a pass. `WriteJSON` re-runs the gate before any success
status (below 400), so a handler that skipped the gate call still answers
500. The login lockout
counters and the TOTP last-step key follow the same rule (Authentication
Flow).

#### Exceptions, and why each one is allowed

An endpoint that needs a different set writes the difference where the
auditor reads it, with `Skip*` calls. These are every `Skip*` site;
anything not here runs all five.

| Handler (route) | Skips | Why |
|---|---|---|
| `HealthHandler` (`/health`) | all five | Public liveness probe; reads nothing |
| `AuthHandler` (`/auth`) | all five | No session yet. Brute force is answered by the per-address + per-name lockout (`FAILED_LOGIN_ATTEMPT_THRESHOLD` failures in `FAILED_LOGIN_ATTEMPT_WINDOW_DURATION`, locked for `FAILED_LOGIN_LOCKOUT_DURATION`; 7 / 5min / 5min locally), checked before the password and counting unknown names. The 4 KiB body is read first |
| `RefreshHandler` (`/refresh`) | CSRF, grant | A GET; any live session refreshes |
| `LogoffHandler` (`/logoff`) | CSRF, grant | Ending one's own session needs neither |
| `ChangeSecurityContextHandler` (`/changeSecurityContext`) | CSRF, grant | Kept by decision (doc/700 B005, skipped): the switch writes no table row; it ends the old session and mints a new one. The grant is replaced by a check after the gate that the login holds a group in the target tenant (403 "Security context not available"). The 4 KiB body is decoded before `CheckJWT` |
| `UserSelfCreateHandler` (`/userSelfCreate`) | JWT, block, CSRF, grant | Public self-registration, per-address rate limit only. It creates a tenant-1 login in group 3 and answers 409 "Username already taken" for a name in use (a name probe). Kept beside invite links (doc/700 B002 skipped, B035); a derived app keeps or removes it |
| `MFASetupHandler`, `MFAVerifyHandler` (`/mfaSetup`, `/mfaVerify`) | CSRF, grant; JWT on the temp-token path | Temp token from the password step, or a session. The rate check runs before the JWT is read, so it is per address and the `BLOCKED*` flag is not consulted on either path (only matters after a restart, when revocations are lost). The 4 KiB body is read before the gate because it picks the path |
| `UserInviteHandler.CheckInvite`, `.RedeemInvite` (`/inviteCheck`, `/inviteRedeem`) | JWT, block, CSRF, grant | Public; the token is the credential. Per-address rate limit, then the `/auth` lockout per address plus per token (see Invite Links) |

#### What proves it

- `internal/handler/route_check_contract_test.go`: every route in
  `main.go` has a row (`TestRouteContract_EveryRouteIsListed`), every skip
  has a reason (`TestRouteContract_SkipsAreExplained`), and each row,
  failing one check at a time, answers 401 / 429 / 403 / 403 / 403 with that
  check's message before any repository is touched, while a skipped check
  never answers (`TestRouteContract_ChecksAnswerBeforeTheRepository`).
- `example_handler_test.go` `TestExampleHandler_CheckSequence`: the copy-me
  handler's sequence with a mocked repository.
- `auth_handler_test.go`: the lockout answers 429 before the repository
  (`TestAuthHandler_LockoutAnswersBeforeTheRepository`) and a lockout read
  error is 500 (`TestAuthHandler_LockoutReadErrorIs500`).

### Security Headers (All Responses)
- **Content-Security-Policy**: `default-src 'none'; frame-ancestors
  'none'; base-uri 'none'; form-action 'none'`. The API answers JSON only
  and serves no page, script or asset (nginx serves the web bundle), so
  nothing may load from a response. A derived app's page policy belongs in
  nginx, not here
- **Strict-Transport-Security**: Forces HTTPS connections (1 year + subdomains + preload)
- **X-Frame-Options**: `DENY` on every path
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
- **Data Segregation**: All queries filter by `security_context_id`, taken
  from the JWT; a handler overwrites any tenant a client sends
- **Grant Isolation**: Permissions scoped to security contexts, JWT contains only current context grants
- **Context Switching**: `/changeSecurityContext` endpoint allows users to switch contexts
- **Home tenant**: a user's `app_user` row stays in its home tenant when a
  switch moves the JWT to another. `/me` therefore reads the row by the
  JWT's subject (`user_name` is unique installation-wide), refuses it with
  401 if its id is not the JWT's `uid`, and `PUT /me` writes it scoped by
  the row's own tenant, not the session's.

## Configuration

### Environment Variables (.env format)
Environment files follow the naming convention `loc-{service}.env` for
local development configurations. The API's keys, required and optional,
are listed in `src/api-srv/README.md` (Configuration); among them
`INVITE_TTL_HOURS` (invite link lifetime), `TRUSTED_PROXY_COUNT`
(`ClientIP`), the rate limits and the lockout thresholds above.

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
- Defaults to "preauth" for unauthenticated endpoints
- Included in every error log line (`WriteError`); never in the response

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
- **429**: Too Many Requests: the per-minute rate limit, or a login
  lockout (`/auth`, `/mfaVerify`, `/inviteCheck`, `/inviteRedeem`) with
  its own message
- **500**: Internal Server Error, including every fail-closed refusal (an
  unchecked check, a KV error in a check or a lockout counter); the
  message is generic and the cause is logged only

The body of every refusal is `{"error": "<message>"}`, written for a
person: the web app shows it. How the web app reads a status
(`src/web-app/utils/api.ts`, table in `src/web-app/README.md`, API
Errors): any 5xx or 429, or a failed fetch, is `ApiUnreachableError`
and raises the one "API unreachable" banner with the server's message (a
lockout 429 is shown as the login form's message); a 401 on a session call
is `SessionExpiredError` and signs the user out; every other refusal,
403 included, is an `ApiError` carrying `status` and the message, and
does not sign out. So a 5xx means "the server could not do it", never "you
asked wrong": a validation failure must be a 4xx.

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

### Go Tests
- **Handlers**: testify mocks that panic on an unexpected call, so a test
  proves a refusal never reached the repository; the route contract
  table (Security Checks) covers every route
- **Repositories**: testify/suite against the scratch test database
  (`TEST_DATABASE_URL` from `etc/loc-test.env`), tenants 2 and 3 only

### Integration Tests
- **Python-based**: pytest with requests
- **Coverage**: Comprehensive API testing including:
  - Security context isolation and multi-tenancy
  - Error scenarios (401, 403, 404, 409, 429, 400)
  - Bulk operations performance and correctness
  - Authentication and authorization flows
  - Optimistic locking and conflict resolution
  - Rate limiting and CSRF protection
- **Environment**: a private test API and scratch database that
  `bin/300-run-tests` builds; see `tests/README.md`

### Test Data
- **Isolation**: tenant 2 is the test tenant; test databases refuse
  tenant-1 rows (`sql/fixtures/test-tenant-lock.sql`)
- **Cleanup**: tests create throwaway logins and rows and remove them
- **Fixtures**: `sql/fixtures/test-seed.sql`

## Deployment

### Build Process
- **Commands**: `go build ./...`, `go test ./...`
- **Configuration**: Environment-specific config files

### Runtime Requirements
- **PostgreSQL**: Database server
- **One API process**: temp tokens, session revocations, block flags,
  rate counters, lockout counters and the TOTP last-step keys live in the
  in-process KV (`internal/cache/kv.go`)

### Operational Considerations
- **Health Checks**: `/health` endpoint for load balancers
- **Metrics**: Basic request counting (future enhancement)
- **Scaling**: single process. Two API processes would each count rate
  limits and lockouts separately (halving every lockout), reject a temp
  token the other minted, and miss the other's revocations. A second
  process needs a shared store first
- **Security**: No sensitive data in logs, secure cookie settings

## Development Workflow

### Local Development
1. Build the database: `bin/510-reset-test-db`
2. Run server: `bin/215-run-api-srv`
3. Run tests: `go test ./...`

### Code Quality
- **Formatting**: `gofmt`; the gate fails on any `gofmt -l` output
- **Static analysis**: `go vet` and pinned `staticcheck`, zero findings
  (`bin/300-run-tests go`, per module: `src/api-srv`, `src/load-test`)
- **Testing**: `bin/300-run-tests go` runs all of the above plus
  `go test ./...`
- **Documentation**: Inline comments, README updates

## Future Enhancements

### High Priority
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
- **Integration Tests**: pytest suite covering authentication, authorization, CRUD operations, error scenarios, and security features
- **MFA**: TOTP enrolment and verify, replay refused, admin reset
- **Session revocation, user blocking, invite links**: see Security
- **User Management**: Full CRUD operations for users with proper RBAC controls
- **User Group Management**: Group-based permission assignment and management
- **Security Context Switching**: Multi-tenant context switching with JWT updates
- **Enhanced Error Handling**: Specific error codes and improved logging
- **Database Optimization**: Strategic indexing for update_ts fields and optimistic locking
- **Code Quality**: gofmt, go vet and staticcheck gate; comprehensive test coverage

This specification captures the current implementation and serves as a foundation for future development and maintenance of the ProtoStack-ReactGo API server.
