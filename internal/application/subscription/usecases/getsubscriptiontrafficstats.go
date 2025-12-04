package usecases

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetSubscriptionTrafficStatsQuery represents the query parameters for subscription traffic stats
type GetSubscriptionTrafficStatsQuery struct {
	SubscriptionID uint
	From           time.Time
	To             time.Time
	Granularity    string // hour, day, month
	Page           int
	PageSize       int
}

// SubscriptionTrafficStatsRecord represents a single traffic stats record
type SubscriptionTrafficStatsRecord struct {
	NodeID   uint      `json:"node_id"`
	Upload   uint64    `json:"upload"`
	Download uint64    `json:"download"`
	Total    uint64    `json:"total"`
	Period   time.Time `json:"period"`
}

// SubscriptionTrafficSummary represents aggregated traffic summary
type SubscriptionTrafficSummary struct {
	TotalUpload   uint64 `json:"total_upload"`
	TotalDownload uint64 `json:"total_download"`
	Total         uint64 `json:"total"`
}

// GetSubscriptionTrafficStatsResponse represents the response for subscription traffic stats
type GetSubscriptionTrafficStatsResponse struct {
	Records  []*SubscriptionTrafficStatsRecord `json:"records"`
	Summary  *SubscriptionTrafficSummary       `json:"summary"`
	Total    int                               `json:"total"`
	Page     int                               `json:"page"`
	PageSize int                               `json:"page_size"`
}

// GetSubscriptionTrafficStatsUseCase handles retrieving traffic statistics for a subscription
type GetSubscriptionTrafficStatsUseCase struct {
	trafficRepo subscription.SubscriptionTrafficRepository
	logger      logger.Interface
}

// NewGetSubscriptionTrafficStatsUseCase creates a new GetSubscriptionTrafficStatsUseCase
func NewGetSubscriptionTrafficStatsUseCase(
	trafficRepo subscription.SubscriptionTrafficRepository,
	logger logger.Interface,
) *GetSubscriptionTrafficStatsUseCase {
	return &GetSubscriptionTrafficStatsUseCase{
		trafficRepo: trafficRepo,
		logger:      logger,
	}
}

// Execute retrieves traffic statistics for a subscription
func (uc *GetSubscriptionTrafficStatsUseCase) Execute(
	ctx context.Context,
	query GetSubscriptionTrafficStatsQuery,
) (*GetSubscriptionTrafficStatsResponse, error) {
	uc.logger.Infow("fetching subscription traffic stats",
		"subscription_id", query.SubscriptionID,
		"from", query.From,
		"to", query.To,
		"granularity", query.Granularity,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid subscription traffic stats query", "error", err)
		return nil, err
	}

	filter := uc.buildFilter(query)

	trafficRecords, err := uc.trafficRepo.GetTrafficStats(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscription traffic stats", "error", err)
		return nil, errors.NewInternalError("failed to fetch traffic statistics")
	}

	// Convert records and calculate summary
	records := make([]*SubscriptionTrafficStatsRecord, 0, len(trafficRecords))
	summary := &SubscriptionTrafficSummary{}

	for _, record := range trafficRecords {
		records = append(records, &SubscriptionTrafficStatsRecord{
			NodeID:   record.NodeID(),
			Upload:   record.Upload(),
			Download: record.Download(),
			Total:    record.Total(),
			Period:   record.Period(),
		})
		summary.TotalUpload += record.Upload()
		summary.TotalDownload += record.Download()
		summary.Total += record.Total()
	}

	response := &GetSubscriptionTrafficStatsResponse{
		Records:  records,
		Summary:  summary,
		Total:    len(records),
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}

	uc.logger.Infow("subscription traffic stats fetched successfully",
		"subscription_id", query.SubscriptionID,
		"count", len(records),
	)

	return response, nil
}

func (uc *GetSubscriptionTrafficStatsUseCase) validateQuery(query GetSubscriptionTrafficStatsQuery) error {
	if query.SubscriptionID == 0 {
		return errors.NewValidationError("subscription_id is required")
	}

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
		return errors.NewValidationError("page_size must be non-negative")
	}

	return nil
}

func (uc *GetSubscriptionTrafficStatsUseCase) buildFilter(query GetSubscriptionTrafficStatsQuery) subscription.TrafficStatsFilter {
	page := query.Page
	if page == 0 {
		page = 1
	}

	pageSize := query.PageSize
	if pageSize == 0 {
		pageSize = 100
	}

	filter := subscription.TrafficStatsFilter{
		SubscriptionID: &query.SubscriptionID,
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
