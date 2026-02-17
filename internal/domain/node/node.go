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
	anytlsConfig      *vo.AnyTLSConfig
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
	dnsConfig         *vo.DnsConfig   // DNS configuration for DNS-based unlocking
	lastSeenAt        *time.Time      // last time the node agent reported status
	publicIPv4        *string         // public IPv4 address reported by agent
	publicIPv6        *string         // public IPv6 address reported by agent
	agentVersion      *string         // agent software version (e.g., "1.2.3")
	platform          *string         // OS platform (linux, darwin, windows)
	arch              *string         // CPU architecture (amd64, arm64, arm, 386)
	expiresAt         *time.Time      // expiration time (nil = never expires)
	costLabel         *string         // cost label for display (e.g., "35$/m", "35¥/y")
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
	anytlsConfig *vo.AnyTLSConfig,
	metadata vo.NodeMetadata,
	sortOrder int,
	routeConfig *vo.RouteConfig,
	dnsConfig *vo.DnsConfig,
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
	if protocol.IsAnyTLS() && anytlsConfig == nil {
		return nil, fmt.Errorf("anytls config is required for AnyTLS protocol")
	}

	// Validate route config if provided
	if routeConfig != nil {
		if err := routeConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid route config: %w", err)
		}
	}

	// Validate dns config if provided
	if dnsConfig != nil {
		if err := dnsConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid dns config: %w", err)
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
		anytlsConfig:     anytlsConfig,
		status:           vo.NodeStatusInactive,
		metadata:         metadata,
		apiToken:         plainToken,
		tokenHash:        tokenHash,
		sortOrder:        sortOrder,
		routeConfig:      routeConfig,
		dnsConfig:        dnsConfig,
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
	anytlsConfig *vo.AnyTLSConfig,
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
	dnsConfig *vo.DnsConfig,
	lastSeenAt *time.Time,
	publicIPv4 *string,
	publicIPv6 *string,
	agentVersion *string,
	platform *string,
	arch *string,
	expiresAt *time.Time,
	costLabel *string,
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
		anytlsConfig:      anytlsConfig,
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
		dnsConfig:         dnsConfig,
		lastSeenAt:        lastSeenAt,
		publicIPv4:        publicIPv4,
		publicIPv6:        publicIPv6,
		agentVersion:      agentVersion,
		platform:          platform,
		arch:              arch,
		expiresAt:         expiresAt,
		costLabel:         costLabel,
		version:           version,
		originalVersion:   version, // preserve original version for optimistic locking
		createdAt:         createdAt,
		updatedAt:         updatedAt,
		tokenGenerator:    services.NewTokenGenerator(),
	}, nil
}

// --- Getters ---

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

// AnyTLSConfig returns the AnyTLS configuration
func (n *Node) AnyTLSConfig() *vo.AnyTLSConfig {
	return n.anytlsConfig
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
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.routeConfig
}

// DnsConfig returns the DNS configuration
func (n *Node) DnsConfig() *vo.DnsConfig {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.dnsConfig
}

// LastSeenAt returns the last time the node agent reported status
func (n *Node) LastSeenAt() *time.Time {
	return n.lastSeenAt
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

// ExpiresAt returns the expiration time (nil means never expires)
func (n *Node) ExpiresAt() *time.Time {
	return n.expiresAt
}

// CostLabel returns the cost label for display (nil means not set)
func (n *Node) CostLabel() *string {
	return n.costLabel
}

// MuteNotification returns whether notifications are muted for this node
func (n *Node) MuteNotification() bool {
	return n.muteNotification
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

// --- SetID ---

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

// --- Validate ---

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
	if n.protocol.IsAnyTLS() && n.anytlsConfig == nil {
		return fmt.Errorf("anytls config is required for AnyTLS protocol")
	}
	if n.status == vo.NodeStatusMaintenance && n.maintenanceReason == nil {
		return fmt.Errorf("maintenance reason is required when in maintenance mode")
	}
	return nil
}

// --- Token operations ---

// GenerateAPIToken generates a new API token and its hash
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

// VerifyAPIToken verifies a plain API token against the stored hash
func (n *Node) VerifyAPIToken(plainToken string) bool {
	if n.tokenGenerator == nil {
		n.tokenGenerator = services.NewTokenGenerator()
	}
	computedHash := n.tokenGenerator.HashToken(plainToken)
	return subtle.ConstantTimeCompare([]byte(n.tokenHash), []byte(computedHash)) == 1
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
