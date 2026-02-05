package utils

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/orris-inc/orris/internal/shared/errors"
)

// APIResponse represents a standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorInfo represents error information in API response
type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// NOTE: Pagination and ParsePagination have been moved to pagination.go

// GetUserIDFromContext retrieves user_id from gin context (set by auth middleware).
// Returns error if not found or invalid type.
func GetUserIDFromContext(c *gin.Context) (uint, error) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return 0, errors.NewUnauthorizedError("user not authenticated")
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		return 0, errors.NewInternalError("invalid user ID type in context")
	}

	return userID, nil
}

// GetSubscriptionIDFromContext retrieves subscription_id from gin context (set by middleware).
// Returns error if not found or invalid type.
func GetSubscriptionIDFromContext(c *gin.Context) (uint, error) {
	subscriptionIDInterface, exists := c.Get("subscription_id")
	if !exists {
		return 0, errors.NewUnauthorizedError("subscription context not available")
	}

	subscriptionID, ok := subscriptionIDInterface.(uint)
	if !ok {
		return 0, errors.NewInternalError("invalid subscription ID type in context")
	}

	return subscriptionID, nil
}

// SuccessResponse sends a successful response with custom status code
func SuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	response := APIResponse{
		Success: true,
		Data:    data,
		Message: message,
	}

	c.JSON(statusCode, response)
}

// CreatedResponse sends a created response
func CreatedResponse(c *gin.Context, data interface{}, message ...string) {
	response := APIResponse{
		Success: true,
		Data:    data,
	}

	if len(message) > 0 {
		response.Message = message[0]
	} else {
		response.Message = "Resource created successfully"
	}

	c.JSON(http.StatusCreated, response)
}

// ErrorResponse sends an error response with custom status code and message
func ErrorResponse(c *gin.Context, statusCode int, message string) {
	errorInfo := ErrorInfo{
		Type:    "error",
		Message: message,
	}

	response := APIResponse{
		Success: false,
		Error:   &errorInfo,
	}

	c.JSON(statusCode, response)
}

// ErrorResponseWithError sends an error response based on error type
func ErrorResponseWithError(c *gin.Context, err error) {
	var statusCode int
	var errorInfo ErrorInfo

	if appError := errors.GetAppError(err); appError != nil {
		statusCode = appError.Code
		errorInfo = ErrorInfo{
			Type:    string(appError.Type),
			Message: appError.Message,
			Details: appError.Details,
		}
	} else if validationErrs, ok := err.(validator.ValidationErrors); ok {
		// Handle Gin binding validation errors as 400 Bad Request
		statusCode = http.StatusBadRequest
		errorInfo = ErrorInfo{
			Type:    string(errors.ErrorTypeValidation),
			Message: "Request validation failed",
			Details: formatValidationErrors(validationErrs),
		}
	} else {
		// For non-AppError, do not expose internal error details to prevent information leakage
		statusCode = http.StatusInternalServerError
		errorInfo = ErrorInfo{
			Type:    string(errors.ErrorTypeInternal),
			Message: "Internal server error occurred",
		}
	}

	response := APIResponse{
		Success: false,
		Error:   &errorInfo,
	}

	c.JSON(statusCode, response)
}

// ListSuccessResponse sends a successful list response with pagination
func ListSuccessResponse(c *gin.Context, items interface{}, total int64, page, pageSize int, message ...string) {
	listResponse := ListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: TotalPages(total, pageSize),
	}

	response := APIResponse{
		Success: true,
		Data:    listResponse,
	}

	if len(message) > 0 {
		response.Message = message[0]
	}

	c.JSON(http.StatusOK, response)
}

// NoContentResponse sends a no content response
func NoContentResponse(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// formatValidationErrors formats validator.ValidationErrors into a readable string.
func formatValidationErrors(errs validator.ValidationErrors) string {
	if len(errs) == 0 {
		return ""
	}

	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		messages = append(messages, formatFieldError(err))
	}

	if len(messages) == 1 {
		return messages[0]
	}
	return strings.Join(messages, "; ")
}

// formatFieldError formats a single field validation error.
// Uses snake_case for field names in API responses.
func formatFieldError(fe validator.FieldError) string {
	field := toSnakeCase(fe.Field())
	return FormatFieldError(field, fe.Tag(), fe.Param(), fe.Kind())
}

// toSnakeCase converts PascalCase or camelCase to snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
