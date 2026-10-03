# Tests

Pytest suite exercising the Go API's HTTP surface: authentication, CSRF,
authorization, and example/user CRUD. Runs against a live API instance
over plain HTTP — there is no TLS locally.

## Running

```bash
bin/300-run-tests py        # from the repo root; starts a private API
                             # instance from etc/e2e-api-srv.env on
                             # http://127.0.0.1:8082, runs pytest, stops it
```

There is no supported way to point the suite at a server you started
yourself: it creates and deletes users and examples, so `conftest.py`
refuses to start (exit 3) unless the target is the private test API.

`bin/300-run-tests` bootstraps the venv at `tests/.venv` itself (creating
it and installing `tests/requirements.txt` on first run, and rebuilding it
whenever `requirements.txt` changes), checks
`GET /health`, then runs pytest with an HTML report written to
`tmp/test-reports/` (gitignored).

## Configuration and guards

`bin/300-run-tests py` exports two variables:

- `TEST_API_ENV` — the config file the private API was started from (a copy
  of `etc/e2e-api-srv.env` in `tmp/test-logs/`).
- `TEST_BASE_URL` — that file's `TEST_BASE_URL` (`http://127.0.0.1:8082`).

Before any request, `conftest.py` exits the run with status 3 unless both
are set, `TEST_BASE_URL` equals the one in `TEST_API_ENV`, that file's
`DATABASE_URL` names a scratch database (`*_test_db` / `*_e2e_db`), and it
sets `TRUSTED_PROXY_COUNT=1`. `bin/300` writes that last key into its copy
only; `etc/e2e-api-srv.env` (Playwright) and `etc/loc-api-srv.env` stay at
0, so no developer API ever trusts a client-sent `X-Forwarded-For`. The
suite therefore runs only through `bin/300`; there is no manual mode.

## Forwarded addresses and lockouts

The server counts failed logins and unauthenticated requests per client
address, and locks an address or a name after
`FAILED_LOGIN_ATTEMPT_THRESHOLD` failures (read from `TEST_API_ENV` by
`failed_login_threshold()`). The rest of the suite signs in from loopback,
so a test that spends that allowance sends its own address instead:

```python
client_ip = new_forwarded_ip()           # conftest.py
requests.post(url, json=body, headers=forwarded_for(client_ip))
```

`new_forwarded_ip()` hands out a new address from 198.18.0.0/15 on every
call: a /24 picked by the run's start second (runs started under 512 s
apart never share one) and a host counter within it (no two tests in a run
share one). With one trusted hop the server keys on that header, so a
lockout case never locks loopback or the next run. The MFA lockout case
also enrols and deletes its own throwaway user rather than locking the
seeded `mfa` user, and tells the lockout 429 from the rate limiter's by the
`error` message.

A test that edits the row it signs in as, or spends a login's rate-limit
bucket, makes its own login with `create_throwaway_login(test_client,
prefix)` (a Basic User in tenant 2) and removes it with
`delete_user(test_client, app_user_id)`, so no seeded user is ever left
edited or throttled: the `/me` round trip and the rate-limit case do.
The rate-limit case reads `MAX_USER_REQ_PER_MIN` from `TEST_API_ENV` and
starts its burst with enough of the clock minute left (the limiter buckets
per login per minute) to expect exactly that many 200s, then a 429.

CSRF cases post `CHANGE_PASSWORD_PROBE` to `/changeOwnPassword` and check
the refusal with `assert_csrf_refused`: a core route, so a derived app
that deletes the `/example` scaffold keeps them.

Tenants: the fixture logins live in `TEST_TENANT = 2`; seed groups 4 and 5
in `OTHER_TENANT = 3`, the side the isolation tests aim at. Tenant 1 does
not exist in a test database (`sql/fixtures/test-tenant-lock.sql`). Every
login through `TestClient.authenticate` and the MFA test's `/mfaVerify`
calls `require_test_tenant`, which exits the run (status 3) if the user
landed in any tenant but 2. Request bodies carry no `securityContext` except
where a test proves the handler ignores it for the JWT's.

## Test Client

`conftest.py` defines `TestClient`: authenticates against `POST /auth`,
captures the JWT from the `auth` cookie and the CSRF token from the
response body, and attaches `X-CSRF-Token` to state-changing requests.
The fixture users (`admin`, `basic`, seeded by
`src/api-srv/sql/fixtures/test-seed.sql`) are MFA-exempt, so the client
authenticates in one step. Fixtures:

- `test_client` (class-scoped) — authenticated as `admin`
- `basic_user_client` (function-scoped) — authenticated as `basic`;
  cleans up its tracked examples after each test
- `example_tracker` — tracks examples created by an individual test

## Test Files

- `test_a_example_crud.py` — full lifecycle of example CRUD
- `test_auth_flow.py` — authentication and CSRF token handling;
  userName bounds; JWT `iss` is `APP_NAME`
- `test_basic_user_authorization.py` — basic users denied admin-only ops
- `test_bulk_operations.py` — bulk CRUD via `/exampleList`
- `test_error_reporting.py` — `POST /error-report` takes a signed-in
  caller's report; no session → 401, no CSRF → 403
- `test_own_profile.py` — GET/PUT `/me` on a throwaway login; the row
  stays home after `/changeSecurityContext`
- `test_security_context_isolation.py` — tenant 2 cannot use or probe
  tenant 3's groups; client-sent tenants are ignored
- `test_session_revocation.py` — sign-out, an admin password change and a
  group grant edit end live sessions (old cookie gets 401)
- `test_user_block.py` — block ends a login's sessions and refuses its
  sign-in; unblock restores sign-in, not the old session; 409/400/404/403
- `test_user_invite.py` — invite links: an invited login cannot sign in;
  check, redeem (once) and sign in; reissue/revoke; one 404 for any dead
  link; 409 on an admin-set password; 403 without CREATE_USER or CSRF
- `test_user_management.py` — user management operations (admin only);
  no `totpSeed` in `/userList`; duplicate user and group names, and a
  group still in use, are 409; a group delete returns its id
- `test_z_error_scenarios.py` — error conditions and status codes
- `test_zz_mfa_totp.py` — TOTP enrolment, sign-in, code and temp-token
  replay, admin reset, and the MFA lockout

## Requirements

`requirements.txt` pins every package the venv holds, transitives included,
so a bump replaces the whole set:

- `pytest==9.0.3` — 8.4.2 carries PYSEC-2026-1845 (predictable
  `/tmp/pytest-of-{user}` directories; a local user on a shared host such
  as UAT can hijack them). Fixed in 9.0.3.
- `requests==2.34.2` — 2.32.5 carries PYSEC-2026-2275 (predictable temp
  file in `extract_zipped_paths`). Fixed in 2.33.0.
- `pytest-html==4.1.1` (HTML report, works with pytest 9), `pyotp==2.9.0`
  (TOTP codes for `test_zz_mfa_totp.py`).
- The rest are their dependencies, from `pip freeze` of a fresh venv;
  three apply only below Python 3.11.

Audit by hand on every bump (not a `bin/300` gate: a test-only advisory
should not block an unrelated push):

```bash
python3 -m venv tmp/audit-venv && tmp/audit-venv/bin/pip install -q pip-audit
tmp/audit-venv/bin/pip-audit -r tests/requirements.txt
```

2026-10-02, pip-audit 2.10.1: "No known vulnerabilities found".
