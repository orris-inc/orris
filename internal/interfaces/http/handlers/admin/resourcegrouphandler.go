// Package admin provides HTTP handlers for administrative operations.
package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/resource/dto"
	"github.com/orris-inc/orris/internal/application/resource/usecases"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/id"
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
	manageNodesUseCase  *usecases.ManageResourceGroupNodesUseCase
	manageAgentsUseCase *usecases.ManageResourceGroupForwardAgentsUseCase
	planRepo            subscription.PlanRepository
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
	manageNodesUC *usecases.ManageResourceGroupNodesUseCase,
	manageAgentsUC *usecases.ManageResourceGroupForwardAgentsUseCase,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *ResourceGroupHandler {
	return &ResourceGroupHandler{
		createUseCase:       createUC,
		getUseCase:          getUC,
		listUseCase:         listUC,
		updateUseCase:       updateUC,
		deleteUseCase:       deleteUC,
		updateStatusUseCase: updateStatusUC,
		manageNodesUseCase:  manageNodesUC,
		manageAgentsUseCase: manageAgentsUC,
		planRepo:            planRepo,
		logger:              logger,
	}
}

// CreateRequest represents the request to create a resource group
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
func (h *ResourceGroupHandler) Create(c *gin.Context) {
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
		// Support both SID (plan_xxx) and internal ID
		if strings.HasPrefix(planIDStr, "plan_") {
			// Resolve SID to internal ID
			plan, err := h.planRepo.GetBySID(c.Request.Context(), planIDStr)
			if err == nil && plan != nil {
				pid := plan.ID()
				planID = &pid
			}
		} else if pid, err := strconv.ParseUint(planIDStr, 10, 64); err == nil {
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

// Get retrieves a resource group by SID (Stripe-style ID: rg_xxx)
func (h *ResourceGroupHandler) Get(c *gin.Context) {
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
func (h *ResourceGroupHandler) Update(c *gin.Context) {
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
func (h *ResourceGroupHandler) Delete(c *gin.Context) {
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
func (h *ResourceGroupHandler) Activate(c *gin.Context) {
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
func (h *ResourceGroupHandler) Deactivate(c *gin.Context) {
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

// AddNodes adds nodes to a resource group by SID (Stripe-style ID: rg_xxx)
func (h *ResourceGroupHandler) AddNodes(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.AddNodesToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for add nodes", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageNodesUseCase.AddNodesBySID(c.Request.Context(), sid, req.NodeSIDs)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to add nodes to resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Nodes added to resource group", result)
}

// RemoveNodes removes nodes from a resource group by SID (Stripe-style ID: rg_xxx)
func (h *ResourceGroupHandler) RemoveNodes(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.RemoveNodesFromGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for remove nodes", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageNodesUseCase.RemoveNodesBySID(c.Request.Context(), sid, req.NodeSIDs)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to remove nodes from resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Nodes removed from resource group", result)
}

// ListNodes lists all nodes in a resource group by SID (Stripe-style ID: rg_xxx)
func (h *ResourceGroupHandler) ListNodes(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

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

	result, err := h.manageNodesUseCase.ListNodesBySID(c.Request.Context(), sid, page, pageSize)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to list nodes in resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Items, result.Total, result.Page, result.PageSize)
}

// AddForwardAgents adds forward agents to a resource group by SID (Stripe-style ID: rg_xxx)
func (h *ResourceGroupHandler) AddForwardAgents(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.AddForwardAgentsToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for add forward agents", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageAgentsUseCase.AddAgentsBySID(c.Request.Context(), sid, req.AgentSIDs)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to add forward agents to resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agents added to resource group", result)
}

// RemoveForwardAgents removes forward agents from a resource group by SID (Stripe-style ID: rg_xxx)
func (h *ResourceGroupHandler) RemoveForwardAgents(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

	var req dto.RemoveForwardAgentsFromGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for remove forward agents", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.manageAgentsUseCase.RemoveAgentsBySID(c.Request.Context(), sid, req.AgentSIDs)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to remove forward agents from resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward agents removed from resource group", result)
}

// ListForwardAgents lists all forward agents in a resource group by SID (Stripe-style ID: rg_xxx)
func (h *ResourceGroupHandler) ListForwardAgents(c *gin.Context) {
	sid := c.Param("id")
	if err := id.ValidatePrefix(sid, id.PrefixResourceGroup); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid resource group ID format, expected rg_xxxxx")
		return
	}

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

	result, err := h.manageAgentsUseCase.ListAgentsBySID(c.Request.Context(), sid, page, pageSize)
	if err != nil {
		if err == resource.ErrGroupNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "resource group not found")
			return
		}
		h.logger.Errorw("failed to list forward agents in resource group", "error", err, "sid", sid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Items, result.Total, result.Page, result.PageSize)
}
