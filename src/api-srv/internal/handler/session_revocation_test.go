package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/security"
)

// sessionToken signs a JWT for userName (id userID, context 1).
func sessionToken(
	t *testing.T,
	s *app.Services,
	userName string,
	userID int,
	sessionID string,
	grantList []int,
) string {
	t.Helper()
	token, err := auth.GenerateJWT(
		"Test", userName, userID, sessionID, 1, grantList,
		s.Config.JWTSecret, s.Config.JWTSessionExpireHours,
	)
	require.NoError(t, err)
	return token
}

// sessionRequest builds a request carrying token and its CSRF tail.
func sessionRequest(
	method string,
	target string,
	body string,
	token string,
	sessionID string,
) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.AddCookie(&http.Cookie{Name: "auth", Value: token})
	r.Header.Set("X-CSRF-Token", sessionID[len(sessionID)-3:])
	return r
}

// jwtStatus is what CheckJWT answers for token: 200 when it passes.
func jwtStatus(t *testing.T, s *app.Services, token string) int {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/me", nil)
	r.AddCookie(&http.Cookie{Name: "auth", Value: token})
	w := httptest.NewRecorder()
	security.NewAppHelper(w, r, s, "test").CheckJWT()
	return w.Code
}

func authCookie(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "auth" {
			return cookie.Value
		}
	}
	t.Fatal("no auth cookie set")
	return ""
}

func TestLogoff_EndsTheSession(t *testing.T) {
	s := testServices(t)
	token := sessionToken(t, s, "alice", 1, "session-one", nil)
	otherDevice := sessionToken(t, s, "alice", 1, "session-two", nil)

	w := httptest.NewRecorder()
	LogoffHandler(s)(w, sessionRequest(
		http.MethodPost, "/logoff", "", token, "session-one"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assert.Equal(t, http.StatusUnauthorized, jwtStatus(t, s, token),
		"a copy of the signed-out token is refused")
	assert.Equal(t, http.StatusOK, jwtStatus(t, s, otherDevice),
		"another session of the login is untouched")

	w = httptest.NewRecorder()
	LogoffHandler(s)(w, sessionRequest(
		http.MethodPost, "/logoff", "", token, "session-one"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefresh_KeepsTheSessionSoLogoffEndsBoth(t *testing.T) {
	s := testServices(t)
	user := &model.AppUserModel{
		AppUserID: 1, UserName: "alice", SecurityContext: 1,
		AppUserGroupMembershipList: []int{1}, MFAExempt: true,
	}
	repo := new(MockUserRepository)
	repo.On("RetrieveUserByUserName", mock.Anything, "alice").
		Return(user, nil)
	repo.On("GetGrantsForGroups", mock.Anything, []int{1}).
		Return(map[int][]int{1: {model.GrantLogon}}, nil)
	original := sessionToken(t, s, "alice", 1, "session-one", nil)

	w := httptest.NewRecorder()
	RefreshHandler(s, repo)(w, sessionRequest(
		http.MethodGet, "/refresh", "", original, "session-one"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	refreshed := authCookie(t, w)
	claims, err := auth.ValidateJWT(refreshed, s.Config.JWTSecret)
	require.NoError(t, err)
	assert.Equal(t, "session-one", claims.SessionID)
	assert.Contains(t, w.Body.String(), `"csrfToken":"one"`)

	w = httptest.NewRecorder()
	LogoffHandler(s)(w, sessionRequest(
		http.MethodPost, "/logoff", "", refreshed, "session-one"))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.StatusUnauthorized, jwtStatus(t, s, original))
	assert.Equal(t, http.StatusUnauthorized, jwtStatus(t, s, refreshed))
}

func TestChangeSecurityContext_EndsThePreviousSession(t *testing.T) {
	s := testServices(t)
	user := &model.AppUserModel{
		AppUserID: 1, UserName: "alice", SecurityContext: 1,
		AppUserGroupMembershipList: []int{1, 4},
	}
	repo := new(MockUserRepository)
	repo.On("RetrieveUserByUserName", mock.Anything, "alice").
		Return(user, nil)
	repo.On("GetGrantsForGroups", mock.Anything, []int{1, 4}).
		Return(map[int][]int{
			1: {model.GrantLogon}, 2: {model.GrantLogon},
		}, nil)
	original := sessionToken(t, s, "alice", 1, "session-one", nil)

	w := httptest.NewRecorder()
	ChangeSecurityContextHandler(s, repo)(w, sessionRequest(
		http.MethodPost, "/changeSecurityContext",
		`{"securityContext":2}`, original, "session-one"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	switched := authCookie(t, w)

	assert.Equal(t, http.StatusUnauthorized, jwtStatus(t, s, original))
	assert.Equal(t, http.StatusOK, jwtStatus(t, s, switched))
}

func TestChangeOwnPassword_EndsTheOtherSessions(t *testing.T) {
	s := testServices(t)
	hash, err := auth.HashPassword("old-pass-1")
	require.NoError(t, err)
	repo := new(MockUserRepository)
	repo.On("RetrieveUserByUserName", mock.Anything, "alice").
		Return(&model.AppUserModel{AppUserID: 1, UserName: "alice",
			PwHash: hash, SecurityContext: 1}, nil)
	repo.On("UpdateUserPassword",
		mock.Anything, 1, mock.Anything, mock.Anything, 1, 1).Return(nil)
	grantList := []int{model.GrantChangeOwnPassword}
	current := sessionToken(t, s, "alice", 1, "session-one", grantList)
	otherDevice := sessionToken(t, s, "alice", 1, "session-two", grantList)

	w := httptest.NewRecorder()
	ChangeOwnPasswordHandler(s, repo)(w, sessionRequest(
		http.MethodPost, "/changeOwnPassword",
		`{"oldPassword":"old-pass-1","newPassword":"new-pass-2"}`,
		current, "session-one"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assert.Equal(t, http.StatusOK, jwtStatus(t, s, current),
		"the session that changed the password stays signed in")
	assert.Equal(t, http.StatusUnauthorized, jwtStatus(t, s, otherDevice))
}

func TestChangeOwnPassword_WrongOldPasswordRevokesNothing(t *testing.T) {
	s := testServices(t)
	hash, err := auth.HashPassword("old-pass-1")
	require.NoError(t, err)
	repo := new(MockUserRepository)
	repo.On("RetrieveUserByUserName", mock.Anything, "alice").
		Return(&model.AppUserModel{AppUserID: 1, UserName: "alice",
			PwHash: hash}, nil)
	grantList := []int{model.GrantChangeOwnPassword}
	current := sessionToken(t, s, "alice", 1, "session-one", grantList)
	otherDevice := sessionToken(t, s, "alice", 1, "session-two", grantList)

	w := httptest.NewRecorder()
	ChangeOwnPasswordHandler(s, repo)(w, sessionRequest(
		http.MethodPost, "/changeOwnPassword",
		`{"oldPassword":"wrong-pass","newPassword":"new-pass-2"}`,
		current, "session-one"))
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, http.StatusOK, jwtStatus(t, s, otherDevice))
}

func TestChangeOtherUserPassword_EndsTheirSessions(t *testing.T) {
	s := testServices(t)
	repo := new(MockUserRepository)
	repo.On("RetrieveUserList", mock.Anything, 1, []int{9}, float64(0)).
		Return([]*model.AppUserModel{{AppUserID: 9, UserName: "bob",
			SecurityContext: 1}}, nil)
	repo.On("UpdateUserPassword",
		mock.Anything, 9, mock.Anything, mock.Anything, 1, 1).Return(nil)
	admin := sessionToken(t, s, "alice", 1, "session-admin",
		[]int{model.GrantChangeOtherUserPassword})
	bob := sessionToken(t, s, "bob", 9, "session-bob", nil)
	handler := NewUserHandler(s, repo, new(MockAppUserGroupRepository))

	w := httptest.NewRecorder()
	handler.ChangeOtherUserPassword(w, sessionRequest(
		http.MethodPost, "/changeOtherUserPassword",
		`{"appUserId":9,"newPassword":"new-pass-2"}`,
		admin, "session-admin"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assert.Equal(t, http.StatusUnauthorized, jwtStatus(t, s, bob))
	assert.Equal(t, http.StatusOK, jwtStatus(t, s, admin))
}

func TestRevokeUserSessionList_KVErrorIs500(t *testing.T) {
	s := testServices(t)
	r := httptest.NewRequest(http.MethodPost, "/user", nil)
	w := httptest.NewRecorder()
	helper := security.NewAppHelper(w, r, s, "test")
	s.KV.Close()
	assert.False(t, revokeUserSessionList(helper, w, s, "", "bob"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
