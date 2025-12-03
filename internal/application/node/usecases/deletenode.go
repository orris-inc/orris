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
	NodeID uint
	Force  bool
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
	uc.logger.Infow("executing delete node use case", "node_id", cmd.NodeID, "force", cmd.Force)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid delete node command", "error", err, "node_id", cmd.NodeID)
		return nil, err
	}

	// Check if node exists
	existingNode, err := uc.nodeRepo.GetByID(ctx, cmd.NodeID)
	if err != nil {
		uc.logger.Errorw("failed to get node", "error", err, "node_id", cmd.NodeID)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Check if node is part of any node groups (business validation)
	if !cmd.Force {
		if err := uc.checkNodeGroupAssociations(ctx, cmd.NodeID); err != nil {
			return nil, err
		}
	}

	// Soft delete the node
	// Note: Foreign key constraints have been removed to support soft deletes.
	// Associated records in node_group_nodes will remain but queries should filter by deleted_at.
	if err := uc.nodeRepo.Delete(ctx, cmd.NodeID); err != nil {
		uc.logger.Errorw("failed to delete node from database", "error", err, "node_id", cmd.NodeID)
		return nil, fmt.Errorf("failed to delete node: %w", err)
	}

	uc.logger.Infow("node deleted successfully",
		"node_id", cmd.NodeID,
		"name", existingNode.Name(),
		"address", existingNode.ServerAddress().Value(),
	)

	return &DeleteNodeResult{
		NodeID:    cmd.NodeID,
		DeletedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (uc *DeleteNodeUseCase) validateCommand(cmd DeleteNodeCommand) error {
	if cmd.NodeID == 0 {
		return errors.NewValidationError("node id is required")
	}

	return nil
}

// checkNodeGroupAssociations checks if the node is part of any node groups
// This provides business-level protection against accidentally deleting nodes that are in use
func (uc *DeleteNodeUseCase) checkNodeGroupAssociations(ctx context.Context, nodeID uint) error {
	// Get all node groups and check if any contains this node
	filter := node.NodeGroupFilter{}
	filter.Page = 1
	filter.PageSize = 1000 // Large enough to get all groups

	groups, _, err := uc.nodeGroupRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list node groups", "error", err, "node_id", nodeID)
		return fmt.Errorf("failed to check node group associations: %w", err)
	}

	// Check if any group contains this node
	for _, group := range groups {
		if group.ContainsNode(nodeID) {
			return errors.NewValidationError(fmt.Sprintf(
				"cannot delete node that is part of node group '%s'. Use force=true to override",
				group.Name(),
			))
		}
	}

	return nil
}
