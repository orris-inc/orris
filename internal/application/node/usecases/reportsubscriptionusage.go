package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ReportSubscriptionUsageCommand represents the command to report subscription usage data
type ReportSubscriptionUsageCommand struct {
	NodeID        uint
	Subscriptions []dto.SubscriptionUsageItem
}

// ReportSubscriptionUsageResult contains the result of usage reporting
type ReportSubscriptionUsageResult struct {
	Success              bool
	SubscriptionsUpdated int
}

// SubscriptionUsageItem represents a single subscription's usage data for batch recording
// Uses internal subscription ID for database storage
type SubscriptionUsageItem struct {
	SubscriptionID uint
	Upload         int64
	Download       int64
}

// SubscriptionUsageRecorder defines the interface for recording subscription usage
type SubscriptionUsageRecorder interface {
	RecordSubscriptionUsage(ctx context.Context, nodeID uint, subscriptionID uint, upload, download int64) error
	BatchRecordSubscriptionUsage(ctx context.Context, nodeID uint, items []SubscriptionUsageItem) error
}

// SubscriptionIDResolver defines the interface for resolving subscription SIDs to internal IDs
type SubscriptionIDResolver interface {
	GetIDBySID(ctx context.Context, sid string) (uint, error)
	GetIDsBySIDs(ctx context.Context, sids []string) (map[string]uint, error)
}

// ReportSubscriptionUsageUseCase handles reporting subscription usage from node agents
type ReportSubscriptionUsageUseCase struct {
	usageRecorder      SubscriptionUsageRecorder
	subscriptionLookup SubscriptionIDResolver
	logger             logger.Interface
}

// NewReportSubscriptionUsageUseCase creates a new instance of ReportSubscriptionUsageUseCase
func NewReportSubscriptionUsageUseCase(
	usageRecorder SubscriptionUsageRecorder,
	subscriptionLookup SubscriptionIDResolver,
	logger logger.Interface,
) *ReportSubscriptionUsageUseCase {
	return &ReportSubscriptionUsageUseCase{
		usageRecorder:      usageRecorder,
		subscriptionLookup: subscriptionLookup,
		logger:             logger,
	}
}

// Execute processes subscription usage report from node agent
func (uc *ReportSubscriptionUsageUseCase) Execute(ctx context.Context, cmd ReportSubscriptionUsageCommand) (*ReportSubscriptionUsageResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	if len(cmd.Subscriptions) == 0 {
		return &ReportSubscriptionUsageResult{
			Success:              true,
			SubscriptionsUpdated: 0,
		}, nil
	}

	// Collect valid SIDs for batch lookup
	validSIDs := make([]string, 0, len(cmd.Subscriptions))
	usageMap := make(map[string]dto.SubscriptionUsageItem)

	for _, subscription := range cmd.Subscriptions {
		// Validate SID format
		if err := id.ValidatePrefix(subscription.SubscriptionSID, id.PrefixSubscription); err != nil {
			uc.logger.Warnw("skipping subscription usage with invalid subscription SID",
				"node_id", cmd.NodeID,
				"subscription_sid", subscription.SubscriptionSID,
			)
			continue
		}

		// Skip if no usage to report
		if subscription.Upload == 0 && subscription.Download == 0 {
			continue
		}

		validSIDs = append(validSIDs, subscription.SubscriptionSID)
		usageMap[subscription.SubscriptionSID] = subscription
	}

	// If no valid items, return early
	if len(validSIDs) == 0 {
		return &ReportSubscriptionUsageResult{
			Success:              true,
			SubscriptionsUpdated: 0,
		}, nil
	}

	// Batch lookup SIDs to internal IDs
	sidToID, err := uc.subscriptionLookup.GetIDsBySIDs(ctx, validSIDs)
	if err != nil {
		uc.logger.Errorw("failed to lookup subscription IDs by SIDs",
			"error", err,
			"node_id", cmd.NodeID,
			"subscription_count", len(validSIDs),
		)
		return nil, fmt.Errorf("failed to lookup subscription IDs: %w", err)
	}

	// Build usage items with internal IDs
	validItems := make([]SubscriptionUsageItem, 0, len(sidToID))
	for sid, internalID := range sidToID {
		usage := usageMap[sid]
		validItems = append(validItems, SubscriptionUsageItem{
			SubscriptionID: internalID,
			Upload:         usage.Upload,
			Download:       usage.Download,
		})
	}

	if len(validItems) == 0 {
		uc.logger.Warnw("no subscription IDs found for provided SIDs",
			"node_id", cmd.NodeID,
			"sids", validSIDs,
		)
		return &ReportSubscriptionUsageResult{
			Success:              true,
			SubscriptionsUpdated: 0,
		}, nil
	}

	// Batch record subscription usage for improved performance
	if err := uc.usageRecorder.BatchRecordSubscriptionUsage(ctx, cmd.NodeID, validItems); err != nil {
		uc.logger.Errorw("failed to batch record subscription usage",
			"error", err,
			"node_id", cmd.NodeID,
			"subscription_count", len(validItems),
		)
		return nil, fmt.Errorf("failed to batch record subscription usage: %w", err)
	}

	uc.logger.Debugw("subscription usage reported",
		"node_id", cmd.NodeID,
		"subscriptions_recorded", len(validItems),
	)

	return &ReportSubscriptionUsageResult{
		Success:              true,
		SubscriptionsUpdated: len(validItems),
	}, nil
}
