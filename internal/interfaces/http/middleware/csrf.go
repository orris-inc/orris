package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/utils"
)

// csrfExactPaths lists exact paths exempt from CSRF validation.
// These are unauthenticated endpoints with no cookie session to protect.
var csrfExactPaths = map[string]struct{}{
	"/auth/login":           {},
	"/auth/register":        {},
	"/auth/verify-email":    {},
	"/auth/forgot-password": {},
	"/auth/reset-password":  {},
	"/auth/refresh":         {},
	// Logout is exempt because the CSRF cookie may have expired alongside the access token.
	// It is already protected by RequireAuth middleware.
	"/auth/logout":          {},
	"/payments/callback":    {},
}

// csrfPrefixPaths lists path prefixes exempt from CSRF validation.
// These are directory-style paths for OAuth flows, webhooks, or machine-to-machine APIs.
var csrfPrefixPaths = []string{
	"/auth/oauth/",
	"/auth/passkey/login/",
	"/auth/passkey/signup/",
	"/webhooks/",
	// Machine-to-machine APIs (token-header authenticated, not cookie-based)
	"/agents/",
	"/forward-agent-api/",
	"/ws/",
}

// CSRF returns a middleware that validates CSRF tokens using the Double Submit Cookie pattern.
// For mutating requests (POST, PUT, DELETE, PATCH), it compares the csrf_token cookie value
// against the X-CSRF-Token header value. Safe methods (GET, HEAD, OPTIONS) are always skipped.
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip safe HTTP methods
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		// Skip exempt paths (exact match first, then prefix match)
		path := c.Request.URL.Path
		if _, ok := csrfExactPaths[path]; ok {
			c.Next()
			return
		}
		for _, prefix := range csrfPrefixPaths {
			if strings.HasPrefix(path, prefix) {
				c.Next()
				return
			}
		}

		// Read CSRF token from cookie
		cookieToken, err := c.Cookie(utils.CSRFTokenCookie)
		if err != nil || cookieToken == "" {
			utils.ErrorResponse(c, http.StatusForbidden, "missing CSRF token")
			c.Abort()
			return
		}

		// Read CSRF token from header
		headerToken := c.GetHeader(utils.CSRFTokenHeader)
		if headerToken == "" {
			utils.ErrorResponse(c, http.StatusForbidden, "missing CSRF token header")
			c.Abort()
			return
		}

		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
			utils.ErrorResponse(c, http.StatusForbidden, "invalid CSRF token")
			c.Abort()
			return
		}

		c.Next()
	}
}

// isSafeMethod returns true for HTTP methods that do not mutate state.
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
