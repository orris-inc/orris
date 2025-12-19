package entitlement

import "context"

// Repository defines the interface for entitlement persistence operations
type Repository interface {
	// Create creates a new entitlement
	Create(ctx context.Context, e *Entitlement) error

	// Update updates an existing entitlement
	Update(ctx context.Context, e *Entitlement) error

	// Delete deletes an entitlement by ID
	Delete(ctx context.Context, id uint) error

	// GetByID retrieves an entitlement by ID
	GetByID(ctx context.Context, id uint) (*Entitlement, error)

	// GetBySubject retrieves all entitlements for a subject
	GetBySubject(ctx context.Context, subjectType SubjectType, subjectID uint) ([]*Entitlement, error)

	// GetActiveBySubject retrieves all active entitlements for a subject
	// Active means status is active and not expired
	GetActiveBySubject(ctx context.Context, subjectType SubjectType, subjectID uint) ([]*Entitlement, error)

	// GetByResource retrieves all entitlements for a specific resource
	GetByResource(ctx context.Context, resourceType ResourceType, resourceID uint) ([]*Entitlement, error)

	// GetBySource retrieves all entitlements from a specific source
	GetBySource(ctx context.Context, sourceType SourceType, sourceID uint) ([]*Entitlement, error)

	// Exists checks if an entitlement exists for a subject-resource pair
	Exists(ctx context.Context, subjectType SubjectType, subjectID uint,
		resourceType ResourceType, resourceID uint) (bool, error)

	// BatchCreate creates multiple entitlements in a single transaction
	BatchCreate(ctx context.Context, entitlements []*Entitlement) error

	// BatchUpdateStatus updates the status of multiple entitlements
	BatchUpdateStatus(ctx context.Context, ids []uint, status EntitlementStatus) error

	// RevokeBySource revokes all entitlements from a specific source
	// This is useful when a subscription is cancelled or a promotion ends
	RevokeBySource(ctx context.Context, sourceType SourceType, sourceID uint) error

	// GetExpiredEntitlements retrieves all entitlements that have passed their expiration time
	// but haven't been marked as expired yet
	GetExpiredEntitlements(ctx context.Context) ([]*Entitlement, error)
}
