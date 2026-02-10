package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/interfaces/http/handlers"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
)

// PaymentRouteConfig holds dependencies for payment routes.
type PaymentRouteConfig struct {
	PaymentHandler *handlers.PaymentHandler
	AuthMiddleware *middleware.AuthMiddleware
}

// SetupPaymentRoutes configures payment routes.
func SetupPaymentRoutes(engine *gin.Engine, cfg *PaymentRouteConfig) {
	payments := engine.Group("/payments")
	{
		payments.POST("/callback", cfg.PaymentHandler.HandleCallback)

		paymentsProtected := payments.Group("")
		paymentsProtected.Use(cfg.AuthMiddleware.RequireAuth())
		{
			paymentsProtected.POST("", cfg.PaymentHandler.CreatePayment)
		}
	}
}
