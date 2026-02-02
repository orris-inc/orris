// Package valueobjects provides value objects for the forward domain.
package valueobjects

import (
	"fmt"
)

const (
	// DefaultAgentWeight is the default weight for an exit agent.
	DefaultAgentWeight uint16 = 50
	// MinAgentWeight is the minimum allowed weight.
	MinAgentWeight uint16 = 1
	// MaxAgentWeight is the maximum allowed weight.
	MaxAgentWeight uint16 = 100
	// MaxExitAgents is the maximum number of exit agents allowed per rule.
	MaxExitAgents = 10
)

// AgentWeight represents a weighted exit agent for load balancing.
type AgentWeight struct {
	agentID uint
	weight  uint16
}

// NewAgentWeight creates a new AgentWeight with validation.
func NewAgentWeight(agentID uint, weight uint16) (AgentWeight, error) {
	if agentID == 0 {
		return AgentWeight{}, fmt.Errorf("agent ID cannot be zero")
	}
	if weight < MinAgentWeight || weight > MaxAgentWeight {
		return AgentWeight{}, fmt.Errorf("weight must be between %d and %d, got %d", MinAgentWeight, MaxAgentWeight, weight)
	}
	return AgentWeight{
		agentID: agentID,
		weight:  weight,
	}, nil
}

// NewAgentWeightWithDefault creates a new AgentWeight with default weight.
func NewAgentWeightWithDefault(agentID uint) (AgentWeight, error) {
	return NewAgentWeight(agentID, DefaultAgentWeight)
}

// ReconstructAgentWeight recreates an AgentWeight from persistence without validation.
// This should only be used by the mapper layer.
func ReconstructAgentWeight(agentID uint, weight uint16) AgentWeight {
	return AgentWeight{
		agentID: agentID,
		weight:  weight,
	}
}

// AgentID returns the agent ID.
func (aw AgentWeight) AgentID() uint {
	return aw.agentID
}

// Weight returns the weight.
func (aw AgentWeight) Weight() uint16 {
	return aw.weight
}

// ValidateAgentWeights validates a slice of AgentWeight values.
func ValidateAgentWeights(weights []AgentWeight) error {
	if len(weights) == 0 {
		return nil
	}

	if len(weights) > MaxExitAgents {
		return fmt.Errorf("too many exit agents: maximum %d allowed, got %d", MaxExitAgents, len(weights))
	}

	// Check for duplicate agent IDs
	seen := make(map[uint]bool, len(weights))
	for _, w := range weights {
		if seen[w.agentID] {
			return fmt.Errorf("duplicate agent ID in exit agents: %d", w.agentID)
		}
		seen[w.agentID] = true
	}

	return nil
}

// GetAgentIDs extracts all agent IDs from a slice of AgentWeight.
func GetAgentIDs(weights []AgentWeight) []uint {
	if len(weights) == 0 {
		return nil
	}
	ids := make([]uint, len(weights))
	for i, w := range weights {
		ids[i] = w.agentID
	}
	return ids
}
