package usecases

import (
	"context"
	"time"

	dto "github.com/orris-inc/orris/internal/application/admin/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
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
	usageStatsRepo     subscription.SubscriptionUsageStatsRepository
	hourlyTrafficCache cache.HourlyTrafficCache
	logger             logger.Interface
}

// NewGetTrafficTrendUseCase creates a new GetTrafficTrendUseCase
func NewGetTrafficTrendUseCase(
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyTrafficCache cache.HourlyTrafficCache,
	logger logger.Interface,
) *GetTrafficTrendUseCase {
	return &GetTrafficTrendUseCase{
		usageStatsRepo:     usageStatsRepo,
		hourlyTrafficCache: hourlyTrafficCache,
		logger:             logger,
	}
}

// Execute retrieves traffic trend data
func (uc *GetTrafficTrendUseCase) Execute(
	ctx context.Context,
	query GetTrafficTrendQuery,
) (*dto.TrafficTrendResponse, error) {
	uc.logger.Debugw("fetching traffic trend",
		"from", query.From,
		"to", query.To,
		"resource_type", query.ResourceType,
		"granularity", query.Granularity,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Warnw("invalid traffic trend query", "error", err)
		return nil, err
	}

	var points []dto.TrafficTrendPoint

	if query.Granularity == "hour" {
		// Get hourly data from Redis
		hourlyPoints, err := uc.getHourlyTrendFromRedis(ctx, query)
		if err != nil {
			uc.logger.Errorw("failed to fetch hourly traffic trend from Redis", "error", err)
			return nil, errors.NewInternalError("failed to fetch traffic trend")
		}
		points = hourlyPoints
	} else {
		// Adjust 'to' time to end of day to include all records from that day
		adjustedTo := biztime.EndOfDayUTC(query.To)

		// Get usage trend data from subscription_usage_stats table
		trendPoints, err := uc.usageStatsRepo.GetUsageTrend(
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
		points = make([]dto.TrafficTrendPoint, 0, len(trendPoints))
		for _, point := range trendPoints {
			points = append(points, dto.TrafficTrendPoint{
				Period:   uc.formatPeriod(point.Period, query.Granularity),
				Upload:   point.Upload,
				Download: point.Download,
				Total:    point.Total,
			})
		}
	}

	response := &dto.TrafficTrendResponse{
		Points:      points,
		Granularity: query.Granularity,
	}

	uc.logger.Debugw("traffic trend fetched successfully",
		"count", len(points),
		"granularity", query.Granularity,
	)

	return response, nil
}

// getHourlyTrendFromRedis retrieves hourly traffic data from Redis HourlyTrafficCache.
// It aggregates all subscriptions' hourly data within the specified time range.
func (uc *GetTrafficTrendUseCase) getHourlyTrendFromRedis(
	ctx context.Context,
	query GetTrafficTrendQuery,
) ([]dto.TrafficTrendPoint, error) {
	// Adjust 'to' time to end of day to include all hours from that day
	adjustedTo := biztime.EndOfDayUTC(query.To)

	// Truncate to hour boundaries in business timezone
	fromHour := biztime.TruncateToHourInBiz(query.From)
	toHour := biztime.TruncateToHourInBiz(adjustedTo)

	// Cap time range to last 48 hours (Redis TTL constraint)
	now := biztime.NowUTC()
	maxFrom := now.Add(-48 * time.Hour)
	if fromHour.Before(maxFrom) {
		fromHour = maxFrom
	}
	if toHour.After(now) {
		toHour = now
	}

	// Build a map to aggregate traffic by hour
	hourlyAggregates := make(map[string]*dto.TrafficTrendPoint)

	// Iterate through each hour
	current := fromHour
	for !current.After(toHour) {
		// Get all traffic data for this hour
		hourlyData, err := uc.hourlyTrafficCache.GetAllHourlyTraffic(ctx, current)
		if err != nil {
			uc.logger.Warnw("failed to get hourly traffic data",
				"hour", current,
				"error", err,
			)
			current = current.Add(time.Hour)
			continue
		}

		// Filter by resource type and aggregate
		var upload, download uint64
		for _, data := range hourlyData {
			// Filter by resource type if specified
			if query.ResourceType != nil && data.ResourceType != *query.ResourceType {
				continue
			}
			// Safe conversion: treat negative int64 values as 0 to prevent uint64 underflow
			if data.Upload > 0 {
				upload += uint64(data.Upload)
			}
			if data.Download > 0 {
				download += uint64(data.Download)
			}
		}

		// Only add if there's data
		if upload > 0 || download > 0 {
			hourKey := uc.formatPeriod(current, "hour")
			hourlyAggregates[hourKey] = &dto.TrafficTrendPoint{
				Period:   hourKey,
				Upload:   upload,
				Download: download,
				Total:    upload + download,
			}
		}

		current = current.Add(time.Hour)
	}

	// Convert map to sorted slice
	points := make([]dto.TrafficTrendPoint, 0, len(hourlyAggregates))
	current = fromHour
	for !current.After(toHour) {
		hourKey := uc.formatPeriod(current, "hour")
		if point, exists := hourlyAggregates[hourKey]; exists {
			points = append(points, *point)
		}
		current = current.Add(time.Hour)
	}

	return points, nil
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
	// Convert UTC to business timezone for display
	// The period represents a business timezone boundary stored as UTC
	bizTime := biztime.ToBizTimezone(t)
	switch granularity {
	case "hour":
		return bizTime.Format("2006-01-02 15:00")
	case "day":
		return bizTime.Format("2006-01-02")
	case "month":
		return bizTime.Format("2006-01")
	default:
		return bizTime.Format("2006-01-02 15:04:05")
	}
}
