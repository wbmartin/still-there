package handler

import (
	"context"
	"net/http"

	"stillthere-api/internal/app"
	"stillthere-api/internal/model"
	"stillthere-api/internal/security"
)

// The caller's tenant's reminder settings: one row, created by the first
// PUT. GET before then is 404.

// TenantSettingRepositoryInterface is what TenantSettingHandler needs from
// repository.TenantSettingRepository.
type TenantSettingRepositoryInterface interface {
	RetrieveTenantSetting(
		ctx context.Context, securityContext int,
	) (*model.TenantSettingModel, error)
	SaveTenantSetting(
		ctx context.Context, setting *model.TenantSettingModel,
		originalUpdateTS float64,
	) (*model.TenantSettingModel, error)
}

// TenantSettingHandler serves GET and PUT /tenantSetting.
type TenantSettingHandler struct {
	appsrv                  *app.Services
	tenantSettingRepository TenantSettingRepositoryInterface
}

// NewTenantSettingHandler creates the tenant setting handler.
func NewTenantSettingHandler(
	appServices *app.Services,
	tenantSettingRepository TenantSettingRepositoryInterface,
) *TenantSettingHandler {
	return &TenantSettingHandler{
		appsrv:                  appServices,
		tenantSettingRepository: tenantSettingRepository,
	}
}

// RetrieveTenantSetting handles GET /tenantSetting.
func (h *TenantSettingHandler) RetrieveTenantSetting(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv,
		"RetrieveTenantSetting")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantTenantSettingRead)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	setting, err := h.tenantSettingRepository.RetrieveTenantSetting(
		r.Context(), appHelper.GetClaims().SecurityContext)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to retrieve the settings",
			"TSET_GET_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: setting})
}

// UpdateTenantSetting handles PUT /tenantSetting {timeZone,
// remindIntervalMin, dailyReminderTime, updateTs}: saves the caller's
// tenant's row under the optimistic lock, or creates it on the first save.
func (h *TenantSettingHandler) UpdateTenantSetting(
	w http.ResponseWriter, r *http.Request,
) {
	appHelper := security.NewAppHelper(w, r, h.appsrv,
		"UpdateTenantSetting")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantTenantSettingUpdate)
	if !appHelper.CheckAllChecksComplete() {
		return
	}

	jwtClaims := appHelper.GetClaims()
	var setting model.TenantSettingModel
	if !decodeJSONBody(appHelper, w, r, maxDeviceBodyBytes, &setting) {
		return
	}
	if problem := model.TenantSettingProblem(&setting); problem != "" {
		appHelper.WriteError(w, http.StatusBadRequest, problem,
			"BAD_REQUEST", nil)
		return
	}
	originalUpdateTS := setting.UpdateTS
	setting.SecurityContext = jwtClaims.SecurityContext
	setting.UpdateTS, setting.UpdateUserID = model.Now(), jwtClaims.UserID
	saved, err := h.tenantSettingRepository.SaveTenantSetting(
		r.Context(), &setting, originalUpdateTS)
	if err != nil {
		mapRepoError(appHelper, w, err, "Failed to save the settings",
			"TSET_SAVE_FAIL")
		return
	}
	appHelper.WriteJSON(w, http.StatusOK, model.DataWrapper{Data: saved})
}
