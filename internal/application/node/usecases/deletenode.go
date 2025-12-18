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
	ShortID string // External API identifier
	Force   bool
}

type DeleteNodeResult struct {
	NodeID    uint
	DeletedAt string
}

type DeleteNodeUseCase struct {
	nodeRepo      node.NodeRepository
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewDeleteNodeUseCase(
	nodeRepo node.NodeRepository,
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *DeleteNodeUseCase {
	return &DeleteNodeUseCase{
		nodeRepo:      nodeRepo,
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *DeleteNodeUseCase) Execute(ctx context.Context, cmd DeleteNodeCommand) (*DeleteNodeResult, error) {
	uc.logger.Infow("executing delete node use case", "short_id", cmd.ShortID, "force", cmd.Force)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid delete node command", "error", err, "short_id", cmd.ShortID)
		return nil, err
	}

	// Retrieve the node
	existingNode, err := uc.nodeRepo.GetByShortID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get node by short ID", "short_id", cmd.ShortID, "error", err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	nodeID := existingNode.ID()

	// Check if node is part of any node groups (business validation)
	if !cmd.Force {
		if err := uc.checkNodeGroupAssociations(ctx, nodeID); err != nil {
			return nil, err
		}
	}

	// Soft delete the node
	// Note: Foreign key constraints have been removed to support soft deletes.
	// Associated records in node_group_nodes will remain but queries should filter by deleted_at.
	if err := uc.nodeRepo.Delete(ctx, nodeID); err != nil {
		uc.logger.Errorw("failed to delete node from database", "error", err, "short_id", cmd.ShortID)
		return nil, fmt.Errorf("failed to delete node: %w", err)
	}

	uc.logger.Infow("node deleted successfully",
		"short_id", cmd.ShortID,
		"name", existingNode.Name(),
		"address", existingNode.ServerAddress().Value(),
	)

	return &DeleteNodeResult{
		NodeID:    nodeID,
		DeletedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (uc *DeleteNodeUseCase) validateCommand(cmd DeleteNodeCommand) error {
	if cmd.ShortID == "" {
		return errors.NewValidationError("short ID must be provided")
	}

	return nil
}

// checkNodeGroupAssociations checks if the node is part of any node groups
// This provides business-level protection against accidentally deleting nodes that are in use
func (uc *DeleteNodeUseCase) checkNodeGroupAssociations(ctx context.Context, nodeID uint) error {
	inGroup, err := uc.nodeGroupRepo.IsNodeInAnyGroup(ctx, nodeID)
	if err != nil {
		uc.logger.Errorw("failed to check node group associations", "error", err, "node_id", nodeID)
		return fmt.Errorf("failed to check node group associations: %w", err)
	}
	if inGroup {
		return errors.NewValidationError("cannot delete node that is part of a node group. Use force=true to override")
	}
	return nil
}
