package user

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// BatchCreateForwardRulesRequest represents a request to create multiple forward rules.
type BatchCreateForwardRulesRequest struct {
	Rules []CreateUserForwardRuleRequest `json:"rules" binding:"required,min=1,max=100,dive"`
}

// BatchDeleteForwardRulesRequest represents a request to delete multiple forward rules.
type BatchDeleteForwardRulesRequest struct {
	RuleIDs []string `json:"rule_ids" binding:"required,min=1,max=100,dive,required"`
}

// BatchToggleStatusRequest represents a request to enable/disable multiple forward rules.
type BatchToggleStatusRequest struct {
	RuleIDs []string `json:"rule_ids" binding:"required,min=1,max=100,dive,required"`
	Status  string   `json:"status" binding:"required,oneof=enabled disabled"`
}

// BatchUpdateForwardRulesRequest represents a request to update multiple forward rules.
type BatchUpdateForwardRulesRequest struct {
	Updates []BatchUpdateItem `json:"updates" binding:"required,min=1,max=100,dive"`
}

// BatchUpdateItem represents a single rule update.
// Supports: name, remark, sort_order, agent_id (entry), exit_agent_id (exit).
// Note: chain_agent_ids is NOT supported in batch update - use single rule update instead.
type BatchUpdateItem struct {
	RuleID      string  `json:"rule_id" binding:"required"`
	Name        *string `json:"name,omitempty"`
	Remark      *string `json:"remark,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
	AgentID     *string `json:"agent_id,omitempty"`      // entry agent ID
	ExitAgentID *string `json:"exit_agent_id,omitempty"` // exit agent ID (for entry type rules)
}

// getUserIDFromContext extracts user ID from gin context.
// Returns the user ID and true if successful, or 0 and false if failed (error response already sent).
func (h *Handler) getUserIDFromContext(c *gin.Context) (uint, bool) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Warnw("user_id not found in context", "ip", c.ClientIP())
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return 0, false
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		h.logger.Warnw("invalid user_id type in context", "user_id", userIDInterface, "ip", c.ClientIP())
		utils.ErrorResponse(c, http.StatusInternalServerError, "invalid user ID type")
		return 0, false
	}

	return userID, true
}

// BatchCreateRules handles POST /user/forward-rules/batch
func (h *Handler) BatchCreateRules(c *gin.Context) {
	userID, ok := h.getUserIDFromContext(c)
	if !ok {
		return
	}

	var req BatchCreateForwardRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for batch create rules", "user_id", userID, "error", err, "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Convert request to UseCase commands with partial failure mode for validation
	cmds := make([]usecases.CreateUserForwardRuleCommand, 0, len(req.Rules))
	cmdIndices := make([]int, 0, len(req.Rules)) // track original indices for valid commands
	preValidationFailed := make([]dto.BatchOperationErr, 0)

	for i, r := range req.Rules {
		identifier := fmt.Sprintf("index_%d", i)
		if r.Name != "" {
			identifier = r.Name
		}

		// Validate Stripe-style IDs (partial failure mode - collect errors, continue processing)
		if err := id.ValidatePrefix(r.AgentID, id.PrefixForwardAgent); err != nil {
			h.logger.Warnw("invalid agent_id format in batch create", "agent_id", r.AgentID, "user_id", userID, "index", i, "error", err)
			preValidationFailed = append(preValidationFailed, dto.BatchOperationErr{
				ID:     identifier,
				Reason: "invalid agent_id format, expected fa_xxxxx",
			})
			continue
		}

		var exitAgentShortID string
		if r.ExitAgentID != "" {
			if err := id.ValidatePrefix(r.ExitAgentID, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid exit_agent_id format in batch create", "exit_agent_id", r.ExitAgentID, "user_id", userID, "index", i, "error", err)
				preValidationFailed = append(preValidationFailed, dto.BatchOperationErr{
					ID:     identifier,
					Reason: "invalid exit_agent_id format, expected fa_xxxxx",
				})
				continue
			}
			exitAgentShortID = r.ExitAgentID
		}

		// Validate chain agent IDs
		var chainAgentShortIDs []string
		var chainValidationFailed bool
		if len(r.ChainAgentIDs) > 0 {
			chainAgentShortIDs = make([]string, len(r.ChainAgentIDs))
			for j, chainAgentID := range r.ChainAgentIDs {
				if err := id.ValidatePrefix(chainAgentID, id.PrefixForwardAgent); err != nil {
					h.logger.Warnw("invalid chain_agent_id format in batch create", "chain_agent_id", chainAgentID, "user_id", userID, "index", i, "error", err)
					preValidationFailed = append(preValidationFailed, dto.BatchOperationErr{
						ID:     identifier,
						Reason: "invalid chain_agent_id format, expected fa_xxxxx",
					})
					chainValidationFailed = true
					break
				}
				chainAgentShortIDs[j] = chainAgentID
			}
		}
		if chainValidationFailed {
			continue
		}

		// Validate chain port config
		var chainPortConfig map[string]uint16
		var chainPortValidationFailed bool
		if len(r.ChainPortConfig) > 0 {
			chainPortConfig = make(map[string]uint16, len(r.ChainPortConfig))
			for agentIDStr, port := range r.ChainPortConfig {
				if err := id.ValidatePrefix(agentIDStr, id.PrefixForwardAgent); err != nil {
					h.logger.Warnw("invalid agent_id in chain_port_config in batch create", "agent_id", agentIDStr, "user_id", userID, "index", i, "error", err)
					preValidationFailed = append(preValidationFailed, dto.BatchOperationErr{
						ID:     identifier,
						Reason: "invalid agent_id in chain_port_config, expected fa_xxxxx",
					})
					chainPortValidationFailed = true
					break
				}
				chainPortConfig[agentIDStr] = port
			}
		}
		if chainPortValidationFailed {
			continue
		}

		var targetNodeSID string
		if r.TargetNodeID != "" {
			if err := id.ValidatePrefix(r.TargetNodeID, id.PrefixNode); err != nil {
				h.logger.Warnw("invalid target_node_id format in batch create", "target_node_id", r.TargetNodeID, "user_id", userID, "index", i, "error", err)
				preValidationFailed = append(preValidationFailed, dto.BatchOperationErr{
					ID:     identifier,
					Reason: "invalid target_node_id format, expected node_xxxxx",
				})
				continue
			}
			targetNodeSID = r.TargetNodeID
		}

		cmds = append(cmds, usecases.CreateUserForwardRuleCommand{
			UserID:             userID,
			AgentShortID:       r.AgentID,
			RuleType:           r.RuleType,
			ExitAgentShortID:   exitAgentShortID,
			ChainAgentShortIDs: chainAgentShortIDs,
			ChainPortConfig:    chainPortConfig,
			Name:               r.Name,
			ListenPort:         r.ListenPort,
			TargetAddress:      r.TargetAddress,
			TargetPort:         r.TargetPort,
			TargetNodeSID:      targetNodeSID,
			BindIP:             r.BindIP,
			IPVersion:          r.IPVersion,
			Protocol:           r.Protocol,
			TrafficMultiplier:  r.TrafficMultiplier,
			SortOrder:          r.SortOrder,
			Remark:             r.Remark,
		})
		cmdIndices = append(cmdIndices, i)
	}

	// If all rules failed pre-validation, return early with failures
	if len(cmds) == 0 {
		result := &dto.BatchCreateResponse{
			Succeeded: make([]dto.BatchCreateResult, 0),
			Failed:    preValidationFailed,
		}
		utils.SuccessResponse(c, http.StatusOK, "Batch create completed", result)
		return
	}

	ucResult, err := h.batchRuleUC.BatchCreateUser(c.Request.Context(), usecases.BatchCreateUserCommand{
		UserID:     userID,
		Rules:      cmds,
		CmdIndices: cmdIndices,
	})
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Merge pre-validation failures with UseCase result
	ucResult.Failed = append(preValidationFailed, ucResult.Failed...)

	utils.SuccessResponse(c, http.StatusOK, "Batch create completed", ucResult)
}

// BatchDeleteRules handles DELETE /user/forward-rules/batch
func (h *Handler) BatchDeleteRules(c *gin.Context) {
	userID, ok := h.getUserIDFromContext(c)
	if !ok {
		return
	}

	var req BatchDeleteForwardRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for batch delete rules", "user_id", userID, "error", err, "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.batchRuleUC.BatchDelete(c.Request.Context(), usecases.BatchDeleteCommand{
		RuleSIDs: req.RuleIDs,
		UserID:   &userID,
	})
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Batch delete completed", result)
}

// BatchToggleStatus handles PATCH /user/forward-rules/batch/status
func (h *Handler) BatchToggleStatus(c *gin.Context) {
	userID, ok := h.getUserIDFromContext(c)
	if !ok {
		return
	}

	var req BatchToggleStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for batch toggle status", "user_id", userID, "error", err, "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, err)
		return
	}

	enable := req.Status == "enabled"
	result, err := h.batchRuleUC.BatchToggleStatus(c.Request.Context(), usecases.BatchToggleStatusCommand{
		RuleSIDs: req.RuleIDs,
		Enable:   enable,
		UserID:   &userID,
	})
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Batch status update completed", result)
}

// BatchUpdateRules handles PATCH /user/forward-rules/batch
func (h *Handler) BatchUpdateRules(c *gin.Context) {
	userID, ok := h.getUserIDFromContext(c)
	if !ok {
		return
	}

	var req BatchUpdateForwardRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for batch update rules", "user_id", userID, "error", err, "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Convert to use case command
	updates := make([]usecases.BatchUpdateItem, 0, len(req.Updates))
	for _, u := range req.Updates {
		// Validate rule ID format
		if err := id.ValidatePrefix(u.RuleID, id.PrefixForwardRule); err != nil {
			h.logger.Warnw("invalid rule_id format in batch update", "rule_id", u.RuleID, "user_id", userID, "error", err)
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid rule_id format, expected fr_xxxxx"))
			return
		}

		// Validate agent_id format if provided
		if u.AgentID != nil && *u.AgentID != "" {
			if err := id.ValidatePrefix(*u.AgentID, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid agent_id format in batch update", "agent_id", *u.AgentID, "rule_id", u.RuleID, "user_id", userID, "error", err)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid agent_id format, expected fa_xxxxx"))
				return
			}
		}

		// Validate exit_agent_id format if provided
		if u.ExitAgentID != nil && *u.ExitAgentID != "" {
			if err := id.ValidatePrefix(*u.ExitAgentID, id.PrefixForwardAgent); err != nil {
				h.logger.Warnw("invalid exit_agent_id format in batch update", "exit_agent_id", *u.ExitAgentID, "rule_id", u.RuleID, "user_id", userID, "error", err)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid exit_agent_id format, expected fa_xxxxx"))
				return
			}
		}

		updates = append(updates, usecases.BatchUpdateItem{
			RuleSID:          u.RuleID,
			Name:             u.Name,
			Remark:           u.Remark,
			SortOrder:        u.SortOrder,
			AgentShortID:     u.AgentID,
			ExitAgentShortID: u.ExitAgentID,
		})
	}

	result, err := h.batchRuleUC.BatchUpdate(c.Request.Context(), usecases.BatchUpdateCommand{
		Updates: updates,
		UserID:  &userID,
	})
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Batch update completed", result)
}
