"""Server-side session revocation: a token stops working when its session
ends, not only when it expires.

Each test keeps a copy of a signed-in cookie, makes the change that must
end it, and replays the copy against GET /me.
"""
import os
import uuid

import requests

from conftest import TestClient

GRANT_LOGON = 1
GRANT_CHANGE_OWN_PASSWORD = 8001


def _base_url() -> str:
    return os.environ["TEST_BASE_URL"]


def _me_with(token: str) -> int:
    """GET /me with only this auth cookie; returns the status."""
    response = requests.get(
        f"{_base_url()}/me", cookies={"auth": token}, verify=False
    )
    return response.status_code


def _sign_in(user_name: str, password: str) -> TestClient:
    client = TestClient(_base_url())
    client.authenticate(user_name, password)
    assert client.jwt_token, "sign-in set no auth cookie"
    return client


def _create_user(admin: TestClient, password: str, group_list) -> dict:
    suffix = uuid.uuid4().hex[:12]
    response = admin.make_request("POST", "/user", json={
        "userName": f"revoke{suffix}",
        "firstName": "Revoke",
        "lastName": "Test",
        "email": f"revoke{suffix}@example.com",
        "password": password,
        "mfaExempt": True,
        "appUserGroupMembershipList": group_list,
    })
    assert response.status_code == 201, response.text
    return response.json()["data"]


def _delete_user(admin: TestClient, app_user_id: int) -> None:
    listed = admin.make_request(
        "GET", "/userList", params={"appUserIdList": app_user_id}
    )
    assert listed.status_code == 200, listed.text
    update_ts = listed.json()["data"][0]["updateTs"]
    response = admin.make_request("DELETE", "/user", params={
        "appUserId": str(app_user_id), "updateTs": f"{update_ts:.7f}",
    })
    assert response.status_code == 200, response.text


class TestSessionRevocation:
    def test_logoff_ends_the_session(self):
        client = _sign_in("basic", "basic123!")
        token = client.jwt_token
        assert _me_with(token) == 200

        response = client.make_request("POST", "/logoff")
        assert response.status_code == 200, response.text

        assert _me_with(token) == 401, "a copy of the old cookie still works"

    def test_admin_password_change_ends_the_users_sessions(self):
        admin = _sign_in("admin", "admin123!")
        user = _create_user(admin, "first-pass-1", [1])
        try:
            victim = _sign_in(user["userName"], "first-pass-1")
            token = victim.jwt_token
            assert _me_with(token) == 200

            response = admin.make_request(
                "POST", "/changeOtherUserPassword",
                json={"appUserId": user["appUserId"],
                      "newPassword": "second-pass-2"},
            )
            assert response.status_code == 200, response.text

            assert _me_with(token) == 401
            assert _me_with(admin.jwt_token) == 200, "the admin is untouched"
            fresh = _sign_in(user["userName"], "second-pass-2")
            assert _me_with(fresh.jwt_token) == 200, "a new sign-in works"
        finally:
            _delete_user(admin, user["appUserId"])

    def test_group_grant_edit_ends_the_members_sessions(self):
        admin = _sign_in("admin", "admin123!")
        response = admin.make_request("POST", "/appUserGroup", json={
            "groupName": f"Revoke {uuid.uuid4().hex[:12]}",
            "userGrantList": [GRANT_LOGON],
        })
        assert response.status_code == 201, response.text
        group = response.json()["data"]
        user = _create_user(admin, "member-pass-1", [group["appUserGroupId"]])
        try:
            member = _sign_in(user["userName"], "member-pass-1")
            token = member.jwt_token
            assert _me_with(token) == 200

            group["userGrantList"] = [GRANT_LOGON, GRANT_CHANGE_OWN_PASSWORD]
            response = admin.make_request("PUT", "/appUserGroup", json=group)
            assert response.status_code == 200, response.text
            group = response.json()["data"]

            assert _me_with(token) == 401, "the member kept the old grants"
            assert _me_with(admin.jwt_token) == 200, "a non-member is untouched"
        finally:
            _delete_user(admin, user["appUserId"])
            response = admin.make_request("DELETE", "/appUserGroup", params={
                "appUserGroupId": str(group["appUserGroupId"]),
                "updateTs": f"{group['updateTs']:.7f}",
            })
            assert response.status_code == 200, response.text
