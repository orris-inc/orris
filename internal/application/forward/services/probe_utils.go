package services

import (
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/node"
)

// probeError represents a probe error.
type probeError struct {
	message string
}

func (e *probeError) Error() string {
	return e.message
}

// resolveNodeAddress selects the appropriate node address based on IP version preference.
// ipVersion: "auto", "ipv4", or "ipv6"
func (s *ProbeService) resolveNodeAddress(n *node.Node, ipVersion vo.IPVersion) string {
	serverAddr := n.ServerAddress().Value()
	ipv4 := ""
	ipv6 := ""

	if n.PublicIPv4() != nil {
		ipv4 = *n.PublicIPv4()
	}
	if n.PublicIPv6() != nil {
		ipv6 = *n.PublicIPv6()
	}

	// Check if server_address is a valid usable address
	isValidServerAddr := serverAddr != "" && serverAddr != "0.0.0.0" && serverAddr != "::"

	switch ipVersion {
	case vo.IPVersionIPv6:
		// Prefer IPv6: ipv6 > server_address > ipv4
		if ipv6 != "" {
			return ipv6
		}
		if isValidServerAddr {
			return serverAddr
		}
		if ipv4 != "" {
			return ipv4
		}

	case vo.IPVersionIPv4:
		// Prefer IPv4: ipv4 > server_address > ipv6
		if ipv4 != "" {
			return ipv4
		}
		if isValidServerAddr {
			return serverAddr
		}
		if ipv6 != "" {
			return ipv6
		}

	default: // "auto" or unknown
		// Default priority: server_address > ipv4 > ipv6
		if isValidServerAddr {
			return serverAddr
		}
		if ipv4 != "" {
			return ipv4
		}
		if ipv6 != "" {
			return ipv6
		}
	}

	return serverAddr
}
