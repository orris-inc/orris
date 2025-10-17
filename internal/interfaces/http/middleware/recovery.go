package middleware

import (
	"net"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

// Recovery returns a Gin middleware that recovers from panics
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Check if connection is broken
		if checkBrokenConnection(recovered) {
			logger.Error("connection broken during request",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Any("error", recovered))
			c.Abort()
			return
		}

		// Log the panic with stack trace
		httpRequest, _ := httputil.DumpRequest(c.Request, false)
		headers := strings.Split(string(httpRequest), "\r\n")
		for idx, header := range headers {
			current := strings.Split(header, ":")
			if current[0] == "Authorization" {
				headers[idx] = current[0] + ": *"
			}
		}

		logger.Error("panic recovered",
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.Strings("headers", headers),
			zap.Any("error", recovered),
			zap.String("stack", string(debug.Stack())))

		// Return internal server error
		utils.ErrorResponse(c, 500, "Internal server error occurred")
	})
}

// checkBrokenConnection checks if the error is a broken connection
func checkBrokenConnection(err interface{}) bool {
	var brokenConnections = []string{
		"connection reset by peer",
		"broken pipe",
		"connection refused",
	}

	if ne, ok := err.(*net.OpError); ok {
		if se, ok := ne.Err.(*os.SyscallError); ok {
			errStr := strings.ToLower(se.Error())
			for _, s := range brokenConnections {
				if strings.Contains(errStr, s) {
					return true
				}
			}
		}
	}
	return false
}

// ErrorHandler returns a middleware that handles application errors
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle errors from handlers
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Log the error
			logger.Error("handler error occurred",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Error(err))

			// Send error response if not already sent
			if !c.Writer.Written() {
				utils.ErrorResponseWithError(c, err)
			}
		}
	}
}