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

// GetTrafficOverviewQuery represents the query parameters for traffic overview
type GetTrafficOverviewQuery struct {
	From         time.Time
	To           time.Time
	ResourceType *string
}

// GetTrafficOverviewUseCase handles retrieving global traffic overview
type GetTrafficOverviewUseCase struct {
	usageStatsRepo     subscription.SubscriptionUsageStatsRepository
	hourlyTrafficCache cache.HourlyTrafficCache
	logger             logger.Interface
}

// NewGetTrafficOverviewUseCase creates a new GetTrafficOverviewUseCase
func NewGetTrafficOverviewUseCase(
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyTrafficCache cache.HourlyTrafficCache,
	logger logger.Interface,
) *GetTrafficOverviewUseCase {
	return &GetTrafficOverviewUseCase{
		usageStatsRepo:     usageStatsRepo,
		hourlyTrafficCache: hourlyTrafficCache,
		logger:             logger,
	}
}

// Execute retrieves global traffic overview
func (uc *GetTrafficOverviewUseCase) Execute(
	ctx context.Context,
	query GetTrafficOverviewQuery,
) (*dto.TrafficOverviewResponse, error) {
	uc.logger.Debugw("fetching traffic overview",
		"from", query.From,
		"to", query.To,
		"resource_type", query.ResourceType,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Warnw("invalid traffic overview query", "error", err)
		return nil, err
	}

	// Adjust 'to' time to end of day to include all records from that day
	adjustedTo := biztime.EndOfDayUTC(query.To)

	// Calculate time boundaries
	now := biztime.NowUTC()
	// Redis stores data for the last 48 hours
	redisDataStart := now.Add(-48 * time.Hour)

	var totalUpload, totalDownload, totalTraffic uint64

	// Check if query range overlaps with Redis data window (last 48 hours)
	if !adjustedTo.Before(redisDataStart) {
		// Query overlaps with Redis data window - get data from Redis
		redisFrom := query.From
		if redisFrom.Before(redisDataStart) {
			redisFrom = redisDataStart
		}

		resourceType := ""
		if query.ResourceType != nil {
			resourceType = *query.ResourceType
		}

		redisTraffic, err := uc.hourlyTrafficCache.GetPlatformTotalTraffic(ctx, resourceType, redisFrom, adjustedTo)
		if err != nil {
			uc.logger.Warnw("failed to get traffic from Redis, continuing with MySQL only",
				"error", err,
			)
		} else {
			totalUpload += redisTraffic.Upload
			totalDownload += redisTraffic.Download
			totalTraffic += redisTraffic.Total
			uc.logger.Debugw("got traffic from Redis",
				"redis_from", redisFrom,
				"redis_to", adjustedTo,
				"upload", redisTraffic.Upload,
				"download", redisTraffic.Download,
			)
		}
	}

	// Get historical data from MySQL (data older than 48 hours)
	mysqlTo := adjustedTo
	if !adjustedTo.Before(redisDataStart) && !query.From.After(redisDataStart) {
		// Query spans both historical and Redis window - only query MySQL for data before Redis window
		mysqlTo = redisDataStart.Add(-time.Nanosecond)
	}

	// Only query MySQL if there's a valid historical range (before Redis data window)
	if query.From.Before(redisDataStart) && mysqlTo.After(query.From) {
		totalUsage, err := uc.usageStatsRepo.GetPlatformTotalUsageByResourceType(ctx, query.ResourceType, query.From, mysqlTo)
		if err != nil {
			uc.logger.Errorw("failed to fetch platform total usage", "error", err)
			return nil, errors.NewInternalError("failed to fetch platform usage")
		}
		totalUpload += totalUsage.Upload
		totalDownload += totalUsage.Download
		totalTraffic += totalUsage.Total
	}

	response := &dto.TrafficOverviewResponse{
		TotalUpload:   totalUpload,
		TotalDownload: totalDownload,
		TotalTraffic:  totalTraffic,
	}

	uc.logger.Debugw("traffic overview fetched successfully",
		"total_traffic", response.TotalTraffic,
	)

	return response, nil
}

func (uc *GetTrafficOverviewUseCase) validateQuery(query GetTrafficOverviewQuery) error {
	if query.From.IsZero() {
		return errors.NewValidationError("from time is required")
	}

	if query.To.IsZero() {
		return errors.NewValidationError("to time is required")
	}

	if query.To.Before(query.From) {
		return errors.NewValidationError("to time must be after from time")
	}

	return nil
}
