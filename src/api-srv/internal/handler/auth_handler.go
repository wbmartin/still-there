package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
)

// AuthHandler handles initial authentication (username/password)
func AuthHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository, appUserGroupRepo *repository.AppUserGroupRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var csrfToken string
		appServices.Logger.Info("STRT", "Auth request started")

		// Create app helper for error handling
		app := app_lib.NewAppHelper(w, r, appServices)

		// Parse request
		var req model.AuthRequestModel
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			app.WriteError(w, http.StatusBadRequest, "Invalid request body", "IRB0", err)
			return
		}

		appServices.Logger.Debug("PRSE",
			"Auth request parsed for user: "+req.UserName)

		// Read user from database
		user, err := userRepo.RetrieveUserByUserName(r.Context(), req.UserName)
		if err != nil {
			appServices.Logger.Warn("USLK",
				"User lookup failed for user: "+req.UserName+", error: "+err.Error())
			app.WriteError(w, http.StatusUnauthorized, "Invalid credentials", "ICRD", err)
			return
		}

		// Verify password
		if !app_lib.CheckPasswordHash(req.Password, user.PwHash) {
			appServices.Logger.Warn("PSFL",
				"Password verification failed for user: "+req.UserName)
			app.WriteError(w, http.StatusUnauthorized, "Invalid credentials", "ICRD", nil)
			return
		}

		// Check if MFA is required
		if !user.MFAExempt {
			// Generate temp token for MFA
			tempTokenBytes := make([]byte, 32)
			if _, err := rand.Read(tempTokenBytes); err != nil {
				appServices.Logger.Error("TKGN",
					"Failed to generate temp token: "+err.Error())
				app.WriteError(w, http.StatusInternalServerError, "Internal server error", "ISE0", err)
				return
			}
			tempToken := hex.EncodeToString(tempTokenBytes)

			// Store temp token in key-value store with expiration
			tempTokenKey := "TEMP_AUTH_" + tempToken
			if err := appServices.KV.Set(tempTokenKey, req.UserName, "10m"); err != nil {
				appServices.Logger.Error("TKST", "Failed to store temp token: "+err.Error())
				app.WriteError(w, http.StatusBadRequest, "Invalid request body", "IRB0", err)
				return
			}

			response := model.AuthResponseModel{
				AppUserID:                    user.AppUserID,
				UserName:                     user.UserName,
				FirstName:                    user.FirstName,
				LastName:                     user.LastName,
				SecurityContext:              user.SecurityContext,
				GrantList:                    []int{}, // Empty for MFA flow
				AvailableSecurityContextList: []int{}, // Empty for MFA flow
				TempToken:                    tempToken,
			}

			wrapper := model.AuthResponseWrapper{User: response}
			writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
			appServices.Logger.Info("TKIS",
				"Auth temp token issued for user: "+req.UserName)
			return
		}

		// Get all grants for user's groups
		grantsMap, err := userRepo.GetGrantsForGroups(r.Context(), user.AppUserGroupMembershipList)
		if err != nil {
			appServices.Logger.Error("GRFL", "Failed to get grants for user: "+err.Error())
			app.WriteError(w, http.StatusInternalServerError, "Internal server error", "ISE0", err)
			return
		}

		// Extract available security contexts from grant map keys
		availableContextList := make([]int, 0, len(grantsMap))
		for ctx := range grantsMap {
			availableContextList = append(availableContextList, ctx)
		}

		// Issue full JWT
		jwt, sessionID, err := createJWTForUser(user, grantsMap[user.SecurityContext], appServices)
		if err != nil {
			appServices.Logger.Error("JWFL", "Failed to create JWT: "+err.Error())
			app.WriteError(w, http.StatusInternalServerError, "Internal server error", "ISE0", err)
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

		response := model.Auth2ResponseModel{
			AppUserID:                    user.AppUserID,
			UserName:                     user.UserName,
			FirstName:                    user.FirstName,
			LastName:                     user.LastName,
			SecurityContext:              user.SecurityContext,
			GrantList:                    grantsMap[user.SecurityContext],
			AvailableSecurityContextList: availableContextList,
		}

		wrapper := model.Auth2ResponseWrapper{User: response, CsrfToken: csrfToken}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
		appServices.Logger.Info("AUTH", "Authentication successful for user: "+user.UserName)
	}
}

// RefreshHandler handles session refresh
func RefreshHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository, appUserGroupRepo *repository.AppUserGroupRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appServices.Logger.Info("RFST", "Refresh request started")

		// Security checks
		app := app_lib.NewAppHelper(w, r, appServices)
		app.CheckJWT()
		app.CheckRateLimitAndUserBlock()
		app.SkipCSRF()          // CSRF not required for read-only refresh
		app.SkipAuthorization() // Authorization not required for token refresh

		if !app.CheckAllChecksComplete() {
			return
		}

		jwtClaims := app.GetClaims()

		// Read user
		user, err := userRepo.RetrieveUserByUserName(r.Context(), jwtClaims.Subject)
		if err != nil {
			appServices.Logger.Error("USLK", "User lookup failed during refresh for user: "+jwtClaims.Subject+", error: "+err.Error())
			app.WriteError(w, http.StatusInternalServerError, "Internal server error", "ISE0", err)
			return
		}

		// Get all grants for user's groups
		grantsMap, err := userRepo.GetGrantsForGroups(r.Context(), user.AppUserGroupMembershipList)
		if err != nil {
			appServices.Logger.Error("GRFL", "Failed to get grants for refresh: "+err.Error())
			app.WriteError(w, http.StatusInternalServerError, "Internal server error", "ISE0", err)
			return
		}

		// Extract available security contexts from grant map keys
		availableContextList := make([]int, 0, len(grantsMap))
		for ctx := range grantsMap {
			availableContextList = append(availableContextList, ctx)
		}
		user.SecurityContext = jwtClaims.SecurityContext

		// Issue updated JWT
		jwt, sessionID, err := createJWTForUser(user, grantsMap[user.SecurityContext], appServices)
		if err != nil {
			appServices.Logger.Error("JWFL", "Failed to create JWT: "+err.Error())
			app.WriteError(w, http.StatusInternalServerError, "Internal server error", "ISE0", err)
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

		response := model.Auth2ResponseModel{
			AppUserID:                    user.AppUserID,
			UserName:                     user.UserName,
			FirstName:                    user.FirstName,
			LastName:                     user.LastName,
			SecurityContext:              user.SecurityContext,
			GrantList:                    grantsMap[user.SecurityContext],
			AvailableSecurityContextList: availableContextList,
		}

		csrfToken := sessionID[len(sessionID)-3:]
		wrapper := model.Auth2ResponseWrapper{User: response, CsrfToken: csrfToken}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
		appServices.Logger.Info("RFSC", "Refresh successful for user: "+user.UserName)
	}
}

// ChangeSecurityContextHandler handles changing the user's security context
func ChangeSecurityContextHandler(appServices *app_lib.AppServices, userRepo *repository.UserRepository, appUserGroupRepo *repository.AppUserGroupRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appServices.Logger.Info("CSCS", "Change security context request started")

		// Create app helper for error handling
		app := app_lib.NewAppHelper(w, r, appServices)

		// Parse request
		var req model.ChangeSecurityContextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			app.WriteError(w, http.StatusBadRequest, "Invalid request body", "IRB0", err)
			return
		}

		// Security checks - JWT validation only
		app.CheckJWT()
		app.CheckRateLimitAndUserBlock()
		app.SkipCSRF()
		app.SkipAuthorization() // Authorization checked below

		if !app.CheckAllChecksComplete() {
			return
		}

		jwtClaims := app.GetClaims()
		userName := jwtClaims.Subject

		// Read current user
		dbUser, err := userRepo.RetrieveUserByUserName(r.Context(), userName)
		if err != nil {
			appServices.Logger.Error("USLK", "User lookup failed during context change for user: "+userName+", error: "+err.Error())
			app.WriteError(w, http.StatusInternalServerError, "Internal server error", "ISE0", err)
			return
		}

		// Get all grants for user's groups
		grantsMap, err := userRepo.GetGrantsForGroups(r.Context(), dbUser.AppUserGroupMembershipList)
		if err != nil {
			appServices.Logger.Error("GRFL", "Failed to get grants for context change: "+err.Error())
			app.WriteError(w, http.StatusInternalServerError, "Internal server error", "ISE0", err)
			return
		}

		// Extract available security contexts from grant map keys
		availableContextList := make([]int, 0, len(grantsMap))
		for ctx := range grantsMap {
			availableContextList = append(availableContextList, ctx)
		}

		// Check if requested context is available
		if _, exists := grantsMap[req.SecurityContext]; !exists {
			appServices.Logger.Warn("CSCF", "User "+userName+" attempted to switch to unauthorized context: "+strconv.Itoa(req.SecurityContext))
			app.WriteError(w, http.StatusForbidden, "Security context not available", "SCNA", nil)
			return
		}

		// Create new user object with updated security context
		updatedUser := *dbUser
		updatedUser.SecurityContext = req.SecurityContext

		// Issue new JWT with updated context
		jwt, sessionID, err := createJWTForUser(&updatedUser, grantsMap[req.SecurityContext], appServices)
		if err != nil {
			appServices.Logger.Error("JWFL", "Failed to create JWT for context change: "+err.Error())
			app.WriteError(w, http.StatusInternalServerError, "Internal server error", "ISE0", err)
			return
		}

		// Set new JWT cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    jwt,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   3600, // 1 hour
		})

		// CSRF token is last 3 characters of session ID
		csrfToken := sessionID[len(sessionID)-3:]

		response := model.Auth2ResponseModel{
			AppUserID:                    dbUser.AppUserID,
			UserName:                     dbUser.UserName,
			FirstName:                    dbUser.FirstName,
			LastName:                     dbUser.LastName,
			SecurityContext:              req.SecurityContext,
			GrantList:                    grantsMap[req.SecurityContext],
			AvailableSecurityContextList: availableContextList,
		}

		wrapper := model.Auth2ResponseWrapper{User: response, CsrfToken: csrfToken}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
		appServices.Logger.Info("CSCN", "Security context changed for user: "+userName+" to context: "+strconv.Itoa(req.SecurityContext))
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

		w.WriteHeader(http.StatusOK)
		appServices.Logger.Info("LOFN", "Logoff successful")
	}
}

// createJWTForUser creates a JWT token for the authenticated user
func createJWTForUser(user *model.AppUserModel, grantList []int, appServices *app_lib.AppServices) (string, string, error) {
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
		grantList,
		appServices.Config.JWTSecret,
		appServices.Config.JWTSessionExpireHours,
	)
	if err != nil {
		return "", "", err
	}
	return jwt, sessionID, nil
}
