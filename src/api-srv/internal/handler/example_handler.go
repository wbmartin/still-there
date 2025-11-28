package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/wbmartin/starter-api/internal/app_lib"
	"github.com/wbmartin/starter-api/internal/model"
	"github.com/wbmartin/starter-api/internal/repository"
)

// ExampleHandler handles example CRUD operations
type ExampleHandler struct {
	appsrv      *app_lib.AppServices
	exampleRepo *repository.ExampleRepository
}

// NewExampleHandler creates a new example handler
func NewExampleHandler(appServices *app_lib.AppServices, exampleRepo *repository.ExampleRepository) *ExampleHandler {
	return &ExampleHandler{
		appsrv:      appServices,
		exampleRepo: exampleRepo,
	}
}

// GetExample handles GET /example
func (h *ExampleHandler) GetExample(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimit()
	sec.CheckUserBlocked()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantExampleGet)

	if !sec.CheckAllChecksComplete() {
		return
	}

	claims := sec.GetClaims()

	// Parse query parameters
	exampleIDStr := r.URL.Query().Get("exampleId")

	if exampleIDStr == "" {
		writeError(w, http.StatusBadRequest, "exampleId is required", logger)
		return
	}

	exampleID, err := strconv.Atoi(exampleIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid exampleId", logger)
		return
	}

	// Caller information automatically added to log messages

	// Get example
	example, err := h.exampleRepo.GetExample(r.Context(), exampleID, claims.SecurityContext)
	if err != nil {
		logger.Error("example", "GTF0", "Failed to get example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve example", logger)
		return
	}

	logger.Debug("example", "RET0", fmt.Sprintf("Example retrieved: %+v", example))
	writeJSON(w, http.StatusOK, example, logger)
	logger.Info("example", "GTD0", "Example retrieval completed")
}

// CreateExample handles POST /example
func (h *ExampleHandler) CreateExample(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
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
	var example model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&example); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger)
		return
	}

	logger.Debug("example", "CRP0", fmt.Sprintf("Example creation request parsed: %+v", example))

	// Validate required fields
	if example.ExampleText == "" {
		writeError(w, http.StatusBadRequest, "example_text is required", logger)
		return
	}

	// Set security context and creation metadata
	now := time.Now().Unix()
	example.SecurityContext = claims.SecurityContext
	example.CreateTS = float64(now)
	example.CreateUserID = claims.UserID
	example.UpdateTS = float64(now)
	example.UpdateUserID = claims.UserID

	// Caller information automatically added to log messages

	// Create example
	createdExample, err := h.exampleRepo.CreateExample(r.Context(), &example)
	if err != nil {
		logger.Error("example", "CRF0", "Failed to create example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to create example", logger)
		return
	}

	logger.Debug("example", "CRD0", fmt.Sprintf("Example created: %+v", createdExample))
	writeJSON(w, http.StatusCreated, createdExample, logger)
	logger.Info("example", "CRD1", "Example creation completed")
}

// UpdateExample handles PUT /example
func (h *ExampleHandler) UpdateExample(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
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
	var example model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&example); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger)
		return
	}

	logger.Debug("example", "UPP0", fmt.Sprintf("Example update request parsed: %+v", example))

	// Validate required fields
	if example.ExampleID == 0 || example.ExampleText == "" {
		writeError(w, http.StatusBadRequest, "example_id and example_text are required", logger)
		return
	}

	// Fetch existing example to verify ownership
	existing, err := h.exampleRepo.GetExample(r.Context(), example.ExampleID, claims.SecurityContext)
	if err != nil {
		logger.Error("example", "UFF0", "Failed to fetch existing example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve example", logger)
		return
	}

	// Set security context and update metadata
	now := time.Now().Unix()
	example.SecurityContext = claims.SecurityContext
	example.UpdateTS = float64(now)
	example.UpdateUserID = claims.UserID

	// Caller information automatically added to log messages

	// Update example with optimistic locking
	err = h.exampleRepo.UpdateExample(r.Context(), &example, existing.UpdateTS)
	if err != nil {
		if err.Error() == "conflict: example was updated by another user" {
			writeError(w, http.StatusConflict, "Example was modified by another user", logger)
			return
		}
		if err.Error() == "gone: example no longer exists" {
			writeError(w, http.StatusGone, "Example no longer exists", logger)
			return
		}
		logger.Error("example", "UPF0", "Failed to update example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to update example", logger)
		return
	}

	// Return updated example (keeping original create fields)
	example.CreateTS = example.CreateTS         // From request
	example.CreateUserID = example.CreateUserID // From request

	logger.Debug("example", "UPD0", fmt.Sprintf("Example updated: %+v", example))
	writeJSON(w, http.StatusOK, example, logger)
	logger.Info("example", "UPD1", "Example update completed")
}

// DeleteExample handles DELETE /example
func (h *ExampleHandler) DeleteExample(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
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
	updateTSStr := r.URL.Query().Get("updateTs")

	if exampleIDStr == "" || updateTSStr == "" {
		writeError(w, http.StatusBadRequest, "exampleId and updateTs are required", logger)
		return
	}

	exampleID, err := strconv.Atoi(exampleIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid exampleId", logger)
		return
	}

	updateTS, err := strconv.ParseFloat(updateTSStr, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid updateTs", logger)
		return
	}

	// Caller information automatically added to log messages

	// Delete example with optimistic locking
	now := time.Now().Unix()
	err = h.exampleRepo.DeleteExample(r.Context(), exampleID, claims.SecurityContext, updateTS, float64(now), claims.UserID)
	if err != nil {
		if err.Error() == "conflict: example was updated by another user" {
			writeError(w, http.StatusConflict, "Example was modified by another user", logger)
			return
		}
		if err.Error() == "gone: example no longer exists" {
			writeError(w, http.StatusGone, "Example no longer exists", logger)
			return
		}
		logger.Error("example", "DLF0", "Failed to delete example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to delete example", logger)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Example deleted successfully"}, logger)
	logger.Info("example", "DLD0", "Example deletion completed")
}

// GetExampleList handles GET /exampleList?ids={comma-separated}
func (h *ExampleHandler) GetExampleList(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
	sec.CheckJWT()
	sec.CheckRateLimit()
	sec.CheckUserBlocked()
	sec.CheckCSRF()
	sec.CheckAuthorization(model.GrantExampleGet)

	if !sec.CheckAllChecksComplete() {
		return
	}

	claims := sec.GetClaims()

	// Parse query parameters
	idsStr := r.URL.Query().Get("ids")
	if idsStr == "" {
		writeError(w, http.StatusBadRequest, "ids parameter is required", logger)
		return
	}

	idStrs := strings.Split(idsStr, ",")
	ids := make([]int, 0, len(idStrs))
	for _, idStr := range idStrs {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid id in ids parameter", logger)
			return
		}
		ids = append(ids, id)
	}

	// Get examples (repo filters by security context)
	examples, err := h.exampleRepo.GetExampleList(r.Context(), ids, claims.SecurityContext)
	if err != nil {
		logger.Error("example", "LIF0", "Failed to get examples: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve examples", logger)
		return
	}

	// Return as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(examples); err != nil {
		logger.Error("example", "ENF0", "Failed to encode response: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to encode response", logger)
		return
	}

	logger.Info("example", "LIR0", fmt.Sprintf("Returned %d examples", len(examples)))
}

// CreateExampleList handles POST /exampleList
func (h *ExampleHandler) CreateExampleList(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger

	// Security checks
	sec := app_lib.NewSecurityHelper(w, r, h.appsrv.KV, h.appsrv.Config, logger)
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
	var examples []model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&examples); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger)
		return
	}

	if len(examples) == 0 {
		writeError(w, http.StatusBadRequest, "At least one example required", logger)
		return
	}

	// Convert to slice of pointers
	examplePtrs := make([]*model.ExampleModel, len(examples))
	for i := range examples {
		examplePtrs[i] = &examples[i]
	}

	// Validate all examples have the same security context and it's allowed
	userSecurityContext := claims.SecurityContext
	for i, example := range examples {
		if example.SecurityContext != userSecurityContext {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Example %d has invalid security context", i), logger)
			return
		}
		if example.ExampleText == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Example %d missing example_text", i), logger)
			return
		}
	}

	// Generate timestamps once
	now := time.Now().Unix()
	createTS := float64(now)
	updateTS := float64(now)

	// Set metadata for all
	for i := range examples {
		examples[i].CreateTS = createTS
		examples[i].CreateUserID = claims.UserID
		examples[i].UpdateTS = updateTS
		examples[i].UpdateUserID = claims.UserID
	}

	// Create in transaction
	createdExamples, err := h.exampleRepo.CreateExampleList(r.Context(), examplePtrs)
	if err != nil {
		logger.Error("example", "BCF0", "Failed to create examples: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to create examples", logger)
		return
	}

	// Return the created examples with IDs
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(createdExamples); err != nil {
		logger.Error("example", "ENF0", "Failed to encode response: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to encode response", logger)
		return
	}

	logger.Info("example", "BCR0", fmt.Sprintf("Created %d examples", len(createdExamples)))
}
