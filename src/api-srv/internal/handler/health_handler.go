package handler

import (
	"net/http"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
)

// HealthHandler handles health check requests
func HealthHandler(deps *app_lib.AppServices) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := "PreAuth"
		deps.Logger.Info(
			"HEALTH_REQUEST",
			"Health check requested",
			"username",
			username,
		)

		// Simple health check - just return OK
		data := map[string]string{"status": "ok"}
		wrapper := model.DataWrapper{Data: data}
		writeJSON(w, http.StatusOK, wrapper, deps.Logger)

		deps.Logger.Info(
			"HEALTH_RESPONSE",
			"Health check completed",
			"username",
			username,
		)
	}
}
