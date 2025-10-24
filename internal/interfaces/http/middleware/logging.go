package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"orris/internal/shared/logger"
)

func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		args := []any{
			"method", param.Method,
			"path", param.Path,
			"status", param.StatusCode,
			"latency", param.Latency,
			"client_ip", param.ClientIP,
			"user_agent", param.Request.UserAgent(),
		}

		if param.ErrorMessage != "" {
			args = append(args, "error", param.ErrorMessage)
		}

		if param.StatusCode >= 500 {
			logger.Error("HTTP request completed", args...)
		} else if param.StatusCode >= 400 {
			logger.Warn("HTTP request completed", args...)
		} else {
			logger.Info("HTTP request completed", args...)
		}

		return ""
	})
}

func CustomLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)

		args := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"query", c.Request.URL.RawQuery,
			"status", c.Writer.Status(),
			"latency", latency,
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
			"body_size", c.Writer.Size(),
		}

		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			args = append(args, "request_id", requestID)
		}

		if userID, exists := c.Get("user_id"); exists {
			args = append(args, "user_id", userID)
		}

		status := c.Writer.Status()
		switch {
		case status >= 500:
			logger.Error("HTTP request completed with server error", args...)
		case status >= 400:
			logger.Warn("HTTP request completed with client error", args...)
		case status >= 300:
			logger.Info("HTTP request completed with redirect", args...)
		default:
			logger.Info("HTTP request completed successfully", args...)
		}
	}
}
