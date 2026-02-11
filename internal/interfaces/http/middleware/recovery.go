package middleware

import (
	"net"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

func Recovery(log logger.Interface) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if checkBrokenConnection(recovered) {
			log.Errorw("connection broken during request",
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
			if current[0] == "Authorization" || current[0] == "Cookie" {
				headers[idx] = current[0] + ": *"
			}
		}

		log.Errorw("panic recovered",
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
