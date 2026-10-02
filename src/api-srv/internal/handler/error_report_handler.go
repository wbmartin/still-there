package handler

import (
	"net/http"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/security"
)

// maxErrorReportBodyBytes caps what an unauthenticated caller can make the
// server allocate. Generous against the 2000 characters actually kept, so a
// real stack trace still arrives intact, but bounded.
const maxErrorReportBodyBytes = 8 << 10 // 8 KiB

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

// HandleErrorReport handles POST /error-report for client-side error reporting
func (h *ErrorReportHandler) HandleErrorReport(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Security checks - rate limiting only, skip others for error reports
	app := security.NewAppHelper(w, r, h.appServices, "HandleErrorReport")
	app.SkipJWT()           // No auth required for error reports
	app.SkipCSRF()          // Error reports aren't form submissions
	app.SkipAuthorization() // No grants needed
	app.SkipUserBlock()     // Not relevant for error reports
	app.CheckRateLimit()    // Rate limit to prevent abuse

	if !app.CheckAllChecksComplete() {
		return
	}

	// Bounded before it is read, not after: the truncation below runs on a
	// string already in memory, so it protects only the log line.
	body, ok := readBody(app, w, r, maxErrorReportBodyBytes)
	if !ok {
		return
	}

	// Truncate to 2000 characters
	report := string(body)
	if len(report) > 2000 {
		report = report[:2000] + "...[truncated]"
	}

	// Log the error report
	app.Logger().Error("CLER", "Client Error Report: "+report)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"reported"}`))
}
