// Package valueobjects provides value objects for the forward domain.
package valueobjects

import "sort"

// BlockedProtocol represents a protocol that can be blocked.
type BlockedProtocol string

const (
	// Proxy protocols.

	// BlockedProtocolHTTPConnect represents HTTP CONNECT proxy protocol.
	BlockedProtocolHTTPConnect BlockedProtocol = "http_connect"
	// BlockedProtocolSOCKS4 represents SOCKS4 proxy protocol.
	BlockedProtocolSOCKS4 BlockedProtocol = "socks4"
	// BlockedProtocolSOCKS5 represents SOCKS5 proxy protocol.
	BlockedProtocolSOCKS5 BlockedProtocol = "socks5"

	// Application layer protocols.

	// BlockedProtocolHTTP represents HTTP application protocol.
	BlockedProtocolHTTP BlockedProtocol = "http"
	// BlockedProtocolTLS represents TLS application protocol.
	BlockedProtocolTLS BlockedProtocol = "tls"
	// BlockedProtocolSSH represents SSH application protocol.
	BlockedProtocolSSH BlockedProtocol = "ssh"
	// BlockedProtocolFTP represents FTP application protocol.
	BlockedProtocolFTP BlockedProtocol = "ftp"
)

var validBlockedProtocols = map[BlockedProtocol]bool{
	BlockedProtocolHTTPConnect: true,
	BlockedProtocolSOCKS4:      true,
	BlockedProtocolSOCKS5:      true,
	BlockedProtocolHTTP:        true,
	BlockedProtocolTLS:         true,
	BlockedProtocolSSH:         true,
	BlockedProtocolFTP:         true,
}

// String returns the string representation.
func (p BlockedProtocol) String() string {
	return string(p)
}

// IsValid checks if the blocked protocol is valid.
func (p BlockedProtocol) IsValid() bool {
	return validBlockedProtocols[p]
}

// BlockedProtocols represents a list of blocked protocols.
type BlockedProtocols []BlockedProtocol

// NewBlockedProtocols creates a new BlockedProtocols from a string slice.
// It validates each protocol, removes duplicates, and sorts the result.
// Invalid protocols are silently ignored.
func NewBlockedProtocols(protocols []string) BlockedProtocols {
	if len(protocols) == 0 {
		return nil
	}

	// Use a map for deduplication.
	seen := make(map[BlockedProtocol]bool)
	result := make(BlockedProtocols, 0, len(protocols))

	for _, p := range protocols {
		bp := BlockedProtocol(p)
		if bp.IsValid() && !seen[bp] {
			seen[bp] = true
			result = append(result, bp)
		}
	}

	if len(result) == 0 {
		return nil
	}

	// Sort for consistent ordering.
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result
}

// Contains checks if the given protocol is in the blocked list.
func (bp BlockedProtocols) Contains(protocol BlockedProtocol) bool {
	for _, p := range bp {
		if p == protocol {
			return true
		}
	}
	return false
}

// ToStringSlice converts the blocked protocols to a string slice.
func (bp BlockedProtocols) ToStringSlice() []string {
	if len(bp) == 0 {
		return nil
	}

	result := make([]string, len(bp))
	for i, p := range bp {
		result[i] = p.String()
	}
	return result
}

// ValidateBlockedProtocols checks if all protocols in the input are valid.
// Returns a list of invalid protocol names if any are found.
func ValidateBlockedProtocols(protocols []string) []string {
	if len(protocols) == 0 {
		return nil
	}

	var invalid []string
	for _, p := range protocols {
		bp := BlockedProtocol(p)
		if !bp.IsValid() {
			invalid = append(invalid, p)
		}
	}
	return invalid
}

// ValidBlockedProtocolNames returns a list of all valid blocked protocol names.
func ValidBlockedProtocolNames() []string {
	return []string{
		string(BlockedProtocolHTTPConnect),
		string(BlockedProtocolSOCKS4),
		string(BlockedProtocolSOCKS5),
		string(BlockedProtocolHTTP),
		string(BlockedProtocolTLS),
		string(BlockedProtocolSSH),
		string(BlockedProtocolFTP),
	}
}
