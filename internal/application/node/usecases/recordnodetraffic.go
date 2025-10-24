package usecases

import (
	"context"
	"time"

	"orris/internal/domain/node"
	"orris/internal/domain/shared/events"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type RecordNodeTrafficCommand struct {
	NodeID         uint
	UserID         *uint
	SubscriptionID *uint
	Upload         uint64
	Download       uint64
}

type RecordNodeTrafficUseCase struct {
	trafficRepo     node.NodeTrafficRepository
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

func NewRecordNodeTrafficUseCase(
	trafficRepo node.NodeTrafficRepository,
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *RecordNodeTrafficUseCase {
	return &RecordNodeTrafficUseCase{
		trafficRepo:     trafficRepo,
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

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
		NodeID:   &cmd.NodeID,
		UserID:   cmd.UserID,
		From:     period,
		To:       period.Add(time.Hour),
		Page:     1,
		PageSize: 1,
	}

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
