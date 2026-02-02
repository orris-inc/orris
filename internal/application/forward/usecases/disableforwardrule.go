package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DisableForwardRuleCommand represents the input for disabling a forward rule.
type DisableForwardRuleCommand struct {
	ShortID string // External API identifier
}

// DisableForwardRuleUseCase handles disabling a forward rule.
type DisableForwardRuleUseCase struct {
	repo          forward.Repository
	configSyncSvc ConfigSyncNotifier
	logger        logger.Interface
}

// NewDisableForwardRuleUseCase creates a new DisableForwardRuleUseCase.
func NewDisableForwardRuleUseCase(
	repo forward.Repository,
	configSyncSvc ConfigSyncNotifier,
	logger logger.Interface,
) *DisableForwardRuleUseCase {
	return &DisableForwardRuleUseCase{
		repo:          repo,
		configSyncSvc: configSyncSvc,
		logger:        logger,
	}
}

// Execute disables a forward rule.
func (uc *DisableForwardRuleUseCase) Execute(ctx context.Context, cmd DisableForwardRuleCommand) error {
	if cmd.ShortID == "" {
		return errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing disable forward rule use case", "short_id", cmd.ShortID)
	rule, err := uc.repo.GetBySID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward rule", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward rule: %w", err)
	}
	if rule == nil {
		return errors.NewNotFoundError("forward rule", cmd.ShortID)
	}

	if err := rule.Disable(); err != nil {
		return errors.NewValidationError(err.Error())
	}

	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to disable forward rule", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to disable forward rule: %w", err)
	}

	uc.logger.Infow("forward rule disabled successfully", "short_id", cmd.ShortID)

	// Notify config sync asynchronously (failure only logs warning, doesn't block)
	if uc.configSyncSvc != nil {
		// Notify entry agent
		go func() {
			if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), rule.AgentID(), cmd.ShortID, "removed"); err != nil {
				uc.logger.Debugw("config sync notification skipped for entry agent", "rule_id", cmd.ShortID, "agent_id", rule.AgentID(), "reason", err.Error())
			}
		}()

		// Notify additional agents based on rule type
		switch rule.RuleType().String() {
		case "entry":
			// Notify all exit agents for entry type rules (supports load balancing)
			for _, exitAgentID := range rule.GetAllExitAgentIDs() {
				go func(aid uint) {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), aid, cmd.ShortID, "removed"); err != nil {
						uc.logger.Debugw("config sync notification skipped for exit agent", "rule_id", cmd.ShortID, "agent_id", aid, "reason", err.Error())
					}
				}(exitAgentID)
			}
		case "chain", "direct_chain":
			// Notify all chain agents for chain and direct_chain type rules
			for _, agentID := range rule.ChainAgentIDs() {
				go func(aid uint) {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), aid, cmd.ShortID, "removed"); err != nil {
						uc.logger.Debugw("config sync notification skipped for chain agent", "rule_id", cmd.ShortID, "agent_id", aid, "reason", err.Error())
					}
				}(agentID)
			}
		}
	}

	return nil
}
