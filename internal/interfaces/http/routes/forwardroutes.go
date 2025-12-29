// Package routes provides HTTP route configurations.
package routes

import (
	"github.com/gin-gonic/gin"

	forwardAgentAPIHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/agent/api"
	forwardAgentCrudHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/agent/crud"
	forwardRuleHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/rule"
	forwardUserHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/user"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/shared/authorization"
)

// ForwardRouteConfig contains dependencies for forward routes.
type ForwardRouteConfig struct {
	ForwardRuleHandler          *forwardRuleHandlers.Handler
	ForwardAgentHandler         *forwardAgentCrudHandlers.Handler
	ForwardAgentAPIHandler      *forwardAgentAPIHandlers.Handler
	UserForwardHandler          *forwardUserHandlers.Handler
	AuthMiddleware              *middleware.AuthMiddleware
	ForwardAgentTokenMiddleware *middleware.ForwardAgentTokenMiddleware
	ForwardRuleOwnerMiddleware  *middleware.ForwardRuleOwnerMiddleware
	ForwardQuotaMiddleware      *middleware.ForwardQuotaMiddleware
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
		forwardRules.PATCH("/reorder", cfg.ForwardRuleHandler.ReorderRules)

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

		// Rule status (aggregated from all agents)
		forwardRules.GET("/:id/status", cfg.ForwardAgentHandler.GetRuleOverallStatus)
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

	// User forward rules API (requires authentication and quota limits)
	userForwardRules := engine.Group("/user/forward-rules")
	userForwardRules.Use(cfg.AuthMiddleware.RequireAuth())
	{
		// Collection operations
		userForwardRules.POST("",
			cfg.ForwardQuotaMiddleware.CheckRuleLimit(),
			cfg.ForwardQuotaMiddleware.CheckRuleTypeAllowed(),
			cfg.UserForwardHandler.CreateRule,
		)
		userForwardRules.GET("", cfg.UserForwardHandler.ListRules)
		userForwardRules.PATCH("/reorder", cfg.UserForwardHandler.ReorderRules)

		// Quota usage
		userForwardRules.GET("/usage", cfg.UserForwardHandler.GetUsage)

		// Single rule operations (require ownership check)
		ruleGroup := userForwardRules.Group("/:id")
		ruleGroup.Use(cfg.ForwardRuleOwnerMiddleware.RequireOwnership())
		{
			ruleGroup.GET("", cfg.UserForwardHandler.GetRule)
			ruleGroup.PUT("", cfg.UserForwardHandler.UpdateRule)
			ruleGroup.DELETE("", cfg.UserForwardHandler.DeleteRule)
			ruleGroup.POST("/enable", cfg.UserForwardHandler.EnableRule)
			ruleGroup.POST("/disable", cfg.UserForwardHandler.DisableRule)
		}
	}

	// User forward agents API (read-only access to agents through subscriptions)
	userForwardAgents := engine.Group("/user/forward-agents")
	userForwardAgents.Use(cfg.AuthMiddleware.RequireAuth())
	{
		userForwardAgents.GET("", cfg.UserForwardHandler.ListAgents)
	}

	// Forward agent API for clients to fetch rules and report traffic
	forwardAgentAPI := engine.Group("/forward-agent-api")
	forwardAgentAPI.Use(cfg.ForwardAgentTokenMiddleware.RequireForwardAgentToken())
	{
		forwardAgentAPI.GET("/rules", cfg.ForwardAgentAPIHandler.GetEnabledRules)
		forwardAgentAPI.GET("/rules/:rule_id", cfg.ForwardAgentAPIHandler.RefreshRule)
		forwardAgentAPI.POST("/traffic", cfg.ForwardAgentAPIHandler.ReportTraffic)
		forwardAgentAPI.POST("/status", cfg.ForwardAgentAPIHandler.ReportStatus)
		forwardAgentAPI.POST("/rule-sync-status", cfg.ForwardAgentAPIHandler.ReportRuleSyncStatus)
		forwardAgentAPI.GET("/exit-endpoint/:agent_id", cfg.ForwardAgentAPIHandler.GetExitEndpoint)
		forwardAgentAPI.POST("/verify-tunnel-handshake", cfg.ForwardAgentAPIHandler.VerifyTunnelHandshake)
	}
}
