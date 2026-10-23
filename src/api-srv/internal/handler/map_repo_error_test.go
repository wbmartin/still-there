package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"stillthere-api/internal/repository"
	"stillthere-api/internal/security"
)

// answerRepoError runs mapRepoError and returns the status and the client
// message it wrote.
func answerRepoError(t *testing.T, err error) (int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	appHelper := security.NewAppHelper(w, r, testServices(t), "Test")
	mapRepoError(appHelper, w, err, "Failed to do the thing", "THING_FAIL")
	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return w.Code, body["error"]
}

// mapRepoError is the only place a repository failure picks a status. Each
// sentinel answers the same bare or wrapped, so no handler needs its text.
func TestMapRepoError(t *testing.T) {
	caseList := []struct {
		sentinel error
		status   int
		msg      string
	}{
		{repository.ErrNotFound, http.StatusNotFound, "Not found"},
		// Soft-deleted reads as missing: the caller cannot tell the id
		// once existed.
		{repository.ErrDeleted, http.StatusNotFound, "Not found"},
		{repository.ErrConflict, http.StatusConflict,
			"The record was modified by another user"},
		{repository.ErrDuplicate, http.StatusConflict,
			"A record with that name already exists"},
		{repository.ErrInUse, http.StatusConflict,
			"The record is still in use"},
		{repository.ErrValidation, http.StatusBadRequest,
			"The request is not valid"},
		{repository.ErrExpectationFailed, http.StatusExpectationFailed,
			"One or more records could not be updated " +
				"due to concurrent modifications"},
	}

	for _, tc := range caseList {
		wrapped := fmt.Errorf("failed to touch row 7: %w", tc.sentinel)
		for name, err := range map[string]error{
			"bare": tc.sentinel, "wrapped": wrapped,
		} {
			t.Run(tc.sentinel.Error()+" "+name, func(t *testing.T) {
				status, msg := answerRepoError(t, err)
				assert.Equal(t, tc.status, status)
				assert.Equal(t, tc.msg, msg)
			})
		}
	}
}

// Anything that is not a sentinel is the server's fault: 500 with the
// handler's words, and the error's own text stays in the log.
func TestMapRepoErrorUnknownIs500WithoutLeak(t *testing.T) {
	err := fmt.Errorf("failed to get user: %w",
		errors.New(`relation "app_user" does not exist`))
	status, msg := answerRepoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, "Failed to do the thing", msg)
	assert.NotContains(t, msg, "app_user")
}

// A ClientMessageError's text reaches the wire, bare or wrapped; an empty
// one falls back to the generic wording.
func TestMapRepoErrorCarriesTheRepositoryMessage(t *testing.T) {
	inUse := &repository.ClientMessageError{
		Sentinel: repository.ErrInUse,
		Message:  "cannot delete group: 2 users are assigned to it",
	}
	status, msg := answerRepoError(t, fmt.Errorf("delete: %w", inUse))
	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, inUse.Message, msg)

	invalid := &repository.ClientMessageError{
		Sentinel: repository.ErrValidation,
		Message:  "username must contain only alphanumeric characters",
	}
	status, msg = answerRepoError(t, invalid)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, invalid.Message, msg)

	_, msg = answerRepoError(t, &repository.ClientMessageError{
		Sentinel: repository.ErrValidation,
	})
	assert.Equal(t, "The request is not valid", msg)
}
