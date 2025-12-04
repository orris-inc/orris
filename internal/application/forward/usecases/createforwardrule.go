package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/value_objects"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// CreateForwardRuleCommand represents the input for creating a forward rule.
type CreateForwardRuleCommand struct {
	AgentID       uint
	RuleType      string // direct, entry, exit
	ExitAgentID   uint   // required for entry type
	WsListenPort  uint16 // required for exit type
	Name          string
	ListenPort    uint16 // required for direct and entry types
	TargetAddress string // required for direct and exit types (mutually exclusive with TargetNodeID)
	TargetPort    uint16 // required for direct and exit types (mutually exclusive with TargetNodeID)
	TargetNodeID  *uint  // optional for direct and exit types (mutually exclusive with TargetAddress/TargetPort)
	Protocol      string
	Remark        string
}

// CreateForwardRuleResult represents the output of creating a forward rule.
type CreateForwardRuleResult struct {
	ID            uint   `json:"id"`
	AgentID       uint   `json:"agent_id"`
	RuleType      string `json:"rule_type"`
	ExitAgentID   uint   `json:"exit_agent_id,omitempty"`
	WsListenPort  uint16 `json:"ws_listen_port,omitempty"`
	Name          string `json:"name"`
	ListenPort    uint16 `json:"listen_port"`
	TargetAddress string `json:"target_address,omitempty"`
	TargetPort    uint16 `json:"target_port,omitempty"`
	TargetNodeID  *uint  `json:"target_node_id,omitempty"`
	Protocol      string `json:"protocol"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

// CreateForwardRuleUseCase handles forward rule creation.
type CreateForwardRuleUseCase struct {
	repo     forward.Repository
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

// NewCreateForwardRuleUseCase creates a new CreateForwardRuleUseCase.
func NewCreateForwardRuleUseCase(
	repo forward.Repository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *CreateForwardRuleUseCase {
	return &CreateForwardRuleUseCase{
		repo:     repo,
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

// Execute creates a new forward rule.
func (uc *CreateForwardRuleUseCase) Execute(ctx context.Context, cmd CreateForwardRuleCommand) (*CreateForwardRuleResult, error) {
	uc.logger.Infow("executing create forward rule use case", "name", cmd.Name, "listen_port", cmd.ListenPort)

	if err := uc.validateCommand(ctx, cmd); err != nil {
		uc.logger.Errorw("invalid create forward rule command", "error", err)
		return nil, err
	}

	// Check if listen port is already in use
	exists, err := uc.repo.ExistsByListenPort(ctx, cmd.ListenPort)
	if err != nil {
		uc.logger.Errorw("failed to check existing forward rule", "port", cmd.ListenPort, "error", err)
		return nil, fmt.Errorf("failed to check existing rule: %w", err)
	}
	if exists {
		uc.logger.Warnw("listen port already in use", "port", cmd.ListenPort)
		return nil, errors.NewConflictError("listen port is already in use", fmt.Sprintf("%d", cmd.ListenPort))
	}

	// Create domain entity
	protocol := vo.ForwardProtocol(cmd.Protocol)
	ruleType := vo.ForwardRuleType(cmd.RuleType)
	rule, err := forward.NewForwardRule(
		cmd.AgentID,
		ruleType,
		cmd.ExitAgentID,
		cmd.WsListenPort,
		cmd.Name,
		cmd.ListenPort,
		cmd.TargetAddress,
		cmd.TargetPort,
		cmd.TargetNodeID,
		protocol,
		cmd.Remark,
	)
	if err != nil {
		uc.logger.Errorw("failed to create forward rule entity", "error", err)
		return nil, fmt.Errorf("failed to create forward rule: %w", err)
	}

	// Persist
	if err := uc.repo.Create(ctx, rule); err != nil {
		uc.logger.Errorw("failed to persist forward rule", "error", err)
		return nil, fmt.Errorf("failed to save forward rule: %w", err)
	}

	result := &CreateForwardRuleResult{
		ID:            rule.ID(),
		AgentID:       rule.AgentID(),
		RuleType:      rule.RuleType().String(),
		ExitAgentID:   rule.ExitAgentID(),
		WsListenPort:  rule.WsListenPort(),
		Name:          rule.Name(),
		ListenPort:    rule.ListenPort(),
		TargetAddress: rule.TargetAddress(),
		TargetPort:    rule.TargetPort(),
		TargetNodeID:  rule.TargetNodeID(),
		Protocol:      rule.Protocol().String(),
		Status:        rule.Status().String(),
		CreatedAt:     rule.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	uc.logger.Infow("forward rule created successfully", "id", result.ID, "name", cmd.Name)
	return result, nil
}

func (uc *CreateForwardRuleUseCase) validateCommand(ctx context.Context, cmd CreateForwardRuleCommand) error {
	if cmd.AgentID == 0 {
		return errors.NewValidationError("agent_id is required")
	}
	if cmd.Name == "" {
		return errors.NewValidationError("name is required")
	}
	if cmd.RuleType == "" {
		return errors.NewValidationError("rule_type is required")
	}
	if cmd.Protocol == "" {
		return errors.NewValidationError("protocol is required")
	}

	// Validate rule type
	ruleType := vo.ForwardRuleType(cmd.RuleType)
	if !ruleType.IsValid() {
		return errors.NewValidationError(fmt.Sprintf("invalid rule_type: %s, must be direct, entry or exit", cmd.RuleType))
	}

	// Validate protocol
	protocol := vo.ForwardProtocol(cmd.Protocol)
	if !protocol.IsValid() {
		return errors.NewValidationError(fmt.Sprintf("invalid protocol: %s, must be tcp, udp or both", cmd.Protocol))
	}

	// Type-specific validation
	switch ruleType {
	case vo.ForwardRuleTypeDirect:
		if cmd.ListenPort == 0 {
			return errors.NewValidationError("listen_port is required for direct forward")
		}
		// Either targetAddress+targetPort OR targetNodeID must be provided
		hasTarget := cmd.TargetAddress != "" && cmd.TargetPort != 0
		hasTargetNode := cmd.TargetNodeID != nil && *cmd.TargetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return errors.NewValidationError("either target_address+target_port or target_node_id is required for direct forward")
		}
		if hasTarget && hasTargetNode {
			return errors.NewValidationError("target_address+target_port and target_node_id are mutually exclusive for direct forward")
		}
		// Validate node existence if targetNodeID is provided
		if hasTargetNode {
			if err := uc.validateNodeExists(ctx, *cmd.TargetNodeID); err != nil {
				return err
			}
		}
	case vo.ForwardRuleTypeEntry:
		if cmd.ListenPort == 0 {
			return errors.NewValidationError("listen_port is required for entry forward")
		}
		if cmd.ExitAgentID == 0 {
			return errors.NewValidationError("exit_agent_id is required for entry forward")
		}
	case vo.ForwardRuleTypeExit:
		if cmd.WsListenPort == 0 {
			return errors.NewValidationError("ws_listen_port is required for exit forward")
		}
		// Either targetAddress+targetPort OR targetNodeID must be provided
		hasTarget := cmd.TargetAddress != "" && cmd.TargetPort != 0
		hasTargetNode := cmd.TargetNodeID != nil && *cmd.TargetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return errors.NewValidationError("either target_address+target_port or target_node_id is required for exit forward")
		}
		if hasTarget && hasTargetNode {
			return errors.NewValidationError("target_address+target_port and target_node_id are mutually exclusive for exit forward")
		}
		// Validate node existence if targetNodeID is provided
		if hasTargetNode {
			if err := uc.validateNodeExists(ctx, *cmd.TargetNodeID); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateNodeExists checks if a node exists with the given ID
func (uc *CreateForwardRuleUseCase) validateNodeExists(ctx context.Context, nodeID uint) error {
	node, err := uc.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		uc.logger.Errorw("failed to get node", "node_id", nodeID, "error", err)
		return fmt.Errorf("failed to validate node: %w", err)
	}
	if node == nil {
		uc.logger.Warnw("target node not found", "node_id", nodeID)
		return errors.NewNotFoundError("node", fmt.Sprintf("%d", nodeID))
	}
	return nil
}
