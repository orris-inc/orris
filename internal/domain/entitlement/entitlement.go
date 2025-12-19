package entitlement

import (
	"fmt"
	"time"
)

// Entitlement represents the entitlement aggregate root
// It manages access rights for subjects (users) to resources (nodes, forward agents, features)
type Entitlement struct {
	id           uint
	subjectType  SubjectType       // Type of subject (user)
	subjectID    uint              // ID of the subject
	resourceType ResourceType      // Type of resource (node, forward_agent, feature)
	resourceID   uint              // ID of the resource
	sourceType   SourceType        // Source of entitlement (subscription, direct, promotion, trial)
	sourceID     uint              // ID of the source
	status       EntitlementStatus // Status of the entitlement (active, expired, revoked)
	expiresAt    *time.Time        // Expiration time (nil means no expiration)
	metadata     map[string]any    // Additional metadata for extensibility
	createdAt    time.Time         // Creation timestamp
	updatedAt    time.Time         // Last update timestamp
	version      int               // Version for optimistic locking
}

// NewEntitlement creates a new entitlement
func NewEntitlement(
	subjectType SubjectType,
	subjectID uint,
	resourceType ResourceType,
	resourceID uint,
	sourceType SourceType,
	sourceID uint,
	expiresAt *time.Time,
) (*Entitlement, error) {
	if !subjectType.IsValid() {
		return nil, fmt.Errorf("invalid subject type: %s", subjectType)
	}
	if subjectID == 0 {
		return nil, fmt.Errorf("subject ID is required")
	}
	if !resourceType.IsValid() {
		return nil, fmt.Errorf("invalid resource type: %s", resourceType)
	}
	if resourceID == 0 {
		return nil, fmt.Errorf("resource ID is required")
	}
	if !sourceType.IsValid() {
		return nil, fmt.Errorf("invalid source type: %s", sourceType)
	}
	if sourceID == 0 {
		return nil, fmt.Errorf("source ID is required")
	}

	now := time.Now()
	return &Entitlement{
		subjectType:  subjectType,
		subjectID:    subjectID,
		resourceType: resourceType,
		resourceID:   resourceID,
		sourceType:   sourceType,
		sourceID:     sourceID,
		status:       EntitlementStatusActive,
		expiresAt:    expiresAt,
		metadata:     make(map[string]any),
		createdAt:    now,
		updatedAt:    now,
		version:      1,
	}, nil
}

// ReconstructEntitlement reconstructs an entitlement from persistence
func ReconstructEntitlement(
	id uint,
	subjectType SubjectType,
	subjectID uint,
	resourceType ResourceType,
	resourceID uint,
	sourceType SourceType,
	sourceID uint,
	status EntitlementStatus,
	expiresAt *time.Time,
	metadata map[string]any,
	createdAt, updatedAt time.Time,
	version int,
) (*Entitlement, error) {
	if id == 0 {
		return nil, fmt.Errorf("entitlement ID cannot be zero")
	}
	if !subjectType.IsValid() {
		return nil, fmt.Errorf("invalid subject type: %s", subjectType)
	}
	if subjectID == 0 {
		return nil, fmt.Errorf("subject ID is required")
	}
	if !resourceType.IsValid() {
		return nil, fmt.Errorf("invalid resource type: %s", resourceType)
	}
	if resourceID == 0 {
		return nil, fmt.Errorf("resource ID is required")
	}
	if !sourceType.IsValid() {
		return nil, fmt.Errorf("invalid source type: %s", sourceType)
	}
	if sourceID == 0 {
		return nil, fmt.Errorf("source ID is required")
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid entitlement status: %s", status)
	}

	if metadata == nil {
		metadata = make(map[string]any)
	}

	return &Entitlement{
		id:           id,
		subjectType:  subjectType,
		subjectID:    subjectID,
		resourceType: resourceType,
		resourceID:   resourceID,
		sourceType:   sourceType,
		sourceID:     sourceID,
		status:       status,
		expiresAt:    expiresAt,
		metadata:     metadata,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		version:      version,
	}, nil
}

// ID returns the entitlement ID
func (e *Entitlement) ID() uint {
	return e.id
}

// SubjectType returns the subject type
func (e *Entitlement) SubjectType() SubjectType {
	return e.subjectType
}

// SubjectID returns the subject ID
func (e *Entitlement) SubjectID() uint {
	return e.subjectID
}

// ResourceType returns the resource type
func (e *Entitlement) ResourceType() ResourceType {
	return e.resourceType
}

// ResourceID returns the resource ID
func (e *Entitlement) ResourceID() uint {
	return e.resourceID
}

// SourceType returns the source type
func (e *Entitlement) SourceType() SourceType {
	return e.sourceType
}

// SourceID returns the source ID
func (e *Entitlement) SourceID() uint {
	return e.sourceID
}

// Status returns the entitlement status
func (e *Entitlement) Status() EntitlementStatus {
	return e.status
}

// ExpiresAt returns the expiration time
func (e *Entitlement) ExpiresAt() *time.Time {
	return e.expiresAt
}

// Metadata returns the entitlement metadata
func (e *Entitlement) Metadata() map[string]any {
	return e.metadata
}

// CreatedAt returns when the entitlement was created
func (e *Entitlement) CreatedAt() time.Time {
	return e.createdAt
}

// UpdatedAt returns when the entitlement was last updated
func (e *Entitlement) UpdatedAt() time.Time {
	return e.updatedAt
}

// Version returns the aggregate version for optimistic locking
func (e *Entitlement) Version() int {
	return e.version
}

// SetID sets the entitlement ID (only for persistence layer use)
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

// Revoke revokes the entitlement
func (e *Entitlement) Revoke() error {
	if e.status == EntitlementStatusRevoked {
		return nil // Already revoked
	}

	if e.status == EntitlementStatusExpired {
		return fmt.Errorf("cannot revoke expired entitlement")
	}

	e.status = EntitlementStatusRevoked
	e.updatedAt = time.Now()
	e.version++

	return nil
}

// Expire marks the entitlement as expired
func (e *Entitlement) Expire() error {
	if e.status == EntitlementStatusExpired {
		return nil // Already expired
	}

	if e.status == EntitlementStatusRevoked {
		return fmt.Errorf("cannot expire revoked entitlement")
	}

	e.status = EntitlementStatusExpired
	e.updatedAt = time.Now()
	e.version++

	return nil
}

// IsActive checks if the entitlement is active and valid
// It checks both the status and expiration time
func (e *Entitlement) IsActive() bool {
	if e.status != EntitlementStatusActive {
		return false
	}

	// Check expiration time if set
	if e.expiresAt != nil && time.Now().After(*e.expiresAt) {
		return false
	}

	return true
}

// IsExpired checks if the entitlement has expired based on expiration time
func (e *Entitlement) IsExpired() bool {
	if e.expiresAt == nil {
		return false
	}
	return time.Now().After(*e.expiresAt)
}

// SetMetadata sets a metadata value
func (e *Entitlement) SetMetadata(key string, value any) {
	if e.metadata == nil {
		e.metadata = make(map[string]any)
	}
	e.metadata[key] = value
	e.updatedAt = time.Now()
	e.version++
}

// GetMetadata gets a metadata value
func (e *Entitlement) GetMetadata(key string) (any, bool) {
	if e.metadata == nil {
		return nil, false
	}
	value, exists := e.metadata[key]
	return value, exists
}

// Validate performs domain-level validation
func (e *Entitlement) Validate() error {
	if !e.subjectType.IsValid() {
		return fmt.Errorf("invalid subject type: %s", e.subjectType)
	}
	if e.subjectID == 0 {
		return fmt.Errorf("subject ID is required")
	}
	if !e.resourceType.IsValid() {
		return fmt.Errorf("invalid resource type: %s", e.resourceType)
	}
	if e.resourceID == 0 {
		return fmt.Errorf("resource ID is required")
	}
	if !e.sourceType.IsValid() {
		return fmt.Errorf("invalid source type: %s", e.sourceType)
	}
	if e.sourceID == 0 {
		return fmt.Errorf("source ID is required")
	}
	if !e.status.IsValid() {
		return fmt.Errorf("invalid status: %s", e.status)
	}
	return nil
}
