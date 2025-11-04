package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/node"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type CreateNodeGroupCommand struct {
	Name        string
	Description string
	IsPublic    bool
	SortOrder   int
	Metadata    map[string]interface{}
}

type CreateNodeGroupResult struct {
	GroupID     uint
	Name        string
	Description string
	IsPublic    bool
	SortOrder   int
	CreatedAt   string
}

type CreateNodeGroupUseCase struct {
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewCreateNodeGroupUseCase(
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *CreateNodeGroupUseCase {
	return &CreateNodeGroupUseCase{
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *CreateNodeGroupUseCase) Execute(ctx context.Context, cmd CreateNodeGroupCommand) (*CreateNodeGroupResult, error) {
	uc.logger.Infow("executing create node group use case", "name", cmd.Name)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid create node group command", "error", err)
		return nil, err
	}

	exists, err := uc.nodeGroupRepo.ExistsByName(ctx, cmd.Name)
	if err != nil {
		uc.logger.Errorw("failed to check node group name existence", "error", err, "name", cmd.Name)
		return nil, fmt.Errorf("failed to check node group name existence: %w", err)
	}
	if exists {
		return nil, errors.NewValidationError("node group name already exists")
	}

	group, err := node.NewNodeGroup(cmd.Name, cmd.Description, cmd.IsPublic, cmd.SortOrder)
	if err != nil {
		uc.logger.Errorw("failed to create node group aggregate", "error", err)
		return nil, fmt.Errorf("failed to create node group: %w", err)
	}

	if err := uc.nodeGroupRepo.Create(ctx, group); err != nil {
		uc.logger.Errorw("failed to create node group in database", "error", err)
		return nil, fmt.Errorf("failed to create node group: %w", err)
	}

	uc.logger.Infow("node group created successfully",
		"group_id", group.ID(),
		"name", cmd.Name,
		"is_public", cmd.IsPublic,
	)

	return &CreateNodeGroupResult{
		GroupID:     group.ID(),
		Name:        group.Name(),
		Description: group.Description(),
		IsPublic:    group.IsPublic(),
		SortOrder:   group.SortOrder(),
		CreatedAt:   group.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (uc *CreateNodeGroupUseCase) validateCommand(cmd CreateNodeGroupCommand) error {
	if cmd.Name == "" {
		return errors.NewValidationError("node group name is required")
	}

	if len(cmd.Name) > 255 {
		return errors.NewValidationError("node group name is too long (max 255 characters)")
	}

	if cmd.SortOrder < 0 {
		return errors.NewValidationError("sort order cannot be negative")
	}

	return nil
}
