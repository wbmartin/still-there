package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
)

// ExampleHandler handles example CRUD operations
type ExampleHandler struct {
	deps *app_lib.AppServices
}

// NewExampleHandler creates a new example handler
func NewExampleHandler(deps *app_lib.AppServices) *ExampleHandler {
	return &ExampleHandler{
		deps: deps,
	}
}

// GetExample handles GET /example
func (h *ExampleHandler) GetExample(w http.ResponseWriter, r *http.Request) {
	logger := h.deps.Logger

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.deps.KV, h.deps.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimit()
	sec.CheckUserBlocked()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantExampleGet)

	if !sec.CheckAllChecksComplete() {
		return
	}

	// Parse query parameters
	exampleIDStr := r.URL.Query().Get("exampleId")
	securityContextStr := r.URL.Query().Get("securityContext")

	if exampleIDStr == "" || securityContextStr == "" {
		writeError(w, http.StatusBadRequest, "exampleId and securityContext are required", logger)
		return
	}

	exampleID, err := strconv.Atoi(exampleIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid exampleId", logger)
		return
	}

	securityContext, err := strconv.Atoi(securityContextStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid securityContext", logger)
		return
	}

	// Check if user can access this security context
	if !sec.CheckTargetContextAuthorization(model.GrantExampleGet, securityContext) {
		return
	}

	// Caller information automatically added to log messages

	// Get example
	example, err := h.deps.ExampleRepo.GetExample(r.Context(), exampleID, securityContext)
	if err != nil {
		logger.Error("example", "GETFAIL", "Failed to get example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve example", logger)
		return
	}

	logger.Debug("example", "RETRIEVED", fmt.Sprintf("Example retrieved: %+v", example))
	writeJSON(w, http.StatusOK, example, logger)
	logger.Info("example", "GETDONE", "Example retrieval completed")
}

// CreateExample handles POST /example
func (h *ExampleHandler) CreateExample(w http.ResponseWriter, r *http.Request) {
	logger := h.deps.Logger

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.deps.KV, h.deps.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimit()
	sec.CheckUserBlocked()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantExampleAdd)

	if !sec.CheckAllChecksComplete() {
		return
	}

	claims := sec.GetClaims()

	// Parse request body
	var example model.Example
	if err := json.NewDecoder(r.Body).Decode(&example); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger)
		return
	}

	logger.Debug("example", "CREATEPARSE", fmt.Sprintf("Example creation request parsed: %+v", example))

	// Validate required fields
	if example.ExampleText == "" {
		writeError(w, http.StatusBadRequest, "example_text is required", logger)
		return
	}

	// Check if user can create in this security context
	if !sec.CheckTargetContextAuthorization(model.GrantExampleAdd, example.SecurityContext) {
		return
	}

	// Set creation metadata
	now := time.Now().Unix()
	example.CreateTS = float64(now)
	example.CreateUserID = claims.UserID
	example.UpdateTS = float64(now)
	example.UpdateUserID = claims.UserID

	// Caller information automatically added to log messages

	// Create example
	createdExample, err := h.deps.ExampleRepo.CreateExample(r.Context(), &example)
	if err != nil {
		logger.Error("example", "CREATEFAIL", "Failed to create example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to create example", logger)
		return
	}

	logger.Debug("example", "CREATED", fmt.Sprintf("Example created: %+v", createdExample))
	writeJSON(w, http.StatusCreated, createdExample, logger)
	logger.Info("example", "CREATEDONE", "Example creation completed")
}

// UpdateExample handles PUT /example
func (h *ExampleHandler) UpdateExample(w http.ResponseWriter, r *http.Request) {
	logger := h.deps.Logger

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.deps.KV, h.deps.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimit()
	sec.CheckUserBlocked()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantExampleChange)

	if !sec.CheckAllChecksComplete() {
		return
	}

	claims := sec.GetClaims()

	// Parse request body
	var example model.Example
	if err := json.NewDecoder(r.Body).Decode(&example); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger)
		return
	}

	logger.Debug("example", "UPDATEPARSE", fmt.Sprintf("Example update request parsed: %+v", example))

	// Validate required fields
	if example.ExampleID == 0 || example.ExampleText == "" || example.UpdateTS == 0 {
		writeError(w, http.StatusBadRequest, "example_id, example_text, and update_ts are required", logger)
		return
	}

	// Check if user can update in this security context
	if !sec.CheckTargetContextAuthorization(model.GrantExampleChange, example.SecurityContext) {
		return
	}

	// Set update metadata
	now := time.Now().Unix()
	example.UpdateTS = float64(now)
	example.UpdateUserID = claims.UserID

	// Caller information automatically added to log messages

	// Update example with optimistic locking
	err := h.deps.ExampleRepo.UpdateExample(r.Context(), &example, example.UpdateTS)
	if err != nil {
		if err.Error() == "conflict: example was updated by another user" {
			writeError(w, http.StatusConflict, "Example was modified by another user", logger)
			return
		}
		if err.Error() == "gone: example no longer exists" {
			writeError(w, http.StatusGone, "Example no longer exists", logger)
			return
		}
		logger.Error("example", "UPDATEFAIL", "Failed to update example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to update example", logger)
		return
	}

	// Return updated example (keeping original create fields)
	example.CreateTS = example.CreateTS         // From request
	example.CreateUserID = example.CreateUserID // From request

	logger.Debug("example", "UPDATED", fmt.Sprintf("Example updated: %+v", example))
	writeJSON(w, http.StatusOK, example, logger)
	logger.Info("example", "UPDATEDONE", "Example update completed")
}

// DeleteExample handles DELETE /example
func (h *ExampleHandler) DeleteExample(w http.ResponseWriter, r *http.Request) {
	logger := h.deps.Logger

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.deps.KV, h.deps.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimit()
	sec.CheckUserBlocked()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantExampleDelete)

	if !sec.CheckAllChecksComplete() {
		return
	}

	claims := sec.GetClaims()

	// Parse query parameters
	exampleIDStr := r.URL.Query().Get("exampleId")
	securityContextStr := r.URL.Query().Get("securityContext")
	updateTSStr := r.URL.Query().Get("updateTs")

	if exampleIDStr == "" || securityContextStr == "" || updateTSStr == "" {
		writeError(w, http.StatusBadRequest, "exampleId, securityContext, and updateTs are required", logger)
		return
	}

	exampleID, err := strconv.Atoi(exampleIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid exampleId", logger)
		return
	}

	securityContext, err := strconv.Atoi(securityContextStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid securityContext", logger)
		return
	}

	updateTS, err := strconv.ParseFloat(updateTSStr, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid updateTs", logger)
		return
	}

	// Check if user can delete in this security context
	if !sec.CheckTargetContextAuthorization(model.GrantExampleDelete, securityContext) {
		return
	}

	// Caller information automatically added to log messages

	// Delete example with optimistic locking
	now := time.Now().Unix()
	err = h.deps.ExampleRepo.DeleteExample(r.Context(), exampleID, securityContext, updateTS, float64(now), claims.UserID)
	if err != nil {
		if err.Error() == "conflict: example was updated by another user" {
			writeError(w, http.StatusConflict, "Example was modified by another user", logger)
			return
		}
		if err.Error() == "gone: example no longer exists" {
			writeError(w, http.StatusGone, "Example no longer exists", logger)
			return
		}
		logger.Error("example", "DELETEFAIL", "Failed to delete example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to delete example", logger)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Example deleted successfully"}, logger)
	logger.Info("example", "DELETEDONE", "Example deletion completed")
}
