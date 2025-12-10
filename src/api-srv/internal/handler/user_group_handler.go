package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// ReadUserGroupList handles GET /userGroupList
func (h *UserGroupHandler) ReadUserGroupList(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("UGGS0", "GetUserGroupList invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserGroupRead)

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()

	// Read user groups
	groups, err := h.userGroupRepo.ReadUserGroupList(r.Context(), jwtClaims.SecurityContext)
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

// CreateUserGroup handles POST /userGroup
func (h *UserGroupHandler) CreateUserGroup(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("UGCS0", "CreateUserGroup invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserGroupCreate)

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()

	// Parse request body
	var group model.UserGroupModel
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger, jwtClaims.Subject)
		return
	}

	logger.Debug("UGCP0", "User group create request parsed")

	// Validate required fields
	if group.GroupName == "" {
		writeError(w, http.StatusBadRequest, "groupName is required", logger, jwtClaims.Subject)
		return
	}

	// Set creation and update metadata
	now := float64(time.Now().UnixNano()) / 1e9
	group.SecurityContext = jwtClaims.SecurityContext
	group.CreateTS = now
	group.CreateUserID = jwtClaims.UserID
	group.UpdateTS = now
	group.UpdateUserID = jwtClaims.UserID

	// Create user group
	createdGroup, err := h.userGroupRepo.CreateUserGroup(r.Context(), &group)
	if err != nil {
		logger.Error("UGCF0", "Failed to create user group: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to create user group", logger, jwtClaims.Subject)
		return
	}

	logger.Debug("UGCD0", "User group created")
	wrapper := model.DataWrapper{Data: createdGroup}
	writeJSON(w, http.StatusCreated, wrapper, logger)
	logger.Info("UGCD1", "User group creation completed")
}

// UpdateUserGroup handles PUT /userGroup
func (h *UserGroupHandler) UpdateUserGroup(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("UGUS0", "UpdateUserGroup invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserGroupUpdate)

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()

	// Parse request body
	var group model.UserGroupModel
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger, jwtClaims.Subject)
		return
	}

	logger.Debug("UGUP0", "User group update request parsed")

	// Validate required fields
	if group.UserGroupID == 0 || group.GroupName == "" {
		writeError(w, http.StatusBadRequest, "userGroupId and groupName are required", logger, jwtClaims.Subject)
		return
	}

	// Set update metadata
	clientUpdateTS := group.UpdateTS
	now := float64(time.Now().UnixNano()) / 1e9
	group.SecurityContext = jwtClaims.SecurityContext
	group.UpdateTS = now
	group.UpdateUserID = jwtClaims.UserID

	// Update user group with optimistic locking
	err := h.userGroupRepo.UpdateUserGroup(r.Context(), &group, clientUpdateTS)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			writeError(w, http.StatusConflict, "User group was modified by another user", logger, jwtClaims.Subject)
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "User group not found", logger, jwtClaims.Subject)
		default:
			logger.Error("UGUF0", "Failed to update user group: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Failed to update user group", logger, jwtClaims.Subject)
		}
		return
	}

	logger.Debug("UGUD0", "User group updated")
	wrapper := model.DataWrapper{Data: group}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("UGUD1", "User group update completed")
}

// DeleteUserGroup handles DELETE /userGroup
func (h *UserGroupHandler) DeleteUserGroup(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("UGDS0", "DeleteUserGroup invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserGroupDelete)

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()

	// Parse query parameters
	groupIDStr := r.URL.Query().Get("userGroupId")
	updateTSStr := r.URL.Query().Get("updateTs")

	if groupIDStr == "" || updateTSStr == "" {
		writeError(w, http.StatusBadRequest, "userGroupId and updateTs are required", logger, jwtClaims.Subject)
		return
	}

	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid userGroupId", logger, jwtClaims.Subject)
		return
	}

	updateTS, err := strconv.ParseFloat(updateTSStr, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid updateTs", logger, jwtClaims.Subject)
		return
	}

	// Delete user group with optimistic locking and assignment check
	now := float64(time.Now().UnixNano()) / 1e9
	deletedID, err := h.userGroupRepo.DeleteUserGroup(r.Context(), groupID, jwtClaims.SecurityContext, updateTS, now, jwtClaims.UserID)
	if err != nil {
		if strings.Contains(err.Error(), "cannot delete group") {
			writeError(w, http.StatusConflict, err.Error(), logger, jwtClaims.Subject)
		} else {
			switch {
			case errors.Is(err, repository.ErrConflict):
				writeError(w, http.StatusConflict, "User group was modified by another user", logger, jwtClaims.Subject)
			case errors.Is(err, repository.ErrNotFound):
				writeError(w, http.StatusNotFound, "User group not found", logger, jwtClaims.Subject)
			default:
				logger.Error("UGDF0", "Failed to delete user group: "+err.Error())
				writeError(w, http.StatusInternalServerError, "Failed to delete user group", logger, jwtClaims.Subject)
			}
		}
		return
	}

	data := map[string]interface{}{"message": "User group deleted successfully", "userGroupId": deletedID}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("UGDD0", "User group deletion completed")
}
