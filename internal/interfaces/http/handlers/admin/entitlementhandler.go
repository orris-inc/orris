// Package admin provides HTTP handlers for administrative operations.
package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/entitlement/dto"
	"github.com/orris-inc/orris/internal/application/entitlement/usecases"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AdminEntitlementHandler handles admin entitlement operations
type AdminEntitlementHandler struct {
	grantUC  *usecases.GrantEntitlementUseCase
	revokeUC *usecases.RevokeEntitlementUseCase
	listUC   *usecases.ListEntitlementsUseCase
	getUC    *usecases.GetEntitlementUseCase
	logger   logger.Interface
}

// NewAdminEntitlementHandler creates a new admin entitlement handler
func NewAdminEntitlementHandler(
	grantUC *usecases.GrantEntitlementUseCase,
	revokeUC *usecases.RevokeEntitlementUseCase,
	listUC *usecases.ListEntitlementsUseCase,
	getUC *usecases.GetEntitlementUseCase,
	logger logger.Interface,
) *AdminEntitlementHandler {
	return &AdminEntitlementHandler{
		grantUC:  grantUC,
		revokeUC: revokeUC,
		listUC:   listUC,
		getUC:    getUC,
		logger:   logger,
	}
}

// GrantEntitlementRequest represents the request to grant an entitlement
type GrantEntitlementRequest struct {
	SubjectType  string                 `json:"subject_type" binding:"required"`
	SubjectID    uint                   `json:"subject_id" binding:"required"`
	ResourceType string                 `json:"resource_type" binding:"required"`
	ResourceID   uint                   `json:"resource_id" binding:"required"`
	SourceType   string                 `json:"source_type" binding:"required"`
	SourceID     uint                   `json:"source_id" binding:"required"`
	ExpiresAt    *string                `json:"expires_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// GrantEntitlement handles POST /admin/entitlements
// Grant entitlement directly (for promotions, trials, etc.)
func (h *AdminEntitlementHandler) GrantEntitlement(c *gin.Context) {
	var req GrantEntitlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for grant entitlement", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Convert to use case request
	ucReq := dto.GrantEntitlementRequest{
		SubjectType:  req.SubjectType,
		SubjectID:    req.SubjectID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		SourceType:   req.SourceType,
		SourceID:     req.SourceID,
		ExpiresAt:    req.ExpiresAt,
		Metadata:     req.Metadata,
	}

	// Execute use case
	result, err := h.grantUC.Execute(c.Request.Context(), ucReq)
	if err != nil {
		h.logger.Errorw("failed to grant entitlement",
			"error", err,
			"subject_type", req.SubjectType,
			"subject_id", req.SubjectID,
			"resource_type", req.ResourceType,
			"resource_id", req.ResourceID,
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Entitlement granted successfully")
}

// RevokeEntitlement handles DELETE /admin/entitlements/:id
// Revoke an entitlement
func (h *AdminEntitlementHandler) RevokeEntitlement(c *gin.Context) {
	// Parse entitlement ID
	entitlementID, err := parseEntitlementID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Execute use case
	ucReq := dto.RevokeEntitlementRequest{
		EntitlementID: entitlementID,
	}

	result, err := h.revokeUC.Execute(c.Request.Context(), ucReq)
	if err != nil {
		h.logger.Errorw("failed to revoke entitlement",
			"error", err,
			"entitlement_id", entitlementID,
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Entitlement revoked successfully", result)
}

// GetEntitlement handles GET /admin/entitlements/:id
// Get a single entitlement by ID
func (h *AdminEntitlementHandler) GetEntitlement(c *gin.Context) {
	// Parse entitlement ID
	entitlementID, err := parseEntitlementID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Execute use case
	result, err := h.getUC.Execute(c.Request.Context(), entitlementID)
	if err != nil {
		h.logger.Errorw("failed to get entitlement",
			"error", err,
			"entitlement_id", entitlementID,
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Entitlement retrieved successfully", result)
}

// ListEntitlements handles GET /admin/entitlements
// Query entitlements (with filtering support)
// Query parameters:
//   - subject_type: filter by subject type
//   - subject_id: filter by subject ID
//   - resource_type: filter by resource type
//   - resource_id: filter by resource ID
//   - status: filter by status (active, expired, revoked)
//   - source_type: filter by source type
//   - page: page number (default: 1)
//   - page_size: items per page (default: 20)
func (h *AdminEntitlementHandler) ListEntitlements(c *gin.Context) {
	// Parse query parameters
	req, err := parseListEntitlementsQuery(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Execute use case
	result, err := h.listUC.Execute(c.Request.Context(), *req)
	if err != nil {
		h.logger.Errorw("failed to list entitlements", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Entitlements, int64(result.Pagination.Total), result.Pagination.Page, result.Pagination.PageSize)
}

// parseEntitlementID parses the entitlement ID from the URL parameter
func parseEntitlementID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	if idStr == "" {
		return 0, errors.NewValidationError("Entitlement ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid entitlement ID format")
	}

	if id == 0 {
		return 0, errors.NewValidationError("Entitlement ID cannot be zero")
	}

	return uint(id), nil
}

// parseListEntitlementsQuery parses query parameters for listing entitlements
func parseListEntitlementsQuery(c *gin.Context) (*dto.ListEntitlementsRequest, error) {
	req := &dto.ListEntitlementsRequest{
		Page:     constants.DefaultPage,
		PageSize: constants.DefaultPageSize,
	}

	// Parse pagination
	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return nil, errors.NewValidationError("Invalid page parameter")
		}
		req.Page = page
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 {
			return nil, errors.NewValidationError("Invalid page_size parameter")
		}
		if pageSize > constants.MaxPageSize {
			pageSize = constants.MaxPageSize
		}
		req.PageSize = pageSize
	}

	// Parse filters
	if subjectType := c.Query("subject_type"); subjectType != "" {
		req.SubjectType = &subjectType
	}

	if subjectIDStr := c.Query("subject_id"); subjectIDStr != "" {
		subjectID, err := strconv.ParseUint(subjectIDStr, 10, 32)
		if err != nil {
			return nil, errors.NewValidationError("Invalid subject_id parameter")
		}
		uid := uint(subjectID)
		req.SubjectID = &uid
	}

	if resourceType := c.Query("resource_type"); resourceType != "" {
		req.ResourceType = &resourceType
	}

	if resourceIDStr := c.Query("resource_id"); resourceIDStr != "" {
		resourceID, err := strconv.ParseUint(resourceIDStr, 10, 32)
		if err != nil {
			return nil, errors.NewValidationError("Invalid resource_id parameter")
		}
		rid := uint(resourceID)
		req.ResourceID = &rid
	}

	if status := c.Query("status"); status != "" {
		req.Status = &status
	}

	if sourceType := c.Query("source_type"); sourceType != "" {
		req.SourceType = &sourceType
	}

	return req, nil
}
