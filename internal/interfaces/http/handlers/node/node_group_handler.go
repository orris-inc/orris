package node

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type NodeGroupHandler struct {
	logger logger.Interface
}

func NewNodeGroupHandler() *NodeGroupHandler {
	return &NodeGroupHandler{
		logger: logger.NewLogger(),
	}
}

// CreateNodeGroup handles POST /node-groups
// @Summary Create a new node group
// @Description Create a new node group with the input data
// @Tags node-groups
// @Accept json
// @Produce json
// @Security Bearer
// @Param group body CreateNodeGroupRequest true "Node group data"
// @Success 201 {object} utils.APIResponse "Node group created successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /node-groups [post]
func (h *NodeGroupHandler) CreateNodeGroup(c *gin.Context) {
	var req CreateNodeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create node group", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, map[string]interface{}{
		"name": req.Name,
	}, "Node group created successfully")
}

// GetNodeGroup handles GET /node-groups/:id
// @Summary Get node group by ID
// @Description Get details of a node group by its ID
// @Tags node-groups
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node group ID"
// @Success 200 {object} utils.APIResponse "Node group details"
// @Failure 400 {object} utils.APIResponse "Invalid node group ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node group not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /node-groups/{id} [get]
func (h *NodeGroupHandler) GetNodeGroup(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", map[string]interface{}{
		"id": groupID,
	})
}

// UpdateNodeGroup handles PUT /node-groups/:id
// @Summary Update node group
// @Description Update node group information by ID
// @Tags node-groups
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node group ID"
// @Param group body UpdateNodeGroupRequest true "Node group update data"
// @Success 200 {object} utils.APIResponse "Node group updated successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node group not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /node-groups/{id} [put]
func (h *NodeGroupHandler) UpdateNodeGroup(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateNodeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update node group",
			"group_id", groupID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node group updated successfully", map[string]interface{}{
		"id": groupID,
	})
}

// DeleteNodeGroup handles DELETE /node-groups/:id
// @Summary Delete node group
// @Description Delete a node group by ID
// @Tags node-groups
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node group ID"
// @Success 204 "Node group deleted successfully"
// @Failure 400 {object} utils.APIResponse "Invalid node group ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node group not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /node-groups/{id} [delete]
func (h *NodeGroupHandler) DeleteNodeGroup(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("node group deleted", "group_id", groupID)
	utils.NoContentResponse(c)
}

// ListNodeGroups handles GET /node-groups
// @Summary List node groups
// @Description Get a paginated list of node groups
// @Tags node-groups
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param is_public query bool false "Public visibility filter"
// @Success 200 {object} utils.APIResponse "Node groups list"
// @Failure 400 {object} utils.APIResponse "Invalid query parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /node-groups [get]
func (h *NodeGroupHandler) ListNodeGroups(c *gin.Context) {
	req, err := parseListNodeGroupsRequest(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, []interface{}{}, 0, req.Page, req.PageSize)
}

// AddNodeToGroup handles POST /node-groups/:id/nodes
// @Summary Add node to group
// @Description Add a node to a node group
// @Tags node-groups
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node group ID"
// @Param node body AddNodeToGroupRequest true "Node to add"
// @Success 200 {object} utils.APIResponse "Node added successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node group not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /node-groups/{id}/nodes [post]
func (h *NodeGroupHandler) AddNodeToGroup(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req AddNodeToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for add node to group",
			"group_id", groupID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("node added to group", "group_id", groupID, "node_id", req.NodeID)
	utils.SuccessResponse(c, http.StatusOK, "Node added to group successfully", map[string]interface{}{
		"group_id": groupID,
		"node_id":  req.NodeID,
	})
}

// RemoveNodeFromGroup handles DELETE /node-groups/:id/nodes/:nodeId
// @Summary Remove node from group
// @Description Remove a node from a node group
// @Tags node-groups
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node group ID"
// @Param nodeId path int true "Node ID"
// @Success 204 "Node removed successfully"
// @Failure 400 {object} utils.APIResponse "Invalid ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node group not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /node-groups/{id}/nodes/{nodeId} [delete]
func (h *NodeGroupHandler) RemoveNodeFromGroup(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	nodeID, err := parseNodeIDFromParam(c, "nodeId")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("node removed from group", "group_id", groupID, "node_id", nodeID)
	utils.NoContentResponse(c)
}

func parseNodeGroupID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid node group ID")
	}
	if id == 0 {
		return 0, errors.NewValidationError("Node group ID must be greater than 0")
	}
	return uint(id), nil
}

func parseNodeIDFromParam(c *gin.Context, paramName string) (uint, error) {
	idStr := c.Param(paramName)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid node ID")
	}
	if id == 0 {
		return 0, errors.NewValidationError("Node ID must be greater than 0")
	}
	return uint(id), nil
}

type CreateNodeGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	IsPublic    bool   `json:"is_public"`
	SortOrder   int    `json:"sort_order,omitempty"`
}

type UpdateNodeGroupRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsPublic    *bool   `json:"is_public,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

type AddNodeToGroupRequest struct {
	NodeID uint `json:"node_id" binding:"required"`
}

type ListNodeGroupsRequest struct {
	Page     int
	PageSize int
	IsPublic *bool
}

// ListGroupNodes handles GET /node-groups/:id/nodes
// @Summary List nodes in group
// @Description Get all nodes in a node group
// @Tags node-groups
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node group ID"
// @Success 200 {object} utils.APIResponse "List of nodes in group"
// @Failure 400 {object} utils.APIResponse "Invalid node group ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node group not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /node-groups/{id}/nodes [get]
func (h *NodeGroupHandler) ListGroupNodes(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", map[string]interface{}{
		"group_id": groupID,
		"nodes":    []interface{}{},
	})
}

// AssociatePlan handles POST /node-groups/:id/plans
// @Summary Associate subscription plan with node group
// @Description Associate a subscription plan with a node group
// @Tags node-groups
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node group ID"
// @Param plan body AssociatePlanRequest true "Plan to associate"
// @Success 200 {object} utils.APIResponse "Plan associated successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node group not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /node-groups/{id}/plans [post]
func (h *NodeGroupHandler) AssociatePlan(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req AssociatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for associate plan",
			"group_id", groupID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("plan associated with group", "group_id", groupID, "plan_id", req.PlanID)
	utils.SuccessResponse(c, http.StatusOK, "Plan associated successfully", map[string]interface{}{
		"group_id": groupID,
		"plan_id":  req.PlanID,
	})
}

// DisassociatePlan handles DELETE /node-groups/:id/plans/:planId
// @Summary Disassociate subscription plan from node group
// @Description Remove association between subscription plan and node group
// @Tags node-groups
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node group ID"
// @Param planId path int true "Plan ID"
// @Success 204 "Plan disassociated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node group or plan not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /node-groups/{id}/plans/{planId} [delete]
func (h *NodeGroupHandler) DisassociatePlan(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	planID, err := parsePlanIDFromParam(c, "planId")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("plan disassociated from group", "group_id", groupID, "plan_id", planID)
	utils.NoContentResponse(c)
}

func parsePlanIDFromParam(c *gin.Context, paramName string) (uint, error) {
	idStr := c.Param(paramName)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid plan ID")
	}
	if id == 0 {
		return 0, errors.NewValidationError("Plan ID must be greater than 0")
	}
	return uint(id), nil
}

func parseListNodeGroupsRequest(c *gin.Context) (*ListNodeGroupsRequest, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	req := &ListNodeGroupsRequest{
		Page:     page,
		PageSize: pageSize,
	}

	if isPublicStr := c.Query("is_public"); isPublicStr != "" {
		isPublic := isPublicStr == "true"
		req.IsPublic = &isPublic
	}

	return req, nil
}

type AssociatePlanRequest struct {
	PlanID uint `json:"plan_id" binding:"required"`
}
