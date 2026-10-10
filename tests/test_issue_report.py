"""The issue-report module over HTTP (doc/150-issue-report-module.md).

Anyone signed in files a bug, polish or feature report with markdown and
files; everyone in the tenant sees the tenant's reports; a reporter adds to
the conversation, withdraws an open report and reopens a closed one. The
e2e API runs with the module on and no relay, so reports stay in the app.
Files follow doc/105, Serving user files: the type is read from the bytes,
only an image is inline, nothing is cached, another tenant's id is 404.
"""

import os
import uuid

import pytest
import requests

from conftest import OTHER_TENANT, TEST_TENANT, TestClient

PNG = (b"\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00"
       b"\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc"
       b"\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND"
       b"\xaeB`\x82")
HTML = b"<!doctype html><html><body><script>alert(1)</script></body></html>"


def _create(client, data=None, files=None):
    fields = {
        "kind": "bug",
        "summary": f"[pytest] Save fails {uuid.uuid4().hex[:6]}",
        "detail": "## Steps\n\n1. Open the page\n2. Click **Save**",
        "pageUrl": "/exampleList",
        "context": '{"browser":"pytest"}',
    }
    fields.update(data or {})
    if files is None:
        files = [("files", ("shot.png", PNG, "image/png"))]
    return client.make_request("POST", "/issueReport", data=fields,
                               files=files)


def _download(client, attachment_id):
    return client.make_request(
        "GET", "/issueReportAttachment",
        params={"issueReportAttachmentId": attachment_id})


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


class TestIssueReport:
    def test_me_says_the_module_is_on(self, test_client):
        me = test_client.make_request("GET", "/me")
        assert me.status_code == 200, me.text
        assert me.json()["data"]["issueReportsEnabled"] is True

    def test_create_list_detail_and_download(self, test_client):
        response = _create(test_client)
        assert response.status_code == 201, response.text
        report = response.json()["data"]
        assert report["kind"] == "bug"
        assert report["state"] == "open"
        assert report["status"] == "Received"
        assert report["relayState"] == "local"
        assert report["securityContext"] == TEST_TENANT
        assert report["detail"].startswith("## Steps")
        assert report["pageUrl"] == "/exampleList"
        assert report["reporterName"]
        # The relay's columns and the reporter's email stay off the wire.
        for hidden in ("reporterEmail", "remoteId", "relayError",
                       "contextJsonb"):
            assert hidden not in report
        assert len(report["attachmentList"]) == 1
        attachment = report["attachmentList"][0]
        assert attachment["filename"] == "shot.png"
        assert attachment["contentType"] == "image/png"
        assert attachment["sizeBytes"] == len(PNG)

        listed = test_client.make_request(
            "GET", "/issueReportList").json()["data"]
        assert listed, "the list came back empty"
        assert report["issueReportId"] in [
            row["issueReportId"] for row in listed]

        detail = test_client.make_request(
            "GET", "/issueReport",
            params={"issueReportId": report["issueReportId"]})
        assert detail.status_code == 200, detail.text
        assert detail.json()["data"]["summary"] == report["summary"]

        download = _download(test_client,
                             attachment["issueReportAttachmentId"])
        assert download.status_code == 200
        assert download.content == PNG
        assert download.headers["Content-Type"] == "image/png"
        assert download.headers["Content-Disposition"].startswith("inline")
        assert download.headers["X-Content-Type-Options"] == "nosniff"
        assert download.headers["X-Frame-Options"] == "DENY"
        assert download.headers["Cache-Control"] == "private, no-store"

    @pytest.mark.parametrize("data,files,status,needle", [
        ({"kind": "idea"}, [], 400, "bug, polish or feature"),
        ({"summary": "   "}, [], 400, "summary is required"),
        ({"summary": "x" * 501}, [], 400, "summary"),
        ({}, [("files", ("run.exe", b"MZ", "application/octet-stream"))],
         415, "attach images"),
        ({}, [("files", (f"f{i}.txt", b"hi", "text/plain"))
              for i in range(6)], 400, "at most 5"),
        ({}, [("files", ("shot.png", HTML, "image/png"))], 415,
         "not what its name says"),
        ({}, [("files", ("doc.pdf", HTML, "application/pdf"))], 415,
         "not what its name says"),
        ({}, [("files", ("notes.txt", HTML, "text/plain"))], 415,
         "not what its name says"),
        ({}, [("files", ("mockup.html", HTML, "text/html"))], 415,
         "not on a bug"),
    ])
    def test_create_refuses_bad_input(self, test_client, data, files,
                                      status, needle):
        response = _create(test_client, data=data, files=files)
        assert response.status_code == status, response.text
        assert needle in response.json()["error"].lower()

    def test_over_the_request_cap_is_413(self, test_client):
        big = b"a" * (18 * 1024 * 1024 + 1)
        response = _create(test_client, files=[
            ("files", ("big.txt", big, "text/plain"))])
        assert response.status_code == 413, response.text[:200]

    @pytest.mark.parametrize("kind", ["polish", "feature"])
    def test_example_html_page_is_a_download(self, test_client, kind):
        response = _create(test_client, data={"kind": kind}, files=[
            ("files", ("mockup.html", HTML, "text/html"))])
        assert response.status_code == 201, response.text
        attachment = response.json()["data"]["attachmentList"][0]
        assert attachment["filename"] == "mockup.html"
        download = _download(test_client,
                             attachment["issueReportAttachmentId"])
        assert download.status_code == 200
        assert download.content == HTML
        # Never rendered: a neutral type, a download, nosniff.
        assert download.headers["Content-Type"] == "application/octet-stream"
        assert download.headers["Content-Disposition"].startswith(
            "attachment")
        assert download.headers["X-Content-Type-Options"] == "nosniff"

    def test_message_withdraw_reopen(self, test_client):
        report = _create(test_client, files=[]).json()["data"]
        report_id = report["issueReportId"]

        message = test_client.make_request(
            "POST", "/issueReportMessage",
            data={"issueReportId": str(report_id),
                  "body": "Also on **mobile**"},
            files=[("files", ("more.png", PNG, "image/png"))])
        assert message.status_code == 200, message.text
        detail = message.json()["data"]
        last = detail["threadList"][-1]
        assert last["authorKind"] == "reporter"
        assert last["body"] == "Also on **mobile**"
        assert len(last["attachmentList"]) == 1
        assert detail["messageCount"] == 1

        empty = test_client.make_request(
            "POST", "/issueReportMessage",
            data={"issueReportId": str(report_id), "body": "  "})
        assert empty.status_code == 400, empty.text

        reopen_open = test_client.make_request(
            "POST", "/issueReportAction",
            json={"issueReportId": report_id, "action": "reopen"})
        assert reopen_open.status_code == 409, reopen_open.text

        withdraw = test_client.make_request(
            "POST", "/issueReportAction",
            json={"issueReportId": report_id, "action": "withdraw",
                  "note": "Found a workaround"})
        assert withdraw.status_code == 200, withdraw.text
        detail = withdraw.json()["data"]
        assert detail["state"] == "closed"
        assert detail["status"] == "Withdrawn"
        assert detail["threadList"][-1]["action"] == "withdraw"
        assert "Found a workaround" in detail["threadList"][-1]["body"]

        again = test_client.make_request(
            "POST", "/issueReportAction",
            json={"issueReportId": report_id, "action": "withdraw"})
        assert again.status_code == 409, again.text

        reopen = test_client.make_request(
            "POST", "/issueReportAction",
            json={"issueReportId": report_id, "action": "reopen"})
        assert reopen.status_code == 200, reopen.text
        assert reopen.json()["data"]["state"] == "reopened"

        unknown = test_client.make_request(
            "POST", "/issueReportAction",
            json={"issueReportId": report_id, "action": "close"})
        assert unknown.status_code == 400, unknown.text

    def test_the_whole_tenant_sees_reports(self, test_client,
                                           basic_user_client):
        report = _create(test_client, files=[]).json()["data"]
        seen = basic_user_client.make_request(
            "GET", "/issueReport",
            params={"issueReportId": report["issueReportId"]})
        assert seen.status_code == 200, seen.text
        mine = _create(basic_user_client, files=[])
        assert mine.status_code == 201, mine.text

    def test_no_session_is_401_and_no_csrf_is_403(self, test_client):
        anonymous = requests.get(f"{test_client.base_url}/issueReportList",
                                 verify=False)
        assert anonymous.status_code == 401
        no_csrf = test_client.session.post(
            f"{test_client.base_url}/issueReport",
            data={"kind": "bug", "summary": "x"})
        assert no_csrf.status_code == 403, no_csrf.text

    def test_another_tenant_cannot_reach_reports(self, test_client):
        """The cross-tenant negative test at the API (doc/110): a report
        and its file made in TEST_TENANT are 404 from OTHER_TENANT, absent
        from its list, and take no message. switcher, homed in TEST_TENANT
        with a group in OTHER_TENANT, is only signed in and switched here,
        never edited."""
        report = _create(test_client).json()["data"]
        attachment_id = report["attachmentList"][0][
            "issueReportAttachmentId"]

        client = TestClient(os.environ["TEST_BASE_URL"])
        client.authenticate("switcher", "basic123!")
        _switch(client, OTHER_TENANT)
        theirs = _create(client, files=[])
        assert theirs.status_code == 201, theirs.text
        assert theirs.json()["data"]["securityContext"] == OTHER_TENANT

        listed = client.make_request(
            "GET", "/issueReportList").json()["data"]
        assert listed, "OTHER_TENANT's list came back empty"
        assert report["issueReportId"] not in [
            row["issueReportId"] for row in listed]
        for method, path, kwargs in [
            ("GET", "/issueReport",
             {"params": {"issueReportId": report["issueReportId"]}}),
            ("GET", "/issueReportAttachment",
             {"params": {"issueReportAttachmentId": attachment_id}}),
            ("POST", "/issueReportMessage",
             {"data": {"issueReportId": str(report["issueReportId"]),
                       "body": "x"}}),
            ("POST", "/issueReportAction",
             {"json": {"issueReportId": report["issueReportId"],
                       "action": "withdraw"}}),
        ]:
            response = client.make_request(method, path, **kwargs)
            assert response.status_code == 404, (path, response.text)

        after = test_client.make_request(
            "GET", "/issueReport",
            params={"issueReportId": report["issueReportId"]})
        assert after.status_code == 200, after.text
        assert after.json()["data"]["threadList"] == []
        assert after.json()["data"]["state"] == "open"
        _switch(client, TEST_TENANT)
