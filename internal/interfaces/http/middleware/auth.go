package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/shared/authorization"
	"github.com/orris-inc/orris/internal/shared/config"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type AuthMiddleware struct {
	jwtService   *auth.JWTService
	userRepo     user.Repository
	cookieConfig config.CookieConfig
	logger       logger.Interface
}

func NewAuthMiddleware(jwtService *auth.JWTService, userRepo user.Repository, cookieConfig config.CookieConfig, logger logger.Interface) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService:   jwtService,
		userRepo:     userRepo,
		cookieConfig: cookieConfig,
		logger:       logger,
	}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from cookie first
		token := utils.GetTokenFromCookie(c, utils.AccessTokenCookie)

		// Fallback to Authorization header for backward compatibility
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				utils.ErrorResponse(c, http.StatusUnauthorized, "missing authorization token")
				c.Abort()
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				utils.ErrorResponse(c, http.StatusUnauthorized, "invalid authorization header format")
				c.Abort()
				return
			}

			token = parts[1]
		}

		claims, err := m.jwtService.Verify(token)
		if err != nil {
			m.logger.Warnw("failed to verify token", "error", err)
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		if claims.TokenType != auth.TokenTypeAccess {
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid token type")
			c.Abort()
			return
		}

		// Look up user by SID to get internal ID
		foundUser, err := m.userRepo.GetBySID(c.Request.Context(), claims.UserUUID)
		if err != nil || foundUser == nil {
			m.logger.Warnw("user not found by uuid", "user_uuid", claims.UserUUID, "error", err)
			utils.ErrorResponse(c, http.StatusUnauthorized, "user not found")
			c.Abort()
			return
		}

		c.Set("user_id", foundUser.ID())
		c.Set("user_uuid", claims.UserUUID)
		c.Set("session_id", claims.SessionID)
		// Use the current role from database, not the stale role from JWT claims.
		// This ensures role changes (e.g. admin demotion) take effect immediately.
		c.Set(constants.ContextKeyUserRole, string(foundUser.Role()))

		// Auto-refresh: if token is about to expire, generate a new one
		if m.jwtService.ShouldRefresh(claims) {
			m.refreshAccessToken(c, claims, foundUser.Role())
		}

		c.Next()
	}
}

// refreshAccessToken generates a new access token and sets it in the cookie.
// freshRole is the user's current role from the database, ensuring the refreshed
// token reflects any role changes (e.g. demotion from admin).
func (m *AuthMiddleware) refreshAccessToken(c *gin.Context, claims *auth.Claims, freshRole authorization.UserRole) {
	newToken, err := m.jwtService.RefreshAccessToken(claims, freshRole)
	if err != nil {
		m.logger.Warnw("failed to auto-refresh access token", "error", err, "user_uuid", claims.UserUUID)
		return
	}

	// Set the new access token in cookie
	accessMaxAge := m.jwtService.AccessExpMinutes() * 60
	utils.SetAccessTokenCookie(c, m.cookieConfig, newToken, accessMaxAge)

	// Refresh CSRF cookie alongside access token
	utils.SetCSRFCookie(c, m.cookieConfig, accessMaxAge)

	m.logger.Debugw("access token auto-refreshed", "user_uuid", claims.UserUUID)
}

func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from cookie first
		token := utils.GetTokenFromCookie(c, utils.AccessTokenCookie)

		// Fallback to Authorization header for backward compatibility
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.Next()
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			} else {
				c.Next()
				return
			}
		}

		claims, err := m.jwtService.Verify(token)
		if err == nil && claims.TokenType == auth.TokenTypeAccess {
			// Look up user by SID to get internal ID
			foundUser, lookupErr := m.userRepo.GetBySID(c.Request.Context(), claims.UserUUID)
			if lookupErr == nil && foundUser != nil {
				c.Set("user_id", foundUser.ID())
				c.Set("user_uuid", claims.UserUUID)
				c.Set("session_id", claims.SessionID)
				// Use the current role from database, not the stale role from JWT claims.
				c.Set(constants.ContextKeyUserRole, string(foundUser.Role()))
			}
		}

		c.Next()
	}
}
