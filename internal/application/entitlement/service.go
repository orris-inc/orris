package entitlement

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/entitlement"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ServiceImpl implements the entitlement.Service interface
// It provides domain service operations for entitlement management
type ServiceImpl struct {
	entitlementRepo entitlement.Repository
	logger          logger.Interface
}

// NewService creates a new entitlement service implementation
func NewService(
	entitlementRepo entitlement.Repository,
	logger logger.Interface,
) *ServiceImpl {
	return &ServiceImpl{
		entitlementRepo: entitlementRepo,
		logger:          logger,
	}
}

// HasAccess checks if a user has access to a specific resource
// It queries for active entitlements owned directly by the user
func (s *ServiceImpl) HasAccess(
	ctx context.Context,
	userID uint,
	resourceType entitlement.ResourceType,
	resourceID uint,
) (bool, error) {
	s.logger.Debugw("checking user access to resource",
		"user_id", userID,
		"resource_type", resourceType,
		"resource_id", resourceID,
	)

	// Get all active entitlements for the user
	entitlements, err := s.entitlementRepo.GetActiveBySubject(
		ctx,
		entitlement.SubjectTypeUser,
		userID,
	)
	if err != nil {
		s.logger.Errorw("failed to get user active entitlements",
			"error", err,
			"user_id", userID,
		)
		return false, err
	}

	// Check if any active entitlement matches the resource
	for _, e := range entitlements {
		if e.ResourceType() == resourceType && e.ResourceID() == resourceID {
			if e.IsActive() {
				s.logger.Debugw("user has access to resource",
					"user_id", userID,
					"resource_type", resourceType,
					"resource_id", resourceID,
					"entitlement_id", e.ID(),
				)
				return true, nil
			}
		}
	}

	s.logger.Debugw("user does not have access to resource",
		"user_id", userID,
		"resource_type", resourceType,
		"resource_id", resourceID,
	)
	return false, nil
}

// GetUserEntitlements retrieves all entitlements for a user
// This includes both active and inactive entitlements
func (s *ServiceImpl) GetUserEntitlements(
	ctx context.Context,
	userID uint,
) ([]*entitlement.Entitlement, error) {
	s.logger.Debugw("getting user entitlements", "user_id", userID)

	entitlements, err := s.entitlementRepo.GetBySubject(
		ctx,
		entitlement.SubjectTypeUser,
		userID,
	)
	if err != nil {
		s.logger.Errorw("failed to get user entitlements",
			"error", err,
			"user_id", userID,
		)
		return nil, err
	}

	s.logger.Debugw("retrieved user entitlements",
		"user_id", userID,
		"count", len(entitlements),
	)
	return entitlements, nil
}

// GetAccessibleResources retrieves all resource IDs that a user has access to
// for a given resource type. Only returns resources with active entitlements.
func (s *ServiceImpl) GetAccessibleResources(
	ctx context.Context,
	userID uint,
	resourceType entitlement.ResourceType,
) ([]uint, error) {
	s.logger.Debugw("getting accessible resources for user",
		"user_id", userID,
		"resource_type", resourceType,
	)

	// Get all active entitlements for the user
	entitlements, err := s.entitlementRepo.GetActiveBySubject(
		ctx,
		entitlement.SubjectTypeUser,
		userID,
	)
	if err != nil {
		s.logger.Errorw("failed to get user active entitlements",
			"error", err,
			"user_id", userID,
		)
		return nil, err
	}

	// Filter by resource type and collect unique resource IDs
	resourceIDsMap := make(map[uint]bool)
	for _, e := range entitlements {
		if e.ResourceType() == resourceType && e.IsActive() {
			resourceIDsMap[e.ResourceID()] = true
		}
	}

	// Convert map to slice
	resourceIDs := make([]uint, 0, len(resourceIDsMap))
	for id := range resourceIDsMap {
		resourceIDs = append(resourceIDs, id)
	}

	s.logger.Debugw("retrieved accessible resources",
		"user_id", userID,
		"resource_type", resourceType,
		"count", len(resourceIDs),
	)
	return resourceIDs, nil
}
