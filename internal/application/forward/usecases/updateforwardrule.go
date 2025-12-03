package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/value_objects"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateForwardRuleCommand represents the input for updating a forward rule.
type UpdateForwardRuleCommand struct {
	ID            uint
	Name          *string
	ListenPort    *uint16
	TargetAddress *string
	TargetPort    *uint16
	Protocol      *string
	Remark        *string
}

// UpdateForwardRuleUseCase handles forward rule updates.
type UpdateForwardRuleUseCase struct {
	repo   forward.Repository
	logger logger.Interface
}

// NewUpdateForwardRuleUseCase creates a new UpdateForwardRuleUseCase.
func NewUpdateForwardRuleUseCase(
	repo forward.Repository,
	logger logger.Interface,
) *UpdateForwardRuleUseCase {
	return &UpdateForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute updates an existing forward rule.
func (uc *UpdateForwardRuleUseCase) Execute(ctx context.Context, cmd UpdateForwardRuleCommand) error {
	uc.logger.Infow("executing update forward rule use case", "id", cmd.ID)

	if cmd.ID == 0 {
		return errors.NewValidationError("rule ID is required")
	}

	// Get existing rule
	rule, err := uc.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		uc.logger.Errorw("failed to get forward rule", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to get forward rule: %w", err)
	}
	if rule == nil {
		return errors.NewNotFoundError("forward rule", fmt.Sprintf("%d", cmd.ID))
	}

	// Update fields
	if cmd.Name != nil {
		if err := rule.UpdateName(*cmd.Name); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.ListenPort != nil {
		// Check if the new port is already in use by another rule
		if *cmd.ListenPort != rule.ListenPort() {
			exists, err := uc.repo.ExistsByListenPort(ctx, *cmd.ListenPort)
			if err != nil {
				uc.logger.Errorw("failed to check listen port", "port", *cmd.ListenPort, "error", err)
				return fmt.Errorf("failed to check listen port: %w", err)
			}
			if exists {
				return errors.NewConflictError("listen port is already in use", fmt.Sprintf("%d", *cmd.ListenPort))
			}
		}
		if err := rule.UpdateListenPort(*cmd.ListenPort); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.TargetAddress != nil || cmd.TargetPort != nil {
		targetAddr := rule.TargetAddress()
		targetPort := rule.TargetPort()
		if cmd.TargetAddress != nil {
			targetAddr = *cmd.TargetAddress
		}
		if cmd.TargetPort != nil {
			targetPort = *cmd.TargetPort
		}
		if err := rule.UpdateTarget(targetAddr, targetPort); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.Protocol != nil {
		protocol := vo.ForwardProtocol(*cmd.Protocol)
		if err := rule.UpdateProtocol(protocol); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.Remark != nil {
		if err := rule.UpdateRemark(*cmd.Remark); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Persist changes
	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to update forward rule", "id", cmd.ID, "error", err)
		return fmt.Errorf("failed to update forward rule: %w", err)
	}

	uc.logger.Infow("forward rule updated successfully", "id", cmd.ID)
	return nil
}
