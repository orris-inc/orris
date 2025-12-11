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

// UpdateForwardRuleCommand represents the input for updating a forward rule.
type UpdateForwardRuleCommand struct {
	ShortID            string // External API identifier
	Name               *string
	AgentShortID       *string  // entry agent ID (for all rule types)
	ExitAgentShortID   *string  // exit agent ID (for entry type rules only)
	ChainAgentShortIDs []string // chain agent IDs (for chain type rules only), nil means no update
	ListenPort         *uint16
	TargetAddress      *string
	TargetPort         *uint16
	TargetNodeShortID  *string // nil means no update, empty string means clear, non-empty means set to this node
	Protocol           *string
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
	rule, err := uc.repo.GetByShortID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward rule", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward rule: %w", err)
	}
	if rule == nil {
		return errors.NewNotFoundError("forward rule", cmd.ShortID)
	}

	// Track original agent ID for config sync notification
	originalAgentID := rule.AgentID()

	// Update fields
	if cmd.Name != nil {
		if err := rule.UpdateName(*cmd.Name); err != nil {
			return errors.NewValidationError(err.Error())
		}
	}

	// Update entry agent ID
	if cmd.AgentShortID != nil {
		agent, err := uc.agentRepo.GetByShortID(ctx, *cmd.AgentShortID)
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
		exitAgent, err := uc.agentRepo.GetByShortID(ctx, *cmd.ExitAgentShortID)
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
			chainAgent, err := uc.agentRepo.GetByShortID(ctx, shortID)
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
	// Priority: if TargetNodeShortID is provided, use it; otherwise use TargetAddress/TargetPort
	if cmd.TargetNodeShortID != nil {
		var targetNodeID *uint
		// If non-empty, resolve short ID to internal ID
		if *cmd.TargetNodeShortID != "" {
			targetNode, err := uc.nodeRepo.GetByShortID(ctx, *cmd.TargetNodeShortID)
			if err != nil {
				uc.logger.Errorw("failed to get target node", "node_short_id", *cmd.TargetNodeShortID, "error", err)
				return fmt.Errorf("failed to validate target node: %w", err)
			}
			if targetNode == nil {
				uc.logger.Warnw("target node not found", "node_short_id", *cmd.TargetNodeShortID)
				return errors.NewNotFoundError("node", *cmd.TargetNodeShortID)
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

	if cmd.Protocol != nil {
		protocol := vo.ForwardProtocol(*cmd.Protocol)
		if err := rule.UpdateProtocol(protocol); err != nil {
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
		go func() {
			// Notify new agent
			if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), newAgentID, cmd.ShortID, "updated"); err != nil {
				uc.logger.Warnw("failed to notify config sync for new agent", "rule_id", cmd.ShortID, "agent_id", newAgentID, "error", err)
			}
			// If agent changed, also notify original agent to remove the rule
			if originalAgentID != newAgentID {
				if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), originalAgentID, cmd.ShortID, "deleted"); err != nil {
					uc.logger.Warnw("failed to notify config sync for original agent", "rule_id", cmd.ShortID, "agent_id", originalAgentID, "error", err)
				}
			}
		}()
	}

	return nil
}
