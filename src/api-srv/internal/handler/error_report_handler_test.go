package handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wbmartin/starter-api/internal/logging"
	"github.com/wbmartin/starter-api/internal/model"
)

// TestErrorReport_RequiresASession: only a signed-in caller with LOGON can
// write a report to the log; every refusal leaves the log without one.
func TestErrorReport_RequiresASession(t *testing.T) {
	const sessionID = "session-abc"
	const report = `{"message":"render failed"}`
	tests := []struct {
		name       string
		grantList  []int
		signed     bool
		dropCSRF   bool
		body       string
		wantStatus int
		wantLogged bool
	}{
		{name: "no auth cookie → 401", body: report,
			wantStatus: http.StatusUnauthorized},
		{name: "no CSRF header → 403", signed: true, dropCSRF: true,
			grantList: []int{model.GrantLogon}, body: report,
			wantStatus: http.StatusForbidden},
		{name: "no LOGON grant → 403", signed: true, grantList: []int{},
			body: report, wantStatus: http.StatusForbidden},
		{name: "over 8 KiB → 413", signed: true,
			grantList:  []int{model.GrantLogon},
			body:       strings.Repeat("a", maxErrorReportBodyBytes+1),
			wantStatus: http.StatusRequestEntityTooLarge},
		{name: "signed in with LOGON → 200, logged with username",
			signed: true, grantList: []int{model.GrantLogon}, body: report,
			wantStatus: http.StatusOK, wantLogged: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServices(t)
			logBuffer := &bytes.Buffer{}
			s.Logger = logging.NewLogger("test", logBuffer, slog.LevelDebug)

			r := httptest.NewRequest(http.MethodPost, "/error-report",
				strings.NewReader(tt.body))
			if tt.signed {
				token := sessionToken(t, s, "alice", 1, sessionID,
					tt.grantList)
				r = sessionRequest(http.MethodPost, "/error-report", tt.body,
					token, sessionID)
			}
			if tt.dropCSRF {
				r.Header.Del("X-CSRF-Token")
			}
			w := httptest.NewRecorder()
			NewErrorReportHandler(s).HandleErrorReport(w, r)

			assert.Equal(t, tt.wantStatus, w.Code, w.Body.String())
			logged := strings.Contains(logBuffer.String(), "render failed")
			assert.Equal(t, tt.wantLogged, logged, logBuffer.String())
			if tt.wantLogged {
				assert.JSONEq(t, `{"status":"reported"}`, w.Body.String())
				assert.Contains(t, logBuffer.String(), "username=alice")
			}
		})
	}
}

func TestErrorReport_LogKeepsTheFirst2000Characters(t *testing.T) {
	const sessionID = "session-abc"
	s := testServices(t)
	logBuffer := &bytes.Buffer{}
	s.Logger = logging.NewLogger("test", logBuffer, slog.LevelDebug)
	token := sessionToken(t, s, "alice", 1, sessionID,
		[]int{model.GrantLogon})
	body := strings.Repeat("a", maxErrorReportLogChars) + "TAIL"

	w := httptest.NewRecorder()
	NewErrorReportHandler(s).HandleErrorReport(w,
		sessionRequest(http.MethodPost, "/error-report", body, token,
			sessionID))

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, logBuffer.String(), "...[truncated]")
	assert.NotContains(t, logBuffer.String(), "TAIL")
}
