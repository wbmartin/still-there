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

// UserSelfCreateHandler handles user self-registration (public endpoint)
func UserSelfCreateHandler(
	appServices *app_lib.AppServices,
	userRepo *repository.UserRepository,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create app helper
		app := app_lib.NewAppHelper(w, r, appServices)

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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			app.WriteError(
				w,
				http.StatusBadRequest,
				"Invalid request body",
				"IRB0",
				err,
			)
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
		hashedPassword, err := app_lib.HashPassword(req.Password)
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
	appServices *app_lib.AppServices,
	userRepo *repository.UserRepository,
) http.HandlerFunc {
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
		r.Body = io.NopCloser(
			bytes.NewReader(bodyBytes),
		) // Reset body for later
		if err := json.Unmarshal(bodyBytes, &tempReq); err == nil &&
			tempReq.TempToken != "" {
			// Temp token path
			tempToken := tempReq.TempToken
			// Get username from temp token
			userNameBytes, err := appServices.KV.Get("TEMP_AUTH_" + tempToken)
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

		// Generate TOTP secret if not exists
		var secret string
		if user.TOTPSeed != nil && *user.TOTPSeed != "" {
			// Decrypt existing seed
			decrypted, err := decryptTOTPSeed(
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
		processedAppName := regexp.MustCompile(`[^a-zA-Z0-9]`).
			ReplaceAllString(appServices.Config.AppName, "")
		qrCodeUrl := "otpauth://totp/" + processedAppName + ":" + userName + "?secret=" + secret + "&issuer=" + appServices.Config.AppName

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
	appServices *app_lib.AppServices,
	userRepo *repository.UserRepository,
) http.HandlerFunc {
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
			app.WriteError(
				w,
				http.StatusBadRequest,
				"Invalid request body",
				"IRB0",
				err,
			)
			return
		}

		var userName string
		if req.TempToken != "" {
			// Temp token path
			userNameBytes, err := appServices.KV.Get(
				"TEMP_AUTH_" + req.TempToken,
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

		if user.TOTPSeed == nil || *user.TOTPSeed == "" {
			app.WriteError(
				w,
				http.StatusBadRequest,
				"MFA not set up",
				"MFNS",
				nil,
			)
			return
		}

		// Decrypt TOTP secret
		decryptedSecret, err := decryptTOTPSeed(
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
		clientIP := getClientIP(r)
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

		// Validate TOTP code
		valid := totp.Validate(req.TOTPCode, decryptedSecret)
		if !valid {
			// Increment failed attempt counters
			err := incrementFailedAttempt(
				appServices.KV,
				appServices.Config,
				clientIP,
				userName,
			)
			if err != nil {
				app.Logger().Error(
					"FAILED_ATTEMPT_INCR_FAIL",
					"Failed to increment failed attempts for user: "+userName+", error: "+err.Error(),
					"username",
					userName,
				)

				// Don't fail the request, just log
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
			appServices.KV.Set(
				"TEMP_AUTH_"+req.TempToken,
				"",
				"0",
			) // Expire immediately

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
				), // TODO: populate
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
	appServices *app_lib.AppServices,
	userRepo *repository.UserRepository,
) http.HandlerFunc {
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

		app.Info("ADM_RST_MFA_S", "Admin reset MFA request started")

		// Parse request
		var req struct {
			UserID int `json:"userId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			app.WriteError(
				w,
				http.StatusBadRequest,
				"Invalid request body",
				"IRB0",
				err,
			)
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

		// Reset MFA seed only
		newUpdateTS, err := userRepo.UpdateUserTOTPSeed(
			r.Context(),
			targetUser.AppUserID,
			"",
			targetUser.UpdateTS,
			app.GetClaims().UserID,
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
