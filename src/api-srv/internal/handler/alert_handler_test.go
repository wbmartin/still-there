package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
)

// MockAlertRepository satisfies AlertRepositoryInterface. A call with no
// expectation panics: a refused request never reaches it.
type MockAlertRepository struct {
	mock.Mock
}

func (m *MockAlertRepository) RetrieveAlertList(
	ctx context.Context, deviceID int, securityContext int,
	resolvedSince float64,
) ([]*model.AlertModel, error) {
	args := m.Called(ctx, deviceID, securityContext, resolvedSince)
	return args.Get(0).([]*model.AlertModel), args.Error(1)
}

func (m *MockAlertRepository) AcknowledgeAlert(
	ctx context.Context, alertID int, securityContext int,
	originalUpdateTS float64, actorName string, now float64, actorUserID int,
) (*model.AlertModel, error) {
	args := m.Called(ctx, alertID, securityContext, originalUpdateTS,
		actorName, now, actorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertModel), args.Error(1)
}

func (m *MockAlertRepository) TurnOffAlert(
	ctx context.Context, alertID int, securityContext int,
	originalUpdateTS float64, actorName string, now float64, actorUserID int,
) (*model.AlertModel, error) {
	args := m.Called(ctx, alertID, securityContext, originalUpdateTS,
		actorName, now, actorUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AlertModel), args.Error(1)
}

// alertAction posts body to the acknowledge or turn-off route as alice
// (login 7, tenant 2), whose login row is user.
func alertAction(
	t *testing.T, turnOff bool, alertRepository *MockAlertRepository,
	user *model.AppUserModel, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	appServices := testServices(t)
	actorRepository := new(MockProfileRepository)
	actorRepository.On("RetrieveUserByUserName", mock.Anything, "alice").
		Return(user, nil)
	handler := NewAlertHandler(appServices, alertRepository, actorRepository)
	target, action := "/alertAcknowledge", handler.AcknowledgeAlert
	if turnOff {
		target, action = "/alertTurnOff", handler.TurnOffAlert
	}
	w := httptest.NewRecorder()
	action(w, helpRequest(t, appServices, http.MethodPost, target, body,
		[]int{model.GrantAlertUpdate}))
	return w
}

// The contacts are told the person's display name, or the login's name
// when the person has none; the tenant and lock come from the session
// and the body.
func TestAlertHandler_ActorIsTheCallersName(t *testing.T) {
	for _, testCase := range []struct {
		user      *model.AppUserModel
		wantActor string
	}{
		{&model.AppUserModel{AppUserID: 7, UserName: "alice",
			FirstName: "Alice", LastName: "Smith"}, "Alice Smith"},
		{&model.AppUserModel{AppUserID: 7, UserName: "alice"}, "alice"},
	} {
		alertRepository := new(MockAlertRepository)
		alertRepository.On("AcknowledgeAlert", mock.Anything, 11, 2, 99.5,
			testCase.wantActor, mock.Anything, 7).
			Return(&model.AlertModel{AlertID: 11,
				AlertState: model.AlertStateAcknowledged}, nil)
		w := alertAction(t, false, alertRepository, testCase.user,
			`{"alertId":11,"updateTs":99.5}`)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		alertRepository.AssertExpectations(t)
	}
}

// A state the action cannot start from is the repository's 409 with its
// message; a token for another login is 401 before any write.
func TestAlertHandler_Refusals(t *testing.T) {
	alice := &model.AppUserModel{AppUserID: 7, UserName: "alice"}

	alertRepository := new(MockAlertRepository)
	alertRepository.On("TurnOffAlert", mock.Anything, 11, 2, 99.5, "alice",
		mock.Anything, 7).
		Return(nil, &repository.ClientMessageError{
			Sentinel: repository.ErrConflict,
			Message:  "This alert is resolved and cannot be turned off"})
	w := alertAction(t, true, alertRepository, alice,
		`{"alertId":11,"updateTs":99.5}`)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.JSONEq(t,
		`{"error":"This alert is resolved and cannot be turned off"}`,
		w.Body.String())

	w = alertAction(t, false, new(MockAlertRepository),
		&model.AppUserModel{AppUserID: 8, UserName: "alice"},
		`{"alertId":11,"updateTs":99.5}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = alertAction(t, false, new(MockAlertRepository), alice,
		`{"updateTs":99.5}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
