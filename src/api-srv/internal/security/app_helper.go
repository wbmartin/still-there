package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/logging"
	"github.com/wbmartin/starter-api/internal/model"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const claimsKey contextKey = "claims"

// AppHelper encapsulates request-specific security checks and state
type AppHelper struct {
	r              *http.Request
	w              http.ResponseWriter
	appServices    *app.Services
	claims         *auth.Claims
	ctx            context.Context
	username       string
	helperLocation string
	blocked        bool
	jwtChecked     bool
	rateChecked    bool
	userChecked    bool
	csrfChecked    bool
	authChecked    bool
}

// NewAppHelper creates a new app helper
func NewAppHelper(
	w http.ResponseWriter,
	r *http.Request,
	appServices *app.Services,
	helperLocation string,
) *AppHelper {
	ah := &AppHelper{
		r:              r,
		w:              w,
		appServices:    appServices,
		helperLocation: helperLocation,
		username:       "preauth",
	}
	ah.appServices.Logger.Info("APP_HELPER_INIT", "AppHelper instantiated", "handler", helperLocation)
	return ah
}

// SkipJWT skips JWT validation
func (sh *AppHelper) SkipJWT() *AppHelper {
	sh.jwtChecked = true
	return sh
}

// SkipRateLimit skips rate limit check
func (sh *AppHelper) SkipRateLimit() *AppHelper {
	sh.rateChecked = true
	return sh
}

// SkipUserBlock skips user block check
func (sh *AppHelper) SkipUserBlock() *AppHelper {
	sh.userChecked = true
	return sh
}

// SkipCSRF skips CSRF check
func (sh *AppHelper) SkipCSRF() *AppHelper {
	sh.csrfChecked = true
	return sh
}

// SkipAuthorization skips authorization check
func (sh *AppHelper) SkipAuthorization() *AppHelper {
	sh.authChecked = true
	return sh
}

// GetClaims returns the validated JWT claims
func (sh *AppHelper) GetClaims() *auth.Claims {
	return sh.claims
}

// GetUsername returns the username for logging
// (preauth or authenticated subject)
func (ah *AppHelper) GetUsername() string {
	return ah.username
}

// Debug logs at debug level with automatic username attachment
func (ah *AppHelper) Debug(code, message string) {
	ah.appServices.Logger.DebugWithCallerSkip(
		code,
		message,
		3,
		"username",
		ah.username,
	)
}

// Info logs at info level with automatic username attachment
func (ah *AppHelper) Info(code, message string) {
	ah.appServices.Logger.InfoWithCallerSkip(
		code,
		message,
		3,
		"username",
		ah.username,
	)
}

// Warn logs at warn level with automatic username attachment
func (ah *AppHelper) Warn(code, message string) {
	ah.appServices.Logger.WarnWithCallerSkip(
		code,
		message,
		3,
		"username",
		ah.username,
	)
}

// Error logs at error level with automatic username attachment
func (ah *AppHelper) Error(code, message string) {
	ah.appServices.Logger.ErrorWithCallerSkip(
		code,
		message,
		3,
		"username",
		ah.username,
	)
}

// ErrorWithCallerSkip logs at error level with custom caller skip
// and automatic username
func (ah *AppHelper) ErrorWithCallerSkip(code, message string, skip int) {
	ah.appServices.Logger.ErrorWithCallerSkip(
		code,
		message,
		skip+1,
		"username",
		ah.username,
	)
}

// InfoWithCallerSkip logs at info level with custom caller skip
// and automatic username
func (ah *AppHelper) InfoWithCallerSkip(code, message string, skip int) {
	ah.appServices.Logger.InfoWithCallerSkip(
		code,
		message,
		skip+1,
		"username",
		ah.username,
	)
}

// DebugWithCallerSkip logs at debug level with custom caller skip
// and automatic username
func (ah *AppHelper) DebugWithCallerSkip(code, message string, skip int) {
	ah.appServices.Logger.DebugWithCallerSkip(
		code,
		message,
		skip+1,
		"username",
		ah.username,
	)
}

// WarnWithCallerSkip logs at warn level with custom caller skip
// and automatic username
func (ah *AppHelper) WarnWithCallerSkip(code, message string, skip int) {
	ah.appServices.Logger.WarnWithCallerSkip(
		code,
		message,
		skip+1,
		"username",
		ah.username,
	)
}

// WriteJSON writes a JSON response
func (ah *AppHelper) WriteJSON(
	w http.ResponseWriter,
	status int,
	data interface{},
) {
	// Only enforce security checks for success responses (status < 400)
	// Error responses (4xx/5xx) are allowed even if checks failed
	if status < 400 && !ah.CheckAllChecksComplete() {
		ah.appServices.Logger.Error(
			"SEC0",
			"Attempted to write JSON before security checks complete",
		)
		http.Error(w, "Security check failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		ah.appServices.Logger.Error(
			"JEN0",
			"Failed to encode JSON response: "+err.Error(),
		)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// WriteError writes an error response with logging
func (ah *AppHelper) WriteError(
	w http.ResponseWriter,
	status int,
	clientMsg string,
	code string,
	err error,
) {
	internalMsg := clientMsg
	if err != nil {
		internalMsg = clientMsg + ": " + err.Error()
	}
	ah.appServices.Logger.ErrorWithCallerSkip(
		code,
		internalMsg,
		3,
		"username",
		ah.username,
	)
	response := map[string]string{"error": clientMsg}
	ah.WriteJSON(w, status, response)
}

// GetContext returns the current context
func (sh *AppHelper) GetContext() context.Context {
	return sh.ctx
}

// CanProceed returns true if processing can continue
func (sh *AppHelper) CanProceed() bool {
	return !sh.blocked
}

// Logger returns the logger
func (ah *AppHelper) Logger() *logging.Logger {
	return ah.appServices.Logger
}

// CheckJWT validates the JWT token
func (sh *AppHelper) CheckJWT() *AppHelper {
	sh.jwtChecked = true
	if sh.blocked {
		return sh
	}

	cookie, err := sh.r.Cookie("auth")
	if err != nil {
		sh.failAndRespond(http.StatusUnauthorized, "Missing auth cookie")
		return sh
	}

	tokenString := cookie.Value
	claims, err := auth.ValidateJWT(tokenString, sh.appServices.Config.JWTSecret)
	if err != nil {
		sh.appServices.Logger.Warn(
			"JWF0",
			"JWT validation failed: "+err.Error(),
		)
		sh.failAndRespond(
			http.StatusUnauthorized,
			"Invalid authentication token",
		)
		return sh
	}

	sh.claims = claims
	sh.ctx = context.WithValue(sh.r.Context(), claimsKey, claims)
	sh.username = claims.Subject

	return sh
}

// ClientIP returns the caller's IP: the first valid entry of
// X-Forwarded-For when a proxy set it, else the host part of RemoteAddr.
// It keys anonymous rate limiting and login lockout, so both paths agree.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip := strings.TrimSpace(strings.Split(xff, ",")[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// CheckRateLimit checks request rate limits
func (sh *AppHelper) CheckRateLimit() *AppHelper {
	sh.rateChecked = true
	if sh.blocked {
		return sh
	}

	// Try to parse JWT if present (even if JWT check was skipped)
	if sh.claims == nil {
		cookie, err := sh.r.Cookie("auth")
		if err == nil {
			tokenString := cookie.Value
			claims, err := auth.ValidateJWT(
				tokenString,
				sh.appServices.Config.JWTSecret,
			)
			if err == nil {
				sh.claims = claims
				sh.username = claims.Subject
			}
		}
	}

	// Exempt admin user from rate limiting
	if sh.claims != nil && sh.claims.Subject == "admin" {
		return sh
	}

	var limit int
	var keyIdentifier string

	if sh.claims != nil {
		limit = sh.appServices.Config.MaxUserReqPerMin
		keyIdentifier = sh.claims.Subject
	} else {
		limit = sh.appServices.Config.MaxUnauthReqPerMin
		keyIdentifier = ClientIP(sh.r)
	}

	currentMinute := time.Now().Minute()
	rateKey := fmt.Sprintf("RPM*%s*%d", keyIdentifier, currentMinute)

	count, err := sh.appServices.KV.Incr(rateKey, "1min")
	if err != nil {
		// The store is in-process: an error here is a bug (closed store,
		// oversized key), not an outage. Fail closed.
		sh.appServices.Logger.Error("RATE_LIMIT_FAIL",
			"Rate limit check failed ("+rateKey+"): "+err.Error())
		sh.failAndRespond(http.StatusInternalServerError, "Rate limit check failed")
		return sh
	}

	if count > limit {
		sh.appServices.Logger.Warn("RATE_LIMIT_EXCEED",
			fmt.Sprintf("Rate limit exceeded - key: %s, count: %d, limit: %d",
				rateKey, count, limit))
		sh.failAndRespond(http.StatusTooManyRequests, "Rate limit exceeded")
		return sh
	}

	sh.appServices.Logger.Debug("RLP0",
		fmt.Sprintf("Rate limit check passed - key: %s, count: %d, limit: %d",
			rateKey, count, limit))
	return sh
}

// CheckRateLimitAndUserBlock performs the rate limit and user block checks
// together, marking both as done
func (sh *AppHelper) CheckRateLimitAndUserBlock() *AppHelper {
	if sh.blocked {
		return sh
	}

	// Exempt admin user from rate limiting
	if sh.claims != nil && sh.claims.Subject == "admin" {
		sh.rateChecked = true
		sh.userChecked = true
		return sh
	}

	currentMinute := time.Now().Minute()

	var rateKey string
	var limit int
	if sh.claims != nil {
		rateKey = fmt.Sprintf("RPM*%s*%d", sh.claims.Subject, currentMinute)
		limit = sh.appServices.Config.MaxUserReqPerMin
	} else {
		rateKey = fmt.Sprintf("RPM*%s*%d", ClientIP(sh.r), currentMinute)
		limit = sh.appServices.Config.MaxUnauthReqPerMin
	}

	rateCount, err := sh.appServices.KV.Incr(rateKey, "1min")
	if err != nil {
		// In-process store: an error is a bug, not an outage. Fail closed.
		sh.appServices.Logger.Error("RATE_LIMIT_FAIL",
			"Rate limit check failed ("+rateKey+"): "+err.Error())
		sh.failAndRespond(http.StatusInternalServerError, "Rate limit check failed")
		return sh
	}
	if rateCount > limit {
		sh.appServices.Logger.Warn("RLE0",
			fmt.Sprintf("Rate limit exceeded - key: %s, count: %d, limit: %d",
				rateKey, rateCount, limit))
		sh.failAndRespond(http.StatusTooManyRequests, "Rate limit exceeded")
		return sh
	}

	sh.appServices.Logger.Debug("RLP0",
		fmt.Sprintf("Rate limit check passed - key: %s, count: %d, limit: %d",
			rateKey, rateCount, limit))
	// Block check applies to authenticated callers only
	if sh.claims != nil {
		blockedValue, err := sh.appServices.KV.Get("BLOCKED*" + sh.claims.Subject)
		if err != nil {
			sh.appServices.Logger.Error("UBK0",
				"Blocked-status check failed: "+err.Error())
			sh.failAndRespond(http.StatusInternalServerError, "User block check failed")
			return sh
		}
		if blockedValue != "" {
			sh.appServices.Logger.Warn(
				"UBK0",
				"User is blocked: "+sh.claims.Subject,
			)
			sh.failAndRespond(http.StatusForbidden, "User account is blocked")
			return sh
		}
	}

	sh.rateChecked = true
	sh.userChecked = true
	return sh
}

// CheckUserBlocked checks if the user is blocked
func (sh *AppHelper) CheckUserBlocked() *AppHelper {
	if sh.blocked || sh.claims == nil {
		return sh
	}

	blockedKey := "BLOCKED*" + sh.claims.Subject
	blockedValue, err := sh.appServices.KV.Get(blockedKey)
	if err != nil {
		sh.appServices.Logger.Error("UBK0",
			"Blocked-status check failed: "+err.Error())
		sh.failAndRespond(http.StatusInternalServerError, "User block check failed")
		return sh
	}

	if blockedValue != "" {
		// Key exists, user is blocked
		sh.appServices.Logger.Warn(
			"UBK0",
			"User is blocked: "+sh.claims.Subject,
		)
		sh.failAndRespond(http.StatusForbidden, "User account is blocked")
		return sh
	}

	return sh
}

// CheckCSRF validates the CSRF token
func (sh *AppHelper) CheckCSRF() *AppHelper {
	sh.csrfChecked = true
	if sh.blocked {
		return sh
	}

	// Skip for safe methods
	switch sh.r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return sh
	}

	if sh.claims == nil || sh.claims.SessionID == "" {
		sh.failAndRespond(http.StatusInternalServerError,
			"CSRF check requires authenticated user")
		return sh
	}

	csrfToken := sh.r.Header.Get("X-CSRF-Token")
	if csrfToken == "" {
		sh.failAndRespond(http.StatusForbidden, "Missing CSRF token")
		return sh
	}

	if len(sh.claims.SessionID) < 3 {
		sh.failAndRespond(http.StatusInternalServerError, "Invalid session ID")
		return sh
	}

	expectedToken := sh.claims.SessionID[len(sh.claims.SessionID)-3:]
	if csrfToken != expectedToken {
		// The expected value is deliberately NOT logged. It is a slice of the
		// session ID, so logging it hands anyone with read access to the logs
		// both the answer to this check and a piece of the session itself.
		// The received value is enough to tell a stale tab from an attack.
		sh.appServices.Logger.Warn(
			"CSM0",
			"CSRF token mismatch - received: "+csrfToken,
		)
		sh.failAndRespond(http.StatusForbidden, "CSRF validation failed")
		return sh
	}

	return sh
}

// CheckAuthorization checks if the user has the required grant
func (sh *AppHelper) CheckAuthorization(requiredGrantID int) *AppHelper {
	sh.authChecked = true
	if sh.blocked {
		return sh
	}

	if sh.claims == nil {
		sh.failAndRespond(http.StatusInternalServerError,
			"Authorization check requires authentication")
		return sh
	}

	if requiredGrantID > 0 {
		rbac := auth.NewRBACManager(sh.appServices.Logger, sh.username)
		if !rbac.HasGrant(
			sh.claims,
			requiredGrantID,
			sh.claims.SecurityContext,
		) {
			sh.appServices.Logger.Warn("ATF0",
				fmt.Sprintf("Authorization failed - required grant: %d, "+
					"security context: %d",
					requiredGrantID, sh.claims.SecurityContext))
			shortCode := model.GrantIDToShortCodeMap[requiredGrantID]
			if shortCode == "" {
				shortCode = fmt.Sprintf("%d", requiredGrantID)
			}
			sh.failAndRespond(
				http.StatusForbidden,
				fmt.Sprintf(
					"Permission denied (grant \"%s\" required)",
					shortCode,
				),
			)
			return sh
		}
	}

	return sh
}

// CheckTargetContextAuthorization checks authorization
// for a specific target context
func (sh *AppHelper) CheckTargetContextAuthorization(
	requiredGrantID int,
	targetSecurityContext int,
) bool {
	if sh.blocked || requiredGrantID == 0 {
		return true
	}

	if sh.claims == nil {
		sh.failAndRespond(http.StatusInternalServerError,
			"Target context authorization requires authentication")
		return false
	}

	rbac := auth.NewRBACManager(sh.appServices.Logger, sh.username)
	if !rbac.HasGrant(sh.claims, requiredGrantID, targetSecurityContext) {
		sh.appServices.Logger.Warn(
			"TAF0",
			fmt.Sprintf(
				"Target context authorization failed - required grant: %d, "+
					"target context: %d",
				requiredGrantID,
				targetSecurityContext,
			),
		)
		sh.failAndRespond(http.StatusForbidden,
			fmt.Sprintf("Permission denied (grant %d required in context %d)",
				requiredGrantID, targetSecurityContext))
		return false
	}

	return true
}

// CheckAllChecksComplete verifies all security checks have been performed
func (sh *AppHelper) CheckAllChecksComplete() bool {
	if sh.blocked {
		return false
	}

	if !sh.jwtChecked {
		sh.failAndRespond(
			http.StatusInternalServerError,
			"JWT check not completed",
		)
		return false
	}

	if !sh.rateChecked {
		sh.failAndRespond(http.StatusInternalServerError,
			"Rate limit check not completed")
		return false
	}

	if !sh.userChecked {
		sh.failAndRespond(http.StatusInternalServerError,
			"User block check not completed")
		return false
	}

	if !sh.csrfChecked {
		sh.failAndRespond(http.StatusInternalServerError,
			"CSRF check not completed")
		return false
	}

	if !sh.authChecked {
		sh.failAndRespond(http.StatusInternalServerError,
			"Authorization check not completed")
		return false
	}

	return true
}

// failAndRespond sends an error response and marks processing as blocked
func (sh *AppHelper) failAndRespond(statusCode int, reason string) {
	if sh.blocked {
		return
	}

	sh.blocked = true
	sh.WriteError(sh.w, statusCode, reason, "SECF", nil)
}
