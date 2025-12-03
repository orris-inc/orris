package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/node/dto"
	"orris/internal/domain/subscription"
	"orris/internal/infrastructure/config"
	"orris/internal/shared/logger"
)

// GetNodeSubscriptionsCommand represents the command to get subscriptions authorized for a node
type GetNodeSubscriptionsCommand struct {
	NodeID uint
}

// GetNodeSubscriptionsResult contains the list of subscriptions for the node
type GetNodeSubscriptionsResult struct {
	Subscriptions *dto.NodeSubscriptionsResponse
}

// GetNodeSubscriptionsUseCase handles fetching subscription list for node agents
type GetNodeSubscriptionsUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	logger           logger.Interface
}

// NewGetNodeSubscriptionsUseCase creates a new instance of GetNodeSubscriptionsUseCase
func NewGetNodeSubscriptionsUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *GetNodeSubscriptionsUseCase {
	return &GetNodeSubscriptionsUseCase{
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// Execute retrieves the list of subscriptions authorized to use the node
func (uc *GetNodeSubscriptionsUseCase) Execute(ctx context.Context, cmd GetNodeSubscriptionsCommand) (*GetNodeSubscriptionsResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	// Get all active subscriptions that can access this node
	subscriptions, err := uc.subscriptionRepo.GetActiveSubscriptionsByNodeID(ctx, cmd.NodeID)
	if err != nil {
		uc.logger.Errorw("failed to get node subscriptions",
			"error", err,
			"node_id", cmd.NodeID,
		)
		return nil, fmt.Errorf("failed to retrieve subscriptions for node")
	}

	// Get HMAC secret from config for password generation
	hmacSecret := config.Get().Auth.JWT.Secret

	// Convert subscriptions to agent subscriptions response
	subscriptionInfos := dto.ToNodeSubscriptionsResponse(subscriptions, hmacSecret)

	uc.logger.Infow("node subscriptions retrieved successfully",
		"node_id", cmd.NodeID,
		"subscription_count", len(subscriptionInfos.Subscriptions),
	)

	return &GetNodeSubscriptionsResult{
		Subscriptions: subscriptionInfos,
	}, nil
}
