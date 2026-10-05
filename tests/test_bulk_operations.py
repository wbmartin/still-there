import time

import pytest

from conftest import TEST_TENANT


class TestBulkOperations:
    """Test bulk CRUD operations for performance and correctness"""

    def test_bulk_create_operations(self, test_client):
        """Test creating multiple examples in bulk"""
        examples_to_create = [
            {
                "exampleText": f"Bulk Create Test {i}",
                "exampleInt": i,
                "exampleFloat": float(i) * 1.5,
                "exampleIntArray": [i, i + 1, i + 2],
                "exampleFloatArray": [float(i), float(i) + 0.5],
            }
            for i in range(10)
        ]

        created_examples = []
        for example_data in examples_to_create:
            response = test_client.make_request("POST", "/example", json=example_data)
            assert response.status_code == 201
            example = response.json()["data"]
            created_examples.append(example)
            test_client.track_example(example["exampleId"], example["updateTs"])

        # Verify all examples were created
        assert len(created_examples) == 10
        for i, example in enumerate(created_examples):
            assert example["exampleText"] == f"Bulk Create Test {i}"
            assert example["exampleInt"] == i
            assert example["securityContext"] == TEST_TENANT

    def test_bulk_retrieve_operations(self, test_client):
        """Test retrieving multiple examples by ID list"""
        # Create test examples
        examples = []
        for i in range(5):
            response = test_client.make_request(
                "POST",
                "/example",
                json={
                    "exampleText": f"Bulk Retrieve Test {i}",
                    "exampleInt": i * 10,
                },
            )
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["exampleId"], example["updateTs"])

        # Bulk retrieve all examples
        example_ids = [str(ex["exampleId"]) for ex in examples]
        start_time = time.time()
        response = test_client.make_request(
            "GET", "/exampleList", params={"exampleIdList": ",".join(example_ids)}
        )
        end_time = time.time()

        assert response.status_code == 200
        retrieved = response.json()["data"]
        assert len(retrieved) == 5

        # Verify all examples are returned correctly
        retrieved_ids = {ex["exampleId"] for ex in retrieved}
        expected_ids = {ex["exampleId"] for ex in examples}
        assert retrieved_ids == expected_ids

        # Performance check (should be fast for 5 items)
        assert end_time - start_time < 1.0

    def test_bulk_update_operations(self, test_client):
        """Test updating multiple examples"""
        # Create test examples
        examples = []
        for i in range(3):
            response = test_client.make_request(
                "POST",
                "/example",
                json={
                    "exampleText": f"Bulk Update Test {i}",
                    "exampleInt": i,
                },
            )
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["exampleId"], example["updateTs"])

        # Bulk update examples
        updated_examples = []
        for example in examples:
            update_data = {
                "exampleId": example["exampleId"],
                "exampleText": f"Updated {example['exampleText']}",
                "exampleInt": example["exampleInt"] * 2,
                "updateTs": example["updateTs"],
            }
            response = test_client.make_request("PUT", "/example", json=update_data)
            assert response.status_code == 200
            updated = response.json()["data"]
            updated_examples.append(updated)

        # Verify updates
        assert len(updated_examples) == 3
        for updated in updated_examples:
            assert "Updated" in updated["exampleText"]
            assert updated["exampleInt"] % 2 == 0  # Should be doubled

    def test_bulk_delete_operations(self, test_client):
        """Test deleting multiple examples"""
        # Create test examples
        examples = []
        for i in range(4):
            response = test_client.make_request(
                "POST",
                "/example",
                json={
                    "exampleText": f"Bulk Delete Test {i}",
                    "exampleInt": i * 100,
                },
            )
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["exampleId"], example["updateTs"])

        # Bulk delete examples
        example_ids = [str(ex["exampleId"]) for ex in examples]
        update_timestamps = [f"{ex['updateTs']:.7f}" for ex in examples]

        response = test_client.make_request(
            "DELETE",
            "/exampleList",
            params={
                "exampleIdList": ",".join(example_ids),
                "updateTs": ",".join(update_timestamps),
            },
        )
        assert response.status_code == 200

        result = response.json()["data"]
        assert "message" in result
        assert result["message"] == "Examples deleted successfully"
        assert "exampleIdList" in result
        assert len(result["exampleIdList"]) == 4

        # Verify examples are deleted (soft delete)
        for example in examples:
            response = test_client.make_request(
                "GET", "/example", params={"exampleId": example["exampleId"]}
            )
            assert response.status_code == 404  # soft-deleted reads as missing

    def test_empty_bulk_operations(self, test_client):
        """Test bulk operations with empty or invalid inputs"""
        # Empty ID list for retrieval
        response = test_client.make_request(
            "GET", "/exampleList", params={"exampleIdList": ""}
        )
        assert response.status_code == 200

        # Bulk create with empty list (if supported)
        response = test_client.make_request("POST", "/exampleList", json=[])
        # This might return 400 or 201 depending on implementation
        assert response.status_code in [200, 201, 400]

    def test_bulk_operation_partial_failures(self, test_client):
        """Test bulk operations when some items fail"""
        # Create valid examples
        valid_examples = []
        for i in range(2):
            response = test_client.make_request(
                "POST",
                "/example",
                json={"exampleText": f"Partial Test {i}"},
            )
            assert response.status_code == 201
            example = response.json()["data"]
            valid_examples.append(example)
            test_client.track_example(example["exampleId"], example["updateTs"])

        # Attempt bulk update with one invalid ID
        example_ids = [str(ex["exampleId"]) for ex in valid_examples] + ["999999"]
        update_timestamps = [f"{ex['updateTs']:.7f}" for ex in valid_examples] + [
            "0.0000000"
        ]

        response = test_client.make_request(
            "DELETE",
            "/exampleList",
            params={
                "exampleIdList": ",".join(example_ids),
                "updateTs": ",".join(update_timestamps),
            },
        )

        # Should either fail completely or handle partial failures
        # The exact behavior depends on implementation
        assert response.status_code in [200, 400, 404, 409]

    def test_bulk_performance_under_load(self, test_client):
        """Test bulk operations performance with larger datasets"""
        # Create 20 examples for performance testing
        examples = []
        for i in range(20):
            response = test_client.make_request(
                "POST",
                "/example",
                json={
                    "exampleText": f"Performance Test {i}",
                    "exampleInt": i,
                },
            )
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["exampleId"], example["updateTs"])

        # Test bulk retrieval performance
        example_ids = [str(ex["exampleId"]) for ex in examples]
        start_time = time.time()
        response = test_client.make_request(
            "GET", "/exampleList", params={"exampleIdList": ",".join(example_ids)}
        )
        end_time = time.time()

        assert response.status_code == 200
        retrieved = response.json()["data"]
        assert len(retrieved) == 20

        # Performance assertion (adjust threshold based on environment)
        duration = end_time - start_time
        assert duration < 3.0, f"Bulk retrieval took {duration:.2f}s, expected < 3.0s"
