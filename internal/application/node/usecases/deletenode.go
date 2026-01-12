package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/biztime"
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
	ruleRepo forward.Repository
	logger   logger.Interface
}

func NewDeleteNodeUseCase(
	nodeRepo node.NodeRepository,
	ruleRepo forward.Repository,
	logger logger.Interface,
) *DeleteNodeUseCase {
	return &DeleteNodeUseCase{
		nodeRepo: nodeRepo,
		ruleRepo: ruleRepo,
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
	if existingNode == nil {
		uc.logger.Warnw("node not found", "sid", cmd.SID)
		return nil, errors.NewNotFoundError("node not found")
	}

	nodeID := existingNode.ID()

	// Check if node is referenced by any forward rules
	if err := uc.checkNodeReferences(ctx, nodeID); err != nil {
		return nil, err
	}

	// Delete the node permanently
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
		DeletedAt: biztime.NowUTC().Format(time.RFC3339),
	}, nil
}

func (uc *DeleteNodeUseCase) validateCommand(cmd DeleteNodeCommand) error {
	if cmd.SID == "" {
		return errors.NewValidationError("SID must be provided")
	}

	return nil
}

// checkNodeReferences checks if the node is referenced by any forward rules.
func (uc *DeleteNodeUseCase) checkNodeReferences(ctx context.Context, nodeID uint) error {
	rules, err := uc.ruleRepo.ListEnabledByTargetNodeID(ctx, nodeID)
	if err != nil {
		uc.logger.Errorw("failed to check node references", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to check node references: %w", err)
	}
	if len(rules) > 0 {
		return errors.NewConflictError(fmt.Sprintf("cannot delete node: %d forward rule(s) use this node as target", len(rules)))
	}

	return nil
}
