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
		metadata,
		0, // sortOrder not used for user nodes
		id.NewNodeID,
	)
	if err != nil {
		uc.logger.Errorw("failed to create node entity", "error", err)
		return nil, fmt.Errorf("failed to create node: %w", err)
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
