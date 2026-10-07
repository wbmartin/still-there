package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/cache"
	"github.com/wbmartin/starter-api/internal/config"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
	"github.com/wbmartin/starter-api/internal/security"
)

// maxUserNameLen matches app_user.user_name VARCHAR(50).
const maxUserNameLen = 50

// isAccountLocked checks if the IP or username is currently locked out
func isAccountLocked(
	kv *cache.KV,
	ip, username string,
) (bool, error) {
	ipLocked, err := kv.Get("loginlock-ip-" + ip)
	if err != nil {
		return false, err
	}
	userLocked, err := kv.Get("loginlock-user-" + username)
	if err != nil {
		return false, err
	}
	// A lock is the value "1" with the lockout TTL; missing reads as ""
	return ipLocked == "1" || userLocked == "1", nil
}

// incrementFailedAttempt increments the failed attempt counters and sets lockout if threshold exceeded
func incrementFailedAttempt(
	kv *cache.KV,
	config *config.Config,
	ip, username string,
) error {
	// Both counters live for the attempt window; INCR only honours the TTL on create
	ipCount, err := kv.Incr("loginfail-ip-"+ip, config.FailedLoginAttemptWindowDuration)
	if err != nil {
		return err
	}
	userCount, err := kv.Incr("loginfail-user-"+username, config.FailedLoginAttemptWindowDuration)
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

// resetFailedAttempts deletes the per-username failure counter once a login
// completes. Del, not a short-TTL "0": Incr keeps an existing key's TTL, so
// the next failure would inherit it. The per-IP counter is left to expire on
// its window: clearing it would let anyone holding one valid account reset
// the IP between guesses at other names. Lock flags are untouched; a locked
// account is refused before any credential check.
func resetFailedAttempts(kv *cache.KV, username string) error {
	return kv.Del("loginfail-user-" + username)
}

// resetFailedAttemptsOrLog clears the counter and only logs a failure:
// clearing is a convenience, not a security control.
func resetFailedAttemptsOrLog(appServices *app.Services, username string) {
	if err := resetFailedAttempts(appServices.KV, username); err != nil {
		appServices.Logger.Warn(
			"RESET_FAILED_ATTEMPTS_FAIL",
			"Failed to reset failed attempts for user: "+username+
				", error: "+err.Error(),
			"username",
			username,
		)
	}
}

// AuthHandler handles initial authentication (username/password)
func AuthHandler(
	appServices *app.Services,
	userRepo UserRepositoryInterface,
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
		appHelper := security.NewAppHelper(w, r, appServices, "AuthHandler")
		// Login is the one endpoint with nothing to check before the body:
		// there is no JWT yet, and brute force is answered by the per-IP +
		// per-username lockout below (7 failures / 5 min), which is stricter
		// than the per-minute limiter. Written out so the exception is read
		// here, not inferred.
		appHelper.SkipJWT()
		appHelper.SkipRateLimit()
		appHelper.SkipUserBlock()
		appHelper.SkipCSRF()
		appHelper.SkipAuthorization()
		if !appHelper.CheckAllChecksComplete() {
			return
		}

		// Parse request. Nothing has checked rate limit or lockout yet, so
		// bound what an anonymous caller can make the decoder allocate.
		var req model.AuthRequestModel
		if !decodeJSONBody(appHelper, w, r, maxAuthBodyBytes, &req) {
			return
		}
		// user_name is VARCHAR(50); anything longer cannot be a user and would
		// only overflow the lockout keys in the store. Same answer as a bad
		// password so the check leaks nothing.
		if req.UserName == "" || len(req.UserName) > maxUserNameLen {
			appHelper.WriteError(
				w, http.StatusUnauthorized, "Invalid credentials",
				"INVALID_CREDS", nil,
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
		clientIP := security.ClientIP(
			r, appServices.Config.TrustedProxyCount,
		)
		locked, err := isAccountLocked(appServices.KV, clientIP, req.UserName)
		if err != nil {
			// The store is in-process, so an error is a bug, not an outage:
			// fail closed, as MFAVerifyHandler already does.
			appHelper.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to check account lockout",
				"LOCKOUT_CHECK_FAIL",
				err,
			)
			return
		}
		if locked {
			appHelper.WriteError(
				w,
				http.StatusTooManyRequests,
				"Account is temporarily locked due to too many failed login attempts",
				"ACCOUNT_LOCKED",
				nil,
			)
			return
		}

		// Read user from database. An unknown name is not an error here: it
		// goes through the same hash check and failure branch as a wrong
		// password. Anything else is an outage, not bad credentials.
		user, err := userRepo.RetrieveUserByUserName(r.Context(), req.UserName)
		if errors.Is(err, repository.ErrNotFound) {
			user = nil
		} else if err != nil {
			appHelper.WriteError(
				w,
				http.StatusInternalServerError,
				"Login is temporarily unavailable",
				"USR_LKUP_FAIL",
				err,
			)
			return
		}

		// Always run argon2 (against a dummy for an unknown name) so response
		// time does not reveal which names exist, and count every failure.
		storedHash := ""
		if user != nil {
			storedHash = user.PwHash
		}
		verified := auth.CheckPasswordHashOrDummy(req.Password, storedHash)
		// A login that is not active (blocked, B034) gets the same counted
		// 401 as a wrong password, and only after the hash check.
		active := user != nil &&
			user.AccountStatus == model.AccountStatusActive
		if user == nil || !verified || !active {
			reason := "Password verification failed for user: "
			if user == nil {
				reason = "Login for unknown user: "
			} else if verified {
				reason = "Login for " + user.AccountStatus + " user: "
			}
			appServices.Logger.Warn("PWD_VFY_FAIL", reason+req.UserName)
			// Increment failed attempt counters
			err := incrementFailedAttempt(
				appServices.KV,
				appServices.Config,
				clientIP,
				req.UserName,
			)
			if err != nil {
				// Fail closed: an uncounted failure is a lockout that never
				// trips, so this is an outage, not a plain bad password.
				appHelper.WriteError(
					w,
					http.StatusInternalServerError,
					"Login is temporarily unavailable",
					"FAILED_ATTEMPT_INCR_FAIL",
					err,
				)
				return
			}
			appHelper.WriteError(
				w,
				http.StatusUnauthorized,
				"Invalid credentials",
				"INVALID_CREDS",
				nil,
			)
			return
		}

		// Check if MFA is required
		if !user.MFAExempt {
			// Generate temp token for MFA
			tempTokenBytes := make([]byte, 32)
			if _, err := rand.Read(tempTokenBytes); err != nil {
				appHelper.WriteError(
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
				appHelper.WriteError(
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
				MFAEnrolled:                  user.MFAEnrolled(),
			}

			wrapper := model.AuthResponseWrapper{User: response}
			appHelper.WriteJSON(w, http.StatusOK, wrapper)
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
			appHelper.WriteError(
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
			appHelper.WriteError(
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
		setAuthCookie(w, jwt, authCookieSessionMaxAge)
		// Login completes here for an MFA-exempt user only; an MFA login
		// completes in MFAVerifyHandler.
		resetFailedAttemptsOrLog(appServices, user.UserName)

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
		appHelper.WriteJSON(w, http.StatusOK, wrapper)
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
	appServices *app.Services,
	userRepo UserRepositoryInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security checks
		appHelper := security.NewAppHelper(w, r, appServices, "RefreshHandler")
		appHelper.CheckJWT()
		appHelper.CheckRateLimitAndUserBlock()
		appHelper.SkipCSRF() // CSRF not required for read-only refresh
		// Authorization not required for token refresh
		appHelper.SkipAuthorization()

		if !appHelper.CheckAllChecksComplete() {
			return
		}

		appServices.Logger.Info(
			"REFRESH_START",
			"Refresh request started",
			"username",
			appHelper.GetUsername(),
		)

		jwtClaims := appHelper.GetClaims()

		// Read user
		user, err := userRepo.RetrieveUserByUserName(
			r.Context(),
			jwtClaims.Subject,
		)
		if err != nil {
			appHelper.WriteError(
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
			appHelper.WriteError(
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

		// Re-sign on the same session id: one login is one session, so
		// sign-out (RevokeSession) ends every token it was refreshed into.
		jwt, sessionID, err := reissueJWTForUser(
			user,
			grantsMap[user.SecurityContext],
			appServices,
			jwtClaims.SessionID,
		)
		if err != nil {
			appHelper.WriteError(
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
		setAuthCookie(w, jwt, authCookieSessionMaxAge)

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
		appHelper.WriteJSON(w, http.StatusOK, wrapper)
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
	appServices *app.Services,
	userRepo UserRepositoryInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create app helper for error handling
		appHelper := security.NewAppHelper(
			w, r, appServices, "ChangeSecurityContextHandler",
		)

		// Parse request first to get security context. This precedes the JWT
		// and rate-limit checks, so bound it.
		var req model.ChangeSecurityContextRequest
		if !decodeJSONBody(appHelper, w, r, maxAuthBodyBytes, &req) {
			return
		}

		// Security checks
		appHelper.CheckJWT()
		appHelper.CheckRateLimitAndUserBlock()
		appHelper.SkipCSRF()
		appHelper.SkipAuthorization() // Authorization checked below

		if !appHelper.CheckAllChecksComplete() {
			return
		}

		appServices.Logger.Info(
			"CHANGE_CTX_START",
			"Change security context request started",
			"username",
			appHelper.GetUsername(),
		)

		jwtClaims := appHelper.GetClaims()
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
			appHelper.WriteError(
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
				appHelper.GetUsername(),
			)
			appHelper.WriteError(
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
			appHelper.WriteError(
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

		// The new context gets a new session id; the old one ends, so a
		// sign-out still ends every token of this login.
		err = security.RevokeSession(
			appServices, appHelper.GetClaims().SessionID,
		)
		if err != nil {
			appHelper.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to end the previous session",
				"SES_REVOKE",
				err,
			)
			return
		}

		// Issue new JWT with updated context
		jwt, sessionID, err := createJWTForUser(
			&updatedUser,
			grantsMap[req.SecurityContext],
			appServices,
		)
		if err != nil {
			appHelper.WriteError(
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
		setAuthCookie(w, jwt, authCookieSessionMaxAge)

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
		appHelper.WriteJSON(w, http.StatusOK, wrapper)
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
func LogoffHandler(appServices *app.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Use AppHelper to extract username from JWT (skip other validations)
		appHelper := security.NewAppHelper(w, r, appServices, "LogoffHandler")
		appHelper.CheckJWT()                   // Extract username from JWT
		appHelper.CheckRateLimitAndUserBlock() // No rate limiting for logoff
		appHelper.SkipCSRF()                   // No CSRF needed for logoff
		// No authorization needed for logoff
		appHelper.SkipAuthorization()

		if !appHelper.CheckAllChecksComplete() {
			return
		}

		appHelper.Logger().Info(
			"LOGOFF_START",
			"Logoff request started",
			"username",
			appHelper.GetUsername(),
		)

		// End the session server-side: expiring the cookie only asks the
		// browser to forget the token; a copy of it would stay valid.
		err := security.RevokeSession(
			appServices, appHelper.GetClaims().SessionID,
		)
		if err != nil {
			appHelper.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to end the session",
				"LOGOFF_REVOKE",
				err,
			)
			return
		}

		// Clear the auth cookie
		setAuthCookie(w, "", authCookieExpireNow)

		w.WriteHeader(http.StatusOK)
		appHelper.Logger().Info(
			"LOGOFF_SUCCESS",
			"Logoff successful",
			"username",
			appHelper.GetUsername(),
		)
	}
}

// ChangeOwnPasswordHandler handles changing the current user's password
func ChangeOwnPasswordHandler(
	appServices *app.Services,
	userRepo UserRepositoryInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security checks
		appHelper := security.NewAppHelper(
			w, r, appServices, "ChangeOwnPasswordHandler",
		)
		appHelper.CheckJWT()
		appHelper.CheckRateLimitAndUserBlock()
		appHelper.CheckCSRF()
		appHelper.CheckAuthorization(model.GrantChangeOwnPassword)

		if !appHelper.CheckAllChecksComplete() {
			return
		}

		appServices.Logger.Info(
			"CHANGE_PW_START",
			"Change own password request started",
			"username",
			appHelper.GetUsername(),
		)

		jwtClaims := appHelper.GetClaims()

		// Parse request body
		var req struct {
			OldPassword string `json:"oldPassword"`
			NewPassword string `json:"newPassword"`
		}
		if !decodeJSONBody(appHelper, w, r, maxLoginLayerBodyBytes, &req) {
			return
		}

		// Validate required fields
		if req.OldPassword == "" || req.NewPassword == "" {
			appHelper.WriteError(
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
			appHelper.WriteError(
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
			appHelper.WriteError(
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
		if !auth.CheckPasswordHash(req.OldPassword, user.PwHash) {
			appServices.Logger.Warn(
				"PWD_VFY_FAIL",
				"Old password verification failed for user: "+jwtClaims.Subject,
			)
			// 400, not 401: the caller's session is valid, only the
			// old password they typed is wrong. A 401 here would trip
			// the web's global session-expired path (utils/api.ts) and
			// sign the user out mid-change (B092).
			appHelper.WriteError(
				w,
				http.StatusBadRequest,
				"Old password is incorrect",
				"INVALID_CREDS",
				nil,
			)
			return
		}

		// Hash new password
		hashedPassword, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			appHelper.WriteError(
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
		now := model.Now()
		err = userRepo.UpdateUserPassword(
			r.Context(),
			jwtClaims.UserID,
			hashedPassword,
			now,
			jwtClaims.UserID,
			user.SecurityContext,
		)
		if err != nil {
			appHelper.WriteError(
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

		// Every other session of this login ends; the one that changed the
		// password stays signed in.
		err = security.RevokeUserSessions(
			appServices, jwtClaims.Subject, jwtClaims.SessionID,
		)
		if err != nil {
			appHelper.WriteError(
				w,
				http.StatusInternalServerError,
				"Password changed; ending the other sessions failed",
				"SES_REVOKE",
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
		appHelper.WriteJSON(w, http.StatusOK, wrapper)
	}
}

// createJWTForUser creates a JWT on a new session id (sign-in, context
// switch).
func createJWTForUser(
	user *model.AppUserModel,
	grantList []int,
	appServices *app.Services,
) (string, string, error) {
	return reissueJWTForUser(user, grantList, appServices, "")
}

// reissueJWTForUser signs a JWT with a fresh expiry on sessionID; an id too
// short to hold a CSRF tail ("" included) is replaced by a new one.
func reissueJWTForUser(
	user *model.AppUserModel,
	grantList []int,
	appServices *app.Services,
	sessionID string,
) (string, string, error) {
	if len(sessionID) < 3 {
		sessionIDBytes := make([]byte, 16)
		if _, err := rand.Read(sessionIDBytes); err != nil {
			return "", "", err
		}
		sessionID = hex.EncodeToString(sessionIDBytes)
	}

	// Create JWT
	jwt, err := auth.GenerateJWT(
		appServices.Config.AppName,
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
