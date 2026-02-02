package usecases

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ExitAgentInput represents input for exit agent with weight.
type ExitAgentInput struct {
	AgentSID string // Stripe-style ID (e.g., "fa_xK9mP2vL3nQ")
	Weight   uint16 // Load balancing weight (1-100)
}

// CreateForwardRuleCommand represents the input for creating a forward rule.
type CreateForwardRuleCommand struct {
	AgentShortID        string            // Stripe-style short ID (without prefix, e.g., "xK9mP2vL3nQ") - not required for external type
	UserID              *uint             // user ID for user-owned rules (nil for admin-created rules)
	RuleType            string            // direct, entry, chain, direct_chain, external
	ExitAgentShortID    string            // for entry type (Stripe-style short ID, mutually exclusive with ExitAgents)
	ExitAgents          []ExitAgentInput  // for entry type with load balancing (mutually exclusive with ExitAgentShortID)
	LoadBalanceStrategy string            // load balance strategy: failover (default), weighted
	ChainAgentShortIDs  []string          // required for chain type (ordered list of Stripe-style short IDs without prefix)
	ChainPortConfig     map[string]uint16 // required for direct_chain type or hybrid chain direct hops (agent short_id -> listen port)
	TunnelHops          *int              // number of hops using tunnel (nil=full tunnel, N=first N hops use tunnel) - for chain type only
	TunnelType          string            // tunnel type: ws or tls (default: ws)
	Name                string
	ListenPort          uint16   // listen port (0 = auto-assign from agent's allowed range, required for external type)
	TargetAddress       string   // required for all types except external (mutually exclusive with TargetNodeSID)
	TargetPort          uint16   // required for all types except external (mutually exclusive with TargetNodeSID)
	TargetNodeSID       string   // optional for all types (Stripe-style short ID without prefix)
	BindIP              string   // optional bind IP address for outbound connections
	IPVersion           string   // auto, ipv4, ipv6 (default: auto)
	Protocol            string   // not required for external type (protocol derived from target_node)
	TrafficMultiplier   *float64 // optional traffic multiplier (nil for auto-calculation, 0-1000000)
	SortOrder           *int     // optional sort order (nil defaults to 0)
	Remark              string
	GroupSIDs           []string // optional resource group SIDs (admin only)
	// External rule fields (only for rule_type=external)
	ServerAddress  string // required for external type - server address for subscription delivery
	ExternalSource string // required for external type - source identifier
	ExternalRuleID string // optional for external type - external reference ID
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
	repo              forward.Repository
	agentRepo         forward.AgentRepository
	nodeRepo          node.NodeRepository
	resourceGroupRepo resource.Repository
	planRepo          subscription.PlanRepository
	configSyncSvc     ConfigSyncNotifier
	logger            logger.Interface
}

// NewCreateForwardRuleUseCase creates a new CreateForwardRuleUseCase.
func NewCreateForwardRuleUseCase(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	resourceGroupRepo resource.Repository,
	planRepo subscription.PlanRepository,
	configSyncSvc ConfigSyncNotifier,
	logger logger.Interface,
) *CreateForwardRuleUseCase {
	return &CreateForwardRuleUseCase{
		repo:              repo,
		agentRepo:         agentRepo,
		nodeRepo:          nodeRepo,
		resourceGroupRepo: resourceGroupRepo,
		planRepo:          planRepo,
		configSyncSvc:     configSyncSvc,
		logger:            logger,
	}
}

// Execute creates a new forward rule.
func (uc *CreateForwardRuleUseCase) Execute(ctx context.Context, cmd CreateForwardRuleCommand) (*CreateForwardRuleResult, error) {
	uc.logger.Infow("executing create forward rule use case", "name", cmd.Name, "listen_port", cmd.ListenPort, "rule_type", cmd.RuleType)

	ruleType := vo.ForwardRuleType(cmd.RuleType)

	// Handle external rules separately (they don't require agent)
	if ruleType.IsExternal() {
		return uc.executeExternalRule(ctx, cmd)
	}

	// Resolve AgentShortID to internal ID (required for non-external rules)
	if cmd.AgentShortID == "" {
		return nil, errors.NewValidationError("agent_id is required")
	}
	agent, err := uc.agentRepo.GetBySID(ctx, cmd.AgentShortID)
	if err != nil {
		uc.logger.Errorw("failed to get agent", "agent_short_id", cmd.AgentShortID, "error", err)
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
			uc.logger.Errorw("failed to auto-assign listen port", "agent_id", agentID, "error", err)
			return nil, err
		}
		cmd.ListenPort = port
		uc.logger.Infow("auto-assigned listen port", "port", port, "agent_id", agentID)
	}

	// Validate listen port against agent's allowed port range
	if !agent.IsPortAllowed(cmd.ListenPort) {
		return nil, errors.NewValidationError(
			fmt.Sprintf("listen port %d is not allowed for this agent, allowed ranges: %s",
				cmd.ListenPort, agent.AllowedPortRange().String()))
	}

	// Resolve exit agent configuration (single exitAgentID OR multiple exitAgents)
	var exitAgentID uint
	var exitAgents []vo.AgentWeight
	if cmd.ExitAgentShortID != "" && len(cmd.ExitAgents) > 0 {
		return nil, errors.NewValidationError("exit_agent_id and exit_agents are mutually exclusive")
	}
	if cmd.ExitAgentShortID != "" {
		exitAgent, err := uc.agentRepo.GetBySID(ctx, cmd.ExitAgentShortID)
		if err != nil {
			uc.logger.Errorw("failed to get exit agent", "exit_agent_short_id", cmd.ExitAgentShortID, "error", err)
			return nil, fmt.Errorf("failed to validate exit agent: %w", err)
		}
		if exitAgent == nil {
			return nil, errors.NewNotFoundError("exit forward agent", cmd.ExitAgentShortID)
		}
		exitAgentID = exitAgent.ID()
	} else if len(cmd.ExitAgents) > 0 {
		// Resolve multiple exit agents with weights
		exitAgents = make([]vo.AgentWeight, 0, len(cmd.ExitAgents))
		for _, input := range cmd.ExitAgents {
			exitAgent, err := uc.agentRepo.GetBySID(ctx, input.AgentSID)
			if err != nil {
				uc.logger.Errorw("failed to get exit agent", "exit_agent_sid", input.AgentSID, "error", err)
				return nil, fmt.Errorf("failed to validate exit agent: %w", err)
			}
			if exitAgent == nil {
				return nil, errors.NewNotFoundError("exit forward agent", input.AgentSID)
			}
			// Use provided weight or default
			weight := input.Weight
			if weight == 0 {
				weight = vo.DefaultAgentWeight
			}
			aw, err := vo.NewAgentWeight(exitAgent.ID(), weight)
			if err != nil {
				return nil, errors.NewValidationError(fmt.Sprintf("invalid exit agent weight: %s", err.Error()))
			}
			exitAgents = append(exitAgents, aw)
		}
		// Validate exit agents
		if err := vo.ValidateAgentWeights(exitAgents); err != nil {
			return nil, errors.NewValidationError(fmt.Sprintf("invalid exit agents: %s", err.Error()))
		}
	}

	// Resolve ChainAgentShortIDs to internal IDs (if provided)
	var chainAgentIDs []uint
	if len(cmd.ChainAgentShortIDs) > 0 {
		chainAgentIDs = make([]uint, len(cmd.ChainAgentShortIDs))
		for i, shortID := range cmd.ChainAgentShortIDs {
			chainAgent, err := uc.agentRepo.GetBySID(ctx, shortID)
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
	// Also validate that each port is within the corresponding agent's allowed port range
	// and check for port conflicts on each chain agent
	var chainPortConfig map[uint]uint16
	if len(cmd.ChainPortConfig) > 0 {
		chainPortConfig = make(map[uint]uint16, len(cmd.ChainPortConfig))
		for shortID, port := range cmd.ChainPortConfig {
			chainAgent, err := uc.agentRepo.GetBySID(ctx, shortID)
			if err != nil {
				uc.logger.Errorw("failed to get chain agent for port config", "chain_agent_short_id", shortID, "error", err)
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
				uc.logger.Errorw("failed to check chain agent port", "chain_agent_id", chainAgent.ID(), "port", port, "error", err)
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
			uc.logger.Errorw("failed to get target node", "target_node_sid", cmd.TargetNodeSID, "error", err)
			return nil, fmt.Errorf("failed to validate target node: %w", err)
		}
		if targetNode == nil {
			return nil, errors.NewNotFoundError("target node", cmd.TargetNodeSID)
		}

		// Ownership constraint: prevent cross-ownership target node assignment
		// User rules can only target user-owned nodes belonging to the same user
		// System rules can only target system nodes (non-user-owned)
		if cmd.UserID != nil && *cmd.UserID != 0 {
			// User rule: target node must be owned by the same user
			if !targetNode.IsOwnedBy(*cmd.UserID) {
				uc.logger.Warnw("user rule cannot target node not owned by user",
					"user_id", *cmd.UserID,
					"target_node_sid", cmd.TargetNodeSID,
				)
				return nil, errors.NewForbiddenError("user rules can only target nodes owned by the same user")
			}
		} else {
			// System rule: target node must not be user-owned
			if targetNode.IsUserOwned() {
				uc.logger.Warnw("system rule cannot target user-owned node",
					"target_node_sid", cmd.TargetNodeSID,
				)
				return nil, errors.NewForbiddenError("system rules cannot target user-owned nodes")
			}
		}

		nodeID := targetNode.ID()
		targetNodeID = &nodeID
	}

	// Validate command with resolved IDs
	if err := uc.validateCommand(ctx, cmd, targetNodeID, chainAgentIDs, chainPortConfig); err != nil {
		uc.logger.Errorw("invalid create forward rule command", "error", err)
		return nil, err
	}

	// Check if listen port is already in use on this agent (including other rules' chain_port_config)
	inUse, err := uc.repo.IsPortInUseByAgent(ctx, agentID, cmd.ListenPort, 0)
	if err != nil {
		uc.logger.Errorw("failed to check existing forward rule", "agent_id", agentID, "port", cmd.ListenPort, "error", err)
		return nil, fmt.Errorf("failed to check existing rule: %w", err)
	}
	if inUse {
		uc.logger.Warnw("listen port already in use on this agent", "agent_id", agentID, "port", cmd.ListenPort)
		return nil, errors.NewConflictError("listen port is already in use on this agent", fmt.Sprintf("%d", cmd.ListenPort))
	}

	// Resolve GroupSIDs to internal IDs and validate plan types (if provided)
	var groupIDs []uint
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

			// Verify the plan type supports forward rules binding (node and hybrid only, not forward)
			plan, err := uc.planRepo.GetByID(ctx, group.PlanID())
			if err != nil {
				uc.logger.Errorw("failed to get plan for resource group", "plan_id", group.PlanID(), "error", err)
				return nil, fmt.Errorf("failed to validate resource group plan: %w", err)
			}
			if plan == nil {
				return nil, fmt.Errorf("plan not found for resource group %s", groupSID)
			}
			if plan.PlanType().IsForward() {
				uc.logger.Warnw("attempted to bind forward rule to forward plan resource group",
					"group_sid", groupSID,
					"plan_id", group.PlanID(),
					"plan_type", plan.PlanType().String())
				return nil, errors.NewValidationError(
					fmt.Sprintf("resource group %s belongs to a forward plan and cannot bind forward rules", groupSID))
			}

			groupIDs = append(groupIDs, group.ID())
		}
	}

	// Create domain entity
	protocol := vo.ForwardProtocol(cmd.Protocol)
	// ruleType already defined at the beginning of Execute
	ipVersion := vo.IPVersion(cmd.IPVersion)
	tunnelType := vo.TunnelType(cmd.TunnelType)
	loadBalanceStrategy := vo.ParseLoadBalanceStrategy(cmd.LoadBalanceStrategy)
	rule, err := forward.NewForwardRule(
		agentID,
		cmd.UserID,
		nil, // subscriptionID is nil for admin-created rules
		ruleType,
		exitAgentID,
		exitAgents,
		loadBalanceStrategy,
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
		uc.logger.Errorw("failed to create forward rule entity", "error", err)
		return nil, errors.NewValidationError(err.Error())
	}

	// Set group IDs if provided
	if len(groupIDs) > 0 {
		rule.SetGroupIDs(groupIDs)
	}

	// Persist
	if err := uc.repo.Create(ctx, rule); err != nil {
		uc.logger.Errorw("failed to persist forward rule", "error", err)
		return nil, fmt.Errorf("failed to save forward rule: %w", err)
	}

	result := &CreateForwardRuleResult{
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

	uc.logger.Infow("forward rule created successfully", "id", result.ID, "name", cmd.Name)

	// Notify config sync asynchronously if rule is enabled (failure only logs warning, doesn't block)
	if rule.IsEnabled() && uc.configSyncSvc != nil {
		// Notify entry agent
		go func() {
			if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), rule.AgentID(), rule.SID(), "added"); err != nil {
				uc.logger.Debugw("config sync notification skipped for entry agent", "rule_id", rule.SID(), "agent_id", rule.AgentID(), "reason", err.Error())
			}
		}()

		// Notify additional agents based on rule type
		switch rule.RuleType().String() {
		case "entry":
			// Notify exit agent(s) for entry type rules
			exitAgentIDs := rule.GetAllExitAgentIDs()
			for _, exitAgentID := range exitAgentIDs {
				go func(aid uint) {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), aid, rule.SID(), "added"); err != nil {
						uc.logger.Debugw("config sync notification skipped for exit agent", "rule_id", rule.SID(), "agent_id", aid, "reason", err.Error())
					}
				}(exitAgentID)
			}
		case "chain", "direct_chain":
			// Notify all chain agents for chain and direct_chain type rules
			for _, agentID := range rule.ChainAgentIDs() {
				go func(aid uint) {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), aid, rule.SID(), "added"); err != nil {
						uc.logger.Debugw("config sync notification skipped for chain agent", "rule_id", rule.SID(), "agent_id", aid, "reason", err.Error())
					}
				}(agentID)
			}
		}
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

	// Validate rule type (external rules are handled separately in executeExternalRule)
	ruleType := vo.ForwardRuleType(cmd.RuleType)
	if !ruleType.IsValid() {
		return errors.NewValidationError(fmt.Sprintf("invalid rule_type: %s, must be direct, entry, chain, direct_chain, or external", cmd.RuleType))
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
		// Either exit_agent_id OR exit_agents is required (mutually exclusive)
		hasExitAgent := cmd.ExitAgentShortID != ""
		hasExitAgents := len(cmd.ExitAgents) > 0
		if !hasExitAgent && !hasExitAgents {
			return errors.NewValidationError("either exit_agent_id or exit_agents is required for entry forward")
		}
		if hasExitAgent && hasExitAgents {
			return errors.NewValidationError("exit_agent_id and exit_agents are mutually exclusive for entry forward")
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
	case vo.ForwardRuleTypeExternal:
		// External rules are handled in executeExternalRule, not here
		// This case should not be reached, but handle it gracefully
		return nil
	default:
		return errors.NewValidationError(fmt.Sprintf("invalid rule_type: %s, must be direct, entry, chain, direct_chain, or external", cmd.RuleType))
	}

	return nil
}

// derefIntOrDefault returns the dereferenced value or the default if nil.
func derefIntOrDefault(ptr *int, defaultVal int) int {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

// Default port range for auto-assignment when agent has no port restrictions.
const (
	defaultPortRangeStart = 10000
	defaultPortRangeEnd   = 60000
	maxPortAssignAttempts = 100
)

// assignAvailablePort finds an available port for the given agent.
// If the agent has allowed port ranges configured, picks from those ranges.
// Otherwise, uses the default range (10000-60000).
// Note: Port uniqueness is checked per-agent, not globally.
func (uc *CreateForwardRuleUseCase) assignAvailablePort(ctx context.Context, agent *forward.ForwardAgent) (uint16, error) {
	portRange := agent.AllowedPortRange()
	agentID := agent.ID()

	for i := 0; i < maxPortAssignAttempts; i++ {
		var port uint16
		if portRange != nil && !portRange.IsEmpty() {
			// Use agent's allowed port range
			port = portRange.RandomPort()
		} else {
			// Use default range: port range is 10000-65000, so result always fits in uint16
			randomOffset := rand.Intn(defaultPortRangeEnd - defaultPortRangeStart + 1)
			portVal := defaultPortRangeStart + randomOffset
			// #nosec G115 -- portVal is bounded by defaultPortRangeStart(10000) to defaultPortRangeEnd(65000)
			port = uint16(portVal)
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

// executeExternalRule handles external rule creation.
// External rules don't require an agent; they use serverAddress for subscription delivery.
func (uc *CreateForwardRuleUseCase) executeExternalRule(ctx context.Context, cmd CreateForwardRuleCommand) (*CreateForwardRuleResult, error) {
	uc.logger.Infow("executing create external forward rule use case", "name", cmd.Name, "server_address", cmd.ServerAddress)

	// Validate required fields for external rules
	if cmd.Name == "" {
		return nil, errors.NewValidationError("name is required")
	}
	if cmd.ServerAddress == "" {
		return nil, errors.NewValidationError("server_address is required for external rules")
	}
	if cmd.ListenPort == 0 {
		return nil, errors.NewValidationError("listen_port is required for external rules")
	}
	if cmd.TargetNodeSID == "" {
		return nil, errors.NewValidationError("target_node_id is required for external rules (protocol info is derived from target node)")
	}
	// external_source is optional

	// Resolve TargetNodeSID to internal ID (optional for external rules, used for protocol info)
	var targetNodeID *uint
	if cmd.TargetNodeSID != "" {
		targetNode, err := uc.nodeRepo.GetBySID(ctx, cmd.TargetNodeSID)
		if err != nil {
			uc.logger.Errorw("failed to get target node", "target_node_sid", cmd.TargetNodeSID, "error", err)
			return nil, fmt.Errorf("failed to validate target node: %w", err)
		}
		if targetNode == nil {
			return nil, errors.NewNotFoundError("target node", cmd.TargetNodeSID)
		}

		// Ownership constraint for external rules
		if cmd.UserID != nil && *cmd.UserID != 0 {
			if !targetNode.IsOwnedBy(*cmd.UserID) {
				uc.logger.Warnw("user rule cannot target node not owned by user",
					"user_id", *cmd.UserID,
					"target_node_sid", cmd.TargetNodeSID,
				)
				return nil, errors.NewForbiddenError("user rules can only target nodes owned by the same user")
			}
		} else {
			if targetNode.IsUserOwned() {
				uc.logger.Warnw("system rule cannot target user-owned node",
					"target_node_sid", cmd.TargetNodeSID,
				)
				return nil, errors.NewForbiddenError("system rules cannot target user-owned nodes")
			}
		}

		nodeID := targetNode.ID()
		targetNodeID = &nodeID
	}

	// Resolve GroupSIDs to internal IDs and validate plan types (if provided)
	var groupIDs []uint
	if len(cmd.GroupSIDs) > 0 {
		groupIDs = make([]uint, 0, len(cmd.GroupSIDs))
		for _, groupSID := range cmd.GroupSIDs {
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

			// Verify the plan type supports forward rules binding
			plan, err := uc.planRepo.GetByID(ctx, group.PlanID())
			if err != nil {
				uc.logger.Errorw("failed to get plan for resource group", "plan_id", group.PlanID(), "error", err)
				return nil, fmt.Errorf("failed to validate resource group plan: %w", err)
			}
			if plan == nil {
				return nil, fmt.Errorf("plan not found for resource group %s", groupSID)
			}
			if plan.PlanType().IsForward() {
				uc.logger.Warnw("attempted to bind forward rule to forward plan resource group",
					"group_sid", groupSID,
					"plan_id", group.PlanID(),
					"plan_type", plan.PlanType().String())
				return nil, errors.NewValidationError(
					fmt.Sprintf("resource group %s belongs to a forward plan and cannot bind forward rules", groupSID))
			}

			groupIDs = append(groupIDs, group.ID())
		}
	}

	// Create external forward rule domain entity
	rule, err := forward.NewExternalForwardRule(
		cmd.UserID,
		nil, // subscriptionID is nil for admin-created rules
		targetNodeID,
		cmd.Name,
		cmd.ServerAddress,
		cmd.ListenPort,
		cmd.ExternalSource,
		cmd.ExternalRuleID,
		cmd.Remark,
		derefIntOrDefault(cmd.SortOrder, 0),
		groupIDs,
		id.NewForwardRuleID,
	)
	if err != nil {
		uc.logger.Errorw("failed to create external forward rule entity", "error", err)
		return nil, errors.NewValidationError(err.Error())
	}

	// Persist
	if err := uc.repo.Create(ctx, rule); err != nil {
		uc.logger.Errorw("failed to persist external forward rule", "error", err)
		return nil, fmt.Errorf("failed to save forward rule: %w", err)
	}

	result := &CreateForwardRuleResult{
		ID:           rule.SID(),
		AgentID:      0, // External rules don't have agents
		RuleType:     rule.RuleType().String(),
		Name:         rule.Name(),
		ListenPort:   rule.ListenPort(),
		TargetNodeID: rule.TargetNodeID(),
		IPVersion:    rule.IPVersion().String(),
		Protocol:     rule.Protocol().String(),
		Status:       rule.Status().String(),
		CreatedAt:    rule.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	uc.logger.Infow("external forward rule created successfully", "id", result.ID, "name", cmd.Name)

	// External rules don't need config sync notification (no agents)
	return result, nil
}
