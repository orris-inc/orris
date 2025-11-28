package routes

import (
	"github.com/gin-gonic/gin"

	"orris/internal/interfaces/http/handlers"
	"orris/internal/interfaces/http/middleware"
	"orris/internal/shared/authorization"
)

// NodeRouteConfig holds dependencies for node routes
type NodeRouteConfig struct {
	NodeHandler         *handlers.NodeHandler
	NodeGroupHandler    *handlers.NodeGroupHandler
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
		nodes.POST("/:id/activate",
			authorization.RequireAdmin(),
			config.NodeHandler.ActivateNode)
		nodes.POST("/:id/deactivate",
			authorization.RequireAdmin(),
			config.NodeHandler.DeactivateNode)
		nodes.POST("/:id/token",
			authorization.RequireAdmin(),
			config.NodeHandler.GenerateToken)

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

	// Node group management routes - require authentication and permission
	nodeGroups := engine.Group("/node-groups")
	nodeGroups.Use(config.AuthMiddleware.RequireAuth())
	{
		// IMPORTANT: Register specific paths BEFORE parameterized paths to avoid route conflicts

		// Collection operations (no ID parameter)
		nodeGroups.POST("",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.CreateNodeGroup)
		nodeGroups.GET("",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.ListNodeGroups)

		// Specific sub-resource endpoints (must come BEFORE generic /:id routes)
		// Batch operations - most specific paths first
		nodeGroups.POST("/:id/nodes/batch",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.BatchAddNodesToGroup)
		nodeGroups.DELETE("/:id/nodes/batch",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.BatchRemoveNodesFromGroup)

		// Node-Group relationship management
		nodeGroups.POST("/:id/nodes",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.AddNodeToGroup)
		nodeGroups.GET("/:id/nodes",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.ListGroupNodes)
		nodeGroups.DELETE("/:id/nodes/:node_id",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.RemoveNodeFromGroup)

		// Subscription plan association
		nodeGroups.POST("/:id/plans",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.AssociatePlan)
		nodeGroups.DELETE("/:id/plans/:plan_id",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.DisassociatePlan)

		// Generic parameterized routes (must come LAST)
		nodeGroups.GET("/:id",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.GetNodeGroup)
		nodeGroups.PUT("/:id",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.UpdateNodeGroup)
		nodeGroups.DELETE("/:id",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.DeleteNodeGroup)
	}

	// Subscription routes - public access with token validation
	sub := engine.Group("/sub")
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
