package valueobjects

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// VMess security types
const (
	// SecurityAuto represents auto security selection
	SecurityAuto = "auto"
	// SecurityAES128GCM represents AES-128-GCM encryption
	SecurityAES128GCM = "aes-128-gcm"
	// SecurityChacha20Poly1305 represents ChaCha20-Poly1305 encryption
	SecurityChacha20Poly1305 = "chacha20-poly1305"
	// SecurityNone represents no encryption
	SecurityNone = "none"
	// SecurityZero represents zero encryption (no encryption, no authentication)
	SecurityZero = "zero"
)

// VMess transport types (reuse existing constants where possible)
const (
	// VMess specific transports
	VMessTransportTCP  = "tcp"
	VMessTransportWS   = "ws"
	VMessTransportGRPC = "grpc"
	VMessTransportHTTP = "http"
	VMessTransportQUIC = "quic"
)

var validVMessSecurities = map[string]bool{
	SecurityAuto:             true,
	SecurityAES128GCM:        true,
	SecurityChacha20Poly1305: true,
	SecurityNone:             true,
	SecurityZero:             true,
}

var validVMessTransports = map[string]bool{
	VMessTransportTCP:  true,
	VMessTransportWS:   true,
	VMessTransportGRPC: true,
	VMessTransportHTTP: true,
	VMessTransportQUIC: true,
}

// VMessConfig represents the VMess protocol configuration
// This is an immutable value object following DDD principles
type VMessConfig struct {
	alterID       int    // Usually 0 for modern clients
	security      string // auto, aes-128-gcm, chacha20-poly1305, none, zero
	transportType string // tcp, ws, grpc, http, quic
	host          string // WebSocket/HTTP host header
	path          string // WebSocket/HTTP path
	serviceName   string // gRPC service name
	tls           bool   // Enable TLS
	sni           string // TLS Server Name Indication
	allowInsecure bool   // Allow insecure TLS connection
}

// NewVMessConfig creates a new VMessConfig with validation
func NewVMessConfig(
	alterID int,
	security string,
	transportType string,
	host string,
	path string,
	serviceName string,
	tls bool,
	sni string,
	allowInsecure bool,
) (VMessConfig, error) {
	// Validate alterID
	if alterID < 0 {
		return VMessConfig{}, fmt.Errorf("alterID must be non-negative, got %d", alterID)
	}

	// Validate security
	if !isValidVMessSecurity(security) {
		return VMessConfig{}, fmt.Errorf("unsupported security type: %s (must be auto, aes-128-gcm, chacha20-poly1305, none, or zero)", security)
	}

	// Validate transport type
	if !isValidVMessTransport(transportType) {
		return VMessConfig{}, fmt.Errorf("unsupported transport type: %s (must be tcp, ws, grpc, http, or quic)", transportType)
	}

	// Validate WebSocket-specific requirements
	if transportType == VMessTransportWS {
		if path == "" {
			return VMessConfig{}, fmt.Errorf("path is required for WebSocket transport")
		}
	}

	// Validate HTTP-specific requirements
	if transportType == VMessTransportHTTP {
		if path == "" {
			return VMessConfig{}, fmt.Errorf("path is required for HTTP transport")
		}
	}

	// Validate gRPC-specific requirements
	if transportType == VMessTransportGRPC {
		if serviceName == "" {
			return VMessConfig{}, fmt.Errorf("serviceName is required for gRPC transport")
		}
	}

	return VMessConfig{
		alterID:       alterID,
		security:      security,
		transportType: transportType,
		host:          host,
		path:          path,
		serviceName:   serviceName,
		tls:           tls,
		sni:           sni,
		allowInsecure: allowInsecure,
	}, nil
}

// AlterID returns the alter ID
func (vc VMessConfig) AlterID() int {
	return vc.alterID
}

// Security returns the security type
func (vc VMessConfig) Security() string {
	return vc.security
}

// TransportType returns the transport type
func (vc VMessConfig) TransportType() string {
	return vc.transportType
}

// Host returns the host for WebSocket/HTTP
func (vc VMessConfig) Host() string {
	return vc.host
}

// Path returns the path for WebSocket/HTTP
func (vc VMessConfig) Path() string {
	return vc.path
}

// ServiceName returns the gRPC service name
func (vc VMessConfig) ServiceName() string {
	return vc.serviceName
}

// TLS returns whether TLS is enabled
func (vc VMessConfig) TLS() bool {
	return vc.tls
}

// SNI returns the Server Name Indication
func (vc VMessConfig) SNI() string {
	return vc.sni
}

// AllowInsecure returns whether to allow insecure connections
func (vc VMessConfig) AllowInsecure() bool {
	return vc.allowInsecure
}

// vmessJSONConfig represents the v2rayN JSON format for VMess URI
type vmessJSONConfig struct {
	V    string `json:"v"`    // Version (always "2")
	PS   string `json:"ps"`   // Remarks/alias
	Add  string `json:"add"`  // Server address
	Port string `json:"port"` // Server port (as string in v2rayN format)
	ID   string `json:"id"`   // UUID
	Aid  string `json:"aid"`  // Alter ID (as string in v2rayN format)
	Scy  string `json:"scy"`  // Security
	Net  string `json:"net"`  // Network/transport type
	Type string `json:"type"` // Header type (usually "none")
	Host string `json:"host"` // Host header
	Path string `json:"path"` // Path
	TLS  string `json:"tls"`  // TLS setting ("tls" or "")
	SNI  string `json:"sni"`  // Server Name Indication
	ALPN string `json:"alpn"` // ALPN (optional)
}

// ToURI generates a VMess URI string for subscription
// Format: vmess://base64(json) following v2rayN standard
func (vc VMessConfig) ToURI(serverAddr string, serverPort uint16, uuid string, remarks string) (string, error) {
	config := vmessJSONConfig{
		V:    "2",
		PS:   remarks,
		Add:  serverAddr,
		Port: strconv.Itoa(int(serverPort)),
		ID:   uuid,
		Aid:  strconv.Itoa(vc.alterID),
		Scy:  vc.security,
		Net:  vc.transportType,
		Type: "none",
	}

	// Set TLS
	if vc.tls {
		config.TLS = "tls"
		if vc.sni != "" {
			config.SNI = vc.sni
		}
	}

	// Set transport-specific parameters
	switch vc.transportType {
	case VMessTransportWS:
		config.Host = vc.host
		config.Path = vc.path
	case VMessTransportHTTP:
		config.Host = vc.host
		config.Path = vc.path
	case VMessTransportGRPC:
		config.Path = vc.serviceName // gRPC service name goes in path field
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal vmess config: %w", err)
	}

	// Encode to base64
	return "vmess://" + base64.StdEncoding.EncodeToString(jsonData), nil
}

// ToStandardURI generates a VMess URI in standard format (non-v2rayN)
// Format: vmess://uuid@host:port?params#remarks
func (vc VMessConfig) ToStandardURI(serverAddr string, serverPort uint16, uuid string, remarks string) string {
	// Build base URI
	uri := fmt.Sprintf("vmess://%s@%s:%d", uuid, serverAddr, serverPort)

	// Build query parameters
	var params []string

	// Add encryption/security
	if vc.security != "" && vc.security != SecurityAuto {
		params = append(params, "encryption="+url.QueryEscape(vc.security))
	}

	// Add transport type
	params = append(params, "type="+vc.transportType)

	// Add TLS security
	if vc.tls {
		params = append(params, "security=tls")
		if vc.sni != "" {
			params = append(params, "sni="+url.QueryEscape(vc.sni))
		}
		if vc.allowInsecure {
			params = append(params, "allowInsecure=1")
		}
	} else {
		params = append(params, "security=none")
	}

	// Add transport-specific parameters
	switch vc.transportType {
	case VMessTransportWS, VMessTransportHTTP:
		if vc.host != "" {
			params = append(params, "host="+url.QueryEscape(vc.host))
		}
		if vc.path != "" {
			params = append(params, "path="+url.QueryEscape(vc.path))
		}
	case VMessTransportGRPC:
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
func (vc VMessConfig) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("transport=%s", vc.transportType))
	parts = append(parts, fmt.Sprintf("security=%s", vc.security))
	parts = append(parts, fmt.Sprintf("alterID=%d", vc.alterID))

	if vc.tls {
		parts = append(parts, "tls=true")
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

	if vc.sni != "" {
		parts = append(parts, fmt.Sprintf("sni=%s", vc.sni))
	}

	if vc.allowInsecure {
		parts = append(parts, "allowInsecure=true")
	}

	return strings.Join(parts, ", ")
}

// Equals checks if two VMessConfig instances are equal
func (vc VMessConfig) Equals(other VMessConfig) bool {
	return vc.alterID == other.alterID &&
		vc.security == other.security &&
		vc.transportType == other.transportType &&
		vc.host == other.host &&
		vc.path == other.path &&
		vc.serviceName == other.serviceName &&
		vc.tls == other.tls &&
		vc.sni == other.sni &&
		vc.allowInsecure == other.allowInsecure
}

// isValidVMessSecurity validates the security type
func isValidVMessSecurity(security string) bool {
	return validVMessSecurities[security]
}

// isValidVMessTransport validates the transport type
func isValidVMessTransport(transport string) bool {
	return validVMessTransports[transport]
}
