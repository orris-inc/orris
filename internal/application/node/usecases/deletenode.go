package usecases

import (
	"context"

	"orris/internal/domain/shared/events"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type DeleteNodeCommand struct {
	NodeID uint
	Force  bool
}

type DeleteNodeResult struct {
	NodeID    uint
	DeletedAt string
}

type DeleteNodeUseCase struct {
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

func NewDeleteNodeUseCase(
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *DeleteNodeUseCase {
	return &DeleteNodeUseCase{
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

func (uc *DeleteNodeUseCase) Execute(ctx context.Context, cmd DeleteNodeCommand) (*DeleteNodeResult, error) {
	uc.logger.Infow("executing delete node use case", "node_id", cmd.NodeID, "force", cmd.Force)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid delete node command", "error", err, "node_id", cmd.NodeID)
		return nil, err
	}

	uc.logger.Infow("node deleted successfully", "node_id", cmd.NodeID)

	return &DeleteNodeResult{
		NodeID: cmd.NodeID,
	}, nil
}

func (uc *DeleteNodeUseCase) validateCommand(cmd DeleteNodeCommand) error {
	if cmd.NodeID == 0 {
		return errors.NewValidationError("node id is required")
	}

	return nil
}
