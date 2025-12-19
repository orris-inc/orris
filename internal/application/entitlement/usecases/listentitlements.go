package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/entitlement/dto"
	"github.com/orris-inc/orris/internal/domain/entitlement"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ListEntitlementsUseCase handles the business logic for listing entitlements
type ListEntitlementsUseCase struct {
	entitlementRepo entitlement.Repository
	logger          logger.Interface
}

// NewListEntitlementsUseCase creates a new list entitlements use case
func NewListEntitlementsUseCase(
	entitlementRepo entitlement.Repository,
	logger logger.Interface,
) *ListEntitlementsUseCase {
	return &ListEntitlementsUseCase{
		entitlementRepo: entitlementRepo,
		logger:          logger,
	}
}

// Execute executes the list entitlements use case
func (uc *ListEntitlementsUseCase) Execute(
	ctx context.Context,
	request dto.ListEntitlementsRequest,
) (*dto.ListEntitlementsResponse, error) {
	uc.logger.Infow("executing list entitlements use case",
		"page", request.Page,
		"page_size", request.PageSize,
	)

	var entitlements []*entitlement.Entitlement
	var err error

	// Query based on provided filters
	// Priority: subject > resource > all
	if request.SubjectType != nil && request.SubjectID != nil {
		// Filter by subject
		subjectType := entitlement.SubjectType(*request.SubjectType)
		if !subjectType.IsValid() {
			uc.logger.Warnw("invalid subject type", "subject_type", *request.SubjectType)
			return nil, errors.NewValidationError(fmt.Sprintf("invalid subject type: %s", *request.SubjectType))
		}

		// Check if status filter is "active"
		if request.Status != nil && *request.Status == "active" {
			entitlements, err = uc.entitlementRepo.GetActiveBySubject(ctx, subjectType, *request.SubjectID)
		} else {
			entitlements, err = uc.entitlementRepo.GetBySubject(ctx, subjectType, *request.SubjectID)
		}
		if err != nil {
			uc.logger.Errorw("failed to get entitlements by subject", "error", err)
			return nil, fmt.Errorf("failed to get entitlements: %w", err)
		}
	} else if request.ResourceType != nil && request.ResourceID != nil {
		// Filter by resource
		resourceType := entitlement.ResourceType(*request.ResourceType)
		if !resourceType.IsValid() {
			uc.logger.Warnw("invalid resource type", "resource_type", *request.ResourceType)
			return nil, errors.NewValidationError(fmt.Sprintf("invalid resource type: %s", *request.ResourceType))
		}

		entitlements, err = uc.entitlementRepo.GetByResource(ctx, resourceType, *request.ResourceID)
		if err != nil {
			uc.logger.Errorw("failed to get entitlements by resource", "error", err)
			return nil, fmt.Errorf("failed to get entitlements: %w", err)
		}
	} else {
		// No specific filter provided - this would require a GetAll method
		// For now, return validation error requiring at least one filter
		uc.logger.Warnw("no filter provided for listing entitlements")
		return nil, errors.NewValidationError("at least one filter (subject or resource) is required")
	}

	// Apply status filter if specified (for non-active queries)
	if request.Status != nil && *request.Status != "active" {
		entitlements = uc.filterByStatus(entitlements, *request.Status)
	}

	// Apply source type filter if specified
	if request.SourceType != nil {
		entitlements = uc.filterBySourceType(entitlements, *request.SourceType)
	}

	// Apply pagination
	total := len(entitlements)
	pageSize := request.PageSize
	if pageSize <= 0 {
		pageSize = 20 // Default page size
	}
	page := request.Page
	if page <= 0 {
		page = 1 // Default to first page
	}

	totalPages := (total + pageSize - 1) / pageSize
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		start = total
		end = total
	} else if end > total {
		end = total
	}

	paginatedEntitlements := entitlements[start:end]

	// Map to response DTOs
	responses := make([]*dto.EntitlementResponse, len(paginatedEntitlements))
	for i, e := range paginatedEntitlements {
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

	response := &dto.ListEntitlementsResponse{
		Entitlements: responses,
		Pagination: dto.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	uc.logger.Infow("entitlements listed successfully",
		"total", total,
		"page", page,
		"page_size", pageSize,
		"returned", len(responses),
	)

	return response, nil
}

// filterByStatus filters entitlements by status
func (uc *ListEntitlementsUseCase) filterByStatus(entitlements []*entitlement.Entitlement, status string) []*entitlement.Entitlement {
	filtered := make([]*entitlement.Entitlement, 0)
	for _, e := range entitlements {
		if e.Status().String() == status {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// filterBySourceType filters entitlements by source type
func (uc *ListEntitlementsUseCase) filterBySourceType(entitlements []*entitlement.Entitlement, sourceType string) []*entitlement.Entitlement {
	filtered := make([]*entitlement.Entitlement, 0)
	for _, e := range entitlements {
		if e.SourceType().String() == sourceType {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// ValidateRequest validates the list entitlements request
func (uc *ListEntitlementsUseCase) ValidateRequest(request dto.ListEntitlementsRequest) error {
	// At least one filter is required
	hasSubjectFilter := request.SubjectType != nil && request.SubjectID != nil
	hasResourceFilter := request.ResourceType != nil && request.ResourceID != nil

	if !hasSubjectFilter && !hasResourceFilter {
		return errors.NewValidationError("at least one filter (subject or resource) is required")
	}

	// Validate status if provided
	if request.Status != nil {
		status := entitlement.EntitlementStatus(*request.Status)
		if !status.IsValid() {
			return errors.NewValidationError(fmt.Sprintf("invalid status: %s", *request.Status))
		}
	}

	return nil
}
