import pytest
import time


class TestBulkOperations:
    """Test bulk CRUD operations for performance and correctness"""

    def test_bulk_create_operations(self, test_client):
        """Test creating multiple examples in bulk"""
        examples_to_create = [
            {
                "example_text": f"Bulk Create Test {i}",
                "example_int": i,
                "example_float": float(i) * 1.5,
                "example_int_array": [i, i+1, i+2],
                "example_float_array": [float(i), float(i)+0.5],
                "security_context": 1
            }
            for i in range(10)
        ]

        created_examples = []
        for example_data in examples_to_create:
            response = test_client.make_request("POST", "/example", json=example_data)
            assert response.status_code == 201
            example = response.json()["data"]
            created_examples.append(example)
            test_client.track_example(example["example_id"], example["update_ts"])

        # Verify all examples were created
        assert len(created_examples) == 10
        for i, example in enumerate(created_examples):
            assert example["example_text"] == f"Bulk Create Test {i}"
            assert example["example_int"] == i
            assert example["security_context"] == 1

    def test_bulk_retrieve_operations(self, test_client):
        """Test retrieving multiple examples by ID list"""
        # Create test examples
        examples = []
        for i in range(5):
            response = test_client.make_request("POST", "/example", json={
                "example_text": f"Bulk Retrieve Test {i}",
                "example_int": i * 10,
                "security_context": 1
            })
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["example_id"], example["update_ts"])

        # Bulk retrieve all examples
        example_ids = [str(ex["example_id"]) for ex in examples]
        start_time = time.time()
        response = test_client.make_request("GET", "/exampleList",
            params={"exampleIdList": ",".join(example_ids)})
        end_time = time.time()

        assert response.status_code == 200
        retrieved = response.json()["data"]
        assert len(retrieved) == 5

        # Verify all examples are returned correctly
        retrieved_ids = {ex["example_id"] for ex in retrieved}
        expected_ids = {ex["example_id"] for ex in examples}
        assert retrieved_ids == expected_ids

        # Performance check (should be fast for 5 items)
        assert end_time - start_time < 1.0

    def test_bulk_update_operations(self, test_client):
        """Test updating multiple examples"""
        # Create test examples
        examples = []
        for i in range(3):
            response = test_client.make_request("POST", "/example", json={
                "example_text": f"Bulk Update Test {i}",
                "example_int": i,
                "security_context": 1
            })
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["example_id"], example["update_ts"])

        # Bulk update examples
        updated_examples = []
        for example in examples:
            update_data = {
                "example_id": example["example_id"],
                "example_text": f"Updated {example['example_text']}",
                "example_int": example["example_int"] * 2,
                "security_context": 1,
                "update_ts": example["update_ts"]
            }
            response = test_client.make_request("PUT", "/example", json=update_data)
            assert response.status_code == 200
            updated = response.json()["data"]
            updated_examples.append(updated)

        # Verify updates
        assert len(updated_examples) == 3
        for updated in updated_examples:
            assert "Updated" in updated["example_text"]
            assert updated["example_int"] % 2 == 0  # Should be doubled

    def test_bulk_delete_operations(self, test_client):
        """Test deleting multiple examples"""
        # Create test examples
        examples = []
        for i in range(4):
            response = test_client.make_request("POST", "/example", json={
                "example_text": f"Bulk Delete Test {i}",
                "example_int": i * 100,
                "security_context": 1
            })
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["example_id"], example["update_ts"])

        # Bulk delete examples
        example_ids = [str(ex["example_id"]) for ex in examples]
        update_timestamps = [f"{ex['update_ts']:.7f}" for ex in examples]

        response = test_client.make_request("DELETE", "/exampleList", params={
            "exampleIdList": ",".join(example_ids),
            "updateTs": ",".join(update_timestamps)
        })
        assert response.status_code == 200

        result = response.json()["data"]
        assert "message" in result
        assert result["message"] == "Examples deleted successfully"
        assert "example_ids" in result
        assert len(result["example_ids"]) == 4

        # Verify examples are deleted (soft delete)
        for example in examples:
            response = test_client.make_request("GET", "/example",
                params={"exampleId": example["example_id"]})
            assert response.status_code == 410  # Gone

    def test_empty_bulk_operations(self, test_client):
        """Test bulk operations with empty or invalid inputs"""
        # Empty ID list for retrieval
        response = test_client.make_request("GET", "/exampleList",
            params={"exampleIdList": ""})
        assert response.status_code == 400

        # Bulk create with empty list (if supported)
        response = test_client.make_request("POST", "/exampleList", json=[])
        # This might return 400 or 201 depending on implementation
        assert response.status_code in [200, 201, 400]

    def test_bulk_operation_partial_failures(self, test_client):
        """Test bulk operations when some items fail"""
        # Create valid examples
        valid_examples = []
        for i in range(2):
            response = test_client.make_request("POST", "/example", json={
                "example_text": f"Partial Test {i}",
                "security_context": 1
            })
            assert response.status_code == 201
            example = response.json()["data"]
            valid_examples.append(example)
            test_client.track_example(example["example_id"], example["update_ts"])

        # Attempt bulk update with one invalid ID
        example_ids = [str(ex["example_id"]) for ex in valid_examples] + ["999999"]
        update_timestamps = [f"{ex['update_ts']:.7f}" for ex in valid_examples] + ["0.0000000"]

        response = test_client.make_request("DELETE", "/exampleList", params={
            "exampleIdList": ",".join(example_ids),
            "updateTs": ",".join(update_timestamps)
        })

        # Should either fail completely or handle partial failures
        # The exact behavior depends on implementation
        assert response.status_code in [200, 400, 409, 410]

    def test_bulk_performance_under_load(self, test_client):
        """Test bulk operations performance with larger datasets"""
        # Create 20 examples for performance testing
        examples = []
        for i in range(20):
            response = test_client.make_request("POST", "/example", json={
                "example_text": f"Performance Test {i}",
                "example_int": i,
                "security_context": 1
            })
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["example_id"], example["update_ts"])

        # Test bulk retrieval performance
        example_ids = [str(ex["example_id"]) for ex in examples]
        start_time = time.time()
        response = test_client.make_request("GET", "/exampleList",
            params={"exampleIdList": ",".join(example_ids)})
        end_time = time.time()

        assert response.status_code == 200
        retrieved = response.json()["data"]
        assert len(retrieved) == 20

        # Performance assertion (adjust threshold based on environment)
        duration = end_time - start_time
        assert duration < 3.0, f"Bulk retrieval took {duration:.2f}s, expected < 3.0s"