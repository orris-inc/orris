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
	NodeReportHandler   *handlers.NodeReportHandler
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
		// Node CRUD operations
		nodes.POST("",
			authorization.RequireAdmin(),
			config.NodeHandler.CreateNode)
		nodes.GET("",
			authorization.RequireAdmin(),
			config.NodeHandler.ListNodes)
		nodes.GET("/:id",
			authorization.RequireAdmin(),
			config.NodeHandler.GetNode)
		nodes.PUT("/:id",
			authorization.RequireAdmin(),
			config.NodeHandler.UpdateNode)
		nodes.DELETE("/:id",
			authorization.RequireAdmin(),
			config.NodeHandler.DeleteNode)

		// Node activation management
		nodes.POST("/:id/activate",
			authorization.RequireAdmin(),
			config.NodeHandler.ActivateNode)
		nodes.POST("/:id/deactivate",
			authorization.RequireAdmin(),
			config.NodeHandler.DeactivateNode)

		// Node token management
		nodes.POST("/:id/token",
			authorization.RequireAdmin(),
			config.NodeHandler.GenerateToken)

		// Node traffic statistics
		nodes.GET("/:id/traffic",
			authorization.RequireAdmin(),
			config.NodeHandler.GetNodeTraffic)
	}

	// Node group management routes - require authentication and permission
	nodeGroups := engine.Group("/node-groups")
	nodeGroups.Use(config.AuthMiddleware.RequireAuth())
	{
		// Node group CRUD operations
		nodeGroups.POST("",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.CreateNodeGroup)
		nodeGroups.GET("",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.ListNodeGroups)
		nodeGroups.GET("/:id",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.GetNodeGroup)
		nodeGroups.PUT("/:id",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.UpdateNodeGroup)
		nodeGroups.DELETE("/:id",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.DeleteNodeGroup)

		// Node-Group relationship management
		nodeGroups.POST("/:id/nodes",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.AddNodeToGroup)
		nodeGroups.DELETE("/:id/nodes/:node_id",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.RemoveNodeFromGroup)
		nodeGroups.GET("/:id/nodes",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.ListGroupNodes)

		// Subscription plan association
		nodeGroups.POST("/:id/plans",
			authorization.RequireAdmin(),
			config.NodeGroupHandler.AssociatePlan)
		nodeGroups.DELETE("/:id/plans/:plan_id",
			authorization.RequireAdmin(),
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
