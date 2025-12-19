package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type DeleteNodeCommand struct {
	SID   string // External API identifier
	Force bool
}

type DeleteNodeResult struct {
	NodeID    uint
	DeletedAt string
}

type DeleteNodeUseCase struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

func NewDeleteNodeUseCase(
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *DeleteNodeUseCase {
	return &DeleteNodeUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

func (uc *DeleteNodeUseCase) Execute(ctx context.Context, cmd DeleteNodeCommand) (*DeleteNodeResult, error) {
	uc.logger.Infow("executing delete node use case", "sid", cmd.SID, "force", cmd.Force)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid delete node command", "error", err, "sid", cmd.SID)
		return nil, err
	}

	// Retrieve the node
	existingNode, err := uc.nodeRepo.GetBySID(ctx, cmd.SID)
	if err != nil {
		uc.logger.Errorw("failed to get node by SID", "sid", cmd.SID, "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	nodeID := existingNode.ID()

	// Soft delete the node
	if err := uc.nodeRepo.Delete(ctx, nodeID); err != nil {
		uc.logger.Errorw("failed to delete node from database", "error", err, "sid", cmd.SID)
		return nil, fmt.Errorf("failed to delete node: %w", err)
	}

	uc.logger.Infow("node deleted successfully",
		"sid", cmd.SID,
		"name", existingNode.Name(),
		"address", existingNode.ServerAddress().Value(),
	)

	return &DeleteNodeResult{
		NodeID:    nodeID,
		DeletedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (uc *DeleteNodeUseCase) validateCommand(cmd DeleteNodeCommand) error {
	if cmd.SID == "" {
		return errors.NewValidationError("SID must be provided")
	}

	return nil
}
