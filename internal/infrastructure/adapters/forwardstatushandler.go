// Package adapters provides infrastructure adapters.
package adapters

import (
	"context"
	"encoding/json"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ForwardStatusHandler handles status updates from forward agents.
type ForwardStatusHandler struct {
	statusAdapter *ForwardAgentStatusAdapter
	logger        logger.Interface
}

// NewForwardStatusHandler creates a new ForwardStatusHandler.
func NewForwardStatusHandler(
	statusAdapter *ForwardAgentStatusAdapter,
	log logger.Interface,
) *ForwardStatusHandler {
	return &ForwardStatusHandler{
		statusAdapter: statusAdapter,
		logger:        log,
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

	// Persist status to Redis
	ctx := context.Background()
	if err := h.statusAdapter.UpdateStatus(ctx, agentID, &status); err != nil {
		h.logger.Errorw("failed to update forward agent status",
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
