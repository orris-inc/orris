package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
	var appErr *errors.AppError
	var statusCode int
	var errorInfo ErrorInfo

	if appError := errors.GetAppError(err); appError != nil {
		appErr = appError
		statusCode = appErr.Code
		errorInfo = ErrorInfo{
			Type:    string(appErr.Type),
			Message: appErr.Message,
			Details: appErr.Details,
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
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	listResponse := ListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
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
