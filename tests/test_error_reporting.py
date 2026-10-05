import requests


ERROR_REPORT = {
    "message": "Test error message",
    "stack": "Error: Test error\n    at testFunction (test.js:10:5)",
    "url": "https://example.com/test",
}


class TestErrorReporting:
    """POST /error-report: a signed-in caller's client-side error, logged
    with its username. Anonymous callers cannot write to the log."""

    def test_signed_in_report_is_accepted(self, test_client):
        response = test_client.make_request(
            "POST", "/error-report", json=ERROR_REPORT
        )
        assert response.status_code == 200, response.text
        assert response.json() == {"status": "reported"}

    def test_basic_user_can_report(self, basic_user_client):
        response = basic_user_client.make_request(
            "POST", "/error-report", json=ERROR_REPORT
        )
        assert response.status_code == 200, response.text

    def test_no_session_is_401(self, test_client):
        response = requests.post(
            f"{test_client.base_url}/error-report",
            json=ERROR_REPORT,
            verify=False,
        )
        assert response.status_code == 401, response.text

    def test_session_without_csrf_is_403(self, test_client):
        response = test_client.session.post(
            f"{test_client.base_url}/error-report", json=ERROR_REPORT
        )
        assert response.status_code == 403, response.text

    def test_get_is_405(self, test_client):
        response = test_client.make_request("GET", "/error-report")
        assert response.status_code == 405

    def test_raw_body_is_logged_as_sent(self, test_client):
        """The body is logged raw, not parsed: empty or non-JSON is fine."""
        for body in ["", "invalid json {"]:
            response = test_client.make_request(
                "POST", "/error-report", data=body,
                headers={"Content-Type": "application/json"},
            )
            assert response.status_code == 200, (body, response.text)

    def test_over_8_kib_is_413(self, test_client):
        response = test_client.make_request(
            "POST", "/error-report", data="a" * (8 * 1024 + 1)
        )
        assert response.status_code == 413, response.text
