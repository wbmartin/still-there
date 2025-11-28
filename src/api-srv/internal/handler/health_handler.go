package handler

import (
	"net/http"

	"github.com/wbmartin/starter-api/internal/app_lib"
)

// HealthHandler handles health check requests
func HealthHandler(deps *app_lib.AppServices) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps.Logger.Info("health", "REQ", "Health check requested")

		// Simple health check - just return OK
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))

		deps.Logger.Info("health", "RESP", "Health check completed")
	}
}
