package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/externalforward/dto"
	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// CreateExternalForwardRuleCommand represents the input for creating an external forward rule.
type CreateExternalForwardRuleCommand struct {
	SubscriptionID  uint
	SubscriptionSID string
	UserID          uint
	NodeSID         string // optional node SID (node_xxx format)
	Name            string
	ServerAddress   string
	ListenPort      uint16
	ExternalSource  string
	ExternalRuleID  string
	Remark          string
	SortOrder       int
}

// CreateExternalForwardRuleResult represents the output of creating an external forward rule.
type CreateExternalForwardRuleResult struct {
	Rule *dto.ExternalForwardRuleDTO `json:"rule"`
}

// CreateExternalForwardRuleUseCase handles external forward rule creation.
type CreateExternalForwardRuleUseCase struct {
	repo     externalforward.Repository
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

// NewCreateExternalForwardRuleUseCase creates a new use case.
func NewCreateExternalForwardRuleUseCase(
	repo externalforward.Repository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *CreateExternalForwardRuleUseCase {
	return &CreateExternalForwardRuleUseCase{
		repo:     repo,
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

// Execute creates a new external forward rule.
func (uc *CreateExternalForwardRuleUseCase) Execute(ctx context.Context, cmd CreateExternalForwardRuleCommand) (*CreateExternalForwardRuleResult, error) {
	uc.logger.Infow("executing create external forward rule use case",
		"subscription_id", cmd.SubscriptionID,
		"name", cmd.Name,
		"external_source", cmd.ExternalSource,
		"node_sid", cmd.NodeSID,
	)

	// Validate command
	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Warnw("invalid command", "error", err)
		return nil, err
	}

	// Validate and resolve node ID if provided
	var nodeID *uint
	var nodeInfo *dto.NodeInfo
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
		nodeInfo = &dto.NodeInfo{
			SID:           nodeEntity.SID(),
			ServerAddress: nodeEntity.ServerAddress().Value(),
		}
		if nodeEntity.PublicIPv4() != nil {
			nodeInfo.PublicIPv4 = *nodeEntity.PublicIPv4()
		}
		if nodeEntity.PublicIPv6() != nil {
			nodeInfo.PublicIPv6 = *nodeEntity.PublicIPv6()
		}
	}

	// Create domain entity (user-created rules require subscription and user IDs)
	subscriptionID := cmd.SubscriptionID
	userID := cmd.UserID
	rule, err := externalforward.NewExternalForwardRule(
		&subscriptionID,
		&userID,
		nodeID,
		cmd.Name,
		cmd.ServerAddress,
		cmd.ListenPort,
		cmd.ExternalSource,
		cmd.ExternalRuleID,
		cmd.Remark,
		cmd.SortOrder,
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

	uc.logger.Infow("external forward rule created successfully", "id", rule.SID(), "name", cmd.Name)

	return &CreateExternalForwardRuleResult{
		Rule: dto.FromDomain(rule, cmd.SubscriptionSID, nodeInfo),
	}, nil
}

func (uc *CreateExternalForwardRuleUseCase) validateCommand(cmd CreateExternalForwardRuleCommand) error {
	if cmd.SubscriptionID == 0 {
		return errors.NewValidationError("subscription_id is required")
	}
	if cmd.UserID == 0 {
		return errors.NewValidationError("user_id is required")
	}
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
