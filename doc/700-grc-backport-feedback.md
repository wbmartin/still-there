# GRC backport feedback

Decision list of generic improvements made in radley-grc (GRC) that
ProtoStack (PROTO) lacks. Not a plan: a session with Brandon walks it item
by item and ticks **Decision**; tickets come out of that session.

- **PROTO:** `/Users/brandon/Code/ProtoStack-ReactGo` @ `c7da74d` (`main`).
- **GRC:** `/Users/brandon/Code/radley-grc` @ `bedf958` (branch `cleanup`),
derived from PROTO `0c2130a` on 2026-09-17; it already pulled
PS0001-PS0021 as RG0001-RG0018, so only GRC-side changes are here.
- **Date:** 2026-10-02. **Sources:** four read-only reviews (go-api,
web-app, ops, docs), deduped: each finding once, docs under their code.

**How to read.** B-numbers run P1 (security, real PROTO bugs), P2 (template
quality, DX, docs), P3 (nice to have), then PROTO-only bugs, SKIP/veto
list, open-in-both appendix, do-not-copy list, summary. **GRC** = source,
**PROTO** = target, both at the commits above. Web paths are under
`src/web-app/` unless they start `src/`, `tests/`, `tickets/`. Policy
questions read "Brandon's call". `RGnnnn md :x` = that ticket's body file
in GRC `tickets/`.

**Path mapping.** GRC `src/internal/X` == PROTO `src/api-srv/internal/X`;
GRC `src/migrate-srv` (goose) \~ PROTO `cmd/migrate` + `internal/migrate`;
GRC `infra/uat-unified001` \~ `infra/uat`, `infra/prd-unified002` \~
`infra/prd`; GRC `App.tsx` == PROTO `app.tsx`; module paths differ. Ports
derive names from `etc/app.env` (`APP_SLUG`) and `etc/uat-deploy.env`.

**Expansions.** Every item from B004 on carries an "Expanded" block under
its Decision line with four parts: what is wrong in PROTO today (with code
excerpts), what GRC changed, the proposed fix for PROTO as numbered steps,
and an open question for Brandon. Where the expansion found the original
one-paragraph entry wrong or overstated, the expansion says so and the
expansion wins.

**Ground rules.** GRC-only code (no PROTO equivalent path) was not reviewed
unless clearly generic. All GRC docs were considered. A GRC deletion does
not imply a PROTO deletion. Reviewer disagreements are stated.

**Line numbers corrected here:** `user_repository.go:57` -&gt; `:62` (B007);
`doc/340-db-host-setup.md:40-42` -&gt; `:65-66` (B050); `profile_handler.go:147`
-&gt; `:146` (B030); retry "33 call sites" -&gt; 31 call lines (B028).

## Start here

1. **B001 MFA (RG0042):** `/mfaSetup` hands an enrolled user's TOTP secret
 to a password-only caller. Go + web + pytest. Highest priority.
2. **B003-B012 security cleanups** (RG0002, RG0019, RG0043): revocation,
 refresh/CSRF drift, context-switch CSRF, 403 sign-out, `/auth` oracle,
 fail-closed counters, TOTP replay, PS0021, body caps, 409 oracle.
 `LOG_LEVEL` is B031.
3. **B002** public `/userSelfCreate` (Brandon's call); invites are B035.
4. **B014** govulncheck; **B018** PS0022 tenant 2 (GRC RG0041 did it).
5. **B020-B024 infra bugs:** backup, cert renewal, nginx headers, fail2ban
 on `/api/`, 1775 app root for publish.
6. **B025** left nav scrolls independently (AppShell + ScrollArea).
7. **B026** server deploy runbooks: doc/330, 331, 340 + `infra/sql`.
8. **B085-B090** PROTO doc bugs, incl. a real hostname in doc/902.

## Decision queue (2026-10-02, Brandon's marks)

Pull = queued, not started. Adapt = expanded below under the item with
"What is wrong", "What GRC changed", "Adapted fix". Skip = closed.

| Item | Decision | Note |
|---|---|---|
| B001 MFA setup secret leak | pull | queued |
| B002 retire `/userSelfCreate` | skip | template keeps self-registration |
| B003 session revocation | pull | queued |
| B004 refresh keeps session id, CSRF resync | adapt | expanded; identity mismatch reloads tab |
| B005 `/changeSecurityContext` CSRF | skip | |
| B006 403 signs the user out | adapt | expanded; reverse PS0001, 401 only |
| B007 `/auth` username oracle | adapt | expanded; XFF fix in same batch |
| B008 fail closed on counter write | pull | queued |

B009-B039 marked by Brandon in the doc. B040-B090 marked 2026-10-02: nine
by Brandon (B046 pull, B049 skip, B050 skip, B057 skip, B067 skip, B068
skip, B071 adapt post-sales only, B074 pull, B079 skip), the rest "per
recommendation". Unmarked B028 taken per recommendation (pull).
**B091 (new): remove the ticket system from ProtoStack.** It trumps every
ticket-related part of any other item.

# P1: security and real bugs in PROTO

### B002 — Retire public `POST /userSelfCreate`

- **Priority:** P1  **Area:** go-api, web-app, tests, docs
- **GRC:** `src/api-srv/cmd/server/main.go:180-187`; `src/api-srv/README.md:15-21`; `pages/login.tsx:83-85`; `tests/test_user_invite.py:174-179`; `tickets/03-done/RG0035-user-invite-link/RG0035-user-invite-link.md:29-31,57`   **PROTO:** `src/api-srv/internal/handler/user_mfa_handler.go:23-177` (`SkipJWT` :39, uncapped :54, tenant 1 :127-135); `src/api-srv/cmd/server/main.go:107-110,153-154`; `pages/login.tsx:20,88-103`; `components/user-registration-modal.tsx:50`; `constants/api.ts:22`; `src/api-srv/README.md:55`
- **Ticket:** RG0035 / `7001cbd` (Brandon ruling 2026-09-30); new PS ticket
- **What:** An unauthenticated, CSRF-skipped, uncapped route creates a tenant-1
login (group 3) and answers 409 "Username already taken" (enumeration). GRC
removed it and the "Create one here" link; admins create or invite (B035).
- **Port notes:** Brandon's call: template self-registration? If kept: opt-in,
capped (B011), generic answer.
- **Decision:** \[ \] pull / \[ \] adapt / \[ X\] skip

### B005 — `/changeSecurityContext` skipped CSRF and parsed the body first

- **Priority:** P1  **Area:** go-api, web-app, tests, docs
- **GRC:** `src/internal/handler/auth_handler.go:633-680`; `auth_handler_test.go:285-456`; `tests/test_auth_flow.py:152-223`; `stores/auth-store.ts:212-217`; `tickets/03-done/RG0019-change-security-context-csrf/RG0019-change-security-context-csrf.md:39-59,89-111`   **PROTO:** `src/api-srv/internal/handler/auth_handler.go:504-536` (decode :516, `SkipCSRF` :530, new id :622,662); `stores/auth-store.ts:173-197`; `doc/105-api-srv-tech-spec.md:193-204`
- **Ticket:** RG0019 / `fe0f53d`, `25ef2df`, `5f57d0c`, `8fa36b6`
- **What:** PROTO decodes first and skips CSRF on a route that mints a session
in another tenant; the web keeps the old token, so writes 403 after a switch.
GRC: JWT, rate/block, CSRF, gate, then decode; web adopts the new `csrfToken`.
- **Port notes:** Web line first. Pytest needs a tenant-3 group (B018).
doc/105:199: "no row written" is no reason to skip.
- **Decision:** \[ \] pull / \[ \] adapt / \[X \] skip
- BRANDON: I said skip because it's a not changing SERVER side data so seems fine to skip CSRF

### B015 — Literal service-account password shipped in `roles.sql`

- **Priority:** P1  **Area:** go-api, bin
- **GRC:** `src/migrate-srv/sql/bootstrap/010-database.sql:3-11`; `src/migrate-srv/cmd/migrate/bootstrap.go:71-93`; `main.go:149-157`; `bootstrap_test.go:6-20`   **PROTO:** `src/api-srv/sql/roles.sql:30-40` (:37); `bin/600-bundle-uat:97`; `bin/610-deploy-uat:110`
- **Ticket:** RG0043 item 43 / `720f5e1`
- **What:** PROTO's UAT path creates `proto_app` with the repo literal. GRC
ships none; bootstrap fills it from `DATABASE_URL`.
- **Port notes:** No Go bootstrap: `psql -v svc_pw=...` from `etc/*.env` in
`bin/510`/`610`, `:'svc_pw'` in `roles.sql`. Pairs B027.
- **Decision:** \[ \] pull / \[ \] adapt / \[X \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

`src/api-srv/sql/roles.sql` creates the API's login role with a password
written in the repo:

```sql
-- src/api-srv/sql/roles.sql:36-37
CREATE USER proto_app
    WITH PASSWORD 'reallygoodPSQLpassword';
```

The comment at `:10-11` says this is "the local default and is NOT what
any deployed environment connects as". That is false for the presales UAT
path, which ships this exact file to the server and runs it:

- `bin/600-bundle-uat:97` copies it into the bundle: `cp "$ROLES_SQL"
  "$STAGE/000-roles.sql"`.
- `bin/610-deploy-uat:110` runs it on the host as superuser:
  `sudo -u postgres psql -q -v ON_ERROR_STOP=1 -f 000-roles.sql`.
- `bin/610-deploy-uat:115` then makes that role own the database.

So on a fresh UAT host, `proto_app` logs in with a password anyone with
the template (or any derived repo) knows. The operator is told the
opposite: `doc/320-presales-uat-deploy.md:76-77` says to put "a real
value (DB password ...)" in the host env, and `:82-83` says 610 "creates
the role ... itself". Nothing in `bin/`, `infra/` or `doc/` runs
`ALTER ROLE`, and the `IF NOT EXISTS` at `roles.sql:33-35` never touches
an existing role. The operator ends up in one of two states:

1. They follow the doc and set a real password in
   `etc/uat-api-srv.env.example:11` (`proto_app:SERVERONLY@...`). The
   role still has the literal, so the API cannot connect. Correctness bug.
2. They notice and put the literal in the env file. The API works, and
   the UAT database is open to anyone who can reach Postgres on that host
   with the published password. Security hole.

It also leaks forward. `infra/README.md:149` runs `roles.sql` on
post-sales hosts, and `doc/330-presales-to-postsales-migration.md:115-116`
says `proto_app` "keeps its name and password" through the cutover.
`DERIVING.md:40` lists the literal under "nothing forces this", so a
derived app that misses that table ships it.

Postgres on these hosts listens on 127.0.0.1, so the reach is local
users, other services on the box and anything that gains SSRF or a shell.
Kind: security hole (credential in source) plus a deploy correctness bug.

#### What GRC changed

GRC fixed this twice, once per path.

**Bootstrap (local / migrate-srv).** RG0043 item 43 (md `:1179-1186`,
commit `720f5e1`, "2.19 bootstrap service password from DATABASE_URL").
The SQL carries a placeholder
(`src/migrate-srv/sql/bootstrap/010-database.sql:11`):

```sql
CREATE USER radley_grc_svc_account WITH PASSWORD {{SERVICE_PASSWORD}};
```

`servicePasswordLiteral` (`src/migrate-srv/cmd/migrate/bootstrap.go:80-93`)
parses the env file's `DATABASE_URL`, refuses a URL naming another role
or carrying no password, and returns it as a quoted SQL literal (`'`
doubled, `:92`). `main.go:149-157` substitutes it and prints statements
without the password. Test: `TestServicePasswordLiteral`
(`cmd/migrate/bootstrap_test.go:6-20`) covers quoting and three refusals.
Limit: an existing role is skipped (`main.go:157`, "skip (exists)"), so
bootstrap does not rotate a password.

**Presales UAT.** GRC's `bin/610-deploy-uat:46-71` (RG0012, RG0013,
`55b0066`) generates the password on the host the first time
(`DB_PASSWORD="$(openssl rand -hex 32)"`, `:54`), writes it into the env
file (0640, `:58`), and runs CREATE or ALTER (`:64-68`). GRC's 600 no
longer copies a roles file; its header comment at `bin/600-bundle-uat:8`
still says `000-roles.sql` (stale).

#### Proposed fix for PROTO

PROTO has no Go bootstrap command, so do it in psql, as the doc's port
note says. One trap in that note: `:'svc_pw'` cannot sit inside the
`DO $$ ... $$` block at `roles.sql:31-40`, because psql does not
substitute variables inside dollar-quoted bodies (PROTO's own
`bin/610-deploy-uat:118` says so). Use `\gexec` instead.

1. `roles.sql:29-40`: replace the `DO` block with two statements outside
   any `$$`:
   ```sql
   SELECT format('CREATE ROLE proto_app LOGIN PASSWORD %L', :'svc_pw')
   WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'proto_app')
   \gexec
   ALTER ROLE proto_app PASSWORD :'svc_pw';
   ```
   An unset `svc_pw` leaves the text `:'svc_pw'` in place, a syntax error
   under `ON_ERROR_STOP`, so it fails closed. Fix the `:10-13` comment.
   Keep the literal role name; `bin/000-rebrand` rewrites it and
   `bin/600-bundle-uat:52` greps the file for `DB_APP_ROLE`.
2. Pass the value on stdin, not argv (`ps` shows argv; B026's paste
   rules say "no argv passwords"): prepend `\set svc_pw '<pw>'` to the
   file on psql's stdin. Refuse a password containing `'` or a newline.
3. `bin/510-reset-test-db:96`: read the password from the `DATABASE_URL`
   in `etc/loc-api-srv.env` (the local value stays in `etc/`, where
   `DERIVING.md:40` already lists it).
4. `bin/610-deploy-uat:110`: read it from the host env file's
   `DATABASE_URL` (already parsed for `DB_NAME` at `:54`). Move the roles
   step out of the `DO_DB` block (`:107`) so it runs on every deploy,
   `--no-db` included, and the `ALTER` keeps the role equal to the env.
   B027(b) later generates that value on the host, as GRC's 610 does.
5. Docs: `doc/050-dev-env-init.md:25`, `infra/README.md:149`,
   `doc/320-presales-uat-deploy.md:82-83`, `DERIVING.md:40` (drop
   `roles.sql` from the row), `doc/330...:115-116`.
6. Blind-copy traps: no `{{SERVICE_PASSWORD}}` Go substitution, no
   `radley_grc_svc_account`, no `migrate-srv bootstrap` command.
7. Tests: `bin/300` (runs `bin/510`) proves local; one manual 610 run
   proves CREATE then ALTER. No Go code, so no Go test.

#### Open question for Brandon

Should `bin/610` ALTER `proto_app` on every deploy, which rotates the
published password off UAT hosts that already have it but makes the host
env file the only source of truth? Recommendation: yes, and check the
existing UAT host once by hand in the same ticket.

# P2: template quality, DX, docs

### B028 — PS0010: one sentinel set + `mapRepoError`; delete `retry.go`

- **Priority:** P2  **Area:** go-api, tests, docs
- **GRC:** `src/internal/repository/errors.go:10-75`; `src/internal/handler/common.go:94-160`; `user_repository.go:66-68`; `handler/map_repo_error_test.go:16-125`; `repository/errors_test.go:16-91`; `doc/105-api-srv-tech-spec.md:196-198,470-501,522-531`; RG0014 md `:175`   **PROTO:** `src/api-srv/internal/repository/errors.go:1-27`; `strings.Contains` at `handler/user_handler.go:459-470`, `app_user_group_handler.go:341-359`, `user_mfa_handler.go:91-97`; `user_handler.go:587-620,689-714`; `app_user_group_handler.go:237-265`; `repository/retry.go:1-68`; `go.mod:6`; `doc/105-api-srv-tech-spec.md:114-119,259-281`; `doc/810-create-new-entity-type-notes.md:57`
- **Ticket:** RG0014 / `4796119`, `fe914fc`, `9ce276b`; PS0010
- **What:** PROTO picks statuses by error text; `ErrConflict` never fires. GRC:
sentinels, one mapper, `ClientMessageError`. `retry.go` never retries anything
pgx returns (31 call lines); GRC deleted it.
- **Port notes:** Prereq for B007; apply to the example handler. Fix
doc/105:279, doc/810:57. `pgconn.SafeToRetry` skipped on purpose.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

All PROTO paths are under `src/api-srv/internal/`. Three correctness and
quality gaps, none a security hole.

**(1) Text, not sentinels.** `repository/errors.go:14-20` defines
`ErrNotFound`, `ErrConflict`, `ErrDeleted`, `ErrDuplicate`. The user
repository skips them at `user_repository.go:62,207`
(`fmt.Errorf("user not found")`) and `:236,265,294` ("user not found or
concurrent modification").

Handlers match strings at `handler/user_mfa_handler.go:94`,
`handler/user_handler.go:460` and `handler/app_user_group_handler.go:342`
(text from `app_user_group_repository.go:237`). Rewording a message
silently changes a status code.

**(2) Membership writes answer a stale lock with 500.** The doc's
"`ErrConflict` never fires" overstates it: user, group, profile and example
updates return it and get 409 (`user_repository.go:420`,
`user_handler.go:589`). The gap is narrower: `AddUserListToGroup` and
`RemoveUserListFromGroup` return text on zero rows
(`user_repository.go:518-521,579-580`, "user %d not found, already has
group, or concurrent update"). Their handlers test `errors.Is(err,
repository.ErrConflict)` (`user_handler.go:822,939`), never true, so a
stale `updateTs` on a membership change answers 500, not 409.

**(3) The retry helper never retries.** `ExecuteWithRetry` wraps 31 calls
(15 user, 8 group, 8 example). `isRetryableError`
(`repository/retry.go:36-45`) matches `database/sql` sentinels pgx never
produces and compares `err.Error()` to four exact strings pgx always
wraps, so every error is `backoff.Permanent` on the first try. Had it
matched, a 1 s first wait (`:23`) would sit on a request path.
`doc/105-api-srv-tech-spec.md:278-281` and
`doc/810-create-new-entity-type-notes.md:57` tell readers to use it.

Each handler also repeats a 3-4 arm `errors.Is` switch
(`user_handler.go:587-620,689-714`, `app_user_group_handler.go:237-265`).
`tickets/01-open/PS0010-api-entity-template.md` lists all of this; Brandon
deferred it (`:128-133`).

#### What GRC changed

RG0014 (`4796119`, `9ce276b`) and RG0015 (`fe914fc`):

1. `src/internal/repository/errors.go:15-38` adds `ErrInUse`,
   `ErrValidation`; `ClientMessageError` (`:46-63`) pairs a sentinel with
   a wire-safe message. Membership zero rows wrap `ErrConflict`.
2. `src/internal/handler/common.go:111-148`: `mapRepoError(err)` returns
   (status, code, msg): 400 validation, 409 duplicate / in-use /
   conflict, 404 for `ErrNotFound` and `ErrDeleted`, else 500, where the
   caller keeps its own wording and log code
   (`app_user_group_handler.go:153-160`).
3. No `strings.Contains` left; `user_repository.go:66-68` returns
   `ErrNotFound`; RG0015 routed eight MFA sites through the mapper.
4. `retry.go` and `cenkalti/backoff/v4` deleted. `pgconn.SafeToRetry`
   rejected on purpose: pgxpool reconnects. Documented at
   `doc/105-api-srv-tech-spec.md:470-501,522-531`.

Tests: `handler/map_repo_error_test.go:16,91` (`TestMapRepoError`, the
message case); `repository/errors_test.go:40,57`.

#### Proposed fix for PROTO

1. **`repository/errors.go`**: add `ErrInUse`, `ErrValidation`,
   `ClientMessageError`, `clientMessage`; not GRC's `ErrNoAnchor`.
2. **`repository/user_repository.go`**: `:62,207` return `ErrNotFound`
   (B007 may wrap `:62` first). `:236,265,294`: existence check, then
   `ErrNotFound` or `ErrConflict`. `:518-521,579-580`: wrap `ErrConflict`.
   `CreateUser`'s name rule: `clientMessage(ErrValidation, ...)`.
3. **`app_user_group_repository.go:237`**: `clientMessage(ErrInUse, ...)`.
4. **`handler/common.go`**: add `mapRepoError`. Use it in user, group,
   MFA and profile handlers; delete the three `strings.Contains`.
5. **Example handler** answers `ErrDeleted` with 410
   (`example_handler.go:117-120,339-342,457-460`; doc/105:269,276). GRC
   maps it to 404. See the question below before copying that row.
6. **Delete `repository/retry.go`**; unwrap the 31 calls; drop
   `cenkalti/backoff/v4` from `go.mod:6`; `go mod tidy`.
7. **Docs**: rewrite doc/105 `:278-281` (no app retry) and add the
   sentinel table; fix doc/810 `:57`.
8. Traps: module is `github.com/wbmartin/starter-api`; helper is `app`,
   not `appHelper`; GRC's A-002 refs and `writeRepoError` do not apply.
9. Tests. Go: port both GRC test files; a repository test that a stale
   membership `updateTs` is `ErrConflict`. pytest: that case answers 409.
   Reopens a slice of PS0010; B007 needs step 2's `:62` change.

#### Open question for Brandon

Soft-deleted row: 410 Gone (PROTO example and doc/105 today) or 404 like
a missing row (GRC)? Recommendation: 404; 410 tells a caller the id once
existed, and no web code acts on 410.
Brandon: 404

### B029 — RG0004: one JSON writer; session cookie without `Max-Age`

- **Priority:** P2  **Area:** go-api, tests, docs
- **GRC:** `src/internal/handler/auth_handler.go:415,466,493,600,619,785,806,849,1033`; `user_handler.go:520`; `app_user_group_handler.go:166`; `common.go:58-92`; `user_mfa_handler.go:533`; `tests/test_auth_flow.py:255-303`; `doc/105-api-srv-tech-spec.md:363-377`   **PROTO:** `src/api-srv/internal/handler/common.go:14-25`; `auth_handler.go:28-41,300,371,474,493,643,664,698,867`; `user_handler.go:493`; `app_user_group_handler.go:164`; doc/105 after `:209`
- **Ticket:** RG0004 / `d761d42`, `9fe4a78`; RG0043 item 34 / `720f5e1`
- **What:** (a) `handler.writeJSON` skips the gate on 2xx; GRC has one writer.
(b) `Max-Age=3600` on refresh/switch against a 72 h JWT kills refreshed
sessions after an hour; GRC: JWT expiry is the only lifetime.
- **Port notes:** `setAuthCookie` + named constants into `common.go`. Pytest: no
Max-Age, logoff 0. Tests rated the cookie P1.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

All PROTO paths are under `src/api-srv/internal/`. Two separate problems.

**(a) A second JSON writer skips the check gate.** `AppHelper.WriteJSON`
refuses a 2xx when the security checks did not all run:

```go
// security/app_helper.go:214, 219
if status < 400 && !ah.CheckAllChecksComplete() {
    http.Error(w, "Security check failed", http.StatusInternalServerError)
```

`handler/common.go:15-24` defines a package-level `writeJSON` that writes
headers and body with no check. Nine success paths use it:
`auth_handler.go:300,372,493,664,867`, `user_handler.go:492`,
`app_user_group_handler.go:163`, `example_handler.go:222,685`. Each runs
after its handler's own `CheckAllChecksComplete` with `app` in scope, so
nothing leaks today. The example handler is the copy-me template, so a
new handler copied from it that drops the gate still answers 200.
Defense-in-depth gap, not a live hole.

**(b) Refresh and switch cut the session to one hour.** One helper writes
the cookie (`auth_handler.go:28-41`); callers disagree on lifetime:

```go
// auth_handler.go:351 (login), :474 (refresh), :643 (switch), :698 (logoff)
setAuthCookie(w, jwt, 0)
setAuthCookie(w, jwt, 3600) // 1 hour
setAuthCookie(w, "", -1) // expire immediately
```

`user_mfa_handler.go:619` (MFA verify) also passes 0. The JWT inside lives
`JWT_SESSION_EXPIRE_HOURS=72` (`etc/loc-api-srv.env:13`, same in
`etc/uat-api-srv.env.example:26`). The web calls `/refresh` on every page
load (`src/web-app/stores/auth-store.ts:105`, from
`hooks/use-auth-guard.ts:15-20`). Log in, reload once, work an hour
without reloading: the browser drops the cookie, the next call 401s, and
the user lands on login holding a valid 72-hour token. UX and correctness
bug, not a security hole: the server enforces the JWT's `exp` anyway.

#### What GRC changed

RG0043 item 34 (`720f5e1`) deleted `handler.writeJSON`; its seven callers
call `appHelper.WriteJSON` (`src/internal/handler/auth_handler.go:415,496,
619,806,1033`, `user_handler.go:519`, `app_user_group_handler.go:165`).

RG0004 (`d761d42`) moved `setAuthCookie` to
`src/internal/handler/common.go:79-89` with named values (`:93-96`):

```go
authCookieSessionMaxAge = 0
authCookieExpireNow     = -1
```

Every issuing path passes the session value (`auth_handler.go:466,600,785`,
`user_mfa_handler.go:533`); only logoff expires (`auth_handler.go:848`).
The comment at `common.go:70-78` and `doc/105-api-srv-tech-spec.md:363-376`
state the rule: the JWT's expiry is the only session lifetime. `9fe4a78`
added `test_session_cookie_lifetime_is_the_jwt_expiry`
(`tests/test_auth_flow.py:255-303`): `/auth` and `/refresh` carry no
`Max-Age` and keep `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/`;
`/logoff` carries `Max-Age=0`. No Go test covers either change.

#### Proposed fix for PROTO

1. **`handler/common.go`**: delete `writeJSON` (`:14-24`) and unused
   imports; the nine sites call `app.WriteJSON(w, status, wrapper)`.
2. **Move `setAuthCookie`** from `auth_handler.go:28-41` to `common.go`.
   Add `authCookieName = "auth"`, `authCookieSessionMaxAge = 0`,
   `authCookieExpireNow = -1` (`security/app_helper.go:280` keeps its own
   `"auth"` literal; it is another package).
3. **`auth_handler.go:474,643`**: pass `authCookieSessionMaxAge`; drop the
   `// 1 hour` comments. Swap the literals at `:351,698` and
   `user_mfa_handler.go:619` for the constants.
4. **B004 edits the same `/refresh` code** (`:454-474`); land B029 first
   or in the same branch.
5. **doc/105**: add a "Session cookie" section after `:209` stating the
   rule, adapted from GRC `doc/105:363-376` without the RG ticket refs.
6. Keep PROTO's module path and `app` helper name, not GRC's `appHelper`.
7. Tests. pytest in `tests/test_auth_flow.py`: port GRC's cookie test and
   add `/changeSecurityContext` (GRC skipped it; needs a two-tenant user,
   see B030). Go: none for (a); the compiler rejects any leftover caller.

#### Open question for Brandon

Browser-session cookie (gone when the browser closes, as login is today)
or persistent with `Max-Age = JWT_SESSION_EXPIRE_HOURS * 3600`?
Recommendation: browser-session, as GRC chose; it matches login today and
needs no new config.

### B030 — `/me` 404s after a context switch

- **Priority:** P2  **Area:** go-api, tests
- **GRC:** `src/internal/handler/own_profile_handler.go:142-226` (:189, :211); `repository/user_repository.go:44-56,521-557`; `own_profile_check_sequence_test.go:155-454`   **PROTO:** `src/api-srv/internal/handler/profile_handler.go:55-90,92-178` (:72-73, :146, :168); `repository/user_repository.go:614-709`
- **Ticket:** RG0016 / `62403fd`
- **What:** `/me` reads and writes in the JWT's active tenant, so a switched
user gets 404. GRC: row by subject, write in the row's tenant, 200-char
fields, preference shape check.
- **Port notes:** Keep PROTO's `ProfileHandler`; change lookup and scope.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

All PROTO paths are under `src/api-srv/internal/`. A user has one
`app_user` row in one home tenant; groups can reach other tenants.
`/changeSecurityContext` mints a JWT whose `SecurityContext` is the picked
tenant (`handler/auth_handler.go:618-619`); the row stays home. `/me` then
looks the row up in the JWT's active tenant:

```go
// handler/profile_handler.go:72-73 (GET), same at :146-147 (PUT)
user, err := h.userRepo.RetrieveUserByID(
    r.Context(), jwtClaims.UserID, jwtClaims.SecurityContext,
```

`repository/user_repository.go:626` and the PUT at `:678` filter on
`security_context_id`, so after a switch:

1. `GET /me` answers 404 (`profile_handler.go:76-78`). The web re-reads
   `/me` on every switch (`src/web-app/app.tsx:156-160`), logs the failure
   to the console and keeps the stale profile, so nothing looks wrong.
2. `PUT /me` hits zero rows and the existence check
   (`user_repository.go:696-705`) misses too: `ErrNotFound`. Save on the
   Profile page shows a red "User not found" toast; name, email and
   preferences cannot change until the user switches back. The re-read
   after the write (`profile_handler.go:168-170`) has the same scope.

Correctness bug, not a security hole: the lookup uses the signed JWT's
own `uid`. Nothing catches it: no seeded user holds groups in two tenants
(`src/api-srv/sql/fixtures/local-seed.sql:85-116`) and no Go, pytest or
Playwright test calls `/changeSecurityContext`.

Doc correction: "200-char fields, preference shape check" are not GRC
additions; PROTO has both (`profile_handler.go:126,139,190-198`). The
only gap: PROTO counts bytes (`len`), so a 101-character non-ASCII name is
refused though it fits `VARCHAR(200)`. GRC counts runes.

#### What GRC changed

GRC commit `62403fd` (RG0016) added the own-profile API. `PUT /me` reads
the caller's row by JWT subject, which is tenant-free because `user_name`
is unique installation-wide, and scopes the write by the row's own tenant
(`src/internal/handler/own_profile_handler.go:189-212`):

```go
user, err := h.userRepo.RetrieveUserByUserName(r.Context(),
    jwtClaims.Subject)
if user.AppUserID != jwtClaims.UserID { // 401 SESSION_MISMATCH
newUpdateTS, err := h.userRepo.UpdateOwnProfile(r.Context(),
    user.AppUserID, profile, req.UpdateTS, user.SecurityContext)
```

`src/internal/repository/user_repository.go:40-43` records the rule;
`UpdateOwnProfile` (`:521-556`) keeps `security_context_id` in the
predicate. GRC's `GET /me` (`grc_custom_handler.go:49-64`) is GRC-shaped
but reads the profile by user name too. Lengths use
`utf8.RuneCountInString` (`own_profile_handler.go:82`). Tests in
`src/internal/handler/own_profile_check_sequence_test.go`:
`TestOwnProfileWritesTheCallerSOwnRow` (`:359`, the row's tenant is
written, not the session's) and
`TestOwnProfileSessionNamingTwoUsersWritesNothing` (`:435`).

#### Proposed fix for PROTO

1. **`handler/profile_handler.go`**: both handlers read the row with
   `RetrieveUserByUserName(ctx, jwtClaims.Subject)`; if
   `user.AppUserID != jwtClaims.UserID`, 401 `SESSION_MISMATCH`, no write.
   Pass `user.SecurityContext` to `UpdateOwnProfile` and the re-read.
2. **`ProfileRepositoryInterface`** (`:32-41`): swap `RetrieveUserByID`
   for `RetrieveUserByUserName`; `/me` is its only caller, so delete it
   from `user_repository.go:612-658`.
3. **`repository/user_repository.go:61-62`** must return `ErrNotFound`:
   B007's one-line wrap, or B028's full set. Land one first.
4. Swap `len(...) > 200` at `profile_handler.go:126` for
   `utf8.RuneCountInString`.
5. Do not copy GRC's `GrcHandler.Me`, `personId`, roles, `account_status`
   or the `AppUserOwnProfileModel` wire split.
6. Tests. Go: extend `MockProfileRepository` in
   `handler/profile_handler_test.go:19`; assert a JWT with
   `SecurityContext` 1 and a row in tenant 2 gets 200 on GET and PUT and
   the write carries tenant 2; assert the uid/subject mismatch writes
   nothing. pytest: seed a user homed in tenant 2 with one group in a
   second tenant, switch, then GET and PUT `/me`, expect 200 (depends on
   B018's tenant-2 harness).

#### Open question for Brandon

Which tenant may the second group live in for the pytest, since tenant 2
is the only one tests may touch and a switch needs two? Recommendation:
seed a dedicated tenant (for example 3) used only by the context-switch
test, and record it beside the tenant-2 rule in CLAUDE.md and doc/900.

### B031 — `LOG_LEVEL`; per-request and health lines to debug

- **Priority:** P2  **Area:** go-api, etc, docs
- **GRC:** `src/internal/config/config.go:47-49,225-233`; `src/internal/logging/logging.go:38-63`; `src/api-srv/cmd/server/main.go:58-61`; `handler/health_handler.go:36,48`; `config/config_test.go:414`; `src/api-srv/README.md:176-178`   **PROTO:** `src/api-srv/internal/logging/logging.go:28-46` (:34); `config/config.go:73-213`; `cmd/server/main.go:48`; `handler/health_handler.go:33-50`; `etc/loc-api-srv.env:17-23`; `etc/e2e-api-srv.env`; `etc/uat-api-srv.env.example`; `infra/*/etc/*-api.env`
- **Ticket:** RG0043 item 46 / `720f5e1`
- **What:** PROTO logs at debug always, usernames per request, two Info lines
per health poll. GRC: optional `LOG_LEVEL` (debug|info|warn|error, default
info, else refused); health at Debug.
- **Port notes:** Keep PROTO's logger; add a level. Add the key to every env
file and doc/905 (GRC never did; its doc/105:435 is stale).
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

PROTO paths below are under `src/api-srv/`. The API has one logger, built
once in `main` with no level setting:

```go
// cmd/server/main.go:48
logger := logging.NewLogger(config.ServerName, os.Stderr)
// internal/logging/logging.go:32,34
h := slog.NewTextHandler(w, &slog.HandlerOptions{
    Level:     slog.LevelDebug,
```

Every Debug call in the code is therefore written in every environment,
UAT and production included. There is no config key to change it;
`internal/config/config.go` has no log field, and `doc/905` is four lines
that say nothing about logging.

What that writes, per request:

- `internal/security/app_helper.go:69`: `APP_HELPER_INIT` "request
  entered <handler>" on every request.
- `internal/security/app_helper.go:384-386`: `RLP0` "Rate limit check
  passed - key: ..., count: ..." on every rate-limited request.
- `internal/security/app_helper.go:114-121`: the AppHelper wrappers attach
  `username` to every line they write.
- Debug lines that carry data, not just flow:
  `internal/handler/auth_handler.go:163-168` logs the typed login name
  ("Auth request parsed for user: "+req.UserName), so a password pasted
  into the username box lands in the journal;
  `internal/handler/example_handler.go:140,220` logs whole rows with
  `%+v`; `internal/handler/user_handler.go:1080` logs the username on a
  password change.

The health route adds two Info lines on top of the Debug one:

```go
// internal/handler/health_handler.go:33-35, 45-47
app.Logger().Info(
    "HEALTH_REQUEST",
...
app.Logger().Info(
    "HEALTH_RESPONSE",
```

So each `/health` poll writes three lines. The UAT monitor polls every
180 s (`infra/uat/shrmpl-etc/pulsecheck-endpoints/uat-proto-api.env`,
`PULSE_INTERVAL_SEC=180`), and any external checker can poll faster.

Consequence for an operator: journald fills with routine lines, so a real
`Warn`/`Error` is harder to find, and retention shrinks. Row contents and
login names sit in the journal on a production host. This is a
quality/ops gap with a small privacy edge, not a security hole. The test
`internal/logging/logging_test.go:45` (`TestLogger_LevelsAndDebugNotDropped`)
pins "debug is always written" as intended behaviour.

#### What GRC changed

GRC commit `720f5e1` (RG0043 item 2.23, "LOG_LEVEL, per-request lines at
debug") did four things.

1. Config key, optional, default `info`, anything else refused at load
   (`src/internal/config/config.go:47-49` field,
   `src/internal/config/config.go:225-233` parse):
   ```go
   config.LogLevel = env["LOG_LEVEL"]
   if config.LogLevel == "" {
       config.LogLevel = "info"
   ```
2. `logging.ParseLevel` and `NewLoggerAtLevel`
   (`src/internal/logging/logging.go:38-63`); `main` uses them
   (`src/api-srv/cmd/server/main.go:58-59`):
   ```go
   logLevel, _ := logging.ParseLevel(config.LogLevel) // LoadConfig refused anything else
   logger := logging.NewLoggerAtLevel(config.ServerName, logLevel)
   ```
3. Health lines moved to Debug
   (`src/internal/handler/health_handler.go:36,48`), and
   `APP_HELPER_INIT` moved from Info to Debug in `app_helper.go`. PROTO
   already has `APP_HELPER_INIT` and `RLP0` at Debug; they only print
   because the level is pinned.
4. Docs: `src/api-srv/README.md:176-178` describes the key.

Test: `TestLoadConfigLogLevel` (`src/internal/config/config_test.go:414`)
covers absent (info), `debug`, and `verbose` (refused by name).

Corrections to the doc entry: GRC also rewrote its logger and in doing so
dropped `AddSource`; PROTO's `source=file.go:NN` rewrite is better and
must stay. GRC never added `LOG_LEVEL` to its env files or doc/905, and
its `doc/105-api-srv-tech-spec.md:435` still says "all emitted; no level
key", as the entry says.

#### Proposed fix for PROTO

1. `internal/logging/logging.go`: add a `level slog.Level` argument to
   `NewLogger` (keep the `io.Writer` argument and the `ReplaceAttr`
   source rewrite) and add `ParseLevel(name string) (slog.Level, bool)`.
   Update the three callers that pass a buffer
   (`internal/handler/example_handler_test.go:100`,
   `internal/auth/rbac_test.go:12`, `logging_test.go`) to pass
   `slog.LevelDebug`.
2. `internal/config/config.go`: add `LogLevel string` to `Config`; in
   `LoadConfig` read `env["LOG_LEVEL"]`, default `info`, refuse anything
   `ParseLevel` rejects with an error that names the key. If B032 makes
   every key required, drop the default and require it instead.
3. `cmd/server/main.go:48`: parse the level and pass it.
4. `internal/handler/health_handler.go:33,45`: `Info` to `Debug`.
5. Review the "request started" Info lines in `auth_handler.go`
   (`:118`, `:399`, `:537`, `:690`) and `user_mfa_handler.go`; GRC kept
   them at Info. Leave them unless Brandon wants a one-line-per-login
   audit at Info.
6. Add `LOG_LEVEL=` to every API env file: `etc/loc-api-srv.env` (debug),
   `etc/e2e-api-srv.env` (info), `etc/uat-api-srv.env.example`,
   `infra/uat/etc/uat-proto-api.env`, `infra/uat/etc/uat-proto-e2e-api.env`,
   `infra/prd/etc/prd-proto-api.env` (info). Not migrate or backup envs.
7. Document the key in `doc/905` (lands with B032),
   `src/api-srv/README.md` and `doc/105` "Logging".

Tests: Go `TestLoadConfigLogLevel` (absent, each valid value, refused
value); Go logger test that `NewLogger(..., slog.LevelInfo)` drops a
Debug line and keeps Warn; rename `TestLogger_LevelsAndDebugNotDropped`
to pass `LevelDebug` explicitly. No pytest needed.

#### Open question for Brandon

Should a missing `LOG_LEVEL` default to `info`, or be a required key
like the others (B032/B061)? Recommendation: required, since every env
file gets the line anyway and a silent default hides a typo'd file.
Brandon: required

### B032 — Config docs: env grammar and required-key policy

- **Priority:** P2  **Area:** docs, go-api
- **GRC:** `doc/905-config-guidance.md:6-50`; `src/api-srv/README.md:164-172`; `doc/105-api-srv-tech-spec.md:420-428`   **PROTO:** `doc/905-config-guidance.md:1-4`; `src/api-srv/README.md:22-42`; `src/api-srv/internal/config/config.go:64-70,118-193`
- **Ticket:** RG0005 / `5453748`; RG0001; RG0009; RG0015
- **What:** PROTO's 905 is four lines. GRC: grammar (first `=`, one quote pair,
no `export`/`$VAR`/inline comments, last wins), missing keys rejected, only
`TEST_DATABASE_URL` read from env.
- **Port notes:** Grammar applies as-is (`unquote`). Brandon's call: make
optional keys required? Ties to B061, appendix.
- **Decision:** \[X \] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Two things: the env-file grammar is undocumented, and three "optional"
keys break the server when they are absent.

**(1) Grammar lives only in code.** `doc/905-config-guidance.md:1-4` is the
whole guide:

```
- use the simple env file format
- don't worry about environment variables.  The config file has the value you need
```

The parser is already GRC's (RG0005 was ported):
`src/api-srv/internal/config/config.go:52-55` splits on the first `=`,
trims, and `unquote` (`:66-71`) strips one matching quote pair. A later
duplicate key overwrites the earlier one in the map. Nothing tells the
person editing a host file that `export KEY=v` makes a key named
`export KEY`, that `KEY=v # note` keeps `# note` in the value, that
`$VAR` is not expanded, or that a line with no `=` is silently dropped.
A secret with a `#` or `$` in it is safe only because nothing interprets
it, and no doc says so. DX gap.

**(2) Missing integer keys become 0, and 0 breaks things.**
`src/api-srv/README.md:35-36` calls `JWT_SESSION_EXPIRE_HOURS`,
`MAX_USER_REQ_PER_MIN` and `MAX_UNAUTH_REQ_PER_MIN` "optional, integer".
The loader ignores both absence and a parse error:

```go
// src/api-srv/internal/config/config.go:86-88
if jwtHoursStr := env["JWT_SESSION_EXPIRE_HOURS"]; jwtHoursStr != "" {
    if val, err := strconv.Atoi(jwtHoursStr); err == nil {
        config.JWTSessionExpireHours = val
```

`MAX_*` at `:92` and `:98` are the same. The zero then reaches:

- `src/api-srv/internal/auth/jwt.go:46-47`: `ExpiresAt` is `now + 0h`, so
  every token is born expired; login succeeds and the next call 401s.
- `src/api-srv/internal/security/app_helper.go:376`: `if count > limit`
  with limit 0 refuses the first request of every minute with 429.

A typo such as `MAX_USER_REQ_PER_MIN=200/min` does the same with no log
line. Every env file in the repo sets all three keys today, so this bites
only a new or hand-edited host. Correctness bug (fail-confusing, not a
hole). The doc/700 appendix already lists "Numeric config keys silently
0" and unchecked `FAILED_LOGIN_*_DURATION` grammar as open in both repos.

**(3) Small:** the two `FAILED_LOGIN_*_DURATION` emptiness checks run
twice (`config.go:145-155` and again `:205-214`).

#### What GRC changed

- RG0005 (`5453748`): the quote-stripping parser plus a table test. PROTO
  already has both (`config_test.go:9`, `TestReadEnvFile_QuotesAndEdges`).
- `doc/905-config-guidance.md:6-50` (GRC): one flat file per host; the
  required-key list; the full grammar (first `=`, one quote pair, no
  `export`, no `$VAR`, no escapes, no inline comments, last duplicate
  wins); a migration note to grep host files for quoted values before
  upgrading, because a quoted `JWT_SECRET` or `TOTP_ENCRYPTION_KEY` changes
  value under the new parser; and "the only environment variable anything
  here reads is `TEST_DATABASE_URL`".
- RG0009: `SERVER_BIND_ADDRESS` required (`src/internal/config/config.go:284-290`).
- RG0015: folded the duplicate `FAILED_LOGIN_*` checks.

**Correction to the doc entry.** "Missing keys rejected" is overstated.
GRC's `src/api-srv/README.md:164-169` says every key including
`JWT_SESSION_EXPIRE_HOURS`, `MAX_*_REQ_PER_MIN` and `CORS_ALLOWED_ORIGINS`
"is required", but GRC's code still reads those three integers exactly as
PROTO does (`src/internal/config/config.go:177-193`) and defaults CORS to
empty (`:267-275`). GRC's doc/905 list (`:8-14`) matches its code; its
README does not. Do not copy the README sentence. No GRC test covers the
integer keys.

#### Proposed fix for PROTO

1. **`doc/905-config-guidance.md`**: port GRC's `:6-50` text. Replace
   "Radley GRC" with a neutral example, `migrate-srv`/`worker-srv` with
   PROTO's `cmd/migrate` (PROTO has two binaries reading env files, not
   three), and keep the `TEST_DATABASE_URL` line (PROTO reads it at
   `internal/repository/testdb_test.go:31` and
   `internal/migrate/migrate_test.go:83`). Keep the "grep host files for
   quoted values" note only if a PROTO host predates the `unquote` port;
   otherwise drop it.
2. **`src/api-srv/README.md:22-42`**: make the key list match the code
   after step 3, and point to doc/905 for grammar.
3. **`config.go:86-102`** (if Brandon says required): a small
   `requirePositiveInt(env, key)` that errors "KEY is required" when
   absent and "KEY must be a positive integer" otherwise. Use it for the
   three integer keys. Fold the duplicate checks at `:205-214`. PS0011's
   "Config loader" section (table-driven spec, one error naming every
   missing key) is the larger version; Brandon deferred PS0011, so keep
   this narrow.
4. Optional, appendix item: parse `FAILED_LOGIN_*_DURATION` at load with
   the same parser `cache.KV` uses, so a bad value fails start-up instead
   of each failed login.
5. B061 is the same policy for `SERVER_BIND_ADDRESS`; B031 adds
   `LOG_LEVEL`. Decide all three together and document them in one pass
   of doc/905.

Tests: Go table test in `config_test.go` for each integer key (absent,
`0`, `abc`, `200`); Go test that duplicate `FAILED_LOGIN_*` errors still
name the key. No web or pytest change.

#### Open question for Brandon

Should every key the API reads be required (no defaults at all, so
`JWT_SESSION_EXPIRE_HOURS`, `MAX_*_REQ_PER_MIN`, `CORS_ALLOWED_ORIGINS`,
`SERVER_BIND_ADDRESS` and `LOG_LEVEL` all fail start-up when absent)?
Recommendation: yes for the integers and `LOG_LEVEL`, since every env
file already sets them and their zero value breaks login; decide CORS
and the bind address under B061.

### B033 — Remove anonymous `POST /error-report`

- **Priority:** P2  **Area:** go-api, web-app, tests, docs
- **GRC:** deleted; `src/internal/handler/common.go:15-25`; `src/api-srv/README.md:15-21`   **PROTO:** `src/api-srv/internal/handler/error_report_handler.go:1-80`; `src/api-srv/cmd/server/main.go:105,198`; `tests/test_error_reporting.py`; `tests/README.md:53`; `src/api-srv/README.md:81`
- **Ticket:** RG0018 / `68c77d2`
- **What:** Anonymous route writes 2000 chars of client text to the log at
Error. GRC's client posted to the wrong origin; GRC deleted it.
- **Port notes:** Brandon's call: delete, or require a session? Keep
`test_error_reporting.py` unless the route goes.
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip

Brandon: Let's just require a session

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

PROTO paths below are under `src/api-srv/` unless they start with
`src/web-app/` or `tests/`. The API registers an anonymous route:

```go
// cmd/server/main.go:105, 197-198
errorReportHandler := handler.NewErrorReportHandler(appServices)
// Error reporting (no auth required)
mux.HandleFunc("POST /error-report", errorReportHandler.HandleErrorReport)
```

The handler skips JWT, CSRF, authorization and the block check, keeps
only the per-IP rate limit (`internal/handler/error_report_handler.go:37-41`),
caps the body at 8 KiB (`:14`, `:52`), then writes up to 2000 characters
of whatever the caller sent at Error level:

```go
// internal/handler/error_report_handler.go:74
app.Logger().Error("CLER", "Client Error Report: "+report)
```

**Correction to the doc entry: nothing in PROTO's web app calls it.**
PS0001 (`8559400`) already removed the error boundary's
`fetch('/error-report')` and its "We've automatically reported this
issue" text. `src/web-app/utils/report-error.ts:11-13` writes to the
console only, and its header (`:4`) calls the POST "tomorrow maybe". The
only callers are the 7 pytest cases in `tests/test_error_reporting.py`,
and most of them accept either 200 or 429 (`:21`, `:43`, `:66`), so they
pin little. GRC's "client posted to the wrong origin" bug does not exist
in PROTO; PROTO fixed the client half and left the server half.

Consequence, for an attacker with no account:

- Anyone on the internet can write attacker-chosen text into the
  production journal at Error level, 20 lines per minute per IP
  (`MAX_UNAUTH_REQ_PER_MIN=20`, `etc/loc-api-srv.env:20`). The rate key
  is `ClientIP`, which trusts a forged first `X-Forwarded-For` entry until
  B007's XFF fix lands, so the real cap today is none.
- An operator who alerts or greps on `level=ERROR` gets noise that looks
  like real faults, and journal retention shrinks.
- slog's text handler quotes the message, so a newline cannot forge a
  separate log line. This is not log injection.

Classification: a small security exposure (unauthenticated write to an
operator surface) on a route with no reader and no caller. Dead code with
an attack surface.

#### What GRC changed

GRC commit `68c77d2` (RG0018) deleted the route. Its reasoning: the
route's one caller had never worked (it posted to the dev server's
origin, not `/api`), it appended "anonymous, attacker-choosable text to
the log for no reader", and putting it behind a session would not serve
its purpose, because a render crash can happen on the login screen
where there is no session.

Removed: `internal/handler/error_report_handler.go`, its construction and
route in `src/api-srv/cmd/server/main.go`, the 8 KiB cap, and
`tests/test_error_reporting.py` (8 cases). GRC's
`src/internal/handler/common.go:15-25` comment records that the 8 KiB cap
"went with `/error-report` (RG0018)". `src/api-srv/README.md:15-21`
lists the remaining routes and says where `/error-report` went. The
error boundary was changed to report through `reportError` and tell the
user nothing was sent; PROTO already has that. No test replaces the
deleted ones; there is nothing left to test.

#### Proposed fix for PROTO

1. Delete `internal/handler/error_report_handler.go`. This also closes
   PS0011's "`error_report_handler.go:78` unchecked `w.Write`" line.
2. `cmd/server/main.go`: remove `:105` (construction) and `:197-198`
   (comment and route).
3. Delete `tests/test_error_reporting.py`; drop its bullet at
   `tests/README.md:53`.
4. Docs: drop `POST /error-report` from `src/api-srv/README.md:81`; in
   `doc/105-api-srv-tech-spec.md:200` drop `error_report_handler.go` from
   the skipped-checks list (leave `UserSelfCreateHandler`, which stays
   because B002 is SKIP).
5. `src/web-app/utils/report-error.ts:2-5`: replace "tomorrow maybe POST
   /error-report" with the decision: errors go to the console only; a
   server sink would need its own ticket.
6. Check `src/web-app/constants/api.ts` and the mirror test (B054) still
   agree after the route list shrinks; the web side has no `/error-report`
   entry today, so nothing changes there.

Blind-copy traps: GRC's commit also edits `common.go`'s `maxAuthBodyBytes`
comment, which PROTO does not have until B011; do not port that comment
here. GRC's `/grc` link change in the error boundary is GRC-only.

Tests: no new tests; run the Go build, `go vet`, and the pytest suite to
confirm nothing else posts to the route. A route-sweep spec (B066) would
pin that the route answers 404/405.

#### Open question for Brandon

Delete the route, or keep it behind a session? Recommendation: delete;
no client calls it, and a session-only version misses the crashes it
would exist to catch.

Brandon: keep behind a session.

### B034 — User block feature (PROTO ships dead block primitives)

- **Priority:** P2  **Area:** go-api, web-app, tests
- **GRC:** `src/internal/handler/user_block_handler.go:1-165`; `repository/user_repository.go:271-341`; `src/api-srv/cmd/server/main.go:95-109,223-228`; `model/app_user_model.go:48-58`; `handler/user_block_handler_test.go:93-174`; `stores/app-user-store.ts:55-56,380-393`; `pages/app-user-edit-page.tsx:259-283`; `pages/app-user-list-page.tsx:173-178`; `constants/api.ts:33-35`; `types/user-type.ts:3-5`; `tests/test_user_block.py:1-96`   **PROTO:** `src/api-srv/internal/security/app_helper.go:431-456,460-483`; `model/grants.go:22`; `model/security_model.go:87-91`; new web files
- **Ticket:** RG0043 item 11 / `03ad0f3`
- **What:** PROTO ships unused block parts (`BLOCKED*`, 9008,
`CheckUserBlocked`). GRC: `account_status`, block/unblock routes, `/auth`
refuses non-active, KV flag reloaded, revocation, UI, pytest.
- **Port notes:** Needs B035's migration and B003. Brandon's call: port, or
delete the dead primitives (PS0011)?
- **Decision:** \[X \] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

PROTO paths are under `src/api-srv/internal/` unless stated. PROTO has
the read side of a "block a user" feature and no write side.

- The live check. Every handler that calls `CheckRateLimitAndUserBlock`
  (31 call sites) reads a KV flag on each authenticated request:
  ```go
  // security/app_helper.go:436-437, 449
  if sh.claims != nil {
      blockedValue, err := sh.appServices.KV.Get("BLOCKED*" + sh.claims.Subject)
      sh.failAndRespond(http.StatusForbidden, "User account is blocked")
  ```
- `CheckUserBlocked` (`security/app_helper.go:459-485`) has no caller.
- Grant 9008 (`model/grants.go:22`; `src/web-app/constants/grants.ts:13`)
  is seeded to both admin groups (`sql/fixtures/local-seed.sql:52,77`);
  no route checks it. `BlockUserRequest` (`model/security_model.go:87-91`)
  is never decoded.

Verified dead: the only writer of a `BLOCKED*` key is a test fixture
(`handler/example_handler_test.go:170`). No `account_status` column
exists (migrations `0001`-`0003`); `/auth` cannot refuse a disabled login.

Consequence: an admin cannot disable a login except by `DELETE /user`,
which loses the row and (until B003) leaves the JWT working to expiry.
The UI offers a `BLOCK_USER` grant that does nothing. Missing feature
plus misleading dead code, not a hole. PS0011
(`tickets/01-open/PS0011-api-dead-code-config-loader.md:38`) lists
`BlockUserRequest` as dead; Brandon deferred it because such DTOs "may be
placeholders for planned endpoints". This item is that endpoint.

#### What GRC changed

GRC commit `03ad0f3` (RG0043 item 2.2) built the write side and deleted
`CheckUserBlocked`.

1. Column: `app_user.account_status`, values `invited|active|blocked`
   (`src/internal/model/app_user_model.go:48-57`). It came from RG0035's
   migration (`src/migrate-srv/sql/00024_user_invite.sql:8-10`); block
   added the third value.
2. Routes `POST /admin/blockUser` and `/admin/unblockUser`, grant 9008,
   body `{userName}` (`src/api-srv/cmd/server/main.go:223-228`). The
   handler (`src/internal/handler/user_block_handler.go:63-165`) refuses
   self-block (400), another tenant's login (404), a wrong starting state
   (409), and uses optimistic locking on `update_ts`
   (`src/internal/repository/user_repository.go:276-313`).
3. Two halves, because the KV is in-process
   (`user_block_handler.go:15-28`): the row stops new logins
   (`/auth` refuses non-active, `src/internal/handler/auth_handler.go:329`,
   same counted 401 as a bad password); the `BLOCKED*` flag stops live
   sessions with 403. The flag's TTL is `JWT_SESSION_EXPIRE_HOURS`, and
   `main` reloads every blocked login's flag at start
   (`src/api-srv/cmd/server/main.go:95-108`), so a restart does not
   revive old tokens.
4. Block also calls `RevokeUserSessions` (B003), so unblock does not
   revive pre-block sessions (`user_block_handler.go:141-145`).
5. Web: `setUserBlocked`, Block/Unblock buttons, list badge.

Tests: Go `TestAdminBlockUser`, `TestLoadBlockedUsers`
(`src/internal/handler/user_block_handler_test.go:93,164`); pytest
`tests/test_user_block.py` (live session refused then restored; twice
409; self 400; unknown 404; no grant 403).

#### Proposed fix for PROTO

1. **Migration.** Whichever of B034/B035 lands first adds
   `sql/migrations/0004_app_user_account_status.sql` (latest is `0003`):
   `account_status text`, backfill `'active'`, `SET NOT NULL`, no DEFAULT.
   PROTO migrations are plain SQL, no goose markers, no Down section.
2. **Every insert names the column**: `CreateUser`
   (`repository/user_repository.go:143`, also used by `/userSelfCreate`,
   kept by B002 SKIP) and `sql/fixtures/local-seed.sql:92`. Add
   `AccountStatus` to `AppUserModel` and the three user reads.
3. **`/auth`**: add `AccountStatus != "active"` to B007's failure branch.
4. **Handler, repo, startup reload**: port `user_block_handler.go`,
   `UpdateUserAccountStatus`, `RetrieveBlockedUserNameList`, and the
   reload into `cmd/server/main.go`. Traps: module path is
   `github.com/wbmartin/starter-api`; `mapRepoError` exists only after
   B028; `limitBody(maxAuthBodyBytes)` only after B011. Delete
   `CheckUserBlocked`.
5. **Web**: two endpoints in `src/web-app/constants/api.ts`;
   `setUserBlocked` in `stores/app-user-store.ts`; buttons and badge in
   `pages/app-user-edit-page.tsx` / `app-user-list-page.tsx`; type in
   `types/user-type.ts`. Use B037's fetch path if it has landed.

Dependencies: B003 (PULL) first; B007 for `/auth`; B035 shares the
column. B006 interaction: after B003 a blocked user's token fails
`CheckJWT` with 401 and signs out. After a restart revocations are gone,
only the flag answers, with 403, and B006 no longer signs out on 403; the
user sees "User account is blocked" per call until expiry. Note it in
the handler comment.

Tests: Go handler table (self, other tenant, wrong state, success sets
flag and revokes); Go `LoadBlockedUsers`; pytest `test_user_block.py`
on PROTO fixtures in tenant 2; vitest for `setUserBlocked`.

#### Open question for Brandon

Port the block feature, or delete the dead primitives (`BLOCKED*` check,
`CheckUserBlocked`, grant 9008, `BlockUserRequest`) under PS0011?
Recommendation: port it; disabling a login without deleting it is a
baseline admin need, and half the plumbing is already shipped.

Brandon: Port Blocked

### B035 — Invite-link onboarding (replaces self-registration)

- **Priority:** P2  **Area:** go-api, web-app, tests, etc
- **GRC:** `src/internal/handler/user_invite_handler.go:1-538`; `repository/user_invite_repository.go:1-388`; `model/user_invite_model.go`; `auth/token.go:24-44`; `config/config.go:58-71,168-174`; `src/api-srv/cmd/server/main.go:46-51,180-187,208-210`; `user_handler.go:1088-1099`; `src/migrate-srv/sql/00024_user_invite.sql:8-10,16-32,35-38,40-42`; `pages/invite-page.tsx:1-191`; `components/invite-link-modal.tsx:1-53`; `utils/invite-link.ts:1-33`; `stores/app-user-store.ts:208-275`; `pages/app-user-edit-page.tsx:142-145,284-315,366-420`; `pages/app-user-list-page.tsx:158,173-180`; `constants/routes.ts:23-25`; `tests/test_user_invite.py:1-179`; `etc/loc-api-srv.env:7-8`; `user_repository.go:143-210`   **PROTO:** new files; new `src/api-srv/sql/migrations/0004_user_lifecycle.sql`; `src/api-srv/cmd/server/main.go` routes
- **Ticket:** RG0035 / `7001cbd`; RG0036
- **What:** Admin creates an `invited` login and a one-time link (256-bit
fragment token, SHA-256 stored, `INVITE_TTL_HOURS`); reissue revokes; redeem
activates in one transaction.
- **Port notes:** Strip `audit_log` calls (:351-388) or B067. `account_status`
NOT NULL, no DEFAULT. Rewrite pytest off `grc` fixture. No email (RG0040).
Enables B071.
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

PROTO has two ways to make a login, and neither lets an admin onboard a
person without handling their password.

1. **Admin sets a password.** `POST /user`
   (`src/api-srv/internal/handler/user_handler.go:341`) requires a raw
   password and hashes it (`:436`, `auth.HashPassword(req.PwRaw)`). The
   admin chooses it, sends it to the person outside the app, and nothing
   makes the person change it. The admin's "Change Password" for another
   user is itself broken (B013).
2. **Self-registration.** `POST /userSelfCreate`
   (`src/api-srv/cmd/server/main.go:154`) is public. It puts the new login
   in tenant 1, group 3 (`src/api-srv/internal/handler/user_mfa_handler.go:127-128`).
   The login page links to it (`src/web-app/pages/login.tsx:88-97`,
   "Don't have an account? Create one here").

B002 (retire `/userSelfCreate`) is marked **SKIP**, so path 2 stays.
What this item fixes is path 1: the admin knows every new user's first
password. That is a security weakness (shared secret in email or chat,
no forced change) and a UX gap. It is not an open hole.

**Correction to the doc entry's title.** "Replaces self-registration"
was GRC's framing. With B002 skipped, B035 for PROTO means **invite links
are an additional admin path, beside self-registration**, and the
recommended default for admin-created users. "Set a password" stays as
the second admin option, as it did in GRC.

#### What GRC changed

GRC RG0035 (`7001cbd`, merged `92423c9`) added invite links and, in the
same ticket, removed self-registration. The parts:

- **Schema** (`src/migrate-srv/sql/00024_user_invite.sql`):
  `app_user.account_status` (`invited`/`active`), backfilled `active`,
  `NOT NULL`, no DEFAULT (`:8-10`); table `user_invite` with
  `token_hash`, `expires_ts`, `redeemed_ts`, `redeemed_source_ip`,
  `revoked_ts` (`:16-32`); unique index on `token_hash` (`:35`).
- **Token** (`src/internal/auth/token.go:32-42`): 32 random bytes, hex;
  only the SHA-256 is stored ("a fast hash is enough ... the token is 256
  random bits").
- **Routes**: admin `POST /userInvite`, `/userInviteReissue`,
  `/userInviteRevoke` with full checks (`src/api-srv/cmd/server/main.go:208-210`);
  public `POST /inviteCheck`, `/inviteRedeem` (`:186-187`) with
  per-IP rate limit, the `/auth` per-address lockout plus a per-token
  lockout, and one generic 404 for any dead link. Redeem checks the
  password rule, then the token, then runs argon2, and locks the invite
  and user rows `FOR UPDATE` in one transaction.
- Reissue revokes the older link; `/changeOtherUserPassword` 409s on an
  invited login (`src/internal/handler/user_handler.go:1088-1099`);
  `/auth` refuses non-`active` logins as a bad password.
- **Config**: `INVITE_TTL_HOURS` (72), required at api-srv start
  (`src/internal/config/config.go:58-71,168-174`).
- **Web**: `pages/invite-page.tsx` takes the token from the URL fragment
  (never sent to a server), checks, redeems, strips it from history and
  signs in; `invite-link-modal.tsx` shows the link once; new users
  default to "Send an invite link".
- **Audit**: each step appends to GRC's hash-chained audit log
  (`src/internal/repository/user_invite_repository.go:351-388`).
- **Removed self-registration**: route, handler, login link, modal.

Tests: Go `user_invite_handler_test.go`, `user_invite_repository_test.go`;
pytest `tests/test_user_invite.py` (9 cases: issue, invited cannot sign
in, check+redeem+sign in, reissue/revoke, same answer for dead links,
audit, 409 on admin password, needs `CREATE_USER`, self-registration 404).

#### Proposed fix for PROTO

1. **Migration** `src/api-srv/sql/migrations/0004_user_lifecycle.sql`
   (latest is `0003`): `account_status` as in GRC (shared with B034,
   whichever lands first) plus `user_invite`. Plain SQL: drop goose
   markers, the Down section and the per-migration GRANT lines (PROTO's
   `0001` grants once; confirm new tables are covered).
2. **Every insert names `account_status`**: `CreateUser`
   (`internal/repository/user_repository.go:143`) writes `active`; this
   covers `/userSelfCreate` too. `sql/fixtures/local-seed.sql:92` likewise.
3. Port `token.go` functions, `user_invite_repository.go`,
   `user_invite_model.go`, `user_invite_handler.go`. Traps: module path
   `github.com/wbmartin/starter-api`; **strip every
   `appendUserInviteAudit` call** (no audit table until B067);
   `ErrNotFound`/`mapRepoError` need B028 (or B007's minimal wrap);
   `limitBody` needs B011; GRC's `grc` group names and `/grc` landing
   become PROTO's groups and `/dashboard`.
4. `/auth`: refuse non-`active` in B007's failure branch.
   `ChangeOtherUserPassword` (`user_handler.go:965`): 409 for `invited`.
5. **Keep** `/userSelfCreate`, its login link and
   `user-registration-modal.tsx`. Do **not** port GRC's
   `test_self_registration_is_gone`, the login text "Ask your
   administrator for an invite link", or the `main.go` comment that says
   there is no self-registration route.
6. `INVITE_TTL_HOURS` required in `LoadConfig` (PROTO has one server
   binary; no `ValidateInvite` split), added to every API env file.
7. **Web**: invite page, modal, `utils/invite-link.ts`, edit-page invite
   mode, list Status column, route `/invite`, two public endpoints.
   GRC's `anonymous: true` (no CSRF on anonymous calls) needs B037.
8. No email (GRC RG0040 is still open). Enables B071.

Dependencies: B007, B028, B037; B034 shares the column. Tests: Go handler and repo
tests in tenant 2; pytest `test_user_invite.py` rewritten off GRC's
`grc` fixture, minus the self-registration case and the audit case;
vitest for `invite-link.ts`.

#### Open question for Brandon

With self-registration kept, should a self-registered login stay
`active` at once, or land `invited`/pending until an admin approves it?
Recommendation: keep it `active` (today's behaviour) and leave approval
to a later ticket; B035 is about admin onboarding only.

Brandon: Let's adapt so we have both invite links and self create, the generated app will have to decid what it wants.

### B036 — Web: API unreachable seam for 5xx/429; `/refresh` retried

- **Priority:** P2  **Area:** web-app, docs
- **GRC:** `utils/api.ts:47-64,93-125,286-292`; `stores/api-status-store.ts:1-46`; `stores/auth-store.ts:20-33,117-152`; `components/grc/shell.tsx:153-160`; `App.tsx:64-84`; `README.md:33-78`; `README.md:96-140`   **PROTO:** `utils/api.ts:14-56`; `stores/auth-store.ts:102-149`; `app.tsx:194`; `tests/api-unreachable.test.ts:56`; `README.md:27-34`
- **Ticket:** RG0018 / `d752dbd`, `ed5d1cd`; RG0007
- **What:** PROTO notices an outage only at boot; a 502/429 mid-session is a
per-call error. GRC classifies 5xx/429 as unreachable through one seam, shows
one self-clearing notice, retries `/refresh` 1/2/4 s.
- **Port notes:** Keep boot `UNREACHABLE`; seam replaces
`setSessionExpiredListener`; rewrite `api-unreachable.test.ts:56`; README
section.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Web paths are under `src/web-app/`. PROTO tells "the API is down" apart
from "you are signed out" only at page load.

1. **Classification.** `ApiUnreachableError` covers only a rejected fetch
   (connection refused, DNS, offline): `utils/api.ts:50-55`. A 502 from
   nginx while the API restarts, a 503, or a 429 from the per-minute limit
   is a normal response, and `throwForFailedResponse` turns it into a
   plain `Error` (`utils/api.ts:102`), "Request failed (502)" when the body
   is nginx HTML.
2. **Boot does it right, once.** `authenticateWithAuthToken`
   (`stores/auth-store.ts:102-148`) maps 401/403 to `NO` (`:111`) and any
   other non-2xx or a thrown fetch to `UNREACHABLE` (`:115-117`,
   `:138-143`). `app.tsx:194` then swaps the page for
   `<ApiUnreachableNotice />`, whose "Try again" is manual. No retry, so
   a page loaded during a two-second API restart shows the notice until
   the user clicks.
3. **Mid-session it is per call.** On the example list,
   `pages/example-list-page.tsx:109` replaces the whole page with "Error
   loading examples: Request failed (502)", and the store keeps that error
   until the next read. A failed delete goes only to `reportError`
   (`:122-123`), which writes to the console: the user sees nothing. Each
   screen shows the outage its own way, none says "the server is not
   answering, you are still signed in", and nothing clears when the API
   comes back.

UX gap, not a security hole: the server enforces every check; this only
changes what the user is told.

Doc ref check: `tests/api-unreachable.test.ts:56` asserts that a 401 is
not unreachable. That stays true after B036. The test file still needs
rewriting, because it imports `apiRequest` (`:2`), which B037 unexports.

#### What GRC changed

RG0018, commits `d752dbd` (classification) and `ed5d1cd` (notice).

- **One classifier.** `refusal()` in `src/web-app/utils/api.ts:287`:
  ```ts
  if (status >= 500 || status === 429) {
    return unreachable(message, { status });
  ```
  `ApiUnreachableError` gains `status` (`:55-64`).
- **One seam.** `subscribeApiOutcome` (`:114`) announces `ok`,
  `unreachable` or `session-expired` for every `apiJson` call. Two
  subscribers: the auth store (sign out on `session-expired` only) and
  `stores/api-status-store.ts:37-46` (one `unreachable` flag, raised by
  the first failure, cleared by the next `ok`; both setters return early
  so twenty failing lists render once).
- **One notice.** The shell shows an alert, "You are still signed in"
  (`components/grc/shell.tsx:153-159`); the pre-auth gate shows the same
  with "Try again" (`App.tsx:64-84`).
- **`/refresh` retried.** `stores/auth-store.ts:29`,
  `REFRESH_RETRY_DELAY_MS = [1000, 2000, 4000]`; on `ApiUnreachableError`
  wait and retry, then leave `authState` alone (`:133-145`). GRC has no
  `UNREACHABLE` state; it stays `CHECKING` and the gate reads the flag.
- README "Talking to the API" (`src/web-app/README.md:33-78`) has the
  table: unreachable / session-expired / plain `Error`.

Tests: `src/web-app/utils/api.test.ts:47-80` (failed fetch, 500, 429 are
`ApiUnreachableError` and announced). No test for the store, the notice
or the retry loop.

#### Proposed fix for PROTO

Order: B037 (one fetch path), then B004 (CSRF resync), then B006
(401 only, which B006's decision puts after B004), then this. All four
edit `utils/api.ts`.

1. **`utils/api.ts`**: give `ApiUnreachableError` a `status`; in the one
   function every call goes through (B037), throw it for `>= 500` and
   `429`. Add `subscribeApiOutcome` and announce `ok` on every 2xx.
2. **Replace `setSessionExpiredListener`** (`utils/api.ts:81-87`) with the
   seam. The auth store subscribes to `session-expired`. Trap: once
   `/refresh` goes through the seam, every visit to the login page gets a
   401 announced. Change the guard at `stores/auth-store.ts:226` from
   `=== "NO"` to `!== "YES"` (GRC `:187`), or the login page toasts
   "Session ended" to someone who was never signed in.
3. **New `stores/api-status-store.ts`**, ported as is.
4. **`app.tsx`**: when `authState === "YES"` and the flag is up, show one
   banner above `<Outlet />` (still signed in, clears itself). Keep the
   boot `UNREACHABLE` state and `ApiUnreachableNotice` (`:194`); they are
   clearer than GRC's stuck `CHECKING`.
5. **`authenticateWithAuthToken`**: call through `apiJson`; on
   `ApiUnreachableError` retry at 1/2/4 s, then set `UNREACHABLE`. A 401
   or 403 still sets `NO` (B006 kept that for `/refresh`, where 403 means
   blocked).
6. **B004's resync `/refresh` is not retried.** It runs inside a user's
   save; a 7 s stall there is worse than letting the write's 403 stand.
   Use a single attempt and do not announce its outcome.
7. **README**: an "Errors from the API" section after "API Proxy"
   (`src/web-app/README.md:27-34`), with GRC's three-row table.

Tests (vitest): rewrite `tests/api-unreachable.test.ts` against `apiJson`
(500, 502, 503, 429 unreachable; 401 not); store: raised once, cleared by
`ok`; refresh with fake timers: 502, 502, 200 ends `YES`, four 502s end
`UNREACHABLE` after 7 s, 401 ends `NO` with no retry; a 401 while
`CHECKING` shows no toast.

#### Open question for Brandon

Should a 429 show the "server not answering" banner like a 5xx (GRC), or
stay a per-call error saying "too many requests"? Recommendation: banner,
since PROTO's bucket resets each minute and the banner clears itself on
the next success.

Brandon: banner

### B037 — Web: one fetch path; unexport `apiRequest`

- **Priority:** P2  **Area:** web-app, docs
- **GRC:** `utils/api.ts:1-34,221-279`; `README.md:33-43,67-71`; `CLAUDE.md:79-84`   **PROTO:** `utils/api.ts:34-56,149`; raw calls `stores/create-entity-store.ts:98-101,149-154,167-172,191-195`, `stores/app-user-store.ts:57-62,85-88`, `stores/app-user-group-store.ts:30-33`, `stores/examples-store.ts:22-26`, `stores/auth-store.ts:105-115,153`; `eslint.config.js:28,35`; `constants/api.ts:1-4`; `CLAUDE.md:51`
- **Ticket:** RG0007 / `b659bea`, `3ddb71d`
- **What:** PROTO exports the raw fetch; each store re-does `!response.ok`; CSRF
goes on anonymous calls. GRC: one path (`apiJson` + `jsonBody`), no CSRF on
anonymous.
- **Port notes:** Keep PROTO's store design; swap fetches. CLAUDE.md "Web:"
bullet (apiJson only, console only in `report-error.ts`, kebab-case, mirrors
name source); skip max-len 200.
- **Decision:** \[X \] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Web paths are under `src/web-app/`. `utils/api.ts` offers two ways to
call the API, and the stores use the raw one for most entity traffic.

1. **Raw path exported.** `apiRequest` (`utils/api.ts:34-57`) returns the
   bare `Response`; `:149` exports it. `apiJson` (`:129-141`) wraps it and
   throws on non-2xx, but is used at only 11 sites. Ten calls in five
   files take the raw path and repeat the same three steps:
   ```ts
   // stores/create-entity-store.ts:149, 154-155
   const response = await apiRequest(config.itemEndpoint, {
   if (!response.ok) {
     await throwForFailedResponse(response, readErrorMessage);
   ```
   Sites: `stores/create-entity-store.ts:98,149,167,191` (every entity
   list, create, update, delete), `stores/app-user-store.ts:57,85`,
   `stores/app-user-group-store.ts:30`, `stores/examples-store.ts:22`,
   `stores/auth-store.ts:105,153` (`/refresh`, `/logoff`).
2. **Consequence.** Every rule about API errors has to be applied in ten
   places, or in `apiRequest` before any caller sees a status. B004's
   adapted fix already hit this: its step 5 put the CSRF retry in
   `apiRequest` "not `apiJson`" because entity writes bypass `apiJson`.
   B006 (401-only sign-out) and B036 (5xx/429 as unreachable) have the
   same problem. A new store copying any of these sites inherits the
   pattern; nothing stops a bare `fetch` either.
3. **CSRF on anonymous calls.** `apiRequest` sets `X-CSRF-Token` whenever
   a token is in memory (`utils/api.ts:40-43`). `anonymous: true`
   (`:132`, used by `/auth`, MFA setup/verify and self-registration) only
   changes how a failure is classified (`:136`); a leftover token from an
   ended session still rides on `/auth`. The server skips CSRF there, so
   nothing breaks. Quality gap, not a hole.

Overall: a quality/DX gap that makes B004, B006 and B036 harder to land
correctly.

#### What GRC changed

RG0007, commits `b659bea` and `3ddb71d`.

- `apiRequest` is unexported, not renamed. Header of
  `src/web-app/utils/api.ts:21-23`: "a second, raw path is how
  `response.ok` handling grew in nine files". 33 call sites in 10 files
  moved to `apiJson`.
- `apiJson` (`:221-279`) always reads the body (an unread refused body
  left Chromium requests "in flight" and stalled network-idle waits),
  returns the parsed envelope, and throws an `Error` with the API's
  `{"error"}` text. No caller reads `response.ok`.
- `anonymous: true` now withholds the CSRF header (`:237`,
  `if (!options.anonymous && token)`).
- Later, RG0018 and `45d263d` built the error classes, the outcome seam
  and the CSRF resync into the same function (`:270-272`), which only
  works because nothing bypasses it.
- `CLAUDE.md:79-84`: a "Web (RG0007)" rule: `apiJson` only, `console.*`
  only in `utils/report-error.ts`, kebab-case files, mirrors name the Go
  file they mirror.

Tests: `src/web-app/utils/api.test.ts:150` (CSRF header on a session
call, withheld from an anonymous one) plus the classification cases.
No lint rule bans a raw `fetch`; unexporting is the only guard.

#### Proposed fix for PROTO

1. **Move the ten sites to `apiJson`/`jsonBody`.** Keep PROTO's store
   factory and its shapes; only the call changes, e.g.
   `const body = await apiJson<{data?: TEntity[]}>(url)`. The DELETE in
   `create-entity-store.ts:191` stays body-less.
2. **`auth-store` `/refresh` (`:105`)** moves to `apiJson` and maps typed
   errors: `ApiUnreachableError` to `UNREACHABLE`, anything else to `NO`.
   Trap: `apiJson` fires the session-expired listener on 401, and the
   guard at `stores/auth-store.ts:226` only skips `NO`, so the login page
   would toast "Session ended" during `CHECKING`. Change the guard to
   `!== "YES"` here (B036 needs the same).
3. **Unexport**: delete `:149`; make `throwForFailedResponse` private.
   Update the `apiRequest` mention in `constants/api.ts:4`.
4. **Anonymous calls send no CSRF header**: pass `anonymous` down to the
   private request function.
5. **Reconcile with B004.** Brandon's B004 decision puts the CSRF resync
   and retry in `apiRequest`. That still holds: `apiRequest` becomes the
   private function `apiJson` calls, so the retry sits below every call
   and B037 removes the only way around it. B004's reason for avoiding
   `apiJson` (writes bypass it) disappears, but moving the retry up buys
   nothing. Keep the B004 resync's own `/refresh` on a direct `fetch`
   inside `api.ts`, so it cannot recurse into the retry. **Order:** B037
   first, then B004, then B006 and B036. B004's vitest cases then run
   through `apiJson` only, and its "an entity-store write heals" case
   comes free.
6. **Optional, PROTO-only guard**: `no-restricted-globals` for `fetch`
   in `eslint.config.js`, with an override for `utils/api.ts` and
   `tests/**`, next to the `no-console` pair at `:28` and `:35`.
7. **`CLAUDE.md`**: add a "Web" bullet after `:51` (apiJson only; console
   only in `report-error.ts`; kebab-case; `constants/api.ts` and
   `constants/grants.ts` name the Go file they mirror). Skip GRC's
   `max-len` 200; PROTO keeps ~80. Add a header to `constants/grants.ts`
   naming `model/grants.go` (`constants/api.ts:1-4` already has one).

Tests (vitest): rewrite `tests/api-unreachable.test.ts`, which imports
`apiRequest` (`:2`), against `apiJson`; CSRF header present on a session
call and absent on `anonymous`; one entity-store create, update, delete
through mocked `fetch`; `tests/session-expired.test.ts` moves from
`throwForFailedResponse` to `apiJson`. `npm run lint`, `tsc`, vitest.

#### Open question for Brandon

None. The B004 placement question resolves itself: the retry stays in
`apiRequest`, now private, and B037 lands first.

### B038 — Wire renames: `pwHash` -&gt; `password`; `fooList` keys

- **Priority:** P2  **Area:** go-api, web-app, tests
- **GRC:** `src/internal/model/security_model.go:115`; `user_handler.go:765-769,900-904`; `repository/user_repository.go:595-720`; `types/user-type.ts:17,76,87`; `tests/test_user_management.py:22,47,100,134,170`; `tests/test_basic_user_authorization.py:64,72`; `tests/test_security_context_isolation.py:52,73,112,125-126,147,160-161,182,218-219`   **PROTO:** `src/api-srv/internal/model/security_model.go:112`; `user_handler.go:756-760,876-881`; `repository/user_repository.go:478-594`; `pages/app-user-edit-page.tsx:89,127,280,285`; `types/user-type.ts`; same test files (`test_security_context_isolation.py:123,144,183,196-197,218,231-232,253,289-290`)
- **Ticket:** RG0043 items 36, 38 / `720f5e1`
- **What:** A raw password travels as `pwHash`; `userIds`/`groupId` break
PROTO's list rule. GRC: `password`, `userIdList`, `appUserGroupId`, `any`.
- **Port notes:** Breaking: Go, web, pytest together. go-api P3, web/tests P2.
Typo B089.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Two wire names in the user API mislead. Quality/DX gap, not a hole.

**(1) A raw password travels as `pwHash`.** `POST /createUser` decodes:

```go
// src/api-srv/internal/model/security_model.go:110-112
type CreateUserRequest struct {
	UserName                   string `json:"userName"`
	PwRaw                      string `json:"pwHash"` // Maps to frontend's pwHash field
```

`handler/user_handler.go:436` hashes it, so the field named "hash"
carries the plaintext. Every other password input in the API is `password` or
`newPassword` (`model/security_model.go:6,84,95,101`,
`handler/user_mfa_handler.go:52`, `handler/user_handler.go:987`). The web
mirrors the bad name: `src/web-app/types/user-type.ts:11` puts an optional
`pwHash` on the read model `User` (the API never sends it, because
`model/app_user_model.go:9` tags the real hash `json:"-"`), `:28,39` put it
on the form, and `pages/app-user-edit-page.tsx:89,127,280,285` bind the
password input to it. Consequence: a reader assumes the client hashes, or
that the field is a stored hash, and a log filter or reviewer keyed on
"password" walks past it.

**(2) The group-membership endpoints break the `fooList` rule.**
`AddUserListToGroupHandler` and `RemoveUserListFromGroupHandler` decode:

```go
// handler/user_handler.go:756-757 (and again :877-878)
UserIds      []int     `json:"userIds"`
GroupId      int       `json:"groupId"`
```

`userIds` is a plural, which CLAUDE.md forbids, and `groupId` is not the
column/key name `appUserGroupId` used everywhere else. The repository
carries the plural into Go (`repository/user_repository.go:480,539`
`userIds []int`). The web store sends the same keys
(`stores/app-user-store.ts:54,98-110`). pytest sends both bad names in `tests/test_user_management.py:22,47,100,134,170`
and `tests/test_security_context_isolation.py:123,144,183,196-197,218,
231-232,253,289-290`. A derived app copies these names into every new
list endpoint.

Clarification: the item's "`any`" is GRC's Go `interface{}` sweep, not
`types/user-type.ts`. PROTO has 36 `interface{}` in 10 Go files.

#### What GRC changed

Commit `720f5e1` (RG0043 items 36 and 38) renamed both layers at once.

- `src/internal/model/security_model.go:115`: `Password ... json:"password"`
  commented "the raw password; hashed by the handler, never stored".
- `src/internal/handler/user_handler.go:765-766,900-901`: `UserIDList`
  `json:"userIdList"`, `AppUserGroupID` `json:"appUserGroupId"`.
- `src/internal/repository/user_repository.go:597,663` rename the
  parameter to `userIDList`; the `groupId` parameter was left as is.
- `src/web-app/types/user-type.ts:17,76,87`: `password` on `User`
  (commented "create only"), the form, and its default.
- pytest: `password`, `userIdList`, `appUserGroupId` in
  `tests/test_user_management.py`, `tests/test_basic_user_authorization.py:
  63-64,71-72` and `tests/test_security_context_isolation.py:125-126,
  160-161,218-219`.
- 28 Go `interface{}` became `any`; `AppHelper` receivers became `sh`.

No new test; the renamed pytest calls pin it (a wrong key decodes to zero).

#### Proposed fix for PROTO

One PR: Go, web and pytest together, or the UI and tests break.

1. `model/security_model.go:112`: `Password` `json:"password"` with
   GRC's comment; update `handler/user_handler.go:405,411,436`.
2. `handler/user_handler.go:756-757,877-878`: `UserIDList` /
   `userIdList`, `AppUserGroupID` / `appUserGroupId`; update the
   messages at `:776-777,896` and every `req.UserIds` / `req.GroupId`.
3. `repository/user_repository.go:480-481,539-540` (+ the
   `UserRepositoryInterface` signatures): `userIDList`,
   `appUserGroupID`. Go-side renames are cosmetic (P3); do them with
   gopls so nothing is missed.
4. Web: `types/user-type.ts:11` drop `pwHash` from `User` entirely (it
   is never received); `:28,39` and `pages/app-user-edit-page.tsx:89,127,
   280,285` become `password`; `stores/app-user-store.ts:13-20,54,98-110`
   send `userIdList` / `appUserGroupId`.
5. pytest: the two files above. Fold B089 in: PROTO's
   `tests/test_basic_user_authorization.py:64,72` sends `userGroupId`.
6. Keep the "never serialized" asserts (`handler/user_handler_test.go:849`,
   `profile_handler_test.go:137`, `tests/test_own_profile.py:16`); add
   `password` to them. `interface{}` -> `any` rides with B040.

Traps: copy only struct tags, not GRC's `limitBody` /
`handledBodyTooLarge` (B011) or `accountStatus` on `User` (B034/B035).

Tests: Go handler test decoding a create body with `password` (B039);
vitest on the edit page submit payload key; pytest is covered by step 5.

#### Open question for Brandon

Rename the Go internals too (`userIds` params, `interface{}`), or only
the wire keys plus web and pytest? Recommendation: wire, web and pytest
now; Go-internal renames ride with B040's cleanup pass.

### B039 — Go handler tests that prove the check contract

- **Priority:** P2  **Area:** tests, go-api
- **GRC:** `src/internal/handler/auth_handler_test.go:49-546`; `user_route_check_sequence_test.go:115-275`; `own_profile_check_sequence_test.go:155-454`; `map_repo_error_test.go:16-125`; `session_revocation_test.go:14-37`; `user_mfa_handler_test.go:235-752`; `user_block_handler_test.go`; `src/internal/security/client_ip_test.go:9-82`; `src/internal/model/app_user_model_test.go:12-65`; `auth_handler.go:182-186,633-636`   **PROTO:** new `src/api-srv/internal/handler/auth_handler_test.go`; `user_mfa_handler_test.go:47-169`; `user_handler_test.go`; `profile_handler_test.go`; `internal/security/`; `internal/model/`
- **Ticket:** RG0004 `d761d42`; RG0003 `eb4340b`; RG0014; RG0042; RG0043
- **What:** Fakes that panic if reached prove each route runs JWT, rate, CSRF,
authz, gate before the repository, plus lockout, temp token, 9020, 409,
refresh id, logoff, no secrets serialized. PROTO has no auth handler test.
- **Port notes:** Port beside each fix; handlers take `UserRepositoryInterface`
(GRC :182-186, :633-636).
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Each handler must run JWT, rate limit, CSRF, authorization and
`CheckAllChecksComplete` before it reads the body or the repository. Only
a test proves the order. PROTO proves it for two handlers, not the login
layer. Quality gap: no known hole, but a login-layer regression ships
green. PROTO paths are under `src/api-srv/internal/`.

**(1) The auth handlers have no test at all.** There is no
`handler/auth_handler_test.go`. `AuthHandler`, `RefreshHandler`,
`ChangeSecurityContextHandler` and `ChangeOwnPasswordHandler` take the
concrete repository, so a test cannot swap in a fake:

```go
// handler/auth_handler.go:112-114 (same at :383-385, :504-506, :711-713)
func AuthHandler(
	appServices *app.Services,
	userRepo *repository.UserRepository,
```

A forgotten `Skip*` in `AuthHandler` (`:132-137`) would 500 every login;
a call moved after the body read in `ChangeSecurityContextHandler` (which
B005 already found skipping CSRF at `:530`) would pass CI.

**(2) The user-admin tests never call a handler.** `TestCreateUser`,
`TestUpdateUser`, `TestDeleteUser` and `TestRetrieveUserList` in
`handler/user_handler_test.go:380,517,610,714` set a mock expectation and
then call the mock directly:

```go
// handler/user_handler_test.go:691-692
req := httptest.NewRequest("DELETE", "/user", nil)
result, err := handler.userRepo.DeleteUser(
```

They test testify, not `UserHandler`. Nothing proves `/userList` or
`DELETE /user` refuse before the repository.

**Already in PROTO.** `handler/example_handler_test.go:127` and
`handler/profile_handler_test.go:70` walk 401/403 refusals with an
unprimed testify mock (a premature call fails); neither has a 429 case.
`user_mfa_handler_test.go:47,144` cover the MFA token paths.
`user_handler_test.go:840` already pins "no secrets serialized", so that
part of the doc's What is done.

#### What GRC changed

RG0004 (`d761d42`) made `AuthHandler` and `ChangeSecurityContextHandler`
take `UserRepositoryInterface` "so the check sequence can be proved with a
panicking repository" (`src/internal/handler/auth_handler.go:179-182,
633-636`). RG0003 (`eb4340b`) added 409 lines of MFA handler tests. The fake,
`panicUserRepository` (`src/internal/handler/user_mfa_handler_test.go:120`),
embeds the interface and its named methods panic "repository reached after
a failed security check".

Tests that pin each behaviour (all under `src/internal/`):
- `handler/user_route_check_sequence_test.go:115`: 401, 429, blocked,
  CSRF, grant refusals on `/userList`, `DELETE /user`; `:255` all pass.
- `handler/auth_handler_test.go`: `:49` login completes the gate and sets
  a session cookie (HttpOnly, Secure, Strict, no Max-Age); `:184` lockout
  answers before the repository; `:285` change-context sequence; `:457`
  reissue keeps the session id; `:529` unknown login counted.
- `handler/own_profile_check_sequence_test.go:155-454`: sequence, 413,
  stale `updateTs` 409, writes only the caller's row.
- `handler/user_mfa_handler_test.go:235-752`: temp token single use,
  reset needs grant 9020, sentinels mapped, enrolled `/mfaSetup` refused.
- `handler/session_revocation_test.go:14` token refused after logoff;
  `handler/map_repo_error_test.go:16,91`; `security/client_ip_test.go:9`;
  `model/app_user_model_test.go:12` secrets never serialized.

#### Proposed fix for PROTO

1. Change `AuthHandler`, `RefreshHandler`, `ChangeSecurityContextHandler`
   and `ChangeOwnPasswordHandler` (`handler/auth_handler.go:112,383,504,
   711`) to take `UserRepositoryInterface` (`handler/user_handler.go:19`).
   Check that the interface has every method they call; add any missing.
2. Fake: GRC's `panicUserRepository`, or PROTO's testify mock with no
   expectations (fails on an unexpected call). See the open question.
3. New `handler/auth_handler_test.go`: login completes the gate; lockout
   before the repository; wrong password 401 body unchanged.
4. New user-route sequence test, and replace the four mock-only tests in
   `user_handler_test.go` with ones that call the handler.
5. Add a 429 case to the example and profile sequence tests.
6. Port the rest only beside its fix, because each asserts that fix:
   change-context (B005), reissue session id (B004), logoff revocation
   (B003), unknown login (B007), client IP (B007 XFF), 9020 (B010), 409
   (B012), `mapRepoError` (B028), block (B034), body 413 (B011).

Traps: GRC's module path `github.com/Radley-House/radley-grc` and its
`appHelper` / `newTestAppServices` names; PROTO uses `app` and
`testServices` (`handler/example_handler_test.go:88`). Leave out
`AccountStatus` (`auth_handler_test.go:65`) and the invited-login test
(`:146`) unless B035 lands. GRC's cookie test expects no `Max-Age`; that
holds only after B029.

Tests: Go only; this item is the tests.

#### Open question for Brandon

One fake style for "repository must not be reached": GRC's
`panicUserRepository` or PROTO's testify mock? Recommendation: keep
testify for consistency, and add a comment on each sequence test that an
unexpected call is the failure.

### B040 — Go gate: `gofmt -l` + pinned staticcheck; delete `.golangci.yml`

- **Priority:** P2  **Area:** bin, go-api, docs, skills
- **GRC:** `bin/310-test-repo-model:13-37,78-89,91-103`; `CLAUDE.md:93-96`; `.claude/skills/ticket-workflow/SKILL.md:105,107`; `tickets/03-done/RG0015-api-dead-code-config-loader-lint/RG0015-api-dead-code-config-loader-lint.md:65-68,228-244`; `doc/902-project-commands.md:67`; `bin/310:13-18,80-89`   **PROTO:** `bin/300-run-tests:58-89` (`go vet` :64-66); `src/api-srv/.golangci.yml:1-21` (`lll: 120`, unused); `doc/105-api-srv-tech-spec.md:56,349,381`; `CLAUDE.md:51-54`; `.claude/skills/ticket-workflow/SKILL.md:100-102`; `doc/902-project-commands.md:56,66`; `tickets/01-open/PS0011-*:22,81`
- **Ticket:** RG0015 / `93571f5`; RG0043 item 14; RG0006 / `0cf57f7`
- **What:** PROTO's gate has no format check or static analysis; `.golangci.yml`
implies one and says 120 columns. GRC fails on `gofmt -l` output and any
`staticcheck@v0.8.1` finding (`go run`, no install).
- **Port notes:** Run per module (`src/api-srv`, `src/load-test`); expect a
cleanup pass; `U1000` skips exported names. Add govulncheck (B014). Then
CLAUDE.md, skill, doc/902, PS0011.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Quality/DX gap: the Go gate checks less than the repo claims it checks.

**(1) The gate is `go vet` plus tests.** `bin/300-run-tests go` runs:

```bash
# bin/300-run-tests:65-66, 72
echo "[INFO] go vet"
go vet ./...
TEST_DATABASE_URL="$TEST_DATABASE_URL" go test -v ./... > "$GO_LOG" 2>&1 || true
```

It runs in `src/api-srv` only (`:64`). `src/load-test` is a second module
(`src/load-test/go.mod`) and no gate compiles, vets or analyses it. There
is no format check and no static analysis.

**(2) A lint config implies a gate that does not exist.**
`src/api-srv/.golangci.yml:4-21` enables eleven linters and sets
`lll: line-length: 120`. No script runs `golangci-lint`. The 120 contradicts
CLAUDE.md:51 ("break lines around 80 columns").
`doc/105-api-srv-tech-spec.md:56,349,381` describe golangci-lint as the
project's linting. `tickets/01-open/PS0011-api-dead-code-config-loader.md:
22,81` uses it as a baseline (59 issues) and acceptance (at most 10).

**(3) What a real analyser finds today.** Run read-only for this write-up:
`gofmt -l` is empty in both modules. Pinned `staticcheck@v0.8.1` reports:

- `src/api-srv/internal/handler/auth_handler.go:97:6: func
  resetFailedAttempts is unused (U1000)`. A successful login never clears
  the lockout counters. That is B010's bug, found by the tool.
- `src/load-test/main.go:365:2` and `:395:2`: `exampleID` and `updateTs`
  "never used" (SA4006). That is B042's broken create/update cycle.

So the cleanup pass the doc expects is two fixes, both already items.

**(4) The installed tools cannot run.** `~/go/bin/staticcheck` is 0.5.1;
on Go 1.25.4 (both `go.mod:3`) it fails with "internal error in importing
... unsupported version: 2". An unpinned, installed linter drifts out of
step with the toolchain silently.

#### What GRC changed

RG0015 (`93571f5`) added go vet and staticcheck to `bin/310-test-repo-model`;
RG0006 added the gofmt check; `5089ac0` deleted `.golangci.yml`.

```bash
# bin/310-test-repo-model:30, 81-82, 99-100
STATICCHECK_VERSION="v0.8.1"
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
    (cd "$MODULE_DIR" && go run \
        "honnef.co/go/tools/cmd/staticcheck@$STATICCHECK_VERSION" ./...) \
```

Rules from the ticket and script comments (`bin/310:13-21,25-37`): never
rewrite code, only check; zero findings is the gate; fix or write
`//lint:ignore <check> <reason>`; run per module, because a root `./...`
missed a planted finding (RG0015 md `:236-243`). `go run pkg@version`
needs no install and no `go.mod` entry. RG0015 also found `U1000` skips
exported names (`:228-233`). GRC documents it in `CLAUDE.md:93-96`,
`.claude/skills/ticket-workflow/SKILL.md:105,107` and
`doc/902-project-commands.md:67`. No test pins the gate itself.

#### Proposed fix for PROTO

1. `bin/300-run-tests` go block (`:59-88`): before `go vet`, loop over
   `MODULE_DIR_LIST=(src/api-srv src/load-test)`: `gofmt -l .` fails on
   any output; `go vet ./...`; `go run honnef.co/go/tools/cmd/staticcheck@
   $STATICCHECK_VERSION ./...`. Pin the version at the top of the script.
2. Fix the three findings above under B010 and B042, or land B040 after
   them. Do not suppress them.
3. Delete `src/api-srv/.golangci.yml`; drop it from
   `doc/105-api-srv-tech-spec.md:56,349,381`.
4. Docs: one bullet in CLAUDE.md beside the 80-column rule; update
   `.claude/skills/ticket-workflow/SKILL.md:100-102` and
   `doc/902-project-commands.md:56,66`; rewrite PS0011's baseline and
   acceptance (`:22,81`) to "staticcheck zero findings".
5. govulncheck (B014): GRC does not run it in any gate (checked `bin/`,
   CLAUDE.md, doc/902). Adding it is a PROTO choice, not a port. Pin it
   the same way if wanted.

Traps: GRC's `MODULE_DIR_LIST` names five modules under one `go.work`;
PROTO has two modules and no `go.work`, so paths are from the project
root. v0.8.1 needs Go 1.26: `go run` switched the toolchain to go1.26.8
automatically here, so a host with `GOTOOLCHAIN=local` needs 1.26 (ties to
B014 and B047). GRC's message suggests `golines -m 80`; PROTO has no
golines, so say `gofmt -w`.

Tests: none new; verify by planting a finding in each module once.

#### Open question for Brandon

Run govulncheck in the same gate (network on every run, can go red on a
new advisory with no code change), or keep it a manual step? Recommend:
gate it in `bin/300 all` only, not `go`, so the per-commit run stays fast
and offline.

### B041 — Clean-tree gate: eslint as a check; `git status` stays empty

- **Priority:** P2  **Area:** bin, docs, skills, etc, web-app
- **GRC:** `bin/320-test-web-service:7-12,45-56`; `bin/lib/eslint-report.py:1-67`; `CLAUDE.md:89-92`; `README.md:80-82`; `src/api-srv/README.md:227-229`; `tests/README.md:50-56,123`; `.claude/skills/ticket-workflow/SKILL.md:109,113-116`; `src/web-app/.gitignore:14`; `.gitignore:17-20`; RG0006 md `:30-34,238`; RG0007 md `:122-124`; `bin/320:8-9,46-55`   **PROTO:** `bin/300-run-tests:3-4,91-104`; `CLAUDE.md:51-54`; `README.md:28-29`; `.claude/skills/ticket-workflow/SKILL.md:99-103`; `doc/902-project-commands.md:79-81`; `src/web-app/.gitignore`; `src/web-app/package.json:16`; `.gitignore:6-13`
- **Ticket:** RG0006 / `0cf57f7`, `719739b`, `3db3bd1`, `67b8700`, `8f1c530`; RG0007 / `d7750dc`
- **What:** PROTO never enforces lint; `test:coverage` dirties the tree. GRC
fails if `eslint --fix-dry-run` would rewrite and requires empty `git status --short` after a run; ignores root `package*.json`, load-test binary.
- **Port notes:** Add `coverage` to `src/web-app/.gitignore`. Fail on
`errorCount > 0` too? Brandon's call: max-len carve-out (:9-12)? Typo B088.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Quality/DX gap: two promises in the docs are not enforced.

**(1) Nothing runs eslint.** The web app defines a lint script and a
config with one house rule:

```js
// src/web-app/eslint.config.js:27-28
// Errors go through utils/report-error.ts, never straight to the console.
'no-console': 'error',
```

`src/web-app/package.json:11` has `"lint": "eslint ."`, but no `bin/`
script calls it (`grep -rn lint bin/` finds only ticket-lint lines in
`bin/000-rebrand`). `bin/300-run-tests` unit mode runs vitest only
(`:91-104`). CLAUDE.md, `.claude/skills/ticket-workflow/SKILL.md:99-103`
and `doc/902-project-commands.md` list no lint step. A `console.log` or an
unhandled promise ships green. Read-only check for this write-up:
`npx eslint . -f json --fix-dry-run` today reports 0 errors, 0 warnings,
0 files it would rewrite. So the gate can start strict at no cost.

**(2) "No test script writes to the tree" has one hole and one leak.**
`bin/300-run-tests:3-4` says none writes to the source tree, and
`doc/902-project-commands.md:79-81` repeats it. But:

- `src/web-app/package.json:16` `"test:coverage": "vitest run --coverage"`
  writes `src/web-app/coverage/` (vitest's default report dir;
  `vitest.config.ts:12-17` sets none). Neither `src/web-app/.gitignore`
  nor the root `.gitignore` lists `coverage`, so the run dirties
  `git status`.
- Correction to the doc: the "load-test binary" is worse than unignored.
  `src/load-test/load-test` (11 MB) is **tracked** in git
  (`git ls-files`), added by `6b0ecf0`. `bin/330-test-load:25` now builds
  to `dist/load-test`, so the tracked file is a stale binary nobody uses.
- `.gitignore:23` names `bin/200-run-tests`, a retired script (B088).

Nothing checks `git status --short` after a run, so a regression in
either direction goes unseen.

#### What GRC changed

RG0006 (`0cf57f7`, docs in `719739b`, merged `3db3bd1`) made eslint a
check in `bin/320-test-web-service`, run before vitest and pytest:

```bash
# bin/320-test-web-service:50, 55
(cd src/web-app && npx eslint . -f json --fix-dry-run > "$ESLINT_JSON") || true
python3 "$SCRIPT_DIR/lib/eslint-report.py" "$ESLINT_JSON"
```

`bin/lib/eslint-report.py:25,45-57` fails only when eslint *would rewrite*
a file (`output` present in the JSON). Errors and warnings are printed,
not fatal (`:26-43`), because GRC carries a `max-len` backlog at 200
columns (`:9-12`; `src/web-app/eslint.config.js:37`). The report goes to
`tmp/test-reports/`. GRC added `coverage` to `src/web-app/.gitignore:14`,
`src/load-test/load-test` to `.gitignore:14`, and `/package.json`,
`/package-lock.json` to `.gitignore:19-20` (RG0002 `67b8700` untracked
them). The rule "`git status --short` MUST be empty after a run" is in
`CLAUDE.md:89-92` and `.claude/skills/ticket-workflow/SKILL.md:109,113-116`.
No test pins it; it is a manual check.

#### Proposed fix for PROTO

1. `bin/300-run-tests` unit block (`:91-104`): before vitest, run
   `npx eslint . --max-warnings 0` from `src/web-app`; fail on non-zero.
   PROTO has no `max-len` backlog, so the plain exit code is enough and
   `bin/lib/eslint-report.py` is not needed. `eslint` never gets `--fix`.
2. `src/web-app/.gitignore`: add `coverage`.
3. `git rm --cached src/load-test/load-test`; add
   `src/load-test/load-test` to `.gitignore`. Skip GRC's root
   `package*.json` lines: PROTO has none and `.gitignore:12-13` already
   covers a stray root `node_modules`.
4. End of `bin/300-run-tests`: if `git status --porcelain` differs from
   its value at start, print it and fail. Comparing start to end lets a
   developer run the gate on a dirty tree.
5. Docs: CLAUDE.md one bullet; skill `:99-103` add `git status --short`
   must be empty; `doc/902-project-commands.md:56,79-81` name the lint
   step. Fix `.gitignore:23` (B088).

Traps: GRC's step is in `bin/320`; PROTO has one gate, `bin/300`. Do not
copy the max-len carve-out or GRC's `--skip-lint` flag.

Tests: none new; prove once by adding a `console.log` (gate red) and by
running `npm run test:coverage` (tree stays clean).

#### Open question for Brandon

Fail on any eslint error or warning (`--max-warnings 0`), or only on
auto-fixable rewrites as GRC does? Recommend: any error or warning,
since PROTO is at zero today and the GRC carve-out exists only for a
backlog PROTO does not have.

### B042 — Load test: fake metrics, broken cycle, dev target, argv password

- **Priority:** P2  **Area:** tests, bin, etc, skills
- **GRC:** `src/load-test/main.go:79-86,163-166,179-186,219-223,244-248,285,322-358,666-695`; `src/load-test/client.go:24-25,62-81,122-131`; `bin/330-test-load:10-23,54-72`; `etc/loc-test-api-srv.env:52-58`; `.claude/skills/ticket-workflow/SKILL.md:117-119`; `src/load-test/main.go:163-185`; `etc/loc-test-api-srv.env:1-9,19-21,31-36`   **PROTO:** `src/load-test/main.go:182-183,335-460` (:365,395,413,417), `:60-103,239-246,185-190,589-592,722-766,814-857` (`SystemMonitor`), `:263,298`; `src/load-test/client.go:126,193`; `bin/330-test-load:6-29`; `etc/loc-api-srv.env:36`; `etc/loc-test.env:1-41`; `etc/e2e-api-srv.env:1-60`
- **Ticket:** RG0006 / `c37a717`; RG0041 / `31970ae`; RG0043 item 33 / `720f5e1`
- **What:** PROTO fakes CPU/memory, times a create as update, double-closes
`stopChan`, writes the dev DB as tenant 1, takes `-password` on argv. GRC:
measured only, test API, `TEST_PERSONA_PASSWORD`, tenant 2.
- **Port notes:** Keep the example cycle (+`UpdateExample`); 330 starts a
private instance (bin/300 :140-182) or needs 8082. After B018.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

`bin/330-test-load` runs 100 workers for 60 s, each looping login,
create, update, get, delete, logout. Correctness bugs plus a data-safety
gap; no API hole. The report is untrustworthy and the dev DB gets dirty.

**(1) Invented server metrics.** `--monitor-server` appends constants:

```go
// src/load-test/main.go:85-90
// Placeholder: collect CPU and memory usage
...
metrics.CPUUsage = append(
	metrics.CPUUsage,
	50.0,
) // Placeholder
```
(`:91-94` does the same with 256 MB.)

The HTML report prints them as "Server CPU" and "Server Memory"
(`:590-591`, aggregated at `:726-762`, console at `:815-848`). A reader
sees a measured 50 % / 256 MB that nobody measured.

**(2) The cycle is broken.** `runCycle` (`:335`) logs in, then creates
an example three times (`:365`, `:395`, `:413`). It sends a second
`OpLogin` metric (`:381-387`) carrying the create's duration, and labels
the third create `OpUpdate` (`:417`). `UpdateExample`
(`src/load-test/client.go:181`) is never called. Only the last row is
deleted (`:439`), so two rows leak per cycle. The "update" and "login"
latencies in the report are wrong. staticcheck flags `:365` and `:395`
(SA4006, see B040).

**(3) Ctrl-C panics.** The signal handler closes `stopChan` (`:263`);
the normal end closes it again (`:298`). A Ctrl-C during the run panics
"close of closed channel" instead of writing the report.

**(4) Under-count.** After `close(metricsChan)` (`:303`) nothing waits
for the collector (`:266-279`) to drain before the report reads it.

**(5) It writes the dev database as tenant 1.** `bin/330-test-load:15-18`
takes `TEST_BASE_URL` from `etc/loc-api-srv.env:36`
(`http://localhost:8080`, the dev API). The default login is `admin`
(`main.go:182`), which the fixtures put in tenant 1, and the subject
`admin` is exempt from the per-user rate limit
(`src/api-srv/internal/security/app_helper.go:348`). So a run writes
thousands of leaked example rows into the developer's real tenant and
never meets the limiter it claims to load.

**(6) Password on argv.** `main.go:183` defaults `-password` to
`admin123!`; any other must go on the command line, visible in `ps` and
shell history. Low impact locally (fixture password), real on a shared
build host.

#### What GRC changed

- RG0006 `c37a717` deleted `--monitor-server`, `SystemMonitor` and every
  report line it fed: "a report must not carry invented numbers".
- RG0043 item 33 / `720f5e1`: `runCycle`
  (`src/load-test/main.go:322-358`) is one create, one update, one get,
  one delete, each timed by a `record` helper. `sync.Once` guards the
  close (`:222-223`); `collectorDone` waits for the drain (`:246-248`,
  `:285`).
- Password: `-env <file>` reads `TEST_PERSONA_PASSWORD` (`:166`,
  `:179-186`, `readEnvKey` `:666-695`); `-user` and the password are
  required (`:79-85`). `etc/loc-test-api-srv.env:58` holds it, committed
  in clear (scratch databases only).
- RG0041 `31970ae`: `bin/330-test-load:56-62` reads the test API env
  only; `client.go:24-25,122-131` refuses a login outside tenant 2.

#### Proposed fix for PROTO

1. Delete `SystemMonitor`, `SystemMetrics`, `--monitor-server` and the
   report/summary blocks (`main.go:57-103,185-190,239-246,482-502,
   589-592,722-766,814-857`).
2. Rewrite `runCycle` as create, `UpdateExample`, get, delete. Keep the
   example cycle (PROTO's only scaffold entity) rather than GRC's
   controls.
3. `sync.Once` around `close(stopChan)`; a `collectorDone` channel.
4. Drop the `-password` flag and its default; add `-env` and read one key
   with a tiny stdlib reader (the module is stdlib only). PROTO has no
   `TEST_PERSONA_PASSWORD`: add one to `etc/e2e-api-srv.env` (or to
   `etc/loc-test.env`, which is a sourced shell file with `export`, so
   the reader must strip that). Make `-user` required, default none.
5. Target the private test API, not 8080. Either `bin/330` starts it as
   `bin/300` py mode does (`bin/300-run-tests:140-182`, `:8082` from
   `etc/e2e-api-srv.env:55,60`), or it refuses unless `:8082` answers.
6. After B018 lands tenant-2 logins, refuse a login outside tenant 2 in
   `client.go` Login, and use a non-admin, MFA-exempt test user so the
   rate limiter is exercised.
7. Update `doc/902-project-commands.md:100-108`, which documents
   `--password 'admin123!'` at `:105`.

Traps: GRC's `CreateControl`/`UpdateControl`, its module path in
`go build` (`bin/330:41`) and `bin/216` do not exist in PROTO.
Depends on B018 (tenant 2) and pairs with B040 (staticcheck sees it).

Tests: Go unit test for the env-key reader; a smoke run with
`--iterations 1 --concurrency 1`.

#### Open question for Brandon

Should `bin/330` start its own private API like `bin/300 py`, or require
one already running on 8082? Recommend: start its own, so a load run can
never hit the dev database by accident.

### B043 — pytest: core tests off `/example`; regression gates

- **Priority:** P2  **Area:** tests
- **GRC:** `tests/test_auth_flow.py:36-61,93-121,225-253`; `tests/test_z_error_scenarios.py:74-98`; `tests/test_security_context_isolation.py:295-421`; `tests/test_user_management.py:196-214,216-278,280-301,303-331`; `tests/test_own_profile.py:1-11,25-59,62-67,127-167,201-217,235-261`   **PROTO:** `tests/test_auth_flow.py:34-59,89`; `tests/test_z_error_scenarios.py:27-55,179-197`; `tests/test_security_context_isolation.py`; `tests/test_user_management.py:196`; `tests/test_own_profile.py:9-69`; `bin/300-run-tests:142-155`; `bin/300-run-tests:140-155`; `handler/app_user_group_handler.go:364`; `src/api-srv/internal/auth/jwt.go:44`; `auth_handler.go:25-26,158`
- **Ticket:** `bab2d0d`; `7bceb6b`; RG0041 / `8e05f63`; RG0001 / `d2ad0c8`; RG0005 / `b6c0412`; RG0014 / `b3a1ea2`; RG0002 / `95c7e61`; RG0016 / `47fa7c4`
- **What:** CSRF tests target `/changeOwnPassword`, surviving a derived app's
scaffold deletion; rate-limit test reads the server limit and aligns to the
minute. New gates that pass today: userName bounds, JWT `iss`, error classes,
`totpSeed` absent, duplicate 409, own-profile on a throwaway login.
- **Port notes:** Needs `api_env`/`db` (B018). GRC `/me` wraps `data.profile`;
PROTO `data`. Keys per B038.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Quality gap in the pytest suite (`tests/`). Three things: core security
tests hang off the scaffold entity, two tests are fragile, and several
behaviours that hold today have no test to keep them holding.

**(1) Core tests use `/example`.** The CSRF tests post to the scaffold
route:

```python
# tests/test_auth_flow.py:37-40 (same at :44-48, :54-58)
response = test_client.session.post(
    f"{test_client.base_url}/example",
    json={"exampleText": "Test", "securityContext": 1},
```

`tests/test_z_error_scenarios.py:33-35,47-49` and the rate-limit test
(`:184-186`) do the same. A derived app deletes `/example` first. The
CSRF tests then get 404 or 405 and fail, or get rewritten in a hurry, and
the template's CSRF regression test is lost with the scaffold. None of
them checks the error body, so any 403 passes.

**(2) The rate-limit test is fragile.** It hardcodes the limit in a
comment and sends `range(250)` with `time.sleep(0.01)`
(`test_z_error_scenarios.py:181-191`). It only passes because
`bin/300-run-tests:148-155` copies `MAX_USER_REQ_PER_MIN=200` from
`etc/loc-api-srv.env:19` into the test API. Raise the dev limit above 250
and it fails. The limiter buckets by clock minute
(`src/api-srv/internal/security/app_helper.go:404-409`,
`RPM*<user>*<minute>`); a burst that starts late in a minute splits
across two buckets and may never see a 429.

**(3) The own-profile test edits a shared fixture user.**
`tests/test_own_profile.py:9-62` changes `basic`'s name, e-mail and
preferences, then restores them at the end. A failed assert in between
leaves `basic` modified for every later test. It also reads `/me` as
`data` (`:13`).

**(4) Behaviour that holds today but is unpinned:**
- `/auth` refuses an empty or >50-char `userName` with the same 401
  (`src/api-srv/internal/handler/auth_handler.go:25-26,158-160`).
- JWT `iss` is `APP_NAME` without quotes (`auth/jwt.go:44`;
  `config/config.go:55` strips quotes).
- `/userList` never carries `totpSeed` (only Go pins it,
  `handler/user_handler_test.go:840`).
- Duplicate user name or group name answers 409, and deleting a group
  with members answers 409 by string match
  (`handler/app_user_group_handler.go:342`).
- `DELETE /appUserGroup` returns the deleted id
  (`handler/app_user_group_handler.go:364`).

When B028 replaces string matching with sentinels, nothing over HTTP
proves the status codes survived.

#### What GRC changed

- `bab2d0d` (scaffold removal) re-pointed CSRF tests at
  `/changeOwnPassword` (`tests/test_auth_flow.py:36-61`). That route
  needs only LOGON and runs CSRF before the body.
- `7bceb6b` + RG0041 `8e05f63`: the rate test reads the limit with
  `api_env("MAX_USER_REQ_PER_MIN")`, waits for a fresh minute, sends
  `limit + 50` with no sleep, then waits out the minute
  (`tests/test_z_error_scenarios.py:74-98`).
- RG0001 `d2ad0c8`: userName bounds 0, 51, 300 are 401, and 50 is looked
  up (`test_auth_flow.py:93-121`). RG0005 `b6c0412`: `iss` equals
  `api_env("APP_NAME")`, no quotes (`:225-253`).
- RG0014 `b3a1ea2`: duplicate, in-use and invalid-value keep 409/409/400
  (`test_security_context_isolation.py:295-421`). RG0002 `95c7e61`:
  deleted group id (`test_user_management.py:303-331`); plus `totpSeed`
  absent (`:196-214`) and duplicate 409s (`:216-301`).
- RG0016 `47fa7c4`: own-profile tests run on a throwaway login created
  and deleted by a class fixture (`test_own_profile.py:1-67`).

#### Proposed fix for PROTO

1. CSRF tests in `test_auth_flow.py:34-59` and
   `test_z_error_scenarios.py:27-53`: post to `/changeOwnPassword` with
   `{"currentPassword": "x", "newPassword": "y"}`; also assert the error
   text ("Missing CSRF token" / "CSRF validation failed").
2. Rate test: read the limit from the test API's env (needs B018's
   `api_env`), align to the minute, no sleep, wait out the minute after.
   Target `/me` (LOGON only) instead of `/example`.
3. Own profile: a class fixture that creates a throwaway user in the
   Basic group, signs in, and deletes it after. Keep PROTO's `data`
   shape for `/me`; GRC's `data.profile` is from its own `/me` change.
4. New gates, each passing today: userName bounds; JWT `iss`
   (`APP_NAME` from the test env); `totpSeed` absent from `/userList`;
   duplicate user and group 409; group in use 409; deleted group id.
5. Keys per B038 (`password`, `userIdList`). Leave out GRC's `grc`
   fixture, `/controlList`, `invalid-value` 400 (needs B028's
   `ErrInvalidValue`) and any 413 case (B011).

Depends on B018 (`api_env`, tenant 2 users) and B038 (wire keys).
Tests: pytest only; this item is the tests.

#### Open question for Brandon

None.

### B044 — tests/README.md rewrite

- **Priority:** P2  **Area:** tests, docs
- **GRC:** `tests/README.md:1-9,90-97,105-114,116-123`   **PROTO:** `tests/README.md:1-61` (dev server :15-20,29-32; pyotp :61; `test_own_profile.py`/`test_zz_mfa_totp.py` absent)
- **Ticket:** RG0041 / `31970ae`; RG0006; RG0003
- **What:** PROTO's README runs pytest against the dev API on 8080 and is stale
on files and deps. GRC states test API + tenant 2 only, MFA ordering/XFF,
helpers, reports.
- **Port notes:** Write fresh for bin/300; GRC's is stale at :79,84,129,136.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Doc gap: `tests/README.md` (61 lines) is the only guide to the pytest
suite, and parts of it are wrong or missing. A new developer following
it can point the suite at the dev database.

**(1) It offers a manual run against the dev API.**

```bash
# tests/README.md:18-19
bin/215-run-api-srv                        # reads TEST_BASE_URL from etc/loc-api-srv.env
TEST_BASE_URL=https://localhost:8080 tests/.venv/bin/python -m pytest tests/
```

`bin/215` serves the developer's database. The suite creates and deletes
users, groups and examples, changes `basic`'s profile, and locks
accounts. Run this way it does that to the dev data in tenant 1, under
dev rate limits. It also says `https` on `:19` while `:5` says there is
no TLS locally and `:32` says `http`.

**(2) The configuration section describes code that does not exist.**
`:29-32` says `bin/300-run-tests` reads `TEST_BASE_URL` from
`etc/loc-api-srv.env` "or the file `$TEST_API_ENV` points at".
`TEST_API_ENV` appears nowhere in `bin/`, `tests/` or `etc/`. The real
flow: `bin/300-run-tests:148-155` copies `etc/e2e-api-srv.env` (with the
two rate limits taken from the dev file), reads `TEST_BASE_URL` from that
copy (`:156`, `http://127.0.0.1:8082` per `etc/e2e-api-srv.env:60`),
refuses if something already listens there (`:158-160`), starts a private
API, and exports `TEST_BASE_URL` to pytest (`:200`).

**(3) The file list is stale.** `:49-57` omits `test_own_profile.py`
(GET/PUT `/me`) and `test_zz_mfa_totp.py` (enrol, verify, lockout).
`:61` lists "pytest, requests, pytest-html"; `tests/requirements.txt`
also pins `pyotp==2.9.0`. Nothing explains the `test_a_` / `test_z_` /
`test_zz_` prefixes: pytest collects alphabetically, so the rate-limit
and lockout tests run last on purpose (`tests/test_auth_flow.py:110-112`
says so in a comment).

**(4) Missing rules:** which tenant tests may touch, how lockout tests
avoid locking the shared loopback IP, where reports go, and that the run
must leave `git status` clean.

#### What GRC changed

RG0041 (`31970ae`) and RG0006/RG0003 rewrote GRC's `tests/README.md`:

- `:3`: the suite runs against the **test** API, "tenant 2 of the scratch
  `loc_radley_grc_e2e_db`, never the real database or tenant 1 (RG0041;
  `conftest.py` aborts otherwise)".
- `:5-9`: each pinned dependency with the advisory that set its floor.
- `:90-97`: per-file coverage. The MFA file is collected last
  (`test_zz_`), and "every request that can increment a counter carries
  its own `X-Forwarded-For` (a fresh address per run)" so lockouts never
  touch the loopback bucket; the lockout runs on a throwaway user.
- `:105-114`: `TestClient`, the MFA helpers and each fixture.
- `:116-123`: reports under `tmp/test-reports/`, gitignored (RG0006).

GRC's own copy is stale too: `:79` says the script reads
`etc/loc-api-srv.env` (it reads `etc/loc-test-api-srv.env`), and `:84`,
`:129`, `:136` still show port 8080 and the dev env file. Copying it
would import those errors.

#### Proposed fix for PROTO

Write `tests/README.md` fresh around `bin/300-run-tests py`. Sections:

1. **What it is**: HTTP tests against a private API on `:8082` over the
   scratch `E2E_DB` (`etc/loc-test.env`), started and stopped by
   `bin/300`. Tenant rule per B018 (tests act in tenant 2 only).
2. **Running**: `bin/300-run-tests py`, and `all`. Manual run only
   against an API started from `etc/e2e-api-srv.env`, never `bin/215`.
3. **Configuration**: the real flow from `bin/300-run-tests:141-160,200`,
   including why the rate limits are copied from the dev file.
4. **Files**: every `test_*.py`, with the ordering prefixes explained.
5. **Client and fixtures**: `TestClient`, `test_client`,
   `basic_user_client`, `example_tracker` (`tests/conftest.py:14,147,
   169,194`); add `api_env` and `db` when B018 adds them.
6. **Lockout hygiene**: per-run `X-Forwarded-For` and a throwaway lockout
   user, once B019 lands; until then, say the MFA file runs last because
   it locks the loopback IP.
7. **Requirements**: list all four pins; point at B017 for the bumps.
8. **Reports and clean tree**: `tmp/test-reports/`, `tmp/test-logs/`;
   `git status` empty after a run (B041).

Traps: no `bin/320`, `bin/216`, `bin/510 --go-only`, `grc` fixture,
`test_grc_api.py`, eslint step or `matrix.json` in PROTO. Use names from
`etc/app.env`-derived files, not `loc_radley_grc_*`. Write it last in
the test batch (after B017, B018, B019, B043) so it describes the
final state.

Tests: none; check every command in it by running it once.

#### Open question for Brandon

None.

### B045 — infra scripts derive names from the app root; shared copies equal

- **Priority:** P2  **Area:** infra, docs
- **GRC:** `infra/uat-unified001/bin/uat-grc-install-api:21-30`; `uat-grc-build:16-24`; `uat-grc-publish-web-client:14-21`; `uat-grc-renew-web-certs:8-10`; `infra/prd-unified002/bin/prd-grc-promote-uat:1-19,44-48`; `prd-grc-promote-uat-and-deploy:13-47`; `prd-grc-backup-all:17-19`; `infra/README.md:21-25`   **PROTO:** `infra/uat/bin/*`, `infra/prd/bin/*` (e.g. `uat-proto-install-api:5`); `infra/prd/bin/prd-proto-promote-uat:1-61`; `prd-proto-promote-uat-and-deploy:1-52`; `infra/README.md:44-48`
- **Ticket:** RG0013 / `3a3fc94`, `a57d6ad`
- **What:** `APP_ENV=$(basename $SRV_ROOT)` names env, lock, scripts, so
`install-api`, `publish-web-client`, `renew-web-certs` are byte-identical
across hosts. Hook scripts re-exec as `shrmpl-hook`; PROTO's uat ones run as
root and leave root-owned files.
- **Port notes:** Fewer names for `bin/000-rebrand:88`. README: diff shared
scripts after edits. Promote passes `$DEPLOY_TAG`.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Each host's scripts live in `infra/uat/bin/` and `infra/prd/bin/` and are
copied verbatim to `/srv/<env>-proto/bin/` (`infra/README.md:44-48`). Every
script already finds its app root from its own path, then hard-codes its
environment in the names it builds from that root:

```bash
# infra/uat/bin/uat-proto-install-api:4-5, 29, 81
SRV_ROOT="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/.." && pwd)"
DEPLOY_ENV="$SRV_ROOT/etc/uat-proto-deploy.env"
MIGRATE_ENV="$SRV_ROOT/etc/uat-proto-migrate.env"
echo "          systemctl status uat-proto-restart.path"
```

So the uat and prd copies of one job are different files, and `diff`
shows drift beyond the env name. `prd-proto-install-api:14-19` and
`prd-proto-publish-web-client:18-24` refuse any user but `shrmpl-hook`;
the uat copies have no user check. `publish-web-client:29` has a different
hint per host. `renew-web-certs` differs only in the env name (`:6`). A
fix to one copy must be retyped into the other, and nothing checks it.

**Running as root leaves root-owned files.** The uat build re-execs itself
as `shrmpl-hook` (`uat-proto-build:5-8`), but an operator who reinstalls a
tag by hand runs `sudo /srv/uat-proto/bin/uat-proto-install-api <tag>` as
root, and nothing stops it. The redirect at `uat-proto-install-api:57-60`
then creates `libexec/proto-api.revision` as root, mode 0644. On the next
webhook build, `shrmpl-hook` cannot open that file: `install-api` dies
under `set -e` after the new binary is swapped in (`:55-56`) and before the
restart request (`:65`). The new binary sits on disk unrestarted and Slack
reports a failed build. A root-run `publish-web-client` likewise leaves a
root-owned `web-pub/` that the hook's `rm -rf "$OLD_DIR"` (`:48`) cannot
delete. This is a correctness bug on the deploy path, not a security hole.

#### What GRC changed

RG0013 / `3a3fc94` ("Scripts derive every name from their app root, so the
shared uat/prd copies are byte-identical") and `a57d6ad` (README).

1. One more line after `SRV_ROOT`: `APP_ENV="$(basename "$SRV_ROOT")"`,
   and every name is built from it: `uat-grc-install-api:27-29`
   (`DEPLOY_ENV="$SRV_ROOT/etc/$APP_ENV-deploy.env"`), the restart-unit
   hint (`:102`), the build lock (`uat-grc-build:49`), the scripts the
   build calls (`:137`, `:139`), promote (`prd-grc-promote-uat:18-19`),
   promote-and-deploy (`:20-21`, `:39-41`), backup (`prd-grc-backup-all:18-19`).
2. Every hook-run script re-execs instead of refusing
   (`uat-grc-install-api:21-24`): `exec sudo -u "$TARGET_USER" -H "$0" "$@"`.
   Root can run it by hand and it still writes as `shrmpl-hook`.
3. `install-api`, `publish-web-client` and `renew-web-certs` are
   byte-identical across GRC's two host dirs (confirmed with `diff`).
   `infra/README.md:21-25` states the rule and says to `diff` them after an
   edit. No automated check enforces it.
4. Promote-and-deploy passes the tag explicitly (`:40`,
   `install-api "$DEPLOY_TAG"`). PROTO's call with no argument
   (`prd-proto-promote-uat-and-deploy:37`) already defaults to `DEPLOY_TAG`,
   so this is cosmetic.

GRC still hard-codes the slug in blob names and in `*-start`/`*-stop`.

#### Proposed fix for PROTO

1. In every `infra/{uat,prd}/bin/*` script except `*-start`/`*-stop`, add
   `APP_ENV="$(basename "$SRV_ROOT")"` and build every env file, lock,
   unit and called-script name from it. Keep the literal slug in blob and
   binary names (`proto-api`, `proto-web`, `proto-migrate`); rebrand
   rewrites those.
2. Put the `shrmpl-hook` re-exec first in `build`, `install-api`,
   `publish-web-client`, `promote-uat`, `promote-uat-and-deploy`. Replace
   the prd "Must run as" refusals with it. `renew-web-certs` and
   `backup-all` stay root (cron).
3. Make `install-api`, `publish-web-client`, `renew-web-certs` identical
   across `infra/uat/bin` and `infra/prd/bin` (pick one error hint).
4. Add a check that fails when a pair differs, e.g. a few lines in
   `scripts/` beside B053's path check, run by the same gate.
5. Drop `BACKUP_ENV` from `infra/prd/etc/prd-proto-backup.env` and derive
   it. Coordinate with B020, which rewrites the same backup script.
6. `infra/README.md:44-48`: state the basename rule, the constraint it
   implies (the app root must be named `<env>-<slug>`), and the pair list.
7. Correction to the item: `bin/000-rebrand:88` renames files, and the
   filenames still carry the slug after this change. The gain for rebrand
   is fewer in-file `uat-proto` tokens for step 1 (`:61`), which it already
   handles. The real gain is one file per job.
8. Order: B024 first (app root `root:shrmpl-hook 1775`), or a hook-run
   publish cannot rename `web-pub/`. B046 and B047 edit the same scripts;
   land them on top of this.

Tests: the pair-diff check above. Shell syntax with `bash -n` on each.

#### Open question for Brandon

Keep two byte-identical copies per shared script plus a diff check (GRC's
model, keeps `infra/<host>` a file-for-file mirror), or keep one copy under
`infra/shared/bin/` that both hosts install? Recommendation: two copies
plus the check, so each host dir stays a literal mirror of the server.

### B046 — install-api: run the incoming migrator before touching libexec

- **Priority:** P2  **Area:** infra
- **GRC:** `infra/uat-unified001/bin/uat-grc-install-api:1-17,47-70,72-80,114-117`   **PROTO:** `infra/uat/bin/uat-proto-install-api:27-58,93-96` (:32-33) (+ prd); `infra/README.md:17-29`
- **Ticket:** RG0013 / `3a3fc94`
- **What:** PROTO installs the migrator before `verify`, so a refusal strands an
older migrator beside a newer API. GRC runs the schema step from the pull dir,
swaps nothing until it succeeds, keeps `*.prev`, prints rollback.
- **Port notes:** Skip worker wait (:84-104). Brandon's call: `api-srv` in
post-sales too (RG0013 #4)?
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (Brandon: yes, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

`infra/uat/bin/uat-proto-install-api` (prd twin identical here) pulls a
build into `stg/pull.$$`, migrates, swaps the API binary and restarts. The
incoming migrator is installed into `libexec/` first, and only then asked
whether the database is acceptable:

```bash
# infra/uat/bin/uat-proto-install-api:32-33, 35-37
install -m 0755 "$PULL_DIR/proto-migrate" "$SRV_ROOT/libexec/proto-migrate.new"
mv -f "$SRV_ROOT/libexec/proto-migrate.new" "$SRV_ROOT/libexec/proto-migrate"
MIGRATE="$SRV_ROOT/libexec/proto-migrate -config $MIGRATE_ENV"
$MIGRATE verify
```

`verify` (`src/api-srv/internal/migrate/migrate.go:215-230`) fails when the
database has an applied version with no file (it is ahead of the binary) or
a file whose checksum drifted. Under `set -e` the script stops there. The
comment at `:28` says "a failure leaves the old binary running", which is
true for `proto-api` but not for `proto-migrate`.

Concrete case: an operator reinstalls last week's tag after a migration
shipped. `verify` correctly refuses, but `libexec/` now holds last week's
migrator beside this week's API. Whoever next runs `libexec/proto-migrate`
by hand (`doc/330-presales-to-postsales-migration.md:169`) gets a drift
error from a binary that does not match what runs. The next good install
overwrites it; nothing is lost, but the host is in a state nobody chose.

Also: no previous binary is kept (`:55-56` overwrite `proto-api`). If
`/health` fails (`:94-96`), reinstalling an older `sha-` tag is the only
way back, and its migrator refuses `verify` if this deploy migrated. And an
artifact with no migrator is tolerated (`:49-50`, `[WARN]`), though the
build always ships one (`infra/uat/bin/uat-proto-build:73-74`). This is a
correctness and operability gap, not a security hole.

#### What GRC changed

RG0013 / `3a3fc94`. The ticket states the intent: "install-api migrates
with the **incoming** `migrate-srv` from the pull directory before swapping
anything". GRC's UAT install script (prd copy byte-identical):

1. Requires every binary and fixes the exec bit in the pull dir
   (`uat-grc-install-api:52-55`).
2. Runs `verify`, `pending`, `pg_dump`, `up` from the pull dir (`:57-70`):
   `MIGRATE=("$PULL_DIR/migrate-srv" "$MIGRATE_ENV")`. `libexec/` is
   untouched until the schema step succeeds.
3. Then swaps each binary, keeping the old one (`:74-78`):
   `cp -p "$LIBEXEC/$binary" "$LIBEXEC/$binary.prev"`.
4. On a health failure prints the exact rollback (`:115-117`): move each
   `.prev` back; the schema stays.
5. The header (`:14-17`): rolling back across a migration is a restore
   from `backup/`, not a reinstall.

`:84-104` wait for a worker service PROTO lacks. The same commit added
`verify`/`pending` to GRC's goose migrator, tested by `TestMigrationPlan`
(`src/migrate-srv/cmd/migrate/verify_test.go:11`). No test covers the
install script.

#### Proposed fix for PROTO

1. In `infra/uat/bin/uat-proto-install-api` and
   `infra/prd/bin/prd-proto-install-api`, after the pull: refuse unless both
   `proto-api` and `proto-migrate` are present, then `chmod 0755` both in
   `$PULL_DIR`. Delete the `[WARN]` branch.
2. Build the migrator command from the pull dir:
   `MIGRATE=("$PULL_DIR/proto-migrate" -config "$MIGRATE_ENV")`. Trap: PROTO's
   migrator takes `-config <file>`; GRC's goose binary takes the env file
   positionally. Do not copy GRC's array literally.
3. Keep `verify` -> `pending` -> `pg_dump` -> `up` as today, via the array.
   B050 takes the database URL (with password) off `pg_dump`'s argv here.
4. Then swap both binaries in a loop, `cp -p` each to `<name>.prev` first.
5. Add the rollback line to the health-failure message at `:94-96`, and a
   header comment stating the "rollback across a migration is a restore"
   rule. Fix the `:28` comment.
6. Skip GRC's worker wait and `grc.revision`; keep `proto-api.revision`.
7. Do it with or after B045, so uat and prd share one file.
8. Update `infra/README.md:107-109` (promotion-flow box): "the incoming
   `proto-migrate`, from the pull dir".

Tests: none automated for shell. Rehearse on a UAT host: install tag N+1
with a migration, then install tag N; expect `verify` to refuse and
`libexec/` unchanged (`sha256sum` before and after).

#### Open question for Brandon

GRC gave post-sales the presales binary names so the cutover moves no
names (RG0013 "Corrections", item 4). In PROTO only the API differs
(`api-srv` presales, `proto-api` post-sales: `infra/README.md:20`,
doc/330:39); the migrator is `proto-migrate` in both. Should post-sales
install `libexec/api-srv`? Recommendation: yes, in this ticket, since it
removes a row from the doc/330 name table and one rename step at cutover.

### B047 — UAT build: lock waits, per-step failure report, full-sha Slack

- **Priority:** P2  **Area:** infra
- **GRC:** `infra/uat-unified001/bin/uat-grc-build:1-13,40-41,45,47-54,56-62,63-73,77-83,85-94,140-159`; `infra/uat-unified001/shrmpl-etc/hooks/build-grc_HOOKID.env:11-13`   **PROTO:** `infra/uat/bin/uat-proto-build:21-24,44-52,62-66,129-138`; `infra/uat/shrmpl-etc/hooks/build-proto_HOOKID.env:9`
- **Ticket:** RG0013 / `da5eebd`; RG0043 / `535e7cb`, `b167658`
- **What:** `flock -n` drops a push landing mid-build. GRC waits 30 min (hook
3600), reports "FAILED at <step>", Slacks full sha at start/deploy/test,
preflights `git psql pg_dump`, names a stale host Go.
- **Port notes:** Toolchain check only if needed (B014). Replace "Radley GRC"
and module path (:99-101).
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

`infra/uat/bin/uat-proto-build` is what the GitHub webhook runs on the UAT
host for every push to `BRANCH_TO_CHECKOUT`: reset the checkout, run the
test gate, build, push to the registry, install, publish, Slack.

**(1) A push that lands during a build is dropped.** The lock is
non-blocking: `flock -n 200 || { echo "[ERROR] Another build ..."; exit 1; }`
(`:22-24`). A run includes `bin/300-run-tests` (Go, vitest, Playwright,
pytest), so it takes minutes. Push A starts a build; push B arrives two
minutes later; B's run exits 1 at once. A checked out the head before B
existed, so B is never built and UAT sits one commit behind until the next
push. The `notify_failure` trap is armed later (`:52`), so this exit sends
no Slack. Whether shrmpl-hook reports a non-zero exit itself is not
visible in this repo. This is a correctness bug.

**(2) A failure does not say where or which commit.** `:49` sends
`":x: proto UAT build FAILED on $(hostname) (exit $exit_code)..."`. The
same message covers a red test suite, a compile error, a registry
outage, a refused migration and a failed health check. The operator must
log in and read hook output to learn which. The commit is missing too,
so two failures in a row cannot be told apart. The success message
(`:132`) does carry the full sha; there is no "started" message at all.
This is an operability gap.

**(3) Thin preflight.** `:28` checks `go npm oras jq`. The run also needs
`git` (`:56-58`), `psql` (the test gate, via `PSQL` in
`infra/uat/etc/uat-proto-test.env:12`) and `pg_dump` (install-api `:43`).
A missing one fails mid-run with a shell "command not found".

**(4) Host Go version.** `src/api-srv/go.mod:3` says `go 1.25.4`. A host
whose packaged Go is older and does not auto-fetch toolchains fails at
compile time with a version error, minutes in. B014 raises the `go` line
again.

**(5) Hook timeout.** `infra/uat/shrmpl-etc/hooks/build-proto_HOOKID.env:9`
sets `HOOK_TIMEOUT=600`. Waiting on the lock (fix 1) needs more.

#### What GRC changed

RG0013 / `da5eebd` ("The build lock waits instead of dropping a push;
HOOK_TIMEOUT 3600"), RG0043 / `b167658` (full sha in Slack), and the
toolchain check from `a57d6ad`. In GRC's `uat-grc-build`:

1. The lock waits up to 30 min (`:49-54`): try `flock -n`, print
   "waiting", then `flock -w 1800 200`, and Slack if it gives up. The hook
   timeout became 3600 (`build-grc_HOOKID.env:11-13`).
2. Preflight adds `git psql pg_dump` (`:56-58`) and checks `stg`,
   `libexec`, `backup` and `BUILD_STG` are writable (`:59-61`).
3. A `STEP` variable is set before each phase (`:64`, `:88`, `:96`,
   `:109`, `:115`, `:135`, `:138`). The trap (`:68-73`) sends "deploy
   FAILED at *$STEP* (exit N)", the commit and the host.
4. Slack at start (`:83`) with the full sha and commit subject, and again
   after deploy. `b167658` replaced `git rev-parse --short` with the full
   sha everywhere, because the registry tag (`sha-<commit>`), the revision
   file and the prd promote all carry the full one.
5. `GOTOOLCHAIN=auto` (`:45`) lets the host's older packaged Go fetch the
   required toolchain; `:88-94` then compares `go env GOVERSION` with the
   required version and fails by name.

No tests (shell). GRC also moved the tests after the deploy: that is B058.

#### Proposed fix for PROTO

All in `infra/uat/bin/uat-proto-build` unless named.

1. Replace `:24` with GRC's wait: `flock -n` then `flock -w 1800`, Slack
   on give-up. Set `HOOK_TIMEOUT=3600` in `build-proto_HOOKID.env:9`.
2. Add `git psql pg_dump` to the tool list at `:28`.
3. Add a `slack()` helper; set `STEP` before checkout, tests, go build, web
   build, registry push, install, publish. Put `$STEP` and `$GIT_SHA`
   (initialised to `?`) in `notify_failure`.
4. Slack "started" after `:61` with the full `GIT_SHA` and
   `git log -1 --format=%s | cut -c1-90`. Never `--short`.
5. Blind-copy trap: GRC's failure text says the service is "still on the
   previous build". That is false once install has swapped the binary
   (health failure) or for a publish failure. Say it only when `STEP` is
   before install.
6. Toolchain: export `GOTOOLCHAIN=auto`; check against
   `sed -n 's/^go //p' src/api-srv/go.mod` (PROTO has no `go.work`).
   Matters once B014 lifts `go` above the host's packaged Go.
7. Do not copy GRC's `go build` lines (`:99-104`): they use GRC's module
   path and three binaries. Keep PROTO's `./cmd/server`, `./cmd/migrate`.
8. Message prefix: GRC hard-codes its display name. Use `$APP_ENV` from
   B045 (`uat-proto`), so `bin/000-rebrand` has nothing to rewrite.
9. Keep PROTO's order (tests gate the push) unless B058 is pulled.

Tests: none automated. Rehearse: two pushes 60 s apart; expect the second
build to log "waiting" and then build the second commit.

#### Open question for Brandon

None.

### B048 — nginx: keepalive actually used; plain redirect; IPv6

- **Priority:** P2  **Area:** infra
- **GRC:** `infra/uat-unified001/etc/nginx_uat-grc.atticustechnology.com:13-25,43-55`   **PROTO:** `infra/uat/etc/nginx_uat-proto.example.com:9-27,40-51` (+ prd); `src/api-srv/internal/security/app_helper.go:308-311`
- **Ticket:** RG0013 / `3a3fc94`
- **What:** `keepalive 64` is unused without `Connection ""`. GRC adds it,
`X-Real-IP`, drops `Origin`, plain 301, `[::]:443`.
- **Port notes:** Keep PROTO's 8088 and 20M body.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

PROTO ships `infra/uat/etc/nginx_uat-proto.example.com` and
`infra/prd/etc/nginx_proto.example.com`; they differ only in hostname and
app root, so each point applies to both. nginx terminates TLS, serves the
bundle and proxies `/api/` to the Go API on loopback port 8088.

**(1) The upstream keepalive pool is never used.** `upstream
proto_api_server` asks for `keepalive 64;` (`:5-8`). `location /api/`
(`:40-51`) sets `proxy_http_version 1.1` but never clears the `Connection`
header. nginx's default for proxied requests is `Connection: close`, so Go
closes every connection after one response and the pool stays empty. Each
API call opens a new TCP connection to 8088 and leaves a socket in
TIME_WAIT. On loopback the handshake is cheap. Under a burst (a load test,
a list page firing many calls) TIME_WAIT sockets pile up and can exhaust
ephemeral ports. This is a quality gap: the config claims a behaviour it
does not have. It is not a security hole.

**(2) The port-80 block is Certbot's generated form.** `:12-14` wraps the
`return 301` in `if ($host = uat-proto.example.com)`, and `:19` falls
through to `return 404; # managed by Certbot`. A request whose `Host`
matches `server_name` always takes the `if`. The 404 is reached only when
this block is the port-80 default server and the request names some other
host. The behaviour is acceptable; the text is noise a reader must decode.
The renew script uses `certbot certonly`
(`infra/uat/bin/uat-proto-renew-web-certs:16`), which never edits this
file, so the "managed by Certbot" comments only record its origin.

**(3) No IPv6 on 443.** Port 80 listens on both stacks (`:16-17`), port 443
only on IPv4 (`:24`, `listen 443 ssl;`). On a host whose name has an AAAA
record, an IPv6 client gets the 301 over IPv6, then a refused connection on
`[::]:443`. Browsers usually fall back to IPv4 after a delay; `curl`,
monitors and some mobile networks do not. With no AAAA record nothing
happens. This is a latent correctness bug.

**(4) A redundant `Origin` line.** `:46` sets `Origin $http_origin`; nginx
forwards `Origin` unchanged anyway, so CORS at
`src/api-srv/internal/security/security_headers.go:76` is unaffected.

#### What GRC changed

GRC commit `3a3fc94` (RG0013, the post-sales infra mirrors) rewrote the site.
In GRC's UAT conf (`infra/<uat-host>/etc/nginx_<site>`; prd is the same):

1. Port 80 is a plain redirect (`:13-18`): `listen 80; listen [::]:80;`
   then `return 301 https://<site>$request_uri;`. No `if`, no 404.
2. Port 443 listens on both stacks (`:21-22`): `listen 443 ssl;` and
   `listen [::]:443 ssl;`.
3. `location /api/` (`:43-55`) adds `proxy_set_header Connection "";`
   (`:46`), so nginx reuses the pooled connections, and
   `proxy_set_header X-Real-IP $remote_addr;` (`:48`). It drops the
   `Origin` line. `X-Forwarded-For $proxy_add_x_forwarded_for` stays.

The ticket gives no separate reason; the lines arrived with the rewrite.
No test pins them. Nothing in GRC's Go code reads
`X-Real-IP` (grep finds only `X-Forwarded-For`, in
`src/internal/security/client_ip.go:26`), so that header is set and unused.
GRC also differs in port (8089) and body cap (`50m`, for PDF uploads); those
are GRC-specific and do not port.

#### Proposed fix for PROTO

1. In both `infra/uat/etc/nginx_uat-proto.example.com` and
   `infra/prd/etc/nginx_proto.example.com`, replace the port-80 block
   (`:11-20`) with `listen 80; listen [::]:80; server_name <site>;
   return 301 https://<site>$request_uri;`.
2. Add `listen [::]:443 ssl;` under `:24`.
3. In `location /api/`, add `proxy_set_header Connection "";` after
   `proxy_http_version 1.1` and delete the `Origin` line (`:46`).
4. Keep `X-Forwarded-For $proxy_add_x_forwarded_for` exactly as it is.
   B007's agreed fix makes `ClientIP` take the entry `TRUSTED_PROXY_COUNT`
   hops from the right; that relies on nginx appending `$remote_addr`.
   Do not switch to overwriting the header.
5. Leave out `X-Real-IP`: no PROTO code reads it, and a second client-IP
   header invites trusting the wrong one. Keep 8088 and `20M`.
6. Security headers are B022's change to the same file (move them off the
   server level into each web location). Land B022 and B048 in one edit to
   avoid two passes over the same lines.
7. B072 (presales nginx template) should start from the result.

Tests: none automated. On UAT: `nginx -t`; `curl -sI http://<site>/x`
gives a 301; `curl -6 -sI https://<site>/` connects when AAAA exists;
`ss -tn '( dport = :8088 )'` shows pooled connections after page loads.

#### Open question for Brandon

None.

### B049 — infra/README: runs-as table; layout with owners and modes

- **Priority:** P2  **Area:** infra, docs
- **GRC:** `infra/README.md:11,35-65`; `infra/sql/grc-host-db.sql`   **PROTO:** `infra/README.md:86-136,124-155`; `doc/340-db-host-setup.md`
- **Ticket:** RG0013 / `a57d6ad`, `3a3fc94`, `cc30f35`
- **What:** GRC's script table adds Runs-as; its layout gives owner and mode per
dir. PROTO's has partial owners and a wrong step 1 (B024).
- **Port notes:** Keep PROTO's comparison and Deriving sections; strip
blobs/worker. Docs P3.
- **Decision:** \[ \] pull / \[ \] adapt / \[X\] skip  (Brandon: bin/ stays hook-owned; fix in an app if it bites, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

A docs gap with two real bugs underneath. `infra/README.md` is the only
place an operator learns which Unix user runs each script and who owns
each directory under `/srv/<env>-proto/`. It answers neither fully.

1. **The script table has no "runs as" column.** `infra/README.md:86-93`
   lists Script / Sources / Required keys. The user is implied only for the
   two root cron jobs (`:91`, `:93`) and the start/stop tools (`:95-96`).
   The scripts themselves disagree: `infra/uat/bin/uat-proto-build:5-8`
   re-execs as `shrmpl-hook`; the four prd hook scripts refuse any other
   user (`infra/prd/bin/prd-proto-install-api:14-17`); the UAT
   `install-api`, `publish-web-client` and `run-tests` check nothing, so a
   by-hand run as root leaves root-owned files in `stg/` and `libexec/`
   that break the next hook build (B045 fixes the scripts).
2. **The layout gives owners for two of eight dirs.** `infra/README.md:126-136`:
   ```
   ├── bin/         copies of infra/<host>/bin/      (shrmpl-hook:shrmpl-hook)
   ├── etc/         filled env files, units, nginx   (root:<group> 0640)
   ├── libexec/     proto-api, proto-migrate, proto-api.revision
   ```
   `libexec/`, `stg/`, `backup/`, `crt/`, `log/`, `web-pub/` and the app
   root itself have no owner or mode. `0640` on `etc/` describes the files;
   as a directory mode it would block traversal.
3. **Setup step 1 contradicts the publish script (B024).** `:143-146` leaves
   the root root-owned and gives `web-pub/` to `www-data`. The publish
   script runs as `shrmpl-hook` and renames `web-pub/` inside the root
   (`infra/uat/bin/uat-proto-publish-web-client:46-48`), which that setup
   forbids, so the first post-sales web publish fails. `crt/` is also
   missing from the `mkdir` list at `:144`.
4. **The build's own hint is unsafe.** `uat-proto-build:39` tells the
   operator `chown -R shrmpl-hook:shrmpl-hook $SRV_ROOT`. That hands the
   hook `etc/`, whose `*-deploy.env` is sourced by the root cron job
   `*-renew-web-certs`: a hook compromise becomes root.
5. **Unit user drift.** `infra/prd/etc/prd-proto-api.service:11` says
   `User=prd-proto`; step 1 creates `proto`, which the UAT unit uses
   (`infra/uat/etc/uat-proto-api.service:11`). On prd the API won't start.

#### What GRC changed

GRC rewrote `infra/README.md` (RG0013, commits `a57d6ad`, `3a3fc94`;
`cc30f35` renamed the service account to a `svc-` prefix). Two additions:

- **"What runs where"** (`infra/README.md:35-47`): columns Script / Runs as
  / Sources / Does. Every hook-driven script says `shrmpl-hook`; certs,
  backup and start/stop say `root`. The scripts match: GRC's shared
  install and publish scripts re-exec as the hook user like the build.
- **Layout with owner and mode per dir** (`:52-65`), led by
  ```
  /srv/<env>-grc/   root:shrmpl-hook 1775
  ```
  `web-pub/` and `stg/` are hook-owned, `etc/` files `0640` grouped to the
  reader, `crt/` `root:www-data`, `log/` `www-data`.
- `:11` points at a single idempotent psql file for the three DB roles.
  That file and the runbooks that use it are B026, not this item.

No tests; docs only. One GRC flaw not to copy: `:56` makes `bin/` owned by
`shrmpl-hook`, yet `:45-46` run `renew-web-certs` and `backup-all` from
`bin/` as root. The hook user can rewrite a script root executes.

#### Proposed fix for PROTO

1. `infra/README.md:86-93`: add a **Runs as** column. Hook scripts
   `shrmpl-hook`; `*-renew-web-certs`, `prd-proto-backup-all`, `*-start`,
   `*-stop` `root`. Keep PROTO's Sources and Required keys columns.
2. `infra/README.md:124-136`: replace the tree with one owner and mode per
   line: root `root:shrmpl-hook 1775`; `bin/` `root:root 0755` (not GRC's
   hook-owned bin); `etc/` `root:root 0755`, files per their header
   (`root:proto 0640` api, `root:shrmpl-hook 0640` deploy/migrate/test,
   `root:root 0600` backup); `libexec/`, `stg/`, `backup/`, `web-pub/`
   `shrmpl-hook`, `web-pub/` world-readable; `crt/` as B021 sets it;
   `log/` `www-data`. One sentence on why 1775: sticky, so the hook can
   rename `web-pub/` but not root-owned `etc/` or `bin/`.
3. `infra/README.md:143-148` step 1: `chown root:shrmpl-hook` + `chmod
   1775` on the root, add `crt` to the `mkdir`, move `web-pub` to the hook.
   This is B024's doc half; land it with B024's script change.
4. `uat-proto-build:39`: hint names only `stg/` and `libexec/` (B047).
5. `prd-proto-api.service:11`: `User=proto`. One-line bug; fold here or B074.
6. Keep PROTO's "Which path am I on?" (`:10-29`) and "Deriving"
   (`:189-197`); GRC dropped both. Leave out blobs, worker, real hostnames.
   Spell names `proto` / `<env>-proto` so `bin/000-rebrand` renames them.
7. Deps: B024 (publish script), B021 (`crt/`), B026 (SQL file, runbooks),
   B045 (scripts that enforce the runs-as column). Tests: none (docs).

#### Open question for Brandon

Should `bin/` be root-owned on the host (hook can run but not edit the
scripts root's cron executes) instead of hook-owned as both repos say
today? Recommend root-owned: deploys already copy `bin/` as root, and it
closes a hook-to-root path that GRC still has open.

### B050 — Host hardening found by GRC's review, shared by PROTO

- **Priority:** P2  **Area:** infra, bin, docs, tickets
- **GRC:** `tickets/02-working/RG0043-code-review-stale-artifacts-and-insecure-practice/RG0043-code-review-stale-artifacts-and-insecure-practice.md:292-296,298-300,320-324,325-330,331-334,339-341`   **PROTO:** `doc/340-db-host-setup.md:65-66`; `src/api-srv/sql/roles.sql`; `infra/prd/bin/prd-proto-backup-all:18-37`; `infra/README.md:88,92`; `bin/100`, `215`, `216`, `510`, `515` (`set -e` only); `bin/220-run-web-app`; `bin/216-stop-local-stack:10-11`
- **Ticket:** RG0043 §4 (open in GRC too)
- **What:** Open in both: API role can `TRUNCATE`, shared cluster roles (4.18);
unencrypted backup of `etc/` and keys, root umask (4.19); one push-capable
registry credential (4.20); argv passwords (4.22); no `pipefail` (4.12);
`pkill -f vite`, cwd-relative `tmp/` (4.10/4.11).
- **Port notes:** One PS ticket: app NOLOGIN roles with S/I/U/D, `umask 077` +
encrypt, pull-only prod creds, `set -euo pipefail` sweep.
- **Decision:** \[ \] pull / \[ \] adapt / \[X\] skip  (Brandon: no second registry, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

GRC's review (RG0043 §4) found these in GRC; PROTO has the same code.

**(1) The API's DB role can TRUNCATE (security, defence in depth).**
`doc/340-db-host-setup.md:65-66` (copied at `doc/330:144-145`):
```sql
... IN SCHEMA public GRANT ALL    ON TABLES    TO db_writer;
```
`src/api-srv/sql/roles.sql:43` puts `proto_app` (the API) in `db_writer`.
`ALL` adds `TRUNCATE`, `REFERENCES`, `TRIGGER` to DML, so an injection or
a stolen API password empties any table in one statement. doc/340:12 says
"DML only", which the grant does not deliver. Also, `db_reader`/`db_writer`
are cluster-wide and shared by every app on the server (`roles.sql:4-5`):
another app's role in `db_writer` gets `ALL` on PROTO's tables, stopped
only by the `REVOKE CONNECT` that doc/340:34-37 marks "optional".

**(2) Prod backup ships secrets in clear (security).**
`infra/prd/bin/prd-proto-backup-all:28,30` tars all of `/srv/prd-proto`
and scps it to `REMOTE_SERVER`: `etc/` (`JWT_SECRET`,
`TOTP_ENCRYPTION_KEY`, DB and registry passwords) and `crt/` TLS keys
(`infra/README.md:129,133`). No encryption, no `umask`, so under cron's
usual 022 the tarball is world-readable on prod. `:34` adds every cluster
role's password hash. A reader of the copy can forge any JWT and decrypt
every TOTP seed. `:33-34` pipe `pg_dump | gzip` inside `su - postgres -c`,
a shell without `pipefail`; the script's own `set -euo pipefail` (`:2`)
does not reach it, so a failed dump exits 0.

**(3) One push-capable registry credential per host (security).**
`infra/uat/etc/uat-proto-deploy.env:4-6` and the prd copy; listed at
`infra/README.md:88,92`. UAT runs `npm ci` on every push
(`infra/uat/bin/uat-proto-build:89`), so any dependency's install script
can read the credential and retag what prod installs.

**(4) DB passwords on argv (security on a shared host).**
`infra/uat/bin/uat-proto-install-api:43` (prd `:50`) runs `pg_dump -Fc
"<DDL DATABASE_URL>"`, password visible in `ps`. `bin/510-reset-test-db:98`
puts the local superuser password in `-url`. Registry logins are fine.

**(5) No strict mode (DX gap).** `bin/100`, `215`, `216`, `510`, `515` are
`set -e` only; `bin/220-run-web-app` sets nothing. `bin/515-reseed-db:53`
pipes `psql -lqt | cut | grep`: a failed psql reads as "does not exist
yet" and execs bin/510. `bin/216-stop-local-stack:11` `pkill -f "vite"`
kills Playwright's Vite and any other project's.

**Corrected.** 4.11 (cwd-relative `tmp/`) is already fixed in PROTO:
`bin/215-run-api-srv:87-88` uses `$PROJECT_ROOT/tmp/api-srv` + `mkdir -p`.
4.10's "kills the test instance" does not apply: `bin/216:10` matches
`dist/$APP_SLUG-api`; bin/300 runs its own binary under `tmp/test-logs/`.

#### What GRC changed

Nothing substantive; all five are open in GRC. Its host SQL still grants
`ALL` (`infra/sql/<app>-host-db.sql:36-38`), its backup still tars `etc/`
and `crt/` unencrypted (`prd-*-backup-all:42,46`), its bin scripts are
still `set -e`. Partial: its test-web script kills only `vite --port
$WEB_PORT` (`bin/221-run-test-web-app:15`). No tests.

#### Proposed fix for PROTO

One PS ticket, after B020 (backup rewrite) and B015 (roles.sql password).
1. **Grants.** doc/340:64-66, doc/330:143-145 and doc/900:38-43: tables
   get `SELECT, INSERT, UPDATE, DELETE`; sequences `USAGE, SELECT`. Add a
   one-off `REVOKE TRUNCATE, REFERENCES, TRIGGER ON ALL TABLES IN SCHEMA
   public FROM db_writer` for existing hosts. Per-app NOLOGIN roles
   `proto_reader`/`proto_writer` in `roles.sql` replace the shared pair,
   so `bin/000-rebrand` renames them. `REVOKE CONNECT` becomes mandatory.
   Trap: GRC's role names and audit tables do not exist in PROTO.
2. **Backup** (on B020's version): `umask 077` first, dir `0700`, dump as
   root via `sudo -u postgres pg_dump` so `pipefail` covers it, then
   `age -r "$BACKUP_AGE_RECIPIENT"` each file before scp. New key in
   `etc/prd-proto-backup.env`; the private key never lives on either host.
3. **Registry.** Two credentials: UAT pushes the build repository only;
   prd reads it and writes a prd repository. `prd-proto-promote-uat:54-55`
   moves from `oras tag` to `oras cp`. See the open question.
4. **Argv.** Both `*-install-api`: export `PGPASSWORD` parsed from the env
   file, pass a password-free URL. bin/510:98 likewise (`pgx.Connect` in
   `cmd/migrate/main.go:84` reads `PGPASSWORD`). Trap: B015's port note
   `psql -v svc_pw=...` is argv again; feed it on stdin instead.
5. **Strict mode.** `set -euo pipefail` in bin/100, 215, 216, 510, 515,
   220; fix the unset reads it surfaces. bin/216:11 kills `vite --port
   5173` only.
6. **Tests.** pytest: as `proto_app`, `TRUNCATE` fails with insufficient
   privilege. Shell: `shellcheck bin/* infra/*/bin/*` clean.

#### Open question for Brandon

Promote retags in the registry from prod (`prd-proto-promote-uat:54-55`),
so prod cannot be pull-only with one repository: split into a UAT build
repository and a prod repository that promote copies into, or accept that
UAT's credential can move `:prd`? Recommendation: split; it is the only
way a UAT compromise cannot ship to prod.

### B051 — doc/105 and api-srv README: describe the real API

- **Priority:** P2  **Area:** docs
- **GRC:** `doc/105-api-srv-tech-spec.md:144-146,213-247` (MFA routes, flow), `:295-308,310-324,352-361` (fail-closed, rate keys, `ClientIP`, check tests), `:337-350` (exceptions table), `:584-588` (single process), `:432-440` (logging); `src/api-srv/README.md:121-132,134-148,150-160`   **PROTO:** `doc/105-api-srv-tech-spec.md:83-112,129-133,163-209,232-248,331-332,337,356`; `src/api-srv/README.md:83-105`; `app_helper.go:364,409,412`; `main.go:157-158,179`; `model/app_user_model.go:9,17`
- **Ticket:** RG0003 / `37978ce`; RG0011 / `06a1396`; RG0001 / `debb044`; RG0002 / `b33c9b9`; RG0004 / `d761d42`; RG0019 / `8fa36b6`; RG0005 / `5453748`
- **What:** PROTO omits the MFA routes, shows `/auth2` (:129-133), lists TOTP as
future (:356), claims stateless scaling (:337). GRC: real routes, exceptions
table, fail-closed contract, single process.
- **Port notes:** Write PROTO's truth until B002/B005/B033 land; check B059
first. README facts follow each code item.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

A docs gap with security weight: `doc/105-api-srv-tech-spec.md` is the API
spec a derived app and an auditor read, and it misstates routes, grants,
the MFA flow and the scaling model. Checked against
`src/api-srv/cmd/server/main.go:130-198` (33 routes) and the handlers.

1. **Endpoint table (`:85-112`).** 26 rows. Seven real routes are missing:
   `/changeOwnPassword`, `/userSelfCreate`, `/mfaSetup`, `/mfaVerify`,
   `/changeOtherUserPassword`, `/admin/resetUserMFA`, `/error-report`
   (`main.go:136,154,157-158,166,179,198`). Four rows name paths that do
   not exist (`:109-112`, `/userGroupList`, `/userGroup`); the routes are
   `/appUserGroupList`, `/appUserGroup` (`main.go:186-194`). No grant name
   in the table exists: `GET_EXAMPLE`, `ADD_USER`, `CHG_USER_GROUP` and the
   rest are not in `model/grants.go:36-53` (`READ_EXAMPLE`, `CREATE_USER`,
   `UPDATE_USER_GROUP`). `/grantList` is listed as `GET_USER`; the handler
   checks `GrantUserCreate` (`handler/user_handler.go:313`).
2. **MFA flow (`:129-133`)** describes a route that does not exist:
   ```
   2. **MFA Auth** (`/auth2`):
      - Verify TOTP (placeholder implementation)
   ```
   The login completes at `POST /mfaVerify` with `{tempToken, totpCode}`
   (`handler/user_mfa_handler.go:390-391`) against a real TOTP check
   (`:533`). Enrolment (`/mfaSetup`) is not described at all. A dead
   `Auth2RequestModel` remains at `model/security_model.go:27-31`.
3. **`:356`** lists "TOTP implementation for MFA" as a future high
   priority. It shipped.
4. **`:337`** "Scaling: Stateless design supports horizontal scaling". False.
   The same doc (`:127,156,168`) puts temp tokens, blocks and rate limits
   in the in-process KV, and lockout counters live there too. Two API
   processes would halve every lockout and reject a temp token minted by the
   other process. `:331-332` (Runtime Requirements) lists only PostgreSQL.
5. **Check contract (`:163-209`)** is right on order and `Skip*`, but omits:
   a KV error fails closed with 500 (`security/app_helper.go:367-373`); the
   `admin` subject skips rate limit and block (`:348,398`); the rate key is
   `RPM*<subject>` or `RPM*<ip>` in one namespace (`:364,409,412`);
   `ClientIP` trusts the first `X-Forwarded-For` entry (`:310-315`).
   The exceptions list (`:193-204`) leaves out `ChangeSecurityContextHandler`,
   which skips CSRF (`handler/auth_handler.go:530`) and decodes the body
   before any check (`:512-525`).
6. **`src/api-srv/README.md:83-105`** lists the five checks and nothing
   else: no exceptions, no fail-closed rule, no lockout, no MFA paragraph,
   no admin exemption. "admin blocks" (`:95`) names a feature no route
   sets (B034). The route list (`:49-81`) omits `GET /me` and `PUT /me`.
   True but unwritten: `PwHash` and `TOTPSeed` are `json:"-"`
   (`model/app_user_model.go:9,17`), so no response carries either.

#### What GRC changed

GRC rewrote the same sections across seven commits as each code fix
landed: `debb044` (RG0001, fail closed), `b33c9b9` (RG0002), `37978ce`
(RG0003, MFA), `d761d42` (RG0004), `5453748` (RG0005), `06a1396` (RG0011),
`8fa36b6` (RG0019). Results in GRC `doc/105-api-srv-tech-spec.md`:

- MFA routes in the table (`:144-146`) and a three-step flow (`:213-247`),
  "there is no `/auth2` route" (`:227`), seed and hash `json:"-"` (`:241-243`).
- "the fail-closed contract" (`:295-308`): an unchecked check answers 500;
  a KV error answers 500.
- Rate keys, `ClientIP` and the `admin` exemption stated (`:310-324`).
- "Exceptions, and why each one is allowed" (`:337-350`), a table with a
  reason per `Skip*`; "anything not here runs all five."
- Which tests prove the sequence (`:352-361`).
- "Single process" (`:586-588`): two processes "need a shared store first".
- README security layer (`:121-160`): fail closed, MFA, body caps.

GRC's text still says `ClientIP` takes the first XFF entry (`:316-318`);
B007's decision changes that, so do not copy the sentence.

#### Proposed fix for PROTO

1. Regenerate the table at `doc/105:85-112` from `main.go:130-198` and the
   `CheckAuthorization` argument in each handler, using `grants.go` short
   codes. Write PROTO's truth today: `/userSelfCreate` (B002),
   `/error-report` (B033) and the CSRF skip on `/changeSecurityContext`
   (B005) stay in, each marked with its B-number, until those items land.
2. Replace `:121-133` with GRC's enrolment and verify steps, trimmed to
   what PROTO does (no 409 for enrolled users until B001, no counter clear
   until B010, temp token deleted at `user_mfa_handler.go:624-630`).
3. Delete `:356`; replace `:337` with the single-process note in `:331-332`.
4. Add an exceptions table after `:204`, one row per `Skip*` site from the
   grep above, with the reason from the code comment; add the fail-closed
   paragraph and the admin exemption as stated facts.
5. `src/api-srv/README.md:83-105`: one paragraph each for fail closed,
   lockout, MFA; add `/me` to the route list.
6. Order: B059 changes the rate-key shape and B007 changes `ClientIP`; write
   those sentences after they land, or update them in the same PR. B040
   removes `.golangci.yml` (`doc/105:56,349,381`). B053's checker then
   guards the paths.
7. Blind-copy traps: drop GRC resources, the 24 §24.3 matrix, invite routes,
   `worker.log`, 4 MiB caps; keep `APP_NAME` as the JWT `iss`.
8. Tests: none for prose. B039 adds the check-sequence tests the new
   "tested by" paragraph will cite.

#### Open question for Brandon

Should the doc state the `admin` rate-limit and block exemption as
designed, or should B051 wait for a decision to remove it (doc/700
appendix flags it for a security look)? Recommendation: document it now as
it is, with a pointer to the appendix line, so the spec stops hiding it.

### B052 — Rules: customer data never in git; never assert the unverified

- **Priority:** P2  **Area:** docs
- **GRC:** `CLAUDE.md:39-42,99-100`; `README.md:38-42`; RG0043 md `:912-940,1084-1095`; `tickets/01-open/RG0044-fresh-repo-webhooks-deploy-keys.md:19-26`; `tickets/04-archive/engineering-log/260924_0740-TKT-NONE-round-11-on-main.md:9-21`; RG0006 md `:151-158`; RG0037 md `:133-134`   **PROTO:** `DERIVING.md:36-45`; `CLAUDE.md:25,29-54`
- **Ticket:** RG0043 1.7 / `03ad0f3`; RG0044; round-11 log
- **What:** (a) GRC committed a tenant's real import set and mock data, then
needed a history purge; source data lives outside the repo, fixtures
synthetic. (b) Review found the app counting attempts as successes and faking
a metric: never show what the code has not verified.
- **Port notes:** (a) as a DERIVING.md rule citing RG0044. Docs part 4 rated (a)
P1.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

This is a quality/process gap, not a code bug. Two rules that GRC learned
the hard way are absent from PROTO's guidance. A derived app inherits
PROTO's `CLAUDE.md` and `DERIVING.md` and nothing else.

**(a) Nothing says where customer data lives.** `DERIVING.md:36-44` is the
"Secrets and passwords" table. It covers `roles.sql`, seed logins, JWT and
TOTP keys, and `SERVERONLY` placeholders:

```
| `etc/uat-api-srv.env.example`, `infra/*/etc/*-api.env` | `SERVERONLY`
  placeholders | stay placeholders in git; real values only on the host |
```

It never mentions customer source data (exports, spreadsheets, import
sets) or mock data built from it. `CLAUDE.md:24` lists
`roles.sql` and `fixtures/` with no rule that fixtures are synthetic.
`.gitignore` has no data directory. The first derived app (GRC) put a
tenant's raw export, 42 employees' records and 226 risks, into
`doc/evidence-for-import/`, and its mock data (82 files) was real tenant
records, not fakes. CLAUDE.md described that layout as intended. Removing it
from the tree was easy. Removing it from history needs a new repository,
new webhooks, a new deploy key and a re-clone of every checkout (GRC
RG0044). The rule has to exist before the first commit of real data,
because after it the cost is a history purge.

**(b) Nothing forbids the app asserting what it has not verified.**
`CLAUDE.md:29-54` (Conventions) covers schema, tenant numbering, check
order, naming, line length, config and PR summaries. No line says a screen,
report or count must reflect only verified results. PROTO already carries
one instance: the load test's server monitor appends fixed numbers.

```go
// src/load-test/main.go:86-89
// This would use gopsutil or similar on real systems
metrics.CPUUsage = append(
    metrics.CPUUsage,
    50.0,
```

The `--monitor-server` report prints those as measured server CPU and
memory. B042 removes the code; this item adds the rule that would have
stopped it.

#### What GRC changed

(a) GRC commit `03ad0f3` (RG0043 1.7) moved the import set and mock data to
`../radley-grc-data/`, outside git, behind a config key `IMPORT_DATA_DIR`.
The rule now sits in `CLAUDE.md:40-42`:

```
The first-tenant source data (byte-for-byte exports; `prepared/`,
`generated/`, `mock-data/` derived) is customer data and lives
outside the repo in `IMPORT_DATA_DIR` (`etc/loc-api-srv.env`, RG0043).
```

`README.md:38-42` repeats it. The Go chain test switched to a synthetic
golden fixture so no test reads real data. History is open: RG0044
(`tickets/01-open/RG0044-fresh-repo-webhooks-deploy-keys.md:19-26`) plans a
fresh root commit because "personal data of 42 people and
customer-confidential registers must not stay reachable by sha, PR ref or
fork."

(b) `CLAUDE.md:99-100`: "Real data only: nothing fabricated reaches the
running app; assumptions go on the import report and the plan." The source
is the round-11 review log
(`tickets/04-archive/engineering-log/260924_0740-TKT-NONE-round-11-on-main.md:9-21`):
"most of the defects were the app asserting something untrue." Examples:
"Record completions" counted fifteen 403s as fifteen records; a hash minted
from the clock with no bytes stored; a screen saying a rule "runs on the
schedule" when nothing read it. RG0006 (`:151-158`) deleted the load test's
fake 50% CPU / 256 MB under the same rule. RG0037 (`:133-134`) applied it to
data: an import with no completion date may not invent one. No test pins
either rule; they are review rules.

#### Proposed fix for PROTO

1. `DERIVING.md` section 2: add a row or short paragraph. Customer source
   data, and anything derived from it, never enters git. It lives in a
   sibling directory named by a config key in the app's env file. Fixtures
   and mock data are synthetic. Cite the cost: a leak means a new repo
   (GRC RG0044). Do not copy GRC's `IMPORT_DATA_DIR` key; PROTO has no
   import path, so the key is the derived app's to add.
2. `CLAUDE.md` Conventions (`:44-54`): add two bullets.
   "Customer data never in git; fixtures are synthetic (DERIVING.md 2)."
   "Show only what the code verified: a count counts confirmed results, a
   metric is measured or absent, a missing value stays missing."
3. `sql/fixtures/local-seed.sql` is already synthetic (`admin`, `basic`).
   No `.gitignore` entry: the data directory sits outside the repo.
4. Dependency: B042 removes the load-test placeholder that (b) forbids.
   Land them together or note B042 in the commit.
5. Tests: none. A grep gate for "Placeholder" in Go is possible but would
   be noisy; not proposed.

#### Open question for Brandon

Should (a) be written as a hard rule in `CLAUDE.md` (every agent sees it)
or only in `DERIVING.md` (read once at derivation)? Recommendation: both,
one line in `CLAUDE.md` pointing at the `DERIVING.md` paragraph, because
the leak happens during later import work, long after derivation.

### B053 — `scripts/check-doc-paths.sh`: backticked paths must exist

- **Priority:** P2  **Area:** docs, bin
- **GRC:** `tickets/03-done/RG0011-doc-history-to-engineering-log/check-doc-paths.sh:1-4` (whole file); RG0010 md `:155-163`   **PROTO:** new `scripts/check-doc-paths.sh`; `CLAUDE.md:70-74`; `doc/902-project-commands.md`
- **Ticket:** RG0011
- **What:** Resolves every backticked path in docs and READMEs; exits 1 on a
miss. On PROTO: 35 of 207 missing in 16 docs (B090).
- **Port notes:** `cd` to repo root; allowlist placeholders
(`src/web-app/README.md:77-80`). Wire into bin/300 or the skill.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

A docs/DX gap. Nothing checks that a path a document names still exists.
PROTO's only doc-side check is `scripts/lint-tickets.py` (`CLAUDE.md:70-74`),
which reads ticket frontmatter, not paths. `bin/300-run-tests` runs Go,
vitest, Playwright and pytest; no doc step.

GRC's script, run read-only against PROTO on 2026-10-02 (copied to the
scratchpad, `cd` pointed at PROTO, `doc/700` excluded because it names GRC
paths on purpose): "checked 207 path tokens in 16 documents, 34 missing".
The doc/700 entry says 35; the run gives 34. "16 docs" is the number
scanned; the misses sit in 6 of them. Sorted by cause:

1. **Stale, real (4).** `doc/330-presales-to-postsales-migration.md:37` and
   `:131` name `sql/ddl-role-host-setup.sql`, deleted by PS0015 (`0981ddb`,
   replaced by `doc/340`). `doc/810-create-new-entity-type-notes.md:20`
   names `user-group-edit-page.tsx`; the file is
   `app-user-group-edit-page.tsx`. `doc/050-dev-env-init.md:35` backticks
   the deleted `sql/010-initialcreate.sql` in prose about history.
2. **Placeholders (7).** `src/web-app/README.md:77-80` (`widget-*` copy
   targets) and `doc/810:51,55,61` (`entity_model.go`,
   `entity_repository.go`, `entity_handler.go` meaning "your entity").
3. **Host or bundle paths (23).** `doc/320` (8), `doc/330` (13), `doc/340`
   (2): `deploy.sh`, `db.sql`, `000-roles.sql`, `etc/api-srv.env` inside
   the presales bundle, and `bin/uat-proto-build`, `etc/uat-proto-*.env`,
   which exist under `infra/uat/` but GRC's resolver does not look there.

So the real rot is small today (4), but a reader cannot tell a stale path
from a host path, and nothing stops the next rename from adding more.

#### What GRC changed

RG0011 wrote
`tickets/03-done/RG0011-doc-history-to-engineering-log/check-doc-paths.sh`.
The doc/700 ref ":1-4 (whole file)" is wrong: the file is 82 lines. It
greps backticked tokens from `doc/*.md` and the READMEs (`:33`), keeps
tokens that end in a repo extension or start at a top-level dir (`:67-71`),
skips URLs, absolute paths, `<`/`*`/`{`/`NNN` placeholders and gitignored
runtime dirs (`:63-66,72`), and resolves by prefix, against package roots,
or by basename anywhere (`:44-58`). A miss prints file:line; exit 1
(`:75,82`). Its rule (`:25-26`): "A document that names a file which was
deleted says so in prose, without backticks."

RG0010 (`:155-163`) used a one-line version of the same check for
`CLAUDE.md` and `doc/902`. RG0011 closed at "228 path tokens in 13
documents, 0 missing".

GRC never wired it into a gate: it lives in a done ticket folder, and no
`bin/`, `CLAUDE.md` or `doc/` file calls it. Run read-only on GRC today:
"checked 345 path tokens in 17 documents, 16 missing". It rotted from 0 to
16 in ten days. That is the strongest argument for wiring it in.

#### Proposed fix for PROTO

1. Add `scripts/check-doc-paths.sh` next to `lint-tickets.py`. `cd` with
   `"$(dirname "$0")/.."`, not GRC's `../../..` (`:31`).
2. `BASES` (`:34-35`): drop GRC's `src/migrate-srv`, `src/worker-srv`,
   `src/internal`, `doc/evidence-for-import`. Use `. src src/api-srv
   src/api-srv/internal src/web-app tests doc bin infra/uat infra/prd`.
   Adding `infra/*` resolves 5 of the host-path misses.
3. `DOCS` (`:33`): `doc/*.md` minus `doc/700` (it cites GRC), the four
   READMEs, `DERIVING.md`, `infra/README.md`. Leave `.claude/skills` out
   for now: a trial run gave 42 misses there, nearly all placeholders and
   bundle names.
4. Add an allowlist file (for example `scripts/doc-paths-allow.txt`,
   `file:token` per line) for the placeholders and bundle names, rather
   than widening the skip regex. Better still, rewrite placeholders as
   `<entity>_model.go`, which the existing `<` skip already handles.
5. Fix the 4 real misses in the same change so the script lands green.
6. Wire it in: a `docs` mode in `bin/300-run-tests` and in `all`, plus a
   line in `CLAUDE.md` next to the `lint-tickets.py` block and in
   `doc/902-project-commands.md`.
7. Run under `/bin/bash` 3.2 on macOS: the copy failed to parse once
   without an explicit UTF-8 locale (likely the `'…'` at `:64`). Replace
   the ellipsis with ASCII or set `LC_ALL` in the script.
8. Dependency: B051 rewrites `doc/105`; run the script after it.
   Tests: the script is the test; no unit test for it.

#### Open question for Brandon

Should a doc-path miss fail the `all` gate (blocking a merge) or only
print a warning? Recommendation: fail it, since GRC's unwired copy went
from 0 to 16 misses in ten days.

### B054 — Mirror drift tests: hand-coded contract, tested both sides

- **Priority:** P2  **Area:** docs, web-app, tests
- **GRC:** `tickets/03-done/RG0020-retire-as-built-generator-chain/RG0020-retire-as-built-generator-chain.md:118-120,223-224`; `CLAUDE.md:74-78`; `types/grc-contract.test.ts:1-25,150-246`; `utils/permissions.test.ts`   **PROTO:** `CLAUDE.md:29-54`; `src/web-app/README.md:85`; `constants/grants.ts` vs `src/api-srv/internal/model/grants.go`; `constants/api.ts`; new `tests/contract.test.ts`
- **Ticket:** RG0020 (Brandon 2026-09-22)
- **What:** Ruling: mirrors are hand-coded and drift-tested; Go writes a JSON
fixture, vitest replays it, with a floor and parser canary. PROTO mirrors
grants and routes unchecked.
- **Port notes:** Start with `AllKnownGrants` -&gt; `testdata/grants.json` vs
`grants.ts`. Web P3, docs P2.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

A correctness risk with a live instance, plus a missing rule. The web app
keeps two hand-written mirrors of Go values and nothing compares them.

**Grants.** `src/web-app/constants/grants.ts:2-22` lists grant ids; the Go
source is `src/api-srv/internal/model/grants.go:4-32` and
`AllKnownGrants` (`:35-54`). They have already drifted:

```ts
// src/web-app/constants/grants.ts:21
   RESET_MFA_TOKEN: 9020,
```

`grants.go` has no 9020 at all. The web app shows "Reset MFA" only to users
holding 9020 (`pages/app-user-edit-page.tsx:313`); the seed hands 9020 to
Admin (`sql/fixtures/local-seed.sql:52`). The API registers
`POST /admin/resetUserMFA` (`cmd/server/main.go:179`), whose handler checks
`model.GrantUserUpdate` (9003) at `handler/user_mfa_handler.go:684`. The
button and the server disagree about who may reset MFA, and no test
notices. B010 fixes the grant; this item makes the next drift fail a test.

**Routes.** `constants/api.ts:1-4` says it "Mirrors the routes in
src/api-srv/cmd/server/main.go by hand". Today the 20 paths in
`API_ENDPOINTS` all exist among the routes at `main.go:130-198`, so there is
no route drift yet. Nothing would catch a rename.

**Docs.** `src/web-app/README.md:84-86` tells the next entity author to add
api paths "(mirrors `src/api-srv/cmd/server/main.go`; no generator)" and
"the four `*_WIDGET` grant ids from `roles.sql`". `roles.sql` holds no
grants; they live in `model/grants.go`. `grants.ts:1` names no Go file,
although PS0013's surviving rule says each mirror "carries a header naming
the Go file it mirrors". `CLAUDE.md:29-54` has no line on mirrors.

#### What GRC changed

Ruling, `tickets/03-done/RG0020-retire-as-built-generator-chain/RG0020-retire-as-built-generator-chain.md:118-120`:
"the intent was always to decouple the layers by coding the shared values
in each ... So keep the contract, delete the chain, and test the drift."
GRC's `CLAUDE.md:74-78` makes it a convention: the contract is "coded in
each layer, not generated", with "drift tests in the gate".

The pattern is one fixture, both sides replay it:

1. Go `src/internal/model/contract_fixture_test.go:221`
   (`TestContractFixtureIsCurrent`) walks the model source and asserts the
   committed `testdata/contract.json` is what it would write. Only
   `bin/536-gen-contract-fixture` passes `-update-contract`, so tests never
   write the tree. A floor (`:39`, `enumFieldFloor = 100`; ticket
   `:224-226`) fails if the parser stops finding fields, so an empty
   contract cannot pass on both sides.
2. vitest `src/web-app/types/grc-contract.test.ts` loads the same JSON with
   Vite `?raw` (`:22`). A canary (`:151-157`) asserts the regex parser
   still finds >90 interfaces. Checks run both directions: every Go enum
   literal is in the TS union (`:166`), and no TS union lacks a Go check
   (`:199`). It also checks `API_ENDPOINTS` does not collide with resource
   routes (`:232`).
3. Same arrangement for permissions: `authz/testdata/matrix.json` replayed
   by `utils/permissions.test.ts` and `authz/matrix_test.go`.

Correction to the doc/700 entry: GRC has no drift test for grant ids. Its
`grants.ts` vs `grants.go` agreement was counted by hand in the RG0043
review ("15 grant ids = `grants.go`"). "Start with `AllKnownGrants`" is a
PROTO proposal, not a port.

#### Proposed fix for PROTO

1. Go: add `TestGrantsFixtureIsCurrent` in `internal/model` that marshals
   `AllKnownGrants` (sorted) and compares it with committed
   `internal/model/testdata/grants.json`; a `-update-grants` flag rewrites
   it, passed only by a new `bin/` helper (number per `bin/` scheme).
2. vitest: `src/web-app/tests/contract.test.ts` (the vitest include is
   `tests/**`, `vitest.config.ts:11`; the doc's `tests/contract.test.ts`
   path means this one). Import the JSON with `?raw`, compare with
   `grantConstants` both ways, and assert a floor (at least 18 entries).
3. Routes, second test in the same file: read `cmd/server/main.go` with
   `?raw`, extract `"METHOD /path"` patterns, assert every `API_ENDPOINTS`
   value is registered. Canary: the parse finds at least 25 routes.
4. Resolve 9020 first (B010), or the new test lands red.
5. Fix `src/web-app/README.md:86` (`roles.sql` to `model/grants.go`) and
   give `grants.ts:1` a header naming `grants.go`.
6. `CLAUDE.md` Conventions: one line, "Go and web mirrors are hand-coded
   and drift-tested; a fixture written by Go, replayed by vitest."
7. Do not copy GRC's `contract.json` walker, entity enums or matrix;
   PROTO has no generic entity layer (B068).

#### Open question for Brandon

Should the route test go beyond "web paths exist in Go" and also flag Go
routes the web never calls? Recommendation: no; `/health` and test-only
routes are legitimately web-unused, so one direction is enough.

### B055 — `npm update` is not a security fix

- **Priority:** P2  **Area:** docs, web-app
- **GRC:** `tickets/03-done/RG0007-web-hygiene-deps-errors-console/260922_0215-TKT-RG0007-web-hygiene-deps-errors-console.md:15-38`; `RG0007-web-hygiene-deps-errors-console.md:126-136,392-401`   **PROTO:** `src/web-app/README.md:112-118`
- **Ticket:** RG0007
- **What:** PROTO's README has every web ticket run `npm update`. In GRC that
bumped the linter (4 -&gt; 34 findings); seven of nine advisories were in unused
deps.
- **Port notes:** Order: depcheck, `npm audit --omit=dev`, `npm update <advised>`, toolchain as its own ticket. Same for `go get -u`.
- **Decision:** \[ \] pull / \[X\] adapt / \[ \] skip  (per recommendation, minus the standing ticket (B091), 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

A process/DX gap in the written rule, not a code bug. `src/web-app/README.md`
tells every web ticket to bump every package:

```
// src/web-app/README.md:114-118
Caret ranges. Any ticket that touches the web app runs `npm update
--legacy-peer-deps` (in-range bumps; plain `npm update` trips an arborist
bug in npm 10 on this peer set) and `npx depcheck` (no unused or undeclared
packages). Major versions
(Mantine, ESLint, Vitest, Vite plugins) are their own ticket, one at a time.
```

The rule came from PS0013 ("Rules that survived",
`tickets/04-archive/PS0013-web-hygiene-deps-schemas-constants.md:71-72`).
It has three consequences.

1. An unrelated web ticket drags in toolchain changes. An in-range bump of
   `typescript-eslint` or `eslint-plugin-react-hooks` ships new lint rules.
   The ticket's diff then carries lint fallout and lockfile churn its
   reviewer did not ask for, and the "lint clean" gate can go red for
   reasons outside the change.
2. It treats "bump everything" as the security step. It never says to run
   `npm audit`, and it never separates production from dev advisories.
3. It runs `depcheck` after the bump, so the cheapest fix (deleting an
   unused package) is not the first one tried.

The same gap exists for Go: nothing in `CLAUDE.md`, `README.md`, `doc/` or
`bin/` mentions `go get -u`, `govulncheck` or how a Go advisory is handled.

Current state, checked 2026-10-02: PROTO's lock already resolves
`react-router-dom` 7.18.4, `typescript-eslint` 8.70.1 and
`eslint-plugin-react-hooks` 7.1.1, and `npm audit --omit=dev` reports 0. So
there is no open advisory today; the problem is the rule's next run.

#### What GRC changed

GRC's web README never carried PROTO's cadence section (no match for
"Dependency cadence" in its history). The lesson lives only in ticket
RG0007, which followed the same "npm update" ruling.

The engineering log
(`tickets/03-done/RG0007-web-hygiene-deps-errors-console/260922_0215-TKT-RG0007-web-hygiene-deps-errors-console.md:17-38`)
records what the bare update did:

```
moved **every** package to the top of its range. Nine advisories
went to zero and `npm run lint` went from 4 non-`max-len` findings to **34**,
```

22 of those were auto-fixable, which would have broken the "eslint would
rewrite nothing" gate built a day earlier (RG0006). GRC reverted the lock
and re-ran the update for one package, `react-router-dom` 7.9.6 to 7.18.4,
the only one the advisories named. And `:36-38`: "seven of the nine
production advisories were in packages nothing imported. Removing dead
dependencies was the security fix." The ticket body (`:126-134`) states the
narrowed update; `:392-401` files the toolchain bump (19 dev advisories,
~30 new findings) as its own ticket. No test pins this; it is procedure.

#### Proposed fix for PROTO

1. Rewrite `src/web-app/README.md:112-118` ("Dependency cadence") as an
   ordered procedure for a ticket that has a security reason to touch deps:
   (a) `npx depcheck`; remove unused packages first.
   (b) `npm audit --omit=dev`; production advisories are the target.
   (c) `npm update --legacy-peer-deps <advised package>` only; keep the
   arborist note.
   (d) Dev-only advisories and toolchain bumps (eslint, typescript-eslint,
   vite, vitest) are their own ticket with the lint fallout in the same
   change.
2. Drop "any ticket that touches the web app runs `npm update`". Keep
   `npx depcheck` on every web ticket; it is cheap and never bumps.
3. Add the Go equivalent where Go conventions live (`src/api-srv/README.md`
   or `doc/902`): `govulncheck ./...` first; `go get <module>@<fixed>`
   for the named module; no blanket `go get -u ./...` inside a feature
   ticket. B014 (Go dependency bumps) is the first ticket to follow it.
4. Related: B041 (eslint as a clean-tree check) is the gate a blanket bump
   breaks; B069 (depcheck config, zod pin) touches the same section. Land
   B055's text with or before them.
5. Tests: none. Optional `bin/` helper that runs depcheck + `npm audit
   --omit=dev` and prints both is B069/B070 territory, not this item.

#### Open question for Brandon

Should PROTO keep a periodic toolchain-bump ticket (for example, monthly)
so dev advisories do not pile up once feature tickets stop bumping?
Recommendation: yes, a standing ticket raised by `ticket-triage` when
`npm audit` (dev included) is non-zero, since nothing else will trigger it.

### B056 — Review checklist skill (RG0043 method + "checked and clean")

- **Priority:** P2  **Area:** skills
- **GRC:** RG0043 md `:25-31,431-488,490-507,509-525`   **PROTO:** new `.claude/skills/stale-and-insecure-review/SKILL.md`
- **Ticket:** RG0043
- **What:** Four read-only passes plus a history secret scan, severities, a
checklist of template invariants (check order, tenant predicates, no
`innerHTML`, raw `fetch` only in `api.ts`, `pipefail`), and a required "Not
covered" section.
- **Port notes:** Drop GRC matrix/blob/Entra lines. Brandon's call: skill or
doc?
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

A quality/process gap. PROTO has no procedure for a whole-tree review. Its
skills are `compact-tickets`, `handoff`, `presales-deploy`, `ticket-triage`
and `ticket-workflow` (`.claude/skills/`); none reviews code. The global
`security-considerations` skill is per-PR: it asks questions about one
change type. Nothing asks "does the whole tree still honour the template's
invariants?" and nothing records what a past review covered.

The invariants are stated as rules in `CLAUDE.md:35-53` (schema shape,
tenant column, explicit check order, naming, no env lookups) but are
checked by nobody between PRs. A spot check on 2026-10-02 shows the cost:

- Holds: 0 `innerHTML`/`dangerouslySetInnerHTML`/`eval(` in
  `src/web-app`; raw `fetch(` only in `utils/api.ts` and the e2e fixture
  `tests/e2e/fixtures/api-client.ts`; 0 `os.Getenv` in Go.
- Drifted: six bash scripts lack `pipefail`. `bin/100-build-api-server:6`,
  `bin/215-run-api-srv:3`, `bin/216-stop-local-stack:3`,
  `bin/510-reset-test-db:31` and `bin/515-reseed-db:25` use `set -e` only;
  `bin/220-run-web-app` sets nothing. A failing command on the left of a
  pipe goes unnoticed.

Without a written method, a second review cannot tell which areas the
first one cleared, so it either re-reads everything or misses the gaps.

#### What GRC changed

GRC wrote the method into ticket RG0043, not into a skill (GRC's
`.claude/skills/` has the same four ticket skills as PROTO, no review skill).
`tickets/02-working/RG0043-code-review-stale-artifacts-and-insecure-practice/RG0043-code-review-stale-artifacts-and-insecure-practice.md`:

- `:25-31` Method: "read-only review ... four parallel passes: Go backend,
  web app, infra / config / scripts / tests, docs / tasks / tickets / data,
  plus a git-history secret scan across all refs. Every bullet was
  confirmed by reading the cited file." Severity: H exploitable or privacy
  or broken on a host; M latent bug or misleading drift; L hygiene.
- `:431-488` "Checked and clean (so the next reviewer knows what was
  covered)": counted assertions, for example "all 28 `NewAppHelper` sites
  ... call JWT → rate-limit+block → CSRF → authorization →
  `CheckAllChecksComplete`; every `Skip*` carries a reason", tenant
  predicate in every repository write, "0 `dangerouslySetInnerHTML`
  ... raw `fetch` only in `utils/api.ts`", "11 bash scripts `set -euo
  pipefail`", "15 grant ids = `grants.go`", secrets scan over 486 commits.
- `:490-507` "Not covered": live host state, named files not read, page
  bodies read only by sweep.
- `:509-525` a suggested split of findings into batches for triage.

The review found the items behind many B-numbers in doc/700 (B007, B009,
B015, B052 and others). No test pins the method; it is procedure.

#### Proposed fix for PROTO

1. Add `.claude/skills/stale-and-insecure-review/SKILL.md` (or a doc, see
   the question). Contents: read-only; one ticket holds the findings; four
   passes (Go `src/api-srv`, web `src/web-app`, infra/bin/tests/config,
   docs/tickets) run as parallel subagents; a `git log -p --all` secret scan
   over `etc/ infra/ bin/ tests/ src/ scripts/ .claude`; H/M/L definitions
   as above; every finding cites `path:line` read by the reviewer.
2. A checklist of PROTO invariants, each phrased as a countable assertion:
   check order and `Skip*` reasons at every `NewAppHelper`;
   `security_context_id` in every repository read/write; tenant 2 reserved;
   no FK/trigger/`DEFAULT` in migrations; no `os.Getenv`; 0 `innerHTML`
   family; raw `fetch` only in `utils/api.ts`; `constants/grants.ts` equals
   `model/grants.go` and `constants/api.ts` paths are registered (B054
   makes this a test); `set -euo pipefail` in every bash script; every
   committed `infra/**/*.env` value is `SERVERONLY`; backticked doc paths
   exist (B053 makes this a script).
3. Required output sections: findings by severity, "Checked and clean"
   with counts, "Not covered". A review without "Not covered" is incomplete.
4. Leave out GRC-only lines: `authz/matrix.go`/`permissions.ts` replay,
   blob store, Entra client assertion, pwsh runner, 85-entity counts, goose.
5. Derived names: the skill must not say "proto"; use `etc/app.env` slugs.
6. Dependencies: B053 and B054 turn two checklist lines into automated
   checks; reference them once landed. First run would flag the six
   `pipefail` gaps above.
7. Tests: none (procedure). `scripts/lint-tickets.py` already validates the
   findings ticket.

#### Open question for Brandon

Skill or doc? Recommendation: a skill, because it is a procedure an agent
runs on request ("review the tree"), and skills are where PROTO's other
procedures live; keep the invariant list in one place inside it.

### B057 — Branch model uat/main promote-on-push: document, do not adopt

- **Priority:** P2  **Area:** docs, infra, skills
- **GRC:** `CLAUDE.md:139-148`; `infra/README.md:9-10,39,44`; `infra/uat-unified001/etc/uat-grc-deploy.env:14-15`; `infra/uat-unified001/shrmpl-etc/hooks/build-grc_HOOKID.env:7-9`; `infra/prd-unified002/shrmpl-etc/hooks/prd-promote-grc_*.env:8-13`; `infra/prd-unified002/bin/prd-grc-promote-uat-and-deploy:1-10`; `.claude/skills/ticket-workflow/SKILL.md:3,10,21-35,136-147`; RG0043 md `:71-73,538-547,596-597,669-677,758-762,771-782,789-790`; `doc/331-prd-postsales-cutover.md:439-444`   **PROTO:** `CLAUDE.md:76-84`; `infra/README.md:98-123`; `infra/uat/etc/uat-proto-deploy.env:13-14`; `infra/uat/shrmpl-etc/hooks/build-proto_HOOKID.env:7`; `infra/prd/shrmpl-etc/hooks/prd-promote-proto_HOOKID.env:7`; `DERIVING.md`; `.claude/skills/ticket-workflow/SKILL.md:3,10,21-31,107-121`
- **Ticket:** RG0013 / `5644d5a`, `f941ae1`; RG0043 / `5bd0fd4`
- **What:** With both hooks on `main`, each push promoted the previous UAT
build. GRC: `uat` builds UAT, `main` promotes digests, merge commits, two
approvals, full shas, rollback by `sha-<commit>`. Hazard: metadata on `main`
is a prod release.
- **Port notes:** Keep PROTO `main`-only; document as the post-sales model in
DERIVING.md + infra/README. Ops P3, docs P2, skills SKIP.
- **Decision:** \[ \] pull / \[ \] adapt / \[X\] skip  (superseded by B091, ticket system removed, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Not a live bug: PROTO's post-sales pipeline is a placeholder
(`infra/README.md:3-8`, "not used until post-sales"). It is a latent
release-safety gap in the template that a derived app walks into when it
goes post-sales. PROTO has one branch, `main` (`CLAUDE.md:78-81`), and the
UAT build hook listens on it:

```
# infra/uat/etc/uat-proto-deploy.env:12-13
# Must match HOOK_ALLOWED_BRANCHES in shrmpl-etc/.
BRANCH_TO_CHECKOUT=main
# infra/uat/shrmpl-etc/hooks/build-proto_HOOKID.env:7
HOOK_ALLOWED_BRANCHES=main
```

The prod hook's branch is left blank:
`infra/prd/shrmpl-etc/hooks/prd-promote-proto_HOOKID.env:7` is
`HOOK_ALLOWED_BRANCHES=SERVERONLY`. The bring-up steps
(`infra/README.md:172-173`) say only "`HOOK_ALLOWED_BRANCHES` must equal
`BRANCH_TO_CHECKOUT`". `main` is the only branch PROTO has, so an operator
sets the prod hook to `main` too. Then one push fires both hooks at once:

1. UAT's hook starts `uat-proto-build`: tests
   (`infra/uat/bin/uat-proto-build:64`), build, then moves `:uat` to the
   new commit (`:113,121`), minutes later.
2. Prod's hook starts `prd-proto-promote-uat-and-deploy` at the same time.
   With no argument, `prd-proto-promote-uat:15-16` promotes
   `PROMOTE_FROM_TAG` (`uat`, `infra/prd/etc/prd-proto-deploy.env:11`),
   which still points at the previous commit.

Every push to `main` releases the previous UAT build to prod, not the commit
just pushed, and no human approves the step. That includes ticket-metadata
commits: the ticket-workflow skill claims numbers on `origin/main`
(`SKILL.md:3`) and says `main` "is also what the UAT server deploys from"
(`:23-25`). Under a both-on-main setup, filing a ticket is a prod release.
Nothing in `DERIVING.md` (Hosts, `:46-54`) or `infra/README.md:98-117`
(Promotion flow) names the prod hook's branch.

#### What GRC changed

GRC hit exactly this. `doc/331-prd-postsales-cutover.md:442-443`: "First
both hooks were on `main`, which promoted the previous build on every
push". The model flipped three times in a day: `5644d5a` (prod hook on
`main`), `f941ae1` (prod on `uat`, `main` builds UAT), `5bd0fd4` (RG0043,
final): `uat` builds UAT, `main` releases prod.

- `CLAUDE.md:141-146`: "`uat` is the integration branch and what UAT
  deploys from; `main` is ... what prod deploys from: a push to `main`
  promotes UAT's build." PRs target `uat`; only Brandon moves `main`.
- Hooks: `infra/uat-unified001/etc/uat-grc-deploy.env:14-15` and
  `.../hooks/build-grc_HOOKID.env:7-9` on `uat`;
  `infra/prd-unified002/shrmpl-etc/hooks/prd-promote-grc_*.env:8-10` on
  `main`. `infra/README.md:9-10,44` documents both.
- Release (RG0043 `:538-547,669-677`): two approvals, A1 merges a PR into
  `uat`, A2 merges `uat` into `main` with a merge commit, never squash or
  rebase, so `main` never holds commits `uat` lacks. Freeze `uat` while
  releasing (`:596-597`) so `:uat` is still the build that was signed off.
- Full shas in Slack (`:758-762`); rollback promotes an explicit
  `sha-<commit>` (`:771-782`), which PROTO's promote already supports.
- Open in GRC (`:71-73,789-790`): the skill still puts ticket metadata on
  `main` (GRC `ticket-workflow/SKILL.md:27-29`, "batch those commits and
  ask before pushing"), so filing a ticket is still a prod release. No
  test pins any of this; it is host config and procedure.

#### Proposed fix for PROTO

Document only. PROTO keeps `main`-only while presales.

1. `infra/README.md` Promotion flow (`:98-117`): add the hazard in two
   lines and the post-sales model: `uat` branch builds UAT, `main`
   promotes `:uat`, merge commit `uat` to `main`, freeze `uat` during a
   release, rollback with `prd-proto-promote-uat-and-deploy <full sha>`.
2. `infra/README.md:172-173`: the build hook equals `BRANCH_TO_CHECKOUT`;
   the prod hook must name a different branch. Never both on one branch.
3. Repo copies: comment the prod hook file's line 7 with the same rule.
   Leave the value `SERVERONLY`.
4. `DERIVING.md` section 3 (Post-sales bullet): one line pointing at (1),
   and that `CLAUDE.md` Branches and the ticket-workflow skill change at
   cutover. Do not edit the skill now (doc/700 rates skills SKIP).
5. Blind-copy traps: GRC's host names (`uat-unified001`, `prd-unified002`),
   `Radley-House` repo URL, hook id and `atticustechnology.com` stay out;
   names derive from `etc/app.env`. GRC's deploy-then-test build (B058)
   means its `:uat` can be a test-failed build; PROTO tags `:uat` only after
   tests (`uat-proto-build:64` before `:113`), keep that.
6. Related: B047 (full sha in Slack), B058 (UAT order). Tests: none.

#### Open question for Brandon

In the post-sales model, where do ticket-metadata commits go, given a push
to `main` is a prod release? Recommendation: to `uat`, so `main` only
moves on a deliberate release; GRC left this open too.

# P3: nice to have

### B058 — UAT order: deploy, then test the deployed commit

- **Priority:** P3  **Area:** infra, docs
- **GRC:** `infra/uat-unified001/bin/uat-grc-build:1-13,143-160`; `infra/uat-unified001/bin/uat-grc-run-tests:1-85` (result file :51-64, `_test_db` guard :41-42); `infra/README.md:39-40`; RG0013 md `:71-77`; RG0043 md `:941-952`   **PROTO:** `infra/uat/bin/uat-proto-build:62-66`; `infra/uat/bin/uat-proto-run-tests:1-14`; `infra/README.md:98-111` (:104)
- **Ticket:** RG0013 / `da5eebd` (Brandon 2026-10-01); RG0043 / `518cec1`
- **What:** PROTO: red tests block the deploy. GRC: build, push, install,
publish, then test; failure Slacks "don't promote" and exits 0; runner from
the checkout writes `stg/run-tests.result`.
- **Port notes:** Brandon's call (ops P3, docs P2). Generic either way: result
file, `_test_db` drop guard.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Nothing is broken; this is an ordering policy. A push to the UAT branch runs
`infra/uat/bin/uat-proto-build`, which runs the whole test gate first:

```bash
# infra/uat/bin/uat-proto-build:63-64
# Tests gate the registry push: nothing below runs on a red suite.
"$SRV_BIN/uat-proto-run-tests"
```

`uat-proto-run-tests:13-14` execs the checkout's `bin/300-run-tests` in mode
`all` (Go, vitest, Playwright, pytest). A red test aborts the run and UAT
keeps the old build (`infra/README.md:104`, "(gate)"). One flaky Playwright
test blocks every deploy, and nobody can look at a change on UAT until the
suite is green. The lock is `flock -n` (`:24`), so a push landing during a
long run is dropped, not queued. An ops/DX trade-off, not a bug.

Two port notes are already true in PROTO: the runner comes from the checkout
(`$BUILD_STG/bin/300-run-tests`), and the drop guard exists
(`bin/510-reset-test-db:44-45` refuses names not ending `_test_db`/`_e2e_db`).

#### What GRC changed

RG0013 (`da5eebd`, Brandon 2026-10-01) reversed the order
(`infra/uat-unified001/bin/uat-grc-build:1-14`): build, push, tag `:uat`,
install, publish, Slack "deployed", then test. A deploy failure exits 1 and
names its `STEP` (`:63-73`). A test failure is reported and exits 0 (`:149`,
`:157` "the deploy stands; don't promote", `:160`). `uat-grc-run-tests:52-64`
writes a one-line `stg/run-tests.result` (counts, failing step, first five
failing tests) for Slack. RG0043 (`518cec1`) made the build call the
checkout's runner. The lock waits 30 min (`:51-53`). GRC tests less on UAT:
Go and vitest only (`run-tests:23-24`). GRC's own review (RG0043 item 8)
found `prd-grc-promote-uat` never reads the result, so a red build can reach
prd; Brandon marked it "by design" (2026-10-02). No tests pin this.

#### Proposed fix for PROTO

1. `infra/uat/bin/uat-proto-build`: move `:63-65` after install/publish
   (`:127-128`); a `STEP`-naming deploy trap until publish, then `set +e`.
2. Three Slack messages: deployed, tests passed, tests FAILED "don't
   promote". Drop the hard-coded "playwright green" at `:132`.
3. `bin/300-run-tests`: write `SUMMARY` to a caller-given result file.
4. `flock -n` at `:24` becomes `flock -w 1800`.
5. Keep PROTO's `all` mode; do not narrow to GRC's Go plus vitest.
6. Update `infra/README.md:104`. Verify with one deliberately red UAT run.

#### Open question for Brandon

Should `prd-proto-promote-uat` refuse when `run-tests.result` says FAILED?
Recommend yes: deploy-then-test is safe only if promote cannot ship red.

### B059 — API: CSP trimmed; `ip:` rate bucket; CSRF log without token

- **Priority:** P3  **Area:** go-api
- **GRC:** `src/internal/security/headers.go:11-31`; `src/internal/security/app_helper.go:320-329,368,424,543-550`   **PROTO:** `src/api-srv/internal/security/security_headers.go:12-35`; `security_headers_test.go:39-48`; `security/app_helper.go:360,412,521-527`
- **Ticket:** RG0043 item 32 / `720f5e1`; RG0002 / `be58262`
- **What:** PROTO's API CSP allows scripts, inline styles, images and exempts
`/assets/` for a JSON-only server; GRC: `default-src 'none'`, XFO DENY always.
Anonymous buckets get `ip:`; CSRF mismatch logs no token.
- **Port notes:** Confirm no bundle served; update the header test.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Three small hardening gaps, all under `src/api-srv/internal/security/`. None
is an exploitable hole today.

**(1) CSP written for an HTML page.** Every response gets a page-shaped
policy (`security_headers.go:14-17`: `script-src 'self'`, `style-src 'self'
'unsafe-inline'`, `img-src 'self' data: https:`), and `:33-35` skips
`X-Frame-Options: DENY` for `/assets/`. The API serves no bundle: no
`FileServer` or `/assets` route exists in `src/api-srv`; nginx serves
`web-pub/` (`infra/uat/etc/nginx_uat-proto.example.com:30,59`). The
allowances are dead weight, pinned by `security_headers_test.go:39-48`, and
a derived app copying the policy to nginx inherits `'unsafe-inline'`.

**(2) Anonymous rate buckets share the user key space.** An anonymous key is
`RPM*<ip>*<minute>` (`app_helper.go:360,364` and `:412`); a signed-in key is
`RPM*<username>*<minute>` (`:357,409`; `Subject` is the username,
`auth/jwt.go:45`). A user named `10.0.0.5` would share a bucket with
anonymous traffic from that address. Unlikely, but the fix is one prefix.

**(3) CSRF log keeps the received token.** `app_helper.go:523-526` logs
`"CSRF token mismatch - received: "+csrfToken`. The expected value is
already omitted (`:519-522`). The received value from a stale tab is the
last three characters of an earlier session id, so it is still a credential
fragment in the log.

#### What GRC changed

RG0002 (`be58262`) added `anonRateIdentifier`, returning `"ip:"+ClientIP(r)`
(`src/internal/security/app_helper.go:328-329`), used at `:368` and `:424`,
and cut the CSRF log to `"CSRF token mismatch"` (`:546-549`). GRC started
from one shared `anon` bucket; PROTO is already per-address, so only the
prefix is new here. RG0043 item 32 (`720f5e1`) set
`default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action
'none'` and unconditional XFO DENY (`headers.go:19-21,31`). Test:
`TestAnonRateIdentifierIsPerAddress` (`client_ip_test.go:70-82`). No CSP
test.

#### Proposed fix for PROTO

1. `security_headers.go:13-24,32-35`: GRC's four-directive CSP; always set
   XFO DENY; drop the `strings` import. Leave out GRC's document-route
   relaxation (RG0026); PROTO has no file routes.
2. `security_headers_test.go`: delete `:39-48`; assert the exact CSP and
   XFO DENY on an `/assets/` path.
3. `app_helper.go`: add `anonRateIdentifier`, use it at `:360` and `:412`.
   Land with or after B007's `ClientIP` / `TRUSTED_PROXY_COUNT` change.
4. `app_helper.go:519-526`: log `"CSRF token mismatch"` only.
5. Go test: port `TestAnonRateIdentifierIsPerAddress` verbatim.

#### Open question for Brandon

None.

### B060 — Go hygiene: one `model.Now()`; RG0015 small fixes

- **Priority:** P3  **Area:** go-api
- **GRC:** `src/internal/model/time.go:1-40`; `model/time_test.go`; `src/internal/auth/totp.go:60-73`; `src/internal/config/config.go:236-252`; `src/api-srv/cmd/server/main.go:246-254`; `src/internal/handler/app_user_group_handler.go:29,52-56,88,176,267`   **PROTO:** `src/api-srv/internal/handler/common.go:158-162`; `repository/user_repository.go:18-21,508,567`; `auth/totp.go:55-60`; `config/config.go:135-205`; `cmd/server/main.go:201-204`; `handler/app_user_group_handler.go:55-56`
- **Ticket:** RG0015 / `1fc6e86`, `6ee3abc`; RG0043 items 31, 35; PS0011
- **What:** Three timestamp spellings (UnixNano/1e9 drifts) become
`model.Now()`; TOTP regexp compiled once; threshold checked once;
`net.JoinHostPort`; `WriteError`; stale comments.
- **Port notes:** One mechanical hygiene ticket.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Six small quality gaps under `src/api-srv/`. One is a real, rare bug.

**(1) Three timestamp spellings, one racy.** `create_ts`/`update_ts` are
epoch seconds in `NUMERIC(17,7)` (`sql/migrations/0001_initial_schema.sql:36`);
`update_ts` is the optimistic lock. `handler/common.go:161` uses
`float64(time.Now().UnixNano()) / 1e9`, as do inline copies at
`repository/user_repository.go:511,570`. A second function differs:

```go
// internal/repository/user_repository.go:20
return float64(time.Now().Unix()) + float64(time.Now().Nanosecond())/1e9
```

It reads the clock twice. If the second rolls over between the two reads,
the result is up to one second in the past (`…00.999` read as `…00` +
`0.001`). The `UnixNano()/1e9`
form loses up to ~128 ns at today's magnitudes. A rare correctness bug.
**(2)** `auth/totp.go:57` compiles a regexp per MFA setup. **(3)**
`config/config.go:146` and `:206` both require the window duration; the
threshold is parsed at `:135` and range-checked apart at `:196`. **(4)**
`cmd/server/main.go:202` uses `fmt.Sprintf("%s:%d", ...)`, so an IPv6 bind
`::1` yields the invalid `::1:8080`. **(5)**
`handler/app_user_group_handler.go:56` uses bare `http.Error`, not
`app.WriteError`. **(6)** Comments at
`:31,90,174,283` name `/userGroup*`; routes are `/appUserGroup*`
(`cmd/server/main.go:187`). Open ticket PS0011 already lists (1) and (2).

#### What GRC changed

RG0015 (`1fc6e86`, `6ee3abc`) added `model.Now()` over
`EpochSeconds(t) = float64(t.Unix()) + float64(t.Nanosecond())/1e9`, one
`time.Time` read (`src/internal/model/time.go:26-34`); the comment at
`:5-23` gives the reasoning. Tests: `TestEpochSecondsIsExactForAWholeSecond`,
`TestEpochSecondsKeepsSubSecondDetail`, `TestNowMovesForward`
(`model/time_test.go:13,22,33`). Also a package-level `totpLabelDisallowed`
(`auth/totp.go:65`), one read-and-check for each `FAILED_LOGIN_*` key
(`config/config.go:235-253`), `net.JoinHostPort` (`cmd/server/main.go:248`),
`WriteError` and corrected route comments
(`handler/app_user_group_handler.go:29,52-56,89,176,267`).

#### Proposed fix for PROTO

1. Add `internal/model/time.go` with `EpochSeconds` and `Now`; port the three
   tests. Module path is `github.com/wbmartin/starter-api`.
2. Replace both `CurrentTimestamp` functions and the inline copies with
   `model.Now()`; delete the two functions (about 20 call sites, mechanical).
3. `auth/totp.go:57`: hoist the regexp to a package `var`.
4. `config/config.go`: fold `:196-200` into `:135-143`; delete `:206-210`.
5. `cmd/server/main.go:202`: `net.JoinHostPort` (pairs with B061).
6. `app_user_group_handler.go`: `WriteError` at `:56`, fix four comments.
7. Do it as PS0011's first slice and refresh that ticket's line numbers.

#### Open question for Brandon

None.

### B061 — `SERVER_BIND_ADDRESS` required (fail-closed bind)

- **Priority:** P3  **Area:** go-api
- **GRC:** `src/internal/config/config.go:284-290`   **PROTO:** `src/api-srv/internal/config/config.go:106-109`; `etc/loc-api-srv.env:27-28`
- **Ticket:** RG0009 / `d36bfda`
- **What:** GRC refuses to start without a bind address; PROTO's empty default
is deliberate for LAN/phone dev.
- **Port notes:** Brandon's call; B016 covers deployed hosts.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

The bind address is optional, and empty means every interface:

```go
// src/api-srv/internal/config/config.go:106-109
// Optional. Empty listens on every interface, which is what every config
// did before the key existed. 127.0.0.1 is right behind nginx and for the
config.ServerBindAddress = env["SERVER_BIND_ADDRESS"]
```

`cmd/server/main.go:202` then listens on `":8080"`. Every deployed env file
sets `127.0.0.1` (`infra/uat/etc/uat-proto-api.env:29`,
`infra/prd/etc/prd-proto-api.env:29`, `etc/e2e-api-srv.env:58`). Three do
not: `etc/loc-api-srv.env:27-28` leaves it commented out on purpose,
`etc/uat-api-srv.env.example` (the presales UAT file) omits it, and any
derived app's new env file omits it by default. Forgetting the key fails
open: the API answers on every interface of the host, beside nginx rather
than behind it, and only a firewall stands between it and the internet. A
security default, not a live hole on the infra hosts; the presales UAT
gap is B016 (P1).

The stated reason for the dev default is weaker than it reads.
`etc/loc-api-srv.env:27` says the vite proxy and a phone on the LAN need it.
The proxy targets `http://localhost:8080` (`src/web-app/.env.development`),
which a loopback bind serves. A phone reaches vite, not the API, and
`npm run dev` (plain `vite`) listens on localhost only. Nothing in the repo
needs the API on every interface; a developer who points a device straight
at the API does.

#### What GRC changed

GRC refuses to start without the key, and the error names the fix
(`src/internal/config/config.go:284-290`: "SERVER_BIND_ADDRESS is required
(the interface to listen on, e.g. 127.0.0.1; 0.0.0.0 for every
interface)"). Every GRC env file sets `127.0.0.1`, including dev
(`etc/loc-api-srv.env:18-20`). `config_test.go:229` sets the key in its
fixture; no test asserts the refusal. The doc's ticket ref is off: the
check came in with RG0001 (`e54967b`); RG0009 (`d36bfda`) only moved
`config.go` into `internal/config`.

#### Proposed fix for PROTO

1. `config/config.go:106-109`: return GRC's error when the key is empty.
2. `etc/loc-api-srv.env:27-28`: set `SERVER_BIND_ADDRESS=127.0.0.1` with
   GRC's comment ("0.0.0.0 publishes it" for device testing).
3. `etc/uat-api-srv.env.example`: add the key (do with B016).
4. `cmd/server/main.go:202`: `net.JoinHostPort` (B060 item 4).
5. `src/api-srv/README.md:37` and `doc/330:190-191`: "required".
6. Go test: PROTO has no `LoadConfig` test today; add one in
   `config_test.go` that loads a temp env without the key and expects the
   error.

#### Open question for Brandon

Require the key everywhere, dev included? Recommend yes: dev works on
loopback, and fail-closed protects every env file a derived app adds.

### B062 — Preferences: off the admin list; validated against options

- **Priority:** P3  **Area:** go-api, web-app
- **GRC:** `src/internal/model/app_user_model.go:5-37,60-76`; `src/internal/repository/user_repository.go:399-465`; `src/internal/model/app_user_model_test.go:12-44`; `utils/preference.ts:14-59`; `utils/preference.test.ts`; RG0016 md `:128,145`   **PROTO:** `src/api-srv/internal/model/app_user_model.go:19`; `repository/user_repository.go:302-373` (:308); `types/profile-type.ts:5-25` (:21-26)
- **Ticket:** RG0016 / `62403fd`, `47fa7c4`
- **What:** `READ_USER` on `/userList` returns others' preferences; GRC serves
them only on `/me`. PROTO's `resolvePreference` trusts any `rowsPerPage`
(100000 renders every row); GRC allow-lists.
- **Port notes:** Keep PROTO naming. Docs rated P2.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Two small gaps around `app_user.preference_jsonb`, the UI settings (colour
scheme, rows per page) a user saves through `PUT /me`.

**(1) Admins read everyone's preferences.** The admin wire shape carries it:

```go
// src/api-srv/internal/model/app_user_model.go:19
PreferenceJSONB json.RawMessage `json:"preferenceJsonb,omitempty" ...
```

`RetrieveUserList` selects it (`repository/user_repository.go:308`), so a
`READ_USER` holder gets every user's document from `GET /userList`. Nothing
in it is secret today. A least-privilege gap: a derived app that stores
more there leaks that too.

**(2) The web app trusts any stored value.** The API checks only "JSON
object, at most 8 KiB" (`handler/profile_handler.go:190-194`). The form
limits rows to 5-200 (`pages/user-preferences-page.tsx:74-77`), but a direct
`PUT /me` skips the form, and the reader takes any positive number and any
colour string (`types/profile-type.ts:20-24`):

```ts
colorScheme: preference?.colorScheme ?? defaultPreference.colorScheme,
  preference?.rowsPerPage && preference.rowsPerPage > 0
```

A stored `rowsPerPage: 100000` renders every row of every list; a stored
`"purple"` reaches Mantine. A user can write only their own row, so this is
a robustness bug, not a hole.

#### What GRC changed

RG0016 (`62403fd`, `47fa7c4`) tagged the field `json:"-"` with the reason
(`src/internal/model/app_user_model.go:13-17,31`), dropped it from the list
query (`repository/user_repository.go:402-408`), and serves it only through
`AppUserOwnProfileModel` on `/me` (`model/app_user_model.go:59-76`).
`src/web-app/utils/preference.ts:15-59` allow-lists both fields
(`ROWS_PER_PAGE_LIST = [10, 20, 25, 50, 100]`) and falls back per field.
Tests: six cases in `utils/preference.test.ts` (`:37` "refuses a page size
outside the offered list"). No Go test pins that preferences stay off
`/userList`.

#### Proposed fix for PROTO

1. Trap: PROTO's `GET`/`PUT /me` serialise `AppUserModel` itself
   (`handler/profile_handler.go:86,176`), so `json:"-"` alone empties `/me`.
   Add an own-profile struct for those two responses (PROTO naming, no GRC
   `profile` wrapper, no `AccountStatus`), then tag the admin field.
2. `user_repository.go:308,351`: drop the column from `RetrieveUserList`.
3. `types/profile-type.ts`: allow-list `resolvePreference`; the form's
   `NumberInput` becomes a `Select` over the same list.
4. vitest: port GRC's six `resolvePreference` cases.
5. pytest `tests/test_own_profile.py`: `/userList` rows lack
   `preferenceJsonb`; `/me` keeps it.

#### Open question for Brandon

Rows per page as a fixed list (GRC: 10/20/25/50/100) instead of 5-200?
Recommend the list; "valid" becomes a one-line check.

### B063 — Web: pinned page header and sticky filter row

- **Priority:** P3  **Area:** web-app
- **GRC:** `index.css:53-76`; `hooks/use-pinned-page-header.ts:1-41`; `components/grc/sticky-filters.tsx:1-37`; `components/grc/shell.tsx:164`   **PROTO:** `index.css`; new `hooks/use-pinned-page-header.ts`, `components/sticky-filters.tsx`
- **Ticket:** `b52fa25`, `3d48e65`
- **What:** Title block and filters stay visible while long tables scroll.
- **Port notes:** After B025; rename `grc-page`/`--grc-*`.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

A UX gap, not a bug, and a small one in PROTO today. The shell is a plain
flex row (`src/web-app/app.tsx:176-196`): a 200 px `<nav>` and a `<main>`,
with no fixed top bar. Each page opens with a heading row that scrolls away
with the content:

```tsx
// src/web-app/pages/app-user-list-page.tsx:121-122
<Group justify="space-between" mb="lg">
  <Title order={1}>User Admin</Title>
```

On a long table the title and the "Add User" button leave the screen.
PROTO's three list pages have no filter row at all (no `TextInput` or
`Select` in `pages/*list-page.tsx`), and they paginate at 15 rows by default
(`types/profile-type.ts:14`), so tables are short. The problem grows in a
derived app that adds filters and long pages; the template barely shows it.

#### What GRC changed

The shell wraps the routed page in `<div className="grc-page">`
(`components/grc/shell.tsx:164`), inside Mantine's `AppShell` with a fixed
header (B025). Two pieces build on that.

1. **Pinned header** (`b52fa25`, Brandon 2026-09-29). CSS pins the page
   root's first child that is, or holds, the `<h2>`
   (`src/web-app/index.css:58-66`): `position: sticky`,
   `top: var(--app-shell-header-offset, 56px)`, body background, `z-index:
   3`. A `scroll-state(stuck: top)` container query hides the subtitle
   `<p>` once pinned (`:72-76`); browsers without it keep the subtitle.
   `hooks/use-pinned-page-header.ts:17-41` measures that block with a
   `ResizeObserver` and `MutationObserver` and writes
   `--grc-page-header-height` on the page element.
2. **Sticky filters** (`988b62e`, 2026-09-23). `<StickyFilters>`
   (`components/grc/sticky-filters.tsx:21-37`) is a sticky `Box` at
   `calc(header offset + --grc-page-header-height)`, `z-index: 2`, so a
   Select dropdown still opens above it.

The doc cites `3d48e65` for this; that commit put the title in the top bar
and `b52fa25` replaced it. Sticky filters came from `988b62e`. No tests.

#### Proposed fix for PROTO

1. Depends on B025: no `AppShell` header means no
   `--app-shell-header-offset`; without it, pin to `top: 0`.
2. `index.css`: add the two rules under a PROTO class (`app-page`,
   `--app-page-header-height`), not `grc-page` / `--grc-*`.
3. Trap: GRC's selector keys on `h2`; PROTO pages use `Title order={1}`.
   Change the selector to `h1`, or move pages to `order={2}`.
4. Add `hooks/use-pinned-page-header.ts` (renamed variables) and
   `components/sticky-filters.tsx`; call the hook in `AppLayout`.
5. No list page gains `StickyFilters` until it has a filter row.
6. Playwright: scroll the example list; the title stays in the viewport.

#### Open question for Brandon

Port now, or wait until a derived app has filter rows and long pages?
Recommend porting the pinned header with B025 and `StickyFilters` only
when the first filter row lands.

### B064 — Web: colour scheme only when signed in; list-page URL fixes

- **Priority:** P3  **Area:** web-app
- **GRC:** `App.tsx:283-309`; `pages/app-user-list-page.tsx:25-37,95`; `pages/app-user-group-list-page.tsx:46-58,80-93`   **PROTO:** `app.tsx:162-164`; `pages/app-user-list-page.tsx:116`; `pages/app-user-group-list-page.tsx:36-70`
- **Ticket:** RG0016; RG0008
- **What:** PROTO writes the default scheme before `/me` (reload flash, leaks to
next user); GRC applies on `YES`. Stale `?page=` clamps; sort keeps other
params; group list sorts.
- **Port notes:** Scaffolding pages: per page, no factoring.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Two small web-app UX bugs. Paths are under `src/web-app/`.

**(1) Colour-scheme flash on reload.** Mantine keeps the scheme in
localStorage (`main.tsx:13`, `defaultColorScheme="auto"`). `AppLayout`
applies the preference unconditionally:

```tsx
// app.tsx:162-164
useEffect(() => {
  setColorScheme(colorScheme);
}, [colorScheme, setColorScheme]);
```

`usePreference` returns the default `"auto"` until `readProfile` answers,
and that is a second call made only after `authState` turns `YES`
(`app.tsx:156-160`). A dark-mode user who reloads sees stored dark, then
`"auto"` written over it, then dark again when `/me` lands. The doc's
"leaks to next user" is overstated: sign-out clears the profile store
(`stores/profile-store.ts:58,61`), so the effect writes `"auto"` and the
next person starts from the default.

**(2) List URLs.** `pages/app-user-list-page.tsx:27` takes `?page=` as is;
a stale link (`?page=9` on a two-page list, or the last row deleted) shows
an empty table. The sort at `:116` is `setSearchParams({ sort, dir })`,
which wipes every other query parameter a derived page adds. `:40` casts
any `?dir=` to `"asc" | "desc"`. `pages/app-user-group-list-page.tsx`
has the same page parse (`:38`) and does not sort at all (plain `<Th>`,
`:120-123`). `pages/example-list-page.tsx:33,133` repeats both URL bugs.

#### What GRC changed

RG0016 moved the effect into `ColorSchemePreference`
(`src/web-app/App.tsx:292-309`): it sets the scheme only when `authState`
is `YES`, and calls `clearColorScheme()` only on a real `YES` to signed-out
transition, so a cold load keeps the stored scheme. GRC's profile arrives
with the identity answer (`stores/profile-store.ts:59-67`). RG0008 added
`setListParams`, which merges into the query string
(`pages/app-user-list-page.tsx:29-35`), sorts with `page: "1"`, parses
`dir` strictly (`:28`), and clamps `currentPage` (`:95`). The group list
gained the same plus a name/id sort (`app-user-group-list-page.tsx:44-58,
80-95`). No tests pin either.

#### Proposed fix for PROTO

1. Trap: PROTO loads the profile after `YES`, so gating on `YES` alone still
   flashes. Gate `app.tsx:162-164` on a loaded profile (profile not null);
   call `clearColorScheme()` on the signed-in to signed-out edge.
2. Each list page, separately (scaffolding, no shared hook): merge-style
   `setListParams`, strict `dir`, `Math.min(page, Math.max(1, pages))`.
3. Group list: `sort`/`dir` params and sortable headers for id and name.
4. Playwright: `?page=99` on the user list shows rows; as a dark-mode user,
   delay `/me` with `page.route`, reload, and expect `<html
   data-mantine-color-scheme="dark">` before `/me` answers.

#### Open question for Brandon

None.

### B065 — Web: small UI helpers

- **Priority:** P3  **Area:** web-app
- **GRC:** `components/grc/help-tip.tsx:1-34`; `components/grc/sort-th.tsx:1-18`; `hooks/use-sort.ts:1-25`; `components/grc/save-bar.tsx:1-13`; `utils/safe-href.ts:1-26`; `hooks/use-draft.ts:17-24`   **PROTO:** new under `components/`, `hooks/`, `utils/`; `pages/app-user-list-page.tsx:135-148`
- **Ticket:** `e2a110f`; RG0043 1.6
- **What:** Info-icon labels, sortable `<Th>`, save bar, link allow-list for
untrusted content.
- **Port notes:** `use-sort` to `useSearchParams`; `safe-href` once untrusted
HTML renders; `use-draft` lacks try/finally.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Nothing is broken. This is a DX gap: six small helpers GRC grew that PROTO
lacks. PROTO's sortable header is hand-written per column, four times on
the user list alone:

```tsx
// src/web-app/pages/app-user-list-page.tsx:150-155
<Table.Th
  className="sortable-column"
  onClick={() => handleSort("name")}
>
  Name{" "}
  {sortColumn === "name" && (sortDirection === "asc" ? "↑" : "↓")}
```

The header is a clickable `<th>`, not a button, so a keyboard user cannot
sort. Field help, where it exists, goes in Mantine's `description` prop as
grey text under the label (three uses in `pages/` and `components/`).
PROTO renders no untrusted HTML (no `dangerouslySetInnerHTML` in the web
app), so it has no use for a link sanitiser yet. `components/`, `hooks/`
and `utils/` hold no shared leaf UI.

#### What GRC changed

Paths are under GRC `src/web-app/`.

1. `components/grc/help-tip.tsx:13-34`: `HelpTip` (info icon, tooltip,
   `aria-label`) and `LabelWithHelp` for an input's `label`. From `e2a110f`
   (Brandon 2026-09-29: help "in information icons / popup help").
2. `components/grc/sort-th.tsx:5-18`: `SortTh`, a `<Th>` holding an
   `UnstyledButton` and a chevron icon, so it is keyboard-reachable.
3. `hooks/use-sort.ts:6-25`: client-side sort in `useState`, nulls last,
   numeric-aware `localeCompare`.
4. `components/grc/save-bar.tsx:4-13`: Cancel/Save, shown only when dirty.
5. `utils/safe-href.ts:5-26`: `isSafeHref` keeps `http:`, `https:`,
   `mailto:`; `neutraliseLinks` strips the rest and adds `rel=noopener`.
   RG0043 1.6, for links inside tenant-uploaded `.docx` previews.
6. `hooks/use-draft.ts:10-26`: an edit buffer. `save` (`:17-24`) has no
   `try/finally`: if `update` throws, `saving` stays `true` and the error
   escapes uncaught. `blankAudited` (`:7`) hard-codes `securityContext: 1`.
No tests for any of the six.

#### Proposed fix for PROTO

1. Add `components/help-tip.tsx` and `components/sort-th.tsx` (no `grc/`
   folder). `@tabler/icons-react` is already a dependency.
2. Trap: `use-sort` keeps state in `useState`; PROTO's lists keep `sort`
   and `dir` in the URL. Write `SortTh` against PROTO's `handleSort`, or a
   URL-backed hook (same fix as B064's merge-style `setListParams`).
3. Swap the hand-written headers page by page. Each page keeps its own
   sort code (scaffolding); only the leaf `<Th>` is shared.
4. `save-bar`: add only with the first in-place edit page.
5. `safe-href`: defer until a page renders untrusted HTML.
6. Do not port `use-draft` as is: wrap `save` in `try/finally` and drop
   `blankAudited`'s tenant 1 (the tenant comes from the JWT).
7. vitest: `SortTh` toggles on Enter; `isSafeHref` refuses `javascript:`.

#### Open question for Brandon

None.

### B066 — Route-sweep Playwright spec (not GRC's `smoke/*.mjs`)

- **Priority:** P3  **Area:** tests, web-app
- **GRC:** `smoke/smoke.mjs:1-69`; `smoke/guard.mjs:1-60` (:24-31); `smoke/exercise.mjs`; `package.json` scripts   **PROTO:** `tests/e2e/`; `playwright.config.ts:47-80`; new `tests/e2e/route-sweep.spec.ts`; `tests/e2e/global-setup.ts`
- **Ticket:** `5b5deb6`, `e9f945a`; RG0041; RG0037
- **What:** GRC visits every route as every persona and flags error/blank pages
and console errors, via node scripts with no runner.
- **Port notes:** Keep Playwright; one spec over `ROUTES` x users; copy
`requireTestTenant` into `global-setup.ts`.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

A test-coverage gap. PROTO's Playwright suite (`src/web-app/tests/e2e/`,
`playwright.config.ts:46-73`) tests chosen flows: harness and login
(`smoke.spec.ts`), entity pages, examples, MFA, profile. Nothing visits
every route. Specs reuse one saved session as `admin`
(`playwright.config.ts:43,68`; `tests/e2e/global-setup.ts:44`). So a route
that crashes for a user without admin grants, or for anyone on a route no
spec opens, shows the router's "Something went wrong" (`app.tsx:42`) or the
boundary's "This page ran into a problem"
(`components/error-boundary.tsx:59-60`) and no test fails. The route list
is small today (`constants/routes.ts:4-20`, ten entries), but every
derived app adds pages faster than specs.

#### What GRC changed

GRC wrote a route-by-role sweep as plain node scripts, not Playwright tests
(`5b5deb6`, `e9f945a`). Paths are under GRC `src/web-app/smoke/`; the doc's
`smoke/` refs omit that prefix.

1. `smoke.mjs:13-32` lists 88 routes, `:12` five roles. For each pair
   it navigates, then flags `ERROR_PAGE`, `NOT_FOUND_TEXT`, `BLANK`
   (under 40 characters) and console errors (`:49-62`). It prints flagged
   rows and a count; it never exits non-zero, so nothing gates on it.
2. `guard.mjs:16-21` refuses any base URL but the test web app (RG0041);
   `requireTestTenant` (`:24-31`) reads `/api/me` after sign-in and exits
   unless `securityContext` is 2. `signInAs` (`:46-60`) posts `/auth` as
   a persona.
3. `package.json` scripts `smoke` and `smoke:crud`. RG0037 anchored the
   not-found regex (`smoke.mjs:56-58`). No assertions, no CI wiring.

#### Proposed fix for PROTO

1. New `tests/e2e/route-sweep.spec.ts`: import `ROUTES`, add `?id=new` and
   `?id=<fixture>` forms for edit pages, and generate one `test()` per
   route x user. Assert no error heading, body text over 40 characters,
   and no `console`/`pageerror` errors. Real assertions replace GRC's
   printout, and `bin/300-run-tests` gates on them.
2. Users: `admin` and `basic` (both MFA-exempt in
   `sql/fixtures/local-seed.sql:114-119`). Sign each in with
   `fixtures/api-client.ts` `signIn` and per-describe `storageState`.
   Skip `mfa`.
3. Trap: copying `requireTestTenant` into `global-setup.ts` fails at once.
   Every fixture user sits in tenant 1 (`local-seed.sql:103`; the comment
   at `:23-26` says so until PS0022). Land B018/PS0022 first, then add the
   tenant-2 check to `global-setup.ts:44-63`.
4. Leave out GRC's role matrix, personas, `exercise.mjs` and the Chrome
   path. Expected refusals (basic on `/appUserList`) are an allow-list in
   the spec.

#### Open question for Brandon

None.

### B067 — Hash-chained audit log for login-layer writes (pattern)

- **Priority:** P3  **Area:** go-api
- **GRC:** `src/internal/repository/writes.go:16-33`; `user_invite_repository.go:351-388`; `audit_repository.go`   **PROTO:** new
- **Ticket:** `853f92c`; RG0035
- **What:** Tamper-evident entries in the write's transaction; GRC has not
covered user/group/password writes (RG0043 2.3).
- **Port notes:** Feature; needs roles forbidding TRUNCATE (B050).
- **Decision:** \[ \] pull / \[ \] adapt / \[X\] skip  (Brandon: no audit log in the template, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

A missing feature, not a bug. PROTO keeps no record of who changed what.
The login-layer writes in `src/api-srv/internal/repository/user_repository.go`
(`CreateUser` `:126`, `UpdateUserPassword` `:186`, `UpdateUserTOTPSeed`
`:214`, `UpdateUserMFAExempt` `:272`, `UpdateUser` `:375`, `DeleteUser`
`:427`, group membership `:478,537`) each update the row and stamp
`update_user_id`. Only the last writer survives. There is no `audit` table
in `sql/migrations/`. After an incident nobody can answer "who made this
user MFA-exempt, and when". The API role could not protect such a log
anyway: `sql/migrations/0001_initial_schema.sql:112` grants `db_writer`
(held by `proto_app`, `sql/roles.sql:43`) `ALL PRIVILEGES`, which includes
`UPDATE`, `DELETE` and `TRUNCATE`.

#### What GRC changed

GRC built a per-tenant hash chain for its GRC entities (`853f92c`, build
step 4). `audit_log` (`src/migrate-srv/sql/00003_grc_entities.sql:1825-1847`)
holds actor, action, object, `before`/`after` JSON, `prev_hash`, `hash`.
`AuditRepository.Append` (`src/internal/repository/audit_repository.go:48-94`)
must run in the write's transaction. It locks the tenant's `audit_anchor`
row `FOR UPDATE` (`:51-57`) so writers serialise, reads the chain head,
and stores `sha256(prev | at | actor | action | type | id | before |
after)` over canonical sorted-key JSON (`:36-42`). `Actor`
(`writes.go:16-23`) carries user, e-mail, source and IP. Verification
exists as `migrate audit-verify` (`migrate-srv/cmd/migrate/hostops.go:30`)
and a handler (`handler/grc_custom_handler.go:167`). Test:
`TestFixtureChainVerifies` (`repository/entity_repository_test.go:219`)
against `testdata/audit_chain_golden.json`.

GRC has not covered its own login layer. Only invites are chained
(`user_invite_repository.go:351-388`); its header (`:16-18`) admits the
rest "predate the audit log", and RG0043 §2 item 3 lists user, group,
password and TOTP writes as open. The doc's "RG0043 2.3" means that item.

Two limits. The chain is tamper-evident, not tamper-proof: the anchor
lives in the same database, so a holder of the API role can rewrite every
row and the anchor together. And GRC's app role still has `TRUNCATE` too
(RG0043 §4.18, B050).

#### Proposed fix for PROTO

1. B050 first: app role gets `SELECT, INSERT` only on the audit tables.
2. Migration `NNNN_audit_log.sql`: `audit_log` and `audit_anchor` by
   PROTO's rules (`<table>_id`, `security_context_id`, no FK, no
   defaults); leave out `actor_person_id` and `object_label`.
3. `repository/audit_repository.go`: port `Append`, `ChainHash`,
   `Canonical`; module path `github.com/wbmartin/starter-api`.
4. Wrap each write above in `Begin`/`Commit` with its entry; pass an
   `Actor` from the handler (JWT subject, `ClientIP`).
5. Go tests: golden chain verifies; an edited row breaks verification;
   a rolled-back write leaves no entry.

#### Open question for Brandon

Is a template-level audit log wanted, and is a same-database anchor
enough? Recommend yes, and export the head hash to the daily backup so a
rewrite is detectable off-box.

### B068 — Generic entity registry: ADR, not a backport

- **Priority:** P3  **Area:** go-api
- **GRC:** `src/internal/model/entity.go:1-40`; `model/grc_entities_gen.go`; `repository/entity_repository.go`; `handler/grc_handler.go`; `handler/grc_routes_gen.go`; `repository/writes.go:35-155`   **PROTO:** `src/api-srv/internal/model/example_model.go`, `repository/example_repository.go`, `handler/example_handler.go`; PS0010
- **Ticket:** `853f92c`; generator retired RG0020
- **What:** One generic repository/handler drives 85 entities from struct tags;
checks stay explicit in one handler. PROTO: \~2300 lines per entity.
- **Port notes:** Conflicts with "scaffolding" and moves check differences into
a matrix. ADR first.
- **Decision:** \[ \] pull / \[ \] adapt / \[X\] skip  (Brandon, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Nothing is wrong; this is a cost trade-off for derived apps. Adding an
entity means copying the `example` entity end to end. Measured line counts
for that entity (`wc -l`):

- Go: `model/example_model.go` 24, `repository/example_repository.go` 476,
  `handler/example_handler.go` 930 (8 methods). 1,430 lines.
- Web: `types/example-type.ts` 44, `stores/examples-store.ts` 40,
  `pages/example-list-page.tsx` 228, `pages/example-edit-page.tsx` 256.
  568 lines.
- Tests: Go 719, pytest `tests/test_a_example_crud.py` 288, Playwright 61.

About 2,000 production lines, 3,070 with tests. The doc's "~2300" sits in
between. Each handler method spells out its checks
(`handler/example_handler.go:73-81`: `CheckJWT`, `CheckRateLimitAndUserBlock`,
`CheckCSRF`, `CheckAuthorization(model.GrantExampleRead)`,
`CheckAllChecksComplete`). That is the CLAUDE.md rule ("exceptions are
written in the handler"), and the web pages are copy-paste by Brandon's
2026-09-21 ruling (PS0012). PS0010 (open) already targets the Go ceremony:
error mapping, the duplicated validation block, the stale doc/810 recipe.

#### What GRC changed

GRC needed 85 entities, so it built one generic stack (`853f92c`, build
step 4). Under `src/internal/`:

1. `model/entity.go:10-37`: an embedded `Audited` block, an `Entity`
   interface, and `EntityInfo` (resource, table, append-only, masked).
   `model/grc_entities_gen.go` holds the 85 structs and registry entries
   (2,749 lines, about 32 per entity).
2. `repository/entity_repository.go` (396 lines): `EntityRepository[T]`
   builds column lists from struct tags, with tenant on every predicate.
   `repository/writes.go:46` and on: `CreateEntity`/`UpdateEntity`/
   `DeleteEntity`, each row write in a transaction with its audit entry.
3. `handler/grc_handler.go` (495 lines): `listHandler[T]` and kin. Every
   route calls `begin` (`:49-64`), the same explicit chain ending in
   `CheckAuthorization(model.GrantLogon)`. The per-entity decision then
   comes from a matrix, `authz.Decide(c.Roles, obj, verb)` (`:96-108`).
4. `handler/grc_routes_gen.go:19+`: one `registerResource[T]` line per
   entity (85 lines). RG0020 retired the generator; files are hand-edited.

Per-entity cost is a struct plus a route line plus matrix rows. The
policy moved out of handlers: a role denied a list gets `200 []`, not 403
(`grc_handler.go:184-190`), a rule visible only in the generic code.

#### Proposed fix for PROTO

1. Do not port. Write an ADR, `doc/920-entity-registry-adr.md`, that
   records the trade-off: per-entity lines vs. checks visible per handler.
2. The ADR names the trigger for revisiting (a derived app passing about
   10 entities) and what would stay explicit (`begin`-style check chain).
3. Ship PS0010 first; its lean example sets the baseline the ADR compares.
4. No tests.

#### Open question for Brandon

Should a derived app with many entities be allowed a generic registry,
given it moves authorization differences into a matrix? Recommend an ADR
that permits it only for derived apps past ~10 entities, never the
template.

### B069 — Web dependency hygiene: depcheck config, zod pin

- **Priority:** P3  **Area:** web-app
- **GRC:** `.depcheckrc.yml`; `package.json` (`"zod": "^4.6.5"`)   **PROTO:** `package.json` (`"zod": "4.1.13"`)
- **Ticket:** RG0007 / `d7750dc`
- **What:** depcheck carries a recorded exception list. PROTO's other deps are
newer.
- **Port notes:** Brandon's call: caret or exact zod? `coverage` ignore is B041.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Two small things, one real and one already solved.

1. zod is the only exact pin. `src/web-app/package.json:31`:

       "zod": "4.1.13",

   Every other dependency is a range (`:21-51`; caret, typescript tilde), and
   `src/web-app/README.md:114-118` says "Caret ranges. Any ticket that
   touches the web app runs `npm update --legacy-peer-deps`". PS0013
   (`tickets/04-archive/PS0013-web-hygiene-deps-schemas-constants.md:30-31`)
   "declared `zod` at its already-resolved version", which wrote the bare
   version, not `^4.1.13`.
   Consequence: the README's in-range update never moves zod. The lock
   holds 4.1.13 while GRC's holds 4.6.5. A future zod advisory in 4.x is
   not picked up by the documented cadence. Quality gap, not a bug today.
2. depcheck config. PROTO has no `.depcheckrc.yml`, and needs none now:
   `npx --no-install depcheck` in `src/web-app` prints "No depcheck issue"
   (run 2026-10-02). The feedback line "PROTO's other deps are newer" is
   true of the lock (vite 7.3.6 vs 7.2.7, vitest 4.1.11 vs 4.0.15,
   @mantine/core 8.3.18 vs 8.3.10), not of `package.json`, whose shared
   ranges are identical.

#### What GRC changed

RG0007 (`d7750dc`) found zod imported by `schemas/form-schemas.ts` but
undeclared, and declared it as `"zod": "^4.6.5"` (GRC
`src/web-app/package.json:33`). PROTO already declares zod (PS0013), so
only the caret matters here. A follow-up, `606a9d5`, added
`src/web-app/.depcheckrc.yml:13-14` with one ignore:

    ignores:
      - "@vitest/coverage-v8"

Reason (`:5-12`): `test:coverage` loads the provider through a CLI flag,
which depcheck cannot see; "Nothing else belongs in this list." The commit
itself records the file was unnecessary while the lock was fully updated,
because newer vitest makes the provider reachable through its own metadata;
it became necessary only after GRC pinned the lock back. No tests.

#### Proposed fix for PROTO

1. `src/web-app/package.json:31`: `"zod": "^4.1.13"`, then
   `npm update --legacy-peer-deps zod` per the README cadence. The only
   consumer constraint is `mantine-form-zod-resolver` peer `zod >=3.25.0`.
2. Do not add `.depcheckrc.yml`. PROTO's lock is newer and depcheck is
   clean. If a later lock trips the `@vitest/coverage-v8` false positive,
   copy GRC's file verbatim, comment included.
3. The `coverage/` gitignore belongs to B041, not here.
4. Tests: none new; `npm run build`, `npm run test:run`, and
   `npx depcheck` must stay clean after the bump.

#### Open question for Brandon

Caret or exact for zod? Recommendation: caret, because the README already
says caret ranges and the exact pin is an accident of PS0013.

### B070 — bin dev conveniences: targeted restart, test API, dirty flag, coverage

- **Priority:** P3  **Area:** bin
- **GRC:** `bin/215-run-api-srv:33-36,44,53`; `bin/216-run-test-api-srv:1-38`; `bin/600-bundle-uat:52-53,98`; `bin/620-export-tenant-data:71`; `bin/310-test-repo-model:121-147`   **PROTO:** `bin/215-run-api-srv:36,44,53`; `bin/300-run-tests:58-89,140-182`; `bin/600-bundle-uat:58,113`
- **Ticket:** RG0041 / `31970ae`; RG0012 / `2890e8a`; RG0006 / `0cf57f7`
- **What:** Restart kills only the dev instance (binary + env path); a
long-lived test API on 8082; staged changes mark a bundle dirty;
repository/model coverage HTML.
- **Port notes:** PROTO's per-run instance is the better gate. Keep
`tmp/api-srv/`. 221 is SKIP.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

1. Restart matches too wide. `bin/215-run-api-srv:36` (also `:44,53`):

       EXISTING_PIDS=$(pgrep -f "dist/$APP_SLUG-api" || true)

   Any process whose argv contains `dist/proto-api` dies, including the
   dev API of a second checkout or agent worktree of the same app.
   `bin/216-stop-local-stack:11` is wider: `pkill -f "vite"` kills every
   Vite server on the machine, any project. DX gap.
2. Dirty flag misses staged work. `bin/600-bundle-uat:58,113` use
   `git diff --quiet`, which compares work tree to index only. Changes
   `git add`ed but not committed ship stamped clean in `MANIFEST`, so UAT
   claims a commit it is not running. Small provenance bug.
3. No standing test API. Deliberate: `bin/300-run-tests` `py` builds a
   private binary (`:165-167`), stops it on exit (`:171-175`), and refuses
   to start if 8082 already answers (`:158-160`). Not a gap.
4. No Go coverage report: nothing under `bin/` passes `-cover`; only
   vitest has a floor (`src/web-app/vitest.config.ts:12-17`). DX gap.

#### What GRC changed

- RG0041 (`31970ae`): `bin/215-run-api-srv:33-36,44,53` match binary plus
  env file, `pgrep -f "proto-go-app $CONFIG_PATH"`, so a dev restart
  spares the test instance. New `bin/216-run-test-api-srv:1-38` keeps a
  long-lived API on 8082 (e2e DB, tenant 2) for pytest, smoke and load.
- RG0012 (`2890e8a`): `bin/600-bundle-uat:52` sets `DIRTY` from
  `git diff --quiet && git diff --cached --quiet`, used at `:53,98`;
  `bin/620-export-tenant-data:71` repeats it.
- RG0006 (`0cf57f7`): `bin/310-test-repo-model:121-147` writes a
  coverprofile, `-func` summary and HTML to `tmp/test-reports/`.
- No tests pin any of these.

#### Proposed fix for PROTO

1. `bin/215-run-api-srv:36,44,53`: match `"$BINARY_PATH $CONFIG_PATH"`.
   Keep `tmp/api-srv/` for log and pid (`:87-91`); GRC writes loose `tmp/`
   and hardcodes `proto-go-app` where PROTO derives `$APP_SLUG`.
2. `bin/216-stop-local-stack:10-11`: same binary+env match for the API;
   scope the Vite kill to this checkout's path (check real argv first).
   Not in GRC; same class of bug.
3. `bin/600-bundle-uat:58,113`: compute `DIRTY` once with GRC's two-part
   test (or `git status --porcelain` to count untracked files too).
4. Optional: add `-coverprofile` to the existing `go test` in
   `bin/300-run-tests:72` and write HTML to `tmp/test-reports/`; report,
   no floor.
5. Do not port GRC `bin/216-run-test-api-srv`. PROTO's 216 is taken, a
   standing 8082 instance makes bin/300 `py` refuse, and Playwright reuses
   it (`playwright.config.ts:84`), risking a stale binary. 221 is SKIP.
6. Tests: none; check by hand that restarting one checkout's API leaves
   another checkout's running.

#### Open question for Brandon

Add Go coverage to `bin/300 go`? Recommendation: yes, as a report with no
threshold, since it rides the existing `go test` run.

### B071 — Host survey and scrubbed tenant export

- **Priority:** P3  **Area:** bin, docs
- **GRC:** `bin/605-survey-host:1-97` (:27, :31,42,52,59); `bin/620-export-tenant-data:1-97` (:77); `bin/lib/export-scrub.sql:1-35` (:22-24); `doc/902-project-commands.md:96-97`   **PROTO:** new `bin/605-survey-host`, `bin/620-export-tenant-data`; `doc/330-presales-to-postsales-migration.md`; `doc/902-project-commands.md`
- **Ticket:** RG0013 / `3a3fc94`, `a57d6ad`, `cc30f35`
- **What:** 605 records a host read-only (key names only, masked) into
`tmp/survey/`. 620 copies the DB, sets every login `invited`, dumps
`--no-owner`, MANIFEST + sha256.
- **Port notes:** Names from `APP_SLUG`; mask is best-effort. 620 needs B035.
- **Decision:** \[ \] pull / \[X\] adapt / \[ \] skip  (Brandon: post-sales only, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Two operator gaps around moving an app onto a server.

1. No host survey. `doc/330-presales-to-postsales-migration.md:58-84`
   lists what must be true of the host; nothing records what is there
   (users, units, env keys, roles, owners, nginx, ports) before step 1
   changes it, so the "before" state is lost.
2. The data that reaches a server carries laptop credentials.
   `bin/600-bundle-uat:92-93` dumps the whole local database:

       docker exec "$PG_CONTAINER" pg_dump -U postgres -d "$LOCAL_DB" \
           --no-owner --no-privileges --format=plain --encoding=UTF8

   `bin/610-deploy-uat:117` restores it verbatim. Every `app_user` row
   arrives with its `pw_hash`, `totp_seed` and `mfa_setup`
   (`0001_initial_schema.sql:47,53-55`), including the fixture logins
   `doc/050-dev-env-init.md:30` seeds into the dev DB (`admin /
   admin123!`, per `bin/510-reset-test-db:29`). doc/330 then keeps that
   database for good. No tenant check either. Security gap at the
   presales-to-post-sales boundary.

#### What GRC changed

RG0013 (`3a3fc94`, `a57d6ad`, `cc30f35`):

- `bin/605-survey-host:1-97`: one sudo probe over `ssh -t`, teed to
  `tmp/survey/<host>-<stamp>.txt` (`:17-18,95`). Env files as key names
  only (`keys()`, `:26`). `mask()` (`:27`) is used only on crontab
  (`:88`); nginx sites (`:73`) and units (`:53`) print unmasked. GRC
  names are hardcoded (`:31,42,52,59`).
- `bin/620-export-tenant-data:1-97`: verify schema and audit chain, copy
  to a throwaway `_export` DB (`:48-55`), scrub, dump `-Fc --no-owner
  --no-privileges` (`:64`), MANIFEST with counts, logins, sha256
  (`:67-83`). `bin/lib/export-scrub.sql` refuses rows outside tenant 1
  (`:13-20`) and named test logins (`:22-25`), revokes open invites, and
  sets every login `invited` with empty `pw_hash`/`totp_seed` (`:33-35`).
  First access on the host is an invite link. No tests.

#### Proposed fix for PROTO

1. `bin/605-survey-host`: port the probe; derive users, `/srv/<env>-<slug>`,
   unit and database names from `etc/app.env` and `etc/uat-deploy.env`;
   swap `goose_db_version` (`:63`) for `schema_migration`; drop GRC's
   `pwsh`, `oras`, worker units. Apply `mask` to unit and nginx output
   too, and say in the header it is best-effort.
2. `bin/620-export-tenant-data` + `bin/lib/export-scrub.sql`: needs B035
   (`account_status`, invite redeem) or nobody can sign in after the
   scrub. Drop the audit-chain steps (B067), the blob tarball, GRC's
   table list (`:77`) and fixture names (`:22-27`); refuse tenant 2 and
   the PROTO fixture logins instead.
3. `doc/330` "Before you start": run 605 and keep its file; load from a
   620 export, not the last presales `db.sql`. One row each in `doc/902`.
4. Tests: a pytest or Go integration test that runs the scrub on the
   test DB and asserts no `pw_hash`/`totp_seed` survives and tenant 2
   rows make it fail.

#### Open question for Brandon

Should presales UAT itself stop shipping laptop credentials (scrub in
bin/600), or only the post-sales load? Recommendation: post-sales only
for now, since presales UAT is demo data and the scrub needs B035.

### B072 — Presales: install the nginx site from a template

- **Priority:** P3  **Area:** bin, etc
- **GRC:** `bin/610-deploy-uat:43-44,119-129`; `etc/uat-nginx-site.conf.template:1-34`; `bin/600-bundle-uat:93,102`   **PROTO:** `doc/320-presales-uat-deploy.md:79`; `etc/uat-deploy.env`
- **Ticket:** RG0012 / `2890e8a`, `d9eb2d2`
- **What:** First run writes and links the site, `nginx -t && reload`, prints
the certbot line.
- **Port notes:** GRC's template lacks security headers (add B022's); PROTO body
cap; `UAT_DOMAIN` in `etc/uat-deploy.env`.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

The presales path leaves nginx to the operator. `doc/320-presales-uat-deploy.md:79-80`,
one-time host prep step 3:

    3. Point nginx at `web-pub/` for static files and proxy `/api/` to
       `127.0.0.1:$SERVER_PORT/`, stripping the `/api` prefix.

`bin/610-deploy-uat:11` assumes it is done ("nginx configured"). There is
no file to start from: the only nginx sites in the repo are the
post-sales ones (`infra/uat/etc/nginx_uat-proto.example.com`, TLS,
upstream block), which do not match the presales layout. So each presales
host gets a hand-written site, with whatever headers, body cap and
forwarding headers the operator remembers. The `/api` strip is easy to
get wrong (a missing trailing `/` on `proxy_pass` sends `/api/auth` to an
API that has no `/api` prefix). DX gap with a security edge: a hand
site may omit the security headers and the `X-Forwarded-For` shape that
B007's `ClientIP` fix relies on.

#### What GRC changed

RG0012 (`2890e8a`, `d9eb2d2`). `etc/uat-nginx-site.conf.template:1-34` is
an HTTP-only site with `__UAT_DOMAIN__` and `__PORT__` placeholders:
`root /srv/uat-grc/web-pub` (`:12`), logs under the app root (`:15-16`),
`client_max_body_size 50m` (`:19`), `location /api/` with
`proxy_pass http://127.0.0.1:__PORT__/;` plus `Host`, `X-Real-IP`,
`X-Forwarded-For $proxy_add_x_forwarded_for`, `X-Forwarded-Proto`
(`:21-28`), and the SPA fallback (`:31-33`). `bin/600-bundle-uat:93`
ships it; `:102` writes `domain=$UAT_DOMAIN` into MANIFEST.
`bin/610-deploy-uat:43-44` reads the domain back and refuses a bundle
without one. On first run only (`:119-129`) it writes
`$APP_ROOT/etc/nginx_<domain>`, symlinks it into `sites-enabled`, runs
`nginx -t && systemctl reload nginx`, and prints the `certbot --nginx -d`
line. The template carries no security headers. No tests.

#### Proposed fix for PROTO

1. New `etc/uat-nginx-site.conf.template`. Derive paths from
   `__APP_ROOT__`, not GRC's literal `/srv/uat-grc`. Use PROTO's 20M body
   cap (post-sales site `:32`), not 50m. Put B022's headers inside
   `location /` only, so they do not stack on the API's own headers in
   `/api/`. Keep `X-Forwarded-For $proxy_add_x_forwarded_for` (B007).
2. `etc/uat-deploy.env`: add `UAT_DOMAIN` (no default; 600 stops if
   missing, per the file's header `:3-4`).
3. `bin/600-bundle-uat`: copy the template into the stage; write
   `domain=` into MANIFEST next to the existing keys (`:108-115`).
4. `bin/610-deploy-uat`: after the unit step, the GRC first-run block,
   with the port it already reads (`:56-57`). Never overwrite an existing
   site file.
5. `doc/320:79-80`: replace step 3 with "first deploy installs the site;
   run the printed certbot line once DNS points here".
6. Tests: none automated; `nginx -t` on the host is the check. Order:
   after B022 (header placement) and with B027 if both touch 610.

#### Open question for Brandon

None.

### B073 — Generated schema reference (doc/003) and dev migrate script

- **Priority:** P3  **Area:** docs, bin, go-api
- **GRC:** `bin/520-schema-reference:1-12`; `bin/205-migrate-db:1-18` (:3-6); `src/migrate-srv/cmd/migrate/schema_reference.go:11-120`; `doc/003-ai-postgres-schema-reference.sql:1-4`; `doc/900-database-guidelines.md:20`; `CLAUDE.md:35-38`; `doc/902-project-commands.md:101`; `src/internal/model/schema_contract_test.go:7-8,21,39-41`   **PROTO:** `doc/900-database-guidelines.md:27`; `src/api-srv/cmd/migrate/main.go`; new `bin/2xx`/`5xx`; `bin/510-reset-test-db`
- **Ticket:** `c30d256`, `fc3e098`
- **What:** A committed CREATE TABLE/INDEX snapshot regenerated after migrate;
doc/900 points SQL authors at it. PROTO makes readers replay every migration.
- **Port notes:** Add `schema-reference` to `cmd/migrate` (pgx, sorted);
regenerate in bin/510; fail bin/300 on diff. A `bin/205` around `up` is
trivial.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

`doc/900-database-guidelines.md:27` tells every SQL author, human or
agent:

    CRITICAL: For ALL SQL queries, the migrations in
    `src/api-srv/sql/migrations/` are the exclusive and absolute source ...

To know a table's columns a reader must replay every migration in their
head: `app_user` is `0001` plus `0003_app_user_preference_jsonb.sql:5`
(`ADD COLUMN preference_jsonb`); `example` is `0001` plus the two
`ALTER`s in `0002`. Three files today; a derived app grows to dozens, and
an agent that reads only `0001` writes SQL for a column set that no longer
exists. Separately, no script migrates the dev database: `doc/050` gives a
hand-typed `go run ./cmd/migrate -url "postgres://postgres:<pw>@..." up`
(`doc/050-dev-env-init.md:27`). Quality/DX gap, not a bug.

#### What GRC changed

Commit `c30d256` added a `schema-reference` command to the migrator
(`src/migrate-srv/cmd/migrate/schema_reference.go:11-120`). It reads
`information_schema.tables/columns` and `pg_indexes`, sorted by name, and
prints `CREATE TABLE` / `CREATE INDEX` text, collapsing `nextval(...)`
defaults to `SERIAL` (`:61-63`). `bin/520-schema-reference:11` writes it to
the committed `doc/003-ai-postgres-schema-reference.sql` (2,482 lines,
header `:1-4` says "GENERATED ... Do not edit"). `bin/205-migrate-db:16-18`
runs bootstrap, `up`, then 520. GRC `doc/900-database-guidelines.md:20`
and `CLAUDE.md:35-38` name doc/003 as the only source of column names.
Test: `src/internal/model/schema_contract_test.go:7-8,21,39-41` reads
doc/003 as text and checks model fields against it, with no database
(that half is B054). GRC generates from the dev DB, so a dev DB that has
drifted produces a wrong reference; nothing checks that.

#### Proposed fix for PROTO

1. `src/api-srv/cmd/migrate/main.go`: add a `schema-reference` command.
   Port the query logic on `pgx.Conn` (PROTO has no `database/sql` here),
   exclude PROTO's `schema_migration` table, keep name ordering. Usage
   lines `:3-8` and `:41` gain it. Header text says "proto-migrate",
   derived from the binary, not GRC's "goose"/"migrate-srv".
2. `bin/510-reset-test-db`: after `up` on the scratch test DB, regenerate
   `doc/003-...sql`. A DB built only from migrations makes the output
   deterministic, unlike GRC's dev-DB source.
3. `bin/300-run-tests` `go` mode: after 510, fail if
   `git diff --exit-code doc/003-...sql` shows a change ("regenerate and
   commit"). This stops a migration landing without its reference.
4. `doc/900:27`: point at doc/003 for reading, keep migrations as the only
   way to change schema. Add one row each to `doc/902`.
5. Optional `bin/205-migrate-db`: `go run ./cmd/migrate -config
   etc/loc-api-srv.env up` for the dev DB. Check the env file's role can
   run DDL first; doc/050:29 implies it may not.
6. Tests: one Go integration test that runs `schema-reference` on the
   test DB and asserts `CREATE TABLE app_user` with `preference_jsonb`.

#### Open question for Brandon

Should bin/300 fail when the committed doc/003 differs from the migrated
schema? Recommendation: yes, or the reference rots like any hand doc.

### B074 — infra small: systemd unit, monitor placement, env headers

- **Priority:** P3  **Area:** infra, docs
- **GRC:** `infra/uat-unified001/etc/uat-grc-api.service:1,11-14,21`; `uat-grc-restart.path:1-2`; `uat-grc-restart.service:1-2`; `infra/prd-unified002/shrmpl-etc/nack-monitors/uat-grc-renew-web-certs_*.env:1-5`; `infra/prd-unified002/shrmpl-etc/pulsecheck-endpoints/uat-grc-api.env:1-3`, `uat-grc-web.env:1-3`; `infra/README.md:28-30`; `doc/330-uat-postsales-cutover.md:313-314,329-330`; `infra/uat-unified001/etc/uat-grc-api.env:1-14,33-37`; `uat-grc-migrate.env:1-6`; `uat-grc-test.env:1-12`   **PROTO:** `infra/uat/etc/uat-proto-api.service:1,11-13,20` (+ prd); `infra/README.md:57-72,143`; `infra/uat/shrmpl-etc/nack-monitors/uat-proto-renew-web-certs_NACKID.env`; `infra/uat/shrmpl-etc/pulsecheck-endpoints/uat-proto-{api,web}.env`; `infra/uat/etc/uat-proto-api.env:1-12,25-29`; `infra/uat/etc/uat-proto-migrate.env:1-4`
- **Ticket:** RG0013 / `3a3fc94`, `cc30f35`, `a57d6ad`, `16f3acd`, `55b0066`
- **What:** Unit gains `WorkingDirectory` and `svc-<app>`; PROTO's
`ReadWritePaths=log/` is unneeded. UAT monitors belong on the monitoring host.
Env headers: secret recipe, back up `TOTP_ENCRYPTION_KEY` with the DB,
`sslmode=disable`.
- **Port notes:** `svc-proto` touches README:143, doc/330/340, rebrand. Both
lack `ProtectKernel*`, `SystemCallFilter` etc.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (Brandon: fix in the template; old hosts stay as they are, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Three unrelated infra nits; none is a bug in a running host today.

1. The unit. `infra/uat/etc/uat-proto-api.service:1` is a blank line (no
   note of where the file lives or that it is symlinked). `:11-12` run as
   `User=proto`, but `infra/prd/etc/prd-proto-api.service:11` says
   `User=prd-proto`, and `infra/README.md:143` creates `proto`: the two
   hosts already disagree. `:20` grants `ReadWritePaths=/srv/uat-proto/log`,
   but the API logs to the journal (`:22-23`) and writes no files; only
   nginx writes `log/` (`nginx_uat-proto.example.com:28-29`). There is no
   `WorkingDirectory`, so the API runs in `/` and any relative path in
   its env would resolve there. Neither PROTO nor GRC sets
   `ProtectKernel*`, `SystemCallFilter`, `CapabilityBoundingSet`.
2. Monitor placement. `infra/uat/shrmpl-etc/nack-monitors/` and
   `pulsecheck-endpoints/uat-proto-{api,web}.env` put UAT's monitors on
   the UAT host. A monitor on the host it watches cannot report that
   host down. `infra/README.md:57-72` describes the agents, not placement.
3. Env headers. `infra/uat/etc/uat-proto-api.env:1-10` says "redacted
   copy" and lists the secrets but gives no recipe for generating them and
   no warning that `TOTP_ENCRYPTION_KEY` must travel with the database (a
   restore under a new key makes every MFA enrolment undecryptable).
   `uat-proto-migrate.env:1-4` likewise. Docs gap.

#### What GRC changed

- `cc30f35`: service account `svc-grc` everywhere; `uat-grc-api.service:1`
  names its path and symlink, `:11-14` add `WorkingDirectory=/srv/uat-grc`,
  `:21` writes only `data/blobs` (GRC's blob store, not in PROTO).
  `uat-grc-restart.{path,service}:1-2` gain the same header comments.
- `a57d6ad`, `16f3acd`: UAT monitors moved to
  `infra/prd-unified002/shrmpl-etc/`, headers `:1-3` (pulsecheck) and
  `:1-5` (nackmon) say why; `infra/README.md:28-30`. GRC's stated reason is
  host-specific: UAT's pulsecheck "is not running (survey 2026-10-01)";
  `doc/330-uat-postsales-cutover.md:313-314` records it as a lesson.
- `55b0066`: every secret line carries `# note: generate with  openssl
  rand -hex 32` (`uat-grc-api.env:7,9,13`; `uat-grc-migrate.env:5`);
  `:11-12` tie the TOTP key to the database; URLs add `?sslmode=disable`
  (`:8`). `:33-37` explain the loopback bind. No tests.

#### Proposed fix for PROTO

1. Units (uat + prd, api + restart pair): add a one-line header naming
   the host path and symlink; add `WorkingDirectory=/srv/<env>-proto`;
   drop `ReadWritePaths=.../log`, check the API starts under
   `ProtectSystem=strict`. Pick one account name for both hosts.
2. Optional hardening: `ProtectKernelTunables`, `ProtectKernelModules`,
   `ProtectControlGroups`, `RestrictAddressFamilies=AF_INET AF_INET6
   AF_UNIX`, `SystemCallFilter=@system-service`; `systemd-analyze security`
   on a host to confirm.
3. Move `infra/uat/shrmpl-etc/{nack-monitors,pulsecheck-endpoints}` under
   `infra/prd/shrmpl-etc/`, prefixed `uat-`; note in `infra/README.md`
   "monitors live on the monitoring host, never the host they watch".
4. Env headers in `infra/{uat,prd}/etc/*-api.env` and `*-migrate.env`:
   the `openssl rand -hex 32` recipe, the TOTP-key warning, a loopback
   note. `sslmode=disable` is optional: pgx's default `prefer` already
   falls back on loopback; add it only to avoid a TLS attempt.
5. If renamed to `svc-proto`: `infra/README.md:143`, `doc/330`, `doc/340`,
   both units, env ownership notes; `bin/000-rebrand` must rewrite it
   (derive from `APP_SLUG`, never literal `svc-grc`). Tests: none.

#### Open question for Brandon

Rename the service account to `svc-<slug>` (GRC) or keep `<slug>`?
Recommendation: `svc-<slug>`, and fix the uat/prd mismatch in the same
change.

### B075 — Dev docs: doc/050, doc/060, doc/902, doc/810

- **Priority:** P3  **Area:** docs
- **GRC:** `doc/050-dev-env-init.md:1-8,10-14,16-35,118-127`; `doc/060-dev-env-start.md:1-43`; `doc/902-project-commands.md:3-8,40-42,46-51,60-63,67`; `doc/810-create-new-entity-type-notes.md:23-24,85-101,130-131,147-151`   **PROTO:** `doc/050-dev-env-init.md:5,42-44`; `doc/060-dev-env-start.md:1-34` (:31); `doc/902-project-commands.md:1-140`; `doc/810-create-new-entity-type-notes.md:1,84-118`
- **Ticket:** RG0011 / `06a1396`; RG0021 / `1663a3e`; RG0010; RG0006 / `719739b`; RG0016 / `5593a9a`; RG0020 / `c9ac1aa`; RG0008 / `48a557a`
- **What:** 050: real dev topology replaces the nginx TODO, tools, Linux box.
060: sections, repo-relative commands, `/health`. 902: one row per script,
gate contract. 810: one column end-to-end as a layer table.
- **Port notes:** Drop PowerShell/worker/imports. 810 feeds PS0010. Gate text
after B040/B041/B018.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Four developer docs describe a stack that no longer exists. Docs gap;
each item below makes a newcomer run the wrong command.

- `doc/050-dev-env-init.md:43-44` ends on a stray TODO: "wil mayneed to
  have a container running nginx". There is no nginx in dev (Vite proxies
  `/api`). No tools list (Go, Node, Docker, Python versions), no Linux
  notes.
- `doc/060-dev-env-start.md:17-20` says "First time, or to rebuild the
  schema: `bin/510-reset-test-db`". 510 builds `loc_proto_test_db` and
  refuses any name not ending `_test_db`/`_e2e_db`; the dev API reads
  `loc_proto_db` (`etc/loc-api-srv.env:7`), which stays empty (same bug
  as B085). `:31` hardcodes `~/Code/ProtoStack-ReactGo/bin/220-run-web-app`.
  No `/health` check, no headings that say which shell.
- `doc/902-project-commands.md`: `:17` curls another project's real host
  (B087); `:22` says `tmp/server.log`, but 215 writes
  `tmp/api-srv/server.log` (`bin/215-run-api-srv:87-89`); `:71` says
  `npm run test:unit`, which `package.json` does not define (bin/300 runs
  `npx vitest run`, `:95`). `bin/000-rebrand`, `100-build-api-server`
  and `400-make-local-dev-certs` have no entry.
- `doc/810-create-new-entity-type-notes.md:1` flags itself "procedure ...
  out of date" (PS0010). `:103-105` require `max<Entity>UpdateTs`; the
  stores use `maxUpdateTs` (`stores/create-entity-store.ts:10`).

Correction to B086: `./515-run-postgres.sh` (`doc/060:7`) is not a
missing `bin/` script. `:5` first `cd`s into the external infrahub
`loc-postgres` checkout, where the script exists. Only `:5`'s placeholder
path and `:31` need fixing.

#### What GRC changed

RG0011 (`06a1396`), RG0021 (`1663a3e`), RG0006 (`719739b`), RG0020
(`c9ac1aa`): `doc/050:1-8` states the dev topology (Postgres only,
in-process KV, slog to stderr, loopback bind); `:10-14` tools;
`:16-35` "A second machine (Linux)"; `:118-127` "no nginx in the dev
path". `doc/060:1-43` has one section per shell, commands relative to the
repo root, `curl -s http://localhost:8080/health` (`:22`).
`doc/902:3-8` says "one line per script ... a new `bin/` script means a
new row here", tables per number range, and `:60-63` the gate contract
("`git status --short` is empty after a run"). `doc/810:85-101` walks
one real column (`app_user.preference_jsonb`) through every layer as a
table; `:130-151` lists conventions. No tests.

#### Proposed fix for PROTO

1. `doc/050`: replace `:42-44` with a "Files and the web app" note (Vite
   proxy, no nginx); add Tools; keep a short Linux section. Drop GRC's
   PowerShell, worker, import, `ADMIN_DATABASE_URL`, smoke paths.
2. `doc/060`: three headed sections; dev DB via doc/050 or `bin/205` if
   B073 adds it; `bin/220-run-web-app` relative; add the `/health` curl.
3. `doc/902`: one table row per `bin/` script, fix `:17,22,71`, add
   the three missing scripts. Gate text waits for B040/B041/B018.
4. `doc/810`: remove the `:1` banner; add the layer table using PROTO
   paths (`0003_app_user_preference_jsonb.sql`, `profile_handler.go`, not
   GRC's `own_profile_handler.go`/`utils/preference.ts`, which PROTO
   lacks); fix `maxUpdateTs`. Feeds PS0010; land with or after it.
5. Optional guard: B053's path checker catches the next stale path.

#### Open question for Brandon

None.

### B076 — Web README: house patterns and small rules

- **Priority:** P3  **Area:** docs, web-app
- **GRC:** `README.md:106-108,120-122,136-139`; RG0008 md `:183`; `tickets/02-working/RG0034-capture-procedure-for-evidence/RG0034-capture-procedure-for-evidence.md:66`; RG0036 md `:28-38`   **PROTO:** `src/web-app/README.md:73-110`
- **Ticket:** RG0007; RG0008; RG0018; RG0034; RG0036
- **What:** Store test per store; empty load shows `<Alert />`; store throws,
page notifies; body keys equal json tags; Zustand `set` hides misspelled keys;
reload form on route-id change.
- **Port notes:** Docs only; example test from B013.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

`src/web-app/README.md:73-110` gives the copy list and a five-bullet house
pattern. Six rules GRC learned the hard way are missing, and PROTO's
example page already breaks two of them. Docs gap with real bugs behind it.

1. No store test in the copy list (`:75-80`). `tests/` holds
   `api-unreachable`, `entity-store-cache`, `session-expired`; no
   per-entity store test, so a wrong request body ships untested.
2. Body keys must equal the Go `json` tags. Not stated. PROTO has the bug
   (B013): `stores/app-user-store.ts:112-116` posts `{ userId, newPassword
   }`; `handler/user_handler.go:986` decodes `json:"appUserId"`, so the
   id is 0 and every admin "Change Password" answers 400.
3. Empty load renders a blank form. `pages/example-edit-page.tsx:62-87`:
   `readExampleById` returns `null` on 404/400 (`examples-store.ts:25`);
   the page only sets loading false (`:86`), passes the loading return
   (`:100-102`) and renders the edit form for a record it never loaded.
4. Store throws, page notifies. The load `.catch` (`:82-84`) only calls
   `reportError`, which writes to the console; the user sees nothing.
5. Zustand `set` hides misspelled keys. Not stated; `set` takes a
   `Partial<T>`-shaped union, so TypeScript skips the excess-property
   check and a typo is silently a new key.
6. Reload on route-id change. The `hasFetched` one-shot (`:57-61`) never
   resets, so if the router reuses the page for another `?exampleId=`,
   the old record stays on screen.

#### What GRC changed

GRC `src/web-app/README.md:106-108` adds `stores/<entity>-store.test.ts`
to the copy list ("vitest in the default node environment, `fetch`
stubbed, no DOM"). `:120-122`: the early returns end with "a load that
came back empty → an `<Alert />`. A form is never rendered for a record
that was not loaded." `:136-139`: "The store throws, the page tells the
user ... `notifications.show` and ... `reportError`." RG0008 md `:183`
records the zustand `Partial<T>` trap. RG0036 md `:28-38` is the
`userId`/`appUserId` root cause; its test is
`stores/app-user-store.test.ts:39`. RG0034 md `:66-71`: a Create that
navigated to the new id showed a blank step because the form reloaded
only once; "the form now records which route id it was loaded for".

#### Proposed fix for PROTO

1. `src/web-app/README.md` house pattern: add the six rules above, one
   bullet each, in PROTO's voice; add a store-test row to the copy table.
2. Ship B013's fix and its fetch-stubbing vitest as the example store
   test (`tests/app-user-store.test.ts`; PROTO keeps tests in `tests/`,
   not beside the store, and runs vitest under jsdom, not node).
3. Make `example-edit-page.tsx` obey rules 3, 4, 6, since it is the copy
   source (memory: entity pages are deliberate copy-paste). Apply the same
   edit to `app-user-edit-page.tsx` and `app-user-group-edit-page.tsx`.
4. Leave out GRC's `constants/grc-routes.ts`, `grcLink`, and the `App.tsx`
   name (PROTO's is `app.tsx`).
5. Tests: one vitest per rule-3/4 change is optional; the store test in
   step 2 is required.

#### Open question for Brandon

None.

### B077 — Security architecture doc for the template

- **Priority:** P3  **Area:** docs, infra
- **GRC:** `doc/requirements/20-architecture.md:22,24-26,33,40,58,60,186`; `doc/requirements/22-threat-model.md:85-131,143-150,156-160,214-217,227-238,246-253,306-309,320-324,330-344,349-359,366-368`; `doc/331-prd-postsales-cutover.md:283-290`   **PROTO:** new `doc/110-security-architecture.md` (or doc/105 Security `:151-225`); `doc/900-database-guidelines.md:1-43`; `doc/105-api-srv-tech-spec.md:7-18`; `infra/README.md:185-187`
- **Ticket:** GRC spec (A-002, A-004, A-038, A-047, A-049; C-1..C-14)
- **What:** App-layer isolation (no RLS) so a cross-tenant test per table;
cross-tenant 404 never 403; STRIDE for B1/B2/B7 (T-9 `mfa_exempt` humans, T-11
3-char CSRF); testable controls; pull-based backup target.
- **Port notes:** Cite `scoped_query.go`; mark controls PROTO meets; A-047 vs
PS0016 UUID.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

PROTO has security mechanisms but no written model of what they defend.
`doc/105-api-srv-tech-spec.md:151-225` lists controls (auth, RBAC, the
per-request checks, headers); multi-tenancy is five lines (`:221-225`):

    - **Data Segregation**: All queries filter by `security_context`

Nothing says isolation is enforced only in application code (no Postgres
RLS), so nothing says what follows: each tenant table needs a
cross-tenant negative test, and a missed predicate has nothing behind it.
`repository/scoped_query.go:8-24` encodes the rule for list reads, but
writes and `EXISTS` checks do not use it, and another tenant's id answers
409, which leaks existence (B012). Other gaps a model would surface:
`mfa_exempt` is one column an admin can set through the API
(`repository/user_repository.go:273-286`); the CSRF token is the session
id's last 3 characters (`handler/auth_handler.go:354,488,646`); the prd
backup pushes to `REMOTE_SERVER` by `scp` (`infra/README.md:185-187`), so
the primary holds credentials to its own backups. Docs gap; the derived
app inherits the template's risks without a list of them.

#### What GRC changed

GRC wrote a product spec, not a template doc. `doc/requirements/20-architecture.md:22`
(A-002): "Isolation is enforced at the application layer only ... **No
Postgres row-level security**"; `:24-26`: "Every new tenant-scoped table
must ship with a cross-tenant negative test or CI fails". A-004 `:33` no
hard delete; A-047 `:58` integer keys, never UUIDs; A-049 `:60` enums as
text; A-038 `:186` pull-based backup. `22-threat-model.md:85-131` sets
trust boundaries B1-B7 as a table; STRIDE rows include T-9 `mfa_exempt`
(`:144`, residual **H**), T-11 3-char CSRF (`:146`), T-14 403-vs-404
enumeration (`:149`), T-2 missing predicate (`:156`). `:227-238` and
`:246-253` state the consequences; C-1, C-4, C-6, C-7, C-9..C-11, C-14
(`:306-309,320-324,330-344,349-359,366-368`) are testable controls.
Correction: GRC's own backup is push too (`prd-grc-backup-all:46`, `scp`
to `REMOTE_SERVER`; `doc/331-prd-postsales-cutover.md:283-290`), so
A-038 is a GRC intent, not GRC practice. No tests pin the doc.

#### Proposed fix for PROTO

1. New `doc/110-security-architecture.md` (doc/105 keeps the mechanics,
   links to it). Sections: boundaries B1 (internet to API), B2 (API to
   Postgres), B7 (operations) only; B3-B6 are GRC features (vault,
   worker, external users, support, blobs).
2. Isolation rule: app-layer only, cite `scoped_query.go`; cross-tenant id
   answers 404, never 403/409 (lands with B012); one negative test per
   tenant table (`tests/test_security_context_isolation.py`).
3. A STRIDE table for B1/B2/B7, each row marked "met / partial / open"
   against PROTO code: T-9 (`mfa_exempt`), T-11 (3-char CSRF, B004),
   T-14 (B012), T-2, T-43/A-038 (push backup).
4. Controls adapted from C-1, C-7, C-9, C-10; drop C-6's magic links
   unless B035 lands; drop C-14 unless B067 lands.
5. A-047 vs PS0016: say PROTO keeps integer keys and shows a UUID column
   only as an attribute (`0002_example_jsonb_uuid.sql:1-3`).
6. Link from `CLAUDE.md` Conventions and `doc/900`. Tests: none new.

#### Open question for Brandon

Should the template doc set a rule that `mfa_exempt` is forbidden on
human accounts in production (GRC C-7)? Recommendation: yes, stated as
policy now, enforced by a test once B010/B018 land.

### B078 — Testing hygiene rules

- **Priority:** P3  **Area:** docs, tests
- **GRC:** `tickets/03-done/RG0037-batch-gate-stale-checks/RG0037-batch-gate-stale-checks.md:26-29,53-58,95-97`; `tickets/03-done/RG0021-entra-check-runtime/RG0021-entra-check-runtime.md:335,346,507`   **PROTO:** `tests/README.md:47-58`; `src/api-srv/README.md`; `bin/300-run-tests:24-27,74-78`
- **Ticket:** RG0037 / `4ce1484`; RG0021
- **What:** A "not in Y" assertion needs proof Y was read; macOS path for OS
probes; anchored detectors; gates fail, not skip; `<X>_LIVE=1` for live tests.
- **Port notes:** "Rebuild before pytest" is SKIP: bin/300 already rebuilds.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

PROTO has no written rules for how a test proves something. The pieces
that exist are code, not guidance:

- `bin/300-run-tests:24-27` explains why Go integration suites may not
  skip, and `:74-78` greps for `connectTestDB`'s exact skip message and
  fails the run. That is the "gates fail, not skip" rule, applied to one
  message only, and written down only in this script.
- `tests/README.md:47-58` is a file list; it has no "Writing tests"
  section. `src/api-srv/README.md` has no testing section at all (its
  headings stop at Architecture, `:128`).

Nothing tells the next author that a negative assertion ("the token is
not in the command line", "no secret in the log") must prove it read the
thing it checked; that a probe of OS state must work on macOS as well as
Linux; that a text detector must be anchored; or how to keep a test that
needs a live outside service out of the gate. PROTO has no such test
today (no `/proc` probes, no `_LIVE` tests), so this is a quality/DX gap
that bites the first derived app that writes one.

#### What GRC changed

GRC wrote these down in tickets after a batch gate failed for reasons the
batch did not cause. RG0037 md `:26-29`: a pwsh test read
`/proc/$PID/cmdline`, which macOS lacks; it was "built on the Entra
machine". The fix (`:53-58`) reads `ps -o command= -p $PID` off Linux and
makes `runner_test.go` "also fail if the command-line row is empty.
Without that, a read that silently failed would pass the 'token is not in
it' check without having looked." `:95-97`: the smoke detector anchors
"No such" to the start of a line "so data inside a row cannot trip it".
RG0021 md `:335`: `bin/340` "fails (rather than skips) without pwsh, and
fails if a subprocess test skips while pwsh is installed"; `:346`:
`TestLiveTokenAgainstTheTenant` "skips unless `ENTRA_LIVE=1`, so the gate
never depends on the tenant, the network or a consent". Commit `4ce1484`.
The rules live in ticket bodies, not in a GRC README.

#### Proposed fix for PROTO

1. Add a "Writing tests" section to `tests/README.md` (pytest) and a
   short "Testing" section to `src/api-srv/README.md` (Go), five bullets:
   - A "not in Y" assertion first asserts Y is non-empty.
   - An OS probe has a macOS path and a Linux path (dev is macOS, hosts
     are Linux).
   - Text detectors are anchored (`^`, word boundary), never bare
     substrings over rendered data.
   - A gate fails when a prerequisite is missing; it never skips. Cite
     `bin/300-run-tests:74-78` as the pattern.
   - A test that needs a live external service skips unless
     `<SERVICE>_LIVE=1`, and the gate never sets it.
2. Optional: generalise `bin/300:74-78` to fail on any Go `--- SKIP`
   whose message is not on an allow list.
3. Do not port "rebuild before pytest" (RG0021 `:507`): `bin/300 py`
   already builds a fresh binary (`:165-167`).
4. Do not copy GRC's pwsh, Entra or smoke specifics. Tests: none.

#### Open question for Brandon

None.

### B079 — Process rules: batched testing; readability over DRY

- **Priority:** P3  **Area:** docs, skills
- **GRC:** `CLAUDE.md:85-88,129-134`; `.claude/skills/ticket-workflow/SKILL.md:110,121-129`; RG0037 md `:19-22,30-35`; `tickets/04-archive/engineering-log/260924_0740-TKT-NONE-round-11-on-main.md:57-62`; RG0010 md `:85`; `…/260922_0114-TKT-RG0010-agent-guidance-consolidation.md:30-39`   **PROTO:** `CLAUDE.md:51,56-74`; `.claude/skills/ticket-workflow/SKILL.md:99-103`; `tickets/04-archive/PS0005-agent-guidance-consolidation.md:30`
- **Ticket:** `77e1283`, `090f350`, `a605945` (Brandon 2026-09-22); RG0010 / `fbdb9f3`
- **What:** (a) Per ticket run only changed-layer units; reviewer runs the full
gate per batch (RG0037: a test never run). (b) Readability, maintainability,
speed; tolerate duplication; reviewable parts (not covered by the global
file).
- **Port notes:** Brandon's call on both; drop GRC's 440/106.
- **Decision:** \[ \] pull / \[ \] adapt / \[X\] skip  (superseded by B091, ticket system removed, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Two process rules; neither is a bug, both are policy.

(a) Per-ticket testing. `.claude/skills/ticket-workflow/SKILL.md:99-103`
makes every ticket run the full gate before asking for the merge:

    cd src/api-srv && go build ./...
    cd src/web-app && npx tsc --noEmit -p tsconfig.app.json
    bin/300-run-tests all   # rebuilds the scratch DBs, then go + playwright + pytest

That is right for a template with few tickets in flight. It gets slow
when many small tickets land in a day, which is what pushed GRC to batch.

(b) Readability over DRY. `CLAUDE.md:51-54` keeps naming, line length,
config and git rules, but nothing on how to trade readability against
abstraction. PROTO used to say it: the deleted `doc/903-roles-and-goals.md`
(last at `d2fa20f`) read "Optimize for readability, maintainability, and
then speed ... We can tolerate some code duplication until we verify it
the use cases are similar enough" (sic). PS0005 deleted it as "already
covered by the global `~/.claude/CLAUDE.md`"
(`tickets/04-archive/PS0005-agent-guidance-consolidation.md:30`).
Correction: the global file covers tone and brevity, not this; a grep of
it for "DRY", "duplicat" or "readab" finds nothing. The rule was lost.
The memory note "web pages are scaffolding" covers only web pages.

#### What GRC changed

(a) Brandon, 2026-09-22 (`77e1283`, `090f350`, `a605945`): GRC
`CLAUDE.md:129-134` and `ticket-workflow/SKILL.md:121-129`: a ticket runs
only its changed layer's unit checks and "**writes** its pytest cases
without running them"; the reviewer runs the full gate "once per batch of
merges", and the ticket's Checks section says "deferred to the batch run".
The cost showed in RG0037 md `:19-22,30-35`: the first batch gate failed
three checks, one a pytest "written in RG0033/RG0034 and never run".

(b) RG0010 (`fbdb9f3`) kept doc/903's content as two bullets, GRC
`CLAUDE.md:85-88`: "Readability first, then maintainability, then speed.
DRY unless the abstraction hurts it; tolerate duplication until the cases
prove similar" and "produce it in reviewable parts". The engineering log
(`260922_0114-...md:30-39`) records why: a rule an agent "can be held to"
beats a role line. No tests for either.

#### Proposed fix for PROTO

1. (b) Add GRC's two bullets to `CLAUDE.md` Conventions after `:51`
   verbatim; they carry no GRC names.
2. (a) If adopted: rewrite `SKILL.md:99-103` into "per ticket: changed
   layer only (`bin/300-run-tests go` or `unit`), write new pytest cases"
   and "batch gate: `bin/300-run-tests all`, run by the reviewer before
   main advances"; add one line to `CLAUDE.md` Tickets. Drop GRC's
   `bin/310`/`320`, smoke and the 440/106 flag set; PROTO has none.
3. (a) Close RG0037's hole: a ticket that writes a pytest case runs
   `bin/300-run-tests py` (pytest only, its own API) even when the full
   gate is deferred. Never "written, not run".

#### Open question for Brandon

Adopt batched testing in the template, or keep `bin/300-run-tests all`
per ticket? Recommendation: keep per ticket in the template, since PROTO
has no 440-check smoke to make it slow; a derived app can batch.

### B080 — CLAUDE.md structure rules and small conventions

- **Priority:** P3  **Area:** docs
- **GRC:** `CLAUDE.md:21-25,33-34`; `src/api-srv/README.md:223-224`; RG0043 md `:806-829`; `tickets/03-done/RG0027-pdf-of-record-required/RG0027-pdf-of-record-required.md:134`; `tickets/03-done/RG0009-split-app-lib-by-role/RG0009-split-app-lib-by-role.md:144-150`; `tickets/02-working/RG0033-evidence-capture-set-design/RG0033-evidence-capture-set-design.md:183`   **PROTO:** `CLAUDE.md:11-12,18-24`; `doc/900-database-guidelines.md`; `doc/105-api-srv-tech-spec.md`; `src/api-srv/internal/handler/*.go` (33 `app :=` sites)
- **Ticket:** RG0006; RG0013; RG0043 1.1; RG0027; RG0009; RG0015 ruling 3; RG0033
- **What:** `cmd/` only shipped binaries; Layout links doc/902 and runbooks;
fixtures never in migrations; 409 when record state refuses; `app :=` shadows
package `app` (33 sites); pin deps that raise `go`.
- **Port notes:** One docs ticket; the `app` rename is code.
- **Decision:** \[ \] pull / \[X\] adapt / \[ \] skip  (per recommendation, minus every ticket rule (B091), 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Six small rules; five are doc gaps, one is a code smell (DX, not a bug).
1. `cmd/` holds only shipped binaries: already true (`cmd/` is `migrate`,
   `server`; the load tester is `src/load-test`) but unwritten.
2. `CLAUDE.md:11-12` lists `bin/` ranges and `:25` says only "`doc/` -
   reference docs"; nothing points at `doc/902` (script list) or the
   runbooks `doc/320`/`330`/`340`.
3. Fixtures never in migrations: PROTO already obeys it (no `INSERT` in
   `sql/migrations/`) and `doc/900-database-guidelines.md:60` says so. The
   doc entry overstates the gap; only a `CLAUDE.md` line is missing.
4. `doc/105-api-srv-tech-spec.md:268` defines 409 only as "Conflict
   (optimistic locking)", so a refusal driven by record state has no rule.
5. 33 handler sites shadow the imported package `app`:
   `handler/app_user_group_handler.go:10` imports `.../internal/app`;
   `:35` has `app := security.NewAppHelper(w, r, h.appsrv, ...)`. It
   compiles only because `*app.Services` appears outside function bodies.
6. `src/api-srv/go.mod:3` is `go 1.25.4`; one `go get` can silently raise it.

#### What GRC changed

- `CLAUDE.md:21-25` links `doc/902` and names its runbooks; `:33-34` says
  "`cmd/` holds only shipped binaries, RG0006" (also README `:223-224`).
- RG0043 item 1 (`:806-829`): shipped seed migrations had created
  `admin`/`admin123!` on every host; GRC commented them out and moved the
  logins to a test-DB fixture.
- RG0027 `:133-135`: "409 rather than 400 because the request is well
  formed and it is the state of the record that refuses it."
- RG0009 `:143-149` found the shadow; RG0015 ruling 3 renamed all sites to
  `appHelper` via `gopls rename`, no behaviour change.
- RG0033 `:183-184` pinned pdfcpu v0.15.0 to avoid `go 1.26`. GRC's
  `src/go.mod:3` is now `go 1.26.0` anyway, by a later deliberate bump.

#### Proposed fix for PROTO

1. `CLAUDE.md` Layout: `bin/` line names `doc/902`; `doc/` line names the
   runbooks; `src/api-srv` line adds "`cmd/` holds only shipped binaries".
2. `CLAUDE.md` Conventions: "fixtures in `sql/fixtures/`, never in a
   migration"; "pin a dependency below a release that raises `go`, unless
   the bump is the point".
3. `doc/105:268`: 409 also when record state refuses a valid request.
4. Code: `gopls rename` every `app :=` in `internal/handler/*.go` to
   `appHelper`; rewrap past 80 cols. Separate commit.
5. Traps: PROTO's module is `src/api-srv/go.mod`, not `src/go.mod`; leave
   out goose, `migrate-srv`, `go.work`. Tests: none new; `go build`.

#### Open question for Brandon

Should the 33-site `appHelper` rename ride this docs ticket or its own?
Recommend its own, so the mechanical diff does not bury the doc lines.

### B081 — DERIVING.md: paste-ready "Template lineage" block

- **Priority:** P3  **Area:** docs
- **GRC:** `CLAUDE.md:14-17,156-173`   **PROTO:** `DERIVING.md:108-118,126-127`
- **Ticket:** `59d351c`
- **What:** GRC wrote the section DERIVING asks for: derive commit, last tag and
range, three `git` commands, PS id in `raised_by:`.
- **Port notes:** Add to DERIVING §8.
- **Decision:** \[ \] pull / \[X\] adapt / \[ \] skip  (per recommendation, lineage block must not cite tickets/ (B091), 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Docs/DX gap. `DERIVING.md` §8 tells a derived app how to pull later
template changes with generic commands, `DERIVING.md:108-116`
(`git log template-<last-sync>..template/main -- tickets/04-archive`).
It then says only (`:126-127`): "record the template tag you synced to in
your own `CLAUDE.md` or a ticket". It gives no shape for that record, and
§1 (`:13-34`) never says to note the commit the app was derived from. §5
(`:70-75`) says the `CLAUDE.md` Conventions/Tickets/Branches sections
"carry over" but names no lineage section, and says `DERIVING.md` "can
go". A derived app that deletes it loses the starting point
and the commands; its first sync starts with archaeology.

#### What GRC changed

GRC commit `59d351c` (2026-09-21) wrote that record into its `CLAUDE.md`:
- `:14-17`: "Derived from ProtoStack-ReactGo (`<remote>`) at its commit
  `0c2130a` on 2026-09-17", plus which local decisions override template
  conventions.
- `:156-173`, "## Template lineage": the last reviewed tag
  (`template-2026-09-21`), the template ticket range it covered and the
  local ticket range it became, the three `git` commands with the real tag
  filled in, and the rule: decide pull/adapt/skip per archive file, file a
  local ticket with the template id in `raised_by:`, never cherry-pick
  blind (module path, binaries and migrator differ), then update the tag.

#### Proposed fix for PROTO

1. `DERIVING.md` §1: after the `bin/000-rebrand` commit, record
   `git rev-parse --short HEAD~1` (the template commit) and the date.
2. `DERIVING.md` §8: add a fenced, paste-ready "## Template lineage" block
   with placeholders `<template-remote>`, `<derive-sha>`, `<derive-date>`,
   `<last-tag>`, `<template-id range>`, the three commands, and GRC's
   pull/adapt/skip paragraph. Replace `:126-127` with "paste it into
   `CLAUDE.md` and update `<last-tag>` after each sync".
3. `DERIVING.md` §5: list "Template lineage" among sections the derived
   `CLAUDE.md` must have, before `DERIVING.md` is deleted.
4. Trap: `bin/000-rebrand` rewrites every `PS` + four digits in tracked
   files outside `tickets/` (step 3, `--ticket-prefix`). A literal
   `PS0001-PS0021` in the block becomes `AC0001`. Use placeholders only.
5. Trap: the §7 check (`DERIVING.md:89-91`) greps for `wbmartin`; a
   literal template remote in the block makes that check print. Keep
   `<template-remote>` as a placeholder, as `:110` already does.
6. Do not copy GRC ids, goose wording or sha. Test: §7 grep after rebrand.

#### Open question for Brandon

Should `bin/000-rebrand` print the derive sha and pre-filled lineage block
in its "Left for a human" list? Recommend yes: the sha is known at that
moment and nowhere else so cheaply.

### B082 — Later post-sales docs: repository reset; release checklist

- **Priority:** P3  **Area:** docs, skills
- **GRC:** `tickets/01-open/RG0044-fresh-repo-webhooks-deploy-keys.md:32-51,53-82,84-90` (:66-68); RG0043 md `:549-564,566-620,632-665,687-739,771-782`   **PROTO:** new `doc/350-repository-reset.md` or `infra/README.md:138-188`; `DERIVING.md:46-55`; new `.claude/skills/release/SKILL.md` or `infra/README.md` after `:122`
- **Ticket:** RG0044 (open, not run); RG0043 §10 (shipped 2026-10-02)
- **What:** What hangs off a GitHub repo and how to re-root it. A release record
(pre-flight, approvals, UAT, prod, rollback, hash-checked host edits) as a
checklist.
- **Port notes:** Wait for RG0044's run; strip hosts and hashes; after B057.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Docs gap: two post-sales procedures a derived app will need are unwritten.
1. **Repository reset.** Several things hang off a GitHub repository and do
   not move with it: the UAT build checkout's read-only deploy key, the
   UAT build webhook (payload URL plus `HOOK_SECRET`), the prod promote
   webhook, collaborators and branch protection. PROTO documents setting
   them up piecemeal (`infra/README.md:138-187`, step 7 is hooks), but not
   how to re-root a repo (one fresh root commit, to purge data that leaked
   into history) without breaking deploys; `DERIVING.md:46-55` is silent.
2. **Release checklist.** `infra/README.md:119-122` describes the restart
   mechanics and `doc/320-presales-uat-deploy.md:111` has a presales
   rollback, but there is no post-sales release record: pre-flight,
   approvals, UAT, prod, rollback targets, and how to move a hand-edited
   host file safely.

#### What GRC changed

- **RG0044** (`tickets/01-open/RG0044-fresh-repo-webhooks-deploy-keys.md`,
  status `ready`, not yet run). `:32-51` lists what must be re-created
  (deploy key, two webhooks, settings). `:53-82` is the procedure: record
  hooks and key, `git checkout --orphan` a single root commit naming the
  old sha, rename the old repo and recreate the same name (keeps `REPO_URL`
  valid), re-create hooks before the first push, re-clone everywhere,
  delete the old repo after one release. `:84-90` are the checks.
- **RG0043 §10** (shipped 2026-10-02) is the release record: what ships by
  layer (`:549-564`), pre-flight with literal paths and recorded rollback
  shas (`:566-620`), UAT steps (`:632-665`), prod steps where a host file
  is moved with `shasum -a 256` locally, `sha256sum` and `diff` on the
  host, `.before` copy, `nginx -t`, reload, `curl` (`:687-739`); rollback
  per host (`:771-782`).
- Correction: RG0044 `:66-68` says a `main` push builds UAT and `uat`
  promotes; GRC `CLAUDE.md:23-24`, RG0043 P1 `:578-590` say the reverse.

#### Proposed fix for PROTO

1. Wait for RG0044 to run; copy what it proved, not what it planned.
2. New `doc/350-repository-reset.md`: GRC's list and steps with every host
   name, hook id, sha and org path replaced by `<env>-<slug>` names from
   `etc/app.env`. Link from `DERIVING.md` §3 and `infra/README.md` step 7.
3. Release checklist: a template section in `infra/README.md` after `:122`
   (or a `release` skill): P/A/U/R/rollback skeleton, hash-checked
   host-file move, none of RG0043's content.
4. Traps: GRC's branch model is B057 (documented, not adopted). Drop hook
   ids and real hostnames.
5. Tests: none; B053 linter checks the new paths. Depends on B057.

#### Open question for Brandon

Release checklist as a doc section or a skill? Recommend a doc section
first; make it a skill once two derived apps have used it.

### B083 — Serving user-uploaded files: header override, sniff by bytes

- **Priority:** P3  **Area:** docs
- **GRC:** `tickets/03-done/RG0026-file-routes-embeddable/RG0026-file-routes-embeddable.md:24`; `tickets/03-done/RG0023-policy-document-tab/RG0023-policy-document-tab.md:166`; `tickets/03-done/RG0024-evidence-artifact-file-route/RG0024-evidence-artifact-file-route.md:75,116`   **PROTO:** `doc/105-api-srv-tech-spec.md` \~\`:214`;` src/api-srv/internal/security/security\_headers.go:13-34`;` DERIVING.md\`
- **Ticket:** RG0023, RG0024, RG0026
- **What:** Override XFO/CSP for inline PDFs; cap declared and actual size
(413); type from magic bytes (415); `private, no-store`; every miss 404.
- **Port notes:** Docs until a blob route exists (PS0018 avatars); then P2.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Nothing is broken yet. PROTO has no file route; PS0018 dropped the avatar
because "the template has no file storage"
(`tickets/04-archive/PS0018-user-profile-page.md:19-20`). This is a DX gap:
the first derived app to serve a file meets traps nothing documents.
`HeadersMiddleware` forbids framing on every API response:
```go
// src/api-srv/internal/security/security_headers.go:21-22, 34
"object-src 'none'; "+  "frame-src 'none'; "+
w.Header().Set("X-Frame-Options", "DENY")
```
A PDF served that way cannot render in an `<object>`/`<iframe>`; Chrome
shows a broken-document icon. nginx's `/api/` location has no `add_header`,
so it inherits server-level `X-Frame-Options "DENY"`
(`infra/uat/etc/nginx_uat-proto.example.com:34,40`); a handler override to
`SAMEORIGIN` would arrive beside a second `DENY`, which browsers treat as a
block. `doc/105-api-srv-tech-spec.md:211-219` says the headers apply to
"All Responses" and says nothing on judging uploads, so a naive route
trusts the client's `Content-Type`, letting the uploader pick the type.

#### What GRC changed

RG0023/24/26 put the rules in `src/internal/handler/upload_artifact.go`:
```go
// upload_artifact.go:286-288 (RG0026; a handler Set beats the middleware)
func allowSameOriginEmbed(w http.ResponseWriter) {
    w.Header().Set("X-Frame-Options", "SAMEORIGIN")
    w.Header().Set("Content-Security-Policy", "frame-ancestors 'self'")
```
Each file route calls it, then sets `Cache-Control: private, no-store`
(`evidence_file_handler.go:171-172`). Size: declared `Content-Length`, then
`MaxBytesReader`, then bytes read, over 20 MiB is 413 (`:226-258`). Type
from magic bytes (`%PDF-`, PNG/JPEG, DOCX zip; `:95-97,162-186`), else 415;
the stored type is re-sniffed on the way out (`evidence_file_handler.go:150`).
`safeFileName` (`:189`) cleans the name; every miss is 404. Tests:
`upload_artifact_embed_test.go:15,38` (embed override; default still DENY).

#### Proposed fix for PROTO

1. Docs now: a "Serving user files" subsection after
   `doc/105-api-srv-tech-spec.md:219` with the five rules, the nginx
   `DENY` trap, SPA CSP needs `object-src`/`frame-src 'self'`; DERIVING link.
2. Code (P2) when a blob route lands. Leave out GRC's `GRC_*` error codes,
   its PDF/DOCX/OLE2 set, evidence `LogRead`, and tombstone 404s.
3. Depends on B022. Tests then: Go override vs default `DENY`; 413/415.

#### Open question for Brandon

None.

### B084 — Skills and tickets: small fixes

- **Priority:** P3  **Area:** skills, tickets
- **GRC:** `.claude/skills/ticket-workflow/SKILL.md:104-106,142`; `scripts/lint-tickets.py:154`; `tasks/todo.md:143-304`; `.claude/skills/handoff/SKILL.md:42,67-68,71`   **PROTO:** `.claude/skills/ticket-workflow/SKILL.md:100-101,116`; `scripts/lint-tickets.py:87,154`; `.claude/skills/handoff/SKILL.md:3,42-45,67-68,71-73`; `tasks/` (absent)
- **Ticket:** RG0006 / `719739b`; RG0043 / `5bd0fd4`; `59d351c`; `8d5fd52`, `9e234e3`, `8e1b197`, `a25f75b`
- **What:** Web checks use `tsc -b` and judge lint by non-max-len delta; "or a
PR Brandon merges" reconciles skill with CLAUDE.md; lint message hardcodes
"PS" (`ID_PREFIX` :87); handoff's `tasks/todo.md §0` has no file.
- **Port notes:** s/uat/main/; handoff creates `tasks/todo.md` if absent; skip
GRC's rounds.
- **Decision:** \[ \] pull / \[ \] adapt / \[X\] skip  (superseded by B091, skills and tickets removed, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Four small DX gaps in ticket tooling; none touches the shipped app.

1. **Web check misses a config and lint.**
   `.claude/skills/ticket-workflow/SKILL.md:101` runs
   `npx tsc --noEmit -p tsconfig.app.json`, and `tsconfig.app.json:28`
   excludes `vite.config.ts` (only `tsconfig.node.json:25` covers it). No
   step runs ESLint, `bin/300-run-tests` included. Today `npx eslint .`
   and `npx tsc -b --noEmit` are both clean, so the gap is latent.
2. **Merge path disagrees.** `CLAUDE.md:79` says "pull requests target
   `main`, and only Brandon advances `main`"; the skill (`SKILL.md:108-116`)
   has the agent merge and push after Brandon's yes, with no PR option.
3. **Hardcoded prefix.** `scripts/lint-tickets.py:87` sets `ID_PREFIX`
   for `bin/000-rebrand`, but `:154` prints `is not PS followed by four
   digits`; a derived app's message names the wrong prefix.
4. **Missing file.** `.claude/skills/handoff/SKILL.md:3,66-68` routes loose
   work and decisions to `tasks/todo.md` "§0"; PROTO has no `tasks/`.

#### What GRC changed

- RG0006 (`719739b`): `SKILL.md:106` is
  `npx tsc -b && npm run lint   # judge lint by the non-max-len delta`.
  The delta exists because GRC's eslint has a 200-col `max-len` error with
  old violations (`src/web-app/eslint.config.js:37`). PROTO has no
  `max-len` rule, so plain pass/fail fits PROTO.
- RG0043 (`5bd0fd4`): `SKILL.md:142` adds "(Or the same as a PR into
  `uat` that Brandon merges.)".
- `59d351c`: `lint-tickets.py:154` uses `{ID_PREFIX}`. No tests pin these.
- Correction: GRC did not change the handoff skill (same text at
  `:42,67-68,71`). It just has `tasks/todo.md`, grown by `8d5fd52`,
  `9e234e3`, `8e1b197`, `a25f75b` into dated "Round N" logs
  (`tasks/todo.md:143-304`) that break the skill's rule at `:71`.

#### Proposed fix for PROTO

1. `SKILL.md:101`: `cd src/web-app && npx tsc -b && npm run lint`; drop
   GRC's "delta" comment. Eslint in `bin/300` is B041.
2. `SKILL.md:116`: add "(Or the same as a PR into `main` that Brandon
   merges.)". Trap: GRC says `uat`; PROTO's target is `main` (B057).
3. `scripts/lint-tickets.py:154`: `is not {ID_PREFIX} followed by ...`.
4. Handoff skill: create `tasks/todo.md` with `## 0. Waiting on Brandon`
   if absent. Do not copy GRC's file or rounds.
5. Test: `python3 scripts/lint-tickets.py` passes before and after.

#### Open question for Brandon

Is `tasks/todo.md` committed (handoff assumes so) or transient, per your
global "remove completed tasks/todo.md"? Recommend: commit it, queue only.

# Bugs found in PROTO along the way

### B085 — READMEs say `bin/510` builds the local database

- **Priority:** P2  **Area:** docs
- **GRC:** `bin/205-migrate-db:1-17`; `src/web-app/README.md:12-18` (GRC's answer)   **PROTO:** `README.md:46`; `src/api-srv/README.md:9-12`; `bin/510-reset-test-db:43-45`; `doc/050-dev-env-init.md:21-31`
- **Ticket:** none
- **What:** Both READMEs send newcomers to `bin/510`, which refuses non-scratch
names; the dev DB is built per doc/050.
- **Port notes:** Point at doc/050; dev migrate script is B073.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Docs bug in onboarding. `README.md:46` ("`./bin/510-reset-test-db` |
Rebuild the local database") and `src/api-srv/README.md:9-11` ("Build the
local database: `bin/510-reset-test-db`") send newcomers to `bin/510`. So
do `doc/105-api-srv-tech-spec.md:343` and `doc/060-dev-env-start.md:17-19`,
which the item's refs miss.

`bin/510` builds a scratch database only: no argument means
`loc_<slug>_test_db` (`bin/510-reset-test-db:39`), and `:43-45` refuses
any name not ending `_test_db`/`_e2e_db`. The API connects to
`loc_proto_db` (`etc/loc-api-srv.env:7`). Following the README literally:
`bin/510` succeeds and builds `loc_proto_test_db`; `bin/215-run-api-srv`
then fails `Ping` and exits with `DBP0 Failed to ping database`
(`src/api-srv/cmd/server/main.go:63-67`); passing `loc_proto_db` to
`bin/510` is refused. Pointing `DATABASE_URL` at the test database is no
escape, because `bin/300 go` drops it every run
(`doc/902-project-commands.md:66`). The working steps are the "Manual
path" in `doc/050-dev-env-init.md:20-33`, which no README links.

#### What GRC changed

GRC added a dev-database script and points its docs at it.
`bin/205-migrate-db:3-4,16-18` bootstraps database and roles
idempotently, runs `up`, and refreshes the schema reference.
`src/web-app/README.md:12-15` says the app needs "the tenant seeded
(`doc/050-dev-env-init.md`)" and runs `./bin/205-migrate-db &&
./bin/206-seed-dev-tenant` once; `doc/105-api-srv-tech-spec.md:599`
agrees. `bin/510` stays scratch-only (`src/api-srv/README.md:217`). No
tests. The script wraps goose and `ADMIN_DATABASE_URL`, which PROTO lacks.

#### Proposed fix for PROTO

1. `README.md:46`: relabel "Rebuild a scratch test database"; add a line
   above the table: "First-time dev database: `doc/050-dev-env-init.md`."
2. `src/api-srv/README.md:9-12`, `doc/105:343`, `doc/060:17-19`: replace
   the `bin/510` step with a pointer to doc/050's Manual path.
3. When B073 adds a PROTO dev-migrate script (`cmd/migrate up` plus
   `local-seed.sql`), point all four places at it. Do not copy GRC's
   `bin/205`: it calls goose, `migrate-srv` and `bootstrap`.
4. Write `loc_<slug>_db`, not `loc_proto_db`, so `bin/000-rebrand` has
   nothing to rewrite.
5. Tests: none; B053's doc-path check guards future drift.

#### Open question for Brandon

Fix the pointers now, or wait for B073's script? Recommend now: it is
minutes of work and the current text sends every newcomer into a
startup failure.

### B086 — doc/060 names a script that does not exist

- **Priority:** P3  **Area:** docs
- **GRC:** `doc/060-dev-env-start.md:1-43`   **PROTO:** `doc/060-dev-env-start.md:7` (`./515-run-postgres.sh`), `:31` (absolute `~/Code/...` path)
- **Ticket:** none
- **What:** `bin/515` is `reseed-db`; no `515-run-postgres.sh` exists.
- **Port notes:** Fix now or in B075.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Docs/DX gap. The daily-start page starts Postgres like this
(`doc/060-dev-env-start.md:5-7`):

```
cd path/to/infrahub/loc-postgres  # your local infrahub checkout, outside this repo
source ./brandon_env_setup-mac1.sh
./515-run-postgres.sh
```

The item says "no `515-run-postgres.sh` exists"; that overstates it. The
script is not in PROTO, but `:5` says to `cd` into a separate "infrahub"
repo first, and that repo exists on Brandon's machine. Its script runs
`docker compose` to create container `loc-postgres` (Postgres 17, port
5432). Corrected fact: the page depends on a private repo and a personal
env file that nobody deriving the template has. The `515` prefix also
reads like PROTO's own `bin/515-reseed-db`, which misled the review.
A new developer cannot start Postgres from this page, yet the scripts
need a container with that exact name (`bin/510-reset-test-db:49`,
`doc/050-dev-env-init.md:4`).

`:31` runs `~/Code/ProtoStack-ReactGo/bin/220-run-web-app`, an absolute
path that breaks in any other checkout. `:17-19` is B085; no title.

#### What GRC changed

GRC rewrote the page under RG0011 (`06a1396`). `doc/060-dev-env-start.md`
gains a title and "One-time setup ... is `doc/050-dev-env-init.md`" at
`:1-3`, repo-relative script calls, a migrations note at `:15-16`, and
`curl -s http://localhost:8080/health` at `:22`. It did NOT fix the
database step: `:10-12` still runs `./515-run-postgres.sh` from the
infrahub repo, now via an absolute personal path. No tests.

#### Proposed fix for PROTO

1. `doc/060:3-15`: replace the infrahub block with a self-contained
   command that yields the expected container, e.g. `docker run -d
   --name loc-postgres -e POSTGRES_PASSWORD=<pw> -p 127.0.0.1:5432:5432
   postgres:17`, then `docker start loc-postgres` on later days.
2. `:31`: `./bin/220-run-web-app`. `:1`: add a title and GRC's doc/050
   pointer. `:17-19`: per B085.
3. Do not copy GRC's `:10-12` (personal path) or its worker, three-binary
   and `bin/211-stop-app` lines; PROTO has none of these.
4. Tests: none. Overlaps B075 (dev docs); do it there if B075 is taken.

#### Open question for Brandon

Infrahub or a plain `docker run` for derived apps? Recommend `docker
run`; a derived app's developers will not have infrahub.

### B087 — doc/902 health URL names another project's real host

- **Priority:** P2  **Area:** docs
- **GRC:** `doc/902-project-commands.md:40-42`   **PROTO:** `doc/902-project-commands.md:17`
- **Ticket:** none
- **What:** The `curl -k https://...:8443/health` line names a personal dev
host from another project (the same host as GRC's `TLS_NAME`).
- **Port notes:** Use `http://localhost:8080/health`.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Docs bug with a small disclosure side. `doc/902-project-commands.md:15-18`
says to check the local API with `curl -k https://<host>:8443/health`,
where `<host>` is a real personal dev hostname under another project's
domain (not repeated here). Three problems:

1. It is not this machine. Local dev is plain HTTP on 8080
   (`etc/loc-api-srv.env:26` `SERVER_PORT=8080`, `:30` `ENABLE_TLS=false`).
   The command fails DNS or, if the name resolves, asks a different real
   server whether it is up.
2. `-k` disables certificate checks, a habit the template should not teach.
3. The host is the same string as GRC's `TLS_NAME` in
   `etc/loc-api-srv.env:24` (checked by comparison). Every derived app
   inherits it.

The item misses a second place: `doc/310-api-srv-cert-renewal.md:2-12`
names the same host five times, plus an ssh alias for a VM of that other
project and an absolute `~/Code/ProtoStack-ReactGo/etc/crt/` path.
`tickets/04-archive/PS0001-backport-derivatives.md:19` and
`PS0005-agent-guidance-consolidation.md:31` also mention it, as history.

#### What GRC changed

GRC rewrote the line under RG0011 (`06a1396`).
`doc/902-project-commands.md:40-42`:

```
Is the API up? `curl -s http://localhost:8080/health` (the port is
`SERVER_PORT` in `etc/loc-api-srv.env`). Logs of a backgrounded server:
```

GRC did not clean its `doc/310`; it still names the host. No tests.

#### Proposed fix for PROTO

1. `doc/902:15-18`: GRC's sentence, `curl -s http://localhost:8080/health`,
   naming `SERVER_PORT` in `etc/loc-api-srv.env` as the source. Keep
   PROTO's `bin/216-stop-local-stack`, not GRC's `bin/211-stop-app`.
2. `doc/310`: replace host, ssh alias and absolute path with
   `<api-host>`, `<uat-vm>` and repo-relative `etc/crt/`. B021 (cert
   renewal) does not touch this page, so fix it here.
3. Grep the tree for the domain, excluding `tickets/04-archive/`; expect
   no hits.
4. Tests: none; optionally deny-list the domain in B053's checker.

#### Open question for Brandon

Scrub the two archived tickets too, or leave archive history as written?
Recommend leaving the archive and fixing only live docs.

### B088 — Broken comments: UAT env example and `.gitignore`

- **Priority:** P3  **Area:** etc
- **GRC:** n/a   **PROTO:** `etc/uat-api-srv.env.example:7` (ends "It intentionally omits"), `:38` (dangling `# --- logging / KV ---`); `.gitignore:23` (`bin/200-run-tests`)
- **Ticket:** none
- **What:** Cut sentence; header with no keys; wrong script name (`bin/300`).
- **Port notes:** Fix with B016/B041.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Quality gap: three comments outlived what they described. No runtime
effect; each misleads a reader.

1. **Cut-off sentence.** `etc/uat-api-srv.env.example:6-7` ends "...at
   $APP_ROOT/etc/api-srv.env). It intentionally omits" and stops. In
   PS0001 (`5fac67d`) it went on: "the VAULT_* keys from
   etc/loc-api-srv.env; LoadConfig() only calls the vault when all five
   VAULT_* keys are present." PS0003 (`533a3bd`) removed vault support
   and deleted those lines but kept the first half.
2. **Header with no keys.** `etc/uat-api-srv.env.example:38` is
   `# --- logging / KV ---` and the file ends. It used to hold
   `KV_BACKEND` and `LOG_RECEIVER_HOST_PORT`, both removed in PS0003 (KV
   is in-process, logs go to stderr). An operator copying the file
   looks for logging keys that do not exist.
3. **Wrong script.** `.gitignore:23` says "(bin/200-run-tests writes
   tmp/test-logs/)". `bin/200-run-tests` was retired in PS0014
   (`6b0ecf0`); `bin/300-run-tests:46` writes `tmp/test-logs`. The
   comment also sits over `logs/` and `*.log`; `tmp/` is ignored at
   `.gitignore:5`.

#### What GRC changed

Nothing equivalent; PROTO-only defects. GRC dropped the example file for
`etc/uat-api-srv.env.template`, filled on the host with generated secrets
(B027); its header (`:1-4`) is complete and it has no logging section.
GRC's `.gitignore` has `tmp/` at `:6` with no script-name comment.

#### Proposed fix for PROTO

1. `etc/uat-api-srv.env.example:7`: delete "It intentionally omits" or
   finish it, e.g. "Secrets are inline; there is no external store."
2. `:38`: delete the header. If B031 (`LOG_LEVEL`) lands, use
   `# --- logging ---` with `LOG_LEVEL=info` under it.
3. `.gitignore:23`: move the comment beside `tmp/` (`:5`), say
   `bin/300-run-tests`; leave `logs/` and `*.log` under `# Logs`.
4. Same commit as B016 (adds `SERVER_BIND_ADDRESS` to this example) or
   B041 (edits `.gitignore`), whichever lands first. GRC's template is
   B027's scope; do not copy it here.
5. Tests: none.

#### Open question for Brandon

None.

### B089 — `userGroupId` in a pytest matches neither API name

- **Priority:** P2  **Area:** tests
- **GRC:** `tests/test_basic_user_authorization.py:64,72`   **PROTO:** `tests/test_basic_user_authorization.py:64,72`; `src/api-srv/internal/handler/user_handler.go:756-757,877-878`
- **Ticket:** none
- **What:** Sends `userGroupId`; API reads `groupId`; passes only because authz
refuses first.
- **Port notes:** Fix regardless of B038.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Test-quality bug (false assurance), not a runtime defect. Two pytest
cases prove a basic user cannot change group membership, but send
`{"userIdList": [1], "userGroupId": 1}` to `/addUserListToGroup` and
`/removeUserListFromGroup` (`tests/test_basic_user_authorization.py:62-65`,
`:70-73`).

The handler decodes `json:"userIds"`, `json:"groupId"` and
`json:"updateTsList"` (`src/api-srv/internal/handler/user_handler.go:756-758`,
same at `:877-879`).

The item understates it. `userGroupId` matches neither PROTO's `groupId`
nor GRC's `appUserGroupId` (B038); `userIdList` is GRC's name, not
PROTO's `userIds`; `updateTsList` is missing. The sibling tests
(`tests/test_security_context_isolation.py:195-198`) and the web store
(`src/web-app/stores/app-user-store.ts:98-102`) use the right keys.

The tests pass only because `CheckAuthorization(GrantUserGroupUpdate)`
(`user_handler.go:744`, `:865`) answers 403 before the body is decoded
(`:760`). Consequences: anyone copying them for an admin "allowed" case
gets a 400 from the length check (`:771-783`) and a confusing failure;
and the tests assert only the status, while the handler has a second 403
(`SEC_CTX_VIOLATION`, `:786-797`) that could pass for the grant check.

#### What GRC changed

GRC fixed the keys with its B038 wire rename (`720f5e1`, RG0043 section 2):
`tests/test_basic_user_authorization.py:63-64,71-72` now send
`"userIdList": [1], "appUserGroupId": 1`, matching GRC's handler
(`src/internal/handler/user_handler.go:765-766,900-901`). GRC still omits
`updateTsList` and still asserts only `403`.

#### Proposed fix for PROTO

1. `tests/test_basic_user_authorization.py:62-65,70-73`: send
   `"userIds": [1], "groupId": 1, "updateTsList": [0]`, a body that would
   pass decoding and the length check.
2. Assert the response names the grant ("Permission denied (grant ...
   required)", `internal/security/app_helper.go:562-567`), so a
   `SEC_CTX_VIOLATION` 403 cannot pass.
3. Independent of B038; do not use GRC's `userIdList`/`appUserGroupId`
   unless B038 lands in the same change.
4. Run: `bin/300-run-tests py`.

#### Open question for Brandon

None.

### B090 — Doc paths that do not exist

- **Priority:** P3  **Area:** docs
- **GRC:** `infra/sql/grc-host-db.sql:1-43` (fixes the doc/330 case)   **PROTO:** `doc/330-presales-to-postsales-migration.md:37,131` (`sql/ddl-role-host-setup.sql`); `doc/810-create-new-entity-type-notes.md:20,30,51,55,61`; `doc/340-db-host-setup.md:13,21`
- **Ticket:** none
- **What:** B053's linter: 35 of 207 tokens missing; doc/330's SQL file never
existed.
- **Port notes:** doc/330 via B026; rest by hand or B053.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (per recommendation, 2026-10-02)

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Docs bug: some backticked paths in `doc/` name files not in the repo.
GRC's RG0011 checker (B053), run read-only against PROTO with PROTO's
package roots and `doc/700` excluded, reports 34 misses in 6 documents
(the item's "35 of 207 in 16 docs" was another run). Most are not defects.

**Real, doc/330.** `doc/330-presales-to-postsales-migration.md:37` lists
`sql/ddl-role-host-setup.sql` as the post-sales DB setup; `:131` sends the
reader to its header for why `REASSIGN OWNED` fails. The item says the
file "never existed"; wrong. PS0001 (`e6aeb0e`) added it and PS0015
(`0981ddb`, "doc/340-db-host-setup.md replaces the commented-out SQL
file") deleted it. doc/330 was not updated, so an operator moving a host
to post-sales hits a dead pointer at the database-ownership step.

**Real, trivial.** `doc/810-create-new-entity-type-notes.md:20` names
`user-group-edit-page.tsx`; the file is `app-user-group-edit-page.tsx`.

**Not defects (the item overstates):** `doc/810:30` resolves under
`src/api-srv/sql/`. `doc/810:51,55,61` (`entity_model.go` etc.) are
deliberate placeholders. `doc/340-db-host-setup.md:13`
(`etc/uat-proto-test.env`) is a host path whose source is
`infra/uat/etc/uat-proto-test.env`; `:21` (`000-roles.sql`) is the name
`bin/600-bundle-uat:97` gives `roles.sql` in the bundle. doc/320/330's
`deploy.sh`, `db.sql` and `etc/api-srv.env` are bundle or host files.
`doc/050:35` names "the old `sql/010-initialcreate.sql`" on purpose.

#### What GRC changed

GRC replaced the pointer with a real file: `infra/sql/grc-host-db.sql:1-12`
is an idempotent, superuser-run host setup creating ddl/app/ci roles
"(ProtoStack PS0015's model)", called from GRC's doc/330 at `:74` and
`:153`. RG0011 added the checker. No other tests.

#### Proposed fix for PROTO

1. `doc/330:37`: replace the SQL path with `doc/340-db-host-setup.md`.
2. `doc/330:131`: explain the `REASSIGN OWNED` refusal inline or point at
   doc/340. If B026 (runbooks, `infra/sql`) is ported, point at its host
   SQL file instead, in B026's change; do not copy GRC's role names.
3. `doc/810:20`: `app-user-group-edit-page.tsx`.
4. If B053 is ported, allowlist the placeholder and host-path tokens or
   write them as `<entity>_model.go`, which the checker skips.
5. Test: B053's checker, wired into `bin/300` or the ticket skill.

#### Open question for Brandon

None.

### B092 — Wrong old password on "change my password" signs the user out
- **Priority:** P2  **Area:** go-api, web-app
- **GRC:** not checked   **PROTO:** `src/api-srv/internal/handler/auth_handler.go` `ChangeOwnPasswordHandler` ("Old password is incorrect", 401); `src/web-app/stores/auth-store.ts` (the change-password call, not `anonymous`)
- **Ticket:** found while implementing B006 (2026-10-02)
- **What:** The handler answers a mistyped old password with 401. The web
  treats a 401 on an established session as an expired session, so the
  user is signed out for a typo. This predates B006; B006 made 401 the only
  sign-out trigger, so the fix is now one status code.
- **Port notes:** Answer 403 (or 400) "Old password is incorrect" so the
  page shows the message and the session survives. Count the attempt like
  a failed login if the handler already does. Pytest: wrong old password
  returns the new status and the session cookie still works.
- **Decision:** \[ \] pull / \[ \] adapt / \[ X\] skip  (new, not yet decided)

# SKIP / Brandon-veto list (tick to override)

**Recommendations against porting**

- [ ] goose (`src/migrate-srv`, `go.mod:9`): PROTO's `cmd/migrate` has checksum drift detection (`migrate.go:101,209-215`), advisory lock, tests (`src/api-srv/cmd/migrate/main.go:1-209`, `src/api-srv/internal/migrate/migrate.go:1-354`); GRC (`cmd/migrate/main.go:3-13`) lost drift detection. Keep only the idempotent bootstrap idea (`bootstrap.go:52-59`) if PROTO ever needs one; `schema-reference` is B073.
- [ ] `go.work` multi-module (`go.work:1-15`): PROTO gets many binaries from one module via `cmd/*`; GRC drifted (pgx, go directive) until RG0043 2.6; its bin copes via `bin/100-build-api-server:13-17`, `bin/600-bundle-uat:66-69`, `bin/310-test-repo-model:32-37,96-102`, `bin/330-test-load:39`. Revisit when a derived app adds a second service; document `go work sync` then.
- [ ] `smoke/*.mjs` + `bin/221-run-test-web-app:1-23`: PROTO keeps Playwright (B066).
- [ ] Web write-refusal notification in the store (GRC `stores/create-entity-store.ts:23-57`, PROTO `:143-205`): conflicts with doc/810 "toasts belong to the page".
- [ ] uat/main branch model in PROTO's own skill/CLAUDE.md: document only (B057).

**PROTO's version is better or equal**

- [ ] `cache/kv.go` `Batch`; GRC `logging.go` rewrite; GRC `auth.go`/`token.go` split; `GenerateTOTPUri` spelling; `security/client_ip.go` (file move of `app_helper.go:310-324`).
- [ ] Web: `App.css` (dead), `error-boundary.tsx`, `report-error.ts`, `password-utils.ts` (equal), `password-generator.tsx` (stray label), modal prop renames, `confirm-delete.tsx`, `cache-utils.ts`, `use-auth-guard.ts`, `use-preference.ts`, `profile-store.ts`, `user-preferences-page.tsx`, GRC `create-entity-store` factory, group store/edit page (PS0012), `routes.ts`, `grants.ts`/`api.ts` headers, `schemas/` (+ README), `types/api.ts`, `vite.config.ts`, `tsconfig.app.json`, `eslint.config.js` max-len 200, `index.html`, `main.tsx`.
- [ ] bin: `222-build-web-app`, `211-stop-app`, `220-run-web-app` (hardcoded path), `100-build-api-server` (go.work), `320` venv activate, GRC `510` body, `600`/`610` inline-default overrides (break PS0004), blob bundling and goose MANIFEST (PROTO `bin/610-deploy-uat:143-160` already verifies), `310` package list.
- [ ] etc: one combined test env file (PROTO's two-file split is better); dev bind loopback and other GRC dev keys (PROTO `etc/loc-api-srv.env:1-36`, bind deliberate at `:27-28`).
- [ ] tests: soft-deleted -&gt; 404 not 410 (PROTO keeps 410, `tests/test_z_error_scenarios.py:87`); GRC MFA-lockout placeholder (`test_auth_flow.py:324-328`, PROTO moved it).
- [ ] docs: `doc/904` (identical), `doc/310` (path rename), GRC doc/900 goose wording, `compact-tickets`/`ticket-triage` skills (rename only), `STATUS.md` (same generator), presales-deploy skill (PROTO only).

**GRC-specific**

- [ ] Domain code: `grc_*` handlers/routes, `upload_artifact.go`, capture/evidence/policy/check/connector handlers, `identity_repository.go`, `entity_repository.go`, `job_queue.go`, `audit_repository.go`, `model/entity.go`, contract/schema-contract tests, grants 10001/10002/18001.
- [ ] Config keys `BLOB_ROOT`, `WORKER_*`, `ENTRA_*`, `IMPORT_DATA_DIR`, `ADMIN_DATABASE_URL`, `MAX_USER_REQ_PER_MIN=3000`, `TLS_NAME`; `INVITE_TTL_HOURS` only with B035.
- [ ] Web domain: `pages/grc/*`, `components/grc/{forbidden,gate,coming-soon}.tsx`, `use-can`, `permissions.ts`, `use-initial-query`, `use-route-id`, `use-logged-read`, `use-me`, `me-store`, `search.ts`, `as-of`/`grc-stores`, `grc-types`, `use-draft` pattern, deps `@mantine/dropzone`/`docx-preview`/`playwright-core`; `apiUpload`/`apiBytes` until a file endpoint exists.
- [ ] bin: `bin/206-seed-dev-tenant:1-17`, `217-run-worker-srv`, `340-test-check-runtime`, `500`, `505`, `535`/`536`, `540`.
- [ ] infra: worker unit and MainPID wait, blob paths, three-binary oras artifact, GRC run-tests body, `uat-grc-test.env` keys, `client_max_body_size 50m`, other apps' jails, restart/start/stop comment-only diffs; byte-identical fail2ban files.
- [ ] tests: `test_grc_api.py`, `test_check_runtime.py`, `grc` fixture (`tests/conftest.py:281-356`), `/control` re-pointed tests (`tests/test_security_context_isolation.py:9-40`, `tests/test_z_error_scenarios.py:20-72`); 404-not-410 at `tests/test_z_error_scenarios.py:42-51`.
- [ ] docs: `doc/320-connect-an-azure-tenant.md`; doc/105 GRC sections (job queue, PDF guard, resource routes, logged reads, go.work tree); doc/050 PowerShell/worker/imports; doc/810 registry steps; doc/902 generators; requirements outside 20/22; GRC `CLAUDE.md:3-12,58-62,72-73,122-123`, `README.md:1-53`, `src/api-srv/README.md:9-119`, `tests/README.md:98-114`, `src/web-app/schemas/README.md:1-66` (product, spec, matrix, R1, `deployed: none`); handoff/ticket-workflow goose paths; Postgres job-queue recipe (RG0025); JWT `iss` validation rollout (RG0005; PROTO already reads `AppName`, `src/api-srv/internal/auth/jwt.go:29-44`).
- [ ] GRC rulings PROTO does not share: pr-ticket-docs/security-considerations "not required" (`CLAUDE.md:150-154`); RG0015 ruling 1 "dead code stays" (PS0011 owns it).
- [ ] Already in PROTO: RG0001-RG0018 pulls (RG0002 batch, RG0006 body, RG0010 rest (PROTO `CLAUDE.md:47-50`), RG0012); fixture logins out of migrations; RG0004 nil-pool trick; old engineering-log files.
- [ ] Hygiene with nothing to port: RG0043 items 37, 39, 40; PROTO `example_*` stays (GRC deleted its own); PROTO-only `bin/000`, `230`, `400`, `515`, `etc/app.env`, `etc/uat-deploy.env`, `uat-proto-e2e-api.env`, `tests/test_a_example_crud.py`, `test_bulk_operations.py`.

# Appendix: open in both repos

Found by RG0043 (§2 numbering), unfixed in GRC; PROTO has the same code.

- 12-bit CSRF token `sessionID[len-3:]`: GRC `security/app_helper.go:541`, `auth/token.go:13-19`; PROTO `security/app_helper.go:517-520` (2.2). Rationale (SameSite + HttpOnly is the defence) belongs in doc/105 (B077).
- `http.Server` without Read/ReadHeader/Write/Idle timeouts: GRC `cmd/server/main.go:252-255`; PROTO `cmd/server/main.go:201-204` (2.5).
- Numeric config keys silently 0; `TOTP_ENCRYPTION_KEY`/`JWT_SECRET` length and `FAILED_LOGIN_*_DURATION` grammar unchecked (500 per failed login): GRC `config/config.go:177-262`; PROTO `config/config.go:84-205` (2.6).
- `/mfaVerify` ignores counter write errors: GRC `user_mfa_handler.go:440-452`; PROTO `:540-552` (2.7; fix in B008).
- Raw `encoding/json` error text echoed: GRC `auth_handler.go:236`, `user_handler.go:396,553,773`; PROTO `auth_handler.go:149`, `user_handler.go:363,526,764` (2.8).
- DB error in `validate*SecurityContext` answered 403: GRC `user_handler.go:429-438,583-592`; PROTO `user_handler.go:390-401,550-561` (2.9).
- CORS answers 200 to every OPTIONS before the mux; HSTS `preload` with `ENABLE_TLS=false`: GRC `security/headers.go:24-27,101-105`; PROTO `security_headers.go:27-30,105-109` (2.10).
- Calendar-minute rate buckets allow 2x burst: GRC `app_helper.go:371-373`; PROTO `app_helper.go:363-365` (2.11).
- `RetrieveUserList` selects `totp_seed`: GRC `user_repository.go:403`; PROTO `user_repository.go:306` (2.12).
- Admin can soft-delete their own login: GRC `user_handler.go:659-740`; PROTO `user_handler.go:645-733` (2.13).
- Password policy is length &gt;= 8 only, in four places (2.14).
- `/mfaSetup`, `/mfaVerify` `SkipCSRF` also on the JWT path: GRC `user_mfa_handler.go:34,265`; PROTO `user_mfa_handler.go:188,385` (2.15).
- First `X-Forwarded-For` entry trusted for lockout/rate keys: GRC `security/client_ip.go:25-38`; PROTO `security/app_helper.go:310-324`. GRC will not fix behind nginx; the template should document it or count from the right. **Folded into B007 (Brandon, 2026-10-02).**
- `admin` subject exempt from rate limit and block: GRC `app_helper.go:356,409`; PROTO `app_helper.go:348,398` (`:347-348`). A literal tied to a fixture login; doc/105 states it as fact (worth a security look).
- Group grant edit does not end member sessions: GRC `app_user_group_handler.go:176-262` (fix in B003).
- systemd units lack kernel/syscall hardening (B074).

# Do not copy from GRC

On any port use `HOOKID`/`NACKID`, `SERVERONLY`, `example.com`, `proto`.

- Nackmon check-in codes in filenames: `infra/prd-unified002/shrmpl-etc/nack-monitors/{prd-grc-backup-all,prd-grc-renew-web-certs,uat-grc-renew-web-certs}_<code>.env` (a holder can forge "job succeeded"; `e64e253`, `16f3acd`).
- Live hook id in a filename and a commented `HOOK_URL` (l.7): `infra/prd-unified002/shrmpl-etc/hooks/prd-promote-grc_<live-id>.env` (`f941ae1` renamed it from `HOOKID` on purpose).
- `REPO_URL=git@github.com:Radley-House/radley-grc.git`: `infra/uat-unified001/etc/uat-grc-deploy.env:16` (PROTO keeps `SERVERONLY`, `infra/uat/etc/uat-proto-deploy.env:15`).
- Real hostnames `uat-grc.atticustechnology.com`, `grc.atticustechnology.com`, `prd-unified002-hook.*` in nginx confs, `SITE_NAME`, `PULSE_URL`, CORS (GRC `infra/README.md:19-20` keeps them; PROTO `infra/README.md:49-53` forbids).
- `TLS_NAME` (`etc/loc-api-srv.env:24`): a personal dev hostname; PROTO already leaks the same host (B087).
- Other apps' fail2ban jails (`uat-meetingapp`, `uat-onesheet`, `cicd`, `prd-meetingapp`) and the real `ignoreip`.
- DB names `uat_radley_grc`/`prd_radley_grc`, user `svc-grc`, Slack "Radley GRC", module path in `uat-grc-build:99-101`, the prod host IP (doc/331:56), port 8089, `brandon-martin`.
- GRC `infra/README.md:30` claims placeholder filenames; its prd files do not have them.
- GRC follow-up for Brandon (not PROTO work): rename the three nack files and the hook file back to placeholders and blank `HOOK_URL` before RG0044's fresh root commit.

# Summary

Counts by tier and primary area (first area listed on each item).


| Area      | P1     | P2     | P3     | Bug P2 | Bug P3 | Total  |
| --------- | ------: | ------: | ------: | ------: | ------: | ------: |
| go-api    | 13     | 8      | 6      | 0      | 0      | 27     |
| docs      | 0      | 8      | 10     | 2      | 2      | 22     |
| infra     | 5      | 6      | 2      | 0      | 0      | 13     |
| web-app   | 2      | 3      | 4      | 0      | 0      | 9      |
| tests     | 3      | 4      | 1      | 1      | 0      | 9      |
| bin       | 0      | 3      | 3      | 0      | 0      | 6      |
| etc       | 1      | 0      | 0      | 0      | 1      | 2      |
| skills    | 0      | 1      | 1      | 0      | 0      | 2      |
| **Total** | **24** | **33** | **27** | **3**  | **3**  | **90** |


Primary area undercounts cross-cutting work: as a secondary area, docs
appears on 26 more items, tests 18, web-app 14, bin 7, tickets/etc/skills 6
each. The SKIP list holds 21 grouped veto lines; the appendix 16 items.

# Completed

### B091 — COMPLETED — Remove the ticket system from ProtoStack (Brandon, 2026-10-02)
- **Completed:** 2026-10-02. Deleted `tickets/`, `scripts/lint-tickets.py` and the `ticket-workflow`, `ticket-triage` and `compact-tickets` skills; kept `handoff` (Brandon) and rewrote its ticket parts to record outstanding work in `tasks/todo.md` only. `CLAUDE.md` loses the Tickets section and ticket lines (branches are now `feature/<slug>`), `DERIVING.md` §8 becomes a commit-based review from `template-YYYY-MM-DD` tags, and `bin/000-rebrand` drops `--ticket-prefix` (re-verified on a scratch copy: exit 0, no lowercase slug left). PS ids and ticket wording were stripped from docs, env comments, Go/TS/py comments and the seed; the two migrations that name PS ids were left alone (checksums). Tests: all green, go=189 unit=28 e2e=8 py=67
- **Priority:** P1  **Area:** tickets, skills, docs, scripts
- **GRC:** n/a   **PROTO:** `tickets/` (whole tree: `01-open/`, `04-archive/`, `engineering-log/`, `README.md`, `STATUS.md`); `scripts/lint-tickets.py`; `.claude/skills/ticket-workflow/`, `.claude/skills/ticket-triage/`, `.claude/skills/compact-tickets/`, `.claude/skills/handoff/` (ticket-bound: records to the ticket and `tasks/todo.md`; remove or rewrite, flagged for veto); `CLAUDE.md` "Tickets" section and the ticket lines in "Layout" and "Branches"; `DERIVING.md` §8 template-lineage via `tickets/04-archive/` and `template-` tags; `README.md` ticket references; `doc/902-project-commands.md` lint-tickets lines; the `pr-ticket-docs` / `security-considerations` wording in `CLAUDE.md` that points them at the ticket folder.
- **Ticket:** Brandon's directive 2026-10-02: "rip out the whole ticket scheme from protostack ... the skills, everything related to tickets"
- **What:** The template carries a full ticket workflow (PS numbering, folder
  states, linter, four skills, lineage tags) that a derived app inherits
  whether or not it wants one. Brandon is removing it from the template.
  Derived apps keep whatever tracker they choose.
- **Port notes:** Trumps every other item that mentions tickets: B057 and
  B079 skip; B055, B080, B081 lose their ticket parts; B084 skips. Items
  whose Port notes say "new PS ticket" simply become work. Open PS tickets
  in `tickets/01-open/` (PS0010, PS0011, PS0017, PS0021, PS0022) are
  covered by B028, B060, B010, B018 or are dropped (PS0017 passkey);
  record that in the removal commit. Branch naming `TKT-PS####-slug` goes
  with it; pick a plain `feature/<slug>` convention in `CLAUDE.md`. The
  global `pr-ticket-docs` and `security-considerations` skills are
  Brandon's own and stay; only the lines binding them to `tickets/` go.
  The `backport-grc` work itself is tracked by this doc, not a ticket.
- **Decision:** \[X\] pull / \[ \] adapt / \[ \] skip  (Brandon, 2026-10-02)

### B001 — COMPLETED — MFA: enrolled login lands on setup; `/mfaSetup` returns the secret
- **Completed:** 2026-10-02. `/auth` now returns `mfaEnrolled` (new `AppUserModel.MFAEnrolled()`: `mfa_setup` and a stored seed) and `/mfaSetup` answers 409 `MFA_ALREADY_SETUP` to an enrolled user with no secret or QR; `mfaAlreadySetup` is gone from its answer. The web store keeps `mfaEnrolled`, login routes an enrolled user to the new `/mfa-verify` route (same page, verify only, never calls `/mfaSetup`), and the page detects a stale 409 by status code via `apiRequest`, not by message text. `AuthHandler` now takes `UserRepositoryInterface` so it can be unit tested; doc/105 MFA flow (the dead `/auth2` step replaced by `/mfaSetup` + `/mfaVerify`) and the api-srv README route note updated. Tests: all green, go=198 unit=31 e2e=8 py=67
- **Priority:** P1  **Area:** go-api, web-app, tests, docs, tickets
- **GRC:** `src/internal/handler/user_mfa_handler.go:130-142,241-244`; `src/internal/model/app_user_model.go:40-46`; `src/internal/model/security_model.go:19-21`; `src/internal/handler/auth_handler.go:411`; `types/auth-type.ts:10-11`; `stores/auth-store.ts:37-38,65,88,198`; `pages/login.tsx:50-51`; `constants/routes.ts:21-22`; `App.tsx:204-208`; `pages/mfa-onboarding-page.tsx:30-32,38,45-47,55-59,71,120`; `pages/invite-page.tsx:119`; tests `user_mfa_handler_test.go:712-752`, `model/app_user_model_test.go:45-65`, `stores/auth-store.test.ts:69-90`, `tests/test_zz_mfa_totp.py:194-235`; `tickets/02-working/RG0042-mfa-login-goes-to-setup/RG0042-mfa-login-goes-to-setup.md:23-51`; RG0042 md `:23-30`   **PROTO:** `src/api-srv/internal/handler/user_mfa_handler.go:276-372` (`mfaAlreadySetup` :362,367); `auth_handler.go:284-300`; `model/security_model.go:10-19`; `pages/login.tsx:54-56`; `pages/mfa-onboarding-page.tsx:31,33,37,43-57,70,119`; `stores/auth-store.ts:22-36,63-71`; `types/auth-type.ts:15-22`; `constants/routes.ts:19`; `app.tsx:97-101`; `tests/e2e/mfa.spec.ts`; `tests/test_zz_mfa_totp.py:98-108`; `pages/mfa-onboarding-page.tsx:31-57`
- **Ticket:** RG0042 / `63ccf75`; new PS ticket
- **What:** PROTO sends every MFA login to setup; `/mfaSetup` returns an
enrolled user's decrypted seed + QR to a password-only temp token, and pytest
asserts it (:103). GRC: `/auth` returns `mfaEnrolled`, login routes to
`/mfa-verify`, `/mfaSetup` 409s.
- **Port notes:** One change: Go, web, pytest, e2e (enrolled re-login never
calls `/mfaSetup`). Test 409 by status, not GRC's regex (:56); drop `/grc`
(:103). Harness B019; docs B051.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

### B003 — COMPLETED — Server-side session revocation
- **Completed:** 2026-10-02. New `security/session_revocation.go`: `CheckJWT` refuses a revoked session id (`SESSION_REVOKED*`) or a token issued before a per-login cutoff (`REVOKED_BEFORE*`) with 401, and a KV read error with 500; entries live `JWT_SESSION_EXPIRE_HOURS`. Revoked by `/logoff`, own password change (the changing session survives, as in GRC), `/changeOtherUserPassword`, `/admin/resetUserMFA`, `PUT`/`DELETE /user`, add/remove group members, and `PUT /appUserGroup` (every member, via new `RetrieveMemberUserNameList`; GRC's gap). Deviations: the cutoff compares nanoseconds (jti already holds issue ns, `Claims.IssuedAtNano`) so a token issued in the second of a revocation is not let through as in GRC; B004's minimal piece landed here (`/refresh` re-signs on the same session id via `reissueJWTForUser`) and `/changeSecurityContext` revokes the id it replaces, so one login stays one live id. Restart-forgets-revocations documented in doc/105 Session Revocation; Refresh, ChangeSecurityContext and ChangeOwnPassword handlers now take `UserRepositoryInterface` for tests. Tests: all green, go=211 unit=31 e2e=8 py=70
- **Priority:** P1  **Area:** go-api, tests
- **GRC:** `src/internal/security/session_revocation.go:1-86`; `security/app_helper.go:297-311`; `handler/auth_handler.go:837-846,1016-1021`; `handler/common.go:291-312,314-328`; `user_handler.go:641-646,725-730,865-871,992-998,1139-1142`; `user_mfa_handler.go:719-721`; `user_block_handler.go:142-146`; `security/session_revocation_test.go:16-65`; `handler/session_revocation_test.go:14-37`   **PROTO:** new `src/api-srv/internal/security/session_revocation.go`; `security/app_helper.go:274-305`; `handler/auth_handler.go:677-708,711-870`; `user_handler.go:504-643,645-733,735-963,965-1087`; `user_mfa_handler.go:674-768`; `auth/jwt.go:47`
- **Ticket:** RG0043 item 41 / `720f5e1`
- **What:** PROTO JWTs live to expiry after logoff, password/MFA reset, delete,
group change. GRC checks `SESSION_REVOKED*<sid>` and `REVOKED_BEFORE*<user>`
in `CheckJWT`; KV error 500, revoked 401.
- **Port notes:** Drop-in; restart forgets revocations. Fix GRC's gap: group
grant edit (`app_user_group_handler.go:176-262`) must revoke. Add HTTP tests.
Pairs B004, B034.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

### B004 — COMPLETED — Refresh keeps its session id; web re-syncs CSRF once
- **Completed:** 2026-10-02. The Go half (`/refresh` re-signs on the same session id via `reissueJWTForUser`) landed with B003; this adds `TestRefresh_TwiceKeepsTheSessionAndCsrfToken` (two refreshes keep id and token, an id under 3 chars is replaced) and makes CheckCSRF's two 403s answer `{"error": ..., "code": "CSRF"}` through a new `refuseCSRF` in `security/app_helper.go` (WriteError untouched; pinned by `security/csrf_test.go`). On the web, `utils/api.ts` `apiRequest` resends a CSRF-refused write once after a single-flight `/refresh` when the refreshed user and tenant match the tab (checked through `setSessionResyncListener`, registered in `stores/auth-store.ts`); on a mismatch it drops the write with `SessionChangedError`, and the store adopts the session, clears the entity caches, notifies and navigates to the dashboard through `setRouteNavigator` (registered in `app.tsx`). `changeSecurityContext` now stores the new `csrfToken`. Tests: `tests/csrf-resync.test.ts` (the GRC cases on `apiRequest`, shared refresh, user and tenant mismatch, entity-store heal, context-switch token) and an e2e two-tab save in `tests/e2e/example-pages.spec.ts`; pages still show their own error toast for a dropped write beside the "Session changed" notice (they do not special-case `isSessionChanged`). Tests: all green, go=216 unit=41 e2e=9 py=70

- **Priority:** P1  **Area:** go-api, web-app
- **GRC:** `src/internal/handler/auth_handler.go:574-600,1037-1105`; `auth_handler_test.go:457-528`; `utils/api.ts:172-219,221-279`; `utils/api.test.ts:171-240`   **PROTO:** `src/api-srv/internal/handler/auth_handler.go:453-475,870-899` (new id :454); `utils/api.ts:34-56,129-143`
- **Ticket:** `fbd5b80` (Go), `45d263d` (web)
- **What:** PROTO's `/refresh` mints a new session id, and the CSRF token is its
last 3 chars, so concurrent refreshes desync and writes 403. GRC re-signs on
the same id; client resends a CSRF-refused write once.
- **Port notes:** Low risk. Client matches `app_helper.go:508,527` message text;
prefer an error code. 12-bit token: appendix.
- **Decision:** \[ \] pull / \[X \] adapt / \[ \] skip

**Expanded for the adapt decision (2026-10-02).**

#### What is wrong in PROTO today

Paths: Go under `src/api-srv/internal/`, web under `src/web-app/`.

1. **Login.** `createJWTForUser` mints a random 32-hex session id S1
   (`handler/auth_handler.go:878-882`, `rand.Read(sessionIDBytes)` then
   `hex.EncodeToString`), signs it into the JWT as claim `ses`
   (`auth/jwt.go:14`), and the CSRF token is its tail
   (`auth_handler.go:354`, `csrfToken = sessionID[len(sessionID)-3:]`). The
   JWT rides in an HttpOnly, SameSite=Strict cookie that every tab of the
   origin shares. The token lives only in that tab's JS memory
   (`utils/api.ts:1`, `let csrfToken`; set at `stores/auth-store.ts:73`).
2. **What calls /refresh.** No timer, no 401 handler.
   `hooks/use-auth-guard.ts:16-20` calls `authenticateWithAuthToken()` while the
   store is `"TBD"`, i.e. once per page load (new tab, reload, deep link).
   "Try again" (`app.tsx:72`) calls it too; `<StrictMode>` (`main.tsx:12`)
   doubles it on every dev boot, two refreshes in flight at once.
3. **What /refresh does.** The same `createJWTForUser` as login
   (`auth_handler.go:454`): a new id S2, the shared cookie overwritten (`:474`,
   `setAuthCookie(w, jwt, 3600)`), tail(S2) returned (`:488`). Only the calling
   tab stores it (`auth-store.ts:129`, `setCsrfToken(data.csrfToken)`).
4. **The drift.** Tab A loaded at 9:00 holds tail(S1). Tab B opened at 9:05
   sets the cookie to S2. Tab A keeps sending tail(S1) (`api.ts:40-42`); the
   server checks the cookie's session (`security/app_helper.go:517-518`,
   `expectedToken := sh.claims.SessionID[len(sh.claims.SessionID)-3:]` /
   `if csrfToken != expectedToken {`) and answers 403
   `"CSRF validation failed"` (`:527`). PROTO needs no overlap: one refresh
   breaks every other open tab. True overlap (StrictMode, double-clicked "Try
   again") can break one tab: cookie and memory land independently, so the
   cookie can end on S2 while memory holds tail(S3).
5. **What the user sees.** Reads keep working (`app_helper.go:495-497` skips
   GET/HEAD/OPTIONS/TRACE). The first Save or Delete gets the 403, which
   `throwForFailedResponse` treats as a dead session (`api.ts:98-100`,
   `if (response.status === 401 || response.status === 403)`). The listener
   (`auth-store.ts:223-246`) clears state, toasts "Session ended: Your session
   has ended. Please sign in again." and the guard sends the tab to login; the
   unsaved form is lost. That is B006 making it worse: without the 403 sign-out
   this would be an error toast. Signing in again mints S3, which breaks tab B
   the same way, so two open tabs evict each other in turn.
6. **Classification.** Correctness/UX bug, not a security hole. "Drift" means
   the tab's copy of the 3-char tail is stale against the session in the
   cookie; the server fails closed and nothing is leaked or wrongly accepted.
   Rotating the id on refresh bought no protection: refresh already requires a
   valid JWT, and the id is server-random, so there is no fixation to close.
   The token being only 12 bits is a separate issue (appendix).

#### What GRC changed

**Go, `fbd5b80`.** `createJWTForUser` now mints an id and delegates to a new
`reissueJWTForUser(user, grantList, appServices, sessionID)`
(`src/internal/handler/auth_handler.go:1038-1049`, `:1074-1105`). `/refresh`
passes the caller's id (`:579-584`,
`reissueJWTForUser(... jwtClaims.SessionID,)`). The JWT is re-signed with a
fresh expiry; only the id is kept, so racing refreshes all return the same
token. Login, MFA verify and the context switch still mint (`:447`,
`user_mfa_handler.go:516`, `auth_handler.go:764`). An id under 3 chars is
replaced (`:1084-1090`, `if len(sessionID) < 3 {`), so the slice cannot
panic. Safe: the caller gets nothing it did not already hold. Pinned by
`TestReissueKeepsTheSessionAndSoTheCsrfToken` (`auth_handler_test.go:457-515`):
two reissues keep the id and token; "a login still mints a new session"
(`:494`); "a session id too short to slice is replaced" (`:504`); no HTTP test.

**Web, `45d263d`.** Heals a tab that already drifted. `apiJson` calls
`apiJsonOnce(..., true)` (`src/web-app/utils/api.ts:226`); then (`:270-272`):
`if (mayResync && !options.anonymous && isCsrfRefusal(...) && isWrite(method)
&& await resyncCsrfToken()) return apiJsonOnce<T>(endpoint, init, options,
false);`. Detection is text: `:213-214`,
`status === 403 && /csrf/i.test(message)`, which matches both "Missing CSRF
token" and "CSRF validation failed". `isWrite` (`:218`) reads the method, so a
body-less DELETE counts. The fresh token is a bare
`fetch("/api/refresh", { credentials: "include" })` (`:196`) then
`setCsrfToken(body.csrfToken)` (`:200`), shared by concurrent failures
(`resync ??=`, `:194`) and cleared by `queueMicrotask` (`:206`). Loop guard:
the retry passes `mayResync = false`, and a failed refresh means no retry.
Tests `utils/api.test.ts:171-240`: happy path and call order (`:174`), at most
once (`:190`), a GET 403 left alone (`:204`), a permission 403 left alone
(`:216`), a failed refresh not retried (`:229`).

#### Adapted fix for PROTO

1. **Go helper.** Split `createJWTForUser` (`auth_handler.go:872-899`) into
   mint + `reissueJWTForUser`, keeping the `< 3` guard. Keep PROTO's
   `auth.GenerateJWT(appServices.Config.AppName, user.UserName, ...)` order
   (issuer first, PS0007); GRC's call puts AppName sixth and will not compile.
   PROTO has no `auth.NewSessionID`; keep the inline `rand`/`hex`.
2. **RefreshHandler.** `:454` calls `reissueJWTForUser(...,
   jwtClaims.SessionID)`. Its repo param (`:385`) becomes
   `UserRepositoryInterface` so tests can use `MockUserRepository`.
3. **Other minters stay.** Login (`:332`), MFA verify
   (`user_mfa_handler.go:602`) and `/changeSecurityContext` (`:622`) keep
   minting (B005): a tenant change should change the id so a stale tab is
   caught (step 6). Separate one-line web bug, whatever B005's skip means for
   CSRF: `auth-store.ts` `changeSecurityContext` (`:173-197`) never calls
   `setCsrfToken(data.csrfToken)`, so the switching tab's own next write 403s
   and signs it out. Add it.
4. **Error code, not text.** PROTO's body is only `{"error": clientMsg}`
   (`app_helper.go:254`); `"SECF"` is a log code shared by every security
   failure. Recommend CheckCSRF's two 403s (`:508`, `:527`) answer
   `{"error": ..., "code": "CSRF"}` and the client tests `code === "CSRF"`.
   Rewording a message silently disables a text match, `/csrf/i` would catch
   any future message naming CSRF, and a Go test can pin a code.
5. **Retry in `apiRequest`, not `apiJson`.** PROTO's writes mostly bypass
   `apiJson`: `create-entity-store.ts:149,167,191`, `app-user-store.ts:57,85`,
   `app-user-group-store.ts:30` and `examples-store.ts:22` call `apiRequest`
   (`api.ts:34-57`) then `throwForFailedResponse`. A GRC-style `apiJson`-only
   retry misses every entity write. In `apiRequest`: on 403, a write method,
   not yet retried, and `response.clone().json()` carries `code: "CSRF"`, run a
   single-flight resync and re-send once. A second 403 flows on unchanged.
6. **Identity guard (not in GRC).** `/refresh` answers for whoever the cookie
   holds now. If another tab switched tenant or signed in as someone else, a
   blind retry runs this tab's write as that tenant or user while the screen
   shows the old one; today the CSRF mismatch is what stops it. The resync
   compares the response's `user.userName` and `user.securityContext` with the
   tab's store and skips the retry on mismatch, via a registered callback like
   `setSessionExpiredListener` (`api.ts` cannot import the store, `:78-80`).
   With 5-6 in place drift never reaches B006's sign-out.
7. **B003.** Revocation is keyed by session id (`SESSION_REVOKED*<sid>`). With
   rotation, logoff revokes only the newest id, and JWTs carrying earlier ids
   from that login live to their 72 h expiry. Keeping the id makes one login
   one id, so logoff ends the chain. Land with or before B003. Do not copy
   GRC's "no server-side state" comment (`:1068`); 720f5e1 made it false.
8. **Tests.** Go, new `handler/auth_handler_test.go`: sign a JWT with id
   `"session-abc"` as `user_mfa_handler_test.go:160` does, call
   `RefreshHandler` twice with `testServices` (`example_handler_test.go:88`)
   and a mock repo; assert both `csrfToken`s are `"abc"` and the cookie's `ses`
   is unchanged; id `"ab"` gets a fresh 32-char id; a CSRF 403 body carries
   `"code":"CSRF"`. Vitest, new `utils/api.test.ts`: GRC's five cases on
   `apiRequest`, plus: identity mismatch, no retry; an entity-store write
   heals; two refused writes share one `/refresh`.
9. **Blind-copy traps.** Module `github.com/Radley-House/radley-grc` vs
   `github.com/wbmartin/starter-api`; GRC's `appHelper` is PROTO's `app`;
   cookie `MaxAge` 0 vs PROTO's `3600` (`:474`) is B029; `refusal()` and
   `announce()` do not exist in PROTO; PS0001's 403-as-signed-out
   (`api.ts:62-63,98`) means the retry must run before `throwForFailedResponse`.

#### Is the CSRF control still working as designed? (Brandon, 2026-10-02)

Yes. The control exists to stop a request from another site riding the
user's cookie into a write. Three layers do that today and B004 touches
none of them: the cookie is `SameSite=Strict`
(`handler/auth_handler.go:38`), so a cross-site link or form post never
carries it; the token travels in a custom `X-CSRF-Token` header
(`utils/api.ts:42`, `security/app_helper.go:505`), which an HTML form cannot
set and which forces a CORS preflight that only `CORS_ALLOWED_ORIGINS`
passes (`security/security_headers.go:73-100`); and the token value is read
from a same-origin JSON body another site cannot see. The defence rests on
the attacker not knowing the token, not on the token changing. Rotating the
id on refresh never added protection, because the attacker never had the
old token either. The client-side retry runs only in the app's own code in
response to its own write, so it is not reachable cross-site. The 12-bit
token and the permissive OPTIONS handling are pre-existing appendix items,
unchanged by this fix.

#### Open question for Brandon

Step 6 goes beyond GRC. When the cookie now belongs to another tenant or user,
is today's outcome (the 403 stands and the tab is signed out) acceptable, or
should the tab reload into the identity the cookie now carries?

**Answered (Brandon, 2026-10-02): the tab reloads into the new identity.**
So the resync in step 6 does not retry the write when `/refresh` answers
with a different `user.userName` or `user.securityContext` than the tab
holds. It drops the pending write, adopts the refreshed session (user,
tenant, grants, CSRF token) into the auth store, clears the entity caches,
and navigates to the app's landing route for that identity, with a
notification saying the session changed in another tab. The user's
unsaved form is lost, which is the honest outcome: it was authored under
an identity that no longer holds the cookie. Add a vitest case for it
beside the five GRC cases in step 8.
- ADAPT REQUEST, Describe the fix before we continue

### B006 — COMPLETED — A 403 signs the user out
- **Completed:** 2026-10-02. `utils/api.ts` `throwForFailedResponse` now treats only a 401 as session expired (listener + `SessionExpiredError`); a 403 is a plain `Error` with the server's message and the session is untouched. Comments in `api.ts` rewritten; `src/web-app/README.md` gains an "API Errors" section with GRC's rule adapted. `tests/session-expired.test.ts`: the 401/403 case is 401 only and a new 403 case pins a plain Error, no "Session ended" toast, `authState` "YES"; `authenticateWithAuthToken` and `api-unreachable.test.ts:137` unchanged; no Playwright/pytest asserted a 403 sign-out. Deviation: the optional `err.message` toast follow-up skipped (8 copies across 4 scaffolding pages, no shared place). Tests: all green, go=216 unit=41 e2e=9 py=70

- **Priority:** P1  **Area:** web-app, docs
- **GRC:** `utils/api.ts:66-91,286-298`; `utils/api.test.ts:81-130`; `stores/auth-store.ts:183-203,256-266`; `README.md:45-65`; `tickets/03-done/RG0018-web-api-error-semantics-error-boundary/RG0018-web-api-error-semantics-error-boundary.md:65-96`   **PROTO:** `utils/api.ts:62-63,93-102,124`; `stores/auth-store.ts:217-245`; `tests/session-expired.test.ts:50`; `tests/api-unreachable.test.ts:137`; `src/web-app/utils/api.ts:62-66`
- **Ticket:** RG0018 / `d752dbd`
- **What:** PROTO 403s on grant (`app_helper.go:562-563`), block (`:449,480`),
CSRF (`:508,527`), handler refusals
(`user_handler.go:396,556,791,804,908,921`, `auth_handler.go:609`); the client
signs out on each. GRC: only an established session's 401 expires it.
- **Port notes:** Brandon's call: reverse PS0001's "403 too" (`api.ts:62-63`)?
After B004 + B005. Update both tests.
- **Decision:** \[ \] pull / \[ X\] adapt / \[ \] skip

**Expanded for the adapt decision (2026-10-02).**

#### What is wrong in PROTO today

Every authenticated web call ends in one function. Any 401 or 403 there
fires the session-expired listener (`src/web-app/utils/api.ts:98-100`):

    if (response.status === 401 || response.status === 403) {
      sessionExpiredListener?.();
      throw new SessionExpiredError(response.status, message);

The auth store registers that listener at load
(`stores/auth-store.ts:223-246`). It clears the CSRF token and every entity
cache, sets `authState: "NO"`, wipes `grantList` and `userName`, and shows an
orange toast titled "Session ended": "Your session has ended. Please sign in
again." `hooks/use-auth-guard.ts:22-28` then sees `NO`, saves the current
path in `postAuthDestination` and navigates to `/login`. The user is signed
out locally even though the `auth` cookie is still valid.

The API answers 403 for many things that are not a dead session; each site
and when a legitimate user meets it:

- `security/app_helper.go:562-563` "Permission denied (grant ... required)":
  a page guarded on grant A makes one call that needs grant B.
- `app_helper.go:449,480` "User account is blocked": an admin blocks the
  account mid-session.
- `app_helper.go:508,527` "Missing CSRF token" / "CSRF validation failed":
  the in-memory token no longer matches the session id.
- `handler/user_handler.go:396,556` (`CreateUser`, `UpdateUser`) and
  `:791,804,908,921` (add/remove users to a group) "Access denied: ... same
  security context": a stale id from another tenant, or a DB error, since
  the check is `!valid || err != nil`.
- `handler/auth_handler.go:609` "Security context not available": switching
  to a tenant the user just lost.

Three concrete scenarios:

1. Missing grant. `pages/app-user-edit-page.tsx:63` guards on
   `[requiredGrant, grants.READ_USER]`, but `:77` also calls
   `readAppUserGroupList()`, which the API gates on `GrantUserGroupRead`
   (`app_user_group_handler.go:39`). A user with user-update but not
   group-read opens the page, gets a 403, and lands on the login screen.
2. CSRF drift. The token is the session id's last 3 chars
   (`auth_handler.go:354,488,646`); `/refresh` mints a new id (`:454`, B004),
   so tab A's refresh leaves tab B's token stale. `changeSecurityContext`
   (`stores/auth-store.ts:173-197`) never calls `setCsrfToken`, though the
   server sends a new one (`:646,662`, B005). Either way the next write
   answers 403 "CSRF validation failed" and the user is signed out.
3. Blocked mid-session. Every authenticated call runs the block check, so
   the next call 403s "User account is blocked". The user is signed out and
   told the session "ended", which is the wrong reason.

Why PS0001 chose this. The PS0001 ticket records only the outcome:
"auth-store signs out only on 401/403, not on network/5xx"
(`tickets/04-archive/PS0001-backport-derivatives.md:31`). It was ported from
atticus-meeting in commit `8559400`. The only stated reason is the comment at
`utils/api.ts:62-64`: "403 is included: CSRF validation rejects with 403, and
from the user's side that is the same event -- the credential they were
carrying is no longer good." That holds for a stale CSRF token, which a fresh
sign-in does fix. It does not hold for a missing grant, a security-context
refusal or a DB error dressed as 403, where signing in again changes
nothing and the user loops back to the same refusal.

Severity: a UX and correctness issue, not a security hole. The server
enforces every check on every call; the client only decides what the user
sees, so not signing out on a 403 grants nothing. A blocked user keeps a
UI whose every call 403s, and the next page load signs them out (step 2).

#### What GRC changed

The rule: only a 401 on a non-anonymous call is "session expired". In the
API only `CheckJWT` writes a 401 (missing, invalid or expired `auth` cookie;
PROTO matches: `app_helper.go:282,294`). A 403 stays a plain `Error`
carrying the server's message. GRC `src/web-app/utils/api.ts:71,77-78`: "**A
403 is not this** ... A 403 therefore stays a plain `Error`: the screen shows
the API's reason and the session is untouched." The classifier, `:292-297`:

    if (status === 401 && !options.anonymous) {
      const error = new SessionExpiredError(message, status);
      announce({ kind: "session-expired", error });
      return error;
    }
    return new Error(message);

The store reacts only to that announcement (`stores/auth-store.ts:263-266`)
and only from an established session (`:187`,
`if (get().authState !== "YES") return;`), then clears state as PROTO does
(`:188-202`). The page-load `/refresh` path is separate: any non-outage
failure there, 401 or 403, still sets `NO` (`auth-store.ts:145-150`).

README rule (`src/web-app/README.md:51-52,54-57`): `SessionExpiredError` is
"**401** on a call that was not `anonymous`"; plain `Error` is "everything
else: 400, **403**, 404, 409". "A **403 is not an expired session** ...
signing out on one would sign a signed-in user out of the first list their
role may not read." RG0018 (`:80-84`) adds that the body carries no code,
"so the client cannot tell a grant refusal from a dead session". Tests:
`utils/api.test.ts:108-117` pins a 403 as a plain `Error` with the API's
message and no announcement; `:81-90` pins the 401 path, `:92-106` the
anonymous exception. Commit `d752dbd` touches only those two files.

#### Adapted fix for PROTO

1. `utils/api.ts:98`: change the test to `response.status === 401`. Rewrite
   the doc comment at `:59-65` with GRC's reasoning (a 403 is a refusal, not
   an expiry; list the four 403 sources) and fix the `:124` comment
   ("a 401/403 is a SessionExpiredError" becomes "a 401").
2. `stores/auth-store.ts:223-246`: no logic change; the `NO` guard at
   `:226` already plays GRC's `:187` role. Leave `authenticateWithAuthToken`
   (`:111`) alone: `/refresh` skips CSRF and authorization
   (`auth_handler.go:390-393`), so its only 403 is the block check and a
   blocked user is still signed out at the next page load, as in GRC.
3. Tests. `tests/session-expired.test.ts:50`: change `it.each([401, 403])` to
   401 only, and add a 403 case asserting a plain `Error` with the server's
   message, not `SessionExpiredError`, and `authState` still `"YES"`.
   `tests/api-unreachable.test.ts:137` ("reports NO on a 403") tests the
   `/refresh` path, not `throwForFailedResponse`; it stays, since that 403
   means "blocked". B006's "update both tests" overstates it: one flips.
4. Callers. Nothing outside the tests uses `SessionExpiredError` or
   `isSessionExpired` (grep of `src/web-app`), and no page tests for 403.
   After the change a 403 reaches existing catch blocks: list reads set the
   store `error` (`stores/create-entity-store.ts:139-144`) and log via
   `reportError`, which only writes to the console; saves show a generic
   toast ("Failed to save user", `app-user-edit-page.tsx:154-160`). A
   refused read becomes silent (an empty group picker). Optional follow-up:
   show `err.message` in those toasts.
5. Order. Land after B004 and after B005's web line (`changeSecurityContext`
   adopting `data.csrfToken`). Today sign-out is the only recovery from a
   stale CSRF token. If B006 lands first, scenario 2 leaves the user signed
   in with every write failing until a reload. B005 is marked "skip" for
   the server CSRF change; its one-line web fix is independent and needed.
6. Do not take GRC's `components/grc/gate.tsx` and `forbidden.tsx`. They
   mirror GRC's role/verb matrix (`useCan`, auditor/external roles) and have
   no PROTO counterpart; PROTO's `useAuthGuard` already redirects on missing
   page grants. GRC's `subscribeApiOutcome` seam and refresh retries are
   separate items, not part of this port.

#### Open question for Brandon

PS0001 decided a 403 signs the user out; GRC reversed it so only a 401 on an
established session does. Recommendation: reverse it (after B004 and the
B005 web line), because a sign-in cannot fix a missing grant or a context
refusal, and signing out hides the real reason behind "Session ended".

**Answered (Brandon, 2026-10-02): reverse it.** Only a 401 on an
established session signs the user out. A 403 stays a plain error with the
server's message. Order holds: land after B004 and after the one-line
`changeSecurityContext` CSRF-token fix.
- Brandon: Need you to explain more about how you'd fix

### B007 — COMPLETED — `/auth` username oracle: unknown login fast and uncounted
- **Completed:** 2026-10-02. `/auth` maps `repository.ErrNotFound` (now wrapped by `RetrieveUserByUserName`, text kept) to a nil user and any other lookup error to 500 "Login is temporarily unavailable"; it always runs `auth.CheckPasswordHashOrDummy` (lazy `sync.OnceValue` dummy built with `HashPassword`) and folds `user == nil || !verified` into the counted wrong-password branch, so unknown names cost argon2, hit both lockout keys and get the identical 401; no `AccountStatus`, B008's fail-open counter branch left as is. XFF: `security.ClientIP(r, trustedProxyCount)` takes the entry `TRUSTED_PROXY_COUNT` hops from the right (all `X-Forwarded-For` lines joined), `RemoteAddr` at 0 or when the header is short or the hop is not an address, ports stripped; new optional config key (default 0, non-negative) set to 0 in `etc/loc-api-srv.env`, `etc/e2e-api-srv.env`, `infra/uat/etc/uat-proto-e2e-api.env` and 1 in `etc/uat-api-srv.env.example`, `infra/uat/etc/uat-proto-api.env`, `infra/prd/etc/prd-proto-api.env`; `bin/300-run-tests` writes `TRUSTED_PROXY_COUNT=1` into the pytest API copy (loopback-bound) so pytest cases send their own `X-Forwarded-For`. Rate limit and both lockout handlers (`/auth`, `/mfaVerify`) all key on `ClientIP`; both nginx confs already set `$proxy_add_x_forwarded_for`. Docs: doc/105 (/auth failures, ClientIP), `src/api-srv/README.md` (key). Tests: all green, go=239 unit=41 e2e=9 py=72

- **Priority:** P1  **Area:** go-api, tests
- **GRC:** `src/internal/handler/auth_handler.go:304-366`; `src/internal/auth/auth.go:58-81`; `auth_handler_test.go:529-546`   **PROTO:** `src/api-srv/internal/handler/auth_handler.go:196-243`; `auth/password.go:37-90`; `repository/user_repository.go:62`
- **Ticket:** RG0043 item 42 / `720f5e1`
- **What:** An unknown login 401s at once (no argon2, no count) and `/auth`
skips the minute limit: timing reveals logins, spraying never locks. GRC
hashes a once-made dummy, counts both lockouts, same 401, DB error 500.
- **Port notes:** Needs `ErrNotFound` (B028). Omit `AccountStatus` (:329) unless
B034/B035. Add a pytest (GRC has none).
- **Decision:** \[ \] pull / \[X \] adapt / \[ \] skip

**Expanded for the adapt decision (2026-10-02).**

#### What is wrong in PROTO today

All PROTO paths are under `src/api-srv/internal/`. `/auth` takes a username
and password. It checks a lockout, looks the user up, then runs argon2id.

**(1) Timing oracle.** A name with no row returns before any hashing:

```go
// handler/auth_handler.go:197-198, 205-207
user, err := userRepo.RetrieveUserByUserName(r.Context(), req.UserName)
if err != nil {
    app.WriteError(w, http.StatusUnauthorized, "Invalid credentials", ...
```

A real name with a wrong password reaches `auth.CheckPasswordHash` at
`handler/auth_handler.go:216`, which runs argon2id at `auth/password.go:89`.
The cost comes from `auth/jwt.go:22-24`: `t=3`, `m=64 MiB`, `p=4`. That
measured about 36 ms per hash on an Apple-silicon laptop, and a small VM is
slower (roughly 50-150 ms). The unknown-name path costs one indexed query,
about 1 ms. The bodies match (both "Invalid credentials"), but the gap is
tens of times larger than normal network jitter. A few requests per name
tell an attacker, over the internet, which usernames exist.

**(2) Unknown names are never counted.** The counter lives only in the
wrong-password branch (`handler/auth_handler.go:220-225`) and keys on both
IP and username:

```go
// handler/auth_handler.go:67, 71
ipCount, err := kv.Incr("loginfail-ip-"+ip, ...WindowDuration)
userCount, err := kv.Incr("loginfail-user-"+username, ...
```

`etc/loc-api-srv.env:15-17` sets 7 failures in 5 min, lock for 5 min. The
unknown-name branch (`:198-213`) returns without incrementing either key.
`/auth` also opts out of the per-minute limit (`MAX_UNAUTH_REQ_PER_MIN=20`,
`etc/loc-api-srv.env:20`) on purpose:

```go
// handler/auth_handler.go:132-137
app.SkipJWT()
app.SkipRateLimit()
...
if !app.CheckAllChecksComplete() {
```

The comment at `:127-131` justifies the skip by saying the lockout handles
brute force. That holds only for names that exist. Probing unknown names
has no cost, no counter and no rate cap, so the attacker enumerates at
full speed. They then spend the counted attempts only on confirmed names.
Guessing passwords against real names still trips the IP lock after 7, so
"spraying never locks" overstates it. The real gap is free enumeration.

**DB error looks like bad credentials.** `repository/user_repository.go:62`
returns a plain string error, `fmt.Errorf("user not found")`, with no
sentinel. The handler treats every error the same way. So a dead database
also answers 401 "Invalid credentials", which hides an outage from callers
and monitoring.

#### What GRC changed

GRC commit `720f5e1` (RG0043 item 42, "2.14 /auth: dummy argon2 + counted
unknown logins") changed four things.

1. A dummy hash is built once, lazily, from 32 random bytes. An empty hash
   is checked against it, so the unknown-name path pays full argon2 cost
   (`src/internal/auth/auth.go:62,75-78`):
   ```go
   var dummyPasswordHash = sync.OnceValue(func() string {
   if encodedHash == "" {
       _ = CheckPasswordHash(password, dummyPasswordHash())
       return false
   ```
2. Not-found is distinguished from failure, and a real DB error is a 500
   (`src/internal/handler/auth_handler.go:307-313`):
   ```go
   if err != nil && !errors.Is(err, repository.ErrNotFound) {
       appHelper.WriteError(w, http.StatusInternalServerError,
           "Login is temporarily unavailable", ...
   ```
3. Unknown name, inactive account and wrong password share one branch
   (`:328-330`). That branch calls `incrementFailedAttempt` (`:334-339`),
   so both the `loginfail-user-` and `loginfail-ip-` keys count.
4. Every failure gets the same 401 "Invalid credentials" (`:358-364`).

GRC kept `SkipRateLimit` (`:213`) for the same reason PROTO gives. The Go
test `TestAuthHandlerUnknownLoginIsCountedLikeABadPassword`
(`src/internal/handler/auth_handler_test.go:529-546`) sends 7 unknown-name
attempts and expects the same 401 body. The eighth must be a 429. No test
checks timing, and GRC has no pytest for this. One detail not to copy:
`auth.go:59` left the `CheckPasswordHash` doc line sitting above the dummy
hash, so `CheckPasswordHash` at `:83` lost its doc comment.

#### Adapted fix for PROTO

1. **`repository/user_repository.go:61-62`**: return
   `fmt.Errorf("user not found: %w", ErrNotFound)`. `ErrNotFound` already
   exists in `repository/errors.go`. B028 (PS0010) is the larger cleanup:
   one sentinel set, one `mapRepoError` mapper and no string matching.
   This item does not need to wait for it. Wrapping keeps the "user not
   found" text, so the `strings.Contains` at `handler/user_mfa_handler.go:94`
   and the asserts at `repository/user_repository_test.go:100,264` still
   pass. B028 can drop that text later.
2. **`auth/password.go`**: add `dummyPasswordHash` (`sync.OnceValue`) and
   `CheckPasswordHashOrDummy` below `CheckPasswordHash`, with the doc comment
   kept in place. Build the dummy with `HashPassword` so it always uses the
   current `jwt.go` parameters.
3. **`handler/auth_handler.go:196-243`**: on `errors.Is(err,
   repository.ErrNotFound)`, keep `user` nil. On any other error, return 500
   "Login is temporarily unavailable" (`USR_LKUP_FAIL`). Then always call
   `CheckPasswordHashOrDummy` and fold `user == nil || !verified` into the
   existing branch. That branch already counts both keys and sends the
   standard 401. Keep the `SkipRateLimit` comment at `:127-131`, which is
   now true. B008 (pull) makes the counter fail closed at `:233` in the
   same branch, so do the two together.
4. **No `AccountStatus`.** GRC's check at `:329` reads a column PROTO
   lacks. It belongs to B034 (user block) and B035 (invite onboarding,
   whose `0004_user_lifecycle` migration adds the column). Skip it here.
   B034 adds it to the same condition later.
5. **Go test**: change `AuthHandler` to take `UserRepositoryInterface`
   (`handler/user_handler.go:19`, which the MFA handlers already use), then
   port GRC's test with a mock that returns `repository.ErrNotFound`. Add a
   case where a generic error produces a 500.
6. **pytest in `tests/test_auth_flow.py`**: send each case from its own
   `X-Forwarded-For` address (as in B019), so the loopback IP never locks.
   (a) Timing: send 5 requests with a random unknown name and 5 with a real
   name and a wrong password. Assert the unknown-name median is at least
   half the wrong-password median. Keep each IP under the threshold.
   (b) Lockout: send `FAILED_LOGIN_ATTEMPT_THRESHOLD` (default 7, the same
   read as `tests/test_zz_mfa_totp.py:122`) unknown-name attempts. Expect
   401 each time, then 429.

#### Open question for Brandon

The IP key is spoofable. `security/app_helper.go:310-315` (`ClientIP`)
trusts the first `X-Forwarded-For` entry, so a client can rotate that
header and avoid the IP lock completely. After this fix, the username lock
holds, but probing many names from rotating forged IPs is still capped
only by argon2's cost. Should B007 wait for the XFF fix (doc/700
appendix), or ship first and leave that gap to the XFF item?

**Answered (Brandon, 2026-10-02): fix both in the same batch.** B007
carries the `X-Forwarded-For` fix with it. `ClientIP`
(`security/app_helper.go:310-324`) must stop trusting the first entry.
Shape for the port: a `TRUSTED_PROXY_COUNT` config key (0 for local dev,
1 behind the host's nginx); take the entry that many hops from the right
of `X-Forwarded-For`, and with 0 trust only `RemoteAddr`. Because nginx
appends the real client address last, the right-hand side is the only
part the client cannot forge. Both the lockout keys and the per-minute
rate keys use `ClientIP`, so one change covers both. Add a Go unit test
for the header shapes (absent, one entry, forged leading entries) and
confirm the infra nginx confs set `X-Forwarded-For` with
`$proxy_add_x_forwarded_for`. The appendix line for XFF is closed by
this item.

### B008 — COMPLETED — Fail closed when the failed-attempt counter cannot be written
- **Completed:** 2026-10-02. A failed-attempt counter write error now answers 500 "Login is temporarily unavailable" (`FAILED_ATTEMPT_INCR_FAIL`, logged by `WriteError`) instead of logging and sending the 401, in `/auth`'s wrong-password/unknown-name branch (`handler/auth_handler.go`) and `/mfaVerify`'s wrong-code branch (`handler/user_mfa_handler.go`). The lockout read before the password/code check already failed closed in both (500 `LOCKOUT_CHECK_FAIL`); unchanged. No KV seam added: the tests make the in-process store refuse `Incr` with an unparseable `FAILED_LOGIN_ATTEMPT_WINDOW_DURATION` and refuse `Get` with `KV.Close()`. B009 (same `/mfaVerify` branch) not folded in. doc/105 /auth bullet updated. Tests: all green, go=242 unit=41 e2e=9 py=72

- **Priority:** P1  **Area:** go-api
- **GRC:** `src/internal/handler/auth_handler.go:332-358`   **PROTO:** `src/api-srv/internal/handler/auth_handler.go:216-243` (:233); `user_mfa_handler.go:540-552`
- **Ticket:** RG0003 / `0d5146b`; RG0043 r1 / `720f5e1`
- **What:** PROTO logs and continues, silently disabling the throttle. GRC
answers 500 "Login is temporarily unavailable".
- **Port notes:** Trivial. `/mfaVerify` fails open in both (GRC
`user_mfa_handler.go:440-452`, RG0043 2.7); fix PROTO :540-552 too.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

### B009 — COMPLETED — TOTP code replay inside the ±1-step window
- **Completed:** 2026-10-02. `auth.ValidateTOTPCodeAt` (totp.go) returns the matched 30 s step (skew ±1, constant-time compare); `/mfaVerify` (the only place a code is accepted: temp-token login, enrolment and session paths) records it with a new atomic `KV.SetIfGreater` on `TOTP_LAST_STEP*<user>` (3 min), so a replayed or older step is a counted 401 "Invalid TOTP code" and a store error is 500 `TOTP_STEP_STORE`. Deviation: compare-and-set under the KV lock instead of GRC's read-then-write, closing the race; `/admin/resetUserMFA` deletes the key (500 `TOTP_STEP_CLEAR` before the seed write if it cannot). Go tests (totp_test.go, kv_test.go, user_mfa_handler_test.go), pytest `fresh_code` + `test_totp_code_replay_is_refused`, Playwright `freshTotpCode`; doc/105. Tests: all green, go=252 unit=41 e2e=9 py=73

- **Priority:** P1  **Area:** go-api, tests
- **GRC:** `src/internal/auth/totp.go:136-170`; `handler/user_mfa_handler.go:419-431,460-468,736-754`; `tests/conftest.py:23-40`; `tests/test_zz_mfa_totp.py:266-292`; `user_mfa_handler.go:419-468`   **PROTO:** `src/api-srv/internal/auth/totp.go:125-128`; `handler/user_mfa_handler.go:530-560`; `tests/test_zz_mfa_totp.py:70,94,106`; `user_mfa_handler.go:626`
- **Ticket:** RG0043 item 44 / `720f5e1`; pytest `dd8b4c3`
- **What:** PROTO accepts one code repeatedly for \~90 s. GRC stores the last
accepted step per login (`TOTP_LAST_STEP*<user>`, 2 min); a step at or below
it is a wrong code; KV error fails closed.
- **Port notes:** Keep `GenerateTOTPURI`. Port `fresh_code()` with it
(`TOTP.now()` would flake). Harness B019.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

All PROTO Go paths are under `src/api-srv/internal/`. A login with MFA is
two calls. `/auth` checks the password and returns a temp token. Then
`/mfaVerify` takes that temp token plus a six-digit TOTP code and issues
the session cookie. The code check is one library call:

```go
// auth/totp.go:126-127
func ValidateTOTPCode(code, secret string) bool {
	return totp.Validate(code, secret)
```

`totp.Validate` (`github.com/pquerna/otp` v1.5.0, `totp/totp.go:37-44`)
uses a 30 s period and `Skew: 1`. It accepts the code for the current
30 s step and for the step on either side. The handler calls it at
`handler/user_mfa_handler.go:533` and does nothing else with the result:

```go
// handler/user_mfa_handler.go:533-534
valid := auth.ValidateTOTPCode(req.TOTPCode, decryptedSecret)
if !valid {
```

Nothing records that a code was spent. The temp token is single-use
(deleted at `:627-631`, PS0020), but the TOTP code is not. Any number of
fresh temp tokens can each be paired with the same code while its step
is in the window.

Consequence. An attacker who already has the password, and who sees one
code (shoulder-surfing, a phishing relay page, a screen share, a proxy or
log that records request bodies), can open their own session with that
code. RFC 6238 section 5.2 says a verifier must not accept the same code
a second time after a success; PROTO does. This is a security hole in the
second factor, moderate in size because the password is still required.

Correction to the doc entry: "accepts one code repeatedly for ~90 s"
describes the acceptance window (three steps). A code is shown on the
phone only during its own step, so after the user types it, it stays
replayable for 30 to 60 s, not 90 s.

#### What GRC changed

GRC commit `720f5e1` (RG0043 item 44, was 2.20) made a code good for one
use per login.

1. `src/internal/auth/totp.go:152-169` adds `ValidateTOTPCodeAt`, which
   walks the same three steps by hand and returns the step that matched,
   with a constant-time compare:
   ```go
   for _, step := range []int64{current - 1, current, current + 1} {
       expected, err := totp.GenerateCodeCustom(secret, ...
           subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
           return step, true
   ```
   `ValidateTOTPCode` (`:140-143`) becomes a wrapper over it.
2. `src/internal/handler/user_mfa_handler.go:425-434`: a valid code whose
   step is at or below the last accepted step for that user is treated as
   a wrong code. It counts toward the lockout and answers 401 "Invalid
   TOTP code" (`ITCC`), the same as a typo. A KV read error answers 500
   `TOTP_REPLAY_CHECK` (fail closed).
3. `:463-468`: after a good code, `KV.Set("TOTP_LAST_STEP*"+userName,
   step, "2min")`. A write error answers 500 `TOTP_STEP_STORE`, so a code
   is never accepted without being recorded. Helpers sit at `:736-754`.

Tests. pytest only: `tests/test_zz_mfa_totp.py:266-292`
(`test_totp_code_replay_is_refused`) signs in, then reuses the same code
on a second temp token and expects 401. It ends with a good login so the
counted failure does not push the next test over the lockout. Commit
`dd8b4c3` added `fresh_code()` (`tests/conftest.py:23-40`), which hands
out the oldest unused step and sleeps to the next boundary only when all
three are spent. `pyotp.TOTP(secret).now()` flakes once replay is refused,
because two calls in one 30 s step return the same, now-spent, code. GRC
added no Go test for `ValidateTOTPCodeAt` or the replay branch.

#### Proposed fix for PROTO

1. **`auth/totp.go:125-128`**: add `ValidateTOTPCodeAt(code, secret,
   now) (int64, bool)` as GRC wrote it, and make `ValidateTOTPCode` call
   it. Trap: PROTO's URI helper is `GenerateTOTPURI` (`:63`); GRC renamed
   it `GenerateTOTPUri`. Keep PROTO's name; copy only the two functions,
   the `totpPeriodSeconds` const and their imports.
2. **`handler/user_mfa_handler.go:532-560`**: call `ValidateTOTPCodeAt`,
   then the replay check, then the existing wrong-code branch. After the
   branch, `Set` the step and fail closed on error. Add the three helpers
   at the bottom of the file. Key length is safe: `user_name` is
   `VARCHAR(50)` (`handler/auth_handler.go:25-26`), so the key stays under
   the KV's 100-char cap (`cache/kv.go:27`).
3. **Admin reset** (`AdminResetUserMFAHandler`, `:674`): also `Del` the
   `TOTP_LAST_STEP*` key. GRC does not. Without it, a user re-enrolling a
   new secret within 2 min can have a fresh code refused once. Add the
   new key family to the list at `cache/kv.go:3-5`.
4. Do with **B008**: the wrong-code branch at `:536-551` still logs and
   continues on a counter write error; make it 500 in the same edit.
   **B010** clears counters after a completed login, which the GRC replay
   test relies on. **B019** supplies the per-test `X-Forwarded-For`
   harness (`fwd(IP_REPLAY)`) the pytest uses.
5. **pytest**: port `fresh_code()` into `tests/test_zz_mfa_totp.py` (PROTO's
   `conftest.py` never mints codes; its users are `mfa_exempt`, `:42-50`).
   Replace `pyotp.TOTP(secret).now()` at `:70`, `:94` and `:106`; `:106`
   would fail outright, since it reuses `:70`'s step within the same 30 s.

Tests to add:
- Go `auth/totp_test.go` (new): codes at s-1, s, s+1 return that step;
  s-2, s+2, 5-digit and non-digit codes fail.
- Go `handler/user_mfa_handler_test.go`: same code on two temp tokens,
  second is 401 `ITCC` and counts one failure; a closed KV gives 500.
- pytest `test_totp_code_replay_is_refused`, ported with `fresh_code`.

Residual, also in GRC: the replay read and the step write are not atomic,
so two requests racing with one code can both pass. A compare-and-set in
`cache/kv.go` would close it; the window is milliseconds.

#### Open question for Brandon

Should a replayed code count toward the lockout like a typo (GRC's
choice), or be refused without counting so a user who double-submits is
not penalised? Recommendation: count it, as GRC does; a legitimate double
submit is one count out of seven and the answer stays indistinguishable
from a wrong code.
Brandon: yes

### B010 — COMPLETED — PS0021: reset-MFA grant 9020; completed login clears counters
- **Completed:** 2026-10-02. New `GrantResetUserMFA = 9020` (`RESET_MFA_TOKEN`) in all three maps of `model/grants.go`, and `/admin/resetUserMFA` checks it instead of 9003; `/mfaVerify` treats an empty seed like NULL (400 `TOTP_SECRET_MISSING`). A completed login (`/auth` for an MFA-exempt user, the temp-token branch of `/mfaVerify`) `Del`s only `loginfail-user-<name>` via `resetFailedAttemptsOrLog` (auth_handler.go; logs, never fails the login); per Brandon the per-IP counter is left to its window, a deviation from GRC. No migration: groups and their grant arrays are fixture rows only, so `local-seed.sql` gives `Admin2` 9020; a real tenant's admin group needs 9020 added by hand. Go tests in auth_handler_test.go / user_mfa_handler_test.go (the existing reset test now uses 9020), pytest 403 case and success-between-typos case, doc/105. Tests: all green, go=260 unit=41 e2e=9 py=75

- **Priority:** P1  **Area:** go-api, tests, docs, tickets
- **GRC:** `src/internal/model/grants.go:25-31,52,74,87`; `handler/user_mfa_handler.go:614-620,536-543`; `handler/auth_handler.go:135-177,483-490`; tests `user_mfa_handler_test.go:435-497`, `tests/test_zz_mfa_totp.py:294-346,348-396,398-415,506-518`; `tickets/03-done/RG0003-mfa-temp-token-and-loose-ends/RG0003-mfa-temp-token-and-loose-ends.md:67,81,101-109`   **PROTO:** `src/api-srv/internal/model/grants.go:20-30`; `handler/user_mfa_handler.go:684` (9003), `:483`, `:620-630`; `handler/auth_handler.go:96-110,245-251` (commented :246); `constants/grants.ts:21`; `tests/test_zz_mfa_totp.py:110-116`; `tickets/01-open/PS0021-mfa-loose-ends.md:20-47`; `doc/105-api-srv-tech-spec.md` \~\`:183-200\`
- **Ticket:** RG0003 / `0d5146b`, `37978ce`; closes PS0021 items 1, 3
- **What:** (a) Web gates reset on 9020, API checks 9003; GRC checks 9020,
clears seed + `mfa_setup`, 400s on a nil or `''` seed (PROTO :483: nil only).
(b) Counters never clear on success; GRC `Del`s both after a completed login
(after `/mfaVerify`).
- **Port notes:** `Del`, not `"0"` 1 s (`Incr` keeps TTL). Write "where a login
completes" into doc/105, PS0021. Open: per-IP clear behind NAT.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

PROTO Go paths are under `src/api-srv/internal/`. PS0021 filed three MFA
loose ends. Two are real; one was fixed before PS0021 was written.

**(a) Grant 9020 exists only on the web.** The web gates Reset MFA on a
grant the API does not know (`src/web-app/constants/grants.ts:21`
`RESET_MFA_TOKEN: 9020`; `src/web-app/pages/app-user-edit-page.tsx:313`).
`model/grants.go:18-31` stops at 9015, and the route checks the general edit
grant (`handler/user_mfa_handler.go:684`):

```go
app.CheckAuthorization(model.GrantUserUpdate)   // 9003
```

Consequence: anyone who can edit a user can strip that user's second factor,
returning the account to password-only. A privilege-separation hole. It
looks right only because the tenant 1 Admin group carries both 9003 and 9020
(`src/api-srv/sql/fixtures/local-seed.sql:52`). A group with 9003 and no
9020 gets an API that resets and a web that hides the button; the tenant 2
`Admin2` group (`local-seed.sql:77`) is exactly that shape.

**(b) An empty seed is decrypted.** Reset writes `''`, not NULL
(`handler/user_mfa_handler.go:738-744` passes `""`). `/mfaVerify` checks only
NULL (`:483`, `if user.TOTPSeed == nil {`), so `''` fails in
`auth.DecryptTOTPSeed` ("ciphertext too short", `auth/totp.go:114-115`) and
the caller gets 500 "Failed to decrypt TOTP secret" (`:497-503`) instead of
400 "Please restart MFA setup". `/mfaSetup` already treats nil and `''`
alike (`:282`). A correctness bug: the web calls `/mfaSetup` first, but any
direct API client hits it right after a reset.

**Correction to doc/700 and PS0021 item 2.** Both say reset leaves
`mfa_setup` true. It does not: `repository/user_repository.go:221` has read
`SET totp_seed = $1, mfa_setup = false, ...` since `eba4e08` (2026-05-19).
GRC's repository is identical and GRC changed nothing there. PS0021 item 2
closes as already done.

**(c) A successful login never clears the failure counters.** Each failure
increments `loginfail-ip-<addr>` and `loginfail-user-<name>`
(`handler/auth_handler.go:67,71`), living 5 minutes. The clear is commented
out (`:245-250`, `// err = resetFailedAttempts(appServices.KV, ...)`).

Consequence: six typos at 09:00, a good login at 09:01, one typo at 09:03,
and the user is locked out for 5 minutes. A UX bug that reads as a security
event. The dead helper (`:96-109`) is wrong if merely uncommented: it writes
`"0"` with a 1 s TTL, and `Incr` honours a TTL only on create
(`cache/kv.go:209-211`), so the next real failure inherits 1 s and vanishes.

#### What GRC changed

RG0003, commits `0d5146b` (Go) and `37978ce` (pytest). Paths under
`src/internal/`.

1. `model/grants.go:25-31` adds `GrantResetUserMFA = 9020`, commented as its
   own authority, in all three maps (`AllKnownGrants` `:55`, `UserGrants`
   `:74`, `GrantIDToShortCodeMap` `:87`). The handler checks it
   (`handler/user_mfa_handler.go:620`,
   `appHelper.CheckAuthorization(model.GrantResetUserMFA)`).
2. `/mfaVerify` treats `''` as missing (`handler/user_mfa_handler.go:373`,
   `if user.TOTPSeed == nil || *user.TOTPSeed == "" {`), answering 400
   `TOTP_SECRET_MISSING`. The doc's ref `:536-543` is the counter clear.
3. `resetFailedAttempts` `Del`s both counter keys
   (`handler/auth_handler.go:135-154`); `resetFailedAttemptsOrLog`
   (`:156-172`) logs a failure without failing the login. Lock flags
   (`loginlock-*`) stay: a locked account 429s before any password check.
4. The clear runs only where a login completes: `/auth` for an
   `mfa_exempt` user (`handler/auth_handler.go:483-490`) and the temp-token
   branch of `/mfaVerify` (`handler/user_mfa_handler.go:537-543`). Not after
   the password step of an MFA login, or anyone with the password but no
   TOTP code could clear the counters at will.

Tests: Go `TestAdminResetUserMFARequiresGrant9020`
(`handler/user_mfa_handler_test.go:435`; 9003-only caller gets 403). pytest
in `tests/test_zz_mfa_totp.py`: `:294` counters reset (6 fails, a success,
3 fails, all 401); `:348` reset clears seed and `mfaSetup`, `/mfaVerify`
asks to enrol; `:398` Basic user's reset is 403 naming `RESET_MFA_TOKEN`.

#### Proposed fix for PROTO

1. **`model/grants.go`**: add `GrantResetUserMFA = 9020` and
   `"RESET_MFA_TOKEN"` to all three maps. Without the short-code entry the
   403 at `security/app_helper.go:558-561` names `9020`, not the grant.
   Update the expected maps in `model/user_grant_model_test.go:34-57,70-93`.
2. **`handler/user_mfa_handler.go:684`**: check `model.GrantResetUserMFA`.
3. **`handler/user_mfa_handler.go:483`**: `nil || *user.TOTPSeed == ""`.
4. **`handler/auth_handler.go:96-109`**: two `kv.Del` calls plus GRC's doc
   comment; add `resetFailedAttemptsOrLog`; delete `:245-250`. Call it after
   `setAuthCookie` at `auth_handler.go:351` (exempt login) and at
   `user_mfa_handler.go:619` (temp-token branch). `clientIP` is in scope
   (`auth_handler.go:171`, `user_mfa_handler.go:509`).
5. **Seed**: add 9020 to `Admin2` (`local-seed.sql:77`). Trap: GRC's
   fixture is `bin/lib/test-db-fixture-logins.sql` plus a goose migration;
   PROTO's is `local-seed.sql`. No migration (grants are code). A real
   tenant whose admin group lacks 9020 loses reset on deploy; note it.
6. **Docs**: extend the lockout sentence at
   `doc/105-api-srv-tech-spec.md:194-197`: counters clear only where a login
   completes (`/auth` for MFA-exempt, `/mfaVerify` with a temp token
   otherwise). Close PS0021 items 1-3, item 2 as already done.
7. **Traps**: GRC's helper var is `appHelper`, PROTO's `app`. GRC tests use
   `stubUserRepository`; PROTO's use testify `MockUserRepository`. GRC's
   `authCookieSessionMaxAge` is B029; keep PROTO's `0`.
8. **Dependencies**: B008 (fail closed) and B009 (TOTP replay) edit the
   same `/mfaVerify` branch; land together. B018/B019 (tenant 2, per-test
   `X-Forwarded-For`) are needed for the counter pytest not to lock the
   loopback IP.

Tests:
- Go: `TestAdminResetUserMFARequiresGrant9020` on `MockUserRepository`.
- Go: after `resetFailedAttempts`, `Incr` returns 1 with the full window.
- Go: `/mfaVerify` with seed `""` answers 400 `TOTP_SECRET_MISSING`.
- pytest: port GRC `:294`, `:348`, `:398` into `tests/test_zz_mfa_totp.py`.

#### Open question for Brandon

Should a completed login clear the per-IP counter too, or only the
per-username one? Clearing the IP key lets an attacker who owns any valid
account log in every six guesses and keep the IP below 7 while spraying
other names; behind NAT one user's success also wipes a colleague's
strikes. Recommendation: clear only `loginfail-user-<name>` and let the IP
key expire on its window, a one-line deviation from GRC.
Brandon: Per User only

### B011 — COMPLETED — Body caps on every login-layer write; 413, not 400

- **Completed:** 2026-10-02. New `decodeJSONBody` / `readBody` in `handler/common.go` cap `r.Body` with `http.MaxBytesReader` and answer 413 "Request body too large" (`*http.MaxBytesError` via `errors.As`) or a generic 400 "Invalid request body", decoder text to the log only; every JSON-body handler in user, group, auth, MFA, profile, example and error-report handlers now reads through them (one helper instead of GRC's `limitBody` + `handledBodyTooLarge` pair, so no branch can forget the 413). Caps: 4 KiB `maxAuthBodyBytes` on `/auth`, `/mfaSetup`, `/mfaVerify`, `/changeSecurityContext` and the public `/userSelfCreate` (`maxMFABodyBytes` deleted), 64 KiB `maxLoginLayerBodyBytes` on the previously unbounded login-layer writes, `PUT /me` kept at 12 KiB and `/error-report` at 8 KiB (now 413; B033 undecided), and per the open question a separate `maxEntityBodyBytes = 1 << 20` in `example_handler.go`. Check order unchanged; `/changeSecurityContext` (B005 skipped) and `/mfaSetup` still read the body before their checks, now bounded. Docs: doc/105 Request Body Limits (nginx `client_max_body_size 20M` as outer bound) and 413 row, api-srv README. Tests: Go `body_limit_test.go` (helper, `CreateUser` 65 KiB, `PUT /me` 13 KiB, `/auth` 5 KiB, example list under/over 1 MiB), pytest `TestBodyLimits` (5 KiB `/auth`, 65 KiB `POST /user` then the session still reads and writes) and a no-decoder-text assertion in `test_invalid_request_body_returns_400`. Tests: all green, go=274 unit=41 e2e=9 py=77

- **Priority:** P1  **Area:** go-api, tests, docs
- **GRC:** `src/internal/handler/common.go:14-56`; `auth_handler.go:187,643,893`; `user_handler.go:392,549,769,904,1028`; `user_mfa_handler.go:29,260,632`; `app_user_group_handler.go:109,196`; `own_profile_handler.go:165`; `user_block_handler.go:86`; `user_invite_handler.go:131,318,345,503`; `tests/test_auth_flow.py:123-150`; `doc/105-api-srv-tech-spec.md:378-398`   **PROTO:** unbounded `user_handler.go:359,522,760,881,989`, `app_user_group_handler.go:110,194`, `user_mfa_handler.go:696`, `auth_handler.go:741`; capped `auth_handler.go:143,514`, `user_mfa_handler.go:201,395`, `profile_handler.go:109`; 400 at `auth_handler.go:145-152`; doc/105 after `:209`
- **Ticket:** RG0002 / `be58262`, `b33c9b9`; RG0043 item 45 / `720f5e1`
- **What:** Authenticated callers stream unbounded bodies; PROTO's existing caps
answer 400 with decoder text. GRC caps every decode (4 KiB unauth, 64 KiB
login layer) and answers 413 `BODY_TOO_LARGE`.
- **Port notes:** One `limitBody` spelling incl. example handlers; add a 64 KiB
pytest; doc/105 lists PROTO's caps.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Paths are under `src/api-srv/internal/handler/` unless stated. Neither Go
nor the server at `cmd/server/main.go:201-204` limits `r.Body`; only a
handler that wraps it in `http.MaxBytesReader` first is bounded.
**(1) Most writes are unbounded.**

```go
// user_handler.go:358-359 (CreateUser)
var req model.CreateUserRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
```

Same shape at `user_handler.go:522,760,881,989`,
`app_user_group_handler.go:110,194`, `user_mfa_handler.go:696` (admin MFA
reset) and `auth_handler.go:741` (change own password, any signed-in
user). The copy-me template is unbounded too:
`example_handler.go:165,251,595,723`. All of these run their JWT, rate,
CSRF and grant checks first, so the caller is signed in.

**Correction to the item:** it says only authenticated callers are
unbounded. `user_mfa_handler.go:54` (`POST /userSelfCreate`, public) also
decodes with no cap, behind only the anonymous per-IP limiter. B002 was
skipped, so that route stays and needs a cap.

`json.Decoder` holds each JSON value whole, so one large string field sits
in memory in full. On servers nginx bounds a request at 20 MiB
(`client_max_body_size 20M`, `infra/uat/etc/nginx_uat-proto.example.com:32`).
Local dev, or an API port reachable without nginx (B016), has no bound. A
few concurrent 20 MiB bodies from one signed-in user cost hundreds of MiB.
This is a robustness gap (DoS by a signed-in user); nothing leaks.

**(2) The caps that exist answer 400.** Five sites are capped at 4 KiB:
`auth_handler.go:143,514` (`maxAuthBodyBytes`, `:23`),
`user_mfa_handler.go:201,395` (`maxMFABodyBytes`, `:21`), and
`profile_handler.go:109` (4 KiB + 8 KiB). Over the cap the read fails and
the generic branch answers 400:

```go
// auth_handler.go:145-149
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    app.WriteError(w, http.StatusBadRequest,
        fmt.Sprintf("Invalid request body for auth: %v", err),
```

"400 with decoder text" holds only for `/auth` and `/changeSecurityContext`
(`:516-523`): the client sees "...: http: request body too large". The MFA
sites say "Failed to read request body", `/me` says "Invalid request body".
No client, test or log search can tell "too big" from "malformed".
`error_report_handler.go:52` (8 KiB) also answers 400. doc/105 has no
body-limit section; its status table (`:261-271`) has no 413. A DX gap.

#### What GRC changed

RG0002 (`be58262`, `b33c9b9`) capped the pre-auth routes and made the cap
answer 413. RG0043 item 45 (`720f5e1`) capped every authenticated
login-layer write. The pieces live in `src/internal/handler/common.go:15-60`:

```go
// common.go:31, 35-36
const maxLoginLayerBodyBytes int64 = 64 << 10
func limitBody(w http.ResponseWriter, r *http.Request, maxBytes int64) {
```

`handledBodyTooLarge` (`:43-60`) uses `errors.As` for `*http.MaxBytesError`;
on a hit it writes 413 "Request body too large" (`BODY_TOO_LARGE`) and
returns true. Each decode error branch calls it before its own 400, as at
`auth_handler.go:229-233`. Caps: 4 KiB at `auth_handler.go:187,643`,
`user_mfa_handler.go:29,260`, `user_invite_handler.go:318,345`; 64 KiB at
`auth_handler.go:893`, `user_handler.go:392,549,769,904,1028`,
`user_mfa_handler.go:632`, `app_user_group_handler.go:109,196`,
`user_invite_handler.go:131,503`; 12 KiB on `/me`
(`own_profile_handler.go:165`). 64 KiB holds a membership list of
thousands of ids (`common.go:28-30`).

Tests: pytest `test_oversized_login_body_is_413`
(`tests/test_auth_flow.py:123-150`, 5 KiB `/auth` body, 413, then a normal
login) and Go `TestOwnProfileBodyOverTheCapIs413`
(`own_profile_check_sequence_test.go:337`). Nothing tests the 64 KiB cap,
and GRC's doc/105 "Request Body Limits" (`doc/105-api-srv-tech-spec.md:378-397`)
omits it. GRC slip not to copy: `user_block_handler.go:86` caps but never
calls `handledBodyTooLarge`, so it still answers 400.

#### Proposed fix for PROTO

1. **New `handler/common.go`**: `maxAuthBodyBytes` (moved from
   `auth_handler.go:23`), `maxLoginLayerBodyBytes = 64 << 10`, `limitBody`,
   `handledBodyTooLarge`. Delete `maxMFABodyBytes` (same 4 KiB). Keep
   `maxPreferenceBytes` (`profile_handler.go:29`). Traps: GRC's local is
   `appHelper`, PROTO's is `app`; import path is
   `github.com/wbmartin/starter-api/...`.
2. **Five existing caps**: replace `http.MaxBytesReader` with `limitBody`
   and put `if handledBodyTooLarge(app, w, err) { return }` first in each
   error branch. At the MFA sites the error comes from `io.ReadAll`
   (`user_mfa_handler.go:202,396`); `errors.As` works there too.
3. **Unbounded sites**: `limitBody(w, r, maxLoginLayerBodyBytes)` plus the
   413 branch at `user_handler.go:359,522,760,881,989`,
   `app_user_group_handler.go:110,194`, `user_mfa_handler.go:696`,
   `auth_handler.go:741`. `maxAuthBodyBytes` at `user_mfa_handler.go:54`.
4. **Example handlers** `example_handler.go:165,251,595,723`: cap them so
   every derived entity starts capped (size: open question). Leave
   `/error-report` at 400 if B033 removes it; else add the 413 branch.
5. **Docs**: "Request Body Limits" in doc/105 after `:209` with PROTO's own
   caps (no invite, block or 4 MiB GRC resource caps), the 413 rule, and a
   413 row at `:261-271`. One line in `src/api-srv/README.md` `## Security`
   (`:83`): cap before decode.
6. **Dependencies**: none hard. B016 makes nginx's 20 MiB the outer bound
   on servers. B028 edits the same error branches; either order works.

Tests: Go unit test for `handledBodyTooLarge`; Go 13 KiB `PUT /me` is 413
(`profile_handler_test.go`); Go 65 KiB `CreateUser` is 413 with the repo
untouched (`user_handler_test.go`); pytest port of
`test_oversized_login_body_is_413`; pytest 65 KiB `POST /user` is 413.

#### Open question for Brandon

What cap should the copy-me example handlers carry, given a bulk
`POST /exampleList` can legitimately exceed 64 KiB? Recommendation: a
separate `maxEntityBodyBytes = 1 << 20` (1 MiB) in `example_handler.go`, so
derived apps inherit a cap they raise per entity.

### B012 — COMPLETED — Cross-tenant ids answered 409; unscoped writes and member count

- **Completed:** 2026-10-02. New `repository/scope.go` `zeroRowOutcome` (takes `*pgxpool.Pool`, no Querier port) explains zero-row writes with the tenant predicate in UpdateUser/DeleteUser/UpdateAppUserGroup/DeleteAppUserGroup and the three MFA writes, so another tenant's id is 404, never 409. `UpdateUserPassword`, `UpdateUserTOTPSeed`, `UpdateUserMFASetup`, `UpdateUserMFAExempt` take the loaded row's own tenant (interface, mock and callers in auth/user/user_mfa handlers updated); bulk membership writes match the tenant and wrap `ErrConflict` (stale updateTs now 409, was 500); the group-delete member count is scoped to the group's tenant (open question: took the recommendation); four row loops check `rows.Err()`. Deviations: Go and pytest cross-tenant tests use tenant 2 (doc/900: isolation is 1 vs 2), not GRC's 3; pytest probes seed groups 4/5 with `updateTs: 0` via the API (no `db` fixture until B018), and DELETE /user cross-tenant is covered only by Go repo tests (no tenant-2 user is seeded); PS0010 tick skipped (no `tickets/` folder); `ErrInUse`/`ClientMessageError` left for B028. Rule added to doc/105 (Optimistic Locking), doc/900 and src/api-srv/README.md. Tests: all green, go=278 unit=41 e2e=9 py=79

- **Priority:** P1  **Area:** go-api, tests, docs
- **GRC:** `src/internal/repository/scope.go:1-72`; `user_repository.go:220-244,246-270,307,343-397,496,550,586,595-720`; `app_user_group_repository.go:199-204,220-237,260`; `tests/test_security_context_isolation.py:24-40,228-293`; `doc/105-api-srv-tech-spec.md:503-521`; `src/api-srv/README.md:209`; `tickets/03-done/RG0014-login-layer-repository-drift/RG0014-login-layer-repository-drift.md:44-52,66,97,109,180`   **PROTO:** `src/api-srv/internal/repository/user_repository.go:186-212,214-300,405-421,457-473,478-594` (`EXISTS` :409,461,696); `app_user_group_repository.go:192-210,224-240,265-280` (:196,269); `handler/app_user_group_handler.go:240-255`; `tests/test_security_context_isolation.py:99-112,297`; `doc/900-database-guidelines.md`; `src/api-srv/README.md:141`; `tickets/01-open/PS0010-api-entity-template.md:36-102`
- **Ticket:** RG0014 / `c85313a`, `b5d15fe`, `b3a1ea2`; PS0010 drift half
- **What:** PROTO's zero-row `EXISTS` ignores tenant: another tenant's id gets
409 (existence leak); group delete counts members across tenants. GRC:
`existsInContext` 404; writes carry the row's own tenant.
- **Port notes:** 4 signatures + fakes; `rows.Err()`. Pytest via `db` (B018).
Rule into doc/900, README:141.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

All PROTO paths are under `src/api-srv/internal/`. Every user and group
`UPDATE` already carries the caller's tenant. When it matches zero rows, a
second query explains why: absent (404) or stale `update_ts` (409). That
query has no tenant (`repository/app_user_group_repository.go:196-197`):
`SELECT EXISTS(... WHERE app_user_group_id = $1 AND delete_user_id IS NULL)`.
The same check sits in `UpdateAppUserGroup` (`:196`), `DeleteAppUserGroup`
(`:269`), `UpdateUser` (`repository/user_repository.go:409`) and
`DeleteUser` (`:461`).

**(1) Existence oracle (security hole, low severity).** An admin in tenant
1 sends `PUT /appUserGroup` or `DELETE /user` with any id and `updateTs: 0`.
The scoped write misses; the unscoped `EXISTS` finds the row if it lives in
*any* tenant, so the answer is 409 "modified by another user"
(`handler/app_user_group_handler.go:239-246`), not 404. Ids are serial, so
walking them maps other tenants' ids. Nothing crosses the boundary but the
answer. It needs the user or group admin grant. **Correction:** doc/700
lists `user_repository.go:696` here; that check (`UpdateOwnProfile`) is
already scoped (`:697`). The template entity is too
(`example_repository.go:165-166`).

**(2) Group delete counts members in every tenant.**
`app_user_group_repository.go:225-227` runs `COUNT(*) ... WHERE $1 =
ANY(app_user_group_membership_list)` with no tenant, and the count reaches
the client: 409 "cannot delete group: N users are assigned to it"
(`handler/app_user_group_handler.go:342-348`).

**(3) Writes with no tenant (defence-in-depth gap, not reachable).**
`UpdateUserPassword`, `UpdateUserTOTPSeed`, `UpdateUserMFASetup`,
`UpdateUserMFAExempt` (`user_repository.go:191,220,249,278`) and the bulk
membership writes (`:508,567`) match `app_user_id` alone. Every caller
loads the row in the caller's tenant first (`handler/user_handler.go:787-809`,
`:1026`; `handler/user_mfa_handler.go:708`), so no route crosses today. A
future handler that skips the pre-check would. `UpdateUserMFAExempt` has no
production caller.

**(4) Correctness bugs in the same code.** The bulk writes return plain
`fmt.Errorf` on zero rows (`user_repository.go:519-524,578-583`), so
`errors.Is(err, repository.ErrConflict)` at `handler/user_handler.go:822`
never fires: a stale `updateTs` answers 500. Four row loops
(`user_repository.go:94,338`; `app_user_group_repository.go:47,98`) never
check `rows.Err()`, so a mid-stream failure returns a short list as success.

#### What GRC changed

RG0014 (`c85313a`, `b5d15fe`, `b3a1ea2`), login-layer repositories:

1. `src/internal/repository/scope.go:30-50` adds `existsInContext`
   (predicate always `<pk> = $1 AND security_context_id = $2 AND
   delete_user_id IS NULL`, table names are constants `:10-16`) and
   `zeroRowOutcome` (`:56-72`): false is `ErrNotFound`, true `ErrConflict`.
   Callers: `user_repository.go:496,550,586`,
   `app_user_group_repository.go:201,260`.
2. Member count scoped: `AND security_context_id = $2`
   (`app_user_group_repository.go:222-226`).
3. Bulk writes add `security_context_id = $6` (`user_repository.go:628,692`)
   and answer `ErrConflict` on zero rows, ambiguous on purpose (`:639-641`).
4. The four self-service writes take the **row's own** tenant as a new
   last parameter (`user_repository.go:220-227`), not the JWT's: after
   `/changeSecurityContext` the active tenant is not the home tenant.
   Callers pass `user.SecurityContext` (`handler/auth_handler.go:991-998`).
5. Row loops check `rows.Err()` (e.g. `user_repository.go:458`).
6. `RetrieveUserByUserName` and `GetGrantsForGroups` stay unscoped by
   design, written up in `doc/105-api-srv-tech-spec.md:503-520`.

Tests: Go `TestUpdateAppUserGroupOtherSecurityContextIsNotFound`,
`TestDeleteAppUserGroupInUse` (`app_user_group_repository_test.go:317,379`),
`TestUpdateUserPasswordOtherSecurityContext` (`user_repository_test.go:279`);
pytest `test_cross_security_context_group_is_404_not_409`
(`tests/test_security_context_isolation.py:228-293`): inserts a tenant-3
group via `db`, PUT and DELETE with its real `updateTs` both 404, row
untouched and unlisted.

#### Proposed fix for PROTO

1. **New `repository/scope.go`** with `existsInContext`/`zeroRowOutcome`.
   Trap: GRC's take `Querier` (`entity_repository.go:18`), which PROTO
   lacks; take `*pgxpool.Pool`. Keep `ExecuteWithRetry` until B028 deletes
   `retry.go`.
2. **`UpdateUser`, `DeleteUser`, `UpdateAppUserGroup`,
   `DeleteAppUserGroup`**: replace the inline `EXISTS` with `zeroRowOutcome`.
3. **Member count** (`app_user_group_repository.go:224-227`): add the
   tenant. Keep the "cannot delete group" text so
   `handler/app_user_group_handler.go:342` still matches; GRC's `ErrInUse`
   and `ClientMessageError` belong to B028.
4. **Four writes**: add `securityContext int` last in the repo,
   `UserRepositoryInterface` (`handler/user_handler.go:49-76`) and
   `MockUserRepository` (`handler/user_handler_test.go`). Callers pass the
   loaded row's `SecurityContext`, never `jwtClaims.SecurityContext`:
   `handler/auth_handler.go:835`, `handler/user_handler.go:1062`,
   `handler/user_mfa_handler.go:322,564,738` (all already hold the row).
   Password: wrap `ErrNotFound` keeping "user not found" text
   (`user_repository_test.go:264`); the other three use `zeroRowOutcome`,
   so their text asserts (`:388,399,449`) become `errors.Is`.
5. **Bulk writes**: add `AND security_context_id = $6`; wrap `ErrConflict`
   in the zero-row error, turning the dead 500 into 409.
6. **`rows.Err()`** after the four loops. Skip GRC's
   `UpdateUserAccountStatus` and `RetrieveBlockedUserNameList` (B034).
7. **Docs**: rule "the check that explains a zero-row write uses the
   write's tenant predicate; two named lookups are exempt" into
   `doc/900-database-guidelines.md` and the `internal/repository/` line at
   `src/api-srv/README.md:141`. Tick the group line of PS0010
   (`tickets/01-open/PS0010-api-entity-template.md:40-41`).

Dependencies: B028 overlaps steps 3-5, need not land first. The pytest needs
B018: PROTO has no `db` fixture or `psycopg2` and tests hit `loc_proto_db`.

Tests:
- Go: the four `zeroRowOutcome` callers and `UpdateUserPassword`, given
  another tenant's id (real `update_ts`), return `ErrNotFound`.
- pytest: port GRC's 404-not-409 test, seeding tenant 3 (never 1 or 2).
- pytest: stale `updateTs` on `/addUserListToGroup` answers 409, not 500.

#### Open question for Brandon

A user homed in tenant A can hold a tenant B group (how
`GetGrantsForGroups`, `user_repository.go:80-83`, grants a second context).
The scoped count lets B's admin delete that group while A's user still
lists it. Accept that, as GRC does? Recommendation: yes; the group is B's
and `GetGrantsForGroups` skips deleted groups, so the leftover id is inert.

### B013 — COMPLETED — Admin "Change Password" for another user always 400s

- **Completed:** 2026-10-02. `src/web-app/stores/app-user-store.ts` now posts `{ appUserId, newPassword }` to `/changeOtherUserPassword` (the handler's json tag); the parameter is renamed `appUserId` in the interface and implementation, `postOrThrow` kept. New vitest `src/web-app/tests/app-user-store.test.ts` stubs `fetch` and asserts the recorded body. New Playwright `src/web-app/tests/e2e/change-other-password.spec.ts` creates an MFA-exempt user through the API, changes its password from `/appUserEdit` via the modal, sees the success notification, signs in as that user with the new password, and deletes the user in `finally`; both tests fail on the old store (the e2e one on the API's 400 "appUserId and newPassword are required"). Tests: all green, go=278 unit=42 e2e=10 py=79

- **Priority:** P1  **Area:** web-app
- **GRC:** `stores/app-user-store.ts:346-356`; `stores/app-user-store.test.ts:39`; `tickets/03-done/RG0036-change-other-password-field/RG0036-change-other-password-field.md:28-38`   **PROTO:** `stores/app-user-store.ts:112-116`; `src/api-srv/internal/handler/user_handler.go:986`; `stores/app-user-store.ts:24-27`
- **Ticket:** RG0036 / `e6588b5`
- **What:** Store posts `userId`; handler reads `appUserId`, gets 0, 400s every
time. pytest posts `appUserId`, so only the web is wrong.
- **Port notes:** One line + fetch-stubbing vitest in `tests/`. README rule in
B076.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Web paths are under `src/web-app/`; Go paths under `src/api-srv/internal/`.
An admin with `GrantChangeOtherUserPassword` opens `/appUserEdit` for
another user and clicks "Change Password". The page calls the store
(`pages/app-user-edit-page.tsx:202`):

```ts
await changeOtherUserPassword(numericId, newPassword);
```

The store posts the id under the key `userId`
(`stores/app-user-store.ts:112-116`):

```ts
changeOtherUserPassword: (userId, newPassword) =>
  postOrThrow(API_ENDPOINTS.CHANGE_OTHER_USER_PASSWORD, {
    userId,
    newPassword,
```

The handler reads a different key (`handler/user_handler.go:985-987`):

```go
var req struct {
    UserID      int    `json:"appUserId"`
    NewPassword string `json:"newPassword"`
```

Go's JSON decoder ignores the unknown `userId` key, so `req.UserID` stays
0. The required-field check at `handler/user_handler.go:1001-1008` then
answers 400 "appUserId and newPassword are required" on every call. The
store's `postOrThrow` (`stores/app-user-store.ts:69-77`) puts that text in
the store error and rethrows; the page shows it in the modal
(`pages/app-user-edit-page.tsx:209-212`). The password never changes.

Consequence: the admin feature has never worked from the web. This is a
correctness bug, not a security hole. It fails safe: no password is set
and no other user is touched.

Why nothing caught it. The API side is right, and pytest proves it:
`tests/test_user_management.py:113` posts `{"appUserId": ...}` and gets
200. vitest only runs `tests/**/*.test.ts` (`vitest.config.ts:11`), and no
test there covers this store. No Playwright spec under `tests/e2e/`
exercises "Change Password". The `userId` key dates to before PS0001
(`8559400` carried it into the entity store factory unchanged).

The other bodies in the same store match their handlers:
`userIds`/`groupId`/`updateTsList` (`handler/user_handler.go:756-758`,
`:877-879`) and `userId` for `resetUserMFA`
(`handler/user_mfa_handler.go:694`). That handler does not read the
`updateTs` the store also sends (`stores/app-user-store.ts:119`); it is
ignored, harmless.

#### What GRC changed

GRC commit `e6588b5` (RG0036) changed one key and added a test.

1. `src/web-app/stores/app-user-store.ts:353-356` sends the key the
   handler reads:
   ```ts
   await apiJson<void>(API_ENDPOINTS.CHANGE_OTHER_USER_PASSWORD, {
     method: "POST",
     ...jsonBody({ appUserId: userId, newPassword }),
   ```
   The comment at `:351-352` names RG0036. The function shape is GRC's own
   (no `postOrThrow` there); only the key matters.
2. `src/web-app/stores/app-user-store.test.ts:38-46` replaces `fetch` with
   a recorder, calls `changeOtherUserPassword(51, ...)`, and asserts the
   body equals `{ appUserId: 51, newPassword: ... }`.

The ticket (`RG0036-change-other-password-field.md:28-38`) confirms the API
is correct and the web caller is the only fault. It also notes that
`resetUserMFA` sending `userId` is correct.

#### Proposed fix for PROTO

1. **`stores/app-user-store.ts:113-116`**: send `{ appUserId: userId,
   newPassword }`. Keep `postOrThrow`; do not copy GRC's function body,
   which predates PROTO's PS0013 `apiJson`/`postOrThrow` shape. Optionally
   rename the parameter at `:24` and `:112` to `appUserId` so the key and
   the name agree.
2. **vitest** in `tests/app-user-store.test.ts` (new). Trap: GRC keeps the
   test beside the store at `stores/`, where PROTO's
   `vitest.config.ts:11` would never run it. Import from
   `../stores/app-user-store` and stub `fetch` with `vi.stubGlobal`, as
   `tests/entity-store-cache.test.ts` already does.
3. No Go, pytest or doc change. `src/api-srv/README.md:71` lists the
   endpoint without a body; leave it.
4. **B076** (P3, docs) carries the general rule "a body key equals the Go
   json tag" into `src/web-app/README.md` and points at this test as the
   example. B013 does not wait for it.

Tests to add:
- vitest `tests/app-user-store.test.ts`: the recorded POST body is
  `{ appUserId: 51, newPassword }` to `/changeOtherUserPassword`.
- Optional Playwright in `tests/e2e/`: admin changes another user's
  password and sees the success notification.

#### Open question for Brandon

None.

### B014 — COMPLETED — Go dependency bumps: three called vulnerabilities

- **Completed:** 2026-10-02. `src/api-srv/go.mod`: jwt v5.2.1 -> v5.3.1, pgx v5.5.5 -> v5.10.0, x/crypto v0.20.0 -> v0.55.0, x/text v0.14.0 -> v0.41.0 (not `@latest`: v0.42.0 and x/crypto v0.56.0+ require `go 1.26`), tidy; `go 1.25.4` unchanged. Deviation: govulncheck (v1.8.0) also reported 19 called **standard-library** advisories against go1.25.4 (fixed by go1.25.5-go1.25.13), which the module bumps cannot clear, so `src/api-srv/go.mod` and `src/load-test/go.mod` gain `toolchain go1.25.14` (GOTOOLCHAIN=auto fetches it; the `go` minimum is unchanged; `doc/105-api-srv-tech-spec.md:9` says so). `bin/300-run-tests` `all` mode (not `go`) now runs a pinned govulncheck v1.7.0 (newest that builds on Go 1.25), installed into `tmp/tools/bin/`, over `src/api-srv` and `src/load-test`; exit 3 (called vulnerability) or any other failure (install, vuln.go.dev unreachable) fails the run with the log path; documented in `doc/902-project-commands.md`. govulncheck before: api-srv "Your code is affected by 22 vulnerabilities from 3 modules and the Go standard library" (GO-2025-3553 jwt, GO-2026-5004 pgx, GO-2026-5970 x/text, 19 stdlib); load-test "affected by 22 vulnerabilities from the Go standard library". After: both "No vulnerabilities found. Your code is affected by 0 vulnerabilities." (api-srv still lists 3 uncalled x/crypto advisories, ssh GO-2026-6354/6355 fixed only in go-1.26 v0.56.0, openpgp GO-2026-5932 no fix; neither package is imported). Not done here: a build host with `GOTOOLCHAIN=local` and an older Go ignores the toolchain line and its gate will fail on stdlib advisories (B047's toolchain check). Tests: all green, go=278 unit=42 e2e=10 py=79

- **Priority:** P1  **Area:** go-api, bin
- **GRC:** `src/go.mod:3-14`   **PROTO:** `src/api-srv/go.mod:3-30` (go :3, backoff :6, jwt :7, pgx :8, x/crypto :12, x/text :27)
- **Ticket:** RG0043 item 15; RG0014 / `9ce276b`
- **What:** govulncheck: jwt v5.2.1 GO-2025-3553 (`auth/jwt.go:69`, unauth); pgx
v5.5.5 GO-2026-5004 (`migrate.go:152`); x/text v0.14.0 GO-2026-5970
(`cmd/migrate/main.go:84`). GRC's versions clear all.
- **Port notes:** Bump jwt 5.3.1, pgx 5.10.0, x/crypto 0.55.0, x/text; tidy; go
1.26 vs host (B047); gate (B040).
- **Decision:** \[X \] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

PROTO pins three Go modules at versions a reviewer's `govulncheck` run
reported as vulnerable, and nothing in the repo runs `govulncheck`, so no
gate notices. The pins (`src/api-srv/go.mod`, 29 lines, not 30):

```
// src/api-srv/go.mod:3, 7-8, 12, 27
go 1.25.4
github.com/golang-jwt/jwt/v5 v5.2.1
github.com/jackc/pgx/v5 v5.5.5
golang.org/x/crypto v0.20.0
golang.org/x/text v0.14.0 // indirect
```

**(1) jwt v5.2.1, GO-2025-3553. Security hole, reachable unauthenticated.**
The advisory is excessive memory allocation while splitting a malformed
token header (fixed in v5.2.2). PROTO parses every `auth` cookie it
receives before it knows who the caller is:

```go
// internal/security/app_helper.go:280, 287
cookie, err := sh.r.Cookie("auth")
claims, err := auth.ValidateJWT(tokenString, sh.appServices.Config.JWTSecret)
// internal/auth/jwt.go:69
token, err := jwt.ParseWithClaims(tokenString, &Claims{},
```

Any client can send a crafted cookie to any JWT-checked route and make the
API allocate memory. The consequence is a cheap memory-pressure DoS, not a
signature bypass.

**(2) pgx v5.5.5, GO-2026-5004.** Reported reachable through the migrator's
query at `internal/migrate/migrate.go:152` (`m.conn.Query(ctx, ...`) and
the API's pool. Inputs there are the operator's own migrations, so the
exposure is lower than (1).

**(3) x/text v0.14.0, GO-2026-5970.** Indirect (pulled in by pgx). Reported
reachable via `pgx.Connect` at `cmd/migrate/main.go:84`.

**Not verified here.** `govulncheck` is not installed on this machine, so
the three advisory ids and their call paths are the reviewer's figures.
GO-2025-3553 against jwt < 5.2.2 matches the public advisory; the two
GO-2026 ids were not checked. `x/crypto v0.20.0` (`go.mod:12`, used by
`internal/auth/password.go` for argon2) had no advisory called, but it is
two years old. `cenkalti/backoff/v4` (`go.mod:6`) is used only by
`internal/repository/retry.go`, which B028 deletes.

Kind: security hole (1), dependency hygiene (2, 3), plus a DX gap: no
vulnerability scan anywhere in `bin/300-run-tests` (Go gate is `go vet` at
`:64-66` and `go test`).

#### What GRC changed

The doc's commit ref is partly wrong. `9ce276b` (RG0014) only deletes
`retry.go` and the `backoff` dependency. The version bumps came in
`5089ac0` (RG0043 item 15, "was 2.6"), and the motive was drift, not
vulnerabilities: GRC's five modules pinned different pgx and x/crypto
versions and a `go.work` workspace masked it. RG0043 md `:1019-1027`:

> `go work sync` wrote the workspace's build list into every module
> (pgx v5.10.0, x/crypto v0.55.0 ... so no runtime change), `go 1.26.0`
> in all five, `GOWORK=off go mod tidy` each.

Result in GRC `src/go.mod:3,7-8,13,29`: `go 1.26.0`, jwt `v5.3.1`, pgx
`v5.10.0`, x/crypto `v0.55.0`, x/text `v0.41.0`. GRC added no test and no
`govulncheck` gate (no file in GRC mentions it). Its UAT build copes with
an old host Go by exporting `GOTOOLCHAIN=auto`
(`infra/uat-unified001/bin/uat-grc-build:45`) and naming a toolchain
mismatch by step (`:85-90`; host Debian go is 1.24.4).

#### Proposed fix for PROTO

1. In `src/api-srv`: `go get github.com/golang-jwt/jwt/v5@v5.3.1
   github.com/jackc/pgx/v5@v5.10.0 golang.org/x/crypto@v0.55.0
   golang.org/x/text@latest`, then `go mod tidy`. PROTO has no `go.work`;
   do not copy GRC's `go work sync` step or its module path
   (`github.com/Radley-House/radley-grc`). PROTO's is
   `github.com/wbmartin/starter-api`.
2. Keep `go 1.25.4` unless tidy raises it. pgx v5.10.0 declares
   `go 1.25.0`, jwt v5.3.1 `go 1.21`, so neither forces 1.26. If x/crypto
   v0.55.0 does, update `doc/105-api-srv-tech-spec.md:9` ("Go 1.25.4")
   and `src/load-test/go.mod:3` to match.
3. Leave `cenkalti/backoff` to B028; it drops out when `retry.go` goes.
4. Run `govulncheck ./...` in `src/api-srv` and `src/load-test` and
   record a clean result in the ticket. Re-check the two GO-2026 ids then.
5. Gate: add `go run golang.org/x/vuln/cmd/govulncheck@<pinned> ./...`
   after `go vet` in `bin/300-run-tests:64-66`, pinned like B040's
   staticcheck. B040 owns the gate shape; do both together or add this
   line in B040.
6. Host toolchain: `infra/uat/bin/uat-proto-build:72-73` runs plain
   `go build` and does not set `GOTOOLCHAIN`. Only matters if step 2 raises
   the `go` line above the host's Go; then take B047's toolchain check.
   Presales `bin/600-bundle-uat` builds on the laptop, so it is unaffected.
7. Tests: none new. Existing Go suite (`bin/300 go`) proves pgx and jwt
   still behave; `internal/auth/jwt_test.go:11,45` cover `ValidateJWT`
   and `password_test.go` covers argon2 on the new x/crypto.

#### Open question for Brandon

Should `govulncheck` fail the `bin/300` gate (blocking a push on a new
upstream advisory), or only warn? Recommendation: fail, since a template
that ships known-vulnerable pins propagates them to every derived app.

### B016 — COMPLETED — UAT env example: API not bound to loopback

- **Completed:** 2026-10-02. `etc/uat-api-srv.env.example` now sets `SERVER_BIND_ADDRESS=127.0.0.1` with a comment that nginx is the only way in and `TRUSTED_PROXY_COUNT=1` depends on it; the same comment sits over the existing bind in `infra/uat/etc/uat-proto-api.env` and `infra/prd/etc/prd-proto-api.env`, and `infra/uat/etc/uat-proto-e2e-api.env` (already loopback) says why its pytest copy can trust XFF. Open question took the recommendation: `JWT_SESSION_EXPIRE_HOURS=12` in the UAT example and the UAT/prd infra API env files; local dev (`etc/loc-api-srv.env`, `etc/e2e-api-srv.env`) and the build-host e2e harness file stay 72. B088's two example-file defects fixed in the same edit (cut-off "It intentionally omits" sentence ended, empty `# --- logging / KV ---` header removed; B088's `.gitignore` part not done); `doc/330-presales-to-postsales-migration.md` env-diff paragraph reworded. Not done: the optional `bin/610-deploy-uat` guard. Already-deployed presales hosts keep their env file (bin/610 never overwrites it) and need a hand edit: add `SERVER_BIND_ADDRESS=127.0.0.1`, set `JWT_SESSION_EXPIRE_HOURS=12`, restart the API. Verified: the built API run with a test env bound to 127.0.0.1 shows `TCP 127.0.0.1:18088 (LISTEN)` in lsof; /health on loopback 200, on the LAN address no connection. Tests: all green, go=278 unit=42 e2e=10 py=79

- **Priority:** P1  **Area:** etc
- **GRC:** `etc/uat-api-srv.env.template:6,23-26`   **PROTO:** `etc/uat-api-srv.env.example:17-23,26`; `infra/uat/etc/uat-proto-api.env:29`
- **Ticket:** RG0012 / `2890e8a`
- **What:** The presales API listens on all interfaces (8088 exposed unless
firewalled); infra env already binds 127.0.0.1. GRC binds loopback, 12 h
sessions.
- **Port notes:** Add the key with GRC's comment. Brandon's call: deployed
session 12 h or 72 h? File defects B088.
- **Decision:** \[X \] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

The API's listen address is `SERVER_BIND_ADDRESS:SERVER_PORT`:

```go
// src/api-srv/cmd/server/main.go:202
Addr:    fmt.Sprintf("%s:%d", config.ServerBindAddress, config.ServerPort),
```

The key is optional and empty means every interface, by design
(`src/api-srv/internal/config/config.go:106-109`: "Empty listens on every
interface ... 127.0.0.1 is right behind nginx"). Local dev leaves it unset
on purpose so the Vite proxy and a phone on the LAN can reach it
(`etc/loc-api-srv.env:27-28`).

The presales UAT example, the file the operator copies to the host
(`doc/320-presales-uat-deploy.md:76`), never sets it:

```
# etc/uat-api-srv.env.example:20-21
SERVER_PORT=8088
ENABLE_TLS=false
```

So a presales UAT API listens on `0.0.0.0:8088` in plain HTTP. Every other
deployed env file already binds loopback:
`infra/uat/etc/uat-proto-api.env:29`, `infra/prd/etc/prd-proto-api.env:29`,
`infra/uat/etc/uat-proto-e2e-api.env:27`. `doc/330-presales-to-postsales-
migration.md:189-191` even records the gap ("without it the API binds every
interface") but only as a post-sales diff to apply later.

Consequence, unless a host firewall happens to drop 8088 (nothing in the
presales path sets one up):

1. Anyone on the network can talk to the API directly on
   `http://<host>:8088`, bypassing nginx. Passwords and the session cookie
   cross the wire unencrypted (`ENABLE_TLS=false`).
2. nginx's protections do not apply: TLS, its access logs, and the
   fail2ban jails that read those logs
   (`infra/uat/fail2ban/etc/jail.local`).
3. The caller controls `X-Forwarded-For` end to end. Today `ClientIP`
   trusts the first entry anyway (B007). After B007's fix with
   `TRUSTED_PROXY_COUNT=1`, a direct caller's own header is the "one hop"
   the API trusts, so the IP lockout and per-minute limits are spoofable
   again. B007's fix is only sound when nginx is the only way in.

Kind: security hole (exposed plaintext service, control bypass). The
example also has two broken comments, tracked as B088: `:7` stops at "It
intentionally omits" and `:38` is a `# --- logging / KV ---` header with
no keys under it.

#### What GRC changed

GRC's presales template binds loopback with a reason (RG0012, `2890e8a`,
"pre-sales UAT bundle and deploy"), at
`etc/uat-api-srv.env.template:23-26`:

```
# TLS terminates at nginx; the API listens on loopback only
ENABLE_TLS=false
SERVER_BIND_ADDRESS=127.0.0.1
SERVER_PORT=__PORT__
```

It also sets `JWT_SESSION_EXPIRE_HOURS=12` (`:6`), as do GRC's infra
`uat-grc-api.env:16` and `prd-grc-api.env:16`. Neither the RG0012 ticket
nor the commit gives a reason for 12. GRC's template is filled on the host
by its 610 (B027); PROTO's is a hand-copied example. GRC added no test.

#### Proposed fix for PROTO

1. `etc/uat-api-srv.env.example:17-23`: add `SERVER_BIND_ADDRESS=127.0.0.1`
   after `ENABLE_TLS=false`, with GRC's one-line comment ("TLS terminates
   at nginx; the API listens on loopback only"). Drop the commented
   `TLS_CERT_PATH`/`TLS_KEY_PATH` lines if Brandon agrees they never apply
   behind nginx (optional).
2. Same edit fixes B088's two comment defects in this file: finish or cut
   `:6-7`, and either fill or delete the empty `:38` section.
3. `doc/330-presales-to-postsales-migration.md:189-191`: the presales file
   no longer lacks the key; reword the "adds `SERVER_BIND_ADDRESS`" diff.
4. `bin/610-deploy-uat`: optional guard. After reading `SERVER_PORT` at
   `:56`, warn (or refuse) when the host env has no
   `SERVER_BIND_ADDRESS=127.0.0.1`. Existing presales hosts keep their old
   env file (610 never overwrites it, `doc/320:78`), so the example edit
   alone does not fix a host already deployed. Check those by hand.
5. Order: land before or with B007's `TRUSTED_PROXY_COUNT`, since that
   fix assumes only nginx reaches the API.
6. Blind-copy traps: GRC's `__PORT__`/`__UAT_DOMAIN__` placeholders belong
   to B027's host-side fill; PROTO keeps `SERVERONLY` and port 8088 until
   B027. Leave out `INVITE_TTL_HOURS`, `BLOB_ROOT`, `WORKER_*`.
7. Tests: none automated. Manual: after a deploy, `ss -ltn` on the host
   shows `127.0.0.1:8088`, and `curl http://<host>:8088/health` from
   outside fails.

#### Open question for Brandon

Should deployed sessions last 12 h (GRC) or stay at 72 h (every PROTO env
file: `etc/uat-api-srv.env.example:26`, `infra/uat/etc/uat-proto-api.env:12`,
`infra/prd/etc/prd-proto-api.env:12`)? The value is how long an issued
token lives; `GET /refresh` (`auth_handler.go:383`) issues a new one, so
it bounds a stolen cookie and an idle tab, not an active user.
Recommendation: 12 h for UAT and prd templates, keep 72 h for local dev.

### B017 — COMPLETED — pytest/requests pins carry advisories

- **Completed:** 2026-10-02. `tests/requirements.txt`: pytest 8.4.2 -> 9.0.3 (PYSEC-2026-1845, the `/tmp/pytest-of-{user}` issue), requests 2.32.5 -> 2.34.2, pytest-html 4.1.1 and pyotp 2.9.0 kept, and every transitive now pinned (urllib3 2.8.0, idna 3.20, ...; exceptiongroup/tomli/typing_extensions only below Python 3.11); psycopg2-binary left out. Deviation from the expansion: pip-audit 2.10.1 shows requests 2.32.5 **is** flagged, by PYSEC-2026-2275 / CVE-2026-25645 (predictable temp file in `extract_zipped_paths`, fixed 2.33.0), not by CVE-2024-35195, so the requests bump is a fix; the stale venv also held urllib3 2.5.0 (PYSEC-2026-141/1994/1996/1998/4175/4177) and idna 3.11 (PYSEC-2026-215). `bin/300-run-tests` rebuilds `tests/.venv` when `requirements.txt` differs from the copy saved in the venv at install time (and skips pip otherwise), so no stale package outlives a bump; the existing venv was rebuilt on Python 3.14.6 and the 3.10 install was checked too. Open question took the recommendation: pip-audit stays manual; `tests/README.md` documents each pin, the venv rebuild and how to run pip-audit. pip-audit before: "Found 4 known vulnerabilities in 2 packages" (pytest 8.4.2 PYSEC-2026-1845, requests 2.32.5 PYSEC-2026-2275); after, on `-r tests/requirements.txt` and on the rebuilt venv's freeze: "No known vulnerabilities found"; pytest-html report renders (79 Passed). Tests: all green, go=278 unit=42 e2e=10 py=79

- **Priority:** P1  **Area:** tests
- **GRC:** `tests/requirements.txt:1-5`; `tests/README.md:5-9`   **PROTO:** `tests/requirements.txt:1-4`
- **Ticket:** RG0006 / `818f592`, `ad2c1dc`
- **What:** pytest 8.4.2 carries PYSEC-2026-1845; requests 2.32.5 predates the
CVE-2024-35195 bump. GRC pins 9.0.3/2.34.2 with reasons; pip-audit clean.
- **Port notes:** Re-run pip-audit at port time; psycopg2 only with B018. Tests
P1, docs P2.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

The pytest suite pins its Python dependencies in one file:

```
# tests/requirements.txt:1-4
pytest==8.4.2
requests==2.32.5
pytest-html==4.1.1
pyotp==2.9.0
```

`bin/300-run-tests:188-195` creates `tests/.venv` once and runs
`pip install -q -r tests/requirements.txt` on every `py` run. The UAT host
runs the same script (`infra/uat/bin/uat-proto-run-tests:14` execs
`bin/300-run-tests`), so these pins are installed on a shared server too.

**(1) pytest 8.4.2, PYSEC-2026-1845. Real, low severity.** Per GRC's
`ad2c1dc` commit text: pytest through 9.0.2 on UNIX uses predictable
`/tmp/pytest-of-{user}` directories, so another local user can cause a
denial of service or possibly gain privileges; fixed in 9.0.3. On a
single-user laptop this is noise. On the UAT host, where `bin/300` runs
as the build user next to other service accounts, it is a local
privilege-escalation path. Kind: security hygiene (local attacker only).

**(2) requests 2.32.5. The doc overstates this.** doc/700 says 2.32.5
"predates the CVE-2024-35195 bump". It does not. CVE-2024-35195 (once a
`Session` sends a request with `verify=False`, later requests on it also
skip verification) was fixed in the 2.32 line, and GRC's own commit
`818f592` calls 2.34.2 "well past the 2.32.5 fix". GRC bumped from 2.31.0;
PROTO is already past it. PROTO does set `self.session.verify = False`
(`tests/conftest.py:22`), so the CVE would matter if PROTO were below the
fix, but it is not. Bumping requests is hygiene, not a fix.

**(3) Unpinned transitives.** `urllib3` is not pinned; the current venv
holds `urllib3 2.5.0`. Because the venv is reused and `pip install -r`
does not upgrade a satisfied dependency, transitives freeze at whatever
the first install picked. Whether 2.5.0 carries an advisory is not
verified here.

**Not verified here.** `pip-audit` is not installed on this machine, so
PYSEC-2026-1845 is taken from GRC's commit message, not re-run.

#### What GRC changed

RG0006 ("test gate, no source mutation", `tickets/03-done/RG0006-...`)
raised the pins in two commits:

- `818f592`: pytest 7.4.3 to 8.4.2, requests 2.31.0 to 2.34.2 for
  CVE-2024-35195.
- `ad2c1dc`: `pip-audit` flagged 8.4.2 itself, so pytest went to 9.0.3.
  pytest-html 4.1.1 was checked against pytest 9 (71 tests collected,
  report written) and kept. "`pip-audit -r tests/requirements.txt`: No
  known vulnerabilities found."

Result, `tests/requirements.txt:1-5` with a reason on each non-obvious pin:

```
pytest==9.0.3
requests==2.34.2  # RG0006: 2.31.0 carries CVE-2024-35195 ...
psycopg2-binary>=2.9.10  # conftest's `db` fixture; ...
pyotp==2.9.0  # RG0003: TOTP codes for the MFA integration tests
```

`tests/README.md:5-9` names each pin and why, and records the clean
`pip-audit`. No test pins this; it is a gate input.

#### Proposed fix for PROTO

1. `tests/requirements.txt`: `pytest==9.0.3`, `requests==2.34.2` (or
   whatever is current at port time), keep `pytest-html==4.1.1` and
   `pyotp==2.9.0`. Add a reason comment only where a pin is held for a
   cause. Do not copy GRC's requests comment as is: it blames 2.31.0,
   which PROTO never had.
2. Leave out `psycopg2-binary`. Only GRC's `db` fixture imports it, and
   PROTO's conftest has none. B018's write-up confirms B018 does not need
   it either, which retires the doc's "psycopg2 only with B018" note.
3. Run `pip-audit -r tests/requirements.txt` at port time and paste the
   result in the ticket; re-check PYSEC-2026-1845 and `urllib3` then.
4. `bin/300-run-tests:195`: add `--upgrade` (or `--upgrade-strategy
   eager`) so an existing venv picks up new transitive fixes, or rebuild
   the venv when `requirements.txt` is newer than it.
5. Docs (P2): `tests/README.md:59-61` ("pytest, requests, pytest-html")
   lists three of four packages and no versions; mirror GRC's short pin
   paragraph. B044 rewrites that README; fold this in there if B044 is
   pulled.
6. Tests: one full `bin/300 py` run on the new pins; confirm the pytest-
   html report still renders.

#### Open question for Brandon

Should `pip-audit` (and the matching `npm audit`) become a step in
`bin/300`, failing on a new advisory, or stay a manual check at bump
time? Recommendation: a manual check recorded in the ticket, since a
test-only dependency advisory should not block an unrelated push.

### B018 — COMPLETED — PS0022: enforce tenant 2 at every test harness
- **Completed:** 2026-10-02. Seed split: `sql/fixtures/local-seed.sql` stays the developer seed (tenant 1); new `test-seed.sql` (logins and groups 1-3 in 2, groups 4-5 in 3, examples in 2) and `test-tenant-lock.sql` (catalog-driven `CHECK (security_context_id <> 1)`, idempotent) are run by `bin/510` and `bin/515`. Guards: `repository/testdb_test.go` `checkScratch` + `testTenant`/`otherTenant` (repository suites and B012 isolation tests now 2 vs 3), `bin/300` scratch-name check and `TEST_API_ENV` export, `conftest.py` exits 3 on a non-test target or a login outside tenant 2, Playwright global setup throws on a non-2 login; pytest bodies dropped `securityContext: 1`; docs in CLAUDE.md, doc/900, doc/050, doc/902, tests/README.md. Deviations: handler unit tests keep tenant 1 as a mock value (hermetic, no database); the load test only drops the body `securityContext` -- its tenant-2 refusal waits for B042, since `bin/330` still targets the developer API (tenant 1). Tests: all green, go=279 unit=42 e2e=10 py=79

- **Priority:** P1  **Area:** tests, bin, go-api, docs, tickets
- **GRC:** `bin/lib/test-db-fixture-logins.sql:1-43`; `bin/lib/test-db-tenant-lock.sql:1-15`; `bin/510-reset-test-db:12-16,70-73`; `bin/310-test-repo-model:55-73`; `src/internal/testdb/testdb.go:1-49`; `src/internal/repository/testdb_test.go:1-22`; `tests/conftest.py:15-20,43-52,119,197,283-317`; `tests/requirements.txt:4`; `CLAUDE.md:124-128`; `tickets/03-done/RG0041-tests-own-databases-tenant-2/RG0041-tests-own-databases-tenant-2.md:35-42,49-55,62-70,121-122`; RG0043 md `:353-359`; `src/api-srv/README.md:216-220`   **PROTO:** `src/api-srv/sql/fixtures/local-seed.sql:23-26,31-125`; `bin/510-reset-test-db:41-46,100-101`; `bin/515-reseed-db:20-24`; `bin/300-run-tests:39-41,72,148-156,200`; `src/api-srv/internal/repository/testdb_test.go:1-48`; `tests/conftest.py:31-68,151-160`; `tests/test_auth_flow.py:25`; `tickets/01-open/PS0022-test-tenant-two.md:32-63`; `CLAUDE.md:38-43`; `CLAUDE.md:35-37`
- **Ticket:** RG0041 / `31970ae`, `78be82d`, `8e05f63`; RG0043 1.1 / `5089ac0`; closes PS0022
- **What:** PROTO seeds test logins in tenant 1 and trusts URLs. GRC: fixtures
in tenant 2 (3 = other); scratch-only `CHECK (security_context_id <> 1)`; Go
`CheckScratch` + gate `case`; conftest exits on a foreign tenant or
non-`*_e2e_db`; load test refuses tenant != 2.
- **Port notes:** Brandon's call: other tenant 3 (GRC) or one user kept in 1
(PS0022 :35-38, which the lock forbids, as does `bin/515`)? Renumber in
`local-seed.sql`; skip GRC grants; export `TEST_API_ENV`; close GRC 4.25 now;
CLAUDE.md rule. bin/go-api P2.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

CLAUDE.md:38-43 says tenant 2 is the test tenant and every automated test
runs there. Nothing in the code does that yet; PS0022 is still in
`tickets/01-open/`. This is a correctness and safety gap in the test
harness, not a hole in the shipped API.

1. **Every test login is tenant 1.** One seed file feeds both scratch
   databases (`bin/510-reset-test-db:100-101`) and the developer's own
   `loc_proto_db` (`doc/050-dev-env-init.md:30`). It puts all three logins
   in tenant 1 (`src/api-srv/sql/fixtures/local-seed.sql:85-125`):
   ```sql
   -- Users, all in context 1: admin (Admin group) and basic ...
       1,   -- security_context_id, :103
   ```
   Tenant 2 holds only groups 4 and 5 (`:57-79`); examples are tenant 1
   (`:127-161`). The header admits it (`:23-26`).
2. **pytest trusts whatever URL it is handed.** `tests/conftest.py:151-153`
   reads `TEST_BASE_URL` from the environment and `authenticate`
   (`:31-68`) never checks where the login landed. `bin/300-run-tests:200`
   sets a private API, but `TEST_BASE_URL=http://localhost:8080 pytest`
   by hand runs the whole suite against the developer's API and database,
   creating and deleting users and examples there. `tests/README.md:29-32`
   still tells people the URL comes from `etc/loc-api-srv.env`, port 8080.
3. **Go repository tests trust whatever DSN they are handed.**
   `connectTestDB` (`src/api-srv/internal/repository/testdb_test.go:31-34`)
   takes `TEST_DATABASE_URL` unchecked; only `migrate_test.go:92` checks.
   The suites write tenant-1 rows (`user_repository_test.go:61`:
   `SecurityContext: 1,`).
4. **The load test acts as tenant-1 admin on the developer's API.**
   `bin/330-test-load:17` takes `TEST_BASE_URL` from `etc/loc-api-srv.env`;
   `src/load-test/client.go:126` posts examples with `"securityContext": 1`.
5. **Assertions bake in tenant 1**, e.g. `tests/test_auth_flow.py:25`
   (`assert user["securityContext"] == 1`). Tenant literals: 49 lines in
   repository `_test.go`, 95 in handler `_test.go`, 48 in `tests/*.py`
   (grep `ecurityContext|security_context`).

Consequence: a mis-set variable turns a test run into silent writes to
real data. In a derived app with real tenant-1 data, that is data loss.

#### What GRC changed

RG0041 (`31970ae`, merged `78be82d`; `api_env` in `8e05f63`) followed a
real incident: tests had written into GRC's real database (469 logins,
710 audit rows cleaned by hand; RG0041 md `:32-45`). RG0043 1.1
(`5089ac0`) then moved the fixture logins out of migrations.

1. **Fixture logins live in test setup only.**
   `bin/lib/test-db-fixture-logins.sql:15-43` puts groups 1-3 and
   admin/basic/mfa in tenant 2, groups 4-5 in tenant 3, ids unchanged so
   the tests' group ids hold. `bin/510-reset-test-db:70-72` runs it.
2. **Scratch databases refuse tenant 1.**
   `bin/lib/test-db-tenant-lock.sql:7-15` raises if any tenant-1 row
   exists, then adds `CHECK (security_context_id <> 1)` to every table
   with that column. Test databases only; never a migration.
3. **Go gate.** `src/internal/testdb/testdb.go:31-42` (`CheckScratch`)
   fails any DSN not ending `_test_db`/`_e2e_db`; `URL` (`:18-29`) calls
   it. `src/internal/repository/testdb_test.go:19-22` defines
   `testTenant = 2`, `otherTenant = 3`. `bin/310-test-repo-model:65-68`
   repeats the suffix check in shell. Pinned by
   `src/internal/testdb/testdb_test.go:5-17` (`TestCheckScratch`).
4. **pytest aborts.** `tests/conftest.py:43-52` (`require_test_tenant`)
   calls `pytest.exit(..., returncode=3)` when a login reply names any
   tenant but 2; wired at `:119` (`/auth`) and `:197` (`/mfaVerify`).
   `db_url` (`:300-309`) exits on a non-`_e2e_db` database. `api_env`
   (`:287-297`) reads the test API's own env file.
5. **Load test** refuses a login outside tenant 2
   (`src/load-test/client.go:123-129`).
6. **Rule** in GRC `CLAUDE.md:124-128`.

Open in GRC (RG0043 md `:353-359`, section 4 item 25): the tenant guard
fires only after a login, so a hand-set `TEST_BASE_URL` still reaches the
real API. Its fix (refuse any port but the test API's) is unimplemented.

#### Proposed fix for PROTO

1. **Split the seed.** Keep `sql/fixtures/local-seed.sql` as the dev
   seed (tenant 1, `doc/050:30` uses it). Add
   `sql/fixtures/test-seed.sql`: same three users, same hashes, PROTO's
   grant arrays (`local-seed.sql:51-53,76-77`), groups 1-3 and users in
   tenant 2, groups 4-5 in tenant 3, example rows in 2. Blind-copy traps:
   GRC strips grants 1001-1004 and writes `account_status`; PROTO has
   neither change (the column is B035). Ids stay fixed.
2. **Lock.** Add `sql/fixtures/test-tenant-lock.sql` from GRC's file
   unchanged. `bin/510:101` runs test-seed then lock. `bin/515-reseed-db`
   loads test-seed instead of local-seed: TRUNCATE keeps the CHECK, so a
   seed with tenant-1 rows would now fail there. Fix 515's stale "the two
   users" (`:17`).
3. **Go.** In `repository/testdb_test.go`, `t.Fatal` unless the DSN name
   ends `_test_db`/`_e2e_db` (a `checkScratch` helper; no new package,
   module path is `github.com/wbmartin/starter-api`). Add
   `testTenant`/`otherTenant`; replace tenant literal `1` in repository
   and handler tests. Add the suffix `case` to `bin/300:72`.
4. **pytest.** `TEST_TENANT = 2`, `OTHER_TENANT = 3`; call
   `require_test_tenant` after `conftest.py:38`. Before any request,
   refuse unless `TEST_API_ENV` is set and `TEST_BASE_URL` equals that
   file's `TEST_BASE_URL` (closes GRC 4.25 here). `bin/300` exports
   `TEST_API_ENV="$PY_API_ENV"` (`:148`). Rewrite `tests/README.md:29-32`.
   Drop request-body `"securityContext": 1`; flip assertions.
   `test_security_context_isolation.py` becomes "tenant 2 cannot use
   tenant 3's groups"; groups 4/5 keep their ids. No psycopg2: PROTO's
   pytest has no `db` fixture.
5. **Load test.** `src/load-test/client.go`: refuse a login outside 2,
   drop the body's `securityContext`; retargeting `bin/330` is B042.
6. **Docs.** Update `local-seed.sql:23-26`, `doc/050:16-17`, close
   PS0022 (rewrite its `:34-39` and `:58-62` to the chosen shape).

Depends on nothing; B019 reuses `TEST_API_ENV`. Tests: Go
`TestCheckScratch` table (accept both suffixes, refuse `loc_<slug>_db`,
`*_test_db_old`); negative runs: a tenant-1 seed row fails 510, pytest
on 8080 exits 3, Go on the dev DSN fails.

#### Open question for Brandon

GRC's shape has no tenant 1 in test databases (CHECK `<> 1`, isolation
proved between 2 and 3); PS0022 (`:36-38`) keeps one tenant-1 user so
isolation is proved between 1 and 2. The two cannot coexist: the lock
forbids that user. Which? Recommendation: GRC's shape, because a test
database that can hold tenant 1 cannot catch a test that silently writes
there, and 2-vs-3 proves the same isolation property.
Brandon: accept reccomendation.

### B019 — COMPLETED — MFA pytest harness: per-run XFF, throwaway lockout user
- **Completed:** 2026-10-02. `tests/conftest.py` gains `api_env()`, `failed_login_threshold()`, `new_forwarded_ip()` (198.18.0.0/15, a /24 per run from the start second plus a per-call host counter) and `forwarded_for()`, and the session-start guard now exits (status 3) unless the test API's env sets `TRUSTED_PROXY_COUNT=1`; `test_auth_flow.py` (B007/B010 cases) uses them instead of its own `run_ip`. `test_zz_mfa_totp.py` sends a per-test forwarded address on every /auth, /mfaSetup, /mfaVerify, uses `wrong_code()` instead of "000000", and its lockout case enrols a throwaway `mfalock<hex>` user (pwHash), asserts exactly threshold 401s then 429 with the lockout message (distinct from "Rate limit exceeded", so no Go change) and deletes it in `finally`; header caveat dropped, `zz` name kept as history only. Deviation: GRC's MFA client helpers (`MFARequired`, `begin_auth` etc.) and its 11-case split were not ported; the existing four cases already cover the portable behaviour, and `tests/README.md` now documents bin/300-only runs, the proxy guard and the forwarded-address helper. `bin/300-run-tests py` run twice back to back, both 79 passed. Tests: all green, go=279 unit=42 e2e=10 py=79

- **Priority:** P1  **Area:** tests
- **GRC:** `tests/conftest.py:55-68,87-137,161-206`; `tests/test_zz_mfa_totp.py:1-26,38-61,68-81,91-110,116-192,237-264,417-495,498-503`; `tests/README.md:90-97`   **PROTO:** `tests/conftest.py:31-68` (`pytest.fail` :49-50); `tests/test_zz_mfa_totp.py:1-155` (:28-45, :49-108, default 7 at :122); `src/api-srv/internal/security/app_helper.go:308-311`
- **Ticket:** RG0003 / `37978ce`; RG0042 / `63ccf75`; RG0043 / `dd8b4c3`
- **What:** PROTO locks out loopback and the shared `mfa` user, so re-runs fail.
GRC: MFA client, fresh XFF per test, disposable lockout user, threshold via
`api_env`, nine cases.
- **Port notes:** Prereq for B001/B009/B010 tests. Relies on XFF trust
(appendix). Case split restructures PS0020 behaviour.
- **Decision:** \[X \] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

A test-harness quality gap, not a product bug: the MFA file spends shared
lockout state and cannot tell a lockout from a rate limit.

**(1) The lockout test locks loopback and the shared `mfa` user.** The
file says so itself (`tests/test_zz_mfa_totp.py:3-4`): "the lockout case
spends the shared IP's failed-attempt allowance, which would lock every
later test out of /auth". The test sends wrong codes as the seeded `mfa`
user from 127.0.0.1 (`:131-135`), then checks `/auth` is locked (`:142-145`).
The server keys both counters on that IP and that name
(`src/api-srv/internal/handler/auth_handler.go:67,71`):
```go
ipCount, err := kv.Incr("loginfail-ip-"+ip, ...
userCount, err := kv.Incr("loginfail-user-"+username, ...
```
So after the test, `loginlock-ip-127.0.0.1` and `loginlock-user-mfa` live
for `FAILED_LOGIN_LOCKOUT_DURATION` (5 min, `etc/e2e-api-srv.env:31`).

**(2) When that bites.** `bin/300-run-tests py` rebuilds the database
(`bin/300-run-tests:138`) and starts a fresh API (`:169`), and the KV store
is in-process, so a `bin/300` re-run starts clean. The doc's "re-runs fail"
is true only for the manual mode in `tests/README.md:18-19` (pytest against
a long-running `bin/215` API). There, a re-run within 5 minutes fails at
the first fixture: `admin` signs in from 127.0.0.1 and gets 429. A run that
fails mid-file leaves `mfa` enrolled, so the next run fails at `:60`
(`mfaAlreadySetup is False`) until the DB is reset by hand.

**(3) One bucket for the whole suite.** Every test comes from 127.0.0.1
and nothing clears the IP counter on success (`auth_handler.go:246` is
commented out, B010), so the suite stays under 7 failures by luck.

**(4) The lockout assertion is loose.** `/mfaSetup` and `/mfaVerify` are
rate-limited per IP (`user_mfa_handler.go:384`; key from `ClientIP` at
`security/app_helper.go:360`, 20 per minute). The file sends 16 such
requests from 127.0.0.1. The test only checks `assert 429 in statuses` and
`statuses[-1] == 429` (`test_zz_mfa_totp.py:138-139`), and the rate limit
also answers 429 ("Rate limit exceeded", `app_helper.go:380`). A rate-limit
429 would pass as a lockout. Minor: the wrong code is the literal "000000"
(`:66,134`), which is a valid code about once in a million.

**(5) No MFA client.** `TestClient.authenticate` fails on any temp token
(`tests/conftest.py:49-50`), so each MFA step is a hand-rolled request,
and the threshold comes from pytest's env with a default of 7
(`test_zz_mfa_totp.py:122`), not from the server's config.

#### What GRC changed

RG0003 (`37978ce`) rewrote the file after a fixed-address draft failed a
back-to-back run; RG0042 (`63ccf75`), RG0043 (`dd8b4c3`) added cases.

1. **Client helpers** in `tests/conftest.py`: `MFARequired` carries the
   temp token (`:55-68`); `authenticate(..., totp_secret, headers)`
   completes `/mfaVerify` itself (`:87-117`); `begin_auth`, `mfa_setup`,
   `mfa_verify`, `adopt_session` (`:161-206`). `fresh_code` (`:30-40`)
   never reuses a TOTP step, needed once replay is refused (B009).
2. **Per-run client addresses** (`tests/test_zz_mfa_totp.py:45-58`):
   ```python
   RUN_OCTET = random.randint(0, 255)
   def run_ip(host: int) -> str:
       return f"198.18.{RUN_OCTET}.{host}"
   ```
   One address per test in 198.18.0.0/15, sent as `X-Forwarded-For` on
   every request that can count (`fwd`, `:64-65`).
3. **Throwaway lockout user** (`:417-495`): admin creates
   `mfalock<uuid>`, enrols it, sends exactly `threshold` wrong codes and
   asserts each is 401, then asserts 429 with the lockout message (not the
   rate-limit one) and that `/auth` is locked too; `finally` deletes it.
4. **Threshold from the server's env file** via `api_env`
   (`:498-503`, `conftest.py:287-297`). `wrong_code` (`:68-81`) uses a
   code from minutes ago instead of "000000".

Correction: the doc says "nine cases". RG0003 shipped nine; the file now
has 11 (RG0042 added `:194`, RG0043 `:266`). **Trap:** GRC still trusts
the first XFF entry (`src/internal/security/client_ip.go:25-30`), so its
harness works only because its server is still spoofable.

#### Proposed fix for PROTO

B007 makes `ClientIP` take the entry `TRUSTED_PROXY_COUNT` hops from the
right, `RemoteAddr` only at 0 (local dev). At 0 every pytest request keys
on 127.0.0.1 whatever it sends, so per-test XFF needs one trusted hop.

1. **`bin/300-run-tests:153`**: add `TRUSTED_PROXY_COUNT=1` to the sed that
   builds `py-api-srv.env`, so pytest plays the proxy. Safe: that instance
   binds 127.0.0.1 (`etc/e2e-api-srv.env:58`) and serves only the scratch
   DB. Leave `etc/e2e-api-srv.env` (Playwright) and `etc/loc-api-srv.env`
   at 0. A request without the header still keys on `RemoteAddr`.
2. **`tests/conftest.py`**: port `MFARequired`, the `authenticate`
   signature, `begin_auth`, `mfa_setup`, `mfa_verify`, `adopt_session`,
   `fresh_code`. Leave out `require_test_tenant` (B018) and the GRC `db`,
   `grc` fixtures. Port `api_env`, reading the file named by an exported
   `TEST_API_ENV` (B018 exports it), not GRC's hard-coded
   `etc/loc-test-api-srv.env`. Have conftest fail fast when
   `TRUSTED_PROXY_COUNT` in that file is not 1.
3. **`tests/test_zz_mfa_totp.py`**: split the two PROTO tests into GRC's
   cases with `fwd(run_ip(n))` on every counted request, `wrong_code`,
   `failed_login_threshold()`. The throwaway user must send `"pwHash"`,
   not GRC's `"password"`: PROTO's `CreateUserRequest` maps `pwHash`
   (`src/api-srv/internal/model/security_model.go:111`) until B038.
4. **Cases that wait on other items**: enrolled login gets 409 (B001;
   today PROTO asserts the leak, `test_zz_mfa_totp.py:103`); code replay
   (B009); counters reset on success and grant 9020 (B010; PROTO checks
   `GrantUserUpdate`, `user_mfa_handler.go:684`). Port each with its item.
   Portable now: temp token not a session, enrolment, wrong code keeps
   the token, right code gives a session, temp-token replay, admin reset
   (PROTO's "TOTP secret not found..." matches, `user_mfa_handler.go:487`),
   throwaway lockout.
5. **`tests/README.md`**: pytest runs through `bin/300` (or an API with
   `TRUSTED_PROXY_COUNT=1`); fix or drop the manual mode at `:18-19`.

Tests: pytest as above; Go, B007's `ClientIP` table test; pytest, after a
lock on X, `X-Forwarded-For: Y, X` is still 429 (forged prefix ignored).

#### Open question for Brandon

Should the manual "pytest against your own `bin/215` API" mode stay
supported (which forces `TRUSTED_PROXY_COUNT=1` into `etc/loc-api-srv.env`
too), or does pytest run only through `bin/300`? Recommendation: `bin/300`
only, so no developer config ever trusts a forged header.
Brandon: Bin/300

### B020 — COMPLETED — Nightly backup cannot write its dump
- **Completed:** 2026-10-02. `infra/prd/bin/prd-proto-backup-all` now dumps first with `sudo -u postgres pg_dump -Fc ... > file` (root's shell opens the file, no second gzip, a failed dump exits non-zero), gzips the globals as `.sql.gz`, makes `BACKUP_ROOT/prd-proto` 0700, drops VACUUM, tars the app root without `backup/` and `stg/`, and ships all three files in one `scp` to `REMOTE_SERVER:REMOTE_DIR/`. `REMOTE_DIR` added to `infra/prd/etc/prd-proto-backup.env` and `infra/README.md` (table row, step 9); `etc/` still ships in clear per Brandon. B045 is not done yet, so `BACKUP_ENV` stays; no UAT copy exists. Verified in ubuntu:24.04 with postgresql 16: old script fails `Permission denied` on the dump; new script writes a dump that `pg_restore --list` reads, dir `root:root 700`, and a missing database exits 1 with no scp or check-in. Tests: all green, go=279 unit=42 e2e=10 py=79

- **Priority:** P1  **Area:** infra
- **GRC:** `infra/prd-unified002/bin/prd-grc-backup-all:6-8,27-49`; `infra/prd-unified002/etc/prd-grc-backup.env:1-8`   **PROTO:** `infra/prd/bin/prd-proto-backup-all:13-40` (:22, :33-34); `infra/prd/etc/prd-proto-backup.env:1-8`; `infra/README.md:93,185-187`
- **Ticket:** RG0013 / `3a3fc94`, `a57d6ad`
- **What:** `su - postgres -c "pg_dump | gzip > FILE"` writes as postgres into a
root-made dir: permission denied. GRC: root writes `sudo -u postgres pg_dump -Fc`, dir 0700, DB first, excludes `backup`/`stg`, `REMOTE_DIR`.
- **Port notes:** `APP_ENV` from app root (B045); drop blob wording; encryption
in B050.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

`infra/prd/bin/prd-proto-backup-all` runs as root from cron every night
(`infra/README.md:93,185-187`; `NACK_CRON=30 7 * * *` in
`infra/prd/shrmpl-etc/nack-monitors/prd-proto-backup-all_NACKID.env`). It
creates its output directory as root and then asks the `postgres` user to
write into it.

1. `infra/prd/bin/prd-proto-backup-all:14,22`: root makes the directory.

       BACKUP_DST=$BACKUP_ROOT/$BACKUP_ENV
       mkdir -p "$BACKUP_DST"

   With `BACKUP_ROOT=/backup` and `BACKUP_ENV=prd-proto`
   (`infra/prd/etc/prd-proto-backup.env:4-5`) that is `/backup/prd-proto`,
   owned `root:root`, mode 0755 under root's default umask 022.
2. `:24-30`: the filesystem tarball is written (as root, so it works) and
   already shipped with `scp` to `REMOTE_SERVER`.
3. `:33-34`: the database dumps.

       su - postgres -c "pg_dump -Fc $DATABASE_NAME | gzip > $DB_BACKUP_FILE"
       su - postgres -c "pg_dumpall --globals-only | gzip > $DB_GLOBALS_FILE"

   The whole quoted string, redirect included, runs in a shell owned by
   `postgres`. That shell must create `$DB_BACKUP_FILE` inside the
   root-owned 0755 directory and cannot: it prints `Permission denied` for
   the `.dump.gz` path, `gzip` never starts, `pg_dump` dies on a broken
   pipe, and the pipeline exits 1. `su` passes the 1 back, and
   `set -euo pipefail` (`:2`) stops the script on line 33.

What the operator sees: no database dump, ever, on a host built from the
README. The curl to `HEALTHCHECK_URL` (`:39`) is never reached, so nackmon
alerts every morning. Only the filesystem tarball leaves the box. Nothing
in `infra/README.md` step 9 or `doc/` chowns `/backup` to postgres, so the
failure is deterministic. Correctness bug: prod has no database backup.

Smaller problems in the same script:

- `:25,28`: `cd /srv/prd-proto; tar -czf ... .` tars the whole app root,
  including `stg/` (registry pulls, build output) and `backup/`
  (pre-migrate dumps, `prd-proto-install-api:47-48`), large and not needed
  to rebuild the host.
- The tarball holds `etc/` (every filled env file: DB passwords, JWT key,
  registry credentials). It lands world-readable (0644 in a 0755 dir) in
  `/backup/prd-proto`. That is a local-disclosure gap (B050 owns
  encryption; the directory mode belongs here).
- `:19,33`: `pg_dump -Fc` is already compressed; piping it through `gzip`
  makes a `.dump.gz` that `pg_restore` cannot open until it is gunzipped.
- `scp` lands in the remote user's home (`"$REMOTE_SERVER:~/"`,
  `:30,36-37`); there is no directory key.

#### What GRC changed

GRC's script runs as root and does every write as root; only `pg_dump`
itself runs as postgres, streaming to root's stdout.

- `infra/prd-unified002/bin/prd-grc-backup-all:17-19` derives the env name
  from the app root (`APP_ENV="$(basename "$SRV_ROOT")"`) and the env file
  from that; `:23` requires `REMOTE_DIR` and no `BACKUP_ENV`.
- `:33`: `mkdir -p "$DST"; chmod 0700 "$DST"`. Only root reads them.
- `:36-37`, the actual fix:

      sudo -u postgres pg_dump -Fc "$DATABASE_NAME" > "$DB_FILE"
      sudo -u postgres pg_dumpall --globals-only | gzip > "$GLOBALS_FILE"

  The redirect is parsed by root's shell, so root opens the file. The
  custom-format dump is written as plain `.dump` (no second gzip); the
  globals, which are SQL text, are gzipped.
- `:35-43`: database first, then files, so every blob a dumped row names
  is in the tarball (`:12-13`). PROTO has no blobs; the order is harmless.
- `:42`: `tar -C "$SRV_ROOT" --exclude=./backup --exclude=./stg -czf ...`.
  The excludes are the in-root `backup/` and `stg/`, not `BACKUP_ROOT`.
- `:46`: one `scp` of all three files to `"$REMOTE_SERVER:$REMOTE_DIR/"`.
- `VACUUM` (PROTO `:35`) is gone. `prd-grc-backup.env:1-8` adds
  `REMOTE_DIR` and drops `BACKUP_ENV`.

Why: commit `3a3fc94` (RG0013) built the GRC mirrors from PROTO's. The
RG0013 ticket does not list the backup among its "Corrections to
ProtoStack's version" (`RG0013 md:50-58`); the intent is in the script
header (`:1-13`). It has run on a real host: GRC
`doc/331-prd-postsales-cutover.md:283-288` runs it once by hand and checks
the three files arrived; `RG0043 md:615` marks the prod backup done on
2026-10-02. No automated test covers it.

#### Proposed fix for PROTO

1. `infra/prd/bin/prd-proto-backup-all:33-34`: replace the two `su - postgres
   -c "..."` lines with `sudo -u postgres pg_dump -Fc "$DATABASE_NAME" >
   "$DB_BACKUP_FILE"` and `sudo -u postgres pg_dumpall --globals-only | gzip
   > "$DB_GLOBALS_FILE"`. Rename the dump suffix `.dump.gz` to `.dump`
   (`:19`) and the globals to `.sql.gz` (`:20`).
2. `:22`: add `chmod 0700 "$BACKUP_DST"` after the `mkdir`.
3. Move the database block (`:32-35`) above the files; drop `VACUUM`.
4. `:28`: `tar -C "$BACKUP_SRC" --exclude=./backup --exclude=./stg -czf ...`
   and remove the `cd` at `:25`.
5. Add `REMOTE_DIR` to the required list (`:9`), to
   `infra/prd/etc/prd-proto-backup.env` (as `SERVERONLY`) and to the README
   table row (`infra/README.md:93`); ship the three files in one `scp` to
   `"$REMOTE_SERVER:$REMOTE_DIR/"`.
6. If B045 is taken: derive `APP_ENV` from `basename "$SRV_ROOT"`, drop
   `BACKUP_ENV` from the env file, the required list and README `:93`.
   Without B045, keep `BACKUP_ENV`; the fix in steps 1-2 stands alone.

Blind-copy traps: drop GRC's header (`:1-13`: blob store,
`TOTP_ENCRYPTION_KEY`, A-004/A-007; none exist in PROTO) and its database
and file names; PROTO's derive from `etc/app.env` via `bin/000-rebrand`.
`sudo` must be installed (Ubuntu default); root needs no sudoers entry.

Depends on nothing. Related: B045 (names), B050 (`umask 077`, encrypt the
tarball), B026 (runbook: cron line, remote directory).

Tests: no harness covers `infra/`. `shellcheck` the script; on a prd host
run it once as root, then `pg_restore -l <file>.dump` must list the schema
and `/backup/prd-proto` must be root-owned 0700.

#### Open question for Brandon

Should the nightly tarball keep shipping `etc/` (secrets) in clear to
`REMOTE_SERVER` until B050's encryption lands? Recommendation: yes, ship
this fix now with the 0700 directory, because today there is no database
backup at all and the secrets already travel in clear.
Brandon: yes

### B021 — COMPLETED — Cert renewal: non-interactive; key not owned by www-data
- **Completed:** 2026-10-02. `infra/uat/bin/uat-proto-renew-web-certs` and the prd copy now run certbot with `--agree-tos -n`, copy the pem files with `install` (dir root:www-data 0750, `fullchain.pem` root:www-data 0640, `privkey.pem` root:root 0600), run `nginx -t` before the reload, and carry a header (root, cron, nackmon, why `--force-renewal`). Deviation: the key is root-only 0600 rather than GRC's 0640, since the nginx master reads it as root and www-data needs no access. Both nginx site headers say to run the script before linking, and `infra/README.md` step 6 is reordered (copy site, run once, check modes, link, `nginx -t && reload`, then cron); B045 is not done, so the copies still differ only in the env name. Verified in ubuntu:24.04 with a fake certbot and Let's Encrypt tree: old script exits 1 with EOFError on a closed stdin and leaves the key www-data-owned; new script fixes ownership on an old www-data crt dir, www-data cannot read the key, and `nginx -t` passes with each site linked (fails if linked first). Tests: all green, go=279 unit=42 e2e=10 py=79

- **Priority:** P1  **Area:** infra
- **GRC:** `infra/uat-unified001/bin/uat-grc-renew-web-certs:1-29`; `infra/uat-unified001/etc/nginx_uat-grc.atticustechnology.com:1-6`   **PROTO:** `infra/uat/bin/uat-proto-renew-web-certs:15-25` (:16, :21, :24); `infra/prd/bin/prd-proto-renew-web-certs`
- **Ticket:** RG0013 / `3a3fc94`
- **What:** No `--agree-tos -n` (first issue hangs under cron); `chown -R www-data` hands over `privkey.pem`. GRC: flags, `root:www-data` 0750/0640,
`nginx -t` before reload.
- **Port notes:** Drop-in. Keep the note: renew once before linking the site.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

`infra/uat/bin/uat-proto-renew-web-certs` (and the prd copy, identical but
for the env filename at `:6`) runs as root from cron on the 3rd of each
month (`NACK_CRON=30 3 3 * *` in both `*-renew-web-certs_NACKID.env`). It
asks certbot for a certificate, copies it into `crt/<site>/`, which is
what the nginx site reads (`infra/uat/etc/nginx_uat-proto.example.com:25-26`),
and reloads nginx. Three lines bite.

1. Interactive certbot, `infra/uat/bin/uat-proto-renew-web-certs:16`:

       /usr/bin/certbot --nginx certonly -d "$SITE_NAME" \
         --email "$CERTBOT_EMAIL" --force-renewal #--quiet

   There is no `--agree-tos` and no `-n` (`--non-interactive`). On a host
   where certbot has never registered an ACME account, certbot asks the
   operator to accept the Let's Encrypt Terms of Service. That question is
   forced interactive (certbot 5.8 `_internal/main.py:732`,
   `yesno(msg, cli_flag="--agree-tos", force_interactive=True)`), so certbot
   reads stdin even with no terminal. Cron gives the job no input: the read
   hits end-of-file and certbot dies with an `EOFError`. It waits forever
   only when stdin is held open (an ssh session, a hook runner). So the
   item's "hangs under cron" is wrong in detail; the outcome is the same:
   no certificate, `set -e` stops the script, the nackmon check-in (`:25`)
   is skipped and the operator learns of it from the alert.
   `infra/README.md:168-170` (step 6) says to link the site and put the
   script in cron; it never says to run it once by hand.
2. Wrong order on a fresh host. README step 6 links the nginx site first.
   The site names `crt/<site>/fullchain.pem` and `privkey.pem`, which do
   not exist until this script has run once, so `nginx -t` fails on the
   missing files. A failing config blocks every later reload of the
   host's shared nginx, and a restart (reboot) would leave every site on
   the box down. PROTO's nginx header (`:1-3`) does not warn about this.
3. Key ownership, `:20-21`:

       cp "/etc/letsencrypt/live/$SITE_NAME/"{fullchain.pem,privkey.pem} ...
       chown -R www-data:www-data "$CRT_FOLDER_PATH/$SITE_NAME"

   The TLS private key ends up owned by `www-data`. The nginx master
   process reads keys as root, so www-data needs no access at all. As
   owner, any process running as www-data (nginx workers, and on a shared
   host other apps' presales units; `infra/README.md:24` lists
   `User=www-data` for presales) can read the key, rewrite it, or replace
   the certificate. This is a security weakness: the site's TLS identity
   is exposed to the least-trusted service account on the box.
4. No explicit config test, `:24`: `systemctl reload nginx` runs straight
   after the copy. This is hygiene, not a live bug: on the stock
   Debian/Ubuntu unit `ExecReload` is `nginx ... -s reload`, which parses
   the config first and exits non-zero on an error, so `set -e` already
   stops the script (not checked on a target host).

Summary: a correctness bug (unattended first issue fails), a doc ordering
bug, a security weakness (key owner), and a missing explicit `nginx -t`.

#### What GRC changed

GRC's script is the same shape with these fixes
(`infra/uat-unified001/bin/uat-grc-renew-web-certs`, byte-identical to the
prd copy because names derive from the app root, `:11-12`, B045):

- `:22`: `certbot --nginx certonly -d "$SITE_NAME" --email
  "$CERTBOT_EMAIL" --agree-tos -n --force-renewal`. Never prompts.
- `:25`: `chown -R root:www-data "$CRT_DIR"; chmod 0750 "$CRT_DIR";
  chmod 0640 "$CRT_DIR"/*.pem`. Root owns the files; www-data can read
  but not change them; other users cannot list the directory.
- `:26-27`: `nginx -t` before `systemctl reload nginx`: a bad config
  stops the script with nginx's own error before any reload is tried.
- `:28`: check-in output to `/dev/null`; `:5` documents why
  `--force-renewal` stays (a monthly reissue, so a failure is never
  mistaken for "not due yet").
- The nginx site header (`etc/nginx_uat-grc.*:2-4`): run the script "once
  before linking this file, or `nginx -t` fails on the missing pem."

Why: commit `3a3fc94` (RG0013) wrote the scripts; the RG0013 ticket does not
single out the cert changes, so intent comes from the comments and the
runbooks GRC executed on both hosts. GRC
`doc/331-prd-postsales-cutover.md:216-253` pulls the first certificate by
hand with `--agree-tos -n --dry-run`, then for real, sets `root:www-data
0750/0640`, and links the site only if `nginx -t` passes (removing the link
otherwise). Its reason (`:217-220`): certbot's nginx plugin edits and
reloads the live nginx every app on the host shares, and a failed first
issue can leave it down. The cron script "takes over from the first
renewal". No automated test covers it.

#### Proposed fix for PROTO

1. `infra/uat/bin/uat-proto-renew-web-certs:16` and the prd copy: add
   `--agree-tos -n`; delete the dead `#--quiet`.
2. `:21` in both: replace with `chown -R root:www-data "$CRT_DIR"; chmod
   0750 "$CRT_DIR"; chmod 0640 "$CRT_DIR"/*.pem`.
3. `:24` in both: put `nginx -t` on the line before the reload.
4. Header comment in both scripts: root, cron, nackmon, why
   `--force-renewal`.
5. `infra/uat/etc/nginx_uat-proto.example.com:1-3` and the prd site: add
   "run `*-renew-web-certs` once before linking this file, or `nginx -t`
   fails on the missing pem."
6. `infra/README.md:168-170` (step 6): reorder to: copy the site file;
   run the renew script once by hand (or the `--dry-run` then real
   `certbot` pair); check `ls -l crt/<site>`; link; `nginx -t && systemctl
   reload nginx`; then the cron line. The full runbook is B026.
7. If B045 is taken, derive `DEPLOY_ENV` from `basename "$SRV_ROOT"` so
   the two copies become byte-identical; otherwise keep `:6` as is.

Blind-copy traps: GRC's site filename and `SITE_NAME` are real hostnames;
PROTO keeps `uat-proto.example.com` and `proto.example.com`. Keep PROTO's
variable names (`CRT_FOLDER_PATH` or rename to `CRT_DIR`, either is fine)
and `CERT_HEALTHCHECK_URL`, which already matches. Do not import GRC's
`doc/310-api-srv-cert-renewal.md`; it is a GRC laptop note.

Depends on nothing. Related: B045 (name derivation), B026 (runbook), B048
(the same nginx files).

Tests: none automated. `shellcheck` both scripts; `diff` the uat and prd
copies (only `:6` may differ without B045); on a host, run once and check
`stat -c '%U:%G %a' crt/<site> crt/<site>/*.pem` shows `root:www-data 750`
and `640`.

#### Open question for Brandon

None.

### B022 — COMPLETED — nginx: security headers per web location, none at server level
- **Completed:** 2026-10-02. The four browser security headers moved out of the server block into `location /` in `infra/uat/etc/nginx_uat-proto.example.com` and `infra/prd/etc/nginx_proto.example.com` (`/assets/` and `= /index.html` keep theirs); `/api/` sets none, and one comment before it states the rule, replacing the old repeat-in-every-location comment. `doc/105-api-srv-tech-spec.md` notes that nginx adds none of the API headers on `/api/`. No deviation; B048 proxy lines left alone. Verified in ubuntu:24.04 with dummy certs and a stub API on 8088 sending SAMEORIGIN and HSTS with preload: old conf gave `/api/x` two X-Frame-Options (DENY and SAMEORIGIN), two HSTS and two nosniff; new conf gives one of each (the API's), while `/`, an SPA route, `/favicon.ico` and `/assets/` still carry all four; `nginx -t` passes for both sites. Tests: all green, go=279 unit=42 e2e=10 py=79

- **Priority:** P1  **Area:** infra
- **GRC:** `infra/uat-unified001/etc/nginx_uat-grc.atticustechnology.com:36-42,69-76`   **PROTO:** `infra/uat/etc/nginx_uat-proto.example.com:33-36,69-71` (+ prd); `src/api-srv/internal/security/security_headers.go:32-34`
- **Ticket:** RG0043 1.5 / `535e7cb`
- **What:** Server-level headers inherit into `/api/`: duplicates, and a relaxed
header gets overridden by `DENY`. GRC moves them into `location /`.
- **Port notes:** Add to `location /` when removing :33-36.
- **Decision:** \[X \] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Two layers set browser security headers, and on `/api/` both do.

1. nginx, server level. `infra/uat/etc/nginx_uat-proto.example.com:33-36`
   (the prd site is identical apart from names):

       add_header X-Content-Type-Options "nosniff" always;
       add_header X-Frame-Options "DENY" always;
       add_header Referrer-Policy "strict-origin-when-cross-origin" always;
       add_header Strict-Transport-Security "max-age=31536000; ..." always;

2. The Go API, on every response. `src/api-srv/cmd/server/main.go:118`
   wraps the whole mux in `security.HeadersMiddleware`, which sets CSP,
   HSTS (`...; includeSubDomains; preload`, `security_headers.go:27-30`),
   `X-Frame-Options: DENY` (`:32-35`), `nosniff`, Referrer-Policy,
   Permissions-Policy and cache headers.

nginx's rule: a location inherits the server-level `add_header` lines only
if it sets none of its own. `location /api/` (`:40-51`) sets none, so it
inherits all four, and nginx adds them on top of the headers the API
already sent through `proxy_pass`. Every API response therefore carries
two `X-Frame-Options`, two `X-Content-Type-Options`, two `Referrer-Policy`
and two `Strict-Transport-Security` headers, the last pair with different
values (the API's has `preload`, nginx's does not). `curl -sI
https://<site>/api/health | grep -ic '^x-frame-options'` should print 2
(read from the config, not run on a host).

What that does in a browser today: nothing visible, because the values
agree. The real cost is latent. The moment any API route relaxes a header
(the usual case is `X-Frame-Options: SAMEORIGIN` so the SPA can show a
PDF from `/api/...` in an iframe; B083 describes that pattern), nginx
still appends `DENY`, the browser sees two conflicting values and refuses
to frame the page. The developer sees a correct Go handler and a broken
page, and the cause is in a file on the host. A feature
breaks; no protection is lost. So in PROTO this is a correctness trap and
a quality gap, not a security hole; the item's P1 is high for PROTO,
where no route relaxes a header yet.

The file already knows about the inheritance rule: the comment at
`:56-58` explains why `/assets/` (`:59-67`) and `= /index.html`
(`:75-82`) repeat all four headers. It just did not apply the same logic
to `/api/`. `location /` (`:69-71`) sets no header and so relies on the
server-level block; it serves real files that sit at the web root
(anything Vite copies from `public/`, such as a favicon). `/` itself and
every SPA route fall through `try_files` to `/index.html`, an internal
redirect into `location = /index.html`, which has its own headers.

#### What GRC changed

GRC hit the latent bug. Its file routes send `X-Frame-Options: SAMEORIGIN`
so the SPA can embed uploaded PDFs (RG0026), and the server-level `DENY`
broke the inline viewer. RG0043 item 1.5 (headline 5 in
`RG0043 md:876-896`) records it and a first wrong fix: `proxy_hide_header
X-Frame-Options` under `/api/` would have stripped Go's `SAMEORIGIN` and
kept nginx's `DENY`.

The fix, commit `535e7cb`, in both GRC site files
(`infra/uat-unified001/etc/nginx_uat-grc.*`, prd in lockstep):

- `:36-39`: comment: headers are set per web location, never at server
  level, because a server-level `add_header` is inherited by `/api/`.
- `:42`: in `location /api/`: "No add_header here: the API's own
  security headers (security.HeadersMiddleware) stand alone."
- `:70-76`: `location /` gains the four `add_header` lines.
- `/assets/` (`:59-67`) and `= /index.html` (`:79-86`) keep theirs.

No automated test. The ticket's host check is `curl -sI .../api/health |
grep -ic '^x-frame-options'` returning 1, not 2, then opening a PDF in the
browser (`RG0043 md:642-653`). The ticket notes the file was not run
through `nginx -t` locally.

#### Proposed fix for PROTO

1. `infra/uat/etc/nginx_uat-proto.example.com:33-36`: delete the four
   server-level `add_header` lines.
2. Same file, `location / {` (`:69-71`): add the same four lines, with
   `always`, above `try_files`.
3. Replace the comment at `:56-58` with one placed before `location /api/`
   that states the rule: headers are per web location, none at server
   level, and `/api/` sets none because `HeadersMiddleware`
   (`src/api-srv/internal/security/security_headers.go`) already does.
4. Apply 1-3 identically to `infra/prd/etc/nginx_proto.example.com`; keep
   the two files equal except for names.
5. `doc/105-api-srv-tech-spec.md:212-219` lists the API's headers; add
   one line that nginx adds none on `/api/`.

Blind-copy traps: GRC's site has a different upstream port, a 50m body
cap, `X-Real-IP` and `Connection ""`; keep PROTO's 8088 and 20M and leave
the proxy lines to B048, which edits the same two files (land them
together or rebase one on the other). Drop the RG0026/RG0043 references
and the document-route wording from the comment. Hostnames stay
`example.com`.

Trade-off: nginx-generated errors under `/api/` (502 when the API is down,
413 over the body cap) lose the four headers, since the API never sees
those requests. HSTS still arrives on every page load from the web
locations, so nothing material is lost.

Depends on nothing. Related: B048 (same files), B083 (the first PROTO
route that would relax a header), B059 (the API's own header set).

Tests: none automated for `infra/`. On each host after `nginx -t &&
systemctl reload nginx`: `curl -sI https://<site>/api/health` shows one
`X-Frame-Options` and one HSTS; `curl -sI https://<site>/` and
`https://<site>/assets/<file>` still show all four.

#### Open question for Brandon

None.

### B023 — COMPLETED — fail2ban app jail bans real users on `/api/` 4xx
- **Completed:** 2026-10-02. The `[uat-proto]` and `[prd-proto]` jails in `infra/{uat,prd}/fail2ban/etc/jail.local` gain `ignoreregex = "(GET|POST|HEAD) /api/` with a short why, and uat `maxretry` goes 20 to 10 (Brandon: 10 on both). The shared `nginx-404.conf` filter is unchanged apart from its header, which now says app jails exclude `/api/`. `infra/README.md` drops the false prd-10/uat-20 sentence, says the `[DEFAULT]` values are the host's and why the app jail ignores `/api/`, and setup step 8 adds the `fail2ban-regex` check before the reload; B026's runbook is not written yet, so the check lives only in the README. Verified in ubuntu:24.04 on 12 sample access-log lines: old filter matched 10 (all six `/api/` 400/401/403/404 lines plus `/wp-login.php` 404, `/.env` 404, a stale `/assets/` 404 and a 405); with the jail ignoreregex 4 matched and the 6 `/api/` lines were ignored, and `fail2ban-client -d` shows `addignoreregex` and `maxretry 10` on both app jails. Tests: all green, go=279 unit=42 e2e=10 py=79

- **Priority:** P1  **Area:** infra, docs
- **GRC:** `infra/uat-unified001/fail2ban/etc/jail.local:24,43-51`; `infra/prd-unified002/fail2ban/etc/jail.local:33-41`; `infra/README.md:31-33`; `doc/330-uat-postsales-cutover.md:319-320`   **PROTO:** `infra/uat/fail2ban/etc/jail.local:45,51-55`; `infra/prd/fail2ban/etc/jail.local:45,51-55`; `infra/*/fail2ban/etc/nginx-404.conf:6`; `infra/README.md:74-84`
- **Ticket:** RG0013 / `a57d6ad`
- **What:** Filter counts 400-406; the API 401/403/404s by design, so a busy
user hits `maxretry` in 60 s and ufw bans them. GRC: `ignoreregex = "(GET|POST|HEAD) /api/`.
- **Port notes:** Both jails; README: auth brute force is the API's job.
Brandon's call: uat `maxretry` 10 or 20? Ops P1, docs P2.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

fail2ban watches each app's nginx access log and, through ufw, blocks a
source IP that produces too many "failures". PROTO's jail treats every
4xx on every path as a failure, including the API's normal answers.

1. The filter, `infra/uat/fail2ban/etc/nginx-404.conf:6` (prd identical):

       failregex = ^<HOST> -.*"(GET|POST|HEAD).*HTTP.*" 40[0123456]

   Any GET, POST or HEAD answered 400-406 counts. The header (`:2-4`)
   says why: "400/404/405/406 is scanning, 401/403 is auth probing."
2. The thresholds, `infra/uat/fail2ban/etc/jail.local:44-46` (shared by
   every jail on the host, since they sit in `[DEFAULT]`):

       findtime = 60
       maxretry = 20          # prd copy: 10
       banaction = ufw

3. The app jail, `:51-55`: `[uat-proto]`, `port = http,https`,
   `filter = nginx-404`, `logpath = /srv/uat-proto/log/nginx_access.log`.
   No `ignoreregex`, so `/api/` lines count.

The API answers 4xx to legitimate users all the time, by design:

- `GET /api/refresh` runs on every page load of a signed-out visitor
  (`hooks/use-auth-guard.ts:16-21` calls `authenticateWithAuthToken`,
  `stores/auth-store.ts:105`), and `CheckJWT` answers 401 with no cookie
  (`security/app_helper.go:282`). One count per page load.
- `POST /api/auth` with a mistyped password: 401
  (`handler/auth_handler.go:235-237`). A wrong TOTP code: 401
  (`user_mfa_handler.go:552-554`).
- A missing grant, a stale CSRF token, a blocked user: 403 (the list is in
  B006). A page that reads a list the role cannot read 403s on every visit.
- A stale or deleted id: 404. A form that fails validation: 400.

What happens: on prd, 10 of those from one IP inside 60 seconds and ufw
denies that IP on 80 and 443 for `bantime = 10m` (`:20`). The user sees
connection timeouts with no message, on every site the host serves, not
just this app, because ufw blocks the port, not the vhost. The count is
per source IP, so an office behind one NAT address pools every user's
4xx: a morning of people opening bookmarks (401 each) and retyping
passwords trips it quickly. This is an availability bug that locks out
real users; it adds no security the API lacks, because the API already
rate-limits (`CheckRateLimitAndUserBlock`, `app_helper.go:392`, 429) and
locks out failed logins per user and per IP
(`auth_handler.go:61-90`).

`infra/README.md:74-84` documents the jail and claims `maxretry` is
"deliberately stricter on prd (10) than uat (20)". GRC's survey of the
real hosts found both run 10 (`RG0013 md:55-56`); the README describes a
split no host has.

#### What GRC changed

GRC's API answers 404 for rows outside the caller's scope and 401 when a
session lapses, so its users walked into bans the same way. RG0013 (commit
`3a3fc94`, then `a57d6ad` which replaced the mirrors with the surveyed live
files) adds one line to GRC's own jail only:

- `infra/uat-unified001/fail2ban/etc/jail.local:43-51` (prd `:33-41`):

      [uat-grc]
      ...
      filter  = nginx-404
      # ... a busy user walks 4xx on /api/ legitimately: this jail counts
      # only the scanner traffic that hits the static site.
      ignoreregex = "(GET|POST|HEAD) /api/

  The leading `"` is part of the regex: it anchors on the quote that opens
  the request line in nginx's log format.
- `:24`: `maxretry = 10`, as the hosts run.
- The shared filter `nginx-404.conf` is unchanged: other apps' jails on
  the host use it, so the exclusion sits in the app's jail.
- GRC `infra/README.md:31-33` and `doc/330-uat-postsales-cutover.md:319-320`
  (Lessons: "ProtoStack's jail counts 401/403/404 on every path; an app
  that answers 404 for out-of-scope rows must exclude `/api/`").

No automated test, and no `fail2ban-regex` run in the GRC record.

Correction to GRC's comment: "counts only the scanner traffic that hits
the static site" overstates what is left. `location /` answers every
unknown path with `index.html` and 200 (`try_files $uri $uri/
/index.html`), so `/wp-login.php` or `/.env` never 404 there. After the
change the jail catches stale `/assets/` 404s, nginx 400s and 405s
(a POST to a static file), little else. In practice the app jail becomes
close to a no-op, and scanners on `/api/` are left to the API.

#### Proposed fix for PROTO

1. `infra/uat/fail2ban/etc/jail.local:51-55` and
   `infra/prd/fail2ban/etc/jail.local:51-55`: add to the app jail a
   comment (the API answers 401/403/404/400 to real users; auth brute
   force is the API's job) and `ignoreregex = "(GET|POST|HEAD) /api/`.
2. Leave `nginx-404.conf` alone; fix its header (`:2-4`) to say app jails
   exclude `/api/`.
3. `jail.local:45` (uat): set `maxretry` per the open question.
4. `infra/README.md:74-84`: drop the prd-10/uat-20 sentence, state that
   `[DEFAULT]` values are the host's (shared by every jail), and that the
   app jail ignores `/api/` because the API rate-limits and locks out.
5. Verification step in the README and B026's runbook: `fail2ban-regex
   /srv/<env>-proto/log/nginx_access.log /etc/fail2ban/filter.d/nginx-404.conf
   '"(GET|POST|HEAD) /api/'` shows API lines ignored, then
   `fail2ban-client reload && fail2ban-client status <env>-proto`.

Blind-copy traps: GRC's mirrors carry other apps' jails and log paths
from its hosts; copy only the two lines. The jail name stays
`[<env>-proto]` (rebranded by `bin/000-rebrand`). Keep `ignoreip`
redacted as `SERVERONLY`.

Depends on nothing. Related: B006, B012 (the API's 4xx), B007 (the API's
per-IP counting), B026 (runbook).

Tests: none automated for `infra/`; step 5's `fail2ban-regex` is the check.

#### Open question for Brandon

`maxretry` lives in `[DEFAULT]`, so it is a host value shared by every jail
on the box: set both PROTO copies to 10 (what the surveyed hosts run) or
keep uat at 20? Recommendation: 10 on both, because the template mirrors
the host and cannot choose a per-app value there.

Brandon: 10 on both

### B024 — COMPLETED — App root `root:shrmpl-hook 1775` so web publish works
- **Completed:** 2026-10-02. Took GRC's `root:shrmpl-hook 1775` app root (recommendation). `infra/README.md` layout gives owner/mode for the root, `log/` and `web-pub/` plus a sticky-bit note, and step 1 now chowns the root `root:shrmpl-hook` 1775, `web-pub` to the hook and only `log` to www-data; `doc/330-presales-to-postsales-migration.md` step 6 matches. Both `*-publish-web-client` scripts gain a header on the 1775 requirement, `tar --no-same-owner`, and `chmod -R a+rX` on the stage dir before the swap; the uat copy gains prd's refuse-unless-shrmpl-hook guard so the two differ only in env name. `uat-proto-build`'s unwritable-dir hint no longer says `chown -R hook $SRV_ROOT` (which would hand root's etc/crt to the hook): it prints the 1775 command for the root and a per-dir chown for stg/libexec. Not done: B024 step 6 (`bin/` to root:root), since B049/B050 are skipped. Verified in ubuntu:24.04 as shrmpl-hook: old layout fails at `mkdir web-pub.incoming.<pid>: Permission denied`; a 1775 root with www-data web-pub fails at the `mv` (Operation not permitted); the new layout publishes twice cleanly for uat and prd (files hook-owned 644 from a uid-1234 0600 tarball, www-data reads them), root is refused, and the hook cannot mv/rm/rmdir `etc/`, `crt/` or `log/`. Tests: all green, go=279 unit=42 e2e=10 py=79

- **Priority:** P1  **Area:** infra, docs
- **GRC:** `infra/uat-unified001/bin/uat-grc-publish-web-client:1-10,41-55`; `infra/README.md:52-65`; `doc/330-uat-postsales-cutover.md:315-316`   **PROTO:** `infra/uat/bin/uat-proto-publish-web-client:29-53` (+ prd); `infra/README.md:124-147`
- **Ticket:** RG0013 / `3a3fc94`
- **What:** Publish renames `web-pub/` as `shrmpl-hook` in a root-owned root,
which PROTO's setup forbids. GRC: root `root:shrmpl-hook 1775`, hook-owned
`web-pub/`, `--no-same-owner`, chmod before the swap (PROTO after, :50).
- **Port notes:** Small script change; README/doc in B049, B026.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

The web publish swaps the live bundle by renaming directories that sit
directly in the app root. The setup steps make the app root writable only
by root, and the publish runs as the webhook user. The two cannot both
hold.

1. Who runs it. `infra/uat/bin/uat-proto-build:5-8` re-execs itself as
   `shrmpl-hook` and calls the publish at `:128`; the prd copy refuses any
   other user (`infra/prd/bin/prd-proto-publish-web-client:18-23`).
2. What the setup makes. `infra/README.md:143-146` (step 1), as root:

       mkdir -p /srv/<env>-proto/{bin,etc,libexec,stg,backup,web-pub,log}
       chown -R shrmpl-hook:shrmpl-hook .../{bin,libexec,stg,backup}
       chown -R www-data:www-data .../{web-pub,log}

   `/srv/<env>-proto` itself stays `root:root 0755`. The layout
   (`:124-136`) gives no owner for the root.
3. What the publish does, `infra/uat/bin/uat-proto-publish-web-client`:

       33  STAGE_DIR="$WEB_PUB_DIR.incoming.$$"
       36  mkdir -p "$STAGE_DIR"
       46  mv "$WEB_PUB_DIR" "$OLD_DIR"
       47  mv "$STAGE_DIR" "$WEB_PUB_DIR"
       48  rm -rf "$OLD_DIR"
       50  chmod -R a+rX "$WEB_PUB_DIR"

   Every one of `:36`, `:46` and `:47` creates or renames an entry in
   `/srv/<env>-proto`, which needs write on that directory. As
   `shrmpl-hook` on a root-owned 0755 root, `:36` fails first: `mkdir:
   cannot create directory '/srv/uat-proto/web-pub.incoming.<pid>':
   Permission denied`. `set -e` stops the script. Were the root writable,
   `:48` would fail next: the old `web-pub/` is `www-data`-owned per step
   1, so the hook cannot delete the files inside it.

What the operator sees: every UAT build passes its tests, pushes, installs
the new API (`:127`, which only writes inside hook-owned `libexec/` and
`stg/`), then dies at the publish. Slack reports "proto UAT build FAILED
... Service state: active" (`uat-proto-build:44-52`). The new API now runs
behind the old web bundle. Prod's promote-and-deploy fails the same way
(`prd-proto-promote-uat-and-deploy:37-38`). Correctness bug: no web
deploy can succeed on a host built from the README. It has never shown
because `infra/` is a placeholder never run on a host (`infra/README.md:3-8`).

Two smaller defects in the same script:

- `:37`: `tar -xzf` without `--no-same-owner`. The uat copy has no user
  check, so a run as root restores whatever uid the build host stored.
- `:50`: `chmod -R a+rX` runs after the swap (`:46-48`). For that moment
  nginx (`www-data`) serves files with tar's modes; a non-world-readable
  file answers 403.

Adjacent finding, not in B024 and open in GRC too: `bin/` is hook-owned
(`infra/README.md:128,145`) yet root's cron runs `bin/*-renew-web-certs`
and `bin/prd-proto-backup-all` (`:91,93`). Any code a push runs as
`shrmpl-hook` (the UAT test gate runs the repo's tests) can rewrite a
script root will execute. No hook script writes into `bin/`, so it can be
`root:root 0755`. Belongs with B050.

#### What GRC changed

RG0013 (commit `3a3fc94`) lists this among its corrections to PROTO
(`RG0013 md:51-52`): the publish renames `web-pub/` in the app root as
`shrmpl-hook`, which PROTO's root-owned root forbids; GRC makes the root
`root:shrmpl-hook 1775` ("sticky: the hook renames only its own").

- The app root is `root:shrmpl-hook 1775`. Group write lets the hook
  create and rename entries; the sticky bit (the `1`) lets it rename or
  delete only entries it owns, so root's `etc/` and www-data's `log/` are
  safe. GRC `infra/README.md:52-65` gives owner and mode for every
  directory; `web-pub/` is `shrmpl-hook`, world-readable.
- `doc/330-uat-postsales-cutover.md:125-131` (step A4) creates it:
  `chown root:shrmpl-hook $APP && chmod 1775 $APP`, and chowns `web-pub`
  to the hook. Lessons `:315-316` record the finding.
- `infra/uat-unified001/bin/uat-grc-publish-web-client` (byte-identical
  to prd): `:7-10` explains the mode; `:14-17` re-execs as `shrmpl-hook`
  (`exec sudo -u "$TARGET_USER" -H "$0" "$@"`) instead of refusing;
  `:47` `tar --no-same-owner -xzf`; `:49` `chmod -R a+rX "$STAGE_DIR"`
  before the swap at `:51-54`.

No automated test. RG0013's own Checks (`:115-116`) predate any host
run; GRC has since run the `doc/330` cutover (its Lessons carry entries
dated during the run) and released to prod (commit `a504af6`).

#### Proposed fix for PROTO

1. `infra/README.md:143-146` (step 1): after `mkdir`, add `chown
   root:shrmpl-hook /srv/<env>-proto && chmod 1775 /srv/<env>-proto`;
   move `web-pub` from the `www-data` chown to the `shrmpl-hook` one;
   keep `log` as `www-data`.
2. `infra/README.md:124-136` (layout): owner and mode on the root line
   and on `web-pub/`, with one sentence on the sticky bit. The full
   owner/mode table is B049.
3. Both `*-publish-web-client` scripts: `tar --no-same-owner -xzf` at
   `:37`; move `chmod -R a+rX` from `:50` to before `mkdir -p
   "$WEB_PUB_DIR"` (`:45`), applied to `$STAGE_DIR`; add a header comment
   stating the 1775 requirement.
4. uat copy: add the user guard. Use GRC's `exec sudo -u shrmpl-hook -H
   "$0" "$@"` in both, or keep prd's refuse-with-hint; either works as
   long as the two match (B045 makes them byte-identical).
5. `doc/330-presales-to-postsales-migration.md:218-220` repeats step 1's
   chowns: same change there (the runbook rewrite is B026).
6. Optional, with B050: `bin/` to `root:root 0755` (adjacent finding).

Blind-copy traps: GRC's blob and tarball names, `data/`, its service user
and worker do not exist in PROTO; keep `proto-web` and
`proto-web-client.tar.gz` (rebranded from `etc/app.env`).

Depends on nothing. Related: B045, B049 (layout), B026, B050 (`bin/`).

Tests: none automated. On a host: `stat -c '%U:%G %a' /srv/<env>-proto`
shows `root:shrmpl-hook 1775`; `sudo -u shrmpl-hook <bin>/...publish-web-client`
ends "published"; `sudo -u shrmpl-hook mv /srv/<env>-proto/etc /tmp/x`
must fail.

#### Open question for Brandon

Take GRC's `root:shrmpl-hook 1775` app root, or keep the root `root:root`
and move the swap into a hook-owned `web/` parent (nginx `root` changes)?
Recommendation: 1775, because GRC already runs it on real hosts and the
sticky bit protects everything root and www-data own.

### B025 — COMPLETED — Left nav scrolls independently of the main screen
- **Completed:** 2026-10-02. `AppLayout` in `src/web-app/app.tsx` is now a Mantine `AppShell`: fixed 56 px header (burger under `sm`, `APP_NAME`, account menu with Profile and Logout), 240 px navbar whose links sit in `AppShell.Section grow component={ScrollArea}`, and `AppShell.Main` with the `UNREACHABLE` notice or `Outlet`; the shell is `disabled` unless signed in. Logout moved from the nav into the header menu (Brandon: Header); the five grant-gated nav links are unchanged. The phone drawer closes via `onClick` on each link rather than a `useEffect` on the pathname, so tapping the page already shown also closes it. `index.css` loses the 1024 px toast positioning rule. Deviations: no vitest render test (would need `@testing-library/react`, not a dependency); the Playwright scroll test runs at 1280x200 because five links fit in a 400 px window. Tests: new `tests/e2e/app-shell.spec.ts`; `smoke` and `mfa` specs sign out through a new `signOut` fixture (account menu). Tests: all green, go=279 unit=42 e2e=12 py=79

- **Priority:** P2  **Area:** web-app
- **GRC:** `components/grc/shell.tsx:4,26,65,69,87-98,100-102,149-165`   **PROTO:** `app.tsx:146-199` (`<nav>` :180-191, `<main>` :193-195); `index.css` notification rule
- **Ticket:** `75ed426` (shell base `e666896`)
- **What:** PROTO's nav scrolls away with the page; no fixed header or phone
layout. GRC: AppShell header + 240 px navbar (Burger under `sm`); the nav list
in `AppShell.Section grow component={ScrollArea}` scrolls on its own.
- **Port notes:** Rewrite `AppLayout`: header + account menu (GRC :87-98),
navbar NavLinks, main `Outlet`. Drop GRC extras. Fix `index.css`
notifications. Then B063.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

All paths are under `src/web-app/`. Every route, including `/login`,
renders inside `AppLayout` (`app.tsx:80-84`). `AppLayout` is a plain
two-column flex row inside a centred Mantine `Container size="xl"` (1320
px), with the nav shown only when signed in (`app.tsx:176-195`):

```tsx
<Flex>
  {authState === 'YES' && (
    <nav style={{ width: '200px', padding: '1rem', borderRight: ... }}>
  <main style={{ flex: 1, padding: '1rem' }}>
```

Nothing is fixed or scrolls on its own: the whole document scrolls, and
the nav scrolls with it. What a user sees:

1. **Short window or long list.** Scroll down a 50-row list and the nav,
   including Logout (`:187-189`), leaves the screen. To change page the
   user scrolls back to the top. There is no header, so the app name,
   the signed-in user and sign-out have no fixed home.
2. **Phone (360-414 px wide).** There is no breakpoint. The nav keeps its
   200 px; after container and `main` padding the page column gets
   roughly 100-150 px. List tables and edit forms overflow sideways or wrap one
   word per line. There is no way to hide the nav.
3. **A derived app adds nav entries.** The list grows the page; with
   15-20 entries on a laptop the lower links sit below the fold.

Separately, `index.css:45-52` positions toasts against a 1024 px column,
but the content is 1320 px wide, so on wide screens toasts float inside
the content. It also repeats the selector from `:35`.

UX problem, not a security or correctness bug.

#### What GRC changed

GRC's `src/web-app/components/grc/shell.tsx` uses Mantine `AppShell`
(base `e666896`; nav scroll in `75ed426`):

- A fixed 56 px header and a 240 px navbar that collapses below the `sm`
  breakpoint, opened by a `Burger` shown only on small screens
  (`shell.tsx:26,28,65,69`):
  ```tsx
  <AppShell header={{ height: HEADER_HEIGHT }} navbar={{ width: 240,
    breakpoint: "sm", collapsed: { mobile: !opened } }} padding="md">
  ```
- An account `Menu` in the header holds Account settings and Sign out
  (`:87-96`).
- The nav list sits in a section that scrolls by itself under the fixed
  header (`:100-102`):
  ```tsx
  <AppShell.Section grow component={ScrollArea} type="auto">
  ```
- Page content renders in `AppShell.Main` (`:151-165`).

GRC kept the same toast rule (`src/web-app/index.css:45-51`). No shell
test.

#### Proposed fix for PROTO

1. **Rewrite `AppLayout`** (`app.tsx:146-200`) on `AppShell`: header with
   `Burger` (`hiddenFrom="sm"`), the app name from `constants/app`
   (`APP_NAME`, already used by `pages/login.tsx:2`), and an account
   `Menu` (Profile, Logout) copied from GRC `:87-96`; navbar with PROTO's
   five `NavLink`s in `AppShell.Section grow component={ScrollArea}`;
   `AppShell.Main` holding the existing `UNREACHABLE` notice or `Outlet`.
   Drop the `Center`/`Container`/`Flex` wrappers.
2. **Signed-out pages** (`/login`, `/mfaSetup`) use the same layout: pass
   `disabled={authState !== 'YES'}` so they show no header or navbar.
3. **Close the mobile nav on navigation.** GRC only toggles
   (`shell.tsx:28,69`), so on a phone the drawer stays open over the page
   after a link is tapped. Call `close()` in a `useEffect` on
   `location.pathname`.
4. **Leave out** every GRC extra: search box, as-of date, role `Select`,
   badges, engagement alerts, `usePinnedPageHeader` (B063) and the 31
   icons (`@tabler/icons-react` is already a dependency if wanted).
5. **`index.css:45-52`**: delete the 1024 px positioning rule and let
   `<Notifications position="bottom-right" />` (`main.tsx:15`) place
   toasts; keep the width clamp at `:39-43`.
6. Then B063 (pinned page header, sticky filter row) builds on this.
7. Tests. vitest: render `AppLayout` signed in and assert the nav links
   and account menu; signed out, assert no navbar. Playwright: at
   375x667, open the burger, tap "Examples", assert the drawer closes;
   at 1280x500, scroll the example list, assert header and nav stay put.

#### Open question for Brandon

Should Logout move into a header account menu (GRC) or stay as the last
nav entry (PROTO today)? Recommendation: header menu, since it stays
visible however long the nav grows in a derived app.

Brandon: Header

### B026 — COMPLETED — Server deploy runbooks: doc/330, doc/331, doc/340, infra/sql
- **Completed:** 2026-10-02. Rewrote `doc/330` as the UAT cutover runbook (renamed `doc/330-uat-postsales-cutover.md`), added `doc/331-prd-postsales-install.md` (fresh prod install) and rewrote `doc/340` around one idempotent `infra/sql/proto-host-db.sql` (three roles, refuses a database it does not own, passwords as `\set` lines on psql's stdin, restore with `pg_restore --role=proto_ddl`, negative verify); paste blocks follow the rules (guards, no `<placeholder>`, no inline `#` in local blocks, `fill`/`fillpw` over `SERVERONLY` through the environment, ids from `/proc`). Per Brandon's answer both hosts load the same local dump (presales DB renamed aside on UAT); no ticket or `template-` tag (B091); the branch model is described, not prescribed (B057); B046, B058, B074, B027 are noted as pending. Two fixes the runbooks needed: `bin/000-rebrand` now renames slug-led file names (`infra/sql/proto-host-db.sql`), and `roles.sql` grants `db_reader`/`db_writer` only when missing, because the UAT test gate runs it as the CI role and Postgres 16 refuses even a repeat GRANT there. Verified in ubuntu:24.04 + Postgres 16 by running the docs' own blocks (SQL twice: identical state, no notices; app role DML yes, DDL no; CI role CREATEDB but no CONNECT; DDL role `verify`/`up`; rollback), `bash -n`/shellcheck on all 68 blocks, and a rebrand on a scratch copy. Tests: all green, go=279 unit=42 e2e=12 py=79

- **Priority:** P2  **Area:** docs, infra, tickets
- **GRC:** `doc/330-uat-postsales-cutover.md:1-330` (what changes :12-29; pre-outage :31-75; vars block :77-86; DB renamed aside :88-96; `svc-` :98-101; deploy key :103-123; 1775 + env fill + `all filled` gate :125-148; roles/data `pw` `sha256sum -c` :150-165; units, build, certs, login :167-201; webhook, pulsecheck, cron, fail2ban :205-283; cleanup, rollback :285-307; Lessons :309-330); `doc/331-prd-postsales-cutover.md:1-469` (`git archive` :8-14; ids :61-87; `fill` :95-104; `get-tar-commit-id` :106-111; SQL on stdin :176-193; cert `--dry-run` :216-253; promote hook :329-359; releases :371-409; rollback :411-430; Lessons :432-469); `doc/340-db-host-setup.md:1-60` (:19-34, :36-44, :46-57); `infra/sql/grc-host-db.sql:1-43`; `infra/README.md:1-15`; `tickets/02-working/RG0013-db-roles-and-postsales-infra/RG0013-db-roles-and-postsales-infra.md:47-80,121-122`; RG0043 md `:335-341,566-577`; RG0013 md `:50-58`   **PROTO:** `doc/320-presales-uat-deploy.md` (keep); `doc/330-presales-to-postsales-migration.md:1-325` (:15, :37, :131, :58-325); new `doc/331-prd-postsales-install.md`; `doc/340-db-host-setup.md:1-81` (:27-70); new `infra/sql/<slug>-host-db.sql`; `infra/README.md:31-97,138-187`
- **Ticket:** RG0013 / `a57d6ad`, `791b751`, `9de1d94`, `24a474e`, `33523c7`, `50c56fc`, `5bd0fd4`, `c18626f`, `16f3acd`; new PS ticket + `template-` tag (RG0013 :121-122 asks for this)
- **What:** PROTO 330 is unexercised and cites a missing SQL file; no prod
runbook exists. GRC ran 330 (UAT cutover) and 331 (prod install); 340 is one
idempotent psql file, restore as DDL role, negative verify.
- **Port notes:** Keep GRC numbering and PROTO 330 :18-56, 340 :9-17,:60,:72-78.
Rename-aside replaces `ALTER OWNER`. Genericize names; drop worker/blobs.
Paste rules: `: "${APP_ROOT:?}"`, no `<placeholder>`, no zsh inline `#`, no
argv passwords, `fill` only `SERVERONLY`, `psql < file`, uuid from `/proc`.
Deps B071, B035, B027; lessons B020-B024, B046, B057, B058, B074.
- **Decision:** \[ \] pull / \[X \] adapt / \[ \] skip

Brandon: These docs will have to be adapted for the generic proto template

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

Docs and runbook gap; the paste blocks also carry live hazards.

- **doc/330 has never run.** `doc/330-presales-to-postsales-migration.md:14-15`:
  "Not exercised yet. The first derived app to make this move should
  correct this document." GRC is that app; its corrections are not here.
- **It cites a deleted file.** `:37` and `:131` point at
  `sql/ddl-role-host-setup.sql`; PS0015 deleted it
  (`tickets/04-archive/PS0015-sql-and-cmd-shape.md:27`). No `infra/sql/`
  exists.
- **Ownership is moved in place.** Step 3b (`:129-149`) loops
  `ALTER TABLE ... OWNER TO proto_ddl` on the live database. Rollback
  (`:301-325`) reverses the loop or restores a dump: several manual steps.
- **Paste blocks are unsafe as root.** `:60` and `:321` run
  `curl -s http://127.0.0.1:<port>/health`; bash reads `<port` as an input
  redirect and `>/health` as an output one, so the line fails.
  `:122-123` and doc/340 `:30-31` hold `PASSWORD '<generate>'`. Variables
  come from an earlier block (`:80-82`) with no `${APP_ROOT:?}` guard, so
  the rollback's `cp -a ... "$APP_ROOT/etc/"` (`:308`) writes into
  `/etc/` if the shell lost them.
- **doc/340 is SQL to copy by hand** (`doc/340-db-host-setup.md:27-70`),
  hardcodes `uat_proto_db` (`:32,36-37`), and verifies only the owner row
  (`:68-69`), never that the API role is refused DDL.
- **No production runbook.** `infra/README.md:138-187` is "pointers, not a
  runbook"; nothing covers a fresh prod install or later releases.

#### What GRC changed

RG0013 (nine commits, `a57d6ad` to `16f3acd`) ran both moves on real hosts on
2026-10-01 and recorded Lessons. Section by section, what PROTO lacks:

GRC `doc/330-uat-postsales-cutover.md` (UAT cutover):
- `:12-29` before/after table: service user, certs, logs, monitors.
- `:88-96` A1: DB **renamed aside** (`RENAME TO ..._presales`); rollback
  is one statement. `:98-101` A2: service user `svc-<app>`.
- `:103-123` A3: per-repo deploy key and ssh alias for `shrmpl-hook`.
- `:125-148` A4: app root `root:shrmpl-hook 1775` (B024); secrets
  generated into the files; gate `grep ... SERVERONLY || echo "all filled"`.
- `:150-165` A5: one SQL file; `sha256sum -c` on the export;
  `pg_restore --no-owner` as the DDL role. `:205-307`: webhook,
  pulsecheck, cron, fail2ban, drop-aside a day later, rename-back rollback.

GRC `doc/331-prd-postsales-cutover.md` (fresh prod install, no PROTO
equivalent): `:8-14,106-111` prod has no checkout, so host files are a
`git archive` of a pushed commit, checked with `git get-tar-commit-id`;
`:61-87` ids from `/proc/sys/kernel/random/uuid`; `:95-104` `fill` writes
only over `SERVERONLY`; `:176-193` SQL on psql stdin (`postgres` cannot
read under `/root`); `:216-253` first cert with `--dry-run` and `nginx -t`;
`:329-409` promote hook on `main`, every later release; `:411-469`
rollback and Lessons.

GRC `doc/340-db-host-setup.md:1-60` + `infra/sql/grc-host-db.sql:1-43`:
idempotent `\gexec` roles and database; restore as the DDL role, so no
ownership loop; negative check (`340:56`, `create table nope` must fail).

GRC's own blocks still break two of its rules: `doc/330:153-156` and
`doc/331:180-183` pass passwords as `psql -v` arguments (visible in `ps`),
flagged by RG0043 item 22 and not yet fixed; `doc/330` keeps `<stamp>`
placeholders (`doc/331:453-456`) and `doc/340:56` has `"<api role URL>"`.

#### Proposed fix for PROTO

1. New PS ticket; tag the result `template-<date>` (RG0013 `:121-122`
   asks for the carry-back plus a new tag).
2. **`infra/sql/proto-host-db.sql`** from `grc-host-db.sql`, names from
   `APP_SLUG` (`bin/000-rebrand` rewrites them). Passwords via `\set`
   lines on stdin ahead of the file, never `-v` on argv.
3. **doc/340**: keep `:9-17` (role table), `:60` and `:72-78`; replace
   `:27-70` with "run the file", restore as DDL role, negative verify.
4. **doc/330** rewrite in GRC's shape: rename aside replaces the
   `ALTER OWNER` loop; keep PROTO `:18-56` (which-path and names tables).
   Fix `:37,131`; delete `:14-15` once exercised.
5. **New doc/331** prod install from GRC 331, generic.
6. Paste rules in both: open each host block with `: "${APP_ROOT:?}"`;
   no `<placeholder>`; no inline `#` in zsh blocks; no argv passwords;
   `fill` only over `SERVERONLY`; `psql < file`; ids from `/proc`.
7. Leave out worker, blobs, Entra, `bin/620` export (B071), invites (B035).
8. Fold Lessons in as steps: B020-B024, B046, B057, B058, B074; guard
   from B027.
9. Tests: none runnable; B053's path checker catches `:37`-style dead refs.

#### Open question for Brandon

On cutover, does the presales database's data move to the new
DDL-owned database (dump `--no-owner`, restore as DDL role, original
renamed aside), or does the host start from a fresh export as GRC did?
Recommendation: dump and restore the presales data, so UAT testers keep
their work and rollback is still one rename.

Brandon: Every project will be different, but in general when we go from presales to post sales, we setup UAT and Prod with whatever we've been working on our locals.

### B027 — COMPLETED — Presales deploy: refuse post-sales hosts; generate secrets on host

- **Completed:** 2026-10-02. `bin/610-deploy-uat` exits 1 with `[REFUSED]` before any write or `psql` when `$APP_ROOT/etc/*-$APP_NAME-deploy.env` (post-sales marker, e.g. `uat-proto-deploy.env`) exists; `etc/deploy.env` does not match. `etc/uat-api-srv.env.example` became `etc/uat-api-srv.env.template` (`__DB_APP_ROLE__`, `__DB_NAME__`, `__SERVER_NAME__`, `__APP_DISPLAY_NAME__` filled by `bin/600` from `etc/app.env` + `etc/uat-deploy.env`, which gains `DB_NAME`; `__DB_PASSWORD__`/`__JWT_SECRET__`/`__TOTP_KEY__` left for the host); on first run 610 writes `etc/api-srv.env` root:$SVC_USER 0640 (temp file 0600 until filled) with three 32-byte `/dev/urandom` hex values passed to awk via its environment, never overwrites it after, and on every run sets the role password from `DATABASE_URL` via `psql` stdin (`\set` + `ALTER ROLE ... \gexec`), checking the URL's user is `DB_APP_ROLE`. Deviations: B015 skipped, so `roles.sql` keeps its literal and 610 overrides it with ALTER ROLE rather than `:'svc_pw'`; secrets from `od /dev/urandom` instead of `openssl rand`; no committed shell test (verified in an ubuntu:24.04 + PG17 container: distinct 64-hex secrets, 0 secret hits in a 128-exec `strace -e execve` trace, generated password logs in and the literal does not, second run byte-identical, post-sales host refused with 0 `psql` execs while the old 610 dropped the DB). Open question took the recommendation: doc/320 has a one-time rotation step for hosts still on the literal (tested). Also doc/320, doc/330 (B5 cites the guard), DERIVING.md, `etc/loc-api-srv.env` comment, presales-deploy skill (verify step passes the DB password as `PGPASSWORD`, not argv); three pre-existing shellcheck notes in 610 fixed. Pre-existing, not fixed: `bin/600` vite step fails unless `VITE_API_PROXY_URL` is set (vite.config.ts throws in production mode). Tests: all green, go=279 unit=42 e2e=12 py=79

- **Priority:** P2  **Area:** bin, etc, docs
- **GRC:** `bin/610-deploy-uat:7-8,34-41,46-72`; `bin/600-bundle-uat:13-15,92`; `etc/uat-api-srv.env.template:1-41`; `doc/330-uat-postsales-cutover.md:282-283`   **PROTO:** `bin/610-deploy-uat:11,19-36,51`; `etc/uat-api-srv.env.example:1-38`; `doc/320-presales-uat-deploy.md:76-79,135-137`; `doc/330-presales-to-postsales-migration.md:294-297`
- **Ticket:** RG0012 / `2890e8a`, `55b0066`; RG0013 / `3a3fc94`
- **What:** (a) PROTO only documents "never on a post-sales host"; GRC's 610
exits if `etc/*-<app>-deploy.env` exists. (b) PROTO needs a hand-made env with
three secrets; GRC fills a template on the host once (`openssl rand -hex 32`,
0640).
- **Port notes:** Glob `*-${APP_SLUG}-deploy.env` (not `etc/deploy.env`); names
from `etc/uat-deploy.env`; password only (B015).
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

**Expanded (2026-10-02).**

#### What is wrong in PROTO today

`bin/600-bundle-uat` builds a bundle on the laptop; `bin/610-deploy-uat`
runs from it as root on the host (`deploy.sh`). Every run does
`DROP DATABASE IF EXISTS $DB_NAME` and recreates it from the laptop dump
(`bin/610-deploy-uat:112-115`).

**(a) Nothing stops it on a post-sales host.** A host on the `infra/`
pipeline keeps a persistent, DDL-owned database.
`doc/320-presales-uat-deploy.md:135-137` and
`doc/330-presales-to-postsales-migration.md:294-297` forbid 600/610 there,
in prose only. 610 checks root, bundle files and env file (`:49-51`),
then drops. One stale `UAT_HOST` on any laptop destroys UAT data.
Operational safety gap.

**(b) The host env file is hand-made and its DB password cannot work.**
`doc/320-presales-uat-deploy.md:76-79` tells the operator to copy
`etc/uat-api-srv.env.example` and replace each `SERVERONLY` (DB password,
`JWT_SECRET`, `TOTP_ENCRYPTION_KEY`, lines `:11,14,15`) by hand. 610 only
checks the file exists (`:51`). But 610 creates the role from
`000-roles.sql` (`:110`), which is `src/api-srv/sql/roles.sql:36-37`:
`CREATE USER proto_app WITH PASSWORD 'reallygoodPSQLpassword'`.
Nothing sets the role to the operator's password, so either the API
cannot connect or the operator copies the repo literal: UAT runs on a
password published in git (B015). The other two secrets are only as
strong as the operator makes them. The example also hardcodes
`proto-api`, `"Proto App"` and `uat_proto_db` (`:11,18-19`) instead of
deriving from `etc/app.env`. Security and DX gap.

#### What GRC changed

RG0013 (`3a3fc94`) added a guard before any work
(`bin/610-deploy-uat:34-40`):

```bash
if ls "$APP_ROOT"/etc/*-grc-deploy.env >/dev/null 2>&1; then
  echo "...; presales deploy.sh refuses."; exit 1
```

The post-sales deploy env is the marker;
`doc/330-uat-postsales-cutover.md:282-283` expects the refusal.

RG0012 (`2890e8a`) generates the env file on the host, once
(`bin/610-deploy-uat:48-58`): if `etc/api-srv.env` is missing it fills
`api-srv.env.template` (shipped by `bin/600-bundle-uat:92`) with
`openssl rand -hex 32` for DB password, JWT secret and TOTP key (`55b0066`),
sets `root:www-data 0640`, and creates or re-passwords the role with the
same value via a psql heredoc (`:60-71`). Never overwritten after;
`bin/600:13-14`: the bundle carries no secret. No tests.

#### Proposed fix for PROTO

1. **Guard** in `bin/610-deploy-uat`, right after the root check at
   `:49`: refuse when `"$APP_ROOT"/etc/*-"$APP_NAME"-deploy.env` exists
   (`APP_NAME` is `$APP_SLUG` via `etc/uat-deploy.env:12`). Trap: do not
   glob `etc/deploy.env` or `etc/*deploy.env`; `$APP_ROOT/etc/deploy.env`
   is PROTO's own presales override (`:25-30`, doc/320 step 5). The
   post-sales marker is `uat-proto-deploy.env` (`infra/uat/etc/`).
2. **Template**: rename `etc/uat-api-srv.env.example` to
   `etc/uat-api-srv.env.template` with `__DB_PASSWORD__`, `__JWT_SECRET__`,
   `__TOTP_KEY__`. Have `bin/600` fill the non-secret names (role, DB
   name, `SERVER_NAME`, `APP_NAME`) from `etc/app.env` and
   `etc/uat-deploy.env` at bundle time, and ship the result.
3. **First run in 610**: if `$ENV_FILE` is missing, generate the three
   secrets with `openssl rand -hex 32`, write the file `0640
   root:$SVC_USER`, and replace the `[ -f "$ENV_FILE" ]` exit at `:51`.
   Trap: GRC passes secrets as `sed -e` arguments (`:55-57`), which puts
   them in `ps` output; fill with a heredoc or `awk` reading `ENVIRON`.
4. **Role password**: with B015, `roles.sql` takes `:'svc_pw'`. Feed it on
   stdin (`printf '\\set svc_pw %s\n' ...; cat 000-roles.sql`), not
   `psql -v` on argv. Read the password back from `DATABASE_URL` on every
   run so `roles.sql` and the env file cannot disagree.
5. Leave out GRC's worker, blobs, `BLOB_ROOT`, Entra keys,
   `INVITE_TTL_HOURS` (B035) and the nginx site (B072).
6. Docs: doc/320 `:76-79` becomes "610 writes it on first run";
   doc/330 `:294-297` cites the guard.
7. Tests. Shell test: 610 against a temp `APP_ROOT` holding
   `etc/uat-proto-deploy.env` exits 1 before any `psql`; with no env file
   it writes three distinct 64-hex secrets, mode 0640.
8. Depends on B015. Pairs with B016/B061 (`SERVER_BIND_ADDRESS=127.0.0.1`
   belongs in the template).

#### Open question for Brandon

A presales host built from this template has the repo-literal role
password unless its operator changed it. Rotate by hand once (`ALTER
ROLE` plus the env file), or leave such hosts until cutover?
Recommendation: rotate now via a short step in doc/320; the literal is
public.
