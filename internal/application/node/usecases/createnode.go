package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
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
	// Route configuration for traffic splitting
	Route *dto.RouteConfigDTO
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
	// Validate command
	if err := uc.validateCommand(cmd); err != nil {
		return nil, err
	}

	// Check for duplicate node name
	exists, err := uc.nodeRepo.ExistsByName(ctx, cmd.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing node: %w", err)
	}
	if exists {
		return nil, errors.NewConflictError("node with this name already exists", cmd.Name)
	}

	// Check for duplicate server address and agent port (only when address is specified)
	if cmd.ServerAddress != "" {
		exists, err = uc.nodeRepo.ExistsByAddress(ctx, cmd.ServerAddress, int(cmd.AgentPort))
		if err != nil {
			return nil, fmt.Errorf("failed to check existing node: %w", err)
		}
		if exists {
			return nil, errors.NewConflictError("node with this server address and port already exists", fmt.Sprintf("%s:%d", cmd.ServerAddress, cmd.AgentPort))
		}
	}

	// Create value objects
	serverAddress, err := vo.NewServerAddress(cmd.ServerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid server address: %w", err)
	}

	// Validate and create protocol
	protocol := vo.Protocol(cmd.Protocol)
	if !protocol.IsValid() {
		return nil, errors.NewValidationError(fmt.Sprintf("unsupported protocol: %s", cmd.Protocol))
	}

	// Validate protocol and method compatibility
	if err := uc.validateProtocolMethodCompatibility(protocol, cmd.Method); err != nil {
		return nil, err
	}

	// Create encryption config for Shadowsocks
	var encryptionConfig vo.EncryptionConfig
	var pluginConfig *vo.PluginConfig
	var trojanConfig *vo.TrojanConfig

	if protocol.IsShadowsocks() {
		encryptionConfig, err = vo.NewEncryptionConfig(cmd.Method)
		if err != nil {
			return nil, fmt.Errorf("invalid encryption config: %w", err)
		}

		// Create plugin config if plugin is specified
		if cmd.Plugin != nil && *cmd.Plugin != "" {
			pluginConfig, err = vo.NewPluginConfig(*cmd.Plugin, cmd.PluginOpts)
			if err != nil {
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
			return nil, fmt.Errorf("invalid trojan config: %w", err)
		}
		trojanConfig = &tc
	}

	// Create metadata
	metadata := vo.NewNodeMetadata(cmd.Region, cmd.Tags, cmd.Description)

	// Convert route config from DTO if provided
	var routeConfig *vo.RouteConfig
	if cmd.Route != nil {
		routeConfig, err = dto.FromRouteConfigDTO(cmd.Route)
		if err != nil {
			return nil, fmt.Errorf("invalid route config: %w", err)
		}

		// Validate node references in route config (admin nodes can reference any existing node)
		if err := uc.validateRouteConfigNodeReferences(ctx, routeConfig); err != nil {
			return nil, err
		}
	}

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
		routeConfig,
		id.NewNodeID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	// Get the API token before persisting (it will be cleared after)
	apiToken := nodeEntity.GetAPIToken()

	// Persist the node
	if err := uc.nodeRepo.Create(ctx, nodeEntity); err != nil {
		uc.logger.Errorw("failed to persist node", "error", err, "name", cmd.Name)
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

	// ServerAddress is optional - will use public IP as fallback

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

// validateRouteConfigNodeReferences validates that all node SIDs referenced in route config exist.
// For admin nodes (created via this use case), any existing node can be referenced.
func (uc *CreateNodeUseCase) validateRouteConfigNodeReferences(ctx context.Context, routeConfig *vo.RouteConfig) error {
	if routeConfig == nil {
		return nil
	}

	referencedSIDs := routeConfig.GetReferencedNodeSIDs()
	if len(referencedSIDs) == 0 {
		return nil
	}

	// Admin nodes can reference any existing node
	invalidSIDs, err := uc.nodeRepo.ValidateNodeSIDsExist(ctx, referencedSIDs)
	if err != nil {
		uc.logger.Errorw("failed to validate route config node references", "error", err)
		return errors.NewInternalError("failed to validate route config")
	}

	if len(invalidSIDs) > 0 {
		return errors.NewValidationError(
			fmt.Sprintf("invalid node SIDs in route config (not found): %v", invalidSIDs))
	}

	return nil
}
