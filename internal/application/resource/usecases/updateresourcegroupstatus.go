package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateResourceGroupStatusUseCase handles activating/deactivating a resource group
type UpdateResourceGroupStatusUseCase struct {
	repo   resource.Repository
	logger logger.Interface
}

// NewUpdateResourceGroupStatusUseCase creates a new UpdateResourceGroupStatusUseCase
func NewUpdateResourceGroupStatusUseCase(repo resource.Repository, logger logger.Interface) *UpdateResourceGroupStatusUseCase {
	return &UpdateResourceGroupStatusUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Activate activates a resource group
func (uc *UpdateResourceGroupStatusUseCase) Activate(ctx context.Context, id uint) (*dto.ResourceGroupResponse, error) {
	group, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	group.Activate()

	if err := uc.repo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update resource group status", "error", err, "id", id)
		return nil, fmt.Errorf("failed to update resource group: %w", err)
	}

	uc.logger.Infow("resource group activated", "id", id, "sid", group.SID())

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

// Deactivate deactivates a resource group
func (uc *UpdateResourceGroupStatusUseCase) Deactivate(ctx context.Context, id uint) (*dto.ResourceGroupResponse, error) {
	group, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	group.Deactivate()

	if err := uc.repo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update resource group status", "error", err, "id", id)
		return nil, fmt.Errorf("failed to update resource group: %w", err)
	}

	uc.logger.Infow("resource group deactivated", "id", id, "sid", group.SID())

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
