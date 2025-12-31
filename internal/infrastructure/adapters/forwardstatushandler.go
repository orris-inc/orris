// Package adapters provides infrastructure adapters.
package adapters

import (
	"context"
	"encoding/json"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ForwardStatusHandler handles status updates from forward agents.
type ForwardStatusHandler struct {
	reportStatusUC *usecases.ReportAgentStatusUseCase
	logger         logger.Interface
}

// NewForwardStatusHandler creates a new ForwardStatusHandler.
func NewForwardStatusHandler(
	reportStatusUC *usecases.ReportAgentStatusUseCase,
	log logger.Interface,
) *ForwardStatusHandler {
	return &ForwardStatusHandler{
		reportStatusUC: reportStatusUC,
		logger:         log,
	}
}

// HandleStatus processes status update from a forward agent.
func (h *ForwardStatusHandler) HandleStatus(agentID uint, data any) {
	// Parse data to ForwardAgentStatusDTO
	dataBytes, err := json.Marshal(data)
	if err != nil {
		h.logger.Warnw("failed to marshal forward status data",
			"error", err,
			"agent_id", agentID,
		)
		return
	}

	var status dto.AgentStatusDTO
	if err := json.Unmarshal(dataBytes, &status); err != nil {
		h.logger.Warnw("failed to parse forward status data",
			"error", err,
			"agent_id", agentID,
		)
		return
	}

	// Use the same use case as HTTP handler to ensure consistent behavior
	// This will update both Redis status and DB last_seen_at/agent_info
	ctx := context.Background()
	input := &dto.ReportAgentStatusInput{
		AgentID: agentID,
		Status:  &status,
	}
	if err := h.reportStatusUC.Execute(ctx, input); err != nil {
		h.logger.Errorw("failed to report forward agent status",
			"error", err,
			"agent_id", agentID,
		)
		return
	}

	h.logger.Debugw("forward agent status updated via websocket",
		"agent_id", agentID,
		"cpu", status.CPUPercent,
		"memory", status.MemoryPercent,
		"active_rules", status.ActiveRules,
	)
}

// Ensure ForwardStatusHandler implements StatusHandler interface.
var _ services.StatusHandler = (*ForwardStatusHandler)(nil)
