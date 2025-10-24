package routes

import (
	"github.com/gin-gonic/gin"

	tickethandlers "orris/internal/interfaces/http/handlers/ticket"
	"orris/internal/interfaces/http/middleware"
)

type TicketRouteConfig struct {
	TicketHandler        *tickethandlers.TicketHandler
	AuthMiddleware       *middleware.AuthMiddleware
	PermissionMiddleware *middleware.PermissionMiddleware
}

func SetupTicketRoutes(engine *gin.Engine, config *TicketRouteConfig) {
	tickets := engine.Group("/tickets")
	tickets.Use(config.AuthMiddleware.RequireAuth())
	{
		tickets.POST("",
			config.TicketHandler.CreateTicket)
		tickets.GET("",
			config.TicketHandler.ListTickets)
		tickets.GET("/:id",
			config.TicketHandler.GetTicket)
		tickets.DELETE("/:id",
			config.PermissionMiddleware.RequirePermission("ticket", "delete"),
			config.TicketHandler.DeleteTicket)

		tickets.POST("/:id/assign",
			config.PermissionMiddleware.RequirePermission("ticket", "assign"),
			config.TicketHandler.AssignTicket)
		tickets.POST("/:id/comments",
			config.TicketHandler.AddComment)
		tickets.POST("/:id/close",
			config.TicketHandler.CloseTicket)
	}
}
