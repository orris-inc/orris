package routes

import (
	"github.com/gin-gonic/gin"

	"orris/internal/interfaces/http/handlers"
	"orris/internal/interfaces/http/middleware"
)

// NodeRouteConfig holds dependencies for node routes
type NodeRouteConfig struct {
	NodeHandler              *handlers.NodeHandler
	NodeGroupHandler         *handlers.NodeGroupHandler
	SubscriptionHandler      *handlers.NodeSubscriptionHandler
	NodeReportHandler        *handlers.NodeReportHandler
	AuthMiddleware           *middleware.AuthMiddleware
	PermissionMiddleware     *middleware.PermissionMiddleware
	SubscriptionTokenMW      *middleware.SubscriptionTokenMiddleware
	NodeTokenMW              *middleware.NodeTokenMiddleware
	RateLimiter              *middleware.RateLimiter
}

// SetupNodeRoutes configures all node management routes
func SetupNodeRoutes(engine *gin.Engine, config *NodeRouteConfig) {
	// Node management routes - require authentication and permission
	nodes := engine.Group("/nodes")
	nodes.Use(config.AuthMiddleware.RequireAuth())
	{
		// Node CRUD operations
		nodes.POST("",
			config.PermissionMiddleware.RequirePermission("node", "create"),
			config.NodeHandler.CreateNode)
		nodes.GET("",
			config.PermissionMiddleware.RequirePermission("node", "list"),
			config.NodeHandler.ListNodes)
		nodes.GET("/:id",
			config.PermissionMiddleware.RequirePermission("node", "read"),
			config.NodeHandler.GetNode)
		nodes.PUT("/:id",
			config.PermissionMiddleware.RequirePermission("node", "update"),
			config.NodeHandler.UpdateNode)
		nodes.DELETE("/:id",
			config.PermissionMiddleware.RequirePermission("node", "delete"),
			config.NodeHandler.DeleteNode)

		// Node activation management
		nodes.POST("/:id/activate",
			config.PermissionMiddleware.RequirePermission("node", "update"),
			config.NodeHandler.ActivateNode)
		nodes.POST("/:id/deactivate",
			config.PermissionMiddleware.RequirePermission("node", "update"),
			config.NodeHandler.DeactivateNode)

		// Node token management
		nodes.POST("/:id/token",
			config.PermissionMiddleware.RequirePermission("node", "update"),
			config.NodeHandler.GenerateToken)

		// Node traffic statistics
		nodes.GET("/:id/traffic",
			config.PermissionMiddleware.RequirePermission("node", "read"),
			config.NodeHandler.GetNodeTraffic)
	}

	// Node group management routes - require authentication and permission
	nodeGroups := engine.Group("/node-groups")
	nodeGroups.Use(config.AuthMiddleware.RequireAuth())
	{
		// Node group CRUD operations
		nodeGroups.POST("",
			config.PermissionMiddleware.RequirePermission("node_group", "create"),
			config.NodeGroupHandler.CreateNodeGroup)
		nodeGroups.GET("",
			config.PermissionMiddleware.RequirePermission("node_group", "list"),
			config.NodeGroupHandler.ListNodeGroups)
		nodeGroups.GET("/:id",
			config.PermissionMiddleware.RequirePermission("node_group", "read"),
			config.NodeGroupHandler.GetNodeGroup)
		nodeGroups.PUT("/:id",
			config.PermissionMiddleware.RequirePermission("node_group", "update"),
			config.NodeGroupHandler.UpdateNodeGroup)
		nodeGroups.DELETE("/:id",
			config.PermissionMiddleware.RequirePermission("node_group", "delete"),
			config.NodeGroupHandler.DeleteNodeGroup)

		// Node-Group relationship management
		nodeGroups.POST("/:id/nodes",
			config.PermissionMiddleware.RequirePermission("node_group", "update"),
			config.NodeGroupHandler.AddNodeToGroup)
		nodeGroups.DELETE("/:id/nodes/:node_id",
			config.PermissionMiddleware.RequirePermission("node_group", "update"),
			config.NodeGroupHandler.RemoveNodeFromGroup)
		nodeGroups.GET("/:id/nodes",
			config.PermissionMiddleware.RequirePermission("node_group", "read"),
			config.NodeGroupHandler.ListGroupNodes)

		// Subscription plan association
		nodeGroups.POST("/:id/plans",
			config.PermissionMiddleware.RequirePermission("node_group", "update"),
			config.NodeGroupHandler.AssociatePlan)
		nodeGroups.DELETE("/:id/plans/:plan_id",
			config.PermissionMiddleware.RequirePermission("node_group", "update"),
			config.NodeGroupHandler.DisassociatePlan)
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

	// Node report routes - token authentication
	report := engine.Group("/nodes/report")
	report.Use(config.NodeTokenMW.RequireNodeToken())
	{
		// Report node data (traffic, status, online users)
		report.POST("",
			config.NodeReportHandler.ReportNodeData)

		// Heartbeat endpoint
		report.POST("/heartbeat",
			config.NodeReportHandler.Heartbeat)
	}
}
