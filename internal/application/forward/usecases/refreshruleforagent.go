// Package usecases contains the application use cases for forward domain.
package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// RefreshRuleForAgentQuery represents the input for refreshing a single rule.
type RefreshRuleForAgentQuery struct {
	AgentID     uint
	RuleShortID string // The SID of the rule to refresh (Stripe-style ID like "fr_xK9mP2vL3nQ")
}

// RefreshRuleForAgentResult represents the refreshed rule data.
type RefreshRuleForAgentResult struct {
	Rule *dto.ForwardRuleDTO
}

// RefreshRuleForAgentExecutor defines the interface for this use case.
type RefreshRuleForAgentExecutor interface {
	Execute(ctx context.Context, query RefreshRuleForAgentQuery) (*RefreshRuleForAgentResult, error)
}

// SingleRuleConverter defines the interface for converting a single rule for agent API responses.
type SingleRuleConverter interface {
	// ConvertForAgent converts a single rule to DTO with role-specific information.
	ConvertForAgent(ctx context.Context, rule *forward.ForwardRule, agentID uint) (*dto.ForwardRuleDTO, error)
}

// RefreshRuleForAgentUseCase implements the use case for refreshing a single rule for an agent.
type RefreshRuleForAgentUseCase struct {
	repo          forward.Repository
	ruleConverter SingleRuleConverter
	logger        logger.Interface
}

// NewRefreshRuleForAgentUseCase creates a new RefreshRuleForAgentUseCase.
func NewRefreshRuleForAgentUseCase(
	repo forward.Repository,
	ruleConverter SingleRuleConverter,
	logger logger.Interface,
) *RefreshRuleForAgentUseCase {
	return &RefreshRuleForAgentUseCase{
		repo:          repo,
		ruleConverter: ruleConverter,
		logger:        logger,
	}
}

// Execute retrieves and refreshes a single rule for the authenticated agent.
// It verifies that the agent has access to the rule before returning the refreshed data.
// Access is granted if the agent is:
// - The entry agent (agent_id matches)
// - The exit agent (exit_agent_id matches, for entry type rules)
// - A participant in the chain (for chain and direct_chain type rules)
func (uc *RefreshRuleForAgentUseCase) Execute(ctx context.Context, query RefreshRuleForAgentQuery) (*RefreshRuleForAgentResult, error) {
	if query.AgentID == 0 {
		return nil, fmt.Errorf("agent_id is required")
	}
	if query.RuleShortID == "" {
		return nil, fmt.Errorf("rule_short_id is required")
	}

	uc.logger.Debugw("executing refresh rule for agent use case",
		"agent_id", query.AgentID,
		"rule_short_id", query.RuleShortID,
	)

	// Look up the rule by SID (database stores full prefixed ID like "fr_xxx")
	rule, err := uc.repo.GetBySID(ctx, query.RuleShortID)
	if err != nil {
		uc.logger.Warnw("failed to get rule by SID",
			"rule_short_id", query.RuleShortID,
			"agent_id", query.AgentID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	if rule == nil {
		uc.logger.Debugw("rule not found",
			"rule_short_id", query.RuleShortID,
			"agent_id", query.AgentID,
		)
		return nil, fmt.Errorf("rule not found: %s", query.RuleShortID)
	}

	// Verify that this agent has access to the rule
	if !uc.hasAccess(rule, query.AgentID) {
		uc.logger.Warnw("agent does not have access to rule",
			"rule_short_id", query.RuleShortID,
			"agent_id", query.AgentID,
			"rule_agent_id", rule.AgentID(),
			"rule_type", rule.RuleType().String(),
		)
		return nil, fmt.Errorf("access denied: agent does not have access to rule %s", query.RuleShortID)
	}

	// Convert rule to DTO with role-specific information
	ruleDTO, err := uc.ruleConverter.ConvertForAgent(ctx, rule, query.AgentID)
	if err != nil {
		uc.logger.Errorw("failed to convert rule for agent",
			"rule_short_id", query.RuleShortID,
			"agent_id", query.AgentID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to convert rule: %w", err)
	}

	uc.logger.Infow("rule refresh successful",
		"rule_short_id", query.RuleShortID,
		"agent_id", query.AgentID,
		"rule_type", rule.RuleType().String(),
		"role", ruleDTO.Role,
	)

	return &RefreshRuleForAgentResult{
		Rule: ruleDTO,
	}, nil
}

// hasAccess checks if the agent has access to the rule.
// Access is granted if the agent is:
// - The entry agent (agent_id matches)
// - The exit agent (exit_agent_id matches, for entry type rules)
// - A participant in the chain (for chain and direct_chain type rules)
func (uc *RefreshRuleForAgentUseCase) hasAccess(rule *forward.ForwardRule, agentID uint) bool {
	// Check if agent is the entry agent
	if rule.AgentID() == agentID {
		return true
	}

	// Check if agent is one of the exit agents (for entry type rules, supports load balancing)
	ruleType := rule.RuleType().String()
	if ruleType == "entry" {
		for _, exitAgentID := range rule.GetAllExitAgentIDs() {
			if exitAgentID == agentID {
				return true
			}
		}
	}

	// Check if agent is in the chain (for chain and direct_chain type rules)
	if ruleType == "chain" || ruleType == "direct_chain" {
		if rule.GetChainPosition(agentID) >= 0 {
			return true
		}
	}

	return false
}
