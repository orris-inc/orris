package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type SubscriptionTokenMiddleware struct {
	validateTokenUseCase *usecases.ValidateSubscriptionTokenUseCase
	logger               logger.Interface
}

func NewSubscriptionTokenMiddleware(
	validateTokenUseCase *usecases.ValidateSubscriptionTokenUseCase,
	logger logger.Interface,
) *SubscriptionTokenMiddleware {
	return &SubscriptionTokenMiddleware{
		validateTokenUseCase: validateTokenUseCase,
		logger:               logger,
	}
}

func (m *SubscriptionTokenMiddleware) RequireSubscriptionToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]
		cmd := usecases.ValidateSubscriptionTokenCommand{
			PlainToken:    token,
			RequiredScope: "",
			IPAddress:     c.ClientIP(),
		}

		result, err := m.validateTokenUseCase.Execute(c.Request.Context(), cmd)
		if err != nil {
			m.logger.Warnw("token validation failed", "error", err, "ip", c.ClientIP())
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		if !result.Subscription.IsActive() {
			m.logger.Warnw("subscription is not active",
				"subscription_id", result.Subscription.ID(),
				"status", result.Subscription.Status(),
			)
			utils.ErrorResponse(c, http.StatusForbidden, "subscription is not active")
			c.Abort()
			return
		}

		c.Set("subscription_id", result.Subscription.ID())
		c.Set("subscription", result.Subscription)
		c.Set("subscription_plan", result.Plan)
		c.Set("subscription_token", result.Token)

		c.Next()
	}
}

func (m *SubscriptionTokenMiddleware) RequireSubscriptionTokenWithScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid authorization header format")
			c.Abort()
			return
		}

		token := parts[1]
		cmd := usecases.ValidateSubscriptionTokenCommand{
			PlainToken:    token,
			RequiredScope: scope,
			IPAddress:     c.ClientIP(),
		}

		result, err := m.validateTokenUseCase.Execute(c.Request.Context(), cmd)
		if err != nil {
			m.logger.Warnw("token validation failed",
				"error", err,
				"required_scope", scope,
				"ip", c.ClientIP(),
			)
			utils.ErrorResponse(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		if !result.Subscription.IsActive() {
			m.logger.Warnw("subscription is not active",
				"subscription_id", result.Subscription.ID(),
				"status", result.Subscription.Status(),
			)
			utils.ErrorResponse(c, http.StatusForbidden, "subscription is not active")
			c.Abort()
			return
		}

		c.Set("subscription_id", result.Subscription.ID())
		c.Set("subscription", result.Subscription)
		c.Set("subscription_plan", result.Plan)
		c.Set("subscription_token", result.Token)

		c.Next()
	}
}
