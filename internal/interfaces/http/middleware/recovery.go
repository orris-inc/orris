package middleware

import (
	"net"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"

	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if checkBrokenConnection(recovered) {
			logger.Error("connection broken during request",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"error", recovered)
			c.Abort()
			return
		}

		httpRequest, _ := httputil.DumpRequest(c.Request, false)
		headers := strings.Split(string(httpRequest), "\r\n")
		for idx, header := range headers {
			current := strings.Split(header, ":")
			if current[0] == "Authorization" {
				headers[idx] = current[0] + ": *"
			}
		}

		logger.Error("panic recovered",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"headers", headers,
			"error", recovered,
			"stack", string(debug.Stack()))

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

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			logger.Error("handler error occurred",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"error", err)

			if !c.Writer.Written() {
				utils.ErrorResponseWithError(c, err)
			}
		}
	}
}