package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type CreateUserNodeCommand struct {
	UserID           uint
	Name             string
	ServerAddress    string
	AgentPort        uint16
	SubscriptionPort *uint16
	Protocol         string
	Method           string
	Plugin           *string
	PluginOpts       map[string]string
	// Trojan specific fields
	TransportProtocol string
	Host              string
	Path              string
	SNI               string
	AllowInsecure     bool

	// VLESS specific fields
	VLESSTransportType     string
	VLESSFlow              string
	VLESSSecurity          string
	VLESSSni               string
	VLESSFingerprint       string
	VLESSAllowInsecure     bool
	VLESSHost              string
	VLESSPath              string
	VLESSServiceName       string
	VLESSRealityPrivateKey string // Optional: auto-generated if empty for Reality security
	VLESSRealityPublicKey  string // Optional: auto-generated if empty for Reality security
	VLESSRealityShortID    string // Optional: auto-generated if empty for Reality security
	VLESSRealitySpiderX    string

	// VMess specific fields
	VMessAlterID       int
	VMessSecurity      string
	VMessTransportType string
	VMessHost          string
	VMessPath          string
	VMessServiceName   string
	VMessTLS           bool
	VMessSni           string
	VMessAllowInsecure bool

	// Hysteria2 specific fields
	Hysteria2CongestionControl string
	Hysteria2Obfs              string
	Hysteria2ObfsPassword      string
	Hysteria2UpMbps            *int
	Hysteria2DownMbps          *int
	Hysteria2Sni               string
	Hysteria2AllowInsecure     bool
	Hysteria2Fingerprint       string

	// TUIC specific fields
	TUICCongestionControl string
	TUICUDPRelayMode      string
	TUICAlpn              string
	TUICSni               string
	TUICAllowInsecure     bool
	TUICDisableSNI        bool
}

type CreateUserNodeResult struct {
	NodeSID          string
	APIToken         string
	ServerAddress    string
	AgentPort        uint16
	SubscriptionPort *uint16
	Protocol         string
	Status           string
	CreatedAt        string
}

type CreateUserNodeExecutor interface {
	Execute(ctx context.Context, cmd CreateUserNodeCommand) (*CreateUserNodeResult, error)
}

type CreateUserNodeUseCase struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

func NewCreateUserNodeUseCase(
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *CreateUserNodeUseCase {
	return &CreateUserNodeUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

func (uc *CreateUserNodeUseCase) Execute(ctx context.Context, cmd CreateUserNodeCommand) (*CreateUserNodeResult, error) {
	uc.logger.Infow("executing create user node use case", "user_id", cmd.UserID, "name", cmd.Name)

	// Validate command
	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid create user node command", "error", err)
		return nil, err
	}

	// Check for duplicate node name within user scope
	exists, err := uc.nodeRepo.ExistsByNameForUser(ctx, cmd.Name, cmd.UserID)
	if err != nil {
		uc.logger.Errorw("failed to check existing node by name", "name", cmd.Name, "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to check existing node: %w", err)
	}
	if exists {
		uc.logger.Warnw("node with name already exists for user", "name", cmd.Name, "user_id", cmd.UserID)
		return nil, errors.NewConflictError("node with this name already exists", cmd.Name)
	}

	// Check for duplicate server address and port within user scope (skip if address is empty)
	if cmd.ServerAddress != "" {
		exists, err = uc.nodeRepo.ExistsByAddressForUser(ctx, cmd.ServerAddress, int(cmd.AgentPort), cmd.UserID)
		if err != nil {
			uc.logger.Errorw("failed to check existing node by address", "address", cmd.ServerAddress, "port", cmd.AgentPort, "user_id", cmd.UserID, "error", err)
			return nil, fmt.Errorf("failed to check existing node: %w", err)
		}
		if exists {
			uc.logger.Warnw("node with address and port already exists for user", "address", cmd.ServerAddress, "port", cmd.AgentPort, "user_id", cmd.UserID)
			return nil, errors.NewConflictError("node with this server address and port already exists", fmt.Sprintf("%s:%d", cmd.ServerAddress, cmd.AgentPort))
		}
	}

	// Create value objects
	serverAddress, err := vo.NewServerAddress(cmd.ServerAddress)
	if err != nil {
		uc.logger.Errorw("invalid server address", "error", err)
		return nil, err
	}

	// Validate and create protocol
	protocol := vo.Protocol(cmd.Protocol)
	if !protocol.IsValid() {
		uc.logger.Errorw("invalid protocol", "protocol", cmd.Protocol)
		return nil, errors.NewValidationError(fmt.Sprintf("unsupported protocol: %s", cmd.Protocol))
	}

	// Validate protocol and method compatibility
	if err := uc.validateProtocolMethodCompatibility(protocol, cmd.Method); err != nil {
		uc.logger.Errorw("protocol and method mismatch", "protocol", cmd.Protocol, "method", cmd.Method, "error", err)
		return nil, err
	}

	// Create encryption config for Shadowsocks
	var encryptionConfig vo.EncryptionConfig
	var pluginConfig *vo.PluginConfig
	var trojanConfig *vo.TrojanConfig
	var vlessConfig *vo.VLESSConfig
	var vmessConfig *vo.VMessConfig
	var hysteria2Config *vo.Hysteria2Config
	var tuicConfig *vo.TUICConfig

	if protocol.IsShadowsocks() {
		encryptionConfig, err = vo.NewEncryptionConfig(cmd.Method)
		if err != nil {
			uc.logger.Errorw("invalid encryption config", "error", err)
			return nil, err
		}

		// Create plugin config if plugin is specified
		if cmd.Plugin != nil && *cmd.Plugin != "" {
			pluginConfig, err = vo.NewPluginConfig(*cmd.Plugin, cmd.PluginOpts)
			if err != nil {
				uc.logger.Errorw("invalid plugin config", "error", err)
				return nil, err
			}
		}
	} else if protocol.IsTrojan() {
		// Create Trojan config
		// Default transport protocol to tcp if not specified
		transportProtocol := cmd.TransportProtocol
		if transportProtocol == "" {
			transportProtocol = "tcp"
		}

		// For self-signed certificates, we use a placeholder password
		// The actual password will be derived from subscription UUID
		tc, err := vo.NewTrojanConfig(
			"placeholder", // Password will be replaced by subscription UUID
			transportProtocol,
			cmd.Host,
			cmd.Path,
			cmd.AllowInsecure,
			cmd.SNI,
		)
		if err != nil {
			uc.logger.Errorw("invalid trojan config", "error", err)
			return nil, err
		}
		trojanConfig = &tc
	} else if protocol.IsVLESS() {
		// Create VLESS config
		// Default transport type to tcp if not specified
		transportType := cmd.VLESSTransportType
		if transportType == "" {
			transportType = "tcp"
		}
		// Default security to tls if not specified
		security := cmd.VLESSSecurity
		if security == "" {
			security = "tls"
		}

		// Auto-generate Reality key pair and short ID if security is reality and not provided
		privateKey := cmd.VLESSRealityPrivateKey
		publicKey := cmd.VLESSRealityPublicKey
		shortID := cmd.VLESSRealityShortID

		if security == vo.VLESSSecurityReality && privateKey == "" && publicKey == "" && shortID == "" {
			// Auto-generate key pair
			keyPair, err := vo.GenerateRealityKeyPair()
			if err != nil {
				uc.logger.Errorw("failed to generate Reality key pair", "error", err)
				return nil, fmt.Errorf("failed to generate Reality key pair: %w", err)
			}
			privateKey = keyPair.PrivateKey
			publicKey = keyPair.PublicKey

			// Auto-generate short ID
			shortID, err = vo.GenerateRealityShortID()
			if err != nil {
				uc.logger.Errorw("failed to generate Reality short ID", "error", err)
				return nil, fmt.Errorf("failed to generate Reality short ID: %w", err)
			}

			uc.logger.Infow("auto-generated Reality key pair and short ID for user VLESS node",
				"user_id", cmd.UserID,
				"name", cmd.Name,
				"public_key_prefix", publicKey[:8]+"...",
			)
		}

		vc, err := vo.NewVLESSConfig(
			transportType,
			cmd.VLESSFlow,
			security,
			cmd.VLESSSni,
			cmd.VLESSFingerprint,
			cmd.VLESSAllowInsecure,
			cmd.VLESSHost,
			cmd.VLESSPath,
			cmd.VLESSServiceName,
			privateKey,
			publicKey,
			shortID,
			cmd.VLESSRealitySpiderX,
		)
		if err != nil {
			uc.logger.Errorw("invalid VLESS config", "error", err)
			return nil, err
		}
		vlessConfig = &vc
	} else if protocol.IsVMess() {
		// Create VMess config
		// Default transport type to tcp if not specified
		transportType := cmd.VMessTransportType
		if transportType == "" {
			transportType = "tcp"
		}
		// Default security to auto if not specified
		security := cmd.VMessSecurity
		if security == "" {
			security = "auto"
		}

		vc, err := vo.NewVMessConfig(
			cmd.VMessAlterID,
			security,
			transportType,
			cmd.VMessHost,
			cmd.VMessPath,
			cmd.VMessServiceName,
			cmd.VMessTLS,
			cmd.VMessSni,
			cmd.VMessAllowInsecure,
		)
		if err != nil {
			uc.logger.Errorw("invalid VMess config", "error", err)
			return nil, err
		}
		vmessConfig = &vc
	} else if protocol.IsHysteria2() {
		// Create Hysteria2 config
		// Default congestion control to bbr if not specified
		cc := cmd.Hysteria2CongestionControl
		if cc == "" {
			cc = "bbr"
		}

		hc, err := vo.NewHysteria2Config(
			"placeholder", // Password will be replaced by subscription UUID
			cc,
			cmd.Hysteria2Obfs,
			cmd.Hysteria2ObfsPassword,
			cmd.Hysteria2UpMbps,
			cmd.Hysteria2DownMbps,
			cmd.Hysteria2Sni,
			cmd.Hysteria2AllowInsecure,
			cmd.Hysteria2Fingerprint,
		)
		if err != nil {
			uc.logger.Errorw("invalid Hysteria2 config", "error", err)
			return nil, err
		}
		hysteria2Config = &hc
	} else if protocol.IsTUIC() {
		// Create TUIC config
		// Default congestion control to bbr if not specified
		cc := cmd.TUICCongestionControl
		if cc == "" {
			cc = "bbr"
		}
		// Default UDP relay mode to native if not specified
		relayMode := cmd.TUICUDPRelayMode
		if relayMode == "" {
			relayMode = "native"
		}

		tc, err := vo.NewTUICConfig(
			"placeholder", // UUID will be replaced by subscription UUID
			"placeholder", // Password will be replaced by subscription UUID
			cc,
			relayMode,
			cmd.TUICAlpn,
			cmd.TUICSni,
			cmd.TUICAllowInsecure,
			cmd.TUICDisableSNI,
		)
		if err != nil {
			uc.logger.Errorw("invalid TUIC config", "error", err)
			return nil, err
		}
		tuicConfig = &tc
	}

	// Create metadata (user nodes don't have region/tags/description)
	metadata := vo.NewNodeMetadata("", nil, "")

	// Create node aggregate using domain constructor
	nodeEntity, err := node.NewNode(
		cmd.Name,
		serverAddress,
		cmd.AgentPort,
		cmd.SubscriptionPort,
		protocol,
		encryptionConfig,
		pluginConfig,
		trojanConfig,
		vlessConfig,
		vmessConfig,
		hysteria2Config,
		tuicConfig,
		metadata,
		0,   // sortOrder not used for user nodes
		nil, // routeConfig - can be set later via UpdateRouteConfig
		id.NewNodeID,
	)
	if err != nil {
		uc.logger.Errorw("failed to create node entity", "error", err)
		return nil, err
	}

	// Set user ownership
	nodeEntity.SetUserID(&cmd.UserID)

	// Activate user node by default
	if err := nodeEntity.Activate(); err != nil {
		uc.logger.Warnw("failed to activate user node", "error", err)
		// Continue even if activation fails - node will be in initial state
	}

	// Get the API token before persisting (it will be cleared after)
	apiToken := nodeEntity.GetAPIToken()

	// Persist the node
	if err := uc.nodeRepo.Create(ctx, nodeEntity); err != nil {
		uc.logger.Errorw("failed to persist user node", "user_id", cmd.UserID, "error", err)
		return nil, fmt.Errorf("failed to save node: %w", err)
	}

	// Map to result
	result := &CreateUserNodeResult{
		NodeSID:          nodeEntity.SID(),
		APIToken:         apiToken,
		ServerAddress:    nodeEntity.ServerAddress().Value(),
		AgentPort:        nodeEntity.AgentPort(),
		SubscriptionPort: nodeEntity.SubscriptionPort(),
		Protocol:         nodeEntity.Protocol().String(),
		Status:           nodeEntity.Status().String(),
		CreatedAt:        nodeEntity.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	uc.logger.Infow("user node created successfully", "user_id", cmd.UserID, "node_sid", result.NodeSID)
	return result, nil
}

func (uc *CreateUserNodeUseCase) validateCommand(cmd CreateUserNodeCommand) error {
	if cmd.UserID == 0 {
		return errors.NewValidationError("user ID is required")
	}

	if cmd.Name == "" {
		return errors.NewValidationError("node name is required")
	}

	if cmd.AgentPort == 0 {
		return errors.NewValidationError("agent port is required")
	}

	if cmd.Protocol == "" {
		return errors.NewValidationError("protocol is required")
	}

	// Encryption method is required only for Shadowsocks
	if cmd.Protocol == "shadowsocks" && cmd.Method == "" {
		return errors.NewValidationError("encryption method is required for Shadowsocks protocol")
	}

	// Validate Trojan-specific requirements
	if cmd.Protocol == "trojan" {
		// Validate transport protocol if specified
		if cmd.TransportProtocol != "" &&
			cmd.TransportProtocol != "tcp" &&
			cmd.TransportProtocol != "ws" &&
			cmd.TransportProtocol != "grpc" {
			return errors.NewValidationError("invalid transport protocol for Trojan (must be tcp, ws, or grpc)")
		}

		// WebSocket requires host and path
		if cmd.TransportProtocol == "ws" {
			if cmd.Host == "" {
				return errors.NewValidationError("host is required for WebSocket transport")
			}
			if cmd.Path == "" {
				return errors.NewValidationError("path is required for WebSocket transport")
			}
		}

		// gRPC requires host (service name)
		if cmd.TransportProtocol == "grpc" && cmd.Host == "" {
			return errors.NewValidationError("host (service name) is required for gRPC transport")
		}
	}

	// Validate VLESS-specific requirements
	if cmd.Protocol == "vless" {
		if err := uc.validateVLESSCommand(cmd); err != nil {
			return err
		}
	}

	// Validate VMess-specific requirements
	if cmd.Protocol == "vmess" {
		if err := uc.validateVMessCommand(cmd); err != nil {
			return err
		}
	}

	// Validate Hysteria2-specific requirements
	if cmd.Protocol == "hysteria2" {
		if err := uc.validateHysteria2Command(cmd); err != nil {
			return err
		}
	}

	// Validate TUIC-specific requirements
	if cmd.Protocol == "tuic" {
		if err := uc.validateTUICCommand(cmd); err != nil {
			return err
		}
	}

	return nil
}

// validateVLESSCommand validates VLESS protocol specific requirements
func (uc *CreateUserNodeUseCase) validateVLESSCommand(cmd CreateUserNodeCommand) error {
	// Validate transport type
	validTransports := map[string]bool{"tcp": true, "ws": true, "grpc": true, "h2": true}
	if cmd.VLESSTransportType != "" && !validTransports[cmd.VLESSTransportType] {
		return errors.NewValidationError("invalid VLESS transport type (must be tcp, ws, grpc, or h2)")
	}

	// Validate security type
	validSecurity := map[string]bool{"none": true, "tls": true, "reality": true}
	if cmd.VLESSSecurity != "" && !validSecurity[cmd.VLESSSecurity] {
		return errors.NewValidationError("invalid VLESS security type (must be none, tls, or reality)")
	}

	// Validate flow control
	if cmd.VLESSFlow != "" && cmd.VLESSFlow != "xtls-rprx-vision" {
		return errors.NewValidationError("invalid VLESS flow (must be empty or xtls-rprx-vision)")
	}

	// Reality key validation: if any key is provided, all must be provided
	// If none provided, they will be auto-generated
	if cmd.VLESSSecurity == "reality" {
		hasPrivateKey := cmd.VLESSRealityPrivateKey != ""
		hasPublicKey := cmd.VLESSRealityPublicKey != ""
		hasShortID := cmd.VLESSRealityShortID != ""

		// Partial configuration is not allowed
		if (hasPrivateKey || hasPublicKey || hasShortID) && !(hasPrivateKey && hasPublicKey && hasShortID) {
			return errors.NewValidationError("for VLESS Reality security, either provide all of private_key, public_key, and short_id, or leave all empty for auto-generation")
		}
	}

	// WebSocket/H2 requires host and path
	if cmd.VLESSTransportType == "ws" || cmd.VLESSTransportType == "h2" {
		if cmd.VLESSHost == "" {
			return errors.NewValidationError("host is required for VLESS WebSocket/H2 transport")
		}
		if cmd.VLESSPath == "" {
			return errors.NewValidationError("path is required for VLESS WebSocket/H2 transport")
		}
	}

	// gRPC requires service name
	if cmd.VLESSTransportType == "grpc" && cmd.VLESSServiceName == "" {
		return errors.NewValidationError("service name is required for VLESS gRPC transport")
	}

	return nil
}

// validateVMessCommand validates VMess protocol specific requirements
func (uc *CreateUserNodeUseCase) validateVMessCommand(cmd CreateUserNodeCommand) error {
	// Validate alter ID
	if cmd.VMessAlterID < 0 {
		return errors.NewValidationError("VMess alterID must be non-negative")
	}

	// Validate security type
	validSecurity := map[string]bool{"auto": true, "aes-128-gcm": true, "chacha20-poly1305": true, "none": true, "zero": true}
	if cmd.VMessSecurity != "" && !validSecurity[cmd.VMessSecurity] {
		return errors.NewValidationError("invalid VMess security type (must be auto, aes-128-gcm, chacha20-poly1305, none, or zero)")
	}

	// Validate transport type
	validTransports := map[string]bool{"tcp": true, "ws": true, "grpc": true, "http": true, "quic": true}
	if cmd.VMessTransportType != "" && !validTransports[cmd.VMessTransportType] {
		return errors.NewValidationError("invalid VMess transport type (must be tcp, ws, grpc, http, or quic)")
	}

	// WebSocket requires path
	if cmd.VMessTransportType == "ws" && cmd.VMessPath == "" {
		return errors.NewValidationError("path is required for VMess WebSocket transport")
	}

	// HTTP requires path
	if cmd.VMessTransportType == "http" && cmd.VMessPath == "" {
		return errors.NewValidationError("path is required for VMess HTTP transport")
	}

	// gRPC requires service name
	if cmd.VMessTransportType == "grpc" && cmd.VMessServiceName == "" {
		return errors.NewValidationError("service name is required for VMess gRPC transport")
	}

	return nil
}

// validateHysteria2Command validates Hysteria2 protocol specific requirements
func (uc *CreateUserNodeUseCase) validateHysteria2Command(cmd CreateUserNodeCommand) error {
	// Validate congestion control
	validCC := map[string]bool{"cubic": true, "bbr": true, "new_reno": true}
	if cmd.Hysteria2CongestionControl != "" && !validCC[cmd.Hysteria2CongestionControl] {
		return errors.NewValidationError("invalid Hysteria2 congestion control (must be cubic, bbr, or new_reno)")
	}

	// Validate obfs type
	if cmd.Hysteria2Obfs != "" && cmd.Hysteria2Obfs != "salamander" {
		return errors.NewValidationError("invalid Hysteria2 obfs type (must be empty or salamander)")
	}

	// Salamander obfs requires password
	if cmd.Hysteria2Obfs == "salamander" && cmd.Hysteria2ObfsPassword == "" {
		return errors.NewValidationError("obfs password is required for Hysteria2 Salamander obfuscation")
	}

	// Validate bandwidth limits
	if cmd.Hysteria2UpMbps != nil && *cmd.Hysteria2UpMbps < 0 {
		return errors.NewValidationError("Hysteria2 up_mbps must be non-negative")
	}
	if cmd.Hysteria2DownMbps != nil && *cmd.Hysteria2DownMbps < 0 {
		return errors.NewValidationError("Hysteria2 down_mbps must be non-negative")
	}

	return nil
}

// validateTUICCommand validates TUIC protocol specific requirements
func (uc *CreateUserNodeUseCase) validateTUICCommand(cmd CreateUserNodeCommand) error {
	// Validate congestion control
	validCC := map[string]bool{"cubic": true, "bbr": true, "new_reno": true}
	if cmd.TUICCongestionControl != "" && !validCC[cmd.TUICCongestionControl] {
		return errors.NewValidationError("invalid TUIC congestion control (must be cubic, bbr, or new_reno)")
	}

	// Validate UDP relay mode
	validRelayMode := map[string]bool{"native": true, "quic": true}
	if cmd.TUICUDPRelayMode != "" && !validRelayMode[cmd.TUICUDPRelayMode] {
		return errors.NewValidationError("invalid TUIC UDP relay mode (must be native or quic)")
	}

	return nil
}

// validateProtocolMethodCompatibility validates that the encryption method matches the protocol type
func (uc *CreateUserNodeUseCase) validateProtocolMethodCompatibility(protocol vo.Protocol, method string) error {
	// Shadowsocks encryption methods
	ssMethods := map[string]bool{
		"aes-128-gcm":                   true,
		"aes-256-gcm":                   true,
		"aes-128-cfb":                   true,
		"aes-192-cfb":                   true,
		"aes-256-cfb":                   true,
		"aes-128-ctr":                   true,
		"aes-192-ctr":                   true,
		"aes-256-ctr":                   true,
		"chacha20-ietf":                 true,
		"chacha20-ietf-poly1305":        true,
		"xchacha20-ietf-poly1305":       true,
		"rc4-md5":                       true,
		"2022-blake3-aes-128-gcm":       true,
		"2022-blake3-aes-256-gcm":       true,
		"2022-blake3-chacha20-poly1305": true,
	}

	if protocol.IsShadowsocks() {
		if !ssMethods[method] {
			return errors.NewValidationError(fmt.Sprintf("encryption method '%s' is not compatible with Shadowsocks protocol", method))
		}
	}
	// Trojan doesn't require encryption method validation - it uses TLS

	return nil
}
