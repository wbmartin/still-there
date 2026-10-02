package security

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/auth"
	"github.com/wbmartin/starter-api/internal/cache"
	"github.com/wbmartin/starter-api/internal/config"
	"github.com/wbmartin/starter-api/internal/logging"
)

func revocationServices(t *testing.T) *app.Services {
	t.Helper()
	kv := cache.NewKV()
	t.Cleanup(kv.Close)
	return &app.Services{
		KV: kv,
		Config: &config.Config{
			JWTSecret:             "test-secret",
			JWTSessionExpireHours: 72,
		},
		Logger: logging.NewLogger("test", &bytes.Buffer{}),
	}
}

func signToken(t *testing.T, s *app.Services, sessionID string) string {
	t.Helper()
	token, err := auth.GenerateJWT(
		"Test", "alice", 1, sessionID, 1, nil,
		s.Config.JWTSecret, s.Config.JWTSessionExpireHours,
	)
	require.NoError(t, err)
	return token
}

// checkJWT runs CheckJWT on a request carrying token and returns the status
// written (200 when the check passed and wrote nothing).
func checkJWT(t *testing.T, s *app.Services, token string) int {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/example", nil)
	r.AddCookie(&http.Cookie{Name: "auth", Value: token})
	w := httptest.NewRecorder()
	helper := NewAppHelper(w, r, s, "test")
	helper.CheckJWT()
	if helper.CanProceed() {
		require.NotNil(t, helper.GetClaims())
	}
	return w.Code
}

func TestRevocationTTLCoversTheSession(t *testing.T) {
	s := revocationServices(t)
	assert.Equal(t, "72h", revocationTTL(s))
	s.Config.JWTSessionExpireHours = 0
	assert.Equal(t, "1h", revocationTTL(s))
}

func TestCheckJWT_RevokedSession(t *testing.T) {
	s := revocationServices(t)
	token := signToken(t, s, "session-one")
	other := signToken(t, s, "session-two")
	require.Equal(t, http.StatusOK, checkJWT(t, s, token))

	require.NoError(t, RevokeSession(s, "session-one"))
	assert.Equal(t, http.StatusUnauthorized, checkJWT(t, s, token))
	assert.Equal(t, http.StatusOK, checkJWT(t, s, other),
		"another session of the same login is untouched")
	require.NoError(t, RevokeSession(s, ""), "no id is a no-op")
}

func TestCheckJWT_UserCutoff(t *testing.T) {
	s := revocationServices(t)
	before := signToken(t, s, "session-old")
	kept := signToken(t, s, "session-kept")

	require.NoError(t, RevokeUserSessions(s, "alice", "session-kept"))
	assert.Equal(t, http.StatusUnauthorized, checkJWT(t, s, before),
		"a token issued before the cutoff is refused")
	assert.Equal(t, http.StatusOK, checkJWT(t, s, kept),
		"the kept session survives")

	after := signToken(t, s, "session-new")
	assert.Equal(t, http.StatusOK, checkJWT(t, s, after),
		"a token issued after the cutoff is good, even in the same second")

	require.NoError(t, RevokeUserSessions(s, "alice", ""))
	assert.Equal(t, http.StatusUnauthorized, checkJWT(t, s, kept),
		"keeping none refuses every earlier token")
}

func TestSessionRevoked_CutoffEdges(t *testing.T) {
	s := revocationServices(t)
	restore := timeNow
	t.Cleanup(func() { timeNow = restore })
	issued := time.Unix(1_700_000_000, 500)
	claims := &auth.Claims{SessionID: "session-x"}
	claims.Subject = "alice"
	claims.ID = "1700000000000000500"

	timeNow = func() time.Time { return issued }
	require.NoError(t, RevokeUserSessions(s, "alice", ""))
	revoked, err := sessionRevoked(s, claims)
	require.NoError(t, err)
	assert.False(t, revoked, "issued at the cutoff instant is not before it")

	timeNow = func() time.Time { return issued.Add(time.Nanosecond) }
	require.NoError(t, RevokeUserSessions(s, "alice", ""))
	revoked, err = sessionRevoked(s, claims)
	require.NoError(t, err)
	assert.True(t, revoked, "one ns before the cutoff is refused")

	claims.ID = "not-a-time"
	revoked, err = sessionRevoked(s, claims)
	require.NoError(t, err)
	assert.True(t, revoked, "no readable issue time fails closed")

	require.NoError(t, s.KV.Set(userRevokedPrefix+"alice", "garbage", "1h"))
	_, err = sessionRevoked(s, claims)
	assert.Error(t, err, "an unreadable cutoff is an error, not a pass")
}

func TestCheckJWT_KVErrorFailsClosed(t *testing.T) {
	s := revocationServices(t)
	token := signToken(t, s, "session-one")
	s.KV.Close()
	assert.Equal(t, http.StatusInternalServerError, checkJWT(t, s, token))
}
