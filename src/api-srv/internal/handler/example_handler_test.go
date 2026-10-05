package handler

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/cache"
	"github.com/wbmartin/starter-api/internal/config"
	"github.com/wbmartin/starter-api/internal/logging"
	"github.com/wbmartin/starter-api/internal/model"
)

// MockExampleRepository satisfies ExampleRepositoryInterface. Tests set
// expectations only for the calls a request should reach; an unexpected
// call panics, which is the point: the check sequence must stop a bad
// request before the repository.
type MockExampleRepository struct {
	mock.Mock
}

func (m *MockExampleRepository) RetrieveExample(
	ctx context.Context, exampleID int, securityContext int,
) (*model.ExampleModel, error) {
	args := m.Called(ctx, exampleID, securityContext)
	return args.Get(0).(*model.ExampleModel), args.Error(1)
}

func (m *MockExampleRepository) CreateExample(
	ctx context.Context, example *model.ExampleModel, securityContext int,
) (*model.ExampleModel, error) {
	args := m.Called(ctx, example, securityContext)
	return args.Get(0).(*model.ExampleModel), args.Error(1)
}

func (m *MockExampleRepository) UpdateExample(
	ctx context.Context, example *model.ExampleModel, originalUpdateTS float64,
) error {
	return m.Called(ctx, example, originalUpdateTS).Error(0)
}

func (m *MockExampleRepository) DeleteExample(
	ctx context.Context, exampleID int, securityContext int,
	originalUpdateTS float64, deleteTS float64, deleteUserID int,
) (int, error) {
	args := m.Called(ctx, exampleID, securityContext, originalUpdateTS, deleteTS, deleteUserID)
	return args.Int(0), args.Error(1)
}

func (m *MockExampleRepository) RetrieveExampleList(
	ctx context.Context, idList []int, securityContext int, afterUpdateTs float64,
) ([]*model.ExampleModel, error) {
	args := m.Called(ctx, idList, securityContext, afterUpdateTs)
	return args.Get(0).([]*model.ExampleModel), args.Error(1)
}

func (m *MockExampleRepository) CreateExampleList(
	ctx context.Context, exampleList []*model.ExampleModel, securityContext int,
) ([]*model.ExampleModel, error) {
	args := m.Called(ctx, exampleList, securityContext)
	return args.Get(0).([]*model.ExampleModel), args.Error(1)
}

func (m *MockExampleRepository) UpdateExampleList(
	ctx context.Context, exampleList []*model.ExampleModel,
	originalUpdateTSList []float64, securityContext int,
) error {
	return m.Called(ctx, exampleList, originalUpdateTSList, securityContext).Error(0)
}

func (m *MockExampleRepository) DeleteExampleList(
	ctx context.Context, idList []int, updateTimestampList []float64,
	securityContext int, deleteTS float64, deleteUserID int,
) ([]int, error) {
	args := m.Called(ctx, idList, updateTimestampList, securityContext, deleteTS, deleteUserID)
	return args.Get(0).([]int), args.Error(1)
}

// testServices is a real app.Services with no database behind it: the
// in-process store, a logger writing to a buffer, and just enough config
// for the check sequence to run.
func testServices(t *testing.T) *app.Services {
	t.Helper()
	kv := cache.NewKV()
	t.Cleanup(kv.Close)
	return &app.Services{
		KV: kv,
		Config: &config.Config{
			JWTSecret:             "test-secret",
			JWTSessionExpireHours: 1,
			MaxUserReqPerMin:      1000,
			MaxUnauthReqPerMin:    1000,
		},
		Logger: logging.NewLogger("test", &bytes.Buffer{}, slog.LevelDebug),
	}
}

// signedRequest builds a <method> /example?exampleId=7 carrying a valid
// auth cookie and the matching CSRF header for a user with the given grants.
func signedRequest(t *testing.T, s *app.Services, method string, grants []int) *http.Request {
	t.Helper()
	const sessionID = "session-abc"
	token, err := auth.GenerateJWT(
		"Test", "alice", 1, sessionID, 1, grants,
		s.Config.JWTSecret, s.Config.JWTSessionExpireHours,
	)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(method, "/example?exampleId=7", nil)
	r.AddCookie(&http.Cookie{Name: "auth", Value: token})
	r.Header.Set("X-CSRF-Token", sessionID[len(sessionID)-3:])
	return r
}

// TestExampleHandler_CheckSequence proves the explicit per-handler
// sequence — CheckJWT → CheckRateLimitAndUserBlock → CheckCSRF →
// CheckAuthorization → CheckAllChecksComplete — in the order the handler
// writes it, by showing where each kind of bad request is stopped and
// that the repository is never reached until every check has passed.
func TestExampleHandler_CheckSequence(t *testing.T) {
	tests := []struct {
		name       string
		request    func(t *testing.T, s *app.Services) *http.Request
		wantStatus int
		reachesDB  bool
	}{
		{
			name: "no auth cookie → 401 from CheckJWT",
			request: func(t *testing.T, s *app.Services) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/example?exampleId=7", nil)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "rate limit spent → 429 from CheckRateLimitAndUserBlock, " +
				"before CSRF and authorization",
			request: func(t *testing.T, s *app.Services) *http.Request {
				s.Config.MaxUserReqPerMin = 0
				// no CSRF header, no grant: the limit must be what answers
				r := signedRequest(t, s, http.MethodDelete, nil)
				r.Header.Del("X-CSRF-Token")
				return r
			},
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name: "DELETE with valid JWT but no CSRF header → 403 from CheckCSRF",
			request: func(t *testing.T, s *app.Services) *http.Request {
				r := signedRequest(t, s, http.MethodDelete, []int{model.GrantExampleDelete})
				r.Header.Del("X-CSRF-Token")
				return r
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "GET without CSRF header is allowed — safe methods skip CheckCSRF by design → 200",
			request: func(t *testing.T, s *app.Services) *http.Request {
				r := signedRequest(t, s, http.MethodGet, []int{model.GrantExampleRead})
				r.Header.Del("X-CSRF-Token")
				return r
			},
			wantStatus: http.StatusOK,
			reachesDB:  true,
		},
		{
			name: "valid JWT and CSRF, no READ_EXAMPLE grant → 403 from CheckAuthorization",
			request: func(t *testing.T, s *app.Services) *http.Request {
				return signedRequest(t, s, http.MethodGet, []int{model.GrantExampleCreate})
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "blocked user → 403 from CheckRateLimitAndUserBlock, before authorization",
			request: func(t *testing.T, s *app.Services) *http.Request {
				if err := s.KV.Set("BLOCKED*alice", "1", ""); err != nil {
					t.Fatal(err)
				}
				// no grant either: the block must be what answers
				return signedRequest(t, s, http.MethodGet, nil)
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "every check passes → repository called with the claims' context → 200",
			request: func(t *testing.T, s *app.Services) *http.Request {
				return signedRequest(t, s, http.MethodGet, []int{model.GrantExampleRead})
			},
			wantStatus: http.StatusOK,
			reachesDB:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServices(t)
			repo := new(MockExampleRepository)
			if tt.reachesDB {
				repo.On("RetrieveExample", mock.Anything, 7, 1).
					Return(&model.ExampleModel{ExampleID: 7, SecurityContext: 1}, nil)
			}
			h := NewExampleHandler(s, repo)

			w := httptest.NewRecorder()
			r := tt.request(t, s)
			if r.Method == http.MethodDelete {
				h.DeleteExample(w, r)
			} else {
				h.RetrieveExample(w, r)
			}

			assert.Equal(t, tt.wantStatus, w.Code, w.Body.String())
			repo.AssertExpectations(t) // and, implicitly, no unexpected call
		})
	}
}
