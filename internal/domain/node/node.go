// Package node provides domain models and business logic for node management.
// It includes the Node aggregate root, node groups, traffic tracking, and token validation.
package node

import (
	"crypto/subtle"
	"fmt"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/node/value_objects"
	"github.com/orris-inc/orris/internal/domain/shared/services"
)

// Node represents the node aggregate root
type Node struct {
	id                uint
	name              string
	serverAddress     vo.ServerAddress
	agentPort         uint16  // port for agent connections (required)
	subscriptionPort  *uint16 // port for client subscriptions (if nil, use agentPort)
	protocol          vo.Protocol
	encryptionConfig  vo.EncryptionConfig
	pluginConfig      *vo.PluginConfig
	trojanConfig      *vo.TrojanConfig
	status            vo.NodeStatus
	metadata          vo.NodeMetadata
	apiToken          string
	tokenHash         string
	sortOrder         int
	maintenanceReason *string
	lastSeenAt        *time.Time // last time the node agent reported status
	publicIPv4        *string    // public IPv4 address reported by agent
	publicIPv6        *string    // public IPv6 address reported by agent
	version           int
	createdAt         time.Time
	updatedAt         time.Time
	tokenGenerator    services.TokenGenerator
}

// NewNode creates a new node aggregate
func NewNode(
	name string,
	serverAddress vo.ServerAddress,
	agentPort uint16,
	subscriptionPort *uint16,
	protocol vo.Protocol,
	encryptionConfig vo.EncryptionConfig,
	pluginConfig *vo.PluginConfig,
	trojanConfig *vo.TrojanConfig,
	metadata vo.NodeMetadata,
	sortOrder int,
) (*Node, error) {
	if name == "" {
		return nil, fmt.Errorf("node name is required")
	}
	if agentPort == 0 {
		return nil, fmt.Errorf("agent port is required")
	}
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", protocol)
	}

	// Validate protocol-specific configurations
	if protocol.IsShadowsocks() && encryptionConfig.Method() == "" {
		return nil, fmt.Errorf("encryption config is required for Shadowsocks protocol")
	}
	if protocol.IsTrojan() && trojanConfig == nil {
		return nil, fmt.Errorf("trojan config is required for Trojan protocol")
	}

	tokenGen := services.NewTokenGenerator()
	plainToken, tokenHash, err := tokenGen.GenerateAPIToken("node")
	if err != nil {
		return nil, fmt.Errorf("failed to generate API token: %w", err)
	}

	now := time.Now()
	n := &Node{
		name:             name,
		serverAddress:    serverAddress,
		agentPort:        agentPort,
		subscriptionPort: subscriptionPort,
		protocol:         protocol,
		encryptionConfig: encryptionConfig,
		pluginConfig:     pluginConfig,
		trojanConfig:     trojanConfig,
		status:           vo.NodeStatusInactive,
		metadata:         metadata,
		apiToken:         plainToken,
		tokenHash:        tokenHash,
		sortOrder:        sortOrder,
		version:          1,
		createdAt:        now,
		updatedAt:        now,
		tokenGenerator:   tokenGen,
	}

	return n, nil
}

// ReconstructNode reconstructs a node from persistence
func ReconstructNode(
	id uint,
	name string,
	serverAddress vo.ServerAddress,
	agentPort uint16,
	subscriptionPort *uint16,
	protocol vo.Protocol,
	encryptionConfig vo.EncryptionConfig,
	pluginConfig *vo.PluginConfig,
	trojanConfig *vo.TrojanConfig,
	status vo.NodeStatus,
	metadata vo.NodeMetadata,
	tokenHash string,
	apiToken string,
	sortOrder int,
	maintenanceReason *string,
	lastSeenAt *time.Time,
	publicIPv4 *string,
	publicIPv6 *string,
	version int,
	createdAt, updatedAt time.Time,
) (*Node, error) {
	if id == 0 {
		return nil, fmt.Errorf("node ID cannot be zero")
	}
	if name == "" {
		return nil, fmt.Errorf("node name is required")
	}
	if agentPort == 0 {
		return nil, fmt.Errorf("agent port is required")
	}
	if tokenHash == "" {
		return nil, fmt.Errorf("token hash is required")
	}
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", protocol)
	}

	return &Node{
		id:                id,
		name:              name,
		serverAddress:     serverAddress,
		agentPort:         agentPort,
		subscriptionPort:  subscriptionPort,
		protocol:          protocol,
		encryptionConfig:  encryptionConfig,
		pluginConfig:      pluginConfig,
		trojanConfig:      trojanConfig,
		status:            status,
		metadata:          metadata,
		tokenHash:         tokenHash,
		apiToken:          apiToken,
		sortOrder:         sortOrder,
		maintenanceReason: maintenanceReason,
		lastSeenAt:        lastSeenAt,
		publicIPv4:        publicIPv4,
		publicIPv6:        publicIPv6,
		version:           version,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
		tokenGenerator:    services.NewTokenGenerator(),
	}, nil
}

// ID returns the node ID
func (n *Node) ID() uint {
	return n.id
}

// Name returns the node name
func (n *Node) Name() string {
	return n.name
}

// ServerAddress returns the server address
func (n *Node) ServerAddress() vo.ServerAddress {
	return n.serverAddress
}

// AgentPort returns the agent connection port
func (n *Node) AgentPort() uint16 {
	return n.agentPort
}

// SubscriptionPort returns the subscription port (may be nil)
func (n *Node) SubscriptionPort() *uint16 {
	return n.subscriptionPort
}

// EffectiveSubscriptionPort returns the port to use for subscriptions
// If subscriptionPort is nil, returns agentPort
func (n *Node) EffectiveSubscriptionPort() uint16 {
	if n.subscriptionPort != nil {
		return *n.subscriptionPort
	}
	return n.agentPort
}

// Protocol returns the protocol type
func (n *Node) Protocol() vo.Protocol {
	return n.protocol
}

// EncryptionConfig returns the encryption configuration
func (n *Node) EncryptionConfig() vo.EncryptionConfig {
	return n.encryptionConfig
}

// PluginConfig returns the plugin configuration
func (n *Node) PluginConfig() *vo.PluginConfig {
	return n.pluginConfig
}

// TrojanConfig returns the trojan configuration
func (n *Node) TrojanConfig() *vo.TrojanConfig {
	return n.trojanConfig
}

// Status returns the node status
func (n *Node) Status() vo.NodeStatus {
	return n.status
}

// Metadata returns the node metadata
func (n *Node) Metadata() vo.NodeMetadata {
	return n.metadata
}

// TokenHash returns the API token hash
func (n *Node) TokenHash() string {
	return n.tokenHash
}

// SortOrder returns the sort order
func (n *Node) SortOrder() int {
	return n.sortOrder
}

// MaintenanceReason returns the maintenance reason
func (n *Node) MaintenanceReason() *string {
	return n.maintenanceReason
}

// Version returns the aggregate version for optimistic locking
func (n *Node) Version() int {
	return n.version
}

// CreatedAt returns when the node was created
func (n *Node) CreatedAt() time.Time {
	return n.createdAt
}

// UpdatedAt returns when the node was last updated
func (n *Node) UpdatedAt() time.Time {
	return n.updatedAt
}

// SetID sets the node ID (only for persistence layer use)
func (n *Node) SetID(id uint) error {
	if n.id != 0 {
		return fmt.Errorf("node ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("node ID cannot be zero")
	}
	n.id = id
	return nil
}

// Activate activates the node
func (n *Node) Activate() error {
	if n.status == vo.NodeStatusActive {
		return nil
	}

	if !n.status.CanTransitionTo(vo.NodeStatusActive) {
		return fmt.Errorf("cannot activate node with status %s", n.status)
	}

	n.status = vo.NodeStatusActive
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// Deactivate deactivates the node
func (n *Node) Deactivate() error {
	if n.status == vo.NodeStatusInactive {
		return nil
	}

	if !n.status.CanTransitionTo(vo.NodeStatusInactive) {
		return fmt.Errorf("cannot deactivate node with status %s", n.status)
	}

	n.status = vo.NodeStatusInactive
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// EnterMaintenance puts the node into maintenance mode
func (n *Node) EnterMaintenance(reason string) error {
	if n.status == vo.NodeStatusMaintenance {
		return nil
	}

	if !n.status.CanTransitionTo(vo.NodeStatusMaintenance) {
		return fmt.Errorf("cannot enter maintenance mode from status %s", n.status)
	}

	if reason == "" {
		return fmt.Errorf("maintenance reason is required")
	}

	n.status = vo.NodeStatusMaintenance
	n.maintenanceReason = &reason
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// ExitMaintenance exits maintenance mode and returns to active status
func (n *Node) ExitMaintenance() error {
	if n.status != vo.NodeStatusMaintenance {
		return fmt.Errorf("node is not in maintenance mode")
	}

	n.status = vo.NodeStatusActive
	n.maintenanceReason = nil
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// UpdateServerAddress updates the server address
func (n *Node) UpdateServerAddress(address vo.ServerAddress) error {
	if n.serverAddress.Value() == address.Value() {
		return nil
	}

	n.serverAddress = address
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// UpdateAgentPort updates the agent connection port
func (n *Node) UpdateAgentPort(port uint16) error {
	if port == 0 {
		return fmt.Errorf("agent port cannot be zero")
	}

	if n.agentPort == port {
		return nil
	}

	n.agentPort = port
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// UpdateSubscriptionPort updates the subscription port
func (n *Node) UpdateSubscriptionPort(port *uint16) error {
	if port != nil && *port == 0 {
		return fmt.Errorf("subscription port cannot be zero")
	}

	// Check if values are equal
	if n.subscriptionPort == nil && port == nil {
		return nil
	}
	if n.subscriptionPort != nil && port != nil && *n.subscriptionPort == *port {
		return nil
	}

	n.subscriptionPort = port
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// UpdateEncryption updates the encryption configuration
func (n *Node) UpdateEncryption(config vo.EncryptionConfig) error {
	n.encryptionConfig = config
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// UpdatePlugin updates the plugin configuration
func (n *Node) UpdatePlugin(config *vo.PluginConfig) error {
	n.pluginConfig = config
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// UpdateTrojanConfig updates the trojan configuration
func (n *Node) UpdateTrojanConfig(config *vo.TrojanConfig) error {
	if !n.protocol.IsTrojan() {
		return fmt.Errorf("cannot update trojan config for non-trojan protocol")
	}

	n.trojanConfig = config
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// UpdateMetadata updates the node metadata
func (n *Node) UpdateMetadata(metadata vo.NodeMetadata) error {
	n.metadata = metadata
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// UpdateName updates the node name
func (n *Node) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("node name cannot be empty")
	}

	if n.name == name {
		return nil
	}

	n.name = name
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// UpdateSortOrder updates the sort order
func (n *Node) UpdateSortOrder(order int) error {
	n.sortOrder = order
	n.updatedAt = time.Now()
	n.version++

	return nil
}

func (n *Node) GenerateAPIToken() (string, error) {
	if n.tokenGenerator == nil {
		n.tokenGenerator = services.NewTokenGenerator()
	}

	plainToken, tokenHash, err := n.tokenGenerator.GenerateAPIToken("node")
	if err != nil {
		return "", fmt.Errorf("failed to generate API token: %w", err)
	}

	n.apiToken = plainToken
	n.tokenHash = tokenHash
	n.updatedAt = time.Now()
	n.version++

	return plainToken, nil
}

func (n *Node) VerifyAPIToken(plainToken string) bool {
	if n.tokenGenerator == nil {
		n.tokenGenerator = services.NewTokenGenerator()
	}
	computedHash := n.tokenGenerator.HashToken(plainToken)
	return subtle.ConstantTimeCompare([]byte(n.tokenHash), []byte(computedHash)) == 1
}

// IsAvailable checks if node is available for use
func (n *Node) IsAvailable() bool {
	return n.status == vo.NodeStatusActive
}

// LastSeenAt returns the last time the node agent reported status
func (n *Node) LastSeenAt() *time.Time {
	return n.lastSeenAt
}

// IsOnline checks if node agent is online (reported within 5 minutes)
func (n *Node) IsOnline() bool {
	if n.lastSeenAt == nil {
		return false
	}
	return time.Since(*n.lastSeenAt) < 5*time.Minute
}

// PublicIPv4 returns the public IPv4 address reported by agent
func (n *Node) PublicIPv4() *string {
	return n.publicIPv4
}

// PublicIPv6 returns the public IPv6 address reported by agent
func (n *Node) PublicIPv6() *string {
	return n.publicIPv6
}

// GetAPIToken returns the plain API token (only available after creation)
func (n *Node) GetAPIToken() string {
	return n.apiToken
}

// ClearAPIToken clears the plain API token from memory
func (n *Node) ClearAPIToken() {
	n.apiToken = ""
}

// Validate performs domain-level validation
func (n *Node) Validate() error {
	if n.name == "" {
		return fmt.Errorf("node name is required")
	}
	if n.agentPort == 0 {
		return fmt.Errorf("agent port is required")
	}
	if n.tokenHash == "" {
		return fmt.Errorf("token hash is required")
	}
	if !n.protocol.IsValid() {
		return fmt.Errorf("invalid protocol: %s", n.protocol)
	}
	if n.protocol.IsShadowsocks() && n.encryptionConfig.Method() == "" {
		return fmt.Errorf("encryption config is required for Shadowsocks protocol")
	}
	if n.protocol.IsTrojan() && n.trojanConfig == nil {
		return fmt.Errorf("trojan config is required for Trojan protocol")
	}
	if n.status == vo.NodeStatusMaintenance && n.maintenanceReason == nil {
		return fmt.Errorf("maintenance reason is required when in maintenance mode")
	}
	return nil
}

// GenerateSubscriptionURI generates a subscription URI for this node
// The password parameter should be the subscription UUID
// Uses EffectiveSubscriptionPort() for the port (subscriptionPort if set, otherwise agentPort)
func (n *Node) GenerateSubscriptionURI(password string, remarks string) (string, error) {
	factory := vo.NewProtocolConfigFactory()
	serverAddr := n.serverAddress.Value()
	port := n.EffectiveSubscriptionPort()

	switch n.protocol {
	case vo.ProtocolShadowsocks:
		ssConfig := vo.NewShadowsocksProtocolConfig(n.encryptionConfig, n.pluginConfig)
		return factory.GenerateSubscriptionURI(n.protocol, ssConfig, serverAddr, port, password, remarks)

	case vo.ProtocolTrojan:
		if n.trojanConfig == nil {
			return "", fmt.Errorf("trojan config is required for Trojan protocol")
		}
		trojanConfig := vo.NewTrojanProtocolConfig(*n.trojanConfig)
		return factory.GenerateSubscriptionURI(n.protocol, trojanConfig, serverAddr, port, password, remarks)

	default:
		return "", fmt.Errorf("unsupported protocol: %s", n.protocol)
	}
}

// generateAPIToken generates a new API token and its hash
