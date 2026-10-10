# Security Architecture

What the template defends, where the trust boundaries are, and which
controls hold today. `doc/105-api-srv-tech-spec.md` keeps the mechanics
(endpoint table with the checks each route runs, the `Skip*` exceptions
table, body caps, cookie, revocation and block details); this document is
the model those mechanics serve and does not repeat their tables. A
derived app inherits every row below: it copies this file, keeps the rows
that still apply and adds rows for what it adds.

Each control is marked:

- **met** — the template does it; the row names the code, configuration
  or test that shows it;
- **partial** — part of it is done, or a known gap remains (the row says
  which);
- **open** — not done; the row says what would close it.

Open findings are the rows marked open or partial below; each says what
would close it. A change to a control changes its row here in the same
commit.

## Shape of a deployment

A post-sales host (`infra/README.md`); the presales host differs where a
row says so.

    browser
      | HTTPS 443 (HTTP 80 answers 301)
      v
    nginx (host)  -- serves web-pub/, proxies /api/ --
      | HTTP, 127.0.0.1:8088, X-Forwarded-For appended
      v
    api-srv  (systemd unit, user svc-proto, sandboxed)
      | pgx, 127.0.0.1:5432, role proto_app (DML only)
      v
    Postgres  (schema owned by proto_ddl, used only by the migrator)

    operations: deploy hooks as shrmpl-hook; root cron for certs and the
    nightly backup, which pushes to REMOTE_SERVER by scp

One API process per host. Rate counters, login lockouts, MFA temp tokens,
the TOTP last-step key, block flags and session revocations live in that
process's memory (`src/api-srv/internal/cache/kv.go`); a second process
needs a shared store first.

## Trust boundaries

| Id | Boundary | Crossed by | Trust decision |
|---|---|---|---|
| B1 | Internet to API | every browser request | nginx terminates TLS; the API trusts nothing in a request but a valid session cookie, and takes the tenant from it |
| B2 | API to Postgres | every query | the database trusts the API role completely; tenant isolation is the API's job alone |
| B3 | Operations | deploys, backups, exports, operators on the host | whoever holds root, the env files or a backup holds every secret |

Boundaries a larger app has (a credential vault, a background worker,
external users, support access, a file store) are features the template
does not have. A derived app that adds one adds its boundary and rows here.

## Tenant isolation

`security_context_id` is the tenant (`CLAUDE.md`,
`doc/900-database-guidelines.md`). Isolation is enforced **in application
code only**. There is no Postgres row-level security and no per-tenant
role: `proto_app` can read and write every tenant's rows, so a query that
omits the tenant predicate returns or changes another tenant's data and
nothing behind it notices.

What enforces it:

- **The tenant comes from the JWT.** Handlers take `sctx` from the claims
  and overwrite any tenant a client sends;
  `tests/test_security_context_isolation.py` posts another tenant's id and
  999 and gets rows in the caller's tenant.
- **List reads** build their `WHERE` through `buildScopedWhere`
  (`src/api-srv/internal/repository/scoped_query.go`): the tenant is always
  `$1`, an id list narrows it and never replaces it.
- **Writes** carry the tenant beside the id on every `UPDATE`, and a
  zero-row write is explained by `zeroRowOutcome`
  (`src/api-srv/internal/repository/scope.go`, login-layer tables) or the
  same query inline (`example_repository.go`) with the same predicate. An
  id in another tenant reads as absent: **404, never 409**, so an answer
  never confirms that an id exists elsewhere. Two lookups are unscoped by
  design: `RetrieveUserByUserName` (sign-in, names are unique per
  installation) and `GetGrantsForGroups`; doc/105 Optimistic Locking.
- **Group lists in a user write** (`POST`/`PUT /user`, `/userInvite`,
  `/addUserListToGroup`, `/removeUserListFromGroup`) are read scoped to the
  caller's tenant; any id not found there, foreign or absent alike, is one
  403 `SEC_CTX_VIOLATION`. Same answer both ways, so no oracle, but 403
  where the rule above says 404. A failed lookup is 500 `SEC_CTX_CHECK`
  ("Failed to check the security context", cause logged), never a 403
  that hides the outage.
- **Test databases hold no tenant 1.** `sql/fixtures/test-tenant-lock.sql`
  adds `CHECK (security_context_id <> 1)` to every table with the column,
  read from the catalog; tests run as tenant 2 and prove isolation against
  tenant 3.
- **No hard deletes.** Every repository delete sets `delete_ts` /
  `delete_user_id`; no `DELETE FROM` runs in `src/api-srv/internal`. This
  is an application rule: `db_writer` holds `ALL PRIVILEGES` on the tables
  (migration `0001_initial_schema.sql`), so the role itself could delete.
- **Integer keys.** Every primary key is an integer `<table>_id`. A UUID
  appears only as an attribute (`example_uuid`,
  `0002_example_jsonb_uuid.sql`), never as a key or a reference.

The rule for every new tenant table: one cross-tenant negative test at the
repository layer (read, update and delete of a tenant-3 row from tenant 2
answer not-found), and at the API where a route reaches the table. Today:

| Table | Cross-tenant test | Status |
|---|---|---|
| `app_user` | `user_repository_test.go` (update, delete, password, MFA and status writes by another tenant's id) | met |
| `app_user_group` | `app_user_group_repository_test.go`; pytest `test_cross_security_context_group_is_404_not_409` | met |
| `user_invite` | `user_invite_repository_test.go` (reissue and revoke from another tenant) | met |
| `example` | `example_repository_test.go` `TestCrossTenantIsolation` (list, get, update, delete and the batch writes against a tenant-3 row from tenant 2) | met |
| `issue_report`, `issue_report_message`, `issue_report_attachment` | `issue_report_repository_test.go` `TestCrossTenantIsolation` (list, get, detail, file and a message with an action against a tenant-3 report from tenant 2); pytest `test_another_tenant_cannot_reach_reports`. No delete exists. The relay's listings cross tenants by design (a system job); each of its writes names the row's tenant beside the id | met |

## Threats and controls

STRIDE per boundary. Each Control cell names the code, configuration or
test that shows it; the mechanics are in doc/105.

### B1: internet to API

| # | Threat | Control | Status |
|---|---|---|---|
| 1.1 | Forged or tampered session token | JWT HS256 signed with `JWT_SECRET` (per host, `openssl rand -hex 32`), in cookie `auth`: `HttpOnly`, `Secure`, `SameSite=Strict`, no `Max-Age`. `ValidateJWT` (`src/api-srv/internal/auth/jwt.go`) rejects a non-HMAC algorithm; `exp`/`nbf` are checked by golang-jwt. `iss` is set from `APP_NAME` but not checked | met |
| 1.2 | Stolen token used after logout, password change, block | Session revocation in `CheckJWT` (`src/api-srv/internal/security/session_revocation.go`); servers set `JWT_SESSION_EXPIRE_HOURS=12`. Revocations are process memory: **an API restart forgets them** and a revoked token works again until `exp` (accepted for one process) | partial |
| 1.3 | Password guessing, credential stuffing | Argon2id; unknown names run a dummy hash and get the same counted 401; lockout per address and per name (`FAILED_LOGIN_*`), checked before the password, failing closed on a counter error (`auth_handler_test.go`) | met |
| 1.4 | Username enumeration | Same 401 and timing for unknown, wrong and non-`active` logins. Exception: public `/userSelfCreate` answers 409 for a taken name | partial |
| 1.5 | Second factor bypassed or replayed | TOTP for every login not `mfa_exempt`: `/auth` gives a 10-minute single-use temp token and no grants; `/mfaVerify` takes a code once (last-step key, `SetIfGreater`); an enrolled user's secret is never shown again (409) | met |
| 1.6 | MFA turned off for a person | `mfa_exempt` is one column, set by `POST`/`PUT /user` and `/userInvite` (`ADD_USER`, `CHG_USER`). **Policy: `mfa_exempt` is for test and service logins only, never a human account in production.** Enforced by config (GRC backport B097): unless `ALLOW_MFA_EXEMPT_VIA_API=true`, which only the test harness files set (pinned by `TestLoadConfig_RepoEnvFiles`), the API answers 403 to turning it on (`TestUserHandler_MFAExemptNeedsTheConfigKey`, `TestCreateInvitedUser_MFAExemptNeedsTheConfigKey`). A login already exempt stays so until cleared; the post-sales export clears every one (`src/api-srv/sql/export-scrub.sql`, pinned by `src/api-srv/sql/export_scrub_test.go`), so a post-sales host starts with none. A presales UAT host restored from the laptop dump keeps the seeded exempt logins (row 3.5) | met |
| 1.7 | Cross-site request forgery | The defence is the cookie: `SameSite=Strict` stops a cross-site request carrying it and `HttpOnly` keeps script from reading it; CORS answers only `CORS_ALLOWED_ORIGINS`. `X-CSRF-Token` must equal the session id's last 3 hex characters (12 bits) on every unsafe method: a second check, not a secret of real strength. A mismatch is logged without either value (`csrf_test.go`) | met (by SameSite) |
| 1.8 | Missing or misordered check on a route | Every handler calls JWT, rate limit + block, CSRF, grant and `CheckAllChecksComplete` itself, no middleware; an unrun check is 500 and `WriteJSON` re-runs the gate. `src/api-srv/internal/handler/route_check_contract_test.go` lists every route in `cmd/server/main.go`, requires a reason for each skip and proves each check refuses before any repository call | met |
| 1.9 | Blocked user keeps acting | Block sets `account_status`, revokes every session and sets `BLOCKED*<name>`, which the server reloads at start; `CheckRateLimitAndUserBlock` answers 403 on every session route, so a pre-block token is refused after a restart that lost its revocation. `/mfaSetup` and `/mfaVerify` take no session at all: only `/auth`'s temp token, which a blocked login never gets; a session without one is 401 (GRC backport C004, `TestMFARoutesRefuseASessionWithoutATempToken`). Open: the `admin` subject skips the flag (row 1.10) | partial |
| 1.10 | Request flooding | Sliding 60 s rate window (`rateWindow`: this minute's bucket plus the last minute's, weighted, so a burst across a minute boundary gets no second allowance; `TestRateWindowCatchesTheMinuteBoundaryBurst`; GRC backport C016) over absolute-minute buckets: `RPM*<user>*<minute>` signed in, `RPM*ip:<address>*<minute>` anonymous (the `ip:` prefix keeps the two apart); an unreadable bucket fails closed; body caps per route, nginx `client_max_body_size 20M`; `http.Server` read, write and idle timeouts (doc/105 Server Timeouts; GRC backport C015). Open: the `admin` subject skips the rate limit and block flag | partial |
| 1.11 | Spoofed client address defeats lockout and rate keys | `ClientIP` takes the address `TRUSTED_PROXY_COUNT` hops from the right of `X-Forwarded-For` (1 behind nginx, 0 in dev). Sound only because the API cannot be reached around nginx: `SERVER_BIND_ADDRESS` is required (startup fails without it) and every server env file sets `127.0.0.1` | met |
| 1.12 | Invite link stolen or guessed | 32 random bytes; only the SHA-256 is stored; the token travels in the URL fragment; `INVITE_TTL_HOURS`; single redeem under `FOR UPDATE`; per-address and per-token lockout; every dead link one counted 404; the redeeming address is recorded | met |
| 1.13 | API response rendered as a page | CSP `default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'`, `X-Frame-Options: DENY`, nosniff on every API response, `no-store` on every GET but `/health`; nginx sets its own headers on web locations only. HSTS is sent with `preload` even when `ENABLE_TLS=false` | met |
| 1.14 | Public self-registration | `/userSelfCreate` makes a tenant-1 login in group 3 behind the per-address rate limit only. Kept in the template beside invite links; a derived app removes it or keeps it knowingly | open (by design) |

### B2: API to Postgres

| # | Threat | Control | Status |
|---|---|---|---|
| 2.1 | Missing tenant predicate | See Tenant isolation: app-layer only, `scoped_query.go`, write predicates, test tenant lock, per-table tests (`example_repository_test.go`) | met |
| 2.2 | API used to change the schema | `proto_app` is `db_reader` + `db_writer`, owns nothing; migrations run as `proto_ddl` (`src/api-srv/cmd/migrate`), the only path for DDL | met |
| 2.3 | Credentials read from a dump or a response | Passwords Argon2id; TOTP seeds AES-GCM under `TOTP_ENCRYPTION_KEY`; invite tokens SHA-256 only; `PwHash` and `TOTPSeed` are `json:"-"`. `RetrieveUserList` still selects `totp_seed` it never sends | met |
| 2.4 | Database reached off the host | `DATABASE_URL` names `127.0.0.1:5432` in every server env file | met |
| 2.5 | Repo password on a database role | `src/api-srv/sql/roles.sql` creates `proto_app` with the local dev password; presales `bin/610-deploy-uat` then sets it from the host env file on every run, post-sales `infra/sql/proto-host-db.sql` takes it from psql variables | met |

### B3: operations

| # | Threat | Control | Status |
|---|---|---|---|
| 3.1 | Compromised API process reaches the host | Post-sales units (`infra/prd/etc/prd-proto-api.service`, UAT the same) run as `svc-proto` with a read-only filesystem (`ProtectSystem=strict`, no write path), empty `CapabilityBoundingSet`, `SystemCallFilter=@system-service`, `MemoryDenyWriteExecute` and the rest of the kernel and namespace protections. The presales unit `bin/610-deploy-uat` writes is lighter: `NoNewPrivileges`, `PrivateTmp`, `ProtectSystem=strict`, `ProtectHome`, `ReadWritePaths` on `log/` | partial |
| 3.2 | Secrets leak from the repo | Server env files are redacted copies (`SERVERONLY`, `example.com`); each secret is generated on its host; the API env file is `root:svc-proto 0640` | met |
| 3.3 | Losing the TOTP key with the data | Every API env header says to keep `TOTP_ENCRYPTION_KEY` with every backup and restore the two together; the nightly tarball carries `etc/`, so it holds the key | met |
| 3.4 | Backups destroyed or read from the primary | `infra/prd/bin/prd-proto-backup-all` pushes dumps and the `etc/` tarball to `REMOTE_SERVER` by `scp`, so the primary holds credentials that can overwrite its own off-host copies, and the copies hold every secret. **Target: pull** — the backup host fetches with a key the primary never holds | open |
| 3.5 | Laptop credentials or test data reach a customer host | `bin/620-export-tenant-data` copies the database, runs `src/api-srv/sql/export-scrub.sql` (refuses tenant-2 rows and the fixture logins; empties every password, seed and `mfa_exempt`; every live login becomes `invited`; open invites revoked; issue reports, with their tracker ids, removed) and dumps the copy; the host's first sign-in is a link from `proto-migrate invite` (doc/330 A9: the migrator's DDL credentials, so host root only; refuses tenant 2 and any login not `invited`; `src/api-srv/cmd/migrate/invite_test.go`). Presales `bin/600-bundle-uat` still ships the laptop dump unscrubbed (presales only, by decision), seeded exempt logins included; a host's API cannot add new ones (`ALLOW_MFA_EXEMPT_VIA_API=false`, B097) | partial |
| 3.6 | Monitoring silenced by the host it watches | Nackmon and pulsecheck files for UAT live under `infra/prd/shrmpl-etc/`; prod's own stay on prod for want of a third host (`infra/README.md`) | partial |
| 3.7 | Scanning the web root | fail2ban jail per app on nginx 404s, ignoring `/api/` so a user's 4xx never bans them (`infra/prd/fail2ban/etc/jail.local`) | met |
| 3.8 | Operator access unrecorded | Nothing in the template logs SSH, console or database-superuser sessions off the host | open |

## Controls a derived app keeps

Testable statements, each with what proves it today.

1. **Isolation per table.** Every tenant table ships with a cross-tenant
   negative test at the repository layer, and at the API where a route
   reaches it; another tenant's id answers not-found, never forbidden or
   conflict. Today: three of four tables (above).
2. **Every route states its checks.** A new route fails
   `TestRouteContract_EveryRouteIsListed` until it has a row in
   `routeContractList`; a skip needs a reason.
3. **Invite and reset links.** At least 128 random bits, stored hashed,
   single use under a row lock, expiring, rate limited per address and per
   token. Met by invite links; there is no reset link.
4. **Local-auth hygiene.** Lockout per address and per name, failing
   closed; TOTP for every login that is not exempt; `mfa_exempt` never on
   a human account in production. All three are met: the API turns the
   flag on only where `ALLOW_MFA_EXEMPT_VIA_API=true`, which only the test
   harness files set (GRC backport B097). No breached-password check; the password floor is 8 characters.
5. **Rate limits per boundary.** Per user, per address, per login name and
   per invite token. The `admin` exemption is open.
6. **Backups pulled, encrypted, restored on a schedule.** Open: push today,
   unencrypted beyond the host's disk, no recorded restore test.
7. **User files judged by their bytes.** A file route caps size (413) and
   takes its type from magic bytes against an allowlist (415), serves
   inline only what cannot run script, relaxes framing on that route alone
   and answers every miss 404 (doc/105, Serving user files). No file route
   exists today.

Not carried: an audit log (GRC backport B067, skipped: no audit log in the
template) and host-access logging (3.8, an operator practice before it is
code).
