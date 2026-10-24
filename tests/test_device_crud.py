"""Device monitoring CRUD over HTTP: devices, their bands (beacon), the
people texted about them (contact), the tenant's reminder settings, and
the alert list.

All as the admin of the test tenant (2); every device made here is deleted
after, its bands and contacts with it. Check-ins and cross-tenant reads
are in test_device_checkin.py.
"""
import uuid

import pytest

NOTIFY_EVENT_LIST = ["raised", "reminder", "acknowledged", "turned_off",
                     "reactivated", "daily_reminder"]


def _device_row(client, device_id):
    rows = client.make_request(
        "GET", "/deviceList", params={"deviceIdList": str(device_id)}
    ).json()["data"]
    assert len(rows) == 1, rows
    return rows[0]


@pytest.fixture
def device(test_client):
    response = test_client.make_request("POST", "/device", json={
        "deviceName": f"Crud Pi {uuid.uuid4().hex[:8]}",
        "checkinIntervalSec": 60,
        "missedCheckinBeforeAlarm": 3,
        "absentAfterSec": 600,
        "securityContext": 3,
        "configVersion": 99,
        "lastAgentVersion": "forged",
    })
    assert response.status_code == 201, response.text
    created = response.json()["data"]
    yield created

    row = _device_row(test_client, created["deviceId"])
    test_client.make_request("DELETE", "/device", params={
        "deviceId": str(created["deviceId"]),
        "updateTs": f"{row['updateTs']:.7f}"})


class TestDevice:
    def test_create_takes_server_values(self, device):
        assert device["securityContext"] == 2
        assert device["configVersion"] == 1
        assert "lastAgentVersion" not in device
        assert "tokenIssuedTs" not in device

    def test_bad_settings_are_400(self, test_client):
        for body in [
            {"deviceName": "", "checkinIntervalSec": 60,
             "missedCheckinBeforeAlarm": 3, "absentAfterSec": 600},
            {"deviceName": "Pi", "checkinIntervalSec": 9,
             "missedCheckinBeforeAlarm": 3, "absentAfterSec": 600},
            {"deviceName": "Pi", "checkinIntervalSec": 60,
             "missedCheckinBeforeAlarm": 0, "absentAfterSec": 600},
            {"deviceName": "Pi", "checkinIntervalSec": 60,
             "missedCheckinBeforeAlarm": 3, "absentAfterSec": 9},
        ]:
            response = test_client.make_request("POST", "/device", json=body)
            assert response.status_code == 400, (body, response.text)

    def test_settings_change_bumps_config_version(self, test_client, device):
        renamed = test_client.make_request("PUT", "/device", json={
            **device, "deviceName": device["deviceName"] + " renamed"})
        assert renamed.status_code == 200, renamed.text
        assert renamed.json()["data"]["configVersion"] == 1

        changed = test_client.make_request("PUT", "/device", json={
            **renamed.json()["data"], "checkinIntervalSec": 30})
        assert changed.status_code == 200, changed.text
        assert changed.json()["data"]["configVersion"] == 2
        assert changed.json()["data"]["checkinIntervalSec"] == 30

        stale = test_client.make_request("PUT", "/device", json={
            **device, "checkinIntervalSec": 45})
        assert stale.status_code == 409, stale.text
        assert _device_row(test_client,
                           device["deviceId"])["checkinIntervalSec"] == 30

    def test_delete_hides_the_device_and_its_children(self, test_client):
        created = test_client.make_request("POST", "/device", json={
            "deviceName": "Doomed Pi", "checkinIntervalSec": 60,
            "missedCheckinBeforeAlarm": 3, "absentAfterSec": 600,
        }).json()["data"]
        device_id = created["deviceId"]
        band = test_client.make_request("POST", "/beacon", json={
            "deviceId": device_id, "address": "AA:BB:CC:DD:EE:10",
            "label": "Band"})
        assert band.status_code == 201, band.text
        row = _device_row(test_client, device_id)

        stale = test_client.make_request("DELETE", "/device", params={
            "deviceId": str(device_id), "updateTs": "1"})
        assert stale.status_code == 409, stale.text
        response = test_client.make_request("DELETE", "/device", params={
            "deviceId": str(device_id),
            "updateTs": f"{row['updateTs']:.7f}"})
        assert response.status_code == 200, response.text

        assert test_client.make_request(
            "GET", "/deviceList", params={"deviceIdList": str(device_id)}
        ).json()["data"] == []
        assert test_client.make_request(
            "GET", "/beaconList", params={"deviceId": str(device_id)}
        ).json()["data"] == []
        again = test_client.make_request("DELETE", "/device", params={
            "deviceId": str(device_id),
            "updateTs": f"{row['updateTs']:.7f}"})
        assert again.status_code == 404, again.text

    def test_a_basic_user_has_no_device_grants(self, basic_user_client):
        for path in ["/deviceList", "/beaconList", "/contactList",
                     "/alertList", "/tenantSetting"]:
            response = basic_user_client.make_request("GET", path)
            assert response.status_code == 403, (path, response.text)


class TestBeacon:
    def test_create_normalises_and_bumps_config(self, test_client, device):
        response = test_client.make_request("POST", "/beacon", json={
            "deviceId": device["deviceId"], "address": " c3:00:00:98:f3:19",
            "label": "Wrist band", "present": True})
        assert response.status_code == 201, response.text
        band = response.json()["data"]
        assert band["address"] == "C3:00:00:98:F3:19"
        assert "present" not in band, "presence is the Pi's to report"
        assert _device_row(test_client,
                           device["deviceId"])["configVersion"] == 2

        duplicate = test_client.make_request("POST", "/beacon", json={
            "deviceId": device["deviceId"], "address": "C3:00:00:98:F3:19",
            "label": "Again"})
        assert duplicate.status_code == 409, duplicate.text

        updated = test_client.make_request("PUT", "/beacon", json={
            **band, "label": "Left wrist"})
        assert updated.status_code == 200, updated.text
        assert updated.json()["data"]["label"] == "Left wrist"
        assert _device_row(test_client,
                           device["deviceId"])["configVersion"] == 3

        stale = test_client.make_request("PUT", "/beacon", json={
            **band, "label": "Stale"})
        assert stale.status_code == 409, stale.text

        deleted = test_client.make_request("DELETE", "/beacon", params={
            "beaconId": str(band["beaconId"]),
            "updateTs": f"{updated.json()['data']['updateTs']:.7f}"})
        assert deleted.status_code == 200, deleted.text
        assert _device_row(test_client,
                           device["deviceId"])["configVersion"] == 4

    def test_bad_beacons_are_refused(self, test_client, device):
        for body, status in [
            ({"deviceId": device["deviceId"], "address": "C3-00-00-98-F3-19",
              "label": "x"}, 400),
            ({"deviceId": device["deviceId"], "address": "C3:00:00:98:F3:19",
              "label": ""}, 400),
            ({"address": "C3:00:00:98:F3:19", "label": "x"}, 400),
            ({"deviceId": 999999999, "address": "C3:00:00:98:F3:19",
              "label": "x"}, 404),
        ]:
            response = test_client.make_request("POST", "/beacon", json=body)
            assert response.status_code == status, (body, response.text)


class TestContact:
    def test_create_is_invited_with_every_event(self, test_client, device):
        response = test_client.make_request("POST", "/contact", json={
            "deviceId": device["deviceId"], "contactName": "Sam",
            "phoneE164": "+15551230001", "contactStatus": "active",
            "optedInTs": 5, "inviteSentTs": 5})
        assert response.status_code == 201, response.text
        contact = response.json()["data"]
        assert contact["contactStatus"] == "invited"
        assert contact["notifyEventList"] == NOTIFY_EVENT_LIST
        assert "inviteSentTs" not in contact
        assert "optedInTs" not in contact

        listed = test_client.make_request("GET", "/contactList", params={
            "deviceId": str(device["deviceId"])}).json()["data"]
        assert [c["contactId"] for c in listed] == [contact["contactId"]]

        updated = test_client.make_request("PUT", "/contact", json={
            **contact, "phoneE164": "+15551230002",
            "notifyEventList": ["raised"], "contactStatus": "active"})
        assert updated.status_code == 200, updated.text
        assert updated.json()["data"]["phoneE164"] == "+15551230002"
        assert updated.json()["data"]["notifyEventList"] == ["raised"]
        assert updated.json()["data"]["contactStatus"] == "invited"

        duplicate = test_client.make_request("POST", "/contact", json={
            "deviceId": device["deviceId"], "contactName": "Twin",
            "phoneE164": "+15551230002"})
        assert duplicate.status_code == 409, duplicate.text

        deleted = test_client.make_request("DELETE", "/contact", params={
            "contactId": str(contact["contactId"]),
            "updateTs": f"{updated.json()['data']['updateTs']:.7f}"})
        assert deleted.status_code == 200, deleted.text

    def test_bad_contacts_are_400(self, test_client, device):
        base = {"deviceId": device["deviceId"], "contactName": "Sam",
                "phoneE164": "+15551230001"}
        for change in [{"phoneE164": "5551230001"},
                       {"phoneE164": "+1 555 123 0001"},
                       {"contactName": " "},
                       {"notifyEventList": ["raised", "carrier_pigeon"]}]:
            response = test_client.make_request(
                "POST", "/contact", json={**base, **change})
            assert response.status_code == 400, (change, response.text)


class TestTenantSetting:
    def test_save_and_read(self, test_client):
        current = test_client.make_request("GET", "/tenantSetting")
        assert current.status_code in (200, 404), current.text
        update_ts = (current.json()["data"]["updateTs"]
                     if current.status_code == 200 else 0)
        body = {"timeZone": "America/New_York", "remindIntervalMin": 15,
                "dailyReminderTime": "09:00", "securityContext": 3}

        for change in [{"timeZone": "Mars/Base"}, {"timeZone": ""},
                       {"remindIntervalMin": 0},
                       {"remindIntervalMin": 1441},
                       {"dailyReminderTime": "9:00"},
                       {"dailyReminderTime": "24:00"}]:
            response = test_client.make_request(
                "PUT", "/tenantSetting",
                json={**body, **change, "updateTs": update_ts})
            assert response.status_code == 400, (change, response.text)

        saved = test_client.make_request(
            "PUT", "/tenantSetting", json={**body, "updateTs": update_ts})
        assert saved.status_code == 200, saved.text
        assert saved.json()["data"]["securityContext"] == 2
        assert saved.json()["data"]["remindIntervalMin"] == 15

        stale = test_client.make_request(
            "PUT", "/tenantSetting", json={**body, "updateTs": update_ts})
        assert stale.status_code == 409, stale.text

        read = test_client.make_request("GET", "/tenantSetting")
        assert read.status_code == 200, read.text
        assert read.json()["data"] == saved.json()["data"]


class TestAlert:
    def test_list_and_unknown_alert(self, test_client, device):
        listed = test_client.make_request("GET", "/alertList", params={
            "deviceId": str(device["deviceId"])})
        assert listed.status_code == 200, listed.text
        assert listed.json()["data"] == []

        for path in ["/alertAcknowledge", "/alertTurnOff"]:
            response = test_client.make_request(
                "POST", path, json={"alertId": 999999999, "updateTs": 1})
            assert response.status_code == 404, (path, response.text)
            missing = test_client.make_request(
                "POST", path, json={"updateTs": 1})
            assert missing.status_code == 400, (path, missing.text)
