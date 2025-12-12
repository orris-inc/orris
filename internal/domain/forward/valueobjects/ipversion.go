package valueobjects

// IPVersion represents the IP version preference for target address resolution.
type IPVersion string

const (
	// IPVersionAuto automatically selects the best available IP version (IPv4 preferred).
	IPVersionAuto IPVersion = "auto"
	// IPVersionIPv4 forces the use of IPv4 address.
	IPVersionIPv4 IPVersion = "ipv4"
	// IPVersionIPv6 forces the use of IPv6 address.
	IPVersionIPv6 IPVersion = "ipv6"
)

// IsValid checks if the IP version is valid.
func (v IPVersion) IsValid() bool {
	switch v {
	case IPVersionAuto, IPVersionIPv4, IPVersionIPv6:
		return true
	default:
		return false
	}
}

// String returns the string representation.
func (v IPVersion) String() string {
	return string(v)
}

// IsAuto returns true if the version is auto.
func (v IPVersion) IsAuto() bool {
	return v == IPVersionAuto
}

// IsIPv4 returns true if the version is IPv4.
func (v IPVersion) IsIPv4() bool {
	return v == IPVersionIPv4
}

// IsIPv6 returns true if the version is IPv6.
func (v IPVersion) IsIPv6() bool {
	return v == IPVersionIPv6
}
