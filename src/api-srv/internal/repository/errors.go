package repository

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrDeleted           = errors.New("deleted")
	ErrExpectationFailed = errors.New("expectation failed")
)
