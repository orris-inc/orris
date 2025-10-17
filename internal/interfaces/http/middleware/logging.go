package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"orris/internal/shared/logger"
)

// Logger returns a Gin middleware for logging HTTP requests
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Use structured logging instead of formatted string
		fields := []zap.Field{
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
		}

		// Add error if present
		if param.ErrorMessage != "" {
			fields = append(fields, zap.String("error", param.ErrorMessage))
		}

		// Log with appropriate level based on status code
		if param.StatusCode >= 500 {
			logger.Error("HTTP request completed", fields...)
		} else if param.StatusCode >= 400 {
			logger.Warn("HTTP request completed", fields...)
		} else {
			logger.Info("HTTP request completed", fields...)
		}

		return ""
	})
}

// CustomLogger returns a custom Gin middleware for structured logging
func CustomLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Prepare log fields
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("body_size", c.Writer.Size()),
		}

		// Add request ID if present
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}

		// Add user ID if present in context
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.Any("user_id", userID))
		}

		// Log with appropriate level based on status code
		status := c.Writer.Status()
		switch {
		case status >= 500:
			logger.Error("HTTP request completed with server error", fields...)
		case status >= 400:
			logger.Warn("HTTP request completed with client error", fields...)
		case status >= 300:
			logger.Info("HTTP request completed with redirect", fields...)
		default:
			logger.Info("HTTP request completed successfully", fields...)
		}
	}
}