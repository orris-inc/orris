package node

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/utils"
)

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
		GroupID:     groupID,
		NodeShortID: req.NodeShortID,
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

	nodeShortID := c.Param("node_id")
	if nodeShortID == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("node ID is required"))
		return
	}

	cmd := usecases.RemoveNodeFromGroupCommand{
		GroupID:     groupID,
		NodeShortID: nodeShortID,
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
