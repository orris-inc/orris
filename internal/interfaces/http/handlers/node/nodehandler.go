package node

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"orris/internal/application/node/usecases"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type NodeHandler struct {
	createNodeUC    usecases.CreateNodeExecutor
	getNodeUC       usecases.GetNodeExecutor
	updateNodeUC    usecases.UpdateNodeExecutor
	deleteNodeUC    usecases.DeleteNodeExecutor
	listNodesUC     usecases.ListNodesExecutor
	generateTokenUC usecases.GenerateNodeTokenExecutor
	logger          logger.Interface
}

func NewNodeHandler(
	createNodeUC usecases.CreateNodeExecutor,
	getNodeUC usecases.GetNodeExecutor,
	updateNodeUC usecases.UpdateNodeExecutor,
	deleteNodeUC usecases.DeleteNodeExecutor,
	listNodesUC usecases.ListNodesExecutor,
	generateTokenUC usecases.GenerateNodeTokenExecutor,
) *NodeHandler {
	return &NodeHandler{
		createNodeUC:    createNodeUC,
		getNodeUC:       getNodeUC,
		updateNodeUC:    updateNodeUC,
		deleteNodeUC:    deleteNodeUC,
		listNodesUC:     listNodesUC,
		generateTokenUC: generateTokenUC,
		logger:          logger.NewLogger(),
	}
}

// CreateNode handles POST /nodes
func (h *NodeHandler) CreateNode(c *gin.Context) {
	var req CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create node", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := req.ToCommand()
	result, err := h.createNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Node created successfully")
}

// GetNode handles GET /nodes/:id
func (h *NodeHandler) GetNode(c *gin.Context) {
	nodeID, err := parseNodeID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetNodeQuery{NodeID: nodeID}
	result, err := h.getNodeUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result.Node)
}

// UpdateNode handles PUT /nodes/:id
func (h *NodeHandler) UpdateNode(c *gin.Context) {
	nodeID, err := parseNodeID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update node",
			"node_id", nodeID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := req.ToCommand(nodeID)
	result, err := h.updateNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node updated successfully", result)
}

// DeleteNode handles DELETE /nodes/:id
func (h *NodeHandler) DeleteNode(c *gin.Context) {
	nodeID, err := parseNodeID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DeleteNodeCommand{NodeID: nodeID}
	_, err = h.deleteNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// ListNodes handles GET /nodes
func (h *NodeHandler) ListNodes(c *gin.Context) {
	req, err := parseListNodesRequest(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := req.ToCommand()
	result, err := h.listNodesUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Nodes, int64(result.TotalCount), req.Page, req.PageSize)
}

// GenerateToken handles POST /nodes/:id/tokens
func (h *NodeHandler) GenerateToken(c *gin.Context) {
	nodeID, err := parseNodeID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.GenerateNodeTokenCommand{NodeID: nodeID}
	result, err := h.generateTokenUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Token generated successfully", result)
}

// UpdateNodeStatusRequest represents a request for node status changes
type UpdateNodeStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive maintenance"`
}

// UpdateNodeStatus handles PATCH /nodes/:id/status
func (h *NodeHandler) UpdateNodeStatus(c *gin.Context) {
	nodeID, err := parseNodeID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateNodeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update node status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.UpdateNodeCommand{
		NodeID: nodeID,
		Status: &req.Status,
	}
	result, err := h.updateNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node status updated successfully", result)
}

func parseNodeID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid node ID")
	}
	if id == 0 {
		return 0, errors.NewValidationError("Node ID must be greater than 0")
	}
	return uint(id), nil
}

type CreateNodeRequest struct {
	Name             string            `json:"name" binding:"required" example:"US-Node-01"`
	ServerAddress    string            `json:"server_address" binding:"required" example:"1.2.3.4"`
	ServerPort       uint16            `json:"server_port" binding:"required" example:"8388"`
	Protocol         string            `json:"protocol" binding:"required,oneof=shadowsocks trojan" example:"shadowsocks" comment:"Protocol type: shadowsocks or trojan"`
	EncryptionMethod string            `json:"encryption_method" binding:"required" example:"aes-256-gcm" comment:"Encryption method (for Shadowsocks), password is subscription UUID"`
	Plugin           *string           `json:"plugin,omitempty" example:"obfs-local"`
	PluginOpts       map[string]string `json:"plugin_opts,omitempty"`
	Region           string            `json:"region,omitempty" example:"West Coast"`
	Tags             []string          `json:"tags,omitempty" example:"premium,fast"`
	Description      string            `json:"description,omitempty" example:"High-speed US server"`
	SortOrder        int               `json:"sort_order,omitempty" example:"1"`
}

func (r *CreateNodeRequest) ToCommand() usecases.CreateNodeCommand {
	return usecases.CreateNodeCommand{
		Name:          r.Name,
		ServerAddress: r.ServerAddress,
		ServerPort:    r.ServerPort,
		Protocol:      r.Protocol,
		Method:        r.EncryptionMethod,
		Plugin:        r.Plugin,
		PluginOpts:    r.PluginOpts,
		Region:        r.Region,
		Tags:          r.Tags,
		Description:   r.Description,
		SortOrder:     r.SortOrder,
	}
}

type UpdateNodeRequest struct {
	Name             *string           `json:"name,omitempty" example:"US-Node-01-Updated"`
	ServerAddress    *string           `json:"server_address,omitempty" example:"2.3.4.5"`
	ServerPort       *uint16           `json:"server_port,omitempty" example:"8389"`
	EncryptionMethod *string           `json:"encryption_method,omitempty" example:"chacha20-ietf-poly1305" comment:"Encryption method (for Shadowsocks)"`
	Plugin           *string           `json:"plugin,omitempty" example:"v2ray-plugin"`
	PluginOpts       map[string]string `json:"plugin_opts,omitempty"`
	Status           *string           `json:"status,omitempty" binding:"omitempty,oneof=active inactive maintenance" example:"active"`
	Region           *string           `json:"region,omitempty" example:"Tokyo"`
	Tags             []string          `json:"tags,omitempty" example:"premium,low-latency"`
	Description      *string           `json:"description,omitempty" example:"Updated description"`
	SortOrder        *int              `json:"sort_order,omitempty" example:"2"`
}

func (r *UpdateNodeRequest) ToCommand(nodeID uint) usecases.UpdateNodeCommand {
	return usecases.UpdateNodeCommand{
		NodeID:        nodeID,
		Name:          r.Name,
		ServerAddress: r.ServerAddress,
		ServerPort:    r.ServerPort,
		Method:        r.EncryptionMethod,
		Plugin:        r.Plugin,
		PluginOpts:    r.PluginOpts,
		Status:        r.Status,
		Region:        r.Region,
		Tags:          r.Tags,
		Description:   r.Description,
		SortOrder:     r.SortOrder,
	}
}

type ListNodesRequest struct {
	Page     int
	PageSize int
	Status   *string
}

func parseListNodesRequest(c *gin.Context) (*ListNodesRequest, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	req := &ListNodesRequest{
		Page:     page,
		PageSize: pageSize,
	}

	if status := c.Query("status"); status != "" {
		req.Status = &status
	}

	return req, nil
}

func (r *ListNodesRequest) ToCommand() usecases.ListNodesQuery {
	offset := (r.Page - 1) * r.PageSize
	return usecases.ListNodesQuery{
		Limit:  r.PageSize,
		Offset: offset,
		Status: r.Status,
	}
}
