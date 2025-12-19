package handler

import (
	"encoding/json"
	"errors"
	"net/http"

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

// RetrieveUserList handles GET /userList
func (h *UserHandler) RetrieveUserList(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserRead)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("ULS0", "ReadUserList invoked")

	jwtClaims := app.GetClaims()

	// Parse optional userIdList query parameter
	idList, err := GetCommaSeparatedIntListFromQuery(r, "userIdList")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	// Parse optional afterUpdateTs query parameter
	afterUpdateTs, err := GetFloatFromQuery(r, "afterUpdateTs")
	if err != nil {
		afterUpdateTs = 0 // Default to 0 if not provided
	}

	// Read users (filtered by security context, optional idList, and afterUpdateTs)
	users, err := h.userRepo.RetrieveUserList(r.Context(), jwtClaims.SecurityContext, idList, afterUpdateTs)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError, "Failed to retrieve users", "ULF0", err)
		return
	}

	app.Debug("ULD0", "Users read")

	// Ensure users is never nil
	if users == nil {
		users = []*model.AppUserModel{}
	}

	// Return as JSON
	wrapper := model.DataWrapper{Data: users}
	writeJSON(w, http.StatusOK, wrapper, h.appsrv.Logger)

	app.Info("ULD1", "User list read completed")
}

// RetrieveGrantList handles GET /grantList
func (h *UserHandler) RetrieveGrantList(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserCreate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("RLS0", "ReadGrantList invoked")

	// Convert GrantIDToShortCodeMap to array of objects
	grantList := make([]map[string]interface{}, 0, len(model.GrantIDToShortCodeMap))
	for id, text := range model.GrantIDToShortCodeMap {
		grantList = append(grantList, map[string]interface{}{
			"id":   id,
			"text": text,
		})
	}

	app.Debug("GLD0", "Grant list read")
	wrapper := model.DataWrapper{Data: grantList}
	writeJSON(w, http.StatusOK, wrapper, h.appsrv.Logger)
	app.Info("GLD1", "Grant list read completed")
}

// CreateUser handles POST /user
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserCreate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("UCS0", "CreateUser invoked")

	jwtClaims := app.GetClaims()

	// Parse request body
	var user model.AppUserModel
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "BRQ0", nil)
		return
	}

	app.Debug("UCP0", "User creation request parsed")

	// Validate required fields
	if user.UserName == "" || user.FirstName == "" || user.LastName == "" || user.Email == "" {
		app.WriteError(w, http.StatusBadRequest, "userName, firstName, lastName, and email are required", "BRQ0", nil)
		return
	}

	// Set security context and creation metadata
	now := CurrentTimestamp()
	user.SecurityContext = jwtClaims.SecurityContext
	user.CreateTS = now
	user.CreateUserID = jwtClaims.UserID
	user.UpdateTS = now
	user.UpdateUserID = jwtClaims.UserID

	// Hash password if provided
	if user.PwHash != "" {
		hashed, err := app_lib.HashPassword(user.PwHash)
		if err != nil {
			app.WriteError(w, http.StatusInternalServerError, "Failed to process password", "UCH0", err)
			return
		}
		user.PwHash = hashed
	}

	// Create user
	createdUser, err := h.userRepo.CreateUser(r.Context(), &user)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError, "Failed to create user", "UCF0", err)
		return
	}

	app.Debug("UCD0", "User created")
	wrapper := model.DataWrapper{Data: createdUser}
	writeJSON(w, http.StatusCreated, wrapper, h.appsrv.Logger)
	app.Info("UCD1", "User creation completed")
}

// UpdateUser handles PUT /user
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("UUS0", "UpdateUser invoked")

	jwtClaims := app.GetClaims()

	// Parse request body
	var user model.AppUserModel
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "BRQ0", nil)
		return
	}

	app.Debug("UUP0", "User update request parsed")

	// Validate required fields
	if user.AppUserID == 0 || user.UserName == "" {
		app.WriteError(w, http.StatusBadRequest, "appUserId and userName are required", "BRQ0", nil)
		return
	}

	// Set update metadata
	clientUpdateTS := user.UpdateTS
	now := CurrentTimestamp()
	user.SecurityContext = jwtClaims.SecurityContext
	user.UpdateTS = now
	user.UpdateUserID = jwtClaims.UserID

	// Update user with optimistic locking
	err := h.userRepo.UpdateUser(r.Context(), &user, clientUpdateTS)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			app.WriteError(w, http.StatusConflict, "User was modified by another user", "CNF0", nil)
		case errors.Is(err, repository.ErrNotFound):
			app.WriteError(w, http.StatusNotFound, "User not found", "NFN0", nil)
		default:
			app.WriteError(w, http.StatusInternalServerError, "Failed to update user", "UUF0", err)
		}
		return
	}

	app.Debug("UUD0", "User updated")
	wrapper := model.DataWrapper{Data: user}
	writeJSON(w, http.StatusOK, wrapper, h.appsrv.Logger)
	app.Info("UUD1", "User update completed")
}

// DeleteUser handles DELETE /user
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserDelete)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("UDS0", "DeleteUser invoked")

	jwtClaims := app.GetClaims()

	// Parse query parameters
	userID, err := GetIntFromQuery(r, "userId")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	updateTS, err := GetFloatFromQuery(r, "updateTs")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	// Delete user with optimistic locking
	now := CurrentTimestamp()
	deletedID, err := h.userRepo.DeleteUser(r.Context(), userID, jwtClaims.SecurityContext, updateTS, now, jwtClaims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			app.WriteError(w, http.StatusConflict, "User was modified by another user", "CNF0", nil)
		case errors.Is(err, repository.ErrNotFound):
			app.WriteError(w, http.StatusNotFound, "User not found", "NFN0", nil)
		default:
			app.WriteError(w, http.StatusInternalServerError, "Failed to delete user", "UDF0", err)
		}
		return
	}

	data := map[string]interface{}{"message": "User deleted successfully", "appUserId": deletedID}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, h.appsrv.Logger)
	app.Info("UDD0", "User deletion completed")
}

// AddUserListToGroupHandler handles POST /addUserListToGroup
func (h *UserHandler) AddUserListToGroupHandler(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("AUG0", "AddUserListToGroup invoked")

	jwtClaims := app.GetClaims()

	// Parse request
	var req struct {
		UserIds      []int     `json:"userIds"`
		GroupId      int       `json:"groupId"`
		UpdateTsList []float64 `json:"updateTsList"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "BRQ0", nil)
		return
	}

	if len(req.UserIds) == 0 || len(req.UpdateTsList) != len(req.UserIds) {
		app.WriteError(w, http.StatusBadRequest, "userIds and updateTsList must be provided and have matching lengths", "BRQ0", nil)
		return
	}

	// Add users to group
	updatedUsers, err := h.userRepo.AddUserListToGroup(r.Context(), req.UserIds, req.GroupId, req.UpdateTsList, jwtClaims.UserID, jwtClaims.SecurityContext)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			app.WriteError(w, http.StatusConflict, "Concurrent update detected", "CNF0", nil)
			return
		}
		app.WriteError(w, http.StatusInternalServerError, "Failed to add users to group", "AGF0", err)
		return
	}

	app.Info("AGS0", "Users added to group successfully")
	wrapper := model.DataWrapper{Data: updatedUsers}
	writeJSON(w, http.StatusOK, wrapper, h.appsrv.Logger)
}

// RemoveUserListFromGroupHandler handles POST /removeUserListFromGroup
func (h *UserHandler) RemoveUserListFromGroupHandler(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserGroupUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("RUG0", "RemoveUserListFromGroup invoked")

	jwtClaims := app.GetClaims()

	// Parse request
	var req struct {
		UserIds      []int     `json:"userIds"`
		GroupId      int       `json:"groupId"`
		UpdateTsList []float64 `json:"updateTsList"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "BRQ0", nil)
		return
	}

	if len(req.UserIds) == 0 || len(req.UpdateTsList) != len(req.UserIds) {
		app.WriteError(w, http.StatusBadRequest, "userIds and updateTsList must be provided and have matching lengths", "BRQ0", nil)
		return
	}

	// Remove users from group
	updatedUsers, err := h.userRepo.RemoveUserListFromGroup(r.Context(), req.UserIds, req.GroupId, req.UpdateTsList, jwtClaims.UserID, jwtClaims.SecurityContext)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			app.WriteError(w, http.StatusConflict, "Concurrent update detected", "CNF0", nil)
			return
		}
		app.WriteError(w, http.StatusInternalServerError, "Failed to remove users from group", "RGF0", err)
		return
	}

	app.Info("RGS0", "Users removed from group successfully")
	wrapper := model.DataWrapper{Data: updatedUsers}
	writeJSON(w, http.StatusOK, wrapper, h.appsrv.Logger)
}

// ChangeOtherUserPassword handles POST /changeUserPassword
func (h *UserHandler) ChangeOtherUserPassword(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantUserChangePassword)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("CPWS", "Change user password invoked")

	jwtClaims := app.GetClaims()

	// Parse request body
	var req struct {
		UserID      int    `json:"userId"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "BRQ0", nil)
		return
	}

	// Validate required fields
	if req.UserID == 0 || req.NewPassword == "" {
		app.WriteError(w, http.StatusBadRequest, "userId and newPassword are required", "BRQ0", nil)
		return
	}

	// Basic password validation (minimum length)
	if len(req.NewPassword) < 8 {
		app.WriteError(w, http.StatusBadRequest, "New password must be at least 8 characters long", "BRQ0", nil)
		return
	}

	// Verify the user exists and get their details for logging
	users, err := h.userRepo.RetrieveUserList(r.Context(), jwtClaims.SecurityContext, []int{req.UserID}, 0)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError, "Failed to retrieve user", "USF0", err)
		return
	}
	if len(users) == 0 {
		app.WriteError(w, http.StatusNotFound, "User not found", "NFN0", nil)
		return
	}
	user := users[0]

	// Hash new password
	hashedPassword, err := app_lib.HashPassword(req.NewPassword)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError, "Failed to process password", "UCH0", err)
		return
	}

	// Update password
	now := CurrentTimestamp()
	err = h.userRepo.UpdateUserPassword(r.Context(), req.UserID, hashedPassword, now, jwtClaims.UserID)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError, "Failed to update password", "UUF0", err)
		return
	}

	app.Debug("CPWD", "Password updated for user: "+user.UserName)

	wrapper := model.DataWrapper{Data: map[string]string{"message": "Password changed successfully"}}
	writeJSON(w, http.StatusOK, wrapper, h.appsrv.Logger)
	app.Info("CPWD", "User password change completed")
}
