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
	AggregateDaily(ctx context.Context, date time.Time) error
	AggregateMonthly(ctx context.Context, year int, month int) error
	GetDailyStats(ctx context.Context, resourceType string, resourceID uint, from, to time.Time) ([]*SubscriptionUsage, error)
	GetMonthlyStats(ctx context.Context, resourceType string, resourceID uint, year int) ([]*SubscriptionUsage, error)
	DeleteOldRecords(ctx context.Context, before time.Time) error
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
