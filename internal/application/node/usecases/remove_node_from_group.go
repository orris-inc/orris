package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/node"
	"orris/internal/domain/shared/events"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type RemoveNodeFromGroupCommand struct {
	GroupID uint
	NodeID  uint
}

type RemoveNodeFromGroupResult struct {
	GroupID   uint
	NodeID    uint
	NodeCount int
	Message   string
}

type RemoveNodeFromGroupUseCase struct {
	nodeGroupRepo   node.NodeGroupRepository
	eventDispatcher events.EventDispatcher
	logger          logger.Interface
}

func NewRemoveNodeFromGroupUseCase(
	nodeGroupRepo node.NodeGroupRepository,
	eventDispatcher events.EventDispatcher,
	logger logger.Interface,
) *RemoveNodeFromGroupUseCase {
	return &RemoveNodeFromGroupUseCase{
		nodeGroupRepo:   nodeGroupRepo,
		eventDispatcher: eventDispatcher,
		logger:          logger,
	}
}

func (uc *RemoveNodeFromGroupUseCase) Execute(ctx context.Context, cmd RemoveNodeFromGroupCommand) (*RemoveNodeFromGroupResult, error) {
	uc.logger.Infow("executing remove node from group use case",
		"group_id", cmd.GroupID,
		"node_id", cmd.NodeID,
	)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid remove node from group command", "error", err)
		return nil, err
	}

	group, err := uc.nodeGroupRepo.GetByID(ctx, cmd.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", cmd.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	if !group.ContainsNode(cmd.NodeID) {
		return nil, errors.NewValidationError("node does not exist in this group")
	}

	if err := group.RemoveNode(cmd.NodeID); err != nil {
		uc.logger.Errorw("failed to remove node from group", "error", err)
		return nil, fmt.Errorf("failed to remove node from group: %w", err)
	}

	if err := uc.nodeGroupRepo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update node group in database", "error", err)
		return nil, fmt.Errorf("failed to update node group: %w", err)
	}

	allEvents := group.GetEvents()
	domainEvents := make([]events.DomainEvent, 0, len(allEvents))
	for _, event := range allEvents {
		if de, ok := event.(events.DomainEvent); ok {
			domainEvents = append(domainEvents, de)
		}
	}
	if len(domainEvents) > 0 {
		if err := uc.eventDispatcher.PublishAll(domainEvents); err != nil {
			uc.logger.Warnw("failed to publish events", "error", err)
		}
	}

	uc.logger.Infow("node removed from group successfully",
		"group_id", cmd.GroupID,
		"node_id", cmd.NodeID,
		"node_count", group.NodeCount(),
	)

	return &RemoveNodeFromGroupResult{
		GroupID:   cmd.GroupID,
		NodeID:    cmd.NodeID,
		NodeCount: group.NodeCount(),
		Message:   "node removed from group successfully",
	}, nil
}

func (uc *RemoveNodeFromGroupUseCase) validateCommand(cmd RemoveNodeFromGroupCommand) error {
	if cmd.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	if cmd.NodeID == 0 {
		return errors.NewValidationError("node ID is required")
	}

	return nil
}
