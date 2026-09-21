package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
)

// maxAuthBodyBytes caps the pre-auth body reads in AuthHandler (a username
// and password) and ChangeSecurityContextHandler (one integer), both of which
// decode the body before any rate-limit or JWT check runs.
const maxAuthBodyBytes = 4 << 10 // 4 KiB

// getClientIP extracts the client IP address from the request, checking X-Forwarded-For header first
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the list
		ips := strings.Split(xff, ",")
		ip := strings.TrimSpace(ips[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // fallback if parsing fails
	}
	return ip
}

// isAccountLocked checks if the IP or username is currently locked out
func isAccountLocked(
	kv app_lib.ThisAppKVInterface,
	ip, username string,
) (bool, error) {
	ipLockKey := "loginlock-ip-" + ip
	userLockKey := "loginlock-user-" + username

	// Batch check both lockout keys
	commands := []string{
		"GET " + ipLockKey,
		"GET " + userLockKey,
	}

	results, err := kv.Batch(commands)
	if err != nil {
		// fmt.Print("A %s B %s", starttime, time.Now().UnixNano())
		return false, err
	}

	// Check if either is locked (value "1")
	ipLocked := len(results) > 0 && results[0] == "1"
	userLocked := len(results) > 1 && results[1] == "1"

	return ipLocked || userLocked, nil
}

// incrementFailedAttempt increments the failed attempt counters and sets lockout if threshold exceeded
func incrementFailedAttempt(
	kv app_lib.ThisAppKVInterface,
	config *app_lib.Config,
	ip, username string,
) error {
	ipCounterKey := "loginfail-ip-" + ip
	userCounterKey := "loginfail-user-" + username

	// Batch increment both counters with TTL
	commands := []string{
		fmt.Sprintf(
			"INCR %s %s",
			ipCounterKey,
			config.FailedLoginAttemptWindowDuration,
		),
		fmt.Sprintf(
			"INCR %s %s",
			userCounterKey,
			config.FailedLoginAttemptWindowDuration,
		),
	}

	results, err := kv.Batch(commands)
	if err != nil {
		return err
	}

	// Parse results to int
	ipCount, err := strconv.Atoi(results[0])
	if err != nil {
		return err
	}
	userCount, err := strconv.Atoi(results[1])
	if err != nil {
		return err
	}

	// Check if either counter exceeded threshold
	if ipCount >= config.FailedLoginAttemptThreshold {
		ipLockKey := "loginlock-ip-" + ip
		err = kv.Set(ipLockKey, "1", config.FailedLoginLockoutDuration)
		if err != nil {
			return err
		}
	}

	if userCount >= config.FailedLoginAttemptThreshold {
		userLockKey := "loginlock-user-" + username
		err = kv.Set(userLockKey, "1", config.FailedLoginLockoutDuration)
		if err != nil {
			return err
		}
	}

	return nil
}

// resetFailedAttempts resets the failed attempt counters for user and IP on successful login
func resetFailedAttempts(
	kv app_lib.ThisAppKVInterface,
	ip, username string,
) error {
	userCounterKey := "loginfail-user-" + username
	ipCounterKey := "loginfail-ip-" + ip
	// Reset counters to 0 with short TTL to effectively clear them
	err := kv.Set(userCounterKey, "0", "1s")
	if err != nil {
		return err
	}
	return kv.Set(ipCounterKey, "0", "1s")
}

// AuthHandler handles initial authentication (username/password)
func AuthHandler(
	appServices *app_lib.AppServices,
	userRepo *repository.UserRepository,
	appUserGroupRepo *repository.AppUserGroupRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var csrfToken string
		appServices.Logger.Info(
			"AUTH_REQ_STRT",
			"Auth request started",
			"username",
			"preauth",
		)

		// Create app helper for error handling
		app := app_lib.NewAppHelper(w, r, appServices, "AuthHandler")

		// Parse request. Nothing has checked rate limit or lockout yet, so
		// bound what an anonymous caller can make the decoder allocate.
		r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)
		var req model.AuthRequestModel
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			app.WriteError(
				w,
				http.StatusBadRequest,
				fmt.Sprintf("Invalid request body for auth: %v", err),
				"INV_REQ_BODY",
				err,
			)
			return
		}

		appServices.Logger.Debug(
			"AUTH_REQ_PRSE",
			"Auth request parsed for user: "+req.UserName,
			"username",
			req.UserName,
		)

		// Check for account lockout
		clientIP := getClientIP(r)
		locked, err := isAccountLocked(appServices.KV, clientIP, req.UserName)
		if err != nil {
			// Fail open: if the KV store is unavailable, don't block login. Account lockout is a
			// best-effort brute-force throttle (not authorization), and the write side already
			// fails open (incrementFailedAttempt only logs on error) — so the read side should
			// too, otherwise a KV outage hard-locks every login. Trade-off: while KV is down,
			// failed-attempt lockout is not enforced.
			appServices.Logger.Warn(
				"LOCKOUT_CHECK_FAIL",
				"Account lockout check failed; failing open: "+err.Error(),
				"username",
				req.UserName,
			)
			locked = false
		}
		if locked {
			app.WriteError(
				w,
				http.StatusTooManyRequests,
				"Account is temporarily locked due to too many failed login attempts",
				"ACCOUNT_LOCKED",
				nil,
			)
			return
		}

		// Read user from database
		user, err := userRepo.RetrieveUserByUserName(r.Context(), req.UserName)
		if err != nil {
			appServices.Logger.Warn(
				"USR_LKUP_FAIL",
				"User lookup failed for user: "+req.UserName+", error: "+err.Error(),
				"username",
				req.UserName,
			)
			app.WriteError(
				w,
				http.StatusUnauthorized,
				"Invalid credentials",
				"INVALID_CREDS",
				err,
			)
			return
		}

		// Verify password
		if !app_lib.CheckPasswordHash(req.Password, user.PwHash) {
			appServices.Logger.Warn("PWD_VFY_FAIL",
				"Password verification failed for user: "+req.UserName)
			// Increment failed attempt counters
			err := incrementFailedAttempt(
				appServices.KV,
				appServices.Config,
				clientIP,
				req.UserName,
			)
			if err != nil {
				appServices.Logger.Error(
					"FAILED_ATTEMPT_INCR_FAIL",
					"Failed to increment failed attempts for user: "+req.UserName+", error: "+err.Error(),
					"username",
					req.UserName,
				)
				// Don't fail the request, just log
			}
			app.WriteError(
				w,
				http.StatusUnauthorized,
				"Invalid credentials",
				"INVALID_CREDS",
				nil,
			)
			return
		}

		// Reset failed attempt counters on successful login
		// err = resetFailedAttempts(appServices.KV, clientIP, req.UserName)
		// if err != nil {
		// 	appServices.Logger.Warn("RESET_FAILED_ATTEMPTS_FAIL",
		// 		"Failed to reset failed attempts for user: "+req.UserName+", error: "+err.Error(), "username", req.UserName)
		// 	// Don't fail auth, just log
		// }

		// Check if MFA is required
		if !user.MFAExempt {
			// Generate temp token for MFA
			tempTokenBytes := make([]byte, 32)
			if _, err := rand.Read(tempTokenBytes); err != nil {
				app.WriteError(
					w,
					http.StatusInternalServerError,
					fmt.Sprintf(
						"Failed to generate temp token for MFA: username=%s",
						req.UserName,
					),
					"TMP_TKN_GEN",
					err,
				)
				return
			}
			tempToken := hex.EncodeToString(tempTokenBytes)

			// Store temp token in key-value store with expiration
			tempTokenKey := "TEMP_AUTH_SUCCESS_" + tempToken
			if err := appServices.KV.Set(tempTokenKey, req.UserName, "10min"); err != nil {
				app.WriteError(
					w,
					http.StatusInternalServerError,
					fmt.Sprintf(
						"Failed to set up MFA for user: username=%s",
						req.UserName,
					),
					"TMP_TKN_STORE",
					err,
				)
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
			appServices.Logger.Info("TMP_TKN_ISSUED",
				"Auth temp token issued for user: "+req.UserName)
			return
		}

		// Get all grants for user's groups
		grantsMap, err := userRepo.GetGrantsForGroups(
			r.Context(),
			user.AppUserGroupMembershipList,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to get grants for auth: username=%s",
					user.UserName,
				),
				"GRT_FETCH_FAIL",
				err,
			)
			return
		}

		// Extract available security contexts from grant map keys
		availableContextList := make([]int, 0, len(grantsMap))
		for ctx := range grantsMap {
			availableContextList = append(availableContextList, ctx)
		}

		// Issue full JWT
		jwt, sessionID, err := createJWTForUser(
			user,
			grantsMap[user.SecurityContext],
			appServices,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to generate JWT for auth: username=%s",
					user.UserName,
				),
				"JWT_GEN_FAIL",
				err,
			)
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
			MFARequired: !user.MFAExempt &&
				(user.TOTPSeed == nil || *user.TOTPSeed == ""),
		}

		wrapper := model.Auth2ResponseWrapper{
			User:      response,
			CsrfToken: csrfToken,
		}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
		appServices.Logger.Info(
			"AUTH_SUCCESS",
			"Authentication successful for user: "+user.UserName,
			"username",
			user.UserName,
		)
	}
}

// RefreshHandler handles session refresh
func RefreshHandler(
	appServices *app_lib.AppServices,
	userRepo *repository.UserRepository,
	appUserGroupRepo *repository.AppUserGroupRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security checks
		app := app_lib.NewAppHelper(w, r, appServices, "RefreshHandler")
		app.CheckJWT()
		app.CheckRateLimitAndUserBlock()
		app.SkipCSRF()          // CSRF not required for read-only refresh
		app.SkipAuthorization() // Authorization not required for token refresh

		if !app.CheckAllChecksComplete() {
			return
		}

		appServices.Logger.Info(
			"REFRESH_START",
			"Refresh request started",
			"username",
			app.GetUsername(),
		)

		jwtClaims := app.GetClaims()

		// Read user
		user, err := userRepo.RetrieveUserByUserName(
			r.Context(),
			jwtClaims.Subject,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to lookup user for refresh: username=%s",
					jwtClaims.Subject,
				),
				"USR_LKUP_FAIL",
				err,
			)
			return
		}

		// Get all grants for user's groups
		grantsMap, err := userRepo.GetGrantsForGroups(
			r.Context(),
			user.AppUserGroupMembershipList,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to get grants for refresh: username=%s",
					jwtClaims.Subject,
				),
				"GRT_FETCH_FAIL",
				err,
			)
			return
		}

		// Extract available security contexts from grant map keys
		availableContextList := make([]int, 0, len(grantsMap))
		for ctx := range grantsMap {
			availableContextList = append(availableContextList, ctx)
		}
		user.SecurityContext = jwtClaims.SecurityContext

		// Issue updated JWT
		jwt, sessionID, err := createJWTForUser(
			user,
			grantsMap[user.SecurityContext],
			appServices,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to generate JWT for refresh: username=%s",
					jwtClaims.Subject,
				),
				"JWT_GEN_FAIL",
				err,
			)
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
			MFARequired: !user.MFAExempt &&
				(user.TOTPSeed == nil || *user.TOTPSeed == ""),
		}

		csrfToken := sessionID[len(sessionID)-3:]
		wrapper := model.Auth2ResponseWrapper{
			User:      response,
			CsrfToken: csrfToken,
		}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
		appServices.Logger.Info(
			"REFRESH_SUCCESS",
			"Refresh successful for user: "+user.UserName,
			"username",
			user.UserName,
		)
	}
}

// ChangeSecurityContextHandler handles changing the user's security context
func ChangeSecurityContextHandler(
	appServices *app_lib.AppServices,
	userRepo *repository.UserRepository,
	appUserGroupRepo *repository.AppUserGroupRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create app helper for error handling
		app := app_lib.NewAppHelper(w, r, appServices, "ChangeSecurityContextHandler")

		// Parse request first to get security context. This precedes the JWT
		// and rate-limit checks, so bound it.
		r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)
		var req model.ChangeSecurityContextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			app.WriteError(
				w,
				http.StatusBadRequest,
				fmt.Sprintf("Invalid request body for context change: %v", err),
				"INV_REQ_BODY",
				err,
			)
			return
		}

		// Security checks
		app.CheckJWT()
		app.CheckRateLimitAndUserBlock()
		app.SkipCSRF()
		app.SkipAuthorization() // Authorization checked below

		if !app.CheckAllChecksComplete() {
			return
		}

		appServices.Logger.Info(
			"CHANGE_CTX_START",
			"Change security context request started",
			"username",
			app.GetUsername(),
		)

		jwtClaims := app.GetClaims()
		userName := jwtClaims.Subject

		// Read current user
		dbUser, err := userRepo.RetrieveUserByUserName(r.Context(), userName)
		if err != nil {
			appServices.Logger.Error(
				"USR_LKUP_FAIL",
				"User lookup failed during context change for user: "+userName+", error: "+err.Error(),
				"username",
				userName,
			)
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Internal server error",
				"INTERNAL_SERVER_ERROR",
				err,
			)
			return
		}

		// Get all grants for user's groups
		grantsMap, err := userRepo.GetGrantsForGroups(
			r.Context(),
			dbUser.AppUserGroupMembershipList,
		)
		if err != nil {
			appServices.Logger.Error(
				"GRT_FETCH_FAIL",
				"Failed to get grants for context change: "+err.Error(),
				"username",
				app.GetUsername(),
			)
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to get grants for context change: username=%s",
					userName,
				),
				"INTERNAL_SERVER_ERROR",
				err,
			)
			return
		}

		// Extract available security contexts from grant map keys
		availableContextList := make([]int, 0, len(grantsMap))
		for ctx := range grantsMap {
			availableContextList = append(availableContextList, ctx)
		}

		// Check if requested context is available
		if _, exists := grantsMap[req.SecurityContext]; !exists {
			appServices.Logger.Warn(
				"CHG_CTX_FAIL",
				"User "+userName+" attempted to switch to unauthorized context: "+strconv.Itoa(
					req.SecurityContext,
				),
				"username",
				userName,
			)
			app.WriteError(
				w,
				http.StatusForbidden,
				"Security context not available",
				"SECURITY_CONTEXT_NOT_AVAILABLE",
				nil,
			)
			return
		}

		// Create new user object with updated security context
		updatedUser := *dbUser
		updatedUser.SecurityContext = req.SecurityContext

		// Issue new JWT with updated context
		jwt, sessionID, err := createJWTForUser(
			&updatedUser,
			grantsMap[req.SecurityContext],
			appServices,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to generate JWT for context change: username=%s, context=%d",
					userName,
					req.SecurityContext,
				),
				"JWT_GEN_FAIL",
				err,
			)
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
			MFARequired: !dbUser.MFAExempt &&
				(dbUser.TOTPSeed == nil || *dbUser.TOTPSeed == ""),
		}

		wrapper := model.Auth2ResponseWrapper{
			User:      response,
			CsrfToken: csrfToken,
		}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
		appServices.Logger.Info(
			"CHANGE_CTX_SUCCESS",
			"Security context changed for user: "+userName+" to context: "+strconv.Itoa(
				req.SecurityContext,
			),
			"username",
			userName,
		)
	}
}

// LogoffHandler handles user logoff
func LogoffHandler(appServices *app_lib.AppServices) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Use AppHelper to extract username from JWT (skip other validations)
		app := app_lib.NewAppHelper(w, r, appServices, "LogoffHandler")
		app.CheckJWT()                   // Extract username from JWT
		app.CheckRateLimitAndUserBlock() // No rate limiting for logoff
		app.SkipCSRF()                   // No CSRF needed for logoff
		app.SkipAuthorization()          // No authorization needed for logoff

		if !app.CheckAllChecksComplete() {
			return
		}

		app.Logger().Info(
			"LOGOFF_START",
			"Logoff request started",
			"username",
			app.GetUsername(),
		)

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
		app.Logger().Info(
			"LOGOFF_SUCCESS",
			"Logoff successful",
			"username",
			app.GetUsername(),
		)
	}
}

// ChangeOwnPasswordHandler handles changing the current user's password
func ChangeOwnPasswordHandler(
	appServices *app_lib.AppServices,
	userRepo *repository.UserRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security checks
		app := app_lib.NewAppHelper(w, r, appServices, "ChangeOwnPasswordHandler")
		app.CheckJWT()
		app.CheckRateLimitAndUserBlock()
		app.CheckCSRF()
		app.CheckAuthorization(model.GrantChangeOwnPassword)

		if !app.CheckAllChecksComplete() {
			return
		}

		appServices.Logger.Info(
			"CHANGE_PW_START",
			"Change own password request started",
			"username",
			app.GetUsername(),
		)

		jwtClaims := app.GetClaims()

		// Parse request body
		var req struct {
			OldPassword string `json:"oldPassword"`
			NewPassword string `json:"newPassword"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			app.WriteError(
				w,
				http.StatusBadRequest,
				fmt.Sprintf(
					"Invalid request body for password change: %v",
					err,
				),
				"INV_REQ_BODY",
				err,
			)
			return
		}

		// Validate required fields
		if req.OldPassword == "" || req.NewPassword == "" {
			app.WriteError(
				w,
				http.StatusBadRequest,
				"oldPassword and newPassword are required",
				"BAD_REQUEST",
				nil,
			)
			return
		}

		// Basic password validation (minimum length)
		if len(req.NewPassword) < 8 {
			app.WriteError(
				w,
				http.StatusBadRequest,
				fmt.Sprintf(
					"New password must be at least 8 characters long: len=%d",
					len(req.NewPassword),
				),
				"BAD_REQUEST",
				nil,
			)
			return
		}

		// Retrieve current user
		user, err := userRepo.RetrieveUserByUserName(
			r.Context(),
			jwtClaims.Subject,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to lookup user for password change: username=%s",
					jwtClaims.Subject,
				),
				"USR_LKUP_FAIL",
				err,
			)
			return
		}

		// Verify old password
		if !app_lib.CheckPasswordHash(req.OldPassword, user.PwHash) {
			appServices.Logger.Warn(
				"PWD_VFY_FAIL",
				"Old password verification failed for user: "+jwtClaims.Subject,
			)
			app.WriteError(
				w,
				http.StatusUnauthorized,
				"Old password is incorrect",
				"INVALID_CREDS",
				nil,
			)
			return
		}

		// Hash new password
		hashedPassword, err := app_lib.HashPassword(req.NewPassword)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to process password for user: username=%s",
					jwtClaims.Subject,
				),
				"PWD_HASH_FAIL",
				err,
			)
			return
		}

		// Update password
		now := CurrentTimestamp()
		err = userRepo.UpdateUserPassword(
			r.Context(),
			jwtClaims.UserID,
			hashedPassword,
			now,
			jwtClaims.UserID,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to update password for user: username=%s, user_id=%d",
					jwtClaims.Subject,
					jwtClaims.UserID,
				),
				"PWD_UPD_FAIL",
				err,
			)
			return
		}

		appServices.Logger.Info(
			"CHANGE_PW_SUCCESS",
			"Password changed successfully for user: "+jwtClaims.Subject,
			"username",
			jwtClaims.Subject,
		)

		wrapper := model.DataWrapper{
			Data: map[string]string{"message": "Password changed successfully"},
		}
		writeJSON(w, http.StatusOK, wrapper, appServices.Logger)
	}
}

// createJWTForUser creates a JWT token for the authenticated user
func createJWTForUser(
	user *model.AppUserModel,
	grantList []int,
	appServices *app_lib.AppServices,
) (string, string, error) {
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
