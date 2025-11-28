package app_lib

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// SecurityHelper encapsulates request-specific security checks and state
type SecurityHelper struct {
	r           *http.Request
	w           http.ResponseWriter
	shrmplKV    *ShrmplKVClient
	config      *Config
	logger      LoggerInterface
	claims      *Claims
	ctx         context.Context
	blocked     bool
	jwtChecked  bool
	rateChecked bool
	userChecked bool
	csrfChecked bool
	authChecked bool
}

// NewSecurityHelper creates a new security helper
func NewSecurityHelper(w http.ResponseWriter, r *http.Request, shrmplKV *ShrmplKVClient, config *Config, logger LoggerInterface) *SecurityHelper {
	return &SecurityHelper{
		r:        r,
		w:        w,
		shrmplKV: shrmplKV,
		config:   config,
		logger:   logger.With("component", "SecurityHelper"),
	}
}

// SkipJWT skips JWT validation
func (sh *SecurityHelper) SkipJWT() *SecurityHelper {
	sh.jwtChecked = true
	return sh
}

// SkipRateLimit skips rate limit check
func (sh *SecurityHelper) SkipRateLimit() *SecurityHelper {
	sh.rateChecked = true
	return sh
}

// SkipUserBlock skips user block check
func (sh *SecurityHelper) SkipUserBlock() *SecurityHelper {
	sh.userChecked = true
	return sh
}

// SkipCSRF skips CSRF check
func (sh *SecurityHelper) SkipCSRF() *SecurityHelper {
	sh.csrfChecked = true
	return sh
}

// SkipAuthorization skips authorization check
func (sh *SecurityHelper) SkipAuthorization() *SecurityHelper {
	sh.authChecked = true
	return sh
}

// GetClaims returns the validated JWT claims
func (sh *SecurityHelper) GetClaims() *Claims {
	return sh.claims
}

// GetContext returns the current context
func (sh *SecurityHelper) GetContext() context.Context {
	return sh.ctx
}

// CanProceed returns true if processing can continue
func (sh *SecurityHelper) CanProceed() bool {
	return !sh.blocked
}

// CheckJWT validates the JWT token
func (sh *SecurityHelper) CheckJWT() *SecurityHelper {
	sh.jwtChecked = true
	if sh.blocked {
		return sh
	}

	authHeader := sh.r.Header.Get("Authorization")
	if authHeader == "" {
		sh.failAndRespond(http.StatusUnauthorized, "Missing authorization header")
		return sh
	}

	// Extract token from "Bearer <token>" format
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		sh.failAndRespond(http.StatusUnauthorized, "Invalid authorization header format")
		return sh
	}

	tokenString := parts[1]
	claims, err := ValidateJWT(tokenString, sh.config.JWTSecret)
	if err != nil {
		sh.logger.Warn("JWT validation failed: %s", err.Error())
		sh.failAndRespond(http.StatusUnauthorized, "Invalid authentication token")
		return sh
	}

	sh.claims = claims
	sh.ctx = context.WithValue(sh.ctx, "claims", claims)
	sh.logger = sh.logger.With("username", claims.Subject, "user_id", claims.UserID)

	return sh
}

// CheckRateLimit checks request rate limits
func (sh *SecurityHelper) CheckRateLimit() *SecurityHelper {
	sh.rateChecked = true
	if sh.blocked {
		return sh
	}

	var limit int
	var keyIdentifier string

	if sh.claims != nil {
		limit = sh.config.MaxUserReqPerMin
		keyIdentifier = sh.claims.Subject
	} else {
		limit = sh.config.MaxUnauthReqPerMin
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

	count, err := sh.shrmplKV.Incr(shrmplKey, "1m")
	if err != nil {
		sh.logger.Error("Rate limit check failed: %s", err.Error())
		// Fail open on shrmpl-kv errors
		return sh
	}

	if count > limit {
		sh.logger.Warn("Rate limit exceeded - key: %s, count: %d, limit: %d", shrmplKey, count, limit)
		sh.failAndRespond(http.StatusTooManyRequests, "Rate limit exceeded")
		return sh
	}

	sh.logger.Debug("Rate limit check passed - key: %s, count: %d, limit: %d", shrmplKey, count, limit)
	return sh
}

// CheckUserBlocked checks if the user is blocked
func (sh *SecurityHelper) CheckUserBlocked() *SecurityHelper {
	sh.userChecked = true
	if sh.blocked || sh.claims == nil {
		return sh
	}

	blockKey := fmt.Sprintf("USER_%s_BLOCK", sh.claims.Subject)
	_, err := sh.shrmplKV.Get(blockKey)
	if err == nil {
		// Key exists, user is blocked
		sh.logger.Warn("User is blocked: %s", sh.claims.Subject)
		sh.failAndRespond(http.StatusForbidden, "User account is blocked")
		return sh
	}

	return sh
}

// CheckCSRF validates the CSRF token
func (sh *SecurityHelper) CheckCSRF() *SecurityHelper {
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
		sh.failAndRespond(http.StatusInternalServerError, "CSRF check requires authenticated user")
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
		sh.logger.Warn("CSRF token mismatch - received: %s, expected: %s", csrfToken, expectedToken)
		sh.failAndRespond(http.StatusForbidden, "CSRF validation failed")
		return sh
	}

	return sh
}

// CheckAuthorization checks if the user has the required grant
func (sh *SecurityHelper) CheckAuthorization(requiredGrantID int) *SecurityHelper {
	sh.authChecked = true
	if sh.blocked {
		return sh
	}

	if sh.claims == nil {
		sh.failAndRespond(http.StatusInternalServerError, "Authorization check requires authentication")
		return sh
	}

	if requiredGrantID > 0 {
		rbac := NewRBACManager(sh.logger)
		if !rbac.HasGrant(sh.claims, requiredGrantID, sh.claims.SecurityContext) {
			sh.logger.Warn("Authorization failed - required grant: %d, security context: %d", requiredGrantID, sh.claims.SecurityContext)
			sh.failAndRespond(http.StatusForbidden, fmt.Sprintf("Permission denied (grant %d required)", requiredGrantID))
			return sh
		}
	}

	return sh
}

// CheckTargetContextAuthorization checks authorization for a specific target context
func (sh *SecurityHelper) CheckTargetContextAuthorization(requiredGrantID int, targetSecurityContext int) bool {
	if sh.blocked || requiredGrantID == 0 {
		return true
	}

	if sh.claims == nil {
		sh.failAndRespond(http.StatusInternalServerError, "Target context authorization requires authentication")
		return false
	}

	rbac := NewRBACManager(sh.logger)
	if !rbac.HasGrant(sh.claims, requiredGrantID, targetSecurityContext) {
		sh.logger.Warn("Target context authorization failed - required grant: %d, target context: %d", requiredGrantID, targetSecurityContext)
		sh.failAndRespond(http.StatusForbidden, fmt.Sprintf("Permission denied (grant %d required in context %d)", requiredGrantID, targetSecurityContext))
		return false
	}

	return true
}

// CheckAllChecksComplete verifies all security checks have been performed
func (sh *SecurityHelper) CheckAllChecksComplete() bool {
	if sh.blocked {
		return false
	}

	if !sh.jwtChecked {
		sh.failAndRespond(http.StatusInternalServerError, "JWT check not completed")
		return false
	}

	if !sh.rateChecked {
		sh.failAndRespond(http.StatusInternalServerError, "Rate limit check not completed")
		return false
	}

	if !sh.userChecked {
		sh.failAndRespond(http.StatusInternalServerError, "User block check not completed")
		return false
	}

	if !sh.csrfChecked {
		sh.failAndRespond(http.StatusInternalServerError, "CSRF check not completed")
		return false
	}

	if !sh.authChecked {
		sh.failAndRespond(http.StatusInternalServerError, "Authorization check not completed")
		return false
	}

	return true
}

// failAndRespond sends an error response and marks processing as blocked
func (sh *SecurityHelper) failAndRespond(statusCode int, reason string) {
	if sh.blocked {
		return
	}

	sh.blocked = true
	sh.logger.Warn("Security check failed - status: %d, reason: %s", statusCode, reason)

	http.Error(sh.w, reason, statusCode)
}
