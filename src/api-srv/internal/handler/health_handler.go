package handler

import (
	"net/http"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
)

// HealthHandler handles health check requests
func HealthHandler(deps *app_lib.AppServices) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app := app_lib.NewAppHelper(w, r, deps)
		app.SkipJWT()
		app.SkipRateLimit()
		app.SkipUserBlock()
		app.SkipCSRF()
		app.SkipAuthorization()

		if !app.CheckAllChecksComplete() {
			app.Logger().
				Error("SEC0", "Failed to skip all checks for health handler")
			http.Error(
				w,
				"Internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		username := "PreAuth"
		app.Logger().Info(
			"HEALTH_REQUEST",
			"Health check requested",
			"username",
			username,
		)

		// Simple health check - just return OK
		data := map[string]string{"status": "ok"}
		wrapper := model.DataWrapper{Data: data}
		app.WriteJSON(w, http.StatusOK, wrapper)

		app.Logger().Info(
			"HEALTH_RESPONSE",
			"Health check completed",
			"username",
			username,
		)
	}
}
