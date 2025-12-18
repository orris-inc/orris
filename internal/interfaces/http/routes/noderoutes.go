package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/interfaces/http/handlers"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/shared/authorization"
)

// NodeRouteConfig holds dependencies for node routes
type NodeRouteConfig struct {
	NodeHandler         *handlers.NodeHandler
	SubscriptionHandler *handlers.NodeSubscriptionHandler
	AuthMiddleware      *middleware.AuthMiddleware
	SubscriptionTokenMW *middleware.SubscriptionTokenMiddleware
	NodeTokenMW         *middleware.NodeTokenMiddleware
	RateLimiter         *middleware.RateLimiter
}

// SetupNodeRoutes configures all node management routes
func SetupNodeRoutes(engine *gin.Engine, config *NodeRouteConfig) {
	// Node management routes - require authentication and permission
	nodes := engine.Group("/nodes")
	nodes.Use(config.AuthMiddleware.RequireAuth())
	{
		// IMPORTANT: Register specific paths BEFORE parameterized paths to avoid route conflicts
		// Action endpoints like /:id/activate must come before generic /:id routes

		// Collection operations (no ID parameter)
		nodes.POST("",
			authorization.RequireAdmin(),
			config.NodeHandler.CreateNode)
		nodes.GET("",
			authorization.RequireAdmin(),
			config.NodeHandler.ListNodes)

		// Specific action endpoints (must come BEFORE /:id to avoid conflicts)
		// Using PATCH for state changes as per RESTful best practices
		nodes.PATCH("/:id/status",
			authorization.RequireAdmin(),
			config.NodeHandler.UpdateNodeStatus)
		// Using POST for creating new token sub-resource
		nodes.POST("/:id/tokens",
			authorization.RequireAdmin(),
			config.NodeHandler.GenerateToken)
		// Using GET for retrieving install script
		nodes.GET("/:id/install-script",
			authorization.RequireAdmin(),
			config.NodeHandler.GetInstallScript)

		// Generic parameterized routes (must come LAST)
		nodes.GET("/:id",
			authorization.RequireAdmin(),
			config.NodeHandler.GetNode)
		nodes.PUT("/:id",
			authorization.RequireAdmin(),
			config.NodeHandler.UpdateNode)
		nodes.DELETE("/:id",
			authorization.RequireAdmin(),
			config.NodeHandler.DeleteNode)
	}

	// Subscription routes - public access with token validation
	sub := engine.Group("/s")
	{
		// Base64 subscription format (default)
		sub.GET("/:token",
			config.RateLimiter.Limit(),
			config.SubscriptionHandler.GetSubscription)

		// Clash subscription format
		sub.GET("/:token/clash",
			config.RateLimiter.Limit(),
			config.SubscriptionHandler.GetClashSubscription)

		// V2Ray subscription format
		sub.GET("/:token/v2ray",
			config.RateLimiter.Limit(),
			config.SubscriptionHandler.GetV2RaySubscription)

		// SIP008 subscription format
		sub.GET("/:token/sip008",
			config.RateLimiter.Limit(),
			config.SubscriptionHandler.GetSIP008Subscription)

		// Surge subscription format
		sub.GET("/:token/surge",
			config.RateLimiter.Limit(),
			config.SubscriptionHandler.GetSurgeSubscription)
	}
}
