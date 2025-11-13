package usecases

import (
	"context"
	"time"

	"orris/internal/domain/node"
	"orris/internal/infrastructure/cache"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// DEPRECATED: This use case is deprecated as node-level traffic management has been removed.
// Traffic tracking is now handled at the subscription level through NodeTraffic domain entity.
// This file is kept for backward compatibility and will be removed in a future version.
//
// Migration path:
// - Use subscription-level traffic tracking via NodeTraffic entity
// - Traffic data is stored in node_traffic table with subscription association
// - For node statistics, aggregate from NodeTraffic records
type RecordNodeTrafficCommand struct {
	NodeID         uint
	UserID         *uint
	SubscriptionID *uint
	Upload         uint64
	Download       uint64
}

// DEPRECATED: See RecordNodeTrafficCommand deprecation notice
type RecordNodeTrafficUseCase struct {
	trafficCache cache.TrafficCache
	trafficRepo  node.NodeTrafficRepository
	logger       logger.Interface
}

// DEPRECATED: See RecordNodeTrafficCommand deprecation notice
func NewRecordNodeTrafficUseCase(
	trafficCache cache.TrafficCache,
	trafficRepo node.NodeTrafficRepository,
	logger logger.Interface,
) *RecordNodeTrafficUseCase {
	return &RecordNodeTrafficUseCase{
		trafficCache: trafficCache,
		trafficRepo:  trafficRepo,
		logger:       logger,
	}
}

// DEPRECATED: See RecordNodeTrafficCommand deprecation notice
func (uc *RecordNodeTrafficUseCase) Execute(ctx context.Context, cmd RecordNodeTrafficCommand) error {
	uc.logger.Infow("recording node traffic",
		"node_id", cmd.NodeID,
		"upload", cmd.Upload,
		"download", cmd.Download,
	)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid record traffic command", "error", err)
		return err
	}

	// 1. Record to Redis cache (high performance, real-time tracking)
	err := uc.trafficCache.IncrementTraffic(ctx, cmd.NodeID, cmd.Upload, cmd.Download)
	if err != nil {
		uc.logger.Errorw("failed to record traffic to redis",
			"node_id", cmd.NodeID,
			"error", err,
		)
		// Don't fail the request, Redis is cache layer
		// Traffic will still be recorded in NodeTraffic table below
	}

	// 2. Record to NodeTraffic table for historical tracking
	period := time.Now().Truncate(time.Hour)
	existingTraffic, err := uc.findOrCreateTraffic(ctx, cmd, period)
	if err != nil {
		uc.logger.Errorw("failed to find or create traffic record", "error", err)
		return err
	}

	if err := existingTraffic.Accumulate(cmd.Upload, cmd.Download); err != nil {
		uc.logger.Errorw("failed to accumulate traffic", "error", err)
		return err
	}

	if err := uc.trafficRepo.RecordTraffic(ctx, existingTraffic); err != nil {
		uc.logger.Errorw("failed to persist traffic record", "error", err)
		return errors.NewInternalError("failed to record traffic")
	}

	uc.logger.Infow("traffic recorded successfully",
		"node_id", cmd.NodeID,
		"total", existingTraffic.Total(),
	)

	return nil
}

func (uc *RecordNodeTrafficUseCase) validateCommand(cmd RecordNodeTrafficCommand) error {
	if cmd.NodeID == 0 {
		return errors.NewValidationError("node ID is required")
	}

	if cmd.Upload == 0 && cmd.Download == 0 {
		return errors.NewValidationError("at least one of upload or download must be non-zero")
	}

	return nil
}

func (uc *RecordNodeTrafficUseCase) findOrCreateTraffic(
	ctx context.Context,
	cmd RecordNodeTrafficCommand,
	period time.Time,
) (*node.NodeTraffic, error) {
	filter := node.TrafficStatsFilter{
		NodeID: &cmd.NodeID,
		UserID: cmd.UserID,
		From:   period,
		To:     period.Add(time.Hour),
	}
	filter.Page = 1
	filter.PageSize = 1

	existingRecords, err := uc.trafficRepo.GetTrafficStats(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(existingRecords) > 0 {
		return existingRecords[0], nil
	}

	newTraffic, err := node.NewNodeTraffic(cmd.NodeID, cmd.UserID, cmd.SubscriptionID, period)
	if err != nil {
		return nil, errors.NewInternalError("failed to create traffic record")
	}

	return newTraffic, nil
}
