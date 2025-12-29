// Package api provides HTTP handlers for forward agent REST API.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// GetEnabledRules handles GET /forward-agent-api/rules
// Returns all enabled rules for the authenticated agent.
func (h *Handler) GetEnabledRules(c *gin.Context) {
	agentID, err := h.getAuthenticatedAgentID(c)
	if err != nil {
		h.logger.Warnw("failed to get authenticated agent ID",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "invalid agent authentication")
		return
	}

	result, err := h.getEnabledRulesUC.Execute(c.Request.Context(), usecases.GetEnabledRulesForAgentQuery{
		AgentID: agentID,
	})
	if err != nil {
		h.logger.Errorw("failed to get enabled rules",
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "enabled forward rules retrieved successfully", map[string]any{
		"rules":        result.Rules,
		"client_token": result.ClientToken,
	})
}

// RefreshRule handles GET /forward-agent-api/rules/:rule_id
// This endpoint allows an agent to refresh the configuration for a specific rule.
// It returns the latest next_hop_ws_port and other dynamic configuration.
// This is useful when the agent detects a connection failure to the next hop.
func (h *Handler) RefreshRule(c *gin.Context) {
	agentID, err := h.getAuthenticatedAgentID(c)
	if err != nil {
		h.logger.Warnw("failed to get authenticated agent ID",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "invalid agent authentication")
		return
	}

	ruleID := c.Param("rule_id")

	// Validate rule ID format
	if err := id.ValidatePrefix(ruleID, id.PrefixForwardRule); err != nil {
		h.logger.Warnw("invalid rule_id parameter",
			"rule_id", ruleID,
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid rule_id parameter: must be in format fr_xxx")
		return
	}

	result, err := h.refreshRuleUC.Execute(c.Request.Context(), usecases.RefreshRuleForAgentQuery{
		AgentID:     agentID,
		RuleShortID: ruleID,
	})
	if err != nil {
		h.logger.Errorw("failed to refresh rule",
			"agent_id", agentID,
			"rule_id", ruleID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("rule refresh successful",
		"rule_id", ruleID,
		"agent_id", agentID,
		"rule_type", result.Rule.RuleType,
		"next_hop_ws_port", result.Rule.NextHopWsPort,
		"next_hop_tls_port", result.Rule.NextHopTlsPort,
		"ip", c.ClientIP(),
	)

	utils.SuccessResponse(c, http.StatusOK, "rule refreshed successfully", result.Rule)
}
