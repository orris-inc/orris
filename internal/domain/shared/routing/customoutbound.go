package routing

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strings"
)

const (
	maxSettingsDepth = 10    // Maximum nesting depth for settings
	maxSettingsSize  = 65536 // Maximum total serialized size in bytes (64KB)
)

// customOutboundTagSuffixPattern validates the suffix part of a custom outbound tag.
// Only allows alphanumeric characters, hyphens, and underscores.
var customOutboundTagSuffixPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// hostnamePattern validates a DNS hostname (RFC 952/1123).
// Each label: starts/ends with alphanumeric, may contain hyphens, max 63 chars.
var hostnamePattern = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

// isSupportedCustomProtocol checks if a protocol is a supported sing-box outbound type.
func isSupportedCustomProtocol(protocol string) bool {
	switch protocol {
	case "shadowsocks", "trojan", "vless", "vmess", "hysteria2", "tuic", "anytls", "socks", "http":
		return true
	default:
		return false
	}
}

// CustomOutbound represents a user-defined sing-box outbound configuration.
// It stores the full protocol configuration for outbound connections
// that are not tied to any registered system node.
type CustomOutbound struct {
	tag      string         // Unique identifier, must start with "custom_"
	protocol string         // sing-box outbound type (shadowsocks, trojan, etc.)
	server   string         // Server address (IP or hostname)
	port     uint16         // Server port
	settings map[string]any // Protocol-specific configuration (password, uuid, method, tls, transport, etc.)
}

// NewCustomOutbound creates a new CustomOutbound with validation.
// The settings map is deep-copied to prevent external mutation.
// Returns an error if settings nesting exceeds maxSettingsDepth.
func NewCustomOutbound(tag, protocol, server string, port uint16, settings map[string]any) (*CustomOutbound, error) {
	copiedSettings, err := deepCopyMapWithDepth(settings, 0)
	if err != nil {
		return nil, fmt.Errorf("invalid settings: %w", err)
	}
	co := &CustomOutbound{
		tag:      tag,
		protocol: protocol,
		server:   server,
		port:     port,
		settings: copiedSettings,
	}
	if err := co.Validate(); err != nil {
		return nil, err
	}
	return co, nil
}

// Tag returns the custom outbound tag
func (co *CustomOutbound) Tag() string { return co.tag }

// Protocol returns the protocol type
func (co *CustomOutbound) Protocol() string { return co.protocol }

// Server returns the server address
func (co *CustomOutbound) Server() string { return co.server }

// Port returns the server port
func (co *CustomOutbound) Port() uint16 { return co.port }

// Settings returns a deep copy of the protocol-specific settings
func (co *CustomOutbound) Settings() map[string]any {
	if co.settings == nil {
		return nil
	}
	return deepCopyMap(co.settings)
}

// deepCopyMap performs a deep copy of a map[string]any, handling nested maps and slices.
func deepCopyMap(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopySlice performs a deep copy of a []any, handling nested maps and slices.
func deepCopySlice(s []any) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = deepCopyValue(v)
	}
	return result
}

// deepCopyValue performs a deep copy of a single value from a JSON-unmarshaled structure.
func deepCopyValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return deepCopyMap(val)
	case []any:
		return deepCopySlice(val)
	default:
		return v
	}
}

// deepCopyMapWithDepth performs a deep copy of a map[string]any with nesting depth limit.
// Returns an error if the nesting depth exceeds maxSettingsDepth.
func deepCopyMapWithDepth(m map[string]any, depth int) (map[string]any, error) {
	if depth > maxSettingsDepth {
		return nil, fmt.Errorf("settings nesting too deep: exceeds max depth %d", maxSettingsDepth)
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		copied, err := deepCopyValueWithDepth(v, depth)
		if err != nil {
			return nil, err
		}
		result[k] = copied
	}
	return result, nil
}

// deepCopySliceWithDepth performs a deep copy of a []any with nesting depth limit.
// Returns an error if the nesting depth exceeds maxSettingsDepth.
func deepCopySliceWithDepth(s []any, depth int) ([]any, error) {
	if depth > maxSettingsDepth {
		return nil, fmt.Errorf("settings nesting too deep: exceeds max depth %d", maxSettingsDepth)
	}
	result := make([]any, len(s))
	for i, v := range s {
		copied, err := deepCopyValueWithDepth(v, depth)
		if err != nil {
			return nil, err
		}
		result[i] = copied
	}
	return result, nil
}

// deepCopyValueWithDepth performs a deep copy of a single value with nesting depth limit.
// Returns an error if the nesting depth exceeds maxSettingsDepth.
func deepCopyValueWithDepth(v any, depth int) (any, error) {
	switch val := v.(type) {
	case map[string]any:
		return deepCopyMapWithDepth(val, depth+1)
	case []any:
		return deepCopySliceWithDepth(val, depth+1)
	default:
		return v, nil
	}
}

// Validate validates the custom outbound configuration
func (co *CustomOutbound) Validate() error {
	// Validate tag format
	if !strings.HasPrefix(co.tag, customOutboundPrefix) {
		return fmt.Errorf("custom outbound tag must start with '%s', got: %s", customOutboundPrefix, co.tag)
	}
	suffix := co.tag[len(customOutboundPrefix):]
	if suffix == "" {
		return fmt.Errorf("custom outbound tag must have content after '%s' prefix", customOutboundPrefix)
	}
	if len(suffix) > 64 {
		return fmt.Errorf("custom outbound tag suffix too long: max 64 characters")
	}
	if !customOutboundTagSuffixPattern.MatchString(suffix) {
		return fmt.Errorf("custom outbound tag suffix contains invalid characters: %s (only alphanumeric, hyphens, and underscores allowed)", suffix)
	}

	// Validate protocol
	if !isSupportedCustomProtocol(co.protocol) {
		return fmt.Errorf("unsupported custom outbound protocol: %s", co.protocol)
	}

	// Validate server (must be non-empty, valid IP or hostname)
	if co.server == "" {
		return fmt.Errorf("custom outbound server address is required")
	}
	if len(co.server) > 253 {
		return fmt.Errorf("custom outbound server address too long: max 253 characters")
	}
	if net.ParseIP(co.server) == nil {
		// Not an IP, validate as RFC-compliant hostname using whitelist pattern
		if !hostnamePattern.MatchString(co.server) {
			return fmt.Errorf("invalid custom outbound server address: %s (must be a valid IP or hostname)", co.server)
		}
	}

	// Validate port
	if co.port == 0 {
		return fmt.Errorf("custom outbound port must be between 1 and 65535")
	}

	// Validate settings size to prevent DoS via oversized configurations
	if len(co.settings) > 50 {
		return fmt.Errorf("custom outbound settings has too many keys: %d (max 50)", len(co.settings))
	}

	// Validate total serialized size of settings
	if len(co.settings) > 0 {
		settingsBytes, err := json.Marshal(co.settings)
		if err != nil {
			return fmt.Errorf("custom outbound settings cannot be serialized: %w", err)
		}
		if len(settingsBytes) > maxSettingsSize {
			return fmt.Errorf("custom outbound settings too large: %d bytes (max %d)", len(settingsBytes), maxSettingsSize)
		}
	}

	return nil
}

// Equals compares two CustomOutbound instances for equality, including settings.
func (co *CustomOutbound) Equals(other *CustomOutbound) bool {
	if co == nil && other == nil {
		return true
	}
	if co == nil || other == nil {
		return false
	}
	return co.tag == other.tag &&
		co.protocol == other.protocol &&
		co.server == other.server &&
		co.port == other.port &&
		reflect.DeepEqual(co.settings, other.settings)
}

// IsSupportedCustomProtocol checks if a protocol string is a supported custom outbound protocol.
func IsSupportedCustomProtocol(protocol string) bool {
	return isSupportedCustomProtocol(protocol)
}

// ReconstructCustomOutbound reconstructs a CustomOutbound from persistence data without validation.
func ReconstructCustomOutbound(tag, protocol, server string, port uint16, settings map[string]any) *CustomOutbound {
	return &CustomOutbound{
		tag:      tag,
		protocol: protocol,
		server:   server,
		port:     port,
		settings: settings,
	}
}
