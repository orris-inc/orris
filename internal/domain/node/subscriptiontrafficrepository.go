package node

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/shared/query"
)

type SubscriptionTrafficRepository interface {
	RecordTraffic(ctx context.Context, traffic *SubscriptionTraffic) error
	GetTrafficStats(ctx context.Context, filter TrafficStatsFilter) ([]*SubscriptionTraffic, error)
	GetTotalTraffic(ctx context.Context, nodeID uint, from, to time.Time) (*TrafficSummary, error)
	AggregateDaily(ctx context.Context, date time.Time) error
	AggregateMonthly(ctx context.Context, year int, month int) error
	GetDailyStats(ctx context.Context, nodeID uint, from, to time.Time) ([]*SubscriptionTraffic, error)
	GetMonthlyStats(ctx context.Context, nodeID uint, year int) ([]*SubscriptionTraffic, error)
	DeleteOldRecords(ctx context.Context, before time.Time) error
}

type TrafficStatsFilter struct {
	query.PageFilter
	NodeID         *uint
	UserID         *uint
	SubscriptionID *uint
	From           time.Time
	To             time.Time
	Period         *string
}

type TrafficSummary struct {
	NodeID   uint
	Upload   uint64
	Download uint64
	Total    uint64
	From     time.Time
	To       time.Time
}
