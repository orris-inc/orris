package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/entitlement/dto"
	"github.com/orris-inc/orris/internal/domain/entitlement"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetEntitlementUseCase handles the business logic for getting a single entitlement
type GetEntitlementUseCase struct {
	entitlementRepo entitlement.Repository
	logger          logger.Interface
}

// NewGetEntitlementUseCase creates a new get entitlement use case
func NewGetEntitlementUseCase(
	entitlementRepo entitlement.Repository,
	logger logger.Interface,
) *GetEntitlementUseCase {
	return &GetEntitlementUseCase{
		entitlementRepo: entitlementRepo,
		logger:          logger,
	}
}

// Execute executes the get entitlement use case
func (uc *GetEntitlementUseCase) Execute(
	ctx context.Context,
	id uint,
) (*dto.EntitlementResponse, error) {
	uc.logger.Infow("executing get entitlement use case", "entitlement_id", id)

	if id == 0 {
		uc.logger.Warnw("invalid entitlement ID", "entitlement_id", id)
		return nil, errors.NewValidationError("entitlement ID cannot be zero")
	}

	// Get entitlement from repository
	ent, err := uc.entitlementRepo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get entitlement", "error", err, "entitlement_id", id)
		return nil, fmt.Errorf("failed to get entitlement: %w", err)
	}

	if ent == nil {
		uc.logger.Warnw("entitlement not found", "entitlement_id", id)
		return nil, errors.NewNotFoundError("entitlement not found")
	}

	// Map to response DTO
	response := &dto.EntitlementResponse{
		ID:           ent.ID(),
		SubjectType:  ent.SubjectType().String(),
		SubjectID:    ent.SubjectID(),
		ResourceType: ent.ResourceType().String(),
		ResourceID:   ent.ResourceID(),
		SourceType:   ent.SourceType().String(),
		SourceID:     ent.SourceID(),
		Status:       ent.Status().String(),
		ExpiresAt:    ent.ExpiresAt(),
		Metadata:     ent.Metadata(),
		CreatedAt:    ent.CreatedAt(),
		UpdatedAt:    ent.UpdatedAt(),
	}

	uc.logger.Infow("entitlement retrieved successfully", "entitlement_id", id)

	return response, nil
}
