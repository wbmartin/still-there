package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
func NewUserGroupHandler(appServices *app_lib.AppServices,
	userGroupRepo *repository.UserGroupRepository) *UserGroupHandler {
	return &UserGroupHandler{
		appsrv:        appServices,
		userGroupRepo: userGroupRepo,
	}
}

// RetrieveUserGroupList handles GET /userGroupList
func (h *UserGroupHandler) RetrieveUserGroupList(w http.ResponseWriter,
	r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("URS0", "ReadUserGroupList invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupRead)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Read user groups
	groups, err := h.userGroupRepo.RetrieveUserGroupList(r.Context(),
		jwtClaims.SecurityContext)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError,
			"Failed to retrieve user groups", "UGF0", err)
		return
	}

	logger.Debug("UGD0", "User groups retrieved")

	// Ensure groups is never nil
	if groups == nil {
		groups = []*model.UserGroupModel{}
	}

	wrapper := model.DataWrapper{Data: groups}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("UGD1", "User group list retrieval completed")
}

// CreateUserGroup handles POST /userGroup
func (h *UserGroupHandler) CreateUserGroup(w http.ResponseWriter,
	r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("UCS0", "CreateUserGroup invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupCreate)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Parse request body
	var group model.UserGroupModel
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "ERR", nil)
		return
	}

	logger.Debug("UCP0", "User group create request parsed")

	// Validate required fields
	if group.GroupName == "" {
		app.WriteError(w, http.StatusBadRequest, "groupName is required", "ERR", nil)
		return
	}

	// Set creation and update metadata
	now := CurrentTimestamp()
	group.SecurityContext = jwtClaims.SecurityContext
	group.CreateTS = now
	group.CreateUserID = jwtClaims.UserID
	group.UpdateTS = now
	group.UpdateUserID = jwtClaims.UserID

	// Create user group
	createdGroup, err := h.userGroupRepo.CreateUserGroup(r.Context(), &group)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError,
			"Failed to create user group", "UCF0", err)
		return
	}

	logger.Debug("UCD0", "User group created")
	wrapper := model.DataWrapper{Data: createdGroup}
	writeJSON(w, http.StatusCreated, wrapper, logger)
	logger.Info("UCD1", "User group creation completed")
}

// UpdateUserGroup handles PUT /userGroup
func (h *UserGroupHandler) UpdateUserGroup(w http.ResponseWriter,
	r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("UUS0", "UpdateUserGroup invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Parse request body
	var group model.UserGroupModel
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "ERR", nil)
		return
	}

	logger.Debug("UUP0", "User group update request parsed")

	// Validate required fields
	if group.UserGroupID == 0 || group.GroupName == "" {
		app.WriteError(w, http.StatusBadRequest,
			"userGroupId and groupName are required", "BRQ0", nil)
		return
	}

	// Set update metadata
	clientUpdateTS := group.UpdateTS
	now := CurrentTimestamp()
	group.SecurityContext = jwtClaims.SecurityContext
	group.UpdateTS = now
	group.UpdateUserID = jwtClaims.UserID

	// Update user group with optimistic locking
	err := h.userGroupRepo.UpdateUserGroup(r.Context(), &group, clientUpdateTS)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			app.WriteError(w, http.StatusConflict,
				"User group was modified by another user", "CNF0", nil)
		case errors.Is(err, repository.ErrNotFound):
			app.WriteError(w, http.StatusNotFound, "User group not found", "NFN0", nil)
		default:
			app.WriteError(w, http.StatusInternalServerError,
				"Failed to update user group", "UUF0", err)
		}
		return
	}

	logger.Debug("UUD0", "User group updated")
	wrapper := model.DataWrapper{Data: group}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("UUD1", "User group update completed")
}

// DeleteUserGroup handles DELETE /userGroup
func (h *UserGroupHandler) DeleteUserGroup(w http.ResponseWriter,
	r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("UDS0", "DeleteUserGroup invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupDelete)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Parse query parameters
	groupID, err := GetIntFromQuery(r, "userGroupId")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "ERR", nil)
		return
	}

	updateTS, err := GetFloatFromQuery(r, "updateTs")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "ERR", nil)
		return
	}

	// Delete user group with optimistic locking and assignment check
	now := CurrentTimestamp()
	deletedID, err := h.userGroupRepo.DeleteUserGroup(r.Context(), groupID,
		jwtClaims.SecurityContext, updateTS, now, jwtClaims.UserID)
	if err != nil {
		if strings.Contains(err.Error(), "cannot delete group") {
			app.WriteError(w, http.StatusConflict, err.Error(), "ERR", nil)
		} else {
			switch {
			case errors.Is(err, repository.ErrConflict):
				app.WriteError(w, http.StatusConflict,
					"User group was modified by another user", "CNF0", nil)
			case errors.Is(err, repository.ErrNotFound):
				app.WriteError(w, http.StatusNotFound, "User group not found", "ERR", nil)
			default:
				app.WriteError(w, http.StatusInternalServerError,
					"Failed to delete user group", "UDF0", err)
			}
		}
		return
	}

	data := map[string]interface{}{
		"message": "User group deleted successfully", "userGroupId": deletedID}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("UDD0", "User group deletion completed")
}
