package security

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"stillthere-api/internal/logging"
)

// checkCSRF runs CheckJWT then CheckCSRF on a POST carrying csrfToken.
func checkCSRF(
	t *testing.T,
	csrfToken string,
) *httptest.ResponseRecorder {
	t.Helper()
	s := revocationServices(t)
	r := httptest.NewRequest(http.MethodPost, "/example", nil)
	r.AddCookie(&http.Cookie{
		Name: "auth", Value: signToken(t, s, "session-one"),
	})
	if csrfToken != "" {
		r.Header.Set("X-CSRF-Token", csrfToken)
	}
	w := httptest.NewRecorder()
	NewAppHelper(w, r, s, "test").CheckJWT().CheckCSRF()
	return w
}

// The web client resyncs its token once on code "CSRF" (B004); a reworded
// message must not silently turn that off.
func TestCheckCSRF_RefusalCarriesTheCode(t *testing.T) {
	for name, csrfToken := range map[string]string{
		"missing token": "",
		"stale token":   "xyz",
	} {
		t.Run(name, func(t *testing.T) {
			w := checkCSRF(t, csrfToken)
			require.Equal(t, http.StatusForbidden, w.Code)
			var body map[string]string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, "CSRF", body["code"])
			assert.NotEmpty(t, body["error"])
		})
	}
}

func TestCheckCSRF_MatchingTokenPasses(t *testing.T) {
	w := checkCSRF(t, "one")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Body.String())
}

// A mismatch is logged without either token: both are slices of a session
// ID, and the received one from a stale tab is an earlier session's.
func TestCheckCSRF_MismatchLogsNoToken(t *testing.T) {
	s := revocationServices(t)
	var logBuffer bytes.Buffer
	s.Logger = logging.NewLogger("test", &logBuffer, slog.LevelDebug)
	r := httptest.NewRequest(http.MethodPost, "/example", nil)
	r.AddCookie(&http.Cookie{
		Name: "auth", Value: signToken(t, s, "session-one"),
	})
	r.Header.Set("X-CSRF-Token", "xqz")
	w := httptest.NewRecorder()
	NewAppHelper(w, r, s, "test").CheckJWT().CheckCSRF()

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, logBuffer.String(), "CSRF token mismatch")
	assert.NotContains(t, logBuffer.String(), "xqz")
	assert.NotContains(t, logBuffer.String(), `"one"`)
}
