// Package usecases contains the application use cases for forward domain.
package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetEnabledRulesForAgentQuery represents the input for getting enabled rules.
type GetEnabledRulesForAgentQuery struct {
	AgentID uint
}

// GetEnabledRulesForAgentResult represents the output with rules and client token.
type GetEnabledRulesForAgentResult struct {
	Rules       []*dto.ForwardRuleDTO
	ClientToken string
}

// AgentRuleConverter defines the interface for converting rules for agent API responses.
type AgentRuleConverter interface {
	// ConvertBatch converts multiple rules for the same agent.
	// It optimizes by batching agent SID lookups where possible.
	ConvertBatch(ctx context.Context, rules []*forward.ForwardRule, agentID uint) ([]*dto.ForwardRuleDTO, error)

	// GenerateClientToken generates a client token for the given agent SID.
	GenerateClientToken(agentSID string) string
}

// GetEnabledRulesForAgentExecutor defines the interface for this use case.
type GetEnabledRulesForAgentExecutor interface {
	Execute(ctx context.Context, query GetEnabledRulesForAgentQuery) (*GetEnabledRulesForAgentResult, error)
}

// GetEnabledRulesForAgentUseCase implements the use case for getting enabled rules for an agent.
type GetEnabledRulesForAgentUseCase struct {
	repo          forward.RuleQuerier
	agentRepo     forward.AgentRepository
	ruleConverter AgentRuleConverter
	logger        logger.Interface
}

// NewGetEnabledRulesForAgentUseCase creates a new GetEnabledRulesForAgentUseCase.
func NewGetEnabledRulesForAgentUseCase(
	repo forward.RuleQuerier,
	agentRepo forward.AgentRepository,
	ruleConverter AgentRuleConverter,
	logger logger.Interface,
) *GetEnabledRulesForAgentUseCase {
	return &GetEnabledRulesForAgentUseCase{
		repo:          repo,
		agentRepo:     agentRepo,
		ruleConverter: ruleConverter,
		logger:        logger,
	}
}

// Execute retrieves all enabled rules for a specific agent and generates a client token.
// It queries rules where the agent participates as:
// - Entry agent (agent_id)
// - Exit agent (exit_agent_id)
// - Chain participant (chain_agent_ids)
// The rules are merged and deduplicated before conversion.
func (uc *GetEnabledRulesForAgentUseCase) Execute(ctx context.Context, query GetEnabledRulesForAgentQuery) (*GetEnabledRulesForAgentResult, error) {
	if query.AgentID == 0 {
		return nil, fmt.Errorf("agent_id is required")
	}

	uc.logger.Debugw("executing get enabled rules for agent use case",
		"agent_id", query.AgentID,
	)

	// Retrieve enabled forward rules for this agent (as entry agent)
	entryRules, err := uc.repo.ListEnabledByAgentID(ctx, query.AgentID)
	if err != nil {
		uc.logger.Errorw("failed to retrieve enabled entry rules",
			"agent_id", query.AgentID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to retrieve enabled entry rules: %w", err)
	}

	// Retrieve entry rules where this agent is the exit agent
	exitRules, err := uc.repo.ListEnabledByExitAgentID(ctx, query.AgentID)
	if err != nil {
		uc.logger.Errorw("failed to retrieve enabled exit rules",
			"agent_id", query.AgentID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to retrieve enabled exit rules: %w", err)
	}

	// Retrieve chain rules where this agent participates
	chainRules, err := uc.repo.ListEnabledByChainAgentID(ctx, query.AgentID)
	if err != nil {
		uc.logger.Errorw("failed to retrieve enabled chain rules",
			"agent_id", query.AgentID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to retrieve enabled chain rules: %w", err)
	}

	// Merge rules (avoid duplicates by using a map)
	ruleMap := make(map[uint]*forward.ForwardRule)
	for _, rule := range entryRules {
		ruleMap[rule.ID()] = rule
	}
	for _, rule := range exitRules {
		ruleMap[rule.ID()] = rule
	}
	for _, rule := range chainRules {
		ruleMap[rule.ID()] = rule
	}

	// Convert map back to slice
	allRules := make([]*forward.ForwardRule, 0, len(ruleMap))
	for _, rule := range ruleMap {
		allRules = append(allRules, rule)
	}

	uc.logger.Debugw("enabled forward rules retrieved",
		"total_rules", len(allRules),
		"entry_rules", len(entryRules),
		"exit_rules", len(exitRules),
		"chain_rules", len(chainRules),
		"agent_id", query.AgentID,
	)

	// Convert rules to DTOs with role-specific information
	ruleDTOs, err := uc.ruleConverter.ConvertBatch(ctx, allRules, query.AgentID)
	if err != nil {
		uc.logger.Errorw("failed to convert rules to DTOs",
			"agent_id", query.AgentID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to convert rules: %w", err)
	}

	// Generate client token for the requesting agent
	var clientToken string
	requestingAgent, err := uc.agentRepo.GetByID(ctx, query.AgentID)
	if err != nil {
		uc.logger.Warnw("failed to get requesting agent for token",
			"agent_id", query.AgentID,
			"error", err,
		)
	} else if requestingAgent != nil {
		// Generate token using ruleConverter to ensure correct format (fwd_xxx_xxx)
		clientToken = uc.ruleConverter.GenerateClientToken(requestingAgent.SID())
		uc.logger.Infow("generated client token for agent",
			"agent_id", query.AgentID,
			"short_id", requestingAgent.SID(),
		)
	}

	return &GetEnabledRulesForAgentResult{
		Rules:       ruleDTOs,
		ClientToken: clientToken,
	}, nil
}
