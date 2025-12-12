package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ReportTraffic handles POST /forward-agent-api/traffic
func (h *Handler) ReportTraffic(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated agent ID from context
	agentID, err := h.getAuthenticatedAgentID(c)
	if err != nil {
		h.logger.Warnw("failed to get authenticated agent ID",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request body
	var req ReportTrafficRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid traffic report request body",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Infow("forward client traffic report received",
		"rule_count", len(req.Rules),
		"agent_id", agentID,
		"ip", c.ClientIP(),
	)

	// Build a map of valid rule IDs for this agent (Stripe-style ID -> internal uint ID)
	// Include rules where this agent is the entry agent
	agentRules, err := h.repo.ListByAgentID(ctx, agentID)
	if err != nil {
		h.logger.Errorw("failed to get agent rules for validation",
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to validate rules")
		return
	}

	// Also include rules where this agent is the exit agent
	exitRules, err := h.repo.ListByExitAgentID(ctx, agentID)
	if err != nil {
		h.logger.Errorw("failed to get exit rules for validation",
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to validate rules")
		return
	}

	// Also include chain rules where this agent participates
	chainRules, err := h.repo.ListEnabledByChainAgentID(ctx, agentID)
	if err != nil {
		h.logger.Errorw("failed to get chain rules for validation",
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to validate rules")
		return
	}

	// Merge all rules into validRuleIDs map (use rule.ID() to deduplicate)
	validRuleIDs := make(map[string]uint) // Stripe-style ID -> internal uint ID
	for _, rule := range agentRules {
		stripeID := id.FormatForwardRuleID(rule.ShortID())
		validRuleIDs[stripeID] = rule.ID()
	}
	for _, rule := range exitRules {
		stripeID := id.FormatForwardRuleID(rule.ShortID())
		validRuleIDs[stripeID] = rule.ID()
	}
	for _, rule := range chainRules {
		stripeID := id.FormatForwardRuleID(rule.ShortID())
		validRuleIDs[stripeID] = rule.ID()
	}

	h.logger.Debugw("validated rule sources for traffic report",
		"agent_id", agentID,
		"entry_rules", len(agentRules),
		"exit_rules", len(exitRules),
		"chain_rules", len(chainRules),
		"total_valid_rules", len(validRuleIDs),
	)

	// Update traffic for each rule
	successCount := 0
	errorCount := 0
	deniedCount := 0

	for _, item := range req.Rules {
		// Validate rule belongs to this agent and get internal ID
		internalID, valid := validRuleIDs[item.RuleID]
		if !valid {
			h.logger.Warnw("traffic report for unauthorized rule",
				"rule_id", item.RuleID,
				"agent_id", agentID,
				"ip", c.ClientIP(),
			)
			deniedCount++
			continue
		}

		// Skip invalid traffic data
		if item.UploadBytes < 0 || item.DownloadBytes < 0 {
			h.logger.Warnw("invalid traffic data for rule",
				"rule_id", item.RuleID,
				"agent_id", agentID,
				"upload", item.UploadBytes,
				"download", item.DownloadBytes,
			)
			errorCount++
			continue
		}

		// Skip if no traffic to report
		if item.UploadBytes == 0 && item.DownloadBytes == 0 {
			continue
		}

		err := h.repo.UpdateTraffic(ctx, internalID, item.UploadBytes, item.DownloadBytes)
		if err != nil {
			h.logger.Errorw("failed to update rule traffic",
				"rule_id", item.RuleID,
				"internal_id", internalID,
				"agent_id", agentID,
				"upload", item.UploadBytes,
				"download", item.DownloadBytes,
				"error", err,
			)
			errorCount++
			continue
		}

		successCount++
	}

	h.logger.Infow("forward traffic report processed",
		"success_count", successCount,
		"error_count", errorCount,
		"denied_count", deniedCount,
		"agent_id", agentID,
		"ip", c.ClientIP(),
	)

	// Return success response with statistics
	utils.SuccessResponse(c, http.StatusOK, "traffic reported successfully", map[string]any{
		"rules_updated": successCount,
		"rules_failed":  errorCount,
		"rules_denied":  deniedCount,
	})
}
