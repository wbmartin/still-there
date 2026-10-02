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
