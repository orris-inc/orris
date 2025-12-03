package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ReportOnlineSubscriptionsCommand represents the command to report online subscriptions
type ReportOnlineSubscriptionsCommand struct {
	NodeID        uint
	Subscriptions []dto.OnlineSubscriptionItem
}

// ReportOnlineSubscriptionsResult contains the result of online subscriptions reporting
type ReportOnlineSubscriptionsResult struct {
	Success     bool
	OnlineCount int
}

// OnlineSubscriptionTracker defines the interface for tracking online subscriptions
type OnlineSubscriptionTracker interface {
	UpdateOnlineSubscriptions(ctx context.Context, nodeID uint, subscriptions []OnlineSubscriptionInfo) error
}

// OnlineSubscriptionInfo represents simplified online subscription information for tracking
type OnlineSubscriptionInfo struct {
	SubscriptionID int
	IP             string
}

// ReportOnlineSubscriptionsUseCase handles reporting online subscriptions from node agents
type ReportOnlineSubscriptionsUseCase struct {
	subscriptionTracker OnlineSubscriptionTracker
	logger              logger.Interface
}

// NewReportOnlineSubscriptionsUseCase creates a new instance of ReportOnlineSubscriptionsUseCase
func NewReportOnlineSubscriptionsUseCase(
	subscriptionTracker OnlineSubscriptionTracker,
	logger logger.Interface,
) *ReportOnlineSubscriptionsUseCase {
	return &ReportOnlineSubscriptionsUseCase{
		subscriptionTracker: subscriptionTracker,
		logger:              logger,
	}
}

// Execute processes online subscriptions report from node agent
func (uc *ReportOnlineSubscriptionsUseCase) Execute(ctx context.Context, cmd ReportOnlineSubscriptionsCommand) (*ReportOnlineSubscriptionsResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	// Convert DTO to internal format
	subscriptions := make([]OnlineSubscriptionInfo, 0, len(cmd.Subscriptions))
	for _, s := range cmd.Subscriptions {
		if s.SubscriptionID == 0 {
			uc.logger.Warnw("skipping online subscription with invalid subscription_id",
				"node_id", cmd.NodeID,
			)
			continue
		}

		subscriptions = append(subscriptions, OnlineSubscriptionInfo{
			SubscriptionID: s.SubscriptionID,
			IP:             s.IP,
		})
	}

	// Update online subscriptions tracking
	if err := uc.subscriptionTracker.UpdateOnlineSubscriptions(ctx, cmd.NodeID, subscriptions); err != nil {
		uc.logger.Errorw("failed to update online subscriptions",
			"error", err,
			"node_id", cmd.NodeID,
			"subscription_count", len(subscriptions),
		)
		return nil, fmt.Errorf("failed to update online subscriptions")
	}

	uc.logger.Infow("online subscriptions reported successfully",
		"node_id", cmd.NodeID,
		"online_count", len(subscriptions),
	)

	return &ReportOnlineSubscriptionsResult{
		Success:     true,
		OnlineCount: len(subscriptions),
	}, nil
}
