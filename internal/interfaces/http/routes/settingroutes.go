package routes

import (
	"github.com/gin-gonic/gin"

	adminHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/shared/authorization"
)

// SettingRouteConfig holds the configuration for setting routes
type SettingRouteConfig struct {
	Handler        *adminHandlers.SettingHandler
	AuthMiddleware *middleware.AuthMiddleware
}

// SetupSettingRoutes configures system setting admin routes
func SetupSettingRoutes(engine *gin.Engine, config *SettingRouteConfig) {
	// Admin settings endpoints - all require admin access
	settings := engine.Group("/admin/settings")
	settings.Use(config.AuthMiddleware.RequireAuth())
	settings.Use(authorization.RequireAdmin())
	{
		// IMPORTANT: Register specific paths BEFORE parameterized paths to avoid route conflicts

		// Branding settings
		settings.GET("/branding", config.Handler.GetBrandingSettings)
		settings.PUT("/branding", config.Handler.UpdateBrandingSettings)
		settings.POST("/branding/upload", config.Handler.UploadBrandingImage)

		// Security settings
		settings.GET("/security", config.Handler.GetSecuritySettings)
		settings.PUT("/security", config.Handler.UpdateSecuritySettings)

		// Registration settings
		settings.GET("/registration", config.Handler.GetRegistrationSettings)
		settings.PUT("/registration", config.Handler.UpdateRegistrationSettings)

		// Legal settings
		settings.GET("/legal", config.Handler.GetLegalSettings)
		settings.PUT("/legal", config.Handler.UpdateLegalSettings)

		// System settings
		settings.GET("/system", config.Handler.GetSystemSettings)
		settings.PUT("/system", config.Handler.UpdateSystemSettings)

		// OAuth settings
		settings.GET("/oauth", config.Handler.GetOAuthSettings)
		settings.PUT("/oauth", config.Handler.UpdateOAuthSettings)

		// Email settings
		settings.GET("/email", config.Handler.GetEmailSettings)
		settings.PUT("/email", config.Handler.UpdateEmailSettings)
		settings.POST("/email/test", config.Handler.TestEmailConnection)

		// Setup status
		settings.GET("/setup-status", config.Handler.GetSetupStatus)

		// Telegram configuration endpoints
		settings.GET("/telegram/config", config.Handler.GetTelegramConfig)
		settings.PUT("/telegram/config", config.Handler.UpdateTelegramConfig)
		settings.POST("/telegram/test", config.Handler.TestTelegramConnection)

		// USDT payment settings
		settings.GET("/usdt", config.Handler.GetUSDTSettings)
		settings.PUT("/usdt", config.Handler.UpdateUSDTSettings)

		// Subscription settings
		settings.GET("/subscription", config.Handler.GetSubscriptionSettings)
		settings.PUT("/subscription", config.Handler.UpdateSubscriptionSettings)

		// Category-based settings (parameterized routes last)
		settings.GET("/:category", config.Handler.GetCategorySettings)
		settings.PUT("/:category", config.Handler.UpdateCategorySettings)
	}
}
