package usecases

import (
	"context"
	"time"

	"orris/internal/domain/node"
	"orris/internal/domain/shared/events"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type ResetNodeTrafficCommand struct {
	NodeID         uint
	UserID         *uint
	SubscriptionID *uint
	Reason         string
	OperatorID     uint
}

type ResetNodeTrafficResult struct {
	NodeID         uint      `json:"node_id"`
	UserID         *uint     `json:"user_id,omitempty"`
	SubscriptionID *uint     `json:"subscription_id,omitempty"`
	PreviousTotal  uint64    `json:"previous_total"`
	ResetAt        time.Time `json:"reset_at"`
	Reason         string    `json:"reason"`
}

type ResetNodeTrafficUseCase struct {
	trafficRepo     node.NodeTrafficRepository
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

func NewResetNodeTrafficUseCase(
	trafficRepo node.NodeTrafficRepository,
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *ResetNodeTrafficUseCase {
	return &ResetNodeTrafficUseCase{
		trafficRepo:     trafficRepo,
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

func (uc *ResetNodeTrafficUseCase) Execute(
	ctx context.Context,
	cmd ResetNodeTrafficCommand,
) (*ResetNodeTrafficResult, error) {
	uc.logger.Infow("resetting node traffic",
		"node_id", cmd.NodeID,
		"user_id", cmd.UserID,
		"operator_id", cmd.OperatorID,
		"reason", cmd.Reason,
	)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid reset traffic command", "error", err)
		return nil, err
	}

	filter := node.TrafficStatsFilter{
		NodeID:         &cmd.NodeID,
		UserID:         cmd.UserID,
		SubscriptionID: cmd.SubscriptionID,
		From:           time.Time{},
		To:             time.Now(),
	}
	filter.Page = 1
	filter.PageSize = 10000

	trafficRecords, err := uc.trafficRepo.GetTrafficStats(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to fetch traffic records", "error", err)
		return nil, errors.NewInternalError("failed to fetch traffic records")
	}

	var previousTotal uint64
	for _, record := range trafficRecords {
		previousTotal += record.Total()

		if err := record.Reset(); err != nil {
			uc.logger.Errorw("failed to reset traffic record", "error", err, "record_id", record.ID())
			return nil, errors.NewInternalError("failed to reset traffic record")
		}

		if err := uc.trafficRepo.RecordTraffic(ctx, record); err != nil {
			uc.logger.Errorw("failed to persist reset traffic", "error", err, "record_id", record.ID())
			return nil, errors.NewInternalError("failed to persist traffic reset")
		}
	}

	result := &ResetNodeTrafficResult{
		NodeID:         cmd.NodeID,
		UserID:         cmd.UserID,
		SubscriptionID: cmd.SubscriptionID,
		PreviousTotal:  previousTotal,
		ResetAt:        time.Now(),
		Reason:         cmd.Reason,
	}

	uc.logger.Infow("node traffic reset successfully",
		"node_id", cmd.NodeID,
		"records_reset", len(trafficRecords),
		"previous_total", previousTotal,
	)

	return result, nil
}

func (uc *ResetNodeTrafficUseCase) validateCommand(cmd ResetNodeTrafficCommand) error {
	if cmd.NodeID == 0 {
		return errors.NewValidationError("node ID is required")
	}

	if cmd.OperatorID == 0 {
		return errors.NewValidationError("operator ID is required")
	}

	if cmd.Reason == "" {
		return errors.NewValidationError("reset reason is required")
	}

	return nil
}
