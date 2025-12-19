package entitlement

import "context"

// Service defines the domain service interface for entitlement operations
// It provides high-level business logic operations that involve entitlements
type Service interface {
	// HasAccess checks if a user has access to a specific resource
	// It returns true if there is at least one active entitlement for the user-resource pair
	HasAccess(ctx context.Context, userID uint, resourceType ResourceType, resourceID uint) (bool, error)

	// GetUserEntitlements retrieves all entitlements for a user
	// This includes both active and inactive entitlements
	GetUserEntitlements(ctx context.Context, userID uint) ([]*Entitlement, error)

	// GetAccessibleResources retrieves all resource IDs that a user has access to for a given resource type
	// Only returns resources with active entitlements
	GetAccessibleResources(ctx context.Context, userID uint, resourceType ResourceType) ([]uint, error)
}
