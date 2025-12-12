package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/value_objects"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// CreateForwardRuleCommand represents the input for creating a forward rule.
type CreateForwardRuleCommand struct {
	AgentShortID       string            // Stripe-style short ID (without prefix, e.g., "xK9mP2vL3nQ")
	RuleType           string            // direct, entry, chain, direct_chain
	ExitAgentShortID   string            // required for entry type (Stripe-style short ID without prefix)
	ChainAgentShortIDs []string          // required for chain type (ordered list of Stripe-style short IDs without prefix)
	ChainPortConfig    map[string]uint16 // required for direct_chain type (agent short_id -> listen port)
	Name               string
	ListenPort         uint16 // required for all types (direct, entry, chain, direct_chain)
	TargetAddress      string // required for all types (mutually exclusive with TargetNodeShortID)
	TargetPort         uint16 // required for all types (mutually exclusive with TargetNodeShortID)
	TargetNodeShortID  string // optional for all types (Stripe-style short ID without prefix)
	BindIP             string // optional bind IP address for outbound connections
	IPVersion          string // auto, ipv4, ipv6 (default: auto)
	Protocol           string
	Remark             string
}

// CreateForwardRuleResult represents the output of creating a forward rule.
type CreateForwardRuleResult struct {
	ID            string `json:"id"`       // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	AgentID       uint   `json:"agent_id"` // internal agent ID (will be converted to short ID in handler if needed)
	RuleType      string `json:"rule_type"`
	ExitAgentID   uint   `json:"exit_agent_id,omitempty"`
	Name          string `json:"name"`
	ListenPort    uint16 `json:"listen_port"`
	TargetAddress string `json:"target_address,omitempty"`
	TargetPort    uint16 `json:"target_port,omitempty"`
	TargetNodeID  *uint  `json:"target_node_id,omitempty"`
	IPVersion     string `json:"ip_version"`
	Protocol      string `json:"protocol"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

// CreateForwardRuleUseCase handles forward rule creation.
type CreateForwardRuleUseCase struct {
	repo          forward.Repository
	agentRepo     forward.AgentRepository
	nodeRepo      node.NodeRepository
	configSyncSvc ConfigSyncNotifier
	logger        logger.Interface
}

// NewCreateForwardRuleUseCase creates a new CreateForwardRuleUseCase.
func NewCreateForwardRuleUseCase(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	configSyncSvc ConfigSyncNotifier,
	logger logger.Interface,
) *CreateForwardRuleUseCase {
	return &CreateForwardRuleUseCase{
		repo:          repo,
		agentRepo:     agentRepo,
		nodeRepo:      nodeRepo,
		configSyncSvc: configSyncSvc,
		logger:        logger,
	}
}

// Execute creates a new forward rule.
func (uc *CreateForwardRuleUseCase) Execute(ctx context.Context, cmd CreateForwardRuleCommand) (*CreateForwardRuleResult, error) {
	uc.logger.Infow("executing create forward rule use case", "name", cmd.Name, "listen_port", cmd.ListenPort)

	// Resolve AgentShortID to internal ID
	if cmd.AgentShortID == "" {
		return nil, errors.NewValidationError("agent_id is required")
	}
	agent, err := uc.agentRepo.GetByShortID(ctx, cmd.AgentShortID)
	if err != nil {
		uc.logger.Errorw("failed to get agent", "agent_short_id", cmd.AgentShortID, "error", err)
		return nil, fmt.Errorf("failed to validate agent: %w", err)
	}
	if agent == nil {
		return nil, errors.NewNotFoundError("forward agent", cmd.AgentShortID)
	}
	agentID := agent.ID()

	// Resolve ExitAgentShortID to internal ID (if provided)
	var exitAgentID uint
	if cmd.ExitAgentShortID != "" {
		exitAgent, err := uc.agentRepo.GetByShortID(ctx, cmd.ExitAgentShortID)
		if err != nil {
			uc.logger.Errorw("failed to get exit agent", "exit_agent_short_id", cmd.ExitAgentShortID, "error", err)
			return nil, fmt.Errorf("failed to validate exit agent: %w", err)
		}
		if exitAgent == nil {
			return nil, errors.NewNotFoundError("exit forward agent", cmd.ExitAgentShortID)
		}
		exitAgentID = exitAgent.ID()
	}

	// Resolve ChainAgentShortIDs to internal IDs (if provided)
	var chainAgentIDs []uint
	if len(cmd.ChainAgentShortIDs) > 0 {
		chainAgentIDs = make([]uint, len(cmd.ChainAgentShortIDs))
		for i, shortID := range cmd.ChainAgentShortIDs {
			chainAgent, err := uc.agentRepo.GetByShortID(ctx, shortID)
			if err != nil {
				uc.logger.Errorw("failed to get chain agent", "chain_agent_short_id", shortID, "error", err)
				return nil, fmt.Errorf("failed to validate chain agent: %w", err)
			}
			if chainAgent == nil {
				return nil, errors.NewNotFoundError("chain forward agent", shortID)
			}
			chainAgentIDs[i] = chainAgent.ID()
		}
	}

	// Resolve ChainPortConfig short IDs to internal IDs (if provided for direct_chain type)
	var chainPortConfig map[uint]uint16
	if len(cmd.ChainPortConfig) > 0 {
		chainPortConfig = make(map[uint]uint16, len(cmd.ChainPortConfig))
		for shortID, port := range cmd.ChainPortConfig {
			chainAgent, err := uc.agentRepo.GetByShortID(ctx, shortID)
			if err != nil {
				uc.logger.Errorw("failed to get chain agent for port config", "chain_agent_short_id", shortID, "error", err)
				return nil, fmt.Errorf("failed to validate chain agent in chain_port_config: %w", err)
			}
			if chainAgent == nil {
				return nil, errors.NewNotFoundError("chain forward agent in chain_port_config", shortID)
			}
			chainPortConfig[chainAgent.ID()] = port
		}
	}

	// Resolve TargetNodeShortID to internal ID (if provided)
	var targetNodeID *uint
	if cmd.TargetNodeShortID != "" {
		targetNode, err := uc.nodeRepo.GetByShortID(ctx, cmd.TargetNodeShortID)
		if err != nil {
			uc.logger.Errorw("failed to get target node", "target_node_short_id", cmd.TargetNodeShortID, "error", err)
			return nil, fmt.Errorf("failed to validate target node: %w", err)
		}
		if targetNode == nil {
			return nil, errors.NewNotFoundError("target node", cmd.TargetNodeShortID)
		}
		nodeID := targetNode.ID()
		targetNodeID = &nodeID
	}

	// Validate command with resolved IDs
	if err := uc.validateCommand(ctx, cmd, targetNodeID, chainAgentIDs, chainPortConfig); err != nil {
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
	ipVersion := vo.IPVersion(cmd.IPVersion)
	rule, err := forward.NewForwardRule(
		agentID,
		ruleType,
		exitAgentID,
		chainAgentIDs,
		chainPortConfig,
		cmd.Name,
		cmd.ListenPort,
		cmd.TargetAddress,
		cmd.TargetPort,
		targetNodeID,
		cmd.BindIP,
		ipVersion,
		protocol,
		cmd.Remark,
		id.NewForwardRuleID,
	)
	if err != nil {
		uc.logger.Errorw("failed to create forward rule entity", "error", err)
		return nil, errors.NewValidationError(err.Error())
	}

	// Persist
	if err := uc.repo.Create(ctx, rule); err != nil {
		uc.logger.Errorw("failed to persist forward rule", "error", err)
		return nil, fmt.Errorf("failed to save forward rule: %w", err)
	}

	result := &CreateForwardRuleResult{
		ID:            id.FormatForwardRuleID(rule.ShortID()),
		AgentID:       rule.AgentID(),
		RuleType:      rule.RuleType().String(),
		ExitAgentID:   rule.ExitAgentID(),
		Name:          rule.Name(),
		ListenPort:    rule.ListenPort(),
		TargetAddress: rule.TargetAddress(),
		TargetPort:    rule.TargetPort(),
		TargetNodeID:  rule.TargetNodeID(),
		IPVersion:     rule.IPVersion().String(),
		Protocol:      rule.Protocol().String(),
		Status:        rule.Status().String(),
		CreatedAt:     rule.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	uc.logger.Infow("forward rule created successfully", "id", result.ID, "name", cmd.Name)

	// Notify config sync asynchronously if rule is enabled (failure only logs warning, doesn't block)
	if rule.IsEnabled() && uc.configSyncSvc != nil {
		go func() {
			if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), rule.AgentID(), rule.ShortID(), "added"); err != nil {
				uc.logger.Warnw("failed to notify config sync", "rule_id", rule.ShortID(), "error", err)
			}
		}()
	}

	return result, nil
}

func (uc *CreateForwardRuleUseCase) validateCommand(_ context.Context, cmd CreateForwardRuleCommand, targetNodeID *uint, chainAgentIDs []uint, chainPortConfig map[uint]uint16) error {
	// AgentShortID validation is done in Execute before calling this method
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
		return errors.NewValidationError(fmt.Sprintf("invalid rule_type: %s, must be direct, entry, chain, or direct_chain", cmd.RuleType))
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
		// Either targetAddress+targetPort OR targetNodeShortID must be provided
		hasTarget := cmd.TargetAddress != "" && cmd.TargetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return errors.NewValidationError("either target_address+target_port or target_node_id is required for direct forward")
		}
		if hasTarget && hasTargetNode {
			return errors.NewValidationError("target_address+target_port and target_node_id are mutually exclusive for direct forward")
		}
	case vo.ForwardRuleTypeEntry:
		if cmd.ListenPort == 0 {
			return errors.NewValidationError("listen_port is required for entry forward")
		}
		if cmd.ExitAgentShortID == "" {
			return errors.NewValidationError("exit_agent_id is required for entry forward")
		}
		// Entry rules now also require target information (to be passed to exit agent)
		hasTarget := cmd.TargetAddress != "" && cmd.TargetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return errors.NewValidationError("either target_address+target_port or target_node_id is required for entry forward")
		}
		if hasTarget && hasTargetNode {
			return errors.NewValidationError("target_address+target_port and target_node_id are mutually exclusive for entry forward")
		}
	case vo.ForwardRuleTypeChain:
		if cmd.ListenPort == 0 {
			return errors.NewValidationError("listen_port is required for chain forward")
		}
		if len(cmd.ChainAgentShortIDs) == 0 {
			return errors.NewValidationError("chain_agent_ids is required for chain forward (at least 1 intermediate agent)")
		}
		if len(cmd.ChainAgentShortIDs) > 10 {
			return errors.NewValidationError("chain forward supports maximum 10 intermediate agents")
		}
		// Chain rules require target information (at the end of chain)
		hasTarget := cmd.TargetAddress != "" && cmd.TargetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return errors.NewValidationError("either target_address+target_port or target_node_id is required for chain forward")
		}
		if hasTarget && hasTargetNode {
			return errors.NewValidationError("target_address+target_port and target_node_id are mutually exclusive for chain forward")
		}
	case vo.ForwardRuleTypeDirectChain:
		// Validate chain_agent_ids
		if len(cmd.ChainAgentShortIDs) == 0 {
			return errors.NewValidationError("chain_agent_ids is required for direct_chain forward (at least 1 intermediate agent)")
		}
		if len(cmd.ChainAgentShortIDs) > 10 {
			return errors.NewValidationError("direct_chain forward supports maximum 10 intermediate agents")
		}
		// Validate chain_port_config provides port for each chain agent
		if len(cmd.ChainPortConfig) == 0 {
			return errors.NewValidationError("chain_port_config is required for direct_chain forward")
		}
		// Validate chain_port_config has port for every chain agent
		for _, agentID := range chainAgentIDs {
			if _, exists := chainPortConfig[agentID]; !exists {
				return errors.NewValidationError(fmt.Sprintf("chain_port_config must provide listen port for all chain agents (missing agent_id: %d)", agentID))
			}
		}
		// Direct chain rules require target information (at the end of chain)
		hasTarget := cmd.TargetAddress != "" && cmd.TargetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return errors.NewValidationError("either target_address+target_port or target_node_id is required for direct_chain forward")
		}
		if hasTarget && hasTargetNode {
			return errors.NewValidationError("target_address+target_port and target_node_id are mutually exclusive for direct_chain forward")
		}
	default:
		return errors.NewValidationError(fmt.Sprintf("invalid rule_type: %s, must be direct, entry, chain, or direct_chain", cmd.RuleType))
	}

	return nil
}
