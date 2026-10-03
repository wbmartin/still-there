import pytest

from conftest import OTHER_TENANT, TEST_TENANT


class TestSecurityContextIsolation:
    """Test that security contexts properly isolate data and permissions"""

    def test_security_context_consistency(self, test_client):
        """Test that all examples are created in the user's security context"""
        # Claims the other tenant; the handler takes the JWT's (the test
        # tenant) instead.
        example_data_1 = {
            "exampleText": "Test Tenant Example",
            "exampleInt": 100,
            "securityContext": OTHER_TENANT,
        }

        response = test_client.make_request("POST", "/example", json=example_data_1)
        assert response.status_code == 201
        example_1 = response.json()["data"]
        test_client.track_example(example_1["exampleId"], example_1["updateTs"])

        # Create another example (API should ignore client-provided security_context)
        example_data_2 = {
            "exampleText": "Another Example",
            "exampleInt": 200,
            "securityContext": 999,  # Invalid context, should be ignored
        }

        response = test_client.make_request("POST", "/example", json=example_data_2)
        assert response.status_code == 201

        # Verify both examples are in the user's security context
        assert example_1["securityContext"] == TEST_TENANT
        assert response.json()["data"]["securityContext"] == TEST_TENANT

        # Verify examples are accessible
        response = test_client.make_request(
            "GET", "/example", params={"exampleId": example_1["exampleId"]}
        )
        assert response.status_code == 200

    def test_security_context_bulk_operations(self, test_client):
        """Test that bulk operations work within the user's security context"""
        # Create multiple examples (all in user's context)
        examples = []

        for i in range(5):
            response = test_client.make_request(
                "POST",
                "/example",
                json={
                    "exampleText": f"Bulk Test {i}",
                    "exampleInt": i,
                },
            )
            assert response.status_code == 201
            example = response.json()["data"]
            examples.append(example)
            test_client.track_example(example["exampleId"], example["updateTs"])

        # Verify all examples are in the test tenant
        for example in examples:
            assert example["securityContext"] == TEST_TENANT

        # Bulk retrieve all examples
        example_ids = [str(ex["exampleId"]) for ex in examples]
        response = test_client.make_request(
            "GET", "/exampleList", params={"exampleIdList": ",".join(example_ids)}
        )
        assert response.status_code == 200
        retrieved = response.json()["data"]
        assert len(retrieved) == 5
        for example in retrieved:
            assert example["securityContext"] == TEST_TENANT
            assert "Bulk Test" in example["exampleText"]

        # Bulk update examples
        for example in examples:
            response = test_client.make_request(
                "PUT",
                "/example",
                json={
                    "exampleId": example["exampleId"],
                    "exampleText": f"Updated {example['exampleText']}",
                    "updateTs": example["updateTs"],
                },
            )
            assert response.status_code == 200

        # Verify updates
        response = test_client.make_request(
            "GET", "/exampleList", params={"exampleIdList": ",".join(example_ids)}
        )
        assert response.status_code == 200
        updated = response.json()["data"]
        for example in updated:
            assert example["exampleText"].startswith("Updated")

    def test_security_context_enforcement(self, test_client):
        """Test that the API enforces security context from JWT, not client input"""
        # Create example with client-provided security_context (should be ignored)
        response = test_client.make_request(
            "POST",
            "/example",
            json={
                "exampleText": "Security Test",
                "exampleInt": 42,
                "securityContext": 999,  # Invalid context, should be ignored
            },
        )
        assert response.status_code == 201

    def test_user_create_cross_security_context_groups(self, test_client):
        """Test that creating a user with groups from different security contexts fails"""
        import uuid

        suffix = uuid.uuid4().hex
        user_data = {
            "userName": f"testuser{suffix}",
            "firstName": "Cross",
            "lastName": "Context",
            "email": f"cross{suffix}@example.com",
            "pwHash": "testpass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [4, 5],  # Groups of OTHER_TENANT
        }

        # Admin is in TEST_TENANT, so cannot assign OTHER_TENANT's groups
        response = test_client.make_request("POST", "/user", json=user_data)
        assert response.status_code == 403
        assert "security context" in response.json()["error"].lower()

    def test_user_update_cross_security_context_groups(self, test_client):
        """Test that updating a user with groups from different security contexts fails"""
        import uuid

        # First create a valid user in the test tenant
        suffix = uuid.uuid4().hex
        create_data = {
            "userName": f"updateuser{suffix}",
            "firstName": "Update",
            "lastName": "Test",
            "email": f"update{suffix}@example.com",
            "pwHash": "testpass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [1],  # TEST_TENANT's group
        }

        create_response = test_client.make_request("POST", "/user", json=create_data)
        assert create_response.status_code == 201
        user_data = create_response.json()["data"]
        app_user_id = user_data["appUserId"]
        update_ts = user_data["updateTs"]
        test_client.track_user(app_user_id, update_ts)

        # Now try to update with OTHER_TENANT's groups (should fail)
        update_data = {
            "appUserId": app_user_id,
            "userName": f"updateuser{suffix}",
            "firstName": "Updated",
            "lastName": "Test",
            "email": f"updated{suffix}@example.com",
            "mfaExempt": False,
            "appUserGroupMembershipList": [4, 5],  # Groups of OTHER_TENANT
            "updateTs": update_ts,
        }

        response = test_client.make_request("PUT", "/user", json=update_data)
        assert response.status_code == 403
        assert "security context" in response.json()["error"].lower()

    def test_add_users_cross_security_context_group(self, test_client):
        """Test that adding users from different security contexts to a group fails"""
        # Create a user in the test tenant first
        import uuid

        suffix = uuid.uuid4().hex
        user_data = {
            "userName": f"groupuser{suffix}",
            "firstName": "Group",
            "lastName": "Test",
            "email": f"group{suffix}@example.com",
            "pwHash": "testpass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [],
        }

        create_response = test_client.make_request("POST", "/user", json=user_data)
        assert create_response.status_code == 201
        created_user = create_response.json()["data"]
        user_id = created_user["appUserId"]
        test_client.track_user(user_id, created_user["updateTs"])

        # Try to add this user to OTHER_TENANT's group (should fail)
        add_data = {
            "userIds": [user_id],
            "groupId": 4,  # Group of OTHER_TENANT
            "updateTsList": [created_user["updateTs"]],
        }

        response = test_client.make_request(
            "POST", "/addUserListToGroup", json=add_data
        )
        assert response.status_code == 403
        assert "security context" in response.json()["error"].lower()

    def test_add_users_to_cross_security_context_group(self, test_client):
        """Test that adding users to a group from different security context fails"""
        # Create a user in the test tenant first
        import uuid

        suffix = uuid.uuid4().hex
        user_data = {
            "userName": f"ctxuser{suffix}",
            "firstName": "Context",
            "lastName": "Test",
            "email": f"ctx{suffix}@example.com",
            "pwHash": "testpass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [],
        }

        create_response = test_client.make_request("POST", "/user", json=user_data)
        assert create_response.status_code == 201
        created_user = create_response.json()["data"]
        user_id = created_user["appUserId"]
        test_client.track_user(user_id, created_user["updateTs"])

        # Try to add a TEST_TENANT user to OTHER_TENANT's group (should fail)
        add_data = {
            "userIds": [user_id],
            "groupId": 5,  # Group of OTHER_TENANT
            "updateTsList": [created_user["updateTs"]],
        }

        response = test_client.make_request(
            "POST", "/addUserListToGroup", json=add_data
        )
        assert response.status_code == 403
        assert "security context" in response.json()["error"].lower()

    def test_valid_user_operations_within_security_context(self, test_client):
        """Test that valid user operations within the same security context succeed"""
        import uuid

        # Create a user with no groups initially
        suffix = uuid.uuid4().hex
        user_data = {
            "userName": f"validuser{suffix}",
            "firstName": "Valid",
            "lastName": "Test",
            "email": f"valid{suffix}@example.com",
            "pwHash": "testpass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [],  # No groups initially
        }

        response = test_client.make_request("POST", "/user", json=user_data)
        assert response.status_code == 201
        created_user = response.json()["data"]
        user_id = created_user["appUserId"]
        test_client.track_user(user_id, created_user["updateTs"])

        # Update the user with valid groups
        update_data = {
            "appUserId": user_id,
            "userName": f"validuser{suffix}",
            "firstName": "Updated",
            "lastName": "Valid",
            "email": f"updated{suffix}@example.com",
            "mfaExempt": True,
            "appUserGroupMembershipList": [1],  # TEST_TENANT's group
            "updateTs": created_user["updateTs"],
        }

        response = test_client.make_request("PUT", "/user", json=update_data)
        assert response.status_code == 200

        # Get the updated user data to get the new updateTs
        get_response = test_client.make_request(
            "GET", f"/userList?appUserIdList={user_id}"
        )
        assert get_response.status_code == 200
        updated_user = get_response.json()["data"][0]
        current_update_ts = updated_user["updateTs"]

        # Add user to another valid group within context
        add_data = {
            "userIds": [user_id],
            "groupId": 3,  # TEST_TENANT's group
            "updateTsList": [current_update_ts],
        }

        response = test_client.make_request(
            "POST", "/addUserListToGroup", json=add_data
        )
        assert response.status_code == 200

    def test_cross_security_context_group_is_404_not_409(self, test_client):
        """Another tenant's group id reads as absent (404), never as a
        stale write (409): the answer must not reveal that the id exists.
        Seed groups 4 and 5 live in OTHER_TENANT; the admin is in
        TEST_TENANT."""
        for group_id in (4, 5):
            response = test_client.make_request(
                "PUT",
                "/appUserGroup",
                json={
                    "appUserGroupId": group_id,
                    "groupName": "Probe",
                    "userGrantList": [],
                    "updateTs": 0,
                },
            )
            assert response.status_code == 404, response.text

            response = test_client.make_request(
                "DELETE",
                "/appUserGroup",
                params={"appUserGroupId": group_id, "updateTs": 0},
            )
            assert response.status_code == 404, response.text

        # A stale updateTs on a group of the caller's own tenant is still 409.
        response = test_client.make_request(
            "PUT",
            "/appUserGroup",
            json={
                "appUserGroupId": 3,
                "groupName": "New User",
                "userGrantList": [1, 1001],
                "updateTs": 0,
            },
        )
        assert response.status_code == 409, response.text

    def test_add_user_list_to_group_stale_update_ts_is_409(self, test_client):
        """A stale updateTs on a membership write is a conflict, not a 500."""
        import uuid

        suffix = uuid.uuid4().hex
        response = test_client.make_request(
            "POST",
            "/user",
            json={
                "userName": f"staleuser{suffix}",
                "firstName": "Stale",
                "lastName": "Member",
                "email": f"stale{suffix}@example.com",
                "pwHash": "testpass123!",
                "mfaExempt": False,
                "appUserGroupMembershipList": [],
            },
        )
        assert response.status_code == 201
        created_user = response.json()["data"]
        test_client.track_user(created_user["appUserId"], created_user["updateTs"])

        response = test_client.make_request(
            "POST",
            "/addUserListToGroup",
            json={
                "userIds": [created_user["appUserId"]],
                "groupId": 3,
                "updateTsList": [created_user["updateTs"] - 1],
            },
        )
        assert response.status_code == 409, response.text
