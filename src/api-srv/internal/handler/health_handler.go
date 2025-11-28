package handler

import (
	"net/http"

	"github.com/wbmartin/starter-api/internal/app_lib"
)

// HealthHandler handles health check requests
func HealthHandler(logger app_lib.LoggerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Health check requested")

		// Simple health check - just return OK
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))

		logger.Info("Health check completed")
	}
}
