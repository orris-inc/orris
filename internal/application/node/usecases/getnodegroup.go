package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/domain/node"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type GetNodeGroupQuery struct {
	GroupID uint
}

type GetNodeGroupResult struct {
	ID                  uint                   `json:"id"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	NodeIDs             []uint                 `json:"node_ids"`
	SubscriptionPlanIDs []uint                 `json:"subscription_plan_ids"`
	IsPublic            bool                   `json:"is_public"`
	SortOrder           int                    `json:"sort_order"`
	Metadata            map[string]interface{} `json:"metadata"`
	NodeCount           int                    `json:"node_count"`
	Version             int                    `json:"version"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

type GetNodeGroupUseCase struct {
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewGetNodeGroupUseCase(
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *GetNodeGroupUseCase {
	return &GetNodeGroupUseCase{
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *GetNodeGroupUseCase) Execute(ctx context.Context, query GetNodeGroupQuery) (*GetNodeGroupResult, error) {
	uc.logger.Infow("executing get node group use case", "group_id", query.GroupID)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid get node group query", "error", err)
		return nil, err
	}

	group, err := uc.nodeGroupRepo.GetByID(ctx, query.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", query.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	uc.logger.Infow("node group retrieved successfully",
		"group_id", group.ID(),
		"name", group.Name(),
	)

	return &GetNodeGroupResult{
		ID:                  group.ID(),
		Name:                group.Name(),
		Description:         group.Description(),
		NodeIDs:             group.NodeIDs(),
		SubscriptionPlanIDs: group.SubscriptionPlanIDs(),
		IsPublic:            group.IsPublic(),
		SortOrder:           group.SortOrder(),
		Metadata:            group.Metadata(),
		NodeCount:           group.NodeCount(),
		Version:             group.Version(),
		CreatedAt:           group.CreatedAt(),
		UpdatedAt:           group.UpdatedAt(),
	}, nil
}

func (uc *GetNodeGroupUseCase) validateQuery(query GetNodeGroupQuery) error {
	if query.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	return nil
}
