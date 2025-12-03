package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ReportSubscriptionTrafficCommand represents the command to report subscription traffic data
type ReportSubscriptionTrafficCommand struct {
	NodeID        uint
	Subscriptions []dto.SubscriptionTrafficItem
}

// ReportSubscriptionTrafficResult contains the result of traffic reporting
type ReportSubscriptionTrafficResult struct {
	Success              bool
	SubscriptionsUpdated int
}

// SubscriptionTrafficItem represents a single subscription's traffic data for batch recording
type SubscriptionTrafficItem struct {
	SubscriptionID int
	Upload         int64
	Download       int64
}

// SubscriptionTrafficRecorder defines the interface for recording subscription traffic
type SubscriptionTrafficRecorder interface {
	RecordSubscriptionTraffic(ctx context.Context, nodeID uint, subscriptionID int, upload, download int64) error
	BatchRecordSubscriptionTraffic(ctx context.Context, nodeID uint, items []SubscriptionTrafficItem) error
}

// ReportSubscriptionTrafficUseCase handles reporting subscription traffic from node agents
type ReportSubscriptionTrafficUseCase struct {
	trafficRecorder SubscriptionTrafficRecorder
	logger          logger.Interface
}

// NewReportSubscriptionTrafficUseCase creates a new instance of ReportSubscriptionTrafficUseCase
func NewReportSubscriptionTrafficUseCase(
	trafficRecorder SubscriptionTrafficRecorder,
	logger logger.Interface,
) *ReportSubscriptionTrafficUseCase {
	return &ReportSubscriptionTrafficUseCase{
		trafficRecorder: trafficRecorder,
		logger:          logger,
	}
}

// Execute processes subscription traffic report from node agent
func (uc *ReportSubscriptionTrafficUseCase) Execute(ctx context.Context, cmd ReportSubscriptionTrafficCommand) (*ReportSubscriptionTrafficResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	if len(cmd.Subscriptions) == 0 {
		uc.logger.Infow("no subscription traffic data to report",
			"node_id", cmd.NodeID,
		)
		return &ReportSubscriptionTrafficResult{
			Success:              true,
			SubscriptionsUpdated: 0,
		}, nil
	}

	// Collect valid traffic items for batch processing
	validItems := make([]SubscriptionTrafficItem, 0, len(cmd.Subscriptions))
	for _, subscription := range cmd.Subscriptions {
		if subscription.SubscriptionID == 0 {
			uc.logger.Warnw("skipping subscription traffic with invalid subscription_id",
				"node_id", cmd.NodeID,
			)
			continue
		}

		// Skip if no traffic to report
		if subscription.Upload == 0 && subscription.Download == 0 {
			continue
		}

		validItems = append(validItems, SubscriptionTrafficItem{
			SubscriptionID: subscription.SubscriptionID,
			Upload:         subscription.Upload,
			Download:       subscription.Download,
		})
	}

	// If no valid items, return early
	if len(validItems) == 0 {
		uc.logger.Infow("no valid subscription traffic data to report",
			"node_id", cmd.NodeID,
			"total_subscriptions", len(cmd.Subscriptions),
		)
		return &ReportSubscriptionTrafficResult{
			Success:              true,
			SubscriptionsUpdated: 0,
		}, nil
	}

	// Batch record subscription traffic for improved performance
	if err := uc.trafficRecorder.BatchRecordSubscriptionTraffic(ctx, cmd.NodeID, validItems); err != nil {
		uc.logger.Errorw("failed to batch record subscription traffic",
			"error", err,
			"node_id", cmd.NodeID,
			"subscription_count", len(validItems),
		)
		return nil, fmt.Errorf("failed to batch record subscription traffic: %w", err)
	}

	uc.logger.Infow("subscription traffic reported successfully",
		"node_id", cmd.NodeID,
		"total_subscriptions", len(cmd.Subscriptions),
		"subscriptions_recorded", len(validItems),
	)

	return &ReportSubscriptionTrafficResult{
		Success:              true,
		SubscriptionsUpdated: len(validItems),
	}, nil
}
