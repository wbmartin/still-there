package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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

// ReadExample handles GET /example
func (h *ExampleHandler) ReadExample(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("RTS0", "ReadExample invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleRead)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Parse query parameters
	exampleID, err := GetIntFromQuery(r, "exampleId")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	// Caller information automatically added to log messages

	// Read example
	example, err := h.exampleRepo.ReadExample(r.Context(), exampleID, jwtClaims.SecurityContext)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			app.WriteError(w, http.StatusNotFound, "Example not found", "ERR", nil)
			return
		}
		if errors.Is(err, repository.ErrDeleted) {
			app.WriteError(w, http.StatusGone, "Example has been deleted", "GON0", nil)
			return
		}
		app.WriteError(w, http.StatusInternalServerError, "Failed to retrieve example", "GTF0", err)
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
	logger.Info("CRS0", "CreateExample invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleCreate)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Parse request body
	var example model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&example); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "BRQ0", nil)
		return
	}

	logger.Debug("CRP0", fmt.Sprintf("Example creation request parsed: %+v", example))

	// Validate required fields
	if example.ExampleText == "" {
		app.WriteError(w, http.StatusBadRequest, "example_text is required", "BRQ0", nil)
		return
	}

	// Set security context and creation metadata
	now := CurrentTimestamp()
	example.SecurityContext = jwtClaims.SecurityContext
	example.CreateTS = now
	example.CreateUserID = jwtClaims.UserID
	example.UpdateTS = now
	example.UpdateUserID = jwtClaims.UserID

	// Caller information automatically added to log messages

	// Create example
	createdExample, err := h.exampleRepo.CreateExample(r.Context(), &example)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError, "Failed to create example", "CRF0", err)
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
	logger.Info("UPS0", "UpdateExample invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Parse request body
	var example model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&example); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "BRQ0", nil)
		return
	}

	logger.Debug("UPP0", fmt.Sprintf("Example update request parsed: %+v", example))

	// Validate required fields
	if example.ExampleID == 0 || example.ExampleText == "" {
		app.WriteError(w, http.StatusBadRequest, "example_id and example_text are required", "BRQ0", nil)
		return
	}

	// Set security context and update metadata
	clientUpdateTS := example.UpdateTS
	now := CurrentTimestamp()
	example.SecurityContext = jwtClaims.SecurityContext
	example.UpdateTS = now
	example.UpdateUserID = jwtClaims.UserID

	// Caller information automatically added to log messages

	// Update example with optimistic locking
	err := h.exampleRepo.UpdateExample(r.Context(), &example, clientUpdateTS)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			app.WriteError(w, http.StatusConflict, "Example was modified by another user", "CNF0", nil)
		case errors.Is(err, repository.ErrNotFound):
			app.WriteError(w, http.StatusNotFound, "Example not found", "NFN0", nil)
		case errors.Is(err, repository.ErrDeleted):
			app.WriteError(w, http.StatusGone, "Example has been deleted", "GON0", nil)
		default:
			app.WriteError(w, http.StatusInternalServerError, "Failed to update example", "UPF0", err)
		}
		return
	}

	logger.Debug("UPD0", fmt.Sprintf("Example updated: %+v", example))
	wrapper := model.DataWrapper{Data: example}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("UPD1", "Example update completed")
}

// DeleteExample handles DELETE /example
func (h *ExampleHandler) DeleteExample(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("DES0", "DeleteExample invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleDelete)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Parse query parameters
	exampleID, err := GetIntFromQuery(r, "exampleId")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	updateTS, err := GetFloatFromQuery(r, "updateTs")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	// Caller information automatically added to log messages

	// Delete example with optimistic locking
	now := CurrentTimestamp()
	deletedID, err := h.exampleRepo.DeleteExample(r.Context(), exampleID, jwtClaims.SecurityContext, updateTS, now, jwtClaims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			app.WriteError(w, http.StatusConflict, "Example was modified by another user", "CNF0", nil)
		case errors.Is(err, repository.ErrNotFound):
			app.WriteError(w, http.StatusNotFound, "Example not found", "NFN0", nil)
		case errors.Is(err, repository.ErrDeleted):
			app.WriteError(w, http.StatusGone, "Example has been deleted", "GON0", nil)
		default:
			app.WriteError(w, http.StatusInternalServerError, "Failed to delete example", "DLF0", err)
		}
		return
	}

	data := map[string]interface{}{"message": "Example deleted successfully", "example_id": deletedID}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)
	logger.Info("DLD0", "Example deletion completed")
}

// ReadExampleList handles GET /exampleList?exampleIdList={comma-separated} (optional, if empty returns all)
func (h *ExampleHandler) ReadExampleList(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("RLS0", "ReadExampleList invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleRead)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Parse query parameters
	idList, err := GetCommaSeparatedIntListFromQuery(r, "exampleIdList")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	afterUpdateTs, err := GetOptionalFloatFromQuery(r, "afterUpdateTs")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	// If idList is empty, ReadExampleList returns all or filtered by afterUpdateTs
	examples, err := h.exampleRepo.ReadExampleList(r.Context(), idList, jwtClaims.SecurityContext, afterUpdateTs)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError, "Failed to retrieve examples", "LIF0", err)
		return
	}

	// Ensure examples is never nil
	if examples == nil {
		examples = []*model.ExampleModel{}
	}

	// Return as JSON
	wrapper := model.DataWrapper{Data: examples}
	writeJSON(w, http.StatusOK, wrapper, logger)

	logger.Info("LIR0", fmt.Sprintf("Returned %d examples", len(examples)))
}

// CreateExampleList handles POST /exampleList
func (h *ExampleHandler) CreateExampleList(w http.ResponseWriter, r *http.Request) {
	logger := h.appsrv.Logger
	logger.Info("CLS0", "CreateExampleList invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleCreate)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Parse request body
	var exampleList []model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&exampleList); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "BRQ0", nil)
		return
	}

	if len(exampleList) == 0 {
		app.WriteError(w, http.StatusBadRequest, "At least one example required", "BRQ0", nil)
		return
	}

	// Validate all examples have the same security context and it's allowed
	userSecurityContext := jwtClaims.SecurityContext
	for i, example := range exampleList {
		if example.SecurityContext != userSecurityContext {
			app.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Example %d has invalid security context", i), "BRQ0", nil)
			return
		}
		if example.ExampleText == "" {
			app.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Example %d missing example_text", i), "BRQ0", nil)
			return
		}
	}

	// Convert to slice of pointers
	examplePtrList := make([]*model.ExampleModel, len(exampleList))
	for i := range exampleList {
		examplePtrList[i] = &exampleList[i]
	}

	// Generate timestamps once
	now := CurrentTimestamp()
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
		app.WriteError(w, http.StatusInternalServerError, "Failed to create examples", "BCF0", err)
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
	logger.Info("ULS0", "UpdateExampleList invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Parse request body
	var exampleList []model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&exampleList); err != nil {
		app.WriteError(w, http.StatusBadRequest, "Invalid request body", "BRQ0", nil)
		return
	}

	if len(exampleList) == 0 {
		app.WriteError(w, http.StatusBadRequest, "At least one example required", "BRQ0", nil)
		return
	}

	// Generate update timestamp
	now := CurrentTimestamp()

	// Collect original update timestamps, validate, and set new values
	originalUpdateTSList := make([]float64, len(exampleList))
	examplePtrList := make([]*model.ExampleModel, len(exampleList))
	for i := range exampleList {
		examplePtrList[i] = &exampleList[i]
		originalUpdateTSList[i] = exampleList[i].UpdateTS
		if examplePtrList[i].SecurityContext != jwtClaims.SecurityContext {
			app.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Example %d has invalid security context", i), "BRQ0", nil)
			return
		}
		if examplePtrList[i].ExampleID == 0 {
			app.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Example %d missing example_id", i), "BRQ0", nil)
			return
		}
		exampleList[i].UpdateTS = now
		exampleList[i].UpdateUserID = jwtClaims.UserID
	}

	// Update in transaction
	err := h.exampleRepo.UpdateExampleList(r.Context(), examplePtrList, originalUpdateTSList)
	if err != nil {
		if errors.Is(err, repository.ErrExpectationFailed) {
			app.WriteError(w, http.StatusExpectationFailed, "One or more examples could not be updated due to concurrent modifications", "EXF0", nil)
			return
		}
		app.WriteError(w, http.StatusInternalServerError, "Failed to update examples", "BUF0", err)
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
	logger.Info("DLS0", "DeleteExampleList invoked")

	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleDelete)

	if !app.CheckAllChecksComplete() {
		return
	}

	jwtClaims := app.GetClaims()

	// Parse query parameters
	idList, err := GetCommaSeparatedIntListFromQuery(r, "exampleIdList")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	updateTimestampList, err := GetCommaSeparatedFloats(r, "updateTs")
	if err != nil {
		app.WriteError(w, http.StatusBadRequest, err.Error(), "PRQ0", nil)
		return
	}

	if len(idList) != len(updateTimestampList) {
		app.WriteError(w, http.StatusBadRequest, "exampleIdList and updateTs must have the same number of values", "BRQ0", nil)
		return
	}

	if len(idList) == 0 {
		app.WriteError(w, http.StatusBadRequest, "At least one id required", "BRQ0", nil)
		return
	}

	// Generate delete timestamp
	now := CurrentTimestamp()

	// Delete in transaction
	deletedIDList, err := h.exampleRepo.DeleteExampleList(r.Context(), idList, updateTimestampList, jwtClaims.SecurityContext, now, jwtClaims.UserID)
	if err != nil {
		app.WriteError(w, http.StatusInternalServerError, "Failed to delete examples", "BDF0", err)
		return
	}

	// Return success
	data := map[string]interface{}{"message": "Examples deleted successfully", "example_ids": deletedIDList}
	wrapper := model.DataWrapper{Data: data}
	writeJSON(w, http.StatusOK, wrapper, logger)

	logger.Info("BDR0", fmt.Sprintf("Deleted %d examples", len(idList)))
}
