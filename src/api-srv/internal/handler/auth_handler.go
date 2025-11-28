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
func AuthHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository, userGroupRepo *repository.UserGroupRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appServices.Logger.Info("auth", "START", "Auth request started")

		// Parse request
		var req model.AuthRequestModel
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body", appServices.Logger)
			return
		}

		appServices.Logger.Debug("auth", "PARSE", "Auth request parsed for user: "+req.Username)

		// Get user from database
		user, err := userRepo.GetUserByUsername(r.Context(), req.Username)
		if err != nil {
			appServices.Logger.Warn("auth", "USERLOOKUP", "User lookup failed for user: "+req.Username+", error: "+err.Error())
			writeError(w, http.StatusUnauthorized, "Invalid credentials", appServices.Logger)
			return
		}

		// Verify password
		if !app_lib.CheckPasswordHash(req.Password, user.PwHash) {
			appServices.Logger.Warn("auth", "PASSFAIL", "Password verification failed for user: "+req.Username)
			writeError(w, http.StatusUnauthorized, "Invalid credentials", appServices.Logger)
			return
		}

		// Check if MFA is required
		if !user.MFAExempt {
			// Generate temp token for MFA
			tempTokenBytes := make([]byte, 32)
			if _, err := rand.Read(tempTokenBytes); err != nil {
				appServices.Logger.Error("auth", "TOKGEN", "Failed to generate temp token: "+err.Error())
				writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger)
				return
			}
			tempToken := hex.EncodeToString(tempTokenBytes)

			// Store temp token in key-value store with expiration
			tempTokenKey := "TEMP_AUTH_" + tempToken
			if err := appServices.KV.Set(tempTokenKey, req.Username, "10m"); err != nil {
				appServices.Logger.Error("auth", "TOKSTORE", "Failed to store temp token: "+err.Error())
				writeError(w, http.StatusBadRequest, "Invalid request body", appServices.Logger)
				return
			}

			response := model.AuthResponseModel{
				AppUserID:              user.AppUserID,
				Username:               user.Username,
				FirstName:              user.FirstName,
				LastName:               user.LastName,
				SecurityContext:        user.SecurityContext,
				UserGrantShortCodeList: make(map[int][]string), // Empty for MFA flow
				TempToken:              tempToken,
			}

			writeJSON(w, http.StatusOK, response, appServices.Logger)
			appServices.Logger.Info("auth", "TOKISSUE", "Auth temp token issued for user: "+req.Username)
			return
		}

		// MFA not required, issue full JWT
		jwt, err := createJWTForUser(r.Context(), user, appServices, userGroupRepo)
		if err != nil {
			appServices.Logger.Error("auth", "JWTFAIL", "Failed to create JWT: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger)
			return
		}

		// Set JWT cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    jwt,
			Path:     "/",
			HttpOnly: true,
			Secure:   true, // In production, set based on appServices.Config
			SameSite: http.SameSiteStrictMode,
		})

		// Get user grants for response
		grants, err := userRepo.GetUserGrants(r.Context(), user.UserGroupMembershipList)
		if err != nil {
			appServices.Logger.Error("auth", "GRANTFAIL", "Failed to get user grants: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger)
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

		response := model.AuthResponseModel{
			AppUserID:              user.AppUserID,
			Username:               user.Username,
			FirstName:              user.FirstName,
			LastName:               user.LastName,
			SecurityContext:        user.SecurityContext,
			UserGrantShortCodeList: grantShortCodes,
		}

		writeJSON(w, http.StatusOK, response, appServices.Logger)
		appServices.Logger.Info("auth", "SUCCESS", "Auth successful for user: "+req.Username)
	}
}

// Auth2Handler handles MFA authentication (temp token + TOTP)
func Auth2Handler(appServices *app_lib.AppServices, userRepo *repository.UserRepository, userGroupRepo *repository.UserGroupRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appServices.Logger.Info("auth", "AUTH2START", "Auth2 request started")

		// Parse request
		var req model.Auth2RequestModel
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body", appServices.Logger)
			return
		}

		appServices.Logger.Debug("auth", "AUTH2PARSE", "Auth2 request parsed for token: "+req.TempToken[:8]+"...")

		// Get username from temp token
		tempTokenKey := "TEMP_AUTH_" + req.TempToken
		username, err := appServices.KV.Get(tempTokenKey)
		if err != nil {
			appServices.Logger.Warn("auth", "TOKLOOKUP", "Temp token lookup failed for token: "+req.TempToken[:8]+"..., error: "+err.Error())
			writeError(w, http.StatusUnauthorized, "Invalid or expired temp token", appServices.Logger)
			return
		}

		// Delete temp token
		appServices.KV.Set(tempTokenKey, "", "0s") // Expire immediately

		// Get user
		user, err := userRepo.GetUserByUsername(r.Context(), username)
		if err != nil {
			appServices.Logger.Error("auth", "USERLOOKUP", "User lookup failed during auth2 for user: "+username+", error: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger)
			return
		}

		// TODO: Verify TOTP code here
		// For now, accept any TOTP (this should be implemented with a proper TOTP library)
		appServices.Logger.Warn("auth", "TOTP", "TOTP verification not implemented - accepting any code")

		// Issue full JWT
		jwt, err := createJWTForUser(r.Context(), user, appServices, userGroupRepo)
		if err != nil {
			appServices.Logger.Error("auth", "JWTFAIL", "Failed to create JWT: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger)
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
			appServices.Logger.Error("auth", "GRANTFAIL", "Failed to get user grants: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger)
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

		response := model.Auth2ResponseModel{
			AppUserID:              user.AppUserID,
			Username:               user.Username,
			FirstName:              user.FirstName,
			LastName:               user.LastName,
			SecurityContext:        user.SecurityContext,
			UserGrantShortCodeList: grantShortCodes,
		}

		writeJSON(w, http.StatusOK, response, appServices.Logger)
		appServices.Logger.Info("auth", "AUTH2SUCCESS", "Auth2 successful for user: "+username)
	}
}

// LogoffHandler handles user logoff
func LogoffHandler(appServices *app_lib.AppServices) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appServices.Logger.Info("auth", "LOGOFFSTART", "Logoff request started")

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

		writeJSON(w, http.StatusOK, map[string]string{"message": "Logged off successfully"}, appServices.Logger)
		appServices.Logger.Info("auth", "LOGOFFDONE", "Logoff completed")
	}
}

// createJWTForUser creates a JWT token for the authenticated user
func createJWTForUser(ctx context.Context, user *model.AppUserModel, appServices *app_lib.AppServices, userGroupRepo *repository.UserGroupRepository) (string, error) {
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
		appServices.Config.JWTSecret,
		appServices.Config.JWTSessionExpireHours,
	)
}
