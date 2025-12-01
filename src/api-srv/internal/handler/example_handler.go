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
		logger.Error("GTF0", "Failed to get example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve example", logger)
		return
	}

	logger.Debug("RET0", fmt.Sprintf("Example retrieved: %+v", example))
	wrapper := model.DataWrapper{Data: example}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("GTD0", "Example retrieval completed")
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

	logger.Debug("CRP0", fmt.Sprintf("Example creation request parsed: %+v", example))

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
		logger.Error("CRF0", "Failed to create example: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to create example", logger)
		return
	}

	logger.Debug("CRD0", fmt.Sprintf("Example created: %+v", createdExample))
	wrapper := model.DataWrapper{Data: createdExample}
	writeJSON(w, http.StatusCreated, wrapper, logger)
	logger.Info("CRD1", "Example creation completed")
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

	logger.Debug("UPP0", fmt.Sprintf("Example update request parsed: %+v", example))

	// Validate required fields
	if example.ExampleID == 0 || example.ExampleText == "" {
		writeError(w, http.StatusBadRequest, "example_id and example_text are required", logger)
		return
	}

	// Set security context and update metadata
	clientUpdateTS := example.UpdateTS
	now := float64(time.Now().UnixNano()) / 1e9
	example.SecurityContext = jwtClaims.SecurityContext
	example.UpdateTS = now
	example.UpdateUserID = jwtClaims.UserID

	// Caller information automatically added to log messages

	// Update example with optimistic locking
	err := h.exampleRepo.UpdateExample(r.Context(), &example, clientUpdateTS)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			writeError(w, http.StatusConflict, "Example was modified by another user", logger)
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "Example not found", logger)
		case errors.Is(err, repository.ErrDeleted):
			writeError(w, http.StatusGone, "Example has been deleted", logger)
		default:
			logger.Error("UPF0", "Failed to update example: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Failed to update example", logger)
		}
		return
	}

	// Return updated example (keeping original create fields)
	example.CreateTS = example.CreateTS         // From request
	example.CreateUserID = example.CreateUserID // From request

	logger.Debug("UPD0", fmt.Sprintf("Example updated: %+v", example))
	wrapper := model.DataWrapper{Data: example}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("UPD1", "Example update completed")
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
		switch {
		case errors.Is(err, repository.ErrConflict):
			writeError(w, http.StatusConflict, "Example was modified by another user", logger)
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "Example not found", logger)
		default:
			logger.Error("DLF0", "Failed to delete example: "+err.Error())
			writeError(w, http.StatusInternalServerError, "Failed to delete example", logger)
		}
		return
	}

	data := map[string]interface{}{"message": "Example deleted successfully", "example_id": deletedID}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("DLD0", "Example deletion completed")
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

	idStrList := strings.Split(exampleIdListStr, ",")
	idList := make([]int, 0, len(idStrList))
	for _, idStr := range idStrList {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid id in exampleIdList parameter", logger)
			return
		}
		idList = append(idList, id)
	}
	examples, err := h.exampleRepo.GetExampleList(r.Context(), idList, jwtClaims.SecurityContext)
	if err != nil {
		logger.Error("LIF0", "Failed to get examples: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to retrieve examples", logger)
		return
	}

	// Return as JSON
	wrapper := model.DataWrapper{Data: examples}
	writeJSON(w, http.StatusOK, wrapper, logger)

	logger.Info("LIR0", fmt.Sprintf("Returned %d examples", len(examples)))
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
	var exampleList []model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&exampleList); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger)
		return
	}

	if len(exampleList) == 0 {
		writeError(w, http.StatusBadRequest, "At least one example required", logger)
		return
	}

	// Convert to slice of pointers
	examplePtrList := make([]*model.ExampleModel, len(exampleList))
	for i := range exampleList {
		examplePtrList[i] = &exampleList[i]
	}

	// Validate all examples have the same security context and it's allowed
	userSecurityContext := jwtClaims.SecurityContext
	for i, example := range exampleList {
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
	for i := range exampleList {
		exampleList[i].CreateTS = createTS
		exampleList[i].CreateUserID = jwtClaims.UserID
		exampleList[i].UpdateTS = updateTS
		exampleList[i].UpdateUserID = jwtClaims.UserID
	}

	// Create in transaction
	createdExamples, err := h.exampleRepo.CreateExampleList(r.Context(), examplePtrList)
	if err != nil {
		logger.Error("BCF0", "Failed to create examples: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to create examples", logger)
		return
	}

	// Return the created examples with IDs
	wrapper := model.DataWrapper{Data: createdExamples}
	writeJSON(w, http.StatusCreated, wrapper, logger)

	logger.Info("BCR0", fmt.Sprintf("Created %d examples", len(createdExamples)))
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
	var exampleList []model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&exampleList); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", logger)
		return
	}

	if len(exampleList) == 0 {
		writeError(w, http.StatusBadRequest, "At least one example required", logger)
		return
	}

	// Generate update timestamp
	now := float64(time.Now().UnixNano()) / 1e9

	// Collect original update timestamps, validate, and set new values
	originalUpdateTSList := make([]float64, len(exampleList))
	examplePtrList := make([]*model.ExampleModel, len(exampleList))
	for i := range exampleList {
		examplePtrList[i] = &exampleList[i]
		originalUpdateTSList[i] = exampleList[i].UpdateTS
		if examplePtrList[i].SecurityContext != jwtClaims.SecurityContext {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Example %d has invalid security context", i), logger)
			return
		}
		if examplePtrList[i].ExampleID == 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Example %d missing example_id", i), logger)
			return
		}
		exampleList[i].UpdateTS = now
		exampleList[i].UpdateUserID = jwtClaims.UserID
	}

	// Update in transaction
	err := h.exampleRepo.UpdateExampleList(r.Context(), examplePtrList, originalUpdateTSList)
	if err != nil {
		if errors.Is(err, repository.ErrExpectationFailed) {
			writeError(w, http.StatusExpectationFailed, "One or more examples could not be updated due to concurrent modifications", logger)
			return
		}
		logger.Error("BUF0", "Failed to update examples: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to update examples", logger)
		return
	}

	// Return success
	data := map[string]string{"message": "Examples updated successfully"}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)

	logger.Info("BUR0", fmt.Sprintf("Updated %d examples", len(exampleList)))
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

	idStrList := strings.Split(exampleIdListStr, ",")
	updateTsStrList := strings.Split(updateTsStr, ",")
	if len(idStrList) != len(updateTsStrList) {
		writeError(w, http.StatusBadRequest, "exampleIdList and updateTs must have the same number of values", logger)
		return
	}

	idList := make([]int, 0, len(idStrList))
	updateTimestampList := make([]float64, 0, len(updateTsStrList))
	for i, idStr := range idStrList {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid id in exampleIdList parameter", logger)
			return
		}
		idList = append(idList, id)

		ts, err := strconv.ParseFloat(strings.TrimSpace(updateTsStrList[i]), 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid updateTs value", logger)
			return
		}
		updateTimestampList = append(updateTimestampList, ts)
	}

	if len(idList) == 0 {
		writeError(w, http.StatusBadRequest, "At least one id required", logger)
		return
	}

	// Generate delete timestamp
	now := float64(time.Now().UnixNano()) / 1e9

	// Delete in transaction
	deletedIDList, err := h.exampleRepo.DeleteExampleList(r.Context(), idList, updateTimestampList, jwtClaims.SecurityContext, now, jwtClaims.UserID)
	if err != nil {
		logger.Error("BDF0", "Failed to delete examples: "+err.Error())
		writeError(w, http.StatusInternalServerError, "Failed to delete examples", logger)
		return
	}

	// Return success
	data := map[string]interface{}{"message": "Examples deleted successfully", "example_ids": deletedIDList}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)

	logger.Info("BDR0", fmt.Sprintf("Deleted %d examples", len(idList)))
}
