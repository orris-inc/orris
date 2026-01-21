// Package subscription provides HTTP handlers for subscription-level forward management.
package subscription

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// Handler handles HTTP requests for subscription-level forward rules.
type Handler struct {
	createRuleUC   *usecases.CreateSubscriptionForwardRuleUseCase
	listRulesUC    *usecases.ListSubscriptionForwardRulesUseCase
	getUsageUC     *usecases.GetSubscriptionForwardUsageUseCase
	updateRuleUC   *usecases.UpdateForwardRuleUseCase
	deleteRuleUC   *usecases.DeleteForwardRuleUseCase
	enableRuleUC   *usecases.EnableForwardRuleUseCase
	disableRuleUC  *usecases.DisableForwardRuleUseCase
	getRuleUC      *usecases.GetForwardRuleUseCase
	reorderRulesUC *usecases.ReorderForwardRulesUseCase
	logger         logger.Interface
}

// NewHandler creates a new subscription Handler.
func NewHandler(
	createRuleUC *usecases.CreateSubscriptionForwardRuleUseCase,
	listRulesUC *usecases.ListSubscriptionForwardRulesUseCase,
	getUsageUC *usecases.GetSubscriptionForwardUsageUseCase,
	updateRuleUC *usecases.UpdateForwardRuleUseCase,
	deleteRuleUC *usecases.DeleteForwardRuleUseCase,
	enableRuleUC *usecases.EnableForwardRuleUseCase,
	disableRuleUC *usecases.DisableForwardRuleUseCase,
	getRuleUC *usecases.GetForwardRuleUseCase,
	reorderRulesUC *usecases.ReorderForwardRulesUseCase,
) *Handler {
	return &Handler{
		createRuleUC:   createRuleUC,
		listRulesUC:    listRulesUC,
		getUsageUC:     getUsageUC,
		updateRuleUC:   updateRuleUC,
		deleteRuleUC:   deleteRuleUC,
		enableRuleUC:   enableRuleUC,
		disableRuleUC:  disableRuleUC,
		getRuleUC:      getRuleUC,
		reorderRulesUC: reorderRulesUC,
		logger:         logger.NewLogger(),
	}
}

// CreateForwardRuleRequest represents a request to create a subscription forward rule.
type CreateForwardRuleRequest struct {
	AgentID           string            `json:"agent_id" binding:"required" example:"fa_xK9mP2vL3nQ"`
	RuleType          string            `json:"rule_type" binding:"required,oneof=direct entry chain direct_chain" example:"direct"`
	ExitAgentID       string            `json:"exit_agent_id,omitempty" example:"fa_yL8nQ3wM4oR"`
	ChainAgentIDs     []string          `json:"chain_agent_ids,omitempty" example:"[\"fa_aaa\",\"fa_bbb\"]"`
	ChainPortConfig   map[string]uint16 `json:"chain_port_config,omitempty" example:"{\"fa_xK9mP2vL3nQ\":8080,\"fa_yL8nQ3wM4oR\":9090}"`
	Name              string            `json:"name" binding:"required" example:"MySQL-Forward"`
	ListenPort        uint16            `json:"listen_port,omitempty" example:"13306"`
	TargetAddress     string            `json:"target_address,omitempty" example:"192.168.1.100"`
	TargetPort        uint16            `json:"target_port,omitempty" example:"3306"`
	TargetNodeID      string            `json:"target_node_id,omitempty" example:"node_xK9mP2vL3nQ"`
	BindIP            string            `json:"bind_ip,omitempty" example:"192.168.1.1"`
	IPVersion         string            `json:"ip_version,omitempty" binding:"omitempty,oneof=auto ipv4 ipv6" example:"auto"`
	Protocol          string            `json:"protocol" binding:"required,oneof=tcp udp both" example:"tcp"`
	TrafficMultiplier *float64          `json:"traffic_multiplier,omitempty" binding:"omitempty,gte=0,lte=1000000" example:"1.5"`
	SortOrder         *int              `json:"sort_order,omitempty" binding:"omitempty,gte=0" example:"100"`
	Remark            string            `json:"remark,omitempty" example:"Forward to internal MySQL server"`
}

// UpdateForwardRuleRequest represents a request to update a forward rule.
type UpdateForwardRuleRequest struct {
	Name              *string           `json:"name,omitempty" example:"MySQL-Forward-Updated"`
	AgentID           *string           `json:"agent_id,omitempty" example:"fa_xK9mP2vL3nQ"`
	ExitAgentID       *string           `json:"exit_agent_id,omitempty" example:"fa_yL8nQ3wM4oR"`
	ChainAgentIDs     []string          `json:"chain_agent_ids,omitempty" example:"[\"fa_aaa\",\"fa_bbb\"]"`
	ChainPortConfig   map[string]uint16 `json:"chain_port_config,omitempty" example:"{\"fa_xK9mP2vL3nQ\":8080,\"fa_yL8nQ3wM4oR\":9090}"`
	TunnelHops        *int              `json:"tunnel_hops,omitempty" binding:"omitempty,gte=0,lte=10" example:"2"`
	TunnelType        *string           `json:"tunnel_type,omitempty" binding:"omitempty,oneof=ws tls" example:"ws"`
	ListenPort        *uint16           `json:"listen_port,omitempty" example:"13307"`
	TargetAddress     *string           `json:"target_address,omitempty" example:"192.168.1.101"`
	TargetPort        *uint16           `json:"target_port,omitempty" example:"3307"`
	TargetNodeID      *string           `json:"target_node_id,omitempty" example:"node_xK9mP2vL3nQ"`
	BindIP            *string           `json:"bind_ip,omitempty" example:"192.168.1.1"`
	IPVersion         *string           `json:"ip_version,omitempty" binding:"omitempty,oneof=auto ipv4 ipv6" example:"auto"`
	Protocol          *string           `json:"protocol,omitempty" binding:"omitempty,oneof=tcp udp both" example:"tcp"`
	TrafficMultiplier *float64          `json:"traffic_multiplier,omitempty" binding:"omitempty,gte=0,lte=1000000" example:"1.5"`
	SortOrder         *int              `json:"sort_order,omitempty" example:"100"`
	Remark            *string           `json:"remark,omitempty" example:"Updated remark"`
}

// ReorderForwardRulesRequest represents a request to reorder forward rules.
type ReorderForwardRulesRequest struct {
	RuleOrders []ForwardRuleOrder `json:"rule_orders" binding:"required,min=1,dive"`
}

// ForwardRuleOrder represents a single rule's sort order.
type ForwardRuleOrder struct {
	RuleID    string `json:"rule_id" binding:"required" example:"fr_xK9mP2vL3nQ"`
	SortOrder int    `json:"sort_order" binding:"gte=0" example:"100"`
}

// parseRuleShortID validates a prefixed rule ID and returns the SID.
// Gets rule ID from :rule_id URL parameter.
func parseRuleShortID(c *gin.Context) (string, error) {
	prefixedID := c.Param("rule_id")
	if prefixedID == "" {
		return "", errors.NewValidationError("forward rule ID is required")
	}

	if err := id.ValidatePrefix(prefixedID, id.PrefixForwardRule); err != nil {
		return "", errors.NewValidationError("invalid forward rule ID format, expected fr_xxxxx")
	}

	return prefixedID, nil
}

// getSubscriptionIDFromContext retrieves subscription_id from context (set by SubscriptionOwnerMiddleware).
func getSubscriptionIDFromContext(c *gin.Context, log logger.Interface) (uint, error) {
	subscriptionIDInterface, exists := c.Get("subscription_id")
	if !exists {
		log.Warnw("subscription_id not found in context", "ip", c.ClientIP())
		return 0, errors.NewUnauthorizedError("subscription context not available")
	}

	subscriptionID, ok := subscriptionIDInterface.(uint)
	if !ok {
		log.Warnw("invalid subscription_id type in context", "subscription_id", subscriptionIDInterface, "ip", c.ClientIP())
		return 0, errors.NewInternalError("invalid subscription ID type")
	}

	return subscriptionID, nil
}

// getUserIDFromContext retrieves user_id from context (set by auth middleware).
func getUserIDFromContext(c *gin.Context, log logger.Interface) (uint, error) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		log.Warnw("user_id not found in context", "ip", c.ClientIP())
		return 0, errors.NewUnauthorizedError("user not authenticated")
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		log.Warnw("invalid user_id type in context", "user_id", userIDInterface, "ip", c.ClientIP())
		return 0, errors.NewInternalError("invalid user ID type")
	}

	return userID, nil
}

// CreateRule handles POST /subscriptions/:sid/forward-rules
func (h *Handler) CreateRule(c *gin.Context) {
	userID, err := getUserIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	subscriptionID, err := getSubscriptionIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req CreateForwardRuleRequest
	// Use ShouldBindBodyWith to read cached body (cached by ForwardQuotaMiddleware)
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		h.logger.Warnw("invalid request body for create subscription forward rule", "subscription_id", subscriptionID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Validate Stripe-style IDs
	if err := id.ValidatePrefix(req.AgentID, id.PrefixForwardAgent); err != nil {
		h.logger.Warnw("invalid agent_id format", "agent_id", req.AgentID, "subscription_id", subscriptionID, "error", err)
		utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id format, expected fa_xxxxx"))
		return
	}
	agentShortID := req.AgentID

	var exitAgentShortID string
	if req.ExitAgentID != "" {
		if err := id.ValidatePrefix(req.ExitAgentID, id.PrefixForwardAgent); err != nil {
			h.logger.Warnw("invalid exit_agent_id format", "exit_agent_id", req.ExitAgentID, "subscription_id", subscriptionID, "error", err)
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
				h.logger.Warnw("invalid chain_agent_id format", "chain_agent_id", chainAgentID, "subscription_id", subscriptionID, "error", err)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid chain_agent_id format, expected fa_xxxxx"))
				return
			}
			chainAgentShortIDs[i] = chainAgentID
		}
	}

	// Validate chain port config
	var chainPortConfig map[string]uint16
	if len(req.ChainPortConfig) > 0 {
		chainPortConfig = make(map[string]uint16, len(req.ChainPortConfig))
		for agentIDStr, port := range req.ChainPortConfig {
			if err := id.ValidatePrefix(agentIDStr, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid agent_id in chain_port_config", "agent_id", agentIDStr, "subscription_id", subscriptionID, "error", err)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id in chain_port_config, expected fa_xxxxx"))
				return
			}
			chainPortConfig[agentIDStr] = port
		}
	}

	var targetNodeSID string
	if req.TargetNodeID != "" {
		if err := id.ValidatePrefix(req.TargetNodeID, id.PrefixNode); err != nil {
			h.logger.Warnw("invalid target_node_id format", "target_node_id", req.TargetNodeID, "subscription_id", subscriptionID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid target_node_id format, expected node_xxxxx"))
			return
		}
		targetNodeSID = req.TargetNodeID
	}

	// Get rule limit from context (set by ForwardQuotaMiddleware) for secondary race condition check
	var ruleLimit int
	if ruleLimitVal, exists := c.Get("subscription_rule_limit"); exists {
		if limit, ok := ruleLimitVal.(int); ok {
			ruleLimit = limit
		}
	}

	cmd := usecases.CreateSubscriptionForwardRuleCommand{
		UserID:             userID,
		SubscriptionID:     subscriptionID,
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
		RuleLimit:          ruleLimit,
	}

	result, err := h.createRuleUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Forward rule created successfully")
}

// ListRules handles GET /subscriptions/:sid/forward-rules
// Note: External rules are now part of the unified ForwardRule model with rule_type='external'.
// They are returned together with other rules and can be identified by their rule_type field.
func (h *Handler) ListRules(c *gin.Context) {
	subscriptionID, err := getSubscriptionIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
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

	query := usecases.ListSubscriptionForwardRulesQuery{
		SubscriptionID: subscriptionID,
		Page:           page,
		PageSize:       pageSize,
		Name:           c.Query("name"),
		Protocol:       c.Query("protocol"),
		Status:         c.Query("status"),
		OrderBy:        c.Query("order_by"),
		Order:          c.Query("order"),
	}

	result, err := h.listRulesUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Rules, result.Total, page, pageSize)
}

// GetUsage handles GET /subscriptions/:sid/forward-rules/usage
func (h *Handler) GetUsage(c *gin.Context) {
	subscriptionID, err := getSubscriptionIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetSubscriptionForwardUsageQuery{
		SubscriptionID: subscriptionID,
	}

	result, err := h.getUsageUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetRule handles GET /subscriptions/:sid/forward-rules/:rule_id
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

// UpdateRule handles PUT /subscriptions/:sid/forward-rules/:rule_id
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

	// Validate agent_id if provided
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

	// Validate chain_port_config if provided
	var chainPortConfig map[string]uint16
	if req.ChainPortConfig != nil {
		chainPortConfig = make(map[string]uint16, len(req.ChainPortConfig))
		for agentIDStr, port := range req.ChainPortConfig {
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

// DeleteRule handles DELETE /subscriptions/:sid/forward-rules/:rule_id
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

// EnableRule handles POST /subscriptions/:sid/forward-rules/:rule_id/enable
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

// DisableRule handles POST /subscriptions/:sid/forward-rules/:rule_id/disable
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

// ReorderRules handles PATCH /subscriptions/:sid/forward-rules/reorder
func (h *Handler) ReorderRules(c *gin.Context) {
	subscriptionID, err := getSubscriptionIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req ReorderForwardRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for reorder rules", "subscription_id", subscriptionID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Validate rule IDs
	ruleOrders := make([]usecases.RuleOrder, len(req.RuleOrders))
	for i, order := range req.RuleOrders {
		if err := id.ValidatePrefix(order.RuleID, id.PrefixForwardRule); err != nil {
			h.logger.Warnw("invalid rule_id format", "rule_id", order.RuleID, "subscription_id", subscriptionID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid rule_id format, expected fr_xxxxx"))
			return
		}
		ruleOrders[i] = usecases.RuleOrder{
			RuleSID:   order.RuleID,
			SortOrder: order.SortOrder,
		}
	}

	cmd := usecases.ReorderForwardRulesCommand{
		RuleOrders:     ruleOrders,
		SubscriptionID: &subscriptionID,
	}

	if err := h.reorderRulesUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}
