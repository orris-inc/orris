// Package forward provides domain models and business logic for forward chain management.
package forward

import (
	"fmt"
	"time"

	vo "orris/internal/domain/forward/value_objects"
)

// ChainNode represents a node in the forward chain.
type ChainNode struct {
	AgentID    uint
	ListenPort uint16
	Sequence   int // order in the chain, starting from 1
}

// ForwardChain represents the forward chain aggregate root.
// A chain defines a multi-hop forwarding path: Agent A -> Agent B -> ... -> Final Target
type ForwardChain struct {
	id            uint
	name          string
	protocol      vo.ForwardProtocol
	status        vo.ForwardStatus
	nodes         []ChainNode // ordered list of nodes
	targetAddress string      // final target address
	targetPort    uint16      // final target port
	remark        string
	createdAt     time.Time
	updatedAt     time.Time
}

// NewForwardChain creates a new forward chain aggregate.
func NewForwardChain(
	name string,
	protocol vo.ForwardProtocol,
	nodes []ChainNode,
	targetAddress string,
	targetPort uint16,
	remark string,
) (*ForwardChain, error) {
	if name == "" {
		return nil, fmt.Errorf("chain name is required")
	}
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", protocol)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("at least one node is required")
	}
	if targetAddress == "" {
		return nil, fmt.Errorf("target address is required")
	}
	if targetPort == 0 {
		return nil, fmt.Errorf("target port is required")
	}

	// Validate nodes
	for i, node := range nodes {
		if node.AgentID == 0 {
			return nil, fmt.Errorf("node %d: agent_id is required", i+1)
		}
		if node.ListenPort == 0 {
			return nil, fmt.Errorf("node %d: listen_port is required", i+1)
		}
		nodes[i].Sequence = i + 1
	}

	// Validate target address
	if err := validateAddress(targetAddress); err != nil {
		return nil, fmt.Errorf("invalid target address: %w", err)
	}

	now := time.Now()
	return &ForwardChain{
		name:          name,
		protocol:      protocol,
		status:        vo.ForwardStatusDisabled,
		nodes:         nodes,
		targetAddress: targetAddress,
		targetPort:    targetPort,
		remark:        remark,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// ReconstructForwardChain reconstructs a forward chain from persistence.
func ReconstructForwardChain(
	id uint,
	name string,
	protocol vo.ForwardProtocol,
	status vo.ForwardStatus,
	nodes []ChainNode,
	targetAddress string,
	targetPort uint16,
	remark string,
	createdAt, updatedAt time.Time,
) (*ForwardChain, error) {
	if id == 0 {
		return nil, fmt.Errorf("chain ID cannot be zero")
	}
	if name == "" {
		return nil, fmt.Errorf("chain name is required")
	}
	if !protocol.IsValid() {
		return nil, fmt.Errorf("invalid protocol: %s", protocol)
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	return &ForwardChain{
		id:            id,
		name:          name,
		protocol:      protocol,
		status:        status,
		nodes:         nodes,
		targetAddress: targetAddress,
		targetPort:    targetPort,
		remark:        remark,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}, nil
}

// Getters

// ID returns the chain ID.
func (c *ForwardChain) ID() uint {
	return c.id
}

// Name returns the chain name.
func (c *ForwardChain) Name() string {
	return c.name
}

// Protocol returns the protocol.
func (c *ForwardChain) Protocol() vo.ForwardProtocol {
	return c.protocol
}

// Status returns the status.
func (c *ForwardChain) Status() vo.ForwardStatus {
	return c.status
}

// Nodes returns the chain nodes.
func (c *ForwardChain) Nodes() []ChainNode {
	return c.nodes
}

// NodeCount returns the number of nodes in the chain.
func (c *ForwardChain) NodeCount() int {
	return len(c.nodes)
}

// TargetAddress returns the final target address.
func (c *ForwardChain) TargetAddress() string {
	return c.targetAddress
}

// TargetPort returns the final target port.
func (c *ForwardChain) TargetPort() uint16 {
	return c.targetPort
}

// Remark returns the remark.
func (c *ForwardChain) Remark() string {
	return c.remark
}

// CreatedAt returns when the chain was created.
func (c *ForwardChain) CreatedAt() time.Time {
	return c.createdAt
}

// UpdatedAt returns when the chain was last updated.
func (c *ForwardChain) UpdatedAt() time.Time {
	return c.updatedAt
}

// Setters and business operations

// SetID sets the chain ID (only for persistence layer use).
func (c *ForwardChain) SetID(id uint) error {
	if c.id != 0 {
		return fmt.Errorf("chain ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("chain ID cannot be zero")
	}
	c.id = id
	return nil
}

// Enable enables the forward chain.
func (c *ForwardChain) Enable() {
	if c.status.IsEnabled() {
		return
	}
	c.status = vo.ForwardStatusEnabled
	c.updatedAt = time.Now()
}

// Disable disables the forward chain.
func (c *ForwardChain) Disable() {
	if c.status.IsDisabled() {
		return
	}
	c.status = vo.ForwardStatusDisabled
	c.updatedAt = time.Now()
}

// UpdateName updates the chain name.
func (c *ForwardChain) UpdateName(name string) error {
	if name == "" {
		return fmt.Errorf("chain name cannot be empty")
	}
	if c.name == name {
		return nil
	}
	c.name = name
	c.updatedAt = time.Now()
	return nil
}

// UpdateRemark updates the remark.
func (c *ForwardChain) UpdateRemark(remark string) {
	if c.remark == remark {
		return
	}
	c.remark = remark
	c.updatedAt = time.Now()
}

// IsEnabled checks if the chain is enabled.
func (c *ForwardChain) IsEnabled() bool {
	return c.status.IsEnabled()
}

// GenerateRules generates ForwardRule entities for each node in the chain.
// This method creates the actual forwarding rules that will be deployed to agents.
func (c *ForwardChain) GenerateRules() ([]*ForwardRule, error) {
	if len(c.nodes) == 0 {
		return nil, fmt.Errorf("chain has no nodes")
	}

	rules := make([]*ForwardRule, len(c.nodes))

	for i, node := range c.nodes {
		var nextAgentID uint
		var targetAddr string
		var targetPort uint16

		if i == len(c.nodes)-1 {
			// Last node: forward to final target
			nextAgentID = 0
			targetAddr = c.targetAddress
			targetPort = c.targetPort
		} else {
			// Intermediate node: forward to next agent
			nextAgentID = c.nodes[i+1].AgentID
			targetAddr = ""
			targetPort = 0
		}

		rule, err := NewForwardRule(
			node.AgentID,
			nextAgentID,
			fmt.Sprintf("%s-node-%d", c.name, node.Sequence),
			node.ListenPort,
			targetAddr,
			targetPort,
			c.protocol,
			fmt.Sprintf("Auto-generated for chain: %s", c.name),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create rule for node %d: %w", node.Sequence, err)
		}

		rules[i] = rule
	}

	return rules, nil
}

// Validate performs domain-level validation.
func (c *ForwardChain) Validate() error {
	if c.name == "" {
		return fmt.Errorf("chain name is required")
	}
	if !c.protocol.IsValid() {
		return fmt.Errorf("invalid protocol: %s", c.protocol)
	}
	if !c.status.IsValid() {
		return fmt.Errorf("invalid status: %s", c.status)
	}
	if len(c.nodes) == 0 {
		return fmt.Errorf("at least one node is required")
	}
	if c.targetAddress == "" {
		return fmt.Errorf("target address is required")
	}
	if c.targetPort == 0 {
		return fmt.Errorf("target port is required")
	}
	return nil
}
