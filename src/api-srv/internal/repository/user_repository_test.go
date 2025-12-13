package repository

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"testing"
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
	// Load configuration from the same file used by the application
	config, err := app_lib.LoadConfig("../../../../etc/loc-api-srv.env")
	s.Require().NoError(err)

	// Connect to the database using the configured URL
	db, err := pgxpool.New(context.Background(), config.DatabaseURL)
	s.Require().NoError(err)
	s.db = db
	s.repo = NewUserRepository(db)

	s.cleanup = func() {
		// Clean test users (avoid deleting the admin user)
		_, _ = db.Exec(context.Background(),
			"DELETE FROM app_user WHERE user_name LIKE 'Test%' OR user_name LIKE 'RepoTest%'")
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
	testUser := &model.AppUserModel{
		UserName:                "RepoTestRetrieveUser",
		PwHash:                  "$2a$10$test.hash.for.testing.purposes.only",
		FirstName:               "Retrieve",
		LastName:                "Test",
		Email:                   "retrieve@example.com",
		SecurityContext:         1,
		UserGroupMembershipList: []int{1}, // Assuming group 1 exists
		MFAExempt:               false,
		CreateTS:                1234567890.123,
		CreateUserID:            1,
		UpdateTS:                1234567890.123,
		UpdateUserID:            1,
	}

	created, err := s.repo.CreateUser(context.Background(), testUser)
	s.NoError(err)
	s.NotNil(created)

	// Now retrieve the user by username
	user, err := s.repo.RetrieveUserByUserName(context.Background(), "RepoTestRetrieveUser")
	s.NoError(err)
	s.NotNil(user)
	s.Equal("RepoTestRetrieveUser", user.UserName)
	s.Equal("Test", user.LastName)
	s.Equal("retrieve@example.com", user.Email)
	s.Equal(1, user.SecurityContext)
	s.False(user.MFAExempt)
	s.NotNil(user.UserGroupMembershipList)
	s.True(len(user.UserGroupMembershipList) > 0)
	s.Nil(user.DeleteTS)
	s.Nil(user.DeleteUserID)
}

func (s *UserRepositoryTestSuite) TestRetrieveUserByUserNameNotFound() {
	// Test with non-existent username
	user, err := s.repo.RetrieveUserByUserName(context.Background(), "nonexistentuser")
	s.Error(err)
	s.Nil(user)
	s.Contains(err.Error(), "user not found")
}

func (s *UserRepositoryTestSuite) TestRetrieveUserGrantList() {
	// First, check what groups actually exist
	allGroups, err := s.db.Query(context.Background(), "SELECT user_group_id FROM user_group WHERE delete_user_id IS NULL ORDER BY user_group_id")
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
	grants, err := s.repo.RetrieveUserGrantList(context.Background(), existingGroupIDList)
	s.NoError(err)
	s.NotNil(grants)

	// Should have at least one security context with grants
	totalGrants := 0
	for _, grantList := range grants {
		totalGrants += len(grantList)
	}
	s.True(totalGrants > 0, "Should have at least one grant")
}

func (s *UserRepositoryTestSuite) TestReadUserGrantsEmpty() {
	// Test with empty slice
	grants, err := s.repo.RetrieveUserGrantList(context.Background(), []int{})
	s.NoError(err)
	s.NotNil(grants)
	s.Equal(0, len(grants))
}

func (s *UserRepositoryTestSuite) TestReadUserGrantsNonExistent() {
	// Test with non-existent group IDs
	grants, err := s.repo.RetrieveUserGrantList(context.Background(), []int{99999, 88888})
	s.NoError(err)
	s.Equal(0, len(grants)) // Should return empty map for non-existent groups
}

func (s *UserRepositoryTestSuite) TestCreateUser() {
	// Create a test user
	user := &model.AppUserModel{
		UserName:                "RepoTestUser",
		PwHash:                  "$2a$10$test.hash.for.testing.purposes.only",
		FirstName:               "Test",
		LastName:                "User",
		Email:                   "test@example.com",
		SecurityContext:         1,
		UserGroupMembershipList: []int{1}, // Assuming group 1 exists
		MFAExempt:               false,
		CreateTS:                1234567890.123,
		CreateUserID:            1,
		UpdateTS:                1234567890.123,
		UpdateUserID:            1,
	}

	created, err := s.repo.CreateUser(context.Background(), user)
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
		UserName:                "invalid-user!",
		PwHash:                  "$2a$10$test.hash.for.testing.purposes.only",
		FirstName:               "Test",
		LastName:                "User",
		Email:                   "test@example.com",
		SecurityContext:         1,
		UserGroupMembershipList: []int{1},
		MFAExempt:               false,
		CreateTS:                1234567890.123,
		CreateUserID:            1,
		UpdateTS:                1234567890.123,
		UpdateUserID:            1,
	}

	created, err := s.repo.CreateUser(context.Background(), user)
	s.Error(err)
	s.Nil(created)
	s.Contains(err.Error(), "username must contain only alphanumeric characters")
}

func (s *UserRepositoryTestSuite) TestUpdateUserPassword() {
	// First create a test user
	user := &model.AppUserModel{
		UserName:                "RepoTestPasswordUser",
		PwHash:                  "$2a$10$original.hash",
		FirstName:               "Password",
		LastName:                "Test",
		Email:                   "password@example.com",
		SecurityContext:         1,
		UserGroupMembershipList: []int{1},
		MFAExempt:               false,
		CreateTS:                1234567890.123,
		CreateUserID:            1,
		UpdateTS:                1234567890.123,
		UpdateUserID:            1,
	}

	created, _ := s.repo.CreateUser(context.Background(), user)

	// Update password
	newHash := "$2a$10$new.hash.for.testing"
	err := s.repo.UpdateUserPassword(context.Background(), created.AppUserID, newHash, 1234567891.456, 2)
	s.NoError(err)

	// Verify password was updated
	var updatedHash string
	err = s.db.QueryRow(context.Background(), "SELECT pw_hash FROM app_user WHERE app_user_id = $1", created.AppUserID).Scan(&updatedHash)
	s.NoError(err)
	s.Equal(newHash, updatedHash)
}

func (s *UserRepositoryTestSuite) TestUpdateUserPasswordNotFound() {
	// Test updating password for non-existent user
	err := s.repo.UpdateUserPassword(context.Background(), 99999, "newhash", 1234567890.123, 1)
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
