// Package routes provides HTTP route configurations.
package routes

import (
	"github.com/gin-gonic/gin"

	forwardAgentHubHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/agent/hub"
	nodeHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/node"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
)

// AgentHubRouteConfig contains dependencies for agent hub routes.
type AgentHubRouteConfig struct {
	HubHandler                  *forwardAgentHubHandlers.Handler
	NodeHubHandler              *nodeHandlers.NodeHubHandler
	ForwardAgentTokenMiddleware *middleware.ForwardAgentTokenMiddleware
	NodeTokenMiddleware         *middleware.NodeTokenMiddleware
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

		// Node agent WebSocket connection
		// GET /ws/node-agent?token=xxx (authenticated by node token)
		if cfg.NodeHubHandler != nil && cfg.NodeTokenMiddleware != nil {
			wsAgent.GET("/node-agent",
				cfg.NodeTokenMiddleware.RequireNodeTokenWS(),
				cfg.NodeHubHandler.NodeAgentWS,
			)
		}
	}
}
