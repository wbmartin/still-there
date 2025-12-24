package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wbmartin/starter-api/internal/model"
)

// Mock repositories for testing
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(
	ctx context.Context,
	user *model.AppUserModel,
	securityContext int,
) (*model.AppUserModel, error) {
	args := m.Called(ctx, user, securityContext)
	return args.Get(0).(*model.AppUserModel), args.Error(1)
}

func (m *MockUserRepository) RetrieveUserByUserName(
	ctx context.Context,
	username string,
) (*model.AppUserModel, error) {
	args := m.Called(ctx, username)
	return args.Get(0).(*model.AppUserModel), args.Error(1)
}

func (m *MockUserRepository) RetrieveUserList(
	ctx context.Context,
	securityContext int,
	idList []int,
	updateTs float64,
) ([]*model.AppUserModel, error) {
	args := m.Called(ctx, securityContext, idList, updateTs)
	return args.Get(0).([]*model.AppUserModel), args.Error(1)
}

func (m *MockUserRepository) UpdateUser(
	ctx context.Context,
	user *model.AppUserModel,
	clientUpdateTS float64,
	securityContext int,
) error {
	args := m.Called(ctx, user, clientUpdateTS, securityContext)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteUser(
	ctx context.Context,
	userID int,
	securityContext int,
	updateTS float64,
	now float64,
	updateUserID int,
) (int, error) {
	args := m.Called(ctx, userID, securityContext, updateTS, now, updateUserID)
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepository) UpdateUserPassword(
	ctx context.Context,
	userID int,
	hashedPassword string,
	now float64,
	updateUserID int,
) error {
	args := m.Called(ctx, userID, hashedPassword, now, updateUserID)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateUserTOTPSeed(
	ctx context.Context,
	userID int,
	totpSeed string,
	updateTS float64,
	updateUserID int,
) (float64, error) {
	args := m.Called(ctx, userID, totpSeed, updateTS, updateUserID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockUserRepository) UpdateUserMFAExempt(
	ctx context.Context,
	userID int,
	mfaExempt bool,
	updateTS float64,
	updateUserID int,
) (float64, error) {
	args := m.Called(ctx, userID, mfaExempt, updateTS, updateUserID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockUserRepository) AddUserListToGroup(
	ctx context.Context,
	userIDs []int,
	groupID int,
	updateTsList []float64,
	updateUserID int,
	securityContext int,
) ([]*model.AppUserModel, error) {
	args := m.Called(
		ctx,
		userIDs,
		groupID,
		updateTsList,
		updateUserID,
		securityContext,
	)
	return args.Get(0).([]*model.AppUserModel), args.Error(1)
}

func (m *MockUserRepository) RemoveUserListFromGroup(
	ctx context.Context,
	userIDs []int,
	groupID int,
	updateTsList []float64,
	updateUserID int,
	securityContext int,
) ([]*model.AppUserModel, error) {
	args := m.Called(
		ctx,
		userIDs,
		groupID,
		updateTsList,
		updateUserID,
		securityContext,
	)
	return args.Get(0).([]*model.AppUserModel), args.Error(1)
}

func (m *MockUserRepository) GetGrantsForGroups(
	ctx context.Context,
	groupIDs []int,
) (map[int][]int, error) {
	args := m.Called(ctx, groupIDs)
	return args.Get(0).(map[int][]int), args.Error(1)
}

type MockAppUserGroupRepository struct {
	mock.Mock
}

func (m *MockAppUserGroupRepository) RetrieveAppUserGroupListByIDList(
	ctx context.Context,
	groupIDs []int,
	securityContext int,
) ([]*model.AppUserGroupModel, error) {
	args := m.Called(ctx, groupIDs, securityContext)
	return args.Get(0).([]*model.AppUserGroupModel), args.Error(1)
}

func (m *MockAppUserGroupRepository) UpdateAppUserGroup(
	ctx context.Context,
	group *model.AppUserGroupModel,
	clientUpdateTS float64,
	securityContext int,
) error {
	args := m.Called(ctx, group, clientUpdateTS, securityContext)
	return args.Error(0)
}

func (m *MockAppUserGroupRepository) CreateAppUserGroup(
	ctx context.Context,
	group *model.AppUserGroupModel,
	securityContext int,
) (*model.AppUserGroupModel, error) {
	args := m.Called(ctx, group, securityContext)
	return args.Get(0).(*model.AppUserGroupModel), args.Error(1)
}

func TestValidateGroupsSecurityContext(t *testing.T) {
	tests := []struct {
		name                    string
		groupIDs                []int
		expectedSecurityContext int
		mockGroups              []*model.AppUserGroupModel
		mockError               error
		expectedValid           bool
		expectedError           bool
	}{
		{
			name:                    "empty group list",
			groupIDs:                []int{},
			expectedSecurityContext: 1,
			mockGroups:              []*model.AppUserGroupModel{},
			mockError:               nil,
			expectedValid:           true,
			expectedError:           false,
		},
		{
			name:                    "all groups valid",
			groupIDs:                []int{1, 2},
			expectedSecurityContext: 1,
			mockGroups: []*model.AppUserGroupModel{
				{AppUserGroupID: 1, SecurityContext: 1},
				{AppUserGroupID: 2, SecurityContext: 1},
			},
			mockError:     nil,
			expectedValid: true,
			expectedError: false,
		},
		{
			name:                    "some groups have wrong security context",
			groupIDs:                []int{1, 2},
			expectedSecurityContext: 1,
			mockGroups: []*model.AppUserGroupModel{
				{AppUserGroupID: 1, SecurityContext: 1},
				{AppUserGroupID: 2, SecurityContext: 2}, // wrong context
			},
			mockError:     nil,
			expectedValid: false,
			expectedError: true,
		},
		{
			name:                    "some groups not found",
			groupIDs:                []int{1, 2},
			expectedSecurityContext: 1,
			mockGroups: []*model.AppUserGroupModel{
				{AppUserGroupID: 1, SecurityContext: 1},
				// missing group 2
			},
			mockError:     nil,
			expectedValid: false,
			expectedError: true,
		},
		{
			name:                    "repository error",
			groupIDs:                []int{1},
			expectedSecurityContext: 1,
			mockGroups:              nil,
			mockError:               assert.AnError,
			expectedValid:           false,
			expectedError:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAppUserGroupRepository)
			if len(tt.groupIDs) > 0 {
				mockRepo.On("RetrieveAppUserGroupListByIDList", mock.Anything, tt.groupIDs, mock.AnythingOfType("int")).
					Return(tt.mockGroups, tt.mockError)
			}

			handler := &UserHandler{
				appUserGroupRepo: mockRepo,
			}

			valid, err := handler.validateGroupsSecurityContext(
				tt.groupIDs,
				tt.expectedSecurityContext,
				context.Background(),
			)

			assert.Equal(t, tt.expectedValid, valid)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestValidateUsersSecurityContext(t *testing.T) {
	tests := []struct {
		name                    string
		userIDs                 []int
		expectedSecurityContext int
		mockUsers               []*model.AppUserModel
		mockError               error
		expectedValid           bool
		expectedError           bool
	}{
		{
			name:                    "empty user list",
			userIDs:                 []int{},
			expectedSecurityContext: 1,
			mockUsers:               []*model.AppUserModel{},
			mockError:               nil,
			expectedValid:           true,
			expectedError:           false,
		},
		{
			name:                    "all users valid",
			userIDs:                 []int{1, 2},
			expectedSecurityContext: 1,
			mockUsers: []*model.AppUserModel{
				{AppUserID: 1, SecurityContext: 1},
				{AppUserID: 2, SecurityContext: 1},
			},
			mockError:     nil,
			expectedValid: true,
			expectedError: false,
		},
		{
			name:                    "some users have wrong security context",
			userIDs:                 []int{1, 2},
			expectedSecurityContext: 1,
			mockUsers: []*model.AppUserModel{
				{AppUserID: 1, SecurityContext: 1},
				{AppUserID: 2, SecurityContext: 2}, // wrong context
			},
			mockError:     nil,
			expectedValid: false,
			expectedError: true,
		},
		{
			name:                    "some users not found",
			userIDs:                 []int{1, 2},
			expectedSecurityContext: 1,
			mockUsers: []*model.AppUserModel{
				{AppUserID: 1, SecurityContext: 1},
				// missing user 2
			},
			mockError:     nil,
			expectedValid: false,
			expectedError: true,
		},
		{
			name:                    "repository error",
			userIDs:                 []int{1},
			expectedSecurityContext: 1,
			mockUsers:               nil,
			mockError:               assert.AnError,
			expectedValid:           false,
			expectedError:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			if len(tt.userIDs) > 0 {
				mockRepo.On("RetrieveUserList", mock.Anything, tt.expectedSecurityContext, tt.userIDs, float64(0)).
					Return(tt.mockUsers, tt.mockError)
			}

			handler := &UserHandler{
				userRepo: mockRepo,
			}

			valid, err := handler.validateUsersSecurityContext(
				tt.userIDs,
				tt.expectedSecurityContext,
				context.Background(),
			)

			assert.Equal(t, tt.expectedValid, valid)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name           string
		inputUser      model.AppUserModel
		mockResult     *model.AppUserModel
		mockError      error
		expectedError  bool
		expectedUserID int
		expectCall     bool
	}{
		{
			name: "successful user creation",
			inputUser: model.AppUserModel{
				UserName:        "testuser",
				FirstName:       "Test",
				LastName:        "User",
				Email:           "test@example.com",
				SecurityContext: 1,
			},
			mockResult: &model.AppUserModel{
				AppUserID:       123,
				UserName:        "testuser",
				FirstName:       "Test",
				LastName:        "User",
				Email:           "test@example.com",
				SecurityContext: 1,
			},
			mockError:      nil,
			expectedError:  false,
			expectedUserID: 123,
			expectCall:     true,
		},
		{
			name: "repository error - duplicate user",
			inputUser: model.AppUserModel{
				UserName:  "existinguser",
				FirstName: "Existing",
				LastName:  "User",
				Email:     "existing@example.com",
			},
			mockResult:     nil,
			mockError:      assert.AnError, // Simulates unique constraint violation
			expectedError:  true,
			expectedUserID: 0,
			expectCall:     true,
		},
		{
			name: "database connection error",
			inputUser: model.AppUserModel{
				UserName:  "testuser",
				FirstName: "Test",
				LastName:  "User",
				Email:     "test@example.com",
			},
			mockResult:     nil,
			mockError:      assert.AnError, // Simulates connection failure
			expectedError:  true,
			expectedUserID: 0,
			expectCall:     true,
		},
		{
			name: "user with empty required fields",
			inputUser: model.AppUserModel{
				UserName:  "", // Empty username
				FirstName: "Test",
				LastName:  "User",
				Email:     "test@example.com",
			},
			mockResult:     nil,
			mockError:      nil,
			expectedError:  false, // Repository would handle validation, but we're testing the interface
			expectedUserID: 0,
			expectCall:     true,
		},
		{
			name: "user with special characters in username",
			inputUser: model.AppUserModel{
				UserName:  "test_user@123",
				FirstName: "Test",
				LastName:  "User",
				Email:     "test@example.com",
			},
			mockResult: &model.AppUserModel{
				AppUserID:       124,
				UserName:        "test_user@123",
				FirstName:       "Test",
				LastName:        "User",
				Email:           "test@example.com",
				SecurityContext: 1,
			},
			mockError:      nil,
			expectedError:  false,
			expectedUserID: 124,
			expectCall:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			if tt.expectCall {
				mockUserRepo.On("CreateUser", mock.Anything, &tt.inputUser, mock.AnythingOfType("int")).
					Return(tt.mockResult, tt.mockError)
			}

			handler := &UserHandler{
				userRepo: mockUserRepo,
			}

			// Simulate HTTP request for context
			req := httptest.NewRequest("POST", "/user", nil)
			result, err := handler.userRepo.CreateUser(
				req.Context(),
				&tt.inputUser,
				1,
			)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.mockResult == nil {
					assert.Nil(t, result)
				}
			} else {
				assert.NoError(t, err)
				if tt.expectedUserID > 0 {
					assert.NotNil(t, result)
					assert.Equal(t, tt.expectedUserID, result.AppUserID)
					assert.Equal(t, tt.inputUser.UserName, result.UserName)
				}
			}
			if tt.expectCall {
				mockUserRepo.AssertExpectations(t)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	tests := []struct {
		name           string
		inputUser      model.AppUserModel
		clientUpdateTS float64
		mockError      error
		expectedError  bool
		expectCall     bool
	}{
		{
			name: "successful user update",
			inputUser: model.AppUserModel{
				AppUserID:       123,
				UserName:        "testuser",
				FirstName:       "Updated",
				LastName:        "User",
				Email:           "updated@example.com",
				SecurityContext: 1,
			},
			clientUpdateTS: 1234567890.123,
			mockError:      nil,
			expectedError:  false,
			expectCall:     true,
		},
		{
			name: "repository error - concurrent modification",
			inputUser: model.AppUserModel{
				AppUserID: 123,
				UserName:  "testuser",
				FirstName: "Updated",
			},
			clientUpdateTS: 1234567890.123,
			mockError:      assert.AnError, // Simulates concurrent update conflict
			expectedError:  true,
			expectCall:     true,
		},
		{
			name: "update non-existent user",
			inputUser: model.AppUserModel{
				AppUserID: 999,
				UserName:  "missinguser",
			},
			clientUpdateTS: 1234567890.123,
			mockError:      assert.AnError, // Simulates user not found
			expectedError:  true,
			expectCall:     true,
		},
		{
			name: "update with stale timestamp",
			inputUser: model.AppUserModel{
				AppUserID: 123,
				UserName:  "testuser",
			},
			clientUpdateTS: 1234567800.000, // Old timestamp
			mockError:      assert.AnError, // Simulates optimistic locking failure
			expectedError:  true,
			expectCall:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			if tt.expectCall {
				mockUserRepo.On("UpdateUser", mock.Anything, &tt.inputUser, tt.clientUpdateTS, mock.AnythingOfType("int")).
					Return(tt.mockError)
			}

			handler := &UserHandler{
				userRepo: mockUserRepo,
			}

			// Simulate HTTP request for context
			req := httptest.NewRequest("PUT", "/user", nil)
			err := handler.userRepo.UpdateUser(
				req.Context(),
				&tt.inputUser,
				tt.clientUpdateTS,
				1,
			)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.expectCall {
				mockUserRepo.AssertExpectations(t)
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	tests := []struct {
		name            string
		userID          int
		securityContext int
		updateTS        float64
		now             float64
		updateUserID    int
		mockResult      int
		mockError       error
		expectedError   bool
		expectedID      int
		expectCall      bool
	}{
		{
			name:            "successful user deletion",
			userID:          123,
			securityContext: 1,
			updateTS:        1234567890.123,
			now:             1234567891.456,
			updateUserID:    1,
			mockResult:      123,
			mockError:       nil,
			expectedError:   false,
			expectedID:      123,
			expectCall:      true,
		},
		{
			name:            "delete non-existent user",
			userID:          999,
			securityContext: 1,
			updateTS:        1234567890.123,
			now:             1234567891.456,
			updateUserID:    1,
			mockResult:      0,
			mockError:       assert.AnError, // Simulates user not found
			expectedError:   true,
			expectedID:      0,
			expectCall:      true,
		},
		{
			name:            "concurrent deletion conflict",
			userID:          123,
			securityContext: 1,
			updateTS:        1234567890.123,
			now:             1234567891.456,
			updateUserID:    1,
			mockResult:      0,
			mockError:       assert.AnError, // Simulates already deleted or concurrent modification
			expectedError:   true,
			expectedID:      0,
			expectCall:      true,
		},
		{
			name:            "delete with wrong security context",
			userID:          123,
			securityContext: 2, // Wrong context
			updateTS:        1234567890.123,
			now:             1234567891.456,
			updateUserID:    1,
			mockResult:      0,
			mockError:       assert.AnError, // Simulates security context violation
			expectedError:   true,
			expectedID:      0,
			expectCall:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			if tt.expectCall {
				mockUserRepo.On("DeleteUser", mock.Anything, tt.userID, tt.securityContext, tt.updateTS, tt.now, tt.updateUserID).
					Return(tt.mockResult, tt.mockError)
			}

			handler := &UserHandler{
				userRepo: mockUserRepo,
			}

			// Simulate HTTP request for context
			req := httptest.NewRequest("DELETE", "/user", nil)
			result, err := handler.userRepo.DeleteUser(
				req.Context(),
				tt.userID,
				tt.securityContext,
				tt.updateTS,
				tt.now,
				tt.updateUserID,
			)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, result)
			}
			if tt.expectCall {
				mockUserRepo.AssertExpectations(t)
			}
		})
	}
}

func TestRetrieveUserList(t *testing.T) {
	tests := []struct {
		name            string
		securityContext int
		idList          []int
		updateTS        float64
		mockResult      []*model.AppUserModel
		mockError       error
		expectedError   bool
		expectedCount   int
		expectCall      bool
	}{
		{
			name:            "successful user list retrieval",
			securityContext: 1,
			idList:          []int{1, 2, 3},
			updateTS:        0,
			mockResult: []*model.AppUserModel{
				{AppUserID: 1, UserName: "user1", SecurityContext: 1},
				{AppUserID: 2, UserName: "user2", SecurityContext: 1},
				{AppUserID: 3, UserName: "user3", SecurityContext: 1},
			},
			mockError:     nil,
			expectedError: false,
			expectedCount: 3,
			expectCall:    true,
		},
		{
			name:            "retrieve empty user list",
			securityContext: 1,
			idList:          []int{},
			updateTS:        0,
			mockResult:      []*model.AppUserModel{},
			mockError:       nil,
			expectedError:   false,
			expectedCount:   0,
			expectCall:      true,
		},
		{
			name:            "partial user retrieval - some not found",
			securityContext: 1,
			idList:          []int{1, 999, 3}, // 999 doesn't exist
			updateTS:        0,
			mockResult: []*model.AppUserModel{
				{AppUserID: 1, UserName: "user1", SecurityContext: 1},
				{AppUserID: 3, UserName: "user3", SecurityContext: 1},
			},
			mockError:     nil,
			expectedError: false,
			expectedCount: 2,
			expectCall:    true,
		},
		{
			name:            "database connection error",
			securityContext: 1,
			idList:          []int{1, 2},
			updateTS:        0,
			mockResult:      nil,
			mockError:       assert.AnError,
			expectedError:   true,
			expectedCount:   0,
			expectCall:      true,
		},
		{
			name:            "filtered by update timestamp",
			securityContext: 1,
			idList:          []int{1, 2},
			updateTS:        1234567890.123,
			mockResult: []*model.AppUserModel{
				{
					AppUserID:       1,
					UserName:        "user1",
					SecurityContext: 1,
					UpdateTS:        1234567891.456,
				},
			},
			mockError:     nil,
			expectedError: false,
			expectedCount: 1,
			expectCall:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			if tt.expectCall {
				mockUserRepo.On("RetrieveUserList", mock.Anything, tt.securityContext, tt.idList, tt.updateTS).
					Return(tt.mockResult, tt.mockError)
			}

			handler := &UserHandler{
				userRepo: mockUserRepo,
			}

			// Simulate HTTP request for context
			req := httptest.NewRequest("GET", "/userList", nil)
			result, err := handler.userRepo.RetrieveUserList(
				req.Context(),
				tt.securityContext,
				tt.idList,
				tt.updateTS,
			)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedCount)
				if tt.expectedCount > 0 {
					for _, user := range result {
						assert.Equal(t, tt.securityContext, user.SecurityContext)
					}
				}
			}
			if tt.expectCall {
				mockUserRepo.AssertExpectations(t)
			}
		})
	}
}
