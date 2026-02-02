// Package services provides application services for the forward domain.
package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// TunnelHealthHandler handles tunnel health reports from entry agents.
// When an entry agent detects that a tunnel to an exit agent is unhealthy,
// it reports to the server for logging and monitoring purposes.
type TunnelHealthHandler struct {
	logger logger.Interface
}

// NewTunnelHealthHandler creates a new TunnelHealthHandler.
func NewTunnelHealthHandler(log logger.Interface) *TunnelHealthHandler {
	return &TunnelHealthHandler{
		logger: log,
	}
}

// String returns the handler name for logging purposes.
func (h *TunnelHealthHandler) String() string {
	return "TunnelHealthHandler"
}

// HandleMessage processes tunnel health report messages from agents.
// Returns true if the message was handled, false otherwise.
func (h *TunnelHealthHandler) HandleMessage(agentID uint, msgType string, data any) bool {
	if msgType != dto.MsgTypeTunnelHealthReport {
		return false
	}

	// Parse the tunnel health report
	report, err := h.parseReport(data)
	if err != nil {
		h.logger.Warnw("failed to parse tunnel health report",
			"agent_id", agentID,
			"error", err,
		)
		return true // Message was handled (even if parsing failed)
	}

	// Log the health report
	if report.Healthy {
		h.logger.Debugw("tunnel health report: healthy",
			"reporting_agent_id", agentID,
			"rule_id", report.RuleID,
			"exit_agent_id", report.ExitAgentID,
			"latency_ms", report.LatencyMs,
			"checked_at", report.CheckedAt,
		)
	} else {
		h.logger.Warnw("tunnel health report: unhealthy",
			"reporting_agent_id", agentID,
			"rule_id", report.RuleID,
			"exit_agent_id", report.ExitAgentID,
			"fail_count", report.FailCount,
			"error", report.Error,
			"checked_at", report.CheckedAt,
		)
	}

	return true
}

// parseReport parses and validates the tunnel health report from the message data.
func (h *TunnelHealthHandler) parseReport(data any) (*dto.TunnelHealthReport, error) {
	// Data might be a map or already parsed struct
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var report dto.TunnelHealthReport
	if err := json.Unmarshal(jsonBytes, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report: %w", err)
	}

	// Validate required fields
	if report.RuleID == "" {
		return nil, fmt.Errorf("rule_id is required")
	}
	if !strings.HasPrefix(report.RuleID, "fr_") {
		return nil, fmt.Errorf("invalid rule_id format, expected fr_xxx")
	}

	if report.ExitAgentID == "" {
		return nil, fmt.Errorf("exit_agent_id is required")
	}
	if !strings.HasPrefix(report.ExitAgentID, "fa_") {
		return nil, fmt.Errorf("invalid exit_agent_id format, expected fa_xxx")
	}

	if report.CheckedAt <= 0 {
		return nil, fmt.Errorf("checked_at must be a positive timestamp")
	}

	// Validate fail_count is non-negative
	if report.FailCount < 0 {
		return nil, fmt.Errorf("fail_count cannot be negative")
	}

	return &report, nil
}
