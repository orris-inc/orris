package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateForwardAgentCommand represents the input for updating a forward agent.
type UpdateForwardAgentCommand struct {
	ShortID       string // External API identifier
	Name          *string
	PublicAddress *string
	TunnelAddress *string
	Remark        *string
	GroupSID      *string // Resource group SID (empty string to remove association)
}

// UpdateForwardAgentUseCase handles forward agent updates.
type UpdateForwardAgentUseCase struct {
	repo              forward.AgentRepository
	resourceGroupRepo resource.Repository
	logger            logger.Interface
}

// NewUpdateForwardAgentUseCase creates a new UpdateForwardAgentUseCase.
func NewUpdateForwardAgentUseCase(
	repo forward.AgentRepository,
	resourceGroupRepo resource.Repository,
	logger logger.Interface,
) *UpdateForwardAgentUseCase {
	return &UpdateForwardAgentUseCase{
		repo:              repo,
		resourceGroupRepo: resourceGroupRepo,
		logger:            logger,
	}
}

// Execute updates an existing forward agent.
func (uc *UpdateForwardAgentUseCase) Execute(ctx context.Context, cmd UpdateForwardAgentCommand) error {
	if cmd.ShortID == "" {
		return errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing update forward agent use case", "short_id", cmd.ShortID)

	agent, err := uc.repo.GetBySID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return errors.NewNotFoundError("forward agent", cmd.ShortID)
	}

	// Update fields
	if cmd.Name != nil {
		if err := agent.UpdateName(*cmd.Name); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.PublicAddress != nil {
		if err := agent.UpdatePublicAddress(*cmd.PublicAddress); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.TunnelAddress != nil {
		if err := agent.UpdateTunnelAddress(*cmd.TunnelAddress); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.Remark != nil {
		if err := agent.UpdateRemark(*cmd.Remark); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Handle GroupSID update (resolve SID to internal ID)
	if cmd.GroupSID != nil {
		if *cmd.GroupSID == "" {
			// Empty string means remove the association
			agent.SetGroupID(nil)
		} else {
			// Resolve group SID to internal ID
			group, err := uc.resourceGroupRepo.GetBySID(ctx, *cmd.GroupSID)
			if err != nil {
				uc.logger.Errorw("failed to get resource group by SID", "group_sid", *cmd.GroupSID, "error", err)
				return errors.NewNotFoundError("resource group", *cmd.GroupSID)
			}
			if group == nil {
				return errors.NewNotFoundError("resource group", *cmd.GroupSID)
			}
			groupID := group.ID()
			agent.SetGroupID(&groupID)
		}
	}

	// Persist changes
	if err := uc.repo.Update(ctx, agent); err != nil {
		uc.logger.Errorw("failed to update forward agent", "id", agent.ID(), "short_id", agent.SID(), "error", err)
		return fmt.Errorf("failed to update forward agent: %w", err)
	}

	uc.logger.Infow("forward agent updated successfully", "id", agent.ID(), "short_id", agent.SID())
	return nil
}
