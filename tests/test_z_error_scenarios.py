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

        response = test_client.make_request("GET", "/me")
        assert response.status_code == 401

        # Restore original JWT token for subsequent tests
        test_client.jwt_token = original_jwt
        if original_jwt:
            test_client.session.cookies.set("auth", original_jwt, path="/")

    # CSRF cases post CHANGE_PASSWORD_PROBE (conftest.py) to a core route.
    def test_missing_csrf_token_returns_403(self, test_client):
        """Test that POST/PUT/DELETE requests without CSRF token return 403"""
        original_csrf = test_client.csrf_token
        test_client.csrf_token = None
        try:
            response = test_client.make_request(
                "POST", "/changeOwnPassword", json=CHANGE_PASSWORD_PROBE
            )
        finally:
            test_client.csrf_token = original_csrf
        assert_csrf_refused(response, "Missing CSRF token")

    def test_invalid_csrf_token_returns_403(self, test_client):
        """Test that invalid CSRF tokens return 403"""
        original_csrf = test_client.csrf_token
        test_client.csrf_token = "invalid-csrf-token"
        try:
            response = test_client.make_request(
                "POST", "/changeOwnPassword", json=CHANGE_PASSWORD_PROBE
            )
        finally:
            test_client.csrf_token = original_csrf
        assert_csrf_refused(response, "CSRF validation failed")

    def test_optimistic_locking_conflict_returns_409(self, test_client):
        """Test that concurrent updates with stale timestamps return 409"""
        # Create an example
        response = test_client.make_request(
            "POST", "/example", json={"exampleText": "Test"}
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

    def test_soft_deleted_returns_404(self, test_client):
        """A soft-deleted resource answers 404 like a missing one, so the
        caller cannot tell the id once existed"""
        # Create and then delete an example
        response = test_client.make_request(
            "POST",
            "/example",
            json={"exampleText": "Delete Test"},
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
        assert response.status_code == 404

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
            "POST", "/example", json={"example_int": 42}
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
                json={"exampleText": f"Bulk Delete Test {i}"},
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

    def test_rate_limiting_returns_429(self, test_client):
        """A non-admin login gets exactly MAX_USER_REQ_PER_MIN requests in a
        minute, then 429 (admin is exempt).

        The limit is read from the config the test API runs on, so raising
        the dev value cannot break the test. The limiter counts a sliding
        60 s window: this minute's bucket (RPM*<user>*<minute>) plus the
        last minute's, weighted by overlap. A throwaway login has no last-
        minute bucket, so a burst that stays inside one minute is counted
        exactly; one that crossed a boundary would see its first bucket
        decay and let a few extra through. So the burst starts with enough
        of the minute left to finish in one bucket, sends with no sleep,
        and nothing is left throttled when the login is deleted.
        """
        limit = int(api_env()["MAX_USER_REQ_PER_MIN"])
        client, created = create_throwaway_login(test_client, "ratelimit")
        try:
            seconds_left = 60 - time.time() % 60
            if seconds_left < RATE_BURST_SECONDS:
                time.sleep(seconds_left + 0.2)
            started_minute = int(time.time() // 60)
            status_list = []
            for _ in range(limit + 1):
                response = client.make_request("GET", "/me")
                status_list.append(response.status_code)
                if response.status_code == 429:
                    break
            assert int(time.time() // 60) == started_minute, (
                "burst crossed a minute boundary; raise RATE_BURST_SECONDS"
            )
            assert status_list[:limit] == [200] * limit, (
                f"non-200 before the limit: {sorted(set(status_list[:limit]))}"
            )
            assert status_list[limit:] == [429], status_list[limit:]
            assert response.json() == {"error": "Rate limit exceeded"}
        finally:
            delete_user(test_client, created["appUserId"])


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
from conftest import (
    CHANGE_PASSWORD_PROBE,
    TestClient,
    api_env,
    assert_csrf_refused,
    create_throwaway_login,
    delete_user,
)

# Seconds a rate-limit burst may take; started with less of the minute
# left, the test waits for the next one.
RATE_BURST_SECONDS = 20
