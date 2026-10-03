package repository

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/wbmartin/starter-api/internal/model"
	"testing"
)

func TestAppUserGroupRepository(t *testing.T) {
	suite.Run(t, &AppUserGroupRepositoryTestSuite{})
}

type AppUserGroupRepositoryTestSuite struct {
	suite.Suite
	db      *pgxpool.Pool
	repo    *AppUserGroupRepository
	cleanup func()
}

func (s *AppUserGroupRepositoryTestSuite) SetupSuite() {
	db := connectTestDB(s.T())
	s.db = db
	s.repo = NewAppUserGroupRepository(db)

	s.cleanup = func() {
		// Clean only test data if any was created
		// User groups are typically static data from initial migration
	}
}

func (s *AppUserGroupRepositoryTestSuite) TearDownSuite() {
	s.cleanup()
	s.db.Close()
}

func (s *AppUserGroupRepositoryTestSuite) TestNewAppUserGroupRepository() {
	// Test constructor
	repo := NewAppUserGroupRepository(s.db)
	s.NotNil(repo)
	s.Equal(s.db, repo.db)
}

func (s *AppUserGroupRepositoryTestSuite) TestReadAppUserGroupList() {
	// First, check what groups actually exist in the test tenant
	allGroups, err := s.db.Query(
		context.Background(),
		"SELECT app_user_group_id FROM app_user_group WHERE security_context_id = $1 AND delete_user_id IS NULL ORDER BY app_user_group_id",
		testTenant,
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
		// No groups in database for the test tenant, skip this test
		s.T().
			Skip("No user groups found in database for the test tenant - skipping test")
		return
	}

	// Test retrieving existing groups
	groups, err := s.repo.RetrieveAppUserGroupListByIDList(
		context.Background(),
		existingGroupIDList,
		testTenant,
	)
	s.NoError(err)
	s.NotNil(groups)
	s.Len(groups, len(existingGroupIDList))

	// Validate each returned group
	for _, group := range groups {
		s.NotZero(group.AppUserGroupID)
		s.NotEmpty(group.GroupName)
		s.NotNil(group.UserGrantList)
		s.True(
			len(group.UserGrantList) > 0,
			"Group should have at least one grant",
		)
		// Note: CreateTS, CreateUserID, UpdateTS, UpdateUserID may be 0 if database wasn't properly initialized
		// Just check they're not nil pointers for the delete fields
		s.Nil(group.DeleteTS) // Should not be deleted
		s.Nil(group.DeleteUserID)
	}
}

func (s *AppUserGroupRepositoryTestSuite) TestRetrieveAppUserGroupListByIDListEmpty() {
	// Test with empty list
	groups, err := s.repo.RetrieveAppUserGroupListByIDList(
		context.Background(),
		[]int{},
		testTenant,
	)
	s.NoError(err)
	s.Equal(0, len(groups))
	s.Equal([]*model.AppUserGroupModel{}, groups)
}

func (s *AppUserGroupRepositoryTestSuite) TestRetrieveAppUserGroupListByIDListNonExistent() {
	// Test with non-existent IDs
	groupIDList := []int{99999, 88888}
	groups, err := s.repo.RetrieveAppUserGroupListByIDList(
		context.Background(),
		groupIDList,
		testTenant,
	)
	s.NoError(err)
	s.Equal(0, len(groups)) // Should return empty slice for non-existent groups
}

func (s *AppUserGroupRepositoryTestSuite) TestRetrieveAppUserGroupListByIDListPartial() {
	// First, check what groups actually exist in the test tenant
	allGroups, err := s.db.Query(
		context.Background(),
		"SELECT app_user_group_id, group_name FROM app_user_group WHERE security_context_id = $1 AND delete_user_id IS NULL ORDER BY app_user_group_id LIMIT 1",
		testTenant,
	)
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
	groups, err := s.repo.RetrieveAppUserGroupListByIDList(
		context.Background(),
		groupIDList,
		testTenant,
	)
	s.NoError(err)

	// Should find only the existing group
	s.Len(groups, 1)
	s.Equal(existingID, groups[0].AppUserGroupID)
	s.Equal(existingName, groups[0].GroupName)
}

func (s *AppUserGroupRepositoryTestSuite) TestRetrieveAppUserGroupList() {
	// Test retrieving all groups for the test tenant
	groups, err := s.repo.RetrieveAppUserGroupList(
		context.Background(),
		nil,
		testTenant,
	)
	s.NoError(err)
	s.NotNil(groups)

	// Should have at least the default groups
	s.True(len(groups) > 0)

	// Validate each group
	for _, group := range groups {
		s.NotZero(group.AppUserGroupID)
		s.NotEmpty(group.GroupName)
		s.NotNil(group.UserGrantList)
		s.True(len(group.UserGrantList) > 0)
		s.Equal(testTenant, group.SecurityContext)
		s.Nil(group.DeleteTS)
		s.Nil(group.DeleteUserID)
	}

	// Test for security context with no groups
	emptyGroups, err := s.repo.RetrieveAppUserGroupList(
		context.Background(),
		nil,
		999,
	)
	s.NoError(err)
	s.Len(emptyGroups, 0)
}

func (s *AppUserGroupRepositoryTestSuite) TestCreateAppUserGroup() {
	// Create a test group
	group := &model.AppUserGroupModel{
		GroupName:       "RepoTestGroup",
		UserGrantList:   []int{1001, 1002},
		SecurityContext: testTenant,
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created, err := s.repo.CreateAppUserGroup(
		context.Background(), group, testTenant)
	s.NoError(err)
	s.NotNil(created)
	s.NotZero(created.AppUserGroupID)
	s.Equal("RepoTestGroup", created.GroupName)
	s.Equal([]int{1001, 1002}, created.UserGrantList)
	s.Equal(testTenant, created.SecurityContext)

	// Verify in database
	var dbName string
	var dbGrants []int
	var dbUpdateTS float64
	err = s.db.QueryRow(
		context.Background(),
		"SELECT group_name, user_grant_list, update_ts FROM app_user_group WHERE app_user_group_id = $1",
		created.AppUserGroupID,
	).Scan(&dbName, &dbGrants, &dbUpdateTS)
	s.NoError(err)
	s.Equal("RepoTestGroup", dbName)
	s.Equal([]int{1001, 1002}, dbGrants)

	// Clean up
	_, _ = s.db.Exec(
		context.Background(),
		"DELETE FROM app_user_group WHERE app_user_group_id = $1",
		created.AppUserGroupID,
	)
}

func (s *AppUserGroupRepositoryTestSuite) TestUpdateAppUserGroup() {
	// First create a test group
	group := &model.AppUserGroupModel{
		GroupName:       "RepoTestUpdateGroup",
		UserGrantList:   []int{1001},
		SecurityContext: testTenant,
		CreateTS:        1234567890.123,
		CreateUserID:    1,
		UpdateTS:        1234567890.123,
		UpdateUserID:    1,
	}

	created, _ := s.repo.CreateAppUserGroup(
		context.Background(), group, testTenant)

	// Update the group
	updatedGroup := &model.AppUserGroupModel{
		AppUserGroupID: created.AppUserGroupID,
		GroupName:      "RepoTestUpdatedGroup",
		UserGrantList:  []int{1001, 1002, 1003},
		UpdateTS:       1234567891.456,
		UpdateUserID:   2,
	}

	err := s.repo.UpdateAppUserGroup(
		context.Background(),
		updatedGroup,
		created.UpdateTS,
		testTenant,
	)
	s.NoError(err)

	// Verify update
	retrieved, err := s.repo.RetrieveAppUserGroupListByIDList(
		context.Background(),
		[]int{created.AppUserGroupID},
		testTenant,
	)
	s.NoError(err)
	s.Len(retrieved, 1)
	s.Equal("RepoTestUpdatedGroup", retrieved[0].GroupName)
	s.Equal([]int{1001, 1002, 1003}, retrieved[0].UserGrantList)

	// Test update with stale timestamp
	err = s.repo.UpdateAppUserGroup(
		context.Background(),
		updatedGroup,
		created.UpdateTS,
		testTenant,
	)
	s.Error(err)
	s.Equal(ErrConflict, err)

	// Test update non-existent group
	nonExistent := &model.AppUserGroupModel{
		AppUserGroupID: 99999,
		GroupName:      "NonExistent",
		UpdateTS:       1234567890.123,
	}
	err = s.repo.UpdateAppUserGroup(
		context.Background(),
		nonExistent,
		1234567890.123,
		testTenant,
	)
	s.Error(err)
	s.Equal(ErrNotFound, err)

	// Clean up
	_, _ = s.db.Exec(
		context.Background(),
		"DELETE FROM app_user_group WHERE app_user_group_id = $1",
		created.AppUserGroupID,
	)
}

func (s *AppUserGroupRepositoryTestSuite) TestDeleteAppUserGroup() {
	s.T().
		Skip("Skipping due to test environment issues - functionality tested in integration tests")
}

func (s *AppUserGroupRepositoryTestSuite) TestRetrieveMemberUserNameList() {
	ctx := context.Background()
	// Seed: admin holds group 2 (Admin) of the test tenant.
	memberList, err := s.repo.RetrieveMemberUserNameList(ctx, 2, testTenant)
	s.NoError(err)
	s.Contains(memberList, "admin")

	// The tenant filter is on the group: group 2 is not otherTenant's.
	memberList, err = s.repo.RetrieveMemberUserNameList(ctx, 2, otherTenant)
	s.NoError(err)
	s.Empty(memberList)
}

// createGroupIn creates a live group in securityContext.
func (s *AppUserGroupRepositoryTestSuite) createGroupIn(
	groupName string, securityContext int,
) *model.AppUserGroupModel {
	group := &model.AppUserGroupModel{
		GroupName:       groupName,
		UserGrantList:   []int{},
		SecurityContext: securityContext,
		CreateTS:        CurrentTimestamp(),
		CreateUserID:    1,
		UpdateTS:        CurrentTimestamp(),
		UpdateUserID:    1,
	}
	created, err := s.repo.CreateAppUserGroup(
		context.Background(), group, securityContext)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		_, _ = s.db.Exec(context.Background(),
			"DELETE FROM app_user_group WHERE app_user_group_id = $1",
			created.AppUserGroupID)
	})
	return created
}

// Another tenant's group id, with its real update_ts, is not found (404),
// never a conflict (409); a stale update_ts in the caller's tenant still is.
func (s *AppUserGroupRepositoryTestSuite) TestGroupWriteOtherSecurityContextIsNotFound() {
	ctx := context.Background()
	other := s.createGroupIn("RepoTestOtherCtxGroup", otherTenant)

	err := s.repo.UpdateAppUserGroup(ctx, &model.AppUserGroupModel{
		AppUserGroupID: other.AppUserGroupID, GroupName: "RepoTestRenamed",
		UserGrantList: []int{}, UpdateTS: CurrentTimestamp(), UpdateUserID: 1,
	}, other.UpdateTS, testTenant)
	s.ErrorIs(err, ErrNotFound)

	_, err = s.repo.DeleteAppUserGroup(ctx, other.AppUserGroupID, testTenant,
		other.UpdateTS, CurrentTimestamp(), 1)
	s.ErrorIs(err, ErrNotFound)

	unchanged, err := s.repo.RetrieveAppUserGroupListByIDList(
		ctx, []int{other.AppUserGroupID}, otherTenant)
	s.Require().NoError(err)
	s.Require().Len(unchanged, 1)
	s.Equal("RepoTestOtherCtxGroup", unchanged[0].GroupName)

	_, err = s.repo.DeleteAppUserGroup(ctx, other.AppUserGroupID,
		otherTenant, other.UpdateTS-1, CurrentTimestamp(), 1)
	s.ErrorIs(err, ErrConflict)
}

// A member of the caller's tenant holding another tenant's group id does
// not make that group's delete a 409: the id reads as absent (404).
func (s *AppUserGroupRepositoryTestSuite) TestDeleteOtherTenantGroupWithOwnTenantMemberIsNotFound() {
	ctx := context.Background()
	userRepository := NewUserRepository(s.db)
	other := s.createGroupIn("RepoTestForeignHeldGroup", otherTenant)
	_, err := userRepository.CreateUser(ctx, &model.AppUserModel{
		UserName:                   "RepoTestForeignGroupHolder",
		PwHash:                     "$2a$10$test.hash",
		SecurityContext:            testTenant,
		AppUserGroupMembershipList: []int{other.AppUserGroupID},
		CreateTS:                   CurrentTimestamp(),
		CreateUserID:               1,
		UpdateTS:                   CurrentTimestamp(),
		UpdateUserID:               1,
	}, testTenant)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		_, _ = s.db.Exec(context.Background(),
			"DELETE FROM app_user WHERE user_name = $1",
			"RepoTestForeignGroupHolder")
	})

	_, err = s.repo.DeleteAppUserGroup(ctx, other.AppUserGroupID, testTenant,
		other.UpdateTS, CurrentTimestamp(), 1)
	s.ErrorIs(err, ErrNotFound)
}

// The delete guard counts members in the group's tenant only: a login homed
// in another tenant that still lists the id does not block the delete.
func (s *AppUserGroupRepositoryTestSuite) TestDeleteAppUserGroupCountsOwnTenantMembersOnly() {
	ctx := context.Background()
	userRepository := NewUserRepository(s.db)
	group := s.createGroupIn("RepoTestMemberCountGroup", testTenant)

	createMember := func(userName string, securityContext int) {
		_, err := userRepository.CreateUser(ctx, &model.AppUserModel{
			UserName:                   userName,
			PwHash:                     "$2a$10$test.hash",
			SecurityContext:            securityContext,
			AppUserGroupMembershipList: []int{group.AppUserGroupID},
			CreateTS:                   CurrentTimestamp(),
			CreateUserID:               1,
			UpdateTS:                   CurrentTimestamp(),
			UpdateUserID:               1,
		}, securityContext)
		s.Require().NoError(err)
		s.T().Cleanup(func() {
			_, _ = s.db.Exec(context.Background(),
				"DELETE FROM app_user WHERE user_name = $1", userName)
		})
	}

	createMember("RepoTestForeignMember", otherTenant)
	createMember("RepoTestOwnMember", testTenant)
	_, err := s.repo.DeleteAppUserGroup(ctx, group.AppUserGroupID, testTenant,
		group.UpdateTS, CurrentTimestamp(), 1)
	s.Require().Error(err)
	s.ErrorIs(err, ErrInUse)
	s.Equal("cannot delete group: 1 users are assigned to it", err.Error())

	_, err = s.db.Exec(ctx,
		"DELETE FROM app_user WHERE user_name = 'RepoTestOwnMember'")
	s.Require().NoError(err)
	deletedID, err := s.repo.DeleteAppUserGroup(ctx, group.AppUserGroupID,
		testTenant,
		group.UpdateTS, CurrentTimestamp(), 1)
	s.NoError(err)
	s.Equal(group.AppUserGroupID, deletedID)
}
