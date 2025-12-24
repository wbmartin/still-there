import time

import pytest


class TestExampleCRUD:
    """Full lifecycle testing of example CRUD operations"""

    example_id = None  # Class variable to share across test methods

    @pytest.fixture(autouse=True)
    def setup_method(self, test_client):
        """Setup for each test method"""
        self.client = test_client
        self.example_id = self.__class__.example_id

    def test_create_example(self):
        """Test creating a new example"""
        example_data = {
            "exampleText": "Integration Test Example",
            "exampleInt": 42,
            "exampleFloat": 3.14159,
            "exampleIntArray": [1, 2, 3],
            "exampleFloatArray": [1.1, 2.2, 3.3],
            "securityContext": 1,
        }

        response = self.client.make_request("POST", "/example", json=example_data)
        assert response.status_code == 201

        data = response.json()["data"]
        assert "exampleId" in data
        assert data["exampleText"] == "Integration Test Example"
        assert data["exampleInt"] == 42

        # Store ID and timestamp for subsequent tests and cleanup
        self.example_id = data["exampleId"]
        self.__class__.example_id = self.example_id
        self.client.track_example(self.example_id, data["updateTs"])

    def test_get_example(self):
        """Test retrieving the created example"""
        assert self.example_id is not None, "Example must be created first"

        response = self.client.make_request(
            "GET", "/example", params={"exampleId": self.example_id}
        )
        assert response.status_code == 200

        data = response.json()["data"]
        assert data["exampleId"] == self.example_id
        assert data["exampleText"] == "Integration Test Example"
        assert data["exampleInt"] == 42

    def test_update_example(self):
        """Test updating the example"""
        assert self.example_id is not None, "Example must be created first"

        # Get current update timestamp for optimistic locking
        response = self.client.make_request(
            "GET", "/example", params={"exampleId": self.example_id}
        )
        assert response.status_code == 200
        current_update_ts = response.json()["data"]["updateTs"]

        update_data = {
            "exampleId": self.example_id,
            "exampleText": "Updated Integration Test Example",
            "exampleInt": 43,
            "exampleFloat": 3.14159,
            "exampleIntArray": [1, 2, 3, 4],
            "exampleFloatArray": [1.1, 2.2, 3.3, 4.4],
            "updateTs": current_update_ts,
            "securityContext": 1,
        }

        response = self.client.make_request("PUT", "/example", json=update_data)
        assert response.status_code == 200

        # Verify the update
        response = self.client.make_request(
            "GET", "/example", params={"exampleId": self.example_id}
        )
        assert response.status_code == 200

        data = response.json()["data"]
        assert data["exampleText"] == "Updated Integration Test Example"
        assert data["exampleInt"] == 43
        assert data["exampleIntArray"] == [1, 2, 3, 4]

    def test_delete_example(self):
        """Test deleting the example"""
        assert self.example_id is not None, "Example must be created first"

        # Get current update timestamp for optimistic locking
        response = self.client.make_request(
            "GET", "/example", params={"exampleId": self.example_id}
        )
        assert response.status_code == 200
        update_ts = response.json()["data"]["updateTs"]

        response = self.client.make_request(
            "DELETE",
            "/example",
            params={"exampleId": self.example_id, "updateTs": f"{update_ts:.7f}"},
        )
        assert response.status_code == 200

        # Verify deletion (soft delete returns 410)
        response = self.client.make_request(
            "GET", "/example", params={"exampleId": self.example_id}
        )
        assert response.status_code == 410

        # Reset class variable
        self.__class__.example_id = None

    def test_bulk_operations(self):
        """Test bulk create, update, and delete operations"""
        # Create multiple examples
        examples = [
            {
                "exampleText": "Bulk Test Example 1",
                "exampleInt": 100,
                "securityContext": 1,
            },
            {
                "exampleText": "Bulk Test Example 2",
                "exampleInt": 200,
                "securityContext": 1,
            },
        ]

        response = self.client.make_request("POST", "/exampleList", json=examples)
        assert response.status_code == 201

        created_data = response.json()["data"]
        assert len(created_data) == 2

        example_ids = []
        update_timestamps = []
        for ex in created_data:
            example_ids.append(ex["exampleId"])
            update_timestamps.append(ex["updateTs"])
            self.client.track_example(ex["exampleId"], ex["updateTs"])

        # Update them
        updates = [
            {
                "exampleId": example_ids[0],
                "exampleText": "Updated Bulk Test Example 1",
                "exampleInt": 101,
                "updateTs": update_timestamps[0],
                "securityContext": 1,
            },
            {
                "exampleId": example_ids[1],
                "exampleText": "Updated Bulk Test Example 2",
                "exampleInt": 201,
                "updateTs": update_timestamps[1],
                "securityContext": 1,
            },
        ]

        response = self.client.make_request("PUT", "/exampleList", json=updates)
        assert response.status_code == 200

        # Verify updates
        response = self.client.make_request(
            "GET",
            "/exampleList",
            params={"exampleIdList": f"{example_ids[0]},{example_ids[1]}"},
        )
        assert response.status_code == 200

        retrieved_data = response.json()["data"]
        assert len(retrieved_data) == 2

        for ex in retrieved_data:
            if ex["exampleId"] == example_ids[0]:
                assert ex["exampleText"] == "Updated Bulk Test Example 1"
                assert ex["exampleInt"] == 101
            elif ex["exampleId"] == example_ids[1]:
                assert ex["exampleText"] == "Updated Bulk Test Example 2"
                assert ex["exampleInt"] == 201

        # Delete them (cleanup is handled by fixture)
        response = self.client.make_request(
            "DELETE",
            "/exampleList",
            params={
                "exampleIdList": f"{example_ids[0]},{example_ids[1]}",
                "updateTs": f"{update_timestamps[0]:.7f},{update_timestamps[1]:.7f}",
            },
        )
        assert response.status_code == 200

    def test_error_scenarios(self):
        """Test various error scenarios"""
        # Try to get non-existent example
        response = self.client.make_request(
            "GET", "/example", params={"exampleId": 2147483646}
        )
        assert response.status_code == 404

        # Try to update non-existent example
        update_data = {
            "exampleId": 2147483646,
            "exampleText": "This should fail",
            "securityContext": 1,
        }
        response = self.client.make_request("PUT", "/example", json=update_data)
        assert response.status_code == 404

        # Try to delete non-existent example
        response = self.client.make_request(
            "DELETE",
            "/example",
            params={"exampleId": 2147483646, "updateTs": "0.000000"},
        )
        assert response.status_code == 404

        # Try to create example with missing required field
        invalid_data = {"exampleInt": 42, "securityContext": 1}
        response = self.client.make_request("POST", "/example", json=invalid_data)
        assert response.status_code == 400
