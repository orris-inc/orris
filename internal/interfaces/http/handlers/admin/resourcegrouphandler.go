// Package admin provides HTTP handlers for administrative operations.
package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/application/resource/usecases"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ResourceGroupHandler handles admin resource group operations
type ResourceGroupHandler struct {
	createUseCase       *usecases.CreateResourceGroupUseCase
	getUseCase          *usecases.GetResourceGroupUseCase
	listUseCase         *usecases.ListResourceGroupsUseCase
	updateUseCase       *usecases.UpdateResourceGroupUseCase
	deleteUseCase       *usecases.DeleteResourceGroupUseCase
	updateStatusUseCase *usecases.UpdateResourceGroupStatusUseCase
	logger              logger.Interface
}

// NewResourceGroupHandler creates a new admin resource group handler
func NewResourceGroupHandler(
	createUC *usecases.CreateResourceGroupUseCase,
	getUC *usecases.GetResourceGroupUseCase,
	listUC *usecases.ListResourceGroupsUseCase,
	updateUC *usecases.UpdateResourceGroupUseCase,
	deleteUC *usecases.DeleteResourceGroupUseCase,
	updateStatusUC *usecases.UpdateResourceGroupStatusUseCase,
	logger logger.Interface,
) *ResourceGroupHandler {
	return &ResourceGroupHandler{
		createUseCase:       createUC,
		getUseCase:          getUC,
		listUseCase:         listUC,
		updateUseCase:       updateUC,
		deleteUseCase:       deleteUC,
		updateStatusUseCase: updateStatusUC,
		logger:              logger,
	}
}

// CreateRequest represents the request to create a resource group
type CreateResourceGroupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	PlanID      uint   `json:"plan_id" binding:"required"`
	Description string `json:"description" binding:"max=500"`
}

// UpdateResourceGroupRequest represents the request to update a resource group
type UpdateResourceGroupRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
}

// Create creates a new resource group
func (h *ResourceGroupHandler) Create(c *gin.Context) {
	var req CreateResourceGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create resource group", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	dtoReq := dto.CreateResourceGroupRequest{
		Name:        req.Name,
		PlanID:      req.PlanID,
		Description: req.Description,
	}

	result, err := h.createUseCase.Execute(c.Request.Context(), dtoReq)
	if err != nil {
		if err == resource.ErrGroupNameExists {
			utils.ErrorResponse(c, http.StatusConflict, "resource group name already exists")
			return
		}
		h.logger.Errorw("failed to create resource group", "error", err, "name", req.Name)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Resource group created successfully")
}

// List lists resource groups with pagination
func (h *ResourceGroupHandler) List(c *gin.Context) {
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := constants.DefaultPageSize
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= constants.MaxPageSize {
			pageSize = ps
		}
	}

	var status *string
	if statusStr := c.Query("status"); statusStr != "" {
		status = &statusStr
	}

	var planID *uint
	if planIDStr := c.Query("plan_id"); planIDStr != "" {
		if pid, err := strconv.ParseUint(planIDStr, 10, 64); err == nil {
			pidVal := uint(pid)
			planID = &pidVal
		}
	}

	query := dto.ListResourceGroupsRequest{
		PlanID:   planID,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.listUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to list resource groups", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Items, int64(result.Total), result.Page, result.PageSize)
}

// Get retrieves a resource group by ID
func (h *ResourceGroupHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID")
		return
	}

	result, err := h.getUseCase.ExecuteByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to get resource group", "error", err, "id", id)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetBySID retrieves a resource group by Stripe-style ID
func (h *ResourceGroupHandler) GetBySID(c *gin.Context) {
	sid := c.Param("sid")
	if sid == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group SID")
		return
	}

	result, err := h.getUseCase.ExecuteBySID(c.Request.Context(), sid)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to get resource group by SID", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// Update updates a resource group
func (h *ResourceGroupHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID")
		return
	}

	var req UpdateResourceGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update resource group", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	dtoReq := dto.UpdateResourceGroupRequest{
		Name:        req.Name,
		Description: req.Description,
	}

	result, err := h.updateUseCase.Execute(c.Request.Context(), uint(id), dtoReq)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		if err == resource.ErrGroupNameExists {
			utils.ErrorResponse(c, http.StatusConflict, "resource group name already exists")
			return
		}
		h.logger.Errorw("failed to update resource group", "error", err, "id", id)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Resource group updated successfully", result)
}

// Delete deletes a resource group
func (h *ResourceGroupHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID")
		return
	}

	if err := h.deleteUseCase.Execute(c.Request.Context(), uint(id)); err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		if err == resource.ErrGroupHasResources {
			utils.ErrorResponse(c, http.StatusConflict, "resource group has associated resources")
			return
		}
		h.logger.Errorw("failed to delete resource group", "error", err, "id", id)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// Activate activates a resource group
func (h *ResourceGroupHandler) Activate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID")
		return
	}

	result, err := h.updateStatusUseCase.Activate(c.Request.Context(), uint(id))
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to activate resource group", "error", err, "id", id)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Resource group activated successfully", result)
}

// Deactivate deactivates a resource group
func (h *ResourceGroupHandler) Deactivate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID")
		return
	}

	result, err := h.updateStatusUseCase.Deactivate(c.Request.Context(), uint(id))
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to deactivate resource group", "error", err, "id", id)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Resource group deactivated successfully", result)
}
