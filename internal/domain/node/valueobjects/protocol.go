package valueobjects

// Protocol represents the supported proxy protocol types
type Protocol string

const (
	// ProtocolShadowsocks represents the Shadowsocks protocol
	ProtocolShadowsocks Protocol = "shadowsocks"
	// ProtocolTrojan represents the Trojan protocol
	ProtocolTrojan Protocol = "trojan"
)

var validProtocols = map[Protocol]bool{
	ProtocolShadowsocks: true,
	ProtocolTrojan:      true,
}

// String returns the string representation of the protocol
func (p Protocol) String() string {
	return string(p)
}

// IsValid checks if the protocol is valid
func (p Protocol) IsValid() bool {
	return validProtocols[p]
}

// IsShadowsocks checks if the protocol is Shadowsocks
func (p Protocol) IsShadowsocks() bool {
	return p == ProtocolShadowsocks
}

// IsTrojan checks if the protocol is Trojan
func (p Protocol) IsTrojan() bool {
	return p == ProtocolTrojan
}

// Equals checks if two protocols are equal
func (p Protocol) Equals(other Protocol) bool {
	return p == other
}
