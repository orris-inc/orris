package node

import (
	"context"
	"time"
)

type NodeAccessLogRepository interface {
	Create(ctx context.Context, log *NodeAccessLog) error
	GetByID(ctx context.Context, id uint) (*NodeAccessLog, error)
	List(ctx context.Context, filter AccessLogFilter) ([]*NodeAccessLog, int64, error)

	GetByNodeID(ctx context.Context, nodeID uint, limit int) ([]*NodeAccessLog, error)
	GetByUserID(ctx context.Context, userID uint, limit int) ([]*NodeAccessLog, error)
	GetBySubscriptionID(ctx context.Context, subscriptionID uint, limit int) ([]*NodeAccessLog, error)

	CountByNodeID(ctx context.Context, nodeID uint, from, to time.Time) (int64, error)
	CountByUserID(ctx context.Context, userID uint, from, to time.Time) (int64, error)
	GetActiveConnections(ctx context.Context, nodeID uint) (int64, error)

	DeleteOldLogs(ctx context.Context, before time.Time) error
}

type AccessLogFilter struct {
	NodeID         *uint
	UserID         *uint
	SubscriptionID *uint
	ClientIP       *string
	From           time.Time
	To             time.Time
	Page           int
	PageSize       int
	SortBy         string
	SortDesc       bool
}
