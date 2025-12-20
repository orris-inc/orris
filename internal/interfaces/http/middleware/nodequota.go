package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// NodeQuotaMiddleware validates user node quotas
type NodeQuotaMiddleware struct {
	nodeRepo         node.NodeRepository
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	logger           logger.Interface
}

// NewNodeQuotaMiddleware creates a new node quota middleware
func NewNodeQuotaMiddleware(
	nodeRepo node.NodeRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
) *NodeQuotaMiddleware {
	return &NodeQuotaMiddleware{
		nodeRepo:         nodeRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger.NewLogger(),
	}
}

// CheckNodeLimit validates that the user hasn't exceeded their node limit
func (m *NodeQuotaMiddleware) CheckNodeLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("user not authenticated"))
			c.Abort()
			return
		}
		userID, ok := userIDInterface.(uint)
		if !ok {
			utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("invalid user ID in context"))
			c.Abort()
			return
		}

		// Admin has no limit
		roleInterface, _ := c.Get("user_role")
		role, _ := roleInterface.(string)
		if role == "admin" {
			c.Next()
			return
		}

		// Get user's active subscriptions
		subscriptions, err := m.subscriptionRepo.GetActiveByUserID(c.Request.Context(), userID)
		if err != nil {
			m.logger.Errorw("failed to get user subscriptions", "user_id", userID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewInternalError("failed to check subscription"))
			c.Abort()
			return
		}

		if len(subscriptions) == 0 {
			utils.ErrorResponseWithError(c, errors.NewForbiddenError("no active subscription found"))
			c.Abort()
			return
		}

		// Find the highest node limit from all subscriptions
		var maxNodeLimit int
		hasUnlimited := false

		for _, sub := range subscriptions {
			plan, err := m.planRepo.GetByID(c.Request.Context(), sub.PlanID())
			if err != nil {
				m.logger.Warnw("failed to get plan", "plan_id", sub.PlanID(), "error", err)
				continue
			}

			if !plan.HasNodeLimit() {
				hasUnlimited = true
				break
			}

			limit := plan.GetNodeLimit()
			if limit > maxNodeLimit {
				maxNodeLimit = limit
			}
		}

		// If any plan has unlimited nodes, allow
		if hasUnlimited {
			c.Next()
			return
		}

		// If no node limit found, deny
		if maxNodeLimit == 0 {
			utils.ErrorResponseWithError(c, errors.NewForbiddenError("your subscription does not allow creating nodes"))
			c.Abort()
			return
		}

		// Count user's current nodes
		currentCount, err := m.nodeRepo.CountByUserID(c.Request.Context(), userID)
		if err != nil {
			m.logger.Errorw("failed to count user nodes", "user_id", userID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewInternalError("failed to check node count"))
			c.Abort()
			return
		}

		// Check if user has reached the limit
		if int(currentCount) >= maxNodeLimit {
			m.logger.Warnw("user has reached node limit",
				"user_id", userID,
				"current_count", currentCount,
				"max_limit", maxNodeLimit,
			)
			utils.ErrorResponseWithError(c, errors.NewForbiddenError("you have reached your node limit"))
			c.Abort()
			return
		}

		c.Next()
	}
}
