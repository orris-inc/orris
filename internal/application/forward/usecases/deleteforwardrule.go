package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DeleteForwardRuleCommand represents the input for deleting a forward rule.
type DeleteForwardRuleCommand struct {
	ShortID string // External API identifier
}

// DeleteForwardRuleUseCase handles forward rule deletion.
type DeleteForwardRuleUseCase struct {
	repo          forward.Repository
	trafficCache  cache.ForwardTrafficCache
	configSyncSvc ConfigSyncNotifier
	logger        logger.Interface
}

// NewDeleteForwardRuleUseCase creates a new DeleteForwardRuleUseCase.
func NewDeleteForwardRuleUseCase(
	repo forward.Repository,
	trafficCache cache.ForwardTrafficCache,
	configSyncSvc ConfigSyncNotifier,
	logger logger.Interface,
) *DeleteForwardRuleUseCase {
	return &DeleteForwardRuleUseCase{
		repo:          repo,
		trafficCache:  trafficCache,
		configSyncSvc: configSyncSvc,
		logger:        logger,
	}
}

// Execute deletes a forward rule.
func (uc *DeleteForwardRuleUseCase) Execute(ctx context.Context, cmd DeleteForwardRuleCommand) error {
	if cmd.ShortID == "" {
		return errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing delete forward rule use case", "short_id", cmd.ShortID)
	rule, err := uc.repo.GetBySID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward rule", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward rule: %w", err)
	}
	if rule == nil {
		return errors.NewNotFoundError("forward rule", cmd.ShortID)
	}

	// Store info for notification before deletion
	agentID := rule.AgentID()
	ruleShortID := rule.SID()
	wasEnabled := rule.IsEnabled()
	ruleType := rule.RuleType().String()
	exitAgentID := rule.ExitAgentID()
	chainAgentIDs := rule.ChainAgentIDs()

	// Store rule ID for cache cleanup
	ruleID := rule.ID()

	// Delete the rule using the internal ID
	if err := uc.repo.Delete(ctx, ruleID); err != nil {
		uc.logger.Errorw("failed to delete forward rule", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to delete forward rule: %w", err)
	}

	// Clean up traffic cache (non-blocking, log warning on failure)
	if uc.trafficCache != nil {
		if err := uc.trafficCache.CleanupRuleCache(ctx, ruleID); err != nil {
			uc.logger.Warnw("failed to cleanup traffic cache for deleted rule",
				"short_id", cmd.ShortID,
				"rule_id", ruleID,
				"error", err,
			)
		}
	}

	uc.logger.Infow("forward rule deleted successfully", "short_id", cmd.ShortID)

	// Notify config sync asynchronously if rule was enabled (failure only logs warning, doesn't block)
	if wasEnabled && uc.configSyncSvc != nil {
		// Notify entry agent
		go func() {
			if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), agentID, ruleShortID, "removed"); err != nil {
				uc.logger.Debugw("config sync notification skipped for entry agent", "rule_id", ruleShortID, "agent_id", agentID, "reason", err.Error())
			}
		}()

		// Notify additional agents based on rule type
		switch ruleType {
		case "entry":
			// Notify exit agent for entry type rules
			if exitAgentID != 0 {
				go func() {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), exitAgentID, ruleShortID, "removed"); err != nil {
						uc.logger.Debugw("config sync notification skipped for exit agent", "rule_id", ruleShortID, "agent_id", exitAgentID, "reason", err.Error())
					}
				}()
			}
		case "chain", "direct_chain":
			// Notify all chain agents for chain and direct_chain type rules
			for _, aid := range chainAgentIDs {
				go func(agentID uint) {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), agentID, ruleShortID, "removed"); err != nil {
						uc.logger.Debugw("config sync notification skipped for chain agent", "rule_id", ruleShortID, "agent_id", agentID, "reason", err.Error())
					}
				}(aid)
			}
		}
	}

	return nil
}
