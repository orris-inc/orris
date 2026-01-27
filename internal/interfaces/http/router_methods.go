package http

import (
	"context"

	"github.com/gin-gonic/gin"

	settingApp "github.com/orris-inc/orris/internal/application/setting"
	telegramApp "github.com/orris-inc/orris/internal/application/telegram"
	telegramAdminApp "github.com/orris-inc/orris/internal/application/telegram/admin"
	"github.com/orris-inc/orris/internal/infrastructure/config"
	telegramInfra "github.com/orris-inc/orris/internal/infrastructure/telegram"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/interfaces/http/routes"
	"github.com/orris-inc/orris/internal/shared/authorization"
)

// SetupRoutes configures all HTTP routes
func (r *Router) SetupRoutes(cfg *config.Config) {
	r.engine.Use(middleware.Logger())
	r.engine.Use(middleware.Recovery())
	r.engine.Use(middleware.CORS(cfg.Server.AllowedOrigins))

	r.engine.GET("/health", r.userHandler.HealthCheck)
	r.engine.GET("/version", r.userHandler.Version)

	r.setupAuthRoutes()
	r.setupUserRoutes()
	r.setupAdminRoutes()
	r.setupSubscriptionRoutes()
	r.setupPaymentRoutes()
	r.setupPlanRoutes()
	r.setupNodeRoutes()
	r.setupAgentAPIRoutes()
	r.setupExternalRoutes()
}

// setupAuthRoutes configures authentication routes
func (r *Router) setupAuthRoutes() {
	auth := r.engine.Group("/auth")
	{
		auth.POST("/register", r.rateLimiter.Limit(), r.authHandler.Register)
		auth.POST("/login", r.rateLimiter.Limit(), r.authHandler.Login)
		auth.POST("/verify-email", r.authHandler.VerifyEmail)
		auth.GET("/verify-email", r.authHandler.VerifyEmail)
		auth.POST("/forgot-password", r.rateLimiter.Limit(), r.authHandler.ForgotPassword)
		auth.POST("/reset-password", r.authHandler.ResetPassword)

		auth.GET("/oauth/:provider", r.authHandler.InitiateOAuth)
		auth.GET("/oauth/:provider/callback", r.authHandler.HandleOAuthCallback)

		auth.POST("/refresh", r.authHandler.RefreshToken)
		auth.POST("/logout", r.authMiddleware.RequireAuth(), r.authHandler.Logout)
		auth.GET("/me", r.authMiddleware.RequireAuth(), r.authHandler.GetCurrentUser)

		// Passkey (WebAuthn) authentication routes
		if r.passkeyHandler != nil {
			auth.POST("/passkey/register/start", r.authMiddleware.RequireAuth(), r.passkeyHandler.StartRegistration)
			auth.POST("/passkey/register/finish", r.authMiddleware.RequireAuth(), r.passkeyHandler.FinishRegistration)
			auth.POST("/passkey/login/start", r.rateLimiter.Limit(), r.passkeyHandler.StartAuthentication)
			auth.POST("/passkey/login/finish", r.rateLimiter.Limit(), r.passkeyHandler.FinishAuthentication)
			// Passkey signup (new user registration without password)
			auth.POST("/passkey/signup/start", r.rateLimiter.Limit(), r.passkeyHandler.StartSignup)
			auth.POST("/passkey/signup/finish", r.rateLimiter.Limit(), r.passkeyHandler.FinishSignup)
		}
	}
}

// setupUserRoutes configures user management routes
func (r *Router) setupUserRoutes() {
	users := r.engine.Group("/users")
	users.Use(r.authMiddleware.RequireAuth())
	{
		// Collection operations (no ID parameter)
		users.POST("", authorization.RequireAdmin(), r.userHandler.CreateUser)
		users.GET("", authorization.RequireAdmin(), r.userHandler.ListUsers)

		// Specific named endpoints (must come BEFORE /:id to avoid conflicts)
		users.PATCH("/me", r.profileHandler.UpdateProfile)
		users.PUT("/me/password", r.profileHandler.ChangePassword)
		users.GET("/me/dashboard", r.dashboardHandler.GetDashboard)

		// Passkey management routes
		if r.passkeyHandler != nil {
			users.GET("/me/passkeys", r.passkeyHandler.ListPasskeys)
			users.DELETE("/me/passkeys/:id", r.passkeyHandler.DeletePasskey)
		}

		users.GET("/email/:email", authorization.RequireAdmin(), r.userHandler.GetUserByEmail)

		// Generic parameterized routes (must come LAST)
		users.GET("/:id", authorization.RequireAdmin(), r.userHandler.GetUser)
		users.PATCH("/:id", authorization.RequireAdmin(), r.userHandler.UpdateUser)
		users.DELETE("/:id", authorization.RequireAdmin(), r.userHandler.DeleteUser)
		users.PATCH("/:id/password", authorization.RequireAdmin(), r.userHandler.AdminResetPassword)
	}
}

// setupAdminRoutes configures admin-only routes
func (r *Router) setupAdminRoutes() {
	// Admin subscription routes
	adminSubscriptions := r.engine.Group("/admin/subscriptions")
	adminSubscriptions.Use(r.authMiddleware.RequireAuth(), authorization.RequireAdmin())
	{
		adminSubscriptions.POST("", r.adminSubscriptionHandler.Create)
		adminSubscriptions.GET("", r.adminSubscriptionHandler.List)
		adminSubscriptions.GET("/:id", r.adminSubscriptionHandler.Get)
		adminSubscriptions.PATCH("/:id/status", r.adminSubscriptionHandler.UpdateStatus)
		adminSubscriptions.PATCH("/:id/plan", r.adminSubscriptionHandler.ChangePlan)
		adminSubscriptions.POST("/:id/suspend", r.adminSubscriptionHandler.Suspend)
		adminSubscriptions.POST("/:id/unsuspend", r.adminSubscriptionHandler.Unsuspend)
		adminSubscriptions.POST("/:id/reset-usage", r.adminSubscriptionHandler.ResetUsage)
		adminSubscriptions.DELETE("/:id", r.adminSubscriptionHandler.Delete)
	}

	// Admin resource group routes
	adminResourceGroups := r.engine.Group("/admin/resource-groups")
	adminResourceGroups.Use(r.authMiddleware.RequireAuth(), authorization.RequireAdmin())
	{
		adminResourceGroups.POST("", r.adminResourceGroupHandler.Create)
		adminResourceGroups.GET("", r.adminResourceGroupHandler.List)
		adminResourceGroups.GET("/:id", r.adminResourceGroupHandler.Get)
		adminResourceGroups.PATCH("/:id", r.adminResourceGroupHandler.Update)
		adminResourceGroups.DELETE("/:id", r.adminResourceGroupHandler.Delete)
		adminResourceGroups.POST("/:id/activate", r.adminResourceGroupHandler.Activate)
		adminResourceGroups.POST("/:id/deactivate", r.adminResourceGroupHandler.Deactivate)

		// Node membership management
		adminResourceGroups.POST("/:id/nodes", r.adminResourceGroupHandler.AddNodes)
		adminResourceGroups.DELETE("/:id/nodes", r.adminResourceGroupHandler.RemoveNodes)
		adminResourceGroups.GET("/:id/nodes", r.adminResourceGroupHandler.ListNodes)

		// Forward agent membership management
		adminResourceGroups.POST("/:id/forward-agents", r.adminResourceGroupHandler.AddForwardAgents)
		adminResourceGroups.DELETE("/:id/forward-agents", r.adminResourceGroupHandler.RemoveForwardAgents)
		adminResourceGroups.GET("/:id/forward-agents", r.adminResourceGroupHandler.ListForwardAgents)

		// Forward rule membership management
		adminResourceGroups.POST("/:id/forward-rules", r.adminResourceGroupHandler.AddForwardRules)
		adminResourceGroups.DELETE("/:id/forward-rules", r.adminResourceGroupHandler.RemoveForwardRules)
		adminResourceGroups.GET("/:id/forward-rules", r.adminResourceGroupHandler.ListForwardRules)
	}

	// Admin traffic stats routes
	adminTrafficStats := r.engine.Group("/admin/traffic-stats")
	adminTrafficStats.Use(r.authMiddleware.RequireAuth(), authorization.RequireAdmin())
	{
		adminTrafficStats.GET("/overview", r.adminTrafficStatsHandler.GetOverview)
		adminTrafficStats.GET("/users", r.adminTrafficStatsHandler.GetUserStats)
		adminTrafficStats.GET("/subscriptions", r.adminTrafficStatsHandler.GetSubscriptionStats)
		adminTrafficStats.GET("/nodes", r.adminTrafficStatsHandler.GetNodeStats)
		adminTrafficStats.GET("/ranking/users", r.adminTrafficStatsHandler.GetUserRanking)
		adminTrafficStats.GET("/ranking/subscriptions", r.adminTrafficStatsHandler.GetSubscriptionRanking)
		adminTrafficStats.GET("/trend", r.adminTrafficStatsHandler.GetTrend)
	}

	// Admin telegram routes (only if handler is initialized)
	if r.adminTelegramHandler != nil {
		adminTelegram := r.engine.Group("/admin/telegram")
		adminTelegram.Use(r.authMiddleware.RequireAuth(), authorization.RequireAdmin())
		{
			adminTelegram.GET("/binding", r.adminTelegramHandler.GetBindingStatus)
			adminTelegram.DELETE("/binding", r.adminTelegramHandler.Unbind)
			adminTelegram.PATCH("/preferences", r.adminTelegramHandler.UpdatePreferences)
		}
	}
}

// setupSubscriptionRoutes configures user subscription routes
func (r *Router) setupSubscriptionRoutes() {
	subscriptions := r.engine.Group("/subscriptions")
	subscriptions.Use(r.authMiddleware.RequireAuth())
	{
		// Collection operations (no ownership check needed)
		subscriptions.POST("", r.subscriptionHandler.CreateSubscription)
		subscriptions.GET("", r.subscriptionHandler.ListUserSubscriptions)

		// Operations on specific subscription (ownership verified by middleware)
		subscriptionWithOwnership := subscriptions.Group("/:sid")
		subscriptionWithOwnership.Use(r.subscriptionOwnerMiddleware.RequireOwnership())
		{
			subscriptionWithOwnership.GET("", r.subscriptionHandler.GetSubscription)
			subscriptionWithOwnership.PATCH("/status", r.subscriptionHandler.UpdateStatus)
			subscriptionWithOwnership.PATCH("/plan", r.subscriptionHandler.ChangePlan)
			subscriptionWithOwnership.PUT("/link", r.subscriptionHandler.ResetLink)
			subscriptionWithOwnership.DELETE("", r.subscriptionHandler.DeleteSubscription)

			// Token sub-resource endpoints
			subscriptionWithOwnership.POST("/tokens/:token_id/refresh", r.subscriptionTokenHandler.RefreshToken)
			subscriptionWithOwnership.DELETE("/tokens/:token_id", r.subscriptionTokenHandler.RevokeToken)
			subscriptionWithOwnership.POST("/tokens", r.subscriptionTokenHandler.GenerateToken)
			subscriptionWithOwnership.GET("/tokens", r.subscriptionTokenHandler.ListTokens)

			// Traffic statistics endpoint
			subscriptionWithOwnership.GET("/traffic-stats", r.subscriptionHandler.GetTrafficStats)
		}
	}
}

// setupPaymentRoutes configures payment routes
func (r *Router) setupPaymentRoutes() {
	payments := r.engine.Group("/payments")
	{
		payments.POST("/callback", r.paymentHandler.HandleCallback)

		paymentsProtected := payments.Group("")
		paymentsProtected.Use(r.authMiddleware.RequireAuth())
		{
			paymentsProtected.POST("", r.paymentHandler.CreatePayment)
		}
	}
}

// setupPlanRoutes configures plan routes
func (r *Router) setupPlanRoutes() {
	plans := r.engine.Group("/plans")
	{
		// Public endpoints (no authentication required)
		plans.GET("/public", r.planHandler.GetPublicPlans)

		// Protected endpoints (read operations)
		plansProtected := plans.Group("")
		plansProtected.Use(r.authMiddleware.RequireAuth())
		{
			plansProtected.GET("", r.planHandler.ListPlans)
			plansProtected.GET("/:id", r.planHandler.GetPlan)
			plansProtected.GET("/:id/pricings", r.planHandler.GetPlanPricings)
		}

		// Admin-only endpoints (write operations)
		plansAdmin := plans.Group("")
		plansAdmin.Use(r.authMiddleware.RequireAuth())
		plansAdmin.Use(authorization.RequireAdmin())
		{
			plansAdmin.POST("", r.planHandler.CreatePlan)
			plansAdmin.PATCH("/:id/status", r.planHandler.UpdatePlanStatus)
			plansAdmin.PUT("/:id", r.planHandler.UpdatePlan)
			plansAdmin.DELETE("/:id", r.planHandler.DeletePlan)
		}
	}
}

// setupNodeRoutes configures node routes using routes package
func (r *Router) setupNodeRoutes() {
	routes.SetupNodeRoutes(r.engine, &routes.NodeRouteConfig{
		NodeHandler:         r.nodeHandler,
		NodeHubHandler:      r.nodeHubHandler,
		NodeVersionHandler:  r.nodeVersionHandler,
		NodeSSEHandler:      r.nodeSSEHandler,
		UserNodeHandler:     r.userNodeHandler,
		SubscriptionHandler: r.nodeSubscriptionHandler,
		AuthMiddleware:      r.authMiddleware,
		NodeTokenMW:         r.nodeTokenMiddleware,
		NodeOwnerMW:         r.nodeOwnerMiddleware,
		NodeQuotaMW:         r.nodeQuotaMiddleware,
		RateLimiter:         r.rateLimiter,
	})
}

// setupAgentAPIRoutes configures RESTful Agent API routes
func (r *Router) setupAgentAPIRoutes() {
	agentAPI := r.engine.Group("/agents")
	agentAPI.Use(r.nodeTokenMiddleware.RequireNodeTokenHeader())
	{
		agentAPI.GET("/:nodesid/config", r.agentHandler.GetConfig)
		agentAPI.GET("/:nodesid/subscriptions", r.agentHandler.GetSubscriptions)
		agentAPI.POST("/:nodesid/traffic", r.agentHandler.ReportTraffic)
		agentAPI.PUT("/:nodesid/status", r.agentHandler.UpdateStatus)
		agentAPI.PUT("/:nodesid/online-subscriptions", r.agentHandler.UpdateOnlineSubscriptions)
	}
}

// setupExternalRoutes configures routes from routes package
func (r *Router) setupExternalRoutes() {
	routes.SetupTicketRoutes(r.engine, &routes.TicketRouteConfig{
		TicketHandler:  r.ticketHandler,
		AuthMiddleware: r.authMiddleware,
	})

	routes.SetupNotificationRoutes(r.engine, &routes.NotificationRouteConfig{
		NotificationHandler: r.notificationHandler,
		AuthMiddleware:      r.authMiddleware,
	})

	// Setup Telegram routes (only if handler is initialized)
	if r.telegramHandler != nil {
		routes.SetupTelegramRoutes(r.engine, &routes.TelegramRouteConfig{
			Handler:        r.telegramHandler,
			AuthMiddleware: r.authMiddleware,
		})
	}

	routes.SetupForwardRoutes(r.engine, &routes.ForwardRouteConfig{
		ForwardRuleHandler:          r.forwardRuleHandler,
		ForwardAgentHandler:         r.forwardAgentHandler,
		ForwardAgentVersionHandler:  r.forwardAgentVersionHandler,
		ForwardAgentSSEHandler:      r.forwardAgentSSEHandler,
		ForwardAgentHubHandler:      r.agentHubHandler,
		ForwardAgentAPIHandler:      r.forwardAgentAPIHandler,
		UserForwardHandler:          r.userForwardRuleHandler,
		AuthMiddleware:              r.authMiddleware,
		ForwardAgentTokenMiddleware: r.forwardAgentTokenMiddleware,
		ForwardRuleOwnerMiddleware:  r.forwardRuleOwnerMiddleware,
		ForwardQuotaMiddleware:      r.forwardQuotaMiddleware,
	})

	// Subscription-scoped forward rules routes
	routes.SetupSubscriptionForwardRoutes(r.engine, &routes.SubscriptionForwardRouteConfig{
		SubscriptionForwardHandler:  r.subscriptionForwardRuleHandler,
		AuthMiddleware:              r.authMiddleware,
		SubscriptionOwnerMiddleware: r.subscriptionOwnerMiddleware,
		ForwardRuleOwnerMiddleware:  r.forwardRuleOwnerMiddleware,
		ForwardQuotaMiddleware:      r.forwardQuotaMiddleware,
	})

	routes.SetupAgentHubRoutes(r.engine, &routes.AgentHubRouteConfig{
		HubHandler:                  r.agentHubHandler,
		NodeHubHandler:              r.nodeHubHandler,
		ForwardAgentTokenMiddleware: r.forwardAgentTokenMiddleware,
		NodeTokenMiddleware:         r.nodeTokenMiddleware,
	})

	// Setup Setting routes (Admin only)
	if r.settingHandler != nil {
		routes.SetupSettingRoutes(r.engine, &routes.SettingRouteConfig{
			Handler:        r.settingHandler,
			AuthMiddleware: r.authMiddleware,
		})
	}
}

// GetEngine returns the Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// Run starts the HTTP server
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}

// Shutdown gracefully shuts down the router
func (r *Router) Shutdown() {
	// Stop admin notification scheduler first
	if r.adminNotificationScheduler != nil {
		r.adminNotificationScheduler.Stop()
	}

	// Stop usage aggregation scheduler
	if r.usageAggregationScheduler != nil {
		r.usageAggregationScheduler.Stop()
	}

	// Stop payment scheduler
	if r.paymentScheduler != nil {
		r.paymentScheduler.Stop()
	}

	// Stop USDT monitor scheduler
	if r.usdtServiceManager != nil {
		r.usdtServiceManager.StopScheduler()
	}

	// Stop telegram bot service manager if running
	if r.telegramBotManager != nil {
		r.telegramBotManager.Stop()
	}

	// Close all SSE connections first to allow HTTP server shutdown to proceed quickly
	if r.adminHub != nil {
		r.adminHub.Shutdown()
	}

	// Stop rule traffic flush scheduler goroutine
	if r.ruleTrafficFlushDone != nil {
		close(r.ruleTrafficFlushDone)
	}

	// Stop rule traffic buffer (flushes remaining data to Redis)
	if r.ruleTrafficBuffer != nil {
		r.ruleTrafficBuffer.Stop()
	}

	// Final flush forward traffic from Redis to MySQL
	if r.forwardTrafficCache != nil {
		ctx := context.Background()
		if err := r.forwardTrafficCache.FlushToDatabase(ctx); err != nil {
			r.logger.Errorw("failed to flush forward traffic to database on shutdown", "error", err)
		}
	}

	// Stop subscription traffic flush scheduler goroutine
	if r.subscriptionTrafficFlushDone != nil {
		close(r.subscriptionTrafficFlushDone)
	}

	// Stop subscription traffic buffer (flushes remaining data to Redis)
	if r.subscriptionTrafficBuffer != nil {
		r.subscriptionTrafficBuffer.Stop()
	}

	// Final flush subscription traffic from Redis to MySQL
	if r.subscriptionTrafficCache != nil {
		ctx := context.Background()
		if err := r.subscriptionTrafficCache.FlushToDatabase(ctx); err != nil {
			r.logger.Errorw("failed to flush subscription traffic to database on shutdown", "error", err)
		}
	}
}

// GetTelegramService returns the telegram service for scheduler use
func (r *Router) GetTelegramService() *telegramApp.ServiceDDD {
	return r.telegramService
}

// GetAdminNotificationService returns the admin notification service for scheduler use
func (r *Router) GetAdminNotificationService() *telegramAdminApp.ServiceDDD {
	return r.adminNotificationService
}

// GetTelegramBotManager returns the telegram bot service manager
func (r *Router) GetTelegramBotManager() *telegramInfra.BotServiceManager {
	return r.telegramBotManager
}

// GetSettingService returns the setting service
func (r *Router) GetSettingService() *settingApp.ServiceDDD {
	return r.settingService
}

// StartTelegramPolling starts the telegram bot service using BotServiceManager
func (r *Router) StartTelegramPolling(ctx context.Context) error {
	if r.telegramBotManager == nil {
		return nil
	}
	if err := r.telegramBotManager.Start(ctx); err != nil {
		return err
	}

	// Start admin notification scheduler after telegram bot manager
	if r.adminNotificationScheduler != nil {
		r.adminNotificationScheduler.Start(ctx)
	}

	return nil
}

// StartUsageAggregationScheduler starts the usage aggregation scheduler
func (r *Router) StartUsageAggregationScheduler(ctx context.Context) {
	if r.usageAggregationScheduler != nil {
		r.usageAggregationScheduler.Start(ctx)
	}
}

// StartPaymentScheduler starts the payment expiration scheduler
func (r *Router) StartPaymentScheduler(ctx context.Context) {
	if r.paymentScheduler != nil {
		r.paymentScheduler.Start(ctx)
	}
}

// StartUSDTMonitorScheduler starts the USDT payment monitor scheduler
func (r *Router) StartUSDTMonitorScheduler(ctx context.Context) {
	if r.usdtServiceManager != nil {
		r.usdtServiceManager.StartScheduler(ctx)
	}
}
