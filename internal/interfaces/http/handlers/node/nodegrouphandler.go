package node

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type NodeGroupHandler struct {
	createNodeGroupUC           usecases.CreateNodeGroupExecutor
	getNodeGroupUC              usecases.GetNodeGroupExecutor
	updateNodeGroupUC           usecases.UpdateNodeGroupExecutor
	deleteNodeGroupUC           usecases.DeleteNodeGroupExecutor
	listNodeGroupsUC            usecases.ListNodeGroupsExecutor
	addNodeToGroupUC            usecases.AddNodeToGroupExecutor
	removeNodeFromGroupUC       usecases.RemoveNodeFromGroupExecutor
	batchAddNodesToGroupUC      usecases.BatchAddNodesToGroupExecutor
	batchRemoveNodesFromGroupUC usecases.BatchRemoveNodesFromGroupExecutor
	listGroupNodesUC            usecases.ListGroupNodesExecutor
	associateGroupWithPlanUC    usecases.AssociateGroupWithPlanExecutor
	disassociateGroupFromPlanUC usecases.DisassociateGroupFromPlanExecutor
	logger                      logger.Interface
}

func NewNodeGroupHandler(
	createNodeGroupUC usecases.CreateNodeGroupExecutor,
	getNodeGroupUC usecases.GetNodeGroupExecutor,
	updateNodeGroupUC usecases.UpdateNodeGroupExecutor,
	deleteNodeGroupUC usecases.DeleteNodeGroupExecutor,
	listNodeGroupsUC usecases.ListNodeGroupsExecutor,
	addNodeToGroupUC usecases.AddNodeToGroupExecutor,
	removeNodeFromGroupUC usecases.RemoveNodeFromGroupExecutor,
	batchAddNodesToGroupUC usecases.BatchAddNodesToGroupExecutor,
	batchRemoveNodesFromGroupUC usecases.BatchRemoveNodesFromGroupExecutor,
	listGroupNodesUC usecases.ListGroupNodesExecutor,
	associateGroupWithPlanUC usecases.AssociateGroupWithPlanExecutor,
	disassociateGroupFromPlanUC usecases.DisassociateGroupFromPlanExecutor,
) *NodeGroupHandler {
	return &NodeGroupHandler{
		createNodeGroupUC:           createNodeGroupUC,
		getNodeGroupUC:              getNodeGroupUC,
		updateNodeGroupUC:           updateNodeGroupUC,
		deleteNodeGroupUC:           deleteNodeGroupUC,
		listNodeGroupsUC:            listNodeGroupsUC,
		addNodeToGroupUC:            addNodeToGroupUC,
		removeNodeFromGroupUC:       removeNodeFromGroupUC,
		batchAddNodesToGroupUC:      batchAddNodesToGroupUC,
		batchRemoveNodesFromGroupUC: batchRemoveNodesFromGroupUC,
		listGroupNodesUC:            listGroupNodesUC,
		associateGroupWithPlanUC:    associateGroupWithPlanUC,
		disassociateGroupFromPlanUC: disassociateGroupFromPlanUC,
		logger:                      logger.NewLogger(),
	}
}

// CreateNodeGroup handles POST /node-groups
func (h *NodeGroupHandler) CreateNodeGroup(c *gin.Context) {
	var req CreateNodeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create node group", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.CreateNodeGroupCommand{
		Name:        req.Name,
		Description: req.Description,
		IsPublic:    req.IsPublic,
		SortOrder:   req.SortOrder,
	}

	result, err := h.createNodeGroupUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Node group created successfully")
}

// GetNodeGroup handles GET /node-groups/:id
func (h *NodeGroupHandler) GetNodeGroup(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetNodeGroupQuery{
		GroupID: groupID,
	}

	result, err := h.getNodeGroupUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// UpdateNodeGroup handles PUT /node-groups/:id
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

	// Get version from query parameter (for optimistic locking)
	version := 0
	if versionStr := c.Query("version"); versionStr != "" {
		version, _ = strconv.Atoi(versionStr)
	}

	cmd := usecases.UpdateNodeGroupCommand{
		GroupID:     groupID,
		Name:        req.Name,
		Description: req.Description,
		IsPublic:    req.IsPublic,
		SortOrder:   req.SortOrder,
		Version:     version,
	}

	result, err := h.updateNodeGroupUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node group updated successfully", result)
}

// DeleteNodeGroup handles DELETE /node-groups/:id
func (h *NodeGroupHandler) DeleteNodeGroup(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DeleteNodeGroupCommand{
		GroupID: groupID,
	}

	_, err = h.deleteNodeGroupUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// ListNodeGroups handles GET /node-groups
func (h *NodeGroupHandler) ListNodeGroups(c *gin.Context) {
	req, err := parseListNodeGroupsRequest(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.ListNodeGroupsQuery{
		IsPublic: req.IsPublic,
		Page:     req.Page,
		PageSize: req.PageSize,
		SortBy:   "sort_order",
		SortDesc: false,
	}

	result, err := h.listNodeGroupsUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Groups, result.Total, req.Page, req.PageSize)
}

// AddNodeToGroup handles POST /node-groups/:id/nodes
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

	cmd := usecases.AddNodeToGroupCommand{
		GroupID: groupID,
		NodeID:  req.NodeID,
	}

	result, err := h.addNodeToGroupUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node added to group successfully", result)
}

// RemoveNodeFromGroup handles DELETE /node-groups/:id/nodes/:nodeId
func (h *NodeGroupHandler) RemoveNodeFromGroup(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	nodeID, err := parseNodeIDFromParam(c, "node_id")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.RemoveNodeFromGroupCommand{
		GroupID: groupID,
		NodeID:  nodeID,
	}

	_, err = h.removeNodeFromGroupUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// BatchAddNodesToGroup handles POST /node-groups/:id/nodes/batch
func (h *NodeGroupHandler) BatchAddNodesToGroup(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req BatchAddNodesToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for batch add nodes to group",
			"group_id", groupID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.BatchAddNodesToGroupCommand{
		GroupID: groupID,
		NodeIDs: req.NodeIDs,
	}

	result, err := h.batchAddNodesToGroupUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, result.Message, result)
}

// BatchRemoveNodesFromGroup handles DELETE /node-groups/:id/nodes/batch
func (h *NodeGroupHandler) BatchRemoveNodesFromGroup(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req BatchRemoveNodesFromGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for batch remove nodes from group",
			"group_id", groupID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.BatchRemoveNodesFromGroupCommand{
		GroupID: groupID,
		NodeIDs: req.NodeIDs,
	}

	result, err := h.batchRemoveNodesFromGroupUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, result.Message, result)
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
func (h *NodeGroupHandler) ListGroupNodes(c *gin.Context) {
	groupID, err := parseNodeGroupID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.ListGroupNodesQuery{
		GroupID: groupID,
	}

	result, err := h.listGroupNodesUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// AssociatePlan handles POST /node-groups/:id/plans
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

	cmd := usecases.AssociateGroupWithPlanCommand{
		GroupID: groupID,
		PlanID:  req.PlanID,
	}

	result, err := h.associateGroupWithPlanUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Plan associated successfully", result)
}

// DisassociatePlan handles DELETE /node-groups/:id/plans/:planId
func (h *NodeGroupHandler) DisassociatePlan(c *gin.Context) {
	h.logger.Infow("received disassociate plan request",
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
	)

	groupID, err := parseNodeGroupID(c)
	if err != nil {
		h.logger.Warnw("failed to parse group ID",
			"error", err,
			"param", c.Param("id"),
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	planID, err := parsePlanIDFromParam(c, "plan_id")
	if err != nil {
		h.logger.Warnw("failed to parse plan ID",
			"error", err,
			"param", c.Param("plan_id"),
			"group_id", groupID,
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("parsed disassociate plan request parameters",
		"group_id", groupID,
		"plan_id", planID,
	)

	cmd := usecases.DisassociateGroupFromPlanCommand{
		GroupID: groupID,
		PlanID:  planID,
	}

	_, err = h.disassociateGroupFromPlanUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to disassociate plan from group",
			"error", err,
			"group_id", groupID,
			"plan_id", planID,
		)
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("successfully disassociated plan from group",
		"group_id", groupID,
		"plan_id", planID,
	)

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

type BatchAddNodesToGroupRequest struct {
	NodeIDs []uint `json:"node_ids" binding:"required,min=1,max=100" example:"1,2,3,4,5"`
}

type BatchRemoveNodesFromGroupRequest struct {
	NodeIDs []uint `json:"node_ids" binding:"required,min=1,max=100" example:"1,2,3,4,5"`
}
