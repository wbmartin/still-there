package repository

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// The sentinel set. A failure the caller can act on is one of these,
// returned bare or wrapped with %w, never a bare fmt.Errorf whose text a
// handler would have to match. handler.mapRepoError maps them once.
var (
	// ErrNotFound is "no live row with that id in this tenant". A row of
	// another security context is not found, never forbidden.
	ErrNotFound = errors.New("not found")
	// ErrConflict is an optimistic-locking failure: the row exists in this
	// tenant but its update_ts is not the one the caller read.
	ErrConflict = errors.New("conflict")
	// ErrDeleted is "the row is there but soft-deleted". Answered like
	// ErrNotFound, so the caller cannot tell the id once existed.
	ErrDeleted = errors.New("deleted")
	// ErrExpectationFailed is a bulk write whose rows did not all match.
	ErrExpectationFailed = errors.New("expectation failed")
	// ErrDuplicate is a unique-constraint violation: the caller's error.
	ErrDuplicate = errors.New("duplicate")
	// ErrInUse is a delete refused because live rows still reference the
	// row (DeleteAppUserGroup while the group has members).
	ErrInUse = errors.New("in use")
	// ErrValidation is a request value the repository refuses (CreateUser's
	// user-name character set).
	ErrValidation = errors.New("validation")
)

// ClientMessageError carries a sentinel with a message that is safe to put
// on the wire, for the failures whose detail is the useful half of the
// answer: which rule a value broke, how many members hold a group open.
// errors.Is finds the sentinel; errors.As finds the message.
type ClientMessageError struct {
	Sentinel error
	Message  string
}

func (e *ClientMessageError) Error() string { return e.Message }

func (e *ClientMessageError) Unwrap() error { return e.Sentinel }

// clientMessage builds a ClientMessageError for sentinel.
func clientMessage(sentinel error, format string, argList ...any) error {
	return &ClientMessageError{
		Sentinel: sentinel,
		Message:  fmt.Sprintf(format, argList...),
	}
}

// uniqueViolationSQLState is PostgreSQL 23505, raised when an INSERT or UPDATE
// breaks a unique constraint; repositories return it as ErrDuplicate.
const uniqueViolationSQLState = "23505"

// isUniqueViolation reports whether err (anywhere in its chain) is a
// PostgreSQL unique-constraint violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationSQLState
}
