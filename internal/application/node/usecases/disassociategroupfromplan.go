package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/node"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type DisassociateGroupFromPlanCommand struct {
	GroupID uint
	PlanID  uint
}

type DisassociateGroupFromPlanResult struct {
	GroupID uint
	PlanID  uint
	Message string
}

type DisassociateGroupFromPlanUseCase struct {
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewDisassociateGroupFromPlanUseCase(
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *DisassociateGroupFromPlanUseCase {
	return &DisassociateGroupFromPlanUseCase{
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *DisassociateGroupFromPlanUseCase) Execute(ctx context.Context, cmd DisassociateGroupFromPlanCommand) (*DisassociateGroupFromPlanResult, error) {
	uc.logger.Infow("executing disassociate group from plan use case",
		"group_id", cmd.GroupID,
		"plan_id", cmd.PlanID,
	)

	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid disassociate group from plan command", "error", err)
		return nil, err
	}

	group, err := uc.nodeGroupRepo.GetByID(ctx, cmd.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", cmd.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	if !group.IsAssociatedWithPlan(cmd.PlanID) {
		return nil, errors.NewValidationError("node group is not associated with this plan")
	}

	if err := group.DisassociatePlan(cmd.PlanID); err != nil {
		uc.logger.Errorw("failed to disassociate plan from group", "error", err)
		return nil, fmt.Errorf("failed to disassociate plan from group: %w", err)
	}

	if err := uc.nodeGroupRepo.Update(ctx, group); err != nil {
		uc.logger.Errorw("failed to update node group in database", "error", err)
		return nil, fmt.Errorf("failed to update node group: %w", err)
	}

	uc.logger.Infow("group disassociated from plan successfully",
		"group_id", cmd.GroupID,
		"plan_id", cmd.PlanID,
	)

	return &DisassociateGroupFromPlanResult{
		GroupID: cmd.GroupID,
		PlanID:  cmd.PlanID,
		Message: "node group disassociated from plan successfully",
	}, nil
}

func (uc *DisassociateGroupFromPlanUseCase) validateCommand(cmd DisassociateGroupFromPlanCommand) error {
	if cmd.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	if cmd.PlanID == 0 {
		return errors.NewValidationError("plan ID is required")
	}

	return nil
}
