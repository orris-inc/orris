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
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// UpdateForwardRuleCommand represents the input for updating a forward rule.
type UpdateForwardRuleCommand struct {
	ShortID            string // External API identifier
	Name               *string
	AgentShortID       *string           // entry agent ID (for all rule types)
	ExitAgentShortID   *string           // exit agent ID (for entry type rules only)
	ChainAgentShortIDs []string          // chain agent IDs (for chain type rules only), nil means no update
	ChainPortConfig    map[string]uint16 // chain port config (for direct_chain type rules only), nil means no update
	TunnelHops         *int              // number of tunnel hops for hybrid chain (nil means no update)
	TunnelType         *string           // tunnel type: ws or tls (nil means no update)
	ListenPort         *uint16
	TargetAddress      *string
	TargetPort         *uint16
	TargetNodeSID      *string // nil means no update, empty string means clear, non-empty means set to this node
	BindIP             *string // nil means no update, empty string means clear
	IPVersion          *string // auto, ipv4, ipv6
	Protocol           *string
	TrafficMultiplier  *float64 // nil means no update (0-1000000)
	SortOrder          *int     // nil means no update
	Remark             *string
	GroupSIDs          *[]string // nil means no update, empty slice means clear, non-nil means set
}

// UpdateForwardRuleUseCase handles forward rule updates.
type UpdateForwardRuleUseCase struct {
	repo              forward.Repository
	agentRepo         forward.AgentRepository
	nodeRepo          node.NodeRepository
	resourceGroupRepo resource.Repository
	planRepo          subscription.PlanRepository
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
	configSyncSvc ConfigSyncNotifier,
	logger logger.Interface,
) *UpdateForwardRuleUseCase {
	return &UpdateForwardRuleUseCase{
		repo:              repo,
		agentRepo:         agentRepo,
		nodeRepo:          nodeRepo,
		resourceGroupRepo: resourceGroupRepo,
		planRepo:          planRepo,
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
	originalExitAgentID := rule.ExitAgentID()
	originalChainAgentIDs := rule.ChainAgentIDs()

	// Update fields
	if cmd.Name != nil {
		if err := rule.UpdateName(*cmd.Name); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Update entry agent ID
	if cmd.AgentShortID != nil {
		agent, err := uc.agentRepo.GetBySID(ctx, *cmd.AgentShortID)
		if err != nil {
			uc.logger.Errorw("failed to get agent", "agent_short_id", *cmd.AgentShortID, "error", err)
			return fmt.Errorf("failed to validate agent: %w", err)
		}
		if agent == nil {
			return errors.NewNotFoundError("forward agent", *cmd.AgentShortID)
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

	// Update exit agent ID (for entry type rules)
	if cmd.ExitAgentShortID != nil {
		exitAgent, err := uc.agentRepo.GetBySID(ctx, *cmd.ExitAgentShortID)
		if err != nil {
			uc.logger.Errorw("failed to get exit agent", "exit_agent_short_id", *cmd.ExitAgentShortID, "error", err)
			return fmt.Errorf("failed to validate exit agent: %w", err)
		}
		if exitAgent == nil {
			return errors.NewNotFoundError("exit forward agent", *cmd.ExitAgentShortID)
		}
		if err := rule.UpdateExitAgentID(exitAgent.ID()); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Update chain agent IDs (for chain type rules)
	if cmd.ChainAgentShortIDs != nil {
		chainAgentIDs := make([]uint, len(cmd.ChainAgentShortIDs))
		for i, shortID := range cmd.ChainAgentShortIDs {
			chainAgent, err := uc.agentRepo.GetBySID(ctx, shortID)
			if err != nil {
				uc.logger.Errorw("failed to get chain agent", "chain_agent_short_id", shortID, "error", err)
				return fmt.Errorf("failed to validate chain agent: %w", err)
			}
			if chainAgent == nil {
				return errors.NewNotFoundError("chain forward agent", shortID)
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
			chainAgent, err := uc.agentRepo.GetBySID(ctx, shortID)
			if err != nil {
				uc.logger.Errorw("failed to get chain agent for port config", "chain_agent_short_id", shortID, "error", err)
				return fmt.Errorf("failed to validate chain agent in chain_port_config: %w", err)
			}
			if chainAgent == nil {
				return errors.NewNotFoundError("chain forward agent in chain_port_config", shortID)
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
		}
		// Update targetNodeID (will clear targetAddress and targetPort if set, or clear nodeID if empty)
		if err := rule.UpdateTargetNodeID(targetNodeID); err != nil {
			return errors.NewValidationError(err.Error())
		}
	} else if cmd.TargetAddress != nil || cmd.TargetPort != nil {
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
			groupIDs = make([]uint, 0, len(*cmd.GroupSIDs))
			for _, groupSID := range *cmd.GroupSIDs {
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

				// Verify the plan type supports forward rules binding (node and hybrid only, not forward)
				plan, err := uc.planRepo.GetByID(ctx, group.PlanID())
				if err != nil {
					uc.logger.Errorw("failed to get plan for resource group", "plan_id", group.PlanID(), "error", err)
					return fmt.Errorf("failed to validate resource group plan: %w", err)
				}
				if plan == nil {
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
		newExitAgentID := rule.ExitAgentID()
		newChainAgentIDs := rule.ChainAgentIDs()
		ruleType := rule.RuleType().String()

		go func() {
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

			// For entry type rules, notify exit agent
			if ruleType == "entry" {
				// Notify new exit agent
				if newExitAgentID > 0 {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), newExitAgentID, cmd.ShortID, "updated"); err != nil {
						uc.logger.Debugw("config sync notification skipped for exit agent", "rule_id", cmd.ShortID, "agent_id", newExitAgentID, "reason", err.Error())
					}
				}

				// If exit agent changed, notify original exit agent to remove the rule
				if originalExitAgentID > 0 && originalExitAgentID != newExitAgentID {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), originalExitAgentID, cmd.ShortID, "deleted"); err != nil {
						uc.logger.Debugw("config sync notification skipped for original exit agent", "rule_id", cmd.ShortID, "agent_id", originalExitAgentID, "reason", err.Error())
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
		}()
	}

	return nil
}
