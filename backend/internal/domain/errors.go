package domain

import "errors"

// Sentinel errors shared across layers. Repositories translate database
// errors into these; the HTTP layer translates them into status codes.
var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// ValidationError carries a user-facing message for invalid input.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// Validation constructs a *ValidationError.
func Validation(msg string) error { return &ValidationError{Message: msg} }

// IsValidation reports whether err is a validation error.
func IsValidation(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}
