package utils

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/config"
)

const (
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
	CSRFTokenCookie    = "csrf_token"
	CSRFTokenHeader    = "X-CSRF-Token"
	csrfTokenBytes     = 32
)

// SetAuthCookies sets access and refresh token as HttpOnly cookies
func SetAuthCookies(c *gin.Context, cookieConfig config.CookieConfig, accessToken, refreshToken string, accessMaxAge, refreshMaxAge int) {
	sameSite := parseSameSite(cookieConfig.SameSite)
	c.SetSameSite(sameSite)

	// Set access token cookie
	c.SetCookie(
		AccessTokenCookie,
		accessToken,
		accessMaxAge,
		cookieConfig.Path,
		cookieConfig.Domain,
		cookieConfig.Secure,
		true, // HttpOnly
	)

	// Set refresh token cookie
	c.SetCookie(
		RefreshTokenCookie,
		refreshToken,
		refreshMaxAge,
		cookieConfig.Path,
		cookieConfig.Domain,
		cookieConfig.Secure,
		true, // HttpOnly
	)
}

// SetAccessTokenCookie sets only the access token cookie (used for auto-refresh)
func SetAccessTokenCookie(c *gin.Context, cookieConfig config.CookieConfig, accessToken string, maxAge int) {
	sameSite := parseSameSite(cookieConfig.SameSite)
	c.SetSameSite(sameSite)

	c.SetCookie(
		AccessTokenCookie,
		accessToken,
		maxAge,
		cookieConfig.Path,
		cookieConfig.Domain,
		cookieConfig.Secure,
		true, // HttpOnly
	)
}

// ClearAuthCookies clears access and refresh token cookies
func ClearAuthCookies(c *gin.Context, cookieConfig config.CookieConfig) {
	sameSite := parseSameSite(cookieConfig.SameSite)
	c.SetSameSite(sameSite)

	// Clear access token cookie
	c.SetCookie(
		AccessTokenCookie,
		"",
		-1,
		cookieConfig.Path,
		cookieConfig.Domain,
		cookieConfig.Secure,
		true, // HttpOnly
	)

	// Clear refresh token cookie
	c.SetCookie(
		RefreshTokenCookie,
		"",
		-1,
		cookieConfig.Path,
		cookieConfig.Domain,
		cookieConfig.Secure,
		true, // HttpOnly
	)
}

// GetTokenFromCookie retrieves token from cookie or Authorization header (fallback)
func GetTokenFromCookie(c *gin.Context, cookieName string) string {
	// Try to get token from cookie first
	token, err := c.Cookie(cookieName)
	if err == nil && token != "" {
		return token
	}

	// Fallback to Authorization header for backward compatibility
	// This is handled separately in middleware
	return ""
}

// SetCSRFCookie generates a random CSRF token and sets it as a non-HttpOnly cookie.
// The token is readable by frontend JavaScript for the Double Submit Cookie pattern.
func SetCSRFCookie(c *gin.Context, cookieConfig config.CookieConfig, maxAge int) {
	token := generateCSRFToken()
	sameSite := parseSameSite(cookieConfig.SameSite)

	c.SetSameSite(sameSite)
	c.SetCookie(
		CSRFTokenCookie,
		token,
		maxAge,
		cookieConfig.Path,
		cookieConfig.Domain,
		cookieConfig.Secure,
		false, // HttpOnly=false so frontend JS can read it
	)
}

// ClearCSRFCookie removes the CSRF token cookie.
func ClearCSRFCookie(c *gin.Context, cookieConfig config.CookieConfig) {
	sameSite := parseSameSite(cookieConfig.SameSite)

	c.SetSameSite(sameSite)
	c.SetCookie(
		CSRFTokenCookie,
		"",
		-1,
		cookieConfig.Path,
		cookieConfig.Domain,
		cookieConfig.Secure,
		false,
	)
}

// generateCSRFToken generates a cryptographically random hex token.
func generateCSRFToken() string {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		// Fallback should never happen; crypto/rand.Read only fails on catastrophic OS errors
		panic("csrf: failed to generate random token: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// parseSameSite converts string to http.SameSite
func parseSameSite(sameSite string) http.SameSite {
	switch sameSite {
	case "Strict":
		return http.SameSiteStrictMode
	case "Lax":
		return http.SameSiteLaxMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
