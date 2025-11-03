package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/domain/node"
	"orris/internal/shared/logger"
)

type ListNodeGroupsQuery struct {
	Name     *string
	IsPublic *bool
	Page     int
	PageSize int
	SortBy   string
	SortDesc bool
}

type NodeGroupDTO struct {
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

type ListNodeGroupsResult struct {
	Groups []*NodeGroupDTO `json:"groups"`
	Total  int64           `json:"total"`
}

type ListNodeGroupsUseCase struct {
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewListNodeGroupsUseCase(
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *ListNodeGroupsUseCase {
	return &ListNodeGroupsUseCase{
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *ListNodeGroupsUseCase) Execute(
	ctx context.Context,
	query ListNodeGroupsQuery,
) (*ListNodeGroupsResult, error) {
	uc.logger.Infow("executing list node groups use case",
		"page", query.Page,
		"page_size", query.PageSize,
	)

	filter := node.NodeGroupFilter{
		Name:     query.Name,
		IsPublic: query.IsPublic,
	}
	filter.Page = query.Page
	filter.PageSize = query.PageSize
	filter.SortBy = query.SortBy
	if query.SortDesc {
		filter.SortOrder = "DESC"
	} else {
		filter.SortOrder = "ASC"
	}

	groups, total, err := uc.nodeGroupRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list node groups", "error", err)
		return nil, fmt.Errorf("failed to list node groups: %w", err)
	}

	groupDTOs := make([]*NodeGroupDTO, 0, len(groups))
	for _, group := range groups {
		groupDTOs = append(groupDTOs, uc.toDTO(group))
	}

	uc.logger.Infow("node groups listed successfully",
		"count", len(groups),
		"total", total,
	)

	return &ListNodeGroupsResult{
		Groups: groupDTOs,
		Total:  total,
	}, nil
}

func (uc *ListNodeGroupsUseCase) toDTO(group *node.NodeGroup) *NodeGroupDTO {
	return &NodeGroupDTO{
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
	}
}
