package handler

import (
	"net/http"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
)

// HealthHandler handles health check requests
func HealthHandler(deps *app_lib.AppServices) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps.Logger.Info("health", "REQ", "Health check requested")

		// Simple health check - just return OK
		data := map[string]string{"status": "ok"}
		wrapper := model.DataWrapper{Data: data}
		writeJSON(w, http.StatusOK, wrapper, deps.Logger)

		deps.Logger.Info("health", "RESP", "Health check completed")
	}
}
