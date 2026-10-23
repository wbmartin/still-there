package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stillthere-api/internal/app"
	"stillthere-api/internal/auth"
	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
	"stillthere-api/internal/security"
)

// B034: POST /admin/blockUser and /admin/unblockUser. Logins live in the
// test tenant (2); "stranger" is in 3.

const blockTenant = 2

// fakeBlockRepository holds logins by name and records status writes.
type fakeBlockRepository struct {
	userByName      map[string]*model.AppUserModel
	statusWriteList []string
}

func (f *fakeBlockRepository) RetrieveUserByUserName(
	_ context.Context, username string,
) (*model.AppUserModel, error) {
	user, ok := f.userByName[username]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return user, nil
}

func (f *fakeBlockRepository) UpdateUserAccountStatus(
	_ context.Context, userID int, accountStatus string,
	_ float64, _ int, _ int,
) (float64, error) {
	for _, user := range f.userByName {
		if user.AppUserID == userID {
			user.AccountStatus = accountStatus
			f.statusWriteList = append(f.statusWriteList,
				user.UserName+"="+accountStatus)
		}
	}
	return 2, nil
}

func newFakeBlockRepository() *fakeBlockRepository {
	login := func(
		id int, name string, status string, securityContext int,
	) *model.AppUserModel {
		return &model.AppUserModel{AppUserID: id, UserName: name,
			AccountStatus: status, SecurityContext: securityContext,
			UpdateTS: 1}
	}
	return &fakeBlockRepository{userByName: map[string]*model.AppUserModel{
		"blocker": login(7, "blocker", model.AccountStatusActive,
			blockTenant),
		"target": login(8, "target", model.AccountStatusActive,
			blockTenant),
		"jailed": login(9, "jailed", model.AccountStatusBlocked,
			blockTenant),
		"stranger": login(10, "stranger", model.AccountStatusActive, 3),
	}}
}

// blockSessionToken signs a JWT for userName in the test tenant.
func blockSessionToken(
	t *testing.T,
	appServices *app.Services,
	userName string,
	userID int,
	sessionID string,
	grantList []int,
) string {
	t.Helper()
	token, err := auth.GenerateJWT(
		"Test", userName, userID, sessionID, blockTenant, grantList,
		appServices.Config.JWTSecret,
		appServices.Config.JWTSessionExpireHours,
	)
	require.NoError(t, err)
	return token
}

// blockRequest posts body as the login "blocker" holding grantList.
func blockRequest(
	t *testing.T,
	appServices *app.Services,
	userBlockRepository UserBlockRepository,
	block bool,
	body string,
	grantList []int,
) *httptest.ResponseRecorder {
	t.Helper()
	target := "/admin/unblockUser"
	if block {
		target = "/admin/blockUser"
	}
	token := blockSessionToken(t, appServices, "blocker", 7,
		"session-blk", grantList)
	w := httptest.NewRecorder()
	AdminBlockUserHandler(appServices, userBlockRepository, block)(
		w, sessionRequest(http.MethodPost, target, body, token,
			"session-blk"))
	return w
}

// liveSessionStatus is what the first two checks of every signed-in route
// answer for token: 200 when both pass.
func liveSessionStatus(
	t *testing.T, appServices *app.Services, token string,
) int {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/me", nil)
	r.AddCookie(&http.Cookie{Name: "auth", Value: token})
	w := httptest.NewRecorder()
	security.NewAppHelper(w, r, appServices, "test").
		CheckJWT().
		CheckRateLimitAndUserBlock()
	return w.Code
}

func TestAdminBlockUser(t *testing.T) {
	grantList := []int{model.GrantLogon, model.GrantUserBlock}

	t.Run("without BLOCK_USER is 403 and writes nothing", func(t *testing.T) {
		appServices := testServices(t)
		repo := newFakeBlockRepository()
		w := blockRequest(t, appServices, repo, true,
			`{"userName":"target"}`, []int{model.GrantLogon})
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Empty(t, repo.statusWriteList)
	})

	t.Run("block ends the live session, survives a restart, unblock "+
		"does not revive it", func(t *testing.T) {
		appServices := testServices(t)
		repo := newFakeBlockRepository()
		preBlock := blockSessionToken(t, appServices, "target", 8,
			"session-tgt", nil)
		require.Equal(t, http.StatusOK,
			liveSessionStatus(t, appServices, preBlock))

		w := blockRequest(t, appServices, repo, true,
			`{"userName":"target"}`, grantList)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), `"accountStatus":"blocked"`)
		assert.Equal(t, model.AccountStatusBlocked,
			repo.userByName["target"].AccountStatus)
		flag, err := appServices.KV.Get(blockedKey("target"))
		require.NoError(t, err)
		assert.NotEmpty(t, flag)
		assert.Equal(t, http.StatusUnauthorized,
			liveSessionStatus(t, appServices, preBlock),
			"the session is revoked")

		// A restart loses revocations; the reloaded flag still refuses.
		restarted := testServices(t)
		require.NoError(t, LoadBlockedUsers(restarted, []string{"target"}))
		assert.Equal(t, http.StatusForbidden,
			liveSessionStatus(t, restarted, preBlock))

		w = blockRequest(t, appServices, repo, false,
			`{"userName":"target"}`, grantList)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, model.AccountStatusActive,
			repo.userByName["target"].AccountStatus)
		flag, err = appServices.KV.Get(blockedKey("target"))
		require.NoError(t, err)
		assert.Empty(t, flag, "flag cleared on unblock")
		assert.Equal(t, http.StatusUnauthorized,
			liveSessionStatus(t, appServices, preBlock),
			"unblock does not revive a pre-block session")
		postUnblock := blockSessionToken(t, appServices, "target", 8,
			"session-new", nil)
		assert.Equal(t, http.StatusOK,
			liveSessionStatus(t, appServices, postUnblock))
	})

	refusalList := []struct {
		name       string
		block      bool
		body       string
		wantStatus int
	}{
		{"own login is 400", true, `{"userName":"blocker"}`,
			http.StatusBadRequest},
		{"no userName is 400", true, `{}`, http.StatusBadRequest},
		{"another tenant's login is 404", true, `{"userName":"stranger"}`,
			http.StatusNotFound},
		{"an unknown login is 404", true, `{"userName":"nobody"}`,
			http.StatusNotFound},
		{"blocking a blocked login is 409", true, `{"userName":"jailed"}`,
			http.StatusConflict},
		{"unblocking an active login is 409", false,
			`{"userName":"target"}`, http.StatusConflict},
	}
	for _, testCase := range refusalList {
		t.Run(testCase.name, func(t *testing.T) {
			appServices := testServices(t)
			repo := newFakeBlockRepository()
			w := blockRequest(t, appServices, repo, testCase.block,
				testCase.body, grantList)
			assert.Equal(t, testCase.wantStatus, w.Code, w.Body.String())
			assert.Empty(t, repo.statusWriteList)
		})
	}
}

// A restart empties the in-process KV; LoadBlockedUsers puts the flags back.
func TestLoadBlockedUsers(t *testing.T) {
	appServices := testServices(t)
	require.NoError(t, LoadBlockedUsers(appServices, []string{"a", "b"}))
	for _, userName := range []string{"a", "b"} {
		flag, err := appServices.KV.Get(blockedKey(userName))
		require.NoError(t, err)
		assert.NotEmpty(t, flag, userName)
	}
}

// /auth refuses a blocked login with the right password exactly as a wrong
// password: 401 "Invalid credentials", and the failure is counted.
func TestAuthHandler_BlockedLoginIsRefusedLikeABadPassword(t *testing.T) {
	s := lockoutServices(t)
	hash, err := auth.HashPassword("right")
	require.NoError(t, err)
	repo := new(MockUserRepository)
	repo.On("RetrieveUserByUserName", mock.Anything, "alice").
		Return(&model.AppUserModel{
			AppUserID: 1, UserName: "alice", PwHash: hash,
			SecurityContext: blockTenant, MFAExempt: true,
			AccountStatus: model.AccountStatusBlocked,
		}, nil)
	handler := AuthHandler(s, repo)

	w := authAttempt(t, handler, "192.0.2.9:1", "alice", "right")
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "Invalid credentials"))
	assert.Empty(t, w.Result().Cookies(), "no session for a blocked login")
	userCount, err := s.KV.Get("loginfail-user-alice")
	require.NoError(t, err)
	assert.Equal(t, "1", userCount, "the failure is counted")
	repo.AssertNotCalled(t, "GetGrantsForGroups", mock.Anything,
		mock.Anything)
}
