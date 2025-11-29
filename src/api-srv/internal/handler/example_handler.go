package handler

import (
	"encoding/json"
	"errors"
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

	jwtClaims := sec.GetClaims()

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
	example, err := h.exampleRepo.GetExample(r.Context(), exampleID, jwtClaims.SecurityContext)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Example not found", logger)
			return
		}
		logger.Error("example", "GTF0", "Failed to get example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve example", logger)
		return
	}

	logger.Debug("example", "RET0", fmt.Sprintf("Example retrieved: %+v", example))
	wrapper := model.DataWrapper{Data: example}
	writeJSON(w, http.StatusOK, wrapper, logger)
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

	jwtClaims := sec.GetClaims()

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
	now := float64(time.Now().UnixNano()) / 1e9
	example.SecurityContext = jwtClaims.SecurityContext
	example.CreateTS = now
	example.CreateUserID = jwtClaims.UserID
	example.UpdateTS = now
	example.UpdateUserID = jwtClaims.UserID

	// Caller information automatically added to log messages

	// Create example
	createdExample, err := h.exampleRepo.CreateExample(r.Context(), &example)
	if err != nil {
		logger.Error("example", "CRF0", "Failed to create example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to create example", logger)
		return
	}

	logger.Debug("example", "CRD0", fmt.Sprintf("Example created: %+v", createdExample))
	wrapper := model.DataWrapper{Data: createdExample}
	writeJSON(w, http.StatusCreated, wrapper, logger)
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

	jwtClaims := sec.GetClaims()

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
	existing, err := h.exampleRepo.GetExample(r.Context(), example.ExampleID, jwtClaims.SecurityContext)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Example not found", logger)
			return
		}
		logger.Error("example", "UFF0", "Failed to fetch existing example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve example", logger)
		return
	}

	// Set security context and update metadata
	now := float64(time.Now().UnixNano()) / 1e9
	example.SecurityContext = jwtClaims.SecurityContext
	example.UpdateTS = now
	example.UpdateUserID = jwtClaims.UserID

	// Caller information automatically added to log messages

	// Update example with optimistic locking
	err = h.exampleRepo.UpdateExample(r.Context(), &example, existing.UpdateTS)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			writeError(w, http.StatusConflict, "Example was modified by another user", logger)
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Example not found", logger)
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
	wrapper := model.DataWrapper{Data: example}
	writeJSON(w, http.StatusOK, wrapper, logger)
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

	jwtClaims := sec.GetClaims()

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
	now := float64(time.Now().UnixNano()) / 1e9
	deletedID, err := h.exampleRepo.DeleteExample(r.Context(), exampleID, jwtClaims.SecurityContext, updateTS, now, jwtClaims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			writeError(w, http.StatusConflict, "Example was modified by another user", logger)
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Example not found", logger)
			return
		}
		logger.Error("example", "DLF0", "Failed to delete example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to delete example", logger)
		return
	}

	data := map[string]interface{}{"message": "Example deleted successfully", "example_id": deletedID}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("example", "DLD0", "Example deletion completed")
}

// GetExampleList handles GET /exampleList?exampleIdList={comma-separated}
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

	jwtClaims := sec.GetClaims()

	// Parse query parameters
	exampleIdListStr := r.URL.Query().Get("exampleIdList")
	if exampleIdListStr == "" {
		writeError(w, http.StatusBadRequest, "exampleIdList parameter is required", logger)
		return
	}

	idStrs := strings.Split(exampleIdListStr, ",")
	ids := make([]int, 0, len(idStrs))
	for _, idStr := range idStrs {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid id in exampleIdList parameter", logger)
			return
		}
		ids = append(ids, id)
	}

	// Get examples (repo filters by security context)
	examples, err := h.exampleRepo.GetExampleList(r.Context(), ids, jwtClaims.SecurityContext)
	if err != nil {
		logger.Error("example", "LIF0", "Failed to get examples: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve examples", logger)
		return
	}

	// Return as JSON
	wrapper := model.DataWrapper{Data: examples}
	writeJSON(w, http.StatusOK, wrapper, logger)

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

	jwtClaims := sec.GetClaims()

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
	userSecurityContext := jwtClaims.SecurityContext
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
	now := float64(time.Now().UnixNano()) / 1e9
	createTS := now
	updateTS := now

	// Set metadata for all
	for i := range examples {
		examples[i].CreateTS = createTS
		examples[i].CreateUserID = jwtClaims.UserID
		examples[i].UpdateTS = updateTS
		examples[i].UpdateUserID = jwtClaims.UserID
	}

	// Create in transaction
	createdExamples, err := h.exampleRepo.CreateExampleList(r.Context(), examplePtrs)
	if err != nil {
		logger.Error("example", "BCF0", "Failed to create examples: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to create examples", logger)
		return
	}

	// Return the created examples with IDs
	wrapper := model.DataWrapper{Data: createdExamples}
	writeJSON(w, http.StatusCreated, wrapper, logger)

	logger.Info("example", "BCR0", fmt.Sprintf("Created %d examples", len(createdExamples)))
}

// UpdateExampleList handles PUT /exampleList
func (h *ExampleHandler) UpdateExampleList(w http.ResponseWriter, r *http.Request) {
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

	jwtClaims := sec.GetClaims()

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

	// Convert to slice of pointers and validate
	examplePtrs := make([]*model.ExampleModel, len(examples))
	for i := range examples {
		examplePtrs[i] = &examples[i]
		if examplePtrs[i].SecurityContext != jwtClaims.SecurityContext {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Example %d has invalid security context", i), logger)
			return
		}
		if examplePtrs[i].ExampleID == 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Example %d missing example_id", i), logger)
			return
		}
	}

	// Generate update timestamp
	now := float64(time.Now().UnixNano()) / 1e9
	for i := range examples {
		examples[i].UpdateTS = now
		examples[i].UpdateUserID = jwtClaims.UserID
	}

	// Update in transaction
	err := h.exampleRepo.UpdateExampleList(r.Context(), examplePtrs)
	if err != nil {
		logger.Error("example", "BUF0", "Failed to update examples: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to update examples", logger)
		return
	}

	// Return success
	data := map[string]string{"message": "Examples updated successfully"}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)

	logger.Info("example", "BUR0", fmt.Sprintf("Updated %d examples", len(examples)))
}

// DeleteExampleList handles DELETE /exampleList?exampleIdList={comma-separated}&updateTs={comma-separated}
func (h *ExampleHandler) DeleteExampleList(w http.ResponseWriter, r *http.Request) {
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

	jwtClaims := sec.GetClaims()

	// Parse query parameters
	exampleIdListStr := r.URL.Query().Get("exampleIdList")
	updateTsStr := r.URL.Query().Get("updateTs")
	if exampleIdListStr == "" {
		writeError(w, http.StatusBadRequest, "exampleIdList parameter is required", logger)
		return
	}
	if updateTsStr == "" {
		writeError(w, http.StatusBadRequest, "updateTs parameter is required", logger)
		return
	}

	idStrs := strings.Split(exampleIdListStr, ",")
	updateTsStrs := strings.Split(updateTsStr, ",")
	if len(idStrs) != len(updateTsStrs) {
		writeError(w, http.StatusBadRequest, "exampleIdList and updateTs must have the same number of values", logger)
		return
	}

	ids := make([]int, 0, len(idStrs))
	updateTs := make([]float64, 0, len(updateTsStrs))
	for i, idStr := range idStrs {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid id in exampleIdList parameter", logger)
			return
		}
		ids = append(ids, id)

		ts, err := strconv.ParseFloat(strings.TrimSpace(updateTsStrs[i]), 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid updateTs value", logger)
			return
		}
		updateTs = append(updateTs, ts)
	}

	if len(ids) == 0 {
		writeError(w, http.StatusBadRequest, "At least one id required", logger)
		return
	}

	// Generate delete timestamp
	now := float64(time.Now().UnixNano()) / 1e9

	// Delete in transaction
	deletedIDs, err := h.exampleRepo.DeleteExampleList(r.Context(), ids, updateTs, jwtClaims.SecurityContext, now, jwtClaims.UserID)
	if err != nil {
		logger.Error("example", "BDF0", "Failed to delete examples: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to delete examples", logger)
		return
	}

	// Return success
	data := map[string]interface{}{"message": "Examples deleted successfully", "example_ids": deletedIDs}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)

	logger.Info("example", "BDR0", fmt.Sprintf("Deleted %d examples", len(ids)))
}
