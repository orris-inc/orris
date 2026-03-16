package valueobjects

// AddressPreference represents which address to use when connecting to the next hop agent.
type AddressPreference string

const (
	// AddressPreferenceAuto uses the default behavior: tunnelAddress > publicAddress.
	AddressPreferenceAuto AddressPreference = "auto"
	// AddressPreferencePublic forces the use of publicAddress.
	AddressPreferencePublic AddressPreference = "public"
	// AddressPreferenceTunnel forces the use of tunnelAddress.
	AddressPreferenceTunnel AddressPreference = "tunnel"
)

// IsValid checks if the address preference is valid.
func (p AddressPreference) IsValid() bool {
	switch p {
	case AddressPreferenceAuto, AddressPreferencePublic, AddressPreferenceTunnel:
		return true
	default:
		return false
	}
}

// String returns the string representation.
func (p AddressPreference) String() string {
	return string(p)
}

// IsAuto returns true if the preference is auto.
func (p AddressPreference) IsAuto() bool {
	return p == AddressPreferenceAuto
}
