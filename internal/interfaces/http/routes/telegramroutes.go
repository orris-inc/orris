package routes

import (
	"github.com/gin-gonic/gin"

	telegramHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/telegram"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
)

// TelegramRouteConfig holds the configuration for telegram routes
type TelegramRouteConfig struct {
	Handler        *telegramHandlers.Handler
	AuthMiddleware *middleware.AuthMiddleware
}

// SetupTelegramRoutes configures telegram-related routes
func SetupTelegramRoutes(engine *gin.Engine, config *TelegramRouteConfig) {
	// Public webhook endpoint (Telegram calls this)
	webhooks := engine.Group("/webhooks")
	{
		webhooks.POST("/telegram", config.Handler.HandleWebhook)
	}

	// Protected user endpoints
	telegram := engine.Group("/telegram")
	telegram.Use(config.AuthMiddleware.RequireAuth())
	{
		// GET /telegram/binding - Get current binding status and verify code
		telegram.GET("/binding", config.Handler.GetBindingStatus)

		// DELETE /telegram/binding - Unbind telegram
		telegram.DELETE("/binding", config.Handler.Unbind)

		// PATCH /telegram/preferences - Update notification preferences
		telegram.PATCH("/preferences", config.Handler.UpdatePreferences)
	}
}
