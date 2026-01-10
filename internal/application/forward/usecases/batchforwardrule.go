// Package usecases provides application layer use cases for forward operations.
package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// BatchForwardRuleUseCase handles batch operations for forward rules.
type BatchForwardRuleUseCase struct {
	repo              forward.Repository
	createRuleUC      *CreateForwardRuleUseCase
	createUserRuleUC  *CreateUserForwardRuleUseCase
	deleteRuleUC      *DeleteForwardRuleUseCase
	enableRuleUC      *EnableForwardRuleUseCase
	disableRuleUC     *DisableForwardRuleUseCase
	updateRuleUC      *UpdateForwardRuleUseCase
	txMgr             *db.TransactionManager
	logger            logger.Interface
}

// NewBatchForwardRuleUseCase creates a new BatchForwardRuleUseCase.
func NewBatchForwardRuleUseCase(
	repo forward.Repository,
	createRuleUC *CreateForwardRuleUseCase,
	createUserRuleUC *CreateUserForwardRuleUseCase,
	deleteRuleUC *DeleteForwardRuleUseCase,
	enableRuleUC *EnableForwardRuleUseCase,
	disableRuleUC *DisableForwardRuleUseCase,
	updateRuleUC *UpdateForwardRuleUseCase,
	txMgr *db.TransactionManager,
	logger logger.Interface,
) *BatchForwardRuleUseCase {
	return &BatchForwardRuleUseCase{
		repo:             repo,
		createRuleUC:     createRuleUC,
		createUserRuleUC: createUserRuleUC,
		deleteRuleUC:     deleteRuleUC,
		enableRuleUC:     enableRuleUC,
		disableRuleUC:    disableRuleUC,
		updateRuleUC:     updateRuleUC,
		txMgr:            txMgr,
		logger:           logger,
	}
}

// BatchCreateCommand represents the input for batch creating forward rules (admin).
type BatchCreateCommand struct {
	Rules      []CreateForwardRuleCommand
	CmdIndices []int // optional: original indices for each command (used when handler pre-validates)
}

// BatchCreateUserCommand represents the input for batch creating forward rules (user).
type BatchCreateUserCommand struct {
	UserID     uint
	Rules      []CreateUserForwardRuleCommand
	CmdIndices []int // optional: original indices for each command (used when handler pre-validates)
}

// BatchDeleteCommand represents the input for batch deleting forward rules.
type BatchDeleteCommand struct {
	RuleSIDs []string
	UserID   *uint // optional: for ownership validation (user endpoint only)
}

// BatchToggleStatusCommand represents the input for batch enabling/disabling rules.
type BatchToggleStatusCommand struct {
	RuleSIDs []string
	Enable   bool  // true = enable, false = disable
	UserID   *uint // optional: for ownership validation (user endpoint only)
}

// BatchUpdateCommand represents the input for batch updating forward rules.
type BatchUpdateCommand struct {
	Updates []BatchUpdateItem
	UserID  *uint // optional: for ownership validation (user endpoint only)
}

// BatchUpdateItem represents a single rule update.
// Supports: name, remark, sort_order, agent_id (entry), exit_agent_id (exit).
// Note: chain_agent_ids is NOT supported in batch update - use single rule update instead.
type BatchUpdateItem struct {
	RuleSID          string
	Name             *string
	Remark           *string
	SortOrder        *int
	AgentShortID     *string // entry agent ID
	ExitAgentShortID *string // exit agent ID (for entry type rules)
}

// validateBatchSize validates the batch size is within limits.
func validateBatchSize(size int, itemName string) error {
	if size == 0 {
		return errors.NewValidationError(itemName + " is required")
	}
	if size > dto.BatchLimit {
		return errors.NewValidationError(
			fmt.Sprintf("batch size exceeds limit of %d", dto.BatchLimit))
	}
	return nil
}

// batchValidationResult holds the result of batch rule validation.
type batchValidationResult struct {
	validRules map[string]*forward.ForwardRule
	result     *dto.BatchOperationResult
}

// validateBatchRuleSIDs validates a batch of rule SIDs and returns valid rules.
// It performs SID format validation, existence check, and optional ownership validation.
// Duplicate SIDs are automatically deduplicated.
func (uc *BatchForwardRuleUseCase) validateBatchRuleSIDs(
	ctx context.Context,
	sids []string,
	userID *uint,
	actionName string,
) (*batchValidationResult, error) {
	// Deduplicate SIDs to prevent duplicate operations in transaction
	seen := make(map[string]bool, len(sids))
	uniqueSIDs := make([]string, 0, len(sids))
	for _, sid := range sids {
		if !seen[sid] {
			seen[sid] = true
			uniqueSIDs = append(uniqueSIDs, sid)
		}
	}

	// Batch fetch all rules for validation
	rulesMap, err := uc.repo.GetBySIDs(ctx, uniqueSIDs)
	if err != nil {
		uc.logger.Errorw("failed to batch get rules", "error", err)
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	validRules := make(map[string]*forward.ForwardRule)
	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	for _, sid := range uniqueSIDs {
		// Validate SID format
		if err := id.ValidatePrefix(sid, id.PrefixForwardRule); err != nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     sid,
				Reason: "invalid forward rule ID format",
			})
			continue
		}

		rule, exists := rulesMap[sid]
		if !exists || rule == nil {
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     sid,
				Reason: "forward rule not found",
			})
			continue
		}

		// Ownership validation for user operations
		if userID != nil {
			ruleUserID := rule.UserID()
			if ruleUserID == nil || *ruleUserID != *userID {
				result.Failed = append(result.Failed, dto.BatchOperationErr{
					ID:     sid,
					Reason: "not authorized to " + actionName + " this rule",
				})
				continue
			}
		}

		validRules[sid] = rule
	}

	return &batchValidationResult{
		validRules: validRules,
		result:     result,
	}, nil
}

// BatchCreate creates multiple forward rules (admin endpoint).
// Uses partial failure mode: continues on individual errors.
func (uc *BatchForwardRuleUseCase) BatchCreate(
	ctx context.Context,
	cmd BatchCreateCommand,
) (*dto.BatchCreateResponse, error) {
	if err := validateBatchSize(len(cmd.Rules), "rules"); err != nil {
		return nil, err
	}

	uc.logger.Infow("batch creating forward rules (admin)", "count", len(cmd.Rules))

	result := &dto.BatchCreateResponse{
		Succeeded: make([]dto.BatchCreateResult, 0),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	for i, ruleCmd := range cmd.Rules {
		// Use original index if provided, otherwise use loop index
		originalIndex := i
		if len(cmd.CmdIndices) > i {
			originalIndex = cmd.CmdIndices[i]
		}

		created, err := uc.createRuleUC.Execute(ctx, ruleCmd)
		if err != nil {
			// Use rule name or index as identifier for failed creates
			identifier := fmt.Sprintf("index_%d", originalIndex)
			if ruleCmd.Name != "" {
				identifier = ruleCmd.Name
			}
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     identifier,
				Reason: err.Error(),
			})
			continue
		}
		result.Succeeded = append(result.Succeeded, dto.BatchCreateResult{
			Index: originalIndex,
			ID:    created.ID,
		})
	}

	uc.logger.Infow("batch create completed (admin)",
		"succeeded", len(result.Succeeded),
		"failed", len(result.Failed))

	return result, nil
}

// BatchCreateUser creates multiple forward rules for a user.
// Uses partial failure mode: continues on individual errors.
func (uc *BatchForwardRuleUseCase) BatchCreateUser(
	ctx context.Context,
	cmd BatchCreateUserCommand,
) (*dto.BatchCreateResponse, error) {
	if cmd.UserID == 0 {
		return nil, errors.NewValidationError("user_id is required")
	}
	if err := validateBatchSize(len(cmd.Rules), "rules"); err != nil {
		return nil, err
	}

	uc.logger.Infow("batch creating forward rules (user)", "user_id", cmd.UserID, "count", len(cmd.Rules))

	result := &dto.BatchCreateResponse{
		Succeeded: make([]dto.BatchCreateResult, 0),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	for i, ruleCmd := range cmd.Rules {
		// Use original index if provided, otherwise use loop index
		originalIndex := i
		if len(cmd.CmdIndices) > i {
			originalIndex = cmd.CmdIndices[i]
		}

		// Ensure user ID is set
		ruleCmd.UserID = cmd.UserID

		created, err := uc.createUserRuleUC.Execute(ctx, ruleCmd)
		if err != nil {
			// Use rule name or index as identifier for failed creates
			identifier := fmt.Sprintf("index_%d", originalIndex)
			if ruleCmd.Name != "" {
				identifier = ruleCmd.Name
			}
			result.Failed = append(result.Failed, dto.BatchOperationErr{
				ID:     identifier,
				Reason: err.Error(),
			})
			continue
		}
		result.Succeeded = append(result.Succeeded, dto.BatchCreateResult{
			Index: originalIndex,
			ID:    created.ID,
		})
	}

	uc.logger.Infow("batch create completed (user)",
		"user_id", cmd.UserID,
		"succeeded", len(result.Succeeded),
		"failed", len(result.Failed))

	return result, nil
}

// BatchDelete deletes multiple forward rules.
// Uses transactional mode: all operations succeed or all fail (rollback).
// Pre-validation errors are returned before any deletion is attempted.
func (uc *BatchForwardRuleUseCase) BatchDelete(
	ctx context.Context,
	cmd BatchDeleteCommand,
) (*dto.BatchOperationResult, error) {
	if err := validateBatchSize(len(cmd.RuleSIDs), "rule_ids"); err != nil {
		return nil, err
	}

	uc.logger.Infow("batch deleting forward rules", "count", len(cmd.RuleSIDs), "user_id", cmd.UserID)

	validation, err := uc.validateBatchRuleSIDs(ctx, cmd.RuleSIDs, cmd.UserID, "delete")
	if err != nil {
		return nil, err
	}

	// If any pre-validation failed, return errors without executing any deletions
	if len(validation.result.Failed) > 0 {
		uc.logger.Warnw("batch delete aborted due to validation errors",
			"failed_count", len(validation.result.Failed))
		return validation.result, nil
	}

	// Collect validated SIDs for transaction execution (already deduplicated)
	validSIDs := make([]string, 0, len(validation.validRules))
	for sid := range validation.validRules {
		validSIDs = append(validSIDs, sid)
	}

	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0, len(validSIDs)),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	// Execute all deletions in a transaction
	txErr := uc.txMgr.RunInTransaction(ctx, func(txCtx context.Context) error {
		for _, sid := range validSIDs {
			deleteCmd := DeleteForwardRuleCommand{ShortID: sid}
			if err := uc.deleteRuleUC.Execute(txCtx, deleteCmd); err != nil {
				// Transaction will rollback, record the failure
				result.Failed = append(result.Failed, dto.BatchOperationErr{
					ID:     sid,
					Reason: err.Error(),
				})
				return fmt.Errorf("failed to delete rule %s: %w", sid, err)
			}
			result.Succeeded = append(result.Succeeded, sid)
		}
		return nil
	})

	if txErr != nil {
		// Transaction failed, clear succeeded list (all were rolled back)
		uc.logger.Warnw("batch delete transaction rolled back", "error", txErr)
		result.Succeeded = make([]string, 0)
	}

	uc.logger.Infow("batch delete completed",
		"succeeded", len(result.Succeeded),
		"failed", len(result.Failed))

	return result, nil
}

// BatchToggleStatus enables or disables multiple forward rules.
// Uses transactional mode: all operations succeed or all fail (rollback).
// Pre-validation errors are returned before any status change is attempted.
func (uc *BatchForwardRuleUseCase) BatchToggleStatus(
	ctx context.Context,
	cmd BatchToggleStatusCommand,
) (*dto.BatchOperationResult, error) {
	if err := validateBatchSize(len(cmd.RuleSIDs), "rule_ids"); err != nil {
		return nil, err
	}

	action := "disabling"
	if cmd.Enable {
		action = "enabling"
	}
	uc.logger.Infow("batch "+action+" forward rules", "count", len(cmd.RuleSIDs), "user_id", cmd.UserID)

	validation, err := uc.validateBatchRuleSIDs(ctx, cmd.RuleSIDs, cmd.UserID, "modify")
	if err != nil {
		return nil, err
	}

	// If any pre-validation failed, return errors without executing any operations
	if len(validation.result.Failed) > 0 {
		uc.logger.Warnw("batch "+action+" aborted due to validation errors",
			"failed_count", len(validation.result.Failed))
		return validation.result, nil
	}

	// Collect validated SIDs for transaction execution (already deduplicated)
	validSIDs := make([]string, 0, len(validation.validRules))
	for sid := range validation.validRules {
		validSIDs = append(validSIDs, sid)
	}

	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0, len(validSIDs)),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	// Execute all status changes in a transaction
	txErr := uc.txMgr.RunInTransaction(ctx, func(txCtx context.Context) error {
		for _, sid := range validSIDs {
			var opErr error
			if cmd.Enable {
				opErr = uc.enableRuleUC.Execute(txCtx, EnableForwardRuleCommand{ShortID: sid})
			} else {
				opErr = uc.disableRuleUC.Execute(txCtx, DisableForwardRuleCommand{ShortID: sid})
			}

			if opErr != nil {
				result.Failed = append(result.Failed, dto.BatchOperationErr{
					ID:     sid,
					Reason: opErr.Error(),
				})
				verb := "disable"
				if cmd.Enable {
					verb = "enable"
				}
				return fmt.Errorf("failed to %s rule %s: %w", verb, sid, opErr)
			}
			result.Succeeded = append(result.Succeeded, sid)
		}
		return nil
	})

	if txErr != nil {
		// Transaction failed, clear succeeded list (all were rolled back)
		uc.logger.Warnw("batch "+action+" transaction rolled back", "error", txErr)
		result.Succeeded = make([]string, 0)
	}

	uc.logger.Infow("batch "+action+" completed",
		"succeeded", len(result.Succeeded),
		"failed", len(result.Failed))

	return result, nil
}

// BatchUpdate updates multiple forward rules (basic fields only: name, remark, sort_order).
// Uses transactional mode: all operations succeed or all fail (rollback).
// Pre-validation errors are returned before any update is attempted.
func (uc *BatchForwardRuleUseCase) BatchUpdate(
	ctx context.Context,
	cmd BatchUpdateCommand,
) (*dto.BatchOperationResult, error) {
	if err := validateBatchSize(len(cmd.Updates), "updates"); err != nil {
		return nil, err
	}

	uc.logger.Infow("batch updating forward rules", "count", len(cmd.Updates), "user_id", cmd.UserID)

	// Deduplicate updates by SID (keep last occurrence for each SID)
	updateMap := make(map[string]BatchUpdateItem, len(cmd.Updates))
	for _, update := range cmd.Updates {
		updateMap[update.RuleSID] = update
	}

	// Collect unique SIDs for validation
	sids := make([]string, 0, len(updateMap))
	for sid := range updateMap {
		sids = append(sids, sid)
	}

	validation, err := uc.validateBatchRuleSIDs(ctx, sids, cmd.UserID, "update")
	if err != nil {
		return nil, err
	}

	// If any pre-validation failed, return errors without executing any updates
	if len(validation.result.Failed) > 0 {
		uc.logger.Warnw("batch update aborted due to validation errors",
			"failed_count", len(validation.result.Failed))
		return validation.result, nil
	}

	result := &dto.BatchOperationResult{
		Succeeded: make([]string, 0, len(validation.validRules)),
		Failed:    make([]dto.BatchOperationErr, 0),
	}

	// Execute all updates in a transaction using validated SIDs
	txErr := uc.txMgr.RunInTransaction(ctx, func(txCtx context.Context) error {
		for sid := range validation.validRules {
			update := updateMap[sid]
			updateCmd := UpdateForwardRuleCommand{
				ShortID:          update.RuleSID,
				UserID:           cmd.UserID, // pass UserID for agent access validation
				Name:             update.Name,
				Remark:           update.Remark,
				SortOrder:        update.SortOrder,
				AgentShortID:     update.AgentShortID,
				ExitAgentShortID: update.ExitAgentShortID,
			}
			if err := uc.updateRuleUC.Execute(txCtx, updateCmd); err != nil {
				result.Failed = append(result.Failed, dto.BatchOperationErr{
					ID:     update.RuleSID,
					Reason: err.Error(),
				})
				return fmt.Errorf("failed to update rule %s: %w", update.RuleSID, err)
			}
			result.Succeeded = append(result.Succeeded, update.RuleSID)
		}
		return nil
	})

	if txErr != nil {
		// Transaction failed, clear succeeded list (all were rolled back)
		uc.logger.Warnw("batch update transaction rolled back", "error", txErr)
		result.Succeeded = make([]string, 0)
	}

	uc.logger.Infow("batch update completed",
		"succeeded", len(result.Succeeded),
		"failed", len(result.Failed))

	return result, nil
}
