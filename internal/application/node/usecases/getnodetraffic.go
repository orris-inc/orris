package usecases

import (
	"context"

	"orris/internal/infrastructure/cache"
	"orris/internal/shared/logger"
)

type GetNodeTrafficQuery struct {
	NodeID uint
}

type GetNodeTrafficResult struct {
	NodeID       uint   `json:"node_id"`
	TrafficUsed  uint64 `json:"traffic_used"`
	TrafficLimit uint64 `json:"traffic_limit"`
	Exceeded     bool   `json:"exceeded"`
}

type GetNodeTrafficUseCase struct {
	trafficCache cache.TrafficCache
	logger       logger.Interface
}

func NewGetNodeTrafficUseCase(
	trafficCache cache.TrafficCache,
	logger logger.Interface,
) *GetNodeTrafficUseCase {
	return &GetNodeTrafficUseCase{
		trafficCache: trafficCache,
		logger:       logger,
	}
}

func (uc *GetNodeTrafficUseCase) Execute(ctx context.Context, query GetNodeTrafficQuery) (*GetNodeTrafficResult, error) {
	uc.logger.Infow("getting node traffic", "node_id", query.NodeID)

	// Get real-time traffic from Redis + MySQL
	trafficUsed, err := uc.trafficCache.GetNodeTraffic(ctx, query.NodeID)
	if err != nil {
		uc.logger.Errorw("failed to get node traffic", "error", err, "node_id", query.NodeID)
		return nil, err
	}

	// TODO: Get traffic limit from node
	// For now, return traffic used only

	return &GetNodeTrafficResult{
		NodeID:      query.NodeID,
		TrafficUsed: trafficUsed,
	}, nil
}
