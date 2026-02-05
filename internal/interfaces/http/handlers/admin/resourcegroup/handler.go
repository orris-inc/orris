// Package resourcegroup provides HTTP handlers for admin resource group operations.
package resourcegroup

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/application/resource/usecases"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// Handler handles admin resource group operations
type Handler struct {
	createUseCase       *usecases.CreateResourceGroupUseCase
	getUseCase          *usecases.GetResourceGroupUseCase
	listUseCase         *usecases.ListResourceGroupsUseCase
	updateUseCase       *usecases.UpdateResourceGroupUseCase
	deleteUseCase       *usecases.DeleteResourceGroupUseCase
	updateStatusUseCase *usecases.UpdateResourceGroupStatusUseCase
	manageNodesUseCase  *usecases.ManageResourceGroupNodesUseCase
	manageAgentsUseCase *usecases.ManageResourceGroupForwardAgentsUseCase
	manageRulesUseCase  *usecases.ManageResourceGroupForwardRulesUseCase
	planRepo            subscription.PlanRepository
	logger              logger.Interface
}

// NewHandler creates a new admin resource group handler
func NewHandler(
	createUC *usecases.CreateResourceGroupUseCase,
	getUC *usecases.GetResourceGroupUseCase,
	listUC *usecases.ListResourceGroupsUseCase,
	updateUC *usecases.UpdateResourceGroupUseCase,
	deleteUC *usecases.DeleteResourceGroupUseCase,
	updateStatusUC *usecases.UpdateResourceGroupStatusUseCase,
	manageNodesUC *usecases.ManageResourceGroupNodesUseCase,
	manageAgentsUC *usecases.ManageResourceGroupForwardAgentsUseCase,
	manageRulesUC *usecases.ManageResourceGroupForwardRulesUseCase,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *Handler {
	return &Handler{
		createUseCase:       createUC,
		getUseCase:          getUC,
		listUseCase:         listUC,
		updateUseCase:       updateUC,
		deleteUseCase:       deleteUC,
		updateStatusUseCase: updateStatusUC,
		manageNodesUseCase:  manageNodesUC,
		manageAgentsUseCase: manageAgentsUC,
		manageRulesUseCase:  manageRulesUC,
		planRepo:            planRepo,
		logger:              logger,
	}
}

// CreateResourceGroupRequest represents the request to create a resource group
type CreateResourceGroupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	PlanSID     string `json:"plan_id" binding:"required"`
	Description string `json:"description" binding:"max=500"`
}

// UpdateResourceGroupRequest represents the request to update a resource group
type UpdateResourceGroupRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
}

// Create creates a new resource group
func (h *Handler) Create(c *gin.Context) {
	var req CreateResourceGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create resource group", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	dtoReq := dto.CreateResourceGroupRequest{
		Name:        req.Name,
		PlanSID:     req.PlanSID,
		Description: req.Description,
	}

	result, err := h.createUseCase.Execute(c.Request.Context(), dtoReq)
	if err != nil {
		if err == resource.ErrGroupNameExists {
			utils.ErrorResponse(c, http.StatusConflict, "resource group name already exists")
			return
		}
		if err == subscription.ErrPlanNotFound {
			utils.ErrorResponse(c, http.StatusBadRequest, "plan not found")
			return
		}
		h.logger.Errorw("failed to create resource group", "error", err, "name", req.Name)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Resource group created successfully")
}

// List lists resource groups with pagination
func (h *Handler) List(c *gin.Context) {
	p := utils.ParsePagination(c)

	var status *string
	if statusStr := c.Query("status"); statusStr != "" {
		status = &statusStr
	}

	var planSID *string
	if planIDStr := c.Query("plan_id"); planIDStr != "" {
		planSID = &planIDStr
	}

	query := dto.ListResourceGroupsRequest{
		PlanSID:  planSID,
		Status:   status,
		Page:     p.Page,
		PageSize: p.PageSize,
	}

	result, err := h.listUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to list resource groups", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Items, int64(result.Total), result.Page, result.PageSize)
}

// Get retrieves a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) Get(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	result, err := h.getUseCase.ExecuteBySID(c.Request.Context(), sid)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to get resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// Update updates a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) Update(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
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

	result, err := h.updateUseCase.ExecuteBySID(c.Request.Context(), sid, dtoReq)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		if err == resource.ErrGroupNameExists {
			utils.ErrorResponse(c, http.StatusConflict, "resource group name already exists")
			return
		}
		h.logger.Errorw("failed to update resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Resource group updated successfully", result)
}

// Delete deletes a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) Delete(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	if err := h.deleteUseCase.ExecuteBySID(c.Request.Context(), sid); err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		if err == resource.ErrGroupHasResources {
			utils.ErrorResponse(c, http.StatusConflict, "resource group has associated resources")
			return
		}
		h.logger.Errorw("failed to delete resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// Activate activates a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) Activate(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	result, err := h.updateStatusUseCase.ActivateBySID(c.Request.Context(), sid)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to activate resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Resource group activated successfully", result)
}

// Deactivate deactivates a resource group by SID (Stripe-style ID: rg_xxx)
func (h *Handler) Deactivate(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	result, err := h.updateStatusUseCase.DeactivateBySID(c.Request.Context(), sid)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to deactivate resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Resource group deactivated successfully", result)
}
