package usecases

import (
	"context"
	"fmt"
	"math"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ListResourceGroupsUseCase handles listing resource groups with pagination
type ListResourceGroupsUseCase struct {
	repo   resource.Repository
	logger logger.Interface
}

// NewListResourceGroupsUseCase creates a new ListResourceGroupsUseCase
func NewListResourceGroupsUseCase(repo resource.Repository, logger logger.Interface) *ListResourceGroupsUseCase {
	return &ListResourceGroupsUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute lists resource groups with optional filtering and pagination
func (uc *ListResourceGroupsUseCase) Execute(ctx context.Context, req dto.ListResourceGroupsRequest) (*dto.ListResourceGroupsResponse, error) {
	// Build filter
	filter := resource.ListFilter{
		PlanID:   req.PlanID,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	if req.Status != nil {
		status := resource.GroupStatus(*req.Status)
		filter.Status = &status
	}

	// Query repository
	groups, total, err := uc.repo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list resource groups", "error", err)
		return nil, fmt.Errorf("failed to list resource groups: %w", err)
	}

	// Convert to response
	items := make([]dto.ResourceGroupResponse, 0, len(groups))
	for _, group := range groups {
		items = append(items, dto.ResourceGroupResponse{
			ID:          group.ID(),
			SID:         group.SID(),
			Name:        group.Name(),
			PlanID:      group.PlanID(),
			Description: group.Description(),
			Status:      string(group.Status()),
			CreatedAt:   group.CreatedAt(),
			UpdatedAt:   group.UpdatedAt(),
		})
	}

	totalPages := int(math.Ceil(float64(total) / float64(req.PageSize)))

	return &dto.ListResourceGroupsResponse{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}
