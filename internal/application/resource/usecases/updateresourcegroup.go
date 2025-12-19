package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateResourceGroupUseCase handles updating a resource group
type UpdateResourceGroupUseCase struct {
	repo   resource.Repository
	logger logger.Interface
}

// NewUpdateResourceGroupUseCase creates a new UpdateResourceGroupUseCase
func NewUpdateResourceGroupUseCase(repo resource.Repository, logger logger.Interface) *UpdateResourceGroupUseCase {
	return &UpdateResourceGroupUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute updates a resource group by its internal ID
func (uc *UpdateResourceGroupUseCase) Execute(ctx context.Context, id uint, req dto.UpdateResourceGroupRequest) (*dto.ResourceGroupResponse, error) {
	// Get existing resource group
	group, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	// Update name if provided
	if req.Name != nil {
		// Check if new name already exists (for different group)
		exists, err := uc.repo.ExistsByName(ctx, *req.Name)
		if err != nil {
			uc.logger.Errorw("failed to check name existence", "error", err)
			return nil, fmt.Errorf("failed to check name existence: %w", err)
		}
		if exists && *req.Name != group.Name() {
			return nil, resource.ErrGroupNameExists
		}
		if err := group.UpdateName(*req.Name); err != nil {
			return nil, fmt.Errorf("failed to update name: %w", err)
		}
	}

	// Update description if provided
	if req.Description != nil {
		group.UpdateDescription(*req.Description)
	}

	// Save changes
	if err := uc.repo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update resource group", "error", err, "id", id)
		return nil, fmt.Errorf("failed to update resource group: %w", err)
	}

	uc.logger.Infow("resource group updated successfully", "id", id, "sid", group.SID())

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
