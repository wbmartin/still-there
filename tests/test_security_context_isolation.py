import pytest


class TestSecurityContextIsolation:
    """Test that security contexts properly isolate data and permissions"""

    def test_security_context_consistency(self, test_client):
        """Test that all examples are created in the user's security context"""
        # Create example in security context 1 (user's context)
        example_data_1 = {
            "example_text": "Context 1 Example",
            "example_int": 100,
            "security_context": 1
        }

        response = test_client.make_request("POST", "/example", json=example_data_1)
        assert response.status_code == 201
        example_1 = response.json()["data"]
        test_client.track_example(example_1["example_id"], example_1["update_ts"])

        # Create another example (API should ignore client-provided security_context)
        example_data_2 = {
            "example_text": "Another Example",
            "example_int": 200,
            "security_context": 999  # Invalid context, should be ignored
        }

        response = test_client.make_request("POST", "/example", json=example_data_2)
        assert response.status_code == 201
        example_2 = response.json()["data"]
        test_client.track_example(example_2["example_id"], example_2["update_ts"])

        # Verify both examples are in the user's security context (1)
        assert example_1["security_context"] == 1
        assert example_2["security_context"] == 1

        # Verify examples are accessible
        response = test_client.make_request("GET", "/example",
            params={"exampleId": example_1["example_id"]})
        assert response.status_code == 200

        response = test_client.make_request("GET", "/example",
            params={"exampleId": example_2["example_id"]})
        assert response.status_code == 200

    def test_security_context_bulk_operations(self, test_client):
        """Test that bulk operations work within the user's security context"""
        # Create multiple examples (all in user's context)
        examples = []

        for i in range(5):
            response = test_client.make_request("POST", "/example", json={
                "example_text": f"Bulk Test {i}",
                "example_int": i,
                "security_context": 1  # User's context
            })
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["example_id"], example["update_ts"])

        # Verify all examples are in context 1
        for example in examples:
            assert example["security_context"] == 1

        # Bulk retrieve all examples
        example_ids = [str(ex["example_id"]) for ex in examples]
        response = test_client.make_request("GET", "/exampleList",
            params={"exampleIdList": ",".join(example_ids)})
        assert response.status_code == 200
        retrieved = response.json()["data"]
        assert len(retrieved) == 5
        for example in retrieved:
            assert example["security_context"] == 1
            assert "Bulk Test" in example["example_text"]

        # Bulk update examples
        for example in examples:
            response = test_client.make_request("PUT", "/example", json={
                "example_id": example["example_id"],
                "example_text": f"Updated {example['example_text']}",
                "security_context": 1,
                "update_ts": example["update_ts"]
            })
            assert response.status_code == 200

        # Verify updates
        response = test_client.make_request("GET", "/exampleList",
            params={"exampleIdList": ",".join(example_ids)})
        assert response.status_code == 200
        updated = response.json()["data"]
        for example in updated:
            assert example["example_text"].startswith("Updated")

    def test_security_context_enforcement(self, test_client):
        """Test that the API enforces security context from JWT, not client input"""
        # Create example with client-provided security_context (should be ignored)
        response = test_client.make_request("POST", "/example", json={
            "example_text": "Security Test",
            "example_int": 42,
            "security_context": 999  # Invalid context, should be ignored
        })
        assert response.status_code == 201
        example = response.json()["data"]
        test_client.track_example(example["example_id"], example["update_ts"])

        # Verify the example was created in the user's actual security context (1)
        assert example["security_context"] == 1

        # Verify it's accessible
        response = test_client.make_request("GET", "/example",
            params={"exampleId": example["example_id"]})
        assert response.status_code == 200
        retrieved = response.json()["data"]
        assert retrieved["security_context"] == 1
