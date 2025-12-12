// Package value_objects provides value objects for the forward domain.
package valueobjects

// ForwardProtocol represents the network protocol for forwarding.
type ForwardProtocol string

const (
	ForwardProtocolTCP  ForwardProtocol = "tcp"
	ForwardProtocolUDP  ForwardProtocol = "udp"
	ForwardProtocolBoth ForwardProtocol = "both"
)

var validForwardProtocols = map[ForwardProtocol]bool{
	ForwardProtocolTCP:  true,
	ForwardProtocolUDP:  true,
	ForwardProtocolBoth: true,
}

// String returns the string representation.
func (p ForwardProtocol) String() string {
	return string(p)
}

// IsValid checks if the protocol is valid.
func (p ForwardProtocol) IsValid() bool {
	return validForwardProtocols[p]
}

// IsTCP checks if the protocol is TCP.
func (p ForwardProtocol) IsTCP() bool {
	return p == ForwardProtocolTCP || p == ForwardProtocolBoth
}

// IsUDP checks if the protocol is UDP.
func (p ForwardProtocol) IsUDP() bool {
	return p == ForwardProtocolUDP || p == ForwardProtocolBoth
}

// IsBoth checks if the protocol forwards both TCP and UDP.
func (p ForwardProtocol) IsBoth() bool {
	return p == ForwardProtocolBoth
}
