package entitlement

import (
	"errors"
	"fmt"
)

var (
	// ErrEntitlementNotFound is returned when an entitlement is not found
	ErrEntitlementNotFound = errors.New("entitlement not found")

	// ErrEntitlementExpired is returned when an entitlement has expired
	ErrEntitlementExpired = errors.New("entitlement expired")

	// ErrEntitlementRevoked is returned when an entitlement has been revoked
	ErrEntitlementRevoked = errors.New("entitlement revoked")

	// ErrEntitlementInactive is returned when an entitlement is not active
	ErrEntitlementInactive = errors.New("entitlement inactive")

	// ErrInvalidSubjectType is returned when an invalid subject type is provided
	ErrInvalidSubjectType = errors.New("invalid subject type")

	// ErrInvalidResourceType is returned when an invalid resource type is provided
	ErrInvalidResourceType = errors.New("invalid resource type")

	// ErrInvalidSourceType is returned when an invalid source type is provided
	ErrInvalidSourceType = errors.New("invalid source type")

	// ErrInvalidStatus is returned when an invalid entitlement status is provided
	ErrInvalidStatus = errors.New("invalid entitlement status")

	// ErrSubjectIDRequired is returned when subject ID is missing
	ErrSubjectIDRequired = errors.New("subject ID is required")

	// ErrResourceIDRequired is returned when resource ID is missing
	ErrResourceIDRequired = errors.New("resource ID is required")

	// ErrSourceIDRequired is returned when source ID is missing
	ErrSourceIDRequired = errors.New("source ID is required")

	// ErrDuplicateEntitlement is returned when an entitlement already exists
	ErrDuplicateEntitlement = errors.New("entitlement already exists")

	// ErrAccessDenied is returned when a user does not have access to a resource
	ErrAccessDenied = errors.New("access denied")
)

// ErrInvalidStatusTransition returns an error for invalid status transitions
func ErrInvalidStatusTransition(from, to EntitlementStatus) error {
	return fmt.Errorf("invalid status transition from %s to %s", from, to)
}

// ErrResourceAccessDenied returns an error when access to a specific resource is denied
func ErrResourceAccessDenied(resourceType ResourceType, resourceID uint) error {
	return fmt.Errorf("%w: %s (ID: %d)", ErrAccessDenied, resourceType, resourceID)
}
