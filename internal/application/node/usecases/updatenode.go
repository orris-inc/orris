package usecases

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/value_objects"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type UpdateNodeCommand struct {
	NodeID        uint
	Name          *string
	ServerAddress *string
	ServerPort    *uint16
	Method        *string
	Plugin        *string
	PluginOpts    map[string]string
	Region        *string
	Tags          []string
	Description   *string
	SortOrder     *int
	Status        *string
}

type UpdateNodeResult struct {
	NodeID        uint
	Name          string
	ServerAddress string
	ServerPort    uint16
	Status        string
	UpdatedAt     string
}

type UpdateNodeUseCase struct {
	logger   logger.Interface
	nodeRepo node.NodeRepository
}

func NewUpdateNodeUseCase(
	logger logger.Interface,
	nodeRepo node.NodeRepository,
) *UpdateNodeUseCase {
	return &UpdateNodeUseCase{
		logger:   logger,
		nodeRepo: nodeRepo,
	}
}

func (uc *UpdateNodeUseCase) Execute(ctx context.Context, cmd UpdateNodeCommand) (*UpdateNodeResult, error) {
	uc.logger.Infow("executing update node use case", "node_id", cmd.NodeID)

	// Validate command
	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid update node command", "error", err, "node_id", cmd.NodeID)
		return nil, err
	}

	// Get existing node from repository
	existingNode, err := uc.nodeRepo.GetByID(ctx, cmd.NodeID)
	if err != nil {
		uc.logger.Errorw("failed to get node", "error", err, "node_id", cmd.NodeID)
		return nil, errors.NewNotFoundError("node not found")
	}

	// Apply updates based on command fields
	if err := uc.applyUpdates(existingNode, cmd); err != nil {
		uc.logger.Errorw("failed to apply updates", "error", err, "node_id", cmd.NodeID)
		return nil, err
	}

	// Save updated node with optimistic locking
	if err := uc.nodeRepo.Update(ctx, existingNode); err != nil {
		uc.logger.Errorw("failed to update node", "error", err, "node_id", cmd.NodeID)
		// Check if it's an optimistic lock error
		if errors.IsConflictError(err) {
			return nil, errors.NewConflictError("node was modified by another process, please retry")
		}
		return nil, errors.NewInternalError("failed to update node")
	}

	uc.logger.Infow("node updated successfully", "node_id", cmd.NodeID)

	// Build and return result
	return &UpdateNodeResult{
		NodeID:        existingNode.ID(),
		Name:          existingNode.Name(),
		ServerAddress: existingNode.ServerAddress().Value(),
		ServerPort:    existingNode.ServerPort(),
		Status:        existingNode.Status().String(),
		UpdatedAt:     existingNode.UpdatedAt().Format(time.RFC3339),
	}, nil
}

// applyUpdates applies all updates from command to the node domain object
func (uc *UpdateNodeUseCase) applyUpdates(n *node.Node, cmd UpdateNodeCommand) error {
	// Update name
	if cmd.Name != nil {
		if err := n.UpdateName(*cmd.Name); err != nil {
			return errors.NewValidationError("invalid node name: " + err.Error())
		}
	}

	// Update server address
	if cmd.ServerAddress != nil {
		serverAddr, err := vo.NewServerAddress(*cmd.ServerAddress)
		if err != nil {
			return errors.NewValidationError("invalid server address: " + err.Error())
		}
		if err := n.UpdateServerAddress(serverAddr); err != nil {
			return errors.NewValidationError("failed to update server address: " + err.Error())
		}
	}

	// Update server port
	if cmd.ServerPort != nil {
		if err := n.UpdateServerPort(*cmd.ServerPort); err != nil {
			return errors.NewValidationError("invalid server port: " + err.Error())
		}
	}

	// Update encryption config (method only)
	// Note: Protocol type cannot be changed after node creation
	// Only the encryption method within the same protocol can be updated
	if cmd.Method != nil {
		// Validate that the new method is compatible with the existing protocol
		if err := uc.validateProtocolMethodCompatibility(n.Protocol(), *cmd.Method); err != nil {
			return err
		}

		encryptionConfig, err := vo.NewEncryptionConfig(*cmd.Method)
		if err != nil {
			return errors.NewValidationError("invalid encryption config: " + err.Error())
		}
		if err := n.UpdateEncryption(encryptionConfig); err != nil {
			return errors.NewValidationError("failed to update encryption: " + err.Error())
		}
	}

	// Update plugin config
	if cmd.Plugin != nil {
		var pluginConfig *vo.PluginConfig
		if *cmd.Plugin != "" {
			var err error
			pluginConfig, err = vo.NewPluginConfig(*cmd.Plugin, cmd.PluginOpts)
			if err != nil {
				return errors.NewValidationError("invalid plugin config: " + err.Error())
			}
		}
		if err := n.UpdatePlugin(pluginConfig); err != nil {
			return errors.NewValidationError("failed to update plugin: " + err.Error())
		}
	}

	// Update metadata (region, tags, description)
	needMetadataUpdate := cmd.Region != nil || cmd.Tags != nil || cmd.Description != nil
	if needMetadataUpdate {
		currentMeta := n.Metadata()
		region := currentMeta.Region()
		tags := currentMeta.Tags()
		description := currentMeta.Description()

		if cmd.Region != nil {
			region = *cmd.Region
		}
		if cmd.Tags != nil {
			tags = cmd.Tags
		}
		if cmd.Description != nil {
			description = *cmd.Description
		}

		newMetadata := vo.NewNodeMetadata(region, tags, description)
		if err := n.UpdateMetadata(newMetadata); err != nil {
			return errors.NewValidationError("failed to update metadata: " + err.Error())
		}
	}

	// Update sort order
	if cmd.SortOrder != nil {
		if err := n.UpdateSortOrder(*cmd.SortOrder); err != nil {
			return errors.NewValidationError("failed to update sort order: " + err.Error())
		}
	}

	// Update status
	if cmd.Status != nil {
		status := vo.NodeStatus(*cmd.Status)
		if !status.IsValid() {
			return errors.NewValidationError("invalid node status: " + *cmd.Status)
		}

		// Handle different status transitions
		switch status {
		case vo.NodeStatusActive:
			if err := n.Activate(); err != nil {
				return errors.NewValidationError("failed to activate node: " + err.Error())
			}
		case vo.NodeStatusInactive:
			if err := n.Deactivate(); err != nil {
				return errors.NewValidationError("failed to deactivate node: " + err.Error())
			}
		case vo.NodeStatusMaintenance:
			// For maintenance, we need a reason but it's not in the command
			// Use a default reason or require it in the command
			reason := "Maintenance mode set via update"
			if err := n.EnterMaintenance(reason); err != nil {
				return errors.NewValidationError("failed to enter maintenance: " + err.Error())
			}
		}
	}

	return nil
}

func (uc *UpdateNodeUseCase) validateCommand(cmd UpdateNodeCommand) error {
	if cmd.NodeID == 0 {
		return errors.NewValidationError("node id is required")
	}

	if cmd.Name == nil && cmd.ServerAddress == nil && cmd.ServerPort == nil &&
		cmd.Method == nil && cmd.Region == nil && cmd.Tags == nil && cmd.Description == nil &&
		cmd.SortOrder == nil && cmd.Status == nil {
		return errors.NewValidationError("at least one field must be provided for update")
	}

	if cmd.Name != nil && *cmd.Name == "" {
		return errors.NewValidationError("node name cannot be empty")
	}

	if cmd.ServerAddress != nil && *cmd.ServerAddress == "" {
		return errors.NewValidationError("server address cannot be empty")
	}

	if cmd.ServerPort != nil && *cmd.ServerPort == 0 {
		return errors.NewValidationError("server port cannot be zero")
	}

	if cmd.Method != nil && *cmd.Method == "" {
		return errors.NewValidationError("encryption method cannot be empty")
	}

	return nil
}

// validateProtocolMethodCompatibility validates that the encryption method matches the protocol type
func (uc *UpdateNodeUseCase) validateProtocolMethodCompatibility(protocol vo.Protocol, method string) error {
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
			return errors.NewValidationError("encryption method '" + method + "' is not compatible with Shadowsocks protocol")
		}
	} else if protocol.IsTrojan() {
		// Trojan doesn't use these encryption methods, it uses TLS
		if ssMethods[method] {
			return errors.NewValidationError("encryption method '" + method + "' is not compatible with Trojan protocol")
		}
	}

	return nil
}
