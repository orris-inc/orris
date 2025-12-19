package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateResourceGroupStatusUseCase handles activating/deactivating a resource group
type UpdateResourceGroupStatusUseCase struct {
	repo     resource.Repository
	planRepo subscription.PlanRepository
	logger   logger.Interface
}

// NewUpdateResourceGroupStatusUseCase creates a new UpdateResourceGroupStatusUseCase
func NewUpdateResourceGroupStatusUseCase(repo resource.Repository, planRepo subscription.PlanRepository, logger logger.Interface) *UpdateResourceGroupStatusUseCase {
	return &UpdateResourceGroupStatusUseCase{
		repo:     repo,
		planRepo: planRepo,
		logger:   logger,
	}
}

// Activate activates a resource group by internal ID
func (uc *UpdateResourceGroupStatusUseCase) Activate(ctx context.Context, id uint) (*dto.ResourceGroupResponse, error) {
	group, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeActivate(ctx, group)
}

// ActivateBySID activates a resource group by Stripe-style SID
func (uc *UpdateResourceGroupStatusUseCase) ActivateBySID(ctx context.Context, sid string) (*dto.ResourceGroupResponse, error) {
	group, err := uc.repo.GetBySID(ctx, sid)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "sid", sid)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeActivate(ctx, group)
}

// executeActivate performs the actual activation logic
func (uc *UpdateResourceGroupStatusUseCase) executeActivate(ctx context.Context, group *resource.ResourceGroup) (*dto.ResourceGroupResponse, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	group.Activate()

	if err := uc.repo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update resource group status", "error", err, "id", group.ID(), "sid", group.SID())
		return nil, fmt.Errorf("failed to update resource group: %w", err)
	}

	uc.logger.Infow("resource group activated", "id", group.ID(), "sid", group.SID())

	return uc.buildResponse(ctx, group)
}

// Deactivate deactivates a resource group by internal ID
func (uc *UpdateResourceGroupStatusUseCase) Deactivate(ctx context.Context, id uint) (*dto.ResourceGroupResponse, error) {
	group, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeDeactivate(ctx, group)
}

// DeactivateBySID deactivates a resource group by Stripe-style SID
func (uc *UpdateResourceGroupStatusUseCase) DeactivateBySID(ctx context.Context, sid string) (*dto.ResourceGroupResponse, error) {
	group, err := uc.repo.GetBySID(ctx, sid)
	if err != nil {
		uc.logger.Errorw("failed to get resource group by SID", "error", err, "sid", sid)
		return nil, fmt.Errorf("failed to get resource group: %w", err)
	}
	return uc.executeDeactivate(ctx, group)
}

// executeDeactivate performs the actual deactivation logic
func (uc *UpdateResourceGroupStatusUseCase) executeDeactivate(ctx context.Context, group *resource.ResourceGroup) (*dto.ResourceGroupResponse, error) {
	if group == nil {
		return nil, resource.ErrGroupNotFound
	}

	group.Deactivate()

	if err := uc.repo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update resource group status", "error", err, "id", group.ID(), "sid", group.SID())
		return nil, fmt.Errorf("failed to update resource group: %w", err)
	}

	uc.logger.Infow("resource group deactivated", "id", group.ID(), "sid", group.SID())

	return uc.buildResponse(ctx, group)
}

// buildResponse builds a ResourceGroupResponse from a ResourceGroup entity
func (uc *UpdateResourceGroupStatusUseCase) buildResponse(ctx context.Context, group *resource.ResourceGroup) (*dto.ResourceGroupResponse, error) {
	planSID := ""
	plan, err := uc.planRepo.GetByID(ctx, group.PlanID())
	if err != nil {
		uc.logger.Warnw("failed to get plan for SID lookup", "error", err, "plan_id", group.PlanID())
	} else if plan != nil {
		planSID = plan.SID()
	}

	return &dto.ResourceGroupResponse{
		ID:          group.ID(),
		SID:         group.SID(),
		Name:        group.Name(),
		PlanSID:     planSID,
		Description: group.Description(),
		Status:      string(group.Status()),
		CreatedAt:   group.CreatedAt(),
		UpdatedAt:   group.UpdatedAt(),
	}, nil
}
