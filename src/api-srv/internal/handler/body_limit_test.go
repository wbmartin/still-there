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
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/security"
)

// bodyOfSize is a JSON object of exactly size bytes: one long string field.
func bodyOfSize(size int) string {
	const head, tail = `{"pad":"`, `"}`
	return head + strings.Repeat("a", size-len(head)-len(tail)) + tail
}

func decodeOnce(
	t *testing.T, body string, maxBytes int64,
) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	s := testServices(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	app := security.NewAppHelper(w, r, s, "Test")
	var dst map[string]any
	return w, decodeJSONBody(app, w, r, maxBytes, &dst)
}

func TestDecodeJSONBody(t *testing.T) {
	t.Run("at the cap → decoded, nothing written", func(t *testing.T) {
		w, ok := decodeOnce(t, bodyOfSize(1024), 1024)
		assert.True(t, ok)
		assert.Zero(t, w.Body.Len())
	})

	t.Run("one byte over the cap → 413", func(t *testing.T) {
		w, ok := decodeOnce(t, bodyOfSize(1025), 1024)
		assert.False(t, ok)
		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
		assert.JSONEq(t, `{"error":"Request body too large"}`, w.Body.String())
	})

	for _, body := range []string{`{`, `{"pad":1,}`, `[1,2]`, ``} {
		t.Run("malformed "+body+" → 400 without decoder text",
			func(t *testing.T) {
				w, ok := decodeOnce(t, body, 1024)
				assert.False(t, ok)
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.JSONEq(t, `{"error":"Invalid request body"}`,
					w.Body.String())
			})
	}
}

func TestReadBody_OverTheCapIs413(t *testing.T) {
	s := testServices(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/",
		strings.NewReader(strings.Repeat("a", 4097)))
	_, ok := readBody(security.NewAppHelper(w, r, s, "Test"), w, r, 4096)
	assert.False(t, ok)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

// The cap is applied after the security checks: an unsigned oversized body
// is still a 401, a signed one a 413, and the repository is never reached.
func TestCreateUser_BodyOverTheCapIs413(t *testing.T) {
	s := testServices(t)
	repo := new(MockUserRepository)
	handler := NewUserHandler(s, repo, new(MockAppUserGroupRepository))
	body := bodyOfSize(int(maxLoginLayerBodyBytes) + 1024)

	w := httptest.NewRecorder()
	handler.CreateUser(w,
		httptest.NewRequest(http.MethodPost, "/user", strings.NewReader(body)))
	assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())

	token := sessionToken(t, s, "alice", 1, "session-abc",
		[]int{model.GrantUserCreate})
	w = httptest.NewRecorder()
	handler.CreateUser(w,
		sessionRequest(http.MethodPost, "/user", body, token, "session-abc"))
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, w.Body.String())
	repo.AssertExpectations(t)
}

func TestUpdateOwnProfile_BodyOverTheCapIs413(t *testing.T) {
	s := testServices(t)
	repo := new(MockProfileRepository)
	w := httptest.NewRecorder()
	NewProfileHandler(s, repo).UpdateOwnProfile(w,
		meRequest(t, s, http.MethodPut, []int{model.GrantLogon},
			bodyOfSize(13<<10)))
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, w.Body.String())
	repo.AssertExpectations(t)
}

func TestAuthHandler_BodyOverTheCapIs413(t *testing.T) {
	s := testServices(t)
	w := httptest.NewRecorder()
	AuthHandler(s, new(MockUserRepository))(w, httptest.NewRequest(
		http.MethodPost, "/auth", strings.NewReader(bodyOfSize(5<<10))))
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, w.Body.String())
}

// The example cap is the separate 1 MiB entity cap, not the login layer's.
func TestCreateExampleList_EntityBodyCap(t *testing.T) {
	exampleListBody := func(size int) string {
		item, err := json.Marshal(model.ExampleModel{ExampleText: "x"})
		require.NoError(t, err)
		count := size / (len(item) + 1)
		return "[" + strings.TrimSuffix(
			strings.Repeat(string(item)+",", count), ",") + "]"
	}

	t.Run("over the login-layer cap, under 1 MiB → reaches the repository",
		func(t *testing.T) {
			s := testServices(t)
			repo := new(MockExampleRepository)
			repo.On("CreateExampleList", mock.Anything, mock.Anything, 1).
				Return([]*model.ExampleModel{}, nil)
			body := exampleListBody(int(maxLoginLayerBodyBytes) * 2)
			w := httptest.NewRecorder()
			NewExampleHandler(s, repo).CreateExampleList(w,
				sessionRequest(http.MethodPost, "/exampleList", body,
					sessionToken(t, s, "alice", 1, "session-abc",
						[]int{model.GrantExampleCreate}),
					"session-abc"))
			assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
			repo.AssertExpectations(t)
		})

	t.Run("over 1 MiB → 413, repository untouched", func(t *testing.T) {
		s := testServices(t)
		repo := new(MockExampleRepository)
		body := exampleListBody(int(maxEntityBodyBytes) + 1024)
		w := httptest.NewRecorder()
		NewExampleHandler(s, repo).CreateExampleList(w,
			sessionRequest(http.MethodPost, "/exampleList", body,
				sessionToken(t, s, "alice", 1, "session-abc",
					[]int{model.GrantExampleCreate}),
				"session-abc"))
		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code,
			w.Body.String())
		repo.AssertExpectations(t)
	})
}
