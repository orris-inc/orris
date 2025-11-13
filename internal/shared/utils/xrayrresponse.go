package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"orris/internal/shared/logger"
)

// V2RaySocksResponse represents the standard v2raysocks API response structure
type V2RaySocksResponse struct {
	Data interface{} `json:"data"`
	Ret  *int        `json:"ret,omitempty"`
}

// V2RaySocksErrorResponse represents the error response structure for v2raysocks API
type V2RaySocksErrorResponse struct {
	Ret     int    `json:"ret"`
	Message string `json:"msg"`
}

// V2RaySocksSuccess sends a successful response with data
// Returns HTTP 200 with format: {"data": data}
func V2RaySocksSuccess(c *gin.Context, data interface{}) {
	response := V2RaySocksResponse{
		Data: data,
	}

	c.JSON(http.StatusOK, response)
}

// V2RaySocksSuccessWithRet sends a successful response with data and return code
// Returns HTTP 200 with format: {"data": data, "ret": ret}
// The ret parameter typically indicates success (1) or different success states
func V2RaySocksSuccessWithRet(c *gin.Context, data interface{}, ret int) {
	response := V2RaySocksResponse{
		Data: data,
		Ret:  &ret,
	}

	c.JSON(http.StatusOK, response)
}

// V2RaySocksError sends an error response with status code and message
// Returns the specified HTTP status code with format: {"ret": 0, "msg": message}
// The ret value is always 0 for errors
func V2RaySocksError(c *gin.Context, statusCode int, message string) {
	errorResponse := V2RaySocksErrorResponse{
		Ret:     0,
		Message: message,
	}

	// Log error for monitoring and debugging
	logger.Error("v2raysocks API error",
		"status_code", statusCode,
		"message", message,
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
	)

	c.JSON(statusCode, errorResponse)
}

// V2RaySocksNotModified sends a 304 Not Modified response
// Used when the client's cached version (identified by ETag) is still valid
// This helps reduce bandwidth and server load by avoiding unnecessary data transfer
func V2RaySocksNotModified(c *gin.Context) {
	c.Status(http.StatusNotModified)
}

// SetETag sets the ETag header for cache validation
// ETag is a unique identifier for a specific version of a resource
// Clients can use this value in subsequent requests to check if content has changed
func SetETag(c *gin.Context, etag string) {
	c.Header("ETag", etag)
}

// CheckETag checks if the client's ETag matches the current resource version
// Returns true if the ETags match (content hasn't changed)
// Returns false if they don't match (content has been updated)
//
// Usage:
//
//	if CheckETag(c, currentETag) {
//	    V2RaySocksNotModified(c)
//	    return
//	}
func CheckETag(c *gin.Context, etag string) bool {
	clientETag := c.GetHeader("If-None-Match")
	return clientETag != "" && clientETag == etag
}

// GenerateETag creates an ETag based on the provided data
// Uses MD5 hash of the JSON representation of the data
// Returns the ETag string that can be used with SetETag and CheckETag
//
// Note: This is a convenience function for generating ETags from response data
// For better performance in production, consider pre-computing and caching ETags
func GenerateETag(data interface{}) (string, error) {
	// Serialize data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Calculate MD5 hash
	hash := md5.Sum(jsonData)
	etag := hex.EncodeToString(hash[:])

	return etag, nil
}

// V2RaySocksSuccessWithETag sends a successful response with ETag support
// Automatically generates and sets ETag based on the response data
// If client's ETag matches, returns 304 Not Modified
// Otherwise, returns 200 OK with the data and new ETag
//
// Usage:
//
//	V2RaySocksSuccessWithETag(c, nodeData)
func V2RaySocksSuccessWithETag(c *gin.Context, data interface{}) {
	// Generate ETag from data
	etag, err := GenerateETag(data)
	if err != nil {
		logger.Warn("failed to generate ETag, returning response without caching",
			"error", err,
			"path", c.Request.URL.Path,
		)
		V2RaySocksSuccess(c, data)
		return
	}

	// Check if client's cached version is still valid
	if CheckETag(c, etag) {
		V2RaySocksNotModified(c)
		return
	}

	// Set ETag header for future requests
	SetETag(c, etag)

	// Return the response data
	V2RaySocksSuccess(c, data)
}
