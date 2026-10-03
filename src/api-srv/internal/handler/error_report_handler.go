package handler

import (
	"net/http"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/security"
)

// maxErrorReportBodyBytes bounds a report: generous against the 2000
// characters kept, so a real stack trace arrives intact.
const maxErrorReportBodyBytes = 8 << 10 // 8 KiB

// maxErrorReportLogChars is how much of a report reaches the log.
const maxErrorReportLogChars = 2000

// ErrorReportHandler handles client-side error reporting
type ErrorReportHandler struct {
	appServices *app.Services
}

// NewErrorReportHandler creates a new error report handler
func NewErrorReportHandler(
	appServices *app.Services,
) *ErrorReportHandler {
	return &ErrorReportHandler{
		appServices: appServices,
	}
}

// HandleErrorReport handles POST /error-report: a signed-in caller's
// client-side error, logged with its username. Anonymous callers get 401,
// so nobody without an account can write to the log.
func (h *ErrorReportHandler) HandleErrorReport(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Security checks
	app := security.NewAppHelper(w, r, h.appServices, "HandleErrorReport")
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantLogon)

	if !app.CheckAllChecksComplete() {
		return
	}

	// Bounded before it is read; the truncation below protects only the log
	// line.
	body, ok := readBody(app, w, r, maxErrorReportBodyBytes)
	if !ok {
		return
	}

	report := string(body)
	if len(report) > maxErrorReportLogChars {
		report = report[:maxErrorReportLogChars] + "...[truncated]"
	}
	app.Error("CLER", "Client Error Report: "+report)

	app.WriteJSON(w, http.StatusOK, map[string]string{"status": "reported"})
}
