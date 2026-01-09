package valueobjects

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	// VLESSTransportTCP represents TCP transport protocol for VLESS
	VLESSTransportTCP = "tcp"
	// VLESSTransportWS represents WebSocket transport protocol for VLESS
	VLESSTransportWS = "ws"
	// VLESSTransportGRPC represents gRPC transport protocol for VLESS
	VLESSTransportGRPC = "grpc"
	// VLESSTransportH2 represents HTTP/2 transport protocol for VLESS
	VLESSTransportH2 = "h2"

	// VLESSSecurityNone represents no security
	VLESSSecurityNone = "none"
	// VLESSSecurityTLS represents TLS security
	VLESSSecurityTLS = "tls"
	// VLESSSecurityReality represents Reality security
	VLESSSecurityReality = "reality"

	// VLESSFlowVision represents XTLS-RPRX-Vision flow control
	VLESSFlowVision = "xtls-rprx-vision"
)

var validVLESSTransports = map[string]bool{
	VLESSTransportTCP:  true,
	VLESSTransportWS:   true,
	VLESSTransportGRPC: true,
	VLESSTransportH2:   true,
}

var validVLESSSecurity = map[string]bool{
	VLESSSecurityNone:    true,
	VLESSSecurityTLS:     true,
	VLESSSecurityReality: true,
}

// VLESSConfig represents the VLESS protocol configuration
// This is an immutable value object following DDD principles
type VLESSConfig struct {
	// Transport layer configuration
	transportType string

	// Flow control (xtls-rprx-vision or empty)
	flow string

	// Security configuration
	security      string
	sni           string
	fingerprint   string
	allowInsecure bool

	// WebSocket/H2 specific
	host string
	path string

	// gRPC specific
	serviceName string

	// Reality specific
	publicKey string
	shortID   string
	spiderX   string
}

// NewVLESSConfig creates a new VLESSConfig with validation
func NewVLESSConfig(
	transportType string,
	flow string,
	security string,
	sni string,
	fingerprint string,
	allowInsecure bool,
	host string,
	path string,
	serviceName string,
	publicKey string,
	shortID string,
	spiderX string,
) (VLESSConfig, error) {
	// Validate transport type
	if !validVLESSTransports[transportType] {
		return VLESSConfig{}, fmt.Errorf("unsupported transport type: %s (must be tcp, ws, grpc, or h2)", transportType)
	}

	// Validate security
	if !validVLESSSecurity[security] {
		return VLESSConfig{}, fmt.Errorf("unsupported security type: %s (must be none, tls, or reality)", security)
	}

	// Validate flow
	if flow != "" && flow != VLESSFlowVision {
		return VLESSConfig{}, fmt.Errorf("unsupported flow: %s (must be empty or xtls-rprx-vision)", flow)
	}

	// Validate Reality-specific requirements
	if security == VLESSSecurityReality {
		if publicKey == "" {
			return VLESSConfig{}, fmt.Errorf("public_key is required for Reality security")
		}
		if shortID == "" {
			return VLESSConfig{}, fmt.Errorf("short_id is required for Reality security")
		}
	}

	// Validate WebSocket/H2-specific requirements
	if transportType == VLESSTransportWS || transportType == VLESSTransportH2 {
		if host == "" {
			return VLESSConfig{}, fmt.Errorf("host is required for %s transport", transportType)
		}
		if path == "" {
			return VLESSConfig{}, fmt.Errorf("path is required for %s transport", transportType)
		}
	}

	// Validate gRPC-specific requirements
	if transportType == VLESSTransportGRPC {
		if serviceName == "" {
			return VLESSConfig{}, fmt.Errorf("service_name is required for gRPC transport")
		}
	}

	return VLESSConfig{
		transportType: transportType,
		flow:          flow,
		security:      security,
		sni:           sni,
		fingerprint:   fingerprint,
		allowInsecure: allowInsecure,
		host:          host,
		path:          path,
		serviceName:   serviceName,
		publicKey:     publicKey,
		shortID:       shortID,
		spiderX:       spiderX,
	}, nil
}

// TransportType returns the transport type
func (vc VLESSConfig) TransportType() string {
	return vc.transportType
}

// Flow returns the flow control setting
func (vc VLESSConfig) Flow() string {
	return vc.flow
}

// Security returns the security type
func (vc VLESSConfig) Security() string {
	return vc.security
}

// SNI returns the Server Name Indication
func (vc VLESSConfig) SNI() string {
	return vc.sni
}

// Fingerprint returns the TLS fingerprint
func (vc VLESSConfig) Fingerprint() string {
	return vc.fingerprint
}

// AllowInsecure returns whether to allow insecure connections
func (vc VLESSConfig) AllowInsecure() bool {
	return vc.allowInsecure
}

// Host returns the host for WebSocket/H2
func (vc VLESSConfig) Host() string {
	return vc.host
}

// Path returns the path for WebSocket/H2
func (vc VLESSConfig) Path() string {
	return vc.path
}

// ServiceName returns the gRPC service name
func (vc VLESSConfig) ServiceName() string {
	return vc.serviceName
}

// PublicKey returns the Reality public key
func (vc VLESSConfig) PublicKey() string {
	return vc.publicKey
}

// ShortID returns the Reality short ID
func (vc VLESSConfig) ShortID() string {
	return vc.shortID
}

// SpiderX returns the Reality spider X parameter
func (vc VLESSConfig) SpiderX() string {
	return vc.spiderX
}

// ToURI generates a VLESS URI string for subscription
// Format: vless://uuid@host:port?type=<transport>&security=<security>[&params]#remarks
func (vc VLESSConfig) ToURI(uuid string, serverAddr string, serverPort uint16, remarks string) string {
	// Build base URI
	uri := fmt.Sprintf("vless://%s@%s:%d", uuid, serverAddr, serverPort)

	// Build query parameters
	var params []string

	// Add transport type
	params = append(params, "type="+vc.transportType)

	// Add security
	params = append(params, "security="+vc.security)

	// Add flow if present
	if vc.flow != "" {
		params = append(params, "flow="+url.QueryEscape(vc.flow))
	}

	// Add TLS/Reality common parameters
	if vc.security == VLESSSecurityTLS || vc.security == VLESSSecurityReality {
		if vc.sni != "" {
			params = append(params, "sni="+url.QueryEscape(vc.sni))
		}
		if vc.fingerprint != "" {
			params = append(params, "fp="+url.QueryEscape(vc.fingerprint))
		}
	}

	// Add TLS-specific parameters
	if vc.security == VLESSSecurityTLS {
		if vc.allowInsecure {
			params = append(params, "allowInsecure=1")
		}
	}

	// Add Reality-specific parameters
	if vc.security == VLESSSecurityReality {
		params = append(params, "pbk="+url.QueryEscape(vc.publicKey))
		params = append(params, "sid="+url.QueryEscape(vc.shortID))
		if vc.spiderX != "" {
			params = append(params, "spx="+url.QueryEscape(vc.spiderX))
		}
	}

	// Add transport-specific parameters
	switch vc.transportType {
	case VLESSTransportWS, VLESSTransportH2:
		if vc.host != "" {
			params = append(params, "host="+url.QueryEscape(vc.host))
		}
		if vc.path != "" {
			params = append(params, "path="+url.QueryEscape(vc.path))
		}
	case VLESSTransportGRPC:
		if vc.serviceName != "" {
			params = append(params, "serviceName="+url.QueryEscape(vc.serviceName))
		}
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
func (vc VLESSConfig) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("transport=%s", vc.transportType))
	parts = append(parts, fmt.Sprintf("security=%s", vc.security))

	if vc.flow != "" {
		parts = append(parts, fmt.Sprintf("flow=%s", vc.flow))
	}

	if vc.sni != "" {
		parts = append(parts, fmt.Sprintf("sni=%s", vc.sni))
	}

	if vc.fingerprint != "" {
		parts = append(parts, fmt.Sprintf("fingerprint=%s", vc.fingerprint))
	}

	if vc.host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", vc.host))
	}

	if vc.path != "" {
		parts = append(parts, fmt.Sprintf("path=%s", vc.path))
	}

	if vc.serviceName != "" {
		parts = append(parts, fmt.Sprintf("serviceName=%s", vc.serviceName))
	}

	if vc.security == VLESSSecurityReality {
		parts = append(parts, fmt.Sprintf("publicKey=%s", vc.publicKey))
		parts = append(parts, fmt.Sprintf("shortID=%s", vc.shortID))
		if vc.spiderX != "" {
			parts = append(parts, fmt.Sprintf("spiderX=%s", vc.spiderX))
		}
	}

	if vc.allowInsecure {
		parts = append(parts, "allowInsecure=true")
	}

	return strings.Join(parts, ", ")
}

// Equals checks if two VLESSConfig instances are equal
func (vc VLESSConfig) Equals(other VLESSConfig) bool {
	return vc.transportType == other.transportType &&
		vc.flow == other.flow &&
		vc.security == other.security &&
		vc.sni == other.sni &&
		vc.fingerprint == other.fingerprint &&
		vc.allowInsecure == other.allowInsecure &&
		vc.host == other.host &&
		vc.path == other.path &&
		vc.serviceName == other.serviceName &&
		vc.publicKey == other.publicKey &&
		vc.shortID == other.shortID &&
		vc.spiderX == other.spiderX
}
