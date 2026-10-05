"""Invite a user with a one-time link over HTTP (B035).

The admin creates an invited login (POST /userInvite), the invited person
reads and redeems the link (POST /inviteCheck, POST /inviteRedeem, no
session), and then signs in with the password they chose. Self-registration
(POST /userSelfCreate) stays beside it; a derived app keeps the paths it
wants.

Every anonymous call here sends its own X-Forwarded-For address
(conftest.new_forwarded_ip): a dead link counts toward the same per-address
lockout /auth reads, so without it the refusals below would count against
loopback and could lock the rest of the run out. Every login here is in the
test tenant (2) and is deleted after.
"""
import os
import uuid

import pytest
import requests

from conftest import forwarded_for, new_forwarded_ip


def _base_url() -> str:
    return os.environ["TEST_BASE_URL"]


def anonymous_post(path: str, body: dict, address: str) -> requests.Response:
    """POST with no session, as the invited person's browser would."""
    return requests.post(
        f"{_base_url()}{path}",
        json=body,
        headers=forwarded_for(address),
        verify=False,
    )


def _user_row(test_client, app_user_id: int) -> dict:
    rows = test_client.make_request(
        "GET", "/userList", params={"appUserIdList": str(app_user_id)}
    ).json()["data"]
    assert rows, f"no row for appUserId {app_user_id}"
    return rows[0]


@pytest.fixture
def invited(test_client):
    """Invite a fresh login as admin; delete it after."""
    suffix = uuid.uuid4().hex[:12]
    body = {
        "userName": f"invitee{suffix}",
        "firstName": "Ivy",
        "lastName": "Vine",
        "email": f"invitee{suffix}@example.com",
        "appUserGroupMembershipList": [1],  # Basic User: carries LOGON
        "mfaExempt": True,
    }
    response = test_client.make_request("POST", "/userInvite", json=body)
    assert response.status_code == 201, response.text[:300]
    data = response.json()["data"]
    yield {"body": body, **data}

    app_user_id = data["user"]["appUserId"]
    row = _user_row(test_client, app_user_id)
    test_client.make_request("DELETE", "/user", params={
        "appUserId": str(app_user_id),
        "updateTs": f"{row['updateTs']:.7f}",
    })


class TestUserInvite:
    def test_invite_issues_a_token_and_an_invited_login(
        self, test_client, invited
    ):
        assert len(invited["inviteToken"]) == 64
        assert invited["user"]["accountStatus"] == "invited"
        assert invited["user"]["securityContext"] == 2
        assert invited["expiresTs"] > 0
        row = _user_row(test_client, invited["user"]["appUserId"])
        assert row["accountStatus"] == "invited"

    def test_an_invited_login_cannot_sign_in(self, invited):
        response = anonymous_post("/auth", {
            "userName": invited["body"]["userName"], "password": "anything1!",
        }, new_forwarded_ip())
        assert response.status_code == 401
        assert "Invalid credentials" in response.text

    def test_check_redeem_then_sign_in(self, test_client, invited):
        token, address = invited["inviteToken"], new_forwarded_ip()

        check = anonymous_post("/inviteCheck", {"token": token}, address)
        assert check.status_code == 200, check.text[:300]
        assert check.json()["data"]["userName"] == invited["body"]["userName"]
        assert check.json()["data"]["firstName"] == "Ivy"

        short = anonymous_post("/inviteRedeem",
                               {"token": token, "password": "short"}, address)
        assert short.status_code == 400

        redeem = anonymous_post("/inviteRedeem", {
            "token": token, "password": "chosen-by-ivy-1!",
        }, address)
        assert redeem.status_code == 200, redeem.text[:300]
        assert redeem.json()["data"]["userName"] == invited["body"]["userName"]

        again = anonymous_post("/inviteRedeem", {
            "token": token, "password": "someone-else-1!",
        }, address)
        assert again.status_code == 404, "a link is single use"

        login = anonymous_post("/auth", {
            "userName": invited["body"]["userName"],
            "password": "chosen-by-ivy-1!",
        }, address)
        assert login.status_code == 200, login.text[:300]

        row = _user_row(test_client, invited["user"]["appUserId"])
        assert row["accountStatus"] == "active"

    def test_reissue_and_revoke(self, test_client, invited):
        old_token = invited["inviteToken"]
        target = {"appUserId": invited["user"]["appUserId"]}
        address = new_forwarded_ip()

        reissue = test_client.make_request(
            "POST", "/userInviteReissue", json=target)
        assert reissue.status_code == 200, reissue.text[:300]
        new_token = reissue.json()["data"]["inviteToken"]
        assert new_token != old_token

        assert anonymous_post("/inviteCheck", {"token": old_token},
                              address).status_code == 404
        assert anonymous_post("/inviteCheck", {"token": new_token},
                              address).status_code == 200

        revoke = test_client.make_request(
            "POST", "/userInviteRevoke", json=target)
        assert revoke.status_code == 200, revoke.text[:300]
        assert anonymous_post("/inviteCheck", {"token": new_token},
                              address).status_code == 404
        again = test_client.make_request(
            "POST", "/userInviteRevoke", json=target)
        assert again.status_code == 409

    def test_every_dead_link_gets_the_same_answer(self):
        address = new_forwarded_ip()
        malformed = anonymous_post("/inviteCheck", {"token": "nope"}, address)
        unknown = anonymous_post("/inviteCheck", {"token": "0" * 64}, address)
        assert malformed.status_code == unknown.status_code == 404
        assert malformed.json() == unknown.json()

    def test_admin_cannot_set_an_invited_logins_password(
        self, test_client, invited
    ):
        response = test_client.make_request(
            "POST", "/changeOtherUserPassword", json={
                "appUserId": invited["user"]["appUserId"],
                "newPassword": "admin-chosen-1!",
            })
        assert response.status_code == 409, response.text[:300]

    def test_inviting_needs_create_user(self, basic_user_client):
        response = basic_user_client.make_request("POST", "/userInvite", json={
            "userName": f"nope{uuid.uuid4().hex[:8]}", "firstName": "N",
            "lastName": "O", "email": "nope@example.com",
            "appUserGroupMembershipList": [], "mfaExempt": True,
        })
        assert response.status_code == 403

    def test_inviting_without_csrf_is_403(self, test_client):
        response = test_client.session.post(
            f"{_base_url()}/userInvite", json={
                "userName": f"nope{uuid.uuid4().hex[:8]}", "firstName": "N",
                "lastName": "O", "email": "nope@example.com",
                "appUserGroupMembershipList": [], "mfaExempt": True,
            })
        assert response.status_code == 403, response.text[:300]
