// Package middleware provides HTTP middleware for the application.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ForwardQuotaMiddleware enforces forward rule quotas based on user's subscription plan
type ForwardQuotaMiddleware struct {
	forwardRuleRepo  forward.Repository
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	logger           logger.Interface
}

// NewForwardQuotaMiddleware creates a new forward quota middleware
func NewForwardQuotaMiddleware(
	forwardRuleRepo forward.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *ForwardQuotaMiddleware {
	return &ForwardQuotaMiddleware{
		forwardRuleRepo:  forwardRuleRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

// CheckRuleLimit verifies that the user hasn't exceeded their forward rule count limit
func (m *ForwardQuotaMiddleware) CheckRuleLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
			c.Abort()
			return
		}

		// Get user role for admin check
		userRole := c.GetString(constants.ContextKeyUserRole)
		isAdmin := userRole == constants.RoleAdmin

		// Admin has unlimited rules
		if isAdmin {
			c.Next()
			return
		}

		currentUserID, ok := userID.(uint)
		if !ok {
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid user ID type")
			c.Abort()
			return
		}

		// Get user's active subscriptions
		subscriptions, err := m.subscriptionRepo.GetActiveByUserID(c.Request.Context(), currentUserID)
		if err != nil {
			m.logger.Errorw("failed to get active subscriptions for quota check",
				"user_id", currentUserID,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to check quota")
			c.Abort()
			return
		}

		// Find the highest rule limit among all active subscriptions
		maxRuleLimit := 0
		for _, sub := range subscriptions {
			plan, err := m.planRepo.GetByID(c.Request.Context(), sub.PlanID())
			if err != nil {
				m.logger.Warnw("failed to get plan for subscription",
					"subscription_id", sub.ID(),
					"plan_id", sub.PlanID(),
					"error", err,
				)
				continue
			}

			if plan == nil {
				continue
			}

			// Skip non-forward plans
			if !plan.PlanType().IsForward() {
				continue
			}

			planFeatures := plan.Features()
			if planFeatures == nil {
				continue
			}

			limit, err := planFeatures.GetForwardRuleLimit()
			if err != nil {
				m.logger.Warnw("failed to get forward rule limit from plan",
					"subscription_id", sub.ID(),
					"error", err,
				)
				continue
			}

			// 0 means unlimited
			if limit == 0 {
				c.Next()
				return
			}

			if limit > maxRuleLimit {
				maxRuleLimit = limit
			}
		}

		// If no subscriptions or all limits are 0, deny access
		if len(subscriptions) == 0 || maxRuleLimit == 0 {
			m.logger.Warnw("user has no active subscription with forward rule limit",
				"user_id", currentUserID,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "no active subscription with forward feature")
			c.Abort()
			return
		}

		// Count current rules owned by the user
		currentCount, err := m.forwardRuleRepo.CountByUserID(c.Request.Context(), currentUserID)
		if err != nil {
			m.logger.Errorw("failed to count user's forward rules",
				"user_id", currentUserID,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to check quota")
			c.Abort()
			return
		}

		// Check if limit exceeded
		if currentCount >= int64(maxRuleLimit) {
			m.logger.Warnw("user exceeded forward rule limit",
				"user_id", currentUserID,
				"current_count", currentCount,
				"max_limit", maxRuleLimit,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "forward rule limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}

// CheckTrafficLimit verifies that the user hasn't exceeded their forward traffic limit
func (m *ForwardQuotaMiddleware) CheckTrafficLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
			c.Abort()
			return
		}

		// Get user role for admin check
		userRole := c.GetString(constants.ContextKeyUserRole)
		isAdmin := userRole == constants.RoleAdmin

		// Admin has unlimited traffic
		if isAdmin {
			c.Next()
			return
		}

		currentUserID, ok := userID.(uint)
		if !ok {
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid user ID type")
			c.Abort()
			return
		}

		// Get user's active subscriptions
		subscriptions, err := m.subscriptionRepo.GetActiveByUserID(c.Request.Context(), currentUserID)
		if err != nil {
			m.logger.Errorw("failed to get active subscriptions for traffic quota check",
				"user_id", currentUserID,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to check traffic quota")
			c.Abort()
			return
		}

		// Find the highest traffic limit among all active subscriptions
		var maxTrafficLimit uint64
		for _, sub := range subscriptions {
			plan, err := m.planRepo.GetByID(c.Request.Context(), sub.PlanID())
			if err != nil {
				m.logger.Warnw("failed to get plan for subscription",
					"subscription_id", sub.ID(),
					"plan_id", sub.PlanID(),
					"error", err,
				)
				continue
			}

			if plan == nil {
				continue
			}

			// Skip non-forward plans
			if !plan.PlanType().IsForward() {
				continue
			}

			planFeatures := plan.Features()
			if planFeatures == nil {
				continue
			}

			limit, err := planFeatures.GetForwardTrafficLimit()
			if err != nil {
				m.logger.Warnw("failed to get forward traffic limit from plan",
					"subscription_id", sub.ID(),
					"error", err,
				)
				continue
			}

			// 0 means unlimited
			if limit == 0 {
				c.Next()
				return
			}

			if limit > maxTrafficLimit {
				maxTrafficLimit = limit
			}
		}

		// If no subscriptions, deny access
		if len(subscriptions) == 0 {
			m.logger.Warnw("user has no active subscription for forward traffic",
				"user_id", currentUserID,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "no active subscription with forward feature")
			c.Abort()
			return
		}

		// If maxTrafficLimit is still 0 after checking all subscriptions, it means unlimited
		if maxTrafficLimit == 0 {
			c.Next()
			return
		}

		// Get total traffic used by the user
		totalTraffic, err := m.forwardRuleRepo.GetTotalTrafficByUserID(c.Request.Context(), currentUserID)
		if err != nil {
			m.logger.Errorw("failed to get user's total forward traffic",
				"user_id", currentUserID,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to check traffic quota")
			c.Abort()
			return
		}

		// Check if traffic limit exceeded
		if uint64(totalTraffic) >= maxTrafficLimit {
			m.logger.Warnw("user exceeded forward traffic limit",
				"user_id", currentUserID,
				"total_traffic", totalTraffic,
				"max_limit", maxTrafficLimit,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "forward traffic limit exceeded")
			c.Abort()
			return
		}

		c.Next()
	}
}

// CheckRuleTypeAllowed verifies that the requested rule type is allowed by the user's subscription plan
func (m *ForwardQuotaMiddleware) CheckRuleTypeAllowed() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
			c.Abort()
			return
		}

		// Get user role for admin check
		userRole := c.GetString(constants.ContextKeyUserRole)
		isAdmin := userRole == constants.RoleAdmin

		// Admin can use all rule types
		if isAdmin {
			c.Next()
			return
		}

		currentUserID, ok := userID.(uint)
		if !ok {
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid user ID type")
			c.Abort()
			return
		}

		// Get rule type from request body using ShouldBindBodyWith to preserve body for handler
		var requestBody struct {
			RuleType string `json:"rule_type"`
		}

		if err := c.ShouldBindBodyWith(&requestBody, binding.JSON); err != nil {
			// If we can't parse the body, let the handler deal with it
			c.Next()
			return
		}

		// If no rule type specified, skip check (handler will validate)
		if requestBody.RuleType == "" {
			c.Next()
			return
		}

		// Get user's active subscriptions
		subscriptions, err := m.subscriptionRepo.GetActiveByUserID(c.Request.Context(), currentUserID)
		if err != nil {
			m.logger.Errorw("failed to get active subscriptions for rule type check",
				"user_id", currentUserID,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusInternalServerError, "failed to check rule type permission")
			c.Abort()
			return
		}

		if len(subscriptions) == 0 {
			m.logger.Warnw("user has no active subscription for forward rule creation",
				"user_id", currentUserID,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "no active subscription with forward feature")
			c.Abort()
			return
		}

		// Check if any active subscription allows the requested rule type
		allowed := false
		for _, sub := range subscriptions {
			plan, err := m.planRepo.GetByID(c.Request.Context(), sub.PlanID())
			if err != nil {
				m.logger.Warnw("failed to get plan for subscription",
					"subscription_id", sub.ID(),
					"plan_id", sub.PlanID(),
					"error", err,
				)
				continue
			}

			if plan == nil {
				continue
			}

			// Skip non-forward plans
			if !plan.PlanType().IsForward() {
				continue
			}

			planFeatures := plan.Features()
			if planFeatures == nil {
				continue
			}

			isAllowed, err := planFeatures.IsForwardRuleTypeAllowed(requestBody.RuleType)
			if err != nil {
				m.logger.Warnw("failed to check if rule type is allowed",
					"subscription_id", sub.ID(),
					"rule_type", requestBody.RuleType,
					"error", err,
				)
				continue
			}

			if isAllowed {
				allowed = true
				break
			}
		}

		if !allowed {
			m.logger.Warnw("user attempted to create disallowed rule type",
				"user_id", currentUserID,
				"rule_type", requestBody.RuleType,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "rule type not allowed by your subscription plan")
			c.Abort()
			return
		}

		c.Next()
	}
}
