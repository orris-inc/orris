package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type AssociateGroupWithPlanCommand struct {
	GroupID uint
	PlanID  uint
}

type AssociateGroupWithPlanResult struct {
	GroupID uint
	PlanID  uint
	Message string
}

type AssociateGroupWithPlanUseCase struct {
	nodeGroupRepo node.NodeGroupRepository
	planRepo      subscription.SubscriptionPlanRepository
	logger        logger.Interface
}

func NewAssociateGroupWithPlanUseCase(
	nodeGroupRepo node.NodeGroupRepository,
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *AssociateGroupWithPlanUseCase {
	return &AssociateGroupWithPlanUseCase{
		nodeGroupRepo: nodeGroupRepo,
		planRepo:      planRepo,
		logger:        logger,
	}
}

func (uc *AssociateGroupWithPlanUseCase) Execute(ctx context.Context, cmd AssociateGroupWithPlanCommand) (*AssociateGroupWithPlanResult, error) {
	uc.logger.Infow("executing associate group with plan use case",
		"group_id", cmd.GroupID,
		"plan_id", cmd.PlanID,
	)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid associate group with plan command", "error", err)
		return nil, err
	}

	group, err := uc.nodeGroupRepo.GetByID(ctx, cmd.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", cmd.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	_, err = uc.planRepo.GetByID(ctx, cmd.PlanID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err, "plan_id", cmd.PlanID)
		return nil, fmt.Errorf("failed to get subscription plan: %w", err)
	}

	if group.IsAssociatedWithPlan(cmd.PlanID) {
		return nil, errors.NewValidationError("node group already associated with this plan")
	}

	if err := group.AssociatePlan(cmd.PlanID); err != nil {
		uc.logger.Errorw("failed to associate plan with group", "error", err)
		return nil, fmt.Errorf("failed to associate plan with group: %w", err)
	}

	if err := uc.nodeGroupRepo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update node group in database", "error", err)
		return nil, fmt.Errorf("failed to update node group: %w", err)
	}

	uc.logger.Infow("group associated with plan successfully",
		"group_id", cmd.GroupID,
		"plan_id", cmd.PlanID,
	)

	return &AssociateGroupWithPlanResult{
		GroupID: cmd.GroupID,
		PlanID:  cmd.PlanID,
		Message: "node group associated with plan successfully",
	}, nil
}

func (uc *AssociateGroupWithPlanUseCase) validateCommand(cmd AssociateGroupWithPlanCommand) error {
	if cmd.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	if cmd.PlanID == 0 {
		return errors.NewValidationError("plan ID is required")
	}

	return nil
}
