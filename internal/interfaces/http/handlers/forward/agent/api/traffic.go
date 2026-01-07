// Package api provides HTTP handlers for forward agent REST API.
package api

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/utils"
)

// ForwardRuleTrafficItem represents traffic data for a single forward rule
type ForwardRuleTrafficItem struct {
	RuleID        string `json:"rule_id" binding:"required"` // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	UploadBytes   int64  `json:"upload_bytes" binding:"min=0"`
	DownloadBytes int64  `json:"download_bytes" binding:"min=0"`
}

// ReportTrafficRequest represents traffic report request from forward client
type ReportTrafficRequest struct {
	Rules []ForwardRuleTrafficItem `json:"rules" binding:"required,dive"`
}

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

	h.logger.Debugw("forward client traffic report received",
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

	// ruleInfo holds rule ID, subscription ID, and effective multiplier for traffic recording
	type ruleInfo struct {
		id                  uint
		subscriptionID      *uint
		effectiveMultiplier float64
	}

	// Merge all rules into validRuleIDs map (use rule.ID() to deduplicate)
	validRuleIDs := make(map[string]ruleInfo) // Stripe-style ID -> ruleInfo
	for _, rule := range agentRules {
		stripeID := rule.SID()
		validRuleIDs[stripeID] = ruleInfo{id: rule.ID(), subscriptionID: rule.SubscriptionID(), effectiveMultiplier: rule.GetEffectiveMultiplier()}
	}
	for _, rule := range exitRules {
		stripeID := rule.SID()
		validRuleIDs[stripeID] = ruleInfo{id: rule.ID(), subscriptionID: rule.SubscriptionID(), effectiveMultiplier: rule.GetEffectiveMultiplier()}
	}
	for _, rule := range chainRules {
		stripeID := rule.SID()
		validRuleIDs[stripeID] = ruleInfo{id: rule.ID(), subscriptionID: rule.SubscriptionID(), effectiveMultiplier: rule.GetEffectiveMultiplier()}
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

	// Maximum reasonable traffic per report: 10TB
	// This prevents overflow and detects potential malicious clients
	const maxReasonableTraffic = 10 * 1024 * 1024 * 1024 * 1024 // 10TB

	for _, item := range req.Rules {
		// Validate rule belongs to this agent and get internal ID
		info, valid := validRuleIDs[item.RuleID]
		if !valid {
			h.logger.Warnw("traffic report for unauthorized rule",
				"rule_id", item.RuleID,
				"agent_id", agentID,
				"ip", c.ClientIP(),
			)
			deniedCount++
			continue
		}

		// Validate traffic data range
		if item.UploadBytes < 0 || item.DownloadBytes < 0 {
			h.logger.Warnw("negative traffic data rejected",
				"rule_id", item.RuleID,
				"agent_id", agentID,
				"upload", item.UploadBytes,
				"download", item.DownloadBytes,
				"ip", c.ClientIP(),
			)
			errorCount++
			continue
		}

		// Check for suspiciously large values
		if item.UploadBytes > maxReasonableTraffic || item.DownloadBytes > maxReasonableTraffic {
			h.logger.Warnw("suspiciously large traffic data rejected",
				"rule_id", item.RuleID,
				"agent_id", agentID,
				"upload", item.UploadBytes,
				"download", item.DownloadBytes,
				"max_allowed", maxReasonableTraffic,
				"ip", c.ClientIP(),
			)
			errorCount++
			continue
		}

		// Skip if no traffic to report
		if item.UploadBytes == 0 && item.DownloadBytes == 0 {
			continue
		}

		// Update traffic in forward_rules table
		err := h.repo.UpdateTraffic(ctx, info.id, item.UploadBytes, item.DownloadBytes)
		if err != nil {
			h.logger.Errorw("failed to update rule traffic",
				"rule_id", item.RuleID,
				"internal_id", info.id,
				"agent_id", agentID,
				"upload", item.UploadBytes,
				"download", item.DownloadBytes,
				"error", err,
			)
			errorCount++
			continue
		}

		// Also record traffic to subscription_usages table (for unified traffic tracking)
		// Apply traffic multiplier before recording to subscription_usages
		// Only record if rule has a subscription (user rule); skip admin rules
		if h.trafficRecorder != nil && info.subscriptionID != nil {
			// Apply multiplier to get the effective traffic for billing/usage tracking
			// Use safe multiplication to prevent integer overflow
			effectiveUpload := safeMultiplyTraffic(item.UploadBytes, info.effectiveMultiplier)
			effectiveDownload := safeMultiplyTraffic(item.DownloadBytes, info.effectiveMultiplier)
			if err := h.trafficRecorder.RecordForwardTraffic(ctx, info.id, info.subscriptionID, effectiveUpload, effectiveDownload); err != nil {
				// Log warning but don't fail the request - forward_rules update already succeeded
				h.logger.Warnw("failed to record forward traffic to subscription_usages",
					"rule_id", item.RuleID,
					"internal_id", info.id,
					"subscription_id", *info.subscriptionID,
					"error", err,
				)
			}
		}

		successCount++
	}

	h.logger.Debugw("forward traffic report processed",
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

// safeMultiplyTraffic safely multiplies traffic bytes by a multiplier,
// capping at math.MaxInt64 to prevent integer overflow.
func safeMultiplyTraffic(bytes int64, multiplier float64) int64 {
	if bytes <= 0 || multiplier <= 0 {
		return 0
	}

	result := float64(bytes) * multiplier

	// Cap at MaxInt64 to prevent overflow when converting back to int64
	if result > float64(math.MaxInt64) {
		return math.MaxInt64
	}

	return int64(result)
}
