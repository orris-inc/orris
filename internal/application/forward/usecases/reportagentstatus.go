// Package usecases contains the application use cases for forward domain.
package usecases

import (
	"context"
	"fmt"
	"sync"
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

// AgentInfoUpdater defines the interface for updating agent info (version, platform, arch).
type AgentInfoUpdater interface {
	UpdateAgentInfo(ctx context.Context, agentID uint, agentVersion, platform, arch string) error
}

// ExitPortChangeNotifier defines the interface for notifying entry agents when exit agent's port changes.
type ExitPortChangeNotifier interface {
	// NotifyExitPortChange notifies all entry agents that have rules pointing to this exit agent.
	NotifyExitPortChange(ctx context.Context, exitAgentID uint) error
}

// ReportAgentStatusUseCase handles agent status reporting.
type ReportAgentStatusUseCase struct {
	agentRepo          forward.AgentRepository
	statusUpdater      AgentStatusUpdater
	statusQuerier      AgentStatusQuerier
	lastSeenUpdater    AgentLastSeenUpdater
	agentInfoUpdater   AgentInfoUpdater
	portChangeNotifier ExitPortChangeNotifier
	logger             logger.Interface

	// Rate limiting for last_seen_at updates
	lastSeenInterval time.Duration
	lastSeenMu       sync.RWMutex
	lastSeenCache    map[uint]time.Time
}

// NewReportAgentStatusUseCase creates a new ReportAgentStatusUseCase.
func NewReportAgentStatusUseCase(
	agentRepo forward.AgentRepository,
	statusUpdater AgentStatusUpdater,
	statusQuerier AgentStatusQuerier,
	lastSeenUpdater AgentLastSeenUpdater,
	agentInfoUpdater AgentInfoUpdater,
	logger logger.Interface,
) *ReportAgentStatusUseCase {
	return &ReportAgentStatusUseCase{
		agentRepo:        agentRepo,
		statusUpdater:    statusUpdater,
		statusQuerier:    statusQuerier,
		lastSeenUpdater:  lastSeenUpdater,
		agentInfoUpdater: agentInfoUpdater,
		logger:           logger,
		lastSeenInterval: 2 * time.Minute,
		lastSeenCache:    make(map[uint]time.Time),
	}
}

// SetPortChangeNotifier sets the port change notifier (used for late binding due to circular dependency).
func (uc *ReportAgentStatusUseCase) SetPortChangeNotifier(notifier ExitPortChangeNotifier) {
	uc.portChangeNotifier = notifier
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

	// Get old status from Redis for change detection
	var oldStatus *dto.AgentStatusDTO
	if uc.statusQuerier != nil {
		oldStatus, err = uc.statusQuerier.GetStatus(ctx, input.AgentID)
		if err != nil {
			uc.logger.Warnw("failed to get old status for change detection",
				"agent_id", input.AgentID,
				"error", err,
			)
		}
	}

	// Update status in Redis
	if err := uc.statusUpdater.UpdateStatus(ctx, input.AgentID, input.Status); err != nil {
		uc.logger.Errorw("failed to update agent status", "agent_id", input.AgentID, "error", err)
		return fmt.Errorf("update status: %w", err)
	}

	// Notify entry agents if tunnel ports have changed
	uc.handlePortChange(ctx, input.AgentID, oldStatus, input.Status)

	// Update agent info immediately if changed (no rate limiting for version updates)
	uc.handleAgentInfoChange(ctx, input.AgentID, oldStatus, input.Status)

	// Update last_seen_at with rate limiting (avoid DB writes on every status report)
	if uc.shouldUpdateLastSeen(input.AgentID) {
		if uc.lastSeenUpdater != nil {
			if err := uc.lastSeenUpdater.UpdateLastSeen(ctx, input.AgentID); err != nil {
				uc.logger.Warnw("failed to update last_seen_at", "agent_id", input.AgentID, "error", err)
			} else {
				uc.lastSeenMu.Lock()
				uc.lastSeenCache[input.AgentID] = time.Now()
				uc.lastSeenMu.Unlock()
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

// handlePortChange notifies entry agents if tunnel ports have changed.
func (uc *ReportAgentStatusUseCase) handlePortChange(ctx context.Context, agentID uint, oldStatus, newStatus *dto.AgentStatusDTO) {
	if uc.portChangeNotifier == nil {
		return
	}

	var oldWsPort, oldTlsPort uint16
	if oldStatus != nil {
		oldWsPort = oldStatus.WsListenPort
		oldTlsPort = oldStatus.TlsListenPort
	}

	wsPortChanged := newStatus.WsListenPort > 0 && oldWsPort > 0 && oldWsPort != newStatus.WsListenPort
	tlsPortChanged := newStatus.TlsListenPort > 0 && oldTlsPort > 0 && oldTlsPort != newStatus.TlsListenPort

	if wsPortChanged || tlsPortChanged {
		uc.logger.Infow("exit agent tunnel port changed, notifying entry agents",
			"agent_id", agentID,
			"old_ws_port", oldWsPort,
			"new_ws_port", newStatus.WsListenPort,
			"old_tls_port", oldTlsPort,
			"new_tls_port", newStatus.TlsListenPort,
		)
		if err := uc.portChangeNotifier.NotifyExitPortChange(ctx, agentID); err != nil {
			uc.logger.Infow("port change notification skipped",
				"agent_id", agentID,
				"reason", err.Error(),
			)
		}
	}
}

// handleAgentInfoChange updates agent info in DB immediately if changed.
func (uc *ReportAgentStatusUseCase) handleAgentInfoChange(ctx context.Context, agentID uint, oldStatus, newStatus *dto.AgentStatusDTO) {
	if uc.agentInfoUpdater == nil {
		return
	}

	// Check if agent info has changed
	var oldVersion, oldPlatform, oldArch string
	if oldStatus != nil {
		oldVersion = oldStatus.AgentVersion
		oldPlatform = oldStatus.Platform
		oldArch = oldStatus.Arch
	}

	versionChanged := newStatus.AgentVersion != "" && newStatus.AgentVersion != oldVersion
	platformChanged := newStatus.Platform != "" && newStatus.Platform != oldPlatform
	archChanged := newStatus.Arch != "" && newStatus.Arch != oldArch

	if versionChanged || platformChanged || archChanged {
		uc.logger.Infow("agent info changed, updating immediately",
			"agent_id", agentID,
			"old_version", oldVersion,
			"new_version", newStatus.AgentVersion,
			"old_platform", oldPlatform,
			"new_platform", newStatus.Platform,
			"old_arch", oldArch,
			"new_arch", newStatus.Arch,
		)
		if err := uc.agentInfoUpdater.UpdateAgentInfo(ctx, agentID, newStatus.AgentVersion, newStatus.Platform, newStatus.Arch); err != nil {
			uc.logger.Warnw("failed to update agent info", "agent_id", agentID, "error", err)
		}
	}
}

// shouldUpdateLastSeen checks if we should update last_seen_at based on rate limiting.
func (uc *ReportAgentStatusUseCase) shouldUpdateLastSeen(agentID uint) bool {
	uc.lastSeenMu.RLock()
	lastUpdate, exists := uc.lastSeenCache[agentID]
	uc.lastSeenMu.RUnlock()

	if !exists {
		return true
	}
	return time.Since(lastUpdate) >= uc.lastSeenInterval
}
