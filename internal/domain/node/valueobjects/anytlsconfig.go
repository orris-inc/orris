package valueobjects

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// validAnyTLSFingerprints defines the valid TLS fingerprints for AnyTLS
var validAnyTLSFingerprints = map[string]bool{
	"chrome":     true,
	"firefox":    true,
	"safari":     true,
	"ios":        true,
	"android":    true,
	"edge":       true,
	"360":        true,
	"qq":         true,
	"random":     true,
	"randomized": true,
	"":           true, // allow empty (use default)
}

// AnyTLSConfig represents the AnyTLS protocol configuration.
// This is an immutable value object following DDD principles.
type AnyTLSConfig struct {
	password                  string
	sni                       string
	allowInsecure             bool
	fingerprint               string
	idleSessionCheckInterval  string // duration string, e.g. "30s"
	idleSessionTimeout        string // duration string, e.g. "30s"
	minIdleSession            int
}

// NewAnyTLSConfig creates a new AnyTLSConfig with validation
func NewAnyTLSConfig(
	password string,
	sni string,
	allowInsecure bool,
	fingerprint string,
	idleSessionCheckInterval string,
	idleSessionTimeout string,
	minIdleSession int,
) (AnyTLSConfig, error) {
	// Validate password
	if len(password) < 8 {
		return AnyTLSConfig{}, fmt.Errorf("password must be at least 8 characters long")
	}

	// Validate fingerprint
	if !validAnyTLSFingerprints[fingerprint] {
		return AnyTLSConfig{}, fmt.Errorf("unsupported TLS fingerprint: %s", fingerprint)
	}

	// Validate duration strings
	if idleSessionCheckInterval != "" {
		if _, err := time.ParseDuration(idleSessionCheckInterval); err != nil {
			return AnyTLSConfig{}, fmt.Errorf("invalid idle_session_check_interval: %w", err)
		}
	}
	if idleSessionTimeout != "" {
		if _, err := time.ParseDuration(idleSessionTimeout); err != nil {
			return AnyTLSConfig{}, fmt.Errorf("invalid idle_session_timeout: %w", err)
		}
	}

	// Validate minIdleSession
	if minIdleSession < 0 {
		return AnyTLSConfig{}, fmt.Errorf("min_idle_session must be non-negative")
	}

	return AnyTLSConfig{
		password:                 password,
		sni:                      sni,
		allowInsecure:            allowInsecure,
		fingerprint:              fingerprint,
		idleSessionCheckInterval: idleSessionCheckInterval,
		idleSessionTimeout:       idleSessionTimeout,
		minIdleSession:           minIdleSession,
	}, nil
}

// Password returns the AnyTLS password
func (c AnyTLSConfig) Password() string {
	return c.password
}

// SNI returns the Server Name Indication
func (c AnyTLSConfig) SNI() string {
	return c.sni
}

// AllowInsecure returns whether to allow insecure connections
func (c AnyTLSConfig) AllowInsecure() bool {
	return c.allowInsecure
}

// Fingerprint returns the TLS fingerprint
func (c AnyTLSConfig) Fingerprint() string {
	return c.fingerprint
}

// IdleSessionCheckInterval returns the idle session check interval
func (c AnyTLSConfig) IdleSessionCheckInterval() string {
	return c.idleSessionCheckInterval
}

// IdleSessionTimeout returns the idle session timeout
func (c AnyTLSConfig) IdleSessionTimeout() string {
	return c.idleSessionTimeout
}

// MinIdleSession returns the minimum idle session count
func (c AnyTLSConfig) MinIdleSession() int {
	return c.minIdleSession
}

// ToURI generates an AnyTLS URI string for subscription
// Format: anytls://password@host:port?security=tls&sni=xxx&allowInsecure=1&fp=chrome#remarks
// Password is passed externally as it's derived from the subscription UUID, not stored in config.
func (c AnyTLSConfig) ToURI(serverAddr string, serverPort uint16, remarks string, password string) string {
	uri := fmt.Sprintf("anytls://%s@%s:%d", password, serverAddr, serverPort)

	var params []string

	// Add security parameter (AnyTLS requires TLS)
	params = append(params, "security=tls")

	// Add allowInsecure parameter
	if c.allowInsecure {
		params = append(params, "allowInsecure=1")
	}

	// Add SNI if provided
	if c.sni != "" {
		params = append(params, "sni="+url.QueryEscape(c.sni))
	}

	// Add fingerprint if provided
	if c.fingerprint != "" {
		params = append(params, "fp="+url.QueryEscape(c.fingerprint))
	}

	// Add idle session parameters if provided
	if c.idleSessionCheckInterval != "" {
		params = append(params, "idle-session-check-interval="+url.QueryEscape(c.idleSessionCheckInterval))
	}
	if c.idleSessionTimeout != "" {
		params = append(params, "idle-session-timeout="+url.QueryEscape(c.idleSessionTimeout))
	}
	if c.minIdleSession > 0 {
		params = append(params, fmt.Sprintf("min-idle-session=%d", c.minIdleSession))
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
func (c AnyTLSConfig) String() string {
	var parts []string

	if c.sni != "" {
		parts = append(parts, fmt.Sprintf("sni=%s", c.sni))
	}

	if c.allowInsecure {
		parts = append(parts, "allowInsecure=true")
	}

	if c.fingerprint != "" {
		parts = append(parts, fmt.Sprintf("fingerprint=%s", c.fingerprint))
	}

	if c.idleSessionCheckInterval != "" {
		parts = append(parts, fmt.Sprintf("idle_check=%s", c.idleSessionCheckInterval))
	}

	if c.idleSessionTimeout != "" {
		parts = append(parts, fmt.Sprintf("idle_timeout=%s", c.idleSessionTimeout))
	}

	if c.minIdleSession > 0 {
		parts = append(parts, fmt.Sprintf("min_idle=%d", c.minIdleSession))
	}

	return strings.Join(parts, ", ")
}

// Equals checks if two AnyTLSConfig instances are equal
func (c AnyTLSConfig) Equals(other AnyTLSConfig) bool {
	return c.password == other.password &&
		c.sni == other.sni &&
		c.allowInsecure == other.allowInsecure &&
		c.fingerprint == other.fingerprint &&
		c.idleSessionCheckInterval == other.idleSessionCheckInterval &&
		c.idleSessionTimeout == other.idleSessionTimeout &&
		c.minIdleSession == other.minIdleSession
}
