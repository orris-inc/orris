package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/node/dto"
	"orris/internal/shared/logger"
)

// ReportUserTrafficCommand represents the command to report user traffic data
type ReportUserTrafficCommand struct {
	NodeID uint
	Users  []dto.UserTrafficItem
}

// ReportUserTrafficResult contains the result of traffic reporting
type ReportUserTrafficResult struct {
	Success      bool
	UsersUpdated int
}

// UserTrafficRecorder defines the interface for recording user traffic
type UserTrafficRecorder interface {
	RecordUserTraffic(ctx context.Context, nodeID uint, userID int, upload, download int64) error
}

// ReportUserTrafficUseCase handles reporting user traffic from XrayR clients
type ReportUserTrafficUseCase struct {
	trafficRecorder UserTrafficRecorder
	logger          logger.Interface
}

// NewReportUserTrafficUseCase creates a new instance of ReportUserTrafficUseCase
func NewReportUserTrafficUseCase(
	trafficRecorder UserTrafficRecorder,
	logger logger.Interface,
) *ReportUserTrafficUseCase {
	return &ReportUserTrafficUseCase{
		trafficRecorder: trafficRecorder,
		logger:          logger,
	}
}

// Execute processes user traffic report from XrayR backend
func (uc *ReportUserTrafficUseCase) Execute(ctx context.Context, cmd ReportUserTrafficCommand) (*ReportUserTrafficResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	if len(cmd.Users) == 0 {
		uc.logger.Infow("no user traffic data to report",
			"node_id", cmd.NodeID,
		)
		return &ReportUserTrafficResult{
			Success:      true,
			UsersUpdated: 0,
		}, nil
	}

	successCount := 0
	for _, user := range cmd.Users {
		if user.UID == 0 {
			uc.logger.Warnw("skipping user traffic with invalid UID",
				"node_id", cmd.NodeID,
			)
			continue
		}

		// Skip if no traffic to report
		if user.Upload == 0 && user.Download == 0 {
			continue
		}

		// Record user traffic
		if err := uc.trafficRecorder.RecordUserTraffic(ctx, cmd.NodeID, user.UID, user.Upload, user.Download); err != nil {
			uc.logger.Errorw("failed to record user traffic",
				"error", err,
				"node_id", cmd.NodeID,
				"user_id", user.UID,
				"upload", user.Upload,
				"download", user.Download,
			)
			// Continue processing other users even if one fails
			continue
		}

		successCount++
	}

	uc.logger.Infow("user traffic reported successfully",
		"node_id", cmd.NodeID,
		"total_users", len(cmd.Users),
		"success_count", successCount,
	)

	return &ReportUserTrafficResult{
		Success:      true,
		UsersUpdated: successCount,
	}, nil
}
