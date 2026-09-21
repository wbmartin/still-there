package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
)

type MockProfileRepository struct {
	mock.Mock
}

func (m *MockProfileRepository) RetrieveUserByID(
	ctx context.Context, userID int, securityContext int,
) (*model.AppUserModel, error) {
	args := m.Called(ctx, userID, securityContext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AppUserModel), args.Error(1)
}

func (m *MockProfileRepository) UpdateOwnProfile(
	ctx context.Context, userID int, securityContext int,
	firstName string, lastName string, email string,
	preference json.RawMessage, expectedUpdateTS float64,
) (float64, error) {
	args := m.Called(ctx, userID, securityContext, firstName, lastName,
		email, preference, expectedUpdateTS)
	return args.Get(0).(float64), args.Error(1)
}

// meRequest is a <method> /me for user 42 in security context 1 with the
// given grants and body, carrying the auth cookie and matching CSRF header.
func meRequest(
	t *testing.T, s *app.Services, method string, grants []int, body string,
) *http.Request {
	t.Helper()
	const sessionID = "session-abc"
	token, err := auth.GenerateJWT(
		"Test", "alice", 42, sessionID, 1, grants,
		s.Config.JWTSecret, s.Config.JWTSessionExpireHours,
	)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(method, "/me", strings.NewReader(body))
	r.AddCookie(&http.Cookie{Name: "auth", Value: token})
	r.Header.Set("X-CSRF-Token", sessionID[len(sessionID)-3:])
	return r
}

const validProfileBody = `{"firstName":"Alice","lastName":"Smith",` +
	`"email":"alice@example.com","preferenceJsonb":{"colorScheme":"dark"},` +
	`"updateTs":1.5}`

// TestProfileHandler_CheckSequence: the /me pair runs the same explicit
// chain as every handler, needs only LOGON, and reads the row of the
// user in the JWT, never one named by the client.
func TestProfileHandler_CheckSequence(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		request    func(t *testing.T, s *app.Services) *http.Request
		wantStatus int
		reachesDB  bool
	}{
		{
			name:   "no auth cookie → 401 from CheckJWT",
			method: http.MethodGet,
			request: func(t *testing.T, s *app.Services) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/me", nil)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "PUT without CSRF header → 403 from CheckCSRF",
			method: http.MethodPut,
			request: func(t *testing.T, s *app.Services) *http.Request {
				r := meRequest(t, s, http.MethodPut, []int{model.GrantLogon}, validProfileBody)
				r.Header.Del("X-CSRF-Token")
				return r
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "no LOGON grant → 403 from CheckAuthorization",
			method: http.MethodGet,
			request: func(t *testing.T, s *app.Services) *http.Request {
				return meRequest(t, s, http.MethodGet, []int{}, "")
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "GET with LOGON only → 200, row of the JWT's user",
			method: http.MethodGet,
			request: func(t *testing.T, s *app.Services) *http.Request {
				return meRequest(t, s, http.MethodGet, []int{model.GrantLogon}, "")
			},
			wantStatus: http.StatusOK,
			reachesDB:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServices(t)
			repo := new(MockProfileRepository)
			if tt.reachesDB {
				repo.On("RetrieveUserByID", mock.Anything, 42, 1).
					Return(&model.AppUserModel{AppUserID: 42, UserName: "alice",
						SecurityContext: 1, PwHash: "secret"}, nil)
			}
			h := NewProfileHandler(s, repo)

			w := httptest.NewRecorder()
			r := tt.request(t, s)
			if tt.method == http.MethodPut {
				h.UpdateOwnProfile(w, r)
			} else {
				h.RetrieveOwnProfile(w, r)
			}

			assert.Equal(t, tt.wantStatus, w.Code, w.Body.String())
			repo.AssertExpectations(t)
			if tt.wantStatus == http.StatusOK {
				assert.NotContains(t, w.Body.String(), "pwHash")
				assert.NotContains(t, w.Body.String(), "secret")
			}
		})
	}
}

func TestUpdateOwnProfile_ValidationAndLock(t *testing.T) {
	bad := []struct{ name, body string }{
		{"missing last name", `{"firstName":"A","lastName":" ","email":"a@b.c","updateTs":1}`},
		{"bad email", `{"firstName":"A","lastName":"B","email":"not-an-email","updateTs":1}`},
		{"display-name email", `{"firstName":"A","lastName":"B","email":"A B <a@b.c>","updateTs":1}`},
		{"no updateTs", `{"firstName":"A","lastName":"B","email":"a@b.c"}`},
		{"preference not an object", `{"firstName":"A","lastName":"B","email":"a@b.c","preferenceJsonb":[1],"updateTs":1}`},
		{"not json", `{`},
	}
	for _, tt := range bad {
		t.Run(tt.name+" → 400, repository untouched", func(t *testing.T) {
			s := testServices(t)
			repo := new(MockProfileRepository)
			w := httptest.NewRecorder()
			NewProfileHandler(s, repo).UpdateOwnProfile(w,
				meRequest(t, s, http.MethodPut, []int{model.GrantLogon}, tt.body))
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			repo.AssertExpectations(t)
		})
	}

	t.Run("valid → repository gets the JWT's user id and the trimmed fields", func(t *testing.T) {
		s := testServices(t)
		repo := new(MockProfileRepository)
		repo.On("UpdateOwnProfile", mock.Anything, 42, 1, "Alice", "Smith",
			"alice@example.com", json.RawMessage(`{"colorScheme":"dark"}`), 1.5).
			Return(2.5, nil)
		repo.On("RetrieveUserByID", mock.Anything, 42, 1).
			Return(&model.AppUserModel{AppUserID: 42, FirstName: "Alice", UpdateTS: 2.5}, nil)
		w := httptest.NewRecorder()
		body := strings.Replace(validProfileBody, `"firstName":"Alice"`, `"firstName":"  Alice "`, 1)
		NewProfileHandler(s, repo).UpdateOwnProfile(w,
			meRequest(t, s, http.MethodPut, []int{model.GrantLogon}, body))
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), `"updateTs":2.5`)
		repo.AssertExpectations(t)
	})

	t.Run("stale updateTs → 409", func(t *testing.T) {
		s := testServices(t)
		repo := new(MockProfileRepository)
		repo.On("UpdateOwnProfile", mock.Anything, 42, 1, "Alice", "Smith",
			"alice@example.com", mock.Anything, 1.5).
			Return(float64(0), repository.ErrConflict)
		w := httptest.NewRecorder()
		NewProfileHandler(s, repo).UpdateOwnProfile(w,
			meRequest(t, s, http.MethodPut, []int{model.GrantLogon}, validProfileBody))
		assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	})
}
