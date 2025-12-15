<role>
Act as a senior software engineer, focused on value and simple elegance. Optimize for readability, maintainability, and then speed. Respond like you are walking another senior engineer through the initial setup of this starter project.  I want you to understand the entirety of the requirements, but I'll ask you to only generate parts of the output at a time.
</role>

<context>
The following assignment describes a starter project for a production ready api server in golang that I’d like you to create. The server will start with standard security features and an endpoint for adding, updating, retrieving a text note.

- When I say “application server” or “api server”, I’m referring to the golang code I'm asking you to write.
- When I say “handler” I’m referring to the code that implements the published API endpoint.
- When I say “CRUD” I’m referring to the Create, Report, Update, Delete functionality.
- In general, honor the Don't Repeat Yourself (DRY) principle unless it makes the code significantly easier to follow.
- Don’t simplify code, take shortcuts, or omit code for brevity because this is a starter project template, not a teaching template or example. If I tell you it’s okay to simplify a point for efficiency,treat that as an exception.
- I understand generating the complete code for this entire starter project might result in a very long response. However, please prioritize completeness over brevity as instructed. If you anticipate exceeding response length limits, please state that clearly and perhaps generate the core structure and most critical handlers first, explicitly noting what remains, rather than silently omitting required code like the health check.
- The server will be equipped to handle multiple tenants so the security boundaries will be built in from the start (security_context).
- The code you generate should follow best golang practices, best security practices, and CLEAN code guidelines. If I ask you to do something you don’t understand or doesn’t make sense it is okay to ask for clarification. I will provide parameters for your output and there will be dependencies that you need to consider found throughout the following guidelines. If I specifically ask you to do something contrary to best or common practices, add a note stating that you honored my request, and what is sacrificed by deviating from best practice.
- The starter project will handle three layers: Web App, database and application server. The front end will be a web based application. Generate this code with an API-First mentality.
- In general, if the golang application server can perform a task the database server could do, the API server should do it. For example, when setting a create/update timestamp, the application server should set it so the database server doesn’t have to. Shifting this type of activity from the database to the application server level makes the application easier to scale. In general this approach minimized the work the database server has to do because the database is the hardest component to scale.
- The project should have minimal dependencies. Avoid the use of middleware and give preference to checking authentication, CSRF tokens, input validation, and authorization in each handler. It should use the latest golang version, preferably 1.24.
- Each tenant will have a security context that should be tracked in most database tables. The application will need a role based access control model per security context. (look in the examples/auth/auth.sql)
- For this project, the difference between a 404 (not found) and 409 (conflict) is  important so we need to do a follow-up query for updates that don't find a match to see if it gone or updated.
- For this project, the difference between a 409 (conflict) and 410 (gone) is important so so we need to do a follow-up query for updates that don't find a match to see if it gone or updated.
- This project will should be cloud agnostic. Don't include guidance for using AWS or Azure specific implementations. This code will run in a docker container on a virtual machine that likely has the database and shrmpl-kv (like redis) server on the same machine. Don't worry about container specific concerns.

This server will be built using the shrmpl-srv app infrastructure located here: https://github.com/wbmartin/shrmpl-srv-preview

</context>

<assignment>
Please create a starter project for an API application server in golang.

The project will have several security and core features and an example set of CRUD endpoints for a single textual note to illustrate how everything fits together.

The starter project will connect to a Postgres database and should use the pgx library.  The infrastructure to run the database will be outside this project. you should start with the create database sql which would go in a sql/000-initial setup.sql

The starter project will also connect to a shrmpl-kv (like redis) server for several security functions described below. Treat the shrmpl-kv connection like the database connection and pass it into every handler.

The starter project will have logging and use the shrmpl-log TCP Connection. There is a rust log client example in the ref folder that we need to turn into a golang module.

The starter project will use minimal state primarily for security purposes tracked in the shrmpl-kv server. The application will use a JWT token that carries security grant information with each request. The expected JWT claims structure should look like this:

```json
{
  "iss": "your-issuer-name", // Standard claim
  "sub": "username", // Standard claim (subject)
  "exp": 1678886400, // Standard claim (expiration)
  "nbf": 1678882800, // Standard claim (not before)
  "iat": 1678882800, // Standard claim (issued at)
  "jti": "unique-jwt-id", // Standard claim (JWT ID)
  "uid": 123, // Custom claim: User ID (app_user_id)
  "ses": "session-identifier", // Custom claim: Session ID; CSRF token will compare against last 3 chars
  "sctx": 1, // Custom claim: Current Security Context
  "grants": {
    // Custom claim: Grants map[security_context][]grant_id
    "1": [1001, 1002, 1003, 8001, 9001, 9002, 9003],
    "5": [1001, 8001]
  }
}
```

instead of using godotenv library to load a .env configuration, just write a simple function.  the .env file that looks like this:

```env
# START CONFIG FILE
SQL_DB_NAME=loc_starter_db
SQL_DB_HOST=192.168.72.1
SQL_DB_USER=app_svc_account
SQL_DB_PASSWORD=reallygoodpassword
SQL_DB_PORT=5432

ENV_SEC_APP_JWT_SECRET_KEY=highentropysecretkey
JWT_SESSION_EXPIRE_HOURS=72

MAX_USER_REQ_PER_MIN=200
MAX_UNAUTH_REQ_PER_MIN=20

SHRMPL_LOG_DEST=127.0.0.1:7379
SERVER_NAME=proto-go-app
# END CONFIG FILE
```

# Server Startup
When launching the server, it should accept a command line argument with the full path to configuration file it should load. When generating the configuration loading code (e.g., in config.go), please check the usage count for each environment variable name string.

# Entity fields
The following columns will be on most tables
* create_user_id
* create_ts
* update_user_id
* update_ts
* delete_user_id
* delete_ts

# Entity Handler Security Checks

Each handler should have a struct to check whether all the security points have been addressed or are being intentionally omitted.  E.g. There should be an object that actually checks the csrf against the csrfCheck complete.  It should also check to see if the user is authorized for the action, is below the rate limit, etc.  If the developer wants to skip a check they can call the csrfBypass method.  There should be a check before processing continues to confirm all checks have been marked completed or it should give a 500 error.  Checks to be completed:
* CheckUserBlocked
* CheckCSRF
* CheckRateLimit
* CheckJWT (confirm it is an authenticated request)
* CheckAuthorization

CheckCSRF: Each entity handler should check the validity of the CSRF token for Create, Update, Delete requests by comparing it to the last 3 characters of thesession id contained in the JWT token. The CSRF token will be sent in the X-CSRF-Token header. Don’t use a library for CSRF. The auth handler and GET requests will not need to check CSRF.

CheckRateLimit: Each authenticated handler should issue an INCR with the shrmpl-kv server to see if the allowed user requests per minute has been exceeded. Each non-authenticated handler (e.g. auth) should check to confirm the requests per minute from the IP address have not exceeded the limit. If the limit is exceeded, the api server should block the request and return a 429 error. The thresholds will be set in the config file. If it doesn’t exist, create the key to expire in one minute and call the key “RPM*”+username +”*”+ [current minute]. That way the shrmpl-kv server can purge the old Request Per Minute counts and the app server can just focus on incrementing and checking to be sure it is below the threshold.

CheckUserBlocked: Each authenticated handler should check shrmpl-kv to see if there is a key present for “USER\_”+username+”\_BLOCK” to ensure the user has not been blocked.

CheckAuthorization: If the user requests an action but does not have the appropriate user_grant_id, the api should respond with a 403 error. The handler should log all requests at INFO level after validating authentication.

# Entity Handler Processing

Each handler should expect a full json entity (including create_user_id, create_ts, update_user_id, update_ts) in the request payload, even though the values for fields like “create_ts” may be empty in the create handler. When the handler is adding, updating or deleting records, it should verify the “update_ts” field from the json request matches the update_ts in the database. Instead of using Postgres optimistic locking for SQL updates, simply compare the “updated_ts” provided in the json request with the value in the database using the SQL where clause.  If the update is not successful, the handler will have to execute a select query to see if the ID is present. If the ID is present is should issue a 409 conflict error, otherwise it should issue a 410 Gone

Each handler, not the database, will be responsible for setting the new “create_ts” and “update_ts” using the epoch time don’t expect this to happen via database triggers. The handler should ignore the create_user_id or update_user_id supplied in the json create request for the purpose of updating and creating.

Each handler should replace the create_user_id and update_user_id with the user_id from the JWT Token for create requests.

Each handler should replace the update_user_id and update_ts with the user_id from the JWT token for update requests.  It should not update the create_user_id or create_ts

Each handler should only update the create_user_id and create_ts when adding a record, not when updating.

The update handler should not need to retrieve the create_user_id or create_ts after updating. When returning the updated value in the json response body, use the create_user_id and create_ts from the request rather than asking the database.

Each handler should also have a DEBUG logging the payload received and response generated. When updating, do not worry about returning entity_id, create_ts, because the application-level handler sets all these. When inserting, only return the new entity id from the database instead of the full new record.The handler should ensure the security_context in the json request matches the value in the database for all updates, deletes. Updates should not be able to change the security context.

Each handler should use the security_context provided in the request json for create actions, but must verify the user can create records in that context using the JWT Cookie. Because the user may be authorized for the activity in some security contexts and not others, the handler should verify the requested action is allowed for the security_context specified in the entity json request.

Each delete handler should should respond with a 409 or 410 error message if the client requests to delete a record that is not present and does not match the provided update_ts.  If a delete fails, the handler must check if the id is present.  if it is present, respond with a 409 conflict.  If it is not present, respond with a 410 gone. The delete handler will only mark the delete_user_id and delete_ts based on the epoch time known to the app server.  Since this application will only do soft deletes, all selects, updates, and deletes will have to look for delete_user_id=null for it to be counted as present.

The starter project should have an authentication endpoint called “auth” that will receive a username and password and respond with a temporary token that can be used with a 2 factor code from an auth app to post and if validated receive a JWT token in the cookie that will be checked by all other endpoints before completing their action. In the response, the JWT token’s payload will also include a list of integers representing user grants by security_context. After the handler verifies the password matches the database record for the app_user, it will query the user_group table (described below) to get the list of user grants assigned by security_context. In addition to sending the user_grant_id list in the JWT Token, please also send in the auth endpoint’s json body response the list of short codes associated with the user by security_context (get the user’s group membership list from the user table, query the user_group table to get the user_grant_list, and then retrieve the relevant short codes from the user_grant table ). This will let the client application have the list of short codes for making UI decisions but it is important to separate them by security context since they may have different privileges in each security context. Don’t put the short codes in the JWT token, only put the user_grant_id list by security_context in the JWT token. The JWT token will also have the current security context defined in the user record’s security_context. The cookie containing the JWT token should be secure and HTTP ONLY. The Auth’s json body response should include “app_user_id”, “username”, “first_name”,”last_name”,”security_context”, “user_grant_short_code_list”, as dictionary of strings indexed by the security_context.

The project should have a starter set of endpoints/handlers for retrieve, add, update, and delete using a example fields described below. The handler will also need to check the JWT payload for the user_grant_id required to execute the rest of the code in the security_context it is requesting (i.e. RBAC per security_context).

the starter table should look like
  example_id serial
  example_text text
  example_int integer
  example_float
  example_int_array integer[]
  example_float_array integer[]
  security_context integer,
  create_ts numeric(17,7),
  create_user_id integer,
  update_ts numeric(17,7),
  update_user_id integer
  delete_ts numeric(17,7),
  delete_user_id integer


In another conversation we worked on a Security Helper approach that pulls all the Security functions into an object and tracks they have been completed or generates an error. Here is the output from that conversation that should be close to what we need for this project:

```golang
package app_lib

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/handler" // Assuming BaseHandler is accessible
)

// SecurityHelper encapsulates request-specific security checks and state.
type SecurityHelper struct {
	// --- Dependencies (Set at creation) ---
	r           *http.Request
	w           http.ResponseWriter
	baseHandler *handler.BaseHandler // To access ErrorResponse
	shrmpl_kv         *shrmpl-kv.Client
	cfg         *Config
	logger      *slog.Logger

	// --- State (Managed internally) ---
	claims            *model.Claims
	ctx               context.Context
	processingBlocked bool // If true, processing has stopped and response sent

	// --- Skip Configuration Flags ---
	skipJWTCheck       bool
	skipRateLimitCheck bool
	skipUserBlockCheck bool
	skipCSRFCheck      bool
	skipAuthzCheck     bool // For the initial CheckAuthorization

	// --- Check Completion Flags (Internal tracking/debugging) ---
	jwtCheckComplete       bool
	rateLimitCheckComplete bool
	userBlockCheckComplete bool
	csrfCheckComplete      bool
	authzCheckComplete     bool
}

// NewSecurityHelper creates a new instance for a given request.
// All checks are enabled by default. Use Skip* methods to disable specific checks.
func NewSecurityHelper(w http.ResponseWriter, r *http.Request, bh *handler.BaseHandler, rdb *shrmpl.Client, cfg *Config, logger *slog.Logger) *SecurityHelper {
	if w == nil || r == nil || bh == nil || rdb == nil || cfg == nil || logger == nil {
		tempLogger := logger
		if tempLogger == nil {
			tempLogger = slog.Default()
		}
		tempLogger.Error("Critical dependency missing during SecurityHelper creation")
		return &SecurityHelper{processingBlocked: true} // Mark as blocked immediately
	}
	return &SecurityHelper{
		r:           r,
		w:           w,
		baseHandler: bh,
		rdb:         rdb,
		cfg:         cfg,
		logger:      logger.With(slog.String("component", "SecurityHelper")),
		ctx:         r.Context(),
		// All skip flags default to false (meaning checks are performed by default)
	}
}

// --- Skip Configuration Methods ---

// SkipJWT configures the helper to skip the JWT validation check.
func (sh *SecurityHelper) SkipJWT() *SecurityHelper {
	sh.skipJWTCheck = true
	return sh
}

// SkipRateLimit configures the helper to skip the rate limit check.
func (sh *SecurityHelper) SkipRateLimit() *SecurityHelper {
	sh.skipRateLimitCheck = true
	return sh
}

// SkipUserBlock configures the helper to skip the user block check.
func (sh *SecurityHelper) SkipUserBlock() *SecurityHelper {
	sh.skipUserBlockCheck = true
	return sh
}

// SkipCSRF configures the helper to skip the CSRF validation check.
// Note: CSRF check is also automatically skipped for safe HTTP methods (GET, HEAD, OPTIONS, TRACE).
func (sh *SecurityHelper) SkipCSRF() *SecurityHelper {
	sh.skipCSRFCheck = true
	return sh
}

// SkipAuthorization configures the helper to skip the initial authorization check
// (verifying a grant against the active JWT context).
func (sh *SecurityHelper) SkipAuthorization() *SecurityHelper {
	sh.skipAuthzCheck = true
	return sh
}


// --- Getter Methods ---

// GetClaims returns the validated JWT claims, if available. Returns nil if checks failed before claims were parsed.
func (sh *SecurityHelper) GetClaims() *model.Claims {
	if sh.processingBlocked && sh.claims == nil {
		return nil
	}
	return sh.claims
}

// GetContext returns the current context (may have been updated with claims).
func (sh *SecurityHelper) GetContext() context.Context {
	return sh.ctx
}

// CanProceed returns true if no security check has failed (and sent a response).
// Handlers should check this and return immediately if it's false.
func (sh *SecurityHelper) CanProceed() bool {
	return !sh.processingBlocked
}

// --- Private Helper Methods ---

// failAndRespond stops processing, logs the error, and sends the HTTP error response.
func (sh *SecurityHelper) failAndRespond(statusCode int, reason string, details ...string) {
	if sh.processingBlocked {
		sh.logger.DebugContext(sh.ctx, "Processing already blocked, ignoring subsequent failure", slog.String("ignoredReason", reason))
		return
	}
	sh.processingBlocked = true
	// BaseHandler's errorResponse handles the logging.
	sh.baseHandler.ErrorResponse(sh.w, sh.r, statusCode, reason, details...)
}

// --- Public Check Methods (Return *SecurityHelper for chaining) ---

// CheckJWT validates the JWT unless SkipJWT() was called.
func (sh *SecurityHelper) CheckJWT() *SecurityHelper {
	sh.jwtCheckComplete = true // Mark attempt
	if sh.processingBlocked || sh.skipJWTCheck {
		sh.logger.DebugContext(sh.ctx, "Skipping JWT check (configured)")
		return sh
	}

	claims, err := ValidateJWT(sh.r, sh.cfg)
	if err != nil {
		errMsg := "Authentication failed"
		status := http.StatusUnauthorized
		logLevel := slog.LevelWarn
		if errors.Is(err, jwt.ErrTokenExpired) || strings.Contains(err.Error(), "missing session cookie") {
			errMsg = err.Error()
		} else {
			sh.logger.Log(sh.ctx, logLevel, "JWT validation failed internally", slog.String("error", err.Error()))
			errMsg = "Invalid authentication token"
		}
		sh.failAndRespond(status, errMsg)
		return sh
	}

	sh.claims = claims
	sh.ctx = ContextWithClaims(sh.ctx, claims)
	sh.logger = sh.logger.With(slog.String("username", claims.Username), slog.Int("userId", claims.UserID))
	sh.logger.DebugContext(sh.ctx, "JWT validated successfully")
	return sh
}

// CheckRateLimit checks request rate limits unless SkipRateLimit() was called.
func (sh *SecurityHelper) CheckRateLimit() *SecurityHelper {
	sh.rateLimitCheckComplete = true // Mark attempt
	if sh.processingBlocked || sh.skipRateLimitCheck {
		return sh
	}

	var limit int
	var keyIdentifier string

	if sh.claims != nil && sh.claims.Username != "" {
		limit = sh.cfg.MaxUserRequestsPerMinute
		keyIdentifier = sh.claims.Username
		sh.logger.DebugContext(sh.ctx, "Applying authenticated user rate limit", slog.Int("limit", limit), slog.String("identifier", keyIdentifier))
	} else {
		limit = sh.cfg.MaxUnauthRequestsPerMin
		ip, _, err := net.SplitHostPort(sh.r.RemoteAddr)
		if err != nil {
			ip = sh.r.RemoteAddr
			sh.logger.WarnContext(sh.ctx, "Could not parse IP from RemoteAddr, using full address for rate limiting", slog.String("remoteAddr", sh.r.RemoteAddr), slog.String("error", err.Error()))
		}
        forwardedFor := sh.r.Header.Get("X-Forwarded-For")
        if forwardedFor != "" {
            parts := strings.Split(forwardedFor, ",")
            clientIP := strings.TrimSpace(parts[0])
            if clientIP != "" { ip = clientIP }
        }
		keyIdentifier = ip
		sh.logger.DebugContext(sh.ctx, "Applying unauthenticated rate limit", slog.Int("limit", limit), slog.String("identifier", keyIdentifier))
	}

	currentMinute := time.Now().Minute()
	shrmplKVKey := fmt.Sprintf("RPM*%s*%d", keyIdentifier, currentMinute)

	count, err := sh.rdb.Incr(sh.ctx, shrmplKVKey).Result()
	if err != nil {
		sh.logger.ErrorContext(sh.ctx, "Failed to increment rate limit counter in shrmplKV", slog.String("key", shrmplKVKey), slog.String("error", err.Error()))
		sh.logger.ErrorContext(sh.ctx, "Failing open on rate limit check due to shrmplKV error")
        return sh // Fail open
	}

	if count == 1 {
		_ = sh.rdb.Expire(sh.ctx, shrmplKVKey, 1*time.Minute).Err() // Best effort expire
	}

	if count > int64(limit) {
		sh.logger.WarnContext(sh.ctx, "Rate limit exceeded", slog.String("key", shrmplKVKey), slog.Int64("count", count), slog.Int("limit", limit))
		sh.failAndRespond(http.StatusTooManyRequests, "Rate limit exceeded")
	} else {
		sh.logger.DebugContext(sh.ctx, "Rate limit check passed", slog.String("key", shrmplKVKey), slog.Int64("count", count), slog.Int("limit", limit))
	}
	return sh
}


// CheckUserBlocked checks if the authenticated user is blocked unless SkipUserBlock() was called.
func (sh *SecurityHelper) CheckUserBlocked() *SecurityHelper {
	sh.userBlockCheckComplete = true // Mark attempt
	if sh.processingBlocked || sh.skipUserBlockCheck {
		sh.logger.DebugContext(sh.ctx, "Skipping user block check (configured)")
		return sh
	}

	if sh.claims == nil || sh.claims.Username == "" {
		sh.logger.DebugContext(sh.ctx, "Skipping user block check (user not authenticated)")
		return sh
	}

	shrmplKVKey := fmt.Sprintf("USER_%s_BLOCK", sh.claims.Username)
	exists, err := sh.rdb.Exists(sh.ctx, shrmplKVKey).Result()
	if err != nil {
		sh.logger.ErrorContext(sh.ctx, "Failed to check user block status in shrmplKV", slog.String("key", shrmplKVKey), slog.String("error", err.Error()))
		sh.failAndRespond(http.StatusInternalServerError, "Failed to verify user status")
		return sh
	}

	if exists > 0 {
		sh.logger.WarnContext(sh.ctx, "User block check failed: User is blocked", slog.String("key", shrmplKVKey))
		sh.failAndRespond(http.StatusForbidden, "User account is blocked")
	} else {
		sh.logger.DebugContext(sh.ctx, "User block check passed", slog.String("key", shrmplKVKey))
	}
	return sh
}

// CheckCSRF validates the CSRF token unless SkipCSRF() was called or method is safe.
func (sh *SecurityHelper) CheckCSRF() *SecurityHelper {
	sh.csrfCheckComplete = true // Mark attempt
	if sh.processingBlocked ||sh.skipCSRFCheck {
		sh.logger.DebugContext(sh.ctx, "Skipping CSRF check (configured)")
		return sh
	}

	// Auto-skip for safe HTTP methods
	switch sh.r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		sh.logger.DebugContext(sh.ctx, "Skipping CSRF check (safe HTTP method)", slog.String("method", sh.r.Method))
		return sh
	}



	// CSRF check requires authenticated user with session ID
	if sh.claims == nil || sh.claims.SessionID == "" {
		sh.logger.WarnContext(sh.ctx, "CSRF check cannot be performed without authenticated user session")
        if !sh.jwtCheckComplete {
             sh.logger.ErrorContext(sh.ctx, "Logic error: CSRF check reached without prior JWT check for non-safe method")
             sh.failAndRespond(http.StatusInternalServerError, "Internal configuration error")
        } else if sh.claims == nil && !sh.skipJWTCheck {
             // If JWT wasn't skipped but claims are nil, JWT check must have failed.
             // failAndRespond was already called by CheckJWT.
             sh.logger.DebugContext(sh.ctx, "CSRF check skipped because prior JWT check failed")
        } else if sh.claims != nil && sh.claims.SessionID == "" {
             // Valid JWT but missing SessionID? Should not happen based on ValidateJWT logic.
             sh.logger.ErrorContext(sh.ctx, "CSRF check failed: Claims present but SessionID missing")
             sh.failAndRespond(http.StatusInternalServerError, "Internal security configuration error")
        }
		return sh
	}

	// Get token from header
	csrfToken := sh.r.Header.Get("X-CSRF-Token")
	if csrfToken == "" {
		sh.logger.WarnContext(sh.ctx, "CSRF check failed: Missing X-CSRF-Token header")
		sh.failAndRespond(http.StatusForbidden, "CSRF validation failed: Missing token")
		return sh
	}

	// Compare with last 3 chars of session ID from JWT
	sessionID := sh.claims.SessionID
	if len(sessionID) < 3 {
		sh.logger.ErrorContext(sh.ctx, "CSRF check cannot be performed: Session ID in JWT is too short", slog.Int("sessionIdLength", len(sessionID)))
		sh.failAndRespond(http.StatusInternalServerError, "Internal security configuration error")
		return sh
	}
	expectedToken := sessionID[len(sessionID)-3:]

	if csrfToken != expectedToken {
		sh.logger.WarnContext(sh.ctx, "CSRF check failed: Token mismatch", slog.String("receivedToken", csrfToken), slog.String("expectedSuffix", expectedToken))
		sh.failAndRespond(http.StatusForbidden, "CSRF validation failed: Token mismatch")
		return sh
	}

	sh.logger.DebugContext(sh.ctx, "CSRF check passed")
	return sh
}

// CheckAuthorization verifies authentication and optionally a grant against the *active JWT context*,
// unless SkipAuthorization() was called.
// Pass `requiredGrantID = 0` to only check authentication status.
func (sh *SecurityHelper) CheckAuthorization(requiredGrantID int) *SecurityHelper {
    sh.authzCheckComplete = true // Mark attempt
    if sh.processingBlocked || sh.skipAuthzCheck {
        return sh
    }

    // Check if authentication is required (implicitly, unless SkipJWT was also called)
    // and if the user is actually authenticated.
    if sh.claims == nil || sh.claims.Username == "" {
         // If JWT wasn't skipped, CheckJWT would have already failed and responded.
         // If JWT *was* skipped, but Authz *wasn't*, this is a logic error.
        if !sh.skipJWTCheck && sh.claims == nil {
             sh.logger.DebugContext(sh.ctx, "Authorization check skipped because prior JWT check failed")
        } else if sh.skipJWTCheck {
             // Logic error: Authorization requested but authentication was explicitly skipped.
             sh.logger.ErrorContext(sh.ctx, "Logic error: Authorization check requested, but JWT check was skipped")
             sh.failAndRespond(http.StatusInternalServerError, "Internal configuration error: Cannot authorize unauthenticated request")
        } else {
            // Should not happen if JWT succeeded but claims are nil.
            sh.logger.ErrorContext(sh.ctx, "Authorization check failed: Claims unavailable despite JWT check completion")
            sh.failAndRespond(http.StatusInternalServerError, "Internal error verifying user permissions")
        }
        return sh
    }

	// Authentication confirmed. Now check the specific grant if required.
	if requiredGrantID > 0 {
		activeContext := sh.claims.SecurityContext
		if !sh.hasGrant(requiredGrantID, activeContext) {
			sh.logger.WarnContext(sh.ctx, "Initial authorization failed",
				slog.Int("requiredGrant", requiredGrantID),
				slog.Int("activeContext", activeContext),
			)
			sh.failAndRespond(http.StatusForbidden, fmt.Sprintf("Permission denied (grant %d required in active context %d)", requiredGrantID, activeContext))
		} else {
			sh.logger.DebugContext(sh.ctx, "Initial authorization check passed", slog.Int("grantId", requiredGrantID), slog.Int("activeContext", activeContext))
		}
	} else {
		// requiredGrantID was 0, so only authentication was needed, which passed.
		sh.logger.DebugContext(sh.ctx, "Authorization check passed (authentication confirmed, no specific grant required)")
	}
    return sh
}


// CheckTargetContextAuthorization verifies a grant against a *specific target context*.
// This method remains unchanged as it's typically called after initial checks and payload decoding.
// Returns true if authorized, false otherwise (and sends response).
func (sh *SecurityHelper) CheckTargetContextAuthorization(requiredGrantID int, targetSecurityContext int) bool {

	if sh.processingBlocked || requiredGrantID == 0 {
		sh.logger.DebugContext(sh.ctx, "Skipping target context authorization check (grant ID is 0)")
		return true
	}

	if sh.claims == nil || sh.claims.Username == "" {
		sh.logger.ErrorContext(sh.ctx, "Attempted target context authorization check without authenticated user claims")
		sh.failAndRespond(http.StatusInternalServerError, "Internal error: authorization check state invalid")
		return false
	}

	if !sh.hasGrant(requiredGrantID, targetSecurityContext) {
		sh.logger.WarnContext(sh.ctx, "Target context authorization failed",
			slog.Int("requiredGrant", requiredGrantID),
			slog.Int("targetContext", targetSecurityContext),
			slog.Int("activeContext", sh.claims.SecurityContext),
		)
		sh.failAndRespond(http.StatusForbidden, fmt.Sprintf("Permission denied (grant %d required in target context %d)", requiredGrantID, targetSecurityContext))
		return false
	}

	sh.logger.DebugContext(sh.ctx, "Target context authorization check passed", slog.Int("grantId", requiredGrantID), slog.Int("targetContext", targetSecurityContext))
	return true
}


// hasGrant checks if the user's claims contain the required grant ID for the specified security context.
// Internal helper. Returns false if claims are nil or grants are missing for the context.
func (sh *SecurityHelper) hasGrant(requiredGrantID int, securityContext int) bool {
	if sh.claims == nil || sh.claims.Grants == nil {
		sh.logger.DebugContext(sh.ctx, "Cannot check grant: Claims or grants map is nil")
		return false
	}
	contextGrants, ok := sh.claims.Grants[securityContext]
	if !ok {
		sh.logger.DebugContext(sh.ctx, "No grants found in JWT for the specified security context", slog.Int("checkContext", securityContext))
		return false
	}
	for _, grantID := range contextGrants {
		if grantID == requiredGrantID {
			return true
		}
	}
	sh.logger.DebugContext(sh.ctx, "Required grant not found in user's JWT grants for the specified security context", slog.Int("requiredGrant", requiredGrantID), slog.Int("checkContext", securityContext))
	return false
}

// CheckUserHasContextAccess verifies the user has *any* grants defined for the target context.
// This method remains unchanged.
// Used for operations like switching context. Returns true if access exists, false otherwise.
func (sh *SecurityHelper) CheckUserHasContextAccess(targetSecurityContext int) bool {
    if sh.processingBlocked { return false }

    if sh.claims == nil || sh.claims.Grants == nil {
        sh.logger.ErrorContext(sh.ctx, "Attempted context access check without authenticated user claims or grants")
        sh.failAndRespond(http.StatusInternalServerError, "Internal error: authorization check state invalid")
        return false
    }

    contextGrants, ok := sh.claims.Grants[targetSecurityContext]
    if !ok || len(contextGrants) == 0 {
        sh.logger.WarnContext(sh.ctx, "Context access check failed: No grants defined for user in target context",
            slog.Int("targetContext", targetSecurityContext),
            slog.Int("activeContext", sh.claims.SecurityContext),
        )
        sh.failAndRespond(http.StatusForbidden, fmt.Sprintf("Access denied: No permissions configured for context %d", targetSecurityContext))
        return false
    }

    sh.logger.DebugContext(sh.ctx, "Context access check passed", slog.Int("targetContext", targetSecurityContext))
    return true
}

func(sh *SecurityHelper) CheckAllChecksComplete() bool{
    // all handlers should verify this is true before allowing a change to the database

    if !sh.processingBlocked {
        sh.failAndRespond(http.StatusForbidden, fmt.Sprintf("Previous Failure"))
        return false
    }
    if !sh.jwtCheckComplete {
        sh.failAndRespond(http.StatusForbidden, fmt.Sprintf("JWT not Checked"))
        return false
    }
	if !sh.rateLimitCheckComplete {
        sh.failAndRespond(http.StatusForbidden, fmt.Sprintf("Rate Limit Not Checked"))
        return false

    }
	if !sh.userBlockCheckComplete {
        sh.failAndRespond(http.StatusForbidden, fmt.Sprintf("User Block Not Checked"))
        return false

    }
	if !sh.csrfCheckComplete      {
        sh.failAndRespond(http.StatusForbidden, fmt.Sprintf("CSRF Not Checke"))
        return false

    }
	if !sh.authzCheckComplete     {
        sh.failAndRespond(http.StatusForbidden, fmt.Sprintf("Authorization Not Checked"))
        return false

    }
    return true
}

```

I expect you to make changes if required of the security helper, but I don't expect many. Please include it with your output so it is comprehensive of the files needed for the starter project.

Here is the standard logic sequence for an entity CRUD handler:

1. Log request start (INFO). (Performed By Security Helper)
2. Validate JWT from cookie (-> 401 if invalid).(Performed By Security Helper)
3. Extract claims.(Performed By Security Helper)
4. Check Rate Limit for authenticated user (-> 429 if exceeded).(Performed By Security Helper)
5. Check if user is blocked via shrmplKV (-> 403 if blocked).(Performed By Security Helper)
6. Check X-CSRF-Token header against JWT session ID (-> 403 if invalid).(Performed By Security Helper)
7. Check RBAC: User must have CHG*NOTE grant (ID 1003) for the securityContext specified \_in the request body* (-> 403 if denied).
8. Decode JSON request body into Note struct (-> 400 if invalid).
9. Log decoded payload (DEBUG).
10. Perform input validation on Note struct fields (-> 400 if invalid).
11. Get current time, generate new epoch timestamp (numeric 17,7).
12. Prepare Note struct for update: Keep NoteID, SecurityContext, NoteText from request. Set UpdateUserID from JWT claims. Set _new_ UpdateTs.
13. Call NoteRepository.UpdateNote, passing the Note struct (with new ts) and the _original_ UpdateTs from the request body (for optimistic lock check).
14. Handle Repository Response:
    - If ErrConflict (or 0 rows affected) -> 409 Conflict.
    - If other DB error -> 500 Internal Server Error.
    - If success: Proceed.
15. Construct successful response body: Use the Note struct passed to the repo (containing the new updateTs/updateUserId) but populate CreateTs/CreateUserId _from the request body_ (as per original requirement).
16. Log response generation (DEBUG).
17. Write JSON response (200 OK).
18. Log request end (INFO).

In the database, create an application user table should be named “app_user” and have fields username, pw_hash, first_name, last_name, email, and a security_context_list. The user table should also have a field called user_group_membership_list of type integer []. You may be tempted to track user groups by security context, but a simple list will suffice because user_group_id is globally unique across all contexts.

In the database, create a table called user_group which includes a field called user_grant_list of type integer []. The primary key of the user_group table should not be a serial because it will be set when creating groups and will not be sequential. The user_group_id will be unique across security_contexts. So the user may be an admin in security_context_1 and basic user in security_context 2. The basic user group in security_context 1 will be completely separate from the basic security group in security_context 2. The handler will need to ensure the security_context requested in the json request and the activity requested (e.g. update/delete) is granted for that security_context before allowing the action to occur.

In the database, create a table called user_grant which includes a field called user_grant_id, grant_short_code, grant_description. The user_grant will not have a security_context because it will be the same for all security_contexts,

In the database, create a user grant record for each entity / table’s add, change, get, delete handler. Create a default set of user grants in the database generation sql. For the starter project’s note example, the grants “short_code” will be “ADD_NOTE”, “CHG_NOTE”, “DEL_NOTE”, “GET_NOTE”. user_grant_id should start with user_grant_id 1000 to make organizing them easier. For non-standard entity user_grant_id’s like change OWN Password, start with 8000. Admin level privileges like creating users will start at 9000. So for the starter “note” entity these would look like 1001 (GET), 1002 (ADD),1003 (UPDATE), 1004 (DELETE).

Required User Grants:

- ID: 1001, ShortCode: GET_NOTE, Description: Retrieve notes
- ID: 1002, ShortCode: ADD_NOTE, Description: Add new notes
- ID: 1003, ShortCode: CHG_NOTE, Description: Change existing notes
- ID: 1004, ShortCode: DEL_NOTE, Description: Delete notes
- ID: 8001, ShortCode: CHG_OWN_PW, Description: Change own password
- ID: 9001, ShortCode: ADD_USER, Description: Add new users
- ID: 9002, ShortCode: BLOCK_USER, Description: Block user access
- ID: 9003, ShortCode: CHG_OTHER_USER_PW, Description: Reset another user password

In the database, Create user_group records for admin and basic groups in security_context 1.
In the database, Create an admin user with a high entropy password and note what it is in the code.

The application will have an enumeration for grant_short_codes to their corresponding user_grant_id for each entity because the client will send the user_grant_id they are entitled to in the JWT token, but the code should be able to check against the short code without looking in the database. Please hard code the ID to the short code corresponding with the IDs you generate for the grants to keep it simple. The developer will be responsible for ensuring the enumeration matches the database. Each table should have the Grants and mappings defined in the model file for that table that correlates 1001 to GrantUserAdd, following the template “Grant” + “Entity Name” + “Action”. These can be stored in the model file and should look like this snippet.

```golang
// GrantNoteGet is the user grant ID for retrieving notes.
const GrantNoteGet = 1001
// GrantNoteAdd is the user grant ID for adding notes.
const GrantNoteAdd = 1002
// GrantNoteChange is the user grant ID for changing notes.
const GrantNoteChange = 1003
// GrantNoteDelete is the user grant ID for deleting notes.
const GrantNoteDelete = 1004

// NoteGrants maps grant short codes to their IDs for the Note entity.
var NoteGrants = map[string]int{
	"GET_NOTE":   GrantNoteGet,
	"ADD_NOTE":   GrantNoteAdd,
	"CHG_NOTE":   GrantNoteChange,
	"DEL_NOTE":   GrantNoteDelete,
}


```

Regarding user grants:

1. Define entity-specific grant constants (e.g., const GrantNoteGet = 1001, const GrantNoteAdd = 1002) directly within their corresponding model files (e.g., in internal/model/note.go for Note-related grants, internal/model/app_user.go for User-related grants like GrantAddUser). Follow the example snippet provided previously for note.go.
2. Define application-wide or non-entity-specific grant constants (e.g., const GrantChangeOwnPassword = 8001) within the internal/model/user_grant.go file.
3. Create centralized maps AllKnownGrants map[string]int and GrantIDToShortCodeMap map[int]string within the internal/model/user_grant.go file.
4. Use an init() function inside internal/model/user_grant.go to populate both AllKnownGrants and GrantIDToShortCodeMap. This init() function should reference all grant constants, including those defined in the individual model files (like GrantNoteGet from note.go) and those defined locally (like GrantChangeOwnPassword). This ensures the maps are complete and act as the single source of truth for lookups, even though the constant definitions are distributed."

Regarding endpoints that should be created:
Create an endpoint for a basic health /heartbeat check.
Create an endpoint for adding new user accounts the endpoint should receive first name, last name, email address, username, password. The request will include the security*context for the user to be created in. The handler must confirm the user in the JWT (requesting the creation) has the “GrantUserAdd” grant for the security context of the requested user.
Create an endpoint that allows a user to change their password. The user must provide a valid JWT token, the current password, and the new password. The handler should verify the current password with the hash in the database before updating the user record.
Create an endpoint to logoff which will overwrite the JWT cookie.
Create an endpoint to change the security_context for the user in the JWT token. The cookie and response body should look like the auth endpoint, but accepts a targetSecurityContext body request parameter that must be tested against the original JWT to confirm they have grants in that security_context.
Create an endpoint to add a shrmplKV key of “USER*” + username + “\_BLOCK” with a value of true. This should be an admin function that will relay to all other handlers that the current username is not authorized for any activity and will require its own grant. The true value is just so that the value is not null. When the handler checks the presence of the key will indicate the user has been blocked.
Create an endpoint that allows an admin to reset another user’s password if they have the “CHG_OTHER_USER_PW” grant.

| Method | Path                          | Auth? | CSRF? | RBAC Grant                 | Key Params (Query/Body)                    | Success | Notes                          |
| ------ | ----------------------------- | ----- | ----- | -------------------------- | ------------------------------------------ | ------- | ------------------------------ |
| GET    | /health                       | No    | No    | N/A                        | N/A                                        | 200     | Basic health check             |
| POST   | /auth                         | No    | No    | N/A                        | Body: {username, password}                 | 200     | Sets cookie, returns user info |
| POST   | /logoff                       | Yes   | No    | N/A                        | N/A                                        | 200     | Clears cookie                  |
| POST   | /change-security-context      | Yes   | No    | N/A (implicit grant check) | Body: {targetSecurityContext}              | 200     | Updates cookie/context         |
| GET    | /note                         | Yes   | No    | GET_NOTE                   | Query: {noteId, securityContext}           | 200     |                                |
| POST   | /note                         | Yes   | Yes   | ADD_NOTE                   | Body: {Note object}                        | 201     |                                |
| PUT    | /note                         | Yes   | Yes   | CHG_NOTE                   | Body: {Note object}                        | 200     | Requires matching updateTs     |
| DELETE | /note                         | Yes   | Yes   | DEL_NOTE                   | Query: {noteId, securityContext, updateTs} | 200/204 | Requires matching updateTs     |
| POST   | /user                         | Yes   | Yes   | ADD_USER                   | Body: {User details}                       | 201     |                                |
| PUT    | /change-current-user-password | Yes   | Yes   | CHG_OWN_PW                 | Body: {currentPassword, newPassword}       | 200     | Verifies current PW            |
| POST   | /block-user                   | Yes   | Yes   | BLOCK_USER                 | Body: {username}                           | 200     | Admin action                   |
| PUT    | /reset-user-password          | Yes   | Yes   | CHG_OTHER_USER_PW          | Body: {username, newPassword}              | 200     | Admin action                   |

</assignment>

<output>

When generating the database sql, please consider these guidelines:
When creating the database tables, please use “entity_name_id” pattern to name primary keys where the table is “entity_name”. Table names should be singular names of the entity. All tables should have a “create_ts” and “update_ts” of type numeric(17,7) for time and a create_user_id and update_user_id integer. Each table should also have a “security_context” field of type integer to enable multi-tenant implementations.
Don’t create foreign keys / Postgres references within the database. That will be managed in the application layer.
Don’t create triggers for updating dates within the database. That will be managed by the application layer.
Don’t plan for a database migration tool to manage schema changes. That will be managed manually.
Database fields should be snake_case.
When generating SQL in a .sql file it is okay to consolidate multiple fields into single lines, but each keyword (e.g. select, from, where)should start a new line.

When generating SQL seed data (data.sql) that uses common calculated values like a timestamp:
Define the common values once in a single Common Table Expression (CTE) at the beginning of the script. (e.g., WITH common_vars AS (SELECT NOW() as ts, 0 as system_id)).
Use the INSERT INTO ... SELECT ... FROM common_vars_cte, (VALUES ...) AS data(...) format for inserting the data. This involves selecting values from both the CTE and the VALUES clause containing the row data.
Ensure you use standard SQL and avoid client-specific commands like psql's \set or \gset.

When generating the application handler code, consider these guidelines:
Do not try to generate update query SQL dynamically. The client is responsible for sending a completely updated version of the record.
Don’t look for entity ID’s in the URL path of POST, PUT requests, these will be in the json request payload.
Don’t look for entity ID’s in the URL path of GET, DELETE requests, these will be in the query string parameters.
Don’t create PATCH verb handlers, expect the client to send a full entity every time.
When creating or updating an entity record, return the full entity in the response payload if successful. For creating records you will need to get the created record id from the database using a returning statement in Postgres SQL.
Note: It is common to retrieve the create_ts, create_user_id, update_ts and update_user_id to populate the response payload. However, for this implementation the handler knows and sets these values, so it does not need to retrieve these values from the database after inserting or updating.
Don’t omit security_context for any table unless it’s specified to be unique across security_contexts.
Don’t created nested endpoints like “/user/password” for changing passwords, instead name the endpoint “/change-current-user-password”.
Don’t rely on database triggers in the handler.
When producing or receiving JSON for the response, use camelCase.
Place each model in a separate file. Don’t create separate structs for the entity and the entity request.
When generating SQL query in golang, separate each field onto a new line for both field names and values.
Show curl commands when describing how to test the initial build.
Generate a readme.md that highlights design decisions, places that require attention to start and instructions to setup the starter project.
When generating the go application code, generate the repository files that contain the SQL code for CRUD Operations. Please implement this using the Data Access Object (DAO) pattern with a struct named [TableName]DAO. This DAO should directly use pgx for PostgreSQL. This struct should have methods for interacting with the PostgreSQL database for the Note entity (Get, Create, Update, Delete) using pgx directly. I don't need a separate interface for the repository in this case since we are only using PostgreSQL.
When generating the go application code, generate the router code for all endpoints described.
When creating the project structure, it is common to create folders for auth, config, database, logging, brace, shrmplKV and security. However, if these folders only have one file in them, please put them all in a “applib” folder under the internal folder.

To reiterate, please generate the complete code for all specified components and handlers (including main.go, configuration, database setup, models, repositories, security helpers, RBAC logic, and handlers for /health, /auth, /note CRUD, /user management, /logoff, /change-security-context, /block-user, /reset-user-password, /change-current-user-password). Do not omit any code, even for simpler handlers like /health, for brevity. If you must omit code please list these omissions at the end so I can request them in a subsequent prompt.

My GitHub handle is “wbmartin”, use it when generating import statements.

Strictly prohibit the use of database DEFAULT clauses, triggers, or SQL functions for generating or managing any application data fields like create_ts or update_ts. All such values must be calculated and set within the Go application code before executing the SQL INSERT or UPDATE statement.
The database schema should define the create_ts and update_ts columns (e.g., numeric(17,7) NOT NULL), but responsibility for providing their values lies entirely with the Go application handlers.

Please organize the files using the following structure:

```text
/starter-api
|-- cmd/
|   `-- server/
|       `-- main.go             # Application entry point, router setup
|-- internal/
|   |-- app_lib/
|   |   |-- auth.go             # JWT handling, password hashing
|   |   |-- config.go           # Configuration loading (.env, env vars)
|   |   |-- database.go         # Postgres connection setup (pgx)
|   |   |-- logging.go          # Slog configuration
|   |   |-- rbac.go             # RBAC aggregation and checking logic
|   |   |-- shrmplKV.go            # shrmplKV connection setup
|       `-- security.go         # CSRF, Rate Limiting, Blocking checks
|   |-- handler/
|   |   |-- auth_handler.go     # Handlers for /auth, /logoff, etc.
|   |   |-- health_handler.go   # Handler for /health
|   |   |-- note_handler.go     # Handlers for /note CRUD
|   |   |-- user_handler.go     # Handlers for user management
|   |   `-- common.go           # Helper funcs for handlers (e.g., writeJSON)
|   |-- model/
|   |   |-- app_user.go         # AppUser struct and grants
|   |   |-- note.go             # Note struct and grants
|   |   |-- user_grant.go       # UserGrant struct
|   |   |-- user_group.go       # UserGroup struct
|   |   `-- request_response.go # Structs for specific request/response bodies
|   |-- repository/
|   |   |-- note_repository.go  # Note database operations
|   |   |-- user_repository.go  # AppUser database operations
|   |   |-- user_group_repository.go # UserGroup database operations
|   |   `-- user_grant_repository.go # UserGrant database operations
|-- sql/
|   |-- schema.sql              # Database table definitions
|   `-- data.sql                # Initial data seeding
|-- .env                        # Environment configuration file (example)
|-- go.mod
|-- go.sum
`-- README.md                   # Project documentation

```

That is a lot and I don't need the full project at one time. Intstead of trying to write all these files at one time, please limit your response to:

1. A summary of the approach
2. an ASCII file tree of the project
3. the config.go file
4. the files in the sql folder
5. the README.md

I will ask you for additional outputs in subsequent prompts.

</output>

Subsequent Prompts
Thank you, that looks good. Next I'd like you to generate the files in the app_lib and handler folders.

Thank you, that looks good. Next I'd like you to generate the files in the model and repository folders. You should generate models for app_user, user_grant, user_group, note. The repositoriy should be the CRUD functions for those models in light of the database you previously generated in this chat.

Thank you, that looks good. Next I'd like you to generate the main.go and go Modules files (go.mod and go.sum).

Thank you, that looks good. Let's do some cleanup. Please do the following:

1. Remove any TODO recommendations about adding middleware.
2. Remove any temporary comments about code that would be placed somewhere else.
3. Remove any comments about placeholders for future code. If it has become clear what code should be there add it. For example, the rbac.go can be generated now.
4. Review all the code wholistically to ensure it is internally consistent. Note any changes you'll make next time you generate the file, but don't generate the files.

Thank you, that looks good. Let's report the final files and insights in sections. This will be section 1:

1. A summary of the approach
2. an ASCII file tree of the project
3. the config.go file
4. the files in the sql folder
5. the main.go
6. the README.md
7. The go Module Files (go.mod and go.sum)
   When you complete this, note you completed section 1.

Let's report the final section2 to include:

1. app_lib folder contents
2. handler folder contents
3. model folder contents
4. repository folder contents
   When you complete this, note you completed section 2.
