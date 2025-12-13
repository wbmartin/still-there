import pytest


class TestSecurityContextIsolation:
    """Test that security contexts properly isolate data and permissions"""

    def test_security_context_consistency(self, test_client):
        """Test that all examples are created in the user's security context"""
        # Create example in security context 1 (user's context)
        example_data_1 = {
            "exampleText": "Context 1 Example",
            "exampleInt": 100,
            "securityContext": 1
        }

        response = test_client.make_request("POST", "/example", json=example_data_1)
        assert response.status_code == 201
        example_1 = response.json()["data"]
        test_client.track_example(example_1["exampleId"], example_1["updateTs"])

        # Create another example (API should ignore client-provided security_context)
        example_data_2 = {
            "exampleText": "Another Example",
            "exampleInt": 200,
            "securityContext": 999  # Invalid context, should be ignored
        }

        response = test_client.make_request("POST", "/example", json=example_data_2)
        assert response.status_code == 201
        example_2 = response.json()["data"]
        test_client.track_example(example_2["exampleId"], example_2["updateTs"])

        # Verify both examples are in the user's security context (1)
        assert example_1["securityContext"] == 1
        assert example_2["securityContext"] == 1

        # Verify examples are accessible
        response = test_client.make_request("GET", "/example",
            params={"exampleId": example_1["exampleId"]})
        assert response.status_code == 200

        response = test_client.make_request("GET", "/example",
            params={"exampleId": example_2["exampleId"]})
        assert response.status_code == 200

    def test_security_context_bulk_operations(self, test_client):
        """Test that bulk operations work within the user's security context"""
        # Create multiple examples (all in user's context)
        examples = []

        for i in range(5):
            response = test_client.make_request("POST", "/example", json={
                "exampleText": f"Bulk Test {i}",
                "exampleInt": i,
                "securityContext": 1  # User's context
            })
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["exampleId"], example["updateTs"])

        # Verify all examples are in context 1
        for example in examples:
            assert example["securityContext"] == 1

        # Bulk retrieve all examples
        example_ids = [str(ex["exampleId"]) for ex in examples]
        response = test_client.make_request("GET", "/exampleList",
            params={"exampleIdList": ",".join(example_ids)})
        assert response.status_code == 200
        retrieved = response.json()["data"]
        assert len(retrieved) == 5
        for example in retrieved:
            assert example["securityContext"] == 1
            assert "Bulk Test" in example["exampleText"]

        # Bulk update examples
        for example in examples:
            response = test_client.make_request("PUT", "/example", json={
                "exampleId": example["exampleId"],
                "exampleText": f"Updated {example['exampleText']}",
                "securityContext": 1,
                "updateTs": example["updateTs"]
            })
            assert response.status_code == 200

        # Verify updates
        response = test_client.make_request("GET", "/exampleList",
            params={"exampleIdList": ",".join(example_ids)})
        assert response.status_code == 200
        updated = response.json()["data"]
        for example in updated:
            assert example["exampleText"].startswith("Updated")

    def test_security_context_enforcement(self, test_client):
        """Test that the API enforces security context from JWT, not client input"""
        # Create example with client-provided security_context (should be ignored)
        response = test_client.make_request("POST", "/example", json={
            "exampleText": "Security Test",
            "exampleInt": 42,
            "securityContext": 999  # Invalid context, should be ignored
        })
        assert response.status_code == 201
        example = response.json()["data"]
        test_client.track_example(example["exampleId"], example["updateTs"])

        # Verify the example was created in the user's actual security context (1)
        assert example["securityContext"] == 1

        # Verify it's accessible
        response = test_client.make_request("GET", "/example",
            params={"exampleId": example["exampleId"]})
        assert response.status_code == 200
        retrieved = response.json()["data"]
        assert retrieved["securityContext"] == 1
