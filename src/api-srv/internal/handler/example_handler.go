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
func NewExampleHandler(
	appServices *app_lib.AppServices,
	exampleRepo *repository.ExampleRepository,
) *ExampleHandler {
	return &ExampleHandler{
		appsrv:      appServices,
		exampleRepo: exampleRepo,
	}
}

// RetrieveExample handles GET /example
func (h *ExampleHandler) RetrieveExample(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleRead)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("READ_EX_START", "ReadExample invoked")

	jwtClaims := app.GetClaims()

	// Parse query parameters
	exampleID, err := GetIntFromQuery(r, "exampleId")
	if err != nil {
		app.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
			"PARSE_QUERY_FAIL",
			nil,
		)
		return
	}

	fmt.Printf(
		"DEBUG: RetrieveExample: exampleID=%d, jwtClaims.SecurityContext=%d\n",
		exampleID,
		jwtClaims.SecurityContext,
	)

	// Read example
	example, err := h.exampleRepo.RetrieveExample(
		r.Context(),
		exampleID,
		jwtClaims.SecurityContext,
	)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			app.WriteError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("Example not found: example_id=%d", exampleID),
				"NOT_FOUND_EX",
				err,
			)
			return
		}
		if errors.Is(err, repository.ErrDeleted) {
			app.WriteError(
				w,
				http.StatusGone,
				fmt.Sprintf(
					"Example has been deleted: example_id=%d",
					exampleID,
				),
				"EXAMPLE_GONE",
				err,
			)
			return
		}
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf("Failed to retrieve example: example_id=%d", exampleID),
			"GET_EX_FAIL",
			err,
		)
		return
	}

	app.Debug("EX_RETRIEVED", fmt.Sprintf("Example retrieved: %+v", example))
	wrapper := model.DataWrapper{Data: example}
	app.WriteJSON(w, http.StatusOK, wrapper)
	app.Info("GET_EX_DONE", "Example retrieval completed")
}

// CreateExample handles POST /example
func (h *ExampleHandler) CreateExample(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleCreate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("EX_CR_START", "CreateExample invoked")

	jwtClaims := app.GetClaims()

	// Parse request body
	var example model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&example); err != nil {
		app.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf("Invalid request body for create example: %v", err),
			"BAD_REQUEST",
			nil,
		)
		return
	}

	// Validate required fields
	if example.ExampleText == "" {
		app.WriteError(
			w,
			http.StatusBadRequest,
			"example_text is required for create example",
			"BAD_REQUEST",
			nil,
		)
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
	createdExample, err := h.exampleRepo.CreateExample(
		r.Context(),
		&example,
		jwtClaims.SecurityContext,
	)
	if err != nil {
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"Failed to create example: example_text='%s' len=%d",
				example.ExampleText,
				len(example.ExampleText),
			),
			"CREATE_EX_FAIL",
			err,
		)
		return
	}

	app.Debug("EX_CREATED", fmt.Sprintf("Example created: %+v", createdExample))
	wrapper := model.DataWrapper{Data: createdExample}
	writeJSON(w, http.StatusCreated, wrapper, h.appsrv.Logger)
	app.Info(
		"EX_CR_DONE",
		fmt.Sprintf(
			"Example creation completed for exampleId=%d",
			createdExample.ExampleID,
		),
	)
}

// UpdateExample handles PUT /example
func (h *ExampleHandler) UpdateExample(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("EX_UP_START", "UpdateExample invoked")

	jwtClaims := app.GetClaims()

	// Parse request body
	var example model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&example); err != nil {
		app.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf("Invalid request body for update example: %v", err),
			"BAD_REQUEST",
			nil,
		)
		return
	}

	app.Debug(
		"UPDATE_PARSE_REQ",
		fmt.Sprintf("Example update request parsed: %+v", example),
	)

	// Validate required fields
	if example.ExampleID == 0 || example.ExampleText == "" {
		app.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf(
				"example_id and example_text are required for update: example_id=%d, example_text='%s'",
				example.ExampleID,
				example.ExampleText,
			),
			"BAD_REQUEST",
			nil,
		)
		return
	}

	// Validate required fields
	if example.ExampleID == 0 || example.ExampleText == "" {
		app.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf(
				"example_id and example_text are required for update: example_id=%d, example_text='%s'",
				example.ExampleID,
				example.ExampleText,
			),
			"BAD_REQUEST",
			nil,
		)
		return
	}

	app.Info(
		"EX_UP_PARAMS",
		fmt.Sprintf("UpdateExample for exampleId=%d", example.ExampleID),
	)

	// Set update metadata
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
			app.WriteError(
				w,
				http.StatusConflict,
				fmt.Sprintf(
					"Example was modified by another user: example_id=%d",
					example.ExampleID,
				),
				"CONFLICT_ERROR",
				err,
			)
		case errors.Is(err, repository.ErrNotFound):
			app.WriteError(
				w,
				http.StatusNotFound,
				fmt.Sprintf(
					"Example not found: example_id=%d",
					example.ExampleID,
				),
				"NOT_FOUND_EX",
				err,
			)
		case errors.Is(err, repository.ErrDeleted):
			app.WriteError(
				w,
				http.StatusGone,
				fmt.Sprintf(
					"Example has been deleted: example_id=%d",
					example.ExampleID,
				),
				"EXAMPLE_GONE",
				err,
			)
		default:
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to update example: example_id=%d",
					example.ExampleID,
				),
				"UPDATE_EX_FAIL",
				err,
			)
		}
		return
	}

	app.Debug("EX_UPDATED", fmt.Sprintf("Example updated: %+v", example))
	wrapper := model.DataWrapper{Data: example}
	app.WriteJSON(w, http.StatusOK, wrapper)
	app.Info(
		"EX_UP_DONE",
		fmt.Sprintf(
			"Example update completed for exampleId=%d",
			example.ExampleID,
		),
	)
}

// DeleteExample handles DELETE /example
func (h *ExampleHandler) DeleteExample(w http.ResponseWriter, r *http.Request) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleDelete)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("EX_DEL_START", "DeleteExample invoked")

	jwtClaims := app.GetClaims()

	// Parse query parameters
	exampleID, err := GetIntFromQuery(r, "exampleId")
	if err != nil {
		app.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
			"PARSE_QUERY_FAIL",
			nil,
		)
		return
	}

	app.Info(
		"EX_DEL_PARAMS",
		fmt.Sprintf("DeleteExample for exampleId=%d", exampleID),
	)

	updateTS, err := GetFloatFromQuery(r, "updateTs")
	if err != nil {
		app.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
			"PARSE_QUERY_FAIL",
			nil,
		)
		return
	}

	// Caller information automatically added to log messages

	// Delete example with optimistic locking
	now := CurrentTimestamp()
	deletedID, err := h.exampleRepo.DeleteExample(
		r.Context(),
		exampleID,
		jwtClaims.SecurityContext,
		updateTS,
		now,
		jwtClaims.UserID,
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			app.WriteError(
				w,
				http.StatusConflict,
				fmt.Sprintf(
					"Example was modified by another user: example_id=%d",
					exampleID,
				),
				"CONFLICT_ERROR",
				err,
			)
		case errors.Is(err, repository.ErrNotFound):
			app.WriteError(
				w,
				http.StatusNotFound,
				fmt.Sprintf("Example not found: example_id=%d", exampleID),
				"NOT_FOUND_EX",
				err,
			)
		case errors.Is(err, repository.ErrDeleted):
			app.WriteError(
				w,
				http.StatusGone,
				fmt.Sprintf(
					"Example has been deleted: example_id=%d",
					exampleID,
				),
				"EXAMPLE_GONE",
				err,
			)
		default:
			app.WriteError(
				w,
				http.StatusInternalServerError,
				fmt.Sprintf(
					"Failed to delete example: example_id=%d",
					exampleID,
				),
				"DELETE_EX_FAIL",
				err,
			)
		}
		return
	}

	data := map[string]interface{}{
		"message":    "Example deleted successfully",
		"example_id": deletedID,
	}
	wrapper := model.DataWrapper{Data: data}
	app.WriteJSON(w, http.StatusOK, wrapper)
	app.Info(
		"EX_DEL_DONE",
		fmt.Sprintf("Example deletion completed for exampleId=%d", deletedID),
	)
}

// RetrieveExampleList handles GET /exampleList?exampleIdList={comma-separated} (optional, if empty returns all)
func (h *ExampleHandler) RetrieveExampleList(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleRead)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("EX_LIST_START", "ReadExampleList invoked")

	jwtClaims := app.GetClaims()

	// Parse query parameters
	idList, err := GetCommaSeparatedIntListFromQuery(r, "exampleIdList")
	if err != nil {
		app.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
			"PARSE_QUERY_FAIL",
			nil,
		)
		return
	}

	afterUpdateTs, err := GetOptionalFloatFromQuery(r, "afterUpdateTs")
	if err != nil {
		app.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
			"PARSE_QUERY_FAIL",
			nil,
		)
		return
	}

	// If idList is empty, ReadExampleList returns all or filtered by afterUpdateTs
	examples, err := h.exampleRepo.RetrieveExampleList(
		r.Context(),
		idList,
		jwtClaims.SecurityContext,
		afterUpdateTs,
	)
	if err != nil {
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"Failed to retrieve examples: idList=%v, afterUpdateTs=%v",
				idList,
				afterUpdateTs,
			),
			"LIST_EX_FAIL",
			err,
		)
		return
	}

	// Ensure examples is never nil
	if examples == nil {
		examples = []*model.ExampleModel{}
	}

	// Return as JSON
	wrapper := model.DataWrapper{Data: examples}
	app.WriteJSON(w, http.StatusOK, wrapper)

	app.Info("EX_LIST_RET", fmt.Sprintf("Returned %d examples", len(examples)))
}

// CreateExampleList handles POST /exampleList
func (h *ExampleHandler) CreateExampleList(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.Info("EX_BLK_CR_ST", "CreateExampleList invoked")
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
		app.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf(
				"Invalid request body for bulk create examples: %v",
				err,
			),
			"BAD_REQUEST",
			nil,
		)
		return
	}

	if len(exampleList) == 0 {
		app.WriteError(
			w,
			http.StatusBadRequest,
			"At least one example required for bulk create",
			"BAD_REQUEST",
			nil,
		)
		return
	}

	// Validate required fields
	for i, example := range exampleList {
		if example.ExampleText == "" {
			app.WriteError(
				w,
				http.StatusBadRequest,
				fmt.Sprintf(
					"Example %d missing example_text in bulk create",
					i,
				),
				"BAD_REQUEST",
				nil,
			)
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
		exampleList[i].SecurityContext = jwtClaims.SecurityContext
		exampleList[i].CreateTS = createTS
		exampleList[i].CreateUserID = jwtClaims.UserID
		exampleList[i].UpdateTS = updateTS
		exampleList[i].UpdateUserID = jwtClaims.UserID
	}

	// Create in transaction
	createdExamples, err := h.exampleRepo.CreateExampleList(
		r.Context(),
		examplePtrList,
		jwtClaims.SecurityContext,
	)
	if err != nil {
		exampleTexts := make([]string, len(examplePtrList))
		for i, ex := range examplePtrList {
			exampleTexts[i] = ex.ExampleText
		}
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"Failed to create examples: count=%d, texts=%v",
				len(examplePtrList),
				exampleTexts,
			),
			"BULK_CREATE_FAIL",
			err,
		)
		return
	}

	// Return the created examples with IDs
	wrapper := model.DataWrapper{Data: createdExamples}
	writeJSON(w, http.StatusCreated, wrapper, h.appsrv.Logger)

	createdIds := make([]int, len(createdExamples))
	for i, ex := range createdExamples {
		createdIds[i] = ex.ExampleID
	}
	app.Info(
		"EX_BLK_CR_RET",
		fmt.Sprintf(
			"Created %d examples: ids=%v",
			len(createdExamples),
			createdIds,
		),
	)
}

// UpdateExampleList handles PUT /exampleList
func (h *ExampleHandler) UpdateExampleList(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleUpdate)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("EX_BLK_UP_ST", "UpdateExampleList invoked")

	jwtClaims := app.GetClaims()

	// Parse request body
	var exampleList []model.ExampleModel
	if err := json.NewDecoder(r.Body).Decode(&exampleList); err != nil {
		app.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf(
				"Invalid request body for bulk update examples: %v",
				err,
			),
			"BAD_REQUEST",
			nil,
		)
		return
	}

	if len(exampleList) == 0 {
		app.WriteError(
			w,
			http.StatusBadRequest,
			"At least one example required for bulk update",
			"BAD_REQUEST",
			nil,
		)
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
		if examplePtrList[i].ExampleID == 0 {
			app.WriteError(
				w,
				http.StatusBadRequest,
				fmt.Sprintf("Example %d missing example_id in bulk update", i),
				"BAD_REQUEST",
				nil,
			)
			return
		}
		exampleList[i].UpdateTS = now
		exampleList[i].UpdateUserID = jwtClaims.UserID
	}

	// Update in transaction
	err := h.exampleRepo.UpdateExampleList(
		r.Context(),
		examplePtrList,
		originalUpdateTSList,
		jwtClaims.SecurityContext,
	)
	if err != nil {
		if errors.Is(err, repository.ErrExpectationFailed) {
			app.WriteError(
				w,
				http.StatusExpectationFailed,
				fmt.Sprintf(
					"One or more examples could not be updated due to concurrent modifications: count=%d",
					len(examplePtrList),
				),
				"EXPECTATION_FAIL",
				err,
			)
			return
		}
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"Failed to update examples: count=%d",
				len(examplePtrList),
			),
			"BULK_UPDATE_FAIL",
			err,
		)
		return
	}

	// Return success
	data := map[string]string{"message": "Examples updated successfully"}
	wrapper := model.DataWrapper{Data: data}
	app.WriteJSON(w, http.StatusOK, wrapper)

	app.Info(
		"EX_BLK_UP_RET",
		fmt.Sprintf("Updated %d examples", len(exampleList)),
	)
}

// DeleteExampleList handles DELETE /exampleList?exampleIdList={comma-separated}&updateTs={comma-separated}
func (h *ExampleHandler) DeleteExampleList(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Security checks
	app := app_lib.NewAppHelper(w, r, h.appsrv)
	app.CheckJWT()
	app.CheckRateLimitAndUserBlock()
	app.CheckCSRF()
	app.CheckAuthorization(model.GrantExampleDelete)

	if !app.CheckAllChecksComplete() {
		return
	}

	app.Info("EX_BLK_DEL_ST", "DeleteExampleList invoked")

	jwtClaims := app.GetClaims()

	// Parse query parameters
	idList, err := GetCommaSeparatedIntListFromQuery(r, "exampleIdList")
	if err != nil {
		app.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
			"PARSE_QUERY_FAIL",
			nil,
		)
		return
	}

	updateTimestampList, err := GetCommaSeparatedFloats(r, "updateTs")
	if err != nil {
		app.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
			"PARSE_QUERY_FAIL",
			nil,
		)
		return
	}

	if len(idList) != len(updateTimestampList) {
		app.WriteError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf(
				"exampleIdList and updateTs must have the same number of values: idList=%d, updateTs=%d",
				len(idList),
				len(updateTimestampList),
			),
			"BAD_REQUEST",
			nil,
		)
		return
	}

	if len(idList) == 0 {
		app.WriteError(
			w,
			http.StatusBadRequest,
			"At least one id required for bulk delete",
			"BAD_REQUEST",
			nil,
		)
		return
	}

	// Generate delete timestamp
	now := CurrentTimestamp()

	// Delete in transaction
	deletedIDList, err := h.exampleRepo.DeleteExampleList(
		r.Context(),
		idList,
		updateTimestampList,
		jwtClaims.SecurityContext,
		now,
		jwtClaims.UserID,
	)
	if err != nil {
		app.WriteError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"Failed to delete examples: count=%d, ids=%v",
				len(idList),
				idList,
			),
			"BULK_DELETE_FAIL",
			err,
		)
		return
	}

	// Return success
	data := map[string]interface{}{
		"message":       "Examples deleted successfully",
		"exampleIdList": deletedIDList,
	}
	wrapper := model.DataWrapper{Data: data}
	app.WriteJSON(w, http.StatusOK, wrapper)

	app.Info(
		"EX_BLK_DEL_RET",
		fmt.Sprintf(
			"Deleted %d examples: ids=%v",
			len(deletedIDList),
			deletedIDList,
		),
	)
}
