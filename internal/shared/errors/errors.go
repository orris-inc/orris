// Package errors provides application-level error types and utilities.
// It defines common error types like validation, not found, conflict, and authorization errors.
package errors

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation_error"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeConflict     ErrorType = "conflict"
	ErrorTypeUnauthorized ErrorType = "unauthorized"
	ErrorTypeForbidden    ErrorType = "forbidden"
	ErrorTypeInternal     ErrorType = "internal_error"
	ErrorTypeBadRequest   ErrorType = "bad_request"
)

// AppError represents an application error with additional context
type AppError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
	Code    int       `json:"code"`
	Details string    `json:"details,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// newAppError is an internal helper that creates AppError with common logic.
func newAppError(errType ErrorType, httpCode int, message string, details ...string) *AppError {
	detail := ""
	if len(details) > 0 {
		detail = details[0]
	}
	return &AppError{
		Type:    errType,
		Message: message,
		Code:    httpCode,
		Details: detail,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(message string, details ...string) *AppError {
	return newAppError(ErrorTypeValidation, http.StatusBadRequest, message, details...)
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string, details ...string) *AppError {
	return newAppError(ErrorTypeNotFound, http.StatusNotFound, message, details...)
}

// NewConflictError creates a new conflict error
func NewConflictError(message string, details ...string) *AppError {
	return newAppError(ErrorTypeConflict, http.StatusConflict, message, details...)
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string, details ...string) *AppError {
	return newAppError(ErrorTypeUnauthorized, http.StatusUnauthorized, message, details...)
}

// NewForbiddenError creates a new forbidden error
func NewForbiddenError(message string, details ...string) *AppError {
	return newAppError(ErrorTypeForbidden, http.StatusForbidden, message, details...)
}

// NewInternalError creates a new internal error
func NewInternalError(message string, details ...string) *AppError {
	return newAppError(ErrorTypeInternal, http.StatusInternalServerError, message, details...)
}

// NewBadRequestError creates a new bad request error
func NewBadRequestError(message string, details ...string) *AppError {
	return newAppError(ErrorTypeBadRequest, http.StatusBadRequest, message, details...)
}

// IsAppError checks if the error is an AppError
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetAppError extracts AppError from error
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// IsConflictError checks if the error is a conflict error
func IsConflictError(err error) bool {
	appErr := GetAppError(err)
	return appErr != nil && appErr.Type == ErrorTypeConflict
}

// IsNotFoundError checks if the error is a not found error
func IsNotFoundError(err error) bool {
	appErr := GetAppError(err)
	return appErr != nil && appErr.Type == ErrorTypeNotFound
}

// IsValidationError checks if the error is a validation error
func IsValidationError(err error) bool {
	appErr := GetAppError(err)
	return appErr != nil && appErr.Type == ErrorTypeValidation
}

// IsDuplicateError checks if the error is a database duplicate key error
func IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// MySQL duplicate entry error
	if strings.Contains(errStr, "Duplicate entry") || strings.Contains(errStr, "duplicate key") {
		return true
	}
	// PostgreSQL unique violation
	if strings.Contains(errStr, "unique constraint") || strings.Contains(errStr, "violates unique constraint") {
		return true
	}
	return false
}
