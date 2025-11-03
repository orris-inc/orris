package usecases

import (
	"context"
	"time"

	"orris/internal/domain/node"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type GetNodeTrafficStatsQuery struct {
	NodeID         *uint
	UserID         *uint
	SubscriptionID *uint
	From           time.Time
	To             time.Time
	Granularity    string
	Page           int
	PageSize       int
}

type NodeTrafficStatsResult struct {
	NodeID         uint      `json:"node_id"`
	UserID         *uint     `json:"user_id,omitempty"`
	SubscriptionID *uint     `json:"subscription_id,omitempty"`
	Upload         uint64    `json:"upload"`
	Download       uint64    `json:"download"`
	Total          uint64    `json:"total"`
	Period         time.Time `json:"period"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type GetNodeTrafficStatsUseCase struct {
	trafficRepo node.NodeTrafficRepository
	logger      logger.Interface
}

func NewGetNodeTrafficStatsUseCase(
	trafficRepo node.NodeTrafficRepository,
	logger logger.Interface,
) *GetNodeTrafficStatsUseCase {
	return &GetNodeTrafficStatsUseCase{
		trafficRepo: trafficRepo,
		logger:      logger,
	}
}

func (uc *GetNodeTrafficStatsUseCase) Execute(
	ctx context.Context,
	query GetNodeTrafficStatsQuery,
) ([]*NodeTrafficStatsResult, error) {
	uc.logger.Infow("fetching node traffic stats",
		"node_id", query.NodeID,
		"from", query.From,
		"to", query.To,
		"granularity", query.Granularity,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid traffic stats query", "error", err)
		return nil, err
	}

	filter := uc.buildFilter(query)

	trafficRecords, err := uc.trafficRepo.GetTrafficStats(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to fetch traffic stats", "error", err)
		return nil, errors.NewInternalError("failed to fetch traffic statistics")
	}

	results := make([]*NodeTrafficStatsResult, 0, len(trafficRecords))
	for _, record := range trafficRecords {
		results = append(results, &NodeTrafficStatsResult{
			NodeID:         record.NodeID(),
			UserID:         record.UserID(),
			SubscriptionID: record.SubscriptionID(),
			Upload:         record.Upload(),
			Download:       record.Download(),
			Total:          record.Total(),
			Period:         record.Period(),
			CreatedAt:      record.CreatedAt(),
			UpdatedAt:      record.UpdatedAt(),
		})
	}

	uc.logger.Infow("traffic stats fetched successfully",
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

func (uc *GetNodeTrafficStatsUseCase) buildFilter(query GetNodeTrafficStatsQuery) node.TrafficStatsFilter {
	page := query.Page
	if page == 0 {
		page = 1
	}

	pageSize := query.PageSize
	if pageSize == 0 {
		pageSize = 100
	}

	filter := node.TrafficStatsFilter{
		NodeID:         query.NodeID,
		UserID:         query.UserID,
		SubscriptionID: query.SubscriptionID,
		From:           query.From,
		To:             query.To,
	}
	filter.Page = page
	filter.PageSize = pageSize

	if query.Granularity != "" {
		filter.Period = &query.Granularity
	}

	return filter
}
