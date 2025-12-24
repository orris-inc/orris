// Package valueobjects provides value objects for the forward domain.
package valueobjects

// TunnelType represents the type of tunnel for forwarding traffic.
type TunnelType string

const (
	// TunnelTypeWS represents WebSocket tunnel.
	TunnelTypeWS TunnelType = "ws"
	// TunnelTypeTLS represents TLS tunnel.
	TunnelTypeTLS TunnelType = "tls"
)

var validTunnelTypes = map[TunnelType]bool{
	TunnelTypeWS:  true,
	TunnelTypeTLS: true,
}

// String returns the string representation.
// Returns "ws" for empty string (default value).
func (t TunnelType) String() string {
	if t == "" {
		return string(TunnelTypeWS)
	}
	return string(t)
}

// IsValid checks if the tunnel type is valid.
// Empty string is considered valid (defaults to WS).
func (t TunnelType) IsValid() bool {
	if t == "" {
		return true
	}
	return validTunnelTypes[t]
}

// IsWS checks if this is a WebSocket tunnel.
func (t TunnelType) IsWS() bool {
	return t == TunnelTypeWS || t == ""
}

// IsTLS checks if this is a TLS tunnel.
func (t TunnelType) IsTLS() bool {
	return t == TunnelTypeTLS
}
