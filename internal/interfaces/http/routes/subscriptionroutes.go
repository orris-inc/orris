// Package routes provides HTTP route configurations.
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/interfaces/http/handlers"
	forwardSubscriptionHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/subscription"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
)

// SubscriptionRouteConfig holds dependencies for user subscription routes.
type SubscriptionRouteConfig struct {
	SubscriptionHandler         *handlers.SubscriptionHandler
	SubscriptionTokenHandler    *handlers.SubscriptionTokenHandler
	AuthMiddleware              *middleware.AuthMiddleware
	SubscriptionOwnerMiddleware *middleware.SubscriptionOwnerMiddleware
}

// SetupSubscriptionRoutes configures user subscription routes.
func SetupSubscriptionRoutes(engine *gin.Engine, cfg *SubscriptionRouteConfig) {
	// User subscription routes - only own subscriptions
	subscriptions := engine.Group("/subscriptions")
	subscriptions.Use(cfg.AuthMiddleware.RequireAuth())
	{
		// Collection operations (no ownership check needed)
		subscriptions.POST("", cfg.SubscriptionHandler.CreateSubscription)
		subscriptions.GET("", cfg.SubscriptionHandler.ListUserSubscriptions)

		// Operations on specific subscription (ownership verified by middleware)
		// :sid is subscription SID (sub_xxx format)
		subscriptionWithOwnership := subscriptions.Group("/:sid")
		subscriptionWithOwnership.Use(cfg.SubscriptionOwnerMiddleware.RequireOwnership())
		{
			subscriptionWithOwnership.GET("", cfg.SubscriptionHandler.GetSubscription)
			subscriptionWithOwnership.PATCH("/status", cfg.SubscriptionHandler.UpdateStatus)
			subscriptionWithOwnership.PATCH("/plan", cfg.SubscriptionHandler.ChangePlan)
			subscriptionWithOwnership.PUT("/link", cfg.SubscriptionHandler.ResetLink)
			subscriptionWithOwnership.DELETE("", cfg.SubscriptionHandler.DeleteSubscription)

			// Token sub-resource endpoints
			// :token_id is token SID (subtk_xxx format)
			subscriptionWithOwnership.POST("/tokens/:token_id/refresh", cfg.SubscriptionTokenHandler.RefreshToken)
			subscriptionWithOwnership.DELETE("/tokens/:token_id", cfg.SubscriptionTokenHandler.RevokeToken)
			subscriptionWithOwnership.POST("/tokens", cfg.SubscriptionTokenHandler.GenerateToken)
			subscriptionWithOwnership.GET("/tokens", cfg.SubscriptionTokenHandler.ListTokens)

			// Traffic statistics endpoint
			subscriptionWithOwnership.GET("/traffic-stats", cfg.SubscriptionHandler.GetTrafficStats)
		}
	}
}

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
