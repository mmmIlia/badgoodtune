package domain

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound           = errors.New("resource not found")
	ErrConflict           = errors.New("resource conflict")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrPreconditionFailed = errors.New("precondition failed")
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAlbumHasReviews    = errors.New("album has reviews")
)

// Violation describes one invalid input field without coupling validation to a transport.
type Violation struct {
	Field       string
	Description string
}

// ValidationError groups all deterministic violations found in one validation pass.
type ValidationError struct {
	Violations []Violation
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Violations) == 0 {
		return "validation failed"
	}

	parts := make([]string, 0, len(e.Violations))
	for _, violation := range e.Violations {
		parts = append(parts, fmt.Sprintf("%s: %s", violation.Field, violation.Description))
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

func NewValidationError(violations ...Violation) error {
	if len(violations) == 0 {
		return nil
	}
	return &ValidationError{Violations: violations}
}
