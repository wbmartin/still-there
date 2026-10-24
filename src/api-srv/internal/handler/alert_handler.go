package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"stillthere-api/internal/app"
	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
	"stillthere-api/internal/security"
)

// The tenant's alarms. The watchdog raises, reminds and resolves them;
// here a person reads them and acknowledges one (reminders slow to one a
// day) or turns it off (no more reminders). Each of those queues a text to
// the alarm's contacts naming who did it.

// AlertRepositoryInterface is what AlertHandler needs from
// repository.AlertRepository.
type AlertRepositoryInterface interface {
	RetrieveAlertList(
		ctx context.Context, deviceID int, securityContext int,
		resolvedSince float64,
	) ([]*model.AlertModel, error)
	AcknowledgeAlert(
		ctx context.Context, alertID int, securityContext int,
		originalUpdateTS float64, actorName string, now float64,
		actorUserID int,
	) (*model.AlertModel, error)
	TurnOffAlert(
		ctx context.Context, alertID int, securityContext int,
		originalUpdateTS float64, actorName string, now float64,
		actorUserID int,
	) (*model.AlertModel, error)
}

// AlertActorRepositoryInterface reads the caller's own login, for the name
// the contacts are told acted.
type AlertActorRepositoryInterface interface {
	RetrieveUserByUserName(
		ctx context.Context, userName string,
	) (*model.AppUserModel, error)
}

// AlertHandler serves the alert routes.
type AlertHandler struct {
	appsrv          *app.Services
	alertRepository AlertRepositoryInterface
	actorRepository AlertActorRepositoryInterface
}

// NewAlertHandler creates the alert handler.
func NewAlertHandler(
	appServices *app.Services,
	alertRepository AlertRepositoryInterface,
	actorRepository AlertActorRepositoryInterface,
) *AlertHandler {
	return &AlertHandler{
		appsrv:          appServices,
		alertRepository: alertRepository,
		actorRepository: actorRepository,
	}
}

// RetrieveAlertList handles GET /alertList?deviceId=<id> (optional): every
// open alarm, then those resolved in the last AlertRecentResolvedSec.
func (h *AlertHandler) RetrieveAlertList(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "RetrieveAlertList")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantAlertRead)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	deviceID, err := GetOptionalIntFromQuery(r, "deviceId")
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest, err.Error(),
			"PARSE_QUERY_FAIL", nil)
		return
	}
	alertList, err := h.alertRepository.RetrieveAlertList(r.Context(),
		deviceID, appHelper.GetClaims().SecurityContext,
		model.Now()-model.AlertRecentResolvedSec)
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Failed to retrieve alerts", "ALERT_LIST_FAIL", err)
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: alertList})
}

// alertChange is AcknowledgeAlert or TurnOffAlert.
type alertChange func(
	ctx context.Context, alertID int, securityContext int,
	originalUpdateTS float64, actorName string, now float64,
	actorUserID int,
) (*model.AlertModel, error)

// AcknowledgeAlert handles POST /alertAcknowledge {alertId, updateTs}:
// from raised only; another state, resolved included, is 409.
func (h *AlertHandler) AcknowledgeAlert(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "AcknowledgeAlert")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantAlertUpdate)
	if !appHelper.CheckAllChecksComplete() {
		return
	}
	h.changeAlert(appHelper, w, r, h.alertRepository.AcknowledgeAlert,
		"ALERT_ACK")
}

// TurnOffAlert handles POST /alertTurnOff {alertId, updateTs}: from raised
// or acknowledged; a turned-off or resolved alarm is 409.
func (h *AlertHandler) TurnOffAlert(w http.ResponseWriter, r *http.Request) {
	appHelper := security.NewAppHelper(w, r, h.appsrv, "TurnOffAlert")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantAlertUpdate)
	if !appHelper.CheckAllChecksComplete() {
		return
	}
	h.changeAlert(appHelper, w, r, h.alertRepository.TurnOffAlert,
		"ALERT_OFF")
}

// changeAlert is the body both actions share, after their checks.
func (h *AlertHandler) changeAlert(
	appHelper *security.AppHelper,
	w http.ResponseWriter,
	r *http.Request,
	change alertChange,
	logCode string,
) {
	var req model.AlertActionRequest
	if !decodeJSONBody(appHelper, w, r, maxDeviceBodyBytes, &req) {
		return
	}
	if req.AlertID <= 0 {
		appHelper.WriteError(w, http.StatusBadRequest,
			"alertId is required", "BAD_REQUEST", nil)
		return
	}
	actorName, ok := h.actorNameOf(appHelper, w, r)
	if !ok {
		return
	}
	jwtClaims := appHelper.GetClaims()
	changed, err := change(r.Context(), req.AlertID,
		jwtClaims.SecurityContext, req.UpdateTS, actorName, model.Now(),
		jwtClaims.UserID)
	if err != nil {
		mapRepoError(appHelper, w, err,
			fmt.Sprintf("Failed to change alert: alertId=%d", req.AlertID),
			logCode+"_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: changed})
	appHelper.Info(logCode+"_DONE", fmt.Sprintf("alertId=%d is now %s",
		changed.AlertID, changed.AlertState))
}

// actorNameOf is the caller's display name (first and last), or the user
// name when both are blank, read from their own login by the JWT's subject
// as reporterOf does. A row that is not the JWT's uid is 401.
func (h *AlertHandler) actorNameOf(
	appHelper *security.AppHelper, w http.ResponseWriter, r *http.Request,
) (string, bool) {
	jwtClaims := appHelper.GetClaims()
	user, err := h.actorRepository.RetrieveUserByUserName(
		r.Context(), jwtClaims.Subject)
	if errors.Is(err, repository.ErrNotFound) ||
		(err == nil && user.AppUserID != jwtClaims.UserID) {
		appHelper.WriteError(w, http.StatusUnauthorized,
			"Your session no longer matches a login", "ALERT_ACTOR", err)
		return "", false
	}
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Failed to read your login", "ALERT_ACTOR", err)
		return "", false
	}
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = user.UserName
	}
	return name, true
}
