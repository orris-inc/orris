package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/shared/logger"
)

type ReportNodeDataCommand struct {
	NodeID      uint
	Upload      uint64
	Download    uint64
	OnlineUsers int
	Status      string
	SystemInfo  *SystemInfo
	Timestamp   time.Time
}

type SystemInfo struct {
	Load        float64
	MemoryUsage float64
	DiskUsage   float64
}

type ReportNodeDataResult struct {
	ShouldReload     bool
	ConfigVersion    int
	TrafficExceeded  bool
	TrafficRemaining uint64
}

type NodeTrafficRecorder interface {
	RecordTraffic(ctx context.Context, nodeID uint, upload, download uint64) error
}

type NodeStatusUpdater interface {
	UpdateStatus(ctx context.Context, nodeID uint, status string, onlineUsers int, systemInfo *SystemInfo) error
}

type NodeLimitChecker interface {
	CheckLimits(ctx context.Context, nodeID uint) (exceeded bool, remaining uint64, err error)
}

type ReportNodeDataUseCase struct {
	trafficRecorder NodeTrafficRecorder
	statusUpdater   NodeStatusUpdater
	limitChecker    NodeLimitChecker
	logger          logger.Interface
}

func NewReportNodeDataUseCase(
	trafficRecorder NodeTrafficRecorder,
	statusUpdater NodeStatusUpdater,
	limitChecker NodeLimitChecker,
	logger logger.Interface,
) *ReportNodeDataUseCase {
	return &ReportNodeDataUseCase{
		trafficRecorder: trafficRecorder,
		statusUpdater:   statusUpdater,
		limitChecker:    limitChecker,
		logger:          logger,
	}
}

func (uc *ReportNodeDataUseCase) Execute(ctx context.Context, cmd ReportNodeDataCommand) (*ReportNodeDataResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	if cmd.Upload > 0 || cmd.Download > 0 {
		if err := uc.trafficRecorder.RecordTraffic(ctx, cmd.NodeID, cmd.Upload, cmd.Download); err != nil {
			uc.logger.Errorw("failed to record traffic",
				"error", err,
				"node_id", cmd.NodeID,
				"upload", cmd.Upload,
				"download", cmd.Download,
			)
			return nil, fmt.Errorf("failed to record traffic: %w", err)
		}
	}

	if err := uc.statusUpdater.UpdateStatus(ctx, cmd.NodeID, cmd.Status, cmd.OnlineUsers, cmd.SystemInfo); err != nil {
		uc.logger.Warnw("failed to update status",
			"error", err,
			"node_id", cmd.NodeID,
			"status", cmd.Status,
		)
	}

	exceeded, remaining, err := uc.limitChecker.CheckLimits(ctx, cmd.NodeID)
	if err != nil {
		uc.logger.Warnw("failed to check limits",
			"error", err,
			"node_id", cmd.NodeID,
		)
	}

	if exceeded {
		uc.logger.Warnw("node traffic limit exceeded",
			"node_id", cmd.NodeID,
			"remaining", remaining,
		)
	}

	uc.logger.Infow("node data reported successfully",
		"node_id", cmd.NodeID,
		"upload", cmd.Upload,
		"download", cmd.Download,
		"online_users", cmd.OnlineUsers,
		"traffic_exceeded", exceeded,
	)

	return &ReportNodeDataResult{
		ShouldReload:     false,
		ConfigVersion:    1,
		TrafficExceeded:  exceeded,
		TrafficRemaining: remaining,
	}, nil
}
