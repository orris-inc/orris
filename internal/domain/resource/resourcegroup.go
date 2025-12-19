// Package resource provides domain models for resource management.
package resource

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// GroupStatus represents the status of a resource group
type GroupStatus string

const (
	// GroupStatusActive indicates the group is active
	GroupStatusActive GroupStatus = "active"
	// GroupStatusInactive indicates the group is inactive
	GroupStatusInactive GroupStatus = "inactive"
)

// IsValid checks if the group status is valid
func (s GroupStatus) IsValid() bool {
	return s == GroupStatusActive || s == GroupStatusInactive
}

// String returns the string representation of the status
func (s GroupStatus) String() string {
	return string(s)
}

// ResourceGroup represents the resource group aggregate root.
// A resource group contains Node and ForwardAgent resources and is associated with a Plan.
type ResourceGroup struct {
	id          uint
	sid         string // Stripe-style ID: rg_xxxxxxxx
	name        string
	planID      uint
	description string
	status      GroupStatus
	createdAt   time.Time
	updatedAt   time.Time
	version     int
}

// generateSID generates a Stripe-style short ID with the given prefix
func generateSID(prefix string) (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(bytes), nil
}

// NewResourceGroup creates a new resource group
func NewResourceGroup(name string, planID uint, description string) (*ResourceGroup, error) {
	if name == "" {
		return nil, fmt.Errorf("group name is required")
	}
	if planID == 0 {
		return nil, fmt.Errorf("plan ID is required")
	}

	sid, err := generateSID("rg")
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := time.Now()
	return &ResourceGroup{
		sid:         sid,
		name:        name,
		planID:      planID,
		description: description,
		status:      GroupStatusActive,
		createdAt:   now,
		updatedAt:   now,
		version:     1,
	}, nil
}

// ReconstructResourceGroup reconstructs a resource group from persistence
func ReconstructResourceGroup(
	id uint,
	sid string,
	name string,
	planID uint,
	description string,
	status string,
	createdAt, updatedAt time.Time,
	version int,
) (*ResourceGroup, error) {
	if id == 0 {
		return nil, fmt.Errorf("group ID cannot be zero")
	}
	if sid == "" {
		return nil, fmt.Errorf("group SID is required")
	}
	if name == "" {
		return nil, fmt.Errorf("group name is required")
	}
	if planID == 0 {
		return nil, fmt.Errorf("plan ID is required")
	}

	groupStatus := GroupStatus(status)
	if !groupStatus.IsValid() {
		return nil, fmt.Errorf("invalid group status: %s", status)
	}

	return &ResourceGroup{
		id:          id,
		sid:         sid,
		name:        name,
		planID:      planID,
		description: description,
		status:      groupStatus,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		version:     version,
	}, nil
}

// ID returns the group ID
func (g *ResourceGroup) ID() uint {
	return g.id
}

// SID returns the Stripe-style short ID
func (g *ResourceGroup) SID() string {
	return g.sid
}

// Name returns the group name
func (g *ResourceGroup) Name() string {
	return g.name
}

// PlanID returns the associated plan ID
func (g *ResourceGroup) PlanID() uint {
	return g.planID
}

// Description returns the group description
func (g *ResourceGroup) Description() string {
	return g.description
}

// Status returns the group status
func (g *ResourceGroup) Status() GroupStatus {
	return g.status
}

// CreatedAt returns when the group was created
func (g *ResourceGroup) CreatedAt() time.Time {
	return g.createdAt
}

// UpdatedAt returns when the group was last updated
func (g *ResourceGroup) UpdatedAt() time.Time {
	return g.updatedAt
}

// Version returns the aggregate version for optimistic locking
func (g *ResourceGroup) Version() int {
	return g.version
}

// SetID sets the group ID (only for persistence layer use)
func (g *ResourceGroup) SetID(id uint) error {
	if g.id != 0 {
		return fmt.Errorf("group ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("group ID cannot be zero")
	}
	g.id = id
	return nil
}

// UpdateName updates the group name
func (g *ResourceGroup) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	if g.name == name {
		return nil
	}
	g.name = name
	g.updatedAt = time.Now()
	g.version++
	return nil
}

// UpdateDescription updates the group description
func (g *ResourceGroup) UpdateDescription(description string) {
	if g.description == description {
		return
	}
	g.description = description
	g.updatedAt = time.Now()
	g.version++
}

// UpdatePlanID updates the associated plan ID
func (g *ResourceGroup) UpdatePlanID(planID uint) error {
	if planID == 0 {
		return fmt.Errorf("plan ID cannot be zero")
	}
	if g.planID == planID {
		return nil
	}
	g.planID = planID
	g.updatedAt = time.Now()
	g.version++
	return nil
}

// Activate activates the resource group
func (g *ResourceGroup) Activate() {
	if g.status == GroupStatusActive {
		return
	}
	g.status = GroupStatusActive
	g.updatedAt = time.Now()
	g.version++
}

// Deactivate deactivates the resource group
func (g *ResourceGroup) Deactivate() {
	if g.status == GroupStatusInactive {
		return
	}
	g.status = GroupStatusInactive
	g.updatedAt = time.Now()
	g.version++
}

// IsActive checks if the group is active
func (g *ResourceGroup) IsActive() bool {
	return g.status == GroupStatusActive
}

// IncrementVersion increments the version for optimistic locking
func (g *ResourceGroup) IncrementVersion() {
	g.version++
}
