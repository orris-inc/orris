// Package node provides domain models and business logic for node management.
// It includes the Node aggregate root, node groups, traffic tracking, and token validation.
package node

import (
	"crypto/subtle"
	"fmt"
	"sync"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/domain/shared/services"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

// Node represents the node aggregate root
type Node struct {
	mu                sync.RWMutex // protects concurrent access to mutable fields
	id                uint
	sid               string // external API identifier (Stripe-style)
	name              string
	serverAddress     vo.ServerAddress
	agentPort         uint16  // port for agent connections (required)
	subscriptionPort  *uint16 // port for client subscriptions (if nil, use agentPort)
	protocol          vo.Protocol
	encryptionConfig  vo.EncryptionConfig
	pluginConfig      *vo.PluginConfig
	trojanConfig      *vo.TrojanConfig
	vlessConfig       *vo.VLESSConfig
	vmessConfig       *vo.VMessConfig
	hysteria2Config   *vo.Hysteria2Config
	tuicConfig        *vo.TUICConfig
	status            vo.NodeStatus
	metadata          vo.NodeMetadata
	groupIDs          []uint // resource group IDs
	userID            *uint  // owner user ID (nil = admin created, non-nil = user created)
	apiToken          string
	tokenHash         string
	sortOrder         int
	muteNotification  bool // mute online/offline notifications for this node
	maintenanceReason *string
	routeConfig       *vo.RouteConfig // routing configuration for traffic splitting
	lastSeenAt        *time.Time      // last time the node agent reported status
	publicIPv4        *string         // public IPv4 address reported by agent
	publicIPv6        *string         // public IPv6 address reported by agent
	agentVersion      *string         // agent software version (e.g., "1.2.3")
	platform          *string         // OS platform (linux, darwin, windows)
	arch              *string         // CPU architecture (amd64, arm64, arm, 386)
	expiresAt         *time.Time      // expiration time (nil = never expires)
	renewalAmount     *float64        // renewal amount for display (nil = not set)
	version           int
	originalVersion   int // version when loaded from database, for optimistic locking
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
	vlessConfig *vo.VLESSConfig,
	vmessConfig *vo.VMessConfig,
	hysteria2Config *vo.Hysteria2Config,
	tuicConfig *vo.TUICConfig,
	metadata vo.NodeMetadata,
	sortOrder int,
	routeConfig *vo.RouteConfig,
	sidGenerator func() (string, error),
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
	if protocol.IsVLESS() && vlessConfig == nil {
		return nil, fmt.Errorf("vless config is required for VLESS protocol")
	}
	if protocol.IsVMess() && vmessConfig == nil {
		return nil, fmt.Errorf("vmess config is required for VMess protocol")
	}
	if protocol.IsHysteria2() && hysteria2Config == nil {
		return nil, fmt.Errorf("hysteria2 config is required for Hysteria2 protocol")
	}
	if protocol.IsTUIC() && tuicConfig == nil {
		return nil, fmt.Errorf("tuic config is required for TUIC protocol")
	}

	// Validate route config if provided
	if routeConfig != nil {
		if err := routeConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid route config: %w", err)
		}
	}

	tokenGen := services.NewTokenGenerator()
	plainToken, tokenHash, err := tokenGen.GenerateAPIToken("node")
	if err != nil {
		return nil, fmt.Errorf("failed to generate API token: %w", err)
	}

	// Generate SID for external API use
	sid, err := sidGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := biztime.NowUTC()
	n := &Node{
		sid:              sid,
		name:             name,
		serverAddress:    serverAddress,
		agentPort:        agentPort,
		subscriptionPort: subscriptionPort,
		protocol:         protocol,
		encryptionConfig: encryptionConfig,
		pluginConfig:     pluginConfig,
		trojanConfig:     trojanConfig,
		vlessConfig:      vlessConfig,
		vmessConfig:      vmessConfig,
		hysteria2Config:  hysteria2Config,
		tuicConfig:       tuicConfig,
		status:           vo.NodeStatusInactive,
		metadata:         metadata,
		apiToken:         plainToken,
		tokenHash:        tokenHash,
		sortOrder:        sortOrder,
		routeConfig:      routeConfig,
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
	sid string,
	name string,
	serverAddress vo.ServerAddress,
	agentPort uint16,
	subscriptionPort *uint16,
	protocol vo.Protocol,
	encryptionConfig vo.EncryptionConfig,
	pluginConfig *vo.PluginConfig,
	trojanConfig *vo.TrojanConfig,
	vlessConfig *vo.VLESSConfig,
	vmessConfig *vo.VMessConfig,
	hysteria2Config *vo.Hysteria2Config,
	tuicConfig *vo.TUICConfig,
	status vo.NodeStatus,
	metadata vo.NodeMetadata,
	groupIDs []uint,
	userID *uint,
	tokenHash string,
	apiToken string,
	sortOrder int,
	muteNotification bool,
	maintenanceReason *string,
	routeConfig *vo.RouteConfig,
	lastSeenAt *time.Time,
	publicIPv4 *string,
	publicIPv6 *string,
	agentVersion *string,
	platform *string,
	arch *string,
	expiresAt *time.Time,
	renewalAmount *float64,
	version int,
	createdAt, updatedAt time.Time,
) (*Node, error) {
	if id == 0 {
		return nil, fmt.Errorf("node ID cannot be zero")
	}
	if sid == "" {
		return nil, fmt.Errorf("node SID is required")
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
		sid:               sid,
		name:              name,
		serverAddress:     serverAddress,
		agentPort:         agentPort,
		subscriptionPort:  subscriptionPort,
		protocol:          protocol,
		encryptionConfig:  encryptionConfig,
		pluginConfig:      pluginConfig,
		trojanConfig:      trojanConfig,
		vlessConfig:       vlessConfig,
		vmessConfig:       vmessConfig,
		hysteria2Config:   hysteria2Config,
		tuicConfig:        tuicConfig,
		status:            status,
		metadata:          metadata,
		groupIDs:          groupIDs,
		userID:            userID,
		tokenHash:         tokenHash,
		apiToken:          apiToken,
		sortOrder:         sortOrder,
		muteNotification:  muteNotification,
		maintenanceReason: maintenanceReason,
		routeConfig:       routeConfig,
		lastSeenAt:        lastSeenAt,
		publicIPv4:        publicIPv4,
		publicIPv6:        publicIPv6,
		agentVersion:      agentVersion,
		platform:          platform,
		arch:              arch,
		expiresAt:         expiresAt,
		renewalAmount:     renewalAmount,
		version:           version,
		originalVersion:   version, // preserve original version for optimistic locking
		createdAt:         createdAt,
		updatedAt:         updatedAt,
		tokenGenerator:    services.NewTokenGenerator(),
	}, nil
}

// ID returns the node ID
func (n *Node) ID() uint {
	return n.id
}

// SID returns the external API identifier
func (n *Node) SID() string {
	return n.sid
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

// VLESSConfig returns the VLESS configuration
func (n *Node) VLESSConfig() *vo.VLESSConfig {
	return n.vlessConfig
}

// VMessConfig returns the VMess configuration
func (n *Node) VMessConfig() *vo.VMessConfig {
	return n.vmessConfig
}

// Hysteria2Config returns the Hysteria2 configuration
func (n *Node) Hysteria2Config() *vo.Hysteria2Config {
	return n.hysteria2Config
}

// TUICConfig returns the TUIC configuration
func (n *Node) TUICConfig() *vo.TUICConfig {
	return n.tuicConfig
}

// Status returns the node status
func (n *Node) Status() vo.NodeStatus {
	return n.status
}

// Metadata returns the node metadata
func (n *Node) Metadata() vo.NodeMetadata {
	return n.metadata
}

// GroupIDs returns the resource group IDs
func (n *Node) GroupIDs() []uint {
	return n.groupIDs
}

// UserID returns the owner user ID (nil for admin-created nodes)
func (n *Node) UserID() *uint {
	return n.userID
}

// SetUserID sets the owner user ID
func (n *Node) SetUserID(userID *uint) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.userID = userID
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// SetGroupIDs sets the resource group IDs
func (n *Node) SetGroupIDs(groupIDs []uint) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.groupIDs = groupIDs
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// AddGroupID adds a resource group ID if not already present
func (n *Node) AddGroupID(groupID uint) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, id := range n.groupIDs {
		if id == groupID {
			return false // already exists
		}
	}
	n.groupIDs = append(n.groupIDs, groupID)
	n.updatedAt = biztime.NowUTC()
	n.version++
	return true
}

// RemoveGroupID removes a resource group ID
func (n *Node) RemoveGroupID(groupID uint) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i, id := range n.groupIDs {
		if id == groupID {
			n.groupIDs = append(n.groupIDs[:i], n.groupIDs[i+1:]...)
			n.updatedAt = biztime.NowUTC()
			n.version++
			return true
		}
	}
	return false // not found
}

// HasGroupID checks if the node belongs to a specific resource group
func (n *Node) HasGroupID(groupID uint) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, id := range n.groupIDs {
		if id == groupID {
			return true
		}
	}
	return false
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

// RouteConfig returns the routing configuration
func (n *Node) RouteConfig() *vo.RouteConfig {
	return n.routeConfig
}

// Version returns the aggregate version for optimistic locking
func (n *Node) Version() int {
	return n.version
}

// OriginalVersion returns the version when the entity was loaded from database.
// This is used for optimistic locking to detect concurrent modifications.
func (n *Node) OriginalVersion() int {
	return n.originalVersion
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
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status == vo.NodeStatusActive {
		return nil
	}

	if !n.status.CanTransitionTo(vo.NodeStatusActive) {
		return fmt.Errorf("cannot activate node with status %s", n.status)
	}

	n.status = vo.NodeStatusActive
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// Deactivate deactivates the node
func (n *Node) Deactivate() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status == vo.NodeStatusInactive {
		return nil
	}

	if !n.status.CanTransitionTo(vo.NodeStatusInactive) {
		return fmt.Errorf("cannot deactivate node with status %s", n.status)
	}

	n.status = vo.NodeStatusInactive
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// EnterMaintenance puts the node into maintenance mode
func (n *Node) EnterMaintenance(reason string) error {
	if reason == "" {
		return fmt.Errorf("maintenance reason is required")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status == vo.NodeStatusMaintenance {
		return nil
	}

	if !n.status.CanTransitionTo(vo.NodeStatusMaintenance) {
		return fmt.Errorf("cannot enter maintenance mode from status %s", n.status)
	}

	n.status = vo.NodeStatusMaintenance
	n.maintenanceReason = &reason
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// ExitMaintenance exits maintenance mode and returns to active status
func (n *Node) ExitMaintenance() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status != vo.NodeStatusMaintenance {
		return fmt.Errorf("node is not in maintenance mode")
	}

	n.status = vo.NodeStatusActive
	n.maintenanceReason = nil
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateServerAddress updates the server address
func (n *Node) UpdateServerAddress(address vo.ServerAddress) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.serverAddress.Value() == address.Value() {
		return nil
	}

	n.serverAddress = address
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateAgentPort updates the agent connection port
func (n *Node) UpdateAgentPort(port uint16) error {
	if port == 0 {
		return fmt.Errorf("agent port cannot be zero")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.agentPort == port {
		return nil
	}

	n.agentPort = port
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateSubscriptionPort updates the subscription port
func (n *Node) UpdateSubscriptionPort(port *uint16) error {
	if port != nil && *port == 0 {
		return fmt.Errorf("subscription port cannot be zero")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Check if values are equal
	if n.subscriptionPort == nil && port == nil {
		return nil
	}
	if n.subscriptionPort != nil && port != nil && *n.subscriptionPort == *port {
		return nil
	}

	n.subscriptionPort = port
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateEncryption updates the encryption configuration
func (n *Node) UpdateEncryption(config vo.EncryptionConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.encryptionConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdatePlugin updates the plugin configuration
func (n *Node) UpdatePlugin(config *vo.PluginConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.pluginConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateTrojanConfig updates the trojan configuration
func (n *Node) UpdateTrojanConfig(config *vo.TrojanConfig) error {
	if !n.protocol.IsTrojan() {
		return fmt.Errorf("cannot update trojan config for non-trojan protocol")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.trojanConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateVLESSConfig updates the VLESS configuration
func (n *Node) UpdateVLESSConfig(config *vo.VLESSConfig) error {
	if !n.protocol.IsVLESS() {
		return fmt.Errorf("cannot update vless config for non-vless protocol")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.vlessConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateVMessConfig updates the VMess configuration
func (n *Node) UpdateVMessConfig(config *vo.VMessConfig) error {
	if !n.protocol.IsVMess() {
		return fmt.Errorf("cannot update vmess config for non-vmess protocol")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.vmessConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateHysteria2Config updates the Hysteria2 configuration
func (n *Node) UpdateHysteria2Config(config *vo.Hysteria2Config) error {
	if !n.protocol.IsHysteria2() {
		return fmt.Errorf("cannot update hysteria2 config for non-hysteria2 protocol")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.hysteria2Config = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateTUICConfig updates the TUIC configuration
func (n *Node) UpdateTUICConfig(config *vo.TUICConfig) error {
	if !n.protocol.IsTUIC() {
		return fmt.Errorf("cannot update tuic config for non-tuic protocol")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.tuicConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateMetadata updates the node metadata
func (n *Node) UpdateMetadata(metadata vo.NodeMetadata) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.metadata = metadata
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateName updates the node name
func (n *Node) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("node name cannot be empty")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.name == name {
		return nil
	}

	n.name = name
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// UpdateSortOrder updates the sort order
func (n *Node) UpdateSortOrder(order int) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.sortOrder = order
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// MuteNotification returns whether notifications are muted for this node
func (n *Node) MuteNotification() bool {
	return n.muteNotification
}

// SetMuteNotification sets the mute notification flag
func (n *Node) SetMuteNotification(mute bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.muteNotification = mute
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// UpdateRouteConfig updates the routing configuration
func (n *Node) UpdateRouteConfig(config *vo.RouteConfig) error {
	if config != nil {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("invalid route config: %w", err)
		}
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.routeConfig = config
	n.updatedAt = biztime.NowUTC()
	n.version++

	return nil
}

// ClearRouteConfig removes the routing configuration
func (n *Node) ClearRouteConfig() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.routeConfig = nil
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// HasRouteConfig checks if the node has a routing configuration
func (n *Node) HasRouteConfig() bool {
	return n.routeConfig != nil
}

func (n *Node) GenerateAPIToken() (string, error) {
	if n.tokenGenerator == nil {
		n.tokenGenerator = services.NewTokenGenerator()
	}

	plainToken, tokenHash, err := n.tokenGenerator.GenerateAPIToken("node")
	if err != nil {
		return "", fmt.Errorf("failed to generate API token: %w", err)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.apiToken = plainToken
	n.tokenHash = tokenHash
	n.updatedAt = biztime.NowUTC()
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

// IsUserOwned returns true if this node is owned by a user (not admin-created)
func (n *Node) IsUserOwned() bool {
	return n.userID != nil
}

// IsOwnedBy checks if the node is owned by the specified user
func (n *Node) IsOwnedBy(userID uint) bool {
	return n.userID != nil && *n.userID == userID
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

// AgentVersion returns the agent software version
func (n *Node) AgentVersion() *string {
	return n.agentVersion
}

// AgentPlatform returns the OS platform
func (n *Node) AgentPlatform() *string {
	return n.platform
}

// AgentArch returns the CPU architecture
func (n *Node) AgentArch() *string {
	return n.arch
}

// EffectiveServerAddress returns the server address to use for outbound connections.
// If serverAddress is configured, it returns that; otherwise, it falls back to publicIPv4.
// Returns empty string if neither is available.
func (n *Node) EffectiveServerAddress() string {
	if n.serverAddress.Value() != "" {
		return n.serverAddress.Value()
	}
	if n.publicIPv4 != nil && *n.publicIPv4 != "" {
		return *n.publicIPv4
	}
	return ""
}

// GetAPIToken returns the plain API token (only available after creation)
func (n *Node) GetAPIToken() string {
	return n.apiToken
}

// ClearAPIToken clears the plain API token from memory
func (n *Node) ClearAPIToken() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.apiToken = ""
}

// ExpiresAt returns the expiration time (nil means never expires)
func (n *Node) ExpiresAt() *time.Time {
	return n.expiresAt
}

// SetExpiresAt sets the expiration time (nil to clear)
func (n *Node) SetExpiresAt(t *time.Time) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.expiresAt = t
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// RenewalAmount returns the renewal amount (nil means not set)
func (n *Node) RenewalAmount() *float64 {
	return n.renewalAmount
}

// SetRenewalAmount sets the renewal amount (nil to clear)
func (n *Node) SetRenewalAmount(amount *float64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.renewalAmount = amount
	n.updatedAt = biztime.NowUTC()
	n.version++
}

// IsExpired checks if the node has expired
func (n *Node) IsExpired() bool {
	if n.expiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*n.expiresAt)
}

// IsExpiringSoon checks if the node will expire within the specified number of days
func (n *Node) IsExpiringSoon(days int) bool {
	if n.expiresAt == nil {
		return false
	}
	threshold := time.Now().UTC().AddDate(0, 0, days)
	return n.expiresAt.Before(threshold)
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
	if n.protocol.IsVLESS() && n.vlessConfig == nil {
		return fmt.Errorf("vless config is required for VLESS protocol")
	}
	if n.protocol.IsVMess() && n.vmessConfig == nil {
		return fmt.Errorf("vmess config is required for VMess protocol")
	}
	if n.protocol.IsHysteria2() && n.hysteria2Config == nil {
		return fmt.Errorf("hysteria2 config is required for Hysteria2 protocol")
	}
	if n.protocol.IsTUIC() && n.tuicConfig == nil {
		return fmt.Errorf("tuic config is required for TUIC protocol")
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
