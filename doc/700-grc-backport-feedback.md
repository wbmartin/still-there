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
| B004 refresh keeps session id, CSRF resync | adapt | expanded |
| B005 `/changeSecurityContext` CSRF | skip | |
| B006 403 signs the user out | adapt | expanded |
| B007 `/auth` username oracle | adapt | expanded |
| B008 fail closed on counter write | pull | queued |

Items from B009 on are not yet decided.

# P1: security and real bugs in PROTO

### B001 — MFA: enrolled login lands on setup; `/mfaSetup` returns the secret

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

### B003 — Server-side session revocation

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

### B004 — Refresh keeps its session id; web re-syncs CSRF once

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

#### Open question for Brandon

Step 6 goes beyond GRC. When the cookie now belongs to another tenant or user,
is today's outcome (the 403 stands and the tab is signed out) acceptable, or
should the tab reload into the identity the cookie now carries?
- ADAPT REQUEST, Describe the fix before we continue

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

### B006 — A 403 signs the user out

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
- Brandon: Need you to explain more about how you'd fix

### B007 — `/auth` username oracle: unknown login fast and uncounted

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
- Brandon: Need you to explain more about how you'd fix

### B008 — Fail closed when the failed-attempt counter cannot be written

- **Priority:** P1  **Area:** go-api
- **GRC:** `src/internal/handler/auth_handler.go:332-358`   **PROTO:** `src/api-srv/internal/handler/auth_handler.go:216-243` (:233); `user_mfa_handler.go:540-552`
- **Ticket:** RG0003 / `0d5146b`; RG0043 r1 / `720f5e1`
- **What:** PROTO logs and continues, silently disabling the throttle. GRC
answers 500 "Login is temporarily unavailable".
- **Port notes:** Trivial. `/mfaVerify` fails open in both (GRC
`user_mfa_handler.go:440-452`, RG0043 2.7); fix PROTO :540-552 too.
- **Decision:** \[ X\] pull / \[ \] adapt / \[ \] skip

### B009 — TOTP code replay inside the ±1-step window

- **Priority:** P1  **Area:** go-api, tests
- **GRC:** `src/internal/auth/totp.go:136-170`; `handler/user_mfa_handler.go:419-431,460-468,736-754`; `tests/conftest.py:23-40`; `tests/test_zz_mfa_totp.py:266-292`; `user_mfa_handler.go:419-468`   **PROTO:** `src/api-srv/internal/auth/totp.go:125-128`; `handler/user_mfa_handler.go:530-560`; `tests/test_zz_mfa_totp.py:70,94,106`; `user_mfa_handler.go:626`
- **Ticket:** RG0043 item 44 / `720f5e1`; pytest `dd8b4c3`
- **What:** PROTO accepts one code repeatedly for \~90 s. GRC stores the last
accepted step per login (`TOTP_LAST_STEP*<user>`, 2 min); a step at or below
it is a wrong code; KV error fails closed.
- **Port notes:** Keep `GenerateTOTPURI`. Port `fresh_code()` with it
(`TOTP.now()` would flake). Harness B019.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B010 — PS0021: reset-MFA grant 9020; completed login clears counters

- **Priority:** P1  **Area:** go-api, tests, docs, tickets
- **GRC:** `src/internal/model/grants.go:25-31,52,74,87`; `handler/user_mfa_handler.go:614-620,536-543`; `handler/auth_handler.go:135-177,483-490`; tests `user_mfa_handler_test.go:435-497`, `tests/test_zz_mfa_totp.py:294-346,348-396,398-415,506-518`; `tickets/03-done/RG0003-mfa-temp-token-and-loose-ends/RG0003-mfa-temp-token-and-loose-ends.md:67,81,101-109`   **PROTO:** `src/api-srv/internal/model/grants.go:20-30`; `handler/user_mfa_handler.go:684` (9003), `:483`, `:620-630`; `handler/auth_handler.go:96-110,245-251` (commented :246); `constants/grants.ts:21`; `tests/test_zz_mfa_totp.py:110-116`; `tickets/01-open/PS0021-mfa-loose-ends.md:20-47`; `doc/105-api-srv-tech-spec.md` \~\`:183-200\`
- **Ticket:** RG0003 / `0d5146b`, `37978ce`; closes PS0021 items 1, 3
- **What:** (a) Web gates reset on 9020, API checks 9003; GRC checks 9020,
clears seed + `mfa_setup`, 400s on a nil or `''` seed (PROTO :483: nil only).
(b) Counters never clear on success; GRC `Del`s both after a completed login
(after `/mfaVerify`).
- **Port notes:** `Del`, not `"0"` 1 s (`Incr` keeps TTL). Write "where a login
completes" into doc/105, PS0021. Open: per-IP clear behind NAT.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B011 — Body caps on every login-layer write; 413, not 400

- **Priority:** P1  **Area:** go-api, tests, docs
- **GRC:** `src/internal/handler/common.go:14-56`; `auth_handler.go:187,643,893`; `user_handler.go:392,549,769,904,1028`; `user_mfa_handler.go:29,260,632`; `app_user_group_handler.go:109,196`; `own_profile_handler.go:165`; `user_block_handler.go:86`; `user_invite_handler.go:131,318,345,503`; `tests/test_auth_flow.py:123-150`; `doc/105-api-srv-tech-spec.md:378-398`   **PROTO:** unbounded `user_handler.go:359,522,760,881,989`, `app_user_group_handler.go:110,194`, `user_mfa_handler.go:696`, `auth_handler.go:741`; capped `auth_handler.go:143,514`, `user_mfa_handler.go:201,395`, `profile_handler.go:109`; 400 at `auth_handler.go:145-152`; doc/105 after `:209`
- **Ticket:** RG0002 / `be58262`, `b33c9b9`; RG0043 item 45 / `720f5e1`
- **What:** Authenticated callers stream unbounded bodies; PROTO's existing caps
answer 400 with decoder text. GRC caps every decode (4 KiB unauth, 64 KiB
login layer) and answers 413 `BODY_TOO_LARGE`.
- **Port notes:** One `limitBody` spelling incl. example handlers; add a 64 KiB
pytest; doc/105 lists PROTO's caps.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B012 — Cross-tenant ids answered 409; unscoped writes and member count

- **Priority:** P1  **Area:** go-api, tests, docs
- **GRC:** `src/internal/repository/scope.go:1-72`; `user_repository.go:220-244,246-270,307,343-397,496,550,586,595-720`; `app_user_group_repository.go:199-204,220-237,260`; `tests/test_security_context_isolation.py:24-40,228-293`; `doc/105-api-srv-tech-spec.md:503-521`; `src/api-srv/README.md:209`; `tickets/03-done/RG0014-login-layer-repository-drift/RG0014-login-layer-repository-drift.md:44-52,66,97,109,180`   **PROTO:** `src/api-srv/internal/repository/user_repository.go:186-212,214-300,405-421,457-473,478-594` (`EXISTS` :409,461,696); `app_user_group_repository.go:192-210,224-240,265-280` (:196,269); `handler/app_user_group_handler.go:240-255`; `tests/test_security_context_isolation.py:99-112,297`; `doc/900-database-guidelines.md`; `src/api-srv/README.md:141`; `tickets/01-open/PS0010-api-entity-template.md:36-102`
- **Ticket:** RG0014 / `c85313a`, `b5d15fe`, `b3a1ea2`; PS0010 drift half
- **What:** PROTO's zero-row `EXISTS` ignores tenant: another tenant's id gets
409 (existence leak); group delete counts members across tenants. GRC:
`existsInContext` 404; writes carry the row's own tenant.
- **Port notes:** 4 signatures + fakes; `rows.Err()`. Pytest via `db` (B018).
Rule into doc/900, README:141.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B013 — Admin "Change Password" for another user always 400s

- **Priority:** P1  **Area:** web-app
- **GRC:** `stores/app-user-store.ts:346-356`; `stores/app-user-store.test.ts:39`; `tickets/03-done/RG0036-change-other-password-field/RG0036-change-other-password-field.md:28-38`   **PROTO:** `stores/app-user-store.ts:112-116`; `src/api-srv/internal/handler/user_handler.go:986`; `stores/app-user-store.ts:24-27`
- **Ticket:** RG0036 / `e6588b5`
- **What:** Store posts `userId`; handler reads `appUserId`, gets 0, 400s every
time. pytest posts `appUserId`, so only the web is wrong.
- **Port notes:** One line + fetch-stubbing vitest in `tests/`. README rule in
B076.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B014 — Go dependency bumps: three called vulnerabilities

- **Priority:** P1  **Area:** go-api, bin
- **GRC:** `src/go.mod:3-14`   **PROTO:** `src/api-srv/go.mod:3-30` (go :3, backoff :6, jwt :7, pgx :8, x/crypto :12, x/text :27)
- **Ticket:** RG0043 item 15; RG0014 / `9ce276b`
- **What:** govulncheck: jwt v5.2.1 GO-2025-3553 (`auth/jwt.go:69`, unauth); pgx
v5.5.5 GO-2026-5004 (`migrate.go:152`); x/text v0.14.0 GO-2026-5970
(`cmd/migrate/main.go:84`). GRC's versions clear all.
- **Port notes:** Bump jwt 5.3.1, pgx 5.10.0, x/crypto 0.55.0, x/text; tidy; go
1.26 vs host (B047); gate (B040).
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B015 — Literal service-account password shipped in `roles.sql`

- **Priority:** P1  **Area:** go-api, bin
- **GRC:** `src/migrate-srv/sql/bootstrap/010-database.sql:3-11`; `src/migrate-srv/cmd/migrate/bootstrap.go:71-93`; `main.go:149-157`; `bootstrap_test.go:6-20`   **PROTO:** `src/api-srv/sql/roles.sql:30-40` (:37); `bin/600-bundle-uat:97`; `bin/610-deploy-uat:110`
- **Ticket:** RG0043 item 43 / `720f5e1`
- **What:** PROTO's UAT path creates `proto_app` with the repo literal. GRC
ships none; bootstrap fills it from `DATABASE_URL`.
- **Port notes:** No Go bootstrap: `psql -v svc_pw=...` from `etc/*.env` in
`bin/510`/`610`, `:'svc_pw'` in `roles.sql`. Pairs B027.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B016 — UAT env example: API not bound to loopback

- **Priority:** P1  **Area:** etc
- **GRC:** `etc/uat-api-srv.env.template:6,23-26`   **PROTO:** `etc/uat-api-srv.env.example:17-23,26`; `infra/uat/etc/uat-proto-api.env:29`
- **Ticket:** RG0012 / `2890e8a`
- **What:** The presales API listens on all interfaces (8088 exposed unless
firewalled); infra env already binds 127.0.0.1. GRC binds loopback, 12 h
sessions.
- **Port notes:** Add the key with GRC's comment. Brandon's call: deployed
session 12 h or 72 h? File defects B088.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B017 — pytest/requests pins carry advisories

- **Priority:** P1  **Area:** tests
- **GRC:** `tests/requirements.txt:1-5`; `tests/README.md:5-9`   **PROTO:** `tests/requirements.txt:1-4`
- **Ticket:** RG0006 / `818f592`, `ad2c1dc`
- **What:** pytest 8.4.2 carries PYSEC-2026-1845; requests 2.32.5 predates the
CVE-2024-35195 bump. GRC pins 9.0.3/2.34.2 with reasons; pip-audit clean.
- **Port notes:** Re-run pip-audit at port time; psycopg2 only with B018. Tests
P1, docs P2.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B018 — PS0022: enforce tenant 2 at every test harness

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B019 — MFA pytest harness: per-run XFF, throwaway lockout user

- **Priority:** P1  **Area:** tests
- **GRC:** `tests/conftest.py:55-68,87-137,161-206`; `tests/test_zz_mfa_totp.py:1-26,38-61,68-81,91-110,116-192,237-264,417-495,498-503`; `tests/README.md:90-97`   **PROTO:** `tests/conftest.py:31-68` (`pytest.fail` :49-50); `tests/test_zz_mfa_totp.py:1-155` (:28-45, :49-108, default 7 at :122); `src/api-srv/internal/security/app_helper.go:308-311`
- **Ticket:** RG0003 / `37978ce`; RG0042 / `63ccf75`; RG0043 / `dd8b4c3`
- **What:** PROTO locks out loopback and the shared `mfa` user, so re-runs fail.
GRC: MFA client, fresh XFF per test, disposable lockout user, threshold via
`api_env`, nine cases.
- **Port notes:** Prereq for B001/B009/B010 tests. Relies on XFF trust
(appendix). Case split restructures PS0020 behaviour.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B020 — Nightly backup cannot write its dump

- **Priority:** P1  **Area:** infra
- **GRC:** `infra/prd-unified002/bin/prd-grc-backup-all:6-8,27-49`; `infra/prd-unified002/etc/prd-grc-backup.env:1-8`   **PROTO:** `infra/prd/bin/prd-proto-backup-all:13-40` (:22, :33-34); `infra/prd/etc/prd-proto-backup.env:1-8`; `infra/README.md:93,185-187`
- **Ticket:** RG0013 / `3a3fc94`, `a57d6ad`
- **What:** `su - postgres -c "pg_dump | gzip > FILE"` writes as postgres into a
root-made dir: permission denied. GRC: root writes `sudo -u postgres pg_dump -Fc`, dir 0700, DB first, excludes `backup`/`stg`, `REMOTE_DIR`.
- **Port notes:** `APP_ENV` from app root (B045); drop blob wording; encryption
in B050.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B021 — Cert renewal: non-interactive; key not owned by www-data

- **Priority:** P1  **Area:** infra
- **GRC:** `infra/uat-unified001/bin/uat-grc-renew-web-certs:1-29`; `infra/uat-unified001/etc/nginx_uat-grc.atticustechnology.com:1-6`   **PROTO:** `infra/uat/bin/uat-proto-renew-web-certs:15-25` (:16, :21, :24); `infra/prd/bin/prd-proto-renew-web-certs`
- **Ticket:** RG0013 / `3a3fc94`
- **What:** No `--agree-tos -n` (first issue hangs under cron); `chown -R www-data` hands over `privkey.pem`. GRC: flags, `root:www-data` 0750/0640,
`nginx -t` before reload.
- **Port notes:** Drop-in. Keep the note: renew once before linking the site.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B022 — nginx: security headers per web location, none at server level

- **Priority:** P1  **Area:** infra
- **GRC:** `infra/uat-unified001/etc/nginx_uat-grc.atticustechnology.com:36-42,69-76`   **PROTO:** `infra/uat/etc/nginx_uat-proto.example.com:33-36,69-71` (+ prd); `src/api-srv/internal/security/security_headers.go:32-34`
- **Ticket:** RG0043 1.5 / `535e7cb`
- **What:** Server-level headers inherit into `/api/`: duplicates, and a relaxed
header gets overridden by `DENY`. GRC moves them into `location /`.
- **Port notes:** Add to `location /` when removing :33-36.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B023 — fail2ban app jail bans real users on `/api/` 4xx

- **Priority:** P1  **Area:** infra, docs
- **GRC:** `infra/uat-unified001/fail2ban/etc/jail.local:24,43-51`; `infra/prd-unified002/fail2ban/etc/jail.local:33-41`; `infra/README.md:31-33`; `doc/330-uat-postsales-cutover.md:319-320`   **PROTO:** `infra/uat/fail2ban/etc/jail.local:45,51-55`; `infra/prd/fail2ban/etc/jail.local:45,51-55`; `infra/*/fail2ban/etc/nginx-404.conf:6`; `infra/README.md:74-84`
- **Ticket:** RG0013 / `a57d6ad`
- **What:** Filter counts 400-406; the API 401/403/404s by design, so a busy
user hits `maxretry` in 60 s and ufw bans them. GRC: `ignoreregex = "(GET|POST|HEAD) /api/`.
- **Port notes:** Both jails; README: auth brute force is the API's job.
Brandon's call: uat `maxretry` 10 or 20? Ops P1, docs P2.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B024 — App root `root:shrmpl-hook 1775` so web publish works

- **Priority:** P1  **Area:** infra, docs
- **GRC:** `infra/uat-unified001/bin/uat-grc-publish-web-client:1-10,41-55`; `infra/README.md:52-65`; `doc/330-uat-postsales-cutover.md:315-316`   **PROTO:** `infra/uat/bin/uat-proto-publish-web-client:29-53` (+ prd); `infra/README.md:124-147`
- **Ticket:** RG0013 / `3a3fc94`
- **What:** Publish renames `web-pub/` as `shrmpl-hook` in a root-owned root,
which PROTO's setup forbids. GRC: root `root:shrmpl-hook 1775`, hook-owned
`web-pub/`, `--no-same-owner`, chmod before the swap (PROTO after, :50).
- **Port notes:** Small script change; README/doc in B049, B026.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

# P2: template quality, DX, docs

### B025 — Left nav scrolls independently of the main screen

- **Priority:** P2  **Area:** web-app
- **GRC:** `components/grc/shell.tsx:4,26,65,69,87-98,100-102,149-165`   **PROTO:** `app.tsx:146-199` (`<nav>` :180-191, `<main>` :193-195); `index.css` notification rule
- **Ticket:** `75ed426` (shell base `e666896`)
- **What:** PROTO's nav scrolls away with the page; no fixed header or phone
layout. GRC: AppShell header + 240 px navbar (Burger under `sm`); the nav list
in `AppShell.Section grow component={ScrollArea}` scrolls on its own.
- **Port notes:** Rewrite `AppLayout`: header + account menu (GRC :87-98),
navbar NavLinks, main `Outlet`. Drop GRC extras. Fix `index.css`
notifications. Then B063.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B026 — Server deploy runbooks: doc/330, doc/331, doc/340, infra/sql

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B027 — Presales deploy: refuse post-sales hosts; generate secrets on host

- **Priority:** P2  **Area:** bin, etc, docs
- **GRC:** `bin/610-deploy-uat:7-8,34-41,46-72`; `bin/600-bundle-uat:13-15,92`; `etc/uat-api-srv.env.template:1-41`; `doc/330-uat-postsales-cutover.md:282-283`   **PROTO:** `bin/610-deploy-uat:11,19-36,51`; `etc/uat-api-srv.env.example:1-38`; `doc/320-presales-uat-deploy.md:76-79,135-137`; `doc/330-presales-to-postsales-migration.md:294-297`
- **Ticket:** RG0012 / `2890e8a`, `55b0066`; RG0013 / `3a3fc94`
- **What:** (a) PROTO only documents "never on a post-sales host"; GRC's 610
exits if `etc/*-<app>-deploy.env` exists. (b) PROTO needs a hand-made env with
three secrets; GRC fills a template on the host once (`openssl rand -hex 32`,
0640).
- **Port notes:** Glob `*-${APP_SLUG}-deploy.env` (not `etc/deploy.env`); names
from `etc/uat-deploy.env`; password only (B015).
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B028 — PS0010: one sentinel set + `mapRepoError`; delete `retry.go`

- **Priority:** P2  **Area:** go-api, tests, docs
- **GRC:** `src/internal/repository/errors.go:10-75`; `src/internal/handler/common.go:94-160`; `user_repository.go:66-68`; `handler/map_repo_error_test.go:16-125`; `repository/errors_test.go:16-91`; `doc/105-api-srv-tech-spec.md:196-198,470-501,522-531`; RG0014 md `:175`   **PROTO:** `src/api-srv/internal/repository/errors.go:1-27`; `strings.Contains` at `handler/user_handler.go:459-470`, `app_user_group_handler.go:341-359`, `user_mfa_handler.go:91-97`; `user_handler.go:587-620,689-714`; `app_user_group_handler.go:237-265`; `repository/retry.go:1-68`; `go.mod:6`; `doc/105-api-srv-tech-spec.md:114-119,259-281`; `doc/810-create-new-entity-type-notes.md:57`
- **Ticket:** RG0014 / `4796119`, `fe914fc`, `9ce276b`; PS0010
- **What:** PROTO picks statuses by error text; `ErrConflict` never fires. GRC:
sentinels, one mapper, `ClientMessageError`. `retry.go` never retries anything
pgx returns (31 call lines); GRC deleted it.
- **Port notes:** Prereq for B007; apply to the example handler. Fix
doc/105:279, doc/810:57. `pgconn.SafeToRetry` skipped on purpose.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B029 — RG0004: one JSON writer; session cookie without `Max-Age`

- **Priority:** P2  **Area:** go-api, tests, docs
- **GRC:** `src/internal/handler/auth_handler.go:415,466,493,600,619,785,806,849,1033`; `user_handler.go:520`; `app_user_group_handler.go:166`; `common.go:58-92`; `user_mfa_handler.go:533`; `tests/test_auth_flow.py:255-303`; `doc/105-api-srv-tech-spec.md:363-377`   **PROTO:** `src/api-srv/internal/handler/common.go:14-25`; `auth_handler.go:28-41,300,371,474,493,643,664,698,867`; `user_handler.go:493`; `app_user_group_handler.go:164`; doc/105 after `:209`
- **Ticket:** RG0004 / `d761d42`, `9fe4a78`; RG0043 item 34 / `720f5e1`
- **What:** (a) `handler.writeJSON` skips the gate on 2xx; GRC has one writer.
(b) `Max-Age=3600` on refresh/switch against a 72 h JWT kills refreshed
sessions after an hour; GRC: JWT expiry is the only lifetime.
- **Port notes:** `setAuthCookie` + named constants into `common.go`. Pytest: no
Max-Age, logoff 0. Tests rated the cookie P1.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B030 — `/me` 404s after a context switch

- **Priority:** P2  **Area:** go-api, tests
- **GRC:** `src/internal/handler/own_profile_handler.go:142-226` (:189, :211); `repository/user_repository.go:44-56,521-557`; `own_profile_check_sequence_test.go:155-454`   **PROTO:** `src/api-srv/internal/handler/profile_handler.go:55-90,92-178` (:72-73, :146, :168); `repository/user_repository.go:614-709`
- **Ticket:** RG0016 / `62403fd`
- **What:** `/me` reads and writes in the JWT's active tenant, so a switched
user gets 404. GRC: row by subject, write in the row's tenant, 200-char
fields, preference shape check.
- **Port notes:** Keep PROTO's `ProfileHandler`; change lookup and scope.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B031 — `LOG_LEVEL`; per-request and health lines to debug

- **Priority:** P2  **Area:** go-api, etc, docs
- **GRC:** `src/internal/config/config.go:47-49,225-233`; `src/internal/logging/logging.go:38-63`; `src/api-srv/cmd/server/main.go:58-61`; `handler/health_handler.go:36,48`; `config/config_test.go:414`; `src/api-srv/README.md:176-178`   **PROTO:** `src/api-srv/internal/logging/logging.go:28-46` (:34); `config/config.go:73-213`; `cmd/server/main.go:48`; `handler/health_handler.go:33-50`; `etc/loc-api-srv.env:17-23`; `etc/e2e-api-srv.env`; `etc/uat-api-srv.env.example`; `infra/*/etc/*-api.env`
- **Ticket:** RG0043 item 46 / `720f5e1`
- **What:** PROTO logs at debug always, usernames per request, two Info lines
per health poll. GRC: optional `LOG_LEVEL` (debug|info|warn|error, default
info, else refused); health at Debug.
- **Port notes:** Keep PROTO's logger; add a level. Add the key to every env
file and doc/905 (GRC never did; its doc/105:435 is stale).
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B032 — Config docs: env grammar and required-key policy

- **Priority:** P2  **Area:** docs, go-api
- **GRC:** `doc/905-config-guidance.md:6-50`; `src/api-srv/README.md:164-172`; `doc/105-api-srv-tech-spec.md:420-428`   **PROTO:** `doc/905-config-guidance.md:1-4`; `src/api-srv/README.md:22-42`; `src/api-srv/internal/config/config.go:64-70,118-193`
- **Ticket:** RG0005 / `5453748`; RG0001; RG0009; RG0015
- **What:** PROTO's 905 is four lines. GRC: grammar (first `=`, one quote pair,
no `export`/`$VAR`/inline comments, last wins), missing keys rejected, only
`TEST_DATABASE_URL` read from env.
- **Port notes:** Grammar applies as-is (`unquote`). Brandon's call: make
optional keys required? Ties to B061, appendix.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B033 — Remove anonymous `POST /error-report`

- **Priority:** P2  **Area:** go-api, web-app, tests, docs
- **GRC:** deleted; `src/internal/handler/common.go:15-25`; `src/api-srv/README.md:15-21`   **PROTO:** `src/api-srv/internal/handler/error_report_handler.go:1-80`; `src/api-srv/cmd/server/main.go:105,198`; `tests/test_error_reporting.py`; `tests/README.md:53`; `src/api-srv/README.md:81`
- **Ticket:** RG0018 / `68c77d2`
- **What:** Anonymous route writes 2000 chars of client text to the log at
Error. GRC's client posted to the wrong origin; GRC deleted it.
- **Port notes:** Brandon's call: delete, or require a session? Keep
`test_error_reporting.py` unless the route goes.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B034 — User block feature (PROTO ships dead block primitives)

- **Priority:** P2  **Area:** go-api, web-app, tests
- **GRC:** `src/internal/handler/user_block_handler.go:1-165`; `repository/user_repository.go:271-341`; `src/api-srv/cmd/server/main.go:95-109,223-228`; `model/app_user_model.go:48-58`; `handler/user_block_handler_test.go:93-174`; `stores/app-user-store.ts:55-56,380-393`; `pages/app-user-edit-page.tsx:259-283`; `pages/app-user-list-page.tsx:173-178`; `constants/api.ts:33-35`; `types/user-type.ts:3-5`; `tests/test_user_block.py:1-96`   **PROTO:** `src/api-srv/internal/security/app_helper.go:431-456,460-483`; `model/grants.go:22`; `model/security_model.go:87-91`; new web files
- **Ticket:** RG0043 item 11 / `03ad0f3`
- **What:** PROTO ships unused block parts (`BLOCKED*`, 9008,
`CheckUserBlocked`). GRC: `account_status`, block/unblock routes, `/auth`
refuses non-active, KV flag reloaded, revocation, UI, pytest.
- **Port notes:** Needs B035's migration and B003. Brandon's call: port, or
delete the dead primitives (PS0011)?
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B038 — Wire renames: `pwHash` -&gt; `password`; `fooList` keys

- **Priority:** P2  **Area:** go-api, web-app, tests
- **GRC:** `src/internal/model/security_model.go:115`; `user_handler.go:765-769,900-904`; `repository/user_repository.go:595-720`; `types/user-type.ts:17,76,87`; `tests/test_user_management.py:22,47,100,134,170`; `tests/test_basic_user_authorization.py:64,72`; `tests/test_security_context_isolation.py:52,73,112,125-126,147,160-161,182,218-219`   **PROTO:** `src/api-srv/internal/model/security_model.go:112`; `user_handler.go:756-760,876-881`; `repository/user_repository.go:478-594`; `pages/app-user-edit-page.tsx:89,127,280,285`; `types/user-type.ts`; same test files (`test_security_context_isolation.py:123,144,183,196-197,218,231-232,253,289-290`)
- **Ticket:** RG0043 items 36, 38 / `720f5e1`
- **What:** A raw password travels as `pwHash`; `userIds`/`groupId` break
PROTO's list rule. GRC: `password`, `userIdList`, `appUserGroupId`, `any`.
- **Port notes:** Breaking: Go, web, pytest together. go-api P3, web/tests P2.
Typo B089.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B039 — Go handler tests that prove the check contract

- **Priority:** P2  **Area:** tests, go-api
- **GRC:** `src/internal/handler/auth_handler_test.go:49-546`; `user_route_check_sequence_test.go:115-275`; `own_profile_check_sequence_test.go:155-454`; `map_repo_error_test.go:16-125`; `session_revocation_test.go:14-37`; `user_mfa_handler_test.go:235-752`; `user_block_handler_test.go`; `src/internal/security/client_ip_test.go:9-82`; `src/internal/model/app_user_model_test.go:12-65`; `auth_handler.go:182-186,633-636`   **PROTO:** new `src/api-srv/internal/handler/auth_handler_test.go`; `user_mfa_handler_test.go:47-169`; `user_handler_test.go`; `profile_handler_test.go`; `internal/security/`; `internal/model/`
- **Ticket:** RG0004 `d761d42`; RG0003 `eb4340b`; RG0014; RG0042; RG0043
- **What:** Fakes that panic if reached prove each route runs JWT, rate, CSRF,
authz, gate before the repository, plus lockout, temp token, 9020, 409,
refresh id, logoff, no secrets serialized. PROTO has no auth handler test.
- **Port notes:** Port beside each fix; handlers take `UserRepositoryInterface`
(GRC :182-186, :633-636).
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B041 — Clean-tree gate: eslint as a check; `git status` stays empty

- **Priority:** P2  **Area:** bin, docs, skills, etc, web-app
- **GRC:** `bin/320-test-web-service:7-12,45-56`; `bin/lib/eslint-report.py:1-67`; `CLAUDE.md:89-92`; `README.md:80-82`; `src/api-srv/README.md:227-229`; `tests/README.md:50-56,123`; `.claude/skills/ticket-workflow/SKILL.md:109,113-116`; `src/web-app/.gitignore:14`; `.gitignore:17-20`; RG0006 md `:30-34,238`; RG0007 md `:122-124`; `bin/320:8-9,46-55`   **PROTO:** `bin/300-run-tests:3-4,91-104`; `CLAUDE.md:51-54`; `README.md:28-29`; `.claude/skills/ticket-workflow/SKILL.md:99-103`; `doc/902-project-commands.md:79-81`; `src/web-app/.gitignore`; `src/web-app/package.json:16`; `.gitignore:6-13`
- **Ticket:** RG0006 / `0cf57f7`, `719739b`, `3db3bd1`, `67b8700`, `8f1c530`; RG0007 / `d7750dc`
- **What:** PROTO never enforces lint; `test:coverage` dirties the tree. GRC
fails if `eslint --fix-dry-run` would rewrite and requires empty `git status --short` after a run; ignores root `package*.json`, load-test binary.
- **Port notes:** Add `coverage` to `src/web-app/.gitignore`. Fail on
`errorCount > 0` too? Brandon's call: max-len carve-out (:9-12)? Typo B088.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B042 — Load test: fake metrics, broken cycle, dev target, argv password

- **Priority:** P2  **Area:** tests, bin, etc, skills
- **GRC:** `src/load-test/main.go:79-86,163-166,179-186,219-223,244-248,285,322-358,666-695`; `src/load-test/client.go:24-25,62-81,122-131`; `bin/330-test-load:10-23,54-72`; `etc/loc-test-api-srv.env:52-58`; `.claude/skills/ticket-workflow/SKILL.md:117-119`; `src/load-test/main.go:163-185`; `etc/loc-test-api-srv.env:1-9,19-21,31-36`   **PROTO:** `src/load-test/main.go:182-183,335-460` (:365,395,413,417), `:60-103,239-246,185-190,589-592,722-766,814-857` (`SystemMonitor`), `:263,298`; `src/load-test/client.go:126,193`; `bin/330-test-load:6-29`; `etc/loc-api-srv.env:36`; `etc/loc-test.env:1-41`; `etc/e2e-api-srv.env:1-60`
- **Ticket:** RG0006 / `c37a717`; RG0041 / `31970ae`; RG0043 item 33 / `720f5e1`
- **What:** PROTO fakes CPU/memory, times a create as update, double-closes
`stopChan`, writes the dev DB as tenant 1, takes `-password` on argv. GRC:
measured only, test API, `TEST_PERSONA_PASSWORD`, tenant 2.
- **Port notes:** Keep the example cycle (+`UpdateExample`); 330 starts a
private instance (bin/300 :140-182) or needs 8082. After B018.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B044 — tests/README.md rewrite

- **Priority:** P2  **Area:** tests, docs
- **GRC:** `tests/README.md:1-9,90-97,105-114,116-123`   **PROTO:** `tests/README.md:1-61` (dev server :15-20,29-32; pyotp :61; `test_own_profile.py`/`test_zz_mfa_totp.py` absent)
- **Ticket:** RG0041 / `31970ae`; RG0006; RG0003
- **What:** PROTO's README runs pytest against the dev API on 8080 and is stale
on files and deps. GRC states test API + tenant 2 only, MFA ordering/XFF,
helpers, reports.
- **Port notes:** Write fresh for bin/300; GRC's is stale at :79,84,129,136.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B046 — install-api: run the incoming migrator before touching libexec

- **Priority:** P2  **Area:** infra
- **GRC:** `infra/uat-unified001/bin/uat-grc-install-api:1-17,47-70,72-80,114-117`   **PROTO:** `infra/uat/bin/uat-proto-install-api:27-58,93-96` (:32-33) (+ prd); `infra/README.md:17-29`
- **Ticket:** RG0013 / `3a3fc94`
- **What:** PROTO installs the migrator before `verify`, so a refusal strands an
older migrator beside a newer API. GRC runs the schema step from the pull dir,
swaps nothing until it succeeds, keeps `*.prev`, prints rollback.
- **Port notes:** Skip worker wait (:84-104). Brandon's call: `api-srv` in
post-sales too (RG0013 #4)?
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B047 — UAT build: lock waits, per-step failure report, full-sha Slack

- **Priority:** P2  **Area:** infra
- **GRC:** `infra/uat-unified001/bin/uat-grc-build:1-13,40-41,45,47-54,56-62,63-73,77-83,85-94,140-159`; `infra/uat-unified001/shrmpl-etc/hooks/build-grc_HOOKID.env:11-13`   **PROTO:** `infra/uat/bin/uat-proto-build:21-24,44-52,62-66,129-138`; `infra/uat/shrmpl-etc/hooks/build-proto_HOOKID.env:9`
- **Ticket:** RG0013 / `da5eebd`; RG0043 / `535e7cb`, `b167658`
- **What:** `flock -n` drops a push landing mid-build. GRC waits 30 min (hook
3600), reports "FAILED at <step>", Slacks full sha at start/deploy/test,
preflights `git psql pg_dump`, names a stale host Go.
- **Port notes:** Toolchain check only if needed (B014). Replace "Radley GRC"
and module path (:99-101).
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B048 — nginx: keepalive actually used; plain redirect; IPv6

- **Priority:** P2  **Area:** infra
- **GRC:** `infra/uat-unified001/etc/nginx_uat-grc.atticustechnology.com:13-25,43-55`   **PROTO:** `infra/uat/etc/nginx_uat-proto.example.com:9-27,40-51` (+ prd); `src/api-srv/internal/security/app_helper.go:308-311`
- **Ticket:** RG0013 / `3a3fc94`
- **What:** `keepalive 64` is unused without `Connection ""`. GRC adds it,
`X-Real-IP`, drops `Origin`, plain 301, `[::]:443`.
- **Port notes:** Keep PROTO's 8088 and 20M body.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B049 — infra/README: runs-as table; layout with owners and modes

- **Priority:** P2  **Area:** infra, docs
- **GRC:** `infra/README.md:11,35-65`; `infra/sql/grc-host-db.sql`   **PROTO:** `infra/README.md:86-136,124-155`; `doc/340-db-host-setup.md`
- **Ticket:** RG0013 / `a57d6ad`, `3a3fc94`, `cc30f35`
- **What:** GRC's script table adds Runs-as; its layout gives owner and mode per
dir. PROTO's has partial owners and a wrong step 1 (B024).
- **Port notes:** Keep PROTO's comparison and Deriving sections; strip
blobs/worker. Docs P3.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B051 — doc/105 and api-srv README: describe the real API

- **Priority:** P2  **Area:** docs
- **GRC:** `doc/105-api-srv-tech-spec.md:144-146,213-247` (MFA routes, flow), `:295-308,310-324,352-361` (fail-closed, rate keys, `ClientIP`, check tests), `:337-350` (exceptions table), `:584-588` (single process), `:432-440` (logging); `src/api-srv/README.md:121-132,134-148,150-160`   **PROTO:** `doc/105-api-srv-tech-spec.md:83-112,129-133,163-209,232-248,331-332,337,356`; `src/api-srv/README.md:83-105`; `app_helper.go:364,409,412`; `main.go:157-158,179`; `model/app_user_model.go:9,17`
- **Ticket:** RG0003 / `37978ce`; RG0011 / `06a1396`; RG0001 / `debb044`; RG0002 / `b33c9b9`; RG0004 / `d761d42`; RG0019 / `8fa36b6`; RG0005 / `5453748`
- **What:** PROTO omits the MFA routes, shows `/auth2` (:129-133), lists TOTP as
future (:356), claims stateless scaling (:337). GRC: real routes, exceptions
table, fail-closed contract, single process.
- **Port notes:** Write PROTO's truth until B002/B005/B033 land; check B059
first. README facts follow each code item.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B053 — `scripts/check-doc-paths.sh`: backticked paths must exist

- **Priority:** P2  **Area:** docs, bin
- **GRC:** `tickets/03-done/RG0011-doc-history-to-engineering-log/check-doc-paths.sh:1-4` (whole file); RG0010 md `:155-163`   **PROTO:** new `scripts/check-doc-paths.sh`; `CLAUDE.md:70-74`; `doc/902-project-commands.md`
- **Ticket:** RG0011
- **What:** Resolves every backticked path in docs and READMEs; exits 1 on a
miss. On PROTO: 35 of 207 missing in 16 docs (B090).
- **Port notes:** `cd` to repo root; allowlist placeholders
(`src/web-app/README.md:77-80`). Wire into bin/300 or the skill.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B054 — Mirror drift tests: hand-coded contract, tested both sides

- **Priority:** P2  **Area:** docs, web-app, tests
- **GRC:** `tickets/03-done/RG0020-retire-as-built-generator-chain/RG0020-retire-as-built-generator-chain.md:118-120,223-224`; `CLAUDE.md:74-78`; `types/grc-contract.test.ts:1-25,150-246`; `utils/permissions.test.ts`   **PROTO:** `CLAUDE.md:29-54`; `src/web-app/README.md:85`; `constants/grants.ts` vs `src/api-srv/internal/model/grants.go`; `constants/api.ts`; new `tests/contract.test.ts`
- **Ticket:** RG0020 (Brandon 2026-09-22)
- **What:** Ruling: mirrors are hand-coded and drift-tested; Go writes a JSON
fixture, vitest replays it, with a floor and parser canary. PROTO mirrors
grants and routes unchecked.
- **Port notes:** Start with `AllKnownGrants` -&gt; `testdata/grants.json` vs
`grants.ts`. Web P3, docs P2.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B055 — `npm update` is not a security fix

- **Priority:** P2  **Area:** docs, web-app
- **GRC:** `tickets/03-done/RG0007-web-hygiene-deps-errors-console/260922_0215-TKT-RG0007-web-hygiene-deps-errors-console.md:15-38`; `RG0007-web-hygiene-deps-errors-console.md:126-136,392-401`   **PROTO:** `src/web-app/README.md:112-118`
- **Ticket:** RG0007
- **What:** PROTO's README has every web ticket run `npm update`. In GRC that
bumped the linter (4 -&gt; 34 findings); seven of nine advisories were in unused
deps.
- **Port notes:** Order: depcheck, `npm audit --omit=dev`, `npm update <advised>`, toolchain as its own ticket. Same for `go get -u`.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B059 — API: CSP trimmed; `ip:` rate bucket; CSRF log without token

- **Priority:** P3  **Area:** go-api
- **GRC:** `src/internal/security/headers.go:11-31`; `src/internal/security/app_helper.go:320-329,368,424,543-550`   **PROTO:** `src/api-srv/internal/security/security_headers.go:12-35`; `security_headers_test.go:39-48`; `security/app_helper.go:360,412,521-527`
- **Ticket:** RG0043 item 32 / `720f5e1`; RG0002 / `be58262`
- **What:** PROTO's API CSP allows scripts, inline styles, images and exempts
`/assets/` for a JSON-only server; GRC: `default-src 'none'`, XFO DENY always.
Anonymous buckets get `ip:`; CSRF mismatch logs no token.
- **Port notes:** Confirm no bundle served; update the header test.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B060 — Go hygiene: one `model.Now()`; RG0015 small fixes

- **Priority:** P3  **Area:** go-api
- **GRC:** `src/internal/model/time.go:1-40`; `model/time_test.go`; `src/internal/auth/totp.go:60-73`; `src/internal/config/config.go:236-252`; `src/api-srv/cmd/server/main.go:246-254`; `src/internal/handler/app_user_group_handler.go:29,52-56,88,176,267`   **PROTO:** `src/api-srv/internal/handler/common.go:158-162`; `repository/user_repository.go:18-21,508,567`; `auth/totp.go:55-60`; `config/config.go:135-205`; `cmd/server/main.go:201-204`; `handler/app_user_group_handler.go:55-56`
- **Ticket:** RG0015 / `1fc6e86`, `6ee3abc`; RG0043 items 31, 35; PS0011
- **What:** Three timestamp spellings (UnixNano/1e9 drifts) become
`model.Now()`; TOTP regexp compiled once; threshold checked once;
`net.JoinHostPort`; `WriteError`; stale comments.
- **Port notes:** One mechanical hygiene ticket.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B061 — `SERVER_BIND_ADDRESS` required (fail-closed bind)

- **Priority:** P3  **Area:** go-api
- **GRC:** `src/internal/config/config.go:284-290`   **PROTO:** `src/api-srv/internal/config/config.go:106-109`; `etc/loc-api-srv.env:27-28`
- **Ticket:** RG0009 / `d36bfda`
- **What:** GRC refuses to start without a bind address; PROTO's empty default
is deliberate for LAN/phone dev.
- **Port notes:** Brandon's call; B016 covers deployed hosts.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B062 — Preferences: off the admin list; validated against options

- **Priority:** P3  **Area:** go-api, web-app
- **GRC:** `src/internal/model/app_user_model.go:5-37,60-76`; `src/internal/repository/user_repository.go:399-465`; `src/internal/model/app_user_model_test.go:12-44`; `utils/preference.ts:14-59`; `utils/preference.test.ts`; RG0016 md `:128,145`   **PROTO:** `src/api-srv/internal/model/app_user_model.go:19`; `repository/user_repository.go:302-373` (:308); `types/profile-type.ts:5-25` (:21-26)
- **Ticket:** RG0016 / `62403fd`, `47fa7c4`
- **What:** `READ_USER` on `/userList` returns others' preferences; GRC serves
them only on `/me`. PROTO's `resolvePreference` trusts any `rowsPerPage`
(100000 renders every row); GRC allow-lists.
- **Port notes:** Keep PROTO naming. Docs rated P2.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B063 — Web: pinned page header and sticky filter row

- **Priority:** P3  **Area:** web-app
- **GRC:** `index.css:53-76`; `hooks/use-pinned-page-header.ts:1-41`; `components/grc/sticky-filters.tsx:1-37`; `components/grc/shell.tsx:164`   **PROTO:** `index.css`; new `hooks/use-pinned-page-header.ts`, `components/sticky-filters.tsx`
- **Ticket:** `b52fa25`, `3d48e65`
- **What:** Title block and filters stay visible while long tables scroll.
- **Port notes:** After B025; rename `grc-page`/`--grc-*`.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B064 — Web: colour scheme only when signed in; list-page URL fixes

- **Priority:** P3  **Area:** web-app
- **GRC:** `App.tsx:283-309`; `pages/app-user-list-page.tsx:25-37,95`; `pages/app-user-group-list-page.tsx:46-58,80-93`   **PROTO:** `app.tsx:162-164`; `pages/app-user-list-page.tsx:116`; `pages/app-user-group-list-page.tsx:36-70`
- **Ticket:** RG0016; RG0008
- **What:** PROTO writes the default scheme before `/me` (reload flash, leaks to
next user); GRC applies on `YES`. Stale `?page=` clamps; sort keeps other
params; group list sorts.
- **Port notes:** Scaffolding pages: per page, no factoring.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B065 — Web: small UI helpers

- **Priority:** P3  **Area:** web-app
- **GRC:** `components/grc/help-tip.tsx:1-34`; `components/grc/sort-th.tsx:1-18`; `hooks/use-sort.ts:1-25`; `components/grc/save-bar.tsx:1-13`; `utils/safe-href.ts:1-26`; `hooks/use-draft.ts:17-24`   **PROTO:** new under `components/`, `hooks/`, `utils/`; `pages/app-user-list-page.tsx:135-148`
- **Ticket:** `e2a110f`; RG0043 1.6
- **What:** Info-icon labels, sortable `<Th>`, save bar, link allow-list for
untrusted content.
- **Port notes:** `use-sort` to `useSearchParams`; `safe-href` once untrusted
HTML renders; `use-draft` lacks try/finally.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B066 — Route-sweep Playwright spec (not GRC's `smoke/*.mjs`)

- **Priority:** P3  **Area:** tests, web-app
- **GRC:** `smoke/smoke.mjs:1-69`; `smoke/guard.mjs:1-60` (:24-31); `smoke/exercise.mjs`; `package.json` scripts   **PROTO:** `tests/e2e/`; `playwright.config.ts:47-80`; new `tests/e2e/route-sweep.spec.ts`; `tests/e2e/global-setup.ts`
- **Ticket:** `5b5deb6`, `e9f945a`; RG0041; RG0037
- **What:** GRC visits every route as every persona and flags error/blank pages
and console errors, via node scripts with no runner.
- **Port notes:** Keep Playwright; one spec over `ROUTES` x users; copy
`requireTestTenant` into `global-setup.ts`.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B067 — Hash-chained audit log for login-layer writes (pattern)

- **Priority:** P3  **Area:** go-api
- **GRC:** `src/internal/repository/writes.go:16-33`; `user_invite_repository.go:351-388`; `audit_repository.go`   **PROTO:** new
- **Ticket:** `853f92c`; RG0035
- **What:** Tamper-evident entries in the write's transaction; GRC has not
covered user/group/password writes (RG0043 2.3).
- **Port notes:** Feature; needs roles forbidding TRUNCATE (B050).
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B068 — Generic entity registry: ADR, not a backport

- **Priority:** P3  **Area:** go-api
- **GRC:** `src/internal/model/entity.go:1-40`; `model/grc_entities_gen.go`; `repository/entity_repository.go`; `handler/grc_handler.go`; `handler/grc_routes_gen.go`; `repository/writes.go:35-155`   **PROTO:** `src/api-srv/internal/model/example_model.go`, `repository/example_repository.go`, `handler/example_handler.go`; PS0010
- **Ticket:** `853f92c`; generator retired RG0020
- **What:** One generic repository/handler drives 85 entities from struct tags;
checks stay explicit in one handler. PROTO: \~2300 lines per entity.
- **Port notes:** Conflicts with "scaffolding" and moves check differences into
a matrix. ADR first.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B069 — Web dependency hygiene: depcheck config, zod pin

- **Priority:** P3  **Area:** web-app
- **GRC:** `.depcheckrc.yml`; `package.json` (`"zod": "^4.6.5"`)   **PROTO:** `package.json` (`"zod": "4.1.13"`)
- **Ticket:** RG0007 / `d7750dc`
- **What:** depcheck carries a recorded exception list. PROTO's other deps are
newer.
- **Port notes:** Brandon's call: caret or exact zod? `coverage` ignore is B041.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B070 — bin dev conveniences: targeted restart, test API, dirty flag, coverage

- **Priority:** P3  **Area:** bin
- **GRC:** `bin/215-run-api-srv:33-36,44,53`; `bin/216-run-test-api-srv:1-38`; `bin/600-bundle-uat:52-53,98`; `bin/620-export-tenant-data:71`; `bin/310-test-repo-model:121-147`   **PROTO:** `bin/215-run-api-srv:36,44,53`; `bin/300-run-tests:58-89,140-182`; `bin/600-bundle-uat:58,113`
- **Ticket:** RG0041 / `31970ae`; RG0012 / `2890e8a`; RG0006 / `0cf57f7`
- **What:** Restart kills only the dev instance (binary + env path); a
long-lived test API on 8082; staged changes mark a bundle dirty;
repository/model coverage HTML.
- **Port notes:** PROTO's per-run instance is the better gate. Keep
`tmp/api-srv/`. 221 is SKIP.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B071 — Host survey and scrubbed tenant export

- **Priority:** P3  **Area:** bin, docs
- **GRC:** `bin/605-survey-host:1-97` (:27, :31,42,52,59); `bin/620-export-tenant-data:1-97` (:77); `bin/lib/export-scrub.sql:1-35` (:22-24); `doc/902-project-commands.md:96-97`   **PROTO:** new `bin/605-survey-host`, `bin/620-export-tenant-data`; `doc/330-presales-to-postsales-migration.md`; `doc/902-project-commands.md`
- **Ticket:** RG0013 / `3a3fc94`, `a57d6ad`, `cc30f35`
- **What:** 605 records a host read-only (key names only, masked) into
`tmp/survey/`. 620 copies the DB, sets every login `invited`, dumps
`--no-owner`, MANIFEST + sha256.
- **Port notes:** Names from `APP_SLUG`; mask is best-effort. 620 needs B035.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B072 — Presales: install the nginx site from a template

- **Priority:** P3  **Area:** bin, etc
- **GRC:** `bin/610-deploy-uat:43-44,119-129`; `etc/uat-nginx-site.conf.template:1-34`; `bin/600-bundle-uat:93,102`   **PROTO:** `doc/320-presales-uat-deploy.md:79`; `etc/uat-deploy.env`
- **Ticket:** RG0012 / `2890e8a`, `d9eb2d2`
- **What:** First run writes and links the site, `nginx -t && reload`, prints
the certbot line.
- **Port notes:** GRC's template lacks security headers (add B022's); PROTO body
cap; `UAT_DOMAIN` in `etc/uat-deploy.env`.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B073 — Generated schema reference (doc/003) and dev migrate script

- **Priority:** P3  **Area:** docs, bin, go-api
- **GRC:** `bin/520-schema-reference:1-12`; `bin/205-migrate-db:1-18` (:3-6); `src/migrate-srv/cmd/migrate/schema_reference.go:11-120`; `doc/003-ai-postgres-schema-reference.sql:1-4`; `doc/900-database-guidelines.md:20`; `CLAUDE.md:35-38`; `doc/902-project-commands.md:101`; `src/internal/model/schema_contract_test.go:7-8,21,39-41`   **PROTO:** `doc/900-database-guidelines.md:27`; `src/api-srv/cmd/migrate/main.go`; new `bin/2xx`/`5xx`; `bin/510-reset-test-db`
- **Ticket:** `c30d256`, `fc3e098`
- **What:** A committed CREATE TABLE/INDEX snapshot regenerated after migrate;
doc/900 points SQL authors at it. PROTO makes readers replay every migration.
- **Port notes:** Add `schema-reference` to `cmd/migrate` (pgx, sorted);
regenerate in bin/510; fail bin/300 on diff. A `bin/205` around `up` is
trivial.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B075 — Dev docs: doc/050, doc/060, doc/902, doc/810

- **Priority:** P3  **Area:** docs
- **GRC:** `doc/050-dev-env-init.md:1-8,10-14,16-35,118-127`; `doc/060-dev-env-start.md:1-43`; `doc/902-project-commands.md:3-8,40-42,46-51,60-63,67`; `doc/810-create-new-entity-type-notes.md:23-24,85-101,130-131,147-151`   **PROTO:** `doc/050-dev-env-init.md:5,42-44`; `doc/060-dev-env-start.md:1-34` (:31); `doc/902-project-commands.md:1-140`; `doc/810-create-new-entity-type-notes.md:1,84-118`
- **Ticket:** RG0011 / `06a1396`; RG0021 / `1663a3e`; RG0010; RG0006 / `719739b`; RG0016 / `5593a9a`; RG0020 / `c9ac1aa`; RG0008 / `48a557a`
- **What:** 050: real dev topology replaces the nginx TODO, tools, Linux box.
060: sections, repo-relative commands, `/health`. 902: one row per script,
gate contract. 810: one column end-to-end as a layer table.
- **Port notes:** Drop PowerShell/worker/imports. 810 feeds PS0010. Gate text
after B040/B041/B018.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B076 — Web README: house patterns and small rules

- **Priority:** P3  **Area:** docs, web-app
- **GRC:** `README.md:106-108,120-122,136-139`; RG0008 md `:183`; `tickets/02-working/RG0034-capture-procedure-for-evidence/RG0034-capture-procedure-for-evidence.md:66`; RG0036 md `:28-38`   **PROTO:** `src/web-app/README.md:73-110`
- **Ticket:** RG0007; RG0008; RG0018; RG0034; RG0036
- **What:** Store test per store; empty load shows `<Alert />`; store throws,
page notifies; body keys equal json tags; Zustand `set` hides misspelled keys;
reload form on route-id change.
- **Port notes:** Docs only; example test from B013.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B077 — Security architecture doc for the template

- **Priority:** P3  **Area:** docs, infra
- **GRC:** `doc/requirements/20-architecture.md:22,24-26,33,40,58,60,186`; `doc/requirements/22-threat-model.md:85-131,143-150,156-160,214-217,227-238,246-253,306-309,320-324,330-344,349-359,366-368`; `doc/331-prd-postsales-cutover.md:283-290`   **PROTO:** new `doc/110-security-architecture.md` (or doc/105 Security `:151-225`); `doc/900-database-guidelines.md:1-43`; `doc/105-api-srv-tech-spec.md:7-18`; `infra/README.md:185-187`
- **Ticket:** GRC spec (A-002, A-004, A-038, A-047, A-049; C-1..C-14)
- **What:** App-layer isolation (no RLS) so a cross-tenant test per table;
cross-tenant 404 never 403; STRIDE for B1/B2/B7 (T-9 `mfa_exempt` humans, T-11
3-char CSRF); testable controls; pull-based backup target.
- **Port notes:** Cite `scoped_query.go`; mark controls PROTO meets; A-047 vs
PS0016 UUID.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B078 — Testing hygiene rules

- **Priority:** P3  **Area:** docs, tests
- **GRC:** `tickets/03-done/RG0037-batch-gate-stale-checks/RG0037-batch-gate-stale-checks.md:26-29,53-58,95-97`; `tickets/03-done/RG0021-entra-check-runtime/RG0021-entra-check-runtime.md:335,346,507`   **PROTO:** `tests/README.md:47-58`; `src/api-srv/README.md`; `bin/300-run-tests:24-27,74-78`
- **Ticket:** RG0037 / `4ce1484`; RG0021
- **What:** A "not in Y" assertion needs proof Y was read; macOS path for OS
probes; anchored detectors; gates fail, not skip; `<X>_LIVE=1` for live tests.
- **Port notes:** "Rebuild before pytest" is SKIP: bin/300 already rebuilds.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B079 — Process rules: batched testing; readability over DRY

- **Priority:** P3  **Area:** docs, skills
- **GRC:** `CLAUDE.md:85-88,129-134`; `.claude/skills/ticket-workflow/SKILL.md:110,121-129`; RG0037 md `:19-22,30-35`; `tickets/04-archive/engineering-log/260924_0740-TKT-NONE-round-11-on-main.md:57-62`; RG0010 md `:85`; `…/260922_0114-TKT-RG0010-agent-guidance-consolidation.md:30-39`   **PROTO:** `CLAUDE.md:51,56-74`; `.claude/skills/ticket-workflow/SKILL.md:99-103`; `tickets/04-archive/PS0005-agent-guidance-consolidation.md:30`
- **Ticket:** `77e1283`, `090f350`, `a605945` (Brandon 2026-09-22); RG0010 / `fbdb9f3`
- **What:** (a) Per ticket run only changed-layer units; reviewer runs the full
gate per batch (RG0037: a test never run). (b) Readability, maintainability,
speed; tolerate duplication; reviewable parts (not covered by the global
file).
- **Port notes:** Brandon's call on both; drop GRC's 440/106.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B080 — CLAUDE.md structure rules and small conventions

- **Priority:** P3  **Area:** docs
- **GRC:** `CLAUDE.md:21-25,33-34`; `src/api-srv/README.md:223-224`; RG0043 md `:806-829`; `tickets/03-done/RG0027-pdf-of-record-required/RG0027-pdf-of-record-required.md:134`; `tickets/03-done/RG0009-split-app-lib-by-role/RG0009-split-app-lib-by-role.md:144-150`; `tickets/02-working/RG0033-evidence-capture-set-design/RG0033-evidence-capture-set-design.md:183`   **PROTO:** `CLAUDE.md:11-12,18-24`; `doc/900-database-guidelines.md`; `doc/105-api-srv-tech-spec.md`; `src/api-srv/internal/handler/*.go` (33 `app :=` sites)
- **Ticket:** RG0006; RG0013; RG0043 1.1; RG0027; RG0009; RG0015 ruling 3; RG0033
- **What:** `cmd/` only shipped binaries; Layout links doc/902 and runbooks;
fixtures never in migrations; 409 when record state refuses; `app :=` shadows
package `app` (33 sites); pin deps that raise `go`.
- **Port notes:** One docs ticket; the `app` rename is code.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B081 — DERIVING.md: paste-ready "Template lineage" block

- **Priority:** P3  **Area:** docs
- **GRC:** `CLAUDE.md:14-17,156-173`   **PROTO:** `DERIVING.md:108-118,126-127`
- **Ticket:** `59d351c`
- **What:** GRC wrote the section DERIVING asks for: derive commit, last tag and
range, three `git` commands, PS id in `raised_by:`.
- **Port notes:** Add to DERIVING §8.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B082 — Later post-sales docs: repository reset; release checklist

- **Priority:** P3  **Area:** docs, skills
- **GRC:** `tickets/01-open/RG0044-fresh-repo-webhooks-deploy-keys.md:32-51,53-82,84-90` (:66-68); RG0043 md `:549-564,566-620,632-665,687-739,771-782`   **PROTO:** new `doc/350-repository-reset.md` or `infra/README.md:138-188`; `DERIVING.md:46-55`; new `.claude/skills/release/SKILL.md` or `infra/README.md` after `:122`
- **Ticket:** RG0044 (open, not run); RG0043 §10 (shipped 2026-10-02)
- **What:** What hangs off a GitHub repo and how to re-root it. A release record
(pre-flight, approvals, UAT, prod, rollback, hash-checked host edits) as a
checklist.
- **Port notes:** Wait for RG0044's run; strip hosts and hashes; after B057.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B083 — Serving user-uploaded files: header override, sniff by bytes

- **Priority:** P3  **Area:** docs
- **GRC:** `tickets/03-done/RG0026-file-routes-embeddable/RG0026-file-routes-embeddable.md:24`; `tickets/03-done/RG0023-policy-document-tab/RG0023-policy-document-tab.md:166`; `tickets/03-done/RG0024-evidence-artifact-file-route/RG0024-evidence-artifact-file-route.md:75,116`   **PROTO:** `doc/105-api-srv-tech-spec.md` \~\`:214`;` src/api-srv/internal/security/security\_headers.go:13-34`;` DERIVING.md\`
- **Ticket:** RG0023, RG0024, RG0026
- **What:** Override XFO/CSP for inline PDFs; cap declared and actual size
(413); type from magic bytes (415); `private, no-store`; every miss 404.
- **Port notes:** Docs until a blob route exists (PS0018 avatars); then P2.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B084 — Skills and tickets: small fixes

- **Priority:** P3  **Area:** skills, tickets
- **GRC:** `.claude/skills/ticket-workflow/SKILL.md:104-106,142`; `scripts/lint-tickets.py:154`; `tasks/todo.md:143-304`; `.claude/skills/handoff/SKILL.md:42,67-68,71`   **PROTO:** `.claude/skills/ticket-workflow/SKILL.md:100-101,116`; `scripts/lint-tickets.py:87,154`; `.claude/skills/handoff/SKILL.md:3,42-45,67-68,71-73`; `tasks/` (absent)
- **Ticket:** RG0006 / `719739b`; RG0043 / `5bd0fd4`; `59d351c`; `8d5fd52`, `9e234e3`, `8e1b197`, `a25f75b`
- **What:** Web checks use `tsc -b` and judge lint by non-max-len delta; "or a
PR Brandon merges" reconciles skill with CLAUDE.md; lint message hardcodes
"PS" (`ID_PREFIX` :87); handoff's `tasks/todo.md §0` has no file.
- **Port notes:** s/uat/main/; handoff creates `tasks/todo.md` if absent; skip
GRC's rounds.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

# Bugs found in PROTO along the way

### B085 — READMEs say `bin/510` builds the local database

- **Priority:** P2  **Area:** docs
- **GRC:** `bin/205-migrate-db:1-17`; `src/web-app/README.md:12-18` (GRC's answer)   **PROTO:** `README.md:46`; `src/api-srv/README.md:9-12`; `bin/510-reset-test-db:43-45`; `doc/050-dev-env-init.md:21-31`
- **Ticket:** none
- **What:** Both READMEs send newcomers to `bin/510`, which refuses non-scratch
names; the dev DB is built per doc/050.
- **Port notes:** Point at doc/050; dev migrate script is B073.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B086 — doc/060 names a script that does not exist

- **Priority:** P3  **Area:** docs
- **GRC:** `doc/060-dev-env-start.md:1-43`   **PROTO:** `doc/060-dev-env-start.md:7` (`./515-run-postgres.sh`), `:31` (absolute `~/Code/...` path)
- **Ticket:** none
- **What:** `bin/515` is `reseed-db`; no `515-run-postgres.sh` exists.
- **Port notes:** Fix now or in B075.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B087 — doc/902 health URL names another project's real host

- **Priority:** P2  **Area:** docs
- **GRC:** `doc/902-project-commands.md:40-42`   **PROTO:** `doc/902-project-commands.md:17`
- **Ticket:** none
- **What:** The `curl -k https://...:8443/health` line names a personal dev
host from another project (the same host as GRC's `TLS_NAME`).
- **Port notes:** Use `http://localhost:8080/health`.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B088 — Broken comments: UAT env example and `.gitignore`

- **Priority:** P3  **Area:** etc
- **GRC:** n/a   **PROTO:** `etc/uat-api-srv.env.example:7` (ends "It intentionally omits"), `:38` (dangling `# --- logging / KV ---`); `.gitignore:23` (`bin/200-run-tests`)
- **Ticket:** none
- **What:** Cut sentence; header with no keys; wrong script name (`bin/300`).
- **Port notes:** Fix with B016/B041.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B089 — `userGroupId` in a pytest matches neither API name

- **Priority:** P2  **Area:** tests
- **GRC:** `tests/test_basic_user_authorization.py:64,72`   **PROTO:** `tests/test_basic_user_authorization.py:64,72`; `src/api-srv/internal/handler/user_handler.go:756-757,877-878`
- **Ticket:** none
- **What:** Sends `userGroupId`; API reads `groupId`; passes only because authz
refuses first.
- **Port notes:** Fix regardless of B038.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

### B090 — Doc paths that do not exist

- **Priority:** P3  **Area:** docs
- **GRC:** `infra/sql/grc-host-db.sql:1-43` (fixes the doc/330 case)   **PROTO:** `doc/330-presales-to-postsales-migration.md:37,131` (`sql/ddl-role-host-setup.sql`); `doc/810-create-new-entity-type-notes.md:20,30,51,55,61`; `doc/340-db-host-setup.md:13,21`
- **Ticket:** none
- **What:** B053's linter: 35 of 207 tokens missing; doc/330's SQL file never
existed.
- **Port notes:** doc/330 via B026; rest by hand or B053.
- **Decision:** \[ \] pull / \[ \] adapt / \[ \] skip

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
- First `X-Forwarded-For` entry trusted for lockout/rate keys: GRC `security/client_ip.go:25-38`; PROTO `security/app_helper.go:310-324`. GRC will not fix behind nginx; the template should document it or count from the right.
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