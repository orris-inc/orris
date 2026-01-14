// Package middleware provides HTTP middleware for the application.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/id"
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

// RequireOwnership ensures the authenticated user owns the subscription.
// Gets subscription SID from :sid URL parameter.
func (m *SubscriptionOwnerMiddleware) RequireOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
			c.Abort()
			return
		}

		subscriptionSID := c.Param("sid")
		if subscriptionSID == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "subscription SID is required")
			c.Abort()
			return
		}

		// Validate Stripe-style ID format (sub_xxx)
		if !strings.HasPrefix(subscriptionSID, id.PrefixSubscription+"_") {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription SID format, expected sub_xxxxx")
			c.Abort()
			return
		}

		// Parse Stripe-style ID to validate format
		if _, parseErr := id.ParseSubscriptionID(subscriptionSID); parseErr != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription SID format")
			c.Abort()
			return
		}

		sub, err := m.subscriptionRepo.GetBySID(c.Request.Context(), subscriptionSID)
		if err != nil {
			m.logger.Warnw("failed to get subscription by SID for ownership check",
				"sid", subscriptionSID,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
			c.Abort()
			return
		}

		if sub == nil {
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
				"subscription_sid", sub.SID(),
			)
			utils.ErrorResponse(c, http.StatusForbidden, "access denied")
			c.Abort()
			return
		}

		// Store subscription in context for handler reuse
		c.Set("subscription", sub)
		c.Set("subscription_id", sub.ID())

		c.Next()
	}
}
