package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/logging"
	"github.com/wbmartin/starter-api/internal/model"
)

// Every handler runs its own security checks, explicitly, in its own body:
//
//	appHelper := security.NewAppHelper(w, r, h.appsrv, "RetrieveExample")
//	appHelper.CheckJWT()
//	appHelper.CheckRateLimitAndUserBlock()
//	appHelper.CheckCSRF()
//	appHelper.CheckAuthorization(model.GrantExampleRead)
//	if !appHelper.CheckAllChecksComplete() { return }
//
// An endpoint that needs less writes the difference with Skip* calls in the
// same place, so an auditor reads the whole policy in the handler. This is
// deliberate and not to be replaced by middleware or a route wrapper
// (Brandon; doc/105 "Security checks"). CheckAllChecksComplete
// refuses to proceed unless all five were either checked or skipped.

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
	ah.appServices.Logger.Debug("APP_HELPER_INIT", "request entered "+helperLocation, "handler", helperLocation)
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
	data any,
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

	// A valid signature is not enough: the session may have been ended or
	// the login's tokens revoked (session_revocation.go). An unreadable KV
	// fails closed.
	revoked, err := sessionRevoked(sh.appServices, claims)
	if err != nil {
		sh.appServices.Logger.Error(
			"JWR0",
			"Session revocation check failed, failing closed: "+err.Error(),
		)
		sh.failAndRespond(
			http.StatusInternalServerError,
			"Session check failed",
		)
		return sh
	}
	if revoked {
		sh.appServices.Logger.Warn(
			"JWR1",
			"Revoked session refused: "+claims.Subject,
		)
		sh.failAndRespond(
			http.StatusUnauthorized,
			"Session has ended; sign in again",
		)
		return sh
	}

	sh.claims = claims
	sh.ctx = context.WithValue(sh.r.Context(), claimsKey, claims)
	sh.username = claims.Subject

	return sh
}

// ClientIP returns the caller's IP. It keys anonymous rate limiting and
// login lockout, so both paths agree.
//
// trustedProxyCount is how many proxies in front of the API append to
// X-Forwarded-For (TRUSTED_PROXY_COUNT). Each appends the address it saw,
// so only the right-hand entries are trustworthy; everything to their left
// is whatever the client sent. 0 ignores the header and uses RemoteAddr.
// N takes the entry N hops from the right (the one the outermost trusted
// proxy wrote), falling back to RemoteAddr when the header is shorter or
// that entry is not an address.
func ClientIP(r *http.Request, trustedProxyCount int) string {
	if trustedProxyCount > 0 {
		var hopList []string
		for _, value := range r.Header.Values("X-Forwarded-For") {
			hopList = append(hopList, strings.Split(value, ",")...)
		}
		if len(hopList) >= trustedProxyCount {
			hop := hopList[len(hopList)-trustedProxyCount]
			if ip := parseHostIP(hop); ip != "" {
				return ip
			}
		}
	}
	if ip := parseHostIP(r.RemoteAddr); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

// anonRateIdentifier is the rate-limit key identifier for an anonymous
// caller: "ip:" plus ClientIP. The prefix keeps anonymous buckets out of the
// signed-in key space, where the identifier is the user name, so a user
// named "10.0.0.5" never shares a bucket with that address.
func anonRateIdentifier(r *http.Request, trustedProxyCount int) string {
	return "ip:" + ClientIP(r, trustedProxyCount)
}

// parseHostIP returns the IP in "ip", "ip:port", "[v6]" or "[v6]:port", in
// canonical form, or "" when there is none.
func parseHostIP(value string) string {
	host := strings.TrimSpace(value)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}
	return ip.String()
}

// rateWindow is the sliding window the rate limit counts in: this clock
// minute's bucket plus the previous minute's, weighted by how much of that
// minute still lies inside the last 60 seconds. Fixed per-minute buckets
// let a client spend its whole allowance at 12:00:59 and a fresh one at
// 12:01:00 -- twice the limit inside a second. Buckets are keyed by
// absolute minute (Unix()/60), so a key never comes round again.
type rateWindow struct {
	currentKey     string
	previousKey    string
	previousWeight float64
}

// rateBucketTTL keeps a bucket alive through the minute after its own, when
// it is the previous one.
const rateBucketTTL = "2min"

// newRateWindow is the window for identifier (a user name, or
// anonRateIdentifier) at now. now is a parameter so the test can stand on
// a minute boundary.
func newRateWindow(identifier string, now time.Time) rateWindow {
	minute := now.Unix() / 60
	elapsed := float64(now.Unix()%60) + float64(now.Nanosecond())/1e9
	return rateWindow{
		currentKey:     fmt.Sprintf("RPM*%s*%d", identifier, minute),
		previousKey:    fmt.Sprintf("RPM*%s*%d", identifier, minute-1),
		previousWeight: 1 - elapsed/60,
	}
}

// count is the window's estimate of requests in the last 60 seconds: the
// current bucket (just incremented) plus the previous one, weighted.
// previous is that bucket's stored value, "" when it is absent. A value
// that is not a count is an error, and the caller fails closed.
func (rw rateWindow) count(current int, previous string) (int, error) {
	if previous == "" {
		return current, nil
	}
	previousCount, err := strconv.Atoi(previous)
	if err != nil {
		return 0, fmt.Errorf("bucket %s holds %q, not a count",
			rw.previousKey, previous)
	}
	return current + int(float64(previousCount)*rw.previousWeight), nil
}

// incrementAndCount adds this request to the current bucket and returns
// the window's count. Any KV error is returned for the caller to fail
// closed on.
func (rw rateWindow) incrementAndCount(appServices *app.Services) (
	int, error,
) {
	current, err := appServices.KV.Incr(rw.currentKey, rateBucketTTL)
	if err != nil {
		return 0, err
	}
	previous, err := appServices.KV.Get(rw.previousKey)
	if err != nil {
		return 0, err
	}
	return rw.count(current, previous)
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
		keyIdentifier = anonRateIdentifier(
			sh.r, sh.appServices.Config.TrustedProxyCount,
		)
	}

	window := newRateWindow(keyIdentifier, time.Now())
	rateKey := window.currentKey

	count, err := window.incrementAndCount(sh.appServices)
	if err != nil {
		// The store is in-process: an error here is a bug (closed store,
		// oversized key, a bucket that is not a count), not an outage.
		// Fail closed.
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

	var keyIdentifier string
	var limit int
	if sh.claims != nil {
		keyIdentifier = sh.claims.Subject
		limit = sh.appServices.Config.MaxUserReqPerMin
	} else {
		keyIdentifier = anonRateIdentifier(
			sh.r, sh.appServices.Config.TrustedProxyCount,
		)
		limit = sh.appServices.Config.MaxUnauthReqPerMin
	}

	window := newRateWindow(keyIdentifier, time.Now())
	rateKey := window.currentKey

	rateCount, err := window.incrementAndCount(sh.appServices)
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
		sh.refuseCSRF("Missing CSRF token")
		return sh
	}

	if len(sh.claims.SessionID) < 3 {
		sh.failAndRespond(http.StatusInternalServerError, "Invalid session ID")
		return sh
	}

	expectedToken := sh.claims.SessionID[len(sh.claims.SessionID)-3:]
	if csrfToken != expectedToken {
		// Neither token is logged. Both are slices of a session ID (the
		// received one from a stale tab is an earlier session's), so logging
		// either hands anyone with read access to the logs a piece of a
		// session.
		sh.appServices.Logger.Warn("CSM0", "CSRF token mismatch")
		sh.refuseCSRF("CSRF validation failed")
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

// refuseCSRF answers 403 with "code": "CSRF" beside the message: the web
// client resyncs its token once on that code, never on the message text.
func (sh *AppHelper) refuseCSRF(reason string) {
	if sh.blocked {
		return
	}

	sh.blocked = true
	sh.appServices.Logger.Error("SECF", reason, "username", sh.username)
	sh.WriteJSON(sh.w, http.StatusForbidden, map[string]string{
		"error": reason,
		"code":  "CSRF",
	})
}
