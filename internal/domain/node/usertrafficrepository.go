package node

import (
	"context"
	"time"

	"orris/internal/shared/query"
)

// UserTrafficRepository defines the interface for user traffic persistence operations
type UserTrafficRepository interface {
	// Create creates a new user traffic record
	Create(ctx context.Context, traffic *UserTraffic) error

	// BatchUpsert batch inserts or updates user traffic records (for XrayR traffic reporting)
	BatchUpsert(ctx context.Context, traffics []*UserTraffic) error

	// GetByUserAndNode retrieves user traffic by user ID, node ID, and period
	GetByUserAndNode(ctx context.Context, userID, nodeID uint, period time.Time) (*UserTraffic, error)

	// GetByUserIDWithDateRange retrieves all traffic records for a user within a date range
	GetByUserIDWithDateRange(ctx context.Context, userID uint, start, end time.Time) ([]*UserTraffic, error)

	// GetTotalByUser calculates total traffic (upload, download, total) for a user across all nodes
	GetTotalByUser(ctx context.Context, userID uint) (upload uint64, download uint64, total uint64, err error)

	// GetTotalBySubscription calculates total traffic for a subscription across all nodes
	GetTotalBySubscription(ctx context.Context, subscriptionID uint) (upload uint64, download uint64, total uint64, err error)

	// IncrementTraffic increments traffic for a user on a specific node (atomic operation)
	IncrementTraffic(ctx context.Context, userID, nodeID uint, upload, download uint64) error

	// DeleteOldRecords deletes traffic records older than the specified time
	DeleteOldRecords(ctx context.Context, before time.Time) error

	// GetTopUsers retrieves top users by traffic usage within a time range
	GetTopUsers(ctx context.Context, limit int, from, to time.Time) ([]*UserTraffic, error)
}

// UserTrafficFilter represents filters for querying user traffic
type UserTrafficFilter struct {
	query.PageFilter
	UserID         *uint
	NodeID         *uint
	SubscriptionID *uint
	From           time.Time
	To             time.Time
}
