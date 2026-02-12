// Package forward provides domain models and business logic for TCP/UDP port forwarding.
package forward

import (
	"fmt"
	"net"
	"strconv"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

// ForwardRule represents the forward rule aggregate root.
type ForwardRule struct {
	id                  uint
	sid                 string // Stripe-style prefixed ID (fr_xxx)
	agentID             uint
	userID              *uint // user ID for user-owned rules (nil for admin-created rules)
	subscriptionID      *uint // subscription ID for subscription-bound rules (nil for admin-created rules)
	ruleType            vo.ForwardRuleType
	exitAgentID         uint                   // exit agent ID (required for entry type, mutually exclusive with exitAgents)
	exitAgents          []vo.AgentWeight       // multiple exit agents with weights for load balancing (mutually exclusive with exitAgentID)
	loadBalanceStrategy vo.LoadBalanceStrategy // load balance strategy for multi-exit rules (default: failover)
	chainAgentIDs       []uint                 // ordered array of intermediate agent IDs for chain forwarding
	chainPortConfig     map[uint]uint16        // map of agent_id -> listen_port for direct_chain type or hybrid chain direct hops
	tunnelHops          *int                   // number of hops using tunnel (nil=full tunnel, N=first N hops use tunnel)
	tunnelType          vo.TunnelType          // tunnel type: ws or tls (default: ws)
	name              string
	listenPort        uint16
	targetAddress     string // final target address (required for direct and exit types if targetNodeID is not set)
	targetPort        uint16 // final target port (required for direct and exit types if targetNodeID is not set)
	targetNodeID      *uint  // target node ID for dynamic address resolution (mutually exclusive with targetAddress/targetPort)
	bindIP            string // bind IP address for outbound connections (optional)
	ipVersion         vo.IPVersion
	protocol          vo.ForwardProtocol
	status            vo.ForwardStatus
	remark            string
	uploadBytes       int64
	downloadBytes     int64
	trafficMultiplier *float64 // traffic multiplier for display. nil means auto-calculate based on node count
	sortOrder         int
	groupIDs          []uint // resource group IDs for access control
	// External rule fields (used when ruleType = external)
	serverAddress  string // server address for external rules (replaces agent's public address)
	externalSource string // external source identifier (required for external rules)
	externalRuleID string // external rule ID for reference (optional)
	createdAt      time.Time
	updatedAt      time.Time
}

// NewForwardRule creates a new forward rule aggregate.
// Parameters depend on ruleType:
// - direct: requires agentID, listenPort, (targetAddress+targetPort OR targetNodeID)
// - entry: requires agentID, listenPort, (exitAgentID OR exitAgents), (targetAddress+targetPort OR targetNodeID)
//   - optionally loadBalanceStrategy for multi-exit rules (default: failover)
//
// - chain: requires agentID, listenPort, chainAgentIDs (at least 1), (targetAddress+targetPort OR targetNodeID)
//   - optionally tunnelHops to create hybrid chain (first N hops tunnel, rest direct)
//   - if tunnelHops > 0, chainPortConfig required for direct hops
//   - optionally exitAgents for last hop load balancing
//
// - direct_chain: requires agentID, listenPort, chainAgentIDs (at least 1), chainPortConfig, (targetAddress+targetPort OR targetNodeID)
func NewForwardRule(
	agentID uint,
	userID *uint,
	subscriptionID *uint,
	ruleType vo.ForwardRuleType,
	exitAgentID uint,
	exitAgents []vo.AgentWeight,
	loadBalanceStrategy vo.LoadBalanceStrategy,
	chainAgentIDs []uint,
	chainPortConfig map[uint]uint16,
	tunnelHops *int,
	tunnelType vo.TunnelType,
	name string,
	listenPort uint16,
	targetAddress string,
	targetPort uint16,
	targetNodeID *uint,
	bindIP string,
	ipVersion vo.IPVersion,
	protocol vo.ForwardProtocol,
	remark string,
	trafficMultiplier *float64,
	sortOrder int,
	shortIDGenerator func() (string, error),
) (*ForwardRule, error) {
	// Pre-construction check: reject external type early (use NewExternalForwardRule instead)
	if ruleType == vo.ForwardRuleTypeExternal {
		return nil, fmt.Errorf("use NewExternalForwardRule to create external forward rules")
	}

	// Apply defaults before constructing the aggregate
	if ipVersion == "" {
		ipVersion = vo.IPVersionAuto
	}
	if loadBalanceStrategy == "" {
		loadBalanceStrategy = vo.DefaultLoadBalanceStrategy
	}

	// Generate SID for external API use
	sid, err := shortIDGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := biztime.NowUTC()
	rule := &ForwardRule{
		sid:                 sid,
		agentID:             agentID,
		userID:              userID,
		subscriptionID:      subscriptionID,
		ruleType:            ruleType,
		exitAgentID:         exitAgentID,
		exitAgents:          exitAgents,
		loadBalanceStrategy: loadBalanceStrategy,
		chainAgentIDs:       chainAgentIDs,
		chainPortConfig:     chainPortConfig,
		tunnelHops:          tunnelHops,
		tunnelType:          tunnelType,
		name:                name,
		listenPort:          listenPort,
		targetAddress:       targetAddress,
		targetPort:          targetPort,
		targetNodeID:        targetNodeID,
		bindIP:              bindIP,
		ipVersion:           ipVersion,
		protocol:            protocol,
		status:              vo.ForwardStatusDisabled,
		remark:              remark,
		uploadBytes:         0,
		downloadBytes:       0,
		trafficMultiplier:   trafficMultiplier,
		sortOrder:           sortOrder,
		createdAt:           now,
		updatedAt:           now,
	}

	// Delegate all validation to the single Validate method
	if err := rule.Validate(); err != nil {
		return nil, err
	}

	return rule, nil
}

// NewExternalForwardRule creates a new external forward rule aggregate.
// External rules are for third-party forward services that don't use agents.
// Parameters:
//   - userID: optional user ID (nil for admin-created rules distributed via resource groups)
//   - subscriptionID: optional subscription ID (nil for admin-created rules)
//   - targetNodeID: required for protocol information (protocol is derived from target node)
//   - name: rule name
//   - serverAddress: required server address for subscription delivery
//   - listenPort: listen port
//   - externalSource: optional source identifier
//   - externalRuleID: optional external reference ID
//   - remark: optional description
//   - sortOrder: display sort order
//   - groupIDs: optional resource group IDs for distribution
//   - shortIDGenerator: function to generate SID
func NewExternalForwardRule(
	userID *uint,
	subscriptionID *uint,
	targetNodeID *uint,
	name string,
	serverAddress string,
	listenPort uint16,
	externalSource string,
	externalRuleID string,
	remark string,
	sortOrder int,
	groupIDs []uint,
	shortIDGenerator func() (string, error),
) (*ForwardRule, error) {
	if name == "" {
		return nil, fmt.Errorf("external forward rule name is required")
	}
	if serverAddress == "" {
		return nil, fmt.Errorf("server address is required for external forward")
	}
	if listenPort == 0 {
		return nil, fmt.Errorf("listen port is required for external forward")
	}
	if targetNodeID == nil || *targetNodeID == 0 {
		return nil, fmt.Errorf("target node ID is required for external forward (protocol is derived from target node)")
	}
	// externalSource is optional

	// Generate SID for external API use
	sid, err := shortIDGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := biztime.NowUTC()
	return &ForwardRule{
		sid:            sid,
		agentID:        0, // External rules don't have agents
		userID:         userID,
		subscriptionID: subscriptionID,
		ruleType:       vo.ForwardRuleTypeExternal,
		name:           name,
		listenPort:     listenPort,
		targetNodeID:   targetNodeID,
		protocol:       vo.ForwardProtocolTCP, // Default, will be determined from targetNodeID
		status:         vo.ForwardStatusEnabled,
		remark:         remark,
		sortOrder:      sortOrder,
		groupIDs:       groupIDs,
		serverAddress:  serverAddress,
		externalSource: externalSource,
		externalRuleID: externalRuleID,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// ReconstructForwardRule reconstructs a forward rule from persistence.
// It performs full validation to ensure data integrity, even for persisted data.
func ReconstructForwardRule(
	id uint,
	sid string,
	agentID uint,
	userID *uint,
	subscriptionID *uint,
	ruleType vo.ForwardRuleType,
	exitAgentID uint,
	exitAgents []vo.AgentWeight,
	loadBalanceStrategy vo.LoadBalanceStrategy,
	chainAgentIDs []uint,
	chainPortConfig map[uint]uint16,
	tunnelHops *int,
	tunnelType vo.TunnelType,
	name string,
	listenPort uint16,
	targetAddress string,
	targetPort uint16,
	targetNodeID *uint,
	bindIP string,
	ipVersion vo.IPVersion,
	protocol vo.ForwardProtocol,
	status vo.ForwardStatus,
	remark string,
	uploadBytes int64,
	downloadBytes int64,
	trafficMultiplier *float64,
	sortOrder int,
	groupIDs []uint,
	serverAddress string,
	externalSource string,
	externalRuleID string,
	createdAt, updatedAt time.Time,
) (*ForwardRule, error) {
	if id == 0 {
		return nil, fmt.Errorf("forward rule ID cannot be zero")
	}
	if sid == "" {
		return nil, fmt.Errorf("forward rule SID is required")
	}
	// Agent ID is required for non-external rules
	if ruleType.RequiresAgent() && agentID == 0 {
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
	// Validate external rule specific fields
	if ruleType.IsExternal() {
		if serverAddress == "" {
			return nil, fmt.Errorf("server address is required for external forward")
		}
		if targetNodeID == nil || *targetNodeID == 0 {
			return nil, fmt.Errorf("target node ID is required for external forward (protocol is derived from target node)")
		}
		// externalSource is optional
	}

	// Validate traffic multiplier
	if trafficMultiplier != nil {
		if *trafficMultiplier < 0 {
			return nil, fmt.Errorf("traffic multiplier cannot be negative: %f", *trafficMultiplier)
		}
		if *trafficMultiplier > 1000000 {
			return nil, fmt.Errorf("traffic multiplier exceeds maximum (1000000): %f", *trafficMultiplier)
		}
	}

	// Default ipVersion to auto if not set
	if ipVersion == "" {
		ipVersion = vo.IPVersionAuto
	}

	// Default loadBalanceStrategy to failover if not set
	if loadBalanceStrategy == "" {
		loadBalanceStrategy = vo.DefaultLoadBalanceStrategy
	}

	rule := &ForwardRule{
		id:                  id,
		sid:                 sid,
		agentID:             agentID,
		userID:              userID,
		subscriptionID:      subscriptionID,
		ruleType:            ruleType,
		exitAgentID:         exitAgentID,
		exitAgents:          exitAgents,
		loadBalanceStrategy: loadBalanceStrategy,
		chainAgentIDs:       chainAgentIDs,
		chainPortConfig:     chainPortConfig,
		tunnelHops:          tunnelHops,
		tunnelType:          tunnelType,
		name:                name,
		listenPort:          listenPort,
		targetAddress:       targetAddress,
		targetPort:          targetPort,
		targetNodeID:        targetNodeID,
		bindIP:              bindIP,
		ipVersion:           ipVersion,
		protocol:            protocol,
		status:              status,
		remark:              remark,
		uploadBytes:         uploadBytes,
		downloadBytes:       downloadBytes,
		trafficMultiplier:   trafficMultiplier,
		sortOrder:           sortOrder,
		groupIDs:            groupIDs,
		serverAddress:       serverAddress,
		externalSource:      externalSource,
		externalRuleID:      externalRuleID,
		createdAt:           createdAt,
		updatedAt:           updatedAt,
	}

	// Perform full validation to catch data corruption or manual DB modifications
	if err := rule.Validate(); err != nil {
		return nil, fmt.Errorf("reconstructed rule failed validation: %w", err)
	}

	return rule, nil
}

// validateAddress validates the target address format.
// It accepts valid IP addresses or RFC 1123 compliant domain names.
func validateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	// Check if it's a valid IP
	if ip := net.ParseIP(address); ip != nil {
		return nil
	}

	// Validate as domain name using RFC 1123 hostname rules
	// Same regex as used in forwardagent.go for consistency
	if len(address) > 253 {
		return fmt.Errorf("domain name too long (max 253 characters)")
	}

	// Use the existing domainNameRegex pattern
	// Pattern: ^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$
	if !domainNameRegex.MatchString(address) {
		return fmt.Errorf("invalid domain name format (must comply with RFC 1123)")
	}

	return nil
}

// --- Getters ---

// ID returns the forward rule ID.
func (r *ForwardRule) ID() uint {
	return r.id
}

// SID returns the Stripe-style prefixed ID (fr_xxx).
func (r *ForwardRule) SID() string {
	return r.sid
}

// AgentID returns the forward agent ID.
func (r *ForwardRule) AgentID() uint {
	return r.agentID
}

// UserID returns the user ID.
func (r *ForwardRule) UserID() *uint {
	return r.userID
}

// SubscriptionID returns the subscription ID.
func (r *ForwardRule) SubscriptionID() *uint {
	return r.subscriptionID
}

// IsSubscriptionBound returns true if the rule is bound to a subscription.
func (r *ForwardRule) IsSubscriptionBound() bool {
	return r.subscriptionID != nil && *r.subscriptionID != 0
}

// IsUserOwned returns true if the rule is owned by a user.
func (r *ForwardRule) IsUserOwned() bool {
	return r.userID != nil && *r.userID != 0
}

// Scope returns the ownership scope of this rule.
// System scope for admin-created rules, User scope for user-owned rules.
func (r *ForwardRule) Scope() vo.RuleScope {
	if r.IsUserOwned() {
		return vo.UserScope(*r.userID)
	}
	return vo.SystemScope()
}

// RuleType returns the rule type.
func (r *ForwardRule) RuleType() vo.ForwardRuleType {
	return r.ruleType
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

// BindIP returns the bind IP address for outbound connections.
func (r *ForwardRule) BindIP() string {
	return r.bindIP
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

// SortOrder returns the sort order.
func (r *ForwardRule) SortOrder() int {
	return r.sortOrder
}

// GroupIDs returns the resource group IDs.
func (r *ForwardRule) GroupIDs() []uint {
	return r.groupIDs
}

// ServerAddress returns the server address for external rules.
func (r *ForwardRule) ServerAddress() string {
	return r.serverAddress
}

// ExternalSource returns the external source identifier.
func (r *ForwardRule) ExternalSource() string {
	return r.externalSource
}

// ExternalRuleID returns the external rule ID reference.
func (r *ForwardRule) ExternalRuleID() string {
	return r.externalRuleID
}

// IsExternal returns true if this is an external forward rule.
func (r *ForwardRule) IsExternal() bool {
	return r.ruleType.IsExternal()
}

// IsEnabled checks if the rule is enabled.
func (r *ForwardRule) IsEnabled() bool {
	return r.status.IsEnabled()
}

// CreatedAt returns when the rule was created.
func (r *ForwardRule) CreatedAt() time.Time {
	return r.createdAt
}

// UpdatedAt returns when the rule was last updated.
func (r *ForwardRule) UpdatedAt() time.Time {
	return r.updatedAt
}
