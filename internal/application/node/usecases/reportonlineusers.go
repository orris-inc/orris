package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/node/dto"
	"orris/internal/shared/logger"
)

// ReportOnlineUsersCommand represents the command to report online users
type ReportOnlineUsersCommand struct {
	NodeID uint
	Users  []dto.OnlineUserItem
}

// ReportOnlineUsersResult contains the result of online users reporting
type ReportOnlineUsersResult struct {
	Success     bool
	OnlineCount int
}

// OnlineUserTracker defines the interface for tracking online users
type OnlineUserTracker interface {
	UpdateOnlineUsers(ctx context.Context, nodeID uint, users []OnlineUserInfo) error
}

// OnlineUserInfo represents simplified online user information for tracking
type OnlineUserInfo struct {
	UserID int
	IP     string
}

// ReportOnlineUsersUseCase handles reporting online users from XrayR clients
type ReportOnlineUsersUseCase struct {
	userTracker OnlineUserTracker
	logger      logger.Interface
}

// NewReportOnlineUsersUseCase creates a new instance of ReportOnlineUsersUseCase
func NewReportOnlineUsersUseCase(
	userTracker OnlineUserTracker,
	logger logger.Interface,
) *ReportOnlineUsersUseCase {
	return &ReportOnlineUsersUseCase{
		userTracker: userTracker,
		logger:      logger,
	}
}

// Execute processes online users report from XrayR backend
func (uc *ReportOnlineUsersUseCase) Execute(ctx context.Context, cmd ReportOnlineUsersCommand) (*ReportOnlineUsersResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	// Convert DTO to internal format
	users := make([]OnlineUserInfo, 0, len(cmd.Users))
	for _, u := range cmd.Users {
		if u.UID == 0 {
			uc.logger.Warnw("skipping online user with invalid UID",
				"node_id", cmd.NodeID,
			)
			continue
		}

		users = append(users, OnlineUserInfo{
			UserID: u.UID,
			IP:     u.IP,
		})
	}

	// Update online users tracking
	if err := uc.userTracker.UpdateOnlineUsers(ctx, cmd.NodeID, users); err != nil {
		uc.logger.Errorw("failed to update online users",
			"error", err,
			"node_id", cmd.NodeID,
			"user_count", len(users),
		)
		return nil, fmt.Errorf("failed to update online users")
	}

	uc.logger.Infow("online users reported successfully",
		"node_id", cmd.NodeID,
		"online_count", len(users),
	)

	return &ReportOnlineUsersResult{
		Success:     true,
		OnlineCount: len(users),
	}, nil
}
