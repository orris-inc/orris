package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DeleteForwardAgentCommand represents the input for deleting a forward agent.
type DeleteForwardAgentCommand struct {
	ShortID string // External API identifier
}

// AgentAlertStateClearer clears alert state for an agent when it is deleted.
// This prevents stale alert states from causing incorrect recovery notifications.
type AgentAlertStateClearer interface {
	ClearAgentAlertState(ctx context.Context, agentID uint) error
}

// DeleteForwardAgentUseCase handles forward agent deletion.
type DeleteForwardAgentUseCase struct {
	agentRepo         forward.AgentRepository
	ruleRepo          forward.Repository
	alertStateClearer AgentAlertStateClearer
	logger            logger.Interface
}

// NewDeleteForwardAgentUseCase creates a new DeleteForwardAgentUseCase.
func NewDeleteForwardAgentUseCase(
	agentRepo forward.AgentRepository,
	ruleRepo forward.Repository,
	logger logger.Interface,
) *DeleteForwardAgentUseCase {
	return &DeleteForwardAgentUseCase{
		agentRepo: agentRepo,
		ruleRepo:  ruleRepo,
		logger:    logger,
	}
}

// WithAlertStateClearer sets the alert state clearer for cleanup on delete.
func (uc *DeleteForwardAgentUseCase) WithAlertStateClearer(clearer AgentAlertStateClearer) *DeleteForwardAgentUseCase {
	uc.alertStateClearer = clearer
	return uc
}

// Execute deletes a forward agent.
func (uc *DeleteForwardAgentUseCase) Execute(ctx context.Context, cmd DeleteForwardAgentCommand) error {
	if cmd.ShortID == "" {
		return errors.NewValidationError("short_id is required")
	}

	uc.logger.Infow("executing delete forward agent use case", "short_id", cmd.ShortID)

	agent, err := uc.agentRepo.GetBySID(ctx, cmd.ShortID)
	if err != nil {
		uc.logger.Errorw("failed to get forward agent", "short_id", cmd.ShortID, "error", err)
		return fmt.Errorf("failed to get forward agent: %w", err)
	}
	if agent == nil {
		return errors.NewNotFoundError("forward agent", cmd.ShortID)
	}

	agentID := agent.ID()

	// Check if agent is referenced by any forward rules
	if err := uc.checkAgentReferences(ctx, agentID); err != nil {
		return err
	}

	// Delete the agent using internal ID
	if err := uc.agentRepo.Delete(ctx, agentID); err != nil {
		uc.logger.Errorw("failed to delete forward agent", "id", agentID, "short_id", agent.SID(), "error", err)
		return fmt.Errorf("failed to delete forward agent: %w", err)
	}

	// Clean up alert state to prevent stale recovery notifications
	if uc.alertStateClearer != nil {
		if err := uc.alertStateClearer.ClearAgentAlertState(ctx, agentID); err != nil {
			// Log but don't fail the deletion - alert state has TTL as safety net
			uc.logger.Warnw("failed to clear agent alert state", "agent_id", agentID, "error", err)
		}
	}

	uc.logger.Infow("forward agent deleted successfully", "id", agentID, "short_id", agent.SID())
	return nil
}

// checkAgentReferences checks if the agent is referenced by any forward rules.
func (uc *DeleteForwardAgentUseCase) checkAgentReferences(ctx context.Context, agentID uint) error {
	// Check rules where this agent is the entry agent (agent_id)
	entryRules, err := uc.ruleRepo.ListByAgentID(ctx, agentID)
	if err != nil {
		uc.logger.Errorw("failed to check entry rules", "agent_id", agentID, "error", err)
		return fmt.Errorf("failed to check agent references: %w", err)
	}
	if len(entryRules) > 0 {
		return errors.NewConflictError(fmt.Sprintf("cannot delete agent: %d forward rule(s) use this agent as entry agent", len(entryRules)))
	}

	// Check rules where this agent is the exit agent (exit_agent_id)
	exitRules, err := uc.ruleRepo.ListByExitAgentID(ctx, agentID)
	if err != nil {
		uc.logger.Errorw("failed to check exit rules", "agent_id", agentID, "error", err)
		return fmt.Errorf("failed to check agent references: %w", err)
	}
	if len(exitRules) > 0 {
		return errors.NewConflictError(fmt.Sprintf("cannot delete agent: %d forward rule(s) use this agent as exit agent", len(exitRules)))
	}

	// Check rules where this agent is in the chain (chain_agent_ids)
	chainRules, err := uc.ruleRepo.ListEnabledByChainAgentID(ctx, agentID)
	if err != nil {
		uc.logger.Errorw("failed to check chain rules", "agent_id", agentID, "error", err)
		return fmt.Errorf("failed to check agent references: %w", err)
	}
	if len(chainRules) > 0 {
		return errors.NewConflictError(fmt.Sprintf("cannot delete agent: %d forward rule(s) use this agent in chain", len(chainRules)))
	}

	return nil
}
