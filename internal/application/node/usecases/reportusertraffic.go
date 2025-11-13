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

// UserTrafficItem represents a single user's traffic data for batch recording
type UserTrafficItem struct {
	UserID   int
	Upload   int64
	Download int64
}

// UserTrafficRecorder defines the interface for recording user traffic
type UserTrafficRecorder interface {
	RecordUserTraffic(ctx context.Context, nodeID uint, userID int, upload, download int64) error
	BatchRecordUserTraffic(ctx context.Context, nodeID uint, items []UserTrafficItem) error
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

	// Collect valid traffic items for batch processing
	validItems := make([]UserTrafficItem, 0, len(cmd.Users))
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

		validItems = append(validItems, UserTrafficItem{
			UserID:   user.UID,
			Upload:   user.Upload,
			Download: user.Download,
		})
	}

	// If no valid items, return early
	if len(validItems) == 0 {
		uc.logger.Infow("no valid user traffic data to report",
			"node_id", cmd.NodeID,
			"total_users", len(cmd.Users),
		)
		return &ReportUserTrafficResult{
			Success:      true,
			UsersUpdated: 0,
		}, nil
	}

	// Batch record user traffic for improved performance
	if err := uc.trafficRecorder.BatchRecordUserTraffic(ctx, cmd.NodeID, validItems); err != nil {
		uc.logger.Errorw("failed to batch record user traffic",
			"error", err,
			"node_id", cmd.NodeID,
			"user_count", len(validItems),
		)
		return nil, fmt.Errorf("failed to batch record user traffic: %w", err)
	}

	uc.logger.Infow("user traffic reported successfully",
		"node_id", cmd.NodeID,
		"total_users", len(cmd.Users),
		"users_recorded", len(validItems),
	)

	return &ReportUserTrafficResult{
		Success:      true,
		UsersUpdated: len(validItems),
	}, nil
}
