package http

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/interfaces/http/routes"
)

// brandingFilenameRegex validates branding upload filenames
// Format: 32 hex chars + extension (.png, .jpg, .ico, .svg)
var brandingFilenameRegex = regexp.MustCompile(`^[a-f0-9]{32}\.(png|jpg|ico|svg)$`)

// isValidBrandingFilename checks if a filename matches the expected format
func isValidBrandingFilename(filename string) bool {
	return brandingFilenameRegex.MatchString(filename)
}

// SetupRoutes configures all HTTP routes.
func (r *Router) SetupRoutes(cfg *config.Config) {
	r.engine.Use(middleware.Logger(r.logger))
	r.engine.Use(middleware.Recovery(r.logger))
	r.engine.Use(middleware.CORS(cfg.Server.AllowedOrigins))
	r.engine.Use(middleware.SecurityHeaders())
	r.engine.Use(middleware.CSRF())
	r.engine.Use(middleware.APIVersion())

	r.engine.GET("/health", r.userHandler.HealthCheck)
	r.engine.GET("/version", r.userHandler.Version)

	routes.SetupAuthRoutes(r.engine, &routes.AuthRouteConfig{
		AuthHandler:    r.authHandler,
		PasskeyHandler: r.passkeyHandler,
		AuthMiddleware: r.authMiddleware,
		RateLimiter:    r.rateLimiter,
	})

	routes.SetupUserRoutes(r.engine, &routes.UserRouteConfig{
		UserHandler:      r.userHandler,
		ProfileHandler:   r.profileHandler,
		DashboardHandler: r.dashboardHandler,
		PasskeyHandler:   r.passkeyHandler,
		AuthMiddleware:   r.authMiddleware,
	})

	routes.SetupAdminRoutes(r.engine, &routes.AdminRouteConfig{
		AdminDashboardHandler:     r.adminDashboardHandler,
		AdminSubscriptionHandler:  r.adminSubscriptionHandler,
		AdminResourceGroupHandler: r.adminResourceGroupHandler,
		AdminTrafficStatsHandler:  r.adminTrafficStatsHandler,
		AdminTelegramHandler:      r.adminTelegramHandler,
		AuthMiddleware:            r.authMiddleware,
	})

	routes.SetupSubscriptionRoutes(r.engine, &routes.SubscriptionRouteConfig{
		SubscriptionHandler:         r.subscriptionHandler,
		SubscriptionTokenHandler:    r.subscriptionTokenHandler,
		AuthMiddleware:              r.authMiddleware,
		SubscriptionOwnerMiddleware: r.subscriptionOwnerMiddleware,
	})

	routes.SetupPaymentRoutes(r.engine, &routes.PaymentRouteConfig{
		PaymentHandler: r.paymentHandler,
		AuthMiddleware: r.authMiddleware,
	})

	routes.SetupPlanRoutes(r.engine, &routes.PlanRouteConfig{
		PlanHandler:    r.planHandler,
		AuthMiddleware: r.authMiddleware,
	})

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

	routes.SetupAgentAPIRoutes(r.engine, &routes.AgentAPIRouteConfig{
		AgentHandler:        r.agentHandler,
		NodeTokenMiddleware: r.nodeTokenMiddleware,
	})

	routes.SetupTicketRoutes(r.engine, &routes.TicketRouteConfig{
		TicketHandler:  r.ticketHandler,
		AuthMiddleware: r.authMiddleware,
	})

	routes.SetupNotificationRoutes(r.engine, &routes.NotificationRouteConfig{
		NotificationHandler: r.notificationHandler,
		AuthMiddleware:      r.authMiddleware,
	})

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

	if r.settingHandler != nil {
		routes.SetupSettingRoutes(r.engine, &routes.SettingRouteConfig{
			Handler:        r.settingHandler,
			AuthMiddleware: r.authMiddleware,
		})
	}

	// Public routes: branding, legal, registration settings, password policy, branding uploads.
	// These are on root paths with rate limiting, no auth required.
	r.setupPublicRoutes()
}

// setupPublicRoutes configures public endpoints that don't require authentication.
func (r *Router) setupPublicRoutes() {
	if r.settingHandler == nil {
		return
	}

	r.engine.GET("/branding", r.rateLimiter.Limit(), r.settingHandler.GetPublicBranding)
	r.engine.GET("/legal", r.rateLimiter.Limit(), r.settingHandler.GetPublicLegal)
	r.engine.GET("/registration-settings", r.rateLimiter.Limit(), r.settingHandler.GetPublicRegistration)
	r.engine.GET("/password-policy", r.rateLimiter.Limit(), r.settingHandler.GetPublicPasswordPolicy)

	// Static file serving for branding uploads with security headers
	r.engine.GET("/uploads/branding/:filename", func(c *gin.Context) {
		filename := c.Param("filename")

		// Validate filename format (hex string + extension only)
		if !isValidBrandingFilename(filename) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// Set security headers to prevent MIME sniffing and XSS
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Cache-Control", "public, max-age=86400") // 24 hours cache

		c.File("./uploads/branding/" + filename)
	})
}
