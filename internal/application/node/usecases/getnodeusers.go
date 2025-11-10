package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/node/dto"
	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
)

// GetNodeUsersCommand represents the command to get users authorized for a node
type GetNodeUsersCommand struct {
	NodeID uint
}

// GetNodeUsersResult contains the list of users for the node
type GetNodeUsersResult struct {
	Users *dto.NodeUsersResponse
}

// GetNodeUsersUseCase handles fetching user list for XrayR clients
type GetNodeUsersUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	logger           logger.Interface
}

// NewGetNodeUsersUseCase creates a new instance of GetNodeUsersUseCase
func NewGetNodeUsersUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *GetNodeUsersUseCase {
	return &GetNodeUsersUseCase{
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// Execute retrieves the list of users authorized to use the node
func (uc *GetNodeUsersUseCase) Execute(ctx context.Context, cmd GetNodeUsersCommand) (*GetNodeUsersResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	// Get all active subscriptions that can access this node
	subscriptions, err := uc.subscriptionRepo.GetActiveSubscriptionsByNodeID(ctx, cmd.NodeID)
	if err != nil {
		uc.logger.Errorw("failed to get node users",
			"error", err,
			"node_id", cmd.NodeID,
		)
		return nil, fmt.Errorf("failed to retrieve users for node")
	}

	// Convert subscriptions to XrayR users response
	users := dto.ToNodeUsersResponse(subscriptions)

	uc.logger.Infow("node users retrieved successfully",
		"node_id", cmd.NodeID,
		"user_count", len(users.Users),
	)

	return &GetNodeUsersResult{
		Users: users,
	}, nil
}
