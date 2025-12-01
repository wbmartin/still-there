package handler

import (
	"encoding/json"
	"net/http"

	"github.com/wbmartin/starter-api/internal/app_lib"
)

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}, logger app_lib.ThisAppLoggerInterface) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("JEN0", "Failed to encode JSON response: "+err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string, logger app_lib.ThisAppLoggerInterface) {
	response := map[string]string{"error": message}
	writeJSON(w, status, response, logger)
}
