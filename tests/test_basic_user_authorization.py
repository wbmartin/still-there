import pytest


class TestBasicUserAuthorization:
    """Test that basic users are denied access to admin-only operations"""

    def test_create_user_denied(self, basic_user_client):
        """Basic user should get 403 when trying to create a user"""
        import uuid
        suffix = uuid.uuid4().hex
        response = basic_user_client.make_request("POST", "/user", json={
            "userName": f"testuser{suffix}",
            "firstName": "Test",
            "lastName": "User",
            "email": f"test{suffix}@example.com",
            "password": "test123!"
        })
        assert response.status_code == 403

    def test_block_user_denied(self, basic_user_client):
        """Basic user should get 403 when trying to block a user"""
        response = basic_user_client.make_request("DELETE", "/user", params={
            "appUserId": "99999",
            "updateTs": "1234567890.123"
        })
        assert response.status_code == 403

    def test_change_user_password_denied(self, basic_user_client):
        """Basic user should get 403 when trying to change another user's password"""
        response = basic_user_client.make_request("PUT", "/user", json={
            "appUserId": 1,
            "newPassword": "newpass123!"
        })
        assert response.status_code == 403

    def test_create_user_group_denied(self, basic_user_client):
        """Basic user should get 403 when trying to create a user group"""
        response = basic_user_client.make_request("POST", "/appUserGroup", json={
            "groupName": "Test Group",
            "userGrantList": [1001]
        })
        assert response.status_code == 403

    def test_update_user_group_denied(self, basic_user_client):
        """Basic user should get 403 when trying to update a user group"""
        response = basic_user_client.make_request("PUT", "/appUserGroup", json={
            "appUserGroupId": 1,
            "groupName": "Updated Group"
        })
        assert response.status_code == 403

    def test_delete_user_group_denied(self, basic_user_client):
        """Basic user should get 403 when trying to delete a user group"""
        response = basic_user_client.make_request("DELETE", "/appUserGroup", params={
            "appUserGroupId": "999",
            "updateTs": "1234567890.123"
        })
        assert response.status_code == 403

    def test_add_user_to_group_denied(self, basic_user_client):
        """Basic user should get 403 when trying to add user to group"""
        # A body that would pass decoding and the length check, so the
        # 403 can only be the grant check (B089).
        response = basic_user_client.make_request("POST", "/addUserListToGroup", json={
            "userIdList": [1],
            "appUserGroupId": 1,
            "updateTsList": [0]
        })
        assert response.status_code == 403
        assert 'grant "UPDATE_USER_GROUP" required' in response.json()["error"]

    def test_remove_user_from_group_denied(self, basic_user_client):
        """Basic user should get 403 when trying to remove user from group"""
        # A body that would pass decoding and the length check, so the
        # 403 can only be the grant check (B089).
        response = basic_user_client.make_request("POST", "/removeUserListFromGroup", json={
            "userIdList": [1],
            "appUserGroupId": 1,
            "updateTsList": [0]
        })
        assert response.status_code == 403
        assert 'grant "UPDATE_USER_GROUP" required' in response.json()["error"]