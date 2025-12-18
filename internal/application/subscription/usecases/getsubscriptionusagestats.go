package usecases

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetSubscriptionUsageStatsQuery represents the query parameters for subscription usage stats
type GetSubscriptionUsageStatsQuery struct {
	SubscriptionID uint
	From           time.Time
	To             time.Time
	Granularity    string // hour, day, month
	Page           int
	PageSize       int
}

// SubscriptionUsageStatsRecord represents a single usage stats record
type SubscriptionUsageStatsRecord struct {
	ResourceType string    `json:"resource_type"`
	ResourceID   uint      `json:"resource_id"`
	Upload       uint64    `json:"upload"`
	Download     uint64    `json:"download"`
	Total        uint64    `json:"total"`
	Period       time.Time `json:"period"`
}

// SubscriptionUsageSummary represents aggregated usage summary
type SubscriptionUsageSummary struct {
	TotalUpload   uint64 `json:"total_upload"`
	TotalDownload uint64 `json:"total_download"`
	Total         uint64 `json:"total"`
}

// GetSubscriptionUsageStatsResponse represents the response for subscription usage stats
type GetSubscriptionUsageStatsResponse struct {
	Records  []*SubscriptionUsageStatsRecord `json:"records"`
	Summary  *SubscriptionUsageSummary       `json:"summary"`
	Total    int                             `json:"total"`
	Page     int                             `json:"page"`
	PageSize int                             `json:"page_size"`
}

// GetSubscriptionUsageStatsUseCase handles retrieving usage statistics for a subscription
type GetSubscriptionUsageStatsUseCase struct {
	usageRepo subscription.SubscriptionUsageRepository
	logger    logger.Interface
}

// NewGetSubscriptionUsageStatsUseCase creates a new GetSubscriptionUsageStatsUseCase
func NewGetSubscriptionUsageStatsUseCase(
	usageRepo subscription.SubscriptionUsageRepository,
	logger logger.Interface,
) *GetSubscriptionUsageStatsUseCase {
	return &GetSubscriptionUsageStatsUseCase{
		usageRepo: usageRepo,
		logger:    logger,
	}
}

// Execute retrieves usage statistics for a subscription
func (uc *GetSubscriptionUsageStatsUseCase) Execute(
	ctx context.Context,
	query GetSubscriptionUsageStatsQuery,
) (*GetSubscriptionUsageStatsResponse, error) {
	uc.logger.Infow("fetching subscription usage stats",
		"subscription_id", query.SubscriptionID,
		"from", query.From,
		"to", query.To,
		"granularity", query.Granularity,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid subscription usage stats query", "error", err)
		return nil, err
	}

	filter := uc.buildFilter(query)

	usageRecords, err := uc.usageRepo.GetUsageStats(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to fetch subscription usage stats", "error", err)
		return nil, errors.NewInternalError("failed to fetch usage statistics")
	}

	// Convert records and calculate summary
	records := make([]*SubscriptionUsageStatsRecord, 0, len(usageRecords))
	summary := &SubscriptionUsageSummary{}

	for _, record := range usageRecords {
		records = append(records, &SubscriptionUsageStatsRecord{
			ResourceType: record.ResourceType(),
			ResourceID:   record.ResourceID(),
			Upload:       record.Upload(),
			Download:     record.Download(),
			Total:        record.Total(),
			Period:       record.Period(),
		})
		summary.TotalUpload += record.Upload()
		summary.TotalDownload += record.Download()
		summary.Total += record.Total()
	}

	response := &GetSubscriptionUsageStatsResponse{
		Records:  records,
		Summary:  summary,
		Total:    len(records),
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}

	uc.logger.Infow("subscription usage stats fetched successfully",
		"subscription_id", query.SubscriptionID,
		"count", len(records),
	)

	return response, nil
}

func (uc *GetSubscriptionUsageStatsUseCase) validateQuery(query GetSubscriptionUsageStatsQuery) error {
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

func (uc *GetSubscriptionUsageStatsUseCase) buildFilter(query GetSubscriptionUsageStatsQuery) subscription.UsageStatsFilter {
	page := query.Page
	if page == 0 {
		page = 1
	}

	pageSize := query.PageSize
	if pageSize == 0 {
		pageSize = 100
	}

	filter := subscription.UsageStatsFilter{
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
