package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// EnableForwardRuleCommand represents the input for enabling a forward rule.
type EnableForwardRuleCommand struct {
	ShortID string // External API identifier
}

// EnableForwardRuleUseCase handles enabling a forward rule.
type EnableForwardRuleUseCase struct {
	repo          forward.Repository
	configSyncSvc ConfigSyncNotifier
	logger        logger.Interface
}

// NewEnableForwardRuleUseCase creates a new EnableForwardRuleUseCase.
func NewEnableForwardRuleUseCase(
	repo forward.Repository,
	configSyncSvc ConfigSyncNotifier,
	logger logger.Interface,
) *EnableForwardRuleUseCase {
	return &EnableForwardRuleUseCase{
		repo:          repo,
		configSyncSvc: configSyncSvc,
		logger:        logger,
	}
}

// Execute enables a forward rule.
func (uc *EnableForwardRuleUseCase) Execute(ctx context.Context, cmd EnableForwardRuleCommand) error {
	if cmd.ShortID == "" {
		return errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing enable forward rule use case", "short_id", cmd.ShortID)
	rule, err := uc.repo.GetBySID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward rule", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward rule: %w", err)
	}
	if rule == nil {
		return errors.NewNotFoundError("forward rule", cmd.ShortID)
	}

	if err := rule.Enable(); err != nil {
		return errors.NewValidationError(err.Error())
	}

	if err := uc.repo.Update(ctx, rule); err != nil {
		uc.logger.Errorw("failed to enable forward rule", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to enable forward rule: %w", err)
	}

	uc.logger.Infow("forward rule enabled successfully", "short_id", cmd.ShortID)

	// Notify config sync asynchronously (failure only logs warning, doesn't block)
	if uc.configSyncSvc != nil {
		// Notify entry agent
		goroutine.SafeGo(uc.logger, "enable-rule-notify-entry-agent", func() {
			if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), rule.AgentID(), cmd.ShortID, "added"); err != nil {
				uc.logger.Debugw("config sync notification skipped for entry agent", "rule_id", cmd.ShortID, "agent_id", rule.AgentID(), "reason", err.Error())
			}
		})

		// Notify additional agents based on rule type
		switch rule.RuleType().String() {
		case "entry":
			// Notify all exit agents for entry type rules (supports load balancing)
			for _, exitAgentID := range rule.GetAllExitAgentIDs() {
				aid := exitAgentID
				goroutine.SafeGo(uc.logger, "enable-rule-notify-exit-agent", func() {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), aid, cmd.ShortID, "added"); err != nil {
						uc.logger.Debugw("config sync notification skipped for exit agent", "rule_id", cmd.ShortID, "agent_id", aid, "reason", err.Error())
					}
				})
			}
		case "chain", "direct_chain":
			// Notify all chain agents for chain and direct_chain type rules
			for _, chainAgentID := range rule.ChainAgentIDs() {
				aid := chainAgentID
				goroutine.SafeGo(uc.logger, "enable-rule-notify-chain-agent", func() {
					if err := uc.configSyncSvc.NotifyRuleChange(context.Background(), aid, cmd.ShortID, "added"); err != nil {
						uc.logger.Debugw("config sync notification skipped for chain agent", "rule_id", cmd.ShortID, "agent_id", aid, "reason", err.Error())
					}
				})
			}
		}
	}

	return nil
}
