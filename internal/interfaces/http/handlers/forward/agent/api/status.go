// Package api provides HTTP handlers for forward agent REST API.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ReportStatusRequest represents status report request from forward client
type ReportStatusRequest struct {
	CPUPercent        float64           `json:"cpu_percent"`
	MemoryPercent     float64           `json:"memory_percent"`
	MemoryUsed        uint64            `json:"memory_used"`
	MemoryTotal       uint64            `json:"memory_total"`
	DiskPercent       float64           `json:"disk_percent"`
	DiskUsed          uint64            `json:"disk_used"`
	DiskTotal         uint64            `json:"disk_total"`
	UptimeSeconds     int64             `json:"uptime_seconds"`
	TCPConnections    int               `json:"tcp_connections"`
	UDPConnections    int               `json:"udp_connections"`
	ActiveRules       int               `json:"active_rules"`
	ActiveConnections int               `json:"active_connections"`
	TunnelStatus      map[string]string `json:"tunnel_status,omitempty"`   // Key is Stripe-style rule ID (e.g., "fr_xK9mP2vL3nQ")
	WsListenPort      uint16            `json:"ws_listen_port,omitempty"`  // WebSocket listen port for exit agent tunnel connections
	TlsListenPort     uint16            `json:"tls_listen_port,omitempty"` // TLS listen port for exit agent tunnel connections
	AgentVersion      string            `json:"agent_version,omitempty"`   // Agent software version (e.g., "1.2.3")
	Platform          string            `json:"platform,omitempty"`        // OS platform (linux, darwin, windows)
	Arch              string            `json:"arch,omitempty"`            // CPU architecture (amd64, arm64, arm, 386)
}

// ReportRuleSyncStatusRequest represents rule sync status report request from forward client
type ReportRuleSyncStatusRequest struct {
	Rules []dto.RuleSyncStatusItem `json:"rules" binding:"required,dive"`
}

// ReportStatus handles POST /forward-agent-api/status
func (h *Handler) ReportStatus(c *gin.Context) {
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
		WsListenPort:      req.WsListenPort,
		TlsListenPort:     req.TlsListenPort,
		AgentVersion:      req.AgentVersion,
		Platform:          req.Platform,
		Arch:              req.Arch,
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

// ReportRuleSyncStatus handles POST /forward-agent-api/rule-sync-status
func (h *Handler) ReportRuleSyncStatus(c *gin.Context) {
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
	var req ReportRuleSyncStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid rule sync status report request body",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Debugw("forward agent rule sync status report received",
		"agent_id", agentID,
		"rules_count", len(req.Rules),
		"ip", c.ClientIP(),
	)

	// Execute use case
	input := &dto.ReportRuleSyncStatusInput{
		AgentID: agentID,
		Rules:   req.Rules,
	}

	if err := h.reportRuleSyncStatusUC.Execute(ctx, input); err != nil {
		h.logger.Errorw("failed to report rule sync status",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to report rule sync status")
		return
	}

	h.logger.Infow("rule sync status reported successfully",
		"agent_id", agentID,
		"rules_count", len(req.Rules),
		"ip", c.ClientIP(),
	)

	// All rules were successfully stored since use case succeeded
	utils.SuccessResponse(c, http.StatusOK, "rule sync status reported successfully", map[string]any{
		"rules_updated": len(req.Rules),
	})
}
