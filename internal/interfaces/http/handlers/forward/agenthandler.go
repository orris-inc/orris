// Package forward provides HTTP handlers for forward rule management.
package forward

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AgentHandler handles RESTful agent API requests for forward client
type AgentHandler struct {
	repo      forward.Repository
	agentRepo forward.AgentRepository
	logger    logger.Interface
}

// NewAgentHandler creates a new AgentHandler instance
func NewAgentHandler(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	logger logger.Interface,
) *AgentHandler {
	return &AgentHandler{
		repo:      repo,
		agentRepo: agentRepo,
		logger:    logger,
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

// GetExitEndpoint handles GET /forward-agent-api/exit-endpoint/:agent_id
func (h *AgentHandler) GetExitEndpoint(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse agent ID from path
	agentIDStr := c.Param("agent_id")
	id, err := strconv.ParseUint(agentIDStr, 10, 32)
	if err != nil {
		h.logger.Warnw("invalid agent_id parameter",
			"agent_id", agentIDStr,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid agent_id parameter")
		return
	}
	if id == 0 {
		h.logger.Warnw("invalid agent_id parameter",
			"agent_id", agentIDStr,
			"error", "agent ID must be greater than 0",
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "agent ID must be greater than 0")
		return
	}
	agentID := uint(id)

	h.logger.Infow("forward client requesting exit endpoint information",
		"agent_id", agentID,
		"ip", c.ClientIP(),
	)

	// Get agent by ID from agent repository
	agent, err := h.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		h.logger.Errorw("failed to get forward agent",
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to retrieve agent information")
		return
	}

	if agent == nil {
		h.logger.Warnw("forward agent not found",
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "forward agent not found")
		return
	}

	// Check if agent has a public address
	if agent.PublicAddress() == "" {
		h.logger.Warnw("agent has no public address configured",
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "agent has no public address configured")
		return
	}

	// Get exit rules for this agent
	exitRule, err := h.repo.GetExitRuleByAgentID(ctx, agentID)
	if err != nil {
		h.logger.Errorw("failed to get exit rule for agent",
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to retrieve exit rule")
		return
	}

	if exitRule == nil {
		h.logger.Warnw("no exit rule found for agent",
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "no exit rule found for this agent")
		return
	}

	h.logger.Infow("exit endpoint information retrieved successfully",
		"agent_id", agentID,
		"address", agent.PublicAddress(),
		"ws_port", exitRule.WsListenPort(),
		"ip", c.ClientIP(),
	)

	// Return the connection information
	utils.SuccessResponse(c, http.StatusOK, "exit endpoint information retrieved successfully", map[string]any{
		"address": agent.PublicAddress(),
		"ws_port": exitRule.WsListenPort(),
	})
}
