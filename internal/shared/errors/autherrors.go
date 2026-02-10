package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
)

// Authentication-specific error types
const (
	ErrorTypeInvalidCredentials ErrorType = "invalid_credentials"
	ErrorTypeAccountLocked      ErrorType = "account_locked"
	ErrorTypeAccountInactive    ErrorType = "account_inactive"
	ErrorTypeTokenExpired       ErrorType = "token_expired"
	ErrorTypeTokenInvalid       ErrorType = "token_invalid"
	ErrorTypeSessionExpired     ErrorType = "session_expired"
	ErrorTypePasswordNotSet     ErrorType = "password_not_set"
	ErrorTypeOAuthError         ErrorType = "oauth_error"
)

// AuthError represents authentication-specific errors with enhanced security context
type AuthError struct {
	*AppError
	// ShouldLog determines if this error should be logged
	// Some auth errors (like invalid credentials) may be expected and don't need error-level logging
	ShouldLog bool
	// SecurityEvent indicates if this should be tracked as a security event
	SecurityEvent bool
}

// Error implements the error interface
func (e *AuthError) Error() string {
	return e.AppError.Error()
}

// Unwrap allows errors.Is and errors.As to work correctly
func (e *AuthError) Unwrap() error {
	return e.AppError
}

// NewInvalidCredentialsError creates an error for invalid login credentials
// This error should not reveal whether the email or password was wrong (security best practice)
func NewInvalidCredentialsError() *AuthError {
	return &AuthError{
		AppError: &AppError{
			Type:    ErrorTypeInvalidCredentials,
			Message: "Invalid email or password",
			Code:    http.StatusUnauthorized,
		},
		ShouldLog:     false, // Expected error, don't clutter logs
		SecurityEvent: true,  // Track for brute force detection
	}
}

// NewAccountLockedError creates an error for locked accounts
func NewAccountLockedError(details ...string) *AuthError {
	detail := "Account is temporarily locked due to too many failed login attempts"
	if len(details) > 0 {
		detail = details[0]
	}
	return &AuthError{
		AppError: &AppError{
			Type:    ErrorTypeAccountLocked,
			Message: "Account is locked",
			Code:    http.StatusForbidden,
			Details: detail,
		},
		ShouldLog:     true, // Important to log
		SecurityEvent: true, // Security-relevant
	}
}

// NewAccountInactiveError creates an error for inactive accounts
func NewAccountInactiveError(details ...string) *AuthError {
	detail := "Account is not active. Please verify your email or contact support"
	if len(details) > 0 {
		detail = details[0]
	}
	return &AuthError{
		AppError: &AppError{
			Type:    ErrorTypeAccountInactive,
			Message: "Account is not active",
			Code:    http.StatusForbidden,
			Details: detail,
		},
		ShouldLog:     false, // Expected state
		SecurityEvent: false,
	}
}

// NewTokenExpiredError creates an error for expired tokens (JWT, refresh, etc.)
func NewTokenExpiredError(tokenType string) *AuthError {
	return &AuthError{
		AppError: &AppError{
			Type:    ErrorTypeTokenExpired,
			Message: fmt.Sprintf("%s has expired", tokenType),
			Code:    http.StatusUnauthorized,
			Details: "Please login again",
		},
		ShouldLog:     false, // Normal expiration
		SecurityEvent: false,
	}
}

// NewTokenInvalidError creates an error for invalid tokens
func NewTokenInvalidError(tokenType string) *AuthError {
	return &AuthError{
		AppError: &AppError{
			Type:    ErrorTypeTokenInvalid,
			Message: fmt.Sprintf("Invalid %s", tokenType),
			Code:    http.StatusUnauthorized,
			Details: "Token is invalid or has been revoked",
		},
		ShouldLog:     true, // May indicate tampering
		SecurityEvent: true, // Potential security issue
	}
}

// NewSessionExpiredError creates an error for expired sessions
func NewSessionExpiredError() *AuthError {
	return &AuthError{
		AppError: &AppError{
			Type:    ErrorTypeSessionExpired,
			Message: "Session has expired",
			Code:    http.StatusUnauthorized,
			Details: "Please login again",
		},
		ShouldLog:     false, // Normal expiration
		SecurityEvent: false,
	}
}

// NewPasswordNotSetError creates an error when password login is not available
// This typically happens for OAuth-only accounts
func NewPasswordNotSetError() *AuthError {
	return &AuthError{
		AppError: &AppError{
			Type:    ErrorTypePasswordNotSet,
			Message: "Password login not available",
			Code:    http.StatusBadRequest,
			Details: "This account uses OAuth login. Please use your social login provider",
		},
		ShouldLog:     false, // Expected for OAuth accounts
		SecurityEvent: false,
	}
}

// NewOAuthError creates an error for OAuth-related failures
func NewOAuthError(provider string, stage string, details ...string) *AuthError {
	detail := fmt.Sprintf("OAuth authentication failed at %s stage", stage)
	if len(details) > 0 {
		detail = details[0]
	}
	return &AuthError{
		AppError: &AppError{
			Type:    ErrorTypeOAuthError,
			Message: fmt.Sprintf("OAuth authentication failed with %s", provider),
			Code:    http.StatusBadGateway,
			Details: detail,
		},
		ShouldLog:     true, // External service issues should be logged
		SecurityEvent: false,
	}
}

// IsAuthError checks if the error is an AuthError (supports wrapped errors via errors.As)
func IsAuthError(err error) bool {
	var authErr *AuthError
	return stderrors.As(err, &authErr)
}

// GetAuthError extracts AuthError from error chain (supports wrapped errors via errors.As)
func GetAuthError(err error) *AuthError {
	var authErr *AuthError
	if stderrors.As(err, &authErr) {
		return authErr
	}
	return nil
}

// ShouldLogAuthError returns true if the authentication error should be logged
// This helps reduce noise in logs from expected auth failures
func ShouldLogAuthError(err error) bool {
	if authErr := GetAuthError(err); authErr != nil {
		return authErr.ShouldLog
	}
	return true // Default to logging if not an AuthError
}

// IsSecurityEvent returns true if the error should be tracked as a security event
func IsSecurityEvent(err error) bool {
	if authErr := GetAuthError(err); authErr != nil {
		return authErr.SecurityEvent
	}
	return false
}
