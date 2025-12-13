import pytest


class TestBasicUserAuthorization:
    """Test that basic users are denied access to admin-only operations"""

    def test_create_user_denied(self, basic_user_client):
        """Basic user should get 403 when trying to create a user"""
        response = basic_user_client.make_request("POST", "/user", json={
            "userName": "testuser",
            "firstName": "Test",
            "lastName": "User",
            "email": "test@example.com",
            "password": "test123!"
        })
        assert response.status_code == 403

    def test_block_user_denied(self, basic_user_client):
        """Basic user should get 403 when trying to block a user"""
        response = basic_user_client.make_request("DELETE", "/user", params={
            "userIdList": "99999",
            "updateTs": "1234567890.123"
        })
        assert response.status_code == 403

    def test_change_user_password_denied(self, basic_user_client):
        """Basic user should get 403 when trying to change another user's password"""
        response = basic_user_client.make_request("PUT", "/user", json={
            "userId": 1,
            "password": "newpass123!"
        })
        assert response.status_code == 403

    def test_create_user_group_denied(self, basic_user_client):
        """Basic user should get 403 when trying to create a user group"""
        response = basic_user_client.make_request("POST", "/userGroup", json={
            "groupName": "Test Group",
            "userGrantList": [1001]
        })
        assert response.status_code == 403

    def test_update_user_group_denied(self, basic_user_client):
        """Basic user should get 403 when trying to update a user group"""
        response = basic_user_client.make_request("PUT", "/userGroup", json={
            "userGroupId": 1,
            "groupName": "Updated Group"
        })
        assert response.status_code == 403

    def test_delete_user_group_denied(self, basic_user_client):
        """Basic user should get 403 when trying to delete a user group"""
        response = basic_user_client.make_request("DELETE", "/userGroup", params={
            "userGroupIdList": "999",
            "updateTs": "1234567890.123"
        })
        assert response.status_code == 403

    def test_add_user_to_group_denied(self, basic_user_client):
        """Basic user should get 403 when trying to add user to group"""
        response = basic_user_client.make_request("POST", "/addUserListToGroup", json={
            "userIdList": [1],
            "userGroupId": 1
        })
        assert response.status_code == 403

    def test_remove_user_from_group_denied(self, basic_user_client):
        """Basic user should get 403 when trying to remove user from group"""
        response = basic_user_client.make_request("POST", "/removeUserListFromGroup", json={
            "userIdList": [1],
            "userGroupId": 1
        })
        assert response.status_code == 403