package valueobjects

// Protocol represents the supported proxy protocol types
type Protocol string

const (
	// ProtocolShadowsocks represents the Shadowsocks protocol
	ProtocolShadowsocks Protocol = "shadowsocks"
	// ProtocolTrojan represents the Trojan protocol
	ProtocolTrojan Protocol = "trojan"
	// ProtocolVLESS represents the VLESS protocol
	ProtocolVLESS Protocol = "vless"
	// ProtocolVMess represents the VMess protocol
	ProtocolVMess Protocol = "vmess"
	// ProtocolHysteria2 represents the Hysteria2 protocol
	ProtocolHysteria2 Protocol = "hysteria2"
	// ProtocolTUIC represents the TUIC protocol
	ProtocolTUIC Protocol = "tuic"
	// ProtocolAnyTLS represents the AnyTLS protocol
	ProtocolAnyTLS Protocol = "anytls"
)

var validProtocols = map[Protocol]bool{
	ProtocolShadowsocks: true,
	ProtocolTrojan:      true,
	ProtocolVLESS:       true,
	ProtocolVMess:       true,
	ProtocolHysteria2:   true,
	ProtocolTUIC:        true,
	ProtocolAnyTLS:      true,
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

// IsVLESS checks if the protocol is VLESS
func (p Protocol) IsVLESS() bool {
	return p == ProtocolVLESS
}

// IsVMess checks if the protocol is VMess
func (p Protocol) IsVMess() bool {
	return p == ProtocolVMess
}

// IsHysteria2 checks if the protocol is Hysteria2
func (p Protocol) IsHysteria2() bool {
	return p == ProtocolHysteria2
}

// IsTUIC checks if the protocol is TUIC
func (p Protocol) IsTUIC() bool {
	return p == ProtocolTUIC
}

// IsAnyTLS checks if the protocol is AnyTLS
func (p Protocol) IsAnyTLS() bool {
	return p == ProtocolAnyTLS
}
