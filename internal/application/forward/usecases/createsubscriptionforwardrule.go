package usecases

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// CreateSubscriptionForwardRuleCommand represents the input for creating a subscription-bound forward rule.
type CreateSubscriptionForwardRuleCommand struct {
	UserID             uint              // user ID (owner of the subscription)
	SubscriptionID     uint              // subscription ID to bind the rule to
	AgentShortID       string            // Stripe-style short ID (e.g., "fa_xK9mP2vL3nQ")
	RuleType           string            // direct, entry, chain, direct_chain
	ExitAgentShortID   string            // required for entry type
	ChainAgentShortIDs []string          // required for chain type
	ChainPortConfig    map[string]uint16 // required for direct_chain type
	TunnelHops         *int              // number of hops using tunnel (for chain type)
	TunnelType         string            // tunnel type: ws or tls
	Name               string
	ListenPort         uint16 // listen port (0 = auto-assign)
	TargetAddress      string
	TargetPort         uint16
	TargetNodeSID      string // optional target node
	BindIP             string
	IPVersion          string
	Protocol           string
	TrafficMultiplier  *float64
	SortOrder          *int
	Remark             string
	RuleLimit          int // rule limit for the subscription (0 = unlimited, used for race condition check)
}

// CreateSubscriptionForwardRuleResult represents the output of creating a subscription-bound forward rule.
type CreateSubscriptionForwardRuleResult struct {
	ID             string `json:"id"`
	SubscriptionID uint   `json:"subscription_id"`
	AgentID        uint   `json:"agent_id"`
	RuleType       string `json:"rule_type"`
	ExitAgentID    uint   `json:"exit_agent_id,omitempty"`
	Name           string `json:"name"`
	ListenPort     uint16 `json:"listen_port"`
	TargetAddress  string `json:"target_address,omitempty"`
	TargetPort     uint16 `json:"target_port,omitempty"`
	TargetNodeID   *uint  `json:"target_node_id,omitempty"`
	IPVersion      string `json:"ip_version"`
	Protocol       string `json:"protocol"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

// CreateSubscriptionForwardRuleUseCase handles subscription-bound forward rule creation.
type CreateSubscriptionForwardRuleUseCase struct {
	repo          forward.Repository
	agentRepo     forward.AgentRepository
	nodeRepo      node.NodeRepository
	configSyncSvc ConfigSyncNotifier
	logger        logger.Interface
}

// NewCreateSubscriptionForwardRuleUseCase creates a new CreateSubscriptionForwardRuleUseCase.
func NewCreateSubscriptionForwardRuleUseCase(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	configSyncSvc ConfigSyncNotifier,
	logger logger.Interface,
) *CreateSubscriptionForwardRuleUseCase {
	return &CreateSubscriptionForwardRuleUseCase{
		repo:          repo,
		agentRepo:     agentRepo,
		nodeRepo:      nodeRepo,
		configSyncSvc: configSyncSvc,
		logger:        logger,
	}
}

// Execute creates a new forward rule bound to a specific subscription.
func (uc *CreateSubscriptionForwardRuleUseCase) Execute(ctx context.Context, cmd CreateSubscriptionForwardRuleCommand) (*CreateSubscriptionForwardRuleResult, error) {
	uc.logger.Infow("executing create subscription forward rule use case",
		"user_id", cmd.UserID,
		"subscription_id", cmd.SubscriptionID,
		"name", cmd.Name,
		"listen_port", cmd.ListenPort,
	)

	// Validate required fields
	if cmd.UserID == 0 {
		return nil, errors.NewValidationError("user_id is required")
	}
	if cmd.SubscriptionID == 0 {
		return nil, errors.NewValidationError("subscription_id is required")
	}

	// Resolve AgentShortID to internal ID
	if cmd.AgentShortID == "" {
		return nil, errors.NewValidationError("agent_id is required")
	}
	agent, err := uc.agentRepo.GetBySID(ctx, cmd.AgentShortID)
	if err != nil {
		uc.logger.Errorw("failed to get agent", "agent_short_id", cmd.AgentShortID, "subscription_id", cmd.SubscriptionID, "error", err)
		return nil, fmt.Errorf("failed to validate agent: %w", err)
	}
	if agent == nil {
		return nil, errors.NewNotFoundError("forward agent", cmd.AgentShortID)
	}
	agentID := agent.ID()

	// Record whether port should be auto-assigned (for retry logic on conflict)
	isAutoAssignPort := cmd.ListenPort == 0

	// Auto-assign listen port if not specified
	if isAutoAssignPort {
		port, err := uc.assignAvailablePort(ctx, agent)
		if err != nil {
			uc.logger.Errorw("failed to auto-assign listen port", "agent_id", agentID, "subscription_id", cmd.SubscriptionID, "error", err)
			return nil, err
		}
		cmd.ListenPort = port
		uc.logger.Infow("auto-assigned listen port", "port", port, "agent_id", agentID, "subscription_id", cmd.SubscriptionID)
	}

	// Validate listen port against agent's allowed port range
	if !agent.IsPortAllowed(cmd.ListenPort) {
		return nil, errors.NewValidationError(
			fmt.Sprintf("listen port %d is not allowed for this agent, allowed ranges: %s",
				cmd.ListenPort, agent.AllowedPortRange().String()))
	}

	// Store the context for port conflict retry
	portRetryCtx := &portRetryContext{
		agent:            agent,
		isAutoAssignPort: isAutoAssignPort,
	}

	// Resolve ExitAgentShortID to internal ID (if provided)
	var exitAgentID uint
	if cmd.ExitAgentShortID != "" {
		exitAgent, err := uc.agentRepo.GetBySID(ctx, cmd.ExitAgentShortID)
		if err != nil {
			uc.logger.Errorw("failed to get exit agent", "exit_agent_short_id", cmd.ExitAgentShortID, "subscription_id", cmd.SubscriptionID, "error", err)
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
				uc.logger.Errorw("failed to get chain agent", "chain_agent_short_id", shortID, "subscription_id", cmd.SubscriptionID, "error", err)
				return nil, fmt.Errorf("failed to validate chain agent: %w", err)
			}
			if chainAgent == nil {
				return nil, errors.NewNotFoundError("chain forward agent", shortID)
			}
			chainAgentIDs[i] = chainAgent.ID()
		}
	}

	// Resolve ChainPortConfig short IDs to internal IDs and validate ports
	var chainPortConfig map[uint]uint16
	if len(cmd.ChainPortConfig) > 0 {
		chainPortConfig = make(map[uint]uint16, len(cmd.ChainPortConfig))
		for shortID, port := range cmd.ChainPortConfig {
			chainAgent, err := uc.agentRepo.GetBySID(ctx, shortID)
			if err != nil {
				uc.logger.Errorw("failed to get chain agent for port config", "chain_agent_short_id", shortID, "subscription_id", cmd.SubscriptionID, "error", err)
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
			// Check if the port is already in use on this chain agent
			inUse, err := uc.repo.IsPortInUseByAgent(ctx, chainAgent.ID(), port, 0)
			if err != nil {
				uc.logger.Errorw("failed to check chain agent port", "chain_agent_id", chainAgent.ID(), "port", port, "subscription_id", cmd.SubscriptionID, "error", err)
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
			uc.logger.Errorw("failed to get target node", "target_node_sid", cmd.TargetNodeSID, "subscription_id", cmd.SubscriptionID, "error", err)
			return nil, fmt.Errorf("failed to validate target node: %w", err)
		}
		if targetNode == nil {
			return nil, errors.NewNotFoundError("target node", cmd.TargetNodeSID)
		}

		// Verify user owns the target node
		if !targetNode.IsOwnedBy(cmd.UserID) {
			uc.logger.Warnw("user attempted to use node they don't own as target",
				"user_id", cmd.UserID,
				"subscription_id", cmd.SubscriptionID,
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
		uc.logger.Errorw("invalid create subscription forward rule command", "subscription_id", cmd.SubscriptionID, "error", err)
		return nil, err
	}

	// Create and persist rule with port conflict retry
	return uc.createRuleWithRetry(ctx, cmd, agentID, exitAgentID, chainAgentIDs, chainPortConfig, targetNodeID, portRetryCtx)
}

// portRetryContext holds context for port conflict retry logic.
type portRetryContext struct {
	agent            *forward.ForwardAgent
	isAutoAssignPort bool
}

// maxPortConflictRetries is the maximum number of retries when auto-assigned port conflicts.
const maxPortConflictRetries = 3

// createRuleWithRetry creates the rule with automatic retry on port conflict for auto-assigned ports.
// It also performs a secondary rule limit check to mitigate race conditions.
func (uc *CreateSubscriptionForwardRuleUseCase) createRuleWithRetry(
	ctx context.Context,
	cmd CreateSubscriptionForwardRuleCommand,
	agentID, exitAgentID uint,
	chainAgentIDs []uint,
	chainPortConfig map[uint]uint16,
	targetNodeID *uint,
	retryCtx *portRetryContext,
) (*CreateSubscriptionForwardRuleResult, error) {
	// Secondary rule limit check to mitigate race conditions (defense in depth)
	// This check is performed closer to the actual creation to reduce the race window
	if cmd.RuleLimit > 0 {
		currentCount, err := uc.repo.CountBySubscriptionID(ctx, cmd.SubscriptionID)
		if err != nil {
			uc.logger.Errorw("failed to count subscription rules for limit check",
				"subscription_id", cmd.SubscriptionID,
				"error", err,
			)
			// Continue anyway - the middleware already checked, this is just defense in depth
		} else if currentCount >= int64(cmd.RuleLimit) {
			uc.logger.Warnw("subscription rule limit exceeded (secondary check)",
				"subscription_id", cmd.SubscriptionID,
				"current_count", currentCount,
				"rule_limit", cmd.RuleLimit,
			)
			return nil, errors.NewValidationError(
				fmt.Sprintf("forward rule limit exceeded: %d/%d rules", currentCount, cmd.RuleLimit))
		}
	}

	protocol := vo.ForwardProtocol(cmd.Protocol)
	ruleType := vo.ForwardRuleType(cmd.RuleType)
	ipVersion := vo.IPVersion(cmd.IPVersion)
	tunnelType := vo.TunnelType(cmd.TunnelType)
	userIDPtr := &cmd.UserID
	subscriptionIDPtr := &cmd.SubscriptionID

	var lastErr error
	maxRetries := 1
	if retryCtx.isAutoAssignPort {
		maxRetries = maxPortConflictRetries
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Re-assign port on retry (only for auto-assign mode)
			newPort, err := uc.assignAvailablePort(ctx, retryCtx.agent)
			if err != nil {
				uc.logger.Errorw("failed to re-assign port on retry",
					"agent_id", agentID,
					"subscription_id", cmd.SubscriptionID,
					"attempt", attempt,
					"error", err,
				)
				return nil, err
			}
			cmd.ListenPort = newPort
			uc.logger.Infow("re-assigned listen port on retry",
				"port", newPort,
				"agent_id", agentID,
				"subscription_id", cmd.SubscriptionID,
				"attempt", attempt,
			)
		}

		// Create domain entity with subscription_id
		rule, err := forward.NewForwardRule(
			agentID,
			userIDPtr,
			subscriptionIDPtr, // Bind to subscription
			ruleType,
			exitAgentID,
			nil,                           // exitAgents: subscription rules don't support load balancing yet
			vo.DefaultLoadBalanceStrategy, // loadBalanceStrategy
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
			uc.logger.Errorw("failed to create forward rule entity", "subscription_id", cmd.SubscriptionID, "error", err)
			return nil, errors.NewValidationError(err.Error())
		}

		// Persist - database unique constraint is the final protection against race conditions
		if err := uc.repo.Create(ctx, rule); err != nil {
			// Check if this is a port conflict error
			if errors.IsConflictError(err) && retryCtx.isAutoAssignPort && attempt < maxRetries-1 {
				uc.logger.Warnw("port conflict on auto-assigned port, retrying",
					"port", cmd.ListenPort,
					"agent_id", agentID,
					"subscription_id", cmd.SubscriptionID,
					"attempt", attempt,
				)
				lastErr = err
				continue
			}
			uc.logger.Errorw("failed to persist subscription forward rule",
				"subscription_id", cmd.SubscriptionID,
				"port", cmd.ListenPort,
				"error", err,
			)
			return nil, err
		}

		result := &CreateSubscriptionForwardRuleResult{
			ID:             rule.SID(),
			SubscriptionID: cmd.SubscriptionID,
			AgentID:        rule.AgentID(),
			RuleType:       rule.RuleType().String(),
			ExitAgentID:    rule.ExitAgentID(),
			Name:           rule.Name(),
			ListenPort:     rule.ListenPort(),
			TargetAddress:  rule.TargetAddress(),
			TargetPort:     rule.TargetPort(),
			TargetNodeID:   rule.TargetNodeID(),
			IPVersion:      rule.IPVersion().String(),
			Protocol:       rule.Protocol().String(),
			Status:         rule.Status().String(),
			CreatedAt:      rule.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		}

		uc.logger.Infow("subscription forward rule created successfully",
			"subscription_id", cmd.SubscriptionID,
			"id", result.ID,
			"name", cmd.Name,
		)

		// Notify config sync asynchronously if rule is enabled
		if rule.IsEnabled() && uc.configSyncSvc != nil {
			// Notify entry agent
			goroutine.SafeGo(uc.logger, "create-sub-rule-notify-entry-agent", func() {
				if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), rule.AgentID(), rule.SID(), "added"); err != nil {
					uc.logger.Debugw("config sync notification skipped for entry agent", "rule_id", rule.SID(), "subscription_id", cmd.SubscriptionID, "agent_id", rule.AgentID(), "reason", err.Error())
				}
			})

			// Notify additional agents based on rule type
			switch rule.RuleType().String() {
			case "entry":
				// Notify all exit agents for entry type rules (supports load balancing)
				for _, exitAgentID := range rule.GetAllExitAgentIDs() {
					aid := exitAgentID
					goroutine.SafeGo(uc.logger, "create-sub-rule-notify-exit-agent", func() {
						if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), aid, rule.SID(), "added"); err != nil {
							uc.logger.Debugw("config sync notification skipped for exit agent", "rule_id", rule.SID(), "subscription_id", cmd.SubscriptionID, "agent_id", aid, "reason", err.Error())
						}
					})
				}
			case "chain", "direct_chain":
				// Notify all chain agents for chain and direct_chain type rules
				for _, chainAgentID := range rule.ChainAgentIDs() {
					aid := chainAgentID
					goroutine.SafeGo(uc.logger, "create-sub-rule-notify-chain-agent", func() {
						if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), aid, rule.SID(), "added"); err != nil {
							uc.logger.Debugw("config sync notification skipped for chain agent", "rule_id", rule.SID(), "subscription_id", cmd.SubscriptionID, "agent_id", aid, "reason", err.Error())
						}
					})
				}
			}
		}

		return result, nil
	}

	// All retries exhausted
	uc.logger.Errorw("failed to create rule after max retries",
		"subscription_id", cmd.SubscriptionID,
		"max_retries", maxRetries,
		"last_error", lastErr,
	)
	return nil, errors.NewConflictError("failed to allocate available port, please try again")
}

func (uc *CreateSubscriptionForwardRuleUseCase) validateCommand(_ context.Context, cmd CreateSubscriptionForwardRuleCommand, targetNodeID *uint, chainAgentIDs []uint, chainPortConfig map[uint]uint16) error {
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
		hasTarget := cmd.TargetAddress != "" && cmd.TargetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return errors.NewValidationError("either target_address+target_port or target_node_id is required for chain forward")
		}
		if hasTarget && hasTargetNode {
			return errors.NewValidationError("target_address+target_port and target_node_id are mutually exclusive for chain forward")
		}
	case vo.ForwardRuleTypeDirectChain:
		if len(cmd.ChainAgentShortIDs) == 0 {
			return errors.NewValidationError("chain_agent_ids is required for direct_chain forward (at least 1 intermediate agent)")
		}
		if len(cmd.ChainAgentShortIDs) > 10 {
			return errors.NewValidationError("direct_chain forward supports maximum 10 intermediate agents")
		}
		if len(cmd.ChainPortConfig) == 0 {
			return errors.NewValidationError("chain_port_config is required for direct_chain forward")
		}
		for _, agentID := range chainAgentIDs {
			if _, exists := chainPortConfig[agentID]; !exists {
				return errors.NewValidationError(fmt.Sprintf("chain_port_config must provide listen port for all chain agents (missing agent_id: %d)", agentID))
			}
		}
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

// Default port range for auto-assignment
const (
	subscriptionDefaultPortRangeStart = 10000
	subscriptionDefaultPortRangeEnd   = 60000
	subscriptionMaxPortAssignAttempts = 100
)

// assignAvailablePort finds an available port for the given agent.
func (uc *CreateSubscriptionForwardRuleUseCase) assignAvailablePort(ctx context.Context, agent *forward.ForwardAgent) (uint16, error) {
	portRange := agent.AllowedPortRange()
	agentID := agent.ID()

	for i := 0; i < subscriptionMaxPortAssignAttempts; i++ {
		var port uint16
		if portRange != nil && !portRange.IsEmpty() {
			port = portRange.RandomPort()
		} else {
			// Use default range: port range is 10000-65000, so result always fits in uint16
			randomOffset := rand.Intn(subscriptionDefaultPortRangeEnd - subscriptionDefaultPortRangeStart + 1)
			portVal := subscriptionDefaultPortRangeStart + randomOffset
			// #nosec G115 -- portVal is bounded by subscriptionDefaultPortRangeStart(10000) to subscriptionDefaultPortRangeEnd(65000)
			port = uint16(portVal)
		}

		if !agent.IsPortAllowed(port) {
			continue
		}

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
