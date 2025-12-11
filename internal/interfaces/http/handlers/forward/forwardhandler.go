// Package forward provides HTTP handlers for forward rule management.
package forward

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/services"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ForwardHandler handles HTTP requests for forward rules.
type ForwardHandler struct {
	createRuleUC   *usecases.CreateForwardRuleUseCase
	getRuleUC      *usecases.GetForwardRuleUseCase
	updateRuleUC   *usecases.UpdateForwardRuleUseCase
	deleteRuleUC   *usecases.DeleteForwardRuleUseCase
	listRulesUC    *usecases.ListForwardRulesUseCase
	enableRuleUC   *usecases.EnableForwardRuleUseCase
	disableRuleUC  *usecases.DisableForwardRuleUseCase
	resetTrafficUC *usecases.ResetForwardRuleTrafficUseCase
	probeService   *services.ProbeService
	logger         logger.Interface
}

// NewForwardHandler creates a new ForwardHandler.
func NewForwardHandler(
	createRuleUC *usecases.CreateForwardRuleUseCase,
	getRuleUC *usecases.GetForwardRuleUseCase,
	updateRuleUC *usecases.UpdateForwardRuleUseCase,
	deleteRuleUC *usecases.DeleteForwardRuleUseCase,
	listRulesUC *usecases.ListForwardRulesUseCase,
	enableRuleUC *usecases.EnableForwardRuleUseCase,
	disableRuleUC *usecases.DisableForwardRuleUseCase,
	resetTrafficUC *usecases.ResetForwardRuleTrafficUseCase,
	probeService *services.ProbeService,
) *ForwardHandler {
	return &ForwardHandler{
		createRuleUC:   createRuleUC,
		getRuleUC:      getRuleUC,
		updateRuleUC:   updateRuleUC,
		deleteRuleUC:   deleteRuleUC,
		listRulesUC:    listRulesUC,
		enableRuleUC:   enableRuleUC,
		disableRuleUC:  disableRuleUC,
		resetTrafficUC: resetTrafficUC,
		probeService:   probeService,
		logger:         logger.NewLogger(),
	}
}

// CreateForwardRuleRequest represents a request to create a forward rule.
// Required fields by rule type:
// - direct: agent_id, listen_port, (target_address+target_port OR target_node_id)
// - entry: agent_id, exit_agent_id, listen_port, (target_address+target_port OR target_node_id)
// - chain: agent_id, chain_agent_ids, listen_port, (target_address+target_port OR target_node_id)
type CreateForwardRuleRequest struct {
	AgentID       string   `json:"agent_id" binding:"required" example:"fa_xK9mP2vL3nQ"`
	RuleType      string   `json:"rule_type" binding:"required,oneof=direct entry chain" example:"direct"`
	ExitAgentID   string   `json:"exit_agent_id,omitempty" example:"fa_yL8nQ3wM4oR"`
	ChainAgentIDs []string `json:"chain_agent_ids,omitempty" example:"[\"fa_aaa\",\"fa_bbb\"]"`
	Name          string   `json:"name" binding:"required" example:"MySQL-Forward"`
	ListenPort    uint16   `json:"listen_port,omitempty" example:"13306"`
	TargetAddress string   `json:"target_address,omitempty" example:"192.168.1.100"`
	TargetPort    uint16   `json:"target_port,omitempty" example:"3306"`
	TargetNodeID  string   `json:"target_node_id,omitempty" example:"node_xK9mP2vL3nQ"`
	Protocol      string   `json:"protocol" binding:"required,oneof=tcp udp both" example:"tcp"`
	Remark        string   `json:"remark,omitempty" example:"Forward to internal MySQL server"`
}

// UpdateForwardRuleRequest represents a request to update a forward rule.
type UpdateForwardRuleRequest struct {
	Name          *string  `json:"name,omitempty" example:"MySQL-Forward-Updated"`
	AgentID       *string  `json:"agent_id,omitempty" example:"fa_xK9mP2vL3nQ"`
	ExitAgentID   *string  `json:"exit_agent_id,omitempty" example:"fa_yL8nQ3wM4oR"`
	ChainAgentIDs []string `json:"chain_agent_ids,omitempty" example:"[\"fa_aaa\",\"fa_bbb\"]"`
	ListenPort    *uint16  `json:"listen_port,omitempty" example:"13307"`
	TargetAddress *string  `json:"target_address,omitempty" example:"192.168.1.101"`
	TargetPort    *uint16  `json:"target_port,omitempty" example:"3307"`
	TargetNodeID  *string  `json:"target_node_id,omitempty" example:"node_xK9mP2vL3nQ"`
	Protocol      *string  `json:"protocol,omitempty" binding:"omitempty,oneof=tcp udp both" example:"tcp"`
	Remark        *string  `json:"remark,omitempty" example:"Updated remark"`
}

// CreateRule handles POST /forward-rules
func (h *ForwardHandler) CreateRule(c *gin.Context) {
	var req CreateForwardRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create forward rule", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Parse Stripe-style IDs to extract short IDs
	agentShortID, err := id.ParseForwardAgentID(req.AgentID)
	if err != nil {
		h.logger.Warnw("invalid agent_id format", "agent_id", req.AgentID, "error", err)
		utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id format, expected fa_xxxxx"))
		return
	}

	var exitAgentShortID string
	if req.ExitAgentID != "" {
		exitAgentShortID, err = id.ParseForwardAgentID(req.ExitAgentID)
		if err != nil {
			h.logger.Warnw("invalid exit_agent_id format", "exit_agent_id", req.ExitAgentID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid exit_agent_id format, expected fa_xxxxx"))
			return
		}
	}

	// Parse chain agent IDs
	var chainAgentShortIDs []string
	if len(req.ChainAgentIDs) > 0 {
		chainAgentShortIDs = make([]string, len(req.ChainAgentIDs))
		for i, chainAgentID := range req.ChainAgentIDs {
			shortID, parseErr := id.ParseForwardAgentID(chainAgentID)
			if parseErr != nil {
				h.logger.Warnw("invalid chain_agent_id format", "chain_agent_id", chainAgentID, "error", parseErr)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid chain_agent_id format, expected fa_xxxxx"))
				return
			}
			chainAgentShortIDs[i] = shortID
		}
	}

	var targetNodeShortID string
	if req.TargetNodeID != "" {
		targetNodeShortID, err = id.ParseNodeID(req.TargetNodeID)
		if err != nil {
			h.logger.Warnw("invalid target_node_id format", "target_node_id", req.TargetNodeID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid target_node_id format, expected node_xxxxx"))
			return
		}
	}

	cmd := usecases.CreateForwardRuleCommand{
		AgentShortID:       agentShortID,
		RuleType:           req.RuleType,
		ExitAgentShortID:   exitAgentShortID,
		ChainAgentShortIDs: chainAgentShortIDs,
		Name:               req.Name,
		ListenPort:         req.ListenPort,
		TargetAddress:      req.TargetAddress,
		TargetPort:         req.TargetPort,
		TargetNodeShortID:  targetNodeShortID,
		Protocol:           req.Protocol,
		Remark:             req.Remark,
	}

	result, err := h.createRuleUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Forward rule created successfully")
}

// GetRule handles GET /forward-rules/:id
func (h *ForwardHandler) GetRule(c *gin.Context) {
	shortID, err := parseRuleShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetForwardRuleQuery{ShortID: shortID}
	result, err := h.getRuleUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// UpdateRule handles PUT /forward-rules/:id
func (h *ForwardHandler) UpdateRule(c *gin.Context) {
	shortID, err := parseRuleShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateForwardRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update forward rule", "short_id", shortID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Parse agent_id if provided
	var agentShortID *string
	if req.AgentID != nil {
		parsedID, err := id.ParseForwardAgentID(*req.AgentID)
		if err != nil {
			h.logger.Warnw("invalid agent_id format", "agent_id", *req.AgentID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id format, expected fa_xxxxx"))
			return
		}
		agentShortID = &parsedID
	}

	// Parse exit_agent_id if provided
	var exitAgentShortID *string
	if req.ExitAgentID != nil {
		parsedID, err := id.ParseForwardAgentID(*req.ExitAgentID)
		if err != nil {
			h.logger.Warnw("invalid exit_agent_id format", "exit_agent_id", *req.ExitAgentID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid exit_agent_id format, expected fa_xxxxx"))
			return
		}
		exitAgentShortID = &parsedID
	}

	// Parse chain_agent_ids if provided
	var chainAgentShortIDs []string
	if req.ChainAgentIDs != nil {
		chainAgentShortIDs = make([]string, len(req.ChainAgentIDs))
		for i, chainAgentID := range req.ChainAgentIDs {
			parsedID, parseErr := id.ParseForwardAgentID(chainAgentID)
			if parseErr != nil {
				h.logger.Warnw("invalid chain_agent_id format", "chain_agent_id", chainAgentID, "error", parseErr)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid chain_agent_id format, expected fa_xxxxx"))
				return
			}
			chainAgentShortIDs[i] = parsedID
		}
	}

	// Parse target_node_id if provided
	var targetNodeShortID *string
	if req.TargetNodeID != nil {
		if *req.TargetNodeID == "" {
			// Empty string means clear the target node
			emptyStr := ""
			targetNodeShortID = &emptyStr
		} else {
			// Parse Stripe-style ID
			nodeShortID, err := id.ParseNodeID(*req.TargetNodeID)
			if err != nil {
				h.logger.Warnw("invalid target_node_id format", "target_node_id", *req.TargetNodeID, "error", err)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid target_node_id format, expected node_xxxxx"))
				return
			}
			targetNodeShortID = &nodeShortID
		}
	}

	cmd := usecases.UpdateForwardRuleCommand{
		ShortID:            shortID,
		Name:               req.Name,
		AgentShortID:       agentShortID,
		ExitAgentShortID:   exitAgentShortID,
		ChainAgentShortIDs: chainAgentShortIDs,
		ListenPort:         req.ListenPort,
		TargetAddress:      req.TargetAddress,
		TargetPort:         req.TargetPort,
		TargetNodeShortID:  targetNodeShortID,
		Protocol:           req.Protocol,
		Remark:             req.Remark,
	}

	if err := h.updateRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rule updated successfully", nil)
}

// DeleteRule handles DELETE /forward-rules/:id
func (h *ForwardHandler) DeleteRule(c *gin.Context) {
	shortID, err := parseRuleShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DeleteForwardRuleCommand{ShortID: shortID}
	if err := h.deleteRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// ListRules handles GET /forward-rules
func (h *ForwardHandler) ListRules(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := usecases.ListForwardRulesQuery{
		Page:     page,
		PageSize: pageSize,
		Name:     c.Query("name"),
		Protocol: c.Query("protocol"),
		Status:   c.Query("status"),
		OrderBy:  c.Query("order_by"),
		Order:    c.Query("order"),
	}

	result, err := h.listRulesUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Rules, result.Total, page, pageSize)
}

// EnableRule handles POST /forward-rules/:id/enable
func (h *ForwardHandler) EnableRule(c *gin.Context) {
	shortID, err := parseRuleShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.EnableForwardRuleCommand{ShortID: shortID}
	if err := h.enableRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rule enabled successfully", nil)
}

// DisableRule handles POST /forward-rules/:id/disable
func (h *ForwardHandler) DisableRule(c *gin.Context) {
	shortID, err := parseRuleShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DisableForwardRuleCommand{ShortID: shortID}
	if err := h.disableRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rule disabled successfully", nil)
}

// UpdateStatusRequest represents a request to update forward rule status.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=enabled disabled" example:"enabled"`
}

// UpdateStatus handles PATCH /forward-rules/:id/status
func (h *ForwardHandler) UpdateStatus(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	if req.Status == "enabled" {
		h.EnableRule(c)
	} else {
		h.DisableRule(c)
	}
}

// ResetTraffic handles POST /forward-rules/:id/reset-traffic
func (h *ForwardHandler) ResetTraffic(c *gin.Context) {
	shortID, err := parseRuleShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.ResetForwardRuleTrafficCommand{ShortID: shortID}
	if err := h.resetTrafficUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Traffic counters reset successfully", nil)
}

// ProbeRuleRequest represents the request body for probing a rule.
type ProbeRuleRequest struct {
	IPVersion string `json:"ip_version"` // optional: auto, ipv4, ipv6
}

// ProbeRule handles POST /forward-rules/:id/probe
func (h *ForwardHandler) ProbeRule(c *gin.Context) {
	shortID, err := parseRuleShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if h.probeService == nil {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Probe service not available")
		return
	}

	// Parse optional request body
	var req ProbeRuleRequest
	// Ignore binding errors for optional body
	_ = c.ShouldBindJSON(&req)

	result, err := h.probeService.ProbeRuleByShortID(c.Request.Context(), shortID, req.IPVersion)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Probe completed", result)
}

// parseRuleShortID extracts the short ID from a prefixed rule ID (e.g., "fr_xK9mP2vL3nQ" -> "xK9mP2vL3nQ").
func parseRuleShortID(c *gin.Context) (string, error) {
	prefixedID := c.Param("id")
	if prefixedID == "" {
		return "", errors.NewValidationError("forward rule ID is required")
	}

	shortID, err := id.ParseForwardRuleID(prefixedID)
	if err != nil {
		return "", errors.NewValidationError("invalid forward rule ID format, expected fr_xxxxx")
	}

	return shortID, nil
}

// parseRuleID is deprecated, use parseRuleShortID instead.
// Kept for backward compatibility with internal routes.
func parseRuleID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	parsedID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid forward rule ID")
	}
	if parsedID == 0 {
		return 0, errors.NewValidationError("Forward rule ID must be greater than 0")
	}
	return uint(parsedID), nil
}
