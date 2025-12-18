package usecases

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// LastSeenUpdateThreshold defines how often we update last_seen_at in database
// This is used to throttle database writes when agents report frequently
const LastSeenUpdateThreshold = 2 * time.Minute

// ReportNodeStatusCommand represents the command to report node system status
type ReportNodeStatusCommand struct {
	NodeID uint
	Status dto.ReportNodeStatusRequest
}

// ReportNodeStatusResult contains the result of status reporting
type ReportNodeStatusResult struct {
	Success bool
}

// NodeSystemStatusUpdater defines the interface for updating node system status
type NodeSystemStatusUpdater interface {
	UpdateSystemStatus(ctx context.Context, nodeID uint, cpu, memory, disk float64, uptime int, publicIPv4, publicIPv6 string) error
}

// NodeLastSeenUpdater defines the interface for updating node last_seen_at and public IPs
type NodeLastSeenUpdater interface {
	GetLastSeenAt(ctx context.Context, nodeID uint) (*time.Time, error)
	UpdateLastSeenAt(ctx context.Context, nodeID uint, publicIPv4, publicIPv6 string) error
}

// ReportNodeStatusUseCase handles reporting node system status from node agents
type ReportNodeStatusUseCase struct {
	statusUpdater   NodeSystemStatusUpdater
	lastSeenUpdater NodeLastSeenUpdater
	logger          logger.Interface
}

// NewReportNodeStatusUseCase creates a new instance of ReportNodeStatusUseCase
func NewReportNodeStatusUseCase(
	statusUpdater NodeSystemStatusUpdater,
	lastSeenUpdater NodeLastSeenUpdater,
	logger logger.Interface,
) *ReportNodeStatusUseCase {
	return &ReportNodeStatusUseCase{
		statusUpdater:   statusUpdater,
		lastSeenUpdater: lastSeenUpdater,
		logger:          logger,
	}
}

// Execute processes node status report from node agent
func (uc *ReportNodeStatusUseCase) Execute(ctx context.Context, cmd ReportNodeStatusCommand) (*ReportNodeStatusResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	// Parse percentage strings (e.g., "45%" -> 0.45)
	cpu := parsePercentage(cmd.Status.CPU)
	memory := parsePercentage(cmd.Status.Mem)
	disk := parsePercentage(cmd.Status.Disk)

	// Update node system status in Redis (always)
	if err := uc.statusUpdater.UpdateSystemStatus(
		ctx,
		cmd.NodeID,
		cpu,
		memory,
		disk,
		cmd.Status.Uptime,
		cmd.Status.PublicIPv4,
		cmd.Status.PublicIPv6,
	); err != nil {
		uc.logger.Errorw("failed to update node system status",
			"error", err,
			"node_id", cmd.NodeID,
		)
		return nil, fmt.Errorf("failed to update node status")
	}

	// Update last_seen_at and public IPs in database (throttled to reduce DB writes)
	uc.updateLastSeenAtThrottled(ctx, cmd.NodeID, cmd.Status.PublicIPv4, cmd.Status.PublicIPv6)

	uc.logger.Infow("node status reported successfully",
		"node_id", cmd.NodeID,
		"cpu", cmd.Status.CPU,
		"memory", cmd.Status.Mem,
		"disk", cmd.Status.Disk,
		"uptime", cmd.Status.Uptime,
		"public_ipv4", cmd.Status.PublicIPv4,
		"public_ipv6", cmd.Status.PublicIPv6,
	)

	return &ReportNodeStatusResult{
		Success: true,
	}, nil
}

// updateLastSeenAtThrottled updates last_seen_at and public IPs only if it hasn't been updated recently
// This reduces database writes when agents report frequently (e.g., every 30 seconds)
func (uc *ReportNodeStatusUseCase) updateLastSeenAtThrottled(ctx context.Context, nodeID uint, publicIPv4, publicIPv6 string) {
	if uc.lastSeenUpdater == nil {
		return
	}

	// Get current last_seen_at value
	lastSeenAt, err := uc.lastSeenUpdater.GetLastSeenAt(ctx, nodeID)
	if err != nil {
		uc.logger.Warnw("failed to get last_seen_at for throttle check",
			"error", err,
			"node_id", nodeID,
		)
		// On error, try to update anyway
		if updateErr := uc.lastSeenUpdater.UpdateLastSeenAt(ctx, nodeID, publicIPv4, publicIPv6); updateErr != nil {
			uc.logger.Warnw("failed to update last_seen_at",
				"error", updateErr,
				"node_id", nodeID,
			)
		}
		return
	}

	// Check if we need to update (first time or exceeded threshold)
	shouldUpdate := lastSeenAt == nil || time.Since(*lastSeenAt) > LastSeenUpdateThreshold

	if shouldUpdate {
		if err := uc.lastSeenUpdater.UpdateLastSeenAt(ctx, nodeID, publicIPv4, publicIPv6); err != nil {
			uc.logger.Warnw("failed to update last_seen_at",
				"error", err,
				"node_id", nodeID,
			)
		} else {
			uc.logger.Debugw("last_seen_at updated",
				"node_id", nodeID,
				"public_ipv4", publicIPv4,
				"public_ipv6", publicIPv6,
			)
		}
	}
}

// parsePercentage converts a percentage string (e.g., "45%") to a float64 (e.g., 0.45)
func parsePercentage(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")

	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}

	// Convert percentage to decimal (0-100 -> 0.0-1.0)
	return value / 100.0
}
