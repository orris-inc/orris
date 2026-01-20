package admin

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AdminUpdateExternalForwardRuleCommand represents the input for updating an external forward rule.
type AdminUpdateExternalForwardRuleCommand struct {
	SID           string
	NodeSID       *string  // optional node SID (node_xxx format), empty string to clear
	Name          *string
	ServerAddress *string
	ListenPort    *uint16
	Remark        *string
	SortOrder     *int
	GroupSIDs     []string // resource group SIDs (nil = no change, empty = clear all)
}

// AdminUpdateExternalForwardRuleUseCase handles admin external forward rule updates.
type AdminUpdateExternalForwardRuleUseCase struct {
	repo              externalforward.Repository
	nodeRepo          node.NodeRepository
	resourceGroupRepo resource.Repository
	planRepo          subscription.PlanRepository
	logger            logger.Interface
}

// NewAdminUpdateExternalForwardRuleUseCase creates a new admin update use case.
func NewAdminUpdateExternalForwardRuleUseCase(
	repo externalforward.Repository,
	nodeRepo node.NodeRepository,
	resourceGroupRepo resource.Repository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *AdminUpdateExternalForwardRuleUseCase {
	return &AdminUpdateExternalForwardRuleUseCase{
		repo:              repo,
		nodeRepo:          nodeRepo,
		resourceGroupRepo: resourceGroupRepo,
		planRepo:          planRepo,
		logger:            logger,
	}
}

// Execute updates an existing external forward rule.
func (uc *AdminUpdateExternalForwardRuleUseCase) Execute(ctx context.Context, cmd AdminUpdateExternalForwardRuleCommand) error {
	uc.logger.Infow("executing admin update external forward rule use case", "sid", cmd.SID)

	// Get existing rule
	rule, err := uc.repo.GetBySID(ctx, cmd.SID)
	if err != nil {
		return err
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

	// Update resource group IDs if provided (nil = no change, empty slice = clear all)
	if cmd.GroupSIDs != nil {
		if len(cmd.GroupSIDs) == 0 {
			// Clear all resource groups
			rule.SetGroupIDs(nil)
		} else {
			// Validate and resolve group SIDs to internal IDs
			groupIDs := make([]uint, 0, len(cmd.GroupSIDs))
			for _, groupSID := range cmd.GroupSIDs {
				// Validate the SID format (rg_xxx)
				if err := id.ValidatePrefix(groupSID, id.PrefixResourceGroup); err != nil {
					return errors.NewValidationError(fmt.Sprintf("invalid resource group ID format: %s", groupSID))
				}

				group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
				if err != nil {
					uc.logger.Errorw("failed to get resource group", "group_sid", groupSID, "error", err)
					return fmt.Errorf("failed to validate resource group: %w", err)
				}
				if group == nil {
					return errors.NewNotFoundError("resource group", groupSID)
				}

				// Verify the plan type supports external forward rules binding (node and hybrid only, not forward)
				plan, err := uc.planRepo.GetByID(ctx, group.PlanID())
				if err != nil {
					uc.logger.Errorw("failed to get plan for resource group", "plan_id", group.PlanID(), "error", err)
					return fmt.Errorf("failed to validate resource group plan: %w", err)
				}
				if plan == nil {
					return fmt.Errorf("plan not found for resource group %s", groupSID)
				}
				if plan.PlanType().IsForward() {
					uc.logger.Warnw("attempted to bind external forward rule to forward plan resource group",
						"group_sid", groupSID,
						"plan_id", group.PlanID(),
						"plan_type", plan.PlanType().String())
					return errors.NewValidationError(
						fmt.Sprintf("resource group %s belongs to a forward plan and cannot bind external forward rules", groupSID))
				}

				groupIDs = append(groupIDs, group.ID())
			}
			rule.SetGroupIDs(groupIDs)
		}
	}

	// Persist
	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to update external forward rule", "sid", cmd.SID, "error", err)
		return fmt.Errorf("failed to update external forward rule: %w", err)
	}

	uc.logger.Infow("external forward rule updated successfully by admin", "sid", cmd.SID)
	return nil
}
