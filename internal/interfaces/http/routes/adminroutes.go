package routes

import (
	"github.com/gin-gonic/gin"

	adminHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin"
	adminResourceGroupHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin/resourcegroup"
	adminSubscriptionHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin/subscription"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/shared/authorization"
)

// AdminRouteConfig holds dependencies for admin-only routes.
type AdminRouteConfig struct {
	AdminSubscriptionHandler  *adminSubscriptionHandlers.Handler
	AdminResourceGroupHandler *adminResourceGroupHandlers.Handler
	AdminTrafficStatsHandler  *adminHandlers.TrafficStatsHandler
	AdminTelegramHandler      *adminHandlers.AdminTelegramHandler // may be nil
	AuthMiddleware            *middleware.AuthMiddleware
}

// SetupAdminRoutes configures admin-only routes.
func SetupAdminRoutes(engine *gin.Engine, cfg *AdminRouteConfig) {
	// Admin subscription routes
	adminSubscriptions := engine.Group("/admin/subscriptions")
	adminSubscriptions.Use(cfg.AuthMiddleware.RequireAuth(), authorization.RequireAdmin())
	{
		adminSubscriptions.POST("", cfg.AdminSubscriptionHandler.Create)
		adminSubscriptions.GET("", cfg.AdminSubscriptionHandler.List)
		adminSubscriptions.GET("/:id", cfg.AdminSubscriptionHandler.Get)
		adminSubscriptions.PATCH("/:id/status", cfg.AdminSubscriptionHandler.UpdateStatus)
		adminSubscriptions.PATCH("/:id/plan", cfg.AdminSubscriptionHandler.ChangePlan)
		adminSubscriptions.POST("/:id/suspend", cfg.AdminSubscriptionHandler.Suspend)
		adminSubscriptions.POST("/:id/unsuspend", cfg.AdminSubscriptionHandler.Unsuspend)
		adminSubscriptions.POST("/:id/reset-usage", cfg.AdminSubscriptionHandler.ResetUsage)
		adminSubscriptions.POST("/:id/renew", cfg.AdminSubscriptionHandler.Renew)
		adminSubscriptions.DELETE("/:id", cfg.AdminSubscriptionHandler.Delete)
	}

	// Admin resource group routes
	adminResourceGroups := engine.Group("/admin/resource-groups")
	adminResourceGroups.Use(cfg.AuthMiddleware.RequireAuth(), authorization.RequireAdmin())
	{
		adminResourceGroups.POST("", cfg.AdminResourceGroupHandler.Create)
		adminResourceGroups.GET("", cfg.AdminResourceGroupHandler.List)
		adminResourceGroups.GET("/:id", cfg.AdminResourceGroupHandler.Get)
		adminResourceGroups.PATCH("/:id", cfg.AdminResourceGroupHandler.Update)
		adminResourceGroups.DELETE("/:id", cfg.AdminResourceGroupHandler.Delete)
		adminResourceGroups.POST("/:id/activate", cfg.AdminResourceGroupHandler.Activate)
		adminResourceGroups.POST("/:id/deactivate", cfg.AdminResourceGroupHandler.Deactivate)

		// Node membership management
		adminResourceGroups.POST("/:id/nodes", cfg.AdminResourceGroupHandler.AddNodes)
		adminResourceGroups.DELETE("/:id/nodes", cfg.AdminResourceGroupHandler.RemoveNodes)
		adminResourceGroups.GET("/:id/nodes", cfg.AdminResourceGroupHandler.ListNodes)

		// Forward agent membership management
		adminResourceGroups.POST("/:id/forward-agents", cfg.AdminResourceGroupHandler.AddForwardAgents)
		adminResourceGroups.DELETE("/:id/forward-agents", cfg.AdminResourceGroupHandler.RemoveForwardAgents)
		adminResourceGroups.GET("/:id/forward-agents", cfg.AdminResourceGroupHandler.ListForwardAgents)

		// Forward rule membership management
		adminResourceGroups.POST("/:id/forward-rules", cfg.AdminResourceGroupHandler.AddForwardRules)
		adminResourceGroups.DELETE("/:id/forward-rules", cfg.AdminResourceGroupHandler.RemoveForwardRules)
		adminResourceGroups.GET("/:id/forward-rules", cfg.AdminResourceGroupHandler.ListForwardRules)
	}

	// Admin traffic stats routes
	adminTrafficStats := engine.Group("/admin/traffic-stats")
	adminTrafficStats.Use(cfg.AuthMiddleware.RequireAuth(), authorization.RequireAdmin())
	{
		adminTrafficStats.GET("/overview", cfg.AdminTrafficStatsHandler.GetOverview)
		adminTrafficStats.GET("/users", cfg.AdminTrafficStatsHandler.GetUserStats)
		adminTrafficStats.GET("/subscriptions", cfg.AdminTrafficStatsHandler.GetSubscriptionStats)
		adminTrafficStats.GET("/nodes", cfg.AdminTrafficStatsHandler.GetNodeStats)
		adminTrafficStats.GET("/ranking/users", cfg.AdminTrafficStatsHandler.GetUserRanking)
		adminTrafficStats.GET("/ranking/subscriptions", cfg.AdminTrafficStatsHandler.GetSubscriptionRanking)
		adminTrafficStats.GET("/trend", cfg.AdminTrafficStatsHandler.GetTrend)
	}

	// Admin telegram routes (only if handler is initialized)
	if cfg.AdminTelegramHandler != nil {
		adminTelegram := engine.Group("/admin/telegram")
		adminTelegram.Use(cfg.AuthMiddleware.RequireAuth(), authorization.RequireAdmin())
		{
			adminTelegram.GET("/binding", cfg.AdminTelegramHandler.GetBindingStatus)
			adminTelegram.DELETE("/binding", cfg.AdminTelegramHandler.Unbind)
			adminTelegram.PATCH("/preferences", cfg.AdminTelegramHandler.UpdatePreferences)
		}
	}
}
