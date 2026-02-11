// Package crud provides HTTP handlers for forward agent CRUD management.
package crud

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
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
	log logger.Interface,
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
		logger:                  log,
	}
}

// CreateForwardAgentRequest represents a request to create a forward agent.
// An agent can participate in multiple rules with different roles (entry/relay/exit) simultaneously.
type CreateForwardAgentRequest struct {
	Name             string   `json:"name" binding:"required" example:"Production Agent"`
	PublicAddress    string   `json:"public_address,omitempty" example:"203.0.113.1"`
	TunnelAddress    string   `json:"tunnel_address,omitempty" example:"192.168.1.100"` // IP or hostname only (no port), configure if agent may serve as relay/exit in any rule
	Remark           string   `json:"remark,omitempty" example:"Forward agent for production environment"`
	GroupSIDs        []string `json:"group_sids,omitempty" example:"[\"rg_xK9mP2vL3nQ\"]"` // Resource group SIDs to associate with
	AllowedPortRange string   `json:"allowed_port_range,omitempty" example:"80,443,8000-9000"`
	BlockedProtocols []string `json:"blocked_protocols,omitempty" example:"socks5,http_connect"` // Protocols to block (e.g., socks5, http_connect, ssh)
	SortOrder        *int     `json:"sort_order,omitempty" example:"100"`                        // Custom sort order for UI display (lower values appear first)
}

// UpdateForwardAgentRequest represents a request to update a forward agent.
type UpdateForwardAgentRequest struct {
	Name             *string   `json:"name,omitempty" example:"Updated Agent Name"`
	PublicAddress    *string   `json:"public_address,omitempty" example:"203.0.113.2"`
	TunnelAddress    *string   `json:"tunnel_address,omitempty" example:"192.168.1.100"` // IP or hostname only (no port), configure if agent may serve as relay/exit in any rule
	Remark           *string   `json:"remark,omitempty" example:"Updated remark"`
	GroupSIDs        []string  `json:"group_sids,omitempty" example:"[\"rg_xK9mP2vL3nQ\"]"` // Resource group SIDs to associate with (empty array to remove all)
	AllowedPortRange *string   `json:"allowed_port_range,omitempty" example:"80,443,8000-9000"`
	BlockedProtocols *[]string `json:"blocked_protocols,omitempty"`                         // Protocols to block (nil: no update, empty array: clear, non-empty: set new)
	SortOrder        *int      `json:"sort_order,omitempty" example:"100"`                  // Custom sort order for UI display (lower values appear first)
	MuteNotification *bool     `json:"mute_notification,omitempty"`                         // Mute online/offline notifications for this agent
	ExpiresAt        *string   `json:"expires_at,omitempty" example:"2025-12-31T23:59:59Z"` // Expiration time in ISO8601 format (empty to clear, omit to keep unchanged)
	CostLabel        *string   `json:"cost_label,omitempty" example:"35$/m"`                // Cost label for display (empty to clear, omit to keep unchanged)
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
		GroupSIDs:        req.GroupSIDs,
		AllowedPortRange: req.AllowedPortRange,
		BlockedProtocols: req.BlockedProtocols,
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
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardAgent, "forward agent")
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
	pagination := utils.ParsePagination(c)

	query := usecases.ListForwardAgentsQuery{
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
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

	utils.ListSuccessResponse(c, result.Agents, result.Total, pagination.Page, pagination.PageSize)
}

// UpdateAgent handles PUT /forward-agents/:id
func (h *Handler) UpdateAgent(c *gin.Context) {
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardAgent, "forward agent")
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
		GroupSIDs:        req.GroupSIDs,
		AllowedPortRange: req.AllowedPortRange,
		BlockedProtocols: req.BlockedProtocols,
		SortOrder:        req.SortOrder,
		MuteNotification: req.MuteNotification,
	}

	// Handle ExpiresAt field
	// If expires_at is provided and is null string, clear it
	// If expires_at is provided and non-null, parse and set it
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			// Empty string means clear
			cmd.ClearExpiresAt = true
		} else {
			// Parse ISO8601 time string
			parsedTime, err := time.Parse(time.RFC3339, *req.ExpiresAt)
			if err != nil {
				h.logger.Warnw("invalid expires_at format", "short_id", shortID, "expires_at", *req.ExpiresAt, "error", err)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid expires_at format, expected ISO8601 (RFC3339)"))
				return
			}
			// Validate expires_at is in the future
			if parsedTime.Before(time.Now().UTC()) {
				h.logger.Warnw("expires_at must be in the future", "short_id", shortID, "expires_at", *req.ExpiresAt)
				utils.ErrorResponseWithError(c, errors.NewValidationError("expires_at must be a future time"))
				return
			}
			cmd.ExpiresAt = &parsedTime
		}
	}

	// Handle CostLabel field
	if req.CostLabel != nil {
		if *req.CostLabel == "" {
			// Empty string means clear the cost label
			cmd.ClearCostLabel = true
		} else {
			if len(*req.CostLabel) > 50 {
				h.logger.Warnw("cost_label exceeds max length", "short_id", shortID, "length", len(*req.CostLabel))
				utils.ErrorResponseWithError(c, errors.NewValidationError("cost_label cannot exceed 50 characters"))
				return
			}
			cmd.CostLabel = req.CostLabel
		}
	}

	if err := h.updateAgentUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agent updated successfully", nil)
}

// DeleteAgent handles DELETE /forward-agents/:id
func (h *Handler) DeleteAgent(c *gin.Context) {
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardAgent, "forward agent")
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
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardAgent, "forward agent")
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
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardAgent, "forward agent")
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
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardAgent, "forward agent")
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
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardAgent, "forward agent")
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
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardAgent, "forward agent")
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
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardAgent, "forward agent")
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
	ruleSID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardRule, "forward rule")
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

	h.logger.Debugw("rule overall status retrieved successfully",
		"rule_id", ruleSID,
		"total_agents", result.TotalAgents,
		"healthy_agents", result.HealthyAgents,
		"ip", c.ClientIP(),
	)

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

