package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// 23505 is the caller asking for a row that already exists: ErrDuplicate,
// a 409. The check has to survive wrapping.
func TestIsUniqueViolation(t *testing.T) {
	unique := &pgconn.PgError{
		Code:           "23505",
		Message:        "duplicate key value violates unique constraint",
		ConstraintName: "app_user_user_name_key",
	}

	if !isUniqueViolation(unique) {
		t.Error("a bare 23505 is not recognised")
	}
	if !isUniqueViolation(fmt.Errorf("failed: %w", unique)) {
		t.Error("a wrapped 23505 is not recognised")
	}
	if isUniqueViolation(&pgconn.PgError{Code: "23503"}) {
		t.Error("23503 (foreign key) recognised as a unique violation")
	}
	if isUniqueViolation(nil) {
		t.Error("nil recognised as a unique violation")
	}
	if isUniqueViolation(errors.New("connection refused")) {
		t.Error("a plain error recognised as a unique violation")
	}
}

// errors.Is has to find the sentinel (it decides the status) and errors.As
// the message (it reaches the wire), before and after a wrap.
func TestClientMessageErrorCarriesBoth(t *testing.T) {
	err := clientMessage(
		ErrInUse, "cannot delete group: %d users are assigned to it", 3,
	)

	if !errors.Is(err, ErrInUse) {
		t.Error("the sentinel is not reachable with errors.Is")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("matched a sentinel it does not carry")
	}
	var clientErr *ClientMessageError
	if !errors.As(err, &clientErr) {
		t.Fatal("the message is not reachable with errors.As")
	}
	if clientErr.Message != "cannot delete group: 3 users are assigned to it" {
		t.Errorf("message not formatted: %q", clientErr.Message)
	}
	if err.Error() != clientErr.Message {
		t.Errorf("Error() is not the message: %q", err.Error())
	}

	wrapped := fmt.Errorf("failed to delete: %w", err)
	if !errors.Is(wrapped, ErrInUse) || !errors.As(wrapped, &clientErr) {
		t.Error("wrapping loses the sentinel or the message")
	}
}
