import time

import pytest


class TestUserManagement:
    """Integration tests for user management operations (admin only)"""

    counter = 0

    def test_admin_create_user(self, test_client):
        """Test admin creating a new user"""
        import uuid

        suffix = uuid.uuid4().hex  # Unique hex UUID suffix
        expected_username = f"testuser{suffix}"
        new_user_data = {
            "userName": expected_username,
            "firstName": "Test",
            "lastName": "User",
            "email": f"testuser{suffix}@example.com",
            "pwHash": "securepass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [],  # No groups initially
        }

        response = test_client.make_request("POST", "/user", json=new_user_data)
        assert response.status_code == 201

        data = response.json()["data"]
        assert data["userName"] == expected_username
        assert data["firstName"] == "Test"
        assert data["lastName"] == "User"
        assert len(data["appUserGroupMembershipList"]) == 0

    def test_admin_update_user(self, test_client):
        """Test admin updating user details"""
        # Create a user
        import uuid

        suffix = uuid.uuid4().hex
        new_user_data = {
            "userName": f"Testuser{suffix}",
            "firstName": "Test",
            "lastName": "User3",
            "email": f"testuser{suffix}@example.com",
            "pwHash": "securepass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [1],
        }

        create_response = test_client.make_request("POST", "/user", json=new_user_data)
        assert create_response.status_code == 201
        user_data = create_response.json()["data"]
        app_user_id = user_data["appUserId"]
        update_ts = user_data["updateTs"]

        test_client.track_user(app_user_id, update_ts)

        # Update the user
        update_data = {
            "appUserId": app_user_id,
            "userName": f"Testuser{suffix}",  # Required for update
            "firstName": "Updated",
            "lastName": "User3",
            "email": f"updated{suffix}@example.com",
            "mfaExempt": True,
            "appUserGroupMembershipList": [1],  # Add basic group
            "updateTs": update_ts,
        }

        response = test_client.make_request("PUT", "/user", json=update_data)
        assert response.status_code == 200

        # Verify update
        get_response = test_client.make_request(
            "GET", f"/userList?appUserIdList={app_user_id}"
        )
        assert get_response.status_code == 200

        updated_list = get_response.json()["data"]
        assert len(updated_list) == 1
        updated_data = updated_list[0]
        assert updated_data["firstName"] == "Updated"
        assert updated_data["email"] == f"updated{suffix}@example.com"
        assert updated_data["mfaExempt"] == True
        assert 1 in updated_data["appUserGroupMembershipList"]

    def test_admin_change_user_password(self, test_client):
        """Test admin changing user password"""
        # Create a user
        import uuid

        suffix = uuid.uuid4().hex
        new_user_data = {
            "userName": f"Testuser{suffix}",
            "firstName": "Password",
            "lastName": "Test",
            "email": f"passwordtest{suffix}@example.com",
            "pwHash": "oldpass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [1],
        }

        create_response = test_client.make_request("POST", "/user", json=new_user_data)
        assert create_response.status_code == 201
        app_user_id = create_response.json()["data"]["appUserId"]
        update_ts = create_response.json()["data"]["updateTs"]

        test_client.track_user(app_user_id, update_ts)

        # Change password
        password_data = {"appUserId": app_user_id, "newPassword": "newpass456!"}

        response = test_client.make_request(
            "POST", "/changeOtherUserPassword", json=password_data
        )
        assert response.status_code == 200

        data = response.json()["data"]
        assert "message" in data

    def test_admin_delete_user(self, test_client):
        """Test admin deleting a user (blocking is test_user_block.py)"""
        # Create a user
        import uuid

        suffix = uuid.uuid4().hex
        new_user_data = {
            "userName": f"Testuser_delete{suffix}",
            "firstName": "Delete",
            "lastName": "Test",
            "email": f"deletetest{suffix}@example.com",
            "pwHash": "deletepass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [],
        }

        create_response = test_client.make_request("POST", "/user", json=new_user_data)
        assert create_response.status_code == 201
        user_data = create_response.json()["data"]
        app_user_id = user_data["appUserId"]
        update_ts = user_data["updateTs"]

        # Delete the user
        delete_data = {"appUserId": str(app_user_id), "updateTs": f"{update_ts:.7f}"}

        response = test_client.make_request("DELETE", "/user", params=delete_data)
        assert response.status_code == 200

        # Verify user is soft deleted
        get_response = test_client.make_request(
            "GET", "/userList?appUserIdList=" + str(app_user_id)
        )
        assert get_response.status_code == 200
        user_list = get_response.json()["data"]
        assert len(user_list) == 0  # Soft deleted users not returned

    def test_user_management_validation(self, test_client):
        """Test validation for user management operations"""
        import uuid

        suffix = uuid.uuid4().hex
        # Test creating user with invalid username
        invalid_user = {
            "userName": f"invalid user{suffix}",
            "firstName": "Invalid",
            "lastName": "User",
            "email": f"invalid{suffix}@example.com",
            "pwHash": "pass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [1],
        }

        response = test_client.make_request("POST", "/user", json=invalid_user)
        assert response.status_code == 400

        # Test creating user with missing required fields
        incomplete_user = {
            "userName": f"incomplete{suffix}",
            "firstName": "Incomplete",
            # Missing lastName, email, pwHash
        }

        response = test_client.make_request("POST", "/user", json=incomplete_user)
        assert response.status_code == 400

        # Test getting non-existent user
        response = test_client.make_request(
            "GET", "/userList", params={"appUserIdList": "999999"}
        )
        assert response.status_code == 200
        user_list = response.json()["data"]
        assert len(user_list) == 0

    def test_zz_cleanup_all_created_entities(self, test_client):
        """Clean up all examples and users created during tests"""
        # test_client.cleanup_examples()
        # test_client.cleanup_users()
