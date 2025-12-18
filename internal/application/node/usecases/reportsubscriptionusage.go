package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/node/dto"
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
type SubscriptionUsageItem struct {
	SubscriptionID int
	Upload         int64
	Download       int64
}

// SubscriptionUsageRecorder defines the interface for recording subscription usage
type SubscriptionUsageRecorder interface {
	RecordSubscriptionUsage(ctx context.Context, nodeID uint, subscriptionID int, upload, download int64) error
	BatchRecordSubscriptionUsage(ctx context.Context, nodeID uint, items []SubscriptionUsageItem) error
}

// ReportSubscriptionUsageUseCase handles reporting subscription usage from node agents
type ReportSubscriptionUsageUseCase struct {
	usageRecorder SubscriptionUsageRecorder
	logger        logger.Interface
}

// NewReportSubscriptionUsageUseCase creates a new instance of ReportSubscriptionUsageUseCase
func NewReportSubscriptionUsageUseCase(
	usageRecorder SubscriptionUsageRecorder,
	logger logger.Interface,
) *ReportSubscriptionUsageUseCase {
	return &ReportSubscriptionUsageUseCase{
		usageRecorder: usageRecorder,
		logger:        logger,
	}
}

// Execute processes subscription usage report from node agent
func (uc *ReportSubscriptionUsageUseCase) Execute(ctx context.Context, cmd ReportSubscriptionUsageCommand) (*ReportSubscriptionUsageResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	if len(cmd.Subscriptions) == 0 {
		uc.logger.Infow("no subscription usage data to report",
			"node_id", cmd.NodeID,
		)
		return &ReportSubscriptionUsageResult{
			Success:              true,
			SubscriptionsUpdated: 0,
		}, nil
	}

	// Collect valid usage items for batch processing
	validItems := make([]SubscriptionUsageItem, 0, len(cmd.Subscriptions))
	for _, subscription := range cmd.Subscriptions {
		if subscription.SubscriptionID == 0 {
			uc.logger.Warnw("skipping subscription usage with invalid subscription_id",
				"node_id", cmd.NodeID,
			)
			continue
		}

		// Skip if no usage to report
		if subscription.Upload == 0 && subscription.Download == 0 {
			continue
		}

		validItems = append(validItems, SubscriptionUsageItem{
			SubscriptionID: subscription.SubscriptionID,
			Upload:         subscription.Upload,
			Download:       subscription.Download,
		})
	}

	// If no valid items, return early
	if len(validItems) == 0 {
		uc.logger.Infow("no valid subscription usage data to report",
			"node_id", cmd.NodeID,
			"total_subscriptions", len(cmd.Subscriptions),
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

	uc.logger.Infow("subscription usage reported successfully",
		"node_id", cmd.NodeID,
		"total_subscriptions", len(cmd.Subscriptions),
		"subscriptions_recorded", len(validItems),
	)

	return &ReportSubscriptionUsageResult{
		Success:              true,
		SubscriptionsUpdated: len(validItems),
	}, nil
}
