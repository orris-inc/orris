package usecases

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/orris-inc/orris/internal/application/node/dto"
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

// NodeSystemStatusUpdater defines the interface for updating node system status
type NodeSystemStatusUpdater interface {
	UpdateSystemStatus(ctx context.Context, nodeID uint, cpu, memory, disk float64, uptime int) error
}

// ReportNodeStatusUseCase handles reporting node system status from node agents
type ReportNodeStatusUseCase struct {
	statusUpdater NodeSystemStatusUpdater
	logger        logger.Interface
}

// NewReportNodeStatusUseCase creates a new instance of ReportNodeStatusUseCase
func NewReportNodeStatusUseCase(
	statusUpdater NodeSystemStatusUpdater,
	logger logger.Interface,
) *ReportNodeStatusUseCase {
	return &ReportNodeStatusUseCase{
		statusUpdater: statusUpdater,
		logger:        logger,
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

	// Update node system status
	if err := uc.statusUpdater.UpdateSystemStatus(
		ctx,
		cmd.NodeID,
		cpu,
		memory,
		disk,
		cmd.Status.Uptime,
	); err != nil {
		uc.logger.Errorw("failed to update node system status",
			"error", err,
			"node_id", cmd.NodeID,
		)
		return nil, fmt.Errorf("failed to update node status")
	}

	uc.logger.Infow("node status reported successfully",
		"node_id", cmd.NodeID,
		"cpu", cmd.Status.CPU,
		"memory", cmd.Status.Mem,
		"disk", cmd.Status.Disk,
		"uptime", cmd.Status.Uptime,
	)

	return &ReportNodeStatusResult{
		Success: true,
	}, nil
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
