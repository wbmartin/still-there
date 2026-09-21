package repository

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/wbmartin/starter-api/internal/model"
	"testing"
	"time"
)

func TestUserRepository(t *testing.T) {
	suite.Run(t, &UserRepositoryTestSuite{})
}

type UserRepositoryTestSuite struct {
	suite.Suite
	db      *pgxpool.Pool
	repo    *UserRepository
	cleanup func()
}

func (s *UserRepositoryTestSuite) SetupSuite() {
	db := connectTestDB(s.T())
	s.db = db
	s.repo = NewUserRepository(db)

	s.cleanup = func() {
		// Clean test users (avoid deleting the admin user)
		_, _ = db.Exec(
			context.Background(),
			"DELETE FROM app_user WHERE user_name LIKE 'Test%' OR user_name LIKE 'RepoTest%'",
		)
	}
}

func (s *UserRepositoryTestSuite) TearDownSuite() {
	s.cleanup()
	s.db.Close()
}

func (s *UserRepositoryTestSuite) TestNewUserRepository() {
	// Test constructor
	repo := NewUserRepository(s.db)
	s.NotNil(repo)
	s.Equal(s.db, repo.db)
}

func (s *UserRepositoryTestSuite) TestRetrieveUserByUserName() {
	// Create a test user first
	testUserName := "TestUser_" + time.Now().
		Format("20060102150405")
		// Underscores are now allowed
	testUser := &model.AppUserModel{
		UserName:                   testUserName,
		PwHash:                     "$2a$10$example.hash.here",
		FirstName:                  "Test",
		LastName:                   "User",
		Email:                      "test@example.com",
		SecurityContext:            1,
		AppUserGroupMembershipList: []int{1}, // Assuming group 1 exists
		MFAExempt:                  false,
		CreateTS:                   1234567890.123,
		CreateUserID:               1,
		UpdateTS:                   1234567890.123,
		UpdateUserID:               1,
	}

	created, err := s.repo.CreateUser(context.Background(), testUser, 1)
	s.NoError(err)
	s.NotNil(created)

	// Now retrieve the user by username
	user, err := s.repo.RetrieveUserByUserName(
		context.Background(),
		testUserName,
	)
	s.NoError(err)
	s.NotNil(user)
	s.Equal(testUserName, user.UserName)
	s.Equal("User", user.LastName)
	s.Equal("test@example.com", user.Email)
	s.Equal(1, user.SecurityContext)
	s.False(user.MFAExempt)
	s.NotNil(user.AppUserGroupMembershipList)
	s.True(len(user.AppUserGroupMembershipList) > 0)
	s.Nil(user.DeleteTS)
	s.Nil(user.DeleteUserID)
}

func (s *UserRepositoryTestSuite) TestRetrieveUserByUserNameNotFound() {
	// Test with non-existent username
	user, err := s.repo.RetrieveUserByUserName(
		context.Background(),
		"nonexistentuser",
	)
	s.Error(err)
	s.Nil(user)
	s.Contains(err.Error(), "user not found")
}

func (s *UserRepositoryTestSuite) TestGetGrantsForGroups() {
	// First, check what groups actually exist
	allGroups, err := s.db.Query(
		context.Background(),
		"SELECT app_user_group_id FROM app_user_group WHERE delete_user_id IS NULL ORDER BY app_user_group_id",
	)
	s.NoError(err)
	defer allGroups.Close()

	var existingGroupIDList []int
	for allGroups.Next() {
		var id int
		err := allGroups.Scan(&id)
		s.NoError(err)
		existingGroupIDList = append(existingGroupIDList, id)
	}

	if len(existingGroupIDList) == 0 {
		// No groups in database, skip this test
		s.T().Skip("No user groups found in database - skipping grants test")
		return
	}

	// Test retrieving grants for existing groups
	grantsMap, err := s.repo.GetGrantsForGroups(
		context.Background(),
		existingGroupIDList,
	)
	s.NoError(err)
	s.NotNil(grantsMap)

	// Should have at least one security context with grants
	totalGrants := 0
	for _, grantList := range grantsMap {
		totalGrants += len(grantList)
	}
	s.True(totalGrants > 0, "Should have at least one grant")
}

func (s *UserRepositoryTestSuite) TestGetGrantsForGroupsEmpty() {
	// Test with empty slice
	grantsMap, err := s.repo.GetGrantsForGroups(context.Background(), []int{})
	s.NoError(err)
	s.NotNil(grantsMap)
	s.Equal(0, len(grantsMap))
}

func (s *UserRepositoryTestSuite) TestGetGrantsForGroupsNonExistent() {
	// Test with non-existent group IDs
	grantsMap, err := s.repo.GetGrantsForGroups(
		context.Background(),
		[]int{99999, 88888},
	)
	s.NoError(err)
	s.Equal(
		0,
		len(grantsMap),
	) // Should return empty map for non-existent groups
}

func (s *UserRepositoryTestSuite) TestCreateUser() {
	// Create a test user
	user := &model.AppUserModel{
		UserName:                   "RepoTestUser",
		PwHash:                     "$2a$10$test.hash.for.testing.purposes.only",
		FirstName:                  "Test",
		LastName:                   "User",
		Email:                      "test@example.com",
		SecurityContext:            1,
		AppUserGroupMembershipList: []int{1}, // Assuming group 1 exists
		MFAExempt:                  false,
		CreateTS:                   1234567890.123,
		CreateUserID:               1,
		UpdateTS:                   1234567890.123,
		UpdateUserID:               1,
	}

	created, err := s.repo.CreateUser(context.Background(), user, 1)
	s.NoError(err)
	s.NotNil(created)
	s.NotZero(created.AppUserID)
	s.Equal("RepoTestUser", created.UserName)
	s.Equal("Test", created.FirstName)
	s.Equal("User", created.LastName)
	s.Equal("test@example.com", created.Email)
}

func (s *UserRepositoryTestSuite) TestCreateUserInvalidUsername() {
	// Create a test user with invalid username
	user := &model.AppUserModel{
		UserName:                   "invalid-user!",
		PwHash:                     "$2a$10$test.hash.for.testing.purposes.only",
		FirstName:                  "Test",
		LastName:                   "User",
		Email:                      "test@example.com",
		SecurityContext:            1,
		AppUserGroupMembershipList: []int{1},
		MFAExempt:                  false,
		CreateTS:                   1234567890.123,
		CreateUserID:               1,
		UpdateTS:                   1234567890.123,
		UpdateUserID:               1,
	}

	created, err := s.repo.CreateUser(context.Background(), user, 1)
	s.Error(err)
	s.Nil(created)
	s.Contains(
		err.Error(),
		"username must contain only alphanumeric characters, hyphens, and underscores",
	)
}

func (s *UserRepositoryTestSuite) TestUpdateUserPassword() {
	// First create a test user
	user := &model.AppUserModel{
		UserName:                   "RepoTestPasswordUser",
		PwHash:                     "$2a$10$original.hash",
		FirstName:                  "Password",
		LastName:                   "Test",
		Email:                      "password@example.com",
		SecurityContext:            1,
		AppUserGroupMembershipList: []int{1},
		MFAExempt:                  false,
		CreateTS:                   1234567890.123,
		CreateUserID:               1,
		UpdateTS:                   1234567890.123,
		UpdateUserID:               1,
	}

	created, _ := s.repo.CreateUser(context.Background(), user, 1)

	// Update password
	newHash := "$2a$10$new.hash.for.testing"
	err := s.repo.UpdateUserPassword(
		context.Background(),
		created.AppUserID,
		newHash,
		1234567891.456,
		2,
	)
	s.NoError(err)

	// Verify password was updated
	var updatedHash string
	err = s.db.QueryRow(context.Background(), "SELECT pw_hash FROM app_user WHERE app_user_id = $1", created.AppUserID).
		Scan(&updatedHash)
	s.NoError(err)
	s.Equal(newHash, updatedHash)
}

func (s *UserRepositoryTestSuite) TestUpdateUserPasswordNotFound() {
	// Test updating password for non-existent user
	err := s.repo.UpdateUserPassword(
		context.Background(),
		99999,
		"newhash",
		1234567890.123,
		1,
	)
	s.Error(err)
	s.Contains(err.Error(), "user not found")
}

func (s *UserRepositoryTestSuite) TestRemoveDuplicates() {
	// Test the removeDuplicates utility function
	input := []int{1, 2, 2, 3, 1, 4, 4, 4}
	expected := []int{1, 2, 3, 4}

	result := removeDuplicates(input)
	s.Equal(expected, result)

	// Test with empty slice
	empty := removeDuplicates([]int{})
	s.Equal([]int{}, empty)

	// Test with no duplicates
	noDups := []int{1, 2, 3, 4}
	result2 := removeDuplicates(noDups)
	s.Equal(noDups, result2)
}

func (s *UserRepositoryTestSuite) TestRetrieveUserList() {
	// Create test users
	user1 := &model.AppUserModel{
		UserName:                   "RepoTestListUser1",
		PwHash:                     "$2a$10$test.hash",
		FirstName:                  "List",
		LastName:                   "User1",
		Email:                      "list1@example.com",
		SecurityContext:            1,
		AppUserGroupMembershipList: []int{1},
		MFAExempt:                  false,
		CreateTS:                   1234567890.123,
		CreateUserID:               1,
		UpdateTS:                   1234567890.123,
		UpdateUserID:               1,
	}

	created1, _ := s.repo.CreateUser(context.Background(), user1, 1)

	// Test retrieve all users in security context 1
	users, err := s.repo.RetrieveUserList(context.Background(), 1, nil, 0)
	s.NoError(err)
	s.NotNil(users)
	s.True(len(users) >= 1) // At least our test user

	// Check that our user is in the list
	userNames := make(map[string]bool)
	for _, u := range users {
		userNames[u.UserName] = true
	}
	s.True(userNames["RepoTestListUser1"])

	// Test retrieve specific user by ID
	idList := []int{created1.AppUserID}
	specificUsers, err := s.repo.RetrieveUserList(
		context.Background(),
		1,
		idList,
		0,
	)
	s.NoError(err)
	s.Len(specificUsers, 1)

	// Test with update_ts filter (should return no users if filter is high)
	futureTs := 2000000000.0
	filteredUsers, err := s.repo.RetrieveUserList(
		context.Background(),
		1,
		nil,
		futureTs,
	)
	s.NoError(err)
	s.Len(filteredUsers, 0)
}

func (s *UserRepositoryTestSuite) TestUpdateUserTOTPSeed() {
	// Create test user
	user := &model.AppUserModel{
		UserName:                   "RepoTestTOTPUser",
		PwHash:                     "$2a$10$test.hash",
		FirstName:                  "TOTP",
		LastName:                   "User",
		Email:                      "totp@example.com",
		SecurityContext:            1,
		AppUserGroupMembershipList: []int{1},
		MFAExempt:                  false,
		CreateTS:                   1234567890.123,
		CreateUserID:               1,
		UpdateTS:                   1234567890.123,
		UpdateUserID:               1,
	}

	created, _ := s.repo.CreateUser(context.Background(), user, 1)

	// Update TOTP seed
	newSeed := "JBSWY3DPEHPK3PXP"
	expectedUpdateTS := created.UpdateTS
	newUpdateTS, err := s.repo.UpdateUserTOTPSeed(
		context.Background(),
		created.AppUserID,
		newSeed,
		expectedUpdateTS,
		2,
	)
	s.NoError(err)
	s.Greater(newUpdateTS, expectedUpdateTS)

	// Verify seed was updated
	var updatedSeed string
	err = s.db.QueryRow(context.Background(), "SELECT totp_seed FROM app_user WHERE app_user_id = $1", created.AppUserID).
		Scan(&updatedSeed)
	s.NoError(err)
	s.Equal(newSeed, updatedSeed)

	// Test update with stale timestamp (should fail)
	_, err = s.repo.UpdateUserTOTPSeed(
		context.Background(),
		created.AppUserID,
		"newseed",
		expectedUpdateTS,
		2,
	)
	s.Error(err)
	s.Contains(err.Error(), "user not found or concurrent modification")

	// Test update non-existent user
	_, err = s.repo.UpdateUserTOTPSeed(
		context.Background(),
		99999,
		"seed",
		1234567890.123,
		2,
	)
	s.Error(err)
	s.Contains(err.Error(), "user not found or concurrent modification")
}

func (s *UserRepositoryTestSuite) TestUpdateUserMFAExempt() {
	// Create test user
	user := &model.AppUserModel{
		UserName:                   "RepoTestMFAUser",
		PwHash:                     "$2a$10$test.hash",
		FirstName:                  "MFA",
		LastName:                   "User",
		Email:                      "mfa@example.com",
		SecurityContext:            1,
		AppUserGroupMembershipList: []int{1},
		MFAExempt:                  false,
		CreateTS:                   1234567890.123,
		CreateUserID:               1,
		UpdateTS:                   1234567890.123,
		UpdateUserID:               1,
	}

	created, _ := s.repo.CreateUser(context.Background(), user, 1)

	// Update MFA exempt to true
	expectedUpdateTS := created.UpdateTS
	newUpdateTS, err := s.repo.UpdateUserMFAExempt(
		context.Background(),
		created.AppUserID,
		true,
		expectedUpdateTS,
		2,
	)
	s.NoError(err)
	s.Greater(newUpdateTS, expectedUpdateTS)

	// Verify MFA exempt was updated
	var mfaExempt bool
	err = s.db.QueryRow(context.Background(), "SELECT mfa_exempt FROM app_user WHERE app_user_id = $1", created.AppUserID).
		Scan(&mfaExempt)
	s.NoError(err)
	s.True(mfaExempt)

	// Test update with stale timestamp
	_, err = s.repo.UpdateUserMFAExempt(
		context.Background(),
		created.AppUserID,
		false,
		expectedUpdateTS,
		2,
	)
	s.Error(err)
	s.Contains(err.Error(), "user not found or concurrent modification")
}

func (s *UserRepositoryTestSuite) TestUpdateUser() {
	// Create test user
	user := &model.AppUserModel{
		UserName:                   "RepoTestUpdateUser",
		PwHash:                     "$2a$10$test.hash",
		FirstName:                  "Update",
		LastName:                   "User",
		Email:                      "update@example.com",
		SecurityContext:            1,
		AppUserGroupMembershipList: []int{1},
		MFAExempt:                  false,
		CreateTS:                   1234567890.123,
		CreateUserID:               1,
		UpdateTS:                   1234567890.123,
		UpdateUserID:               1,
	}

	created, _ := s.repo.CreateUser(context.Background(), user, 1)

	// Update user details
	updatedUser := &model.AppUserModel{
		AppUserID:                  created.AppUserID,
		UserName:                   "RepoTestUpdatedUser",
		FirstName:                  "Updated",
		LastName:                   "User",
		Email:                      "updated@example.com",
		AppUserGroupMembershipList: []int{1, 2},
		MFAExempt:                  true,
		UpdateTS:                   1234567891.456,
		UpdateUserID:               2,
	}

	err := s.repo.UpdateUser(
		context.Background(),
		updatedUser,
		created.UpdateTS,
		1,
	)
	s.NoError(err)

	// Verify update
	retrieved, err := s.repo.RetrieveUserByUserName(
		context.Background(),
		"RepoTestUpdatedUser",
	)
	s.NoError(err)
	s.Equal("Updated", retrieved.FirstName)
	s.Equal("updated@example.com", retrieved.Email)
	s.True(retrieved.MFAExempt)
	s.Contains(retrieved.AppUserGroupMembershipList, 2)

	// Test update with stale timestamp
	err = s.repo.UpdateUser(
		context.Background(),
		updatedUser,
		created.UpdateTS,
		1,
	)
	s.Error(err)
	s.Equal(ErrConflict, err)

	// Test update non-existent user
	nonExistent := &model.AppUserModel{
		AppUserID: 99999,
		UserName:  "nonexistent",
		UpdateTS:  1234567890.123,
	}
	err = s.repo.UpdateUser(
		context.Background(),
		nonExistent,
		1234567890.123,
		1,
	)
	s.Error(err)
	s.Equal(ErrNotFound, err)
}

func (s *UserRepositoryTestSuite) TestDeleteUser() {
	// Create test user
	user := &model.AppUserModel{
		UserName:                   "RepoTestDeleteUser",
		PwHash:                     "$2a$10$test.hash",
		FirstName:                  "Delete",
		LastName:                   "User",
		Email:                      "delete@example.com",
		SecurityContext:            1,
		AppUserGroupMembershipList: []int{1},
		MFAExempt:                  false,
		CreateTS:                   1234567890.123,
		CreateUserID:               1,
		UpdateTS:                   1234567890.123,
		UpdateUserID:               1,
	}

	created, _ := s.repo.CreateUser(context.Background(), user, 1)

	// Delete user
	deleteTS := 1234567892.789
	deletedID, err := s.repo.DeleteUser(
		context.Background(),
		created.AppUserID,
		1,
		created.UpdateTS,
		deleteTS,
		2,
	)
	s.NoError(err)
	s.Equal(created.AppUserID, deletedID)

	// Verify user is soft deleted
	retrieved, err := s.repo.RetrieveUserByUserName(
		context.Background(),
		"RepoTestDeleteUser",
	)
	s.Error(err)
	s.Nil(retrieved)

	// Test delete with stale timestamp
	err = s.repo.UpdateUserPassword(
		context.Background(),
		created.AppUserID,
		"newhash",
		1234567890.123,
		2,
	)
	s.Error(err) // Should fail because user is deleted

	// Test delete non-existent user
	_, err = s.repo.DeleteUser(
		context.Background(),
		99999,
		1,
		1234567890.123,
		deleteTS,
		2,
	)
	s.Error(err)
	s.Equal(ErrNotFound, err)
}

func (s *UserRepositoryTestSuite) TestAddUserListToGroup() {
	// Create test user
	user1 := &model.AppUserModel{
		UserName:                   "RepoTestGroupUser1",
		PwHash:                     "$2a$10$test.hash",
		FirstName:                  "Group",
		LastName:                   "User1",
		Email:                      "group1@example.com",
		SecurityContext:            1,
		AppUserGroupMembershipList: []int{}, // Start with no groups
		MFAExempt:                  false,
		CreateTS:                   CurrentTimestamp(),
		CreateUserID:               1,
		UpdateTS:                   CurrentTimestamp(),
		UpdateUserID:               1,
	}

	created1, _ := s.repo.CreateUser(context.Background(), user1, 1)

	// Verify user starts with empty group membership
	var groups1 []int
	_ = s.db.QueryRow(context.Background(), "SELECT app_user_group_membership_list FROM app_user WHERE app_user_id = $1", created1.AppUserID).
		Scan(&groups1)
	s.Equal([]int{}, groups1)

	// Use the update timestamp from the created user object
	testGroupID := 3 // Use group 3 which should exist in security context 1
	userIDs := []int{created1.AppUserID}
	updateTSList := []float64{created1.UpdateTS}

	updatedUsers, err := s.repo.AddUserListToGroup(
		context.Background(),
		userIDs,
		testGroupID,
		updateTSList,
		2,
		1,
	)
	s.NoError(err)
	s.Len(updatedUsers, 1)

	// Verify user is in group
	s.Contains(updatedUsers[0].AppUserGroupMembershipList, testGroupID)
}

func (s *UserRepositoryTestSuite) TestRemoveUserListFromGroup() {
	// Create test user with group membership
	user := &model.AppUserModel{
		UserName:                   "RepoTestRemoveUser",
		PwHash:                     "$2a$10$test.hash",
		FirstName:                  "Remove",
		LastName:                   "User",
		Email:                      "remove@example.com",
		SecurityContext:            1,
		AppUserGroupMembershipList: []int{1, 2}, // Start with groups
		MFAExempt:                  false,
		CreateTS:                   1234567890.123,
		CreateUserID:               1,
		UpdateTS:                   1234567890.123,
		UpdateUserID:               1,
	}

	created, _ := s.repo.CreateUser(context.Background(), user, 1)

	// Remove user from group 1
	updatedUsers, err := s.repo.RemoveUserListFromGroup(
		context.Background(),
		[]int{created.AppUserID},
		1,
		[]float64{created.UpdateTS},
		2,
		1,
	)
	s.NoError(err)
	s.Len(updatedUsers, 1)

	// Verify user is no longer in group 1 but still in group 2
	u := updatedUsers[0]
	s.NotContains(u.AppUserGroupMembershipList, 1)
	s.Contains(u.AppUserGroupMembershipList, 2)

	// Test removing from group not in membership (should succeed)
	updatedUsers2, err := s.repo.RemoveUserListFromGroup(
		context.Background(),
		[]int{created.AppUserID},
		3,
		[]float64{u.UpdateTS},
		2,
		1,
	)
	s.NoError(err)
	s.Len(updatedUsers2, 1)
}

func (s *UserRepositoryTestSuite) TestCurrentTimestamp() {
	ts1 := CurrentTimestamp()
	time.Sleep(10 * time.Millisecond) // Longer delay to ensure difference
	ts2 := CurrentTimestamp()

	s.True(ts2 > ts1, "Second timestamp should be greater than first")
	s.True(ts1 > 0, "Timestamp should be positive")
}
