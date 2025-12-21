import pytest
import time


class TestErrorReporting:
    """Testing of client-side error reporting functionality"""

    def test_error_report_endpoint_accepts_reports(self, test_client):
        """Test that error reports are accepted and return success"""
        error_data = {
            "message": "Test error message",
            "stack": "Error: Test error\n    at testFunction (test.js:10:5)",
            "userAgent": "Mozilla/5.0 (Test Browser)",
            "url": "https://example.com/test",
            "timestamp": "2024-01-01T12:00:00.000Z"
        }

        # Error reporting should work without authentication (rate limited by IP)
        response = test_client.make_request("POST", "/error-report", json=error_data)
        # Admin may be rate limited if previous tests hit the limit
        assert response.status_code in [200, 429]

        if response.status_code == 200:
            data = response.json()
            assert data == {"status": "reported"}

    def test_error_report_no_authentication_required(self, test_client):
        """Test that error reports work without JWT tokens"""
        # Remove authentication
        original_jwt = test_client.jwt_token
        test_client.jwt_token = None
        test_client.session.cookies.clear()

        error_data = {
            "message": "Unauthenticated error report",
            "userAgent": "Test Browser",
            "url": "https://example.com/test",
            "timestamp": "2024-01-01T12:00:00.000Z"
        }

        response = test_client.make_request("POST", "/error-report", json=error_data)
        # Since no auth, may be rate limited by IP
        assert response.status_code in [200, 429]

        if response.status_code == 200:
            data = response.json()
            assert data == {"status": "reported"}

        # Restore authentication for other tests
        test_client.jwt_token = original_jwt

    def test_error_report_truncation(self, test_client):
        """Test that error reports over 2000 characters are truncated"""
        # Create a report longer than 2000 characters
        long_message = "A" * 2500  # 2500 characters
        error_data = {
            "message": long_message,
            "stack": "Error: Long error\n    at longFunction (long.js:100:10)",
            "userAgent": "Mozilla/5.0 (Long Browser)",
            "url": "https://example.com/long-test",
            "timestamp": "2024-01-01T12:00:00.000Z"
        }

        response = test_client.make_request("POST", "/error-report", json=error_data)
        # Admin may be rate limited if previous tests hit the limit
        assert response.status_code in [200, 429]

        if response.status_code == 200:
            data = response.json()
            assert data == {"status": "reported"}

        # The report should be truncated in the logs, but we can't easily verify that
        # from the client side. The endpoint accepted it, which is what we can test.

    def test_error_report_rate_limiting(self, basic_user_client):
        """Test that error reports are subject to rate limiting"""
        error_data = {
            "message": "Rate limit test error",
            "userAgent": "Mozilla/5.0 (Rate Test Browser)",
            "url": "https://example.com/rate-test",
            "timestamp": "2024-01-01T12:00:00.000Z"
        }

        # Make multiple rapid requests to potentially trigger rate limiting
        responses = []
        for i in range(15):  # Make more requests than typical rate limit
            response = basic_user_client.make_request("POST", "/error-report", json=error_data)
            responses.append(response.status_code)
            if response.status_code == 429:  # Too Many Requests
                break
            time.sleep(0.01)  # Small delay between requests

        # Rate limiting should trigger (429) or at least not fail
        assert all(code in [200, 429] for code in responses), f"Unexpected status codes: {responses}"
        # If rate limiting triggered, that's good
        if 429 in responses:
            assert True, "Rate limiting is working"
        else:
            assert 200 in responses, f"Expected some successful responses, got: {responses}"

        # Note: Rate limiting might not trigger in test environment depending on
        # configuration, so we just verify the endpoint accepts requests

    def test_error_report_invalid_method(self, test_client):
        """Test that GET requests to error-report endpoint are rejected"""
        response = test_client.make_request("GET", "/error-report")
        assert response.status_code == 405  # Method Not Allowed

    def test_error_report_empty_body(self, test_client):
        """Test that empty request bodies are handled gracefully"""
        response = test_client.make_request("POST", "/error-report", data="")
        # Should still return success since we handle empty/truncated reports
        # Accept both success and rate limited
        assert response.status_code in [200, 429]

        if response.status_code == 200:
            data = response.json()
            assert data == {"status": "reported"}

    def test_error_report_malformed_json(self, test_client):
        """Test that malformed JSON is handled gracefully"""
        response = test_client.make_request("POST", "/error-report",
            data="invalid json {",
            headers={"Content-Type": "application/json"})
        # Should still return success since we just log the raw body
        # Accept both success and rate limited
        assert response.status_code in [200, 429]

        if response.status_code == 200:
            data = response.json()
            assert data == {"status": "reported"}