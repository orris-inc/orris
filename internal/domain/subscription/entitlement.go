package subscription

import (
	"fmt"
	"time"
)

// EntitlementResourceType represents the type of resource linked to a plan
type EntitlementResourceType string

const (
	EntitlementResourceTypeNode         EntitlementResourceType = "node"
	EntitlementResourceTypeForwardAgent EntitlementResourceType = "forward_agent"
)

// IsValid checks if the entitlement resource type is valid
func (rt EntitlementResourceType) IsValid() bool {
	switch rt {
	case EntitlementResourceTypeNode, EntitlementResourceTypeForwardAgent:
		return true
	default:
		return false
	}
}

// String returns the string representation of the entitlement resource type
func (rt EntitlementResourceType) String() string {
	return string(rt)
}

// Entitlement represents the association between a plan and a resource
// This is a simple association entity without complex domain logic
type Entitlement struct {
	id           uint
	planID       uint // References subscription_plans.id (the template)
	resourceType EntitlementResourceType
	resourceID   uint
	createdAt    time.Time
}

// NewEntitlement creates a new entitlement association
func NewEntitlement(planID uint, resourceType EntitlementResourceType, resourceID uint) (*Entitlement, error) {
	if planID == 0 {
		return nil, fmt.Errorf("plan ID is required")
	}
	if !resourceType.IsValid() {
		return nil, fmt.Errorf("invalid resource type: %s", resourceType)
	}
	if resourceID == 0 {
		return nil, fmt.Errorf("resource ID is required")
	}

	return &Entitlement{
		planID:       planID,
		resourceType: resourceType,
		resourceID:   resourceID,
		createdAt:    time.Now(),
	}, nil
}

// ReconstructEntitlement reconstructs an entitlement from persistence
func ReconstructEntitlement(id, planID uint, resourceType string, resourceID uint, createdAt time.Time) (*Entitlement, error) {
	if id == 0 {
		return nil, fmt.Errorf("entitlement ID cannot be zero")
	}

	rt := EntitlementResourceType(resourceType)
	if !rt.IsValid() {
		return nil, fmt.Errorf("invalid resource type: %s", resourceType)
	}

	return &Entitlement{
		id:           id,
		planID:       planID,
		resourceType: rt,
		resourceID:   resourceID,
		createdAt:    createdAt,
	}, nil
}

// ID returns the entitlement ID
func (e *Entitlement) ID() uint {
	return e.id
}

// SetID sets the entitlement ID (for persistence layer)
func (e *Entitlement) SetID(id uint) error {
	if e.id != 0 {
		return fmt.Errorf("entitlement ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("entitlement ID cannot be zero")
	}
	e.id = id
	return nil
}

// PlanID returns the plan ID
func (e *Entitlement) PlanID() uint {
	return e.planID
}

// ResourceType returns the resource type
func (e *Entitlement) ResourceType() EntitlementResourceType {
	return e.resourceType
}

// ResourceID returns the resource ID
func (e *Entitlement) ResourceID() uint {
	return e.resourceID
}

// CreatedAt returns the creation timestamp
func (e *Entitlement) CreatedAt() time.Time {
	return e.createdAt
}
