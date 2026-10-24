package handler

import (
	"context"
	"fmt"
	"net/http"

	"stillthere-api/internal/app"
	"stillthere-api/internal/model"
	"stillthere-api/internal/security"
)

// The bands a device watches (beacon). Every write bumps the device's
// config version: the band list is part of the Pi's config.

// BeaconRepositoryInterface is what BeaconHandler needs from
// repository.BeaconRepository.
type BeaconRepositoryInterface interface {
	RetrieveBeaconList(
		ctx context.Context, idList []int, deviceID int, securityContext int,
		afterUpdateTs float64,
	) ([]*model.BeaconModel, error)
	CreateBeacon(
		ctx context.Context, beacon *model.BeaconModel, securityContext int,
	) (*model.BeaconModel, error)
	UpdateBeacon(
		ctx context.Context, beacon *model.BeaconModel,
		originalUpdateTS float64,
	) (*model.BeaconModel, error)
	DeleteBeacon(
		ctx context.Context, beaconID int, securityContext int,
		originalUpdateTS float64, deleteTS float64, deleteUserID int,
	) error
}

// BeaconHandler serves the beacon routes.
type BeaconHandler struct {
	appsrv           *app.Services
	beaconRepository BeaconRepositoryInterface
}

// NewBeaconHandler creates the beacon handler.
func NewBeaconHandler(
	appServices *app.Services, beaconRepository BeaconRepositoryInterface,
) *BeaconHandler {
	return &BeaconHandler{
		appsrv: appServices, beaconRepository: beaconRepository,
	}
}

// RetrieveBeaconList handles GET
// /beaconList?deviceId=<id>&beaconIdList=1,2&afterUpdateTs=<ts>; each
// filter is optional.
func (h *BeaconHandler) RetrieveBeaconList(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "RetrieveBeaconList")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantBeaconRead)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	idList, afterUpdateTs, ok := listQueryOf(appHelper, w, r, "beaconIdList")
	if !ok {
		return
	}
	deviceID, err := GetOptionalIntFromQuery(r, "deviceId")
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest, err.Error(),
			"PARSE_QUERY_FAIL", nil)
		return
	}
	beaconList, err := h.beaconRepository.RetrieveBeaconList(r.Context(),
		idList, deviceID, appHelper.GetClaims().SecurityContext,
		afterUpdateTs)
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Failed to retrieve beacons", "BCN_LIST_FAIL", err)
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: beaconList})
}

// CreateBeacon handles POST /beacon {deviceId, address, label}. The device
// must be a live one of the caller's tenant (404 otherwise).
func (h *BeaconHandler) CreateBeacon(w http.ResponseWriter, r *http.Request) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "CreateBeacon")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantBeaconCreate)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	jwtClaims := appHelper.GetClaims()
	beacon, ok := decodeBeacon(appHelper, w, r)
	if !ok {
		return
	}
	if beacon.DeviceID <= 0 {
		appHelper.WriteError(w, http.StatusBadRequest,
			"deviceId is required", "BAD_REQUEST", nil)
		return
	}
	now := model.Now()
	beacon.CreateTS, beacon.CreateUserID = now, jwtClaims.UserID
	beacon.UpdateTS, beacon.UpdateUserID = now, jwtClaims.UserID
	created, err := h.beaconRepository.CreateBeacon(
		r.Context(), beacon, jwtClaims.SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to create beacon",
			"BCN_CREATE_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusCreated,
		model.DataWrapper{Data: created})
}

// UpdateBeacon handles PUT /beacon: address and label, under the
// optimistic lock. The device is fixed at create.
func (h *BeaconHandler) UpdateBeacon(w http.ResponseWriter, r *http.Request) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "UpdateBeacon")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantBeaconUpdate)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	jwtClaims := appHelper.GetClaims()
	beacon, ok := decodeBeacon(appHelper, w, r)
	if !ok {
		return
	}
	if beacon.BeaconID <= 0 {
		appHelper.WriteError(w, http.StatusBadRequest,
			"beaconId is required", "BAD_REQUEST", nil)
		return
	}
	originalUpdateTS := beacon.UpdateTS
	beacon.SecurityContext = jwtClaims.SecurityContext
	beacon.UpdateTS, beacon.UpdateUserID = model.Now(), jwtClaims.UserID
	updated, err := h.beaconRepository.UpdateBeacon(
		r.Context(), beacon, originalUpdateTS)
	if err != nil {
		mapRepoError(appHelper, w, err,
			fmt.Sprintf("Failed to update beacon: beaconId=%d",
				beacon.BeaconID),
			"BCN_UPDATE_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: updated})
}

// DeleteBeacon handles DELETE /beacon?beaconId=<id>&updateTs=<ts>.
func (h *BeaconHandler) DeleteBeacon(w http.ResponseWriter, r *http.Request) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "DeleteBeacon")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantBeaconDelete)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	beaconID, updateTS, ok := deleteQueryOf(appHelper, w, r, "beaconId")
	if !ok {
		return
	}
	jwtClaims := appHelper.GetClaims()
	err := h.beaconRepository.DeleteBeacon(r.Context(), beaconID,
		jwtClaims.SecurityContext, updateTS, model.Now(), jwtClaims.UserID)
	if err != nil {
		mapRepoError(appHelper, w, err,
			fmt.Sprintf("Failed to delete beacon: beaconId=%d", beaconID),
			"BCN_DELETE_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{
		Data: map[string]any{"beaconId": beaconID},
	})
}

// decodeBeacon reads a beacon body, normalises its address and checks it.
func decodeBeacon(
	appHelper *security.AppHelper, w http.ResponseWriter, r *http.Request,
) (*model.BeaconModel, bool) {
	var beacon model.BeaconModel
	if !decodeJSONBody(appHelper, w, r, maxDeviceBodyBytes, &beacon) {
		return nil, false
	}
	if problem := model.BeaconProblem(&beacon); problem != "" {
		appHelper.WriteError(w, http.StatusBadRequest, problem,
			"BAD_REQUEST", nil)
		return nil, false
	}
	return &beacon, true
}
