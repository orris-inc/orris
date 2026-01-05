package user

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ListAgents handles GET /user/forward-agents
// Returns forward agents accessible to the user through their subscriptions.
func (h *Handler) ListAgents(c *gin.Context) {
	// Get user_id from context (set by auth middleware)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Warnw("user_id not found in context", "ip", c.ClientIP())
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		h.logger.Warnw("invalid user_id type in context", "user_id", userIDInterface, "ip", c.ClientIP())
		utils.ErrorResponse(c, http.StatusInternalServerError, "invalid user ID type")
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(constants.DefaultPageSize)))
	if pageSize < 1 || pageSize > constants.MaxPageSize {
		pageSize = constants.DefaultPageSize
	}

	query := usecases.ListUserForwardAgentsQuery{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
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

	utils.ListSuccessResponse(c, result.Agents, result.Total, page, pageSize)
}
