"""The TOTP second factor, end to end.

Runs last (zz): the lockout case spends the shared IP's failed-attempt
allowance, which would lock every later test out of /auth.

Fixture user `mfa` (password basic123!) is not MFA-exempt and is seeded
with no seed, so enrolment is part of the test and nothing depends on the
environment's TOTP_ENCRYPTION_KEY. The admin session that resets the user
at the end is taken BEFORE the lockout, because the lockout also locks the
IP out of /auth (resetUserMFA itself needs only a session + CSRF).
"""
import os

import pyotp
import pytest
import requests

from conftest import TestClient

MFA_USER = "mfa"
MFA_PASSWORD = "basic123!"


def _base_url() -> str:
    return os.environ["TEST_BASE_URL"]


def _password_step(session: requests.Session, enrolled: bool) -> str:
    """POST /auth for the mfa user: no session yet, a temp token instead.

    mfaEnrolled tells the web app to verify a code (True) or enrol (False).
    """
    response = session.post(
        f"{_base_url()}/auth", json={"userName": MFA_USER, "password": MFA_PASSWORD}
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

        # --- Enrolment: password → temp token → setup → verify ---
        session = requests.Session()
        temp_token = _password_step(session, enrolled=False)

        response = session.post(f"{_base_url()}/mfaSetup", json={"tempToken": temp_token})
        assert response.status_code == 200, response.text
        setup = response.json()["data"]
        secret = setup["secret"]
        assert setup["qrCodeImage"]

        # A wrong code is refused and leaves the temp token usable.
        response = session.post(
            f"{_base_url()}/mfaVerify", json={"tempToken": temp_token, "totpCode": "000000"}
        )
        assert response.status_code == 401

        code = pyotp.TOTP(secret).now()
        response = session.post(
            f"{_base_url()}/mfaVerify", json={"tempToken": temp_token, "totpCode": code}
        )
        assert response.status_code == 200, response.text
        body = response.json()
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
        assert response.json()["data"]["mfaSetup"] is True

        # The temp token is single-use: replaying it with a fresh code fails.
        response = requests.post(
            f"{_base_url()}/mfaVerify",
            json={"tempToken": temp_token, "totpCode": pyotp.TOTP(secret).now()},
        )
        assert response.status_code == 401

        # --- Second sign-in: enrolled user, same secret ---
        # /mfaSetup refuses an enrolled user: the stored secret and its QR
        # code are never shown again to a holder of the password alone.
        session2 = requests.Session()
        temp_token2 = _password_step(session2, enrolled=True)
        response = session2.post(f"{_base_url()}/mfaSetup", json={"tempToken": temp_token2})
        assert response.status_code == 409, response.text
        assert secret not in response.text
        assert "qrCodeImage" not in response.text
        response = session2.post(
            f"{_base_url()}/mfaVerify",
            json={"tempToken": temp_token2, "totpCode": pyotp.TOTP(secret).now()},
        )
        assert response.status_code == 200, response.text

        # --- Admin reset: the old secret no longer works ---
        _reset_mfa(admin, app_user_id)
        session3 = requests.Session()
        temp_token3 = _password_step(session3, enrolled=False)
        response = session3.post(f"{_base_url()}/mfaSetup", json={"tempToken": temp_token3})
        assert response.status_code == 200
        assert response.json()["data"]["secret"] != secret

    def test_mfa_lockout_after_failed_attempts(self):
        """MFA failures count towards the login lockout and lock the user."""
        admin = TestClient(_base_url())
        admin.authenticate()
        threshold = int(os.environ.get("FAILED_LOGIN_ATTEMPT_THRESHOLD", "7"))

        session = requests.Session()
        temp_token = _password_step(session, enrolled=False)
        response = session.post(f"{_base_url()}/mfaSetup", json={"tempToken": temp_token})
        assert response.status_code == 200
        app_user_id = None

        statuses = []
        for _ in range(threshold + 1):
            response = session.post(
                f"{_base_url()}/mfaVerify",
                json={"tempToken": temp_token, "totpCode": "000000"},
            )
            statuses.append(response.status_code)
        assert statuses[0] == 401
        assert 429 in statuses, statuses
        assert statuses[-1] == 429

        # Locked: even the password step is refused now.
        response = requests.post(
            f"{_base_url()}/auth", json={"userName": MFA_USER, "password": MFA_PASSWORD}
        )
        assert response.status_code == 429

        # Leave the user un-enrolled for the next run (the seed is what a
        # later test would otherwise inherit); the lock expires on its own.
        response = admin.make_request("GET", "/userList")
        assert response.status_code == 200
        for row in response.json()["data"]:
            if row["userName"] == MFA_USER:
                app_user_id = row["appUserId"]
        assert app_user_id is not None
        _reset_mfa(admin, app_user_id)
