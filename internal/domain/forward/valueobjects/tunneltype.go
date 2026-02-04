// Package valueobjects provides value objects for the forward domain.
package valueobjects

// TunnelType represents the type of tunnel for forwarding traffic.
type TunnelType string

const (
	// TunnelTypeWS represents WebSocket tunnel.
	TunnelTypeWS TunnelType = "ws"
	// TunnelTypeTLS represents TLS tunnel.
	TunnelTypeTLS TunnelType = "tls"
	// TunnelTypeWSSmux represents WebSocket tunnel with SMUX multiplexing.
	TunnelTypeWSSmux TunnelType = "ws_smux"
	// TunnelTypeTLSSmux represents TLS tunnel with SMUX multiplexing.
	TunnelTypeTLSSmux TunnelType = "tls_smux"
)

var validTunnelTypes = map[TunnelType]bool{
	TunnelTypeWS:      true,
	TunnelTypeTLS:     true,
	TunnelTypeWSSmux:  true,
	TunnelTypeTLSSmux: true,
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

// IsWS checks if this is a WebSocket tunnel (with or without SMUX).
func (t TunnelType) IsWS() bool {
	return t == TunnelTypeWS || t == TunnelTypeWSSmux || t == ""
}

// IsTLS checks if this is a TLS tunnel (with or without SMUX).
func (t TunnelType) IsTLS() bool {
	return t == TunnelTypeTLS || t == TunnelTypeTLSSmux
}

// IsWSSmux checks if this is a WebSocket tunnel with SMUX.
func (t TunnelType) IsWSSmux() bool {
	return t == TunnelTypeWSSmux
}

// IsTLSSmux checks if this is a TLS tunnel with SMUX.
func (t TunnelType) IsTLSSmux() bool {
	return t == TunnelTypeTLSSmux
}

// IsSmux checks if this tunnel uses SMUX multiplexing.
func (t TunnelType) IsSmux() bool {
	return t == TunnelTypeWSSmux || t == TunnelTypeTLSSmux
}
