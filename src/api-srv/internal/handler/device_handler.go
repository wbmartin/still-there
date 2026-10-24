package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"stillthere-api/internal/app"
	"stillthere-api/internal/auth"
	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
	"stillthere-api/internal/security"
)

// Still There monitoring (tasks/todo.md, Server): the Pi base station in a
// home is a device. An admin manages devices (GET /deviceList, POST / PUT /
// DELETE /device) and issues a device's token (POST /deviceToken), shown
// once and stored as its SHA-256. The Pi sends that token on every POST
// /deviceCheckin, the one route here with no session: the token is the
// credential, and it names the device and so the tenant.

// maxDeviceBodyBytes caps every signed-in device-monitoring write: device,
// beacon, contact, alert and tenant-setting bodies are one short row.
const maxDeviceBodyBytes int64 = 16 << 10

// deviceTokenPattern is the only shape auth.NewDeviceToken produces.
var deviceTokenPattern = inviteTokenPattern

// invalidDeviceTokenMessage is the one answer to every check-in that does
// not carry a live device's token, so the answer says nothing about which.
const invalidDeviceTokenMessage = "Invalid device token"

// DeviceRepositoryInterface is what DeviceHandler needs from
// repository.DeviceRepository; an interface so the check order can be
// proved with a repository that fails on contact.
type DeviceRepositoryInterface interface {
	RetrieveDeviceList(
		ctx context.Context, idList []int, securityContext int,
		afterUpdateTs float64,
	) ([]*model.DeviceModel, error)
	CreateDevice(
		ctx context.Context, device *model.DeviceModel, securityContext int,
	) (*model.DeviceModel, error)
	UpdateDevice(
		ctx context.Context, device *model.DeviceModel,
		originalUpdateTS float64,
	) (*model.DeviceModel, error)
	DeleteDevice(
		ctx context.Context, deviceID int, securityContext int,
		originalUpdateTS float64, deleteTS float64, deleteUserID int,
	) error
	IssueDeviceToken(
		ctx context.Context, deviceID int, securityContext int,
		originalUpdateTS float64, tokenHash string, issuedTS float64,
		actorUserID int,
	) error
	RetrieveDeviceByTokenHash(
		ctx context.Context, tokenHash string,
	) (*model.DeviceModel, error)
	RecordDeviceCheckin(
		ctx context.Context, deviceID int, securityContext int,
		record *model.DeviceCheckinRecord,
		beaconList []model.DeviceCheckinBeacon,
	) (int, []string, error)
}

// DeviceHandler serves the device routes.
type DeviceHandler struct {
	appsrv           *app.Services
	deviceRepository DeviceRepositoryInterface
}

// NewDeviceHandler creates the device handler.
func NewDeviceHandler(
	appServices *app.Services, deviceRepository DeviceRepositoryInterface,
) *DeviceHandler {
	return &DeviceHandler{
		appsrv: appServices, deviceRepository: deviceRepository,
	}
}

// RetrieveDeviceList handles GET
// /deviceList?deviceIdList=1,2&afterUpdateTs=<ts>.
func (h *DeviceHandler) RetrieveDeviceList(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "RetrieveDeviceList")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantDeviceRead)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	idList, afterUpdateTs, ok := listQueryOf(appHelper, w, r, "deviceIdList")
	if !ok {
		return
	}
	deviceList, err := h.deviceRepository.RetrieveDeviceList(r.Context(),
		idList, appHelper.GetClaims().SecurityContext, afterUpdateTs)
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Failed to retrieve devices", "DEV_LIST_FAIL", err)
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: deviceList})
}

// CreateDevice handles POST /device: a device with no token yet, at config
// version 1.
func (h *DeviceHandler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "CreateDevice")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantDeviceCreate)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	jwtClaims := appHelper.GetClaims()
	device, ok := decodeDevice(appHelper, w, r)
	if !ok {
		return
	}
	now := model.Now()
	device.CreateTS, device.CreateUserID = now, jwtClaims.UserID
	device.UpdateTS, device.UpdateUserID = now, jwtClaims.UserID
	created, err := h.deviceRepository.CreateDevice(
		r.Context(), device, jwtClaims.SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to create device",
			"DEV_CREATE_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusCreated,
		model.DataWrapper{Data: created})
	appHelper.Info("DEV_CREATED",
		fmt.Sprintf("Device created: deviceId=%d", created.DeviceID))
}

// UpdateDevice handles PUT /device: the name and alarm settings, under the
// optimistic lock. A settings change bumps configVersion.
func (h *DeviceHandler) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "UpdateDevice")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantDeviceUpdate)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	jwtClaims := appHelper.GetClaims()
	device, ok := decodeDevice(appHelper, w, r)
	if !ok {
		return
	}
	if device.DeviceID <= 0 {
		appHelper.WriteError(w, http.StatusBadRequest,
			"deviceId is required", "BAD_REQUEST", nil)
		return
	}
	originalUpdateTS := device.UpdateTS
	device.SecurityContext = jwtClaims.SecurityContext
	device.UpdateTS, device.UpdateUserID = model.Now(), jwtClaims.UserID
	updated, err := h.deviceRepository.UpdateDevice(
		r.Context(), device, originalUpdateTS)
	if err != nil {
		mapRepoError(appHelper, w, err,
			fmt.Sprintf("Failed to update device: deviceId=%d",
				device.DeviceID),
			"DEV_UPDATE_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: updated})
}

// DeleteDevice handles DELETE /device?deviceId=<id>&updateTs=<ts>. Its
// token stops working with it.
func (h *DeviceHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "DeleteDevice")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantDeviceDelete)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	deviceID, updateTS, ok := deleteQueryOf(appHelper, w, r, "deviceId")
	if !ok {
		return
	}
	jwtClaims := appHelper.GetClaims()
	err := h.deviceRepository.DeleteDevice(r.Context(), deviceID,
		jwtClaims.SecurityContext, updateTS, model.Now(), jwtClaims.UserID)
	if err != nil {
		mapRepoError(appHelper, w, err,
			fmt.Sprintf("Failed to delete device: deviceId=%d", deviceID),
			"DEV_DELETE_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{
		Data: map[string]any{"deviceId": deviceID},
	})
}

// IssueDeviceToken handles POST /deviceToken {deviceId, updateTs}: a new
// token for the device, answered once; the previous one stops working.
// The token is never read back, so a lost one is replaced.
func (h *DeviceHandler) IssueDeviceToken(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "IssueDeviceToken")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantDeviceIssueToken)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	jwtClaims := appHelper.GetClaims()
	var req model.DeviceTokenRequest
	if !decodeJSONBody(appHelper, w, r, maxDeviceBodyBytes, &req) {
		return
	}
	if req.DeviceID <= 0 {
		appHelper.WriteError(w, http.StatusBadRequest,
			"deviceId is required", "BAD_REQUEST", nil)
		return
	}
	token, err := auth.NewDeviceToken()
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Failed to generate the device token", "DEV_TOKEN_GEN", err)
		return
	}
	now := model.Now()
	err = h.deviceRepository.IssueDeviceToken(r.Context(), req.DeviceID,
		jwtClaims.SecurityContext, req.UpdateTS, auth.HashDeviceToken(token),
		now, jwtClaims.UserID)
	if err != nil {
		mapRepoError(appHelper, w, err,
			fmt.Sprintf("Failed to issue device token: deviceId=%d",
				req.DeviceID),
			"DEV_TOKEN_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{
		Data: model.DeviceTokenIssuedModel{
			DeviceID: req.DeviceID, DeviceToken: token, TokenIssuedTS: now,
			UpdateTS: now,
		},
	})
	appHelper.Info("DEV_TOKEN_ISSUED",
		fmt.Sprintf("Device token issued: deviceId=%d", req.DeviceID))
}

// DeviceCheckin handles POST /deviceCheckin from a Pi (src/pi-agent): its
// state, recorded against the device the Bearer token names, in that
// device's tenant. Answers {"data":{"configVersion":N}} so the Pi can tell
// its config is stale.
func (h *DeviceHandler) DeviceCheckin(w http.ResponseWriter, r *http.Request) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "DeviceCheckin")
	// The device token is the credential. Each exception is stated here:
	// no JWT: a Pi has no login and no session
	appHelper.SkipJWT()
	// the per-address bucket (MAX_UNAUTH_REQ_PER_MIN); a Pi checks in about
	// once a minute, and a 256-bit token leaves nothing to guess
	appHelper.CheckRateLimit()
	// the block list is keyed by JWT subject, and there is none
	appHelper.SkipUserBlock()
	// CSRF needs a session to derive the expected token from. The forgery
	// it guards against cannot happen here: no cookie authorizes the call,
	// only an Authorization header a cross-site page cannot set
	appHelper.SkipCSRF()
	// the token is the authorization: RetrieveDeviceByTokenHash
	appHelper.SkipAuthorization()
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	token, found := strings.CutPrefix(r.Header.Get("Authorization"),
		"Bearer ")
	if !found || !deviceTokenPattern.MatchString(token) {
		h.refuseCheckin(appHelper, w)
		return
	}
	device, err := h.deviceRepository.RetrieveDeviceByTokenHash(
		r.Context(), auth.HashDeviceToken(token))
	if errors.Is(err, repository.ErrNotFound) {
		h.refuseCheckin(appHelper, w)
		return
	}
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Check-in could not be recorded", "CHECKIN_READ_FAIL", err)
		return
	}

	body, ok := readBody(appHelper, w, r, model.DeviceCheckinMaxBodyBytes)
	if !ok {
		return
	}
	var req model.DeviceCheckinRequest
	if err := json.Unmarshal(body, &req); err != nil {
		appHelper.WriteError(w, http.StatusBadRequest,
			"Invalid request body", "INV_REQ_BODY", err)
		return
	}
	// The agent always says why it checked in; without it the body is not
	// a check-in (a bare null or {} would otherwise be stored as one).
	if req.Reason == "" {
		appHelper.WriteError(w, http.StatusBadRequest,
			"reason is required", "BAD_REQUEST", nil)
		return
	}
	record := &model.DeviceCheckinRecord{
		ReceivedTS:   model.Now(),
		Reason:       req.Reason,
		BodyJSON:     body,
		SourceIP:     security.ClientIP(r, h.appsrv.Config.TrustedProxyCount),
		AgentVersion: req.AgentVersion,
	}
	if req.SentAt != nil {
		sentTS := model.EpochSeconds(*req.SentAt)
		record.SentTS = &sentTS
	}
	configVersion, unknownAddressList, err :=
		h.deviceRepository.RecordDeviceCheckin(r.Context(), device.DeviceID,
			device.SecurityContext, record, req.BeaconList)
	if errors.Is(err, repository.ErrNotFound) {
		// deleted since the lookup
		h.refuseCheckin(appHelper, w)
		return
	}
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Check-in could not be recorded", "CHECKIN_WRITE_FAIL", err)
		return
	}
	if len(unknownAddressList) > 0 {
		appHelper.Warn("CHECKIN_UNKNOWN_BEACON", fmt.Sprintf(
			"deviceId=%d reported addresses it does not watch: %s",
			device.DeviceID, strings.Join(unknownAddressList, ", ")))
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{
		Data: model.DeviceCheckinReply{ConfigVersion: configVersion},
	})
	appHelper.Debug("CHECKIN_DONE", fmt.Sprintf(
		"Check-in recorded: deviceId=%d reason=%q", device.DeviceID,
		req.Reason))
}

// refuseCheckin answers a check-in without a live device's token: one 401
// and one message for a missing, malformed, unknown or revoked token.
func (h *DeviceHandler) refuseCheckin(
	appHelper *security.AppHelper, w http.ResponseWriter,
) {
	appHelper.WriteError(w, http.StatusUnauthorized,
		invalidDeviceTokenMessage, "CHECKIN_TOKEN_INVALID", nil)
}

// decodeDevice reads and checks a device body.
func decodeDevice(
	appHelper *security.AppHelper, w http.ResponseWriter, r *http.Request,
) (*model.DeviceModel, bool) {
	var device model.DeviceModel
	if !decodeJSONBody(appHelper, w, r, maxDeviceBodyBytes, &device) {
		return nil, false
	}
	if problem := model.DeviceProblem(&device); problem != "" {
		appHelper.WriteError(w, http.StatusBadRequest, problem,
			"BAD_REQUEST", nil)
		return nil, false
	}
	return &device, true
}

// listQueryOf reads a list route's ?<idListKey>=1,2 and ?afterUpdateTs=,
// answering 400 itself when either does not parse.
func listQueryOf(
	appHelper *security.AppHelper, w http.ResponseWriter, r *http.Request,
	idListKey string,
) ([]int, float64, bool) {
	idList, err := GetCommaSeparatedIntListFromQuery(r, idListKey)
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest, err.Error(),
			"PARSE_QUERY_FAIL", nil)
		return nil, 0, false
	}
	afterUpdateTs, err := GetOptionalFloatFromQuery(r, "afterUpdateTs")
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest, err.Error(),
			"PARSE_QUERY_FAIL", nil)
		return nil, 0, false
	}
	return idList, afterUpdateTs, true
}

// deleteQueryOf reads a delete route's ?<idKey>=<id>&updateTs=<ts>,
// answering 400 itself when either is missing or does not parse.
func deleteQueryOf(
	appHelper *security.AppHelper, w http.ResponseWriter, r *http.Request,
	idKey string,
) (int, float64, bool) {
	id, err := GetIntFromQuery(r, idKey)
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest, err.Error(),
			"PARSE_QUERY_FAIL", nil)
		return 0, 0, false
	}
	updateTS, err := GetFloatFromQuery(r, "updateTs")
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest, err.Error(),
			"PARSE_QUERY_FAIL", nil)
		return 0, 0, false
	}
	return id, updateTS, true
}
