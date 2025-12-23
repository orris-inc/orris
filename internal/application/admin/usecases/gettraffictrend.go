package usecases

import (
	"context"
	"time"

	dto "github.com/orris-inc/orris/internal/application/admin/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// GetTrafficTrendQuery represents the query parameters for traffic trend
type GetTrafficTrendQuery struct {
	From         time.Time
	To           time.Time
	ResourceType *string
	Granularity  string // "hour", "day", "month"
}

// GetTrafficTrendUseCase handles retrieving traffic trend data
type GetTrafficTrendUseCase struct {
	usageRepo subscription.SubscriptionUsageRepository
	logger    logger.Interface
}

// NewGetTrafficTrendUseCase creates a new GetTrafficTrendUseCase
func NewGetTrafficTrendUseCase(
	usageRepo subscription.SubscriptionUsageRepository,
	logger logger.Interface,
) *GetTrafficTrendUseCase {
	return &GetTrafficTrendUseCase{
		usageRepo: usageRepo,
		logger:    logger,
	}
}

// Execute retrieves traffic trend data
func (uc *GetTrafficTrendUseCase) Execute(
	ctx context.Context,
	query GetTrafficTrendQuery,
) (*dto.TrafficTrendResponse, error) {
	uc.logger.Infow("fetching traffic trend",
		"from", query.From,
		"to", query.To,
		"resource_type", query.ResourceType,
		"granularity", query.Granularity,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid traffic trend query", "error", err)
		return nil, err
	}

	// Adjust 'to' time to end of day to include all records from that day
	adjustedTo := utils.AdjustToEndOfDay(query.To)

	// Get usage trend data
	trendPoints, err := uc.usageRepo.GetUsageTrend(
		ctx,
		query.ResourceType,
		query.From,
		adjustedTo,
		query.Granularity,
	)
	if err != nil {
		uc.logger.Errorw("failed to fetch traffic trend", "error", err)
		return nil, errors.NewInternalError("failed to fetch traffic trend")
	}

	// Convert to DTO
	points := make([]dto.TrafficTrendPoint, 0, len(trendPoints))
	for _, point := range trendPoints {
		points = append(points, dto.TrafficTrendPoint{
			Period:   uc.formatPeriod(point.Period, query.Granularity),
			Upload:   point.Upload,
			Download: point.Download,
			Total:    point.Total,
		})
	}

	response := &dto.TrafficTrendResponse{
		Points:      points,
		Granularity: query.Granularity,
	}

	uc.logger.Infow("traffic trend fetched successfully",
		"count", len(points),
		"granularity", query.Granularity,
	)

	return response, nil
}

func (uc *GetTrafficTrendUseCase) validateQuery(query GetTrafficTrendQuery) error {
	if query.From.IsZero() {
		return errors.NewValidationError("from time is required")
	}

	if query.To.IsZero() {
		return errors.NewValidationError("to time is required")
	}

	if query.To.Before(query.From) {
		return errors.NewValidationError("to time must be after from time")
	}

	if query.Granularity == "" {
		return errors.NewValidationError("granularity is required")
	}

	if query.Granularity != "hour" && query.Granularity != "day" && query.Granularity != "month" {
		return errors.NewValidationError("granularity must be one of: hour, day, month")
	}

	return nil
}

func (uc *GetTrafficTrendUseCase) formatPeriod(t time.Time, granularity string) string {
	switch granularity {
	case "hour":
		return t.Format("2006-01-02T15:00:00Z07:00")
	case "day":
		return t.Format("2006-01-02")
	case "month":
		return t.Format("2006-01")
	default:
		return t.Format(time.RFC3339)
	}
}
