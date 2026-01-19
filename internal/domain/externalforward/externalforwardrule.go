// Package externalforward provides domain models for third-party forward rules.
package externalforward

import (
	"fmt"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/externalforward/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
)

// ExternalForwardRule represents the external forward rule aggregate root.
type ExternalForwardRule struct {
	id             uint
	sid            string // Stripe-style prefixed ID (efr_xxx)
	subscriptionID *uint  // subscription ID (nil for admin-created rules distributed via resource groups)
	userID         *uint  // user ID (nil for admin-created rules distributed via resource groups)
	nodeID         *uint  // optional node ID for direct routing

	// Rule metadata
	name          string
	serverAddress string
	listenPort    uint16

	// External reference
	externalSource string
	externalRuleID string

	// Status
	status    vo.Status
	sortOrder int
	remark    string

	// Resource group association (for subscription distribution)
	groupIDs []uint

	createdAt time.Time
	updatedAt time.Time
}

// NewExternalForwardRule creates a new external forward rule aggregate.
// For user-created rules, subscriptionID and userID are required.
// For admin-created rules distributed via resource groups, they can be nil.
func NewExternalForwardRule(
	subscriptionID *uint,
	userID *uint,
	nodeID *uint,
	name string,
	serverAddress string,
	listenPort uint16,
	externalSource string,
	externalRuleID string,
	remark string,
	sortOrder int,
	sidGenerator func() (string, error),
) (*ExternalForwardRule, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if serverAddress == "" {
		return nil, fmt.Errorf("server address is required")
	}
	if listenPort == 0 {
		return nil, fmt.Errorf("listen port is required")
	}
	if externalSource == "" {
		return nil, fmt.Errorf("external source is required")
	}

	sid, err := sidGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := biztime.NowUTC()
	return &ExternalForwardRule{
		sid:            sid,
		subscriptionID: subscriptionID,
		userID:         userID,
		nodeID:         nodeID,
		name:           name,
		serverAddress:  serverAddress,
		listenPort:     listenPort,
		externalSource: externalSource,
		externalRuleID: externalRuleID,
		status:         vo.StatusEnabled,
		sortOrder:      sortOrder,
		remark:         remark,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// NewExternalForwardRuleWithGroups creates a new external forward rule with resource group IDs.
// This is used for admin-created rules that are distributed via resource groups.
func NewExternalForwardRuleWithGroups(
	subscriptionID *uint,
	userID *uint,
	nodeID *uint,
	name string,
	serverAddress string,
	listenPort uint16,
	externalSource string,
	externalRuleID string,
	remark string,
	sortOrder int,
	groupIDs []uint,
	sidGenerator func() (string, error),
) (*ExternalForwardRule, error) {
	rule, err := NewExternalForwardRule(
		subscriptionID,
		userID,
		nodeID,
		name,
		serverAddress,
		listenPort,
		externalSource,
		externalRuleID,
		remark,
		sortOrder,
		sidGenerator,
	)
	if err != nil {
		return nil, err
	}
	// Set groupIDs directly without updating updatedAt
	rule.groupIDs = groupIDs
	return rule, nil
}

// ReconstructExternalForwardRule reconstructs an external forward rule from persistence.
func ReconstructExternalForwardRule(
	id uint,
	sid string,
	subscriptionID *uint,
	userID *uint,
	nodeID *uint,
	name string,
	serverAddress string,
	listenPort uint16,
	externalSource string,
	externalRuleID string,
	status vo.Status,
	sortOrder int,
	remark string,
	groupIDs []uint,
	createdAt, updatedAt time.Time,
) (*ExternalForwardRule, error) {
	if id == 0 {
		return nil, fmt.Errorf("external forward rule ID cannot be zero")
	}
	if sid == "" {
		return nil, fmt.Errorf("external forward rule SID is required")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if serverAddress == "" {
		return nil, fmt.Errorf("server address is required")
	}
	if listenPort == 0 {
		return nil, fmt.Errorf("listen port is required")
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid status: %s", status)
	}
	if externalSource == "" {
		return nil, fmt.Errorf("external source is required")
	}

	return &ExternalForwardRule{
		id:             id,
		sid:            sid,
		subscriptionID: subscriptionID,
		userID:         userID,
		nodeID:         nodeID,
		name:           name,
		serverAddress:  serverAddress,
		listenPort:     listenPort,
		externalSource: externalSource,
		externalRuleID: externalRuleID,
		status:         status,
		sortOrder:      sortOrder,
		remark:         remark,
		groupIDs:       groupIDs,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}, nil
}

// Getters

// ID returns the internal ID.
func (r *ExternalForwardRule) ID() uint {
	return r.id
}

// SID returns the Stripe-style prefixed ID (efr_xxx).
func (r *ExternalForwardRule) SID() string {
	return r.sid
}

// SubscriptionID returns the subscription ID (may be nil for admin-created rules).
func (r *ExternalForwardRule) SubscriptionID() *uint {
	return r.subscriptionID
}

// UserID returns the user ID (may be nil for admin-created rules).
func (r *ExternalForwardRule) UserID() *uint {
	return r.userID
}

// NodeID returns the node ID (may be nil if no specific node is assigned).
func (r *ExternalForwardRule) NodeID() *uint {
	return r.nodeID
}

// GroupIDs returns the resource group IDs.
func (r *ExternalForwardRule) GroupIDs() []uint {
	return r.groupIDs
}

// SetGroupIDs sets the resource group IDs.
func (r *ExternalForwardRule) SetGroupIDs(groupIDs []uint) {
	r.groupIDs = groupIDs
	r.updatedAt = biztime.NowUTC()
}

// AddGroupID adds a resource group ID if not already present.
func (r *ExternalForwardRule) AddGroupID(groupID uint) bool {
	for _, id := range r.groupIDs {
		if id == groupID {
			return false // already exists
		}
	}
	r.groupIDs = append(r.groupIDs, groupID)
	r.updatedAt = biztime.NowUTC()
	return true
}

// RemoveGroupID removes a resource group ID if present.
func (r *ExternalForwardRule) RemoveGroupID(groupID uint) bool {
	for i, id := range r.groupIDs {
		if id == groupID {
			r.groupIDs = append(r.groupIDs[:i], r.groupIDs[i+1:]...)
			r.updatedAt = biztime.NowUTC()
			return true
		}
	}
	return false
}

// HasGroupID checks if the rule belongs to a specific resource group.
func (r *ExternalForwardRule) HasGroupID(groupID uint) bool {
	for _, id := range r.groupIDs {
		if id == groupID {
			return true
		}
	}
	return false
}

// Name returns the rule name.
func (r *ExternalForwardRule) Name() string {
	return r.name
}

// ServerAddress returns the server address.
func (r *ExternalForwardRule) ServerAddress() string {
	return r.serverAddress
}

// ListenPort returns the listen port.
func (r *ExternalForwardRule) ListenPort() uint16 {
	return r.listenPort
}

// ExternalSource returns the external source identifier.
func (r *ExternalForwardRule) ExternalSource() string {
	return r.externalSource
}

// ExternalRuleID returns the external rule ID.
func (r *ExternalForwardRule) ExternalRuleID() string {
	return r.externalRuleID
}

// Status returns the status.
func (r *ExternalForwardRule) Status() vo.Status {
	return r.status
}

// SortOrder returns the sort order.
func (r *ExternalForwardRule) SortOrder() int {
	return r.sortOrder
}

// Remark returns the remark.
func (r *ExternalForwardRule) Remark() string {
	return r.remark
}

// CreatedAt returns when the rule was created.
func (r *ExternalForwardRule) CreatedAt() time.Time {
	return r.createdAt
}

// UpdatedAt returns when the rule was last updated.
func (r *ExternalForwardRule) UpdatedAt() time.Time {
	return r.updatedAt
}

// IsEnabled returns true if the rule is enabled.
func (r *ExternalForwardRule) IsEnabled() bool {
	return r.status.IsEnabled()
}

// Setters and business operations

// SetID sets the internal ID (only for persistence layer use).
func (r *ExternalForwardRule) SetID(id uint) error {
	if r.id != 0 {
		return fmt.Errorf("external forward rule ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("external forward rule ID cannot be zero")
	}
	r.id = id
	return nil
}

// Enable enables the rule.
func (r *ExternalForwardRule) Enable() {
	if !r.status.IsEnabled() {
		r.status = vo.StatusEnabled
		r.updatedAt = biztime.NowUTC()
	}
}

// Disable disables the rule.
func (r *ExternalForwardRule) Disable() {
	if r.status.IsEnabled() {
		r.status = vo.StatusDisabled
		r.updatedAt = biztime.NowUTC()
	}
}

// UpdateName updates the rule name.
func (r *ExternalForwardRule) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if r.name == name {
		return nil
	}
	r.name = name
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateServerAddress updates the server address.
func (r *ExternalForwardRule) UpdateServerAddress(serverAddress string) error {
	if serverAddress == "" {
		return fmt.Errorf("server address cannot be empty")
	}
	if r.serverAddress == serverAddress {
		return nil
	}
	r.serverAddress = serverAddress
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateListenPort updates the listen port.
func (r *ExternalForwardRule) UpdateListenPort(listenPort uint16) error {
	if listenPort == 0 {
		return fmt.Errorf("listen port cannot be zero")
	}
	if r.listenPort == listenPort {
		return nil
	}
	r.listenPort = listenPort
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateSortOrder updates the sort order.
func (r *ExternalForwardRule) UpdateSortOrder(order int) error {
	if order < 0 {
		return fmt.Errorf("sort order must be non-negative, got %d", order)
	}
	if r.sortOrder == order {
		return nil
	}
	r.sortOrder = order
	r.updatedAt = biztime.NowUTC()
	return nil
}

// UpdateRemark updates the remark.
func (r *ExternalForwardRule) UpdateRemark(remark string) {
	if r.remark == remark {
		return
	}
	r.remark = remark
	r.updatedAt = biztime.NowUTC()
}

// UpdateNodeID updates the node ID.
func (r *ExternalForwardRule) UpdateNodeID(nodeID *uint) {
	// Compare pointers and values
	if r.nodeID == nodeID {
		return
	}
	if r.nodeID != nil && nodeID != nil && *r.nodeID == *nodeID {
		return
	}
	r.nodeID = nodeID
	r.updatedAt = biztime.NowUTC()
}
