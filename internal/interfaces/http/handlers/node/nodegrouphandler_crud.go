package node

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/utils"
)

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
