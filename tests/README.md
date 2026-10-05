# Tests

Pytest suite for the Go API's HTTP surface: authentication, CSRF, MFA,
authorization, tenant isolation, sessions, and user, group and example
CRUD. It runs against a **private test API** that `bin/300-run-tests`
starts on `http://127.0.0.1:8082` over the scratch database
`loc_<APP_SLUG>_e2e_db`, as tenant 2 only. It never touches the dev
database or tenant 1.

## Running

```bash
bin/300-run-tests py     # this suite only
bin/300-run-tests all    # every gate; pytest is the last one
```

`py` rebuilds the e2e database (`bin/510-reset-test-db`), builds the API to
`tmp/test-logs/api-srv-e2e`, starts it, waits for `GET /health`, runs
`pytest tests --maxfail=1`, and stops the API whatever happens. It refuses
to start if something already listens on :8082.

There is no manual mode. The suite creates and deletes users, groups and
examples and locks accounts, so `conftest.py` exits the run (status 3)
unless it was started by `bin/300`. Never aim it at an API from
`bin/215-run-api-srv`: that serves the developer's data.

The venv lives at `tests/.venv` (gitignored). `bin/300` creates it, and
rebuilds it whenever `tests/requirements.txt` changes or its python no
longer runs.

## Configuration and guards

Settings come from `etc/loc-test.env` (`TEST_ENV` picks another file on a
build host): the scratch database names and `E2E_API_ENV`
(`etc/e2e-api-srv.env`). `bin/300` runs the API from a **copy**,
`tmp/test-logs/py-api-srv.env`, made from that file with two changes:

- `MAX_USER_REQ_PER_MIN` and `MAX_UNAUTH_REQ_PER_MIN` come from
  `etc/loc-api-srv.env`. The e2e file lifts them so Playwright's page loads
  cannot fail; the rate-limit test needs a limit a burst reaches in seconds.
- `TRUSTED_PROXY_COUNT=1`. The private test API trusts exactly one proxy
  hop, so pytest plays the proxy and a test sends its own `X-Forwarded-For`
  (below). `etc/e2e-api-srv.env` and the dev file stay at 0, so no other
  API trusts a client-sent header.

It exports `TEST_API_ENV` (that copy) and `TEST_BASE_URL` (its
`TEST_BASE_URL`). Before any request `conftest.py` exits with status 3
unless both are set, the URL equals the file's, its `DATABASE_URL` names a
scratch database (`*_test_db` / `*_e2e_db`), and `TRUSTED_PROXY_COUNT` is 1.
`api_env()` reads the copy for tests that need a server setting
(`FAILED_LOGIN_ATTEMPT_THRESHOLD`, `MAX_USER_REQ_PER_MIN`, `APP_NAME`).

## Tenants

- `TEST_TENANT = 2`: every fixture login lives here; the suite acts only in
  it. Tenant 2 is the reserved test tenant.
- `OTHER_TENANT = 3`: seed groups 4 and 5; the side isolation tests aim at.
- Tenant 1 does not exist in a test database
  (`src/api-srv/sql/fixtures/test-tenant-lock.sql`).

`TestClient.authenticate` and the MFA test's `/mfaVerify` calls run
`require_test_tenant`, which exits the run (status 3) if a login lands in
any tenant but 2. A caller's tenant comes from the JWT, so a test picks its
tenant by which user it signs in as. Request bodies carry no
`securityContext` except where a test proves the handler ignores it.

Seeded users (`src/api-srv/sql/fixtures/test-seed.sql`): `admin` /
`admin123!`; `basic`, `mfa`, `switcher` / `basic123!`. `admin`, `basic` and
`switcher` are MFA-exempt; `mfa` is not. `switcher` is homed in tenant 2
with a group in tenant 3.

## Client and fixtures

`TestClient` (`conftest.py`) signs in with `POST /auth`, keeps the JWT from
the `auth` cookie and the CSRF token from the response body, and sends
`X-CSRF-Token` on state-changing requests.

- `test_client` (class-scoped): signed in as `admin`.
- `basic_user_client` (function-scoped): signed in as `basic`; removes the
  examples it tracked afterward.
- `example_tracker`: tracks examples created by one test.

Helpers:

- `api_env()`, `failed_login_threshold()`: the test API's config.
- `new_forwarded_ip()`, `forwarded_for(ip)`: per-test client addresses.
- `create_throwaway_login(test_client, prefix)` returns `(client, row)`: a
  new MFA-exempt Basic User in tenant 2, signed in. `delete_user(
  test_client, app_user_id)` removes it at its current `updateTs`. Use
  them when a test edits the row it signs in as or spends a login's
  rate-limit bucket, so no seeded user is left edited or throttled (the
  `/me` round trip and the rate-limit case do).
- `CHANGE_PASSWORD_PROBE`, `assert_csrf_refused(response, message)`: CSRF
  cases post the probe to `/changeOwnPassword` (a core route, so deleting
  the `/example` scaffold keeps them) and check the whole 403 body.

## Forwarded addresses and lockouts

The server counts failed logins and unauthenticated requests per client
address and locks an address or a name after
`FAILED_LOGIN_ATTEMPT_THRESHOLD` failures. The rest of the suite signs in
from loopback, so a test that spends that allowance sends its own address:

```python
client_ip = new_forwarded_ip()
requests.post(url, json=body, headers=forwarded_for(client_ip))
```

`new_forwarded_ip()` returns a fresh address from 198.18.0.0/15: a /24
picked by the run's start second (runs started under 512 s apart never
share one) and a host counter within it. A lockout case therefore never
locks loopback or the next run. The MFA lockout case also enrols and
deletes its own throwaway user rather than locking the seeded `mfa` user,
and tells the lockout 429 from the rate limiter's by the `error` message.

The rate-limit case runs as a throwaway login (the admin is exempt). It
reads `MAX_USER_REQ_PER_MIN`, waits for the next clock minute only when
under 20 s of it remain, then expects exactly that many 200s and one 429.
The limiter counts a sliding 60 s window (this minute's bucket plus the
last minute's, weighted); a new login has no last-minute bucket, and a
burst inside one minute is counted exactly. It is the one known flaky
test; rerun once if it alone fails.

## Writing tests

The rules for every suite in the repo (Go, pytest, vitest, Playwright).
`bin/300-run-tests` enforces the skip rule; the rest are on the author.

- A "not in Y" assertion proves it read Y. Assert first that Y is
  non-empty, or that something which must be there is:
  `test_user_list_never_returns_totp_seed` asserts `user_list` before
  `"totpSeed" not in response.text`, and Go's
  `TestRetrieveUserList_OmitsSecrets` finds the row in the body before
  looking for the seed. Otherwise a read that silently came back empty
  passes without having looked.
- A probe of OS state works on macOS and on Linux: development is macOS,
  hosts are Ubuntu. `/proc` is Linux only (`ps -o command= -p PID` reads
  a command line on both); `stat -c` is GNU, `stat -f` is BSD. Use the
  call both have, or branch on `uname`.
- A text detector is anchored (`^`, `$`, a word boundary), never a bare
  substring over output that carries data, which can contain any word.
  `bin/300-run-tests` matches `^FAIL` and `--- SKIP: ` at a line start,
  not "FAIL" anywhere.
- A missing prerequisite fails; a test never skips. `bin/300-run-tests`
  fails a step that reports a skipped test (Go `--- SKIP`, pytest
  `SKIPPED`, vitest or Playwright `skipped`), and names the cause when a
  Go suite skipped because `connectTestDB`
  (`internal/repository/testdb_test.go`) found no database. Seed data a
  test needs and does not find is a failed assertion
  (`s.Require().NotEmpty(...)`), not a skip.
- A test that needs a live outside service (a vendor API, a real
  identity tenant, the internet) skips unless `<SERVICE>_LIVE=1` and is
  named for it: Go `TestLive<Service>...`, pytest
  `test_live_<service>_...`. Those are the only skips the gate allows.
  Nothing in the gate or on a host sets a `_LIVE` variable, so a green
  run never depends on that service; run one by hand with
  `ENTRA_LIVE=1 bin/300-run-tests go` (or `py`). The template has none.

## Test files

pytest collects alphabetically, so the prefixes order the run on purpose:
`test_a_` first, `test_z_` (rate limit, error scenarios) and `test_zz_`
(MFA lockout) last.

- `test_a_example_crud.py`: full lifecycle of example CRUD
- `test_auth_flow.py`: authentication and CSRF token handling; userName
  bounds; JWT `iss` is `APP_NAME`
- `test_basic_user_authorization.py`: basic users denied admin-only ops
- `test_bulk_operations.py`: bulk CRUD via `/exampleList`
- `test_error_reporting.py`: `POST /error-report` takes a signed-in
  caller's report; no session is 401, no CSRF is 403
- `test_own_profile.py`: GET/PUT `/me` on a throwaway login; the row stays
  home after `/changeSecurityContext` (uses `switcher`)
- `test_security_context_isolation.py`: tenant 2 cannot use or probe
  tenant 3's groups; client-sent tenants are ignored
- `test_session_revocation.py`: sign-out, an admin password change and a
  group grant edit end live sessions (old cookie gets 401)
- `test_user_block.py`: block ends a login's sessions and refuses its
  sign-in; unblock restores sign-in, not the old session; 409/400/404/403
- `test_user_invite.py`: invite links: an invited login cannot sign in;
  check, redeem (once) and sign in; reissue/revoke; one 404 for any dead
  link; 409 on an admin-set password; 403 without CREATE_USER or CSRF
- `test_user_management.py`: user management (admin only); no `totpSeed`
  in `/userList`; duplicate user and group names, and a group still in
  use, are 409; a group delete returns its id
- `test_z_error_scenarios.py`: error conditions, status codes, CSRF
  refusals and the rate limit
- `test_zz_mfa_totp.py`: TOTP enrolment, sign-in, code and temp-token
  replay, admin reset, and the MFA lockout

## Reports and a clean tree

`bin/300` writes the pytest log to `tmp/test-logs/pytest.log`, the API's to
`tmp/test-logs/py-api.log`, and an HTML report to
`tmp/test-reports/pytest_<timestamp>.html`. `tmp/` is gitignored. A run
must leave `git status` as it found it; `bin/300` fails otherwise, so a
test never writes into the tree.

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
