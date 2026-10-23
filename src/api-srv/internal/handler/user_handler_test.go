package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stillthere-api/internal/app"
	"stillthere-api/internal/auth"
	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
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
	securityContext int,
) error {
	args := m.Called(
		ctx, userID, hashedPassword, now, updateUserID, securityContext)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateUserMFASetup(
	ctx context.Context,
	userID int,
	mfaSetup bool,
	expectedUpdateTS float64,
	updateUserID int,
	securityContext int,
) (float64, error) {
	args := m.Called(ctx, userID, mfaSetup, expectedUpdateTS, updateUserID,
		securityContext)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockUserRepository) UpdateUserTOTPSeed(
	ctx context.Context,
	userID int,
	totpSeed string,
	updateTS float64,
	updateUserID int,
	securityContext int,
) (float64, error) {
	args := m.Called(ctx, userID, totpSeed, updateTS, updateUserID,
		securityContext)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockUserRepository) UpdateUserMFAExempt(
	ctx context.Context,
	userID int,
	mfaExempt bool,
	updateTS float64,
	updateUserID int,
	securityContext int,
) (float64, error) {
	args := m.Called(ctx, userID, mfaExempt, updateTS, updateUserID,
		securityContext)
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

// UpdateUserAccountStatus makes the mock a UserBlockRepository too.
func (m *MockUserRepository) UpdateUserAccountStatus(
	ctx context.Context,
	userID int,
	accountStatus string,
	expectedUpdateTS float64,
	updateUserID int,
	securityContext int,
) (float64, error) {
	args := m.Called(ctx, userID, accountStatus, expectedUpdateTS,
		updateUserID, securityContext)
	return args.Get(0).(float64), args.Error(1)
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
			expectedError: false, // a mismatch is a 403, not an error
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
			expectedError: false, // a mismatch is a 403, not an error
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
			expectedError: false, // a mismatch is a 403, not an error
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
			expectedError: false, // a mismatch is a 403, not an error
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

// The user-admin routes driven through the handler (B039; these replace
// tests that called the mock directly). The admin is alice, login 7 of the
// test tenant (2): the tenant and actor written are the JWT's, never the
// body's. Each route's check sequence is proved in
// route_check_contract_test.go; an unexpected repository call panics.

const userAdminTenant = 2

// userAdminRequest is <method> target as alice holding grant.
func userAdminRequest(
	t *testing.T,
	s *app.Services,
	method, target, body string,
	grant int,
) *http.Request {
	t.Helper()
	token, err := auth.GenerateJWT(
		"Test", "alice", 7, "session-adm", userAdminTenant, []int{grant},
		s.Config.JWTSecret, s.Config.JWTSessionExpireHours,
	)
	require.NoError(t, err)
	return sessionRequest(method, target, body, token, "session-adm")
}

// bobRow is the target login's row as the repository returns it.
func bobRow() []*model.AppUserModel {
	return []*model.AppUserModel{{AppUserID: 9, UserName: "bob",
		SecurityContext: userAdminTenant}}
}

func TestCreateUser(t *testing.T) {
	const validBody = `{"userName":"bob","password":"long-enough-1",` +
		`"firstName":"Bob","lastName":"B","email":"b@example.com",` +
		`"securityContext":1}`
	createUser := func(
		t *testing.T, repo *MockUserRepository, body string,
	) *httptest.ResponseRecorder {
		t.Helper()
		s := testServices(t)
		handler := NewUserHandler(s, repo, new(MockAppUserGroupRepository))
		w := httptest.NewRecorder()
		handler.CreateUser(w, userAdminRequest(t, s, http.MethodPost,
			"/user", body, model.GrantUserCreate))
		repo.AssertExpectations(t)
		return w
	}

	t.Run("valid → 201, the JWT's tenant and login", func(t *testing.T) {
		repo := new(MockUserRepository)
		repo.On("CreateUser", mock.Anything,
			mock.MatchedBy(func(user *model.AppUserModel) bool {
				return user.UserName == "bob" &&
					user.SecurityContext == userAdminTenant &&
					user.CreateUserID == 7 && user.UpdateUserID == 7
			}), userAdminTenant).
			Return(&model.AppUserModel{AppUserID: 51, UserName: "bob",
				SecurityContext: userAdminTenant}, nil)
		w := createUser(t, repo, validBody)
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), `"appUserId":51`)
	})

	refusalList := []struct{ name, body string }{
		{"malformed JSON", `{`},
		{"no email", `{"userName":"bob","password":"long-enough-1",` +
			`"firstName":"Bob","lastName":"B"}`},
		{"password under 8 characters", `{"userName":"bob",` +
			`"password":"short","firstName":"Bob","lastName":"B",` +
			`"email":"b@example.com"}`},
	}
	for _, refusal := range refusalList {
		t.Run(refusal.name+" → 400, repository untouched",
			func(t *testing.T) {
				w := createUser(t, new(MockUserRepository), refusal.body)
				assert.Equal(t, http.StatusBadRequest, w.Code,
					w.Body.String())
			})
	}

	t.Run("duplicate name → 409", func(t *testing.T) {
		repo := new(MockUserRepository)
		repo.On("CreateUser", mock.Anything, mock.Anything,
			userAdminTenant).
			Return((*model.AppUserModel)(nil),
				fmt.Errorf("insert: %w", repository.ErrDuplicate))
		w := createUser(t, repo, validBody)
		assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	})

	t.Run("other repository error → 500 without its text",
		func(t *testing.T) {
			repo := new(MockUserRepository)
			repo.On("CreateUser", mock.Anything, mock.Anything,
				userAdminTenant).
				Return((*model.AppUserModel)(nil), errors.New("conn reset"))
			w := createUser(t, repo, validBody)
			assert.Equal(t, http.StatusInternalServerError, w.Code)
			assert.NotContains(t, w.Body.String(), "conn reset")
		})
}

// A group the tenant does not hold is 403 SEC_CTX_VIOLATION; a failed group
// lookup is 500 SEC_CTX_CHECK, never a 403 that hides the outage (C005).
func TestCreateUser_SecurityContextCheck(t *testing.T) {
	const body = `{"userName":"bob","password":"long-enough-1",` +
		`"firstName":"Bob","lastName":"B","email":"b@example.com",` +
		`"appUserGroupMembershipList":[4,5]}`
	createUser := func(
		t *testing.T, groupRepository *MockAppUserGroupRepository,
	) *httptest.ResponseRecorder {
		t.Helper()
		s := testServices(t)
		userRepository := new(MockUserRepository)
		handler := NewUserHandler(s, userRepository, groupRepository)
		w := httptest.NewRecorder()
		handler.CreateUser(w, userAdminRequest(t, s, http.MethodPost,
			"/user", body, model.GrantUserCreate))
		userRepository.AssertExpectations(t)
		groupRepository.AssertExpectations(t)
		return w
	}

	t.Run("a group outside the tenant → 403", func(t *testing.T) {
		groupRepository := new(MockAppUserGroupRepository)
		groupRepository.On("RetrieveAppUserGroupListByIDList",
			mock.Anything, []int{4, 5}, userAdminTenant).
			Return([]*model.AppUserGroupModel{{AppUserGroupID: 4,
				SecurityContext: userAdminTenant}}, nil)
		w := createUser(t, groupRepository)
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), "same security context")
	})

	t.Run("lookup error → 500 without its text", func(t *testing.T) {
		groupRepository := new(MockAppUserGroupRepository)
		groupRepository.On("RetrieveAppUserGroupListByIDList",
			mock.Anything, []int{4, 5}, userAdminTenant).
			Return([]*model.AppUserGroupModel(nil),
				errors.New("conn reset"))
		w := createUser(t, groupRepository)
		assert.Equal(t, http.StatusInternalServerError, w.Code,
			w.Body.String())
		assert.Contains(t, w.Body.String(),
			"Failed to check the security context")
		assert.NotContains(t, w.Body.String(), "conn reset")
	})
}

func TestUpdateUser(t *testing.T) {
	const validBody = `{"appUserId":9,"userName":"bob","firstName":"Bob",` +
		`"securityContext":1,"updateTs":12.5}`
	updateUser := func(
		t *testing.T, repo *MockUserRepository, body string,
	) *httptest.ResponseRecorder {
		t.Helper()
		s := testServices(t)
		handler := NewUserHandler(s, repo, new(MockAppUserGroupRepository))
		w := httptest.NewRecorder()
		handler.UpdateUser(w, userAdminRequest(t, s, http.MethodPut,
			"/user", body, model.GrantUserUpdate))
		repo.AssertExpectations(t)
		return w
	}
	primedRepo := func(updateErr error) *MockUserRepository {
		repo := new(MockUserRepository)
		repo.On("RetrieveUserList", mock.Anything, userAdminTenant,
			[]int{9}, float64(0)).Return(bobRow(), nil)
		repo.On("UpdateUser", mock.Anything,
			mock.MatchedBy(func(user *model.AppUserModel) bool {
				return user.AppUserID == 9 &&
					user.SecurityContext == userAdminTenant &&
					user.UpdateUserID == 7 && user.UpdateTS != 12.5
			}), 12.5, userAdminTenant).Return(updateErr)
		return repo
	}

	t.Run("valid → 200; the client's updateTs is the lock, the JWT's "+
		"tenant and login are written", func(t *testing.T) {
		w := updateUser(t, primedRepo(nil), validBody)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})

	t.Run("stale updateTs → 409", func(t *testing.T) {
		w := updateUser(t, primedRepo(
			fmt.Errorf("update: %w", repository.ErrConflict)), validBody)
		assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	})

	refusalList := []struct{ name, body string }{
		{"malformed JSON", `{`},
		{"no appUserId", `{"userName":"bob","updateTs":12.5}`},
		{"no userName", `{"appUserId":9,"updateTs":12.5}`},
	}
	for _, refusal := range refusalList {
		t.Run(refusal.name+" → 400, repository untouched",
			func(t *testing.T) {
				w := updateUser(t, new(MockUserRepository), refusal.body)
				assert.Equal(t, http.StatusBadRequest, w.Code,
					w.Body.String())
			})
	}
}

// mfa_exempt switches a login to password-only, so POST and PUT /user
// turn it on only where ALLOW_MFA_EXEMPT_VIA_API=true (B097). Clearing it
// and keeping a stored true are always allowed. An unexpected repository
// call panics, so a refusal proves nothing was written.
func TestUserHandler_MFAExemptNeedsTheConfigKey(t *testing.T) {
	const createBody = `{"userName":"bob","password":"long-enough-1",` +
		`"firstName":"Bob","lastName":"B","email":"b@example.com",` +
		`"mfaExempt":%t}`
	const updateBody = `{"appUserId":9,"userName":"bob",` +
		`"updateTs":12.5,"mfaExempt":%t}`

	run := func(
		t *testing.T, allow bool, repo *MockUserRepository,
		method, body string,
	) *httptest.ResponseRecorder {
		t.Helper()
		s := testServices(t)
		s.Config.AllowMFAExemptViaAPI = allow
		handler := NewUserHandler(s, repo, new(MockAppUserGroupRepository))
		w := httptest.NewRecorder()
		if method == http.MethodPost {
			handler.CreateUser(w, userAdminRequest(t, s, method, "/user",
				body, model.GrantUserCreate))
		} else {
			handler.UpdateUser(w, userAdminRequest(t, s, method, "/user",
				body, model.GrantUserUpdate))
		}
		repo.AssertExpectations(t)
		return w
	}
	createRepo := func(wantExempt bool) *MockUserRepository {
		repo := new(MockUserRepository)
		repo.On("CreateUser", mock.Anything,
			mock.MatchedBy(func(user *model.AppUserModel) bool {
				return user.MFAExempt == wantExempt
			}), userAdminTenant).
			Return(&model.AppUserModel{AppUserID: 51, UserName: "bob",
				SecurityContext: userAdminTenant}, nil)
		return repo
	}
	// storedRepo holds bob with the given stored flag; write adds the
	// UpdateUser the handler should reach, carrying wantExempt.
	storedRepo := func(
		storedExempt, write, wantExempt bool,
	) *MockUserRepository {
		repo := new(MockUserRepository)
		row := bobRow()
		row[0].MFAExempt = storedExempt
		repo.On("RetrieveUserList", mock.Anything, userAdminTenant,
			[]int{9}, float64(0)).Return(row, nil)
		if write {
			repo.On("UpdateUser", mock.Anything,
				mock.MatchedBy(func(user *model.AppUserModel) bool {
					return user.MFAExempt == wantExempt
				}), 12.5, userAdminTenant).Return(nil)
		}
		return repo
	}

	t.Run("POST true, not allowed → 403, nothing created",
		func(t *testing.T) {
			w := run(t, false, new(MockUserRepository), http.MethodPost,
				fmt.Sprintf(createBody, true))
			assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), "MFA exemption cannot")
		})
	t.Run("POST true, allowed → 201 exempt", func(t *testing.T) {
		w := run(t, true, createRepo(true), http.MethodPost,
			fmt.Sprintf(createBody, true))
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	})
	t.Run("POST false, not allowed → 201", func(t *testing.T) {
		w := run(t, false, createRepo(false), http.MethodPost,
			fmt.Sprintf(createBody, false))
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	})

	t.Run("PUT turning it on, not allowed → 403, nothing written",
		func(t *testing.T) {
			w := run(t, false, storedRepo(false, false, false),
				http.MethodPut, fmt.Sprintf(updateBody, true))
			assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), "MFA exemption cannot")
		})
	t.Run("PUT turning it on, allowed → 200", func(t *testing.T) {
		w := run(t, true, storedRepo(false, true, true),
			http.MethodPut, fmt.Sprintf(updateBody, true))
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})
	t.Run("PUT keeping a stored true, not allowed → 200",
		func(t *testing.T) {
			w := run(t, false, storedRepo(true, true, true),
				http.MethodPut, fmt.Sprintf(updateBody, true))
			assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		})
	t.Run("PUT clearing it, not allowed → 200", func(t *testing.T) {
		w := run(t, false, storedRepo(true, true, false),
			http.MethodPut, fmt.Sprintf(updateBody, false))
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})
}

func TestDeleteUser(t *testing.T) {
	deleteUser := func(
		t *testing.T, repo *MockUserRepository, target string,
	) *httptest.ResponseRecorder {
		t.Helper()
		s := testServices(t)
		handler := NewUserHandler(s, repo, new(MockAppUserGroupRepository))
		w := httptest.NewRecorder()
		handler.DeleteUser(w, userAdminRequest(t, s, http.MethodDelete,
			target, "", model.GrantUserDelete))
		repo.AssertExpectations(t)
		return w
	}
	primedRepo := func(deletedID int, deleteErr error) *MockUserRepository {
		repo := new(MockUserRepository)
		repo.On("RetrieveUserList", mock.Anything, userAdminTenant,
			[]int{9}, float64(0)).Return(bobRow(), nil)
		repo.On("DeleteUser", mock.Anything, 9, userAdminTenant, 12.5,
			mock.AnythingOfType("float64"), 7).Return(deletedID, deleteErr)
		return repo
	}

	t.Run("valid → 200, deleted in the JWT's tenant by its login",
		func(t *testing.T) {
			w := deleteUser(t, primedRepo(9, nil),
				"/user?appUserId=9&updateTs=12.5")
			assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), `"appUserId":9`)
		})

	t.Run("no such login in the tenant → 404", func(t *testing.T) {
		w := deleteUser(t, primedRepo(0,
			fmt.Errorf("delete: %w", repository.ErrNotFound)),
			"/user?appUserId=9&updateTs=12.5")
		assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	})

	for _, target := range []string{
		"/user?updateTs=12.5", "/user?appUserId=9", "/user?appUserId=x",
	} {
		t.Run(target+" → 400, repository untouched", func(t *testing.T) {
			w := deleteUser(t, new(MockUserRepository), target)
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}

	t.Run("own login (alice, 7) → 400, repository untouched",
		func(t *testing.T) {
			repo := new(MockUserRepository)
			w := deleteUser(t, repo, "/user?appUserId=7&updateTs=12.5")
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), "your own login")
			assert.Empty(t, repo.Calls, "no repository call, no delete")
		})
}

func TestRetrieveUserList(t *testing.T) {
	retrieveUserList := func(
		t *testing.T, repo *MockUserRepository, target string,
	) *httptest.ResponseRecorder {
		t.Helper()
		s := testServices(t)
		handler := NewUserHandler(s, repo, new(MockAppUserGroupRepository))
		w := httptest.NewRecorder()
		handler.RetrieveUserList(w, userAdminRequest(t, s, http.MethodGet,
			target, "", model.GrantUserRead))
		repo.AssertExpectations(t)
		return w
	}

	t.Run("id list and afterUpdateTs reach the repository with the "+
		"JWT's tenant → 200", func(t *testing.T) {
		repo := new(MockUserRepository)
		repo.On("RetrieveUserList", mock.Anything, userAdminTenant,
			[]int{1, 9}, 5.5).Return(bobRow(), nil)
		w := retrieveUserList(t, repo,
			"/userList?appUserIdList=1,9&afterUpdateTs=5.5")
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), `"userName":"bob"`)
	})

	t.Run("nothing found → 200 with an empty list, not null",
		func(t *testing.T) {
			repo := new(MockUserRepository)
			repo.On("RetrieveUserList", mock.Anything, userAdminTenant,
				[]int{9}, float64(0)).
				Return(([]*model.AppUserModel)(nil), nil)
			w := retrieveUserList(t, repo, "/userList?appUserIdList=9")
			assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.JSONEq(t, `{"data":[]}`, w.Body.String())
		})

	t.Run("malformed id list → 400, repository untouched",
		func(t *testing.T) {
			w := retrieveUserList(t, new(MockUserRepository),
				"/userList?appUserIdList=1,x")
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})

	t.Run("repository error → 500 without its text", func(t *testing.T) {
		repo := new(MockUserRepository)
		repo.On("RetrieveUserList", mock.Anything, userAdminTenant,
			[]int{9}, float64(0)).
			Return(([]*model.AppUserModel)(nil), errors.New("conn reset"))
		w := retrieveUserList(t, repo, "/userList?appUserIdList=9")
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.NotContains(t, w.Body.String(), "conn reset")
	})
}

// TestRetrieveUserList_OmitsSecrets proves the /userList body — the
// DataWrapper the handler marshals — never carries the encrypted TOTP seed
// or the password hash.
func TestRetrieveUserList_OmitsSecrets(t *testing.T) {
	seed := "encrypted-seed"
	users := []*model.AppUserModel{
		{AppUserID: 1, UserName: "user1", SecurityContext: 1, PwHash: "hash", TOTPSeed: &seed},
	}
	body, err := json.Marshal(model.DataWrapper{Data: users})
	assert.NoError(t, err)
	// The body holds the row, so the absences below were looked for.
	assert.Contains(t, string(body), `"userName":"user1"`)
	assert.NotContains(t, string(body), "totpSeed")
	assert.NotContains(t, string(body), seed)
	assert.NotContains(t, string(body), "pwHash")
	assert.NotContains(t, string(body), "password")
	assert.NotContains(t, string(body), "hash")
	assert.Contains(t, string(body), `"userName":"user1"`)
}
