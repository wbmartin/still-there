import os
import pytest
import requests
import time
import warnings
import urllib3
import psycopg2
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

    def authenticate(self, username: str = "admin", password: str = "admin123!") -> Dict:
        """Authenticate and store JWT/CSRF tokens"""
        # Initial auth request
        response = self.session.post(f"{self.base_url}/auth", json={
            "userName": username,
            "password": password
        })
        response.raise_for_status()

        data = response.json()

        # Check if MFA is required
        if "temp_token" in data:
            # MFA required, complete 2FA
            temp_token = data["temp_token"]
            response = self.session.post(f"{self.base_url}/auth2", json={
                "temp_token": temp_token,
                "totp_code": "123456"  # Placeholder TOTP code
            })
            response.raise_for_status()
            data = response.json()

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

        return self.session.request(method, url, **kwargs)

    def track_example(self, example_id: int, update_ts: float):
        """Track a created example for cleanup"""
        self.created_examples.append((example_id, update_ts))

    def cleanup_examples(self):
        """Clean up all tracked examples"""
        if not self.created_examples:
            return

        # Group examples for bulk delete
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
                print(f"Warning: Cleanup failed with status {response.status_code}")
        except Exception as e:
            print(f"Warning: Cleanup failed with error: {e}")
        finally:
            self.created_examples.clear()


@pytest.fixture(scope="session")
def test_client():
    """Session-scoped test client with authentication"""
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

    # Authenticate once per test session
    client.authenticate()
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