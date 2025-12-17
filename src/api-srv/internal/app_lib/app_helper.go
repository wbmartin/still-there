package app_lib

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const claimsKey contextKey = "claims"

// AppHelper encapsulates request-specific security checks and state
type AppHelper struct {
	r           *http.Request
	w           http.ResponseWriter
	appServices *AppServices
	claims      *Claims
	ctx         context.Context
	username    string
	blocked     bool
	jwtChecked  bool
	rateChecked bool
	userChecked bool
	csrfChecked bool
	authChecked bool
}

// NewAppHelper creates a new app helper
func NewAppHelper(
	w http.ResponseWriter,
	r *http.Request,
	appServices *AppServices,
) *AppHelper {
	return &AppHelper{
		r:           r,
		w:           w,
		appServices: appServices,
		username:    "preauth",
	}
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
func (sh *AppHelper) GetClaims() *Claims {
	return sh.claims
}

// GetUsername returns the username for logging
// (preauth or authenticated subject)
func (ah *AppHelper) GetUsername() string {
	return ah.username
}

// Debug logs at debug level with automatic username attachment
func (ah *AppHelper) Debug(code, message string) {
	ah.appServices.Logger.Debug(code, message, "username", ah.username)
}

// Info logs at info level with automatic username attachment
func (ah *AppHelper) Info(code, message string) {
	ah.appServices.Logger.Info(code, message, "username", ah.username)
}

// Warn logs at warn level with automatic username attachment
func (ah *AppHelper) Warn(code, message string) {
	ah.appServices.Logger.Warn(code, message, "username", ah.username)
}

// Error logs at error level with automatic username attachment
func (ah *AppHelper) Error(code, message string) {
	ah.appServices.Logger.Error(code, message, "username", ah.username)
}

// ErrorWithCallerSkip logs at error level with custom caller skip
func (ah *AppHelper) ErrorWithCallerSkip(code, message string, skip int) {
	ah.appServices.Logger.ErrorWithCallerSkip(code, message, skip, "username", ah.username)
}

// writeJSON writes a JSON response
func writeJSON(
	w http.ResponseWriter,
	status int,
	data interface{},
	logger ThisAppLoggerInterface,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("JEN0", "Failed to encode JSON response: "+err.Error())
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
	ah.appServices.Logger.ErrorWithCallerSkip(code, internalMsg, 3, "username", ah.username)
	response := map[string]string{"error": clientMsg}
	writeJSON(w, status, response, ah.appServices.Logger)
}

// GetContext returns the current context
func (sh *AppHelper) GetContext() context.Context {
	return sh.ctx
}

// CanProceed returns true if processing can continue
func (sh *AppHelper) CanProceed() bool {
	return !sh.blocked
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
	claims, err := ValidateJWT(tokenString, sh.appServices.Config.JWTSecret)
	if err != nil {
		sh.appServices.Logger.Warn("JWF0", "JWT validation failed: "+err.Error())
		sh.failAndRespond(http.StatusUnauthorized, "Invalid authentication token")
		return sh
	}

	sh.claims = claims
	sh.ctx = context.WithValue(sh.r.Context(), claimsKey, claims)
	sh.username = claims.Subject

	return sh
}

// CheckRateLimit checks request rate limits
func (sh *AppHelper) CheckRateLimit() *AppHelper {
	sh.rateChecked = true
	if sh.blocked {
		return sh
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
		// Get client IP
		ip, _, err := net.SplitHostPort(sh.r.RemoteAddr)
		if err != nil {
			ip = sh.r.RemoteAddr
		}
		// Check X-Forwarded-For header
		if forwardedFor := sh.r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			parts := strings.Split(forwardedFor, ",")
			if clientIP := strings.TrimSpace(parts[0]); clientIP != "" {
				ip = clientIP
			}
		}
		keyIdentifier = ip
	}

	currentMinute := time.Now().Minute()
	shrmplKey := fmt.Sprintf("RPM*%s*%d", keyIdentifier, currentMinute)

	count, err := sh.appServices.KV.Incr(shrmplKey, "1min")
	if err != nil {
		sh.appServices.Logger.Info("RLF0",
			"Rate limit check encountered KV error, failing open ("+
				shrmplKey+") "+err.Error())
		// Fail open on shrmpl-kv errors
		return sh
	}

	if count > limit {
		sh.appServices.Logger.Warn("RLE0",
			fmt.Sprintf("Rate limit exceeded - key: %s, count: %d, limit: %d",
				shrmplKey, count, limit))
		sh.failAndRespond(http.StatusTooManyRequests, "Rate limit exceeded")
		return sh
	}

	sh.appServices.Logger.Debug("RLP0",
		fmt.Sprintf("Rate limit check passed - key: %s, count: %d, limit: %d",
			shrmplKey, count, limit))
	return sh
}

// CheckRateLimitAndUserBlock performs rate limit and user block checks
// in a single batch
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

	// Build batch commands
	var commands []string
	var rateKey string
	if sh.claims != nil {
		rateKey = fmt.Sprintf("RPM*%s*%d", sh.claims.Subject, currentMinute)
	} else {
		rateKey = fmt.Sprintf("RPM*%s*%d", "anon", currentMinute)
	}
	commands = append(commands, "INCR "+rateKey+" 1min")

	var blockedKey string
	if sh.claims != nil {
		blockedKey = "BLOCKED*" + sh.claims.Subject
		commands = append(commands, "GET "+blockedKey)
	}

	// Execute batch
	results, err := sh.appServices.KV.Batch(commands)
	if err != nil {
		sh.appServices.Logger.Info("RLF0",
			"Batch check failed, failing open: "+err.Error())
		// Fail open on errors
		return sh
	}

	// Parse results
	rateCount, err := strconv.Atoi(results[0])
	if err != nil {
		sh.appServices.Logger.Warn("RLF0", "Invalid rate count: "+results[0])
		return sh
	}

	limit := 100 // Example limit
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
	// Check block if applicable
	if len(results) > 1 {
		blockedValue := results[1]
		if blockedValue != "" && blockedValue != "*KEY NOT FOUND*" {
			sh.appServices.Logger.Warn("UBK0", "User is blocked: "+sh.claims.Subject)
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
		sh.appServices.Logger.Warn("UBK0",
			"Failed to check blocked status: "+err.Error())
		return sh
	}

	if blockedValue != "" && blockedValue != "*KEY NOT FOUND*" {
		// Key exists, user is blocked
		sh.appServices.Logger.Warn("UBK0", "User is blocked: "+sh.claims.Subject)
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
		sh.appServices.Logger.Warn("CSM0",
			"CSRF token mismatch - received: "+csrfToken+", expected: "+expectedToken)
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
		rbac := NewRBACManager(sh.appServices.Logger, sh.username)
		if !rbac.HasGrant(sh.claims, requiredGrantID, sh.claims.SecurityContext) {
			sh.appServices.Logger.Warn("ATF0",
				fmt.Sprintf("Authorization failed - required grant: %d, "+
					"security context: %d",
					requiredGrantID, sh.claims.SecurityContext))
			sh.failAndRespond(http.StatusForbidden,
				fmt.Sprintf("Permission denied (grant %d required)", requiredGrantID))
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

	rbac := NewRBACManager(sh.appServices.Logger, sh.username)
	if !rbac.HasGrant(sh.claims, requiredGrantID, targetSecurityContext) {
		sh.appServices.Logger.Warn("TAF0",
			fmt.Sprintf("Target context authorization failed - required grant: %d, "+
				"target context: %d", requiredGrantID, targetSecurityContext))
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
		sh.failAndRespond(http.StatusInternalServerError, "JWT check not completed")
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
	sh.appServices.Logger.Warn("CHF0",
		fmt.Sprintf("Security check failed - status: %d, reason: %s",
			statusCode, reason))

	http.Error(sh.w, reason, statusCode)
}
