package handler

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/security"
)

// UserSelfCreateHandler handles user self-registration (public endpoint)
func UserSelfCreateHandler(
	appServices *app.Services,
	userRepo UserRepositoryInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create app helper
		app := security.NewAppHelper(w, r, appServices, "UserSelfCreateHandler")

		// No auth required for self-create
		app.Logger().Info(
			"USC0",
			"User self-create request started",
			"username",
			"anonymous",
		)
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
		if !decodeJSONBody(app, w, r, maxAuthBodyBytes, &req) {
			return
		}

		// Validate input
		if req.Username == "" || req.Password == "" {
			app.WriteError(
				w,
				http.StatusBadRequest,
				"Username and password are required",
				"BRQ0",
				nil,
			)
			return
		}

		if len(req.Password) < 8 {
			app.WriteError(
				w,
				http.StatusBadRequest,
				"Password must be at least 8 characters long",
				"BRQ0",
				nil,
			)
			return
		}

		// Check if username is available
		existingUser, err := userRepo.RetrieveUserByUserName(
			r.Context(),
			req.Username,
		)
		if err != nil {
			if strings.Contains(err.Error(), "user not found") {
				// Username is available
			} else {
				app.WriteError(w, http.StatusInternalServerError, "Error checking username availability", "ISE0", err)
				return
			}
		}
		if existingUser != nil {
			app.WriteError(
				w,
				http.StatusConflict,
				"Username already taken",
				"UNTK",
				nil,
			)
			return
		}

		// Hash password
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to process password",
				"UCH0",
				err,
			)
			return
		}

		// Create user
		now := CurrentTimestamp()
		defaultSecurityContext := 1 // Default security context for self-created users
		defaultSecurityGroupId := 3 // "New User" group ID
		newUser := &model.AppUserModel{
			UserName:        req.Username,
			PwHash:          hashedPassword,
			FirstName:       "",
			LastName:        "",
			Email:           "",
			SecurityContext: defaultSecurityContext,
			AppUserGroupMembershipList: []int{
				defaultSecurityGroupId,
			}, // "New User" group
			MFAExempt:    false,
			TOTPSeed:     nil, // Will be set during MFA setup
			CreateTS:     now,
			CreateUserID: 0, // System
			UpdateTS:     now,
			UpdateUserID: 0, // System
		}

		userID, err := userRepo.CreateUser(
			r.Context(),
			newUser,
			1,
		) // Self-created users go in security context 1
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to create user",
				"UCF0",
				err,
			)
			return
		}

		app.Logger().Info(
			"USCC",
			"User self-created successfully",
			"username",
			req.Username,
			"userId",
			userID,
		)

		wrapper := model.DataWrapper{
			Data: map[string]string{"message": "User created successfully"},
		}
		app.WriteJSON(w, http.StatusCreated, wrapper)
	}
}

// MFASetupHandler handles MFA setup for authenticated users
func MFASetupHandler(
	appServices *app.Services,
	userRepo UserRepositoryInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security checks
		app := security.NewAppHelper(w, r, appServices, "MFASetupHandler")
		app.CheckRateLimitAndUserBlock()
		app.SkipCSRF()
		app.SkipAuthorization() // MFA setup is user-specific

		var userName string
		// Check for temp token in request body first
		var tempReq struct {
			TempToken string `json:"tempToken"`
		}
		// The body has to be read here because it decides which auth path runs
		// below -- temp token or JWT -- so the read necessarily precedes
		// CheckAllChecksComplete(). That makes bounding it the only protection
		// available: an unauthenticated caller reaches this line, and a
		// rate-limited one does too, because nothing checks blocked first.
		bodyBytes, ok := readBody(app, w, r, maxAuthBodyBytes)
		if !ok {
			return
		}
		if err := json.Unmarshal(bodyBytes, &tempReq); err == nil &&
			tempReq.TempToken != "" {
			// Temp token path - skip JWT for temp token authentication.
			// The other checks (rate limit, user block) have already run;
			// this is where a failed one stops the request.
			app.SkipJWT()
			if !app.CheckAllChecksComplete() {
				return
			}
			tempToken := tempReq.TempToken
			// Get username from temp token
			userNameBytes, err := appServices.KV.Get(
				"TEMP_AUTH_SUCCESS_" + tempToken,
			)
			if err != nil || userNameBytes == "" {
				app.WriteError(
					w,
					http.StatusUnauthorized,
					"Invalid temp token",
					"ITTK",
					nil,
				)
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

		app.Info(
			"MFA_STP_STRT",
			"MFA setup request started for user: "+userName,
		)

		// Get current user
		user, err := userRepo.RetrieveUserByUserName(r.Context(), userName)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to retrieve user",
				"USF0",
				err,
			)
			return
		}

		if user.MFAExempt {
			app.WriteError(
				w,
				http.StatusBadRequest,
				"MFA not required for this user",
				"MFAN",
				nil,
			)
			return
		}

		// An enrolled user verifies a code; the stored secret is never shown
		// again. Returning it here would hand the seed and its QR code to
		// anyone holding a temp token, which takes only the password.
		if user.MFAEnrolled() {
			app.WriteError(
				w,
				http.StatusConflict,
				"MFA is already set up: enter the code from your authenticator app",
				"MFA_ALREADY_SETUP",
				nil,
			)
			return
		}

		// Generate a TOTP secret if none exists. An existing seed here is an
		// enrolment not yet verified, so the same secret is shown again.
		var secret string
		if user.TOTPSeed != nil && *user.TOTPSeed != "" {
			// Decrypt existing seed
			decrypted, err := auth.DecryptTOTPSeed(
				*user.TOTPSeed,
				appServices.Config.TOTPEncryptionKey,
			)
			if err != nil {
				app.WriteError(
					w,
					http.StatusInternalServerError,
					"Failed to decrypt TOTP secret",
					"TDT0",
					err,
				)
				return
			}
			// Strip padding from existing secrets and ensure uppercase
			secret = strings.TrimRight(decrypted, "=")
			secret = strings.ToUpper(secret)
		} else {
			// Generate new TOTP secret
			var err error
			secret, err = auth.GenerateTOTPSecret()
			if err != nil {
				app.WriteError(w, http.StatusInternalServerError, "Failed to generate secret", "SGF0", err)
				return
			}

			// Validate generated secret
			if !auth.ValidateTOTPSecret(secret) {
				app.WriteError(w, http.StatusInternalServerError, "Generated invalid TOTP secret", "TOTP_INVALID_SECRET", nil)
				return
			}

			// Encrypt and save
			encrypted, err := auth.EncryptTOTPSeed(secret, appServices.Config.TOTPEncryptionKey)
			if err != nil {
				app.WriteError(w, http.StatusInternalServerError, "Failed to encrypt TOTP secret", "TSE0", err)
				return
			}
			_, err = userRepo.UpdateUserTOTPSeed(
				r.Context(), user.AppUserID, encrypted, user.UpdateTS,
				user.AppUserID, user.SecurityContext,
			)
			if err != nil {
				app.WriteError(w, http.StatusInternalServerError, "Failed to save TOTP secret", "TSS0", err)
				return
			}
		}

		// Generate QR code URL
		qrCodeUrl, err := auth.GenerateTOTPURI(
			appServices.Config.AppName,
			userName,
			secret,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to generate TOTP URI",
				"TOTP_URI_ERROR",
				err,
			)
			return
		}

		// Generate QR code image
		png, err := qrcode.Encode(qrCodeUrl, qrcode.Medium, 256)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to generate QR code",
				"QRCG",
				err,
			)
			return
		}
		qrCodeImage := "data:image/png;base64," + base64.StdEncoding.EncodeToString(
			png,
		)

		response := map[string]interface{}{
			"secret":      secret,
			"qrCodeImage": qrCodeImage,
		}

		wrapper := model.DataWrapper{Data: response}
		app.WriteJSON(w, http.StatusOK, wrapper)
		app.Info("MFA_STP_DATA", "MFA setup data provided")
	}
}

// MFAVerifyHandler handles MFA verification
func MFAVerifyHandler(
	appServices *app.Services,
	userRepo UserRepositoryInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security checks
		app := security.NewAppHelper(w, r, appServices, "MFAVerifyHandler")
		app.CheckRateLimitAndUserBlock()
		app.SkipCSRF()
		app.SkipAuthorization()

		// Parse request
		var req struct {
			TempToken string `json:"tempToken"`
			TOTPCode  string `json:"totpCode"`
		}
		// Same reasoning as MFASetupHandler: read before the auth gate, so
		// bound it.
		if !decodeJSONBody(app, w, r, maxAuthBodyBytes, &req) {
			return
		}

		var userName string
		if req.TempToken != "" {
			// Temp token path - skip JWT for temp token authentication.
			// The other checks (rate limit, user block) have already run;
			// this is where a failed one stops the request.
			app.SkipJWT()
			if !app.CheckAllChecksComplete() {
				return
			}
			userNameBytes, err := appServices.KV.Get(
				"TEMP_AUTH_SUCCESS_" + req.TempToken,
			)
			if err != nil || userNameBytes == "" {
				app.WriteError(
					w,
					http.StatusUnauthorized,
					"Invalid temp token",
					"ITTK",
					nil,
				)
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

		app.Info(
			"MFA_VFY_STRT",
			"MFA verify request started for user: "+userName,
		)

		// Get user
		user, err := userRepo.RetrieveUserByUserName(r.Context(), userName)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to retrieve user",
				"USF0",
				err,
			)
			return
		}

		// Only check MFA setup for logged-in users (JWT path), not during initial setup (temp token path)
		if !user.MFASetup && req.TempToken == "" {
			app.WriteError(
				w,
				http.StatusBadRequest,
				"MFA not set up. Please complete MFA setup first.",
				"MFNS",
				nil,
			)
			return
		}

		// Decrypt TOTP secret; an admin reset leaves '' rather than NULL
		if user.TOTPSeed == nil || *user.TOTPSeed == "" {
			app.WriteError(
				w,
				http.StatusBadRequest,
				"TOTP secret not found. Please restart MFA setup.",
				"TOTP_SECRET_MISSING",
				nil,
			)
			return
		}
		decryptedSecret, err := auth.DecryptTOTPSeed(
			*user.TOTPSeed,
			appServices.Config.TOTPEncryptionKey,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to decrypt TOTP secret",
				"TSD0",
				err,
			)
			return
		}

		// Check for account lockout
		clientIP := security.ClientIP(
			r, appServices.Config.TrustedProxyCount,
		)
		locked, err := isAccountLocked(appServices.KV, clientIP, userName)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to check account lockout",
				"LOCKOUT_CHECK_FAIL",
				err,
			)
			return
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

		// A code is good once: its step must be above the last step accepted
		// for this user, recorded in the same atomic step. A replay is
		// answered and counted exactly like a wrong code.
		step, valid := auth.ValidateTOTPCodeAt(
			req.TOTPCode, decryptedSecret, time.Now(),
		)
		if valid {
			stored, err := appServices.KV.SetIfGreater(
				totpLastStepKey(userName), step, totpLastStepTTL,
			)
			if err != nil {
				// Fail closed: a code accepted unrecorded is replayable.
				app.WriteError(
					w,
					http.StatusInternalServerError,
					"Login is temporarily unavailable",
					"TOTP_STEP_STORE",
					err,
				)
				return
			}
			valid = stored
		}
		if !valid {
			// Increment failed attempt counters
			err := incrementFailedAttempt(
				appServices.KV,
				appServices.Config,
				clientIP,
				userName,
			)
			if err != nil {
				// Fail closed: an uncounted failure is a lockout that never
				// trips, so this is an outage, not a plain bad password.
				app.WriteError(
					w,
					http.StatusInternalServerError,
					"Login is temporarily unavailable",
					"FAILED_ATTEMPT_INCR_FAIL",
					err,
				)
				return
			}
			app.WriteError(
				w,
				http.StatusUnauthorized,
				"Invalid TOTP code",
				"ITCC",
				nil,
			)
			return
		}

		// Update MFA setup status to true after successful validation
		if !user.MFASetup {
			_, err = userRepo.UpdateUserMFASetup(
				r.Context(),
				user.AppUserID,
				true,
				user.UpdateTS,
				user.AppUserID,
				user.SecurityContext,
			)
			if err != nil {
				app.Logger().Error(
					"MFA_SETUP_UPDATE_FAIL",
					"Failed to update MFA setup status for user: "+userName+", error: "+err.Error(),
					"username",
					userName,
				)

				// Don't fail request, MFA validation was successful
			}
		}

		if req.TempToken != "" {
			// Complete authentication for onboarding
			// Get grants
			grantsMap, err := userRepo.GetGrantsForGroups(
				r.Context(),
				user.AppUserGroupMembershipList,
			)
			if err != nil {
				app.WriteError(
					w,
					http.StatusInternalServerError,
					"Failed to get grants",
					"GRFL",
					err,
				)
				return
			}

			// Issue JWT
			jwt, sessionID, err := createJWTForUser(
				user,
				grantsMap[user.SecurityContext],
				appServices,
			)
			if err != nil {
				app.WriteError(
					w,
					http.StatusInternalServerError,
					"Failed to create JWT",
					"JWFL",
					err,
				)
				return
			}

			// Set JWT cookie
			setAuthCookie(w, jwt, 0)
			// The login completes here (password step, then this code)
			resetFailedAttemptsOrLog(appServices, userName)

			// CSRF token
			csrfToken := sessionID[len(sessionID)-3:]

			// The temp token is single-use: delete it. (Set with ttl "0" was
			// an invalid expiration whose error went unread, so the token
			// stayed replayable for its whole 10 minutes.)
			if err := appServices.KV.Del(
				"TEMP_AUTH_SUCCESS_" + req.TempToken,
			); err != nil {
				app.Logger().Error("MFA_TT_DEL_FAIL", "Failed to delete temp token: "+err.Error())
			}

			// Return full user data
			response := model.Auth2ResponseModel{
				AppUserID:       user.AppUserID,
				UserName:        user.UserName,
				FirstName:       user.FirstName,
				LastName:        user.LastName,
				SecurityContext: user.SecurityContext,
				GrantList:       grantsMap[user.SecurityContext],
				AvailableSecurityContextList: make(
					[]int,
					0,
					len(grantsMap),
				),
				MFARequired: false, // Now verified
			}
			for ctx := range grantsMap {
				response.AvailableSecurityContextList = append(
					response.AvailableSecurityContextList,
					ctx,
				)
			}

			wrapper := model.Auth2ResponseWrapper{
				User:      response,
				CsrfToken: csrfToken,
			}
			app.WriteJSON(w, http.StatusOK, wrapper)
			app.Info(
				"MFA_ONB_CMP",
				"MFA onboarding completed for user: "+userName,
			)
		} else {
			// For logged in users, just confirm
			wrapper := model.DataWrapper{Data: map[string]string{"message": "MFA verified successfully"}}
			app.WriteJSON(w, http.StatusOK, wrapper)
			app.Info("MFA_VFY_OK", "MFA verification successful for user: "+userName)
		}
	}
}

// AdminResetUserMFAHandler handles admin reset of user MFA
func AdminResetUserMFAHandler(
	appServices *app.Services,
	userRepo UserRepositoryInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Security checks
		app := security.NewAppHelper(w, r, appServices, "AdminResetUserMFAHandler")
		app.CheckJWT()
		app.CheckRateLimitAndUserBlock()
		app.CheckCSRF()
		app.CheckAuthorization(model.GrantResetUserMFA)

		if !app.CheckAllChecksComplete() {
			return
		}

		app.Info("ADM_RST_MFA_S", "Admin reset MFA request started")

		// Parse request
		var req struct {
			UserID int `json:"userId"`
		}
		if !decodeJSONBody(app, w, r, maxLoginLayerBodyBytes, &req) {
			return
		}

		// Get target user
		users, err := userRepo.RetrieveUserList(
			r.Context(),
			app.GetClaims().SecurityContext,
			[]int{req.UserID},
			0,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to retrieve user",
				"USF0",
				err,
			)
			return
		}
		if len(users) == 0 {
			app.WriteError(
				w,
				http.StatusNotFound,
				"User not found",
				"NFN0",
				nil,
			)
			return
		}

		targetUser := users[0]

		// A new secret starts with no spent step; a stale one could refuse
		// the first code of a re-enrolment.
		if err := appServices.KV.Del(
			totpLastStepKey(targetUser.UserName),
		); err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to reset MFA",
				"TOTP_STEP_CLEAR",
				err,
			)
			return
		}

		// Reset MFA seed only
		newUpdateTS, err := userRepo.UpdateUserTOTPSeed(
			r.Context(),
			targetUser.AppUserID,
			"",
			targetUser.UpdateTS,
			app.GetClaims().UserID,
			targetUser.SecurityContext,
		)
		if err != nil {
			app.WriteError(
				w,
				http.StatusInternalServerError,
				"Failed to reset MFA",
				"MFRE",
				err,
			)
			return
		}

		if !revokeUserSessionList(app, w, appServices, targetUser.UserName) {
			return
		}
		wrapper := model.DataWrapper{
			Data: map[string]interface{}{
				"message":  "MFA reset successfully",
				"updateTs": newUpdateTS,
			},
		}
		app.WriteJSON(w, http.StatusOK, wrapper)
		app.Info(
			"ADM_RST_MFA_D",
			"Admin reset MFA completed for user: "+targetUser.UserName,
		)
	}
}

// totpLastStepTTL outlives the window a code is accepted in (90 s: the step
// either side of now), so a replay inside it is still refused.
const totpLastStepTTL = "3min"

func totpLastStepKey(userName string) string {
	return "TOTP_LAST_STEP*" + userName
}
