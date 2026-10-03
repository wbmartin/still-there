package handler

import (
	"net/http"

	"github.com/wbmartin/starter-api/internal/app"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/security"
)

// HealthHandler handles health check requests
func HealthHandler(deps *app.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appHelper := security.NewAppHelper(w, r, deps, "HealthHandler")
		appHelper.SkipJWT()
		appHelper.SkipRateLimit()
		appHelper.SkipUserBlock()
		appHelper.SkipCSRF()
		appHelper.SkipAuthorization()

		if !appHelper.CheckAllChecksComplete() {
			appHelper.Logger().
				Error("SEC0", "Failed to skip all checks for health handler")
			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		username := "PreAuth"
		appHelper.Logger().Debug(
			"HEALTH_REQUEST",
			"Health check requested",
			"username",
			username,
		)

		// Simple health check - just return OK
		data := map[string]string{"status": "ok"}
		wrapper := model.DataWrapper{Data: data}
		appHelper.WriteJSON(w, http.StatusOK, wrapper)

		appHelper.Logger().Debug(
			"HEALTH_RESPONSE",
			"Health check completed",
			"username",
			username,
		)
	}
}
