package subscription

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/shared/query"
)

type SubscriptionUsageRepository interface {
	RecordUsage(ctx context.Context, usage *SubscriptionUsage) error
	GetUsageStats(ctx context.Context, filter UsageStatsFilter) ([]*SubscriptionUsage, error)
	GetTotalUsage(ctx context.Context, resourceType string, resourceID uint, from, to time.Time) (*UsageSummary, error)
	// GetTotalUsageBySubscriptionIDs retrieves total usage for a resource type across multiple subscriptions
	GetTotalUsageBySubscriptionIDs(ctx context.Context, resourceType string, subscriptionIDs []uint, from, to time.Time) (*UsageSummary, error)
	AggregateDaily(ctx context.Context, date time.Time) error
	AggregateMonthly(ctx context.Context, year int, month int) error
	GetDailyStats(ctx context.Context, resourceType string, resourceID uint, from, to time.Time) ([]*SubscriptionUsage, error)
	GetMonthlyStats(ctx context.Context, resourceType string, resourceID uint, year int) ([]*SubscriptionUsage, error)
	DeleteOldRecords(ctx context.Context, before time.Time) error

	// Admin analytics methods
	// GetPlatformTotalUsage retrieves total usage across the entire platform
	GetPlatformTotalUsage(ctx context.Context, resourceType *string, from, to time.Time) (*UsageSummary, error)
	// GetUsageGroupedBySubscription retrieves usage data grouped by subscription with pagination
	GetUsageGroupedBySubscription(ctx context.Context, resourceType *string, from, to time.Time, page, pageSize int) ([]SubscriptionUsageSummary, int64, error)
	// GetUsageGroupedByResourceID retrieves usage data grouped by resource ID with pagination
	GetUsageGroupedByResourceID(ctx context.Context, resourceType string, from, to time.Time, page, pageSize int) ([]ResourceUsageSummary, int64, error)
	// GetTopSubscriptionsByUsage retrieves top N subscriptions by total usage
	GetTopSubscriptionsByUsage(ctx context.Context, resourceType *string, from, to time.Time, limit int) ([]SubscriptionUsageSummary, error)
	// GetUsageTrend retrieves usage trend data with specified granularity (hour/day/month)
	GetUsageTrend(ctx context.Context, resourceType *string, from, to time.Time, granularity string) ([]UsageTrendPoint, error)
}

type UsageStatsFilter struct {
	query.PageFilter
	ResourceType   *string
	ResourceID     *uint
	SubscriptionID *uint
	From           time.Time
	To             time.Time
	Period         *string
}

type UsageSummary struct {
	ResourceType string
	ResourceID   uint
	Upload       uint64
	Download     uint64
	Total        uint64
	From         time.Time
	To           time.Time
}

// SubscriptionUsageSummary represents aggregated usage data grouped by subscription
type SubscriptionUsageSummary struct {
	SubscriptionID uint
	Upload         uint64
	Download       uint64
	Total          uint64
}

// ResourceUsageSummary represents aggregated usage data grouped by resource
type ResourceUsageSummary struct {
	ResourceType string
	ResourceID   uint
	Upload       uint64
	Download     uint64
	Total        uint64
}

// UsageTrendPoint represents usage data at a specific time period
type UsageTrendPoint struct {
	Period   time.Time
	Upload   uint64
	Download uint64
	Total    uint64
}
