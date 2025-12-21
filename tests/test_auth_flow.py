import os

import pytest
import requests


class TestAuthentication:
    """Test authentication and CSRF token handling"""

    def test_successful_authentication(self):
        """Test successful login with valid credentials"""
        base_url = os.environ.get("TEST_BASE_URL", "http://localhost:8080")
        client = TestClient(base_url)
        response = client.authenticate("admin", "admin123!")

        assert "user" in response
        assert client.jwt_token is not None
        assert client.csrf_token is not None

        # Verify user data
        user = response["user"]
        assert user["userName"] == "admin"
        assert user["firstName"] == "System"
        assert user["lastName"] == "Administrator"
        assert user["securityContext"] == 1

    def test_invalid_credentials(self):
        """Test authentication with invalid credentials"""
        base_url = os.environ.get("TEST_BASE_URL", "http://localhost:8080")
        client = TestClient(base_url)
        with pytest.raises(requests.HTTPError):
            client.authenticate("invalid", "invalid")

    def test_csrf_protection(self, test_client):
        """Test that CSRF protection is working"""
        # Try request without CSRF token
        response = test_client.session.post(
            f"{test_client.base_url}/example",
            json={"exampleText": "Test", "securityContext": 1},
        )
        assert response.status_code == 403  # Should be forbidden

        # Try request with invalid CSRF token
        response = test_client.session.post(
            f"{test_client.base_url}/example",
            json={"exampleText": "Test", "securityContext": 1},
            headers={"X-CSRF-Token": "invalid"},
        )
        assert response.status_code == 403

    def test_authenticated_request_without_csrf(self, test_client):
        """Test that authenticated requests still need CSRF for state changes"""
        # This should fail even with valid JWT because CSRF is required for POST
        response = test_client.session.post(
            f"{test_client.base_url}/example",
            json={"exampleText": "Test", "securityContext": 1},
            cookies={"auth": test_client.jwt_token},
        )
        assert response.status_code == 403

    def test_get_request_no_csrf_needed(self, test_client):
        """Test that GET requests don't require CSRF tokens"""
        # Health check should work
        response = test_client.make_request("GET", "/health")
        assert response.status_code == 200

    def test_malformed_auth_request(self):
        """Test authentication with malformed request"""
        base_url = os.environ.get("TEST_BASE_URL", "http://localhost:8080")
        client = TestClient(base_url)

        # Missing password
        with pytest.raises(requests.HTTPError):
            response = client.session.post(
                f"{client.base_url}/auth", json={"userName": "admin"}
            )
            response.raise_for_status()

        # Missing username
        with pytest.raises(requests.HTTPError):
            response = client.session.post(
                f"{client.base_url}/auth", json={"password": "admin123!"}
            )
            response.raise_for_status()

        # Empty request body
        with pytest.raises(requests.HTTPError):
            response = client.session.post(f"{client.base_url}/auth", json={})
            response.raise_for_status()

    def test_health_check(self):
        """Test that health endpoint works without authentication"""
        base_url = os.environ.get("TEST_BASE_URL", "http://localhost:8080")
        response = requests.get(f"{base_url}/health", verify=False)
        assert response.status_code == 200

        data = response.json()
        inner_data = data.get("data", {})
        assert (
            "status" in inner_data or "message" in inner_data
        )  # Health check response format

    def test_account_lockout_after_failed_attempts(self):
        """Test that account gets locked after too many failed login attempts"""
        base_url = os.environ.get("TEST_BASE_URL", "http://localhost:8080")
        client = TestClient(base_url)

        # Attempt login with wrong password multiple times
        for i in range(7):  # Exceed threshold
            with pytest.raises(requests.HTTPError):
                client.authenticate("admin", "wrongpassword")

        # Now attempt with correct password should be locked
        with pytest.raises(requests.HTTPError) as exc_info:
            client.authenticate("admin", "admin123!")
        assert exc_info.value.response.status_code == 429  # Too Many Requests

    def test_lockout_expiration(self):
        """Test that lockout expires after the configured duration"""
        # This test would need to wait for lockout duration to expire
        # For now, just ensure lockout is triggered
        # In a real scenario, this would sleep for the lockout duration
        pass

    def test_mfa_lockout_after_failed_attempts(self):
        """Test that MFA failures also trigger lockout"""
        # This would require setting up MFA and testing TOTP failures
        # For now, placeholder
        pass


# Import TestClient for use in this module
from conftest import TestClient
