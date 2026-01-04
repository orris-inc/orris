package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/interfaces/http/handlers"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/shared/authorization"
)

type NotificationRouteConfig struct {
	NotificationHandler *handlers.NotificationHandler
	AuthMiddleware      *middleware.AuthMiddleware
}

func SetupNotificationRoutes(engine *gin.Engine, config *NotificationRouteConfig) {
	public := engine.Group("/public")
	{
		public.GET("/announcements", config.NotificationHandler.ListPublicAnnouncements)
	}

	notifications := engine.Group("/notifications")
	notifications.Use(config.AuthMiddleware.RequireAuth())
	{
		// IMPORTANT: Register specific paths BEFORE parameterized paths to avoid route conflicts

		// Collection operations (no ID parameter)
		notifications.GET("", config.NotificationHandler.ListNotifications)

		// Specific named endpoints (must come BEFORE /:id to avoid conflicts)
		notifications.GET("/unread-count", config.NotificationHandler.GetUnreadCount)
		// Using PATCH for batch state changes as per RESTful best practices
		notifications.PATCH("/status", config.NotificationHandler.UpdateAllNotificationsStatus)

		// Specific action endpoints for individual notifications
		// Using PATCH for state changes as per RESTful best practices
		notifications.PATCH("/:id/status", config.NotificationHandler.UpdateNotificationStatus)

		// Generic parameterized route (must come LAST)
		notifications.DELETE("/:id", config.NotificationHandler.DeleteNotification)
	}

	announcements := engine.Group("/announcements")
	announcements.Use(config.AuthMiddleware.RequireAuth())
	{
		// IMPORTANT: Register specific paths BEFORE parameterized paths to avoid route conflicts

		// Collection operations (no ID parameter)
		announcements.GET("", config.NotificationHandler.ListAnnouncements)
		announcements.POST("",
			authorization.RequireAdmin(),
			config.NotificationHandler.CreateAnnouncement)

		// Specific action endpoints (must come BEFORE /:id to avoid conflicts)
		// Using PATCH for state changes as per RESTful best practices
		announcements.PATCH("/:id/status",
			authorization.RequireAdmin(),
			config.NotificationHandler.UpdateAnnouncementStatus)

		// Generic parameterized routes (must come LAST)
		announcements.GET("/:id", config.NotificationHandler.GetAnnouncement)
		announcements.PUT("/:id",
			authorization.RequireAdmin(),
			config.NotificationHandler.UpdateAnnouncement)
		announcements.DELETE("/:id",
			authorization.RequireAdmin(),
			config.NotificationHandler.DeleteAnnouncement)
	}

	templates := engine.Group("/notification-templates")
	templates.Use(config.AuthMiddleware.RequireAuth())
	templates.Use(authorization.RequireAdmin())
	{
		templates.GET("", config.NotificationHandler.ListTemplates)
		templates.POST("/render", config.NotificationHandler.RenderTemplate)
		templates.POST("", config.NotificationHandler.CreateTemplate)
	}
}
