import time
import uuid

import pytest

from conftest import delete_user


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
            "password": "securepass123!",
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
            "password": "securepass123!",
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
            "password": "oldpass123!",
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
            "password": "deletepass123!",
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
            "password": "pass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [1],
        }

        response = test_client.make_request("POST", "/user", json=invalid_user)
        assert response.status_code == 400

        # Test creating user with missing required fields
        incomplete_user = {
            "userName": f"incomplete{suffix}",
            "firstName": "Incomplete",
            # Missing lastName, email, password
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

    def test_user_list_never_returns_totp_seed(self, test_client):
        """READ_USER is not permission to read anyone's second factor."""
        response = test_client.make_request("GET", "/userList")
        assert response.status_code == 200
        user_list = response.json()["data"]
        assert user_list, "no users to check"
        assert "totpSeed" not in response.text
        for user in user_list:
            # The flags that say whether MFA is set up are still there.
            assert "mfaSetup" in user and "mfaExempt" in user, user

    def test_duplicate_user_name_returns_409(self, test_client):
        """A taken userName is 409 on create and on rename, not 500."""
        suffix = uuid.uuid4().hex[:12]
        user_data = {
            "userName": f"dupuser{suffix}",
            "firstName": "Dup",
            "lastName": "User",
            "email": f"dupuser{suffix}@example.com",
            "password": "securepass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [],
        }
        first = test_client.make_request("POST", "/user", json=user_data)
        assert first.status_code == 201, first.text
        created = first.json()["data"]
        other_created = None
        try:
            # Same name, different e-mail: the name is what collides.
            second = test_client.make_request("POST", "/user", json={
                **user_data, "email": f"other{suffix}@example.com",
            })
            assert second.status_code == 409, second.text
            assert "already exists" in second.json()["error"]

            other = test_client.make_request("POST", "/user", json={
                **user_data,
                "userName": f"dupother{suffix}",
                "email": f"dupother{suffix}@example.com",
            })
            assert other.status_code == 201, other.text
            other_created = other.json()["data"]
            rename = test_client.make_request("PUT", "/user", json={
                "appUserId": other_created["appUserId"],
                "userName": created["userName"],
                "firstName": "Dup",
                "lastName": "User",
                "email": other_created["email"],
                "mfaExempt": False,
                "appUserGroupMembershipList": [],
                "updateTs": other_created["updateTs"],
            })
            assert rename.status_code == 409, rename.text
            assert "already exists" in rename.json()["error"]
        finally:
            delete_user(test_client, created["appUserId"])
            if other_created:
                delete_user(test_client, other_created["appUserId"])

    def test_duplicate_user_group_name_returns_409(self, test_client):
        """A group name taken in the tenant is 409; reuses a seed group."""
        existing = test_client.make_request("GET", "/appUserGroupList")
        assert existing.status_code == 200
        group_list = existing.json()["data"]
        assert group_list, "no seeded group to collide with"
        response = test_client.make_request("POST", "/appUserGroup", json={
            "groupName": group_list[0]["groupName"], "userGrantList": [],
        })
        assert response.status_code == 409, response.text
        assert "already exists" in response.json()["error"]

    def test_group_in_use_is_409_then_delete_returns_its_id(
        self, test_client
    ):
        """DELETE /appUserGroup refuses a group with a member (409, with the
        count), and once empty answers with the id it deleted."""
        suffix = uuid.uuid4().hex[:12]
        create = test_client.make_request("POST", "/appUserGroup", json={
            "groupName": f"grp-{suffix}", "userGrantList": [],
        })
        assert create.status_code == 201, create.text
        group = create.json()["data"]
        group_id = group["appUserGroupId"]
        assert group_id != 1, "need an id that is not the sentinel 1"
        member = test_client.make_request("POST", "/user", json={
            "userName": f"grpmember{suffix}",
            "firstName": "Group",
            "lastName": "Member",
            "email": f"grpmember{suffix}@example.com",
            "password": "testpass123!",
            "mfaExempt": False,
            "appUserGroupMembershipList": [group_id],
        })
        assert member.status_code == 201, member.text
        created_member = member.json()["data"]
        delete_params = {
            "appUserGroupId": str(group_id),
            "updateTs": f"{group['updateTs']:.7f}",
        }
        try:
            in_use = test_client.make_request(
                "DELETE", "/appUserGroup", params=delete_params
            )
            assert in_use.status_code == 409, in_use.text
            assert in_use.json()["error"] == (
                "cannot delete group: 1 users are assigned to it"
            )

            removed = test_client.make_request(
                "POST", "/removeUserListFromGroup", json={
                    "userIdList": [created_member["appUserId"]],
                    "appUserGroupId": group_id,
                    "updateTsList": [created_member["updateTs"]],
                },
            )
            assert removed.status_code == 200, removed.text

            deleted = test_client.make_request(
                "DELETE", "/appUserGroup", params=delete_params
            )
            assert deleted.status_code == 200, deleted.text
            assert deleted.json()["data"]["userGroupId"] == group_id
            group_id = None
        finally:
            delete_user(test_client, created_member["appUserId"])
            if group_id is not None:
                test_client.make_request(
                    "DELETE", "/appUserGroup", params=delete_params
                )

    def test_zz_cleanup_all_created_entities(self, test_client):
        """Clean up all examples and users created during tests"""
        # test_client.cleanup_examples()
        # test_client.cleanup_users()
