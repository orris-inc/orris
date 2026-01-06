package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type UpdateNodeCommand struct {
	SID              string // External API identifier
	Name             *string
	ServerAddress    *string
	AgentPort        *uint16
	SubscriptionPort *uint16
	Method           *string
	Plugin           *string
	PluginOpts       map[string]string
	Region           *string
	Tags             []string
	Description      *string
	SortOrder        *int
	Status           *string
	GroupSID         *string // Resource group SID (empty string to remove association)
	MuteNotification *bool   // nil: no update, non-nil: set mute notification flag
	// Trojan specific fields
	TrojanTransportProtocol *string
	TrojanHost              *string
	TrojanPath              *string
	TrojanSNI               *string
	TrojanAllowInsecure     *bool
	// Route configuration for traffic splitting
	Route      *dto.RouteConfigDTO // Route config to set (nil = no change)
	ClearRoute bool                // If true, clear the route config
}

type UpdateNodeResult struct {
	NodeID           uint
	Name             string
	ServerAddress    string
	AgentPort        uint16
	SubscriptionPort *uint16
	Protocol         string
	Status           string
	UpdatedAt        string
}

type UpdateNodeUseCase struct {
	logger                logger.Interface
	nodeRepo              node.NodeRepository
	resourceGroupRepo     resource.Repository
	addressChangeNotifier NodeAddressChangeNotifier
	configChangeNotifier  NodeConfigChangeNotifier
}

func NewUpdateNodeUseCase(
	logger logger.Interface,
	nodeRepo node.NodeRepository,
	resourceGroupRepo resource.Repository,
) *UpdateNodeUseCase {
	return &UpdateNodeUseCase{
		logger:            logger,
		nodeRepo:          nodeRepo,
		resourceGroupRepo: resourceGroupRepo,
	}
}

// SetAddressChangeNotifier sets the notifier for address changes.
// This is used to break circular dependencies during initialization.
func (uc *UpdateNodeUseCase) SetAddressChangeNotifier(notifier NodeAddressChangeNotifier) {
	uc.addressChangeNotifier = notifier
}

// SetConfigChangeNotifier sets the notifier for node configuration changes.
// This is used to notify node agents when their configuration (including route) changes.
func (uc *UpdateNodeUseCase) SetConfigChangeNotifier(notifier NodeConfigChangeNotifier) {
	uc.configChangeNotifier = notifier
}

func (uc *UpdateNodeUseCase) Execute(ctx context.Context, cmd UpdateNodeCommand) (*UpdateNodeResult, error) {
	uc.logger.Infow("executing update node use case", "sid", cmd.SID)

	// Validate command
	if err := uc.validateCommand(cmd); err != nil {
		uc.logger.Errorw("invalid update node command", "error", err, "sid", cmd.SID)
		return nil, err
	}

	// Get existing node from repository
	existingNode, err := uc.nodeRepo.GetBySID(ctx, cmd.SID)
	if err != nil {
		uc.logger.Errorw("failed to get node by SID", "sid", cmd.SID, "error", err)
		return nil, errors.NewNotFoundError("node not found")
	}

	// Capture original values for address change detection
	originalAddress := existingNode.ServerAddress().Value()
	originalPort := int(existingNode.AgentPort())

	// Check uniqueness constraints for name update
	if cmd.Name != nil && *cmd.Name != existingNode.Name() {
		exists, err := uc.nodeRepo.ExistsByNameExcluding(ctx, *cmd.Name, existingNode.ID())
		if err != nil {
			uc.logger.Errorw("failed to check name uniqueness", "error", err, "name", *cmd.Name)
			return nil, err
		}
		if exists {
			return nil, errors.NewConflictError("node with this name already exists")
		}
	}

	// Check uniqueness constraints for address/port update
	newAddress := existingNode.ServerAddress().Value()
	newPort := int(existingNode.AgentPort())
	if cmd.ServerAddress != nil {
		newAddress = *cmd.ServerAddress
	}
	if cmd.AgentPort != nil {
		newPort = int(*cmd.AgentPort)
	}
	// Only check if address or port actually changed
	if newAddress != existingNode.ServerAddress().Value() || newPort != int(existingNode.AgentPort()) {
		exists, err := uc.nodeRepo.ExistsByAddressExcluding(ctx, newAddress, newPort, existingNode.ID())
		if err != nil {
			uc.logger.Errorw("failed to check address uniqueness", "error", err, "address", newAddress, "port", newPort)
			return nil, err
		}
		if exists {
			return nil, errors.NewConflictError("node with this address and port already exists")
		}
	}

	// Handle GroupSID update (resolve SID to internal ID)
	if cmd.GroupSID != nil {
		if *cmd.GroupSID == "" {
			// Empty string means remove all group associations
			existingNode.SetGroupIDs(nil)
		} else {
			// Resolve group SID to internal ID and set as the only group
			group, err := uc.resourceGroupRepo.GetBySID(ctx, *cmd.GroupSID)
			if err != nil {
				uc.logger.Errorw("failed to get resource group by SID", "group_sid", *cmd.GroupSID, "error", err)
				return nil, errors.NewNotFoundError("resource group not found")
			}
			if group == nil {
				return nil, errors.NewNotFoundError("resource group not found")
			}
			existingNode.SetGroupIDs([]uint{group.ID()})
		}
	}

	// Validate route config node references before applying updates
	if cmd.Route != nil {
		if err := uc.validateRouteConfigNodeReferences(ctx, cmd.Route, existingNode); err != nil {
			uc.logger.Errorw("failed to validate route config node references", "error", err, "sid", cmd.SID)
			return nil, err
		}
	}

	// Apply updates based on command fields
	if err := uc.applyUpdates(existingNode, cmd); err != nil {
		uc.logger.Errorw("failed to apply updates", "error", err, "sid", cmd.SID)
		return nil, err
	}

	// Save updated node
	if err := uc.nodeRepo.Update(ctx, existingNode); err != nil {
		uc.logger.Errorw("failed to update node", "error", err, "sid", cmd.SID)
		return nil, err
	}

	uc.logger.Infow("node updated successfully", "sid", cmd.SID)

	// Check if address or port changed and notify forward agents
	addressChanged := originalAddress != newAddress || originalPort != newPort

	if addressChanged && uc.addressChangeNotifier != nil {
		uc.logger.Infow("node address changed, notifying forward agents",
			"node_id", existingNode.ID(),
			"old_address", originalAddress,
			"new_address", newAddress,
			"old_port", originalPort,
			"new_port", newPort,
		)
		// Notify asynchronously to avoid blocking the response
		nodeID := existingNode.ID()
		go func() {
			notifyCtx := context.Background()
			if err := uc.addressChangeNotifier.NotifyNodeAddressChange(notifyCtx, nodeID); err != nil {
				uc.logger.Warnw("failed to notify forward agents of node address change",
					"error", err,
					"node_id", nodeID,
				)
			}
		}()
	}

	// Notify node agent of configuration change (including route config)
	if uc.configChangeNotifier != nil {
		nodeID := existingNode.ID()
		go func() {
			notifyCtx := context.Background()
			if err := uc.configChangeNotifier.NotifyConfigChange(notifyCtx, nodeID); err != nil {
				uc.logger.Warnw("failed to notify node agent of config change",
					"error", err,
					"node_id", nodeID,
				)
			}
		}()
	}

	// Build and return result
	return &UpdateNodeResult{
		NodeID:           existingNode.ID(),
		Name:             existingNode.Name(),
		ServerAddress:    existingNode.ServerAddress().Value(),
		AgentPort:        existingNode.AgentPort(),
		SubscriptionPort: existingNode.SubscriptionPort(),
		Protocol:         existingNode.Protocol().String(),
		Status:           existingNode.Status().String(),
		UpdatedAt:        existingNode.UpdatedAt().Format(time.RFC3339),
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

	// Update agent port
	if cmd.AgentPort != nil {
		if err := n.UpdateAgentPort(*cmd.AgentPort); err != nil {
			return errors.NewValidationError("invalid agent port: " + err.Error())
		}
	}

	// Update subscription port
	if cmd.SubscriptionPort != nil {
		if err := n.UpdateSubscriptionPort(cmd.SubscriptionPort); err != nil {
			return errors.NewValidationError("invalid subscription port: " + err.Error())
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

	// Update mute notification
	if cmd.MuteNotification != nil {
		n.SetMuteNotification(*cmd.MuteNotification)
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

	// Update Trojan config (only for Trojan protocol nodes)
	if err := uc.applyTrojanUpdates(n, cmd); err != nil {
		return err
	}

	// Update route config
	if cmd.ClearRoute {
		n.ClearRouteConfig()
	} else if cmd.Route != nil {
		routeConfig, err := dto.FromRouteConfigDTO(cmd.Route)
		if err != nil {
			return errors.NewValidationError("invalid route config: " + err.Error())
		}
		if err := n.UpdateRouteConfig(routeConfig); err != nil {
			return errors.NewValidationError("failed to update route config: " + err.Error())
		}
	}

	return nil
}

func (uc *UpdateNodeUseCase) validateCommand(cmd UpdateNodeCommand) error {
	if cmd.SID == "" {
		return errors.NewValidationError("SID must be provided")
	}

	if cmd.Name == nil && cmd.ServerAddress == nil && cmd.AgentPort == nil &&
		cmd.SubscriptionPort == nil && cmd.Method == nil && cmd.Plugin == nil &&
		len(cmd.PluginOpts) == 0 && cmd.Region == nil && cmd.Tags == nil &&
		cmd.Description == nil && cmd.SortOrder == nil && cmd.Status == nil &&
		cmd.GroupSID == nil && cmd.MuteNotification == nil &&
		cmd.TrojanTransportProtocol == nil && cmd.TrojanHost == nil &&
		cmd.TrojanPath == nil && cmd.TrojanSNI == nil && cmd.TrojanAllowInsecure == nil &&
		cmd.Route == nil && !cmd.ClearRoute {
		return errors.NewValidationError("at least one field must be provided for update")
	}

	if cmd.Name != nil && *cmd.Name == "" {
		return errors.NewValidationError("node name cannot be empty")
	}

	if cmd.AgentPort != nil && *cmd.AgentPort == 0 {
		return errors.NewValidationError("agent port cannot be zero")
	}

	if cmd.Method != nil && *cmd.Method == "" {
		return errors.NewValidationError("encryption method cannot be empty")
	}

	// ClearRoute and Route are mutually exclusive
	if cmd.ClearRoute && cmd.Route != nil {
		return errors.NewValidationError("cannot set both ClearRoute and Route; use ClearRoute to remove config or Route to set new config")
	}

	return nil
}

// applyTrojanUpdates applies Trojan-specific configuration updates
func (uc *UpdateNodeUseCase) applyTrojanUpdates(n *node.Node, cmd UpdateNodeCommand) error {
	// Check if any Trojan fields need updating
	hasTrojanUpdate := cmd.TrojanTransportProtocol != nil ||
		cmd.TrojanHost != nil ||
		cmd.TrojanPath != nil ||
		cmd.TrojanSNI != nil ||
		cmd.TrojanAllowInsecure != nil

	if !hasTrojanUpdate {
		return nil
	}

	// Validate protocol is Trojan
	if !n.Protocol().IsTrojan() {
		return errors.NewValidationError("cannot update Trojan config for non-Trojan protocol node")
	}

	// Get current Trojan config or use defaults for legacy nodes
	currentConfig := n.TrojanConfig()

	// Build new config with updated values (use defaults if no existing config)
	var password, transportProtocol, host, path, sni string
	var allowInsecure bool

	if currentConfig != nil {
		password = currentConfig.Password()
		transportProtocol = currentConfig.TransportProtocol()
		host = currentConfig.Host()
		path = currentConfig.Path()
		sni = currentConfig.SNI()
		allowInsecure = currentConfig.AllowInsecure()
	} else {
		// Default values for legacy Trojan nodes without config
		// For Trojan protocol, actual password is subscription UUID (passed at runtime)
		// Use placeholder for config storage
		password = "placeholder"
		transportProtocol = "tcp"
		allowInsecure = true // Default true for self-signed certs
	}

	if cmd.TrojanTransportProtocol != nil {
		transportProtocol = *cmd.TrojanTransportProtocol
	}
	if cmd.TrojanHost != nil {
		host = *cmd.TrojanHost
	}
	if cmd.TrojanPath != nil {
		path = *cmd.TrojanPath
	}
	if cmd.TrojanSNI != nil {
		sni = *cmd.TrojanSNI
	}
	if cmd.TrojanAllowInsecure != nil {
		allowInsecure = *cmd.TrojanAllowInsecure
	}

	// Create new Trojan config (password remains unchanged)
	newConfig, err := vo.NewTrojanConfig(
		password,
		transportProtocol,
		host,
		path,
		allowInsecure,
		sni,
	)
	if err != nil {
		return errors.NewValidationError("invalid Trojan configuration: " + err.Error())
	}

	// Update the node with new config
	if err := n.UpdateTrojanConfig(&newConfig); err != nil {
		return errors.NewValidationError("failed to update Trojan config: " + err.Error())
	}

	return nil
}

// validateProtocolMethodCompatibility validates that the encryption method matches the protocol type
func (uc *UpdateNodeUseCase) validateProtocolMethodCompatibility(protocol vo.Protocol, method string) error {
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

// validateRouteConfigNodeReferences validates that all node SIDs referenced in route config exist
// and belong to the same user (for user nodes) or exist globally (for admin nodes).
func (uc *UpdateNodeUseCase) validateRouteConfigNodeReferences(ctx context.Context, routeDTO *dto.RouteConfigDTO, currentNode *node.Node) error {
	if routeDTO == nil {
		return nil
	}

	// Convert DTO to domain object to extract referenced node SIDs
	routeConfig, err := dto.FromRouteConfigDTO(routeDTO)
	if err != nil {
		return errors.NewValidationError("invalid route config: " + err.Error())
	}

	referencedSIDs := routeConfig.GetReferencedNodeSIDs()
	if len(referencedSIDs) == 0 {
		return nil
	}

	// Check self-reference (cannot reference itself)
	for _, sid := range referencedSIDs {
		if sid == currentNode.SID() {
			return errors.NewValidationError("route config cannot reference the node itself as outbound")
		}
	}

	var invalidSIDs []string

	if currentNode.IsUserOwned() {
		// User-owned node: validate that referenced nodes belong to the same user
		invalidSIDs, err = uc.nodeRepo.ValidateNodeSIDsForUser(ctx, referencedSIDs, *currentNode.UserID())
	} else {
		// Admin node: validate that referenced nodes exist (admin can reference any node)
		invalidSIDs, err = uc.nodeRepo.ValidateNodeSIDsExist(ctx, referencedSIDs)
	}

	if err != nil {
		uc.logger.Errorw("failed to validate route config node references", "error", err)
		return errors.NewInternalError("failed to validate route config")
	}

	if len(invalidSIDs) > 0 {
		if currentNode.IsUserOwned() {
			return errors.NewValidationError(
				fmt.Sprintf("invalid node SIDs in route config (not found or not owned by user): %v", invalidSIDs))
		}
		return errors.NewValidationError(
			fmt.Sprintf("invalid node SIDs in route config (not found): %v", invalidSIDs))
	}

	return nil
}
