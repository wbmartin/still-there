package handler

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
)

// encryptTOTPSeed encrypts a TOTP seed using AES-GCM
func encryptTOTPSeed(plainSeed, keyHex string) (string, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plainSeed), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptTOTPSeed decrypts a TOTP seed using AES-GCM
func decryptTOTPSeed(encryptedSeed, keyHex string) (string, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedSeed)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

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

	// Parse request body into CreateUserRequest
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "BRQ0", nil)
		return
	}

	app.Debug("UCP0", "User creation request parsed")

	// Validate required fields
	if req.UserName == "" || req.FirstName == "" || req.LastName == "" || req.Email == "" {
		app.WriteError(w, http.StatusBadRequest, "userName, firstName, lastName, and email are required", "BRQ0", nil)
		return
	}

	// Validate password length
	if len(req.PwRaw) < 8 {
		app.WriteError(w, http.StatusBadRequest, "Password must be at least 8 characters long", "BRQ0", nil)
		return
	}

	// Create AppUserModel from request
	now := CurrentTimestamp()
	user := model.AppUserModel{
		UserName:                   req.UserName,
		FirstName:                  req.FirstName,
		LastName:                   req.LastName,
		Email:                      req.Email,
		AppUserGroupMembershipList: req.AppUserGroupMembershipList,
		MFAExempt:                  req.MFAExempt,
		SecurityContext:            jwtClaims.SecurityContext,
		CreateTS:                   now,
		CreateUserID:               jwtClaims.UserID,
		UpdateTS:                   now,
		UpdateUserID:               jwtClaims.UserID,
	}

	// Hash password
	hashed, err := app_lib.HashPassword(req.PwRaw)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError, "Failed to process password", "UCH0", err)
		return
	}
	user.PwHash = hashed

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
	app.CheckAuthorization(model.GrantChangeOtherUserPassword)

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

// UserSelfCreateHandler handles user self-registration (public endpoint)
func UserSelfCreateHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// No auth required for self-create
		appServices.Logger.Info("USC0", "User self-create request started", "username", "anonymous")

		// Create app helper without JWT check
		app := app_lib.NewAppHelper(w, r, appServices)
		app.SkipJWT() // Public endpoint
		app.CheckRateLimit()
		app.SkipUserBlock()
		app.SkipCSRF()          // No CSRF for public registration
		app.SkipAuthorization() // Public

		if !app.CheckAllChecksComplete() {
			return
		}

		// Parse request
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			app.WriteError(w, http.StatusBadRequest, "Invalid request body", "IRB0", err)
			return
		}

		// Validate input
		if req.Username == "" || req.Password == "" {
			app.WriteError(w, http.StatusBadRequest, "Username and password are required", "BRQ0", nil)
			return
		}

		if len(req.Password) < 8 {
			app.WriteError(w, http.StatusBadRequest, "Password must be at least 8 characters long", "BRQ0", nil)
			return
		}

		// Check if username is available
		existingUser, err := userRepo.RetrieveUserByUserName(r.Context(), req.Username)
		if err != nil {
			if strings.Contains(err.Error(), "user not found") {
				// Username is available
			} else {
				app.WriteError(w, http.StatusInternalServerError, "Error checking username availability", "ISE0", err)
				return
			}
		}
		if existingUser != nil {
			app.WriteError(w, http.StatusConflict, "Username already taken", "UNTK", nil)
			return
		}

		// Hash password
		hashedPassword, err := app_lib.HashPassword(req.Password)
		if err != nil {
			app.WriteError(w, http.StatusInternalServerError, "Failed to process password", "UCH0", err)
			return
		}

		// Create user
		now := CurrentTimestamp()
		defaultSecurityContext := 1 // Default security context for self-created users
		defaultSecurityGroupId := 3 // "New User" group ID
		newUser := &model.AppUserModel{
			UserName:                   req.Username,
			PwHash:                     hashedPassword,
			FirstName:                  "",
			LastName:                   "",
			Email:                      "",
			SecurityContext:            defaultSecurityContext,
			AppUserGroupMembershipList: []int{defaultSecurityGroupId}, // "New User" group
			MFAExempt:                  false,
			TOTPSeed:                   nil, // Will be set during MFA setup
			CreateTS:                   now,
			CreateUserID:               0, // System
			UpdateTS:                   now,
			UpdateUserID:               0, // System
		}

		userID, err := userRepo.CreateUser(r.Context(), newUser)
		if err != nil {
			app.WriteError(w, http.StatusInternalServerError, "Failed to create user", "UCF0", err)
			return
		}

		appServices.Logger.Info("USCC", "User self-created successfully", "username", req.Username, "userId", userID)

		wrapper := model.DataWrapper{Data: map[string]string{"message": "User created successfully"}}
		writeJSON(w, http.StatusCreated, wrapper, appServices.Logger)
	}
}

// MFASetupHandler handles MFA setup for authenticated users
func MFASetupHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security checks
		app := app_lib.NewAppHelper(w, r, appServices)
		app.CheckRateLimitAndUserBlock()
		app.SkipCSRF()
		app.SkipAuthorization() // MFA setup is user-specific

		var userName string
		// Check for temp token in request body first
		var tempReq struct {
			TempToken string `json:"tempToken"`
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes)) // Reset body for later
		if err := json.Unmarshal(bodyBytes, &tempReq); err == nil && tempReq.TempToken != "" {
			// Temp token path
			tempToken := tempReq.TempToken
			// Get username from temp token
			userNameBytes, err := appServices.KV.Get("TEMP_AUTH_" + tempToken)
			if err != nil || userNameBytes == "" {
				app.WriteError(w, http.StatusUnauthorized, "Invalid temp token", "ITTK", nil)
				return
			}
			userName = userNameBytes
		} else {
			// JWT path, there is no standard authorization/csrf for the temp token path
			app.CheckJWT()
			if !app.CheckAllChecksComplete() {
				return
			}
			jwtClaims := app.GetClaims()
			userName = jwtClaims.Subject
		}

		app.Info("MFS0", "MFA setup request started for user: "+userName)

		// Get current user
		user, err := userRepo.RetrieveUserByUserName(r.Context(), userName)
		if err != nil {
			app.WriteError(w, http.StatusInternalServerError, "Failed to retrieve user", "USF0", err)
			return
		}

		if user.MFAExempt {
			app.WriteError(w, http.StatusBadRequest, "MFA not required for this user", "MFAN", nil)
			return
		}

		// Generate TOTP secret if not exists
		var secret string
		if user.TOTPSeed != nil && *user.TOTPSeed != "" {
			// Decrypt existing seed
			decrypted, err := decryptTOTPSeed(*user.TOTPSeed, appServices.Config.TOTPEncryptionKey)
			if err != nil {
				app.WriteError(w, http.StatusInternalServerError, "Failed to decrypt TOTP secret", "TSD0", err)
				return
			}
			secret = decrypted
		} else {
			// Generate new TOTP secret
			secretBytes := make([]byte, 32)
			if _, err := rand.Read(secretBytes); err != nil {
				app.WriteError(w, http.StatusInternalServerError, "Failed to generate secret", "SGF0", err)
				return
			}
			secret = base32.StdEncoding.EncodeToString(secretBytes)

			// Encrypt and save
			encrypted, err := encryptTOTPSeed(secret, appServices.Config.TOTPEncryptionKey)
			if err != nil {
				app.WriteError(w, http.StatusInternalServerError, "Failed to encrypt TOTP secret", "TSE0", err)
				return
			}
			_, err = userRepo.UpdateUserTOTPSeed(r.Context(), user.AppUserID, encrypted, user.UpdateTS, user.AppUserID)
			if err != nil {
				app.WriteError(w, http.StatusInternalServerError, "Failed to save TOTP secret", "TSS0", err)
				return
			}
		}

		// Generate QR code URL
		processedAppName := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(appServices.Config.AppName, "")
		qrCodeUrl := "otpauth://totp/" + processedAppName + ":" + userName + "?secret=" + secret + "&issuer=" + appServices.Config.AppName

		// Generate QR code image
		png, err := qrcode.Encode(qrCodeUrl, qrcode.Medium, 256)
		if err != nil {
			app.WriteError(w, http.StatusInternalServerError, "Failed to generate QR code", "QRCG", err)
			return
		}
		qrCodeImage := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

		response := map[string]interface{}{
			"secret":      secret,
			"qrCodeImage": qrCodeImage,
		}

		wrapper := model.DataWrapper{Data: response}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
		app.Info("MFS1", "MFA setup data provided")
	}
}

// MFAVerifyHandler handles MFA verification
func MFAVerifyHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security checks
		app := app_lib.NewAppHelper(w, r, appServices)
		app.CheckRateLimitAndUserBlock()
		app.SkipCSRF()
		app.SkipAuthorization()

		// Parse request
		var req struct {
			TempToken string `json:"tempToken"`
			TOTPCode  string `json:"totpCode"`
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes)) // Reset body
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			app.WriteError(w, http.StatusBadRequest, "Invalid request body", "IRB0", err)
			return
		}

		var userName string
		if req.TempToken != "" {
			// Temp token path
			userNameBytes, err := appServices.KV.Get("TEMP_AUTH_" + req.TempToken)
			if err != nil || userNameBytes == "" {
				app.WriteError(w, http.StatusUnauthorized, "Invalid temp token", "ITTK", nil)
				return
			}
			userName = userNameBytes
		} else {
			// JWT path
			app.CheckJWT()
			if !app.CheckAllChecksComplete() {
				return
			}
			jwtClaims := app.GetClaims()
			userName = jwtClaims.Subject
		}

		app.Info("MFV0", "MFA verify request started for user: "+userName)

		// Get user
		user, err := userRepo.RetrieveUserByUserName(r.Context(), userName)
		if err != nil {
			app.WriteError(w, http.StatusInternalServerError, "Failed to retrieve user", "USF0", err)
			return
		}

		if user.TOTPSeed == nil || *user.TOTPSeed == "" {
			app.WriteError(w, http.StatusBadRequest, "MFA not set up", "MFNS", nil)
			return
		}

		// Decrypt TOTP secret
		decryptedSecret, err := decryptTOTPSeed(*user.TOTPSeed, appServices.Config.TOTPEncryptionKey)
		if err != nil {
			app.WriteError(w, http.StatusInternalServerError, "Failed to decrypt TOTP secret", "TSD0", err)
			return
		}

		// Validate TOTP code
		valid := totp.Validate(req.TOTPCode, decryptedSecret)
		if !valid {
			app.WriteError(w, http.StatusUnauthorized, "Invalid TOTP code", "ITCC", nil)
			return
		}

		if req.TempToken != "" {
			// Complete authentication for onboarding
			// Get grants
			grantsMap, err := userRepo.GetGrantsForGroups(r.Context(), user.AppUserGroupMembershipList)
			if err != nil {
				app.WriteError(w, http.StatusInternalServerError, "Failed to get grants", "GRFL", err)
				return
			}

			// Issue JWT
			jwt, sessionID, err := createJWTForUser(user, grantsMap[user.SecurityContext], appServices)
			if err != nil {
				app.WriteError(w, http.StatusInternalServerError, "Failed to create JWT", "JWFL", err)
				return
			}

			// Set JWT cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "auth",
				Value:    jwt,
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
			})

			// CSRF token
			csrfToken := sessionID[len(sessionID)-3:]

			// Remove temp token
			appServices.KV.Set("TEMP_AUTH_"+req.TempToken, "", "0") // Expire immediately

			// Return full user data
			response := model.Auth2ResponseModel{
				AppUserID:                    user.AppUserID,
				UserName:                     user.UserName,
				FirstName:                    user.FirstName,
				LastName:                     user.LastName,
				SecurityContext:              user.SecurityContext,
				GrantList:                    grantsMap[user.SecurityContext],
				AvailableSecurityContextList: make([]int, 0, len(grantsMap)), // TODO: populate
				MFARequired:                  false,                          // Now verified
			}
			for ctx := range grantsMap {
				response.AvailableSecurityContextList = append(response.AvailableSecurityContextList, ctx)
			}

			wrapper := model.Auth2ResponseWrapper{User: response, CsrfToken: csrfToken}
			writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
			app.Info("MFV1", "MFA onboarding completed for user: "+userName)
		} else {
			// For logged in users, just confirm
			wrapper := model.DataWrapper{Data: map[string]string{"message": "MFA verified successfully"}}
			writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
			app.Info("MFV1", "MFA verification successful for user: "+userName)
		}
	}
}

// AdminResetUserMFAHandler handles admin reset of user MFA
func AdminResetUserMFAHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security checks
		app := app_lib.NewAppHelper(w, r, appServices)
		app.CheckJWT()
		app.CheckRateLimitAndUserBlock()
		app.CheckCSRF()
		app.CheckAuthorization(model.GrantUserUpdate)

		if !app.CheckAllChecksComplete() {
			return
		}

		app.Info("ARM0", "Admin reset MFA request started")

		// Parse request
		var req struct {
			UserID int `json:"userId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			app.WriteError(w, http.StatusBadRequest, "Invalid request body", "IRB0", err)
			return
		}

		// Get target user
		users, err := userRepo.RetrieveUserList(r.Context(), app.GetClaims().SecurityContext, []int{req.UserID}, 0)
		if err != nil {
			app.WriteError(w, http.StatusInternalServerError, "Failed to retrieve user", "USF0", err)
			return
		}
		if len(users) == 0 {
			app.WriteError(w, http.StatusNotFound, "User not found", "NFN0", nil)
			return
		}

		targetUser := users[0]

		// Reset MFA
		newUpdateTS, err := userRepo.UpdateUserTOTPSeed(r.Context(), targetUser.AppUserID, "", targetUser.UpdateTS, app.GetClaims().UserID)
		if err != nil {
			app.WriteError(w, http.StatusInternalServerError, "Failed to reset MFA", "MFRE", err)
			return
		}

		// Also reset MFA exempt if it was set
		if targetUser.MFAExempt {
			newUpdateTS, err = userRepo.UpdateUserMFAExempt(r.Context(), targetUser.AppUserID, false, newUpdateTS, app.GetClaims().UserID)
			if err != nil {
				app.WriteError(w, http.StatusInternalServerError, "Failed to reset MFA exempt", "MFER", err)
				return
			}
		}

		wrapper := model.DataWrapper{Data: map[string]interface{}{"message": "MFA reset successfully", "updateTs": newUpdateTS}}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
		app.Info("ARM1", "Admin reset MFA completed for user: "+targetUser.UserName)
	}
}
