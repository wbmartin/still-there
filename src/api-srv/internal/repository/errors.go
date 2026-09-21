package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// uniqueViolationSQLState is PostgreSQL 23505, raised when an INSERT or UPDATE
// breaks a unique constraint. Mapped to ErrDuplicate so handlers can answer
// 409 rather than leaking a 500 for what is an ordinary client mistake.
const uniqueViolationSQLState = "23505"

var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrDeleted           = errors.New("deleted")
	ErrExpectationFailed = errors.New("expectation failed")
	ErrDuplicate         = errors.New("duplicate")
)

// isUniqueViolation reports whether err (anywhere in its chain) is a
// PostgreSQL unique-constraint violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationSQLState
}
