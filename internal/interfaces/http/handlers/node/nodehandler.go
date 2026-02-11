package node

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type NodeHandler struct {
	createNodeUC                 createNodeUseCase
	getNodeUC                    getNodeUseCase
	updateNodeUC                 updateNodeUseCase
	deleteNodeUC                 deleteNodeUseCase
	listNodesUC                  listNodesUseCase
	generateTokenUC              generateNodeTokenUseCase
	generateInstallScriptUC      generateNodeInstallScriptUseCase
	generateBatchInstallScriptUC generateBatchInstallScriptUseCase
	apiURL                       string
	logger                       logger.Interface
}

func NewNodeHandler(
	createNodeUC createNodeUseCase,
	getNodeUC getNodeUseCase,
	updateNodeUC updateNodeUseCase,
	deleteNodeUC deleteNodeUseCase,
	listNodesUC listNodesUseCase,
	generateTokenUC generateNodeTokenUseCase,
	generateInstallScriptUC generateNodeInstallScriptUseCase,
	generateBatchInstallScriptUC generateBatchInstallScriptUseCase,
	apiURL string,
	log logger.Interface,
) *NodeHandler {
	return &NodeHandler{
		createNodeUC:                 createNodeUC,
		getNodeUC:                    getNodeUC,
		updateNodeUC:                 updateNodeUC,
		deleteNodeUC:                 deleteNodeUC,
		listNodesUC:                  listNodesUC,
		generateTokenUC:              generateTokenUC,
		generateInstallScriptUC:      generateInstallScriptUC,
		generateBatchInstallScriptUC: generateBatchInstallScriptUC,
		apiURL:                       apiURL,
		logger:                       log,
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
	sid, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetNodeQuery{SID: sid}
	result, err := h.getNodeUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result.Node)
}

// UpdateNode handles PUT /nodes/:id
func (h *NodeHandler) UpdateNode(c *gin.Context) {
	sid, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update node",
			"sid", sid,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Validate and parse expires_at if provided and non-empty
	var parsedExpiresAt *time.Time
	var clearExpiresAt bool
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			// Empty string means clear
			clearExpiresAt = true
		} else {
			t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
			if err != nil {
				h.logger.Warnw("invalid expires_at format", "sid", sid, "expires_at", *req.ExpiresAt, "error", err)
				utils.ErrorResponseWithError(c, errors.NewValidationError("invalid expires_at format, expected ISO8601 (RFC3339)"))
				return
			}
			// Validate expires_at is in the future
			if t.Before(time.Now().UTC()) {
				h.logger.Warnw("expires_at must be in the future", "sid", sid, "expires_at", *req.ExpiresAt)
				utils.ErrorResponseWithError(c, errors.NewValidationError("expires_at must be a future time"))
				return
			}
			parsedExpiresAt = &t
		}
	}

	// Validate cost_label length if provided and non-empty
	if req.CostLabel != nil && *req.CostLabel != "" && len(*req.CostLabel) > 50 {
		h.logger.Warnw("cost_label exceeds max length", "sid", sid, "length", len(*req.CostLabel))
		utils.ErrorResponseWithError(c, errors.NewValidationError("cost_label cannot exceed 50 characters"))
		return
	}

	cmd := req.ToCommand(sid)

	// Override expires_at with validated values (handler is the source of truth)
	cmd.ExpiresAt = parsedExpiresAt
	cmd.ClearExpiresAt = clearExpiresAt
	result, err := h.updateNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node updated successfully", result)
}

// DeleteNode handles DELETE /nodes/:id
func (h *NodeHandler) DeleteNode(c *gin.Context) {
	sid, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DeleteNodeCommand{SID: sid}
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
	sid, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.GenerateNodeTokenCommand{SID: sid}
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
	sid, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
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
		SID:    sid,
		APIURL: apiURL,
		Token:  token,
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
	sid, err := utils.ParseSIDParam(c, "id", id.PrefixNode, "node")
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
		SID:    sid,
		Status: &req.Status,
	}
	result, err := h.updateNodeUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Node status updated successfully", result)
}


type CreateNodeRequest struct {
	Name             string            `json:"name" binding:"required" example:"US-Node-01"`
	ServerAddress    string            `json:"server_address,omitempty" example:"1.2.3.4"`
	AgentPort        uint16            `json:"agent_port" binding:"required" example:"8388" comment:"Port for agent connections"`
	SubscriptionPort *uint16           `json:"subscription_port,omitempty" example:"8389" comment:"Port for client subscriptions (if null, uses agent_port)"`
	Protocol         string            `json:"protocol" binding:"required,oneof=shadowsocks trojan vless vmess hysteria2 tuic anytls" example:"shadowsocks" comment:"Protocol type"`
	EncryptionMethod string            `json:"encryption_method,omitempty" example:"aes-256-gcm" comment:"Encryption method (for Shadowsocks)"`
	Plugin           *string           `json:"plugin,omitempty" example:"obfs-local"`
	PluginOpts       map[string]string `json:"plugin_opts,omitempty"`
	Region           string            `json:"region,omitempty" example:"West Coast"`
	Tags             []string          `json:"tags,omitempty" example:"premium,fast"`
	Description      string            `json:"description,omitempty" example:"High-speed US server"`
	SortOrder        int               `json:"sort_order,omitempty" example:"1"`
	GroupSIDs        []string          `json:"group_sids,omitempty" example:"[\"rg_xK9mP2vL3nQ\"]" comment:"Resource group SIDs to associate with"`
	// Trojan specific fields
	TransportProtocol string `json:"transport_protocol,omitempty" binding:"omitempty,oneof=tcp ws grpc" example:"tcp" comment:"Transport protocol for Trojan (tcp, ws, grpc)"`
	Host              string `json:"host,omitempty" example:"cdn.example.com" comment:"WebSocket host header or gRPC service name"`
	Path              string `json:"path,omitempty" example:"/trojan" comment:"WebSocket path"`
	SNI               string `json:"sni,omitempty" example:"example.com" comment:"TLS Server Name Indication"`
	AllowInsecure     bool   `json:"allow_insecure,omitempty" example:"true" comment:"Allow insecure TLS connection (for self-signed certs)"`
	// Route configuration for traffic splitting (sing-box compatible)
	Route *dto.RouteConfigDTO `json:"route,omitempty" comment:"Route configuration for traffic splitting"`

	// VLESS specific fields
	VLESSTransportType     string `json:"vless_transport_type,omitempty" binding:"omitempty,oneof=tcp ws grpc h2" example:"tcp" comment:"VLESS transport type"`
	VLESSFlow              string `json:"vless_flow,omitempty" example:"xtls-rprx-vision" comment:"VLESS flow control"`
	VLESSSecurity          string `json:"vless_security,omitempty" binding:"omitempty,oneof=none tls reality" example:"tls" comment:"VLESS security type"`
	VLESSSni               string `json:"vless_sni,omitempty" example:"example.com" comment:"VLESS TLS SNI"`
	VLESSFingerprint       string `json:"vless_fingerprint,omitempty" example:"chrome" comment:"VLESS TLS fingerprint"`
	VLESSAllowInsecure     bool   `json:"vless_allow_insecure,omitempty" comment:"VLESS allow insecure TLS"`
	VLESSHost              string `json:"vless_host,omitempty" comment:"VLESS WS/H2 host header"`
	VLESSPath              string `json:"vless_path,omitempty" comment:"VLESS WS/H2 path"`
	VLESSServiceName       string `json:"vless_service_name,omitempty" comment:"VLESS gRPC service name"`
	VLESSRealityPrivateKey string `json:"vless_reality_private_key,omitempty" comment:"VLESS Reality private key (optional, auto-generated if empty)"`
	VLESSRealityPublicKey  string `json:"vless_reality_public_key,omitempty" comment:"VLESS Reality public key (optional, auto-generated if empty)"`
	VLESSRealityShortID    string `json:"vless_reality_short_id,omitempty" comment:"VLESS Reality short ID (optional, auto-generated if empty)"`
	VLESSRealitySpiderX    string `json:"vless_reality_spider_x,omitempty" comment:"VLESS Reality spider X"`

	// VMess specific fields
	VMessAlterID       int    `json:"vmess_alter_id,omitempty" example:"0" comment:"VMess alter ID"`
	VMessSecurity      string `json:"vmess_security,omitempty" binding:"omitempty,oneof=auto aes-128-gcm chacha20-poly1305 none zero" example:"auto" comment:"VMess security"`
	VMessTransportType string `json:"vmess_transport_type,omitempty" binding:"omitempty,oneof=tcp ws grpc http quic" example:"tcp" comment:"VMess transport type"`
	VMessHost          string `json:"vmess_host,omitempty" comment:"VMess WS/HTTP host header"`
	VMessPath          string `json:"vmess_path,omitempty" comment:"VMess WS/HTTP path"`
	VMessServiceName   string `json:"vmess_service_name,omitempty" comment:"VMess gRPC service name"`
	VMessTLS           bool   `json:"vmess_tls,omitempty" comment:"VMess TLS enabled"`
	VMessSni           string `json:"vmess_sni,omitempty" comment:"VMess TLS SNI"`
	VMessAllowInsecure bool   `json:"vmess_allow_insecure,omitempty" comment:"VMess allow insecure TLS"`

	// Hysteria2 specific fields
	Hysteria2CongestionControl string `json:"hysteria2_congestion_control,omitempty" binding:"omitempty,oneof=cubic bbr new_reno" example:"bbr" comment:"Hysteria2 congestion control"`
	Hysteria2Obfs              string `json:"hysteria2_obfs,omitempty" binding:"omitempty,oneof=salamander" example:"salamander" comment:"Hysteria2 obfuscation type"`
	Hysteria2ObfsPassword      string `json:"hysteria2_obfs_password,omitempty" comment:"Hysteria2 obfuscation password"`
	Hysteria2UpMbps            *int   `json:"hysteria2_up_mbps,omitempty" comment:"Hysteria2 upstream bandwidth limit"`
	Hysteria2DownMbps          *int   `json:"hysteria2_down_mbps,omitempty" comment:"Hysteria2 downstream bandwidth limit"`
	Hysteria2Sni               string `json:"hysteria2_sni,omitempty" comment:"Hysteria2 TLS SNI"`
	Hysteria2AllowInsecure     bool   `json:"hysteria2_allow_insecure,omitempty" comment:"Hysteria2 allow insecure TLS"`
	Hysteria2Fingerprint       string `json:"hysteria2_fingerprint,omitempty" comment:"Hysteria2 TLS fingerprint"`

	// TUIC specific fields
	TUICCongestionControl string `json:"tuic_congestion_control,omitempty" binding:"omitempty,oneof=cubic bbr new_reno" example:"bbr" comment:"TUIC congestion control"`
	TUICUDPRelayMode      string `json:"tuic_udp_relay_mode,omitempty" binding:"omitempty,oneof=native quic" example:"native" comment:"TUIC UDP relay mode"`
	TUICAlpn              string `json:"tuic_alpn,omitempty" comment:"TUIC ALPN protocols"`
	TUICSni               string `json:"tuic_sni,omitempty" comment:"TUIC TLS SNI"`
	TUICAllowInsecure     bool   `json:"tuic_allow_insecure,omitempty" comment:"TUIC allow insecure TLS"`
	TUICDisableSNI        bool   `json:"tuic_disable_sni,omitempty" comment:"TUIC disable SNI"`

	// AnyTLS specific fields
	AnyTLSSni                      string `json:"anytls_sni,omitempty" comment:"AnyTLS TLS SNI"`
	AnyTLSAllowInsecure            bool   `json:"anytls_allow_insecure,omitempty" comment:"AnyTLS allow insecure TLS"`
	AnyTLSFingerprint              string `json:"anytls_fingerprint,omitempty" comment:"AnyTLS TLS fingerprint"`
	AnyTLSIdleSessionCheckInterval string `json:"anytls_idle_session_check_interval,omitempty" comment:"AnyTLS idle session check interval"`
	AnyTLSIdleSessionTimeout       string `json:"anytls_idle_session_timeout,omitempty" comment:"AnyTLS idle session timeout"`
	AnyTLSMinIdleSession           int    `json:"anytls_min_idle_session,omitempty" comment:"AnyTLS minimum idle sessions"`
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
		GroupSIDs:         r.GroupSIDs,
		TransportProtocol: r.TransportProtocol,
		Host:              r.Host,
		Path:              r.Path,
		SNI:               r.SNI,
		AllowInsecure:     r.AllowInsecure,
		Route:             r.Route,
		// VLESS
		VLESSTransportType:     r.VLESSTransportType,
		VLESSFlow:              r.VLESSFlow,
		VLESSSecurity:          r.VLESSSecurity,
		VLESSSni:               r.VLESSSni,
		VLESSFingerprint:       r.VLESSFingerprint,
		VLESSAllowInsecure:     r.VLESSAllowInsecure,
		VLESSHost:              r.VLESSHost,
		VLESSPath:              r.VLESSPath,
		VLESSServiceName:       r.VLESSServiceName,
		VLESSRealityPrivateKey: r.VLESSRealityPrivateKey,
		VLESSRealityPublicKey:  r.VLESSRealityPublicKey,
		VLESSRealityShortID:    r.VLESSRealityShortID,
		VLESSRealitySpiderX:    r.VLESSRealitySpiderX,
		// VMess
		VMessAlterID:       r.VMessAlterID,
		VMessSecurity:      r.VMessSecurity,
		VMessTransportType: r.VMessTransportType,
		VMessHost:          r.VMessHost,
		VMessPath:          r.VMessPath,
		VMessServiceName:   r.VMessServiceName,
		VMessTLS:           r.VMessTLS,
		VMessSni:           r.VMessSni,
		VMessAllowInsecure: r.VMessAllowInsecure,
		// Hysteria2
		Hysteria2CongestionControl: r.Hysteria2CongestionControl,
		Hysteria2Obfs:              r.Hysteria2Obfs,
		Hysteria2ObfsPassword:      r.Hysteria2ObfsPassword,
		Hysteria2UpMbps:            r.Hysteria2UpMbps,
		Hysteria2DownMbps:          r.Hysteria2DownMbps,
		Hysteria2Sni:               r.Hysteria2Sni,
		Hysteria2AllowInsecure:     r.Hysteria2AllowInsecure,
		Hysteria2Fingerprint:       r.Hysteria2Fingerprint,
		// TUIC
		TUICCongestionControl: r.TUICCongestionControl,
		TUICUDPRelayMode:      r.TUICUDPRelayMode,
		TUICAlpn:              r.TUICAlpn,
		TUICSni:               r.TUICSni,
		TUICAllowInsecure:     r.TUICAllowInsecure,
		TUICDisableSNI:        r.TUICDisableSNI,
		// AnyTLS
		AnyTLSSni:                      r.AnyTLSSni,
		AnyTLSAllowInsecure:            r.AnyTLSAllowInsecure,
		AnyTLSFingerprint:              r.AnyTLSFingerprint,
		AnyTLSIdleSessionCheckInterval: r.AnyTLSIdleSessionCheckInterval,
		AnyTLSIdleSessionTimeout:       r.AnyTLSIdleSessionTimeout,
		AnyTLSMinIdleSession:           r.AnyTLSMinIdleSession,
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
	GroupSIDs        []string          `json:"group_sids,omitempty" example:"[\"rg_xK9mP2vL3nQ\"]" comment:"Resource group SIDs to associate with (empty array to remove all)"`
	MuteNotification *bool             `json:"mute_notification,omitempty" example:"false" comment:"Mute online/offline notifications for this node"`
	// Trojan specific fields
	TransportProtocol *string `json:"transport_protocol,omitempty" binding:"omitempty,oneof=tcp ws grpc" example:"ws" comment:"Transport protocol for Trojan (tcp, ws, grpc)"`
	Host              *string `json:"host,omitempty" example:"cdn.example.com" comment:"WebSocket host header or gRPC service name"`
	Path              *string `json:"path,omitempty" example:"/trojan" comment:"WebSocket path"`
	SNI               *string `json:"sni,omitempty" example:"example.com" comment:"TLS Server Name Indication"`
	AllowInsecure     *bool   `json:"allow_insecure,omitempty" example:"false" comment:"Allow insecure TLS connection"`
	// Route configuration for traffic splitting (sing-box compatible)
	Route      *dto.RouteConfigDTO `json:"route,omitempty" comment:"Route configuration for traffic splitting"`
	ClearRoute bool                `json:"clear_route,omitempty" comment:"Set to true to clear route configuration"`

	// VLESS specific fields
	VLESSTransportType     *string `json:"vless_transport_type,omitempty" binding:"omitempty,oneof=tcp ws grpc h2" comment:"VLESS transport type"`
	VLESSFlow              *string `json:"vless_flow,omitempty" comment:"VLESS flow control"`
	VLESSSecurity          *string `json:"vless_security,omitempty" binding:"omitempty,oneof=none tls reality" comment:"VLESS security type"`
	VLESSSni               *string `json:"vless_sni,omitempty" comment:"VLESS TLS SNI"`
	VLESSFingerprint       *string `json:"vless_fingerprint,omitempty" comment:"VLESS TLS fingerprint"`
	VLESSAllowInsecure     *bool   `json:"vless_allow_insecure,omitempty" comment:"VLESS allow insecure TLS"`
	VLESSHost              *string `json:"vless_host,omitempty" comment:"VLESS WS/H2 host header"`
	VLESSPath              *string `json:"vless_path,omitempty" comment:"VLESS WS/H2 path"`
	VLESSServiceName       *string `json:"vless_service_name,omitempty" comment:"VLESS gRPC service name"`
	VLESSRealityPrivateKey *string `json:"vless_reality_private_key,omitempty" comment:"VLESS Reality private key (optional, auto-generated if empty)"`
	VLESSRealityPublicKey  *string `json:"vless_reality_public_key,omitempty" comment:"VLESS Reality public key (optional, auto-generated if empty)"`
	VLESSRealityShortID    *string `json:"vless_reality_short_id,omitempty" comment:"VLESS Reality short ID (optional, auto-generated if empty)"`
	VLESSRealitySpiderX    *string `json:"vless_reality_spider_x,omitempty" comment:"VLESS Reality spider X"`

	// VMess specific fields
	VMessAlterID       *int    `json:"vmess_alter_id,omitempty" comment:"VMess alter ID"`
	VMessSecurity      *string `json:"vmess_security,omitempty" binding:"omitempty,oneof=auto aes-128-gcm chacha20-poly1305 none zero" comment:"VMess security"`
	VMessTransportType *string `json:"vmess_transport_type,omitempty" binding:"omitempty,oneof=tcp ws grpc http quic" comment:"VMess transport type"`
	VMessHost          *string `json:"vmess_host,omitempty" comment:"VMess WS/HTTP host header"`
	VMessPath          *string `json:"vmess_path,omitempty" comment:"VMess WS/HTTP path"`
	VMessServiceName   *string `json:"vmess_service_name,omitempty" comment:"VMess gRPC service name"`
	VMessTLS           *bool   `json:"vmess_tls,omitempty" comment:"VMess TLS enabled"`
	VMessSni           *string `json:"vmess_sni,omitempty" comment:"VMess TLS SNI"`
	VMessAllowInsecure *bool   `json:"vmess_allow_insecure,omitempty" comment:"VMess allow insecure TLS"`

	// Hysteria2 specific fields
	Hysteria2CongestionControl *string `json:"hysteria2_congestion_control,omitempty" binding:"omitempty,oneof=cubic bbr new_reno" comment:"Hysteria2 congestion control"`
	Hysteria2Obfs              *string `json:"hysteria2_obfs,omitempty" binding:"omitempty,oneof=salamander" comment:"Hysteria2 obfuscation type"`
	Hysteria2ObfsPassword      *string `json:"hysteria2_obfs_password,omitempty" comment:"Hysteria2 obfuscation password"`
	Hysteria2UpMbps            *int    `json:"hysteria2_up_mbps,omitempty" comment:"Hysteria2 upstream bandwidth limit"`
	Hysteria2DownMbps          *int    `json:"hysteria2_down_mbps,omitempty" comment:"Hysteria2 downstream bandwidth limit"`
	Hysteria2Sni               *string `json:"hysteria2_sni,omitempty" comment:"Hysteria2 TLS SNI"`
	Hysteria2AllowInsecure     *bool   `json:"hysteria2_allow_insecure,omitempty" comment:"Hysteria2 allow insecure TLS"`
	Hysteria2Fingerprint       *string `json:"hysteria2_fingerprint,omitempty" comment:"Hysteria2 TLS fingerprint"`

	// TUIC specific fields
	TUICCongestionControl *string `json:"tuic_congestion_control,omitempty" binding:"omitempty,oneof=cubic bbr new_reno" comment:"TUIC congestion control"`
	TUICUDPRelayMode      *string `json:"tuic_udp_relay_mode,omitempty" binding:"omitempty,oneof=native quic" comment:"TUIC UDP relay mode"`
	TUICAlpn              *string `json:"tuic_alpn,omitempty" comment:"TUIC ALPN protocols"`
	TUICSni               *string `json:"tuic_sni,omitempty" comment:"TUIC TLS SNI"`
	TUICAllowInsecure     *bool   `json:"tuic_allow_insecure,omitempty" comment:"TUIC allow insecure TLS"`
	TUICDisableSNI        *bool   `json:"tuic_disable_sni,omitempty" comment:"TUIC disable SNI"`

	// AnyTLS specific fields
	AnyTLSSni                      *string `json:"anytls_sni,omitempty" comment:"AnyTLS TLS SNI"`
	AnyTLSAllowInsecure            *bool   `json:"anytls_allow_insecure,omitempty" comment:"AnyTLS allow insecure TLS"`
	AnyTLSFingerprint              *string `json:"anytls_fingerprint,omitempty" comment:"AnyTLS TLS fingerprint"`
	AnyTLSIdleSessionCheckInterval *string `json:"anytls_idle_session_check_interval,omitempty" comment:"AnyTLS idle session check interval"`
	AnyTLSIdleSessionTimeout       *string `json:"anytls_idle_session_timeout,omitempty" comment:"AnyTLS idle session timeout"`
	AnyTLSMinIdleSession           *int    `json:"anytls_min_idle_session,omitempty" comment:"AnyTLS minimum idle sessions"`

	// Expiration and cost label fields
	ExpiresAt *string `json:"expires_at,omitempty" example:"2025-12-31T23:59:59Z" comment:"Expiration time in ISO8601 format (empty string to clear, omit to keep unchanged)"`
	CostLabel *string `json:"cost_label,omitempty" example:"35$/m" comment:"Cost label for display (empty string to clear, omit to keep unchanged)"`
}

func (r *UpdateNodeRequest) ToCommand(sid string) usecases.UpdateNodeCommand {
	cmd := usecases.UpdateNodeCommand{
		SID:                     sid,
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
		GroupSIDs:               r.GroupSIDs,
		MuteNotification:        r.MuteNotification,
		TrojanTransportProtocol: r.TransportProtocol,
		TrojanHost:              r.Host,
		TrojanPath:              r.Path,
		TrojanSNI:               r.SNI,
		TrojanAllowInsecure:     r.AllowInsecure,
		Route:                   r.Route,
		ClearRoute:              r.ClearRoute,
		// VLESS
		VLESSTransportType:     r.VLESSTransportType,
		VLESSFlow:              r.VLESSFlow,
		VLESSSecurity:          r.VLESSSecurity,
		VLESSSni:               r.VLESSSni,
		VLESSFingerprint:       r.VLESSFingerprint,
		VLESSAllowInsecure:     r.VLESSAllowInsecure,
		VLESSHost:              r.VLESSHost,
		VLESSPath:              r.VLESSPath,
		VLESSServiceName:       r.VLESSServiceName,
		VLESSRealityPrivateKey: r.VLESSRealityPrivateKey,
		VLESSRealityPublicKey:  r.VLESSRealityPublicKey,
		VLESSRealityShortID:    r.VLESSRealityShortID,
		VLESSRealitySpiderX:    r.VLESSRealitySpiderX,
		// VMess
		VMessAlterID:       r.VMessAlterID,
		VMessSecurity:      r.VMessSecurity,
		VMessTransportType: r.VMessTransportType,
		VMessHost:          r.VMessHost,
		VMessPath:          r.VMessPath,
		VMessServiceName:   r.VMessServiceName,
		VMessTLS:           r.VMessTLS,
		VMessSni:           r.VMessSni,
		VMessAllowInsecure: r.VMessAllowInsecure,
		// Hysteria2
		Hysteria2CongestionControl: r.Hysteria2CongestionControl,
		Hysteria2Obfs:              r.Hysteria2Obfs,
		Hysteria2ObfsPassword:      r.Hysteria2ObfsPassword,
		Hysteria2UpMbps:            r.Hysteria2UpMbps,
		Hysteria2DownMbps:          r.Hysteria2DownMbps,
		Hysteria2Sni:               r.Hysteria2Sni,
		Hysteria2AllowInsecure:     r.Hysteria2AllowInsecure,
		Hysteria2Fingerprint:       r.Hysteria2Fingerprint,
		// TUIC
		TUICCongestionControl: r.TUICCongestionControl,
		TUICUDPRelayMode:      r.TUICUDPRelayMode,
		TUICAlpn:              r.TUICAlpn,
		TUICSni:               r.TUICSni,
		TUICAllowInsecure:     r.TUICAllowInsecure,
		TUICDisableSNI:        r.TUICDisableSNI,
		// AnyTLS
		AnyTLSSni:                      r.AnyTLSSni,
		AnyTLSAllowInsecure:            r.AnyTLSAllowInsecure,
		AnyTLSFingerprint:              r.AnyTLSFingerprint,
		AnyTLSIdleSessionCheckInterval: r.AnyTLSIdleSessionCheckInterval,
		AnyTLSIdleSessionTimeout:       r.AnyTLSIdleSessionTimeout,
		AnyTLSMinIdleSession:           r.AnyTLSMinIdleSession,
	}

	// Note: ExpiresAt is handled by the handler layer after ToCommand returns.
	// The handler parses, validates (format + future time), and sets cmd.ExpiresAt/cmd.ClearExpiresAt directly.
	// This ensures all validation is done in one place and avoids silent failures.

	// Handle CostLabel field
	// Note: length validation (max 50 chars) is done in handler layer before calling ToCommand
	if r.CostLabel != nil {
		if *r.CostLabel == "" {
			// Empty string means clear
			cmd.ClearCostLabel = true
		} else {
			cmd.CostLabel = r.CostLabel
		}
	}

	return cmd
}

type ListNodesRequest struct {
	Page             int
	PageSize         int
	Status           *string
	IncludeUserNodes bool
	SortBy           string
	SortOrder        string
}

func parseListNodesRequest(c *gin.Context) (*ListNodesRequest, error) {
	pagination := utils.ParsePagination(c)

	req := &ListNodesRequest{
		Page:      pagination.Page,
		PageSize:  pagination.PageSize,
		SortBy:    c.DefaultQuery("sort_by", "sort_order"),
		SortOrder: c.DefaultQuery("sort_order", "asc"),
	}

	if status := c.Query("status"); status != "" {
		req.Status = &status
	}

	// Parse include_user_nodes parameter (default: false - only show admin nodes)
	if includeUserNodes := c.Query("include_user_nodes"); includeUserNodes == "true" || includeUserNodes == "1" {
		req.IncludeUserNodes = true
	}

	return req, nil
}

func (r *ListNodesRequest) ToCommand() usecases.ListNodesQuery {
	offset := (r.Page - 1) * r.PageSize
	return usecases.ListNodesQuery{
		Limit:            r.PageSize,
		Offset:           offset,
		Status:           r.Status,
		IncludeUserNodes: r.IncludeUserNodes,
		SortBy:           r.SortBy,
		SortOrder:        r.SortOrder,
	}
}

// BatchInstallScriptRequest represents the request body for generating batch install script.
type BatchInstallScriptRequest struct {
	NodeIDs []string `json:"node_ids" binding:"required,min=1,max=100"`
}

// GetBatchInstallScript handles POST /nodes/batch-install-script
func (h *NodeHandler) GetBatchInstallScript(c *gin.Context) {
	var req BatchInstallScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for batch install script", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Use query param to override API URL if provided
	apiURL := c.Query("api_url")
	if apiURL == "" {
		apiURL = h.apiURL
	}

	query := usecases.GenerateBatchInstallScriptQuery{
		SIDs:   req.NodeIDs,
		APIURL: apiURL,
	}

	result, err := h.generateBatchInstallScriptUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Batch install command generated successfully", result)
}
