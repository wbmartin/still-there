import os
import pytest
import requests
import time
import warnings
import urllib3
from typing import Dict, Optional, List, Tuple

# Suppress SSL warnings for local development
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
warnings.filterwarnings("ignore", category=urllib3.exceptions.InsecureRequestWarning)


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
        if "tempToken" in data or "temp_token" in data:
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