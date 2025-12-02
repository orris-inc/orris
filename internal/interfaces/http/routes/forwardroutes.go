// Package routes provides HTTP route configurations.
package routes

import (
	"github.com/gin-gonic/gin"

	forwardHandlers "orris/internal/interfaces/http/handlers/forward"
	"orris/internal/interfaces/http/middleware"
	"orris/internal/shared/authorization"
)

// ForwardRouteConfig contains dependencies for forward routes.
type ForwardRouteConfig struct {
	ForwardRuleHandler          *forwardHandlers.ForwardHandler
	ForwardAgentHandler         *forwardHandlers.ForwardAgentHandler
	ForwardAgentAPIHandler      *forwardHandlers.AgentHandler
	AuthMiddleware              *middleware.AuthMiddleware
	ForwardAgentTokenMiddleware *middleware.ForwardAgentTokenMiddleware
}

// SetupForwardRoutes configures forward-related routes.
func SetupForwardRoutes(engine *gin.Engine, cfg *ForwardRouteConfig) {
	// Forward rules management (admin only)
	forwardRules := engine.Group("/forward-rules")
	forwardRules.Use(cfg.AuthMiddleware.RequireAuth())
	forwardRules.Use(authorization.RequireAdmin())
	{
		// Collection operations
		forwardRules.POST("", cfg.ForwardRuleHandler.CreateRule)
		forwardRules.GET("", cfg.ForwardRuleHandler.ListRules)

		// Resource operations
		forwardRules.GET("/:id", cfg.ForwardRuleHandler.GetRule)
		forwardRules.PUT("/:id", cfg.ForwardRuleHandler.UpdateRule)
		forwardRules.DELETE("/:id", cfg.ForwardRuleHandler.DeleteRule)

		// Status operations
		forwardRules.PATCH("/:id/status", cfg.ForwardRuleHandler.UpdateStatus)
		forwardRules.POST("/:id/enable", cfg.ForwardRuleHandler.EnableRule)
		forwardRules.POST("/:id/disable", cfg.ForwardRuleHandler.DisableRule)

		// Traffic operations
		forwardRules.POST("/:id/reset-traffic", cfg.ForwardRuleHandler.ResetTraffic)
	}

	// Forward agents management (admin only)
	forwardAgents := engine.Group("/forward-agents")
	forwardAgents.Use(cfg.AuthMiddleware.RequireAuth())
	forwardAgents.Use(authorization.RequireAdmin())
	{
		// Collection operations
		forwardAgents.POST("", cfg.ForwardAgentHandler.CreateAgent)
		forwardAgents.GET("", cfg.ForwardAgentHandler.ListAgents)

		// Resource operations
		forwardAgents.GET("/:id", cfg.ForwardAgentHandler.GetAgent)
		forwardAgents.PUT("/:id", cfg.ForwardAgentHandler.UpdateAgent)
		forwardAgents.DELETE("/:id", cfg.ForwardAgentHandler.DeleteAgent)

		// Status operations
		forwardAgents.PATCH("/:id/status", cfg.ForwardAgentHandler.UpdateStatus)
		forwardAgents.POST("/:id/enable", cfg.ForwardAgentHandler.EnableAgent)
		forwardAgents.POST("/:id/disable", cfg.ForwardAgentHandler.DisableAgent)

		// Token operations
		forwardAgents.POST("/:id/regenerate-token", cfg.ForwardAgentHandler.RegenerateToken)
	}

	// Forward agent API for clients to fetch rules and report traffic
	forwardAgentAPI := engine.Group("/forward-agent-api")
	forwardAgentAPI.Use(cfg.ForwardAgentTokenMiddleware.RequireForwardAgentToken())
	{
		forwardAgentAPI.GET("/rules", cfg.ForwardAgentAPIHandler.GetEnabledRules)
		forwardAgentAPI.POST("/traffic", cfg.ForwardAgentAPIHandler.ReportTraffic)
	}
}
