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

	// GetTotalBySubscriptionIDs retrieves total aggregated usage across multiple subscriptions
	GetTotalBySubscriptionIDs(ctx context.Context, subscriptionIDs []uint, granularity Granularity, from, to time.Time) (*UsageSummary, error)

	// GetByResourceID retrieves aggregated usage stats for a specific resource within a time range
	GetByResourceID(ctx context.Context, resourceType string, resourceID uint, granularity Granularity, from, to time.Time) ([]*SubscriptionUsageStats, error)

	// DeleteOldRecords deletes aggregated usage records older than the specified time
	DeleteOldRecords(ctx context.Context, granularity Granularity, before time.Time) error

	// GetDailyStatsByPeriod retrieves daily aggregated stats within a time range using cursor-based pagination.
	// lastID is the ID of the last record from the previous page (use 0 for the first page).
	// limit is the maximum number of records to return.
	// Used by monthly aggregation to aggregate from daily stats.
	GetDailyStatsByPeriod(ctx context.Context, from, to time.Time, lastID uint, limit int) ([]*SubscriptionUsageStats, error)
}
