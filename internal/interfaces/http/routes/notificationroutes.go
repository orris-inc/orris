package routes

import (
	"github.com/gin-gonic/gin"

	"orris/internal/interfaces/http/handlers"
	"orris/internal/interfaces/http/middleware"
)

type NotificationRouteConfig struct {
	NotificationHandler  *handlers.NotificationHandler
	AuthMiddleware       *middleware.AuthMiddleware
	PermissionMiddleware *middleware.PermissionMiddleware
}

func SetupNotificationRoutes(engine *gin.Engine, config *NotificationRouteConfig) {
	public := engine.Group("/public")
	{
		public.GET("/announcements", config.NotificationHandler.ListPublicAnnouncements)
	}

	notifications := engine.Group("/notifications")
	notifications.Use(config.AuthMiddleware.RequireAuth())
	{
		notifications.GET("", config.NotificationHandler.ListNotifications)
		notifications.GET("/unread-count", config.NotificationHandler.GetUnreadCount)
		notifications.PUT("/read-all", config.NotificationHandler.MarkAllAsRead)
		notifications.PUT("/:id/read", config.NotificationHandler.MarkAsRead)
		notifications.POST("/:id/archive", config.NotificationHandler.ArchiveNotification)
		notifications.DELETE("/:id", config.NotificationHandler.DeleteNotification)
	}

	announcements := engine.Group("/announcements")
	announcements.Use(config.AuthMiddleware.RequireAuth())
	{
		announcements.GET("", config.NotificationHandler.ListAnnouncements)
		announcements.GET("/:id", config.NotificationHandler.GetAnnouncement)

		announcements.POST("",
			config.PermissionMiddleware.RequirePermission("announcement", "create"),
			config.NotificationHandler.CreateAnnouncement)
		announcements.PUT("/:id",
			config.PermissionMiddleware.RequirePermission("announcement", "update"),
			config.NotificationHandler.UpdateAnnouncement)
		announcements.DELETE("/:id",
			config.PermissionMiddleware.RequirePermission("announcement", "delete"),
			config.NotificationHandler.DeleteAnnouncement)
		announcements.POST("/:id/publish",
			config.PermissionMiddleware.RequirePermission("announcement", "publish"),
			config.NotificationHandler.PublishAnnouncement)
	}

	templates := engine.Group("/notification-templates")
	templates.Use(config.AuthMiddleware.RequireAuth())
	{
		templates.GET("", config.NotificationHandler.ListTemplates)
		templates.POST("/render", config.NotificationHandler.RenderTemplate)
		templates.POST("",
			config.PermissionMiddleware.RequirePermission("notification_template", "create"),
			config.NotificationHandler.CreateTemplate)
	}
}
