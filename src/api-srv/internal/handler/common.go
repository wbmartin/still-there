package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
func writeError(w http.ResponseWriter, status int, message string, logger app_lib.ThisAppLoggerInterface, username string) {
	logger.Error("ERR", message, "username", username)
	response := map[string]string{"error": message}
	writeJSON(w, status, response, logger)
}

// ParseCommaSeparatedInts parses a comma-separated string of integers into a slice of ints
func ParseCommaSeparatedInts(s string) ([]int, error) {
	if s == "" {
		return []int{}, nil
	}
	idStrList := strings.Split(s, ",")
	idList := make([]int, 0, len(idStrList))
	for _, idStr := range idStrList {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			return nil, err
		}
		idList = append(idList, id)
	}
	return idList, nil
}

// ParseCommaSeparatedFloats parses a comma-separated string of floats into a slice of float64
func ParseCommaSeparatedFloats(s string) ([]float64, error) {
	if s == "" {
		return []float64{}, nil
	}
	strList := strings.Split(s, ",")
	floatList := make([]float64, 0, len(strList))
	for _, str := range strList {
		f, err := strconv.ParseFloat(strings.TrimSpace(str), 64)
		if err != nil {
			return nil, err
		}
		floatList = append(floatList, f)
	}
	return floatList, nil
}

// GetIntFromQuery safely parses an integer from query parameters (max 10 chars)
func GetIntFromQuery(r *http.Request, key string) (int, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return 0, fmt.Errorf("missing required parameter: %s", key)
	}
	if len(val) > 10 {
		return 0, fmt.Errorf("parameter %s too long: max 10 characters", key)
	}
	num, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be integer", key)
	}
	return num, nil
}

// GetFloatFromQuery safely parses a float from query parameters (max 20 chars)
func GetFloatFromQuery(r *http.Request, key string) (float64, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return 0, fmt.Errorf("missing required parameter: %s", key)
	}
	if len(val) > 20 {
		return 0, fmt.Errorf("parameter %s too long: max 20 characters", key)
	}
	num, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be number", key)
	}
	return num, nil
}

// GetStringFromQuery safely retrieves a string from query parameters (max 255 chars)
func GetStringFromQuery(r *http.Request, key string) (string, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return "", fmt.Errorf("missing required parameter: %s", key)
	}
	if len(val) > 255 {
		return "", fmt.Errorf("parameter %s too long: max 255 characters", key)
	}
	return val, nil
}

// GetOptionalStringFromQuery safely retrieves an optional string from query parameters (max 255 chars if present)
func GetOptionalStringFromQuery(r *http.Request, key string) (string, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return "", nil
	}
	if len(val) > 255 {
		return "", fmt.Errorf("parameter %s too long: max 255 characters", key)
	}
	return val, nil
}

// GetCommaSeparatedIntListFromQuery parses a comma-separated list of integers from query parameters (optional, returns empty slice if missing)
func GetCommaSeparatedIntListFromQuery(r *http.Request, key string) ([]int, error) {
	str, err := GetOptionalStringFromQuery(r, key)
	if err != nil {
		return nil, err
	}
	return ParseCommaSeparatedInts(str)
}

// GetCommaSeparatedFloats parses a comma-separated list of floats from query parameters (optional, returns empty slice if missing)
func GetCommaSeparatedFloats(r *http.Request, key string) ([]float64, error) {
	str, err := GetOptionalStringFromQuery(r, key)
	if err != nil {
		return nil, err
	}
	return ParseCommaSeparatedFloats(str)
}

// GetOptionalFloatFromQuery safely parses an optional float from query parameters (max 20 chars if present)
func GetOptionalFloatFromQuery(r *http.Request, key string) (float64, error) {
	str, err := GetOptionalStringFromQuery(r, key)
	if err != nil {
		return 0, err
	}
	if str == "" {
		return 0, nil
	}
	num, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be number", key)
	}
	return num, nil
}

// CurrentTimestamp returns the current timestamp as a float64 (seconds since Unix epoch with nanosecond precision)
func CurrentTimestamp() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}
