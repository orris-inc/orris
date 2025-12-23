package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// RuleOrder represents a single rule's sort order.
type RuleOrder struct {
	RuleSID   string
	SortOrder int
}

// ReorderForwardRulesCommand represents the input for reordering forward rules.
type ReorderForwardRulesCommand struct {
	RuleOrders []RuleOrder
	UserID     *uint // optional: if set, only reorder rules owned by this user
}

// ReorderForwardRulesUseCase handles batch reordering of forward rules.
type ReorderForwardRulesUseCase struct {
	repo   forward.Repository
	txMgr  *db.TransactionManager
	logger logger.Interface
}

// NewReorderForwardRulesUseCase creates a new ReorderForwardRulesUseCase.
func NewReorderForwardRulesUseCase(
	repo forward.Repository,
	txMgr *db.TransactionManager,
	logger logger.Interface,
) *ReorderForwardRulesUseCase {
	return &ReorderForwardRulesUseCase{
		repo:   repo,
		txMgr:  txMgr,
		logger: logger,
	}
}

// Execute reorders multiple forward rules.
func (uc *ReorderForwardRulesUseCase) Execute(ctx context.Context, cmd ReorderForwardRulesCommand) error {
	if len(cmd.RuleOrders) == 0 {
		return errors.NewValidationError("rule_orders is required")
	}

	uc.logger.Infow("executing reorder forward rules use case", "count", len(cmd.RuleOrders))

	// Collect rule SIDs for batch query
	sids := make([]string, 0, len(cmd.RuleOrders))
	for _, order := range cmd.RuleOrders {
		if order.RuleSID == "" {
			return errors.NewValidationError("rule_id is required for each item")
		}
		sids = append(sids, order.RuleSID)
	}

	// Run in transaction to ensure atomicity
	return uc.txMgr.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Batch query all rules
		rulesMap, err := uc.repo.GetBySIDs(txCtx, sids)
		if err != nil {
			uc.logger.Errorw("failed to get rules by SIDs", "count", len(sids), "error", err)
			return fmt.Errorf("failed to get rules: %w", err)
		}

		// Build sort order map and validate ownership
		ruleOrders := make(map[uint]int, len(cmd.RuleOrders))
		for _, order := range cmd.RuleOrders {
			rule, exists := rulesMap[order.RuleSID]
			if !exists {
				return errors.NewNotFoundError("forward rule", order.RuleSID)
			}

			// If user ID is specified, verify ownership
			if cmd.UserID != nil {
				ruleUserID := rule.UserID()
				if ruleUserID == nil || *ruleUserID != *cmd.UserID {
					uc.logger.Warnw("user attempted to reorder rule they don't own",
						"user_id", *cmd.UserID,
						"rule_sid", order.RuleSID,
						"rule_owner", ruleUserID,
					)
					return errors.NewForbiddenError("cannot reorder this rule")
				}
			}

			ruleOrders[rule.ID()] = order.SortOrder
		}

		// Update sort orders in batch
		if err := uc.repo.UpdateSortOrders(txCtx, ruleOrders); err != nil {
			uc.logger.Errorw("failed to update sort orders", "error", err)
			return fmt.Errorf("failed to update sort orders: %w", err)
		}

		uc.logger.Infow("forward rules reordered successfully", "count", len(cmd.RuleOrders))
		return nil
	})
}
