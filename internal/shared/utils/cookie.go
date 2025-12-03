package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/config"
)

const (
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
)

// SetAuthCookies sets access and refresh token as HttpOnly cookies
func SetAuthCookies(c *gin.Context, cookieConfig config.CookieConfig, accessToken, refreshToken string, accessMaxAge, refreshMaxAge int) {
	sameSite := parseSameSite(cookieConfig.SameSite)

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
	c.SetSameSite(sameSite)

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
	c.SetSameSite(sameSite)
}

// ClearAuthCookies clears access and refresh token cookies
func ClearAuthCookies(c *gin.Context, cookieConfig config.CookieConfig) {
	sameSite := parseSameSite(cookieConfig.SameSite)

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
	c.SetSameSite(sameSite)

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
	c.SetSameSite(sameSite)
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
