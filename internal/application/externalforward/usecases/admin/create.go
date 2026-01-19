package admin

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/externalforward/dto"
	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AdminCreateExternalForwardRuleCommand represents the input for admin creating an external forward rule.
type AdminCreateExternalForwardRuleCommand struct {
	Name           string
	ServerAddress  string
	ListenPort     uint16
	ExternalSource string
	ExternalRuleID string
	Remark         string
	SortOrder      int
	NodeSID        string   // optional node SID (node_xxx format)
	GroupSIDs      []string // resource group SIDs for subscription distribution
}

// AdminCreateExternalForwardRuleResult represents the output of admin creating an external forward rule.
type AdminCreateExternalForwardRuleResult struct {
	Rule *dto.AdminExternalForwardRuleDTO `json:"rule"`
}

// AdminCreateExternalForwardRuleUseCase handles admin creating external forward rules.
type AdminCreateExternalForwardRuleUseCase struct {
	repo              externalforward.Repository
	resourceGroupRepo resource.Repository
	nodeRepo          node.NodeRepository
	planRepo          subscription.PlanRepository
	logger            logger.Interface
}

// NewAdminCreateExternalForwardRuleUseCase creates a new admin create use case.
func NewAdminCreateExternalForwardRuleUseCase(
	repo externalforward.Repository,
	resourceGroupRepo resource.Repository,
	nodeRepo node.NodeRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *AdminCreateExternalForwardRuleUseCase {
	return &AdminCreateExternalForwardRuleUseCase{
		repo:              repo,
		resourceGroupRepo: resourceGroupRepo,
		nodeRepo:          nodeRepo,
		planRepo:          planRepo,
		logger:            logger,
	}
}

// Execute creates a new external forward rule for admin.
func (uc *AdminCreateExternalForwardRuleUseCase) Execute(ctx context.Context, cmd AdminCreateExternalForwardRuleCommand) (*AdminCreateExternalForwardRuleResult, error) {
	uc.logger.Infow("executing admin create external forward rule use case",
		"name", cmd.Name,
		"external_source", cmd.ExternalSource,
		"node_sid", cmd.NodeSID,
		"group_sids", cmd.GroupSIDs,
	)

	// Validate command
	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Warnw("invalid command", "error", err)
		return nil, err
	}

	// Validate and resolve node ID if provided
	var nodeID *uint
	nodeIDToInfo := make(map[uint]*dto.NodeInfo)
	if cmd.NodeSID != "" {
		nodeEntity, err := uc.nodeRepo.GetBySID(ctx, cmd.NodeSID)
		if err != nil {
			uc.logger.Errorw("failed to get node", "node_sid", cmd.NodeSID, "error", err)
			return nil, fmt.Errorf("failed to validate node: %w", err)
		}
		if nodeEntity == nil {
			return nil, errors.NewNotFoundError("node", cmd.NodeSID)
		}
		id := nodeEntity.ID()
		nodeID = &id
		info := &dto.NodeInfo{
			SID:           nodeEntity.SID(),
			ServerAddress: nodeEntity.ServerAddress().Value(),
		}
		if nodeEntity.PublicIPv4() != nil {
			info.PublicIPv4 = *nodeEntity.PublicIPv4()
		}
		if nodeEntity.PublicIPv6() != nil {
			info.PublicIPv6 = *nodeEntity.PublicIPv6()
		}
		nodeIDToInfo[nodeEntity.ID()] = info
	}

	// Resolve GroupSIDs to internal IDs and validate plan types
	var groupIDs []uint
	groupIDToSID := make(map[uint]string)
	if len(cmd.GroupSIDs) > 0 {
		groupIDs = make([]uint, 0, len(cmd.GroupSIDs))
		for _, groupSID := range cmd.GroupSIDs {
			// Validate the SID format (rg_xxx)
			if err := id.ValidatePrefix(groupSID, id.PrefixResourceGroup); err != nil {
				return nil, errors.NewValidationError(fmt.Sprintf("invalid resource group ID format: %s", groupSID))
			}

			group, err := uc.resourceGroupRepo.GetBySID(ctx, groupSID)
			if err != nil {
				uc.logger.Errorw("failed to get resource group", "group_sid", groupSID, "error", err)
				return nil, fmt.Errorf("failed to validate resource group: %w", err)
			}
			if group == nil {
				return nil, errors.NewNotFoundError("resource group", groupSID)
			}

			// Verify the plan type supports external forward rules binding (node and hybrid only, not forward)
			plan, err := uc.planRepo.GetByID(ctx, group.PlanID())
			if err != nil {
				uc.logger.Errorw("failed to get plan for resource group", "plan_id", group.PlanID(), "error", err)
				return nil, fmt.Errorf("failed to validate resource group plan: %w", err)
			}
			if plan == nil {
				return nil, fmt.Errorf("plan not found for resource group %s", groupSID)
			}
			if plan.PlanType().IsForward() {
				uc.logger.Warnw("attempted to bind external forward rule to forward plan resource group",
					"group_sid", groupSID,
					"plan_id", group.PlanID(),
					"plan_type", plan.PlanType().String())
				return nil, errors.NewValidationError(
					fmt.Sprintf("resource group %s belongs to a forward plan and cannot bind external forward rules", groupSID))
			}

			groupIDs = append(groupIDs, group.ID())
			groupIDToSID[group.ID()] = group.SID()
		}
	}

	// Create domain entity (admin-created rules don't have subscription/user IDs)
	rule, err := externalforward.NewExternalForwardRuleWithGroups(
		nil, // subscriptionID
		nil, // userID
		nodeID,
		cmd.Name,
		cmd.ServerAddress,
		cmd.ListenPort,
		cmd.ExternalSource,
		cmd.ExternalRuleID,
		cmd.Remark,
		cmd.SortOrder,
		groupIDs,
		id.NewExternalForwardRuleID,
	)
	if err != nil {
		uc.logger.Errorw("failed to create external forward rule entity", "error", err)
		return nil, errors.NewValidationError(err.Error())
	}

	// Persist
	if err := uc.repo.Create(ctx, rule); err != nil {
		uc.logger.Errorw("failed to persist external forward rule", "error", err)
		return nil, fmt.Errorf("failed to save external forward rule: %w", err)
	}

	uc.logger.Infow("external forward rule created successfully by admin", "id", rule.SID(), "name", cmd.Name)

	return &AdminCreateExternalForwardRuleResult{
		Rule: dto.FromDomainToAdmin(rule, &dto.AdminDTOLookups{
			GroupIDToSID: groupIDToSID,
			NodeIDToInfo: nodeIDToInfo,
		}),
	}, nil
}

func (uc *AdminCreateExternalForwardRuleUseCase) validateCommand(cmd AdminCreateExternalForwardRuleCommand) error {
	if cmd.Name == "" {
		return errors.NewValidationError("name is required")
	}
	if len(cmd.Name) > 100 {
		return errors.NewValidationError("name cannot exceed 100 characters")
	}
	// Validate server_address format and security (SSRF protection)
	if err := utils.ValidateServerAddress(cmd.ServerAddress); err != nil {
		return err
	}
	if len(cmd.ServerAddress) > 255 {
		return errors.NewValidationError("server_address cannot exceed 255 characters")
	}
	// Validate listen_port range (security: no system ports or dangerous ports)
	if err := utils.ValidateListenPort(cmd.ListenPort); err != nil {
		return err
	}
	if cmd.ExternalSource == "" {
		return errors.NewValidationError("external_source is required")
	}
	if len(cmd.ExternalSource) > 50 {
		return errors.NewValidationError("external_source cannot exceed 50 characters")
	}
	if len(cmd.ExternalRuleID) > 100 {
		return errors.NewValidationError("external_rule_id cannot exceed 100 characters")
	}
	if len(cmd.Remark) > 500 {
		return errors.NewValidationError("remark cannot exceed 500 characters")
	}
	return nil
}
