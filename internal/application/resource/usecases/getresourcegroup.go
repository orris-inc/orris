package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetResourceGroupUseCase handles retrieving a resource group by ID or SID
type GetResourceGroupUseCase struct {
	repo   resource.Repository
	logger logger.Interface
}

// NewGetResourceGroupUseCase creates a new GetResourceGroupUseCase
func NewGetResourceGroupUseCase(repo resource.Repository, logger logger.Interface) *GetResourceGroupUseCase {
	return &GetResourceGroupUseCase{
		repo:   repo,
		logger: logger,
	}
}

// ExecuteByID retrieves a resource group by its internal ID
func (uc *GetResourceGroupUseCase) ExecuteByID(ctx context.Context, id uint) (*dto.ResourceGroupResponse, error) {
	group, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by ID", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	return &dto.ResourceGroupResponse{
		ID:          group.ID(),
		SID:         group.SID(),
		Name:        group.Name(),
		PlanID:      group.PlanID(),
		Description: group.Description(),
		Status:      string(group.Status()),
		CreatedAt:   group.CreatedAt(),
		UpdatedAt:   group.UpdatedAt(),
	}, nil
}

// ExecuteBySID retrieves a resource group by its Stripe-style ID
func (uc *GetResourceGroupUseCase) ExecuteBySID(ctx context.Context, sid string) (*dto.ResourceGroupResponse, error) {
	group, err := uc.repo.GetBySID(ctx, sid)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "sid", sid)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	return &dto.ResourceGroupResponse{
		ID:          group.ID(),
		SID:         group.SID(),
		Name:        group.Name(),
		PlanID:      group.PlanID(),
		Description: group.Description(),
		Status:      string(group.Status()),
		CreatedAt:   group.CreatedAt(),
		UpdatedAt:   group.UpdatedAt(),
	}, nil
}
