"""The help module over HTTP (doc/151-help-module.md).

The e2e API runs with HELP_ENABLED=true. Anyone signed in records a help
usage event in their session's tenant; a user admin (READ_USER) reads the
tenant's counts. Events carry a per-run unique entry id, so a count found
is this run's. There is no delete route: the rows stay in the scratch
database like the rest of a run's data.
"""

import os
import uuid

from conftest import OTHER_TENANT, TestClient, assert_csrf_refused


def _event(client, **fields):
    return client.make_request("POST", "/helpEvent", json=fields)


def _insights(client, days=None):
    params = {"days": days} if days is not None else {}
    return client.make_request("GET", "/helpInsights", params=params)


def _unique_id(label):
    return f"pytest-{label}-{uuid.uuid4().hex[:8]}"


def _count_of(count_list, key):
    return next((row["count"] for row in count_list if row["key"] == key), 0)


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


class TestHelp:
    def test_me_says_help_is_on(self, basic_user_client):
        response = basic_user_client.make_request("GET", "/me")
        assert response.status_code == 200, response.text
        assert response.json()["data"]["helpEnabled"] is True

    def test_events_are_counted_for_a_user_admin(
            self, basic_user_client, test_client):
        entry_id = _unique_id("entry")
        tour_id = _unique_id("tour")
        query = f"pytest search {uuid.uuid4().hex[:8]}"
        for fields in (
            {"event": "entry_view", "entryId": entry_id, "route": "/login"},
            {"event": "entry_view", "entryId": entry_id, "route": "/login"},
            {"event": "tour_skip", "entryId": tour_id, "route": "/x"},
            {"event": "search_no_result", "query": query, "route": "/x"},
        ):
            response = _event(basic_user_client, **fields)
            assert response.status_code == 200, response.text
            assert response.json() == {"data": {"recorded": True}}

        response = _insights(test_client, days=1)
        assert response.status_code == 200, response.text
        data = response.json()["data"]
        assert data["days"] == 1
        assert _count_of(data["entryViewList"], entry_id) == 2
        assert _count_of(data["tourSkipList"], tour_id) == 1
        assert _count_of(data["noResultSearchList"], query) == 1
        assert _count_of(data["tourCompleteList"], tour_id) == 0

    def test_insights_need_read_user(self, basic_user_client):
        response = _insights(basic_user_client)
        assert response.status_code == 403, response.text

    def test_a_bad_event_is_refused(self, basic_user_client):
        for fields in (
            {"event": "panel_open", "entryId": "mfa", "route": "/x"},
            {"event": "entry_view", "entryId": "<b>", "route": "/x"},
            {"event": "entry_view", "entryId": "mfa", "route": "x"},
            {"event": "search_no_result", "query": " ", "route": "/x"},
        ):
            response = _event(basic_user_client, **fields)
            assert response.status_code == 400, (fields, response.text)

    def test_an_event_without_csrf_is_refused(self, basic_user_client):
        response = basic_user_client.session.post(
            f"{basic_user_client.base_url}/helpEvent",
            json={"event": "entry_view", "entryId": "mfa", "route": "/x"})
        assert_csrf_refused(response, "Missing CSRF token")

    def test_days_out_of_range_is_refused(self, test_client):
        for days in ("0", "366", "week"):
            response = _insights(test_client, days=days)
            assert response.status_code == 400, (days, response.text)

    def test_another_tenant_cannot_see_the_events(self, test_client):
        # switcher is homed in tenant 2 and can switch to 3 (test-seed.sql).
        switcher = TestClient(os.environ["TEST_BASE_URL"])
        switcher.authenticate("switcher", "basic123!")
        own_id = _unique_id("own")
        response = _event(switcher, event="entry_view", entryId=own_id,
                          route="/dashboard")
        assert response.status_code == 200, response.text

        _switch(switcher, OTHER_TENANT)
        other_id = _unique_id("other")
        response = _event(switcher, event="entry_view", entryId=other_id,
                          route="/dashboard")
        assert response.status_code == 200, response.text

        response = _insights(test_client, days=1)
        assert response.status_code == 200, response.text
        view_list = response.json()["data"]["entryViewList"]
        # Read first: the tenant-2 event is there, so the list was read.
        assert _count_of(view_list, own_id) == 1
        assert _count_of(view_list, other_id) == 0
