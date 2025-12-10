package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserCreate)

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()

	// Read users (filtered by security context)
	users, err := h.userRepo.ReadUserList(r.Context(), jwtClaims.SecurityContext)
	if err != nil {
		logger.Error("ULF0", "Failed to read users: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve users", logger, jwtClaims.Subject)
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
	logger.Info("GLS0", "ReadGrantList invoked")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
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
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserCreate)

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()

	// Parse request body
	var user model.AppUserModel
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger, jwtClaims.Subject)
		return
	}

	logger.Debug("UCP0", "User creation request parsed")

	// Validate required fields
	if user.UserName == "" || user.FirstName == "" || user.LastName == "" || user.Email == "" {
		writeError(w, http.StatusBadRequest, "userName, firstName, lastName, and email are required", logger, jwtClaims.Subject)
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
			writeError(w, http.StatusInternalServerError, "Failed to process password", logger, jwtClaims.Subject)
			return
		}
		user.PwHash = hashed
	}

	// Create user
	createdUser, err := h.userRepo.CreateUser(r.Context(), &user)
	if err != nil {
		logger.Error("UCF0", "Failed to create user: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to create user", logger, jwtClaims.Subject)
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
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserChangePassword) // Assuming change grant

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()

	// Parse request body
	var user model.AppUserModel
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger, jwtClaims.Subject)
		return
	}

	logger.Debug("UUP0", "User update request parsed")

	// Validate required fields
	if user.AppUserID == 0 || user.UserName == "" {
		writeError(w, http.StatusBadRequest, "appUserId and userName are required", logger, jwtClaims.Subject)
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
			writeError(w, http.StatusConflict, "User was modified by another user", logger, jwtClaims.Subject)
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "User not found", logger, jwtClaims.Subject)
		default:
			logger.Error("UUF0", "Failed to update user: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Failed to update user", logger, jwtClaims.Subject)
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
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimitAndUserBlock()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantUserBlock) // Assuming delete grant

	if !sec.CheckAllChecksComplete() {
		return
	}

	jwtClaims := sec.GetClaims()

	// Parse query parameters
	userIDStr := r.URL.Query().Get("userId")
	updateTSStr := r.URL.Query().Get("updateTs")

	if userIDStr == "" || updateTSStr == "" {
		writeError(w, http.StatusBadRequest, "userId and updateTs are required", logger, jwtClaims.Subject)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid userId", logger, jwtClaims.Subject)
		return
	}

	updateTS, err := strconv.ParseFloat(updateTSStr, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid updateTs", logger, jwtClaims.Subject)
		return
	}

	// Delete user with optimistic locking
	now := float64(time.Now().UnixNano()) / 1e9
	deletedID, err := h.userRepo.DeleteUser(r.Context(), userID, jwtClaims.SecurityContext, updateTS, now, jwtClaims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			writeError(w, http.StatusConflict, "User was modified by another user", logger, jwtClaims.Subject)
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "User not found", logger, jwtClaims.Subject)
		default:
			logger.Error("UDF0", "Failed to delete user: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Failed to delete user", logger, jwtClaims.Subject)
		}
		return
	}

	data := map[string]interface{}{"message": "User deleted successfully", "appUserId": deletedID}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("UDD0", "User deletion completed")
}
