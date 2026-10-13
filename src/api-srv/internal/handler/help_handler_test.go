package handler

import (
	"context"
	"encoding/json"
	"errors"
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
)

// MockHelpEventRepository satisfies HelpEventRepositoryInterface. A call
// with no expectation panics: a refused request never reaches it.
type MockHelpEventRepository struct {
	mock.Mock
}

func (m *MockHelpEventRepository) CreateHelpEvent(
	ctx context.Context, event *model.HelpEventModel, securityContext int,
) (int, error) {
	args := m.Called(ctx, event, securityContext)
	return args.Int(0), args.Error(1)
}

func (m *MockHelpEventRepository) RetrieveHelpEventCountSince(
	ctx context.Context, appUserID int, since float64, securityContext int,
) (int, error) {
	args := m.Called(ctx, appUserID, since, securityContext)
	return args.Int(0), args.Error(1)
}

func (m *MockHelpEventRepository) RetrieveHelpInsightCountList(
	ctx context.Context, event string, since float64, limit int,
	securityContext int,
) ([]model.HelpInsightCount, error) {
	args := m.Called(ctx, event, since, limit, securityContext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.HelpInsightCount), args.Error(1)
}

const (
	helpTestTenant = 2
	helpTestUserID = 7
)

// helpServices is testServices with the module on or off.
func helpServices(t *testing.T, on bool) *app.Services {
	t.Helper()
	appServices := testServices(t)
	appServices.Config.HelpEnabled = on
	return appServices
}

// helpRequest is method target as alice (login 7) of tenant 2 holding
// grantList, with the auth cookie and the CSRF header.
func helpRequest(
	t *testing.T, appServices *app.Services, method string, target string,
	body string, grantList []int,
) *http.Request {
	t.Helper()
	const sessionID = "session-help"
	token, err := auth.GenerateJWT(
		"Test", "alice", helpTestUserID, sessionID, helpTestTenant,
		grantList, appServices.Config.JWTSecret,
		appServices.Config.JWTSessionExpireHours,
	)
	require.NoError(t, err)
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.AddCookie(&http.Cookie{Name: "auth", Value: token})
	r.Header.Set("X-CSRF-Token", sessionID[len(sessionID)-3:])
	return r
}

func postHelpEvent(
	t *testing.T, appServices *app.Services,
	repository *MockHelpEventRepository, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	NewHelpHandler(appServices, repository).CreateHelpEvent(w,
		helpRequest(t, appServices, http.MethodPost, "/helpEvent", body,
			[]int{model.GrantLogon}))
	return w
}

// Off, both routes answer 404 once the checks pass and touch nothing; the
// checks still answer first, so an off module tells an anonymous caller
// no more than an on one.
func TestHelpHandler_ModuleOffIs404AfterTheChecks(t *testing.T) {
	appServices := helpServices(t, false)
	repository := new(MockHelpEventRepository)
	handler := NewHelpHandler(appServices, repository)

	w := httptest.NewRecorder()
	handler.CreateHelpEvent(w, httptest.NewRequest(http.MethodPost,
		"/helpEvent", strings.NewReader(`{}`)))
	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"no session: the check answers, not the module")

	w = postHelpEvent(t, appServices, repository,
		`{"event":"entry_view","entryId":"glossary-tenant","route":"/"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Help is not enabled")

	w = httptest.NewRecorder()
	handler.RetrieveHelpInsights(w, helpRequest(t, appServices,
		http.MethodGet, "/helpInsights", "",
		[]int{model.GrantLogon, model.GrantUserRead}))
	assert.Equal(t, http.StatusNotFound, w.Code)
	repository.AssertExpectations(t)
}

// Each malformed event is 400 before the repository is reached.
func TestHelpHandler_EventValidation(t *testing.T) {
	testCaseList := []struct {
		name string
		body string
		want string
	}{
		{"unknown event",
			`{"event":"panel_open","entryId":"a","route":"/x"}`,
			"event is not a help event"},
		{"no route", `{"event":"entry_view","entryId":"a"}`,
			"route must be an app route"},
		{"route with a query string",
			`{"event":"entry_view","entryId":"a","route":"/x?help=a"}`,
			"route must be an app route"},
		{"route too long", `{"event":"entry_view","entryId":"a","route":"/` +
			strings.Repeat("a", 64) + `"}`, "route must be an app route"},
		{"search with no text",
			`{"event":"search_no_result","query":"  ","route":"/x"}`,
			"query is required"},
		{"view with no entry", `{"event":"entry_view","route":"/x"}`,
			"entryId must be a help id"},
		{"entry id with markup",
			`{"event":"tour_skip","entryId":"<b>","route":"/x"}`,
			"entryId must be a help id"},
		{"entry id upper case",
			`{"event":"tour_complete","entryId":"Tour","route":"/x"}`,
			"entryId must be a help id"},
		{"not JSON", `event=entry_view`, ""},
	}
	for _, testCase := range testCaseList {
		t.Run(testCase.name, func(t *testing.T) {
			repository := new(MockHelpEventRepository)
			w := postHelpEvent(t, helpServices(t, true), repository,
				testCase.body)
			assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), testCase.want)
			repository.AssertExpectations(t)
		})
	}
}

// An entry view stores the entry id and no text, in the session's tenant,
// as the caller.
func TestHelpHandler_EntryViewIsStoredInTheSessionTenant(t *testing.T) {
	repository := new(MockHelpEventRepository)
	repository.On("RetrieveHelpEventCountSince", mock.Anything,
		helpTestUserID, mock.Anything, helpTestTenant).Return(3, nil)
	var stored *model.HelpEventModel
	repository.On("CreateHelpEvent", mock.Anything, mock.Anything,
		helpTestTenant).Run(func(args mock.Arguments) {
		stored = args.Get(1).(*model.HelpEventModel)
	}).Return(41, nil)

	w := postHelpEvent(t, helpServices(t, true), repository,
		`{"event":"entry_view","entryId":"glossary-tenant",`+
			`"route":"/exampleList","query":"kept nowhere"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.JSONEq(t, `{"data":{"recorded":true}}`, w.Body.String())
	require.NotNil(t, stored)
	assert.Equal(t, helpTestUserID, stored.AppUserID)
	assert.Equal(t, helpTestUserID, stored.CreateUserID)
	assert.Equal(t, model.HelpEventEntryView, stored.Event)
	assert.Equal(t, "/exampleList", stored.Route)
	require.NotNil(t, stored.EntryID)
	assert.Equal(t, "glossary-tenant", *stored.EntryID)
	assert.Nil(t, stored.Query, "only a failed search keeps text")
	repository.AssertExpectations(t)
}

// A failed search keeps its text, trimmed and cut to 200 characters, and
// no entry id.
func TestHelpHandler_NoResultSearchKeepsItsTextCut(t *testing.T) {
	repository := new(MockHelpEventRepository)
	repository.On("RetrieveHelpEventCountSince", mock.Anything,
		helpTestUserID, mock.Anything, helpTestTenant).Return(0, nil)
	var stored *model.HelpEventModel
	repository.On("CreateHelpEvent", mock.Anything, mock.Anything,
		helpTestTenant).Run(func(args mock.Arguments) {
		stored = args.Get(1).(*model.HelpEventModel)
	}).Return(42, nil)

	longQuery := strings.Repeat("é", model.HelpMaxQueryRunes+20)
	w := postHelpEvent(t, helpServices(t, true), repository,
		`{"event":"search_no_result","entryId":"ignored",`+
			`"route":"/dashboard","query":"  `+longQuery+`  "}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stored)
	require.NotNil(t, stored.Query)
	assert.Equal(t, strings.Repeat("é", model.HelpMaxQueryRunes),
		*stored.Query)
	assert.Nil(t, stored.EntryID)
	repository.AssertExpectations(t)
}

// At the daily cap the event is dropped, and the answer says so.
func TestHelpHandler_DailyCapDropsTheEvent(t *testing.T) {
	repository := new(MockHelpEventRepository)
	repository.On("RetrieveHelpEventCountSince", mock.Anything,
		helpTestUserID, mock.Anything, helpTestTenant).Return(
		model.HelpMaxEventsPerUserPerDay, nil)

	w := postHelpEvent(t, helpServices(t, true), repository,
		`{"event":"tour_complete","entryId":"example-list","route":"/x"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.JSONEq(t, `{"data":{"recorded":false}}`, w.Body.String())
	repository.AssertExpectations(t)
	repository.AssertNotCalled(t, "CreateHelpEvent",
		mock.Anything, mock.Anything, mock.Anything)
}

// A store failure is 500 with a plain message; the error stays in the log.
func TestHelpHandler_EventStoreFailureIs500(t *testing.T) {
	repository := new(MockHelpEventRepository)
	repository.On("RetrieveHelpEventCountSince", mock.Anything,
		mock.Anything, mock.Anything, mock.Anything).Return(0, nil)
	repository.On("CreateHelpEvent", mock.Anything, mock.Anything,
		mock.Anything).Return(0, errors.New("pg: relation is gone"))

	w := postHelpEvent(t, helpServices(t, true), repository,
		`{"event":"tour_skip","entryId":"example-list","route":"/x"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "relation")
}

// Insights reads the session tenant's four lists over the asked window.
func TestHelpHandler_InsightsReadsTheSessionTenant(t *testing.T) {
	repository := new(MockHelpEventRepository)
	for event, countList := range map[string][]model.HelpInsightCount{
		model.HelpEventSearchNoResult: {{Key: "invoice", Count: 4}},
		model.HelpEventEntryView:      {{Key: "glossary-tenant", Count: 9}},
		model.HelpEventTourComplete:   {},
		model.HelpEventTourSkip:       {{Key: "example-list", Count: 1}},
	} {
		repository.On("RetrieveHelpInsightCountList", mock.Anything,
			event, mock.Anything, model.HelpInsightsListLimit,
			helpTestTenant).Return(countList, nil).Once()
	}
	appServices := helpServices(t, true)
	before := model.Now()
	w := httptest.NewRecorder()
	NewHelpHandler(appServices, repository).RetrieveHelpInsights(w,
		helpRequest(t, appServices, http.MethodGet, "/helpInsights?days=7",
			"", []int{model.GrantLogon, model.GrantUserRead}))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Data model.HelpInsightsView `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 7, body.Data.Days)
	assert.Equal(t, "invoice", body.Data.NoResultSearchList[0].Key)
	assert.Equal(t, 9, body.Data.EntryViewList[0].Count)
	assert.Empty(t, body.Data.TourCompleteList)
	assert.Equal(t, "example-list", body.Data.TourSkipList[0].Key)
	repository.AssertExpectations(t)

	since := repository.Calls[0].Arguments.Get(2).(float64)
	assert.InDelta(t, before-7*86400, since, 5,
		"the window starts days before now")
}

// days outside 1..365 or not a number is 400; absent is 30.
func TestHelpHandler_InsightsDays(t *testing.T) {
	for _, raw := range []string{"0", "366", "-1", "seven", "1.5"} {
		t.Run(raw, func(t *testing.T) {
			appServices := helpServices(t, true)
			repository := new(MockHelpEventRepository)
			w := httptest.NewRecorder()
			NewHelpHandler(appServices, repository).RetrieveHelpInsights(w,
				helpRequest(t, appServices, http.MethodGet,
					"/helpInsights?days="+raw, "",
					[]int{model.GrantLogon, model.GrantUserRead}))
			assert.Equal(t, http.StatusBadRequest, w.Code)
			repository.AssertExpectations(t)
		})
	}

	appServices := helpServices(t, true)
	repository := new(MockHelpEventRepository)
	repository.On("RetrieveHelpInsightCountList", mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		[]model.HelpInsightCount{}, nil)
	w := httptest.NewRecorder()
	NewHelpHandler(appServices, repository).RetrieveHelpInsights(w,
		helpRequest(t, appServices, http.MethodGet, "/helpInsights", "",
			[]int{model.GrantLogon, model.GrantUserRead}))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"days":30`)
}

// Without READ_USER a signed-in user cannot read insights, whatever the
// module's state.
func TestHelpHandler_InsightsNeedsReadUser(t *testing.T) {
	appServices := helpServices(t, true)
	repository := new(MockHelpEventRepository)
	w := httptest.NewRecorder()
	NewHelpHandler(appServices, repository).RetrieveHelpInsights(w,
		helpRequest(t, appServices, http.MethodGet, "/helpInsights", "",
			[]int{model.GrantLogon}))
	assert.Equal(t, http.StatusForbidden, w.Code)
	repository.AssertExpectations(t)
}
