package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS returns a Gin middleware for handling Cross-Origin Resource Sharing
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Set CORS headers
		c.Header("Access-Control-Allow-Origin", getAllowedOrigin(origin))
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// getAllowedOrigin returns the allowed origin based on the request origin
func getAllowedOrigin(origin string) string {
	// In production, you should maintain a list of allowed origins
	allowedOrigins := []string{
		"http://localhost:3000",
		"http://localhost:8080",
		"https://your-frontend-domain.com",
	}

	// Check if the origin is in the allowed list
	for _, allowedOrigin := range allowedOrigins {
		if origin == allowedOrigin {
			return origin
		}
	}

	// Default to the first allowed origin if not found
	// In production, you might want to return empty string for security
	if len(allowedOrigins) > 0 {
		return allowedOrigins[0]
	}

	return "*"
}

// SecurityHeaders returns a middleware that sets security headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}
