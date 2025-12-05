// Package usecases contains the application use cases for forward domain.
package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AgentStatusUpdater defines the interface for updating agent status in cache.
type AgentStatusUpdater interface {
	UpdateStatus(ctx context.Context, agentID uint, status *dto.AgentStatusDTO) error
}

// AgentLastSeenUpdater defines the interface for updating agent last seen time.
type AgentLastSeenUpdater interface {
	UpdateLastSeen(ctx context.Context, agentID uint) error
}

// ReportAgentStatusUseCase handles agent status reporting.
type ReportAgentStatusUseCase struct {
	agentRepo       forward.AgentRepository
	statusUpdater   AgentStatusUpdater
	lastSeenUpdater AgentLastSeenUpdater
	logger          logger.Interface

	// Rate limiting for last_seen_at updates
	lastSeenInterval time.Duration
	lastSeenCache    map[uint]time.Time
}

// NewReportAgentStatusUseCase creates a new ReportAgentStatusUseCase.
func NewReportAgentStatusUseCase(
	agentRepo forward.AgentRepository,
	statusUpdater AgentStatusUpdater,
	lastSeenUpdater AgentLastSeenUpdater,
	logger logger.Interface,
) *ReportAgentStatusUseCase {
	return &ReportAgentStatusUseCase{
		agentRepo:        agentRepo,
		statusUpdater:    statusUpdater,
		lastSeenUpdater:  lastSeenUpdater,
		logger:           logger,
		lastSeenInterval: 2 * time.Minute,
		lastSeenCache:    make(map[uint]time.Time),
	}
}

// Execute reports agent status.
func (uc *ReportAgentStatusUseCase) Execute(ctx context.Context, input *dto.ReportAgentStatusInput) error {
	// Verify agent exists
	agent, err := uc.agentRepo.GetByID(ctx, input.AgentID)
	if err != nil {
		uc.logger.Errorw("failed to get agent", "agent_id", input.AgentID, "error", err)
		return fmt.Errorf("get agent: %w", err)
	}
	if agent == nil {
		return fmt.Errorf("agent not found: %d", input.AgentID)
	}

	// Update status in Redis
	if err := uc.statusUpdater.UpdateStatus(ctx, input.AgentID, input.Status); err != nil {
		uc.logger.Errorw("failed to update agent status", "agent_id", input.AgentID, "error", err)
		return fmt.Errorf("update status: %w", err)
	}

	// Update last_seen_at with rate limiting (avoid DB writes on every status report)
	if uc.shouldUpdateLastSeen(input.AgentID) {
		if uc.lastSeenUpdater != nil {
			if err := uc.lastSeenUpdater.UpdateLastSeen(ctx, input.AgentID); err != nil {
				uc.logger.Warnw("failed to update last_seen_at", "agent_id", input.AgentID, "error", err)
				// Don't return error, status update was successful
			} else {
				uc.lastSeenCache[input.AgentID] = time.Now()
			}
		}
	}

	uc.logger.Debugw("agent status reported",
		"agent_id", input.AgentID,
		"cpu", input.Status.CPUPercent,
		"memory", input.Status.MemoryPercent,
		"active_rules", input.Status.ActiveRules,
	)

	return nil
}

// shouldUpdateLastSeen checks if we should update last_seen_at based on rate limiting.
func (uc *ReportAgentStatusUseCase) shouldUpdateLastSeen(agentID uint) bool {
	lastUpdate, exists := uc.lastSeenCache[agentID]
	if !exists {
		return true
	}
	return time.Since(lastUpdate) >= uc.lastSeenInterval
}
