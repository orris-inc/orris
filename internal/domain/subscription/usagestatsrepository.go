package subscription

import (
	"context"
	"time"
)

// SubscriptionUsageStatsRepository defines the interface for aggregated usage statistics persistence
type SubscriptionUsageStatsRepository interface {
	// Upsert inserts or updates an aggregated usage stats record
	Upsert(ctx context.Context, stats *SubscriptionUsageStats) error

	// GetBySubscriptionID retrieves aggregated usage stats for a subscription within a time range
	GetBySubscriptionID(ctx context.Context, subscriptionID uint, granularity Granularity, from, to time.Time) ([]*SubscriptionUsageStats, error)

	// GetTotalBySubscriptionIDs retrieves total aggregated usage across multiple subscriptions.
	// If resourceType is nil, returns usage for all resource types (used for Hybrid plans).
	// If resourceType is specified, returns usage only for that resource type (used for Forward/Node plans).
	GetTotalBySubscriptionIDs(ctx context.Context, subscriptionIDs []uint, resourceType *string, granularity Granularity, from, to time.Time) (*UsageSummary, error)

	// GetByResourceID retrieves aggregated usage stats for a specific resource within a time range
	GetByResourceID(ctx context.Context, resourceType string, resourceID uint, granularity Granularity, from, to time.Time) ([]*SubscriptionUsageStats, error)

	// GetTotalByResourceID retrieves total aggregated usage for a specific resource within a time range
	GetTotalByResourceID(ctx context.Context, resourceType string, resourceID uint, granularity Granularity, from, to time.Time) (*UsageSummary, error)

	// DeleteOldRecords deletes aggregated usage records older than the specified time
	DeleteOldRecords(ctx context.Context, granularity Granularity, before time.Time) error

	// GetDailyStatsByPeriod retrieves daily aggregated stats within a time range using cursor-based pagination.
	// lastID is the ID of the last record from the previous page (use 0 for the first page).
	// limit is the maximum number of records to return.
	// Used by monthly aggregation to aggregate from daily stats.
	GetDailyStatsByPeriod(ctx context.Context, from, to time.Time, lastID uint, limit int) ([]*SubscriptionUsageStats, error)

	// GetPlatformTotalUsage retrieves total platform-wide usage across all subscriptions within a time range.
	// Used for admin summary notifications.
	GetPlatformTotalUsage(ctx context.Context, granularity Granularity, from, to time.Time) (*UsageSummary, error)

	// ========== Admin Analytics Methods ==========

	// GetPlatformTotalUsageByResourceType retrieves total platform-wide usage filtered by resource type.
	// If resourceType is nil, returns usage for all resource types.
	GetPlatformTotalUsageByResourceType(ctx context.Context, resourceType *string, from, to time.Time) (*UsageSummary, error)

	// GetUsageGroupedBySubscription retrieves aggregated usage grouped by subscription with pagination
	GetUsageGroupedBySubscription(ctx context.Context, resourceType *string, from, to time.Time, page, pageSize int) ([]SubscriptionUsageSummary, int64, error)

	// GetUsageGroupedByResourceID retrieves aggregated usage grouped by resource ID with pagination
	GetUsageGroupedByResourceID(ctx context.Context, resourceType string, from, to time.Time, page, pageSize int) ([]ResourceUsageSummary, int64, error)

	// GetTopSubscriptionsByUsage retrieves top N subscriptions by total usage
	GetTopSubscriptionsByUsage(ctx context.Context, resourceType *string, from, to time.Time, limit int) ([]SubscriptionUsageSummary, error)

	// GetUsageTrend retrieves usage trend data grouped by time period with specified granularity (day/month)
	GetUsageTrend(ctx context.Context, resourceType *string, from, to time.Time, granularity string) ([]UsageTrendPoint, error)
}
