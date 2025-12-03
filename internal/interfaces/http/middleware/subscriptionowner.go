// Package middleware provides HTTP middleware for the application.
package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// SubscriptionOwnerMiddleware ensures users can only access their own subscriptions
type SubscriptionOwnerMiddleware struct {
	subscriptionRepo subscription.SubscriptionRepository
	logger           logger.Interface
}

// NewSubscriptionOwnerMiddleware creates a new subscription owner middleware
func NewSubscriptionOwnerMiddleware(
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *SubscriptionOwnerMiddleware {
	return &SubscriptionOwnerMiddleware{
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// RequireOwnership ensures the authenticated user owns the subscription
func (m *SubscriptionOwnerMiddleware) RequireOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
			c.Abort()
			return
		}

		subscriptionIDStr := c.Param("id")
		if subscriptionIDStr == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "subscription ID is required")
			c.Abort()
			return
		}

		subscriptionID, err := strconv.ParseUint(subscriptionIDStr, 10, 64)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
			c.Abort()
			return
		}

		sub, err := m.subscriptionRepo.GetByID(c.Request.Context(), uint(subscriptionID))
		if err != nil {
			m.logger.Warnw("failed to get subscription for ownership check",
				"subscription_id", subscriptionID,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
			c.Abort()
			return
		}

		currentUserID, ok := userID.(uint)
		if !ok {
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid user ID type")
			c.Abort()
			return
		}

		if sub.UserID() != currentUserID {
			m.logger.Warnw("user attempted to access another user's subscription",
				"current_user_id", currentUserID,
				"subscription_owner_id", sub.UserID(),
				"subscription_id", subscriptionID,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "access denied")
			c.Abort()
			return
		}

		// Store subscription in context for handler reuse
		c.Set("subscription", sub)
		c.Set("subscription_id", uint(subscriptionID))

		c.Next()
	}
}
