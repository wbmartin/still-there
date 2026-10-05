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
				TOTPSeed:      testCase.seed,
				AccountStatus: model.AccountStatusActive,
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

// A failure that cannot be counted is an outage, not a bad password: a
// lockout that never trips is no lockout (B008). An unparseable window makes
// the in-process store refuse the Incr, standing in for any write error.
func TestAuthHandler_CounterWriteErrorIs500(t *testing.T) {
	s := lockoutServices(t)
	s.Config.FailedLoginAttemptWindowDuration = "never"
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

	for _, userName := range []string{"alice", "nobody"} {
		w := authAttempt(t, handler, "192.0.2.7:1", userName, "wrong")
		assert.Equal(t, http.StatusInternalServerError, w.Code, userName)
		assert.Contains(t, w.Body.String(), "Login is temporarily unavailable")
		assert.NotContains(t, w.Body.String(), "invalid expiration")
		assert.Empty(t, w.Result().Cookies(), userName)
	}
}

// The lockout read fails closed as well: no store, no login attempt.
func TestAuthHandler_LockoutReadErrorIs500(t *testing.T) {
	s := lockoutServices(t)
	s.KV.Close()
	repo := new(MockUserRepository)
	handler := AuthHandler(s, repo)

	w := authAttempt(t, handler, "192.0.2.8:1", "alice", "pw")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, w.Result().Cookies())
	repo.AssertNotCalled(t, "RetrieveUserByUserName",
		mock.Anything, mock.Anything)
}

// A completed login clears the username's failure counter, so earlier typos
// do not combine with later ones into a lockout; the address's counter runs
// out its window. The password step of an MFA login completes nothing and
// clears nothing (B010).
func TestAuthHandler_CompletedLoginClearsTheUserCounterOnly(t *testing.T) {
	hash, err := auth.HashPassword("right")
	require.NoError(t, err)
	testCaseList := []struct {
		name        string
		mfaExempt   bool
		wantCleared bool
	}{
		{"MFA-exempt login completes at /auth", true, true},
		{"MFA login owes a code, counter kept", false, false},
	}
	for _, testCase := range testCaseList {
		t.Run(testCase.name, func(t *testing.T) {
			s := lockoutServices(t)
			repo := new(MockUserRepository)
			repo.On("RetrieveUserByUserName", mock.Anything, "alice").
				Return(&model.AppUserModel{
					AppUserID: 1, UserName: "alice", PwHash: hash,
					SecurityContext: 1, AppUserGroupMembershipList: []int{1},
					MFAExempt:     testCase.mfaExempt,
					AccountStatus: model.AccountStatusActive,
				}, nil)
			repo.On("GetGrantsForGroups", mock.Anything, []int{1}).
				Return(map[int][]int{1: {model.GrantLogon}}, nil)
			handler := AuthHandler(s, repo)

			for attempt := 1; attempt <= 6; attempt++ {
				w := authAttempt(t, handler, "192.0.2.5:1", "alice", "typo")
				require.Equal(t, http.StatusUnauthorized, w.Code)
			}
			w := authAttempt(t, handler, "192.0.2.5:1", "alice", "right")
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())

			userCount, err := s.KV.Get("loginfail-user-alice")
			require.NoError(t, err)
			ipCount, err := s.KV.Get("loginfail-ip-192.0.2.5")
			require.NoError(t, err)
			assert.Equal(t, "6", ipCount, "the address counter is kept")
			if !testCase.wantCleared {
				assert.Equal(t, "6", userCount)
				return
			}
			assert.Empty(t, userCount)

			// A later typo starts a fresh count rather than locking the user.
			w = authAttempt(t, handler, "192.0.2.6:1", "alice", "typo")
			require.Equal(t, http.StatusUnauthorized, w.Code)
			userCount, err = s.KV.Get("loginfail-user-alice")
			require.NoError(t, err)
			assert.Equal(t, "1", userCount)
			w = authAttempt(t, handler, "192.0.2.6:1", "alice", "right")
			assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		})
	}
}

// A locked address or name is answered 429 before the repository and before
// Argon2id: the lockout is /auth's stand-in for the rate limit
// (route_check_contract_test.go).
func TestAuthHandler_LockoutAnswersBeforeTheRepository(t *testing.T) {
	for _, lockKey := range []string{
		"loginlock-ip-192.0.2.30", "loginlock-user-alice",
	} {
		t.Run(lockKey, func(t *testing.T) {
			s := lockoutServices(t)
			require.NoError(t, s.KV.Set(lockKey, "1", "5min"))
			repo := new(MockUserRepository) // unprimed: a lookup panics
			w := authAttempt(t, AuthHandler(s, repo),
				"192.0.2.30:1", "alice", "right")
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
			assert.Contains(t, w.Body.String(), "temporarily locked")
			assert.Empty(t, w.Result().Cookies())
		})
	}
}

// The wrong-password answer is pinned byte for byte, and a login that is
// not active (invited, B035; blocked, B034) gets exactly that answer with
// the right password, counted, with no session. The unknown-name path (B007)
// is proved equal to it in
// TestAuthHandler_UnknownLoginIsCountedLikeABadPassword.
func TestAuthHandler_WrongPasswordAnswer(t *testing.T) {
	hash, err := auth.HashPassword("right")
	require.NoError(t, err)
	testCaseList := []struct {
		name, status, password string
	}{
		{"active login, wrong password", model.AccountStatusActive, "wrong"},
		{"invited login, right password", model.AccountStatusInvited,
			"right"},
		{"blocked login, right password", model.AccountStatusBlocked,
			"right"},
	}
	for _, testCase := range testCaseList {
		t.Run(testCase.name, func(t *testing.T) {
			s := lockoutServices(t)
			repo := new(MockUserRepository)
			repo.On("RetrieveUserByUserName", mock.Anything, "alice").
				Return(&model.AppUserModel{
					AppUserID: 1, UserName: "alice", PwHash: hash,
					SecurityContext: 2, MFAExempt: true,
					AccountStatus: testCase.status,
				}, nil)
			w := authAttempt(t, AuthHandler(s, repo),
				"192.0.2.31:1", "alice", testCase.password)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Equal(t, "{\"error\":\"Invalid credentials\"}\n",
				w.Body.String())
			assert.Empty(t, w.Result().Cookies())
			count, err := s.KV.Get("loginfail-user-alice")
			require.NoError(t, err)
			assert.Equal(t, "1", count, "the failure is counted")
			repo.AssertExpectations(t) // and no grant lookup
		})
	}
}
