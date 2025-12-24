package valueobjects

import "fmt"

// PlanType represents the type of a subscription plan
type PlanType string

const (
	// PlanTypeNode represents a node-based subscription plan
	PlanTypeNode PlanType = "node"
	// PlanTypeForward represents a forward-based subscription plan
	PlanTypeForward PlanType = "forward"
	// PlanTypeHybrid represents a hybrid subscription plan (both official nodes and user's own forward rules)
	PlanTypeHybrid PlanType = "hybrid"
)

// IsValid checks if the plan type is valid
func (pt PlanType) IsValid() bool {
	return pt == PlanTypeNode || pt == PlanTypeForward || pt == PlanTypeHybrid
}

// String returns the string representation of the plan type
func (pt PlanType) String() string {
	return string(pt)
}

// NewPlanType creates a new PlanType from a string
func NewPlanType(s string) (PlanType, error) {
	pt := PlanType(s)
	if !pt.IsValid() {
		return "", fmt.Errorf("invalid plan type: %s, must be 'node', 'forward', or 'hybrid'", s)
	}
	return pt, nil
}

// IsNode checks if the plan type is node
func (pt PlanType) IsNode() bool {
	return pt == PlanTypeNode
}

// IsForward checks if the plan type is forward
func (pt PlanType) IsForward() bool {
	return pt == PlanTypeForward
}

// IsHybrid checks if the plan type is hybrid
func (pt PlanType) IsHybrid() bool {
	return pt == PlanTypeHybrid
}
