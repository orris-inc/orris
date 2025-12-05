// Package routes provides HTTP route configurations.
package routes

import (
	"github.com/gin-gonic/gin"

	agentHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/agent"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
)

// AgentHubRouteConfig contains dependencies for agent hub routes.
type AgentHubRouteConfig struct {
	HubHandler                  *agentHandlers.HubHandler
	ForwardAgentTokenMiddleware *middleware.ForwardAgentTokenMiddleware
}

// SetupAgentHubRoutes configures agent hub WebSocket routes.
func SetupAgentHubRoutes(engine *gin.Engine, cfg *AgentHubRouteConfig) {
	// WebSocket route for forward agent (probe functionality)
	wsAgent := engine.Group("/ws")
	{
		// Forward agent WebSocket connection
		// GET /ws/forward-agent (authenticated by forward agent token)
		wsAgent.GET("/forward-agent",
			cfg.ForwardAgentTokenMiddleware.RequireForwardAgentToken(),
			cfg.HubHandler.ForwardAgentWS,
		)
	}
}
