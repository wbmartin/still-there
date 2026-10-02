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
	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/model"
)

// A 32-byte AES key as hex, the shape TOTP_ENCRYPTION_KEY takes.
const testTOTPKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

// mfaUser is a non-exempt user whose seed is encrypted under testTOTPKey.
// The plain secret is returned so a test can mint a valid code.
func mfaUser(t *testing.T, mfaSetup bool) (*model.AppUserModel, string) {
	t.Helper()
	secret, err := auth.GenerateTOTPSecret()
	require.NoError(t, err)
	encrypted, err := auth.EncryptTOTPSeed(secret, testTOTPKey)
	require.NoError(t, err)
	return &model.AppUserModel{
		AppUserID: 7, UserName: "mfa", SecurityContext: 1,
		AppUserGroupMembershipList: []int{1},
		TOTPSeed:                   &encrypted, MFASetup: mfaSetup, UpdateTS: 1.5,
	}, secret
}

func mfaVerifyRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/mfaVerify", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// TestMFAVerifyHandler_TempTokenPath: the second step of login. No JWT
// exists yet, so the temp token from /auth is the credential; the rate
// limit still applies; a bad code counts a failed attempt; a good code
// marks MFA set up, issues the session cookie and burns the temp token.
func TestMFAVerifyHandler_TempTokenPath(t *testing.T) {
	newServices := func(t *testing.T) *app.Services {
		s := testServices(t)
		s.Config.TOTPEncryptionKey = testTOTPKey
		s.Config.AppName = "Test"
		return s
	}

	t.Run("unknown temp token → 401, repository never reached", func(t *testing.T) {
		s := newServices(t)
		repo := new(MockUserRepository)
		w := httptest.NewRecorder()
		MFAVerifyHandler(s, repo)(w, mfaVerifyRequest(`{"tempToken":"nope","totpCode":"000000"}`))
		assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
		repo.AssertExpectations(t)
	})

	t.Run("rate limit exceeded → 429 and the handler stops there", func(t *testing.T) {
		s := newServices(t)
		s.Config.MaxUnauthReqPerMin = 1
		require.NoError(t, s.KV.Set("TEMP_AUTH_SUCCESS_tok", "mfa", "10min"))
		repo := new(MockUserRepository)
		h := MFAVerifyHandler(s, repo)
		// First request spends the minute's allowance (and fails on the
		// code, which is fine); the second must be refused before any
		// user lookup.
		user, _ := mfaUser(t, true)
		repo.On("RetrieveUserByUserName", mock.Anything, "mfa").Return(user, nil).Once()
		h(httptest.NewRecorder(), mfaVerifyRequest(`{"tempToken":"tok","totpCode":"000000"}`))
		w := httptest.NewRecorder()
		h(w, mfaVerifyRequest(`{"tempToken":"tok","totpCode":"000000"}`))
		assert.Equal(t, http.StatusTooManyRequests, w.Code, w.Body.String())
		repo.AssertExpectations(t) // exactly one lookup, from the first call
	})

	t.Run("wrong code → 401, a failed attempt is counted, no session", func(t *testing.T) {
		s := newServices(t)
		require.NoError(t, s.KV.Set("TEMP_AUTH_SUCCESS_tok", "mfa", "10min"))
		user, _ := mfaUser(t, true)
		repo := new(MockUserRepository)
		repo.On("RetrieveUserByUserName", mock.Anything, "mfa").Return(user, nil)
		w := httptest.NewRecorder()
		MFAVerifyHandler(s, repo)(w, mfaVerifyRequest(`{"tempToken":"tok","totpCode":"000000"}`))
		assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
		count, err := s.KV.Get("loginfail-user-mfa")
		require.NoError(t, err)
		assert.Equal(t, "1", count)
		assert.Empty(t, w.Result().Cookies())
		stillThere, _ := s.KV.Get("TEMP_AUTH_SUCCESS_tok")
		assert.Equal(t, "mfa", stillThere, "temp token survives a wrong code")
	})

	t.Run("counter write error → 500, no session", func(t *testing.T) {
		s := newServices(t)
		// An unparseable window makes the store refuse the Incr (B008).
		s.Config.FailedLoginAttemptWindowDuration = "never"
		require.NoError(t, s.KV.Set("TEMP_AUTH_SUCCESS_tok", "mfa", "10min"))
		user, _ := mfaUser(t, true)
		repo := new(MockUserRepository)
		repo.On("RetrieveUserByUserName", mock.Anything, "mfa").Return(user, nil)
		w := httptest.NewRecorder()
		MFAVerifyHandler(s, repo)(w, mfaVerifyRequest(`{"tempToken":"tok","totpCode":"000000"}`))
		assert.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), "Login is temporarily unavailable")
		assert.Empty(t, w.Result().Cookies())
		assert.NotContains(t, w.Body.String(), `"csrfToken"`)
	})

	t.Run("valid code on first enrolment → mfa_setup set, cookie issued, temp token burnt", func(t *testing.T) {
		s := newServices(t)
		require.NoError(t, s.KV.Set("TEMP_AUTH_SUCCESS_tok", "mfa", "10min"))
		user, secret := mfaUser(t, false)
		code, err := totp.GenerateCode(secret, time.Now())
		require.NoError(t, err)
		repo := new(MockUserRepository)
		repo.On("RetrieveUserByUserName", mock.Anything, "mfa").Return(user, nil)
		repo.On("UpdateUserMFASetup", mock.Anything, 7, true, 1.5, 7).Return(2.5, nil)
		repo.On("GetGrantsForGroups", mock.Anything, []int{1}).
			Return(map[int][]int{1: {model.GrantLogon}}, nil)
		w := httptest.NewRecorder()
		MFAVerifyHandler(s, repo)(w, mfaVerifyRequest(`{"tempToken":"tok","totpCode":"`+code+`"}`))
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		repo.AssertExpectations(t)
		var authCookie *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == "auth" {
				authCookie = c
			}
		}
		require.NotNil(t, authCookie, "session cookie")
		assert.True(t, authCookie.HttpOnly)
		assert.Contains(t, w.Body.String(), `"csrfToken"`)
		burnt, _ := s.KV.Get("TEMP_AUTH_SUCCESS_tok")
		assert.Empty(t, burnt, "temp token is single-use")
	})

	t.Run("locked user → 429 before the code is looked at", func(t *testing.T) {
		s := newServices(t)
		require.NoError(t, s.KV.Set("TEMP_AUTH_SUCCESS_tok", "mfa", "10min"))
		require.NoError(t, s.KV.Set("loginlock-user-mfa", "1", "5min"))
		user, secret := mfaUser(t, true)
		code, _ := totp.GenerateCode(secret, time.Now())
		repo := new(MockUserRepository)
		repo.On("RetrieveUserByUserName", mock.Anything, "mfa").Return(user, nil)
		w := httptest.NewRecorder()
		MFAVerifyHandler(s, repo)(w, mfaVerifyRequest(`{"tempToken":"tok","totpCode":"`+code+`"}`))
		assert.Equal(t, http.StatusTooManyRequests, w.Code, w.Body.String())
		assert.Empty(t, w.Result().Cookies())
	})
}

// TestMFAVerifyHandler_JWTPath: with a session and no temp token the JWT
// is required, and a user who never completed setup is told so.
func TestMFAVerifyHandler_JWTPath(t *testing.T) {
	t.Run("no cookie, no temp token → 401 from CheckJWT", func(t *testing.T) {
		s := testServices(t)
		repo := new(MockUserRepository)
		w := httptest.NewRecorder()
		MFAVerifyHandler(s, repo)(w, mfaVerifyRequest(`{"totpCode":"000000"}`))
		assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
		repo.AssertExpectations(t)
	})

	t.Run("session but MFA never set up → 400", func(t *testing.T) {
		s := testServices(t)
		s.Config.TOTPEncryptionKey = testTOTPKey
		user, _ := mfaUser(t, false)
		repo := new(MockUserRepository)
		repo.On("RetrieveUserByUserName", mock.Anything, "mfa").Return(user, nil)
		token, err := auth.GenerateJWT("Test", "mfa", 7, "session-abc", 1,
			[]int{model.GrantLogon}, s.Config.JWTSecret, s.Config.JWTSessionExpireHours)
		require.NoError(t, err)
		r := mfaVerifyRequest(`{"totpCode":"000000"}`)
		r.AddCookie(&http.Cookie{Name: "auth", Value: token})
		w := httptest.NewRecorder()
		MFAVerifyHandler(s, repo)(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	})
}

// TestMFASetupHandler_TempTokenPath: /mfaSetup shows a secret only to a user
// still enrolling. An enrolled user is refused 409 with neither the secret
// nor a QR code; holding a temp token takes only the password.
func TestMFASetupHandler_TempTokenPath(t *testing.T) {
	newServices := func(t *testing.T) *app.Services {
		s := testServices(t)
		s.Config.TOTPEncryptionKey = testTOTPKey
		s.Config.AppName = "Test"
		require.NoError(t, s.KV.Set("TEMP_AUTH_SUCCESS_tok", "mfa", "10min"))
		return s
	}
	setupRequest := func() *http.Request {
		r := httptest.NewRequest(
			http.MethodPost, "/mfaSetup",
			strings.NewReader(`{"tempToken":"tok"}`),
		)
		r.Header.Set("Content-Type", "application/json")
		return r
	}

	t.Run("enrolled user → 409, no secret, no QR, seed untouched", func(t *testing.T) {
		s := newServices(t)
		user, secret := mfaUser(t, true)
		repo := new(MockUserRepository)
		repo.On("RetrieveUserByUserName", mock.Anything, "mfa").Return(user, nil)
		w := httptest.NewRecorder()
		MFASetupHandler(s, repo)(w, setupRequest())
		assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
		assert.NotContains(t, w.Body.String(), secret)
		assert.NotContains(t, w.Body.String(), "qrCodeImage")
		repo.AssertNotCalled(t, "UpdateUserTOTPSeed",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything)
	})

	t.Run("seed stored, never verified → 200 with the same secret", func(t *testing.T) {
		s := newServices(t)
		user, secret := mfaUser(t, false)
		repo := new(MockUserRepository)
		repo.On("RetrieveUserByUserName", mock.Anything, "mfa").Return(user, nil)
		w := httptest.NewRecorder()
		MFASetupHandler(s, repo)(w, setupRequest())
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), secret)
		assert.Contains(t, w.Body.String(), "qrCodeImage")
		repo.AssertNotCalled(t, "UpdateUserTOTPSeed",
			mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything)
	})

	t.Run("no seed → 200, a new seed is stored", func(t *testing.T) {
		s := newServices(t)
		user := &model.AppUserModel{
			AppUserID: 7, UserName: "mfa", SecurityContext: 1,
			AppUserGroupMembershipList: []int{1}, UpdateTS: 1.5,
		}
		repo := new(MockUserRepository)
		repo.On("RetrieveUserByUserName", mock.Anything, "mfa").Return(user, nil)
		repo.On("UpdateUserTOTPSeed", mock.Anything, 7, mock.Anything, 1.5, 7).
			Return(2.5, nil)
		w := httptest.NewRecorder()
		MFASetupHandler(s, repo)(w, setupRequest())
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), `"secret"`)
		repo.AssertExpectations(t)
	})
}
