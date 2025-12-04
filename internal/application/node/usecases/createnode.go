package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/value_objects"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type CreateNodeCommand struct {
	Name             string
	ServerAddress    string
	AgentPort        uint16  // port for agent connections (required)
	SubscriptionPort *uint16 // port for client subscriptions (optional, defaults to AgentPort)
	Protocol         string
	Method           string
	Plugin           *string
	PluginOpts       map[string]string
	Region           string
	Tags             []string
	Description      string
	SortOrder        int
	// Trojan specific fields
	TransportProtocol string
	Host              string
	Path              string
	SNI               string
	AllowInsecure     bool
}

type CreateNodeResult struct {
	NodeID           uint
	APIToken         string
	TokenPrefix      string
	ServerAddress    string
	AgentPort        uint16
	SubscriptionPort *uint16
	Protocol         string
	Status           string
	CreatedAt        string
}

type CreateNodeUseCase struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

func NewCreateNodeUseCase(
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *CreateNodeUseCase {
	return &CreateNodeUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

func (uc *CreateNodeUseCase) Execute(ctx context.Context, cmd CreateNodeCommand) (*CreateNodeResult, error) {
	uc.logger.Infow("executing create node use case", "name", cmd.Name)

	// Validate command
	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid create node command", "error", err)
		return nil, err
	}

	// Check for duplicate node name
	exists, err := uc.nodeRepo.ExistsByName(ctx, cmd.Name)
	if err != nil {
		uc.logger.Errorw("failed to check existing node by name", "name", cmd.Name, "error", err)
		return nil, fmt.Errorf("failed to check existing node: %w", err)
	}
	if exists {
		uc.logger.Warnw("node with name already exists", "name", cmd.Name)
		return nil, errors.NewConflictError("node with this name already exists", cmd.Name)
	}

	// Check for duplicate server address and agent port
	exists, err = uc.nodeRepo.ExistsByAddress(ctx, cmd.ServerAddress, int(cmd.AgentPort))
	if err != nil {
		uc.logger.Errorw("failed to check existing node by address", "address", cmd.ServerAddress, "port", cmd.AgentPort, "error", err)
		return nil, fmt.Errorf("failed to check existing node: %w", err)
	}
	if exists {
		uc.logger.Warnw("node with address and port already exists", "address", cmd.ServerAddress, "port", cmd.AgentPort)
		return nil, errors.NewConflictError("node with this server address and port already exists", fmt.Sprintf("%s:%d", cmd.ServerAddress, cmd.AgentPort))
	}

	// Create value objects
	serverAddress, err := vo.NewServerAddress(cmd.ServerAddress)
	if err != nil {
		uc.logger.Errorw("invalid server address", "error", err)
		return nil, fmt.Errorf("invalid server address: %w", err)
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

	if protocol.IsShadowsocks() {
		encryptionConfig, err = vo.NewEncryptionConfig(cmd.Method)
		if err != nil {
			uc.logger.Errorw("invalid encryption config", "error", err)
			return nil, fmt.Errorf("invalid encryption config: %w", err)
		}

		// Create plugin config if plugin is specified
		if cmd.Plugin != nil && *cmd.Plugin != "" {
			pluginConfig, err = vo.NewPluginConfig(*cmd.Plugin, cmd.PluginOpts)
			if err != nil {
				uc.logger.Errorw("invalid plugin config", "error", err)
				return nil, fmt.Errorf("invalid plugin config: %w", err)
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
			return nil, fmt.Errorf("invalid trojan config: %w", err)
		}
		trojanConfig = &tc
	}

	// Create metadata
	metadata := vo.NewNodeMetadata(cmd.Region, cmd.Tags, cmd.Description)

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
		metadata,
		cmd.SortOrder,
	)
	if err != nil {
		uc.logger.Errorw("failed to create node entity", "error", err)
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	// Get the API token before persisting (it will be cleared after)
	apiToken := nodeEntity.GetAPIToken()

	// Persist the node
	if err := uc.nodeRepo.Create(ctx, nodeEntity); err != nil {
		uc.logger.Errorw("failed to persist node", "error", err)
		return nil, fmt.Errorf("failed to save node: %w", err)
	}

	// Extract token prefix for display (first 8 characters)
	tokenPrefix := ""
	if len(apiToken) >= 8 {
		tokenPrefix = apiToken[:8] + "..."
	}

	// Map to result
	result := &CreateNodeResult{
		NodeID:           nodeEntity.ID(),
		APIToken:         apiToken,
		TokenPrefix:      tokenPrefix,
		ServerAddress:    nodeEntity.ServerAddress().Value(),
		AgentPort:        nodeEntity.AgentPort(),
		SubscriptionPort: nodeEntity.SubscriptionPort(),
		Protocol:         nodeEntity.Protocol().String(),
		Status:           nodeEntity.Status().String(),
		CreatedAt:        nodeEntity.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	uc.logger.Infow("node created successfully", "id", result.NodeID, "name", cmd.Name)
	return result, nil
}

func (uc *CreateNodeUseCase) validateCommand(cmd CreateNodeCommand) error {
	if cmd.Name == "" {
		return errors.NewValidationError("node name is required")
	}

	if cmd.ServerAddress == "" {
		return errors.NewValidationError("server address is required")
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

	return nil
}

// validateProtocolMethodCompatibility validates that the encryption method matches the protocol type
func (uc *CreateNodeUseCase) validateProtocolMethodCompatibility(protocol vo.Protocol, method string) error {
	// Shadowsocks encryption methods
	ssMethods := map[string]bool{
		"aes-128-gcm":             true,
		"aes-256-gcm":             true,
		"aes-128-cfb":             true,
		"aes-192-cfb":             true,
		"aes-256-cfb":             true,
		"aes-128-ctr":             true,
		"aes-192-ctr":             true,
		"aes-256-ctr":             true,
		"chacha20-ietf":           true,
		"chacha20-ietf-poly1305":  true,
		"xchacha20-ietf-poly1305": true,
		"rc4-md5":                 true,
	}

	if protocol.IsShadowsocks() {
		if !ssMethods[method] {
			return errors.NewValidationError(fmt.Sprintf("encryption method '%s' is not compatible with Shadowsocks protocol", method))
		}
	}
	// Trojan doesn't require encryption method validation - it uses TLS

	return nil
}
