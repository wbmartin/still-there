package handler

import (
	"net/http"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
)

// UserGroupHandler handles user group operations
type UserGroupHandler struct {
	appsrv        *app_lib.AppServices
	userGroupRepo *repository.UserGroupRepository
}

// NewUserGroupHandler creates a new user group handler
func NewUserGroupHandler(appServices *app_lib.AppServices, userGroupRepo *repository.UserGroupRepository) *UserGroupHandler {
	return &UserGroupHandler{
		appsrv:        appServices,
		userGroupRepo: userGroupRepo,
	}
}

// GetUserGroupList handles GET /userGroupList
func (h *UserGroupHandler) GetUserGroupList(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("UGGS0", "GetUserGroupList invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantAddUser)

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()

	// Get user groups
	groups, err := h.userGroupRepo.GetUserGroupList(r.Context(), jwtClaims.SecurityContext)
	if err != nil {
		logger.Error("UGGF0", "Failed to get user groups: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve user groups", logger, jwtClaims.Subject)
		return
	}

	logger.Debug("UGGD0", "User groups retrieved")
	wrapper := model.DataWrapper{Data: groups}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("UGGD1", "User group list retrieval completed")
}
