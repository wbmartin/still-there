import functools
import itertools
import os
import pytest
import requests
import time
import warnings
import urllib3
import uuid
from typing import Dict, Optional, List, Tuple
from urllib.parse import urlparse

# Suppress SSL warnings for local development
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
warnings.filterwarnings("ignore", category=urllib3.exceptions.InsecureRequestWarning)

# The tenants the suite acts in. Tenant 1 is the first real tenant and does
# not exist in a test database (sql/fixtures/test-tenant-lock.sql); the
# fixture logins are in TEST_TENANT, seed groups 4 and 5 in OTHER_TENANT.
TEST_TENANT = 2
OTHER_TENANT = 3

# pytest.exit status for "pointed at something that is not the test API".
WRONG_TARGET_EXIT_CODE = 3


def _read_env_file(path: str) -> Dict[str, str]:
    """The flat KEY=VALUE config file the API server reads."""
    values: Dict[str, str] = {}
    with open(path, encoding="utf-8") as env_file:
        for line in env_file:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            values[key.strip()] = value.strip().strip('"').strip("'")
    return values


@functools.lru_cache(maxsize=None)
def api_env() -> Dict[str, str]:
    """The config the test API was started from (TEST_API_ENV)."""
    return _read_env_file(os.environ["TEST_API_ENV"])


def failed_login_threshold() -> int:
    """Failures that lock an address or a name, as the server counts them."""
    return int(api_env()["FAILED_LOGIN_ATTEMPT_THRESHOLD"])


# The test API trusts one X-Forwarded-For hop (checked in require_test_api),
# so a test that counts towards a lockout or a per-address rate limit sends
# its own address from 198.18.0.0/15 (benchmarking range, never a real
# client) and leaves loopback, which the rest of the suite signs in from,
# untouched. Each run takes the /24 its start second picks out of 512: runs
# started under 512 s apart never share one, and anything older is past the
# 5 min lockout and attempt window.
_FORWARDED_RUN_BLOCK = int(time.time()) % 512
_forwarded_host_counter = itertools.count(1)


def new_forwarded_ip() -> str:
    """An address no other call in this run (or a recent run) returned."""
    host = next(_forwarded_host_counter)
    if host > 254:
        raise RuntimeError("forwarded addresses used up for this run")
    second_octet = 18 + _FORWARDED_RUN_BLOCK // 256
    third_octet = _FORWARDED_RUN_BLOCK % 256
    return f"198.{second_octet}.{third_octet}.{host}"


def forwarded_for(client_ip: str) -> Dict[str, str]:
    """Headers that make a request come from client_ip."""
    return {"X-Forwarded-For": client_ip}


def require_test_api() -> None:
    """Refuse to start unless TEST_BASE_URL is the private test API.

    TEST_API_ENV names the config the test API was started from
    (bin/300-run-tests exports it). TEST_BASE_URL must equal that file's
    TEST_BASE_URL, and its DATABASE_URL must name a scratch database, so a
    hand-set URL can never aim the suite's writes at a developer's API. It
    must trust exactly one proxy hop, or every new_forwarded_ip() request
    would key on loopback and the lockout cases would lock the suite out.
    """
    api_env_path = os.environ.get("TEST_API_ENV")
    base_url = os.environ.get("TEST_BASE_URL")
    if not api_env_path or not base_url:
        pytest.exit(
            "TEST_API_ENV and TEST_BASE_URL must both be set; run "
            "bin/300-run-tests py, which starts the private test API",
            returncode=WRONG_TARGET_EXIT_CODE,
        )
    try:
        api_config = _read_env_file(api_env_path)
    except OSError as error:
        pytest.exit(f"TEST_API_ENV {api_env_path} unreadable: {error}",
                    returncode=WRONG_TARGET_EXIT_CODE)
    expected_base_url = api_config.get("TEST_BASE_URL", "")
    if base_url.rstrip("/") != expected_base_url.rstrip("/"):
        pytest.exit(
            f"TEST_BASE_URL {base_url} is not the test API's "
            f"({expected_base_url!r} in {api_env_path}): refusing to run",
            returncode=WRONG_TARGET_EXIT_CODE,
        )
    database_name = urlparse(api_config.get("DATABASE_URL", "")).path.lstrip("/")
    if not database_name.endswith(("_test_db", "_e2e_db")):
        pytest.exit(
            f"{api_env_path} serves database {database_name!r}, not a "
            "scratch database (*_test_db / *_e2e_db): refusing to run",
            returncode=WRONG_TARGET_EXIT_CODE,
        )
    proxy_count = api_config.get("TRUSTED_PROXY_COUNT", "")
    if proxy_count != "1":
        pytest.exit(
            f"{api_env_path} sets TRUSTED_PROXY_COUNT={proxy_count!r}, not 1: "
            "per-test X-Forwarded-For addresses would all key on loopback",
            returncode=WRONG_TARGET_EXIT_CODE,
        )


def require_test_tenant(user: Dict, expected_tenant: int = TEST_TENANT) -> None:
    """Stop the whole run when a fixture login lands outside the test tenant:
    the API is then serving a database that is not a test database."""
    tenant = user.get("securityContext")
    if tenant != expected_tenant:
        pytest.exit(
            f"{user.get('userName')!r} signed in to tenant {tenant}, not "
            f"{expected_tenant}: this API does not serve a test database",
            returncode=WRONG_TARGET_EXIT_CODE,
        )


def pytest_sessionstart(session):
    require_test_api()


class TestClient:
    """HTTP client for testing the API with authentication and CSRF support"""
    __test__ = False

    def __init__(self, base_url: str):
        self.base_url = base_url
        self.session = requests.Session()
        # Disable SSL verification for self-signed certificates in testing
        self.session.verify = False
        # Suppress SSL warnings
        import urllib3
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        self.jwt_token: Optional[str] = None
        self.csrf_token: Optional[str] = None
        self.created_examples: List[Tuple[int, float]] = []  # (id, update_ts) pairs
        self.created_users: List[Tuple[int, float]] = []  # (id, update_ts) pairs

    def authenticate(self, username: str = "admin", password: str = "admin123!") -> Dict:
        """Authenticate and store JWT/CSRF tokens"""
        # Initial auth request
        response = self.session.post(f"{self.base_url}/auth", json={
            "userName": username,
            "password": password
        })
        response.raise_for_status()

        data = response.json()

        # The suite's users are seeded mfa_exempt (sql/fixtures). A second
        # factor cannot be completed from a test, so say so rather than
        # pretend: the old /auth2 branch here called an endpoint that does
        # not exist (the real step is POST /mfaVerify with a TOTP code).
        # The token sits inside "user" (AuthResponseWrapper); the top-level
        # check is kept for older shapes. MFA itself is covered by
        # test_zz_mfa_totp.py, which drives /auth directly.
        if "tempToken" in data or "tempToken" in data.get("user", {}):
            pytest.fail(f"user {username} requires MFA; test users must be mfa_exempt")

        # Extract tokens from final response
        # Manually parse Set-Cookie header for JWT token (workaround for HTTP/2 cookie issues)
        self.jwt_token = None
        if 'Set-Cookie' in response.headers:
            cookie_header = response.headers['Set-Cookie']
            if 'auth=' in cookie_header:
                # Extract the token value from the cookie header
                auth_part = cookie_header.split('auth=')[1].split(';')[0]
                self.jwt_token = auth_part

        self.csrf_token = data.get("csrfToken")
        require_test_tenant(data.get("user", {}))

        # Set the auth cookie in the session for future requests
        if self.jwt_token:
            self.session.cookies.set("auth", self.jwt_token, path="/")

        return data

    def make_request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """Make authenticated request with CSRF token"""
        url = f"{self.base_url}{endpoint}"

        # Add CSRF token for state-changing requests
        if method.upper() in ["POST", "PUT", "DELETE"]:
            headers = kwargs.get("headers", {})
            if self.csrf_token:
                headers["X-CSRF-Token"] = self.csrf_token
            kwargs["headers"] = headers

        # For GET requests, append params to url since session.request may not
        if method.upper() == "GET" and "params" in kwargs:
            from urllib.parse import urlencode
            param_str = urlencode(kwargs["params"])
            if param_str:
                separator = "&" if "?" in url else "?"
                url += separator + param_str
            del kwargs["params"]

        return self.session.request(method, url, **kwargs)

    def track_example(self, example_id: int, update_ts: float):
        """Track a created example for cleanup"""
        self.created_examples.append((example_id, update_ts))

    def track_user(self, app_user_id: int, update_ts: float):
        """Track a created user for cleanup"""
        self.created_users.append((app_user_id, update_ts))

    def cleanup_examples(self):
        """Clean up all tracked examples and users"""
        # Clean up examples
        if self.created_examples:
            ids = []
            update_timestamps = []

            for example_id, update_ts in self.created_examples:
                ids.append(str(example_id))
                update_timestamps.append(f"{update_ts:.7f}")

            try:
                response = self.make_request("DELETE", "/exampleList",
                    params={
                        "exampleIdList": ",".join(ids),
                        "updateTs": ",".join(update_timestamps)
                    })
                if response.status_code == 200:
                    print(f"Cleaned up {len(self.created_examples)} test examples")
                else:
                    print(f"Warning: Example cleanup failed with status {response.status_code}")
            except Exception as e:
                print(f"Warning: Example cleanup failed with error: {e}")
            finally:
                self.created_examples.clear()

        # Clean up users (single delete for each)
        if self.created_users:
            cleaned = 0
            for app_user_id, update_ts in self.created_users:
                try:
                    response = self.make_request("DELETE", "/user",
                        params={
                            "appUserId": str(app_user_id),
                            "updateTs": f"{update_ts:.7f}"
                        })
                    if response.status_code == 200:
                        cleaned += 1
                    else:
                        print(f"Warning: Failed to clean user {app_user_id}, status {response.status_code}")
                except Exception as e:
                    print(f"Warning: User cleanup failed for {app_user_id}: {e}")
            if cleaned > 0:
                print(f"Cleaned up {cleaned} test users")
            self.created_users.clear()


# A /changeOwnPassword body for the CSRF cases. The route is core (LOGON
# only) and checks CSRF before it reads the body, so the cases survive a
# derived app deleting the /example scaffold; "x" is never the password, so
# a request that got past CSRF would still change nothing.
CHANGE_PASSWORD_PROBE = {"currentPassword": "x", "newPassword": "y"}


def assert_csrf_refused(response, message: str) -> None:
    """403 from the CSRF check itself, not some other refusal."""
    assert response.status_code == 403, response.text
    assert response.json() == {"error": message, "code": "CSRF"}


BASIC_USER_GROUP_ID = 1  # seed group in TEST_TENANT; carries LOGON


def create_throwaway_login(
    admin_client: TestClient, prefix: str
) -> Tuple[TestClient, Dict]:
    """A new MFA-exempt Basic User login, signed in: (client, created row).

    For a test that edits the row it signs in as, or spends that login's
    rate-limit bucket, so no shared fixture user is ever left changed or
    throttled. Pair with delete_user.
    """
    user_name = f"{prefix}{uuid.uuid4().hex[:12]}"
    password = f"Tmp-{uuid.uuid4().hex[:12]}!"
    response = admin_client.make_request("POST", "/user", json={
        "userName": user_name,
        "firstName": "Throwaway",
        "lastName": "Login",
        "email": f"{user_name}@example.com",
        "password": password,
        "mfaExempt": True,
        "appUserGroupMembershipList": [BASIC_USER_GROUP_ID],
    })
    assert response.status_code == 201, response.text[:300]
    created = response.json()["data"]
    client = TestClient(admin_client.base_url)
    try:
        client.authenticate(user_name, password)
    except Exception:
        delete_user(admin_client, created["appUserId"])
        raise
    return client, created


def delete_user(admin_client: TestClient, app_user_id: int) -> None:
    """Delete a user at its current updateTs (the admin is never throttled)."""
    rows = admin_client.make_request(
        "GET", "/userList", params={"appUserIdList": str(app_user_id)}
    ).json()["data"]
    if rows:
        response = admin_client.make_request("DELETE", "/user", params={
            "appUserId": str(app_user_id),
            "updateTs": f"{rows[0]['updateTs']:.7f}",
        })
        assert response.status_code == 200, response.text[:300]


@pytest.fixture(scope="class")
def test_client():
    """Class-scoped test client with admin authentication"""
    # Get base URL from environment variable set by test script
    base_url = os.environ.get("TEST_BASE_URL")
    if not base_url:
        pytest.fail("TEST_BASE_URL environment variable is required")
    client = TestClient(base_url)

    # Check if application is running
    try:
        response = requests.get(f"{base_url}/health", timeout=5, verify=False)
        if response.status_code != 200:
            pytest.fail(f"Application health check failed. Is the application running on {base_url}?")
    except requests.RequestException:
        pytest.fail(f"Cannot connect to application. Is the application running on {base_url}?")

    # Authenticate once per test session as admin
    client.authenticate()
    yield client


@pytest.fixture(scope="function")
def basic_user_client():
    """Session-scoped test client with basic user authentication"""
    # Get base URL from environment variable set by test script
    base_url = os.environ.get("TEST_BASE_URL")
    if not base_url:
        pytest.fail("TEST_BASE_URL environment variable is required")
    client = TestClient(base_url)

    # Check if application is running
    try:
        response = requests.get(f"{base_url}/health", timeout=5, verify=False)
        if response.status_code != 200:
            pytest.fail(f"Application health check failed. Is the application running on {base_url}?")
    except requests.RequestException:
        pytest.fail(f"Cannot connect to application. Is the application running on {base_url}?")

    # Authenticate once per test session as basic user
    client.authenticate("basic", "basic123!")
    yield client

    # Cleanup all created examples
    client.cleanup_examples()


@pytest.fixture(scope="function")
def example_tracker(test_client):
    """Fixture to track examples created in individual tests"""
    initial_count = len(test_client.created_examples)
    yield test_client
    # Individual test cleanup happens in session fixture
    # This just ensures tracking works per test if needed