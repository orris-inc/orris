package telegram

import (
	"errors"
	"fmt"
)

// ErrCircuitOpen is returned when the circuit breaker is open and requests are rejected.
var ErrCircuitOpen = errors.New("telegram: circuit breaker is open")

// IsCircuitOpen returns true if the error is a circuit breaker open error.
func IsCircuitOpen(err error) bool {
	return errors.Is(err, ErrCircuitOpen)
}

// APIError represents a structured Telegram Bot API error response.
type APIError struct {
	ErrorCode   int    // HTTP-level error code from Telegram (e.g., 400, 403, 429)
	Description string // Human-readable error description
	RetryAfter  int    // Seconds to wait before retrying (only for 429)
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("telegram API error %d: %s (retry_after=%ds)", e.ErrorCode, e.Description, e.RetryAfter)
	}
	return fmt.Sprintf("telegram API error %d: %s", e.ErrorCode, e.Description)
}

// IsBotBlocked returns true if the error indicates the bot was blocked by the user (403).
func IsBotBlocked(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode == 403
	}
	return false
}

// IsRetryAfter returns true if the error is a 429 Too Many Requests with retry_after.
func IsRetryAfter(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode == 429 && apiErr.RetryAfter > 0
	}
	return false
}

// GetRetryAfter extracts the retry_after seconds from a 429 error.
// Returns 0 if the error is not a 429 or has no retry_after.
func GetRetryAfter(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.RetryAfter
	}
	return 0
}

// isNonRetryable returns true if the error should not be retried (400, 403, etc.).
func isNonRetryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		// 400 Bad Request and 403 Forbidden are not retryable
		return apiErr.ErrorCode == 400 || apiErr.ErrorCode == 403
	}
	return false
}
