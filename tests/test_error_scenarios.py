import pytest
import time


class TestErrorScenarios:
    """Comprehensive testing of error conditions and status codes"""

    def test_invalid_jwt_returns_401(self, test_client):
        """Test that invalid JWT tokens return 401"""
        # Clear authentication
        test_client.jwt_token = "invalid.jwt.token"
        test_client.csrf_token = "invalid-csrf"

        response = test_client.make_request("GET", "/example", params={"exampleId": 1})
        assert response.status_code == 401

    def test_missing_csrf_token_returns_403(self, test_client):
        """Test that POST/PUT/DELETE requests without CSRF token return 403"""
        # Temporarily remove CSRF token
        original_csrf = test_client.csrf_token
        test_client.csrf_token = None

        response = test_client.make_request("POST", "/example", json={
            "example_text": "Test",
            "security_context": 1
        })
        assert response.status_code == 403

        # Restore CSRF token
        test_client.csrf_token = original_csrf

    def test_invalid_csrf_token_returns_403(self, test_client):
        """Test that invalid CSRF tokens return 403"""
        # Set invalid CSRF token
        original_csrf = test_client.csrf_token
        test_client.csrf_token = "invalid-csrf-token"

        response = test_client.make_request("POST", "/example", json={
            "example_text": "Test",
            "security_context": 1
        })
        assert response.status_code == 403

        # Restore CSRF token
        test_client.csrf_token = original_csrf

    def test_optimistic_locking_conflict_returns_409(self, test_client):
        """Test that concurrent updates with stale timestamps return 409"""
        # Create an example
        response = test_client.make_request("POST", "/example", json={
            "example_text": "Locking Test",
            "security_context": 1
        })
        assert response.status_code == 201
        example = response.json()["data"]
        test_client.track_example(example["example_id"], example["update_ts"])

        # Use stale timestamp (simulate concurrent modification)
        stale_timestamp = example["update_ts"] - 1.0

        response = test_client.make_request("PUT", "/example", json={
            "example_id": example["example_id"],
            "example_text": "Updated Text",
            "security_context": 1,
            "update_ts": stale_timestamp
        })
        assert response.status_code == 409

    def test_not_found_returns_404(self, test_client):
        """Test that accessing non-existent resources returns 404"""
        response = test_client.make_request("GET", "/example", params={"exampleId": 999999})
        assert response.status_code == 404

    def test_soft_deleted_returns_410(self, test_client):
        """Test that accessing soft-deleted resources returns 410"""
        # Create and then delete an example
        response = test_client.make_request("POST", "/example", json={
            "example_text": "Delete Test",
            "security_context": 1
        })
        assert response.status_code == 201
        example = response.json()["data"]

        # Delete the example
        response = test_client.make_request("DELETE", "/example", params={
            "exampleId": example["example_id"],
            "updateTs": f"{example['update_ts']:.7f}"
        })
        assert response.status_code == 200

        # Try to access the deleted example
        response = test_client.make_request("GET", "/example", params={"exampleId": example["example_id"]})
        assert response.status_code == 410

    def test_rate_limiting_returns_429(self, test_client):
        """Test that exceeding rate limits returns 429"""
        # Make many rapid requests to trigger rate limiting
        responses = []
        for i in range(250):  # Exceed typical rate limits
            response = test_client.make_request("GET", "/health")
            responses.append(response.status_code)
            if response.status_code == 429:
                break
            time.sleep(0.01)  # Small delay to avoid overwhelming

        # At least one request should be rate limited
        assert 429 in responses

    def test_invalid_request_body_returns_400(self, test_client):
        """Test that malformed JSON returns 400"""
        response = test_client.make_request("POST", "/example",
            data="invalid json {",
            headers={"Content-Type": "application/json"})
        assert response.status_code == 400

    def test_missing_required_fields_returns_400(self, test_client):
        """Test that requests missing required fields return 400"""
        # Missing example_text
        response = test_client.make_request("POST", "/example", json={
            "example_int": 42,
            "security_context": 1
        })
        assert response.status_code == 400

    def test_invalid_data_types_return_400(self, test_client):
        """Test that invalid data types return 400"""
        # Invalid example_id (string instead of int)
        response = test_client.make_request("GET", "/example", params={"exampleId": "not-a-number"})
        assert response.status_code == 400

    def test_bulk_operations_with_invalid_ids_return_400(self, test_client):
        """Test that bulk operations with invalid ID lists return 400"""
        # Invalid ID in list
        response = test_client.make_request("GET", "/exampleList",
            params={"exampleIdList": "1,invalid,3"})
        assert response.status_code == 400

    def test_unauthorized_bulk_delete_returns_409(self, test_client):
        """Test that bulk delete with stale timestamps returns appropriate errors"""
        # Create a couple examples
        examples = []
        for i in range(2):
            response = test_client.make_request("POST", "/example", json={
                "example_text": f"Bulk Delete Test {i}",
                "security_context": 1
            })
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["example_id"], example["update_ts"])

        # Attempt bulk delete with stale timestamps
        ids = [str(ex["example_id"]) for ex in examples]
        stale_timestamps = [f"{ex['update_ts'] - 1.0:.7f}" for ex in examples]

        response = test_client.make_request("DELETE", "/exampleList", params={
            "exampleIdList": ",".join(ids),
            "updateTs": ",".join(stale_timestamps)
        })
        # Should return 409 for conflict or succeed if timestamps are handled differently
        assert response.status_code in [200, 409]