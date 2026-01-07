package usecases

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// CreateUserForwardRuleCommand represents the input for creating a user forward rule.
type CreateUserForwardRuleCommand struct {
	UserID             uint              // user ID for user-owned rules
	AgentShortID       string            // Stripe-style short ID (without prefix, e.g., "xK9mP2vL3nQ")
	RuleType           string            // direct, entry, chain, direct_chain
	ExitAgentShortID   string            // required for entry type (Stripe-style short ID without prefix)
	ChainAgentShortIDs []string          // required for chain type (ordered list of Stripe-style short IDs without prefix)
	ChainPortConfig    map[string]uint16 // required for direct_chain type or hybrid chain direct hops (agent short_id -> listen port)
	TunnelHops         *int              // number of hops using tunnel (nil=full tunnel, N=first N hops use tunnel) - for chain type only
	TunnelType         string            // tunnel type: ws or tls (default: ws)
	Name               string
	ListenPort         uint16 // listen port (0 = auto-assign from agent's allowed range)
	TargetAddress      string // required for all types (mutually exclusive with TargetNodeSID)
	TargetPort         uint16 // required for all types (mutually exclusive with TargetNodeSID)
	TargetNodeSID      string // optional for all types (Stripe-style short ID without prefix)
	BindIP             string // optional bind IP address for outbound connections
	IPVersion          string // auto, ipv4, ipv6 (default: auto)
	Protocol          string
	TrafficMultiplier *float64 // optional traffic multiplier (nil for auto-calculation, 0-1000000)
	SortOrder         *int     // optional sort order (nil defaults to 0)
	Remark            string
}

// CreateUserForwardRuleResult represents the output of creating a user forward rule.
type CreateUserForwardRuleResult struct {
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

// CreateUserForwardRuleUseCase handles user forward rule creation.
type CreateUserForwardRuleUseCase struct {
	repo          forward.Repository
	agentRepo     forward.AgentRepository
	nodeRepo      node.NodeRepository
	configSyncSvc ConfigSyncNotifier
	logger        logger.Interface
}

// NewCreateUserForwardRuleUseCase creates a new CreateUserForwardRuleUseCase.
func NewCreateUserForwardRuleUseCase(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	configSyncSvc ConfigSyncNotifier,
	logger logger.Interface,
) *CreateUserForwardRuleUseCase {
	return &CreateUserForwardRuleUseCase{
		repo:          repo,
		agentRepo:     agentRepo,
		nodeRepo:      nodeRepo,
		configSyncSvc: configSyncSvc,
		logger:        logger,
	}
}

// Execute creates a new user forward rule with quota checks.
func (uc *CreateUserForwardRuleUseCase) Execute(ctx context.Context, cmd CreateUserForwardRuleCommand) (*CreateUserForwardRuleResult, error) {
	uc.logger.Infow("executing create user forward rule use case", "user_id", cmd.UserID, "name", cmd.Name, "listen_port", cmd.ListenPort)

	// Validate user ID is provided
	if cmd.UserID == 0 {
		return nil, errors.NewValidationError("user_id is required")
	}

	// Resolve AgentShortID to internal ID
	if cmd.AgentShortID == "" {
		return nil, errors.NewValidationError("agent_id is required")
	}
	agent, err := uc.agentRepo.GetBySID(ctx, cmd.AgentShortID)
	if err != nil {
		uc.logger.Errorw("failed to get agent", "agent_short_id", cmd.AgentShortID, "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to validate agent: %w", err)
	}
	if agent == nil {
		return nil, errors.NewNotFoundError("forward agent", cmd.AgentShortID)
	}
	agentID := agent.ID()

	// Auto-assign listen port if not specified
	if cmd.ListenPort == 0 {
		port, err := uc.assignAvailablePort(ctx, agent)
		if err != nil {
			uc.logger.Errorw("failed to auto-assign listen port", "agent_id", agentID, "user_id", cmd.UserID, "error", err)
			return nil, err
		}
		cmd.ListenPort = port
		uc.logger.Infow("auto-assigned listen port", "port", port, "agent_id", agentID, "user_id", cmd.UserID)
	}

	// Validate listen port against agent's allowed port range
	if !agent.IsPortAllowed(cmd.ListenPort) {
		return nil, errors.NewValidationError(
			fmt.Sprintf("listen port %d is not allowed for this agent, allowed ranges: %s",
				cmd.ListenPort, agent.AllowedPortRange().String()))
	}

	// Resolve ExitAgentShortID to internal ID (if provided)
	var exitAgentID uint
	if cmd.ExitAgentShortID != "" {
		exitAgent, err := uc.agentRepo.GetBySID(ctx, cmd.ExitAgentShortID)
		if err != nil {
			uc.logger.Errorw("failed to get exit agent", "exit_agent_short_id", cmd.ExitAgentShortID, "user_id", cmd.UserID, "error", err)
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
			chainAgent, err := uc.agentRepo.GetBySID(ctx, shortID)
			if err != nil {
				uc.logger.Errorw("failed to get chain agent", "chain_agent_short_id", shortID, "user_id", cmd.UserID, "error", err)
				return nil, fmt.Errorf("failed to validate chain agent: %w", err)
			}
			if chainAgent == nil {
				return nil, errors.NewNotFoundError("chain forward agent", shortID)
			}
			chainAgentIDs[i] = chainAgent.ID()
		}
	}

	// Resolve ChainPortConfig short IDs to internal IDs (if provided for direct_chain type)
	// Also validate that each port is within the corresponding agent's allowed port range
	// and check for port conflicts on each chain agent
	var chainPortConfig map[uint]uint16
	if len(cmd.ChainPortConfig) > 0 {
		chainPortConfig = make(map[uint]uint16, len(cmd.ChainPortConfig))
		for shortID, port := range cmd.ChainPortConfig {
			chainAgent, err := uc.agentRepo.GetBySID(ctx, shortID)
			if err != nil {
				uc.logger.Errorw("failed to get chain agent for port config", "chain_agent_short_id", shortID, "user_id", cmd.UserID, "error", err)
				return nil, fmt.Errorf("failed to validate chain agent in chain_port_config: %w", err)
			}
			if chainAgent == nil {
				return nil, errors.NewNotFoundError("chain forward agent in chain_port_config", shortID)
			}
			// Validate port against chain agent's allowed port range
			if !chainAgent.IsPortAllowed(port) {
				return nil, errors.NewValidationError(
					fmt.Sprintf("listen port %d is not allowed for chain agent %s, allowed ranges: %s",
						port, shortID, chainAgent.AllowedPortRange().String()))
			}
			// Check if the port is already in use on this chain agent (including other rules' chain_port_config)
			inUse, err := uc.repo.IsPortInUseByAgent(ctx, chainAgent.ID(), port, 0)
			if err != nil {
				uc.logger.Errorw("failed to check chain agent port", "chain_agent_id", chainAgent.ID(), "port", port, "user_id", cmd.UserID, "error", err)
				return nil, fmt.Errorf("failed to check chain agent port: %w", err)
			}
			if inUse {
				return nil, errors.NewConflictError(
					fmt.Sprintf("listen port %d is already in use on chain agent %s", port, shortID),
					fmt.Sprintf("%d", port))
			}
			chainPortConfig[chainAgent.ID()] = port
		}
	}

	// Resolve TargetNodeSID to internal ID (if provided)
	var targetNodeID *uint
	if cmd.TargetNodeSID != "" {
		targetNode, err := uc.nodeRepo.GetBySID(ctx, cmd.TargetNodeSID)
		if err != nil {
			uc.logger.Errorw("failed to get target node", "target_node_sid", cmd.TargetNodeSID, "user_id", cmd.UserID, "error", err)
			return nil, fmt.Errorf("failed to validate target node: %w", err)
		}
		if targetNode == nil {
			return nil, errors.NewNotFoundError("target node", cmd.TargetNodeSID)
		}

		// Verify user owns the target node
		if !targetNode.IsOwnedBy(cmd.UserID) {
			uc.logger.Warnw("user attempted to use node they don't own as target",
				"user_id", cmd.UserID,
				"target_node_sid", cmd.TargetNodeSID,
				"node_owner", targetNode.UserID(),
			)
			return nil, errors.NewForbiddenError("cannot use this node as target")
		}

		nodeID := targetNode.ID()
		targetNodeID = &nodeID
	}

	// Validate command with resolved IDs
	if err := uc.validateCommand(ctx, cmd, targetNodeID, chainAgentIDs, chainPortConfig); err != nil {
		uc.logger.Errorw("invalid create user forward rule command", "user_id", cmd.UserID, "error", err)
		return nil, err
	}

	// Check if listen port is already in use on this agent (including other rules' chain_port_config)
	inUse, err := uc.repo.IsPortInUseByAgent(ctx, agentID, cmd.ListenPort, 0)
	if err != nil {
		uc.logger.Errorw("failed to check existing forward rule", "agent_id", agentID, "port", cmd.ListenPort, "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to check existing rule: %w", err)
	}
	if inUse {
		uc.logger.Warnw("listen port already in use on this agent", "agent_id", agentID, "port", cmd.ListenPort, "user_id", cmd.UserID)
		return nil, errors.NewConflictError("listen port is already in use on this agent", fmt.Sprintf("%d", cmd.ListenPort))
	}

	// Create domain entity with user_id
	protocol := vo.ForwardProtocol(cmd.Protocol)
	ruleType := vo.ForwardRuleType(cmd.RuleType)
	ipVersion := vo.IPVersion(cmd.IPVersion)
	tunnelType := vo.TunnelType(cmd.TunnelType)
	userIDPtr := &cmd.UserID
	rule, err := forward.NewForwardRule(
		agentID,
		userIDPtr,
		nil, // subscriptionID is nil for user-created rules (not subscription-bound)
		ruleType,
		exitAgentID,
		chainAgentIDs,
		chainPortConfig,
		cmd.TunnelHops,
		tunnelType,
		cmd.Name,
		cmd.ListenPort,
		cmd.TargetAddress,
		cmd.TargetPort,
		targetNodeID,
		cmd.BindIP,
		ipVersion,
		protocol,
		cmd.Remark,
		cmd.TrafficMultiplier,
		derefIntOrDefault(cmd.SortOrder, 0),
		id.NewForwardRuleID,
	)
	if err != nil {
		uc.logger.Errorw("failed to create forward rule entity", "user_id", cmd.UserID, "error", err)
		return nil, errors.NewValidationError(err.Error())
	}

	// Persist
	if err := uc.repo.Create(ctx, rule); err != nil {
		uc.logger.Errorw("failed to persist user forward rule", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to save forward rule: %w", err)
	}

	result := &CreateUserForwardRuleResult{
		ID:            rule.SID(),
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

	uc.logger.Infow("user forward rule created successfully", "user_id", cmd.UserID, "id", result.ID, "name", cmd.Name)

	// Notify config sync asynchronously if rule is enabled (failure only logs warning, doesn't block)
	if rule.IsEnabled() && uc.configSyncSvc != nil {
		go func() {
			if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), rule.AgentID(), rule.SID(), "added"); err != nil {
				uc.logger.Debugw("config sync notification skipped", "rule_id", rule.SID(), "user_id", cmd.UserID, "reason", err.Error())
			}
		}()
	}

	return result, nil
}

func (uc *CreateUserForwardRuleUseCase) validateCommand(_ context.Context, cmd CreateUserForwardRuleCommand, targetNodeID *uint, chainAgentIDs []uint, chainPortConfig map[uint]uint16) error {
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
		// Either targetAddress+targetPort OR targetNodeSID must be provided
		hasTarget := cmd.TargetAddress != "" && cmd.TargetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return errors.NewValidationError("either target_address+target_port or target_node_id is required for direct forward")
		}
		if hasTarget && hasTargetNode {
			return errors.NewValidationError("target_address+target_port and target_node_id are mutually exclusive for direct forward")
		}
	case vo.ForwardRuleTypeEntry:
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

// Default port range for auto-assignment when agent has no port restrictions.
const (
	userDefaultPortRangeStart = 10000
	userDefaultPortRangeEnd   = 60000
	userMaxPortAssignAttempts = 100
)

// assignAvailablePort finds an available port for the given agent.
// If the agent has allowed port ranges configured, picks from those ranges.
// Otherwise, uses the default range (10000-60000).
// Note: Port uniqueness is checked per-agent, not globally.
func (uc *CreateUserForwardRuleUseCase) assignAvailablePort(ctx context.Context, agent *forward.ForwardAgent) (uint16, error) {
	portRange := agent.AllowedPortRange()
	agentID := agent.ID()

	for i := 0; i < userMaxPortAssignAttempts; i++ {
		var port uint16
		if portRange != nil && !portRange.IsEmpty() {
			// Use agent's allowed port range
			port = portRange.RandomPort()
		} else {
			// Use default range
			port = uint16(userDefaultPortRangeStart + rand.Intn(userDefaultPortRangeEnd-userDefaultPortRangeStart+1))
		}

		// Defensive check: ensure port is within agent's allowed range
		if !agent.IsPortAllowed(port) {
			continue
		}

		// Check if port is already in use on this agent (including chain_port_config)
		inUse, err := uc.repo.IsPortInUseByAgent(ctx, agentID, port, 0)
		if err != nil {
			return 0, fmt.Errorf("failed to check port availability: %w", err)
		}
		if !inUse {
			return port, nil
		}
	}

	return 0, errors.NewValidationError("failed to find available port after maximum attempts")
}
