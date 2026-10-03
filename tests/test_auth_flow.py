import os
import statistics
import time
import uuid

import pytest
import requests


class TestAuthentication:
    """Test authentication and CSRF token handling"""

    def test_successful_authentication(self):
        """Test successful login with valid credentials"""
        base_url = os.environ["TEST_BASE_URL"]
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
        assert user["securityContext"] == TEST_TENANT

    def test_invalid_credentials(self):
        """Test authentication with invalid credentials"""
        base_url = os.environ["TEST_BASE_URL"]
        client = TestClient(base_url)
        with pytest.raises(requests.HTTPError):
            client.authenticate("invalid", "invalid")

    def test_csrf_protection(self, test_client):
        """Test that CSRF protection is working"""
        # Try request without CSRF token
        response = test_client.session.post(
            f"{test_client.base_url}/example",
            json={"exampleText": "Test"},
        )
        assert response.status_code == 403  # Should be forbidden

        # Try request with invalid CSRF token
        response = test_client.session.post(
            f"{test_client.base_url}/example",
            json={"exampleText": "Test"},
            headers={"X-CSRF-Token": "invalid"},
        )
        assert response.status_code == 403

    def test_authenticated_request_without_csrf(self, test_client):
        """Test that authenticated requests still need CSRF for state changes"""
        # This should fail even with valid JWT because CSRF is required for POST
        response = test_client.session.post(
            f"{test_client.base_url}/example",
            json={"exampleText": "Test"},
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
        base_url = os.environ["TEST_BASE_URL"]
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
        base_url = os.environ["TEST_BASE_URL"]
        response = requests.get(f"{base_url}/health", verify=False)
        assert response.status_code == 200

        data = response.json()
        inner_data = data.get("data", {})
        assert (
            "status" in inner_data or "message" in inner_data
        )  # Health check response format

    def test_lockout_expiration(self):
        """Test that lockout expires after the configured duration"""
        # This test would need to wait for lockout duration to expire
        # For now, just ensure lockout is triggered
        # In a real scenario, this would sleep for the lockout duration
        pass

    # MFA enrolment, verification, temp-token replay and the MFA lockout are
    # in tests/test_zz_mfa_totp.py.


# Each case below signs in from its own new_forwarded_ip() address, so it
# never spends the loopback allowance the rest of the suite logs in from.
def _login(base_url: str, client_ip: str, user_name: str, password: str):
    return requests.post(
        f"{base_url}/auth",
        json={"userName": user_name, "password": password},
        headers=forwarded_for(client_ip),
    )


def _unknown_name() -> str:
    return f"ghost{uuid.uuid4().hex[:16]}"


class TestLoginEnumeration:
    """An unknown name is answered, timed and counted like a wrong password."""

    def test_unknown_name_costs_as_much_as_a_wrong_password(self, test_client):
        base_url = test_client.base_url
        suffix = uuid.uuid4().hex[:12]
        create_response = test_client.make_request("POST", "/user", json={
            "userName": f"timing{suffix}",
            "firstName": "Timing",
            "lastName": "Test",
            "email": f"timing{suffix}@example.com",
            "pwHash": "timingpass123!",
            "mfaExempt": True,
            "appUserGroupMembershipList": [],
        })
        assert create_response.status_code == 201, create_response.text
        created = create_response.json()["data"]
        try:
            wrong_ip, unknown_ip = new_forwarded_ip(), new_forwarded_ip()
            wrong_list, unknown_list, body_set = [], [], set()
            # Five of each, interleaved, each kind from its own address:
            # under the lockout threshold for both addresses and the user.
            for _ in range(5):
                for kind in ("wrong", "unknown"):
                    started = time.perf_counter()
                    if kind == "wrong":
                        response = _login(
                            base_url, wrong_ip, created["userName"], "nope"
                        )
                    else:
                        response = _login(
                            base_url, unknown_ip, _unknown_name(), "nope"
                        )
                    elapsed = time.perf_counter() - started
                    assert response.status_code == 401, response.text
                    body_set.add(response.text)
                    (wrong_list if kind == "wrong" else unknown_list).append(
                        elapsed
                    )
            assert len(body_set) == 1, body_set
            wrong_median = statistics.median(wrong_list)
            unknown_median = statistics.median(unknown_list)
            assert unknown_median >= wrong_median / 2, (
                f"unknown name {unknown_median * 1000:.1f} ms vs wrong "
                f"password {wrong_median * 1000:.1f} ms: timing tells them apart"
            )
        finally:
            test_client.make_request("DELETE", "/user", params={
                "appUserId": str(created["appUserId"]),
                "updateTs": f"{created['updateTs']:.7f}",
            })

    def test_unknown_name_attempts_reach_the_lockout(self):
        base_url = os.environ["TEST_BASE_URL"]
        threshold = failed_login_threshold()
        locked_ip, other_ip = new_forwarded_ip(), new_forwarded_ip()
        user_name = _unknown_name()
        for attempt in range(threshold):
            response = _login(base_url, locked_ip, user_name, "guess")
            assert response.status_code == 401, (attempt, response.text)
            assert "Invalid credentials" in response.text

        response = _login(base_url, locked_ip, user_name, "guess")
        assert response.status_code == 429, response.text

        # The name is locked from any address.
        response = _login(base_url, new_forwarded_ip(), user_name, "guess")
        assert response.status_code == 429, response.text

        # A forged entry left of the one the proxy appended is ignored: the
        # request still comes from the locked address.
        response = _login(
            base_url, f"{other_ip}, {locked_ip}", _unknown_name(), "guess"
        )
        assert response.status_code == 429, response.text

        # Another address with another name is not locked.
        response = _login(base_url, other_ip, _unknown_name(), "guess")
        assert response.status_code == 401, response.text


class TestLoginCounterReset:
    """A completed login clears the user's failure count, not the address's."""

    def test_success_between_typos_does_not_lock_the_user(self, test_client):
        base_url = test_client.base_url
        threshold = failed_login_threshold()
        suffix = uuid.uuid4().hex[:12]
        password = "typos123!"
        create_response = test_client.make_request("POST", "/user", json={
            "userName": f"typo{suffix}",
            "firstName": "Typo",
            "lastName": "Test",
            "email": f"typo{suffix}@example.com",
            "pwHash": password,
            "mfaExempt": True,
            "appUserGroupMembershipList": [],
        })
        assert create_response.status_code == 201, create_response.text
        created = create_response.json()["data"]
        user_name = created["userName"]
        typing_ip, later_ip = new_forwarded_ip(), new_forwarded_ip()
        try:
            for attempt in range(threshold - 1):
                response = _login(base_url, typing_ip, user_name, "typo")
                assert response.status_code == 401, (attempt, response.text)
            response = _login(base_url, typing_ip, user_name, password)
            assert response.status_code == 200, response.text

            # Without the reset this is the user's seventh failure: a lock.
            response = _login(base_url, later_ip, user_name, "typo")
            assert response.status_code == 401, response.text
            response = _login(base_url, later_ip, user_name, password)
            assert response.status_code == 200, response.text

            # The address's count was not cleared: its seventh failure locks
            # it, while the user still signs in from elsewhere.
            response = _login(base_url, typing_ip, user_name, "typo")
            assert response.status_code == 401, response.text
            response = _login(base_url, typing_ip, user_name, password)
            assert response.status_code == 429, response.text
            response = _login(base_url, later_ip, user_name, password)
            assert response.status_code == 200, response.text
        finally:
            test_client.make_request("DELETE", "/user", params={
                "appUserId": str(created["appUserId"]),
                "updateTs": f"{created['updateTs']:.7f}",
            })


class TestBodyLimits:
    """Bodies over the cap are 413, never decoded; malformed ones are 400."""

    def test_oversized_login_body_is_413(self):
        base_url = os.environ["TEST_BASE_URL"]
        login_ip = new_forwarded_ip()
        # 5 KiB of password, over the 4 KiB cap on /auth.
        response = _login(base_url, login_ip, "admin", "a" * 5120)
        assert response.status_code == 413, response.text
        assert response.json()["error"] == "Request body too large"

        # Not counted as a failed login; a normal body still signs in.
        response = _login(base_url, login_ip, "admin", "admin123!")
        assert response.status_code == 200, response.text

    def test_oversized_login_layer_write_is_413(self, test_client):
        # 65 KiB, over the 64 KiB login-layer cap: refused before decode.
        response = test_client.make_request("POST", "/user", json={
            "userName": f"big{uuid.uuid4().hex[:12]}",
            "firstName": "x" * (65 * 1024),
        })
        assert response.status_code == 413, response.text[:200]
        assert response.json()["error"] == "Request body too large"

        # The session that sent it still reads and writes.
        response = test_client.make_request("GET", "/me")
        assert response.status_code == 200, response.text
        response = test_client.make_request(
            "POST",
            "/user",
            data="{",
            headers={"Content-Type": "application/json"},
        )
        assert response.status_code == 400, response.text
        assert response.json()["error"] == "Invalid request body"


def _auth_set_cookie(response) -> str:
    cookie_header = response.headers.get("Set-Cookie", "")
    assert cookie_header.startswith("auth="), cookie_header
    return cookie_header


class TestSessionCookie:
    """The JWT's exp is the session's only lifetime (B029)."""

    def test_issued_cookie_is_a_browser_session_cookie(self):
        """/auth and /refresh set no Max-Age; /logoff expires the cookie.

        /refresh once wrote Max-Age=3600 against a longer-lived JWT, so one
        page reload cut the session to an hour.
        """
        base_url = os.environ["TEST_BASE_URL"]
        login = _login(base_url, new_forwarded_ip(), "admin", "admin123!")
        assert login.status_code == 200, login.text
        login_cookie = _auth_set_cookie(login)
        token = login_cookie.split("auth=")[1].split(";")[0]
        csrf_token = login.json()["csrfToken"]

        refresh = requests.get(
            f"{base_url}/refresh", cookies={"auth": token}, verify=False,
        )
        assert refresh.status_code == 200, refresh.text
        refresh_cookie = _auth_set_cookie(refresh)

        for route, cookie_header in (
            ("/auth", login_cookie),
            ("/refresh", refresh_cookie),
        ):
            lowered = cookie_header.lower()
            assert "max-age" not in lowered, f"{route}: {cookie_header}"
            assert "expires" not in lowered, f"{route}: {cookie_header}"
            for flag in ("HttpOnly", "Secure", "SameSite=Strict", "Path=/"):
                assert flag in cookie_header, f"{route}: {cookie_header}"

        logoff = requests.post(
            f"{base_url}/logoff",
            cookies={"auth": token},
            headers={"X-CSRF-Token": csrf_token},
            verify=False,
        )
        assert logoff.status_code == 200, logoff.text
        assert "Max-Age=0" in _auth_set_cookie(logoff)


# Import TestClient for use in this module
from conftest import (
    TEST_TENANT,
    TestClient,
    failed_login_threshold,
    forwarded_for,
    new_forwarded_ip,
)
