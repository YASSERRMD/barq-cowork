// Package domain contains pure business types and repository interfaces.
// It has zero external dependencies: only the Go standard library.
package domain

import "errors"

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("already exists")
	ErrValidation = errors.New("validation error")
)

// ValidationError wraps ErrValidation with a human-readable field message.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func (e *ValidationError) Unwrap() error { return ErrValidation }
