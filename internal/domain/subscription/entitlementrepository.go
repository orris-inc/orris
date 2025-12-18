package subscription

import "context"

// EntitlementRepository defines the interface for entitlement persistence
type EntitlementRepository interface {
	// Create creates a new entitlement association
	Create(ctx context.Context, entitlement *Entitlement) error

	// Delete removes an entitlement association
	Delete(ctx context.Context, id uint) error

	// DeleteByPlanAndResource removes a specific association
	DeleteByPlanAndResource(ctx context.Context, planID uint, resourceType EntitlementResourceType, resourceID uint) error

	// GetByPlan returns all entitlements for a plan
	GetByPlan(ctx context.Context, planID uint) ([]*Entitlement, error)

	// GetByPlanAndType returns entitlements of a specific type for a plan
	GetByPlanAndType(ctx context.Context, planID uint, resourceType EntitlementResourceType) ([]*Entitlement, error)

	// GetResourceIDs returns resource IDs of a specific type for a plan
	GetResourceIDs(ctx context.Context, planID uint, resourceType EntitlementResourceType) ([]uint, error)

	// Exists checks if a specific association exists
	Exists(ctx context.Context, planID uint, resourceType EntitlementResourceType, resourceID uint) (bool, error)

	// BatchCreate creates multiple entitlement associations
	BatchCreate(ctx context.Context, entitlements []*Entitlement) error

	// BatchDelete removes multiple entitlement associations by IDs
	BatchDelete(ctx context.Context, ids []uint) error

	// DeleteAllByPlan removes all entitlements for a plan
	DeleteAllByPlan(ctx context.Context, planID uint) error

	// DeleteAllByResource removes all associations for a specific resource
	DeleteAllByResource(ctx context.Context, resourceType EntitlementResourceType, resourceID uint) error
}
