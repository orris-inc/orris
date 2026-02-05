package node

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// UserNodeHandler handles user node management endpoints
type UserNodeHandler struct {
	createNodeUC            usecases.CreateUserNodeExecutor
	listNodesUC             usecases.ListUserNodesExecutor
	getNodeUC               usecases.GetUserNodeExecutor
	updateNodeUC            usecases.UpdateUserNodeExecutor
	deleteNodeUC            usecases.DeleteUserNodeExecutor
	regenerateTokenUC       usecases.RegenerateUserNodeTokenExecutor
	getUsageUC              usecases.GetUserNodeUsageExecutor
	getInstallScriptUC      usecases.GetUserNodeInstallScriptExecutor
	getBatchInstallScriptUC usecases.GetUserBatchInstallScriptExecutor
	apiURL                  string
	logger                  logger.Interface
}

// NewUserNodeHandler creates a new user node handler
func NewUserNodeHandler(
	createNodeUC usecases.CreateUserNodeExecutor,
	listNodesUC usecases.ListUserNodesExecutor,
	getNodeUC usecases.GetUserNodeExecutor,
	updateNodeUC usecases.UpdateUserNodeExecutor,
	deleteNodeUC usecases.DeleteUserNodeExecutor,
	regenerateTokenUC usecases.RegenerateUserNodeTokenExecutor,
	getUsageUC usecases.GetUserNodeUsageExecutor,
	getInstallScriptUC usecases.GetUserNodeInstallScriptExecutor,
	getBatchInstallScriptUC usecases.GetUserBatchInstallScriptExecutor,
	apiURL string,
) *UserNodeHandler {
	return &UserNodeHandler{
		createNodeUC:            createNodeUC,
		listNodesUC:             listNodesUC,
		getNodeUC:               getNodeUC,
		updateNodeUC:            updateNodeUC,
		deleteNodeUC:            deleteNodeUC,
		regenerateTokenUC:       regenerateTokenUC,
		getUsageUC:              getUsageUC,
		getInstallScriptUC:      getInstallScriptUC,
		getBatchInstallScriptUC: getBatchInstallScriptUC,
		apiURL:                  apiURL,
		logger:                  logger.NewLogger(),
	}
}

// CreateNode handles POST /user/nodes
func (h *UserNodeHandler) CreateNode(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req CreateUserNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create user node", "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := req.ToCommand(userID)
	result, err := h.createNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Node created successfully")
}

// ListNodes handles GET /user/nodes
func (h *UserNodeHandler) ListNodes(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	req, err := parseListUserNodesRequest(c, userID)
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

// GetNode handles GET /user/nodes/:id
func (h *UserNodeHandler) GetNode(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	nodeSID, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetUserNodeQuery{
		UserID:  userID,
		NodeSID: nodeSID,
	}

	result, err := h.getNodeUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// UpdateNode handles PUT /user/nodes/:id
func (h *UserNodeHandler) UpdateNode(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	nodeSID, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateUserNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update user node", "user_id", userID, "node_sid", nodeSID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := req.ToCommand(userID, nodeSID)
	result, err := h.updateNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node updated successfully", result)
}

// DeleteNode handles DELETE /user/nodes/:id
func (h *UserNodeHandler) DeleteNode(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	nodeSID, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DeleteUserNodeCommand{
		UserID:  userID,
		NodeSID: nodeSID,
	}

	if err := h.deleteNodeUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// RegenerateToken handles POST /user/nodes/:id/regenerate-token
func (h *UserNodeHandler) RegenerateToken(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	nodeSID, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.RegenerateUserNodeTokenCommand{
		UserID:  userID,
		NodeSID: nodeSID,
	}

	result, err := h.regenerateTokenUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Token regenerated successfully", result)
}

// GetUsage handles GET /user/nodes/usage
func (h *UserNodeHandler) GetUsage(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetUserNodeUsageQuery{
		UserID: userID,
	}

	result, err := h.getUsageUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetInstallScript handles GET /user/nodes/:id/install-script
func (h *UserNodeHandler) GetInstallScript(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	nodeSID, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Use query param to override API URL if provided
	apiURL := c.Query("api_url")
	if apiURL == "" {
		apiURL = h.apiURL
	}

	query := usecases.GetUserNodeInstallScriptQuery{
		UserID:  userID,
		NodeSID: nodeSID,
		APIURL:  apiURL,
	}

	result, err := h.getInstallScriptUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Install command generated successfully", result)
}


// CreateUserNodeRequest represents the request body for creating a user node
type CreateUserNodeRequest struct {
	Name              string            `json:"name" binding:"required,min=2,max=100" example:"My-Node-01"`
	ServerAddress     string            `json:"server_address,omitempty" example:"1.2.3.4"`
	AgentPort         uint16            `json:"agent_port" binding:"required,min=1,max=65535" example:"8388"`
	SubscriptionPort  *uint16           `json:"subscription_port,omitempty" binding:"omitempty,min=1,max=65535" example:"8389"`
	Protocol          string            `json:"protocol" binding:"required,oneof=shadowsocks trojan" example:"shadowsocks"`
	Method            string            `json:"method,omitempty" example:"aes-256-gcm"`
	Plugin            *string           `json:"plugin,omitempty" example:"obfs-local"`
	PluginOpts        map[string]string `json:"plugin_opts,omitempty"`
	TransportProtocol string            `json:"transport_protocol,omitempty" binding:"omitempty,oneof=tcp ws grpc" example:"tcp"`
	Host              string            `json:"host,omitempty" example:"cdn.example.com"`
	Path              string            `json:"path,omitempty" example:"/trojan"`
	SNI               string            `json:"sni,omitempty" example:"example.com"`
	AllowInsecure     bool              `json:"allow_insecure,omitempty" example:"false"`
}

func (r *CreateUserNodeRequest) ToCommand(userID uint) usecases.CreateUserNodeCommand {
	return usecases.CreateUserNodeCommand{
		UserID:            userID,
		Name:              r.Name,
		ServerAddress:     r.ServerAddress,
		AgentPort:         r.AgentPort,
		SubscriptionPort:  r.SubscriptionPort,
		Protocol:          r.Protocol,
		Method:            r.Method,
		Plugin:            r.Plugin,
		PluginOpts:        r.PluginOpts,
		TransportProtocol: r.TransportProtocol,
		Host:              r.Host,
		Path:              r.Path,
		SNI:               r.SNI,
		AllowInsecure:     r.AllowInsecure,
	}
}

// UpdateUserNodeRequest represents the request body for updating a user node
type UpdateUserNodeRequest struct {
	Name             *string `json:"name,omitempty" binding:"omitempty,min=2,max=100" example:"My-Node-Updated"`
	ServerAddress    *string `json:"server_address,omitempty" example:"2.3.4.5"`
	AgentPort        *uint16 `json:"agent_port,omitempty" binding:"omitempty,min=1,max=65535" example:"8388"`
	SubscriptionPort *uint16 `json:"subscription_port,omitempty" binding:"omitempty,min=1,max=65535" example:"8389"`
}

func (r *UpdateUserNodeRequest) ToCommand(userID uint, nodeSID string) usecases.UpdateUserNodeCommand {
	return usecases.UpdateUserNodeCommand{
		UserID:           userID,
		NodeSID:          nodeSID,
		Name:             r.Name,
		ServerAddress:    r.ServerAddress,
		AgentPort:        r.AgentPort,
		SubscriptionPort: r.SubscriptionPort,
	}
}

// ListUserNodesRequest represents the request parameters for listing user nodes
type ListUserNodesRequest struct {
	UserID    uint
	Page      int
	PageSize  int
	Status    *string
	Search    *string
	SortBy    string
	SortOrder string
}

func parseListUserNodesRequest(c *gin.Context, userID uint) (*ListUserNodesRequest, error) {
	pagination := utils.ParsePagination(c)

	req := &ListUserNodesRequest{
		UserID:    userID,
		Page:      pagination.Page,
		PageSize:  pagination.PageSize,
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}

	if status := c.Query("status"); status != "" {
		req.Status = &status
	}

	if search := c.Query("search"); search != "" {
		req.Search = &search
	}

	return req, nil
}

func (r *ListUserNodesRequest) ToCommand() usecases.ListUserNodesQuery {
	offset := (r.Page - 1) * r.PageSize
	return usecases.ListUserNodesQuery{
		UserID:    r.UserID,
		Status:    r.Status,
		Search:    r.Search,
		Limit:     r.PageSize,
		Offset:    offset,
		SortBy:    r.SortBy,
		SortOrder: r.SortOrder,
	}
}

// UserBatchInstallScriptRequest represents the request body for generating batch install script.
type UserBatchInstallScriptRequest struct {
	NodeIDs []string `json:"node_ids" binding:"required,min=1,max=100"`
}

// GetBatchInstallScript handles POST /user/nodes/batch-install-script
func (h *UserNodeHandler) GetBatchInstallScript(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UserBatchInstallScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for user batch install script", "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Use query param to override API URL if provided
	apiURL := c.Query("api_url")
	if apiURL == "" {
		apiURL = h.apiURL
	}

	query := usecases.GetUserBatchInstallScriptQuery{
		UserID: userID,
		SIDs:   req.NodeIDs,
		APIURL: apiURL,
	}

	result, err := h.getBatchInstallScriptUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Batch install command generated successfully", result)
}
