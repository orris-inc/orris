package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/interfaces/http/handlers"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/shared/authorization"
)

// PlanRouteConfig holds dependencies for plan routes.
type PlanRouteConfig struct {
	PlanHandler    *handlers.PlanHandler
	AuthMiddleware *middleware.AuthMiddleware
}

// SetupPlanRoutes configures plan routes.
func SetupPlanRoutes(engine *gin.Engine, cfg *PlanRouteConfig) {
	plans := engine.Group("/plans")
	{
		// Public endpoints (no authentication required)
		plans.GET("/public", cfg.PlanHandler.GetPublicPlans)

		// Protected endpoints (read operations)
		plansProtected := plans.Group("")
		plansProtected.Use(cfg.AuthMiddleware.RequireAuth())
		{
			plansProtected.GET("", cfg.PlanHandler.ListPlans)
			plansProtected.GET("/:id", cfg.PlanHandler.GetPlan)
			plansProtected.GET("/:id/pricings", cfg.PlanHandler.GetPlanPricings)
		}

		// Admin-only endpoints (write operations)
		plansAdmin := plans.Group("")
		plansAdmin.Use(cfg.AuthMiddleware.RequireAuth())
		plansAdmin.Use(authorization.RequireAdmin())
		{
			plansAdmin.POST("", cfg.PlanHandler.CreatePlan)
			plansAdmin.PATCH("/:id/status", cfg.PlanHandler.UpdatePlanStatus)
			plansAdmin.PUT("/:id", cfg.PlanHandler.UpdatePlan)
			plansAdmin.DELETE("/:id", cfg.PlanHandler.DeletePlan)
		}
	}
}
