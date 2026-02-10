package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateForwardRuleCommand represents the input for updating a forward rule.
type UpdateForwardRuleCommand struct {
	ShortID             string // External API identifier
	UserID              *uint  // optional: for agent access validation (user endpoint only)
	Name                *string
	AgentShortID        *string           // entry agent ID (for all rule types)
	ExitAgentShortID    *string           // exit agent ID (for entry type, mutually exclusive with ExitAgents)
	ExitAgents          []ExitAgentInput  // exit agents for load balancing (for entry type, mutually exclusive with ExitAgentShortID), nil means no update
	LoadBalanceStrategy *string           // load balance strategy: failover, weighted (nil means no update)
	ChainAgentShortIDs  []string          // chain agent IDs (for chain type rules only), nil means no update
	ChainPortConfig     map[string]uint16 // chain port config (for direct_chain type rules only), nil means no update
	TunnelHops          *int              // number of tunnel hops for hybrid chain (nil means no update)
	TunnelType          *string           // tunnel type: ws or tls (nil means no update)
	ListenPort          *uint16
	TargetAddress       *string
	TargetPort          *uint16
	TargetNodeSID       *string // nil means no update, empty string means clear, non-empty means set to this node
	BindIP              *string // nil means no update, empty string means clear
	IPVersion           *string // auto, ipv4, ipv6
	Protocol            *string
	TrafficMultiplier   *float64 // nil means no update (0-1000000)
	SortOrder           *int     // nil means no update
	Remark              *string
	GroupSIDs           *[]string // nil means no update, empty slice means clear, non-nil means set
}

// UpdateForwardRuleUseCase handles forward rule updates.
type UpdateForwardRuleUseCase struct {
	repo              forward.Repository
	agentRepo         forward.AgentRepository
	nodeRepo          node.NodeRepository
	resourceGroupRepo resource.Repository
	planRepo          subscription.PlanRepository
	subscriptionRepo  subscription.SubscriptionRepository
	configSyncSvc     ConfigSyncNotifier
	logger            logger.Interface
}

// NewUpdateForwardRuleUseCase creates a new UpdateForwardRuleUseCase.
func NewUpdateForwardRuleUseCase(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	resourceGroupRepo resource.Repository,
	planRepo subscription.PlanRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	configSyncSvc ConfigSyncNotifier,
	logger logger.Interface,
) *UpdateForwardRuleUseCase {
	return &UpdateForwardRuleUseCase{
		repo:              repo,
		agentRepo:         agentRepo,
		nodeRepo:          nodeRepo,
		resourceGroupRepo: resourceGroupRepo,
		planRepo:          planRepo,
		subscriptionRepo:  subscriptionRepo,
		configSyncSvc:     configSyncSvc,
		logger:            logger,
	}
}

// Execute updates an existing forward rule.
func (uc *UpdateForwardRuleUseCase) Execute(ctx context.Context, cmd UpdateForwardRuleCommand) error {
	if cmd.ShortID == "" {
		return errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing update forward rule use case", "short_id", cmd.ShortID)
	rule, err := uc.repo.GetBySID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward rule", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward rule: %w", err)
	}
	if rule == nil {
		return errors.NewNotFoundError("forward rule", cmd.ShortID)
	}

	// Track original agent IDs for config sync notification
	originalAgentID := rule.AgentID()
	originalExitAgentIDs := rule.GetAllExitAgentIDs() // Includes both single and multiple exit agents
	originalChainAgentIDs := rule.ChainAgentIDs()

	// Collect all agent SIDs that need to be fetched (to avoid N+1 queries)
	allAgentSIDs := make([]string, 0)
	if cmd.AgentShortID != nil {
		allAgentSIDs = append(allAgentSIDs, *cmd.AgentShortID)
	}
	if cmd.ExitAgentShortID != nil {
		allAgentSIDs = append(allAgentSIDs, *cmd.ExitAgentShortID)
	}
	for _, input := range cmd.ExitAgents {
		allAgentSIDs = append(allAgentSIDs, input.AgentSID)
	}
	allAgentSIDs = append(allAgentSIDs, cmd.ChainAgentShortIDs...)
	for shortID := range cmd.ChainPortConfig {
		allAgentSIDs = append(allAgentSIDs, shortID)
	}

	// Batch fetch all agents to avoid N+1 queries
	var agentMap map[string]*forward.ForwardAgent
	if len(allAgentSIDs) > 0 {
		agents, err := uc.agentRepo.GetBySIDs(ctx, allAgentSIDs)
		if err != nil {
			uc.logger.Errorw("failed to batch get agents", "error", err)
			return fmt.Errorf("failed to get agents: %w", err)
		}
		agentMap = make(map[string]*forward.ForwardAgent, len(agents))
		for _, a := range agents {
			agentMap[a.SID()] = a
		}
	}

	// Update fields
	if cmd.Name != nil {
		if err := rule.UpdateName(*cmd.Name); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Update entry agent ID
	if cmd.AgentShortID != nil {
		agent, ok := agentMap[*cmd.AgentShortID]
		if !ok || agent == nil {
			return errors.NewNotFoundError("forward agent", *cmd.AgentShortID)
		}

		// Validate user access to agent (user endpoint only)
		if cmd.UserID != nil {
			if err := uc.validateUserAgentAccess(ctx, *cmd.UserID, agent); err != nil {
				return err
			}
		}

		// Validate current listen port against new agent's allowed port range
		// Use the new listenPort if provided, otherwise use rule's current listenPort
		portToCheck := rule.ListenPort()
		if cmd.ListenPort != nil {
			portToCheck = *cmd.ListenPort
		}
		if !agent.IsPortAllowed(portToCheck) {
			return errors.NewValidationError(
				fmt.Sprintf("listen port %d is not allowed for agent %s, allowed ranges: %s",
					portToCheck, *cmd.AgentShortID, agent.AllowedPortRange().String()))
		}

		// Check if the port is already in use on the new agent (when changing agents)
		// This check is needed even if the port number stays the same
		// Note: We exclude current rule since we're changing its agent
		if agent.ID() != rule.AgentID() {
			inUse, err := uc.repo.IsPortInUseByAgent(ctx, agent.ID(), portToCheck, rule.ID())
			if err != nil {
				uc.logger.Errorw("failed to check listen port on new agent", "agent_id", agent.ID(), "port", portToCheck, "error", err)
				return fmt.Errorf("failed to check listen port: %w", err)
			}
			if inUse {
				return errors.NewConflictError("listen port is already in use on target agent", fmt.Sprintf("%d", portToCheck))
			}
		}

		if err := rule.UpdateAgentID(agent.ID()); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Update exit agent configuration (single exitAgentID OR multiple exitAgents)
	// These are mutually exclusive - setting one clears the other
	if cmd.ExitAgentShortID != nil && len(cmd.ExitAgents) > 0 {
		return errors.NewValidationError("exit_agent_id and exit_agents are mutually exclusive")
	}

	// Update single exit agent ID (for entry type rules)
	if cmd.ExitAgentShortID != nil {
		exitAgent, ok := agentMap[*cmd.ExitAgentShortID]
		if !ok || exitAgent == nil {
			return errors.NewNotFoundError("exit forward agent", *cmd.ExitAgentShortID)
		}

		// Validate user access to exit agent (user endpoint only)
		if cmd.UserID != nil {
			if err := uc.validateUserAgentAccess(ctx, *cmd.UserID, exitAgent); err != nil {
				return err
			}
		}

		if err := rule.UpdateExitAgentID(exitAgent.ID()); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Update multiple exit agents for load balancing (for entry type rules)
	if len(cmd.ExitAgents) > 0 {
		exitAgents := make([]vo.AgentWeight, 0, len(cmd.ExitAgents))
		for _, input := range cmd.ExitAgents {
			exitAgent, ok := agentMap[input.AgentSID]
			if !ok || exitAgent == nil {
				return errors.NewNotFoundError("exit forward agent", input.AgentSID)
			}

			// Validate user access to exit agent (user endpoint only)
			if cmd.UserID != nil {
				if err := uc.validateUserAgentAccess(ctx, *cmd.UserID, exitAgent); err != nil {
					return err
				}
			}

			// Weight of 0 means backup agent, create directly
			aw, err := vo.NewAgentWeight(exitAgent.ID(), input.Weight)
			if err != nil {
				return errors.NewValidationError(fmt.Sprintf("invalid exit agent weight: %s", err.Error()))
			}
			exitAgents = append(exitAgents, aw)
		}

		if err := rule.UpdateExitAgents(exitAgents); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Update load balance strategy
	if cmd.LoadBalanceStrategy != nil {
		strategy := vo.ParseLoadBalanceStrategy(*cmd.LoadBalanceStrategy)
		if err := rule.UpdateLoadBalanceStrategy(strategy); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Update chain agent IDs (for chain type rules)
	if cmd.ChainAgentShortIDs != nil {
		chainAgentIDs := make([]uint, len(cmd.ChainAgentShortIDs))
		for i, shortID := range cmd.ChainAgentShortIDs {
			chainAgent, ok := agentMap[shortID]
			if !ok || chainAgent == nil {
				return errors.NewNotFoundError("chain forward agent", shortID)
			}

			// Validate user access to chain agent (user endpoint only)
			if cmd.UserID != nil {
				if err := uc.validateUserAgentAccess(ctx, *cmd.UserID, chainAgent); err != nil {
					return err
				}
			}

			chainAgentIDs[i] = chainAgent.ID()
		}
		if err := rule.UpdateChainAgentIDs(chainAgentIDs); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Update chain port config (for direct_chain and hybrid chain type rules)
	// Also validate that each port is within the corresponding agent's allowed port range
	// and check for port conflicts on each chain agent
	if cmd.ChainPortConfig != nil {
		oldChainPortConfig := rule.ChainPortConfig()
		chainPortConfig := make(map[uint]uint16, len(cmd.ChainPortConfig))
		for shortID, port := range cmd.ChainPortConfig {
			chainAgent, ok := agentMap[shortID]
			if !ok || chainAgent == nil {
				return errors.NewNotFoundError("chain forward agent in chain_port_config", shortID)
			}

			// Validate user access to chain agent (user endpoint only)
			if cmd.UserID != nil {
				if err := uc.validateUserAgentAccess(ctx, *cmd.UserID, chainAgent); err != nil {
					return err
				}
			}

			// Validate port against chain agent's allowed port range
			if !chainAgent.IsPortAllowed(port) {
				return errors.NewValidationError(
					fmt.Sprintf("listen port %d is not allowed for chain agent %s, allowed ranges: %s",
						port, shortID, chainAgent.AllowedPortRange().String()))
			}
			// Check if the port changed for this agent (skip conflict check if port unchanged)
			oldPort, hadOldPort := oldChainPortConfig[chainAgent.ID()]
			if !hadOldPort || oldPort != port {
				// Port is new or changed, check for conflicts (excluding current rule)
				inUse, err := uc.repo.IsPortInUseByAgent(ctx, chainAgent.ID(), port, rule.ID())
				if err != nil {
					uc.logger.Errorw("failed to check chain agent port", "chain_agent_id", chainAgent.ID(), "port", port, "error", err)
					return fmt.Errorf("failed to check chain agent port: %w", err)
				}
				if inUse {
					return errors.NewConflictError(
						fmt.Sprintf("listen port %d is already in use on chain agent %s", port, shortID),
						fmt.Sprintf("%d", port))
				}
			}
			chainPortConfig[chainAgent.ID()] = port
		}
		if err := rule.UpdateChainPortConfig(chainPortConfig); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Update tunnel hops for hybrid chain (must be after ChainPortConfig update)
	if cmd.TunnelHops != nil {
		if err := rule.UpdateTunnelHops(cmd.TunnelHops); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.ListenPort != nil {
		// Check if the new port is already in use by another rule on the same agent
		// Note: rule.AgentID() returns the updated agent ID if AgentShortID was provided earlier
		if *cmd.ListenPort != rule.ListenPort() {
			inUse, err := uc.repo.IsPortInUseByAgent(ctx, rule.AgentID(), *cmd.ListenPort, rule.ID())
			if err != nil {
				uc.logger.Errorw("failed to check listen port", "agent_id", rule.AgentID(), "port", *cmd.ListenPort, "error", err)
				return fmt.Errorf("failed to check listen port: %w", err)
			}
			if inUse {
				return errors.NewConflictError("listen port is already in use on this agent", fmt.Sprintf("%d", *cmd.ListenPort))
			}

			// Validate listen port against agent's allowed port range
			// Use the agent from updated AgentShortID if provided, otherwise get current agent
			var agentForPortCheck *forward.ForwardAgent
			if cmd.AgentShortID != nil {
				agentForPortCheck, err = uc.agentRepo.GetBySID(ctx, *cmd.AgentShortID)
				if err != nil {
					uc.logger.Errorw("failed to get agent for port validation", "agent_short_id", *cmd.AgentShortID, "error", err)
					return fmt.Errorf("failed to validate agent: %w", err)
				}
			} else {
				agentForPortCheck, err = uc.agentRepo.GetByID(ctx, rule.AgentID())
				if err != nil {
					uc.logger.Errorw("failed to get agent for port validation", "agent_id", rule.AgentID(), "error", err)
					return fmt.Errorf("failed to validate agent: %w", err)
				}
			}
			if agentForPortCheck != nil && !agentForPortCheck.IsPortAllowed(*cmd.ListenPort) {
				return errors.NewValidationError(
					fmt.Sprintf("listen port %d is not allowed for this agent, allowed ranges: %s",
						*cmd.ListenPort, agentForPortCheck.AllowedPortRange().String()))
			}
		}
		if err := rule.UpdateListenPort(*cmd.ListenPort); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Handle target updates
	// Priority: if TargetNodeSID is provided, use it; otherwise use TargetAddress/TargetPort
	if cmd.TargetNodeSID != nil {
		var targetNodeID *uint
		// If non-empty, resolve SID to internal ID
		if *cmd.TargetNodeSID != "" {
			targetNode, err := uc.nodeRepo.GetBySID(ctx, *cmd.TargetNodeSID)
			if err != nil {
				uc.logger.Errorw("failed to get target node", "node_sid", *cmd.TargetNodeSID, "error", err)
				return fmt.Errorf("failed to validate target node: %w", err)
			}
			if targetNode == nil {
				uc.logger.Warnw("target node not found", "node_sid", *cmd.TargetNodeSID)
				return errors.NewNotFoundError("node", *cmd.TargetNodeSID)
			}

			// Ownership constraint: prevent cross-ownership target node assignment
			// User rules can only target user-owned nodes belonging to the same user
			// System rules can only target system nodes (non-user-owned)
			if rule.IsUserOwned() {
				// User rule: target node must be owned by the same user
				if !targetNode.IsOwnedBy(*rule.UserID()) {
					uc.logger.Warnw("user rule cannot target node not owned by user",
						"rule_sid", cmd.ShortID,
						"rule_user_id", *rule.UserID(),
						"target_node_sid", *cmd.TargetNodeSID,
					)
					return errors.NewForbiddenError("user rules can only target nodes owned by the same user")
				}
			} else {
				// System rule: target node must not be user-owned
				if targetNode.IsUserOwned() {
					uc.logger.Warnw("system rule cannot target user-owned node",
						"rule_sid", cmd.ShortID,
						"target_node_sid", *cmd.TargetNodeSID,
					)
					return errors.NewForbiddenError("system rules cannot target user-owned nodes")
				}
			}

			nodeID := targetNode.ID()
			targetNodeID = &nodeID
		} else {
			// Attempting to clear targetNodeID - check if this is allowed for the rule type
			// External rules require targetNodeID (protocol is derived from target node)
			if rule.RuleType().IsExternal() {
				uc.logger.Warnw("cannot clear target_node_id for external rules",
					"rule_sid", cmd.ShortID,
				)
				return errors.NewValidationError("target_node_id is required for external rules (protocol info is derived from target node)")
			}
		}
		// Update targetNodeID (will clear targetAddress and targetPort if set, or clear nodeID if empty)
		if err := rule.UpdateTargetNodeID(targetNodeID); err != nil {
			return errors.NewValidationError(err.Error())
		}
	} else if cmd.TargetAddress != nil || cmd.TargetPort != nil {
		// External rules cannot use static target address/port
		if rule.RuleType().IsExternal() {
			uc.logger.Warnw("external rules cannot use static target address/port",
				"rule_sid", cmd.ShortID,
			)
			return errors.NewValidationError("external rules must use target_node_id, not static target address/port")
		}
		// Update static target address/port (will clear targetNodeID)
		targetAddr := rule.TargetAddress()
		targetPort := rule.TargetPort()
		if cmd.TargetAddress != nil {
			targetAddr = *cmd.TargetAddress
		}
		if cmd.TargetPort != nil {
			targetPort = *cmd.TargetPort
		}
		if err := rule.UpdateTarget(targetAddr, targetPort); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.IPVersion != nil {
		ipVersion := vo.IPVersion(*cmd.IPVersion)
		if err := rule.UpdateIPVersion(ipVersion); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.BindIP != nil {
		if err := rule.UpdateBindIP(*cmd.BindIP); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.Protocol != nil {
		protocol := vo.ForwardProtocol(*cmd.Protocol)
		if err := rule.UpdateProtocol(protocol); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.TunnelType != nil {
		tunnelType := vo.TunnelType(*cmd.TunnelType)
		if err := rule.UpdateTunnelType(tunnelType); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.TrafficMultiplier != nil {
		if err := rule.UpdateTrafficMultiplier(cmd.TrafficMultiplier); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.SortOrder != nil {
		if err := rule.UpdateSortOrder(*cmd.SortOrder); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.Remark != nil {
		if err := rule.UpdateRemark(*cmd.Remark); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Update group IDs if provided
	if cmd.GroupSIDs != nil {
		var groupIDs []uint
		if len(*cmd.GroupSIDs) > 0 {
			// Validate SID formats first
			for _, groupSID := range *cmd.GroupSIDs {
				if err := id.ValidatePrefix(groupSID, id.PrefixResourceGroup); err != nil {
					return errors.NewValidationError(fmt.Sprintf("invalid resource group ID format: %s", groupSID))
				}
			}

			// Batch fetch all groups to avoid N+1 queries
			groupMap, err := uc.resourceGroupRepo.GetBySIDs(ctx, *cmd.GroupSIDs)
			if err != nil {
				uc.logger.Errorw("failed to batch get resource groups", "error", err)
				return fmt.Errorf("failed to get resource groups: %w", err)
			}

			// Collect plan IDs for batch fetch
			planIDs := make([]uint, 0, len(*cmd.GroupSIDs))
			for _, groupSID := range *cmd.GroupSIDs {
				group, ok := groupMap[groupSID]
				if !ok || group == nil {
					return errors.NewNotFoundError("resource group", groupSID)
				}
				planIDs = append(planIDs, group.PlanID())
			}

			// Batch fetch all plans to avoid N+1 queries
			plans, err := uc.planRepo.GetByIDs(ctx, planIDs)
			if err != nil {
				uc.logger.Errorw("failed to batch get plans", "error", err)
				return fmt.Errorf("failed to get plans: %w", err)
			}
			planMap := make(map[uint]*subscription.Plan, len(plans))
			for _, p := range plans {
				planMap[p.ID()] = p
			}

			// Validate each group and build groupIDs
			groupIDs = make([]uint, 0, len(*cmd.GroupSIDs))
			for _, groupSID := range *cmd.GroupSIDs {
				group := groupMap[groupSID]
				plan, ok := planMap[group.PlanID()]
				if !ok || plan == nil {
					return fmt.Errorf("plan not found for resource group %s", groupSID)
				}
				if plan.PlanType().IsForward() {
					uc.logger.Warnw("attempted to bind forward rule to forward plan resource group",
						"group_sid", groupSID,
						"plan_id", group.PlanID(),
						"plan_type", plan.PlanType().String())
					return errors.NewValidationError(
						fmt.Sprintf("resource group %s belongs to a forward plan and cannot bind forward rules", groupSID))
				}
				groupIDs = append(groupIDs, group.ID())
			}
		}
		// Set group IDs (empty slice will clear all groups)
		rule.SetGroupIDs(groupIDs)
	}

	// Persist changes
	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to update forward rule", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to update forward rule: %w", err)
	}

	uc.logger.Infow("forward rule updated successfully", "short_id", cmd.ShortID)

	// Notify config sync asynchronously if rule is enabled (failure only logs warning, doesn't block)
	if rule.IsEnabled() && uc.configSyncSvc != nil {
		newAgentID := rule.AgentID()
		newExitAgentIDs := rule.GetAllExitAgentIDs() // Includes both single and multiple exit agents
		newChainAgentIDs := rule.ChainAgentIDs()
		ruleType := rule.RuleType().String()

		goroutine.SafeGo(uc.logger, "update-rule-notify-agents", func() {
			// Notify entry agent
			if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), newAgentID, cmd.ShortID, "updated"); err != nil {
				uc.logger.Debugw("config sync notification skipped for entry agent", "rule_id", cmd.ShortID, "agent_id", newAgentID, "reason", err.Error())
			}

			// If entry agent changed, notify original agent to remove the rule
			if originalAgentID != newAgentID {
				if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), originalAgentID, cmd.ShortID, "deleted"); err != nil {
					uc.logger.Debugw("config sync notification skipped for original entry agent", "rule_id", cmd.ShortID, "agent_id", originalAgentID, "reason", err.Error())
				}
			}

			// For entry type rules, notify exit agent(s)
			if ruleType == "entry" {
				// Create map of original exit agents for quick lookup
				originalExitAgentMap := make(map[uint]bool)
				for _, agentID := range originalExitAgentIDs {
					originalExitAgentMap[agentID] = true
				}

				// Notify new exit agents
				for _, agentID := range newExitAgentIDs {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), agentID, cmd.ShortID, "updated"); err != nil {
						uc.logger.Debugw("config sync notification skipped for exit agent", "rule_id", cmd.ShortID, "agent_id", agentID, "reason", err.Error())
					}
					// Remove from original map (we'll notify remaining agents for deletion)
					delete(originalExitAgentMap, agentID)
				}

				// Notify removed exit agents
				for agentID := range originalExitAgentMap {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), agentID, cmd.ShortID, "deleted"); err != nil {
						uc.logger.Debugw("config sync notification skipped for removed exit agent", "rule_id", cmd.ShortID, "agent_id", agentID, "reason", err.Error())
					}
				}
			}

			// For chain and direct_chain type rules, notify chain agents
			if ruleType == "chain" || ruleType == "direct_chain" {
				// Create map of original chain agents for quick lookup
				originalChainAgentMap := make(map[uint]bool)
				for _, agentID := range originalChainAgentIDs {
					originalChainAgentMap[agentID] = true
				}

				// Notify new chain agents
				for _, agentID := range newChainAgentIDs {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), agentID, cmd.ShortID, "updated"); err != nil {
						uc.logger.Debugw("config sync notification skipped for chain agent", "rule_id", cmd.ShortID, "agent_id", agentID, "reason", err.Error())
					}
					// Remove from original map (we'll notify remaining agents for deletion)
					delete(originalChainAgentMap, agentID)
				}

				// Notify removed chain agents
				for agentID := range originalChainAgentMap {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), agentID, cmd.ShortID, "deleted"); err != nil {
						uc.logger.Debugw("config sync notification skipped for removed chain agent", "rule_id", cmd.ShortID, "agent_id", agentID, "reason", err.Error())
					}
				}
			}
		})
	}

	return nil
}

// getAccessibleGroupIDs returns the resource group IDs that the user can access.
// Access path: User -> Subscription -> Plan(forward) -> ResourceGroup
func (uc *UpdateForwardRuleUseCase) getAccessibleGroupIDs(ctx context.Context, userID uint) ([]uint, error) {
	// Step 1: Get user's active subscriptions
	subscriptions, err := uc.subscriptionRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user subscriptions: %w", err)
	}
	if len(subscriptions) == 0 {
		return nil, nil
	}

	// Step 2: Collect plan IDs from subscriptions
	planIDs := make([]uint, 0, len(subscriptions))
	for _, sub := range subscriptions {
		planIDs = append(planIDs, sub.PlanID())
	}

	// Step 3: Get plans and filter forward type plans
	plans, err := uc.planRepo.GetByIDs(ctx, planIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get plans: %w", err)
	}

	forwardPlanIDs := make([]uint, 0, len(plans))
	for _, plan := range plans {
		if plan.PlanType().IsForward() {
			forwardPlanIDs = append(forwardPlanIDs, plan.ID())
		}
	}
	if len(forwardPlanIDs) == 0 {
		return nil, nil
	}

	// Step 4: Get active resource groups for these plans (batch query to avoid N+1)
	groupsByPlan, err := uc.resourceGroupRepo.GetByPlanIDs(ctx, forwardPlanIDs)
	if err != nil {
		uc.logger.Warnw("failed to batch get resource groups for plans", "error", err)
		return nil, nil
	}

	groupIDs := make([]uint, 0)
	for _, groups := range groupsByPlan {
		for _, group := range groups {
			if group.IsActive() {
				groupIDs = append(groupIDs, group.ID())
			}
		}
	}

	return groupIDs, nil
}

// validateUserAgentAccess checks if the user has access to the specified agent.
// Returns nil if access is allowed, or an error if access is denied.
// Access is granted if any of the agent's group IDs is in the user's accessible group IDs.
func (uc *UpdateForwardRuleUseCase) validateUserAgentAccess(ctx context.Context, userID uint, agent *forward.ForwardAgent) error {
	// Get user's accessible group IDs
	accessibleGroupIDs, err := uc.getAccessibleGroupIDs(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get accessible groups: %w", err)
	}

	// Check if any of agent's group IDs is in the accessible list
	agentGroupIDs := agent.GroupIDs()
	if len(agentGroupIDs) == 0 {
		// Agent has no group assigned, deny access for user endpoints
		uc.logger.Warnw("user attempted to access agent without group",
			"user_id", userID,
			"agent_sid", agent.SID())
		return errors.NewForbiddenError("agent is not accessible to user")
	}

	// Check if there's any intersection between agent's groups and user's accessible groups
	hasAccess := false
	for _, agentGroupID := range agentGroupIDs {
		if containsUint(accessibleGroupIDs, agentGroupID) {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		uc.logger.Warnw("user attempted to access unauthorized agent",
			"user_id", userID,
			"agent_sid", agent.SID(),
			"agent_group_ids", agentGroupIDs,
			"accessible_groups", accessibleGroupIDs)
		return errors.NewForbiddenError("user does not have access to this agent")
	}

	return nil
}

// containsUint checks if a uint slice contains a specific value.
func containsUint(slice []uint, val uint) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
