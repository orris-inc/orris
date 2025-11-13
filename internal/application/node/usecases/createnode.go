package usecases

import (
	"context"
	"fmt"

	"orris/internal/domain/node"
	vo "orris/internal/domain/node/value_objects"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type CreateNodeCommand struct {
	Name          string
	ServerAddress string
	ServerPort    uint16
	Protocol      string
	Method        string
	Plugin        *string
	PluginOpts    map[string]string
	Region        string
	Tags          []string
	Description   string
	SortOrder     int
}

type CreateNodeResult struct {
	NodeID        uint
	APIToken      string
	TokenPrefix   string
	ServerAddress string
	ServerPort    uint16
	Status        string
	CreatedAt     string
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

	// Check for duplicate server address and port
	exists, err = uc.nodeRepo.ExistsByAddress(ctx, cmd.ServerAddress, int(cmd.ServerPort))
	if err != nil {
		uc.logger.Errorw("failed to check existing node by address", "address", cmd.ServerAddress, "port", cmd.ServerPort, "error", err)
		return nil, fmt.Errorf("failed to check existing node: %w", err)
	}
	if exists {
		uc.logger.Warnw("node with address and port already exists", "address", cmd.ServerAddress, "port", cmd.ServerPort)
		return nil, errors.NewConflictError("node with this server address and port already exists", fmt.Sprintf("%s:%d", cmd.ServerAddress, cmd.ServerPort))
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
	}

	// For Trojan protocol, we would need trojan config parameters in the command
	// Currently the command doesn't have those fields, so we skip it for now
	// This should be added when Trojan support is needed

	// Create metadata
	metadata := vo.NewNodeMetadata(cmd.Region, cmd.Tags, cmd.Description)

	// Create node aggregate using domain constructor
	nodeEntity, err := node.NewNode(
		cmd.Name,
		serverAddress,
		cmd.ServerPort,
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
		NodeID:        nodeEntity.ID(),
		APIToken:      apiToken,
		TokenPrefix:   tokenPrefix,
		ServerAddress: nodeEntity.ServerAddress().Value(),
		ServerPort:    nodeEntity.ServerPort(),
		Status:        nodeEntity.Status().String(),
		CreatedAt:     nodeEntity.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
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

	if cmd.ServerPort == 0 {
		return errors.NewValidationError("server port is required")
	}

	if cmd.Protocol == "" {
		return errors.NewValidationError("protocol is required")
	}

	if cmd.Method == "" {
		return errors.NewValidationError("encryption method is required")
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
	} else if protocol.IsTrojan() {
		// Trojan doesn't use these encryption methods, it uses TLS
		if ssMethods[method] {
			return errors.NewValidationError(fmt.Sprintf("encryption method '%s' is not compatible with Trojan protocol", method))
		}
	}

	return nil
}
