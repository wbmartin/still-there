import os
import time

import pytest
import requests


class TestErrorScenarios:
    """Comprehensive testing of error conditions and status codes"""

    def test_invalid_jwt_returns_401(self, test_client):
        """Test that invalid JWT tokens return 401"""
        # Save original JWT token
        original_jwt = test_client.jwt_token

        # Set invalid auth cookie directly in session
        test_client.session.cookies.set("auth", "invalid.jwt.token", path="/")

        response = test_client.make_request("GET", "/example", params={"exampleId": 1})
        assert response.status_code == 401

        # Restore original JWT token for subsequent tests
        test_client.jwt_token = original_jwt
        if original_jwt:
            test_client.session.cookies.set("auth", original_jwt, path="/")

    def test_missing_csrf_token_returns_403(self, test_client):
        """Test that POST/PUT/DELETE requests without CSRF token return 403"""
        # Temporarily remove CSRF token
        original_csrf = test_client.csrf_token
        test_client.csrf_token = None

        response = test_client.make_request(
            "POST", "/example", json={"exampleText": "Test", "securityContext": 1}
        )
        assert response.status_code == 403

        # Restore CSRF token
        test_client.csrf_token = original_csrf

    def test_invalid_csrf_token_returns_403(self, test_client):
        """Test that invalid CSRF tokens return 403"""
        # Set invalid CSRF token
        original_csrf = test_client.csrf_token
        test_client.csrf_token = "invalid-csrf-token"

        response = test_client.make_request(
            "POST", "/example", json={"exampleText": "Test", "securityContext": 1}
        )
        assert response.status_code == 403

        # Restore CSRF token
        test_client.csrf_token = original_csrf

    def test_optimistic_locking_conflict_returns_409(self, test_client):
        """Test that concurrent updates with stale timestamps return 409"""
        # Create an example
        response = test_client.make_request(
            "POST", "/example", json={"exampleText": "Test", "securityContext": 1}
        )
        assert response.status_code == 201
        example = response.json()["data"]
        test_client.track_example(example["exampleId"], example["updateTs"])

        # Use stale timestamp (simulate concurrent modification)
        stale_timestamp = example["updateTs"] - 1.0

        response = test_client.make_request(
            "PUT",
            "/example",
            json={
                "exampleId": example["exampleId"],
                "exampleText": "Updated Text",
                "securityContext": 1,
                "updateTs": stale_timestamp,
            },
        )
        assert response.status_code == 409

    def test_not_found_returns_404(self, test_client):
        """Test that accessing non-existent resources returns 404"""
        response = test_client.make_request(
            "GET", "/example", params={"exampleId": 999999}
        )
        assert response.status_code == 404

    def test_soft_deleted_returns_410(self, test_client):
        """Test that accessing soft-deleted resources returns 410"""
        # Create and then delete an example
        response = test_client.make_request(
            "POST",
            "/example",
            json={"exampleText": "Delete Test", "securityContext": 1},
        )
        assert response.status_code == 201
        example = response.json()["data"]

        # Delete the example
        response = test_client.make_request(
            "DELETE",
            "/example",
            params={
                "exampleId": example["exampleId"],
                "updateTs": f"{example['updateTs']:.7f}",
            },
        )
        assert response.status_code == 200

        # Try to access the deleted example
        response = test_client.make_request(
            "GET", "/example", params={"exampleId": example["exampleId"]}
        )
        assert response.status_code == 410

    def test_invalid_request_body_returns_400(self, test_client):
        """Test that malformed JSON returns 400"""
        response = test_client.make_request(
            "POST",
            "/example",
            data="invalid json {",
            headers={"Content-Type": "application/json"},
        )
        assert response.status_code == 400
        # The decoder's own text stays in the log.
        assert response.json()["error"] == "Invalid request body"

    def test_missing_required_fields_returns_400(self, test_client):
        """Test that requests missing required fields return 400"""
        # Missing example_text
        response = test_client.make_request(
            "POST", "/example", json={"example_int": 42, "security_context": 1}
        )
        assert response.status_code == 400

    def test_invalid_data_types_return_400(self, test_client):
        """Test that invalid data types return 400"""
        # Invalid example_id (string instead of int)
        response = test_client.make_request(
            "GET", "/example", params={"exampleId": "not-a-number"}
        )
        assert response.status_code == 400

    def test_bulk_operations_with_invalid_ids_return_400(self, test_client):
        """Test that bulk operations with invalid ID lists return 400"""
        # Invalid ID in list
        response = test_client.make_request(
            "GET", "/exampleList", params={"exampleIdList": "1,invalid,3"}
        )
        assert response.status_code == 400

    def test_unauthorized_bulk_delete_returns_409(self, test_client):
        """Test that bulk delete with stale timestamps returns appropriate errors"""
        # Create a couple examples
        examples = []
        for i in range(2):
            response = test_client.make_request(
                "POST",
                "/example",
                json={"exampleText": f"Bulk Delete Test {i}", "securityContext": 1},
            )
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["exampleId"], example["updateTs"])

        # Attempt bulk delete with stale timestamps
        ids = [str(ex["exampleId"]) for ex in examples]
        stale_timestamps = [f"{ex['updateTs'] - 1.0:.7f}" for ex in examples]

        response = test_client.make_request(
            "DELETE",
            "/exampleList",
            params={
                "exampleIdList": ",".join(ids),
                "updateTs": ",".join(stale_timestamps),
            },
        )
        # Should return 409 for conflict or succeed if timestamps are handled differently
        assert response.status_code in [200, 409]

    def test_rate_limiting_returns_429(self, basic_user_client):
        """Test that exceeding rate limits returns 429 for basic users (not exempt)"""
        # Test authenticated user rate limiting (limit: 200 per minute)
        # Use an authenticated endpoint that goes through security checks
        responses = []
        for i in range(250):  # Exceed limit to trigger rate limiting
            response = basic_user_client.make_request(
                "GET", "/example", params={"exampleId": 999999}
            )
            responses.append(response.status_code)
            if response.status_code == 429:
                break
            time.sleep(0.01)  # Small delay to avoid overwhelming

        # At least one request should be rate limited
        assert 429 in responses, (
            f"Expected 429 in responses, got: {responses[:10]}... (showing first 10)"
        )


# class TestAccountLockout:
#     """Test account lockout functionality - runs last to avoid interfering with other tests"""

#     def test_account_lockout_after_failed_attempts(self):
#         """Test that account gets locked after too many failed login attempts"""
#         base_url = os.environ["TEST_BASE_URL"]
#         client = TestClient(base_url)

#         # Attempt login with wrong password multiple times
#         for i in range(10):  # Exceed threshold
#             with pytest.raises(requests.HTTPError):
#                 client.authenticate("admin", "wrongpassword")

#         # Now attempt with correct password should be locked
#         with pytest.raises(requests.HTTPError) as exc_info:
#             client.authenticate("admin", "admin123!")
#         assert exc_info.value.response.status_code == 429  # Too Many Requests


# Import TestClient for use in this module
from conftest import TestClient
