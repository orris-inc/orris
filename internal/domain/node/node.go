package node

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	vo "orris/internal/domain/node/value_objects"
)

// Node represents the node aggregate root
type Node struct {
	id                uint
	name              string
	serverAddress     vo.ServerAddress
	serverPort        uint16
	encryptionConfig  vo.EncryptionConfig
	pluginConfig      *vo.PluginConfig
	status            vo.NodeStatus
	metadata          vo.NodeMetadata
	apiToken          string
	tokenHash         string
	maxUsers          uint
	trafficLimit      uint64
	trafficUsed       uint64
	trafficResetAt    time.Time
	sortOrder         int
	maintenanceReason *string
	version           int
	createdAt         time.Time
	updatedAt         time.Time
	events            []interface{}
	mu                sync.RWMutex
}

// NewNode creates a new node aggregate
func NewNode(
	name string,
	serverAddress vo.ServerAddress,
	serverPort uint16,
	encryptionConfig vo.EncryptionConfig,
	pluginConfig *vo.PluginConfig,
	metadata vo.NodeMetadata,
	maxUsers uint,
	trafficLimit uint64,
	sortOrder int,
) (*Node, error) {
	if name == "" {
		return nil, fmt.Errorf("node name is required")
	}
	if serverPort == 0 {
		return nil, fmt.Errorf("server port is required")
	}

	plainToken, tokenHash, err := generateAPIToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API token: %w", err)
	}

	now := time.Now()
	n := &Node{
		name:             name,
		serverAddress:    serverAddress,
		serverPort:       serverPort,
		encryptionConfig: encryptionConfig,
		pluginConfig:     pluginConfig,
		status:           vo.NodeStatusInactive,
		metadata:         metadata,
		apiToken:         plainToken,
		tokenHash:        tokenHash,
		maxUsers:         maxUsers,
		trafficLimit:     trafficLimit,
		trafficUsed:      0,
		trafficResetAt:   now,
		sortOrder:        sortOrder,
		version:          1,
		createdAt:        now,
		updatedAt:        now,
		events:           []interface{}{},
	}

	n.recordEvent(NewNodeCreatedEvent(
		n.id,
		n.name,
		n.serverAddress.Value(),
		n.serverPort,
		n.status.String(),
		0,
	))

	return n, nil
}

// ReconstructNode reconstructs a node from persistence
func ReconstructNode(
	id uint,
	name string,
	serverAddress vo.ServerAddress,
	serverPort uint16,
	encryptionConfig vo.EncryptionConfig,
	pluginConfig *vo.PluginConfig,
	status vo.NodeStatus,
	metadata vo.NodeMetadata,
	tokenHash string,
	maxUsers uint,
	trafficLimit uint64,
	trafficUsed uint64,
	trafficResetAt time.Time,
	sortOrder int,
	maintenanceReason *string,
	version int,
	createdAt, updatedAt time.Time,
) (*Node, error) {
	if id == 0 {
		return nil, fmt.Errorf("node ID cannot be zero")
	}
	if name == "" {
		return nil, fmt.Errorf("node name is required")
	}
	if serverPort == 0 {
		return nil, fmt.Errorf("server port is required")
	}
	if tokenHash == "" {
		return nil, fmt.Errorf("token hash is required")
	}

	return &Node{
		id:                id,
		name:              name,
		serverAddress:     serverAddress,
		serverPort:        serverPort,
		encryptionConfig:  encryptionConfig,
		pluginConfig:      pluginConfig,
		status:            status,
		metadata:          metadata,
		tokenHash:         tokenHash,
		maxUsers:          maxUsers,
		trafficLimit:      trafficLimit,
		trafficUsed:       trafficUsed,
		trafficResetAt:    trafficResetAt,
		sortOrder:         sortOrder,
		maintenanceReason: maintenanceReason,
		version:           version,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
		events:            []interface{}{},
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

// ServerPort returns the server port
func (n *Node) ServerPort() uint16 {
	return n.serverPort
}

// EncryptionConfig returns the encryption configuration
func (n *Node) EncryptionConfig() vo.EncryptionConfig {
	return n.encryptionConfig
}

// PluginConfig returns the plugin configuration
func (n *Node) PluginConfig() *vo.PluginConfig {
	return n.pluginConfig
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

// MaxUsers returns the maximum number of users
func (n *Node) MaxUsers() uint {
	return n.maxUsers
}

// TrafficLimit returns the traffic limit in bytes
func (n *Node) TrafficLimit() uint64 {
	return n.trafficLimit
}

// TrafficUsed returns the used traffic in bytes
func (n *Node) TrafficUsed() uint64 {
	return n.trafficUsed
}

// TrafficResetAt returns when traffic was last reset
func (n *Node) TrafficResetAt() time.Time {
	return n.trafficResetAt
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

	oldStatus := n.status
	n.status = vo.NodeStatusActive
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeStatusChangedEvent(
		n.id,
		oldStatus.String(),
		n.status.String(),
		"Node activated",
	))

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

	oldStatus := n.status
	n.status = vo.NodeStatusInactive
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeStatusChangedEvent(
		n.id,
		oldStatus.String(),
		n.status.String(),
		"Node deactivated",
	))

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

	oldStatus := n.status
	n.status = vo.NodeStatusMaintenance
	n.maintenanceReason = &reason
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeStatusChangedEvent(
		n.id,
		oldStatus.String(),
		n.status.String(),
		reason,
	))

	return nil
}

// ExitMaintenance exits maintenance mode and returns to active status
func (n *Node) ExitMaintenance() error {
	if n.status != vo.NodeStatusMaintenance {
		return fmt.Errorf("node is not in maintenance mode")
	}

	oldStatus := n.status
	n.status = vo.NodeStatusActive
	n.maintenanceReason = nil
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeStatusChangedEvent(
		n.id,
		oldStatus.String(),
		n.status.String(),
		"Exited maintenance mode",
	))

	return nil
}

// UpdateServerAddress updates the server address
func (n *Node) UpdateServerAddress(address vo.ServerAddress) error {
	if n.serverAddress.Value() == address.Value() {
		return nil
	}

	oldAddress := n.serverAddress.Value()
	n.serverAddress = address
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeUpdatedEvent(
		n.id,
		[]string{"server_address"},
		map[string]interface{}{"server_address": oldAddress},
		map[string]interface{}{"server_address": address.Value()},
		0,
	))

	return nil
}

// UpdateServerPort updates the server port
func (n *Node) UpdateServerPort(port uint16) error {
	if port == 0 {
		return fmt.Errorf("server port cannot be zero")
	}

	if n.serverPort == port {
		return nil
	}

	oldPort := n.serverPort
	n.serverPort = port
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeUpdatedEvent(
		n.id,
		[]string{"server_port"},
		map[string]interface{}{"server_port": oldPort},
		map[string]interface{}{"server_port": port},
		0,
	))

	return nil
}

// UpdateEncryption updates the encryption configuration
func (n *Node) UpdateEncryption(config vo.EncryptionConfig) error {
	n.encryptionConfig = config
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeUpdatedEvent(
		n.id,
		[]string{"encryption_config"},
		map[string]interface{}{},
		map[string]interface{}{},
		0,
	))

	return nil
}

// UpdatePlugin updates the plugin configuration
func (n *Node) UpdatePlugin(config *vo.PluginConfig) error {
	n.pluginConfig = config
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeUpdatedEvent(
		n.id,
		[]string{"plugin_config"},
		map[string]interface{}{},
		map[string]interface{}{},
		0,
	))

	return nil
}

// UpdateMetadata updates the node metadata
func (n *Node) UpdateMetadata(metadata vo.NodeMetadata) error {
	n.metadata = metadata
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeUpdatedEvent(
		n.id,
		[]string{"metadata"},
		map[string]interface{}{},
		map[string]interface{}{},
		0,
	))

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

	oldName := n.name
	n.name = name
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeUpdatedEvent(
		n.id,
		[]string{"name"},
		map[string]interface{}{"name": oldName},
		map[string]interface{}{"name": name},
		0,
	))

	return nil
}

// UpdateMaxUsers updates the maximum number of users
func (n *Node) UpdateMaxUsers(maxUsers uint) error {
	n.maxUsers = maxUsers
	n.updatedAt = time.Now()
	n.version++

	return nil
}

// UpdateTrafficLimit updates the traffic limit
func (n *Node) UpdateTrafficLimit(limit uint64) error {
	n.trafficLimit = limit
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

// GenerateAPIToken generates a new API token
func (n *Node) GenerateAPIToken() (string, error) {
	plainToken, tokenHash, err := generateAPIToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate API token: %w", err)
	}

	n.apiToken = plainToken
	n.tokenHash = tokenHash
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeUpdatedEvent(
		n.id,
		[]string{"api_token"},
		map[string]interface{}{},
		map[string]interface{}{},
		0,
	))

	return plainToken, nil
}

// VerifyAPIToken verifies the provided API token
func (n *Node) VerifyAPIToken(plainToken string) bool {
	hash := sha256.Sum256([]byte(plainToken))
	tokenHash := hex.EncodeToString(hash[:])
	return subtle.ConstantTimeCompare([]byte(n.tokenHash), []byte(tokenHash)) == 1
}

// RecordTraffic records traffic usage
func (n *Node) RecordTraffic(upload, download uint64) error {
	if upload == 0 && download == 0 {
		return nil
	}

	n.trafficUsed += upload + download
	n.updatedAt = time.Now()

	if n.IsTrafficExceeded() {
		n.recordEvent(NewNodeTrafficExceededEvent(
			n.id,
			n.trafficLimit,
			n.trafficUsed,
		))
	}

	return nil
}

// IsTrafficExceeded checks if traffic limit is exceeded
func (n *Node) IsTrafficExceeded() bool {
	if n.trafficLimit == 0 {
		return false
	}
	return n.trafficUsed >= n.trafficLimit
}

// ResetTraffic resets traffic usage
func (n *Node) ResetTraffic() error {
	n.trafficUsed = 0
	n.trafficResetAt = time.Now()
	n.updatedAt = time.Now()
	n.version++

	n.recordEvent(NewNodeUpdatedEvent(
		n.id,
		[]string{"traffic_used"},
		map[string]interface{}{"traffic_used": n.trafficUsed},
		map[string]interface{}{"traffic_used": uint64(0)},
		0,
	))

	return nil
}

// IsAvailable checks if node is available for use
func (n *Node) IsAvailable() bool {
	if n.status != vo.NodeStatusActive {
		return false
	}

	if n.IsTrafficExceeded() {
		return false
	}

	return true
}

// GetAPIToken returns the plain API token (only available after creation)
func (n *Node) GetAPIToken() string {
	return n.apiToken
}

// ClearAPIToken clears the plain API token from memory
func (n *Node) ClearAPIToken() {
	n.apiToken = ""
}

// recordEvent records a domain event
func (n *Node) recordEvent(event interface{}) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = append(n.events, event)
}

// GetEvents returns and clears recorded domain events
func (n *Node) GetEvents() []interface{} {
	n.mu.Lock()
	defer n.mu.Unlock()
	events := n.events
	n.events = []interface{}{}
	return events
}

// ClearEvents clears all recorded events
func (n *Node) ClearEvents() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = []interface{}{}
}

// Validate performs domain-level validation
func (n *Node) Validate() error {
	if n.name == "" {
		return fmt.Errorf("node name is required")
	}
	if n.serverPort == 0 {
		return fmt.Errorf("server port is required")
	}
	if n.tokenHash == "" {
		return fmt.Errorf("token hash is required")
	}
	if n.status == vo.NodeStatusMaintenance && n.maintenanceReason == nil {
		return fmt.Errorf("maintenance reason is required when in maintenance mode")
	}
	return nil
}

// generateAPIToken generates a new API token and its hash
func generateAPIToken() (plainToken string, tokenHash string, err error) {
	tokenBytes := make([]byte, 32)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		return "", "", err
	}

	plainToken = "node_" + base64.RawURLEncoding.EncodeToString(tokenBytes)

	hash := sha256.Sum256([]byte(plainToken))
	tokenHash = hex.EncodeToString(hash[:])

	return plainToken, tokenHash, nil
}
