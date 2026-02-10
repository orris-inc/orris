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
	"github.com/orris-inc/orris/internal/shared/goroutine"
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
	GroupSIDs        []string // Resource group SIDs (empty slice to remove all, nil means no change)
	MuteNotification *bool    // nil: no update, non-nil: set mute notification flag
	// Trojan specific fields
	TrojanTransportProtocol *string
	TrojanHost              *string
	TrojanPath              *string
	TrojanSNI               *string
	TrojanAllowInsecure     *bool
	// Route configuration for traffic splitting
	Route      *dto.RouteConfigDTO // Route config to set (nil = no change)
	ClearRoute bool                // If true, clear the route config

	// VLESS specific fields
	VLESSTransportType     *string
	VLESSFlow              *string
	VLESSSecurity          *string
	VLESSSni               *string
	VLESSFingerprint       *string
	VLESSAllowInsecure     *bool
	VLESSHost              *string
	VLESSPath              *string
	VLESSServiceName       *string
	VLESSRealityPrivateKey *string // Optional: auto-generated if empty when switching to Reality
	VLESSRealityPublicKey  *string // Optional: auto-generated if empty when switching to Reality
	VLESSRealityShortID    *string // Optional: auto-generated if empty when switching to Reality
	VLESSRealitySpiderX    *string

	// VMess specific fields
	VMessAlterID       *int
	VMessSecurity      *string
	VMessTransportType *string
	VMessHost          *string
	VMessPath          *string
	VMessServiceName   *string
	VMessTLS           *bool
	VMessSni           *string
	VMessAllowInsecure *bool

	// Hysteria2 specific fields
	Hysteria2CongestionControl *string
	Hysteria2Obfs              *string
	Hysteria2ObfsPassword      *string
	Hysteria2UpMbps            *int
	Hysteria2DownMbps          *int
	Hysteria2Sni               *string
	Hysteria2AllowInsecure     *bool
	Hysteria2Fingerprint       *string

	// TUIC specific fields
	TUICCongestionControl *string
	TUICUDPRelayMode      *string
	TUICAlpn              *string
	TUICSni               *string
	TUICAllowInsecure     *bool
	TUICDisableSNI        *bool

	// Expiration and cost label fields
	ExpiresAt      *time.Time // nil: no update, set to update expiration time
	ClearExpiresAt bool       // true: clear expiration time
	CostLabel      *string    // nil: no update, set to update cost label
	ClearCostLabel bool       // true: clear cost label
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

	// Handle GroupSIDs update (resolve SIDs to internal IDs)
	if cmd.GroupSIDs != nil {
		if len(cmd.GroupSIDs) == 0 {
			// Empty slice means remove all group associations
			existingNode.SetGroupIDs(nil)
		} else {
			// Deduplicate and filter empty SIDs
			uniqueSIDs := make([]string, 0, len(cmd.GroupSIDs))
			seenSIDs := make(map[string]struct{}, len(cmd.GroupSIDs))
			for _, sid := range cmd.GroupSIDs {
				if sid == "" {
					continue
				}
				if _, exists := seenSIDs[sid]; exists {
					continue
				}
				seenSIDs[sid] = struct{}{}
				uniqueSIDs = append(uniqueSIDs, sid)
			}

			if len(uniqueSIDs) > 0 {
				// Batch fetch all groups to avoid N+1 queries
				groupMap, err := uc.resourceGroupRepo.GetBySIDs(ctx, uniqueSIDs)
				if err != nil {
					uc.logger.Errorw("failed to batch get resource groups", "error", err)
					return nil, fmt.Errorf("failed to get resource groups: %w", err)
				}

				// Resolve SIDs to internal IDs
				resolvedIDs := make([]uint, 0, len(uniqueSIDs))
				for _, sid := range uniqueSIDs {
					group, ok := groupMap[sid]
					if !ok || group == nil {
						return nil, errors.NewNotFoundError(fmt.Sprintf("resource group not found: %s", sid))
					}
					resolvedIDs = append(resolvedIDs, group.ID())
				}
				existingNode.SetGroupIDs(resolvedIDs)
			}
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
		goroutine.SafeGo(uc.logger, "update-node-notify-address-change", func() {
			notifyCtx := context.Background()
			if err := uc.addressChangeNotifier.NotifyNodeAddressChange(notifyCtx, nodeID); err != nil {
				uc.logger.Warnw("failed to notify forward agents of node address change",
					"error", err,
					"node_id", nodeID,
				)
			}
		})
	}

	// Notify node agent of configuration change (including route config)
	if uc.configChangeNotifier != nil {
		nodeID := existingNode.ID()
		goroutine.SafeGo(uc.logger, "update-node-notify-config-change", func() {
			notifyCtx := context.Background()
			if err := uc.configChangeNotifier.NotifyConfigChange(notifyCtx, nodeID); err != nil {
				uc.logger.Warnw("failed to notify node agent of config change",
					"error", err,
					"node_id", nodeID,
				)
			}
		})
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

	// Update VLESS config (only for VLESS protocol nodes)
	if err := uc.applyVLESSUpdates(n, cmd); err != nil {
		return err
	}

	// Update VMess config (only for VMess protocol nodes)
	if err := uc.applyVMessUpdates(n, cmd); err != nil {
		return err
	}

	// Update Hysteria2 config (only for Hysteria2 protocol nodes)
	if err := uc.applyHysteria2Updates(n, cmd); err != nil {
		return err
	}

	// Update TUIC config (only for TUIC protocol nodes)
	if err := uc.applyTUICUpdates(n, cmd); err != nil {
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

	// Update expires_at
	if cmd.ClearExpiresAt {
		n.SetExpiresAt(nil)
	} else if cmd.ExpiresAt != nil {
		n.SetExpiresAt(cmd.ExpiresAt)
	}

	// Update cost_label
	if cmd.ClearCostLabel {
		n.SetCostLabel(nil)
	} else if cmd.CostLabel != nil {
		n.SetCostLabel(cmd.CostLabel)
	}

	return nil
}

func (uc *UpdateNodeUseCase) validateCommand(cmd UpdateNodeCommand) error {
	if cmd.SID == "" {
		return errors.NewValidationError("SID must be provided")
	}

	// Check if at least one field is provided for update
	hasUpdate := cmd.Name != nil || cmd.ServerAddress != nil || cmd.AgentPort != nil ||
		cmd.SubscriptionPort != nil || cmd.Method != nil || cmd.Plugin != nil ||
		len(cmd.PluginOpts) > 0 || cmd.Region != nil || cmd.Tags != nil ||
		cmd.Description != nil || cmd.SortOrder != nil || cmd.Status != nil ||
		cmd.GroupSIDs != nil || cmd.MuteNotification != nil ||
		cmd.TrojanTransportProtocol != nil || cmd.TrojanHost != nil ||
		cmd.TrojanPath != nil || cmd.TrojanSNI != nil || cmd.TrojanAllowInsecure != nil ||
		cmd.Route != nil || cmd.ClearRoute ||
		// VLESS fields
		cmd.VLESSTransportType != nil || cmd.VLESSFlow != nil || cmd.VLESSSecurity != nil ||
		cmd.VLESSSni != nil || cmd.VLESSFingerprint != nil || cmd.VLESSAllowInsecure != nil ||
		cmd.VLESSHost != nil || cmd.VLESSPath != nil || cmd.VLESSServiceName != nil ||
		cmd.VLESSRealityPrivateKey != nil || cmd.VLESSRealityPublicKey != nil ||
		cmd.VLESSRealityShortID != nil || cmd.VLESSRealitySpiderX != nil ||
		// VMess fields
		cmd.VMessAlterID != nil || cmd.VMessSecurity != nil || cmd.VMessTransportType != nil ||
		cmd.VMessHost != nil || cmd.VMessPath != nil || cmd.VMessServiceName != nil ||
		cmd.VMessTLS != nil || cmd.VMessSni != nil || cmd.VMessAllowInsecure != nil ||
		// Hysteria2 fields
		cmd.Hysteria2CongestionControl != nil || cmd.Hysteria2Obfs != nil || cmd.Hysteria2ObfsPassword != nil ||
		cmd.Hysteria2UpMbps != nil || cmd.Hysteria2DownMbps != nil || cmd.Hysteria2Sni != nil ||
		cmd.Hysteria2AllowInsecure != nil || cmd.Hysteria2Fingerprint != nil ||
		// TUIC fields
		cmd.TUICCongestionControl != nil || cmd.TUICUDPRelayMode != nil || cmd.TUICAlpn != nil ||
		cmd.TUICSni != nil || cmd.TUICAllowInsecure != nil || cmd.TUICDisableSNI != nil ||
		// Expiration and cost label fields
		cmd.ExpiresAt != nil || cmd.ClearExpiresAt || cmd.CostLabel != nil || cmd.ClearCostLabel

	if !hasUpdate {
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

// applyVLESSUpdates applies VLESS-specific configuration updates
func (uc *UpdateNodeUseCase) applyVLESSUpdates(n *node.Node, cmd UpdateNodeCommand) error {
	// Check if any VLESS fields need updating
	hasVLESSUpdate := cmd.VLESSTransportType != nil || cmd.VLESSFlow != nil ||
		cmd.VLESSSecurity != nil || cmd.VLESSSni != nil || cmd.VLESSFingerprint != nil ||
		cmd.VLESSAllowInsecure != nil || cmd.VLESSHost != nil || cmd.VLESSPath != nil ||
		cmd.VLESSServiceName != nil || cmd.VLESSRealityPrivateKey != nil ||
		cmd.VLESSRealityPublicKey != nil || cmd.VLESSRealityShortID != nil ||
		cmd.VLESSRealitySpiderX != nil

	if !hasVLESSUpdate {
		return nil
	}

	// Validate protocol is VLESS
	if !n.Protocol().IsVLESS() {
		return errors.NewValidationError("cannot update VLESS config for non-VLESS protocol node")
	}

	// Get current VLESS config or use defaults
	currentConfig := n.VLESSConfig()

	// Build new config with updated values
	var transportType, flow, security, sni, fingerprint, host, path, serviceName string
	var privateKey, publicKey, shortID, spiderX string
	var allowInsecure bool

	if currentConfig != nil {
		transportType = currentConfig.TransportType()
		flow = currentConfig.Flow()
		security = currentConfig.Security()
		sni = currentConfig.SNI()
		fingerprint = currentConfig.Fingerprint()
		allowInsecure = currentConfig.AllowInsecure()
		host = currentConfig.Host()
		path = currentConfig.Path()
		serviceName = currentConfig.ServiceName()
		privateKey = currentConfig.PrivateKey()
		publicKey = currentConfig.PublicKey()
		shortID = currentConfig.ShortID()
		spiderX = currentConfig.SpiderX()
	} else {
		// Default values for VLESS nodes without config
		transportType = "tcp"
		security = "tls"
	}

	if cmd.VLESSTransportType != nil {
		transportType = *cmd.VLESSTransportType
	}
	if cmd.VLESSFlow != nil {
		flow = *cmd.VLESSFlow
	}
	if cmd.VLESSSecurity != nil {
		security = *cmd.VLESSSecurity
	}
	if cmd.VLESSSni != nil {
		sni = *cmd.VLESSSni
	}
	if cmd.VLESSFingerprint != nil {
		fingerprint = *cmd.VLESSFingerprint
	}
	if cmd.VLESSAllowInsecure != nil {
		allowInsecure = *cmd.VLESSAllowInsecure
	}
	if cmd.VLESSHost != nil {
		host = *cmd.VLESSHost
	}
	if cmd.VLESSPath != nil {
		path = *cmd.VLESSPath
	}
	if cmd.VLESSServiceName != nil {
		serviceName = *cmd.VLESSServiceName
	}
	if cmd.VLESSRealityPrivateKey != nil {
		privateKey = *cmd.VLESSRealityPrivateKey
	}
	if cmd.VLESSRealityPublicKey != nil {
		publicKey = *cmd.VLESSRealityPublicKey
	}
	if cmd.VLESSRealityShortID != nil {
		shortID = *cmd.VLESSRealityShortID
	}
	if cmd.VLESSRealitySpiderX != nil {
		spiderX = *cmd.VLESSRealitySpiderX
	}

	// Auto-generate Reality key pair and short ID if security is reality and all keys are empty
	if security == vo.VLESSSecurityReality && privateKey == "" && publicKey == "" && shortID == "" {
		// Auto-generate key pair
		keyPair, err := vo.GenerateRealityKeyPair()
		if err != nil {
			return errors.NewValidationError("failed to generate Reality key pair: " + err.Error())
		}
		privateKey = keyPair.PrivateKey
		publicKey = keyPair.PublicKey

		// Auto-generate short ID
		shortID, err = vo.GenerateRealityShortID()
		if err != nil {
			return errors.NewValidationError("failed to generate Reality short ID: " + err.Error())
		}

		uc.logger.Infow("auto-generated Reality key pair and short ID for VLESS node update",
			"node_sid", n.SID(),
			"public_key_prefix", publicKey[:8]+"...",
		)
	}

	// Create new VLESS config
	newConfig, err := vo.NewVLESSConfig(
		transportType,
		flow,
		security,
		sni,
		fingerprint,
		allowInsecure,
		host,
		path,
		serviceName,
		privateKey,
		publicKey,
		shortID,
		spiderX,
	)
	if err != nil {
		return errors.NewValidationError("invalid VLESS configuration: " + err.Error())
	}

	// Update the node with new config
	if err := n.UpdateVLESSConfig(&newConfig); err != nil {
		return errors.NewValidationError("failed to update VLESS config: " + err.Error())
	}

	return nil
}

// applyVMessUpdates applies VMess-specific configuration updates
func (uc *UpdateNodeUseCase) applyVMessUpdates(n *node.Node, cmd UpdateNodeCommand) error {
	// Check if any VMess fields need updating
	hasVMessUpdate := cmd.VMessAlterID != nil || cmd.VMessSecurity != nil ||
		cmd.VMessTransportType != nil || cmd.VMessHost != nil || cmd.VMessPath != nil ||
		cmd.VMessServiceName != nil || cmd.VMessTLS != nil || cmd.VMessSni != nil ||
		cmd.VMessAllowInsecure != nil

	if !hasVMessUpdate {
		return nil
	}

	// Validate protocol is VMess
	if !n.Protocol().IsVMess() {
		return errors.NewValidationError("cannot update VMess config for non-VMess protocol node")
	}

	// Get current VMess config or use defaults
	currentConfig := n.VMessConfig()

	// Build new config with updated values
	var alterID int
	var security, transportType, host, path, serviceName, sni string
	var tls, allowInsecure bool

	if currentConfig != nil {
		alterID = currentConfig.AlterID()
		security = currentConfig.Security()
		transportType = currentConfig.TransportType()
		host = currentConfig.Host()
		path = currentConfig.Path()
		serviceName = currentConfig.ServiceName()
		tls = currentConfig.TLS()
		sni = currentConfig.SNI()
		allowInsecure = currentConfig.AllowInsecure()
	} else {
		// Default values for VMess nodes without config
		security = "auto"
		transportType = "tcp"
	}

	if cmd.VMessAlterID != nil {
		alterID = *cmd.VMessAlterID
	}
	if cmd.VMessSecurity != nil {
		security = *cmd.VMessSecurity
	}
	if cmd.VMessTransportType != nil {
		transportType = *cmd.VMessTransportType
	}
	if cmd.VMessHost != nil {
		host = *cmd.VMessHost
	}
	if cmd.VMessPath != nil {
		path = *cmd.VMessPath
	}
	if cmd.VMessServiceName != nil {
		serviceName = *cmd.VMessServiceName
	}
	if cmd.VMessTLS != nil {
		tls = *cmd.VMessTLS
	}
	if cmd.VMessSni != nil {
		sni = *cmd.VMessSni
	}
	if cmd.VMessAllowInsecure != nil {
		allowInsecure = *cmd.VMessAllowInsecure
	}

	// Create new VMess config
	newConfig, err := vo.NewVMessConfig(
		alterID,
		security,
		transportType,
		host,
		path,
		serviceName,
		tls,
		sni,
		allowInsecure,
	)
	if err != nil {
		return errors.NewValidationError("invalid VMess configuration: " + err.Error())
	}

	// Update the node with new config
	if err := n.UpdateVMessConfig(&newConfig); err != nil {
		return errors.NewValidationError("failed to update VMess config: " + err.Error())
	}

	return nil
}

// applyHysteria2Updates applies Hysteria2-specific configuration updates
func (uc *UpdateNodeUseCase) applyHysteria2Updates(n *node.Node, cmd UpdateNodeCommand) error {
	// Check if any Hysteria2 fields need updating
	hasHysteria2Update := cmd.Hysteria2CongestionControl != nil || cmd.Hysteria2Obfs != nil ||
		cmd.Hysteria2ObfsPassword != nil || cmd.Hysteria2UpMbps != nil ||
		cmd.Hysteria2DownMbps != nil || cmd.Hysteria2Sni != nil ||
		cmd.Hysteria2AllowInsecure != nil || cmd.Hysteria2Fingerprint != nil

	if !hasHysteria2Update {
		return nil
	}

	// Validate protocol is Hysteria2
	if !n.Protocol().IsHysteria2() {
		return errors.NewValidationError("cannot update Hysteria2 config for non-Hysteria2 protocol node")
	}

	// Get current Hysteria2 config or use defaults
	currentConfig := n.Hysteria2Config()

	// Build new config with updated values
	var password, congestionControl, obfs, obfsPassword, sni, fingerprint string
	var upMbps, downMbps *int
	var allowInsecure bool

	if currentConfig != nil {
		password = currentConfig.Password()
		congestionControl = currentConfig.CongestionControl()
		obfs = currentConfig.Obfs()
		obfsPassword = currentConfig.ObfsPassword()
		upMbps = currentConfig.UpMbps()
		downMbps = currentConfig.DownMbps()
		sni = currentConfig.SNI()
		allowInsecure = currentConfig.AllowInsecure()
		fingerprint = currentConfig.Fingerprint()
	} else {
		// Default values for Hysteria2 nodes without config
		password = "placeholder"
		congestionControl = "bbr"
	}

	if cmd.Hysteria2CongestionControl != nil {
		congestionControl = *cmd.Hysteria2CongestionControl
	}
	if cmd.Hysteria2Obfs != nil {
		obfs = *cmd.Hysteria2Obfs
	}
	if cmd.Hysteria2ObfsPassword != nil {
		obfsPassword = *cmd.Hysteria2ObfsPassword
	}
	if cmd.Hysteria2UpMbps != nil {
		upMbps = cmd.Hysteria2UpMbps
	}
	if cmd.Hysteria2DownMbps != nil {
		downMbps = cmd.Hysteria2DownMbps
	}
	if cmd.Hysteria2Sni != nil {
		sni = *cmd.Hysteria2Sni
	}
	if cmd.Hysteria2AllowInsecure != nil {
		allowInsecure = *cmd.Hysteria2AllowInsecure
	}
	if cmd.Hysteria2Fingerprint != nil {
		fingerprint = *cmd.Hysteria2Fingerprint
	}

	// Create new Hysteria2 config (password remains unchanged)
	newConfig, err := vo.NewHysteria2Config(
		password,
		congestionControl,
		obfs,
		obfsPassword,
		upMbps,
		downMbps,
		sni,
		allowInsecure,
		fingerprint,
	)
	if err != nil {
		return errors.NewValidationError("invalid Hysteria2 configuration: " + err.Error())
	}

	// Update the node with new config
	if err := n.UpdateHysteria2Config(&newConfig); err != nil {
		return errors.NewValidationError("failed to update Hysteria2 config: " + err.Error())
	}

	return nil
}

// applyTUICUpdates applies TUIC-specific configuration updates
func (uc *UpdateNodeUseCase) applyTUICUpdates(n *node.Node, cmd UpdateNodeCommand) error {
	// Check if any TUIC fields need updating
	hasTUICUpdate := cmd.TUICCongestionControl != nil || cmd.TUICUDPRelayMode != nil ||
		cmd.TUICAlpn != nil || cmd.TUICSni != nil ||
		cmd.TUICAllowInsecure != nil || cmd.TUICDisableSNI != nil

	if !hasTUICUpdate {
		return nil
	}

	// Validate protocol is TUIC
	if !n.Protocol().IsTUIC() {
		return errors.NewValidationError("cannot update TUIC config for non-TUIC protocol node")
	}

	// Get current TUIC config or use defaults
	currentConfig := n.TUICConfig()

	// Build new config with updated values
	var uuid, password, congestionControl, udpRelayMode, alpn, sni string
	var allowInsecure, disableSNI bool

	if currentConfig != nil {
		uuid = currentConfig.UUID()
		password = currentConfig.Password()
		congestionControl = currentConfig.CongestionControl()
		udpRelayMode = currentConfig.UDPRelayMode()
		alpn = currentConfig.ALPN()
		sni = currentConfig.SNI()
		allowInsecure = currentConfig.AllowInsecure()
		disableSNI = currentConfig.DisableSNI()
	} else {
		// Default values for TUIC nodes without config
		uuid = "placeholder"
		password = "placeholder"
		congestionControl = "bbr"
		udpRelayMode = "native"
	}

	if cmd.TUICCongestionControl != nil {
		congestionControl = *cmd.TUICCongestionControl
	}
	if cmd.TUICUDPRelayMode != nil {
		udpRelayMode = *cmd.TUICUDPRelayMode
	}
	if cmd.TUICAlpn != nil {
		alpn = *cmd.TUICAlpn
	}
	if cmd.TUICSni != nil {
		sni = *cmd.TUICSni
	}
	if cmd.TUICAllowInsecure != nil {
		allowInsecure = *cmd.TUICAllowInsecure
	}
	if cmd.TUICDisableSNI != nil {
		disableSNI = *cmd.TUICDisableSNI
	}

	// Create new TUIC config (uuid and password remain unchanged)
	newConfig, err := vo.NewTUICConfig(
		uuid,
		password,
		congestionControl,
		udpRelayMode,
		alpn,
		sni,
		allowInsecure,
		disableSNI,
	)
	if err != nil {
		return errors.NewValidationError("invalid TUIC configuration: " + err.Error())
	}

	// Update the node with new config
	if err := n.UpdateTUICConfig(&newConfig); err != nil {
		return errors.NewValidationError("failed to update TUIC config: " + err.Error())
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
