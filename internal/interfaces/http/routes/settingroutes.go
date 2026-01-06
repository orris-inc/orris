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

		// Telegram configuration endpoints (specific paths first)
		settings.GET("/telegram/config", config.Handler.GetTelegramConfig)
		settings.PUT("/telegram/config", config.Handler.UpdateTelegramConfig)
		settings.POST("/telegram/test", config.Handler.TestTelegramConnection)

		// Category-based settings (parameterized routes last)
		settings.GET("/:category", config.Handler.GetCategorySettings)
		settings.PUT("/:category", config.Handler.UpdateCategorySettings)
	}
}
