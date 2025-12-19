package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/entitlement/dto"
	"github.com/orris-inc/orris/internal/domain/entitlement"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetUserEntitlementsUseCase handles the business logic for getting user entitlements
type GetUserEntitlementsUseCase struct {
	entitlementService entitlement.Service
	logger             logger.Interface
}

// NewGetUserEntitlementsUseCase creates a new get user entitlements use case
func NewGetUserEntitlementsUseCase(
	entitlementService entitlement.Service,
	logger logger.Interface,
) *GetUserEntitlementsUseCase {
	return &GetUserEntitlementsUseCase{
		entitlementService: entitlementService,
		logger:             logger,
	}
}

// Execute executes the get user entitlements use case
func (uc *GetUserEntitlementsUseCase) Execute(
	ctx context.Context,
	userID uint,
) ([]*dto.EntitlementResponse, error) {
	uc.logger.Infow("executing get user entitlements use case", "user_id", userID)

	// Validate user ID
	if userID == 0 {
		uc.logger.Warnw("user ID is required")
		return nil, errors.NewValidationError("user ID is required")
	}

	// Get user entitlements from service
	entitlements, err := uc.entitlementService.GetUserEntitlements(ctx, userID)
	if err != nil {
		uc.logger.Errorw("failed to get user entitlements", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to get user entitlements: %w", err)
	}

	// Map to response DTOs
	responses := make([]*dto.EntitlementResponse, len(entitlements))
	for i, e := range entitlements {
		responses[i] = &dto.EntitlementResponse{
			ID:           e.ID(),
			SubjectType:  e.SubjectType().String(),
			SubjectID:    e.SubjectID(),
			ResourceType: e.ResourceType().String(),
			ResourceID:   e.ResourceID(),
			SourceType:   e.SourceType().String(),
			SourceID:     e.SourceID(),
			Status:       e.Status().String(),
			ExpiresAt:    e.ExpiresAt(),
			Metadata:     e.Metadata(),
			CreatedAt:    e.CreatedAt(),
			UpdatedAt:    e.UpdatedAt(),
		}
	}

	uc.logger.Infow("user entitlements retrieved successfully",
		"user_id", userID,
		"count", len(responses),
	)

	return responses, nil
}

// ExecuteAccessibleResources gets accessible resource IDs for a user and resource type
func (uc *GetUserEntitlementsUseCase) ExecuteAccessibleResources(
	ctx context.Context,
	userID uint,
	resourceTypeStr string,
) (*dto.AccessibleResourcesResponse, error) {
	uc.logger.Infow("executing get accessible resources use case",
		"user_id", userID,
		"resource_type", resourceTypeStr,
	)

	// Validate user ID
	if userID == 0 {
		uc.logger.Warnw("user ID is required")
		return nil, errors.NewValidationError("user ID is required")
	}

	// Validate and convert resource type
	resourceType := entitlement.ResourceType(resourceTypeStr)
	if !resourceType.IsValid() {
		uc.logger.Warnw("invalid resource type", "resource_type", resourceTypeStr)
		return nil, errors.NewValidationError(fmt.Sprintf("invalid resource type: %s", resourceTypeStr))
	}

	// Get accessible resources from service
	resourceIDs, err := uc.entitlementService.GetAccessibleResources(ctx, userID, resourceType)
	if err != nil {
		uc.logger.Errorw("failed to get accessible resources",
			"error", err,
			"user_id", userID,
			"resource_type", resourceTypeStr,
		)
		return nil, fmt.Errorf("failed to get accessible resources: %w", err)
	}

	response := &dto.AccessibleResourcesResponse{
		ResourceType: resourceTypeStr,
		ResourceIDs:  resourceIDs,
	}

	uc.logger.Infow("accessible resources retrieved successfully",
		"user_id", userID,
		"resource_type", resourceTypeStr,
		"count", len(resourceIDs),
	)

	return response, nil
}
