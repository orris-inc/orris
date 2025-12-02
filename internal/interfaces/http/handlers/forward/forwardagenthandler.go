// Package forward provides HTTP handlers for forward rule management.
package forward

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"orris/internal/application/forward/usecases"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
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
	Name   string `json:"name" binding:"required" example:"Production Agent"`
	Remark string `json:"remark,omitempty" example:"Forward agent for production environment"`
}

// UpdateForwardAgentRequest represents a request to update a forward agent.
type UpdateForwardAgentRequest struct {
	Name   *string `json:"name,omitempty" example:"Updated Agent Name"`
	Remark *string `json:"remark,omitempty" example:"Updated remark"`
}

// UpdateAgentStatusRequest represents a request to update forward agent status.
type UpdateAgentStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=enabled disabled" example:"enabled"`
}

// CreateAgent handles POST /forward-agents
//
//	@Summary		Create a new forward agent
//	@Description	Create a new forward agent with authentication token
//	@Tags			forward-agents
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			agent	body		CreateForwardAgentRequest	true	"Forward agent data"
//	@Success		201		{object}	utils.APIResponse			"Forward agent created successfully"
//	@Failure		400		{object}	utils.APIResponse			"Bad request"
//	@Failure		401		{object}	utils.APIResponse			"Unauthorized"
//	@Failure		403		{object}	utils.APIResponse			"Forbidden - Requires admin role"
//	@Failure		500		{object}	utils.APIResponse			"Internal server error"
//	@Router			/forward-agents [post]
func (h *ForwardAgentHandler) CreateAgent(c *gin.Context) {
	var req CreateForwardAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create forward agent", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.CreateForwardAgentCommand{
		Name:   req.Name,
		Remark: req.Remark,
	}

	result, err := h.createAgentUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Forward agent created successfully")
}

// GetAgent handles GET /forward-agents/:id
//
//	@Summary		Get forward agent by ID
//	@Description	Get details of a forward agent by its ID
//	@Tags			forward-agents
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int					true	"Forward agent ID"
//	@Success		200	{object}	utils.APIResponse	"Forward agent details"
//	@Failure		400	{object}	utils.APIResponse	"Invalid agent ID"
//	@Failure		401	{object}	utils.APIResponse	"Unauthorized"
//	@Failure		403	{object}	utils.APIResponse	"Forbidden - Requires admin role"
//	@Failure		404	{object}	utils.APIResponse	"Forward agent not found"
//	@Failure		500	{object}	utils.APIResponse	"Internal server error"
//	@Router			/forward-agents/{id} [get]
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
//
//	@Summary		List forward agents
//	@Description	Get a paginated list of forward agents
//	@Tags			forward-agents
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			page		query		int					false	"Page number"			default(1)
//	@Param			page_size	query		int					false	"Page size"				default(20)
//	@Param			name		query		string				false	"Name filter"
//	@Param			status		query		string				false	"Status filter"			Enums(enabled,disabled)
//	@Param			order_by	query		string				false	"Sort field"			default(created_at)
//	@Param			order		query		string				false	"Sort direction"		Enums(asc,desc)	default(desc)
//	@Success		200			{object}	utils.APIResponse	"Forward agents list"
//	@Failure		400			{object}	utils.APIResponse	"Invalid query parameters"
//	@Failure		401			{object}	utils.APIResponse	"Unauthorized"
//	@Failure		403			{object}	utils.APIResponse	"Forbidden - Requires admin role"
//	@Failure		500			{object}	utils.APIResponse	"Internal server error"
//	@Router			/forward-agents [get]
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
//
//	@Summary		Update forward agent
//	@Description	Update forward agent information by ID
//	@Tags			forward-agents
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id		path		int							true	"Forward agent ID"
//	@Param			agent	body		UpdateForwardAgentRequest	true	"Forward agent update data"
//	@Success		200		{object}	utils.APIResponse			"Forward agent updated successfully"
//	@Failure		400		{object}	utils.APIResponse			"Bad request"
//	@Failure		401		{object}	utils.APIResponse			"Unauthorized"
//	@Failure		403		{object}	utils.APIResponse			"Forbidden - Requires admin role"
//	@Failure		404		{object}	utils.APIResponse			"Forward agent not found"
//	@Failure		500		{object}	utils.APIResponse			"Internal server error"
//	@Router			/forward-agents/{id} [put]
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
		ID:     agentID,
		Name:   req.Name,
		Remark: req.Remark,
	}

	if err := h.updateAgentUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agent updated successfully", nil)
}

// DeleteAgent handles DELETE /forward-agents/:id
//
//	@Summary		Delete forward agent
//	@Description	Delete a forward agent by ID
//	@Tags			forward-agents
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path	int	true	"Forward agent ID"
//	@Success		204	"Forward agent deleted successfully"
//	@Failure		400	{object}	utils.APIResponse	"Invalid agent ID"
//	@Failure		401	{object}	utils.APIResponse	"Unauthorized"
//	@Failure		403	{object}	utils.APIResponse	"Forbidden - Requires admin role"
//	@Failure		404	{object}	utils.APIResponse	"Forward agent not found"
//	@Failure		500	{object}	utils.APIResponse	"Internal server error"
//	@Router			/forward-agents/{id} [delete]
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
//
//	@Summary		Enable forward agent
//	@Description	Enable a forward agent to allow client access
//	@Tags			forward-agents
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int					true	"Forward agent ID"
//	@Success		200	{object}	utils.APIResponse	"Forward agent enabled successfully"
//	@Failure		400	{object}	utils.APIResponse	"Invalid agent ID"
//	@Failure		401	{object}	utils.APIResponse	"Unauthorized"
//	@Failure		403	{object}	utils.APIResponse	"Forbidden - Requires admin role"
//	@Failure		404	{object}	utils.APIResponse	"Forward agent not found"
//	@Failure		500	{object}	utils.APIResponse	"Internal server error"
//	@Router			/forward-agents/{id}/enable [post]
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
//
//	@Summary		Disable forward agent
//	@Description	Disable a forward agent to prevent client access
//	@Tags			forward-agents
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int					true	"Forward agent ID"
//	@Success		200	{object}	utils.APIResponse	"Forward agent disabled successfully"
//	@Failure		400	{object}	utils.APIResponse	"Invalid agent ID"
//	@Failure		401	{object}	utils.APIResponse	"Unauthorized"
//	@Failure		403	{object}	utils.APIResponse	"Forbidden - Requires admin role"
//	@Failure		404	{object}	utils.APIResponse	"Forward agent not found"
//	@Failure		500	{object}	utils.APIResponse	"Internal server error"
//	@Router			/forward-agents/{id}/disable [post]
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
//
//	@Summary		Update forward agent status
//	@Description	Update forward agent status (enable or disable)
//	@Tags			forward-agents
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id		path		int						true	"Forward agent ID"
//	@Param			status	body		UpdateAgentStatusRequest	true	"Status update data"
//	@Success		200		{object}	utils.APIResponse		"Status updated successfully"
//	@Failure		400		{object}	utils.APIResponse		"Bad request"
//	@Failure		401		{object}	utils.APIResponse		"Unauthorized"
//	@Failure		403		{object}	utils.APIResponse		"Forbidden - Requires admin role"
//	@Failure		404		{object}	utils.APIResponse		"Forward agent not found"
//	@Failure		500		{object}	utils.APIResponse		"Internal server error"
//	@Router			/forward-agents/{id}/status [patch]
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
//
//	@Summary		Regenerate forward agent token
//	@Description	Regenerate authentication token for a forward agent
//	@Tags			forward-agents
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int					true	"Forward agent ID"
//	@Success		200	{object}	utils.APIResponse	"Token regenerated successfully"
//	@Failure		400	{object}	utils.APIResponse	"Invalid agent ID"
//	@Failure		401	{object}	utils.APIResponse	"Unauthorized"
//	@Failure		403	{object}	utils.APIResponse	"Forbidden - Requires admin role"
//	@Failure		404	{object}	utils.APIResponse	"Forward agent not found"
//	@Failure		500	{object}	utils.APIResponse	"Internal server error"
//	@Router			/forward-agents/{id}/regenerate-token [post]
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
