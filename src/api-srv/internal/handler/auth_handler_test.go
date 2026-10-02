package handler

import (
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/wbmartin/starter-api/internal/repository"
)

// TestAuthHandler_MFAEnrolled: a user who owes a second factor gets a temp
// token and mfaEnrolled, which tells the web app to verify a code (true) or
// to enrol (false) without asking /mfaSetup.
func TestAuthHandler_MFAEnrolled(t *testing.T) {
	hash, err := auth.HashPassword("pw-123")
	require.NoError(t, err)
	seed := "ciphertext"
	testCaseList := []struct {
		name     string
		mfaSetup bool
		seed     *string
		want     bool
	}{
		{"verified seed → enrolled", true, &seed, true},
		{"seed not yet verified → enrol", false, &seed, false},
		{"no seed → enrol", false, nil, false},
	}
	for _, testCase := range testCaseList {
		t.Run(testCase.name, func(t *testing.T) {
			s := testServices(t)
			user := &model.AppUserModel{
				AppUserID: 7, UserName: "mfa", SecurityContext: 1,
				PwHash: hash, MFASetup: testCase.mfaSetup,
				TOTPSeed: testCase.seed,
			}
			repo := new(MockUserRepository)
			repo.On("RetrieveUserByUserName", mock.Anything, "mfa").
				Return(user, nil)
			r := httptest.NewRequest(
				http.MethodPost, "/auth",
				strings.NewReader(`{"userName":"mfa","password":"pw-123"}`),
			)
			w := httptest.NewRecorder()
			AuthHandler(s, repo)(w, r)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.Empty(t, w.Result().Cookies(), "no session before MFA")

			var wrapper model.AuthResponseWrapper
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wrapper))
			assert.NotEmpty(t, wrapper.User.TempToken)
			assert.Equal(t, testCase.want, wrapper.User.MFAEnrolled)
			assert.Contains(t, w.Body.String(), `"mfaEnrolled"`)
		})
	}
}

// refreshOnce calls /refresh with token and returns the new cookie's session
// id and the csrfToken in the body.
func refreshOnce(
	t *testing.T,
	handler http.HandlerFunc,
	jwtSecret string,
	token string,
) (string, string) {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/refresh", nil)
	r.AddCookie(&http.Cookie{Name: "auth", Value: token})
	w := httptest.NewRecorder()
	handler(w, r)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	claims, err := auth.ValidateJWT(authCookie(t, w), jwtSecret)
	require.NoError(t, err)
	var body model.Auth2ResponseWrapper
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return claims.SessionID, body.CsrfToken
}

// Every tab shares the cookie but holds its own CSRF token: a refresh in one
// tab must not change the token another tab sends (B004).
func TestRefresh_TwiceKeepsTheSessionAndCsrfToken(t *testing.T) {
	s := testServices(t)
	repo := new(MockUserRepository)
	repo.On("RetrieveUserByUserName", mock.Anything, "alice").
		Return(&model.AppUserModel{
			AppUserID: 1, UserName: "alice", SecurityContext: 1,
			AppUserGroupMembershipList: []int{1}, MFAExempt: true,
		}, nil)
	repo.On("GetGrantsForGroups", mock.Anything, []int{1}).
		Return(map[int][]int{1: {model.GrantLogon}}, nil)
	handler := RefreshHandler(s, repo)
	secret := s.Config.JWTSecret

	token := sessionToken(t, s, "alice", 1, "session-abc", nil)
	firstID, firstCsrf := refreshOnce(t, handler, secret, token)
	secondID, secondCsrf := refreshOnce(t, handler, secret, token)
	assert.Equal(t, "session-abc", firstID)
	assert.Equal(t, "session-abc", secondID)
	assert.Equal(t, "abc", firstCsrf)
	assert.Equal(t, "abc", secondCsrf)

	short := sessionToken(t, s, "alice", 1, "ab", nil)
	shortID, shortCsrf := refreshOnce(t, handler, secret, short)
	assert.Len(t, shortID, 32, "an id too short to slice is replaced")
	assert.Equal(t, shortID[len(shortID)-3:], shortCsrf)
}

// authAttempt posts one login from remoteAddr and returns the response.
func authAttempt(
	t *testing.T,
	handler http.HandlerFunc,
	remoteAddr, userName, password string,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"userName": userName, "password": password,
	})
	require.NoError(t, err)
	r := httptest.NewRequest(
		http.MethodPost, "/auth", strings.NewReader(string(body)),
	)
	r.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	handler(w, r)
	return w
}

// lockoutServices is testServices with the login lockout configured.
func lockoutServices(t *testing.T) *app.Services {
	t.Helper()
	s := testServices(t)
	s.Config.FailedLoginAttemptThreshold = 7
	s.Config.FailedLoginAttemptWindowDuration = "5min"
	s.Config.FailedLoginLockoutDuration = "5min"
	return s
}

// A name with no login gets the wrong-password answer and counts toward both
// lockouts, so probing names is throttled and the answer does not say which
// names exist (B007).
func TestAuthHandler_UnknownLoginIsCountedLikeABadPassword(t *testing.T) {
	s := lockoutServices(t)
	hash, err := auth.HashPassword("right")
	require.NoError(t, err)
	repo := new(MockUserRepository)
	repo.On("RetrieveUserByUserName", mock.Anything, "alice").
		Return(&model.AppUserModel{
			AppUserID: 1, UserName: "alice", PwHash: hash,
		}, nil)
	repo.On("RetrieveUserByUserName", mock.Anything, mock.Anything).
		Return(
			(*model.AppUserModel)(nil),
			fmt.Errorf("user not found: %w", repository.ErrNotFound),
		)
	handler := AuthHandler(s, repo)

	wrongPassword := authAttempt(t, handler, "192.0.2.1:1", "alice", "wrong")
	require.Equal(t, http.StatusUnauthorized, wrongPassword.Code)

	threshold := s.Config.FailedLoginAttemptThreshold
	for attempt := 1; attempt <= threshold; attempt++ {
		w := authAttempt(t, handler, "192.0.2.22:1234", "nobody", "guess")
		require.Equal(t, http.StatusUnauthorized, w.Code, "attempt %d", attempt)
		assert.Equal(t, wrongPassword.Body.String(), w.Body.String(),
			"attempt %d answers like a wrong password", attempt)
	}

	w := authAttempt(t, handler, "192.0.2.22:1234", "nobody", "guess")
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "eighth attempt")
	w = authAttempt(t, handler, "192.0.2.99:1234", "nobody", "guess")
	assert.Equal(t, http.StatusTooManyRequests, w.Code,
		"the unknown name is locked from any address")
	w = authAttempt(t, handler, "192.0.2.22:1234", "someone-else", "guess")
	assert.Equal(t, http.StatusTooManyRequests, w.Code,
		"the address is locked for any name")
}

// A lookup that fails for any reason but not-found is an outage: 500, not
// "Invalid credentials", and it is not counted against the caller.
func TestAuthHandler_LookupErrorIs500(t *testing.T) {
	s := lockoutServices(t)
	repo := new(MockUserRepository)
	repo.On("RetrieveUserByUserName", mock.Anything, "alice").
		Return(
			(*model.AppUserModel)(nil),
			fmt.Errorf("failed to get user: %w", errors.New("conn reset")),
		)
	handler := AuthHandler(s, repo)

	w := authAttempt(t, handler, "192.0.2.5:1", "alice", "pw")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Login is temporarily unavailable")
	assert.NotContains(t, w.Body.String(), "conn reset")
	count, err := s.KV.Get("loginfail-user-alice")
	require.NoError(t, err)
	assert.Empty(t, count, "an outage is not a failed attempt")
}
