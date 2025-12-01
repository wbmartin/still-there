import pytest
import time


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
            "example_text": "Integration Test Example",
            "example_int": 42,
            "example_float": 3.14159,
            "example_int_array": [1, 2, 3],
            "example_float_array": [1.1, 2.2, 3.3],
            "security_context": 1
        }

        response = self.client.make_request("POST", "/example", json=example_data)
        assert response.status_code == 201

        data = response.json()["data"]
        assert "example_id" in data
        assert data["example_text"] == "Integration Test Example"
        assert data["example_int"] == 42

        # Store ID and timestamp for subsequent tests and cleanup
        self.example_id = data["example_id"]
        self.__class__.example_id = self.example_id
        self.client.track_example(self.example_id, data["update_ts"])

    def test_get_example(self):
        """Test retrieving the created example"""
        assert self.example_id is not None, "Example must be created first"

        response = self.client.make_request("GET", "/example",
            params={"exampleId": self.example_id})
        assert response.status_code == 200

        data = response.json()["data"]
        assert data["example_id"] == self.example_id
        assert data["example_text"] == "Integration Test Example"
        assert data["example_int"] == 42

    def test_update_example(self):
        """Test updating the example"""
        assert self.example_id is not None, "Example must be created first"

        # Get current update timestamp for optimistic locking
        response = self.client.make_request("GET", "/example",
            params={"exampleId": self.example_id})
        assert response.status_code == 200
        current_update_ts = response.json()["data"]["update_ts"]

        update_data = {
            "example_id": self.example_id,
            "example_text": "Updated Integration Test Example",
            "example_int": 43,
            "example_float": 3.14159,
            "example_int_array": [1, 2, 3, 4],
            "example_float_array": [1.1, 2.2, 3.3, 4.4],
            "update_ts": current_update_ts,
            "security_context": 1
        }

        response = self.client.make_request("PUT", "/example", json=update_data)
        assert response.status_code == 200

        # Verify the update
        response = self.client.make_request("GET", "/example",
            params={"exampleId": self.example_id})
        assert response.status_code == 200

        data = response.json()["data"]
        assert data["example_text"] == "Updated Integration Test Example"
        assert data["example_int"] == 43
        assert data["example_int_array"] == [1, 2, 3, 4]

    def test_delete_example(self):
        """Test deleting the example"""
        assert self.example_id is not None, "Example must be created first"

        # Get current update timestamp for optimistic locking
        response = self.client.make_request("GET", "/example",
            params={"exampleId": self.example_id})
        assert response.status_code == 200
        update_ts = response.json()["data"]["update_ts"]

        response = self.client.make_request("DELETE", "/example",
            params={
                "exampleId": self.example_id,
                "updateTs": f"{update_ts:.7f}"
            })
        assert response.status_code == 200

        # Verify deletion (soft delete returns 410)
        response = self.client.make_request("GET", "/example",
            params={"exampleId": self.example_id})
        assert response.status_code == 410

        # Reset class variable
        self.__class__.example_id = None

    def test_bulk_operations(self):
        """Test bulk create, update, and delete operations"""
        # Create multiple examples
        examples = [
            {
                "example_text": "Bulk Test Example 1",
                "example_int": 100,
                "security_context": 1
            },
            {
                "example_text": "Bulk Test Example 2",
                "example_int": 200,
                "security_context": 1
            }
        ]

        response = self.client.make_request("POST", "/exampleList", json=examples)
        assert response.status_code == 201

        created_data = response.json()["data"]
        assert len(created_data) == 2

        example_ids = []
        update_timestamps = []
        for ex in created_data:
            example_ids.append(ex["example_id"])
            update_timestamps.append(ex["update_ts"])
            self.client.track_example(ex["example_id"], ex["update_ts"])

        # Update them
        updates = [
            {
                "example_id": example_ids[0],
                "example_text": "Updated Bulk Test Example 1",
                "example_int": 101,
                "update_ts": update_timestamps[0],
                "security_context": 1
            },
            {
                "example_id": example_ids[1],
                "example_text": "Updated Bulk Test Example 2",
                "example_int": 201,
                "update_ts": update_timestamps[1],
                "security_context": 1
            }
        ]

        response = self.client.make_request("PUT", "/exampleList", json=updates)
        assert response.status_code == 200

        # Verify updates
        response = self.client.make_request("GET", "/exampleList",
            params={"exampleIdList": f"{example_ids[0]},{example_ids[1]}"})
        assert response.status_code == 200

        retrieved_data = response.json()["data"]
        assert len(retrieved_data) == 2

        for ex in retrieved_data:
            if ex["example_id"] == example_ids[0]:
                assert ex["example_text"] == "Updated Bulk Test Example 1"
                assert ex["example_int"] == 101
            elif ex["example_id"] == example_ids[1]:
                assert ex["example_text"] == "Updated Bulk Test Example 2"
                assert ex["example_int"] == 201

        # Delete them (cleanup is handled by fixture)
        response = self.client.make_request("DELETE", "/exampleList",
            params={
                "exampleIdList": f"{example_ids[0]},{example_ids[1]}",
                "updateTs": f"{update_timestamps[0]:.7f},{update_timestamps[1]:.7f}"
            })
        assert response.status_code == 200

    def test_error_scenarios(self):
        """Test various error scenarios"""
        # Try to get non-existent example
        response = self.client.make_request("GET", "/example", params={"exampleId": 99999})
        assert response.status_code == 404

        # Try to update non-existent example
        update_data = {
            "example_id": 99999,
            "example_text": "This should fail",
            "security_context": 1
        }
        response = self.client.make_request("PUT", "/example", json=update_data)
        assert response.status_code == 404

        # Try to delete non-existent example
        response = self.client.make_request("DELETE", "/example",
            params={"exampleId": 99999, "updateTs": "0.000000"})
        assert response.status_code == 404

        # Try to create example with missing required field
        invalid_data = {
            "example_int": 42,
            "security_context": 1
        }
        response = self.client.make_request("POST", "/example", json=invalid_data)
        assert response.status_code == 400