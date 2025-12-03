package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type UpdateNodeGroupCommand struct {
	GroupID     uint
	Name        *string
	Description *string
	IsPublic    *bool
	SortOrder   *int
	Version     int
}

type UpdateNodeGroupResult struct {
	GroupID     uint
	Name        string
	Description string
	IsPublic    bool
	SortOrder   int
	Version     int
	UpdatedAt   string
}

type UpdateNodeGroupUseCase struct {
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewUpdateNodeGroupUseCase(
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *UpdateNodeGroupUseCase {
	return &UpdateNodeGroupUseCase{
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *UpdateNodeGroupUseCase) Execute(ctx context.Context, cmd UpdateNodeGroupCommand) (*UpdateNodeGroupResult, error) {
	uc.logger.Infow("executing update node group use case", "group_id", cmd.GroupID)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid update node group command", "error", err)
		return nil, err
	}

	group, err := uc.nodeGroupRepo.GetByID(ctx, cmd.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", cmd.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	if group.Version() != cmd.Version {
		return nil, errors.NewConflictError("node group has been modified by another user, please refresh and try again")
	}

	if cmd.Name != nil {
		if *cmd.Name != group.Name() {
			exists, err := uc.nodeGroupRepo.ExistsByName(ctx, *cmd.Name)
			if err != nil {
				uc.logger.Errorw("failed to check node group name existence", "error", err, "name", *cmd.Name)
				return nil, fmt.Errorf("failed to check node group name existence: %w", err)
			}
			if exists {
				return nil, errors.NewValidationError("node group name already exists")
			}

			if err := group.UpdateName(*cmd.Name); err != nil {
				uc.logger.Errorw("failed to update node group name", "error", err)
				return nil, fmt.Errorf("failed to update node group name: %w", err)
			}
		}
	}

	if cmd.Description != nil {
		if err := group.UpdateDescription(*cmd.Description); err != nil {
			uc.logger.Errorw("failed to update node group description", "error", err)
			return nil, fmt.Errorf("failed to update node group description: %w", err)
		}
	}

	if cmd.IsPublic != nil {
		if err := group.SetPublic(*cmd.IsPublic); err != nil {
			uc.logger.Errorw("failed to update node group visibility", "error", err)
			return nil, fmt.Errorf("failed to update node group visibility: %w", err)
		}
	}

	if cmd.SortOrder != nil {
		if err := group.UpdateSortOrder(*cmd.SortOrder); err != nil {
			uc.logger.Errorw("failed to update node group sort order", "error", err)
			return nil, fmt.Errorf("failed to update node group sort order: %w", err)
		}
	}

	if err := uc.nodeGroupRepo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update node group in database", "error", err)
		return nil, fmt.Errorf("failed to update node group: %w", err)
	}

	uc.logger.Infow("node group updated successfully",
		"group_id", group.ID(),
		"name", group.Name(),
		"version", group.Version(),
	)

	return &UpdateNodeGroupResult{
		GroupID:     group.ID(),
		Name:        group.Name(),
		Description: group.Description(),
		IsPublic:    group.IsPublic(),
		SortOrder:   group.SortOrder(),
		Version:     group.Version(),
		UpdatedAt:   group.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (uc *UpdateNodeGroupUseCase) validateCommand(cmd UpdateNodeGroupCommand) error {
	if cmd.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	if cmd.Name != nil && *cmd.Name == "" {
		return errors.NewValidationError("node group name cannot be empty")
	}

	if cmd.Name != nil && len(*cmd.Name) > 255 {
		return errors.NewValidationError("node group name is too long (max 255 characters)")
	}

	if cmd.SortOrder != nil && *cmd.SortOrder < 0 {
		return errors.NewValidationError("sort order cannot be negative")
	}

	if cmd.Name == nil && cmd.Description == nil && cmd.IsPublic == nil && cmd.SortOrder == nil {
		return errors.NewValidationError("at least one field must be provided for update")
	}

	return nil
}
