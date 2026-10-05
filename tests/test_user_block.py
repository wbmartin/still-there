"""Block and unblock a login over HTTP (B034).

POST /admin/blockUser sets account_status = blocked (so /auth refuses the
login), ends the login's sessions, and sets the BLOCKED* KV flag that keeps
refusing a pre-block token after an API restart (the restart half is proved
in Go, handler/user_block_handler_test.go). POST /admin/unblockUser sets the
row active and clears the flag. Grant BLOCK_USER (9008): the fixture Admin
group carries it, Basic User does not. Every login here is in the test
tenant (2); the one under test is made here and deleted after, so no shared
fixture login is ever blocked.
"""
import os
import secrets
import uuid

import pytest
import requests

from conftest import TestClient, forwarded_for, new_forwarded_ip


def _base_url() -> str:
    return os.environ["TEST_BASE_URL"]


def _me_with(token: str) -> int:
    """GET /me with only this auth cookie; returns the status."""
    response = requests.get(
        f"{_base_url()}/me", cookies={"auth": token}, verify=False
    )
    return response.status_code


@pytest.fixture
def blockee(test_client):
    """A throwaway active login, signed in: (client, user_name, password)."""
    user_name = f"blockee{uuid.uuid4().hex[:12]}"
    password = "Blk-" + secrets.token_hex(8)
    create = test_client.make_request("POST", "/user", json={
        "userName": user_name,
        "firstName": "Block",
        "lastName": "Target",
        "email": f"{user_name}@example.com",
        "password": password,
        "mfaExempt": True,
        "appUserGroupMembershipList": [1],  # Basic User: carries LOGON
    })
    assert create.status_code == 201, create.text[:300]
    app_user_id = create.json()["data"]["appUserId"]

    client = TestClient(_base_url())
    client.authenticate(user_name, password)
    yield client, user_name, password

    # unblock if a test left it blocked, then delete at the current update_ts
    test_client.make_request(
        "POST", "/admin/unblockUser", json={"userName": user_name}
    )
    rows = test_client.make_request(
        "GET", "/userList", params={"appUserIdList": str(app_user_id)}
    ).json()["data"]
    if rows:
        test_client.make_request("DELETE", "/user", params={
            "appUserId": str(app_user_id),
            "updateTs": f"{rows[0]['updateTs']:.7f}",
        })


def _status_of(test_client: TestClient, user_name: str) -> str:
    rows = test_client.make_request("GET", "/userList").json()["data"]
    return next(r["accountStatus"] for r in rows if r["userName"] == user_name)


class TestUserBlock:
    def test_block_ends_session_and_login_then_unblock_restores(
        self, test_client, blockee
    ):
        client, user_name, password = blockee
        token = client.jwt_token
        assert _me_with(token) == 200
        assert _status_of(test_client, user_name) == "active"

        blocked = test_client.make_request(
            "POST", "/admin/blockUser", json={"userName": user_name}
        )
        assert blocked.status_code == 200, blocked.text[:300]
        assert blocked.json()["data"]["accountStatus"] == "blocked"
        assert _status_of(test_client, user_name) == "blocked"

        # the session minted before the block is ended at once
        assert _me_with(token) == 401

        # and the login itself is refused like a wrong password (from its
        # own address, so the failure counts against no other test)
        login = requests.post(
            f"{_base_url()}/auth",
            json={"userName": user_name, "password": password},
            headers=forwarded_for(new_forwarded_ip()),
            verify=False,
        )
        assert login.status_code == 401, login.text[:300]
        assert "auth=" not in login.headers.get("Set-Cookie", "")

        unblocked = test_client.make_request(
            "POST", "/admin/unblockUser", json={"userName": user_name}
        )
        assert unblocked.status_code == 200, unblocked.text[:300]
        assert unblocked.json()["data"]["accountStatus"] == "active"
        assert _me_with(token) == 401, "unblock revived a pre-block session"

        fresh = TestClient(_base_url())
        fresh.authenticate(user_name, password)
        assert fresh.make_request("GET", "/me").status_code == 200

    def test_blocking_twice_is_409(self, test_client, blockee):
        _, user_name, _ = blockee
        first = test_client.make_request(
            "POST", "/admin/blockUser", json={"userName": user_name}
        )
        assert first.status_code == 200, first.text[:300]
        again = test_client.make_request(
            "POST", "/admin/blockUser", json={"userName": user_name}
        )
        assert again.status_code == 409, again.text[:300]

    def test_unblocking_an_active_login_is_409(self, test_client, blockee):
        _, user_name, _ = blockee
        response = test_client.make_request(
            "POST", "/admin/unblockUser", json={"userName": user_name}
        )
        assert response.status_code == 409, response.text[:300]

    def test_own_login_is_400(self, test_client):
        response = test_client.make_request(
            "POST", "/admin/blockUser", json={"userName": "admin"}
        )
        assert response.status_code == 400, response.text[:300]

    def test_unknown_login_is_404(self, test_client):
        response = test_client.make_request(
            "POST", "/admin/blockUser", json={"userName": "no-such-login-x"}
        )
        assert response.status_code == 404, response.text[:300]

    def test_without_block_user_grant_is_403(self, basic_user_client, blockee):
        _, user_name, _ = blockee
        response = basic_user_client.make_request(
            "POST", "/admin/blockUser", json={"userName": user_name}
        )
        assert response.status_code == 403, response.text[:300]

    def test_without_csrf_is_403(self, test_client, blockee):
        _, user_name, _ = blockee
        response = test_client.session.post(
            f"{_base_url()}/admin/blockUser", json={"userName": user_name}
        )
        assert response.status_code == 403, response.text[:300]
