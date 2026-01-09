package valueobjects

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	// CongestionControlCubic represents Cubic congestion control algorithm
	CongestionControlCubic = "cubic"
	// CongestionControlBBR represents BBR congestion control algorithm
	CongestionControlBBR = "bbr"
	// CongestionControlNewReno represents NewReno congestion control algorithm
	CongestionControlNewReno = "new_reno"

	// ObfsSalamander represents Salamander obfuscation type
	ObfsSalamander = "salamander"
)

// validHysteria2CongestionControls defines valid congestion control algorithms for Hysteria2
var validHysteria2CongestionControls = map[string]bool{
	CongestionControlCubic:   true,
	CongestionControlBBR:     true,
	CongestionControlNewReno: true,
}

// Hysteria2Config represents the Hysteria2 protocol configuration
// This is an immutable value object following DDD principles
type Hysteria2Config struct {
	password          string
	congestionControl string
	obfs              string
	obfsPassword      string
	upMbps            *int
	downMbps          *int
	sni               string
	allowInsecure     bool
	fingerprint       string
}

// NewHysteria2Config creates a new Hysteria2Config with validation
func NewHysteria2Config(
	password string,
	congestionControl string,
	obfs string,
	obfsPassword string,
	upMbps *int,
	downMbps *int,
	sni string,
	allowInsecure bool,
	fingerprint string,
) (Hysteria2Config, error) {
	// Validate password
	if len(password) < 8 {
		return Hysteria2Config{}, fmt.Errorf("password must be at least 8 characters long")
	}

	// Validate congestion control
	if congestionControl == "" {
		congestionControl = CongestionControlBBR
	}
	if !isValidHysteria2CongestionControl(congestionControl) {
		return Hysteria2Config{}, fmt.Errorf("unsupported congestion control: %s (must be cubic, bbr, or new_reno)", congestionControl)
	}

	// Validate obfs and obfs_password
	if obfs != "" && obfs != ObfsSalamander {
		return Hysteria2Config{}, fmt.Errorf("unsupported obfs type: %s (must be salamander or empty)", obfs)
	}
	if obfs == ObfsSalamander && obfsPassword == "" {
		return Hysteria2Config{}, fmt.Errorf("obfs_password is required when obfs is salamander")
	}

	// Validate bandwidth limits
	if upMbps != nil && *upMbps < 0 {
		return Hysteria2Config{}, fmt.Errorf("up_mbps must be non-negative")
	}
	if downMbps != nil && *downMbps < 0 {
		return Hysteria2Config{}, fmt.Errorf("down_mbps must be non-negative")
	}

	return Hysteria2Config{
		password:          password,
		congestionControl: congestionControl,
		obfs:              obfs,
		obfsPassword:      obfsPassword,
		upMbps:            upMbps,
		downMbps:          downMbps,
		sni:               sni,
		allowInsecure:     allowInsecure,
		fingerprint:       fingerprint,
	}, nil
}

// Password returns the Hysteria2 password
func (hc Hysteria2Config) Password() string {
	return hc.password
}

// CongestionControl returns the congestion control algorithm
func (hc Hysteria2Config) CongestionControl() string {
	return hc.congestionControl
}

// Obfs returns the obfuscation type
func (hc Hysteria2Config) Obfs() string {
	return hc.obfs
}

// ObfsPassword returns the obfuscation password
func (hc Hysteria2Config) ObfsPassword() string {
	return hc.obfsPassword
}

// UpMbps returns the upstream bandwidth limit in Mbps
func (hc Hysteria2Config) UpMbps() *int {
	return hc.upMbps
}

// DownMbps returns the downstream bandwidth limit in Mbps
func (hc Hysteria2Config) DownMbps() *int {
	return hc.downMbps
}

// SNI returns the Server Name Indication
func (hc Hysteria2Config) SNI() string {
	return hc.sni
}

// AllowInsecure returns whether to allow insecure connections
func (hc Hysteria2Config) AllowInsecure() bool {
	return hc.allowInsecure
}

// Fingerprint returns the TLS fingerprint
func (hc Hysteria2Config) Fingerprint() string {
	return hc.fingerprint
}

// ToURI generates a Hysteria2 URI string for subscription
// Format: hysteria2://password@host:port?sni=xxx&obfs=salamander&obfs-password=xxx&up=100&down=100#remarks
func (hc Hysteria2Config) ToURI(serverAddr string, serverPort uint16, remarks string) string {
	// Build base URI
	uri := fmt.Sprintf("hysteria2://%s@%s:%d", url.QueryEscape(hc.password), serverAddr, serverPort)

	// Build query parameters
	var params []string

	// Add SNI if provided
	if hc.sni != "" {
		params = append(params, "sni="+url.QueryEscape(hc.sni))
	}

	// Add allowInsecure parameter
	if hc.allowInsecure {
		params = append(params, "insecure=1")
	}

	// Add obfs parameters
	if hc.obfs != "" {
		params = append(params, "obfs="+url.QueryEscape(hc.obfs))
		if hc.obfsPassword != "" {
			params = append(params, "obfs-password="+url.QueryEscape(hc.obfsPassword))
		}
	}

	// Add bandwidth limits
	if hc.upMbps != nil {
		params = append(params, "up="+strconv.Itoa(*hc.upMbps))
	}
	if hc.downMbps != nil {
		params = append(params, "down="+strconv.Itoa(*hc.downMbps))
	}

	// Add fingerprint if provided
	if hc.fingerprint != "" {
		params = append(params, "fp="+url.QueryEscape(hc.fingerprint))
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
func (hc Hysteria2Config) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("congestion_control=%s", hc.congestionControl))

	if hc.obfs != "" {
		parts = append(parts, fmt.Sprintf("obfs=%s", hc.obfs))
	}

	if hc.upMbps != nil {
		parts = append(parts, fmt.Sprintf("up_mbps=%d", *hc.upMbps))
	}

	if hc.downMbps != nil {
		parts = append(parts, fmt.Sprintf("down_mbps=%d", *hc.downMbps))
	}

	if hc.sni != "" {
		parts = append(parts, fmt.Sprintf("sni=%s", hc.sni))
	}

	if hc.allowInsecure {
		parts = append(parts, "allowInsecure=true")
	}

	if hc.fingerprint != "" {
		parts = append(parts, fmt.Sprintf("fingerprint=%s", hc.fingerprint))
	}

	return strings.Join(parts, ", ")
}

// Equals checks if two Hysteria2Config instances are equal
func (hc Hysteria2Config) Equals(other Hysteria2Config) bool {
	if hc.password != other.password ||
		hc.congestionControl != other.congestionControl ||
		hc.obfs != other.obfs ||
		hc.obfsPassword != other.obfsPassword ||
		hc.sni != other.sni ||
		hc.allowInsecure != other.allowInsecure ||
		hc.fingerprint != other.fingerprint {
		return false
	}

	// Compare optional bandwidth limits
	if (hc.upMbps == nil) != (other.upMbps == nil) {
		return false
	}
	if hc.upMbps != nil && *hc.upMbps != *other.upMbps {
		return false
	}

	if (hc.downMbps == nil) != (other.downMbps == nil) {
		return false
	}
	if hc.downMbps != nil && *hc.downMbps != *other.downMbps {
		return false
	}

	return true
}

// isValidHysteria2CongestionControl validates the congestion control algorithm for Hysteria2
func isValidHysteria2CongestionControl(cc string) bool {
	return validHysteria2CongestionControls[cc]
}
