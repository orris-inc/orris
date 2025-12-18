package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type AuthMiddleware struct {
	jwtService *auth.JWTService
	logger     logger.Interface
}

func NewAuthMiddleware(jwtService *auth.JWTService, logger logger.Interface) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
		logger:     logger,
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

		c.Set("user_id", claims.UserID)
		c.Set("session_id", claims.SessionID)
		c.Set(constants.ContextKeyUserRole, string(claims.Role))

		c.Next()
	}
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
			c.Set("user_id", claims.UserID)
			c.Set("session_id", claims.SessionID)
			c.Set(constants.ContextKeyUserRole, string(claims.Role))
		}

		c.Next()
	}
}
