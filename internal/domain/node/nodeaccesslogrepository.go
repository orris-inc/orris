package node

import (
	"context"
	"time"

	"orris/internal/shared/query"
)

type NodeAccessLogRepository interface {
	Create(ctx context.Context, log *NodeAccessLog) error
	GetByID(ctx context.Context, id uint) (*NodeAccessLog, error)
	List(ctx context.Context, filter AccessLogFilter) ([]*NodeAccessLog, int64, error)
	CountByNodeID(ctx context.Context, nodeID uint, from, to time.Time) (int64, error)
	CountByUserID(ctx context.Context, userID uint, from, to time.Time) (int64, error)
	GetActiveConnections(ctx context.Context, nodeID uint) (int64, error)
	DeleteOldLogs(ctx context.Context, before time.Time) error
}

type AccessLogFilter struct {
	query.BaseFilter
	NodeID         *uint
	UserID         *uint
	SubscriptionID *uint
	ClientIP       *string
	From           time.Time
	To             time.Time
}
