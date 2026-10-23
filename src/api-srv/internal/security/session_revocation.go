package security

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"stillthere-api/internal/app"
	"stillthere-api/internal/auth"
)

// Session revocation. A JWT carries its grants and is valid until it
// expires (JWT_SESSION_EXPIRE_HOURS), so without server-side state a
// signed-out session, or one whose password, second factor, groups or login
// changed, stays usable to whoever holds the token. CheckJWT reads two KV
// keys on every request, so they bind every route and every subject:
//
//	SESSION_REVOKED*<session id>  one session ended (sign-out)
//	REVOKED_BEFORE*<user name>    "<unix ns>|<session id kept>": every token
//	                              of that login issued before the time is
//	                              refused, except the one session named
//
// Both expire with the longest session a JWT can carry; after that no token
// they could refuse is left. The KV is in-process: an API restart forgets
// them, and a token revoked before a restart is good again until it expires
// (doc/105, Session revocation).

const (
	sessionRevokedPrefix = "SESSION_REVOKED*"
	userRevokedPrefix    = "REVOKED_BEFORE*"
)

// timeNow is time.Now, replaceable in tests.
var timeNow = time.Now

func revocationTTL(appServices *app.Services) string {
	hours := appServices.Config.JWTSessionExpireHours
	if hours < 1 {
		hours = 1
	}
	return strconv.Itoa(hours) + "h"
}

// RevokeSession ends one session (sign-out).
func RevokeSession(appServices *app.Services, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return appServices.KV.Set(sessionRevokedPrefix+sessionID, "1",
		revocationTTL(appServices))
}

// RevokeUserSessions refuses every token of userName issued before now,
// except the session keepSessionID ("" keeps none).
func RevokeUserSessions(
	appServices *app.Services,
	userName string,
	keepSessionID string,
) error {
	value := fmt.Sprintf("%d|%s", timeNow().UnixNano(), keepSessionID)
	return appServices.KV.Set(userRevokedPrefix+userName, value,
		revocationTTL(appServices))
}

// sessionRevoked reports whether claims belong to a revoked session. An
// error means the answer is unknown and the caller must fail closed.
func sessionRevoked(appServices *app.Services, claims *auth.Claims) (bool, error) {
	ended, err := appServices.KV.Get(sessionRevokedPrefix + claims.SessionID)
	if err != nil {
		return false, err
	}
	if ended != "" {
		return true, nil
	}
	cutoff, err := appServices.KV.Get(userRevokedPrefix + claims.Subject)
	if err != nil || cutoff == "" {
		return false, err
	}
	cutoffText, keepSessionID, _ := strings.Cut(cutoff, "|")
	cutoffNano, err := strconv.ParseInt(cutoffText, 10, 64)
	if err != nil {
		return false, fmt.Errorf("revocation value %q: %w", cutoff, err)
	}
	if keepSessionID != "" && claims.SessionID == keepSessionID {
		return false, nil
	}
	issuedNano, ok := claims.IssuedAtNano()
	if !ok {
		return true, nil
	}
	return issuedNano < cutoffNano, nil
}
