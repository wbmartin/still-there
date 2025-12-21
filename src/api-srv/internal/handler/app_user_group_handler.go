package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
)

// AppUserGroupHandler handles user group operations
type AppUserGroupHandler struct {
	appsrv           *app_lib.AppServices
	appUserGroupRepo *repository.AppUserGroupRepository
}

// NewAppUserGroupHandler creates a new user group handler
func NewAppUserGroupHandler(appServices *app_lib.AppServices,
	appUserGroupRepo *repository.AppUserGroupRepository) *AppUserGroupHandler {
	return &AppUserGroupHandler{
		appsrv:           appServices,
		appUserGroupRepo: appUserGroupRepo,
	}
}

// RetrieveAppUserGroupList handles GET /userGroupList
func (h *AppUserGroupHandler) RetrieveAppUserGroupList(w http.ResponseWriter,
	r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupRead)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("RD_UGRP_LST_S", "ReadUserGroupList invoked")

	jwtClaims := app.GetClaims()

	// Read user groups
	groups, err := h.appUserGroupRepo.RetrieveAppUserGroupList(r.Context(),
		jwtClaims.SecurityContext)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError,
			"Failed to retrieve user groups", "USER_GROUP_LIST_FAIL", err)
		return
	}

	app.Debug("UGRP_LST_READ", "User groups retrieved")

	// Ensure groups is never nil
	if groups == nil {
		groups = []*model.AppUserGroupModel{}
	}

	wrapper := model.DataWrapper{Data: groups}
	writeJSON(w, http.StatusOK, wrapper, h.appsrv.Logger)
	app.Info("UGRP_LST_DONE", "User group list retrieval completed")
}

// CreateAppUserGroup handles POST /userGroup
func (h *AppUserGroupHandler) CreateAppUserGroup(w http.ResponseWriter,
	r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupCreate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("CR_UGRP_START", "CreateUserGroup invoked")

	jwtClaims := app.GetClaims()

	// Parse request body
	var group model.AppUserGroupModel
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "GENERAL_ERROR", nil)
		return
	}

	app.Debug("CR_UGRP_PARSE", "User group create request parsed")

	// Validate required fields
	if group.GroupName == "" {
		app.WriteError(w, http.StatusBadRequest, "groupName is required", "GENERAL_ERROR", nil)
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
	createdGroup, err := h.appUserGroupRepo.CreateAppUserGroup(r.Context(), &group)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError,
			"Failed to create user group", "CREATE_USER_GROUP_FAIL", err)
		return
	}

	app.Debug("UGRP_CREATED", "User group created")
	wrapper := model.DataWrapper{Data: createdGroup}
	writeJSON(w, http.StatusCreated, wrapper, h.appsrv.Logger)
	app.Info("CR_UGRP_DONE", fmt.Sprintf("User group creation completed for groupId=%d, groupName='%s'", createdGroup.AppUserGroupID, createdGroup.GroupName))
}

// UpdateAppUserGroup handles PUT /userGroup
func (h *AppUserGroupHandler) UpdateAppUserGroup(w http.ResponseWriter,
	r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("UP_UGRP_START", "UpdateUserGroup invoked")

	jwtClaims := app.GetClaims()

	// Parse request body
	var group model.AppUserGroupModel
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "GENERAL_ERROR", nil)
		return
	}

	app.Debug("UP_UGRP_PARSE", "User group update request parsed")

	// Validate required fields
	if group.AppUserGroupID == 0 || group.GroupName == "" {
		app.WriteError(w, http.StatusBadRequest,
			"appUserGroupId and groupName are required", "BAD_REQUEST", nil)
		return
	}

	app.Info("UP_UGRP_PARAMS", fmt.Sprintf("UpdateUserGroup for groupId=%d, groupName='%s'", group.AppUserGroupID, group.GroupName))

	// Set update metadata
	clientUpdateTS := group.UpdateTS
	now := CurrentTimestamp()
	group.SecurityContext = jwtClaims.SecurityContext
	group.UpdateTS = now
	group.UpdateUserID = jwtClaims.UserID

	// Update user group with optimistic locking
	err := h.appUserGroupRepo.UpdateAppUserGroup(r.Context(), &group, clientUpdateTS)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			app.WriteError(w, http.StatusConflict,
				"User group was modified by another user", "CONFLICT_ERROR", nil)
		case errors.Is(err, repository.ErrNotFound):
			app.WriteError(w, http.StatusNotFound, "User group not found", "NOT_FOUND_ERROR", nil)
		default:
			app.WriteError(w, http.StatusInternalServerError,
				"Failed to update user group", "USER_GROUP_UPDATE_FAIL", err)
		}
		return
	}

	app.Debug("UGRP_UPDATED", "User group updated")
	wrapper := model.DataWrapper{Data: group}
	writeJSON(w, http.StatusOK, wrapper, h.appsrv.Logger)
	app.Info("UP_UGRP_DONE", fmt.Sprintf("User group update completed for groupId=%d, groupName='%s'", group.AppUserGroupID, group.GroupName))
}

// DeleteAppUserGroup handles DELETE /userGroup
func (h *AppUserGroupHandler) DeleteAppUserGroup(w http.ResponseWriter,
	r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupDelete)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("DEL_UGRP_START", "DeleteUserGroup invoked")

	jwtClaims := app.GetClaims()

	// Parse query parameters
	groupID, err := GetIntFromQuery(r, "appUserGroupId")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "GENERAL_ERROR", nil)
		return
	}

	updateTS, err := GetFloatFromQuery(r, "updateTs")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "GENERAL_ERROR", nil)
		return
	}

	app.Info("DEL_UGRP_PARAMS", fmt.Sprintf("DeleteUserGroup for groupId=%d", groupID))

	// Delete user group with optimistic locking and assignment check
	now := CurrentTimestamp()
	deletedID, err := h.appUserGroupRepo.DeleteAppUserGroup(r.Context(), groupID, jwtClaims.SecurityContext, updateTS, now, jwtClaims.UserID)
	if err != nil {
		if strings.Contains(err.Error(), "cannot delete group") {
			app.WriteError(w, http.StatusConflict, err.Error(), "GENERAL_ERROR", err)
		} else {
			switch {
			case errors.Is(err, repository.ErrConflict):
				app.WriteError(w, http.StatusConflict, "User group was modified by another user", "CONFLICT_ERROR", err)
			case errors.Is(err, repository.ErrNotFound):
				app.WriteError(w, http.StatusNotFound, "User group not found", "GENERAL_ERROR", err)
			default:
				app.WriteError(w, http.StatusInternalServerError, "Failed to delete user group", "DEL_UGRP_FAIL", err)
			}
		}
		return
	}

	data := map[string]interface{}{
		"message": "User group deleted successfully", "userGroupId": deletedID}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, h.appsrv.Logger)
	app.Info("DEL_UGRP_DONE", fmt.Sprintf("User group deletion completed for groupId=%d", deletedID))
}
