package user

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// CreateRule handles POST /user/forward-rules
func (h *Handler) CreateRule(c *gin.Context) {
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

	var req CreateUserForwardRuleRequest
	// Use ShouldBindBodyWith to read cached body (cached by ForwardQuotaMiddleware)
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		h.logger.Warnw("invalid request body for create user forward rule", "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Validate Stripe-style IDs (database stores full SID with prefix)
	if err := id.ValidatePrefix(req.AgentID, id.PrefixForwardAgent); err != nil {
		h.logger.Warnw("invalid agent_id format", "agent_id", req.AgentID, "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id format, expected fa_xxxxx"))
		return
	}
	agentShortID := req.AgentID

	var exitAgentShortID string
	if req.ExitAgentID != "" {
		if err := id.ValidatePrefix(req.ExitAgentID, id.PrefixForwardAgent); err != nil {
			h.logger.Warnw("invalid exit_agent_id format", "exit_agent_id", req.ExitAgentID, "user_id", userID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid exit_agent_id format, expected fa_xxxxx"))
			return
		}
		exitAgentShortID = req.ExitAgentID
	}

	// Validate chain agent IDs
	var chainAgentShortIDs []string
	if len(req.ChainAgentIDs) > 0 {
		chainAgentShortIDs = make([]string, len(req.ChainAgentIDs))
		for i, chainAgentID := range req.ChainAgentIDs {
			if err := id.ValidatePrefix(chainAgentID, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid chain_agent_id format", "chain_agent_id", chainAgentID, "user_id", userID, "error", err)
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
				h.logger.Warnw("invalid agent_id in chain_port_config", "agent_id", agentIDStr, "user_id", userID, "error", err)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id in chain_port_config, expected fa_xxxxx"))
				return
			}
			chainPortConfig[agentIDStr] = port
		}
	}

	var targetNodeSID string
	if req.TargetNodeID != "" {
		if err := id.ValidatePrefix(req.TargetNodeID, id.PrefixNode); err != nil {
			h.logger.Warnw("invalid target_node_id format", "target_node_id", req.TargetNodeID, "user_id", userID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid target_node_id format, expected node_xxxxx"))
			return
		}
		targetNodeSID = req.TargetNodeID
	}

	cmd := usecases.CreateUserForwardRuleCommand{
		UserID:             userID,
		AgentShortID:       agentShortID,
		RuleType:           req.RuleType,
		ExitAgentShortID:   exitAgentShortID,
		ChainAgentShortIDs: chainAgentShortIDs,
		ChainPortConfig:    chainPortConfig,
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
	}

	result, err := h.createRuleUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Forward rule created successfully")
}

// ListRules handles GET /user/forward-rules
func (h *Handler) ListRules(c *gin.Context) {
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

	query := usecases.ListUserForwardRulesQuery{
		UserID:   userID,
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

// GetUsage handles GET /user/forward-rules/usage
func (h *Handler) GetUsage(c *gin.Context) {
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

	query := usecases.GetUserForwardUsageQuery{
		UserID: userID,
	}

	result, err := h.getUsageUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetRule handles GET /user/forward-rules/:id
// Note: Ownership verification is handled by ForwardRuleOwnerMiddleware
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

// UpdateRule handles PUT /user/forward-rules/:id
// Note: Ownership verification is handled by ForwardRuleOwnerMiddleware
func (h *Handler) UpdateRule(c *gin.Context) {
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

	// Validate agent_id if provided (database stores full SID with prefix)
	var agentShortID *string
	if req.AgentID != nil {
		if err := id.ValidatePrefix(*req.AgentID, id.PrefixForwardAgent); err != nil {
			h.logger.Warnw("invalid agent_id format", "agent_id", *req.AgentID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id format, expected fa_xxxxx"))
			return
		}
		agentShortID = req.AgentID
	}

	// Validate exit_agent_id if provided
	var exitAgentShortID *string
	if req.ExitAgentID != nil {
		if err := id.ValidatePrefix(*req.ExitAgentID, id.PrefixForwardAgent); err != nil {
			h.logger.Warnw("invalid exit_agent_id format", "exit_agent_id", *req.ExitAgentID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid exit_agent_id format, expected fa_xxxxx"))
			return
		}
		exitAgentShortID = req.ExitAgentID
	}

	// Validate chain_agent_ids if provided
	var chainAgentShortIDs []string
	if req.ChainAgentIDs != nil {
		chainAgentShortIDs = make([]string, len(req.ChainAgentIDs))
		for i, chainAgentID := range req.ChainAgentIDs {
			if err := id.ValidatePrefix(chainAgentID, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid chain_agent_id format", "chain_agent_id", chainAgentID, "error", err)
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
				h.logger.Warnw("invalid agent_id in chain_port_config", "agent_id", agentIDStr, "error", err)
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
				h.logger.Warnw("invalid target_node_id format", "target_node_id", *req.TargetNodeID, "error", err)
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
		ChainAgentShortIDs: chainAgentShortIDs,
		ChainPortConfig:    chainPortConfig,
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
	}

	if err := h.updateRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rule updated successfully", nil)
}

// DeleteRule handles DELETE /user/forward-rules/:id
// Note: Ownership verification is handled by ForwardRuleOwnerMiddleware
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

// EnableRule handles POST /user/forward-rules/:id/enable
// Note: Ownership verification is handled by ForwardRuleOwnerMiddleware
func (h *Handler) EnableRule(c *gin.Context) {
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

// DisableRule handles POST /user/forward-rules/:id/disable
// Note: Ownership verification is handled by ForwardRuleOwnerMiddleware
func (h *Handler) DisableRule(c *gin.Context) {
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

// ReorderRules handles PATCH /user/forward-rules/reorder
func (h *Handler) ReorderRules(c *gin.Context) {
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

	var req ReorderForwardRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for reorder rules", "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Validate rule IDs
	ruleOrders := make([]usecases.RuleOrder, len(req.RuleOrders))
	for i, order := range req.RuleOrders {
		if err := id.ValidatePrefix(order.RuleID, id.PrefixForwardRule); err != nil {
			h.logger.Warnw("invalid rule_id format", "rule_id", order.RuleID, "user_id", userID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid rule_id format, expected fr_xxxxx"))
			return
		}
		ruleOrders[i] = usecases.RuleOrder{
			RuleSID:   order.RuleID,
			SortOrder: order.SortOrder,
		}
	}

	cmd := usecases.ReorderForwardRulesCommand{
		RuleOrders: ruleOrders,
		UserID:     &userID,
	}

	if err := h.reorderRulesUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}
