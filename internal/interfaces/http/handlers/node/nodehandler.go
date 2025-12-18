package node

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type NodeHandler struct {
	createNodeUC            usecases.CreateNodeExecutor
	getNodeUC               usecases.GetNodeExecutor
	updateNodeUC            usecases.UpdateNodeExecutor
	deleteNodeUC            usecases.DeleteNodeExecutor
	listNodesUC             usecases.ListNodesExecutor
	generateTokenUC         usecases.GenerateNodeTokenExecutor
	generateInstallScriptUC usecases.GenerateNodeInstallScriptExecutor
	apiURL                  string
	logger                  logger.Interface
}

func NewNodeHandler(
	createNodeUC usecases.CreateNodeExecutor,
	getNodeUC usecases.GetNodeExecutor,
	updateNodeUC usecases.UpdateNodeExecutor,
	deleteNodeUC usecases.DeleteNodeExecutor,
	listNodesUC usecases.ListNodesExecutor,
	generateTokenUC usecases.GenerateNodeTokenExecutor,
	generateInstallScriptUC usecases.GenerateNodeInstallScriptExecutor,
	apiURL string,
) *NodeHandler {
	return &NodeHandler{
		createNodeUC:            createNodeUC,
		getNodeUC:               getNodeUC,
		updateNodeUC:            updateNodeUC,
		deleteNodeUC:            deleteNodeUC,
		listNodesUC:             listNodesUC,
		generateTokenUC:         generateTokenUC,
		generateInstallScriptUC: generateInstallScriptUC,
		apiURL:                  apiURL,
		logger:                  logger.NewLogger(),
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
	shortID, err := parseNodeShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetNodeQuery{ShortID: shortID}
	result, err := h.getNodeUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result.Node)
}

// UpdateNode handles PUT /nodes/:id
func (h *NodeHandler) UpdateNode(c *gin.Context) {
	shortID, err := parseNodeShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update node",
			"short_id", shortID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := req.ToCommand(shortID)
	result, err := h.updateNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node updated successfully", result)
}

// DeleteNode handles DELETE /nodes/:id
func (h *NodeHandler) DeleteNode(c *gin.Context) {
	shortID, err := parseNodeShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DeleteNodeCommand{ShortID: shortID}
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
	shortID, err := parseNodeShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.GenerateNodeTokenCommand{ShortID: shortID}
	result, err := h.generateTokenUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Token generated successfully", result)
}

// GetInstallScript handles GET /nodes/:id/install-script
// Query params:
//   - token (optional): API token. If not provided, uses node's current stored token
//   - api_url (optional): Override the default API URL
func (h *NodeHandler) GetInstallScript(c *gin.Context) {
	shortID, err := parseNodeShortID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Token is optional - if not provided, will use node's stored token
	token := c.Query("token")

	// Use query param to override API URL if provided
	apiURL := c.Query("api_url")
	if apiURL == "" {
		apiURL = h.apiURL
	}

	query := usecases.GenerateNodeInstallScriptQuery{
		ShortID: shortID,
		APIURL:  apiURL,
		Token:   token,
	}

	result, err := h.generateInstallScriptUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Install command generated successfully", result)
}

// UpdateNodeStatusRequest represents a request for node status changes
type UpdateNodeStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive maintenance"`
}

// UpdateNodeStatus handles PATCH /nodes/:id/status
func (h *NodeHandler) UpdateNodeStatus(c *gin.Context) {
	shortID, err := parseNodeShortID(c)
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
		ShortID: shortID,
		Status:  &req.Status,
	}
	result, err := h.updateNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node status updated successfully", result)
}

// parseNodeShortID extracts the short ID from a prefixed node ID (e.g., "node_xK9mP2vL3nQ" -> "xK9mP2vL3nQ").
func parseNodeShortID(c *gin.Context) (string, error) {
	prefixedID := c.Param("id")
	if prefixedID == "" {
		return "", errors.NewValidationError("node ID is required")
	}

	shortID, err := id.ParseNodeID(prefixedID)
	if err != nil {
		return "", errors.NewValidationError("invalid node ID format, expected node_xxxxx")
	}

	return shortID, nil
}

// parseNodeID is deprecated, use parseNodeShortID instead.
// Kept for backward compatibility with internal routes.
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
	ServerAddress    string            `json:"server_address,omitempty" example:"1.2.3.4"`
	AgentPort        uint16            `json:"agent_port" binding:"required" example:"8388" comment:"Port for agent connections"`
	SubscriptionPort *uint16           `json:"subscription_port,omitempty" example:"8389" comment:"Port for client subscriptions (if null, uses agent_port)"`
	Protocol         string            `json:"protocol" binding:"required,oneof=shadowsocks trojan" example:"shadowsocks" comment:"Protocol type: shadowsocks or trojan"`
	EncryptionMethod string            `json:"encryption_method,omitempty" example:"aes-256-gcm" comment:"Encryption method (for Shadowsocks)"`
	Plugin           *string           `json:"plugin,omitempty" example:"obfs-local"`
	PluginOpts       map[string]string `json:"plugin_opts,omitempty"`
	Region           string            `json:"region,omitempty" example:"West Coast"`
	Tags             []string          `json:"tags,omitempty" example:"premium,fast"`
	Description      string            `json:"description,omitempty" example:"High-speed US server"`
	SortOrder        int               `json:"sort_order,omitempty" example:"1"`
	// Trojan specific fields
	TransportProtocol string `json:"transport_protocol,omitempty" binding:"omitempty,oneof=tcp ws grpc" example:"tcp" comment:"Transport protocol for Trojan (tcp, ws, grpc)"`
	Host              string `json:"host,omitempty" example:"cdn.example.com" comment:"WebSocket host header or gRPC service name"`
	Path              string `json:"path,omitempty" example:"/trojan" comment:"WebSocket path"`
	SNI               string `json:"sni,omitempty" example:"example.com" comment:"TLS Server Name Indication"`
	AllowInsecure     bool   `json:"allow_insecure,omitempty" example:"true" comment:"Allow insecure TLS connection (for self-signed certs)"`
}

func (r *CreateNodeRequest) ToCommand() usecases.CreateNodeCommand {
	return usecases.CreateNodeCommand{
		Name:              r.Name,
		ServerAddress:     r.ServerAddress,
		AgentPort:         r.AgentPort,
		SubscriptionPort:  r.SubscriptionPort,
		Protocol:          r.Protocol,
		Method:            r.EncryptionMethod,
		Plugin:            r.Plugin,
		PluginOpts:        r.PluginOpts,
		Region:            r.Region,
		Tags:              r.Tags,
		Description:       r.Description,
		SortOrder:         r.SortOrder,
		TransportProtocol: r.TransportProtocol,
		Host:              r.Host,
		Path:              r.Path,
		SNI:               r.SNI,
		AllowInsecure:     r.AllowInsecure,
	}
}

type UpdateNodeRequest struct {
	Name             *string           `json:"name,omitempty" example:"US-Node-01-Updated"`
	ServerAddress    *string           `json:"server_address,omitempty" example:"2.3.4.5"`
	AgentPort        *uint16           `json:"agent_port,omitempty" example:"8388" comment:"Port for agent connections"`
	SubscriptionPort *uint16           `json:"subscription_port,omitempty" example:"8389" comment:"Port for client subscriptions"`
	EncryptionMethod *string           `json:"encryption_method,omitempty" example:"chacha20-ietf-poly1305" comment:"Encryption method (for Shadowsocks)"`
	Plugin           *string           `json:"plugin,omitempty" example:"v2ray-plugin"`
	PluginOpts       map[string]string `json:"plugin_opts,omitempty"`
	Status           *string           `json:"status,omitempty" binding:"omitempty,oneof=active inactive maintenance" example:"active"`
	Region           *string           `json:"region,omitempty" example:"Tokyo"`
	Tags             []string          `json:"tags,omitempty" example:"premium,low-latency"`
	Description      *string           `json:"description,omitempty" example:"Updated description"`
	SortOrder        *int              `json:"sort_order,omitempty" example:"2"`
	// Trojan specific fields
	TransportProtocol *string `json:"transport_protocol,omitempty" binding:"omitempty,oneof=tcp ws grpc" example:"ws" comment:"Transport protocol for Trojan (tcp, ws, grpc)"`
	Host              *string `json:"host,omitempty" example:"cdn.example.com" comment:"WebSocket host header or gRPC service name"`
	Path              *string `json:"path,omitempty" example:"/trojan" comment:"WebSocket path"`
	SNI               *string `json:"sni,omitempty" example:"example.com" comment:"TLS Server Name Indication"`
	AllowInsecure     *bool   `json:"allow_insecure,omitempty" example:"false" comment:"Allow insecure TLS connection"`
}

func (r *UpdateNodeRequest) ToCommand(shortID string) usecases.UpdateNodeCommand {
	return usecases.UpdateNodeCommand{
		ShortID:                 shortID,
		Name:                    r.Name,
		ServerAddress:           r.ServerAddress,
		AgentPort:               r.AgentPort,
		SubscriptionPort:        r.SubscriptionPort,
		Method:                  r.EncryptionMethod,
		Plugin:                  r.Plugin,
		PluginOpts:              r.PluginOpts,
		Status:                  r.Status,
		Region:                  r.Region,
		Tags:                    r.Tags,
		Description:             r.Description,
		SortOrder:               r.SortOrder,
		TrojanTransportProtocol: r.TransportProtocol,
		TrojanHost:              r.Host,
		TrojanPath:              r.Path,
		TrojanSNI:               r.SNI,
		TrojanAllowInsecure:     r.AllowInsecure,
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

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(constants.DefaultPageSize)))
	if pageSize < 1 || pageSize > constants.MaxPageSize {
		pageSize = constants.DefaultPageSize
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
