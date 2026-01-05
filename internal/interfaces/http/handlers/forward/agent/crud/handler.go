// Package crud provides HTTP handlers for forward agent CRUD management.
package crud

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// Handler handles HTTP requests for forward agent management.
type Handler struct {
	createAgentUC           *usecases.CreateForwardAgentUseCase
	getAgentUC              *usecases.GetForwardAgentUseCase
	listAgentsUC            *usecases.ListForwardAgentsUseCase
	updateAgentUC           *usecases.UpdateForwardAgentUseCase
	deleteAgentUC           *usecases.DeleteForwardAgentUseCase
	enableAgentUC           *usecases.EnableForwardAgentUseCase
	disableAgentUC          *usecases.DisableForwardAgentUseCase
	regenerateTokenUC       *usecases.RegenerateForwardAgentTokenUseCase
	getAgentTokenUC         *usecases.GetForwardAgentTokenUseCase
	getAgentStatusUC        *usecases.GetAgentStatusUseCase
	getRuleOverallStatusUC  *usecases.GetRuleOverallStatusUseCase
	generateInstallScriptUC *usecases.GenerateInstallScriptUseCase
	serverURL               string
	logger                  logger.Interface
}

// NewHandler creates a new Handler.
func NewHandler(
	createAgentUC *usecases.CreateForwardAgentUseCase,
	getAgentUC *usecases.GetForwardAgentUseCase,
	listAgentsUC *usecases.ListForwardAgentsUseCase,
	updateAgentUC *usecases.UpdateForwardAgentUseCase,
	deleteAgentUC *usecases.DeleteForwardAgentUseCase,
	enableAgentUC *usecases.EnableForwardAgentUseCase,
	disableAgentUC *usecases.DisableForwardAgentUseCase,
	regenerateTokenUC *usecases.RegenerateForwardAgentTokenUseCase,
	getAgentTokenUC *usecases.GetForwardAgentTokenUseCase,
	getAgentStatusUC *usecases.GetAgentStatusUseCase,
	getRuleOverallStatusUC *usecases.GetRuleOverallStatusUseCase,
	generateInstallScriptUC *usecases.GenerateInstallScriptUseCase,
	serverURL string,
) *Handler {
	return &Handler{
		createAgentUC:           createAgentUC,
		getAgentUC:              getAgentUC,
		listAgentsUC:            listAgentsUC,
		updateAgentUC:           updateAgentUC,
		deleteAgentUC:           deleteAgentUC,
		enableAgentUC:           enableAgentUC,
		disableAgentUC:          disableAgentUC,
		regenerateTokenUC:       regenerateTokenUC,
		getAgentTokenUC:         getAgentTokenUC,
		getAgentStatusUC:        getAgentStatusUC,
		getRuleOverallStatusUC:  getRuleOverallStatusUC,
		generateInstallScriptUC: generateInstallScriptUC,
		serverURL:               serverURL,
		logger:                  logger.NewLogger(),
	}
}

// CreateForwardAgentRequest represents a request to create a forward agent.
// An agent can participate in multiple rules with different roles (entry/relay/exit) simultaneously.
type CreateForwardAgentRequest struct {
	Name             string `json:"name" binding:"required" example:"Production Agent"`
	PublicAddress    string `json:"public_address,omitempty" example:"203.0.113.1"`
	TunnelAddress    string `json:"tunnel_address,omitempty" example:"192.168.1.100"` // IP or hostname only (no port), configure if agent may serve as relay/exit in any rule
	Remark           string `json:"remark,omitempty" example:"Forward agent for production environment"`
	AllowedPortRange string `json:"allowed_port_range,omitempty" example:"80,443,8000-9000"`
	SortOrder        *int   `json:"sort_order,omitempty" example:"100"` // Custom sort order for UI display (lower values appear first)
}

// UpdateForwardAgentRequest represents a request to update a forward agent.
type UpdateForwardAgentRequest struct {
	Name             *string `json:"name,omitempty" example:"Updated Agent Name"`
	PublicAddress    *string `json:"public_address,omitempty" example:"203.0.113.2"`
	TunnelAddress    *string `json:"tunnel_address,omitempty" example:"192.168.1.100"` // IP or hostname only (no port), configure if agent may serve as relay/exit in any rule
	Remark           *string `json:"remark,omitempty" example:"Updated remark"`
	GroupSID         *string `json:"group_sid,omitempty" example:"rg_xK9mP2vL3nQ"` // Resource group SID to associate with (use empty string to remove)
	AllowedPortRange *string `json:"allowed_port_range,omitempty" example:"80,443,8000-9000"`
	SortOrder        *int    `json:"sort_order,omitempty" example:"100"` // Custom sort order for UI display (lower values appear first)
}

// UpdateAgentStatusRequest represents a request to update forward agent status.
type UpdateAgentStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=enabled disabled" example:"enabled"`
}

// CreateAgent handles POST /forward-agents
func (h *Handler) CreateAgent(c *gin.Context) {
	var req CreateForwardAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create forward agent", "error", err, "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.CreateForwardAgentCommand{
		Name:             req.Name,
		PublicAddress:    req.PublicAddress,
		TunnelAddress:    req.TunnelAddress,
		Remark:           req.Remark,
		AllowedPortRange: req.AllowedPortRange,
		SortOrder:        req.SortOrder, // nil if not provided, allowing explicit 0 value
	}

	result, err := h.createAgentUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Forward agent created successfully")
}

// GetAgent handles GET /forward-agents/:id
func (h *Handler) GetAgent(c *gin.Context) {
	shortID, err := parseAgentShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetForwardAgentQuery{ShortID: shortID}
	result, err := h.getAgentUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// ListAgents handles GET /forward-agents
func (h *Handler) ListAgents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(constants.DefaultPageSize)))
	if pageSize < 1 || pageSize > constants.MaxPageSize {
		pageSize = constants.DefaultPageSize
	}

	query := usecases.ListForwardAgentsQuery{
		Page:     page,
		PageSize: pageSize,
		Name:     c.Query("name"),
		Status:   c.Query("status"),
		OrderBy:  c.DefaultQuery("sort_by", "sort_order"),
		Order:    c.DefaultQuery("order", "asc"),
	}

	result, err := h.listAgentsUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Agents, result.Total, page, pageSize)
}

// UpdateAgent handles PUT /forward-agents/:id
func (h *Handler) UpdateAgent(c *gin.Context) {
	shortID, err := parseAgentShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateForwardAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update forward agent", "short_id", shortID, "error", err, "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.UpdateForwardAgentCommand{
		ShortID:          shortID,
		Name:             req.Name,
		PublicAddress:    req.PublicAddress,
		TunnelAddress:    req.TunnelAddress,
		Remark:           req.Remark,
		GroupSID:         req.GroupSID,
		AllowedPortRange: req.AllowedPortRange,
		SortOrder:        req.SortOrder,
	}

	if err := h.updateAgentUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agent updated successfully", nil)
}

// DeleteAgent handles DELETE /forward-agents/:id
func (h *Handler) DeleteAgent(c *gin.Context) {
	shortID, err := parseAgentShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DeleteForwardAgentCommand{ShortID: shortID}
	if err := h.deleteAgentUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// EnableAgent handles POST /forward-agents/:id/enable
func (h *Handler) EnableAgent(c *gin.Context) {
	shortID, err := parseAgentShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.EnableForwardAgentCommand{ShortID: shortID}
	if err := h.enableAgentUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agent enabled successfully", nil)
}

// DisableAgent handles POST /forward-agents/:id/disable
func (h *Handler) DisableAgent(c *gin.Context) {
	shortID, err := parseAgentShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DisableForwardAgentCommand{ShortID: shortID}
	if err := h.disableAgentUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agent disabled successfully", nil)
}

// UpdateStatus handles PATCH /forward-agents/:id/status
func (h *Handler) UpdateStatus(c *gin.Context) {
	var req UpdateAgentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update agent status", "error", err, "ip", c.ClientIP())
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
func (h *Handler) RegenerateToken(c *gin.Context) {
	shortID, err := parseAgentShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.RegenerateForwardAgentTokenCommand{ShortID: shortID}
	result, err := h.regenerateTokenUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Token regenerated successfully", result)
}

// GetToken handles GET /forward-agents/:id/token
func (h *Handler) GetToken(c *gin.Context) {
	shortID, err := parseAgentShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetForwardAgentTokenQuery{ShortID: shortID}
	result, err := h.getAgentTokenUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetAgentStatus handles GET /forward-agents/:id/status
func (h *Handler) GetAgentStatus(c *gin.Context) {
	shortID, err := parseAgentShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetAgentStatusQuery{ShortID: shortID}
	result, err := h.getAgentStatusUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetInstallScript handles GET /forward-agents/:id/install-script
// Query params:
//   - token (optional): API token. If not provided, uses agent's current stored token
//   - server_url (optional): Override the default server URL
func (h *Handler) GetInstallScript(c *gin.Context) {
	shortID, err := parseAgentShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Token is optional - if not provided, will use agent's stored token
	token := c.Query("token")

	// Use query param to override server URL if provided
	serverURL := c.Query("server_url")
	if serverURL == "" {
		serverURL = h.serverURL
	}

	query := usecases.GenerateInstallScriptQuery{
		ShortID:   shortID,
		ServerURL: serverURL,
		Token:     token,
	}

	result, err := h.generateInstallScriptUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Install command generated successfully", result)
}

// GetRuleOverallStatus handles GET /forward-rules/:id/status
func (h *Handler) GetRuleOverallStatus(c *gin.Context) {
	// Parse rule ID from path parameter
	ruleSID, err := parseRuleShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Debugw("getting rule overall status",
		"rule_id", ruleSID,
		"ip", c.ClientIP(),
	)

	// Execute use case
	input := &dto.GetRuleOverallStatusInput{
		RuleSID: ruleSID,
	}

	result, err := h.getRuleOverallStatusUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Errorw("failed to get rule overall status",
			"rule_id", ruleSID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("rule overall status retrieved successfully",
		"rule_id", ruleSID,
		"total_agents", result.TotalAgents,
		"healthy_agents", result.HealthyAgents,
		"ip", c.ClientIP(),
	)

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// parseAgentShortID validates a prefixed agent ID and returns the SID (e.g., "fa_xK9mP2vL3nQ").
// Note: Despite the name, this returns the full SID (with prefix) as stored in the database.
func parseAgentShortID(c *gin.Context) (string, error) {
	prefixedID := c.Param("id")
	if prefixedID == "" {
		return "", errors.NewValidationError("forward agent ID is required")
	}

	// Validate the prefix is correct, but return the full prefixed ID
	// because the database stores SIDs with prefix (e.g., "fa_xxx")
	if err := id.ValidatePrefix(prefixedID, id.PrefixForwardAgent); err != nil {
		return "", errors.NewValidationError("invalid forward agent ID format, expected fa_xxxxx")
	}

	return prefixedID, nil
}

// parseRuleShortID validates a prefixed rule ID and returns the SID (e.g., "fr_xK9mP2vL3nQ").
// Note: Despite the name, this returns the full SID (with prefix) as stored in the database.
func parseRuleShortID(c *gin.Context) (string, error) {
	prefixedID := c.Param("id")
	if prefixedID == "" {
		return "", errors.NewValidationError("forward rule ID is required")
	}

	// Validate the prefix is correct, but return the full prefixed ID
	// because the database stores SIDs with prefix (e.g., "fr_xxx")
	if err := id.ValidatePrefix(prefixedID, id.PrefixForwardRule); err != nil {
		return "", errors.NewValidationError("invalid forward rule ID format, expected fr_xxxxx")
	}

	return prefixedID, nil
}
