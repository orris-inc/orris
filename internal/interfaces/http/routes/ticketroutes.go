package routes

import (
	"github.com/gin-gonic/gin"

	tickethandlers "orris/internal/interfaces/http/handlers/ticket"
	"orris/internal/interfaces/http/middleware"
	"orris/internal/shared/authorization"
)

type TicketRouteConfig struct {
	TicketHandler  *tickethandlers.TicketHandler
	AuthMiddleware *middleware.AuthMiddleware
}

func SetupTicketRoutes(engine *gin.Engine, config *TicketRouteConfig) {
	tickets := engine.Group("/tickets")
	tickets.Use(config.AuthMiddleware.RequireAuth())
	{
		// IMPORTANT: Register specific paths BEFORE parameterized paths to avoid route conflicts

		// Collection operations (no ID parameter)
		tickets.POST("",
			config.TicketHandler.CreateTicket)
		tickets.GET("",
			config.TicketHandler.ListTickets)

		// Specific action endpoints (must come BEFORE /:id to avoid conflicts)
		tickets.POST("/:id/assign",
			authorization.RequireAdmin(),
			config.TicketHandler.AssignTicket)
		tickets.POST("/:id/comments",
			config.TicketHandler.AddComment)
		// Using PATCH for state changes as per RESTful best practices
		tickets.PATCH("/:id/status",
			config.TicketHandler.UpdateTicketStatus)

		// Generic parameterized routes (must come LAST)
		tickets.GET("/:id",
			config.TicketHandler.GetTicket)
		tickets.DELETE("/:id",
			authorization.RequireAdmin(),
			config.TicketHandler.DeleteTicket)
	}
}
