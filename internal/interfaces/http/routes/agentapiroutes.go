package routes

import (
	"github.com/gin-gonic/gin"

	nodeHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/node"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
)

// AgentAPIRouteConfig holds dependencies for node agent REST API routes.
type AgentAPIRouteConfig struct {
	AgentHandler        *nodeHandlers.AgentHandler
	NodeTokenMiddleware *middleware.NodeTokenMiddleware
}

// SetupAgentAPIRoutes configures RESTful Agent API routes.
func SetupAgentAPIRoutes(engine *gin.Engine, cfg *AgentAPIRouteConfig) {
	agentAPI := engine.Group("/agents")
	agentAPI.Use(cfg.NodeTokenMiddleware.RequireNodeTokenHeader())
	{
		agentAPI.GET("/:nodesid/config", cfg.AgentHandler.GetConfig)
		agentAPI.GET("/:nodesid/subscriptions", cfg.AgentHandler.GetSubscriptions)
		agentAPI.POST("/:nodesid/traffic", cfg.AgentHandler.ReportTraffic)
		agentAPI.PUT("/:nodesid/status", cfg.AgentHandler.UpdateStatus)
		agentAPI.PUT("/:nodesid/online-subscriptions", cfg.AgentHandler.UpdateOnlineSubscriptions)
	}
}
