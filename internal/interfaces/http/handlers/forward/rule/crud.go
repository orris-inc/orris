package rule

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// CreateRule handles POST /forward-rules
func (h *Handler) CreateRule(c *gin.Context) {
	var req CreateForwardRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create forward rule", "error", err, "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, err)
		return
	}

	// External rules have different validation requirements
	isExternalRule := req.RuleType == "external"

	// Validate agent_id for non-external rules (external rules don't require agent)
	var agentShortID string
	if !isExternalRule {
		if req.AgentID == "" {
			h.logger.Warnw("agent_id is required for non-external rules", "rule_type", req.RuleType, "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("agent_id is required"))
			return
		}
		if err := id.ValidatePrefix(req.AgentID, id.PrefixForwardAgent); err != nil {
			h.logger.Warnw("invalid agent_id format", "agent_id", req.AgentID, "error", err, "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id format, expected fa_xxxxx"))
			return
		}
		agentShortID = req.AgentID
	}

	// Validate protocol for non-external rules (external rules get protocol from target_node)
	if !isExternalRule && req.Protocol == "" {
		h.logger.Warnw("protocol is required for non-external rules", "rule_type", req.RuleType, "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, errors.NewValidationError("protocol is required"))
		return
	}

	// Validate external rule specific fields
	if isExternalRule {
		if req.ServerAddress == "" {
			h.logger.Warnw("server_address is required for external rules", "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("server_address is required for external rules"))
			return
		}
		if req.ListenPort == 0 {
			h.logger.Warnw("listen_port is required for external rules", "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("listen_port is required for external rules"))
			return
		}
		if req.TargetNodeID == "" {
			h.logger.Warnw("target_node_id is required for external rules", "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("target_node_id is required for external rules (protocol info is derived from target node)"))
			return
		}
		// external_source is optional
	}

	var exitAgentShortID string
	var exitAgents []usecases.ExitAgentInput
	if req.ExitAgentID != "" && len(req.ExitAgents) > 0 {
		h.logger.Warnw("exit_agent_id and exit_agents are mutually exclusive", "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, errors.NewValidationError("exit_agent_id and exit_agents are mutually exclusive"))
		return
	}
	if req.ExitAgentID != "" {
		if err := id.ValidatePrefix(req.ExitAgentID, id.PrefixForwardAgent); err != nil {
			h.logger.Warnw("invalid exit_agent_id format", "exit_agent_id", req.ExitAgentID, "error", err, "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid exit_agent_id format, expected fa_xxxxx"))
			return
		}
		exitAgentShortID = req.ExitAgentID
	} else if len(req.ExitAgents) > 0 {
		// Validate exit_agents array length (max 10)
		if len(req.ExitAgents) > 10 {
			h.logger.Warnw("too many exit_agents", "count", len(req.ExitAgents), "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("exit_agents cannot exceed 10 entries"))
			return
		}
		exitAgents = make([]usecases.ExitAgentInput, 0, len(req.ExitAgents))
		for _, ea := range req.ExitAgents {
			if err := id.ValidatePrefix(ea.AgentID, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid exit_agent_id format in exit_agents", "exit_agent_id", ea.AgentID, "error", err, "ip", c.ClientIP())
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid exit_agent_id format in exit_agents, expected fa_xxxxx"))
				return
			}
			exitAgents = append(exitAgents, usecases.ExitAgentInput{
				AgentSID: ea.AgentID,
				Weight:   ea.Weight,
			})
		}
	}

	// Validate chain agent IDs
	var chainAgentShortIDs []string
	if len(req.ChainAgentIDs) > 0 {
		chainAgentShortIDs = make([]string, len(req.ChainAgentIDs))
		for i, chainAgentID := range req.ChainAgentIDs {
			if err := id.ValidatePrefix(chainAgentID, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid chain_agent_id format", "chain_agent_id", chainAgentID, "error", err, "ip", c.ClientIP())
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid chain_agent_id format, expected fa_xxxxx"))
				return
			}
			chainAgentShortIDs[i] = chainAgentID
		}
	}

	// Validate chain port config (for direct_chain type)
	var chainPortConfig map[string]uint16
	if len(req.ChainPortConfig) > 0 {
		chainPortConfig = make(map[string]uint16, len(req.ChainPortConfig))
		for agentIDStr, port := range req.ChainPortConfig {
			// Validate agent ID from chain_port_config
			if err := id.ValidatePrefix(agentIDStr, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid agent_id in chain_port_config", "agent_id", agentIDStr, "error", err, "ip", c.ClientIP())
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id in chain_port_config, expected fa_xxxxx"))
				return
			}
			chainPortConfig[agentIDStr] = port
		}
	}

	var targetNodeSID string
	if req.TargetNodeID != "" {
		if err := id.ValidatePrefix(req.TargetNodeID, id.PrefixNode); err != nil {
			h.logger.Warnw("invalid target_node_id format", "target_node_id", req.TargetNodeID, "error", err, "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid target_node_id format, expected node_xxxxx"))
			return
		}
		targetNodeSID = req.TargetNodeID
	}

	cmd := usecases.CreateForwardRuleCommand{
		AgentShortID:       agentShortID,
		RuleType:           req.RuleType,
		ExitAgentShortID:   exitAgentShortID,
		ExitAgents:         exitAgents,
		ChainAgentShortIDs: chainAgentShortIDs,
		ChainPortConfig:    chainPortConfig,
		TunnelHops:         req.TunnelHops,
		TunnelType:         req.TunnelType,
		Name:               req.Name,
		ListenPort:         req.ListenPort,
		TargetAddress:      req.TargetAddress,
		TargetPort:         req.TargetPort,
		TargetNodeSID:      targetNodeSID,
		BindIP:             req.BindIP,
		IPVersion:          req.IPVersion,
		Protocol:           req.Protocol,
		TrafficMultiplier:  req.TrafficMultiplier,
		SortOrder:          req.SortOrder,
		Remark:             req.Remark,
		GroupSIDs:          req.GroupSIDs,
		// External rule fields
		ServerAddress:  req.ServerAddress,
		ExternalSource: req.ExternalSource,
		ExternalRuleID: req.ExternalRuleID,
	}

	result, err := h.createRuleUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Forward rule created successfully")
}

// GetRule handles GET /forward-rules/:id
func (h *Handler) GetRule(c *gin.Context) {
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
func (h *Handler) UpdateRule(c *gin.Context) {
	shortID, err := parseRuleShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateForwardRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update forward rule", "short_id", shortID, "error", err, "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Validate agent_id if provided (database stores full SID with prefix)
	var agentShortID *string
	if req.AgentID != nil {
		if err := id.ValidatePrefix(*req.AgentID, id.PrefixForwardAgent); err != nil {
			h.logger.Warnw("invalid agent_id format", "agent_id", *req.AgentID, "error", err, "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id format, expected fa_xxxxx"))
			return
		}
		agentShortID = req.AgentID
	}

	// Validate exit_agent_id and exit_agents (mutually exclusive)
	var exitAgentShortID *string
	var exitAgents []usecases.ExitAgentInput
	if req.ExitAgentID != nil && len(req.ExitAgents) > 0 {
		h.logger.Warnw("exit_agent_id and exit_agents are mutually exclusive", "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, errors.NewValidationError("exit_agent_id and exit_agents are mutually exclusive"))
		return
	}
	if req.ExitAgentID != nil {
		if err := id.ValidatePrefix(*req.ExitAgentID, id.PrefixForwardAgent); err != nil {
			h.logger.Warnw("invalid exit_agent_id format", "exit_agent_id", *req.ExitAgentID, "error", err, "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid exit_agent_id format, expected fa_xxxxx"))
			return
		}
		exitAgentShortID = req.ExitAgentID
	}
	if len(req.ExitAgents) > 0 {
		// Validate exit_agents array length (max 10)
		if len(req.ExitAgents) > 10 {
			h.logger.Warnw("too many exit_agents", "count", len(req.ExitAgents), "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("exit_agents cannot exceed 10 entries"))
			return
		}
		exitAgents = make([]usecases.ExitAgentInput, 0, len(req.ExitAgents))
		for _, ea := range req.ExitAgents {
			if err := id.ValidatePrefix(ea.AgentID, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid exit_agent_id format in exit_agents", "exit_agent_id", ea.AgentID, "error", err, "ip", c.ClientIP())
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid exit_agent_id format in exit_agents, expected fa_xxxxx"))
				return
			}
			exitAgents = append(exitAgents, usecases.ExitAgentInput{
				AgentSID: ea.AgentID,
				Weight:   ea.Weight,
			})
		}
	}

	// Validate chain_agent_ids if provided
	var chainAgentShortIDs []string
	if req.ChainAgentIDs != nil {
		chainAgentShortIDs = make([]string, len(req.ChainAgentIDs))
		for i, chainAgentID := range req.ChainAgentIDs {
			if err := id.ValidatePrefix(chainAgentID, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid chain_agent_id format", "chain_agent_id", chainAgentID, "error", err, "ip", c.ClientIP())
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid chain_agent_id format, expected fa_xxxxx"))
				return
			}
			chainAgentShortIDs[i] = chainAgentID
		}
	}

	// Validate chain_port_config if provided (for direct_chain type)
	var chainPortConfig map[string]uint16
	if req.ChainPortConfig != nil {
		chainPortConfig = make(map[string]uint16, len(req.ChainPortConfig))
		for agentIDStr, port := range req.ChainPortConfig {
			// Validate agent ID from chain_port_config
			if err := id.ValidatePrefix(agentIDStr, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid agent_id in chain_port_config", "agent_id", agentIDStr, "error", err, "ip", c.ClientIP())
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id in chain_port_config, expected fa_xxxxx"))
				return
			}
			chainPortConfig[agentIDStr] = port
		}
	}

	// Parse target_node_id if provided
	var targetNodeSID *string
	if req.TargetNodeID != nil {
		if *req.TargetNodeID == "" {
			// Empty string means clear the target node
			emptyStr := ""
			targetNodeSID = &emptyStr
		} else {
			// Validate Stripe-style ID (database stores full SID with prefix)
			if err := id.ValidatePrefix(*req.TargetNodeID, id.PrefixNode); err != nil {
				h.logger.Warnw("invalid target_node_id format", "target_node_id", *req.TargetNodeID, "error", err, "ip", c.ClientIP())
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid target_node_id format, expected node_xxxxx"))
				return
			}
			targetNodeSID = req.TargetNodeID
		}
	}

	cmd := usecases.UpdateForwardRuleCommand{
		ShortID:            shortID,
		Name:               req.Name,
		AgentShortID:       agentShortID,
		ExitAgentShortID:   exitAgentShortID,
		ExitAgents:         exitAgents,
		ChainAgentShortIDs: chainAgentShortIDs,
		ChainPortConfig:    chainPortConfig,
		TunnelHops:         req.TunnelHops,
		TunnelType:         req.TunnelType,
		ListenPort:         req.ListenPort,
		TargetAddress:      req.TargetAddress,
		TargetPort:         req.TargetPort,
		TargetNodeSID:      targetNodeSID,
		BindIP:             req.BindIP,
		IPVersion:          req.IPVersion,
		Protocol:           req.Protocol,
		TrafficMultiplier:  req.TrafficMultiplier,
		SortOrder:          req.SortOrder,
		Remark:             req.Remark,
		GroupSIDs:          req.GroupSIDs,
	}

	if err := h.updateRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rule updated successfully", nil)
}

// DeleteRule handles DELETE /forward-rules/:id
func (h *Handler) DeleteRule(c *gin.Context) {
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
func (h *Handler) ListRules(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(constants.DefaultPageSize)))
	if pageSize < 1 || pageSize > constants.MaxPageSize {
		pageSize = constants.DefaultPageSize
	}

	// Parse include_user_rules parameter (default: false)
	includeUserRules := c.Query("include_user_rules") == "true"

	query := usecases.ListForwardRulesQuery{
		Page:             page,
		PageSize:         pageSize,
		Name:             c.Query("name"),
		Protocol:         c.Query("protocol"),
		Status:           c.Query("status"),
		OrderBy:          c.Query("order_by"),
		Order:            c.Query("order"),
		IncludeUserRules: includeUserRules,
	}

	result, err := h.listRulesUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Rules, result.Total, page, pageSize)
}
