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
are set, `TEST_BASE_URL` equals the one in `TEST_API_ENV`, and that file's
`DATABASE_URL` names a scratch database (`*_test_db` / `*_e2e_db`).

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
- `test_auth_flow.py` — authentication and CSRF token handling
- `test_basic_user_authorization.py` — basic users denied admin-only ops
- `test_bulk_operations.py` — bulk CRUD via `/exampleList`
- `test_error_reporting.py` — client-side error reporting endpoint
- `test_security_context_isolation.py` — tenant 2 cannot use or probe
  tenant 3's groups; client-sent tenants are ignored
- `test_session_revocation.py` — sign-out, an admin password change and a
  group grant edit end live sessions (old cookie gets 401)
- `test_user_management.py` — user management operations (admin only)
- `test_z_error_scenarios.py` — error conditions and status codes

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
