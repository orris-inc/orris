package node

import (
	"context"
	"time"
)

type NodeTrafficRepository interface {
	RecordTraffic(ctx context.Context, traffic *NodeTraffic) error
	GetTrafficStats(ctx context.Context, filter TrafficStatsFilter) ([]*NodeTraffic, error)
	GetTotalTraffic(ctx context.Context, nodeID uint, from, to time.Time) (*TrafficSummary, error)

	AggregateDaily(ctx context.Context, date time.Time) error
	AggregateMonthly(ctx context.Context, year int, month int) error

	GetDailyStats(ctx context.Context, nodeID uint, from, to time.Time) ([]*NodeTraffic, error)
	GetMonthlyStats(ctx context.Context, nodeID uint, year int) ([]*NodeTraffic, error)

	DeleteOldRecords(ctx context.Context, before time.Time) error
}

type TrafficStatsFilter struct {
	NodeID         *uint
	UserID         *uint
	SubscriptionID *uint
	From           time.Time
	To             time.Time
	Period         *string
	Page           int
	PageSize       int
}

type TrafficSummary struct {
	NodeID   uint
	Upload   uint64
	Download uint64
	Total    uint64
	From     time.Time
	To       time.Time
}
