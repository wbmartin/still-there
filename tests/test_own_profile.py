import os

import pytest
import requests

from conftest import OTHER_TENANT, TEST_TENANT, TestClient


class TestOwnProfile:
    """GET/PUT /me: a user reads and edits their own row and preferences,
    with LOGON only; the row is always the caller's, never one named in
    the body. Restores the fixture user at the end."""

    def test_basic_user_profile_round_trip(self, basic_user_client):
        client = basic_user_client

        response = client.make_request("GET", "/me")
        assert response.status_code == 200
        me = response.json()["data"]
        assert me["userName"] == "basic"
        assert "pwHash" not in me and "totpSeed" not in me
        assert "password" not in me
        assert "preferenceJsonb" not in me  # fixture user has never saved any
        original = {k: me[k] for k in ("firstName", "lastName", "email")}

        # Update name, email and preferences; an appUserId in the body is
        # ignored (the handler only reads the JWT).
        update = {
            "appUserId": 1,
            "firstName": "Basic",
            "lastName": "Profile",
            "email": "basic.profile@example.com",
            "preferenceJsonb": {"colorScheme": "dark", "rowsPerPage": 25},
            "updateTs": me["updateTs"],
        }
        response = client.make_request("PUT", "/me", json=update)
        assert response.status_code == 200, response.text
        saved = response.json()["data"]
        assert saved["appUserId"] == me["appUserId"]
        assert saved["firstName"] == "Basic"
        assert saved["email"] == "basic.profile@example.com"
        assert saved["preferenceJsonb"] == {"colorScheme": "dark", "rowsPerPage": 25}
        assert saved["updateTs"] > me["updateTs"]
        assert saved["updateUserId"] == me["appUserId"]

        # Stale lock is refused.
        response = client.make_request("PUT", "/me", json=update)
        assert response.status_code == 409

        # Validation: bad email, missing name, non-object preferences.
        for bad in (
            {**update, "updateTs": saved["updateTs"], "email": "nope"},
            {**update, "updateTs": saved["updateTs"], "lastName": ""},
            {**update, "updateTs": saved["updateTs"], "preferenceJsonb": [1, 2]},
        ):
            response = client.make_request("PUT", "/me", json=bad)
            assert response.status_code == 400, bad

        # GET reflects the save; then put the fixture user back, clearing
        # the preferences (absent → NULL → absent).
        response = client.make_request("GET", "/me")
        assert response.json()["data"]["preferenceJsonb"]["rowsPerPage"] == 25
        response = client.make_request(
            "PUT", "/me", json={**original, "updateTs": saved["updateTs"]}
        )
        assert response.status_code == 200, response.text
        restored = response.json()["data"]
        assert restored["firstName"] == original["firstName"]
        assert "preferenceJsonb" not in restored

    def test_me_requires_a_session(self, test_client):
        import requests

        response = requests.get(f"{test_client.base_url}/me", verify=False)
        assert response.status_code == 401


def _switch_context(client: TestClient, security_context: int) -> dict:
    """POST /changeSecurityContext and adopt the new cookie and CSRF token.

    The switch issues a browser-session cookie with the same attributes as
    /auth and /refresh (B029): no Max-Age or Expires, HttpOnly, Secure,
    SameSite=Strict, Path=/.
    """
    response = client.make_request(
        "POST", "/changeSecurityContext",
        json={"securityContext": security_context},
    )
    assert response.status_code == 200, response.text
    cookie_header = response.headers.get("Set-Cookie", "")
    assert cookie_header.startswith("auth="), cookie_header
    lowered = cookie_header.lower()
    assert "max-age" not in lowered, cookie_header
    assert "expires" not in lowered, cookie_header
    for flag in ("HttpOnly", "Secure", "SameSite=Strict", "Path=/"):
        assert flag in cookie_header, cookie_header

    client.jwt_token = cookie_header.split("auth=")[1].split(";")[0]
    client.csrf_token = response.json()["csrfToken"]
    client.session.cookies.clear()
    client.session.cookies.set("auth", client.jwt_token, path="/")
    user = response.json()["user"]
    assert user["securityContext"] == security_context
    return user


class TestOwnProfileAfterContextSwitch:
    """A user's row stays in its home tenant when /changeSecurityContext
    moves the session to another; /me still reads and writes that row.
    The fixture user switcher is homed in TEST_TENANT with a group in
    OTHER_TENANT. Restores the user and switches back at the end."""

    def test_me_reads_and_writes_the_home_row_after_a_switch(self):
        client = TestClient(os.environ["TEST_BASE_URL"])
        login = client.authenticate("switcher", "basic123!")
        assert sorted(login["user"]["availableSecurityContextList"]) == [
            TEST_TENANT, OTHER_TENANT,
        ]
        home = client.make_request("GET", "/me")
        assert home.status_code == 200, home.text
        home_me = home.json()["data"]
        assert home_me["securityContext"] == TEST_TENANT
        home_token = client.jwt_token

        _switch_context(client, OTHER_TENANT)
        # The switch ends the session it replaces (B003).
        replaced = requests.get(f"{client.base_url}/me",
                                cookies={"auth": home_token}, verify=False)
        assert replaced.status_code == 401

        response = client.make_request("GET", "/me")
        assert response.status_code == 200, response.text
        me = response.json()["data"]
        assert me["appUserId"] == home_me["appUserId"]
        assert me["userName"] == "switcher"
        assert me["securityContext"] == TEST_TENANT  # the row stays home

        original = {k: me[k] for k in ("firstName", "lastName", "email")}
        response = client.make_request("PUT", "/me", json={
            "firstName": "Switched",
            "lastName": "Context",
            "email": "switched@example.com",
            "preferenceJsonb": {"colorScheme": "dark"},
            "updateTs": me["updateTs"],
        })
        assert response.status_code == 200, response.text
        saved = response.json()["data"]
        assert saved["appUserId"] == me["appUserId"]
        assert saved["securityContext"] == TEST_TENANT
        assert saved["firstName"] == "Switched"
        assert saved["preferenceJsonb"] == {"colorScheme": "dark"}

        _switch_context(client, TEST_TENANT)
        response = client.make_request("GET", "/me")
        assert response.status_code == 200, response.text
        assert response.json()["data"]["firstName"] == "Switched"

        response = client.make_request(
            "PUT", "/me", json={**original, "updateTs": saved["updateTs"]}
        )
        assert response.status_code == 200, response.text
        restored = response.json()["data"]
        assert restored["firstName"] == original["firstName"]
        assert "preferenceJsonb" not in restored
