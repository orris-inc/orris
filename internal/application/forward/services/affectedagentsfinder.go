package services

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AffectedAgentsFinder finds agents affected by rule, node, or agent changes.
// It encapsulates the logic for determining which agents need to be notified
// when configuration changes occur.
type AffectedAgentsFinder struct {
	repo      forward.RuleQuerier
	agentRepo forward.AgentRepository
	logger    logger.Interface
}

// NewAffectedAgentsFinder creates a new AffectedAgentsFinder.
func NewAffectedAgentsFinder(
	repo forward.RuleQuerier,
	agentRepo forward.AgentRepository,
	log logger.Interface,
) *AffectedAgentsFinder {
	return &AffectedAgentsFinder{
		repo:      repo,
		agentRepo: agentRepo,
		logger:    log,
	}
}

// FindByNodeChange finds all agents affected by a node address change.
// Returns a map of agentID -> affected rules for that agent.
// Only returns agents that actually connect to the target node:
// - direct rules: entry agent connects directly to target node
// - entry rules: exit agent connects to target node
// - chain/direct_chain rules: last agent (exit) connects to target node
func (f *AffectedAgentsFinder) FindByNodeChange(ctx context.Context, nodeID uint) (map[uint][]*forward.ForwardRule, error) {
	// Find all enabled rules targeting this node
	rules, err := f.repo.ListEnabledByTargetNodeID(ctx, nodeID)
	if err != nil {
		f.logger.Errorw("failed to list rules by target node ID",
			"node_id", nodeID,
			"error", err,
		)
		return nil, err
	}

	if len(rules) == 0 {
		f.logger.Debugw("no rules targeting node",
			"node_id", nodeID,
		)
		return nil, nil
	}

	// Collect agent IDs that need to be notified with their rules
	agentRulesMap := make(map[uint][]*forward.ForwardRule)
	for _, rule := range rules {
		ruleType := rule.RuleType().String()

		switch ruleType {
		case "direct":
			// Direct rule: entry agent connects directly to target node
			agentRulesMap[rule.AgentID()] = append(agentRulesMap[rule.AgentID()], rule)

		case "entry":
			// Entry rule: exit agent(s) connect to target node
			// Check for multiple exit agents first
			exitAgents := rule.ExitAgents()
			if len(exitAgents) > 0 {
				for _, aw := range exitAgents {
					agentRulesMap[aw.AgentID()] = append(agentRulesMap[aw.AgentID()], rule)
				}
			} else if rule.ExitAgentID() != 0 {
				// Single exit agent
				agentRulesMap[rule.ExitAgentID()] = append(agentRulesMap[rule.ExitAgentID()], rule)
			} else {
				f.logger.Warnw("entry rule has no exit_agent_id or exit_agents, cannot determine affected agent",
					"rule_id", rule.ID(),
					"node_id", nodeID,
				)
			}

		case "chain", "direct_chain":
			// Chain rules: last agent (exit) connects to target node
			chainAgentIDs := rule.ChainAgentIDs()
			if len(chainAgentIDs) > 0 {
				lastAgentID := chainAgentIDs[len(chainAgentIDs)-1]
				agentRulesMap[lastAgentID] = append(agentRulesMap[lastAgentID], rule)
			} else {
				f.logger.Warnw("chain rule has empty chain_agent_ids, cannot determine affected agent",
					"rule_id", rule.ID(),
					"rule_type", ruleType,
					"node_id", nodeID,
				)
			}
		}
	}

	return agentRulesMap, nil
}

// FindByAgentPortChange finds all entry agents affected by an exit agent's port change.
// Returns a map of entryAgentID -> true for agents that need notification.
// This includes:
// - Entry rules where the agent is the exit agent
// - Chain/direct_chain rules where the agent is a next hop for another agent
func (f *AffectedAgentsFinder) FindByAgentPortChange(ctx context.Context, exitAgentID uint) (map[uint]bool, error) {
	// Find all entry rules where this agent is the exit agent
	entryRules, err := f.repo.ListEnabledByExitAgentID(ctx, exitAgentID)
	if err != nil {
		f.logger.Errorw("failed to list entry rules for exit agent",
			"exit_agent_id", exitAgentID,
			"error", err,
		)
		return nil, err
	}

	// Collect unique entry agent IDs from entry rules
	entryAgentIDs := make(map[uint]bool)
	for _, rule := range entryRules {
		if rule.RuleType().String() == "entry" {
			entryAgentIDs[rule.AgentID()] = true
		}
	}

	// Also check chain rules where this agent participates
	chainRules, err := f.repo.ListEnabledByChainAgentID(ctx, exitAgentID)
	if err != nil {
		f.logger.Warnw("failed to list chain rules for exit agent",
			"exit_agent_id", exitAgentID,
			"error", err,
		)
		// Continue with entry rules
	} else {
		for _, rule := range chainRules {
			ruleType := rule.RuleType().String()
			if ruleType == "chain" || ruleType == "direct_chain" {
				// Find agents that have this exit agent as their next hop
				// Full chain: [entry_agent] + chainAgentIDs
				fullChain := append([]uint{rule.AgentID()}, rule.ChainAgentIDs()...)
				for i, agentID := range fullChain {
					// Check if exitAgentID is the next hop for this agent
					if i+1 < len(fullChain) && fullChain[i+1] == exitAgentID {
						entryAgentIDs[agentID] = true
					}
				}
			}
		}
	}

	return entryAgentIDs, nil
}

// FindByRuleChange finds all agents that should be notified about a rule change.
// Returns a slice of agent IDs that need notification based on the rule type:
// - direct: only the entry agent
// - entry: both entry and exit agents
// - chain/direct_chain: entry agent and all chain agents
func (f *AffectedAgentsFinder) FindByRuleChange(ctx context.Context, rule *forward.ForwardRule) ([]uint, error) {
	agentIDs := make(map[uint]bool)

	ruleType := rule.RuleType().String()

	switch ruleType {
	case "direct":
		// Only the entry agent needs to be notified
		agentIDs[rule.AgentID()] = true

	case "entry":
		// Entry agent and all exit agents need to be notified
		agentIDs[rule.AgentID()] = true
		// Check for multiple exit agents first
		exitAgents := rule.ExitAgents()
		if len(exitAgents) > 0 {
			for _, aw := range exitAgents {
				agentIDs[aw.AgentID()] = true
			}
		} else if rule.ExitAgentID() != 0 {
			// Single exit agent
			agentIDs[rule.ExitAgentID()] = true
		}

	case "chain", "direct_chain":
		// Entry agent and all chain agents need to be notified
		agentIDs[rule.AgentID()] = true
		for _, chainAgentID := range rule.ChainAgentIDs() {
			agentIDs[chainAgentID] = true
		}
	}

	// Convert map to slice
	result := make([]uint, 0, len(agentIDs))
	for agentID := range agentIDs {
		result = append(result, agentID)
	}

	return result, nil
}

// GetEntryRulesForExitAgent retrieves enabled rules for an agent that has exitAgentID as its next hop.
// The agentID parameter is the agent that needs to be notified (not necessarily the entry agent).
// This is used when an exit agent's address/port changes to find which rules need to be updated
// for a specific upstream agent.
func (f *AffectedAgentsFinder) GetEntryRulesForExitAgent(ctx context.Context, agentID, exitAgentID uint) ([]*forward.ForwardRule, error) {
	// Get all enabled rules where this agent is the entry agent
	rules, err := f.repo.ListEnabledByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Also get chain rules where this agent participates (for chain middle nodes)
	chainRules, err := f.repo.ListEnabledByChainAgentID(ctx, agentID)
	if err != nil {
		f.logger.Warnw("failed to list chain rules for agent",
			"agent_id", agentID,
			"error", err,
		)
		// Continue with entry rules
	} else {
		rules = append(rules, chainRules...)
	}

	// Deduplicate rules by ID
	ruleMap := make(map[uint]*forward.ForwardRule)
	for _, rule := range rules {
		ruleMap[rule.ID()] = rule
	}

	// Filter rules that have exitAgentID as next hop for this agent
	result := make([]*forward.ForwardRule, 0)
	for _, rule := range ruleMap {
		switch rule.RuleType().String() {
		case "entry":
			// Check single exit agent
			if rule.ExitAgentID() == exitAgentID {
				result = append(result, rule)
				continue
			}
			// Check multiple exit agents
			for _, aw := range rule.ExitAgents() {
				if aw.AgentID() == exitAgentID {
					result = append(result, rule)
					break
				}
			}
		case "chain":
			// Check if this agent has exitAgentID as its next hop
			nextHop := rule.GetNextHopAgentID(agentID)
			if nextHop == exitAgentID {
				result = append(result, rule)
			}
		case "direct_chain":
			// Check if this agent has exitAgentID as its next hop in direct_chain
			nextHop, _, err := rule.GetNextHopForDirectChainSafe(agentID)
			if err != nil {
				f.logger.Warnw("failed to get next hop for direct_chain",
					"rule_id", rule.ID(),
					"agent_id", agentID,
					"error", err,
				)
				continue
			}
			if nextHop == exitAgentID {
				result = append(result, rule)
			}
		}
	}

	return result, nil
}
