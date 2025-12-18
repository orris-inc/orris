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

		prefixedID := c.Param("id")
		if prefixedID == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "forward rule ID is required")
			c.Abort()
			return
		}

		// Parse Stripe-style ID (e.g., "fr_xK9mP2vL3nQ" -> "xK9mP2vL3nQ")
		shortID, err := id.ParseForwardRuleID(prefixedID)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid forward rule ID format, expected fr_xxxxx")
			c.Abort()
			return
		}

		rule, err := m.forwardRuleRepo.GetByShortID(c.Request.Context(), shortID)
		if err != nil {
			m.logger.Warnw("failed to get forward rule for ownership check",
				"forward_rule_short_id", shortID,
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
			c.Set("forward_rule_short_id", shortID)
			c.Next()
			return
		}

		// Check if rule has user_id (user-owned rule)
		if !rule.IsUserOwned() {
			// user_id is NULL, only admin can access
			m.logger.Warnw("non-admin user attempted to access admin-owned forward rule",
				"current_user_id", currentUserID,
				"forward_rule_short_id", shortID,
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
				"forward_rule_short_id", shortID,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "access denied")
			c.Abort()
			return
		}

		// Store forward rule in context for handler reuse
		c.Set("forward_rule", rule)
		c.Set("forward_rule_short_id", shortID)

		c.Next()
	}
}
