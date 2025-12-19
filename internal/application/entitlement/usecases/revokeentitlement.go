package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/entitlement/dto"
	"github.com/orris-inc/orris/internal/domain/entitlement"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// RevokeEntitlementUseCase handles the business logic for revoking entitlements
type RevokeEntitlementUseCase struct {
	entitlementRepo entitlement.Repository
	logger          logger.Interface
}

// NewRevokeEntitlementUseCase creates a new revoke entitlement use case
func NewRevokeEntitlementUseCase(
	entitlementRepo entitlement.Repository,
	logger logger.Interface,
) *RevokeEntitlementUseCase {
	return &RevokeEntitlementUseCase{
		entitlementRepo: entitlementRepo,
		logger:          logger,
	}
}

// Execute executes the revoke entitlement use case
func (uc *RevokeEntitlementUseCase) Execute(
	ctx context.Context,
	request dto.RevokeEntitlementRequest,
) (*dto.EntitlementResponse, error) {
	uc.logger.Infow("executing revoke entitlement use case",
		"entitlement_id", request.EntitlementID,
	)

	// Validate entitlement ID
	if request.EntitlementID == 0 {
		uc.logger.Warnw("entitlement ID is required")
		return nil, errors.NewValidationError("entitlement ID is required")
	}

	// Get the entitlement
	entitlementEntity, err := uc.entitlementRepo.GetByID(ctx, request.EntitlementID)
	if err != nil {
		uc.logger.Errorw("failed to get entitlement", "error", err, "entitlement_id", request.EntitlementID)
		return nil, fmt.Errorf("failed to get entitlement: %w", err)
	}

	if entitlementEntity == nil {
		uc.logger.Warnw("entitlement not found", "entitlement_id", request.EntitlementID)
		return nil, errors.NewNotFoundError(fmt.Sprintf("entitlement with ID %d not found", request.EntitlementID))
	}

	// Revoke the entitlement
	if err := entitlementEntity.Revoke(); err != nil {
		uc.logger.Errorw("failed to revoke entitlement", "error", err, "entitlement_id", request.EntitlementID)
		return nil, fmt.Errorf("failed to revoke entitlement: %w", err)
	}

	// Update the entitlement in the repository
	if err := uc.entitlementRepo.Update(ctx, entitlementEntity); err != nil {
		uc.logger.Errorw("failed to update entitlement", "error", err, "entitlement_id", request.EntitlementID)
		return nil, fmt.Errorf("failed to update entitlement: %w", err)
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

	uc.logger.Infow("entitlement revoked successfully",
		"entitlement_id", response.ID,
		"status", response.Status,
	)

	return response, nil
}

// ValidateRequest validates the revoke entitlement request
func (uc *RevokeEntitlementUseCase) ValidateRequest(request dto.RevokeEntitlementRequest) error {
	if request.EntitlementID == 0 {
		return errors.NewValidationError("entitlement ID is required")
	}
	return nil
}
