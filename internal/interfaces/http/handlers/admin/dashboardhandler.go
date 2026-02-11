package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/admin/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AdminDashboardHandler handles the admin dashboard endpoint.
type AdminDashboardHandler struct {
	dashboardUC *usecases.GetAdminDashboardUseCase
	logger      logger.Interface
}

// NewAdminDashboardHandler creates a new AdminDashboardHandler.
func NewAdminDashboardHandler(
	dashboardUC *usecases.GetAdminDashboardUseCase,
	log logger.Interface,
) *AdminDashboardHandler {
	return &AdminDashboardHandler{
		dashboardUC: dashboardUC,
		logger:      log,
	}
}

// GetDashboard handles GET /admin/dashboard
func (h *AdminDashboardHandler) GetDashboard(c *gin.Context) {
	resp, err := h.dashboardUC.Execute(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get admin dashboard", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Admin dashboard retrieved successfully", resp)
}
