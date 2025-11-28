package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
)

// AuthHandler handles initial authentication (username/password)
func AuthHandler(db *app_lib.Database, shrmplKV *app_lib.ShrmplKVClient, config *app_lib.Config, logger app_lib.LoggerInterface, userRepo *repository.UserRepository, userGroupRepo *repository.UserGroupRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Auth request started")

		// Parse request
		var req model.AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body", logger)
			return
		}

		logger.Debug("Auth request parsed for user: %s", req.Username)

		// Get user from database
		user, err := userRepo.GetUserByUsername(r.Context(), req.Username)
		if err != nil {
			logger.Warn("User lookup failed for user: %s, error: %s", req.Username, err.Error())
			writeError(w, http.StatusUnauthorized, "Invalid credentials", logger)
			return
		}

		// Verify password
		if !app_lib.CheckPasswordHash(req.Password, user.PwHash) {
			logger.Warn("Password verification failed for user: %s", req.Username)
			writeError(w, http.StatusUnauthorized, "Invalid credentials", logger)
			return
		}

		// Check if MFA is required
		if !user.MFAExempt {
			// Generate temp token for MFA
			tempTokenBytes := make([]byte, 32)
			if _, err := rand.Read(tempTokenBytes); err != nil {
				logger.Error("Failed to generate temp token: %s", err.Error())
				writeError(w, http.StatusInternalServerError, "Internal server error", logger)
				return
			}
			tempToken := hex.EncodeToString(tempTokenBytes)

			// Store temp token in shrmpl-kv with expiration
			tempTokenKey := "TEMP_AUTH_" + tempToken
			if err := shrmplKV.Set(tempTokenKey, req.Username, "10m"); err != nil {
				logger.Error("Failed to store temp token: %s", err.Error())
				writeError(w, http.StatusInternalServerError, "Internal server error", logger)
				return
			}

			response := model.AuthResponse{
				AppUserID:              user.AppUserID,
				Username:               user.Username,
				FirstName:              user.FirstName,
				LastName:               user.LastName,
				SecurityContext:        user.SecurityContext,
				UserGrantShortCodeList: make(map[int][]string), // Empty for MFA flow
				TempToken:              tempToken,
			}

			writeJSON(w, http.StatusOK, response, logger)
			logger.Info("Auth temp token issued for user: %s", req.Username)
			return
		}

		// MFA not required, issue full JWT
		jwt, err := createJWTForUser(r.Context(), user, userGroupRepo, config, logger)
		if err != nil {
			logger.Error("Failed to create JWT: %s", err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", logger)
			return
		}

		// Set JWT cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    jwt,
			Path:     "/",
			HttpOnly: true,
			Secure:   true, // In production, set based on config
			SameSite: http.SameSiteStrictMode,
		})

		// Get user grants for response
		grants, err := userRepo.GetUserGrants(r.Context(), user.UserGroupMembershipList)
		if err != nil {
			logger.Error("Failed to get user grants: %s", err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", logger)
			return
		}

		// Convert grants map to short codes
		grantShortCodes := make(map[int][]string)
		for securityContext, grantIDs := range grants {
			var shortCodes []string
			for _, grantID := range grantIDs {
				if shortCode, exists := model.GrantIDToShortCodeMap[grantID]; exists {
					shortCodes = append(shortCodes, shortCode)
				}
			}
			grantShortCodes[securityContext] = shortCodes
		}

		response := model.AuthResponse{
			AppUserID:              user.AppUserID,
			Username:               user.Username,
			FirstName:              user.FirstName,
			LastName:               user.LastName,
			SecurityContext:        user.SecurityContext,
			UserGrantShortCodeList: grantShortCodes,
		}

		writeJSON(w, http.StatusOK, response, logger)
		logger.Info("Auth successful for user: %s", req.Username)
	}
}

// Auth2Handler handles MFA authentication (temp token + TOTP)
func Auth2Handler(db *app_lib.Database, shrmplKV *app_lib.ShrmplKVClient, config *app_lib.Config, logger app_lib.LoggerInterface, userRepo *repository.UserRepository, userGroupRepo *repository.UserGroupRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Auth2 request started")

		// Parse request
		var req model.Auth2Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body", logger)
			return
		}

		logger.Debug("Auth2 request parsed for token: %s...", req.TempToken[:8])

		// Get username from temp token
		tempTokenKey := "TEMP_AUTH_" + req.TempToken
		username, err := shrmplKV.Get(tempTokenKey)
		if err != nil {
			logger.Warn("Temp token lookup failed for token: %s..., error: %s", req.TempToken[:8], err.Error())
			writeError(w, http.StatusUnauthorized, "Invalid or expired temp token", logger)
			return
		}

		// Delete temp token
		shrmplKV.Set(tempTokenKey, "", "0s") // Expire immediately

		// Get user
		user, err := userRepo.GetUserByUsername(r.Context(), username)
		if err != nil {
			logger.Error("User lookup failed during auth2 for user: %s, error: %s", username, err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", logger)
			return
		}

		// TODO: Verify TOTP code here
		// For now, accept any TOTP (this should be implemented with a proper TOTP library)
		logger.Warn("TOTP verification not implemented - accepting any code")

		// Issue full JWT
		jwt, err := createJWTForUser(r.Context(), user, userGroupRepo, config, logger)
		if err != nil {
			logger.Error("Failed to create JWT: %s", err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", logger)
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

		// Get user grants for response
		grants, err := userRepo.GetUserGrants(r.Context(), user.UserGroupMembershipList)
		if err != nil {
			logger.Error("Failed to get user grants: %s", err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", logger)
			return
		}

		// Convert grants map to short codes
		grantShortCodes := make(map[int][]string)
		for securityContext, grantIDs := range grants {
			var shortCodes []string
			for _, grantID := range grantIDs {
				if shortCode, exists := model.GrantIDToShortCodeMap[grantID]; exists {
					shortCodes = append(shortCodes, shortCode)
				}
			}
			grantShortCodes[securityContext] = shortCodes
		}

		response := model.Auth2Response{
			AppUserID:              user.AppUserID,
			Username:               user.Username,
			FirstName:              user.FirstName,
			LastName:               user.LastName,
			SecurityContext:        user.SecurityContext,
			UserGrantShortCodeList: grantShortCodes,
		}

		writeJSON(w, http.StatusOK, response, logger)
		logger.Info("Auth2 successful for user: %s", username)
	}
}

// LogoffHandler handles user logoff
func LogoffHandler(logger app_lib.LoggerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Logoff request started")

		// Clear the auth cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   -1, // Expire immediately
		})

		writeJSON(w, http.StatusOK, map[string]string{"message": "Logged off successfully"}, logger)
		logger.Info("Logoff completed")
	}
}

// createJWTForUser creates a JWT token for the authenticated user
func createJWTForUser(ctx context.Context, user *model.AppUser, userGroupRepo *repository.UserGroupRepository, config *app_lib.Config, logger app_lib.LoggerInterface) (string, error) {
	// Get user grants
	grants, err := userGroupRepo.GetUserGroups(ctx, user.UserGroupMembershipList)
	if err != nil {
		return "", err
	}

	// Build grants map by security context
	grantsMap := make(map[int][]int)
	for _, group := range grants {
		for _, grantID := range group.UserGrantList {
			grantsMap[group.SecurityContext] = append(grantsMap[group.SecurityContext], grantID)
		}
	}

	// Generate session ID
	sessionIDBytes := make([]byte, 16)
	if _, err := rand.Read(sessionIDBytes); err != nil {
		return "", err
	}
	sessionID := hex.EncodeToString(sessionIDBytes)

	// Create JWT
	return app_lib.GenerateJWT(
		user.Username,
		user.AppUserID,
		sessionID,
		user.SecurityContext,
		grantsMap,
		config.JWTSecret,
		config.JWTSessionExpireHours,
	)
}
