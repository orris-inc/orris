// Package routes provides HTTP route configurations.
package routes

import (
	"github.com/gin-gonic/gin"

	forwardSubscriptionHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/subscription"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
)

// SubscriptionForwardRouteConfig contains dependencies for subscription-scoped forward routes.
type SubscriptionForwardRouteConfig struct {
	SubscriptionForwardHandler  *forwardSubscriptionHandlers.Handler
	AuthMiddleware              *middleware.AuthMiddleware
	SubscriptionOwnerMiddleware *middleware.SubscriptionOwnerMiddleware
	ForwardRuleOwnerMiddleware  *middleware.ForwardRuleOwnerMiddleware
	ForwardQuotaMiddleware      *middleware.ForwardQuotaMiddleware
}

// SetupSubscriptionForwardRoutes configures subscription-scoped forward rule routes.
// Routes: /subscriptions/:sid/forward-rules/*
// :sid is subscription SID (sub_xxx format)
// :rule_id is forward rule SID (fr_xxx format)
func SetupSubscriptionForwardRoutes(engine *gin.Engine, cfg *SubscriptionForwardRouteConfig) {
	// Subscription-scoped forward rules API
	// All routes require authentication and subscription ownership verification
	subscriptionForwardRules := engine.Group("/subscriptions/:sid/forward-rules")
	subscriptionForwardRules.Use(cfg.AuthMiddleware.RequireAuth())
	subscriptionForwardRules.Use(cfg.SubscriptionOwnerMiddleware.RequireOwnership())
	{
		// Collection operations
		// Use subscription-based quota checks instead of user-based
		subscriptionForwardRules.POST("",
			cfg.ForwardQuotaMiddleware.CheckSubscriptionRuleLimit(),
			cfg.ForwardQuotaMiddleware.CheckSubscriptionRuleTypeAllowed(),
			cfg.SubscriptionForwardHandler.CreateRule,
		)
		subscriptionForwardRules.GET("", cfg.SubscriptionForwardHandler.ListRules)
		subscriptionForwardRules.PATCH("/reorder", cfg.SubscriptionForwardHandler.ReorderRules)

		// Quota usage for this subscription
		subscriptionForwardRules.GET("/usage", cfg.SubscriptionForwardHandler.GetUsage)

		// Single rule operations (require ownership check)
		ruleGroup := subscriptionForwardRules.Group("/:rule_id")
		ruleGroup.Use(cfg.ForwardRuleOwnerMiddleware.RequireOwnershipByRuleID())
		{
			ruleGroup.GET("", cfg.SubscriptionForwardHandler.GetRule)
			ruleGroup.PUT("", cfg.SubscriptionForwardHandler.UpdateRule)
			ruleGroup.DELETE("", cfg.SubscriptionForwardHandler.DeleteRule)
			ruleGroup.POST("/enable", cfg.SubscriptionForwardHandler.EnableRule)
			ruleGroup.POST("/disable", cfg.SubscriptionForwardHandler.DisableRule)
		}
	}
}
