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
		var csrfToken string
		appServices.Logger.Info("STRT", "Auth request started")

		// Parse request
		var req model.AuthRequestModel
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body", appServices.Logger, "PreAuth")
			return
		}

		appServices.Logger.Debug("PRSE", "Auth request parsed for user: "+req.UserName)

		// Get user from database
		user, err := userRepo.GetUserByUserName(r.Context(), req.UserName)
		if err != nil {
			appServices.Logger.Warn("USLK", "User lookup failed for user: "+req.UserName+", error: "+err.Error())
			writeError(w, http.StatusUnauthorized, "Invalid credentials", appServices.Logger, "PreAuth")
			return
		}

		// Verify password
		if !app_lib.CheckPasswordHash(req.Password, user.PwHash) {
			appServices.Logger.Warn("PSFL", "Password verification failed for user: "+req.UserName)
			writeError(w, http.StatusUnauthorized, "Invalid credentials", appServices.Logger, "PreAuth")
			return
		}

		// Check if MFA is required
		if !user.MFAExempt {
			// Generate temp token for MFA
			tempTokenBytes := make([]byte, 32)
			if _, err := rand.Read(tempTokenBytes); err != nil {
				appServices.Logger.Error("TKGN", "Failed to generate temp token: "+err.Error())
				writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger, "PreAuth")
				return
			}
			tempToken := hex.EncodeToString(tempTokenBytes)

			// Store temp token in key-value store with expiration
			tempTokenKey := "TEMP_AUTH_" + tempToken
			if err := appServices.KV.Set(tempTokenKey, req.UserName, "10m"); err != nil {
				appServices.Logger.Error("TKST", "Failed to store temp token: "+err.Error())
				writeError(w, http.StatusBadRequest, "Invalid request body", appServices.Logger, "PreAuth")
				return
			}

			response := model.AuthResponseModel{
				AppUserID:              user.AppUserID,
				UserName:               user.UserName,
				FirstName:              user.FirstName,
				LastName:               user.LastName,
				SecurityContext:        user.SecurityContext,
				UserGrantShortCodeList: make(map[int][]string), // Empty for MFA flow
				TempToken:              tempToken,
			}

			wrapper := model.AuthResponseWrapper{User: response}
			writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
			appServices.Logger.Info("TKIS", "Auth temp token issued for user: "+req.UserName)
			return
		}

		// Issue full JWT
		jwt, sessionID, err := createJWTForUser(r.Context(), user, appServices, userGroupRepo)
		if err != nil {
			appServices.Logger.Error("JWFL", "Failed to create JWT: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger, "PreAuth")
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

		// CSRF token is last 3 characters of session ID
		csrfToken = sessionID[len(sessionID)-3:]

		// Get user grants for response
		grantList, err := userRepo.GetUserGrantList(r.Context(), user.UserGroupMembershipList)
		if err != nil {
			appServices.Logger.Error("GRFL", "Failed to get user grants: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger, "PreAuth")
			return
		}
		// Convert grants map to short codes
		grantShortCodeList := make(map[int][]string)
		for securityContext, grantIDs := range grantList {
			var shortCodeList []string
			for _, grantID := range grantIDs {
				if shortCode, exists := model.GrantIDToShortCodeMap[grantID]; exists {
					shortCodeList = append(shortCodeList, shortCode)
				}
			}
			grantShortCodeList[securityContext] = shortCodeList
		}

		response := model.Auth2ResponseModel{
			AppUserID:              user.AppUserID,
			UserName:               user.UserName,
			FirstName:              user.FirstName,
			LastName:               user.LastName,
			SecurityContext:        user.SecurityContext,
			UserGrantShortCodeList: grantShortCodeList,
		}

		wrapper := model.Auth2ResponseWrapper{User: response, CsrfToken: csrfToken}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
		appServices.Logger.Info("A2SC", "Auth2 successful for user: "+user.UserName)
	}
}

// Auth2Handler handles MFA authentication (temp token + TOTP)
func Auth2Handler(appServices *app_lib.AppServices, userRepo *repository.UserRepository, userGroupRepo *repository.UserGroupRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var csrfToken string
		appServices.Logger.Info("A2ST", "Auth2 request started")

		// Parse request
		var req model.Auth2RequestModel
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body", appServices.Logger, "PreAuth")
			return
		}

		appServices.Logger.Debug("A2PS", "Auth2 request parsed for token: "+req.TempToken[:8]+"...")

		// Get username from temp token
		tempTokenKey := "TEMP_AUTH_" + req.TempToken
		username, err := appServices.KV.Get(tempTokenKey)
		if err != nil {
			appServices.Logger.Warn("TKLK", "Temp token lookup failed for token: "+req.TempToken[:8]+"..., error: "+err.Error())
			writeError(w, http.StatusUnauthorized, "Invalid or expired temp token", appServices.Logger, "PreAuth")
			return
		}

		// Delete temp token
		appServices.KV.Set(tempTokenKey, "", "0s") // Expire immediately

		// Get user
		user, err := userRepo.GetUserByUserName(r.Context(), username)
		if err != nil {
			appServices.Logger.Error("USLK", "User lookup failed during auth2 for user: "+username+", error: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger, "PreAuth")
			return
		}

		// TODO: Verify TOTP code here
		// For now, accept any TOTP (this should be implemented with a proper TOTP library)
		appServices.Logger.Warn("TOTP", "TOTP verification not implemented - accepting any code")

		// Issue full JWT
		jwt, sessionID, err := createJWTForUser(r.Context(), user, appServices, userGroupRepo)
		if err != nil {
			appServices.Logger.Error("JWFL", "Failed to create JWT: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger, "PreAuth")
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

		// CSRF token is last 3 characters of session ID
		csrfToken = sessionID[len(sessionID)-3:]

		// Get user grants for response
		grantList, err := userRepo.GetUserGrantList(r.Context(), user.UserGroupMembershipList)
		if err != nil {
			appServices.Logger.Error("GRFL", "Failed to get user grants: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger, "PreAuth")
			return
		}

		// Convert grants map to short codes
		grantShortCodeList := make(map[int][]string)
		for securityContext, grantIDs := range grantList {
			var shortCodeList []string
			for _, grantID := range grantIDs {
				if shortCode, exists := model.GrantIDToShortCodeMap[grantID]; exists {
					shortCodeList = append(shortCodeList, shortCode)
				}
			}
			grantShortCodeList[securityContext] = shortCodeList
		}

		response := model.Auth2ResponseModel{
			AppUserID:              user.AppUserID,
			UserName:               user.UserName,
			FirstName:              user.FirstName,
			LastName:               user.LastName,
			SecurityContext:        user.SecurityContext,
			UserGrantShortCodeList: grantShortCodeList,
		}

		wrapper := model.Auth2ResponseWrapper{User: response, CsrfToken: csrfToken}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
		appServices.Logger.Info("A2SC", "Auth2 successful for user: "+user.UserName)
	}
}

// RefreshHandler handles session refresh
func RefreshHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository, userGroupRepo *repository.UserGroupRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appServices.Logger.Info("RFST", "Refresh request started")

		// Security checks
		sec := app_lib.NewSecurityHelper(w, r, appServices.KV, appServices.Config, appServices.Logger)
		sec.CheckJWT()
		sec.CheckRateLimitAndUserBlock()
		sec.SkipCSRF()          // CSRF not required for read-only refresh
		sec.SkipAuthorization() // Authorization not required for token refresh

		if !sec.CheckAllChecksComplete() {
			return
		}

		jwtClaims := sec.GetClaims()

		// Get user
		user, err := userRepo.GetUserByUserName(r.Context(), jwtClaims.Subject)
		if err != nil {
			appServices.Logger.Error("USLK", "User lookup failed during refresh for user: "+jwtClaims.Subject+", error: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger, jwtClaims.Subject)
			return
		}

		// Issue updated JWT
		jwt, sessionID, err := createJWTForUser(r.Context(), user, appServices, userGroupRepo)
		if err != nil {
			appServices.Logger.Error("JWFL", "Failed to create updated JWT: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger, jwtClaims.Subject)
			return
		}

		// Set updated JWT cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    jwt,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   3600, // 1 hour
		})

		// Get user grants for response
		grantList, err := userRepo.GetUserGrantList(r.Context(), user.UserGroupMembershipList)
		if err != nil {
			appServices.Logger.Error("GRFL", "Failed to get user grants: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Internal server error", appServices.Logger, jwtClaims.Subject)
			return
		}

		// Convert grants map to short codes
		grantShortCodeList := make(map[int][]string)
		for securityContext, grantIDs := range grantList {
			var shortCodeList []string
			for _, grantID := range grantIDs {
				if shortCode, exists := model.GrantIDToShortCodeMap[grantID]; exists {
					shortCodeList = append(shortCodeList, shortCode)
				}
			}
			grantShortCodeList[securityContext] = shortCodeList
		}

		response := model.Auth2ResponseModel{
			AppUserID:              user.AppUserID,
			UserName:               user.UserName,
			FirstName:              user.FirstName,
			LastName:               user.LastName,
			SecurityContext:        user.SecurityContext,
			UserGrantShortCodeList: grantShortCodeList,
		}

		csrfToken := sessionID[len(sessionID)-3:]
		wrapper := model.Auth2ResponseWrapper{User: response, CsrfToken: csrfToken}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
		appServices.Logger.Info("RFSC", "Refresh successful for user: "+user.UserName)
	}
}

// LogoffHandler handles user logoff
func LogoffHandler(appServices *app_lib.AppServices) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appServices.Logger.Info("LOST", "Logoff request started")

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
		appServices.Logger.Info("LODN", "Logoff completed")
	}
}

// createJWTForUser creates a JWT token for the authenticated user
func createJWTForUser(ctx context.Context, user *model.AppUserModel, appServices *app_lib.AppServices, userGroupRepo *repository.UserGroupRepository) (string, string, error) {
	// Get user grants
	grants, err := userGroupRepo.GetUserGroupListByIDList(ctx, user.UserGroupMembershipList)
	if err != nil {
		return "", "", err
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
		return "", "", err
	}
	sessionID := hex.EncodeToString(sessionIDBytes)

	// Create JWT
	jwt, err := app_lib.GenerateJWT(
		user.UserName,
		user.AppUserID,
		sessionID,
		user.SecurityContext,
		grantsMap,
		appServices.Config.JWTSecret,
		appServices.Config.JWTSessionExpireHours,
	)
	if err != nil {
		return "", "", err
	}
	return jwt, sessionID, nil
}
