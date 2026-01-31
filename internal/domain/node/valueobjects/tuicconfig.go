package valueobjects

import (
	"fmt"
	"net/url"
	"strings"
)

// TUIC-specific constants
// Congestion control constants (CongestionControlCubic, CongestionControlBBR, CongestionControlNewReno)
// are shared with Hysteria2 and defined in hysteria2config.go

const (
	// UDPRelayModeNative represents native UDP relay mode
	UDPRelayModeNative = "native"
	// UDPRelayModeQuic represents QUIC UDP relay mode
	UDPRelayModeQuic = "quic"
)

var validUDPRelayModes = map[string]bool{
	UDPRelayModeNative: true,
	UDPRelayModeQuic:   true,
}

// TUICConfig represents the TUIC protocol configuration
// This is an immutable value object following DDD principles
type TUICConfig struct {
	uuid              string
	password          string
	congestionControl string
	udpRelayMode      string
	alpn              string
	sni               string
	allowInsecure     bool
	disableSNI        bool
}

// NewTUICConfig creates a new TUICConfig with validation
func NewTUICConfig(
	uuid string,
	password string,
	congestionControl string,
	udpRelayMode string,
	alpn string,
	sni string,
	allowInsecure bool,
	disableSNI bool,
) (TUICConfig, error) {
	// Validate uuid
	if uuid == "" {
		return TUICConfig{}, fmt.Errorf("uuid is required")
	}

	// Validate password
	if password == "" {
		return TUICConfig{}, fmt.Errorf("password is required")
	}

	// Validate congestion control
	if congestionControl == "" {
		congestionControl = CongestionControlBBR // Default to BBR
	}
	if !isValidHysteria2CongestionControl(congestionControl) {
		return TUICConfig{}, fmt.Errorf("unsupported congestion control: %s (must be cubic, bbr, or new_reno)", congestionControl)
	}

	// Validate UDP relay mode
	if udpRelayMode == "" {
		udpRelayMode = UDPRelayModeNative // Default to native
	}
	if !isValidUDPRelayMode(udpRelayMode) {
		return TUICConfig{}, fmt.Errorf("unsupported UDP relay mode: %s (must be native or quic)", udpRelayMode)
	}

	return TUICConfig{
		uuid:              uuid,
		password:          password,
		congestionControl: congestionControl,
		udpRelayMode:      udpRelayMode,
		alpn:              alpn,
		sni:               sni,
		allowInsecure:     allowInsecure,
		disableSNI:        disableSNI,
	}, nil
}

// UUID returns the TUIC UUID
func (tc TUICConfig) UUID() string {
	return tc.uuid
}

// Password returns the TUIC password
func (tc TUICConfig) Password() string {
	return tc.password
}

// CongestionControl returns the congestion control algorithm
func (tc TUICConfig) CongestionControl() string {
	return tc.congestionControl
}

// UDPRelayMode returns the UDP relay mode
func (tc TUICConfig) UDPRelayMode() string {
	return tc.udpRelayMode
}

// ALPN returns the ALPN protocols
func (tc TUICConfig) ALPN() string {
	return tc.alpn
}

// SNI returns the Server Name Indication
func (tc TUICConfig) SNI() string {
	return tc.sni
}

// AllowInsecure returns whether to allow insecure connections
func (tc TUICConfig) AllowInsecure() bool {
	return tc.allowInsecure
}

// DisableSNI returns whether SNI is disabled
func (tc TUICConfig) DisableSNI() bool {
	return tc.disableSNI
}

// ToURI generates a TUIC URI string for subscription
// Format: tuic://uuid:password@host:port?congestion_control=bbr&udp_relay_mode=native&alpn=h3&sni=xxx#remarks
// If uuid or password is empty, it uses the values stored in config (for backward compatibility)
func (tc TUICConfig) ToURI(serverAddr string, serverPort uint16, remarks string, uuid string, password string) string {
	// Use provided values, fallback to config values if empty
	u := uuid
	if u == "" {
		u = tc.uuid
	}
	p := password
	if p == "" {
		p = tc.password
	}

	// Build base URI with uuid:password (URL encoded to handle special characters)
	uri := fmt.Sprintf("tuic://%s:%s@%s:%d", url.QueryEscape(u), url.QueryEscape(p), serverAddr, serverPort)

	// Build query parameters
	var params []string

	// Add congestion control
	if tc.congestionControl != "" {
		params = append(params, "congestion_control="+url.QueryEscape(tc.congestionControl))
	}

	// Add UDP relay mode
	if tc.udpRelayMode != "" {
		params = append(params, "udp_relay_mode="+url.QueryEscape(tc.udpRelayMode))
	}

	// Add ALPN if provided
	if tc.alpn != "" {
		params = append(params, "alpn="+url.QueryEscape(tc.alpn))
	}

	// Add SNI if provided
	if tc.sni != "" {
		params = append(params, "sni="+url.QueryEscape(tc.sni))
	}

	// Add allow_insecure parameter
	if tc.allowInsecure {
		params = append(params, "allow_insecure=1")
	}

	// Add disable_sni parameter
	if tc.disableSNI {
		params = append(params, "disable_sni=1")
	}

	// Append query parameters
	if len(params) > 0 {
		uri += "?" + strings.Join(params, "&")
	}

	// Add remarks if provided
	if remarks != "" {
		uri += "#" + url.QueryEscape(remarks)
	}

	return uri
}

// String returns a string representation of the config
func (tc TUICConfig) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("congestion_control=%s", tc.congestionControl))
	parts = append(parts, fmt.Sprintf("udp_relay_mode=%s", tc.udpRelayMode))

	if tc.alpn != "" {
		parts = append(parts, fmt.Sprintf("alpn=%s", tc.alpn))
	}

	if tc.sni != "" {
		parts = append(parts, fmt.Sprintf("sni=%s", tc.sni))
	}

	if tc.allowInsecure {
		parts = append(parts, "allow_insecure=true")
	}

	if tc.disableSNI {
		parts = append(parts, "disable_sni=true")
	}

	return strings.Join(parts, ", ")
}

// Equals checks if two TUICConfig instances are equal
func (tc TUICConfig) Equals(other TUICConfig) bool {
	return tc.uuid == other.uuid &&
		tc.password == other.password &&
		tc.congestionControl == other.congestionControl &&
		tc.udpRelayMode == other.udpRelayMode &&
		tc.alpn == other.alpn &&
		tc.sni == other.sni &&
		tc.allowInsecure == other.allowInsecure &&
		tc.disableSNI == other.disableSNI
}

// isValidUDPRelayMode validates the UDP relay mode
func isValidUDPRelayMode(mode string) bool {
	return validUDPRelayModes[mode]
}
