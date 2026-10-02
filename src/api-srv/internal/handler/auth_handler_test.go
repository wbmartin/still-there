package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/model"
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
