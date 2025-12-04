// Package forward provides HTTP handlers for forward rule management.
package forward

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ForwardAgentHandler handles HTTP requests for forward agent management.
type ForwardAgentHandler struct {
	createAgentUC     *usecases.CreateForwardAgentUseCase
	getAgentUC        *usecases.GetForwardAgentUseCase
	listAgentsUC      *usecases.ListForwardAgentsUseCase
	updateAgentUC     *usecases.UpdateForwardAgentUseCase
	deleteAgentUC     *usecases.DeleteForwardAgentUseCase
	enableAgentUC     *usecases.EnableForwardAgentUseCase
	disableAgentUC    *usecases.DisableForwardAgentUseCase
	regenerateTokenUC *usecases.RegenerateForwardAgentTokenUseCase
	logger            logger.Interface
}

// NewForwardAgentHandler creates a new ForwardAgentHandler.
func NewForwardAgentHandler(
	createAgentUC *usecases.CreateForwardAgentUseCase,
	getAgentUC *usecases.GetForwardAgentUseCase,
	listAgentsUC *usecases.ListForwardAgentsUseCase,
	updateAgentUC *usecases.UpdateForwardAgentUseCase,
	deleteAgentUC *usecases.DeleteForwardAgentUseCase,
	enableAgentUC *usecases.EnableForwardAgentUseCase,
	disableAgentUC *usecases.DisableForwardAgentUseCase,
	regenerateTokenUC *usecases.RegenerateForwardAgentTokenUseCase,
) *ForwardAgentHandler {
	return &ForwardAgentHandler{
		createAgentUC:     createAgentUC,
		getAgentUC:        getAgentUC,
		listAgentsUC:      listAgentsUC,
		updateAgentUC:     updateAgentUC,
		deleteAgentUC:     deleteAgentUC,
		enableAgentUC:     enableAgentUC,
		disableAgentUC:    disableAgentUC,
		regenerateTokenUC: regenerateTokenUC,
		logger:            logger.NewLogger(),
	}
}

// CreateForwardAgentRequest represents a request to create a forward agent.
type CreateForwardAgentRequest struct {
	Name          string `json:"name" binding:"required" example:"Production Agent"`
	PublicAddress string `json:"public_address,omitempty" example:"203.0.113.1"`
	Remark        string `json:"remark,omitempty" example:"Forward agent for production environment"`
}

// UpdateForwardAgentRequest represents a request to update a forward agent.
type UpdateForwardAgentRequest struct {
	Name          *string `json:"name,omitempty" example:"Updated Agent Name"`
	PublicAddress *string `json:"public_address,omitempty" example:"203.0.113.2"`
	Remark        *string `json:"remark,omitempty" example:"Updated remark"`
}

// UpdateAgentStatusRequest represents a request to update forward agent status.
type UpdateAgentStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=enabled disabled" example:"enabled"`
}

// CreateAgent handles POST /forward-agents
func (h *ForwardAgentHandler) CreateAgent(c *gin.Context) {
	var req CreateForwardAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create forward agent", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.CreateForwardAgentCommand{
		Name:          req.Name,
		PublicAddress: req.PublicAddress,
		Remark:        req.Remark,
	}

	result, err := h.createAgentUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Forward agent created successfully")
}

// GetAgent handles GET /forward-agents/:id
func (h *ForwardAgentHandler) GetAgent(c *gin.Context) {
	agentID, err := parseAgentID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetForwardAgentQuery{ID: agentID}
	result, err := h.getAgentUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// ListAgents handles GET /forward-agents
func (h *ForwardAgentHandler) ListAgents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := usecases.ListForwardAgentsQuery{
		Page:     page,
		PageSize: pageSize,
		Name:     c.Query("name"),
		Status:   c.Query("status"),
		OrderBy:  c.Query("order_by"),
		Order:    c.Query("order"),
	}

	result, err := h.listAgentsUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Agents, result.Total, page, pageSize)
}

// UpdateAgent handles PUT /forward-agents/:id
func (h *ForwardAgentHandler) UpdateAgent(c *gin.Context) {
	agentID, err := parseAgentID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateForwardAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update forward agent", "agent_id", agentID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.UpdateForwardAgentCommand{
		ID:            agentID,
		Name:          req.Name,
		PublicAddress: req.PublicAddress,
		Remark:        req.Remark,
	}

	if err := h.updateAgentUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agent updated successfully", nil)
}

// DeleteAgent handles DELETE /forward-agents/:id
func (h *ForwardAgentHandler) DeleteAgent(c *gin.Context) {
	agentID, err := parseAgentID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DeleteForwardAgentCommand{ID: agentID}
	if err := h.deleteAgentUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// EnableAgent handles POST /forward-agents/:id/enable
func (h *ForwardAgentHandler) EnableAgent(c *gin.Context) {
	agentID, err := parseAgentID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.EnableForwardAgentCommand{ID: agentID}
	if err := h.enableAgentUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agent enabled successfully", nil)
}

// DisableAgent handles POST /forward-agents/:id/disable
func (h *ForwardAgentHandler) DisableAgent(c *gin.Context) {
	agentID, err := parseAgentID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DisableForwardAgentCommand{ID: agentID}
	if err := h.disableAgentUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agent disabled successfully", nil)
}

// UpdateStatus handles PATCH /forward-agents/:id/status
func (h *ForwardAgentHandler) UpdateStatus(c *gin.Context) {
	var req UpdateAgentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update agent status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	if req.Status == "enabled" {
		h.EnableAgent(c)
	} else {
		h.DisableAgent(c)
	}
}

// RegenerateToken handles POST /forward-agents/:id/regenerate-token
func (h *ForwardAgentHandler) RegenerateToken(c *gin.Context) {
	agentID, err := parseAgentID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.RegenerateForwardAgentTokenCommand{ID: agentID}
	result, err := h.regenerateTokenUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Token regenerated successfully", result)
}

func parseAgentID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid forward agent ID")
	}
	if id == 0 {
		return 0, errors.NewValidationError("Forward agent ID must be greater than 0")
	}
	return uint(id), nil
}
