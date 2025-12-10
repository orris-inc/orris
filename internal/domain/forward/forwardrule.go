// Package forward provides domain models and business logic for TCP/UDP port forwarding.
package forward

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/forward/value_objects"
)

// ForwardRule represents the forward rule aggregate root.
type ForwardRule struct {
	id            uint
	shortID       string // external API identifier (Stripe-style)
	agentID       uint
	ruleType      vo.ForwardRuleType
	exitAgentID   uint   // exit agent ID (required for entry type)
	wsListenPort  uint16 // WebSocket listen port (required for exit type)
	name          string
	listenPort    uint16
	targetAddress string // final target address (required for direct and exit types if targetNodeID is not set)
	targetPort    uint16 // final target port (required for direct and exit types if targetNodeID is not set)
	targetNodeID  *uint  // target node ID for dynamic address resolution (mutually exclusive with targetAddress/targetPort)
	ipVersion     vo.IPVersion
	protocol      vo.ForwardProtocol
	status        vo.ForwardStatus
	remark        string
	uploadBytes   int64
	downloadBytes int64
	createdAt     time.Time
	updatedAt     time.Time
}

// NewForwardRule creates a new forward rule aggregate.
// Parameters depend on ruleType:
// - direct: requires agentID, listenPort, (targetAddress+targetPort OR targetNodeID)
// - entry: requires agentID, listenPort, exitAgentID, (targetAddress+targetPort OR targetNodeID)
func NewForwardRule(
	agentID uint,
	ruleType vo.ForwardRuleType,
	exitAgentID uint,
	wsListenPort uint16,
	name string,
	listenPort uint16,
	targetAddress string,
	targetPort uint16,
	targetNodeID *uint,
	ipVersion vo.IPVersion,
	protocol vo.ForwardProtocol,
	remark string,
	shortIDGenerator func() (string, error),
) (*ForwardRule, error) {
	if agentID == 0 {
		return nil, fmt.Errorf("agent ID is required")
	}
	if !ruleType.IsValid() {
		return nil, fmt.Errorf("invalid rule type: %s", ruleType)
	}
	if name == "" {
		return nil, fmt.Errorf("forward rule name is required")
	}
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", protocol)
	}

	// Validate required fields based on rule type
	switch ruleType {
	case vo.ForwardRuleTypeDirect:
		if listenPort == 0 {
			return nil, fmt.Errorf("listen port is required for direct forward")
		}
		// Either targetAddress+targetPort OR targetNodeID must be set
		hasTarget := targetAddress != "" && targetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return nil, fmt.Errorf("either target address+port or target node ID is required for direct forward")
		}
		if hasTarget && hasTargetNode {
			return nil, fmt.Errorf("target address+port and target node ID are mutually exclusive for direct forward")
		}
		if hasTarget {
			if err := validateAddress(targetAddress); err != nil {
				return nil, fmt.Errorf("invalid target address: %w", err)
			}
		}
	case vo.ForwardRuleTypeEntry:
		if listenPort == 0 {
			return nil, fmt.Errorf("listen port is required for entry forward")
		}
		if exitAgentID == 0 {
			return nil, fmt.Errorf("exit agent ID is required for entry forward")
		}
		// Entry rules now also require target information (to be passed to exit agent)
		hasTarget := targetAddress != "" && targetPort != 0
		hasTargetNode := targetNodeID != nil && *targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return nil, fmt.Errorf("either target address+port or target node ID is required for entry forward")
		}
		if hasTarget && hasTargetNode {
			return nil, fmt.Errorf("target address+port and target node ID are mutually exclusive for entry forward")
		}
		if hasTarget {
			if err := validateAddress(targetAddress); err != nil {
				return nil, fmt.Errorf("invalid target address: %w", err)
			}
		}
	}

	// Default ipVersion to auto if not set
	if ipVersion == "" {
		ipVersion = vo.IPVersionAuto
	}
	if !ipVersion.IsValid() {
		return nil, fmt.Errorf("invalid IP version: %s", ipVersion)
	}

	// Generate short ID for external API use
	shortID, err := shortIDGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to generate short ID: %w", err)
	}

	now := time.Now()
	return &ForwardRule{
		shortID:       shortID,
		agentID:       agentID,
		ruleType:      ruleType,
		exitAgentID:   exitAgentID,
		wsListenPort:  wsListenPort,
		name:          name,
		listenPort:    listenPort,
		targetAddress: targetAddress,
		targetPort:    targetPort,
		targetNodeID:  targetNodeID,
		ipVersion:     ipVersion,
		protocol:      protocol,
		status:        vo.ForwardStatusDisabled,
		remark:        remark,
		uploadBytes:   0,
		downloadBytes: 0,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// ReconstructForwardRule reconstructs a forward rule from persistence.
func ReconstructForwardRule(
	id uint,
	shortID string,
	agentID uint,
	ruleType vo.ForwardRuleType,
	exitAgentID uint,
	wsListenPort uint16,
	name string,
	listenPort uint16,
	targetAddress string,
	targetPort uint16,
	targetNodeID *uint,
	ipVersion vo.IPVersion,
	protocol vo.ForwardProtocol,
	status vo.ForwardStatus,
	remark string,
	uploadBytes int64,
	downloadBytes int64,
	createdAt, updatedAt time.Time,
) (*ForwardRule, error) {
	if id == 0 {
		return nil, fmt.Errorf("forward rule ID cannot be zero")
	}
	if shortID == "" {
		return nil, fmt.Errorf("forward rule short ID is required")
	}
	if agentID == 0 {
		return nil, fmt.Errorf("agent ID is required")
	}
	if !ruleType.IsValid() {
		return nil, fmt.Errorf("invalid rule type: %s", ruleType)
	}
	if name == "" {
		return nil, fmt.Errorf("forward rule name is required")
	}
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", protocol)
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	// Default ipVersion to auto if not set
	if ipVersion == "" {
		ipVersion = vo.IPVersionAuto
	}

	return &ForwardRule{
		id:            id,
		shortID:       shortID,
		agentID:       agentID,
		ruleType:      ruleType,
		exitAgentID:   exitAgentID,
		wsListenPort:  wsListenPort,
		name:          name,
		listenPort:    listenPort,
		targetAddress: targetAddress,
		targetPort:    targetPort,
		targetNodeID:  targetNodeID,
		ipVersion:     ipVersion,
		protocol:      protocol,
		status:        status,
		remark:        remark,
		uploadBytes:   uploadBytes,
		downloadBytes: downloadBytes,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}, nil
}

// validateAddress validates the target address format.
func validateAddress(address string) error {
	// Check if it's a valid IP
	if ip := net.ParseIP(address); ip != nil {
		return nil
	}

	// Check if it's a valid domain (basic validation)
	if len(address) > 0 && len(address) <= 253 {
		// Simple domain validation
		parts := strings.Split(address, ".")
		if len(parts) >= 2 {
			for _, part := range parts {
				if len(part) == 0 || len(part) > 63 {
					return fmt.Errorf("invalid domain format")
				}
			}
			return nil
		}
	}

	return fmt.Errorf("address must be a valid IP or domain")
}

// Getters

// ID returns the forward rule ID.
func (r *ForwardRule) ID() uint {
	return r.id
}

// ShortID returns the external API identifier.
func (r *ForwardRule) ShortID() string {
	return r.shortID
}

// AgentID returns the forward agent ID.
func (r *ForwardRule) AgentID() uint {
	return r.agentID
}

// RuleType returns the rule type.
func (r *ForwardRule) RuleType() vo.ForwardRuleType {
	return r.ruleType
}

// ExitAgentID returns the exit agent ID (for entry type rules).
func (r *ForwardRule) ExitAgentID() uint {
	return r.exitAgentID
}

// WsListenPort returns the WebSocket listen port (for exit type rules).
func (r *ForwardRule) WsListenPort() uint16 {
	return r.wsListenPort
}

// Name returns the forward rule name.
func (r *ForwardRule) Name() string {
	return r.name
}

// ListenPort returns the listen port.
func (r *ForwardRule) ListenPort() uint16 {
	return r.listenPort
}

// TargetAddress returns the target address.
func (r *ForwardRule) TargetAddress() string {
	return r.targetAddress
}

// TargetPort returns the target port.
func (r *ForwardRule) TargetPort() uint16 {
	return r.targetPort
}

// TargetNodeID returns the target node ID.
func (r *ForwardRule) TargetNodeID() *uint {
	return r.targetNodeID
}

// HasTargetNode returns true if targetNodeID is set.
func (r *ForwardRule) HasTargetNode() bool {
	return r.targetNodeID != nil && *r.targetNodeID != 0
}

// IPVersion returns the IP version preference.
func (r *ForwardRule) IPVersion() vo.IPVersion {
	return r.ipVersion
}

// Target returns the full target address with port.
func (r *ForwardRule) Target() string {
	return net.JoinHostPort(r.targetAddress, strconv.Itoa(int(r.targetPort)))
}

// Protocol returns the protocol.
func (r *ForwardRule) Protocol() vo.ForwardProtocol {
	return r.protocol
}

// Status returns the status.
func (r *ForwardRule) Status() vo.ForwardStatus {
	return r.status
}

// Remark returns the remark.
func (r *ForwardRule) Remark() string {
	return r.remark
}

// UploadBytes returns the upload bytes count.
func (r *ForwardRule) UploadBytes() int64 {
	return r.uploadBytes
}

// DownloadBytes returns the download bytes count.
func (r *ForwardRule) DownloadBytes() int64 {
	return r.downloadBytes
}

// TotalBytes returns the total bytes count.
func (r *ForwardRule) TotalBytes() int64 {
	return r.uploadBytes + r.downloadBytes
}

// CreatedAt returns when the rule was created.
func (r *ForwardRule) CreatedAt() time.Time {
	return r.createdAt
}

// UpdatedAt returns when the rule was last updated.
func (r *ForwardRule) UpdatedAt() time.Time {
	return r.updatedAt
}

// Setters and business operations

// SetID sets the forward rule ID (only for persistence layer use).
func (r *ForwardRule) SetID(id uint) error {
	if r.id != 0 {
		return fmt.Errorf("forward rule ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("forward rule ID cannot be zero")
	}
	r.id = id
	return nil
}

// Enable enables the forward rule.
func (r *ForwardRule) Enable() error {
	if r.status.IsEnabled() {
		return nil
	}
	r.status = vo.ForwardStatusEnabled
	r.updatedAt = time.Now()
	return nil
}

// Disable disables the forward rule.
func (r *ForwardRule) Disable() error {
	if r.status.IsDisabled() {
		return nil
	}
	r.status = vo.ForwardStatusDisabled
	r.updatedAt = time.Now()
	return nil
}

// UpdateName updates the rule name.
func (r *ForwardRule) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("forward rule name cannot be empty")
	}
	if r.name == name {
		return nil
	}
	r.name = name
	r.updatedAt = time.Now()
	return nil
}

// UpdateListenPort updates the listen port.
func (r *ForwardRule) UpdateListenPort(port uint16) error {
	if port == 0 {
		return fmt.Errorf("listen port cannot be zero")
	}
	if r.listenPort == port {
		return nil
	}
	r.listenPort = port
	r.updatedAt = time.Now()
	return nil
}

// UpdateTarget updates the target address and port.
// This will clear the targetNodeID when setting static address.
func (r *ForwardRule) UpdateTarget(address string, port uint16) error {
	if address == "" {
		return fmt.Errorf("target address cannot be empty")
	}
	if port == 0 {
		return fmt.Errorf("target port cannot be zero")
	}
	if err := validateAddress(address); err != nil {
		return fmt.Errorf("invalid target address: %w", err)
	}

	if r.targetAddress == address && r.targetPort == port && r.targetNodeID == nil {
		return nil
	}

	r.targetAddress = address
	r.targetPort = port
	r.targetNodeID = nil // clear targetNodeID when setting static address
	r.updatedAt = time.Now()
	return nil
}

// UpdateTargetNodeID updates the target node ID for dynamic address resolution.
// This will clear the targetAddress and targetPort when setting node ID.
func (r *ForwardRule) UpdateTargetNodeID(nodeID *uint) error {
	// Only direct and entry types support targetNodeID (exit type has been removed)
	if !r.ruleType.IsDirect() && !r.ruleType.IsEntry() {
		return fmt.Errorf("target node ID can only be set for direct or entry type rules")
	}

	// If nodeID is nil or 0, clear the targetNodeID
	if nodeID == nil || *nodeID == 0 {
		r.targetNodeID = nil
		r.updatedAt = time.Now()
		return nil
	}

	// Check if already set to the same value
	if r.targetNodeID != nil && *r.targetNodeID == *nodeID && r.targetAddress == "" && r.targetPort == 0 {
		return nil
	}

	r.targetNodeID = nodeID
	r.targetAddress = "" // clear static address when setting node ID
	r.targetPort = 0
	r.updatedAt = time.Now()
	return nil
}

// UpdateIPVersion updates the IP version preference.
func (r *ForwardRule) UpdateIPVersion(ipVersion vo.IPVersion) error {
	if ipVersion == "" {
		ipVersion = vo.IPVersionAuto
	}
	if !ipVersion.IsValid() {
		return fmt.Errorf("invalid IP version: %s", ipVersion)
	}
	if r.ipVersion == ipVersion {
		return nil
	}
	r.ipVersion = ipVersion
	r.updatedAt = time.Now()
	return nil
}

// UpdateProtocol updates the protocol.
func (r *ForwardRule) UpdateProtocol(protocol vo.ForwardProtocol) error {
	if !protocol.IsValid() {
		return fmt.Errorf("invalid protocol: %s", protocol)
	}
	if r.protocol == protocol {
		return nil
	}
	r.protocol = protocol
	r.updatedAt = time.Now()
	return nil
}

// UpdateRemark updates the remark.
func (r *ForwardRule) UpdateRemark(remark string) error {
	if r.remark == remark {
		return nil
	}
	r.remark = remark
	r.updatedAt = time.Now()
	return nil
}

// UpdateExitAgentID updates the exit agent ID for entry type rules.
func (r *ForwardRule) UpdateExitAgentID(exitAgentID uint) error {
	if !r.ruleType.IsEntry() {
		return fmt.Errorf("exit agent ID can only be updated for entry type rules")
	}
	if exitAgentID == 0 {
		return fmt.Errorf("exit agent ID cannot be zero")
	}
	if r.exitAgentID == exitAgentID {
		return nil
	}
	r.exitAgentID = exitAgentID
	r.updatedAt = time.Now()
	return nil
}

// UpdateWsListenPort updates the WebSocket listen port (deprecated - exit type has been removed).
func (r *ForwardRule) UpdateWsListenPort(port uint16) error {
	return fmt.Errorf("WebSocket listen port update is not supported (exit type has been removed)")
}

// RecordTraffic records traffic bytes.
func (r *ForwardRule) RecordTraffic(upload, download int64) {
	r.uploadBytes += upload
	r.downloadBytes += download
	r.updatedAt = time.Now()
}

// ResetTraffic resets the traffic counters.
func (r *ForwardRule) ResetTraffic() {
	r.uploadBytes = 0
	r.downloadBytes = 0
	r.updatedAt = time.Now()
}

// IsEnabled checks if the rule is enabled.
func (r *ForwardRule) IsEnabled() bool {
	return r.status.IsEnabled()
}

// Validate performs domain-level validation.
func (r *ForwardRule) Validate() error {
	if r.agentID == 0 {
		return fmt.Errorf("agent ID is required")
	}
	if !r.ruleType.IsValid() {
		return fmt.Errorf("invalid rule type: %s", r.ruleType)
	}
	if r.name == "" {
		return fmt.Errorf("forward rule name is required")
	}
	if !r.protocol.IsValid() {
		return fmt.Errorf("invalid protocol: %s", r.protocol)
	}
	if !r.status.IsValid() {
		return fmt.Errorf("invalid status: %s", r.status)
	}

	// Validate required fields based on rule type
	switch r.ruleType {
	case vo.ForwardRuleTypeDirect:
		if r.listenPort == 0 {
			return fmt.Errorf("listen port is required for direct forward")
		}
		// Either targetAddress+targetPort OR targetNodeID must be set
		hasTarget := r.targetAddress != "" && r.targetPort != 0
		hasTargetNode := r.targetNodeID != nil && *r.targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return fmt.Errorf("either target address+port or target node ID is required for direct forward")
		}
		if hasTarget && hasTargetNode {
			return fmt.Errorf("target address+port and target node ID are mutually exclusive for direct forward")
		}
	case vo.ForwardRuleTypeEntry:
		if r.listenPort == 0 {
			return fmt.Errorf("listen port is required for entry forward")
		}
		if r.exitAgentID == 0 {
			return fmt.Errorf("exit agent ID is required for entry forward")
		}
		// Entry rules now also require target information (to be passed to exit agent)
		hasTarget := r.targetAddress != "" && r.targetPort != 0
		hasTargetNode := r.targetNodeID != nil && *r.targetNodeID != 0
		if !hasTarget && !hasTargetNode {
			return fmt.Errorf("either target address+port or target node ID is required for entry forward")
		}
		if hasTarget && hasTargetNode {
			return fmt.Errorf("target address+port and target node ID are mutually exclusive for entry forward")
		}
	}

	return nil
}
