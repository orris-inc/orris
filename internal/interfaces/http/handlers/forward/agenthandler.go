// Package forward provides HTTP handlers for forward rule management.
package forward

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AgentHandler handles RESTful agent API requests for forward client
type AgentHandler struct {
	repo           forward.Repository
	agentRepo      forward.AgentRepository
	nodeRepo       node.NodeRepository
	reportStatusUC *usecases.ReportAgentStatusUseCase
	logger         logger.Interface
}

// NewAgentHandler creates a new AgentHandler instance
func NewAgentHandler(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	reportStatusUC *usecases.ReportAgentStatusUseCase,
	logger logger.Interface,
) *AgentHandler {
	return &AgentHandler{
		repo:           repo,
		agentRepo:      agentRepo,
		nodeRepo:       nodeRepo,
		reportStatusUC: reportStatusUC,
		logger:         logger,
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

	// Convert to DTOs and resolve dynamic node addresses
	ruleDTOs := dto.ToForwardRuleDTOs(rules)

	// Resolve node addresses for rules with targetNodeID
	for _, ruleDTO := range ruleDTOs {
		if ruleDTO.TargetNodeID != nil && *ruleDTO.TargetNodeID != 0 {
			// Fetch node information
			node, err := h.nodeRepo.GetByID(ctx, *ruleDTO.TargetNodeID)
			if err != nil {
				h.logger.Warnw("failed to get target node for rule",
					"rule_id", ruleDTO.ID,
					"node_id", *ruleDTO.TargetNodeID,
					"error", err,
				)
				// Keep original values if node fetch fails
				continue
			}
			if node == nil {
				h.logger.Warnw("target node not found for rule",
					"rule_id", ruleDTO.ID,
					"node_id", *ruleDTO.TargetNodeID,
				)
				// Keep original values if node not found
				continue
			}

			// Dynamically populate target address and port from node
			ruleDTO.TargetAddress = node.ServerAddress().Value()
			ruleDTO.TargetPort = node.AgentPort()

			h.logger.Debugw("resolved target node address for rule",
				"rule_id", ruleDTO.ID,
				"node_id", *ruleDTO.TargetNodeID,
				"target_address", ruleDTO.TargetAddress,
				"target_port", ruleDTO.TargetPort,
			)
		}
	}

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

// ReportStatusRequest represents status report request from forward client
type ReportStatusRequest struct {
	CPUPercent        float64         `json:"cpu_percent"`
	MemoryPercent     float64         `json:"memory_percent"`
	MemoryUsed        uint64          `json:"memory_used"`
	MemoryTotal       uint64          `json:"memory_total"`
	DiskPercent       float64         `json:"disk_percent"`
	DiskUsed          uint64          `json:"disk_used"`
	DiskTotal         uint64          `json:"disk_total"`
	UptimeSeconds     int64           `json:"uptime_seconds"`
	TCPConnections    int             `json:"tcp_connections"`
	UDPConnections    int             `json:"udp_connections"`
	ActiveRules       int             `json:"active_rules"`
	ActiveConnections int             `json:"active_connections"`
	TunnelStatus      map[uint]string `json:"tunnel_status,omitempty"`
}

// ReportStatus handles POST /forward-agent-api/status
func (h *AgentHandler) ReportStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Get agent ID from context (set by auth middleware)
	agentID, exists := c.Get("forward_agent_id")
	if !exists {
		h.logger.Warnw("forward_agent_id not found in context",
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request body
	var req ReportStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid status report request body",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Debugw("forward agent status report received",
		"agent_id", agentID,
		"cpu", req.CPUPercent,
		"memory", req.MemoryPercent,
		"active_rules", req.ActiveRules,
		"ip", c.ClientIP(),
	)

	// Convert request to DTO
	statusDTO := &dto.AgentStatusDTO{
		CPUPercent:        req.CPUPercent,
		MemoryPercent:     req.MemoryPercent,
		MemoryUsed:        req.MemoryUsed,
		MemoryTotal:       req.MemoryTotal,
		DiskPercent:       req.DiskPercent,
		DiskUsed:          req.DiskUsed,
		DiskTotal:         req.DiskTotal,
		UptimeSeconds:     req.UptimeSeconds,
		TCPConnections:    req.TCPConnections,
		UDPConnections:    req.UDPConnections,
		ActiveRules:       req.ActiveRules,
		ActiveConnections: req.ActiveConnections,
		TunnelStatus:      req.TunnelStatus,
	}

	// Execute use case
	input := &dto.ReportAgentStatusInput{
		AgentID: agentID.(uint),
		Status:  statusDTO,
	}

	if err := h.reportStatusUC.Execute(ctx, input); err != nil {
		h.logger.Errorw("failed to report agent status",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to report status")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "status reported successfully", nil)
}
