package usecases

import (
	"context"
	"fmt"

	commondto "github.com/orris-inc/orris/internal/application/common/dto"
	"github.com/orris-inc/orris/internal/application/node/dto"
	apperrors "github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ReportNodeStatusCommand represents the command to report node system status
type ReportNodeStatusCommand struct {
	NodeID uint
	Status dto.ReportNodeStatusRequest
}

// ReportNodeStatusResult contains the result of status reporting
type ReportNodeStatusResult struct {
	Success bool
}

// NodeStatusUpdate contains all status update data for Redis storage.
// Embeds common SystemStatus for shared fields across all agent types.
type NodeStatusUpdate struct {
	commondto.SystemStatus
}

// NodeSystemStatusUpdater defines the interface for updating node system status in cache
type NodeSystemStatusUpdater interface {
	UpdateSystemStatus(ctx context.Context, nodeID uint, status *NodeStatusUpdate) error
}

// NodeLastSeenUpdater defines the interface for updating node last_seen_at, public IPs, and agent info
type NodeLastSeenUpdater interface {
	UpdateLastSeenAt(ctx context.Context, nodeID uint, publicIPv4, publicIPv6, agentVersion, platform, arch string) error
}

// NodePublicIPQuerier defines the interface for querying node public IPs
type NodePublicIPQuerier interface {
	GetPublicIPs(ctx context.Context, nodeID uint) (string, string, error)
}

// NodeAddressChangeNotifier defines the interface for notifying node address changes
type NodeAddressChangeNotifier interface {
	NotifyNodeAddressChange(ctx context.Context, nodeID uint) error
}

// ReportNodeStatusUseCase handles reporting node system status from node agents
type ReportNodeStatusUseCase struct {
	statusUpdater         NodeSystemStatusUpdater
	lastSeenUpdater       NodeLastSeenUpdater
	publicIPQuerier       NodePublicIPQuerier
	addressChangeNotifier NodeAddressChangeNotifier
	logger                logger.Interface
}

// NewReportNodeStatusUseCase creates a new instance of ReportNodeStatusUseCase
func NewReportNodeStatusUseCase(
	statusUpdater NodeSystemStatusUpdater,
	lastSeenUpdater NodeLastSeenUpdater,
	publicIPQuerier NodePublicIPQuerier,
	logger logger.Interface,
) *ReportNodeStatusUseCase {
	return &ReportNodeStatusUseCase{
		statusUpdater:   statusUpdater,
		lastSeenUpdater: lastSeenUpdater,
		publicIPQuerier: publicIPQuerier,
		logger:          logger,
	}
}

// SetAddressChangeNotifier sets the notifier for address changes.
// This is used to break circular dependencies during initialization.
func (uc *ReportNodeStatusUseCase) SetAddressChangeNotifier(notifier NodeAddressChangeNotifier) {
	uc.addressChangeNotifier = notifier
}

// Execute processes node status report from node agent
func (uc *ReportNodeStatusUseCase) Execute(ctx context.Context, cmd ReportNodeStatusCommand) (*ReportNodeStatusResult, error) {
	if cmd.NodeID == 0 {
		return nil, apperrors.NewValidationError("node_id is required")
	}

	// Build status update from request.
	// Both types embed commondto.SystemStatus, so direct assignment works.
	statusUpdate := &NodeStatusUpdate{
		SystemStatus: cmd.Status.SystemStatus,
	}

	// Update node system status in Redis (always)
	if err := uc.statusUpdater.UpdateSystemStatus(ctx, cmd.NodeID, statusUpdate); err != nil {
		uc.logger.Errorw("failed to update node system status",
			"error", err,
			"node_id", cmd.NodeID,
		)
		return nil, fmt.Errorf("failed to update node status")
	}

	// Check for IP changes and notify forward agents if changed
	uc.checkAndNotifyIPChange(ctx, cmd.NodeID, cmd.Status.PublicIPv4, cmd.Status.PublicIPv6)

	// Update last_seen_at, public IPs, and agent info in database (throttled to reduce DB writes)
	uc.updateLastSeenAtThrottled(ctx, cmd.NodeID, cmd.Status.PublicIPv4, cmd.Status.PublicIPv6, cmd.Status.AgentVersion, cmd.Status.Platform, cmd.Status.Arch)

	uc.logger.Debugw("node status reported",
		"node_id", cmd.NodeID,
		"cpu_percent", cmd.Status.CPUPercent,
		"memory_percent", cmd.Status.MemoryPercent,
	)

	return &ReportNodeStatusResult{
		Success: true,
	}, nil
}

// updateLastSeenAtThrottled updates last_seen_at, public IPs, and agent info
// Throttling is now handled at the database layer using conditional update
// to avoid race conditions when multiple requests arrive simultaneously
func (uc *ReportNodeStatusUseCase) updateLastSeenAtThrottled(ctx context.Context, nodeID uint, publicIPv4, publicIPv6, agentVersion, platform, arch string) {
	if uc.lastSeenUpdater == nil {
		return
	}

	// Database layer handles throttling with conditional update:
	// only updates if last_seen_at is NULL or older than 2 minutes
	if err := uc.lastSeenUpdater.UpdateLastSeenAt(ctx, nodeID, publicIPv4, publicIPv6, agentVersion, platform, arch); err != nil {
		uc.logger.Warnw("failed to update last_seen_at",
			"error", err,
			"node_id", nodeID,
		)
	}
}

// checkAndNotifyIPChange checks if the node's public IP has changed and notifies forward agents
func (uc *ReportNodeStatusUseCase) checkAndNotifyIPChange(ctx context.Context, nodeID uint, newIPv4, newIPv6 string) {
	if uc.publicIPQuerier == nil || uc.addressChangeNotifier == nil {
		return
	}

	// Skip if no new IPs are reported
	if newIPv4 == "" && newIPv6 == "" {
		return
	}

	// Get current IPs from database
	currentIPv4, currentIPv6, err := uc.publicIPQuerier.GetPublicIPs(ctx, nodeID)
	if err != nil {
		uc.logger.Warnw("failed to get current public IPs for change detection",
			"error", err,
			"node_id", nodeID,
		)
		return
	}

	// Check if either IP has changed
	ipv4Changed := newIPv4 != "" && currentIPv4 != "" && newIPv4 != currentIPv4
	ipv6Changed := newIPv6 != "" && currentIPv6 != "" && newIPv6 != currentIPv6

	// Also detect when IP is set for the first time (from empty to non-empty)
	// but only notify if there was a previous value (actual change)
	if !ipv4Changed && !ipv6Changed {
		return
	}

	uc.logger.Infow("node public IP changed, notifying forward agents",
		"node_id", nodeID,
		"old_ipv4", currentIPv4,
		"new_ipv4", newIPv4,
		"old_ipv6", currentIPv6,
		"new_ipv6", newIPv6,
	)

	// Notify forward agents asynchronously to avoid blocking status reporting
	// Use background context since this goroutine outlives the HTTP request
	goroutine.SafeGo(uc.logger, "report-node-status-notify-ip-change", func() {
		notifyCtx := context.Background()
		if err := uc.addressChangeNotifier.NotifyNodeAddressChange(notifyCtx, nodeID); err != nil {
			uc.logger.Warnw("failed to notify forward agents of node IP change",
				"error", err,
				"node_id", nodeID,
			)
		}
	})
}
