package user

import (
	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ListAgents handles GET /user/forward-agents
// Returns forward agents accessible to the user through their subscriptions.
func (h *Handler) ListAgents(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	pagination := utils.ParsePagination(c)

	query := usecases.ListUserForwardAgentsQuery{
		UserID:   userID,
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
		Name:     c.Query("name"),
		Status:   c.Query("status"),
		OrderBy:  c.DefaultQuery("sort_by", "created_at"),
		Order:    c.DefaultQuery("order", "desc"),
	}

	result, err := h.listAgentsUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Agents, result.Total, pagination.Page, pagination.PageSize)
}
