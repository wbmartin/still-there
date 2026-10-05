package handler

import (
	"fmt"
	"net/http"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
	"github.com/wbmartin/starter-api/internal/security"
)

// AppUserGroupHandler handles user group operations
type AppUserGroupHandler struct {
	appsrv           *app.Services
	appUserGroupRepo *repository.AppUserGroupRepository
}

// NewAppUserGroupHandler creates a new user group handler
func NewAppUserGroupHandler(appServices *app.Services,
	appUserGroupRepo *repository.AppUserGroupRepository) *AppUserGroupHandler {
	return &AppUserGroupHandler{
		appsrv:           appServices,
		appUserGroupRepo: appUserGroupRepo,
	}
}

// RetrieveAppUserGroupList handles GET /appUserGroupList
func (h *AppUserGroupHandler) RetrieveAppUserGroupList(w http.ResponseWriter,
	r *http.Request) {
	// Security checks
	appHelper := security.NewAppHelper(
		w, r, h.appsrv, "RetrieveAppUserGroupList",
	)
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantUserGroupRead)

	if !appHelper.CheckAllChecksComplete() {
		return
	}

	appHelper.Info("RD_UGRP_LST_S", "ReadUserGroupList invoked")

	jwtClaims := appHelper.GetClaims()

	// Parse query parameters
	appUserGroupIdList, err := GetCommaSeparatedIntListFromQuery(
		r,
		"appUserGroupIdList",
	)
	if err != nil {
		appHelper.WriteError(w, http.StatusBadRequest,
			"appUserGroupIdList must be comma-separated integers",
			"RD_UGRP_LST_PARSE_ER", err)
		return
	}

	appHelper.Info(
		"RD_UGRP_LST_PARAMS",
		fmt.Sprintf(
			"Retrieving user groups: idList=%v, securityContext=%d",
			appUserGroupIdList,
			jwtClaims.SecurityContext,
		),
	)

	// Read user groups
	groups, err := h.appUserGroupRepo.RetrieveAppUserGroupList(r.Context(),
		appUserGroupIdList, jwtClaims.SecurityContext)
	if err != nil {
		appHelper.WriteError(w, http.StatusInternalServerError,
			"Failed to retrieve user groups", "USER_GROUP_LIST_FAIL", err)
		return
	}

	appHelper.Debug("UGRP_LST_READ", "User groups retrieved")

	// Ensure groups is never nil
	if groups == nil {
		groups = []*model.AppUserGroupModel{}
	}

	wrapper := model.DataWrapper{Data: groups}
	appHelper.WriteJSON(w, http.StatusOK, wrapper)
	appHelper.Info("UGRP_LST_DONE", "User group list retrieval completed")
}

// CreateAppUserGroup handles POST /appUserGroup
func (h *AppUserGroupHandler) CreateAppUserGroup(w http.ResponseWriter,
	r *http.Request) {
	// Security checks
	appHelper := security.NewAppHelper(w, r, h.appsrv, "CreateAppUserGroup")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantUserGroupCreate)

	if !appHelper.CheckAllChecksComplete() {
		return
	}

	appHelper.Info("CR_UGRP_START", "CreateUserGroup invoked")

	jwtClaims := appHelper.GetClaims()

	// Parse request body
	var group model.AppUserGroupModel
	if !decodeJSONBody(appHelper, w, r, maxLoginLayerBodyBytes, &group) {
		return
	}

	appHelper.Debug("CR_UGRP_PARSE", "User group create request parsed")

	// Validate required fields
	if group.GroupName == "" {
		appHelper.WriteError(
			w,
			http.StatusBadRequest,
			"groupName is required",
			"GENERAL_ERROR",
			nil,
		)
		return
	}

	// Set creation and update metadata
	now := model.Now()
	group.SecurityContext = jwtClaims.SecurityContext
	group.CreateTS = now
	group.CreateUserID = jwtClaims.UserID
	group.UpdateTS = now
	group.UpdateUserID = jwtClaims.UserID

	// Create user group
	createdGroup, err := h.appUserGroupRepo.CreateAppUserGroup(
		r.Context(),
		&group,
		jwtClaims.SecurityContext,
	)
	if err != nil {
		mapRepoError(appHelper, w, err,
			"Failed to create user group", "CREATE_USER_GROUP_FAIL")
		return
	}

	appHelper.Debug("UGRP_CREATED", "User group created")
	wrapper := model.DataWrapper{Data: createdGroup}
	appHelper.WriteJSON(w, http.StatusCreated, wrapper)
	appHelper.Info(
		"CR_UGRP_DONE",
		fmt.Sprintf(
			"User group creation completed for groupId=%d, groupName='%s'",
			createdGroup.AppUserGroupID,
			createdGroup.GroupName,
		),
	)
}

// UpdateAppUserGroup handles PUT /appUserGroup
func (h *AppUserGroupHandler) UpdateAppUserGroup(w http.ResponseWriter,
	r *http.Request) {
	// Security checks
	appHelper := security.NewAppHelper(w, r, h.appsrv, "UpdateAppUserGroup")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantUserGroupUpdate)

	if !appHelper.CheckAllChecksComplete() {
		return
	}

	appHelper.Info("UP_UGRP_START", "UpdateUserGroup invoked")

	jwtClaims := appHelper.GetClaims()

	// Parse request body
	var group model.AppUserGroupModel
	if !decodeJSONBody(appHelper, w, r, maxLoginLayerBodyBytes, &group) {
		return
	}

	appHelper.Debug("UP_UGRP_PARSE", "User group update request parsed")

	// Validate required fields
	if group.AppUserGroupID == 0 || group.GroupName == "" {
		appHelper.WriteError(w, http.StatusBadRequest,
			"appUserGroupId and groupName are required", "BAD_REQUEST", nil)
		return
	}

	appHelper.Info(
		"UP_UGRP_PARAMS",
		fmt.Sprintf(
			"UpdateUserGroup for groupId=%d, groupName='%s'",
			group.AppUserGroupID,
			group.GroupName,
		),
	)

	// Set update metadata
	clientUpdateTS := group.UpdateTS
	now := model.Now()
	group.SecurityContext = jwtClaims.SecurityContext
	group.UpdateTS = now
	group.UpdateUserID = jwtClaims.UserID

	// Update user group with optimistic locking
	err := h.appUserGroupRepo.UpdateAppUserGroup(
		r.Context(),
		&group,
		clientUpdateTS,
		jwtClaims.SecurityContext,
	)
	if err != nil {
		mapRepoError(appHelper, w, err,
			"Failed to update user group", "USER_GROUP_UPDATE_FAIL")
		return
	}

	appHelper.Debug("UGRP_UPDATED", "User group updated")
	// The group's grants are in every member's tokens: their sessions end.
	memberUserNameList, err := h.appUserGroupRepo.RetrieveMemberUserNameList(
		r.Context(), group.AppUserGroupID, jwtClaims.SecurityContext,
	)
	if err != nil {
		appHelper.WriteError(
			w,
			http.StatusInternalServerError,
			"Saved; reading the group's members failed",
			"SES_REVOKE",
			err,
		)
		return
	}
	if !revokeUserSessionList(appHelper, w, h.appsrv, memberUserNameList...) {
		return
	}
	wrapper := model.DataWrapper{Data: group}
	appHelper.WriteJSON(w, http.StatusOK, wrapper)
	appHelper.Info(
		"UP_UGRP_DONE",
		fmt.Sprintf(
			"User group update completed for groupId=%d, groupName='%s'",
			group.AppUserGroupID,
			group.GroupName,
		),
	)
}

// DeleteAppUserGroup handles DELETE /appUserGroup
func (h *AppUserGroupHandler) DeleteAppUserGroup(w http.ResponseWriter,
	r *http.Request) {
	// Security checks
	appHelper := security.NewAppHelper(w, r, h.appsrv, "DeleteAppUserGroup")
	appHelper.CheckJWT()
	appHelper.CheckRateLimitAndUserBlock()
	appHelper.CheckCSRF()
	appHelper.CheckAuthorization(model.GrantUserGroupDelete)

	if !appHelper.CheckAllChecksComplete() {
		return
	}

	appHelper.Info("DEL_UGRP_START", "DeleteUserGroup invoked")

	jwtClaims := appHelper.GetClaims()

	// Parse query parameters
	groupID, err := GetIntFromQuery(r, "appUserGroupId")
	if err != nil {
		appHelper.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
			"GENERAL_ERROR",
			nil,
		)
		return
	}

	updateTS, err := GetFloatFromQuery(r, "updateTs")
	if err != nil {
		appHelper.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
			"GENERAL_ERROR",
			nil,
		)
		return
	}

	appHelper.Info(
		"DEL_UGRP_PARAMS",
		fmt.Sprintf("DeleteUserGroup for groupId=%d", groupID),
	)

	// Delete user group with optimistic locking and assignment check
	now := model.Now()
	deletedID, err := h.appUserGroupRepo.DeleteAppUserGroup(
		r.Context(),
		groupID,
		jwtClaims.SecurityContext,
		updateTS,
		now,
		jwtClaims.UserID,
	)
	if err != nil {
		mapRepoError(appHelper, w, err,
			"Failed to delete user group", "DEL_UGRP_FAIL")
		return
	}

	data := map[string]any{
		"message": "User group deleted successfully", "userGroupId": deletedID}
	wrapper := model.DataWrapper{Data: data}
	appHelper.WriteJSON(w, http.StatusOK, wrapper)
	appHelper.Info(
		"DEL_UGRP_DONE",
		fmt.Sprintf("User group deletion completed for groupId=%d", deletedID),
	)
}
