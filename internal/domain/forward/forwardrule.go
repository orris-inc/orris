// Package forward provides domain models and business logic for TCP/UDP port forwarding.
package forward

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	vo "orris/internal/domain/forward/value_objects"
)

// ForwardRule represents the forward rule aggregate root.
type ForwardRule struct {
	id            uint
	agentID       uint
	nextAgentID   uint // 0 means direct forward to target, >0 means chain forward to next agent
	name          string
	listenPort    uint16
	targetAddress string // final target address (used when nextAgentID=0)
	targetPort    uint16 // final target port (used when nextAgentID=0)
	protocol      vo.ForwardProtocol
	status        vo.ForwardStatus
	remark        string
	uploadBytes   int64
	downloadBytes int64
	createdAt     time.Time
	updatedAt     time.Time
}

// NewForwardRule creates a new forward rule aggregate.
// If nextAgentID > 0, it's a chain forward to next agent; targetAddress and targetPort are ignored.
// If nextAgentID = 0, it's a direct forward to targetAddress:targetPort.
func NewForwardRule(
	agentID uint,
	nextAgentID uint,
	name string,
	listenPort uint16,
	targetAddress string,
	targetPort uint16,
	protocol vo.ForwardProtocol,
	remark string,
) (*ForwardRule, error) {
	if agentID == 0 {
		return nil, fmt.Errorf("agent ID is required")
	}
	if name == "" {
		return nil, fmt.Errorf("forward rule name is required")
	}
	if listenPort == 0 {
		return nil, fmt.Errorf("listen port is required")
	}
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", protocol)
	}

	// For direct forward (nextAgentID=0), target is required
	if nextAgentID == 0 {
		if targetAddress == "" {
			return nil, fmt.Errorf("target address is required for direct forward")
		}
		if targetPort == 0 {
			return nil, fmt.Errorf("target port is required for direct forward")
		}
		if err := validateAddress(targetAddress); err != nil {
			return nil, fmt.Errorf("invalid target address: %w", err)
		}
	}

	now := time.Now()
	return &ForwardRule{
		agentID:       agentID,
		nextAgentID:   nextAgentID,
		name:          name,
		listenPort:    listenPort,
		targetAddress: targetAddress,
		targetPort:    targetPort,
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
	agentID uint,
	nextAgentID uint,
	name string,
	listenPort uint16,
	targetAddress string,
	targetPort uint16,
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
	if agentID == 0 {
		return nil, fmt.Errorf("agent ID is required")
	}
	if name == "" {
		return nil, fmt.Errorf("forward rule name is required")
	}
	if listenPort == 0 {
		return nil, fmt.Errorf("listen port is required")
	}
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", protocol)
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	return &ForwardRule{
		id:            id,
		agentID:       agentID,
		nextAgentID:   nextAgentID,
		name:          name,
		listenPort:    listenPort,
		targetAddress: targetAddress,
		targetPort:    targetPort,
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

// AgentID returns the forward agent ID.
func (r *ForwardRule) AgentID() uint {
	return r.agentID
}

// NextAgentID returns the next agent ID in the forward chain.
// Returns 0 if this is a direct forward to target.
func (r *ForwardRule) NextAgentID() uint {
	return r.nextAgentID
}

// IsChainForward returns true if this rule forwards to another agent.
func (r *ForwardRule) IsChainForward() bool {
	return r.nextAgentID > 0
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

	if r.targetAddress == address && r.targetPort == port {
		return nil
	}

	r.targetAddress = address
	r.targetPort = port
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

// UpdateNextAgentID updates the next agent ID for chain forwarding.
func (r *ForwardRule) UpdateNextAgentID(nextAgentID uint) {
	if r.nextAgentID == nextAgentID {
		return
	}
	r.nextAgentID = nextAgentID
	r.updatedAt = time.Now()
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
	if r.name == "" {
		return fmt.Errorf("forward rule name is required")
	}
	if r.listenPort == 0 {
		return fmt.Errorf("listen port is required")
	}
	// For direct forward (nextAgentID=0), target is required
	if r.nextAgentID == 0 {
		if r.targetAddress == "" {
			return fmt.Errorf("target address is required for direct forward")
		}
		if r.targetPort == 0 {
			return fmt.Errorf("target port is required for direct forward")
		}
	}
	if !r.protocol.IsValid() {
		return fmt.Errorf("invalid protocol: %s", r.protocol)
	}
	if !r.status.IsValid() {
		return fmt.Errorf("invalid status: %s", r.status)
	}
	return nil
}
