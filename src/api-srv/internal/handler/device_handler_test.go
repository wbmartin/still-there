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
	"github.com/stretchr/testify/require"
	"stillthere-api/internal/app"
	"stillthere-api/internal/auth"
	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
)

// MockDeviceRepository satisfies DeviceRepositoryInterface. A call with no
// expectation panics: a refused request never reaches it.
type MockDeviceRepository struct {
	mock.Mock
}

func (m *MockDeviceRepository) RetrieveDeviceList(
	ctx context.Context, idList []int, securityContext int,
	afterUpdateTs float64,
) ([]*model.DeviceModel, error) {
	args := m.Called(ctx, idList, securityContext, afterUpdateTs)
	return args.Get(0).([]*model.DeviceModel), args.Error(1)
}

func (m *MockDeviceRepository) CreateDevice(
	ctx context.Context, device *model.DeviceModel, securityContext int,
) (*model.DeviceModel, error) {
	args := m.Called(ctx, device, securityContext)
	return args.Get(0).(*model.DeviceModel), args.Error(1)
}

func (m *MockDeviceRepository) UpdateDevice(
	ctx context.Context, device *model.DeviceModel, originalUpdateTS float64,
) (*model.DeviceModel, error) {
	args := m.Called(ctx, device, originalUpdateTS)
	return args.Get(0).(*model.DeviceModel), args.Error(1)
}

func (m *MockDeviceRepository) DeleteDevice(
	ctx context.Context, deviceID int, securityContext int,
	originalUpdateTS float64, deleteTS float64, deleteUserID int,
) error {
	args := m.Called(ctx, deviceID, securityContext, originalUpdateTS,
		deleteTS, deleteUserID)
	return args.Error(0)
}

func (m *MockDeviceRepository) IssueDeviceToken(
	ctx context.Context, deviceID int, securityContext int,
	originalUpdateTS float64, tokenHash string, issuedTS float64,
	actorUserID int,
) error {
	args := m.Called(ctx, deviceID, securityContext, originalUpdateTS,
		tokenHash, issuedTS, actorUserID)
	return args.Error(0)
}

func (m *MockDeviceRepository) RetrieveDeviceByTokenHash(
	ctx context.Context, tokenHash string,
) (*model.DeviceModel, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DeviceModel), args.Error(1)
}

func (m *MockDeviceRepository) RecordDeviceCheckin(
	ctx context.Context, deviceID int, securityContext int,
	record *model.DeviceCheckinRecord, beaconList []model.DeviceCheckinBeacon,
) (int, []string, error) {
	args := m.Called(ctx, deviceID, securityContext, record, beaconList)
	return args.Int(0), args.Get(1).([]string), args.Error(2)
}

// deviceTestToken is a token of the shape auth.NewDeviceToken makes.
var deviceTestToken = strings.Repeat("ab", 32)

// checkinBody is a check-in as src/pi-agent/checkin.go sends it.
const checkinBody = `{"agentVersion":"0.1.0",` +
	`"sentAt":"2026-10-24T09:17:44Z","uptimeSec":120,"configVersion":1,` +
	`"reason":"interval","beaconList":[{"address":"c3:00:00:98:f3:19",` +
	`"label":"Band","present":true,"lastSeen":"2026-10-24T09:17:40Z",` +
	`"rssi":-61}]}`

func checkinRequest(authorization string, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/deviceCheckin",
		strings.NewReader(body))
	r.RemoteAddr = "192.0.2.10:5555"
	if authorization != "" {
		r.Header.Set("Authorization", authorization)
	}
	return r
}

func postCheckin(
	appServices *app.Services, deviceRepository *MockDeviceRepository,
	r *http.Request,
) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	NewDeviceHandler(appServices, deviceRepository).DeviceCheckin(w, r)
	return w
}

// Every check-in without a live device's token is the same 401: no
// header, another scheme, a malformed token (no database read), an
// unknown one, a device deleted between the lookup and the write.
func TestDeviceCheckin_RefusesWithoutALiveToken(t *testing.T) {
	malformedList := []string{
		"", "Basic " + deviceTestToken, "Bearer", "Bearer short",
		"Bearer " + strings.ToUpper(deviceTestToken),
		"bearer " + deviceTestToken, "Bearer " + deviceTestToken + "0",
	}
	for _, authorization := range malformedList {
		t.Run("malformed "+authorization, func(t *testing.T) {
			deviceRepository := new(MockDeviceRepository)
			w := postCheckin(testServices(t), deviceRepository,
				checkinRequest(authorization, checkinBody))
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.JSONEq(t, `{"error":"Invalid device token"}`,
				w.Body.String())
			deviceRepository.AssertExpectations(t)
		})
	}

	t.Run("unknown token", func(t *testing.T) {
		deviceRepository := new(MockDeviceRepository)
		deviceRepository.On("RetrieveDeviceByTokenHash", mock.Anything,
			auth.HashDeviceToken(deviceTestToken)).
			Return(nil, repository.ErrNotFound)
		w := postCheckin(testServices(t), deviceRepository,
			checkinRequest("Bearer "+deviceTestToken, checkinBody))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.JSONEq(t, `{"error":"Invalid device token"}`, w.Body.String())
		deviceRepository.AssertExpectations(t)
	})

	t.Run("device deleted after the lookup", func(t *testing.T) {
		deviceRepository := new(MockDeviceRepository)
		deviceRepository.On("RetrieveDeviceByTokenHash", mock.Anything,
			mock.Anything).Return(&model.DeviceModel{
			DeviceID: 5, SecurityContext: 3}, nil)
		deviceRepository.On("RecordDeviceCheckin", mock.Anything, 5, 3,
			mock.Anything, mock.Anything).
			Return(0, []string(nil), repository.ErrNotFound)
		w := postCheckin(testServices(t), deviceRepository,
			checkinRequest("Bearer "+deviceTestToken, checkinBody))
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.JSONEq(t, `{"error":"Invalid device token"}`, w.Body.String())
		deviceRepository.AssertExpectations(t)
	})
}

// A good check-in is written in the device's own tenant, carries the body
// as sent, and answers the config version in the house envelope.
func TestDeviceCheckin_RecordsInTheDevicesTenant(t *testing.T) {
	deviceRepository := new(MockDeviceRepository)
	deviceRepository.On("RetrieveDeviceByTokenHash", mock.Anything,
		auth.HashDeviceToken(deviceTestToken)).
		Return(&model.DeviceModel{DeviceID: 5, SecurityContext: 3}, nil)
	var record *model.DeviceCheckinRecord
	var beaconList []model.DeviceCheckinBeacon
	deviceRepository.On("RecordDeviceCheckin", mock.Anything, 5, 3,
		mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			record = args.Get(3).(*model.DeviceCheckinRecord)
			beaconList = args.Get(4).([]model.DeviceCheckinBeacon)
		}).
		Return(7, []string{"AA:BB:CC:DD:EE:FF"}, nil)

	w := postCheckin(testServices(t), deviceRepository,
		checkinRequest("Bearer "+deviceTestToken, checkinBody))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.JSONEq(t, `{"data":{"configVersion":7}}`, w.Body.String())
	deviceRepository.AssertExpectations(t)

	require.NotNil(t, record)
	assert.Equal(t, "interval", record.Reason)
	assert.Equal(t, "0.1.0", record.AgentVersion)
	assert.JSONEq(t, checkinBody, string(record.BodyJSON))
	assert.Equal(t, "192.0.2.10", record.SourceIP)
	require.NotNil(t, record.SentTS)
	assert.Equal(t, float64(1792833464), *record.SentTS)
	assert.Greater(t, record.ReceivedTS, *record.SentTS)
	require.Len(t, beaconList, 1)
	assert.True(t, beaconList[0].Present)
	assert.Equal(t, -61, *beaconList[0].RSSI)
}

func TestDeviceCheckin_BodyRules(t *testing.T) {
	for name, testCase := range map[string]struct {
		body   string
		status int
	}{
		"over 16 KiB":    {bodyOfSize(16<<10 + 1), http.StatusRequestEntityTooLarge},
		"not JSON":       {`{`, http.StatusBadRequest},
		"no reason":      {`{"agentVersion":"0.1.0"}`, http.StatusBadRequest},
		"a bare null":    {`null`, http.StatusBadRequest},
		"sentAt garbage": {`{"reason":"x","sentAt":"yesterday"}`, http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			deviceRepository := new(MockDeviceRepository)
			deviceRepository.On("RetrieveDeviceByTokenHash", mock.Anything,
				mock.Anything).Return(&model.DeviceModel{
				DeviceID: 5, SecurityContext: 3}, nil)
			w := postCheckin(testServices(t), deviceRepository,
				checkinRequest("Bearer "+deviceTestToken, testCase.body))
			assert.Equal(t, testCase.status, w.Code, w.Body.String())
			deviceRepository.AssertNotCalled(t, "RecordDeviceCheckin")
		})
	}
}

// deviceRequest is method target as alice (login 7) of tenant 2 holding
// grantList, with the auth cookie and the CSRF header.
func deviceRequest(
	t *testing.T, appServices *app.Services, method string, target string,
	body string, grantList []int,
) *http.Request {
	t.Helper()
	return helpRequest(t, appServices, method, target, body, grantList)
}

// The token is answered once, 64 hex characters, and only its hash
// reaches the repository, with the caller's tenant and lock.
func TestIssueDeviceToken_AnswersTheTokenAndStoresItsHash(t *testing.T) {
	appServices := testServices(t)
	deviceRepository := new(MockDeviceRepository)
	var storedHash string
	deviceRepository.On("IssueDeviceToken", mock.Anything, 5, 2, 123.5,
		mock.Anything, mock.Anything, 7).
		Run(func(args mock.Arguments) {
			storedHash = args.String(4)
		}).Return(nil)
	w := httptest.NewRecorder()
	NewDeviceHandler(appServices, deviceRepository).IssueDeviceToken(w,
		deviceRequest(t, appServices, http.MethodPost, "/deviceToken",
			`{"deviceId":5,"updateTs":123.5}`,
			[]int{model.GrantDeviceIssueToken}))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	deviceRepository.AssertExpectations(t)

	var reply struct {
		Data model.DeviceTokenIssuedModel `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &reply))
	assert.Regexp(t, `^[0-9a-f]{64}$`, reply.Data.DeviceToken)
	assert.Equal(t, auth.HashDeviceToken(reply.Data.DeviceToken), storedHash)
	assert.NotEqual(t, reply.Data.DeviceToken, storedHash)
	assert.Equal(t, 5, reply.Data.DeviceID)
	assert.Equal(t, reply.Data.UpdateTS, reply.Data.TokenIssuedTS)
}

// A device the settings rules refuse never reaches the repository.
func TestCreateDevice_RefusesBadSettings(t *testing.T) {
	for _, body := range []string{
		`{"deviceName":"","checkinIntervalSec":60,` +
			`"missedCheckinBeforeAlarm":3,"absentAfterSec":600}`,
		`{"deviceName":"Pi","checkinIntervalSec":9,` +
			`"missedCheckinBeforeAlarm":3,"absentAfterSec":600}`,
		`{"deviceName":"Pi","checkinIntervalSec":60,` +
			`"missedCheckinBeforeAlarm":0,"absentAfterSec":600}`,
		`{"deviceName":"Pi","checkinIntervalSec":60,` +
			`"missedCheckinBeforeAlarm":3,"absentAfterSec":9}`,
	} {
		appServices := testServices(t)
		deviceRepository := new(MockDeviceRepository)
		w := httptest.NewRecorder()
		NewDeviceHandler(appServices, deviceRepository).CreateDevice(w,
			deviceRequest(t, appServices, http.MethodPost, "/device", body,
				[]int{model.GrantDeviceCreate}))
		assert.Equal(t, http.StatusBadRequest, w.Code, body)
		deviceRepository.AssertExpectations(t)
	}
}

// The tenant and author come from the session, never the body.
func TestCreateDevice_TakesTheTenantFromTheSession(t *testing.T) {
	appServices := testServices(t)
	deviceRepository := new(MockDeviceRepository)
	deviceRepository.On("CreateDevice", mock.Anything,
		mock.MatchedBy(func(device *model.DeviceModel) bool {
			return device.CreateUserID == 7 && device.UpdateUserID == 7
		}), 2).
		Return(&model.DeviceModel{DeviceID: 9, SecurityContext: 2}, nil)
	w := httptest.NewRecorder()
	NewDeviceHandler(appServices, deviceRepository).CreateDevice(w,
		deviceRequest(t, appServices, http.MethodPost, "/device",
			`{"deviceName":"Pi","checkinIntervalSec":60,`+
				`"missedCheckinBeforeAlarm":3,"absentAfterSec":600,`+
				`"securityContext":3,"createUserId":1}`,
			[]int{model.GrantDeviceCreate}))
	assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	deviceRepository.AssertExpectations(t)
}
