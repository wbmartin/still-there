package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"stillthere-api/internal/auth"
	"stillthere-api/internal/model"
)

// authSetCookie returns the raw Set-Cookie header that writes the auth
// cookie, failing the test when there is none.
func authSetCookie(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	for _, header := range w.Result().Header.Values("Set-Cookie") {
		if strings.HasPrefix(header, authCookieName+"=") {
			return header
		}
	}
	t.Fatal("no auth Set-Cookie header")
	return ""
}

// assertSessionCookie: the cookie is a browser-session cookie (no Max-Age,
// no Expires), so the JWT's exp is the session's only lifetime.
func assertSessionCookie(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	header := authSetCookie(t, w)
	assert.NotContains(t, header, "Max-Age", header)
	assert.NotContains(t, header, "Expires", header)
	assertCookieFlags(t, header)
}

func assertCookieFlags(t *testing.T, header string) {
	t.Helper()
	for _, flag := range []string{
		"Path=/", "HttpOnly", "Secure", "SameSite=Strict",
	} {
		assert.Contains(t, header, flag, header)
	}
}

// aliceRepo answers alice's lookups for every path that issues a token.
func aliceRepo(hash string) *MockUserRepository {
	repo := new(MockUserRepository)
	repo.On("RetrieveUserByUserName", mock.Anything, "alice").
		Return(&model.AppUserModel{
			AppUserID: 1, UserName: "alice", PwHash: hash,
			SecurityContext: 1, AppUserGroupMembershipList: []int{1, 4},
			MFAExempt: true, AccountStatus: model.AccountStatusActive,
		}, nil)
	repo.On("GetGrantsForGroups", mock.Anything, []int{1, 4}).
		Return(map[int][]int{
			1: {model.GrantLogon}, 2: {model.GrantLogon},
		}, nil)
	return repo
}

func TestAuthCookie_LoginIsASessionCookie(t *testing.T) {
	hash, err := auth.HashPassword("right")
	require.NoError(t, err)
	s := testServices(t)
	w := authAttempt(t, AuthHandler(s, aliceRepo(hash)),
		"192.0.2.5:1", "alice", "right")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assertSessionCookie(t, w)
}

func TestAuthCookie_RefreshIsASessionCookie(t *testing.T) {
	s := testServices(t)
	token := sessionToken(t, s, "alice", 1, "session-one", nil)
	w := httptest.NewRecorder()
	RefreshHandler(s, aliceRepo(""))(w, sessionRequest(
		http.MethodGet, "/refresh", "", token, "session-one"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assertSessionCookie(t, w)
}

func TestAuthCookie_ChangeSecurityContextIsASessionCookie(t *testing.T) {
	s := testServices(t)
	token := sessionToken(t, s, "alice", 1, "session-one", nil)
	w := httptest.NewRecorder()
	ChangeSecurityContextHandler(s, aliceRepo(""))(w, sessionRequest(
		http.MethodPost, "/changeSecurityContext",
		`{"securityContext":2}`, token, "session-one"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assertSessionCookie(t, w)
}

func TestAuthCookie_MFAVerifyIsASessionCookie(t *testing.T) {
	s := testServices(t)
	s.Config.TOTPEncryptionKey = testTOTPKey
	s.Config.AppName = "Test"
	require.NoError(t, s.KV.Set("TEMP_AUTH_SUCCESS_tok", "mfa", "10min"))
	user, secret := mfaUser(t, true)
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)
	repo := new(MockUserRepository)
	repo.On("RetrieveUserByUserName", mock.Anything, "mfa").Return(user, nil)
	repo.On("GetGrantsForGroups", mock.Anything, []int{1}).
		Return(map[int][]int{1: {model.GrantLogon}}, nil)
	w := httptest.NewRecorder()
	MFAVerifyHandler(s, repo)(w, mfaVerifyRequest(
		`{"tempToken":"tok","totpCode":"`+code+`"}`))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	repo.AssertExpectations(t)
	assertSessionCookie(t, w)
}

func TestAuthCookie_LogoffExpiresIt(t *testing.T) {
	s := testServices(t)
	token := sessionToken(t, s, "alice", 1, "session-one", nil)
	w := httptest.NewRecorder()
	LogoffHandler(s)(w, sessionRequest(
		http.MethodPost, "/logoff", "", token, "session-one"))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	header := authSetCookie(t, w)
	assert.True(t, strings.HasPrefix(header, authCookieName+"=;"), header)
	assert.Contains(t, header, "Max-Age=0", header)
	assertCookieFlags(t, header)
}
