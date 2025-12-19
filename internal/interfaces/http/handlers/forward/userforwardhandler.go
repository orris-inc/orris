// Package forward provides HTTP handlers for forward rule management.
package forward

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

// UserForwardRuleHandler handles HTTP requests for user-level forward rules and agents.
type UserForwardRuleHandler struct {
	createRuleUC  *usecases.CreateUserForwardRuleUseCase
	listRulesUC   *usecases.ListUserForwardRulesUseCase
	getUsageUC    *usecases.GetUserForwardUsageUseCase
	updateRuleUC  *usecases.UpdateForwardRuleUseCase
	deleteRuleUC  *usecases.DeleteForwardRuleUseCase
	enableRuleUC  *usecases.EnableForwardRuleUseCase
	disableRuleUC *usecases.DisableForwardRuleUseCase
	getRuleUC     *usecases.GetForwardRuleUseCase
	listAgentsUC  *usecases.ListUserForwardAgentsUseCase
	logger        logger.Interface
}

// NewUserForwardRuleHandler creates a new UserForwardRuleHandler.
func NewUserForwardRuleHandler(
	createRuleUC *usecases.CreateUserForwardRuleUseCase,
	listRulesUC *usecases.ListUserForwardRulesUseCase,
	getUsageUC *usecases.GetUserForwardUsageUseCase,
	updateRuleUC *usecases.UpdateForwardRuleUseCase,
	deleteRuleUC *usecases.DeleteForwardRuleUseCase,
	enableRuleUC *usecases.EnableForwardRuleUseCase,
	disableRuleUC *usecases.DisableForwardRuleUseCase,
	getRuleUC *usecases.GetForwardRuleUseCase,
	listAgentsUC *usecases.ListUserForwardAgentsUseCase,
) *UserForwardRuleHandler {
	return &UserForwardRuleHandler{
		createRuleUC:  createRuleUC,
		listRulesUC:   listRulesUC,
		getUsageUC:    getUsageUC,
		updateRuleUC:  updateRuleUC,
		deleteRuleUC:  deleteRuleUC,
		enableRuleUC:  enableRuleUC,
		disableRuleUC: disableRuleUC,
		getRuleUC:     getRuleUC,
		listAgentsUC:  listAgentsUC,
		logger:        logger.NewLogger(),
	}
}

// CreateUserForwardRuleRequest represents a request to create a user forward rule.
// Required fields by rule type:
// - direct: agent_id, listen_port, (target_address+target_port OR target_node_id)
// - entry: agent_id, exit_agent_id, listen_port, (target_address+target_port OR target_node_id)
// - chain: agent_id, chain_agent_ids, listen_port, (target_address+target_port OR target_node_id)
// - direct_chain: agent_id, chain_agent_ids, chain_port_config, (target_address+target_port OR target_node_id)
type CreateUserForwardRuleRequest struct {
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
	Remark            string            `json:"remark,omitempty" example:"Forward to internal MySQL server"`
}

// CreateRule handles POST /user/forward-rules
func (h *UserForwardRuleHandler) CreateRule(c *gin.Context) {
	// Get user_id from context (set by auth middleware)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Warnw("user_id not found in context")
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		h.logger.Warnw("invalid user_id type in context", "user_id", userIDInterface)
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

	// Parse Stripe-style IDs to extract short IDs
	agentShortID, err := id.ParseForwardAgentID(req.AgentID)
	if err != nil {
		h.logger.Warnw("invalid agent_id format", "agent_id", req.AgentID, "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id format, expected fa_xxxxx"))
		return
	}

	var exitAgentShortID string
	if req.ExitAgentID != "" {
		exitAgentShortID, err = id.ParseForwardAgentID(req.ExitAgentID)
		if err != nil {
			h.logger.Warnw("invalid exit_agent_id format", "exit_agent_id", req.ExitAgentID, "user_id", userID, "error", err)
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
				h.logger.Warnw("invalid chain_agent_id format", "chain_agent_id", chainAgentID, "user_id", userID, "error", parseErr)
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
				h.logger.Warnw("invalid agent_id in chain_port_config", "agent_id", agentIDStr, "user_id", userID, "error", parseErr)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id in chain_port_config, expected fa_xxxxx"))
				return
			}
			chainPortConfig[shortID] = port
		}
	}

	var targetNodeSID string
	if req.TargetNodeID != "" {
		targetNodeSID, err = id.ParseNodeID(req.TargetNodeID)
		if err != nil {
			h.logger.Warnw("invalid target_node_id format", "target_node_id", req.TargetNodeID, "user_id", userID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid target_node_id format, expected node_xxxxx"))
			return
		}
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
func (h *UserForwardRuleHandler) ListRules(c *gin.Context) {
	// Get user_id from context (set by auth middleware)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Warnw("user_id not found in context")
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		h.logger.Warnw("invalid user_id type in context", "user_id", userIDInterface)
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
func (h *UserForwardRuleHandler) GetUsage(c *gin.Context) {
	// Get user_id from context (set by auth middleware)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Warnw("user_id not found in context")
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		h.logger.Warnw("invalid user_id type in context", "user_id", userIDInterface)
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
func (h *UserForwardRuleHandler) GetRule(c *gin.Context) {
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
func (h *UserForwardRuleHandler) UpdateRule(c *gin.Context) {
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
	var targetNodeSID *string
	if req.TargetNodeID != nil {
		if *req.TargetNodeID == "" {
			// Empty string means clear the target node
			emptyStr := ""
			targetNodeSID = &emptyStr
		} else {
			// Parse Stripe-style ID
			nodeShortID, err := id.ParseNodeID(*req.TargetNodeID)
			if err != nil {
				h.logger.Warnw("invalid target_node_id format", "target_node_id", *req.TargetNodeID, "error", err)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid target_node_id format, expected node_xxxxx"))
				return
			}
			targetNodeSID = &nodeShortID
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
func (h *UserForwardRuleHandler) DeleteRule(c *gin.Context) {
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
func (h *UserForwardRuleHandler) EnableRule(c *gin.Context) {
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
func (h *UserForwardRuleHandler) DisableRule(c *gin.Context) {
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

// ListAgents handles GET /user/forward-agents
// Returns forward agents accessible to the user through their subscriptions.
func (h *UserForwardRuleHandler) ListAgents(c *gin.Context) {
	// Get user_id from context (set by auth middleware)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Warnw("user_id not found in context")
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		h.logger.Warnw("invalid user_id type in context", "user_id", userIDInterface)
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
	}

	result, err := h.listAgentsUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Agents, result.Total, page, pageSize)
}
