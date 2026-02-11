package usecases

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type GetNodeTrafficStatsQuery struct {
	NodeID         *uint
	SubscriptionID *uint
	From           time.Time
	To             time.Time
	Granularity    string
	Page           int
	PageSize       int
}

type NodeTrafficStatsResult struct {
	NodeID         uint      `json:"node_id"`
	SubscriptionID *uint     `json:"subscription_id,omitempty"`
	Upload         uint64    `json:"upload"`
	Download       uint64    `json:"download"`
	Total          uint64    `json:"total"`
	Period         time.Time `json:"period"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type GetNodeTrafficStatsUseCase struct {
	usageStatsRepo subscription.SubscriptionUsageStatsRepository
	logger         logger.Interface
}

func NewGetNodeTrafficStatsUseCase(
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	logger logger.Interface,
) *GetNodeTrafficStatsUseCase {
	return &GetNodeTrafficStatsUseCase{
		usageStatsRepo: usageStatsRepo,
		logger:         logger,
	}
}

func (uc *GetNodeTrafficStatsUseCase) Execute(
	ctx context.Context,
	query GetNodeTrafficStatsQuery,
) ([]*NodeTrafficStatsResult, error) {
	uc.logger.Debugw("fetching node traffic stats",
		"node_id", query.NodeID,
		"from", query.From,
		"to", query.To,
		"granularity", query.Granularity,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid traffic stats query", "error", err)
		return nil, err
	}

	// Determine granularity for query
	granularity := subscription.GranularityDaily
	if query.Granularity == "month" {
		granularity = subscription.GranularityMonthly
	}

	// Get usage stats from subscription_usage_stats table
	resourceType := subscription.ResourceTypeNode.String()
	var usageRecords []*subscription.SubscriptionUsageStats
	var err error

	if query.NodeID != nil {
		usageRecords, err = uc.usageStatsRepo.GetByResourceID(
			ctx,
			resourceType,
			*query.NodeID,
			granularity,
			query.From,
			query.To,
		)
	} else {
		// If no specific node is requested, return empty result
		// Admin stats use cases should be used for platform-wide queries
		return []*NodeTrafficStatsResult{}, nil
	}

	if err != nil {
		uc.logger.Errorw("failed to fetch traffic stats", "error", err)
		return nil, errors.NewInternalError("failed to fetch traffic statistics")
	}

	results := make([]*NodeTrafficStatsResult, 0, len(usageRecords))
	for _, record := range usageRecords {
		results = append(results, &NodeTrafficStatsResult{
			NodeID:         record.ResourceID(),
			SubscriptionID: record.SubscriptionID(),
			Upload:         record.Upload(),
			Download:       record.Download(),
			Total:          record.Total(),
			Period:         record.Period(),
			CreatedAt:      record.CreatedAt(),
			UpdatedAt:      record.UpdatedAt(),
		})
	}

	uc.logger.Debugw("traffic stats fetched successfully",
		"count", len(results),
	)

	return results, nil
}

func (uc *GetNodeTrafficStatsUseCase) validateQuery(query GetNodeTrafficStatsQuery) error {
	if query.From.IsZero() {
		return errors.NewValidationError("from time is required")
	}

	if query.To.IsZero() {
		return errors.NewValidationError("to time is required")
	}

	if query.To.Before(query.From) {
		return errors.NewValidationError("to time must be after from time")
	}

	if query.Granularity != "" &&
		query.Granularity != "hour" &&
		query.Granularity != "day" &&
		query.Granularity != "month" {
		return errors.NewValidationError("granularity must be one of: hour, day, month")
	}

	if query.Page < 0 {
		return errors.NewValidationError("page must be non-negative")
	}

	if query.PageSize < 0 {
		return errors.NewValidationError("page size must be non-negative")
	}

	return nil
}
