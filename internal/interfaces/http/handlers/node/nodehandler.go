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
	updateNodeUC    usecases.UpdateNodeExecutor
	deleteNodeUC    usecases.DeleteNodeExecutor
	listNodesUC     usecases.ListNodesExecutor
	generateTokenUC usecases.GenerateNodeTokenExecutor
	logger          logger.Interface
}

func NewNodeHandler(
	createNodeUC usecases.CreateNodeExecutor,
	updateNodeUC usecases.UpdateNodeExecutor,
	deleteNodeUC usecases.DeleteNodeExecutor,
	listNodesUC usecases.ListNodesExecutor,
	generateTokenUC usecases.GenerateNodeTokenExecutor,
) *NodeHandler {
	return &NodeHandler{
		createNodeUC:    createNodeUC,
		updateNodeUC:    updateNodeUC,
		deleteNodeUC:    deleteNodeUC,
		listNodesUC:     listNodesUC,
		generateTokenUC: generateTokenUC,
		logger:          logger.NewLogger(),
	}
}

// CreateNode handles POST /nodes
// @Summary Create a new node
// @Description Create a new proxy node with the input data
// @Tags nodes
// @Accept json
// @Produce json
// @Security Bearer
// @Param node body CreateNodeRequest true "Node data"
// @Success 201 {object} utils.APIResponse "Node created successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /nodes [post]
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
// @Summary Get node by ID
// @Description Get details of a node by its ID
// @Tags nodes
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node ID"
// @Success 200 {object} utils.APIResponse "Node details"
// @Failure 400 {object} utils.APIResponse "Invalid node ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /nodes/{id} [get]
func (h *NodeHandler) GetNode(c *gin.Context) {
	nodeID, err := parseNodeID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", map[string]interface{}{
		"id": nodeID,
	})
}

// UpdateNode handles PUT /nodes/:id
// @Summary Update node
// @Description Update node information by ID
// @Tags nodes
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node ID"
// @Param node body UpdateNodeRequest true "Node update data"
// @Success 200 {object} utils.APIResponse "Node updated successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /nodes/{id} [put]
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
// @Summary Delete node
// @Description Delete a node by ID
// @Tags nodes
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node ID"
// @Success 204 "Node deleted successfully"
// @Failure 400 {object} utils.APIResponse "Invalid node ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /nodes/{id} [delete]
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
// @Summary List nodes
// @Description Get a paginated list of nodes
// @Tags nodes
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param status query string false "Node status filter"
// @Param country query string false "Country filter"
// @Success 200 {object} utils.APIResponse "Nodes list"
// @Failure 400 {object} utils.APIResponse "Invalid query parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /nodes [get]
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

// GenerateToken handles POST /nodes/:id/token
// @Summary Generate new API token for node
// @Description Generate a new API token for node authentication
// @Tags nodes
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node ID"
// @Success 200 {object} utils.APIResponse "Token generated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid node ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /nodes/{id}/token [post]
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

// ActivateNode handles POST /nodes/:id/activate
// @Summary Activate node
// @Description Activate a node by ID
// @Tags nodes
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node ID"
// @Success 200 {object} utils.APIResponse "Node activated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid node ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /nodes/{id}/activate [post]
func (h *NodeHandler) ActivateNode(c *gin.Context) {
	nodeID, err := parseNodeID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	status := "active"
	cmd := usecases.UpdateNodeCommand{
		NodeID: nodeID,
		Status: &status,
	}
	result, err := h.updateNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node activated successfully", result)
}

// DeactivateNode handles POST /nodes/:id/deactivate
// @Summary Deactivate node
// @Description Deactivate a node by ID
// @Tags nodes
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node ID"
// @Success 200 {object} utils.APIResponse "Node deactivated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid node ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /nodes/{id}/deactivate [post]
func (h *NodeHandler) DeactivateNode(c *gin.Context) {
	nodeID, err := parseNodeID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	status := "inactive"
	cmd := usecases.UpdateNodeCommand{
		NodeID: nodeID,
		Status: &status,
	}
	result, err := h.updateNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node deactivated successfully", result)
}

// GetNodeTraffic handles GET /nodes/:id/traffic
// @Summary Get node traffic statistics
// @Description Get traffic statistics for a node
// @Tags nodes
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Node ID"
// @Param start query string false "Start time (RFC3339)"
// @Param end query string false "End time (RFC3339)"
// @Success 200 {object} utils.APIResponse "Node traffic statistics"
// @Failure 400 {object} utils.APIResponse "Invalid node ID or parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Node not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /nodes/{id}/traffic [get]
func (h *NodeHandler) GetNodeTraffic(c *gin.Context) {
	nodeID, err := parseNodeID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", map[string]interface{}{
		"node_id":  nodeID,
		"upload":   0,
		"download": 0,
	})
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
	Name          string            `json:"name" binding:"required"`
	ServerAddress string            `json:"server_address" binding:"required"`
	ServerPort    uint16            `json:"server_port" binding:"required"`
	Method        string            `json:"method" binding:"required"`
	Password      string            `json:"password" binding:"required"`
	Plugin        *string           `json:"plugin,omitempty"`
	PluginOpts    map[string]string `json:"plugin_opts,omitempty"`
	Country       string            `json:"country" binding:"required"`
	Region        string            `json:"region,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
	Description   string            `json:"description,omitempty"`
	MaxUsers      uint32            `json:"max_users,omitempty"`
	TrafficLimit  uint64            `json:"traffic_limit,omitempty"`
	SortOrder     int               `json:"sort_order,omitempty"`
}

func (r *CreateNodeRequest) ToCommand() usecases.CreateNodeCommand {
	return usecases.CreateNodeCommand{
		Name:          r.Name,
		ServerAddress: r.ServerAddress,
		ServerPort:    r.ServerPort,
		Method:        r.Method,
		Password:      r.Password,
		Plugin:        r.Plugin,
		PluginOpts:    r.PluginOpts,
		Country:       r.Country,
		Region:        r.Region,
		Tags:          r.Tags,
		Description:   r.Description,
		MaxUsers:      r.MaxUsers,
		TrafficLimit:  r.TrafficLimit,
		SortOrder:     r.SortOrder,
	}
}

type UpdateNodeRequest struct {
	Name          *string           `json:"name,omitempty"`
	ServerAddress *string           `json:"server_address,omitempty"`
	ServerPort    *uint16           `json:"server_port,omitempty"`
	Method        *string           `json:"method,omitempty"`
	Password      *string           `json:"password,omitempty"`
	Plugin        *string           `json:"plugin,omitempty"`
	PluginOpts    map[string]string `json:"plugin_opts,omitempty"`
	Status        *string           `json:"status,omitempty"`
	Country       *string           `json:"country,omitempty"`
	Region        *string           `json:"region,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
	Description   *string           `json:"description,omitempty"`
	MaxUsers      *uint32           `json:"max_users,omitempty"`
	TrafficLimit  *uint64           `json:"traffic_limit,omitempty"`
	SortOrder     *int              `json:"sort_order,omitempty"`
}

func (r *UpdateNodeRequest) ToCommand(nodeID uint) usecases.UpdateNodeCommand {
	return usecases.UpdateNodeCommand{
		NodeID:        nodeID,
		Name:          r.Name,
		ServerAddress: r.ServerAddress,
		ServerPort:    r.ServerPort,
		Method:        r.Method,
		Password:      r.Password,
		Plugin:        r.Plugin,
		PluginOpts:    r.PluginOpts,
		Status:        r.Status,
		Country:       r.Country,
		Region:        r.Region,
		Tags:          r.Tags,
		Description:   r.Description,
		MaxUsers:      r.MaxUsers,
		TrafficLimit:  r.TrafficLimit,
		SortOrder:     r.SortOrder,
	}
}

type ListNodesRequest struct {
	Page     int
	PageSize int
	Status   *string
	Country  *string
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

	if country := c.Query("country"); country != "" {
		req.Country = &country
	}

	return req, nil
}

func (r *ListNodesRequest) ToCommand() usecases.ListNodesQuery {
	offset := (r.Page - 1) * r.PageSize
	return usecases.ListNodesQuery{
		Limit:   r.PageSize,
		Offset:  offset,
		Status:  r.Status,
		Country: r.Country,
	}
}
