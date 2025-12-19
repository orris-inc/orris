package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/entitlement/dto"
	"github.com/orris-inc/orris/internal/domain/entitlement"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GrantEntitlementUseCase handles the business logic for granting entitlements
// This is typically used for direct admin grants, promotions, or trials
type GrantEntitlementUseCase struct {
	entitlementRepo entitlement.Repository
	logger          logger.Interface
}

// NewGrantEntitlementUseCase creates a new grant entitlement use case
func NewGrantEntitlementUseCase(
	entitlementRepo entitlement.Repository,
	logger logger.Interface,
) *GrantEntitlementUseCase {
	return &GrantEntitlementUseCase{
		entitlementRepo: entitlementRepo,
		logger:          logger,
	}
}

// Execute executes the grant entitlement use case
func (uc *GrantEntitlementUseCase) Execute(
	ctx context.Context,
	request dto.GrantEntitlementRequest,
) (*dto.EntitlementResponse, error) {
	uc.logger.Infow("executing grant entitlement use case",
		"subject_type", request.SubjectType,
		"subject_id", request.SubjectID,
		"resource_type", request.ResourceType,
		"resource_id", request.ResourceID,
		"source_type", request.SourceType,
	)

	// Validate and convert subject type
	subjectType := entitlement.SubjectType(request.SubjectType)
	if !subjectType.IsValid() {
		uc.logger.Warnw("invalid subject type", "subject_type", request.SubjectType)
		return nil, errors.NewValidationError(fmt.Sprintf("invalid subject type: %s", request.SubjectType))
	}

	// Validate subject ID
	if request.SubjectID == 0 {
		uc.logger.Warnw("subject ID is required")
		return nil, errors.NewValidationError("subject ID is required")
	}

	// Validate and convert resource type
	resourceType := entitlement.ResourceType(request.ResourceType)
	if !resourceType.IsValid() {
		uc.logger.Warnw("invalid resource type", "resource_type", request.ResourceType)
		return nil, errors.NewValidationError(fmt.Sprintf("invalid resource type: %s", request.ResourceType))
	}

	// Validate resource ID
	if request.ResourceID == 0 {
		uc.logger.Warnw("resource ID is required")
		return nil, errors.NewValidationError("resource ID is required")
	}

	// Validate and convert source type
	sourceType := entitlement.SourceType(request.SourceType)
	if !sourceType.IsValid() {
		uc.logger.Warnw("invalid source type", "source_type", request.SourceType)
		return nil, errors.NewValidationError(fmt.Sprintf("invalid source type: %s", request.SourceType))
	}

	// Validate source ID
	if request.SourceID == 0 {
		uc.logger.Warnw("source ID is required")
		return nil, errors.NewValidationError("source ID is required")
	}

	// Parse expiration time if provided
	var expiresAt *time.Time
	if request.ExpiresAt != nil && *request.ExpiresAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, *request.ExpiresAt)
		if err != nil {
			uc.logger.Warnw("invalid expiration time format", "error", err, "expires_at", *request.ExpiresAt)
			return nil, errors.NewValidationError(fmt.Sprintf("invalid expiration time format: %s (expected RFC3339)", *request.ExpiresAt))
		}
		expiresAt = &parsedTime
	}

	// Check if entitlement already exists
	exists, err := uc.entitlementRepo.Exists(ctx, subjectType, request.SubjectID, resourceType, request.ResourceID)
	if err != nil {
		uc.logger.Errorw("failed to check if entitlement exists", "error", err)
		return nil, fmt.Errorf("failed to check existing entitlement: %w", err)
	}

	if exists {
		uc.logger.Warnw("entitlement already exists",
			"subject_type", request.SubjectType,
			"subject_id", request.SubjectID,
			"resource_type", request.ResourceType,
			"resource_id", request.ResourceID,
		)
		return nil, errors.NewConflictError("entitlement already exists for this subject-resource pair", "")
	}

	// Create new entitlement entity
	entitlementEntity, err := entitlement.NewEntitlement(
		subjectType,
		request.SubjectID,
		resourceType,
		request.ResourceID,
		sourceType,
		request.SourceID,
		expiresAt,
	)
	if err != nil {
		uc.logger.Errorw("failed to create entitlement entity", "error", err)
		return nil, fmt.Errorf("failed to create entitlement: %w", err)
	}

	// Set metadata if provided
	if request.Metadata != nil {
		for key, value := range request.Metadata {
			entitlementEntity.SetMetadata(key, value)
		}
	}

	// Persist the entitlement
	if err := uc.entitlementRepo.Create(ctx, entitlementEntity); err != nil {
		uc.logger.Errorw("failed to persist entitlement", "error", err)
		return nil, fmt.Errorf("failed to save entitlement: %w", err)
	}

	// Map to response DTO
	response := &dto.EntitlementResponse{
		ID:           entitlementEntity.ID(),
		SubjectType:  entitlementEntity.SubjectType().String(),
		SubjectID:    entitlementEntity.SubjectID(),
		ResourceType: entitlementEntity.ResourceType().String(),
		ResourceID:   entitlementEntity.ResourceID(),
		SourceType:   entitlementEntity.SourceType().String(),
		SourceID:     entitlementEntity.SourceID(),
		Status:       entitlementEntity.Status().String(),
		ExpiresAt:    entitlementEntity.ExpiresAt(),
		Metadata:     entitlementEntity.Metadata(),
		CreatedAt:    entitlementEntity.CreatedAt(),
		UpdatedAt:    entitlementEntity.UpdatedAt(),
	}

	uc.logger.Infow("entitlement granted successfully",
		"entitlement_id", response.ID,
		"subject_type", response.SubjectType,
		"subject_id", response.SubjectID,
		"resource_type", response.ResourceType,
		"resource_id", response.ResourceID,
	)

	return response, nil
}

// ValidateRequest validates the grant entitlement request
func (uc *GrantEntitlementUseCase) ValidateRequest(request dto.GrantEntitlementRequest) error {
	if request.SubjectType == "" {
		return errors.NewValidationError("subject type is required")
	}
	if request.SubjectID == 0 {
		return errors.NewValidationError("subject ID is required")
	}
	if request.ResourceType == "" {
		return errors.NewValidationError("resource type is required")
	}
	if request.ResourceID == 0 {
		return errors.NewValidationError("resource ID is required")
	}
	if request.SourceType == "" {
		return errors.NewValidationError("source type is required")
	}
	if request.SourceID == 0 {
		return errors.NewValidationError("source ID is required")
	}
	return nil
}
