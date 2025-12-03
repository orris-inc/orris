package user

import (
	"github.com/orris-inc/orris/internal/shared/errors"
)

// DomainError represents a user domain-specific error
type DomainError struct {
	*errors.AppError
}

// NewDomainError creates a new user domain error
func NewDomainError(message string, details ...string) *DomainError {
	detail := ""
	if len(details) > 0 {
		detail = details[0]
	}

	return &DomainError{
		AppError: errors.NewValidationError(message, detail),
	}
}

// Error implements the error interface
func (e *DomainError) Error() string {
	return e.AppError.Error()
}
