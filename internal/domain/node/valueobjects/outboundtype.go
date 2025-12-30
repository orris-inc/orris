package valueobjects

import (
	"fmt"
	"strings"
)

// OutboundType defines available outbound actions for routing rules.
// Compatible with sing-box outbound configuration.
// Supports:
//   - Preset types: "direct", "block", "proxy"
//   - Node reference: "node_xxx" (routes traffic through the specified node)
type OutboundType string

const (
	// OutboundDirect routes traffic directly without proxy
	OutboundDirect OutboundType = "direct"
	// OutboundBlock blocks/rejects the traffic
	OutboundBlock OutboundType = "block"
	// OutboundProxy routes traffic through the proxy (node's inbound becomes outbound)
	OutboundProxy OutboundType = "proxy"
)

// nodeSIDPrefix is the prefix for node SID references
const nodeSIDPrefix = "node_"

// IsPresetType checks if this is a built-in outbound type (direct/block/proxy)
func (o OutboundType) IsPresetType() bool {
	switch o {
	case OutboundDirect, OutboundBlock, OutboundProxy:
		return true
	default:
		return false
	}
}

// IsNodeReference checks if this outbound references another node (node_xxx format)
func (o OutboundType) IsNodeReference() bool {
	s := string(o)
	return strings.HasPrefix(s, nodeSIDPrefix) && len(s) > len(nodeSIDPrefix)
}

// IsValid checks if the outbound type is valid (either preset type or node reference)
func (o OutboundType) IsValid() bool {
	return o.IsPresetType() || o.IsNodeReference()
}

// NodeSID returns the node SID if this is a node reference, empty string otherwise
func (o OutboundType) NodeSID() string {
	if o.IsNodeReference() {
		return string(o)
	}
	return ""
}

// String returns the string representation
func (o OutboundType) String() string {
	return string(o)
}

// ParseOutboundType parses a string to OutboundType
// Accepts preset types (direct/block/proxy) and node SID references (node_xxx)
func ParseOutboundType(s string) (OutboundType, error) {
	o := OutboundType(s)
	if !o.IsValid() {
		return "", fmt.Errorf("invalid outbound type: %s (must be 'direct', 'block', 'proxy', or node SID like 'node_xxx')", s)
	}
	return o, nil
}

// NewNodeReferenceOutbound creates an OutboundType that references a specific node
func NewNodeReferenceOutbound(nodeSID string) (OutboundType, error) {
	if !strings.HasPrefix(nodeSID, nodeSIDPrefix) {
		return "", fmt.Errorf("invalid node SID format: %s (must start with 'node_')", nodeSID)
	}
	if len(nodeSID) <= len(nodeSIDPrefix) {
		return "", fmt.Errorf("invalid node SID format: %s (missing ID after prefix)", nodeSID)
	}
	return OutboundType(nodeSID), nil
}
