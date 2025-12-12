// Package routes provides HTTP route configurations.
package routes

import (
	"github.com/gin-gonic/gin"

	forwardAgentAPIHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/agent"
	forwardHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/shared/authorization"
)

// ForwardRouteConfig contains dependencies for forward routes.
type ForwardRouteConfig struct {
	ForwardRuleHandler          *forwardHandlers.ForwardHandler
	ForwardAgentHandler         *forwardHandlers.ForwardAgentHandler
	ForwardAgentAPIHandler      *forwardAgentAPIHandlers.Handler
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

		// Probe operations
		forwardRules.POST("/:id/probe", cfg.ForwardRuleHandler.ProbeRule)
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

		// Runtime status (from agent reports)
		forwardAgents.GET("/:id/status", cfg.ForwardAgentHandler.GetAgentStatus)

		// Token operations
		forwardAgents.GET("/:id/token", cfg.ForwardAgentHandler.GetToken)
		forwardAgents.POST("/:id/regenerate-token", cfg.ForwardAgentHandler.RegenerateToken)

		// Install script
		forwardAgents.GET("/:id/install-script", cfg.ForwardAgentHandler.GetInstallScript)
	}

	// Forward agent API for clients to fetch rules and report traffic
	forwardAgentAPI := engine.Group("/forward-agent-api")
	forwardAgentAPI.Use(cfg.ForwardAgentTokenMiddleware.RequireForwardAgentToken())
	{
		forwardAgentAPI.GET("/rules", cfg.ForwardAgentAPIHandler.GetEnabledRules)
		forwardAgentAPI.GET("/rules/:rule_id", cfg.ForwardAgentAPIHandler.RefreshRule)
		forwardAgentAPI.POST("/traffic", cfg.ForwardAgentAPIHandler.ReportTraffic)
		forwardAgentAPI.POST("/status", cfg.ForwardAgentAPIHandler.ReportStatus)
		forwardAgentAPI.GET("/exit-endpoint/:agent_id", cfg.ForwardAgentAPIHandler.GetExitEndpoint)
	}
}
