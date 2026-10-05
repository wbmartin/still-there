"""The TOTP second factor, end to end.

Every request that counts towards a lockout or a per-address rate limit
comes from the test's own new_forwarded_ip() address, and the lockout case
locks a throwaway user, so nothing here spends loopback's or a seeded user's
allowance. The zz prefix is history: the file no longer has to run last, and
the name is kept so references to it stay valid.

Fixture user `mfa` (password basic123!) is not MFA-exempt and is seeded
with no seed, so enrolment is part of the test and nothing depends on the
environment's TOTP_ENCRYPTION_KEY. Each case leaves it un-enrolled.
"""
import os
import time
import uuid

import pyotp
import pytest
import requests

from conftest import (
    TestClient,
    failed_login_threshold,
    forwarded_for,
    new_forwarded_ip,
    require_test_tenant,
)

MFA_USER = "mfa"
MFA_PASSWORD = "basic123!"
LOCKED_MESSAGE = (
    "Account is temporarily locked due to too many failed login attempts"
)


# The API accepts a code once: a step at or below the last one it accepted
# for the user is a replay. It accepts the step before now's, now's and the
# next, so fresh_code hands out the oldest step newer than the last one used
# for the secret, and sleeps to the next 30 s boundary only when all three are
# spent. The step before now's is skipped near a boundary, where it would fall
# out of the window before the request lands.
_last_step_by_secret: dict = {}


def fresh_code(secret: str) -> str:
    while True:
        now = time.time()
        current = int(now) // 30
        for step in (current - 1, current, current + 1):
            if step == current - 1 and now % 30 > 27:
                continue
            if step > _last_step_by_secret.get(secret, -1):
                _last_step_by_secret[secret] = step
                return pyotp.TOTP(secret).at(step * 30)
        time.sleep(30 - now % 30 + 0.5)


def wrong_code(secret: str) -> str:
    """A well-formed code from an hour ago, outside the accepted window."""
    now_step = int(time.time()) // 30
    window = {pyotp.TOTP(secret).at((now_step + delta) * 30)
              for delta in (-1, 0, 1, 2)}
    step = now_step - 120
    while pyotp.TOTP(secret).at(step * 30) in window:
        step -= 1
    return pyotp.TOTP(secret).at(step * 30)


def _base_url() -> str:
    return os.environ["TEST_BASE_URL"]


def _post(session, path: str, client_ip: str, body: dict) -> requests.Response:
    return session.post(
        f"{_base_url()}{path}", json=body, headers=forwarded_for(client_ip)
    )


def _password_step(
    session: requests.Session,
    enrolled: bool,
    client_ip: str,
    user_name: str = MFA_USER,
    password: str = MFA_PASSWORD,
) -> str:
    """POST /auth for a non-exempt user: no session yet, a temp token instead.

    mfaEnrolled tells the web app to verify a code (True) or enrol (False).
    """
    response = _post(
        session, "/auth", client_ip,
        {"userName": user_name, "password": password},
    )
    assert response.status_code == 200, response.text
    user = response.json()["user"]
    assert user["tempToken"], "non-exempt user must get a temp token"
    assert user["grantList"] == [], "no grants before the second factor"
    assert "auth" not in session.cookies, "no session cookie before the second factor"
    assert user["mfaEnrolled"] is enrolled, user
    return user["tempToken"]


def _reset_mfa(admin: TestClient, app_user_id: int) -> None:
    response = admin.make_request(
        "POST", "/admin/resetUserMFA", json={"userId": app_user_id}
    )
    assert response.status_code == 200, response.text


class TestMFATOTP:
    def test_enrol_verify_and_sign_in_again(self):
        admin = TestClient(_base_url())
        admin.authenticate()
        client_ip = new_forwarded_ip()

        # --- Enrolment: password → temp token → setup → verify ---
        session = requests.Session()
        temp_token = _password_step(session, False, client_ip)

        response = _post(
            session, "/mfaSetup", client_ip, {"tempToken": temp_token}
        )
        assert response.status_code == 200, response.text
        setup = response.json()["data"]
        secret = setup["secret"]
        assert setup["qrCodeImage"]

        # A wrong code is refused and leaves the temp token usable.
        response = _post(session, "/mfaVerify", client_ip, {
            "tempToken": temp_token, "totpCode": wrong_code(secret),
        })
        assert response.status_code == 401

        response = _post(session, "/mfaVerify", client_ip, {
            "tempToken": temp_token, "totpCode": fresh_code(secret),
        })
        assert response.status_code == 200, response.text
        body = response.json()
        require_test_tenant(body["user"])
        assert body["csrfToken"]
        assert "auth" in session.cookies, "session cookie issued by /mfaVerify"
        assert body["user"]["userName"] == MFA_USER
        assert 1 in body["user"]["grantList"]  # LOGON
        app_user_id = body["user"]["appUserId"]

        # The session works: an authenticated call succeeds. The cookie is
        # Secure, so over plain http it is sent by hand (as conftest does).
        response = requests.get(
            f"{_base_url()}/me", cookies={"auth": session.cookies["auth"]}
        )
        assert response.status_code == 200
        assert response.json()["data"]["userName"] == MFA_USER
        # The enrolment is stored. /me carries only the own profile (B062),
        # so the flag is read from the admin list.
        response = admin.make_request(
            "GET", "/userList", params={"appUserIdList": str(app_user_id)}
        )
        assert response.status_code == 200, response.text
        assert response.json()["data"][0]["mfaSetup"] is True

        # The temp token is single-use: replaying it with a fresh code fails.
        response = _post(requests, "/mfaVerify", client_ip, {
            "tempToken": temp_token, "totpCode": fresh_code(secret),
        })
        assert response.status_code == 401

        # --- Second sign-in: enrolled user, same secret ---
        # /mfaSetup refuses an enrolled user: the stored secret and its QR
        # code are never shown again to a holder of the password alone.
        session2 = requests.Session()
        temp_token2 = _password_step(session2, True, client_ip)
        response = _post(
            session2, "/mfaSetup", client_ip, {"tempToken": temp_token2}
        )
        assert response.status_code == 409, response.text
        assert response.json()["error"], response.text  # a body was read
        assert secret not in response.text
        assert "qrCodeImage" not in response.text
        response = _post(session2, "/mfaVerify", client_ip, {
            "tempToken": temp_token2, "totpCode": fresh_code(secret),
        })
        assert response.status_code == 200, response.text

        # --- Admin reset: the old secret no longer works ---
        _reset_mfa(admin, app_user_id)
        session3 = requests.Session()
        temp_token3 = _password_step(session3, False, client_ip)
        response = _post(
            session3, "/mfaSetup", client_ip, {"tempToken": temp_token3}
        )
        assert response.status_code == 200
        assert response.json()["data"]["secret"] != secret

    def test_totp_code_replay_is_refused(self):
        """A code that signed in once cannot sign in again.

        Two temp tokens, one code: the second /mfaVerify gets the wrong-code
        answer because its step was already accepted.
        """
        admin = TestClient(_base_url())
        admin.authenticate()
        client_ip = new_forwarded_ip()

        session = requests.Session()
        temp_token = _password_step(session, False, client_ip)
        response = _post(
            session, "/mfaSetup", client_ip, {"tempToken": temp_token}
        )
        assert response.status_code == 200, response.text
        secret = response.json()["data"]["secret"]
        code = fresh_code(secret)
        response = _post(session, "/mfaVerify", client_ip, {
            "tempToken": temp_token, "totpCode": code,
        })
        assert response.status_code == 200, response.text
        app_user_id = response.json()["user"]["appUserId"]

        session2 = requests.Session()
        temp_token2 = _password_step(session2, True, client_ip)
        response = _post(session2, "/mfaVerify", client_ip, {
            "tempToken": temp_token2, "totpCode": code,
        })
        assert response.status_code == 401, response.text
        assert response.json()["error"] == "Invalid TOTP code"
        assert "auth" not in session2.cookies

        # Un-enrol, which also forgets the spent step, for the next test.
        _reset_mfa(admin, app_user_id)

    def test_reset_needs_reset_mfa_grant_not_update_user(self):
        """Editing a user (9003) does not let you strip their second factor.

        A throwaway group holding READ_USER and UPDATE_USER but not
        RESET_MFA_TOKEN (9020), and an exempt user in it, made here and
        removed at the end.
        """
        admin = TestClient(_base_url())
        admin.authenticate()
        suffix = uuid.uuid4().hex[:12]
        password = "noreset123!"
        response = admin.make_request("POST", "/appUserGroup", json={
            "groupName": f"Editors {suffix}",
            "userGrantList": [1, 9002, 9003],
        })
        assert response.status_code == 201, response.text
        group = response.json()["data"]
        response = admin.make_request("POST", "/user", json={
            "userName": f"editor{suffix}",
            "firstName": "Editor",
            "lastName": "NoReset",
            "email": f"editor{suffix}@example.com",
            "password": password,
            "mfaExempt": True,
            "appUserGroupMembershipList": [group["appUserGroupId"]],
        })
        assert response.status_code == 201, response.text
        editor_row = response.json()["data"]
        try:
            editor = TestClient(_base_url())
            body = editor.authenticate(editor_row["userName"], password)
            assert 9003 in body["user"]["grantList"]
            assert 9020 not in body["user"]["grantList"]
            response = editor.make_request(
                "POST", "/admin/resetUserMFA",
                json={"userId": editor_row["appUserId"]},
            )
            assert response.status_code == 403, response.text
            assert "RESET_MFA_TOKEN" in response.text
        finally:
            admin.make_request("DELETE", "/user", params={
                "appUserId": str(editor_row["appUserId"]),
                "updateTs": f"{editor_row['updateTs']:.7f}",
            })
            admin.make_request("DELETE", "/appUserGroup", params={
                "appUserGroupId": str(group["appUserGroupId"]),
                "updateTs": f"{group['updateTs']:.7f}",
            })

    def test_mfa_lockout_after_failed_attempts(self):
        """Wrong codes count towards the login lockout and lock the user.

        A throwaway user, enrolled here from its own address and deleted at
        the end: neither loopback nor the seeded mfa user is locked, so a
        re-run, or any later test, signs in as usual.
        """
        admin = TestClient(_base_url())
        admin.authenticate()
        threshold = failed_login_threshold()
        client_ip = new_forwarded_ip()
        user_name = f"mfalock{uuid.uuid4().hex[:12]}"
        password = "mfalock123!"
        response = admin.make_request("POST", "/user", json={
            "userName": user_name,
            "firstName": "Mfa",
            "lastName": "Lockout",
            "email": f"{user_name}@example.com",
            "password": password,
            "mfaExempt": False,
            "appUserGroupMembershipList": [],
        })
        assert response.status_code == 201, response.text
        app_user_id = response.json()["data"]["appUserId"]
        try:
            session = requests.Session()
            temp_token = _password_step(
                session, False, client_ip, user_name, password
            )
            response = _post(
                session, "/mfaSetup", client_ip, {"tempToken": temp_token}
            )
            assert response.status_code == 200, response.text
            secret = response.json()["data"]["secret"]
            response = _post(session, "/mfaVerify", client_ip, {
                "tempToken": temp_token, "totpCode": fresh_code(secret),
            })
            assert response.status_code == 200, response.text

            session2 = requests.Session()
            temp_token2 = _password_step(
                session2, True, client_ip, user_name, password
            )
            for attempt in range(threshold):
                response = _post(session2, "/mfaVerify", client_ip, {
                    "tempToken": temp_token2, "totpCode": wrong_code(secret),
                })
                assert response.status_code == 401, (attempt, response.text)
                assert response.json()["error"] == "Invalid TOTP code"

            # Locked: even a right code is refused, with the lockout answer
            # rather than the rate limiter's (both are 429).
            response = _post(session2, "/mfaVerify", client_ip, {
                "tempToken": temp_token2, "totpCode": fresh_code(secret),
            })
            assert response.status_code == 429, response.text
            assert response.json()["error"] == LOCKED_MESSAGE
            assert "auth" not in session2.cookies

            # The name is locked from any address, password step included.
            response = _post(requests, "/auth", new_forwarded_ip(), {
                "userName": user_name, "password": password,
            })
            assert response.status_code == 429, response.text
            assert response.json()["error"] == LOCKED_MESSAGE
        finally:
            response = admin.make_request("GET", "/userList")
            assert response.status_code == 200, response.text
            row = next(
                row for row in response.json()["data"]
                if row["appUserId"] == app_user_id
            )
            response = admin.make_request("DELETE", "/user", params={
                "appUserId": str(app_user_id),
                "updateTs": f"{row['updateTs']:.7f}",
            })
            assert response.status_code == 200, response.text
