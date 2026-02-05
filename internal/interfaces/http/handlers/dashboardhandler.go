package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/user/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// DashboardHandler handles user dashboard HTTP requests
type DashboardHandler struct {
	getDashboardUseCase *usecases.GetDashboardUseCase
	logger              logger.Interface
}

// NewDashboardHandler creates a new DashboardHandler
func NewDashboardHandler(
	getDashboardUseCase *usecases.GetDashboardUseCase,
	logger logger.Interface,
) *DashboardHandler {
	return &DashboardHandler{
		getDashboardUseCase: getDashboardUseCase,
		logger:              logger,
	}
}

// GetDashboard handles GET /users/me/dashboard
func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	// Get current user ID from context
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetDashboardQuery{
		UserID: userID,
	}

	result, err := h.getDashboardUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get dashboard", "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}
