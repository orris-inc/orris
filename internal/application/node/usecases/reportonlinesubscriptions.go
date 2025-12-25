package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/shared/id"
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
// Uses internal subscription ID for database storage
type OnlineSubscriptionInfo struct {
	SubscriptionID uint
	IP             string
}

// ReportOnlineSubscriptionsUseCase handles reporting online subscriptions from node agents
type ReportOnlineSubscriptionsUseCase struct {
	subscriptionTracker    OnlineSubscriptionTracker
	subscriptionIDResolver SubscriptionIDResolver
	logger                 logger.Interface
}

// NewReportOnlineSubscriptionsUseCase creates a new instance of ReportOnlineSubscriptionsUseCase
func NewReportOnlineSubscriptionsUseCase(
	subscriptionTracker OnlineSubscriptionTracker,
	subscriptionIDResolver SubscriptionIDResolver,
	logger logger.Interface,
) *ReportOnlineSubscriptionsUseCase {
	return &ReportOnlineSubscriptionsUseCase{
		subscriptionTracker:    subscriptionTracker,
		subscriptionIDResolver: subscriptionIDResolver,
		logger:                 logger,
	}
}

// Execute processes online subscriptions report from node agent
func (uc *ReportOnlineSubscriptionsUseCase) Execute(ctx context.Context, cmd ReportOnlineSubscriptionsCommand) (*ReportOnlineSubscriptionsResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	if len(cmd.Subscriptions) == 0 {
		// Empty list means no online subscriptions - still update tracker
		if err := uc.subscriptionTracker.UpdateOnlineSubscriptions(ctx, cmd.NodeID, []OnlineSubscriptionInfo{}); err != nil {
			uc.logger.Errorw("failed to update online subscriptions",
				"error", err,
				"node_id", cmd.NodeID,
			)
			return nil, fmt.Errorf("failed to update online subscriptions")
		}
		return &ReportOnlineSubscriptionsResult{
			Success:     true,
			OnlineCount: 0,
		}, nil
	}

	// Collect valid SIDs for batch lookup
	validSIDs := make([]string, 0, len(cmd.Subscriptions))
	ipMap := make(map[string]string) // SID -> IP

	for _, s := range cmd.Subscriptions {
		// Validate SID format
		if err := id.ValidatePrefix(s.SubscriptionSID, id.PrefixSubscription); err != nil {
			uc.logger.Warnw("skipping online subscription with invalid subscription SID",
				"node_id", cmd.NodeID,
				"subscription_sid", s.SubscriptionSID,
			)
			continue
		}

		validSIDs = append(validSIDs, s.SubscriptionSID)
		ipMap[s.SubscriptionSID] = s.IP
	}

	if len(validSIDs) == 0 {
		uc.logger.Warnw("no valid subscription SIDs in online report",
			"node_id", cmd.NodeID,
		)
		return &ReportOnlineSubscriptionsResult{
			Success:     true,
			OnlineCount: 0,
		}, nil
	}

	// Batch lookup SIDs to internal IDs
	sidToID, err := uc.subscriptionIDResolver.GetIDsBySIDs(ctx, validSIDs)
	if err != nil {
		uc.logger.Errorw("failed to lookup subscription IDs by SIDs",
			"error", err,
			"node_id", cmd.NodeID,
			"subscription_count", len(validSIDs),
		)
		return nil, fmt.Errorf("failed to lookup subscription IDs: %w", err)
	}

	// Build online subscription info with internal IDs
	subscriptions := make([]OnlineSubscriptionInfo, 0, len(sidToID))
	for sid, internalID := range sidToID {
		subscriptions = append(subscriptions, OnlineSubscriptionInfo{
			SubscriptionID: internalID,
			IP:             ipMap[sid],
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

	uc.logger.Debugw("online subscriptions reported",
		"node_id", cmd.NodeID,
		"online_count", len(subscriptions),
	)

	return &ReportOnlineSubscriptionsResult{
		Success:     true,
		OnlineCount: len(subscriptions),
	}, nil
}
