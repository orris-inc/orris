// Package middleware provides HTTP middleware for the application.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ForwardRuleOwnerMiddleware ensures users can only access their own forward rules
type ForwardRuleOwnerMiddleware struct {
	forwardRuleRepo forward.Repository
	logger          logger.Interface
}

// NewForwardRuleOwnerMiddleware creates a new forward rule owner middleware
func NewForwardRuleOwnerMiddleware(
	forwardRuleRepo forward.Repository,
	logger logger.Interface,
) *ForwardRuleOwnerMiddleware {
	return &ForwardRuleOwnerMiddleware{
		forwardRuleRepo: forwardRuleRepo,
		logger:          logger,
	}
}

// RequireOwnership ensures the authenticated user owns the forward rule
func (m *ForwardRuleOwnerMiddleware) RequireOwnership() gin.HandlerFunc {
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

		ruleID := c.Param("id")
		if ruleID == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "forward rule ID is required")
			c.Abort()
			return
		}

		// Validate Stripe-style ID format (e.g., "fr_xK9mP2vL3nQ")
		if err := id.ValidatePrefix(ruleID, id.PrefixForwardRule); err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid forward rule ID format, expected fr_xxxxx")
			c.Abort()
			return
		}

		// Look up the rule by SID (database stores full prefixed ID like "fr_xxx")
		rule, err := m.forwardRuleRepo.GetBySID(c.Request.Context(), ruleID)
		if err != nil {
			m.logger.Warnw("failed to get forward rule for ownership check",
				"forward_rule_id", ruleID,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusNotFound, "forward rule not found")
			c.Abort()
			return
		}

		if rule == nil {
			utils.ErrorResponse(c, http.StatusNotFound, "forward rule not found")
			c.Abort()
			return
		}

		currentUserID, ok := userID.(uint)
		if !ok {
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid user ID type")
			c.Abort()
			return
		}

		// Admin can access all rules
		if isAdmin {
			c.Set("forward_rule", rule)
			c.Set("forward_rule_id", ruleID)
			c.Next()
			return
		}

		// Check if rule has user_id (user-owned rule)
		if !rule.IsUserOwned() {
			// user_id is NULL, only admin can access
			m.logger.Warnw("non-admin user attempted to access admin-owned forward rule",
				"current_user_id", currentUserID,
				"forward_rule_id", ruleID,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "access denied")
			c.Abort()
			return
		}

		// Verify user owns the rule
		if rule.UserID() == nil || *rule.UserID() != currentUserID {
			m.logger.Warnw("user attempted to access another user's forward rule",
				"current_user_id", currentUserID,
				"forward_rule_owner_id", rule.UserID(),
				"forward_rule_id", ruleID,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "access denied")
			c.Abort()
			return
		}

		// Store forward rule in context for handler reuse
		c.Set("forward_rule", rule)
		c.Set("forward_rule_id", ruleID)

		c.Next()
	}
}

// RequireOwnershipByRuleID ensures the authenticated user owns the forward rule
// and the rule belongs to the subscription specified in the URL.
// This method gets rule ID from :rule_id URL parameter instead of :id.
// Designed for routes like /subscriptions/:sid/forward-rules/:rule_id
//
// RESTful semantics: If the rule exists but doesn't belong to the subscription,
// returns 404 (resource not found in this context) rather than 403.
func (m *ForwardRuleOwnerMiddleware) RequireOwnershipByRuleID() gin.HandlerFunc {
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

		ruleID := c.Param("rule_id")
		if ruleID == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "forward rule ID is required")
			c.Abort()
			return
		}

		// Validate Stripe-style ID format (e.g., "fr_xK9mP2vL3nQ")
		if err := id.ValidatePrefix(ruleID, id.PrefixForwardRule); err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid forward rule ID format, expected fr_xxxxx")
			c.Abort()
			return
		}

		// Look up the rule by SID (database stores full prefixed ID like "fr_xxx")
		rule, err := m.forwardRuleRepo.GetBySID(c.Request.Context(), ruleID)
		if err != nil {
			m.logger.Warnw("failed to get forward rule for ownership check",
				"forward_rule_id", ruleID,
				"error", err,
			)
			utils.ErrorResponse(c, http.StatusNotFound, "forward rule not found")
			c.Abort()
			return
		}

		if rule == nil {
			utils.ErrorResponse(c, http.StatusNotFound, "forward rule not found")
			c.Abort()
			return
		}

		currentUserID, ok := userID.(uint)
		if !ok {
			utils.ErrorResponse(c, http.StatusInternalServerError, "invalid user ID type")
			c.Abort()
			return
		}

		// Verify rule belongs to the subscription specified in URL (RESTful resource hierarchy)
		// This check applies to both admin and regular users to enforce proper resource nesting
		if subscriptionID, exists := c.Get("subscription_id"); exists {
			expectedSubID, ok := subscriptionID.(uint)
			if !ok {
				m.logger.Errorw("invalid subscription_id type in context",
					"subscription_id", subscriptionID,
				)
				utils.ErrorResponse(c, http.StatusInternalServerError, "invalid subscription context")
				c.Abort()
				return
			}

			// Rule must belong to the subscription in the URL path
			ruleSubID := rule.SubscriptionID()
			if ruleSubID == nil || *ruleSubID != expectedSubID {
				m.logger.Warnw("forward rule does not belong to subscription in URL",
					"forward_rule_id", ruleID,
					"url_subscription_id", expectedSubID,
					"rule_subscription_id", ruleSubID,
					"user_id", currentUserID,
				)
				// Return 404 per RESTful semantics: resource not found in this context
				utils.ErrorResponse(c, http.StatusNotFound, "forward rule not found")
				c.Abort()
				return
			}
		}

		// Admin can access all rules (after subscription hierarchy check)
		if isAdmin {
			c.Set("forward_rule", rule)
			c.Set("forward_rule_id", ruleID)
			c.Next()
			return
		}

		// Check if rule has user_id (user-owned rule)
		if !rule.IsUserOwned() {
			// user_id is NULL, only admin can access
			m.logger.Warnw("non-admin user attempted to access admin-owned forward rule",
				"current_user_id", currentUserID,
				"forward_rule_id", ruleID,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "access denied")
			c.Abort()
			return
		}

		// Verify user owns the rule
		if rule.UserID() == nil || *rule.UserID() != currentUserID {
			m.logger.Warnw("user attempted to access another user's forward rule",
				"current_user_id", currentUserID,
				"forward_rule_owner_id", rule.UserID(),
				"forward_rule_id", ruleID,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "access denied")
			c.Abort()
			return
		}

		// Store forward rule in context for handler reuse
		c.Set("forward_rule", rule)
		c.Set("forward_rule_id", ruleID)

		c.Next()
	}
}
