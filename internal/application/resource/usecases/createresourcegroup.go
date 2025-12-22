package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// CreateResourceGroupUseCase handles the creation of a new resource group
type CreateResourceGroupUseCase struct {
	repo     resource.Repository
	planRepo subscription.PlanRepository
	logger   logger.Interface
}

// NewCreateResourceGroupUseCase creates a new CreateResourceGroupUseCase
func NewCreateResourceGroupUseCase(repo resource.Repository, planRepo subscription.PlanRepository, logger logger.Interface) *CreateResourceGroupUseCase {
	return &CreateResourceGroupUseCase{
		repo:     repo,
		planRepo: planRepo,
		logger:   logger,
	}
}

// Execute creates a new resource group
func (uc *CreateResourceGroupUseCase) Execute(ctx context.Context, req dto.CreateResourceGroupRequest) (*dto.ResourceGroupResponse, error) {
	// Resolve plan SID to internal ID
	plan, err := uc.planRepo.GetBySID(ctx, req.PlanSID)
	if err != nil {
		uc.logger.Errorw("failed to get plan by SID", "error", err, "plan_sid", req.PlanSID)
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		uc.logger.Warnw("plan not found by SID", "plan_sid", req.PlanSID)
		return nil, subscription.ErrPlanNotFound
	}

	// Check if name already exists
	exists, err := uc.repo.ExistsByName(ctx, req.Name)
	if err != nil {
		uc.logger.Errorw("failed to check resource group name existence", "error", err, "name", req.Name)
		return nil, fmt.Errorf("failed to check name existence: %w", err)
	}
	if exists {
		return nil, resource.ErrGroupNameExists
	}

	// Create new resource group
	group, err := resource.NewResourceGroup(req.Name, plan.ID(), req.Description)
	if err != nil {
		uc.logger.Errorw("failed to create resource group entity", "error", err)
		return nil, fmt.Errorf("failed to create resource group: %w", err)
	}

	// Save to repository
	if err := uc.repo.Create(ctx, group); err != nil {
		uc.logger.Errorw("failed to save resource group", "error", err)
		return nil, fmt.Errorf("failed to save resource group: %w", err)
	}

	uc.logger.Infow("resource group created successfully",
		"id", group.ID(),
		"sid", group.SID(),
		"name", group.Name(),
		"plan_id", group.PlanID(),
	)

	return &dto.ResourceGroupResponse{
		SID:         group.SID(),
		Name:        group.Name(),
		PlanSID:     plan.SID(),
		Description: group.Description(),
		Status:      string(group.Status()),
		CreatedAt:   group.CreatedAt(),
		UpdatedAt:   group.UpdatedAt(),
	}, nil
}
