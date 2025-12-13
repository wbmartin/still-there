package repository

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"testing"
)

func TestUserGroupRepository(t *testing.T) {
	suite.Run(t, &UserGroupRepositoryTestSuite{})
}

type UserGroupRepositoryTestSuite struct {
	suite.Suite
	db      *pgxpool.Pool
	repo    *UserGroupRepository
	cleanup func()
}

func (s *UserGroupRepositoryTestSuite) SetupSuite() {
	// Load configuration from the same file used by the application
	config, err := app_lib.LoadConfig("../../../../etc/loc-api-srv.env")
	s.Require().NoError(err)

	// Connect to the database using the configured URL
	db, err := pgxpool.New(context.Background(), config.DatabaseURL)
	s.Require().NoError(err)
	s.db = db
	s.repo = NewUserGroupRepository(db)

	s.cleanup = func() {
		// Clean only test data if any was created
		// User groups are typically static data from initial migration
	}
}

func (s *UserGroupRepositoryTestSuite) TearDownSuite() {
	s.cleanup()
	s.db.Close()
}

func (s *UserGroupRepositoryTestSuite) TestNewUserGroupRepository() {
	// Test constructor
	repo := NewUserGroupRepository(s.db)
	s.NotNil(repo)
	s.Equal(s.db, repo.db)
}

func (s *UserGroupRepositoryTestSuite) TestReadUserGroupList() {
	// First, check what groups actually exist in the database
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
		s.T().Skip("No user groups found in database - skipping test")
		return
	}

	// Test retrieving existing groups
	groups, err := s.repo.RetrieveUserGroupListByIDList(context.Background(), existingGroupIDList)
	s.NoError(err)
	s.NotNil(groups)
	s.Len(groups, len(existingGroupIDList))

	// Validate each returned group
	for _, group := range groups {
		s.NotZero(group.UserGroupID)
		s.NotEmpty(group.GroupName)
		s.NotNil(group.UserGrantList)
		s.True(len(group.UserGrantList) > 0, "Group should have at least one grant")
		// Note: CreateTS, CreateUserID, UpdateTS, UpdateUserID may be 0 if database wasn't properly initialized
		// Just check they're not nil pointers for the delete fields
		s.Nil(group.DeleteTS) // Should not be deleted
		s.Nil(group.DeleteUserID)
	}
}

func (s *UserGroupRepositoryTestSuite) TestRetrieveUserGroupListByIDListEmpty() {
	// Test with empty list
	groups, err := s.repo.RetrieveUserGroupListByIDList(context.Background(), []int{})
	s.NoError(err)
	s.Equal(0, len(groups))
	s.Equal([]*model.UserGroupModel{}, groups)
}

func (s *UserGroupRepositoryTestSuite) TestRetrieveUserGroupListByIDListNonExistent() {
	// Test with non-existent IDs
	groupIDList := []int{99999, 88888}
	groups, err := s.repo.RetrieveUserGroupListByIDList(context.Background(), groupIDList)
	s.NoError(err)
	s.Equal(0, len(groups)) // Should return empty slice for non-existent groups
}

func (s *UserGroupRepositoryTestSuite) TestRetrieveUserGroupListByIDListPartial() {
	// First, check what groups actually exist in the database
	allGroups, err := s.db.Query(context.Background(), "SELECT user_group_id, group_name FROM user_group WHERE delete_user_id IS NULL ORDER BY user_group_id LIMIT 1")
	s.NoError(err)
	defer allGroups.Close()

	var existingID int
	var existingName string
	if allGroups.Next() {
		err := allGroups.Scan(&existingID, &existingName)
		s.NoError(err)
	} else {
		s.T().Skip("No user groups found in database - skipping test")
		return
	}

	// Test with mix of existing and non-existing IDs
	groupIDList := []int{existingID, 99999}
	groups, err := s.repo.RetrieveUserGroupListByIDList(context.Background(), groupIDList)
	s.NoError(err)

	// Should find only the existing group
	s.Len(groups, 1)
	s.Equal(existingID, groups[0].UserGroupID)
	s.Equal(existingName, groups[0].GroupName)
}
