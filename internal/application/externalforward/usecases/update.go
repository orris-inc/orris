package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// UpdateExternalForwardRuleCommand represents the input for updating an external forward rule.
type UpdateExternalForwardRuleCommand struct {
	SID             string
	SubscriptionID  uint
	SubscriptionSID string
	NodeSID         *string // optional node SID (node_xxx format), empty string to clear
	Name            *string
	ServerAddress   *string
	ListenPort      *uint16
	Remark          *string
	SortOrder       *int
}

// UpdateExternalForwardRuleUseCase handles external forward rule updates.
type UpdateExternalForwardRuleUseCase struct {
	repo     externalforward.Repository
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

// NewUpdateExternalForwardRuleUseCase creates a new use case.
func NewUpdateExternalForwardRuleUseCase(
	repo externalforward.Repository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *UpdateExternalForwardRuleUseCase {
	return &UpdateExternalForwardRuleUseCase{
		repo:     repo,
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

// Execute updates an existing external forward rule.
func (uc *UpdateExternalForwardRuleUseCase) Execute(ctx context.Context, cmd UpdateExternalForwardRuleCommand) error {
	uc.logger.Infow("executing update external forward rule use case", "sid", cmd.SID)

	// Get existing rule
	rule, err := uc.repo.GetBySID(ctx, cmd.SID)
	if err != nil {
		return err
	}

	// Verify rule belongs to the specified subscription
	if rule.SubscriptionID() == nil || *rule.SubscriptionID() != cmd.SubscriptionID {
		uc.logger.Warnw("external forward rule does not belong to subscription",
			"rule_sid", cmd.SID,
			"rule_subscription_id", rule.SubscriptionID(),
			"requested_subscription_id", cmd.SubscriptionID,
		)
		return errors.NewNotFoundError("external forward rule", cmd.SID)
	}

	// Apply updates with length validation
	if cmd.Name != nil {
		if len(*cmd.Name) > 100 {
			return errors.NewValidationError("name cannot exceed 100 characters")
		}
		if err := rule.UpdateName(*cmd.Name); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.ServerAddress != nil {
		// Validate server_address format and security (SSRF protection)
		if err := utils.ValidateServerAddress(*cmd.ServerAddress); err != nil {
			return err
		}
		if len(*cmd.ServerAddress) > 255 {
			return errors.NewValidationError("server_address cannot exceed 255 characters")
		}
		if err := rule.UpdateServerAddress(*cmd.ServerAddress); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.ListenPort != nil {
		// Validate listen_port range (security: no system ports or dangerous ports)
		if err := utils.ValidateListenPort(*cmd.ListenPort); err != nil {
			return err
		}
		if err := rule.UpdateListenPort(*cmd.ListenPort); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.SortOrder != nil {
		if err := rule.UpdateSortOrder(*cmd.SortOrder); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.Remark != nil {
		if len(*cmd.Remark) > 500 {
			return errors.NewValidationError("remark cannot exceed 500 characters")
		}
		rule.UpdateRemark(*cmd.Remark)
	}

	// Update node ID if provided
	if cmd.NodeSID != nil {
		if *cmd.NodeSID == "" {
			// Clear node
			rule.UpdateNodeID(nil)
		} else {
			// Validate and set node
			nodeEntity, err := uc.nodeRepo.GetBySID(ctx, *cmd.NodeSID)
			if err != nil {
				uc.logger.Errorw("failed to get node", "node_sid", *cmd.NodeSID, "error", err)
				return fmt.Errorf("failed to validate node: %w", err)
			}
			if nodeEntity == nil {
				return errors.NewNotFoundError("node", *cmd.NodeSID)
			}
			nodeID := nodeEntity.ID()
			rule.UpdateNodeID(&nodeID)
		}
	}

	// Persist
	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to update external forward rule", "sid", cmd.SID, "error", err)
		return fmt.Errorf("failed to update external forward rule: %w", err)
	}

	uc.logger.Infow("external forward rule updated successfully", "sid", cmd.SID)
	return nil
}
