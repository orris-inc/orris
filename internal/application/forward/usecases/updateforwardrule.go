package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
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
}

// UpdateForwardRuleUseCase handles forward rule updates.
type UpdateForwardRuleUseCase struct {
	repo          forward.Repository
	agentRepo     forward.AgentRepository
	nodeRepo      node.NodeRepository
	configSyncSvc ConfigSyncNotifier
	logger        logger.Interface
}

// NewUpdateForwardRuleUseCase creates a new UpdateForwardRuleUseCase.
func NewUpdateForwardRuleUseCase(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	configSyncSvc ConfigSyncNotifier,
	logger logger.Interface,
) *UpdateForwardRuleUseCase {
	return &UpdateForwardRuleUseCase{
		repo:          repo,
		agentRepo:     agentRepo,
		nodeRepo:      nodeRepo,
		configSyncSvc: configSyncSvc,
		logger:        logger,
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

	// Update chain port config (for direct_chain type rules)
	if cmd.ChainPortConfig != nil {
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
			chainPortConfig[chainAgent.ID()] = port
		}
		if err := rule.UpdateChainPortConfig(chainPortConfig); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	if cmd.ListenPort != nil {
		// Check if the new port is already in use by another rule
		if *cmd.ListenPort != rule.ListenPort() {
			exists, err := uc.repo.ExistsByListenPort(ctx, *cmd.ListenPort)
			if err != nil {
				uc.logger.Errorw("failed to check listen port", "port", *cmd.ListenPort, "error", err)
				return fmt.Errorf("failed to check listen port: %w", err)
			}
			if exists {
				return errors.NewConflictError("listen port is already in use", fmt.Sprintf("%d", *cmd.ListenPort))
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
				uc.logger.Infow("config sync notification skipped for entry agent", "rule_id", cmd.ShortID, "agent_id", newAgentID, "reason", err.Error())
			}

			// If entry agent changed, notify original agent to remove the rule
			if originalAgentID != newAgentID {
				if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), originalAgentID, cmd.ShortID, "deleted"); err != nil {
					uc.logger.Infow("config sync notification skipped for original entry agent", "rule_id", cmd.ShortID, "agent_id", originalAgentID, "reason", err.Error())
				}
			}

			// For entry type rules, notify exit agent
			if ruleType == "entry" {
				// Notify new exit agent
				if newExitAgentID > 0 {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), newExitAgentID, cmd.ShortID, "updated"); err != nil {
						uc.logger.Infow("config sync notification skipped for exit agent", "rule_id", cmd.ShortID, "agent_id", newExitAgentID, "reason", err.Error())
					}
				}

				// If exit agent changed, notify original exit agent to remove the rule
				if originalExitAgentID > 0 && originalExitAgentID != newExitAgentID {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), originalExitAgentID, cmd.ShortID, "deleted"); err != nil {
						uc.logger.Infow("config sync notification skipped for original exit agent", "rule_id", cmd.ShortID, "agent_id", originalExitAgentID, "reason", err.Error())
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
						uc.logger.Infow("config sync notification skipped for chain agent", "rule_id", cmd.ShortID, "agent_id", agentID, "reason", err.Error())
					}
					// Remove from original map (we'll notify remaining agents for deletion)
					delete(originalChainAgentMap, agentID)
				}

				// Notify removed chain agents
				for agentID := range originalChainAgentMap {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), agentID, cmd.ShortID, "deleted"); err != nil {
						uc.logger.Infow("config sync notification skipped for removed chain agent", "rule_id", cmd.ShortID, "agent_id", agentID, "reason", err.Error())
					}
				}
			}
		}()
	}

	return nil
}
