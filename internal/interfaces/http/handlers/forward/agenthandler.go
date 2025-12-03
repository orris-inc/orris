// Package forward provides HTTP handlers for forward rule management.
package forward

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"orris/internal/application/forward/dto"
	"orris/internal/domain/forward"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

// AgentHandler handles RESTful agent API requests for forward client
type AgentHandler struct {
	repo   forward.Repository
	logger logger.Interface
}

// NewAgentHandler creates a new AgentHandler instance
func NewAgentHandler(
	repo forward.Repository,
	logger logger.Interface,
) *AgentHandler {
	return &AgentHandler{
		repo:   repo,
		logger: logger,
	}
}

// ForwardRuleTrafficItem represents traffic data for a single forward rule
type ForwardRuleTrafficItem struct {
	RuleID        uint  `json:"rule_id" binding:"required"`
	UploadBytes   int64 `json:"upload_bytes" binding:"min=0"`
	DownloadBytes int64 `json:"download_bytes" binding:"min=0"`
}

// ReportTrafficRequest represents traffic report request from forward client
type ReportTrafficRequest struct {
	Rules []ForwardRuleTrafficItem `json:"rules" binding:"required,dive"`
}

func (h *AgentHandler) GetEnabledRules(c *gin.Context) {
	ctx := c.Request.Context()

	h.logger.Infow("forward client requesting enabled rules",
		"ip", c.ClientIP(),
	)

	// Retrieve all enabled forward rules
	rules, err := h.repo.ListEnabled(ctx)
	if err != nil {
		h.logger.Errorw("failed to retrieve enabled forward rules",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to retrieve enabled forward rules")
		return
	}

	h.logger.Infow("enabled forward rules retrieved successfully",
		"rule_count", len(rules),
		"ip", c.ClientIP(),
	)

	// Convert to DTOs
	ruleDTOs := dto.ToForwardRuleDTOs(rules)

	// Return success response
	utils.SuccessResponse(c, http.StatusOK, "enabled forward rules retrieved successfully", ruleDTOs)
}

func (h *AgentHandler) ReportTraffic(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse request body
	var req ReportTrafficRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid traffic report request body",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Infow("forward client traffic report received",
		"rule_count", len(req.Rules),
		"ip", c.ClientIP(),
	)

	// Update traffic for each rule
	successCount := 0
	errorCount := 0

	for _, item := range req.Rules {
		// Skip invalid traffic data
		if item.UploadBytes < 0 || item.DownloadBytes < 0 {
			h.logger.Warnw("invalid traffic data for rule",
				"rule_id", item.RuleID,
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

		err := h.repo.UpdateTraffic(ctx, item.RuleID, item.UploadBytes, item.DownloadBytes)
		if err != nil {
			h.logger.Errorw("failed to update rule traffic",
				"rule_id", item.RuleID,
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
		"ip", c.ClientIP(),
	)

	// Return success response with statistics
	utils.SuccessResponse(c, http.StatusOK, "traffic reported successfully", map[string]any{
		"rules_updated": successCount,
		"rules_failed":  errorCount,
	})
}
