import pytest


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
