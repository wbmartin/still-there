"""A Pi checks in with its device token (POST /deviceCheckin).

The admin creates a device and a band, issues the device's token (POST
/deviceToken, shown once), and the Pi posts its state with the token as a
Bearer header and no session. The check-in updates the band's presence
without moving any update_ts; a reissued token makes the old one 401; and
the device is not there at all for another tenant's admin.

Every check-in sends its own X-Forwarded-For address
(conftest.new_forwarded_ip): the route has only the per-address rate limit,
which would otherwise count against loopback. Rows are made in the test
tenant (2) and deleted after.
"""
import os
import uuid

import pytest
import requests

from conftest import (
    OTHER_TENANT,
    TEST_TENANT,
    TestClient,
    forwarded_for,
    new_forwarded_ip,
)

BAND_ADDRESS = "C3:00:00:98:F3:19"


def checkin(token, body=None, address=None, authorization=None):
    """POST /deviceCheckin as the Pi does: Bearer token, no cookie."""
    headers = forwarded_for(address or new_forwarded_ip())
    headers["Authorization"] = (authorization if authorization is not None
                                else f"Bearer {token}")
    return requests.post(
        f"{os.environ['TEST_BASE_URL']}/deviceCheckin",
        json=body if body is not None else pi_body(present=True),
        headers=headers,
        verify=False,
    )


def pi_body(present, rssi=-61):
    """A check-in as src/pi-agent/checkin.go sends it."""
    return {
        "agentVersion": "0.1.0",
        "sentAt": "2026-10-24T09:17:44Z",
        "uptimeSec": 120,
        "configVersion": 1,
        "reason": "interval",
        "beaconList": [{
            "address": BAND_ADDRESS.lower(),
            "label": "Band",
            "present": present,
            "lastSeen": "2026-10-24T09:17:40Z" if present else None,
            "rssi": rssi if present else None,
        }],
    }


def device_row(client, device_id):
    rows = client.make_request(
        "GET", "/deviceList", params={"deviceIdList": str(device_id)}
    ).json()["data"]
    assert len(rows) == 1, rows
    return rows[0]


def beacon_row(client, device_id):
    rows = client.make_request(
        "GET", "/beaconList", params={"deviceId": str(device_id)}
    ).json()["data"]
    assert len(rows) == 1, rows
    return rows[0]


def issue_token(client, device_id):
    row = device_row(client, device_id)
    response = client.make_request("POST", "/deviceToken", json={
        "deviceId": device_id, "updateTs": row["updateTs"]})
    assert response.status_code == 200, response.text
    return response.json()["data"]["deviceToken"]


@pytest.fixture
def device(test_client):
    """A device with one band, made as admin; deleted after."""
    response = test_client.make_request("POST", "/device", json={
        "deviceName": f"Checkin Pi {uuid.uuid4().hex[:8]}",
        "checkinIntervalSec": 60,
        "missedCheckinBeforeAlarm": 3,
        "absentAfterSec": 600,
    })
    assert response.status_code == 201, response.text
    created = response.json()["data"]
    response = test_client.make_request("POST", "/beacon", json={
        "deviceId": created["deviceId"], "address": BAND_ADDRESS,
        "label": "Band"})
    assert response.status_code == 201, response.text
    yield created

    row = device_row(test_client, created["deviceId"])
    test_client.make_request("DELETE", "/device", params={
        "deviceId": str(created["deviceId"]),
        "updateTs": f"{row['updateTs']:.7f}"})


class TestDeviceCheckin:
    def test_token_checkin_marks_the_band_present(self, test_client, device):
        token = issue_token(test_client, device["deviceId"])
        assert len(token) == 64
        before = device_row(test_client, device["deviceId"])
        band_before = beacon_row(test_client, device["deviceId"])
        assert "present" not in band_before, "no report yet"

        response = checkin(token)
        assert response.status_code == 200, response.text
        assert response.json() == {
            "data": {"configVersion": before["configVersion"]}}

        after = device_row(test_client, device["deviceId"])
        assert after["lastAgentVersion"] == "0.1.0"
        assert after["lastCheckinTs"] > 0
        assert after["lastCheckinJson"]["reason"] == "interval"
        assert after["updateTs"] == before["updateTs"], (
            "a check-in must not move the device's lock")
        assert "deviceToken" not in after and "tokenHash" not in after

        band = beacon_row(test_client, device["deviceId"])
        assert band["present"] is True
        assert band["lastRssi"] == -61
        assert band["lastSeenTs"] == 1792833460
        assert band["updateTs"] == band_before["updateTs"]

        response = checkin(token, body=pi_body(present=False))
        assert response.status_code == 200, response.text
        band = beacon_row(test_client, device["deviceId"])
        assert band["present"] is False
        assert band["lastSeenTs"] == 1792833460, "earlier sighting kept"

    def test_reissued_token_makes_the_old_one_401(self, test_client, device):
        old_token = issue_token(test_client, device["deviceId"])
        assert checkin(old_token).status_code == 200
        new_token = issue_token(test_client, device["deviceId"])
        assert new_token != old_token

        refused = checkin(old_token)
        assert refused.status_code == 401, refused.text
        assert refused.json() == {"error": "Invalid device token"}
        assert checkin(new_token).status_code == 200

    def test_every_bad_token_is_the_same_401(self, test_client, device):
        token = issue_token(test_client, device["deviceId"])
        assert checkin(token).status_code == 200, "the good token works"
        for authorization in ["", f"Basic {token}", "Bearer short",
                              f"Bearer {token.upper()}",
                              f"Bearer {uuid.uuid4().hex * 2}"]:
            response = checkin(None, authorization=authorization)
            assert response.status_code == 401, (authorization,
                                                 response.text)
            assert response.json() == {"error": "Invalid device token"}

    def test_a_session_cookie_is_not_a_device_token(self, test_client,
                                                     device):
        """The admin's own session reaches nothing on this route."""
        response = test_client.session.post(
            f"{test_client.base_url}/deviceCheckin", json=pi_body(True),
            headers=forwarded_for(new_forwarded_ip()))
        assert response.status_code == 401, response.text

    def test_another_tenants_admin_cannot_reach_the_device(
        self, test_client, device
    ):
        """The cross-tenant negative test at the API (doc/110). switcher is
        an admin in OTHER_TENANT (Admin2, test-seed.sql); it is only signed
        in and switched here, never edited."""
        device_id = device["deviceId"]
        row = device_row(test_client, device_id)
        band = beacon_row(test_client, device_id)

        client = TestClient(os.environ["TEST_BASE_URL"])
        client.authenticate("switcher", "basic123!")
        _switch(client, OTHER_TENANT)
        own = client.make_request("GET", "/deviceList")
        assert own.status_code == 200, own.text
        assert device_id not in [d["deviceId"] for d in own.json()["data"]]

        listed = client.make_request(
            "GET", "/deviceList", params={"deviceIdList": str(device_id)})
        assert listed.status_code == 200, listed.text
        assert listed.json()["data"] == []
        beacons = client.make_request(
            "GET", "/beaconList", params={"deviceId": str(device_id)})
        assert beacons.status_code == 200 and beacons.json()["data"] == []

        for method, path, kwargs in [
            ("PUT", "/device", {"json": {**row, "deviceName": "Hijack"}}),
            ("POST", "/deviceToken", {"json": {
                "deviceId": device_id, "updateTs": row["updateTs"]}}),
            ("DELETE", "/device", {"params": {
                "deviceId": str(device_id),
                "updateTs": f"{row['updateTs']:.7f}"}}),
            ("POST", "/beacon", {"json": {
                "deviceId": device_id, "address": "AA:BB:CC:DD:EE:01",
                "label": "x"}}),
            ("PUT", "/beacon", {"json": {**band, "label": "Hijack"}}),
            ("POST", "/contact", {"json": {
                "deviceId": device_id, "contactName": "x",
                "phoneE164": "+15551230000"}}),
        ]:
            response = client.make_request(method, path, **kwargs)
            assert response.status_code == 404, (path, response.text)

        after = device_row(test_client, device_id)
        assert after["deviceName"] == row["deviceName"]
        assert after["updateTs"] == row["updateTs"]
        assert beacon_row(test_client, device_id)["label"] == "Band"
        _switch(client, TEST_TENANT)


def _switch(client, security_context):
    """POST /changeSecurityContext and take the new cookie and token."""
    response = client.make_request(
        "POST", "/changeSecurityContext",
        json={"securityContext": security_context})
    assert response.status_code == 200, response.text
    cookie = response.headers.get("Set-Cookie", "")
    client.jwt_token = cookie.split("auth=")[1].split(";")[0]
    client.csrf_token = response.json()["csrfToken"]
    client.session.cookies.clear()
    client.session.cookies.set("auth", client.jwt_token, path="/")
