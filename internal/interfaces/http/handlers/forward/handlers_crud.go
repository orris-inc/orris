package forward

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/utils"
)

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

	// Parse chain port config (for direct_chain type)
	var chainPortConfig map[string]uint16
	if len(req.ChainPortConfig) > 0 {
		chainPortConfig = make(map[string]uint16, len(req.ChainPortConfig))
		for agentIDStr, port := range req.ChainPortConfig {
			// Parse agent ID from chain_port_config
			shortID, parseErr := id.ParseForwardAgentID(agentIDStr)
			if parseErr != nil {
				h.logger.Warnw("invalid agent_id in chain_port_config", "agent_id", agentIDStr, "error", parseErr)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id in chain_port_config, expected fa_xxxxx"))
				return
			}
			chainPortConfig[shortID] = port
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
		ChainPortConfig:    chainPortConfig,
		Name:               req.Name,
		ListenPort:         req.ListenPort,
		TargetAddress:      req.TargetAddress,
		TargetPort:         req.TargetPort,
		TargetNodeShortID:  targetNodeShortID,
		BindIP:             req.BindIP,
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

	// Parse chain_port_config if provided (for direct_chain type)
	var chainPortConfig map[string]uint16
	if req.ChainPortConfig != nil {
		chainPortConfig = make(map[string]uint16, len(req.ChainPortConfig))
		for agentIDStr, port := range req.ChainPortConfig {
			// Parse agent ID from chain_port_config
			shortID, parseErr := id.ParseForwardAgentID(agentIDStr)
			if parseErr != nil {
				h.logger.Warnw("invalid agent_id in chain_port_config", "agent_id", agentIDStr, "error", parseErr)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id in chain_port_config, expected fa_xxxxx"))
				return
			}
			chainPortConfig[shortID] = port
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
		ChainPortConfig:    chainPortConfig,
		ListenPort:         req.ListenPort,
		TargetAddress:      req.TargetAddress,
		TargetPort:         req.TargetPort,
		TargetNodeShortID:  targetNodeShortID,
		BindIP:             req.BindIP,
		IPVersion:          req.IPVersion,
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
