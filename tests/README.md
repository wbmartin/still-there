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

or, by hand against a server you start yourself:

```bash
bin/215-run-api-srv                        # reads TEST_BASE_URL from etc/loc-api-srv.env
TEST_BASE_URL=https://localhost:8080 tests/.venv/bin/python -m pytest tests/
```

`bin/300-run-tests` bootstraps the venv at `tests/.venv` itself (creating
it and installing `tests/requirements.txt` on first run), checks
`GET /health`, then runs pytest with an HTML report written to
`tmp/test-reports/` (gitignored).

## Configuration

`TEST_BASE_URL` is read from `etc/loc-api-srv.env` (or the file
`$TEST_API_ENV` points at) by `bin/300-run-tests`, then exported
as the `TEST_BASE_URL` environment variable that `conftest.py` fixtures
read directly. Local default: `http://localhost:8080`.

## Test Client

`conftest.py` defines `TestClient`: authenticates against `POST /auth`,
captures the JWT from the `auth` cookie and the CSRF token from the
response body, and attaches `X-CSRF-Token` to state-changing requests.
The fixture users (`admin`, `basic`) are MFA-exempt, so the client
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
- `test_security_context_isolation.py` — security contexts isolate data
  and permissions
- `test_user_management.py` — user management operations (admin only)
- `test_z_error_scenarios.py` — error conditions and status codes

## Requirements

`requirements.txt`: pytest, requests, pytest-html.
