package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
)

// ExampleHandler handles example CRUD operations
type ExampleHandler struct {
	db          *app_lib.Database
	shrmplKV    *app_lib.ShrmplKVClient
	config      *app_lib.Config
	logger      app_lib.LoggerInterface
	exampleRepo *repository.ExampleRepository
}

// NewExampleHandler creates a new example handler
func NewExampleHandler(db *app_lib.Database, shrmplKV *app_lib.ShrmplKVClient, config *app_lib.Config, logger app_lib.LoggerInterface, exampleRepo *repository.ExampleRepository) *ExampleHandler {
	return &ExampleHandler{
		db:          db,
		shrmplKV:    shrmplKV,
		config:      config,
		logger:      logger.With("handler", "example"),
		exampleRepo: exampleRepo,
	}
}

// GetExample handles GET /example
func (h *ExampleHandler) GetExample(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("method", "GET")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.shrmplKV, h.config, logger)
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

	logger = logger.With("example_id", exampleID, "security_context", securityContext)

	// Get example
	example, err := h.exampleRepo.GetExample(r.Context(), exampleID, securityContext)
	if err != nil {
		logger.Error("Failed to get example: %s", err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve example", logger)
		return
	}

	logger.Debug("Example retrieved: %+v", example)
	writeJSON(w, http.StatusOK, example, logger)
	logger.Info("Example retrieval completed")
}

// CreateExample handles POST /example
func (h *ExampleHandler) CreateExample(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("method", "POST")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.shrmplKV, h.config, logger)
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

	logger.Debug("Example creation request parsed: %+v", example)

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

	logger = logger.With("security_context", example.SecurityContext)

	// Create example
	createdExample, err := h.exampleRepo.CreateExample(r.Context(), &example)
	if err != nil {
		logger.Error("Failed to create example: %s", err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to create example", logger)
		return
	}

	logger.Debug("Example created: %+v", createdExample)
	writeJSON(w, http.StatusCreated, createdExample, logger)
	logger.Info("Example creation completed")
}

// UpdateExample handles PUT /example
func (h *ExampleHandler) UpdateExample(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("method", "PUT")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.shrmplKV, h.config, logger)
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

	logger.Debug("Example update request parsed: %+v", example)

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

	logger = logger.With("example_id", example.ExampleID, "security_context", example.SecurityContext)

	// Update example with optimistic locking
	err := h.exampleRepo.UpdateExample(r.Context(), &example, example.UpdateTS)
	if err != nil {
		if err.Error() == "conflict: example was updated by another user" {
			writeError(w, http.StatusConflict, "Example was modified by another user", logger)
			return
		}
		if err.Error() == "gone: example no longer exists" {
			writeError(w, http.StatusGone, "Example no longer exists", logger)
			return
		}
		logger.Error("Failed to update example: %s", err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to update example", logger)
		return
	}

	// Return updated example (keeping original create fields)
	example.CreateTS = example.CreateTS         // From request
	example.CreateUserID = example.CreateUserID // From request

	logger.Debug("Example updated: %+v", example)
	writeJSON(w, http.StatusOK, example, logger)
	logger.Info("Example update completed")
}

// DeleteExample handles DELETE /example
func (h *ExampleHandler) DeleteExample(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("method", "DELETE")

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.shrmplKV, h.config, logger)
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

	logger = logger.With("example_id", exampleID, "security_context", securityContext)

	// Delete example with optimistic locking
	now := time.Now().Unix()
	err = h.exampleRepo.DeleteExample(r.Context(), exampleID, securityContext, updateTS, float64(now), claims.UserID)
	if err != nil {
		if err.Error() == "conflict: example was updated by another user" {
			writeError(w, http.StatusConflict, "Example was modified by another user", logger)
			return
		}
		if err.Error() == "gone: example no longer exists" {
			writeError(w, http.StatusGone, "Example no longer exists", logger)
			return
		}
		logger.Error("Failed to delete example: %s", err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to delete example", logger)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Example deleted successfully"}, logger)
	logger.Info("Example deletion completed")
}
