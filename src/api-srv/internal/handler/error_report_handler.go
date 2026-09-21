package handler

import (
	"io"
	"net/http"

	"github.com/wbmartin/starter-api/internal/app_lib"
)

// maxErrorReportBodyBytes caps what an unauthenticated caller can make the
// server allocate. Generous against the 2000 characters actually kept, so a
// real stack trace still arrives intact, but bounded.
const maxErrorReportBodyBytes = 8 << 10 // 8 KiB

// ErrorReportHandler handles client-side error reporting
type ErrorReportHandler struct {
	appServices *app_lib.AppServices
}

// NewErrorReportHandler creates a new error report handler
func NewErrorReportHandler(
	appServices *app_lib.AppServices,
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
	app := app_lib.NewAppHelper(w, r, h.appServices, "HandleErrorReport")
	app.SkipJWT()           // No auth required for error reports
	app.SkipCSRF()          // Error reports aren't form submissions
	app.SkipAuthorization() // No grants needed
	app.SkipUserBlock()     // Not relevant for error reports
	app.CheckRateLimit()    // Rate limit to prevent abuse

	if !app.CheckAllChecksComplete() {
		return
	}

	// Bounded before it is read, not after. The truncation below runs on a
	// string that is already in memory, so without this an unauthenticated
	// caller allocates the whole body first and the 2000-char cap protects
	// only the log line. MaxBytesReader makes ReadAll fail instead, which the
	// existing error path already turns into a 400.
	r.Body = http.MaxBytesReader(w, r.Body, maxErrorReportBodyBytes)

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		app.WriteError(
			w,
			http.StatusBadRequest,
			"Failed to read request body",
			"CLER",
			err,
		)
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
