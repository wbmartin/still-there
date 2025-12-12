package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
)

// UserHandler handles user management operations
type UserHandler struct {
	appsrv   *app_lib.AppServices
	userRepo *repository.UserRepository
}

// NewUserHandler creates a new user handler
func NewUserHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository) *UserHandler {
	return &UserHandler{
		appsrv:   appServices,
		userRepo: userRepo,
	}
}

// ReadUserList handles GET /userList
func (h *UserHandler) ReadUserList(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("ULS0", "ReadUserList invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserCreate)

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()
	username := sec.GetUsername()

	// Parse optional userIdList query parameter
	idList, err := GetCommaSeparatedIntListFromQuery(r, "userIdList")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), logger, username)
		return
	}

	// Read users (filtered by security context and optional idList)
	users, err := h.userRepo.ReadUserList(r.Context(), jwtClaims.SecurityContext, idList)
	if err != nil {
		logger.Error("ULF0", "Failed to read users: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve users", logger, username)
		return
	}

	logger.Debug("ULD0", "Users read")
	wrapper := model.DataWrapper{Data: users}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("ULD1", "User list read completed")
}

// ReadGrantList handles GET /grantList
func (h *UserHandler) ReadGrantList(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("RLS0", "ReadGrantList invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserCreate)

	if !sec.CheckAllChecksComplete() {
		return
	}

	// Convert GrantIDToShortCodeMap to array of objects
	grantList := make([]map[string]interface{}, 0, len(model.GrantIDToShortCodeMap))
	for id, text := range model.GrantIDToShortCodeMap {
		grantList = append(grantList, map[string]interface{}{
			"id":   id,
			"text": text,
		})
	}

	logger.Debug("GLD0", "Grant list read")
	wrapper := model.DataWrapper{Data: grantList}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("GLD1", "Grant list read completed")
}

// CreateUser handles POST /user
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("UCS0", "CreateUser invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserCreate)

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()
	username := sec.GetUsername()

	// Parse request body
	var user model.AppUserModel
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger, username)
		return
	}

	logger.Debug("UCP0", "User creation request parsed")

	// Validate required fields
	if user.UserName == "" || user.FirstName == "" || user.LastName == "" || user.Email == "" {
		writeError(w, http.StatusBadRequest, "userName, firstName, lastName, and email are required", logger, username)
		return
	}

	// Set security context and creation metadata
	now := float64(time.Now().UnixNano()) / 1e9
	user.SecurityContext = jwtClaims.SecurityContext
	user.CreateTS = now
	user.CreateUserID = jwtClaims.UserID
	user.UpdateTS = now
	user.UpdateUserID = jwtClaims.UserID

	// Hash password if provided
	if user.PwHash != "" {
		hashed, err := app_lib.HashPassword(user.PwHash)
		if err != nil {
			logger.Error("UCH0", "Failed to hash password: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Failed to process password", logger, username)
			return
		}
		user.PwHash = hashed
	}

	// Create user
	createdUser, err := h.userRepo.CreateUser(r.Context(), &user)
	if err != nil {
		logger.Error("UCF0", "Failed to create user: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to create user", logger, username)
		return
	}

	logger.Debug("UCD0", "User created")
	wrapper := model.DataWrapper{Data: createdUser}
	writeJSON(w, http.StatusCreated, wrapper, logger)
	logger.Info("UCD1", "User creation completed")
}

// UpdateUser handles PUT /user
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("UUS0", "UpdateUser invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserChangePassword) // Assuming change grant

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()
	username := sec.GetUsername()

	// Parse request body
	var user model.AppUserModel
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger, username)
		return
	}

	logger.Debug("UUP0", "User update request parsed")

	// Validate required fields
	if user.AppUserID == 0 || user.UserName == "" {
		writeError(w, http.StatusBadRequest, "appUserId and userName are required", logger, username)
		return
	}

	// Set update metadata
	clientUpdateTS := user.UpdateTS
	now := float64(time.Now().UnixNano()) / 1e9
	user.SecurityContext = jwtClaims.SecurityContext
	user.UpdateTS = now
	user.UpdateUserID = jwtClaims.UserID

	// Update user with optimistic locking
	err := h.userRepo.UpdateUser(r.Context(), &user, clientUpdateTS)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			writeError(w, http.StatusConflict, "User was modified by another user", logger, username)
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "User not found", logger, username)
		default:
			logger.Error("UUF0", "Failed to update user: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Failed to update user", logger, username)
		}
		return
	}

	logger.Debug("UUD0", "User updated")
	wrapper := model.DataWrapper{Data: user}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("UUD1", "User update completed")
}

// DeleteUser handles DELETE /user
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("UDS0", "DeleteUser invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserBlock) // Assuming delete grant

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()
	username := sec.GetUsername()

	// Parse query parameters
	userID, err := GetIntFromQuery(r, "userId")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), logger, username)
		return
	}

	updateTS, err := GetFloatFromQuery(r, "updateTs")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), logger, username)
		return
	}

	// Delete user with optimistic locking
	now := float64(time.Now().UnixNano()) / 1e9
	deletedID, err := h.userRepo.DeleteUser(r.Context(), userID, jwtClaims.SecurityContext, updateTS, now, jwtClaims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			writeError(w, http.StatusConflict, "User was modified by another user", logger, username)
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "User not found", logger, username)
		default:
			logger.Error("UDF0", "Failed to delete user: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Failed to delete user", logger, username)
		}
		return
	}

	data := map[string]interface{}{"message": "User deleted successfully", "appUserId": deletedID}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("UDD0", "User deletion completed")
}

// AddUserListToGroupHandler handles POST /addUserListToGroup
func (h *UserHandler) AddUserListToGroupHandler(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("AUG0", "AddUserListToGroup invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantAddUserToGroup)

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()
	username := sec.GetUsername()

	// Parse request
	var req struct {
		UserIds      []int     `json:"userIds"`
		GroupId      int       `json:"groupId"`
		UpdateTsList []float64 `json:"updateTsList"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger, username)
		return
	}

	if len(req.UserIds) == 0 || len(req.UpdateTsList) != len(req.UserIds) {
		writeError(w, http.StatusBadRequest, "userIds and updateTsList must be provided and have matching lengths", logger, username)
		return
	}

	// Add users to group
	updatedUsers, err := h.userRepo.AddUserListToGroup(r.Context(), req.UserIds, req.GroupId, req.UpdateTsList, jwtClaims.UserID, jwtClaims.SecurityContext)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			writeError(w, http.StatusConflict, "Concurrent update detected", logger, username)
			return
		}
		logger.Error("AUGF0", "Failed to add users to group: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to add users to group", logger, username)
		return
	}

	logger.Info("AUGS0", "Users added to group successfully")
	wrapper := model.DataWrapper{Data: updatedUsers}
	writeJSON(w, http.StatusOK, wrapper, logger)
}

// RemoveUserListFromGroupHandler handles POST /removeUserListFromGroup
func (h *UserHandler) RemoveUserListFromGroupHandler(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("RUG0", "RemoveUserListFromGroup invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantRemoveUserFromGroup)

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()
	username := sec.GetUsername()

	// Parse request
	var req struct {
		UserIds      []int     `json:"userIds"`
		GroupId      int       `json:"groupId"`
		UpdateTsList []float64 `json:"updateTsList"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger, username)
		return
	}

	if len(req.UserIds) == 0 || len(req.UpdateTsList) != len(req.UserIds) {
		writeError(w, http.StatusBadRequest, "userIds and updateTsList must be provided and have matching lengths", logger, username)
		return
	}

	// Remove users from group
	updatedUsers, err := h.userRepo.RemoveUserListFromGroup(r.Context(), req.UserIds, req.GroupId, req.UpdateTsList, jwtClaims.UserID, jwtClaims.SecurityContext)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			writeError(w, http.StatusConflict, "Concurrent update detected", logger, username)
			return
		}
		logger.Error("RUGF0", "Failed to remove users from group: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to remove users from group", logger, username)
		return
	}

	logger.Info("RUGS0", "Users removed from group successfully")
	wrapper := model.DataWrapper{Data: updatedUsers}
	writeJSON(w, http.StatusOK, wrapper, logger)
}
