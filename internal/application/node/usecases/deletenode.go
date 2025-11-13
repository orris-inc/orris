package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/domain/node"
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

	// Check if node is part of any node groups
	if !cmd.Force {
		if err := uc.checkNodeGroupAssociations(ctx, cmd.NodeID); err != nil {
			return nil, err
		}
	}

	// Remove node from all node groups before deletion
	if err := uc.removeNodeFromGroups(ctx, cmd.NodeID); err != nil {
		uc.logger.Errorw("failed to remove node from groups", "error", err, "node_id", cmd.NodeID)
		return nil, fmt.Errorf("failed to remove node from groups: %w", err)
	}

	// Delete the node
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

// removeNodeFromGroups removes the node from all node groups
func (uc *DeleteNodeUseCase) removeNodeFromGroups(ctx context.Context, nodeID uint) error {
	// Get all node groups
	filter := node.NodeGroupFilter{}
	filter.Page = 1
	filter.PageSize = 1000 // Large enough to get all groups

	groups, _, err := uc.nodeGroupRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list node groups", "error", err, "node_id", nodeID)
		return fmt.Errorf("failed to list node groups: %w", err)
	}

	// Remove node from each group that contains it
	for _, group := range groups {
		if group.ContainsNode(nodeID) {
			if err := uc.nodeGroupRepo.RemoveNode(ctx, group.ID(), nodeID); err != nil {
				uc.logger.Errorw("failed to remove node from group",
					"error", err,
					"node_id", nodeID,
					"group_id", group.ID(),
					"group_name", group.Name(),
				)
				return fmt.Errorf("failed to remove node from group %s: %w", group.Name(), err)
			}
			uc.logger.Infow("removed node from group",
				"node_id", nodeID,
				"group_id", group.ID(),
				"group_name", group.Name(),
			)
		}
	}

	return nil
}
