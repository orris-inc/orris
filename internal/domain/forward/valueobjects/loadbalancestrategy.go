// Package valueobjects provides value objects for the forward domain.
package valueobjects

// LoadBalanceStrategy represents the load balancing strategy for multi-exit rules.
type LoadBalanceStrategy string

const (
	// LoadBalanceStrategyFailover uses priority-based failover.
	// Agents are tried in order of weight (highest first), with weight=0 as backup.
	LoadBalanceStrategyFailover LoadBalanceStrategy = "failover"

	// LoadBalanceStrategyWeighted distributes traffic based on weight ratios.
	LoadBalanceStrategyWeighted LoadBalanceStrategy = "weighted"

	// DefaultLoadBalanceStrategy is the default strategy if not specified.
	DefaultLoadBalanceStrategy = LoadBalanceStrategyFailover
)

// IsValid checks if the load balance strategy is valid.
func (s LoadBalanceStrategy) IsValid() bool {
	switch s {
	case LoadBalanceStrategyFailover, LoadBalanceStrategyWeighted:
		return true
	default:
		return false
	}
}

// String returns the string representation of the strategy.
func (s LoadBalanceStrategy) String() string {
	return string(s)
}

// IsFailover returns true if the strategy is failover.
func (s LoadBalanceStrategy) IsFailover() bool {
	return s == LoadBalanceStrategyFailover
}

// IsWeighted returns true if the strategy is weighted.
func (s LoadBalanceStrategy) IsWeighted() bool {
	return s == LoadBalanceStrategyWeighted
}

// ParseLoadBalanceStrategy parses a string to LoadBalanceStrategy.
// Returns DefaultLoadBalanceStrategy for empty or invalid input.
func ParseLoadBalanceStrategy(s string) LoadBalanceStrategy {
	strategy := LoadBalanceStrategy(s)
	if strategy.IsValid() {
		return strategy
	}
	return DefaultLoadBalanceStrategy
}
