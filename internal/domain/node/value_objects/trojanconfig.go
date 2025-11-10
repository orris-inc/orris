package value_objects

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	// TransportTCP represents TCP transport protocol
	TransportTCP = "tcp"
	// TransportWS represents WebSocket transport protocol
	TransportWS = "ws"
	// TransportGRPC represents gRPC transport protocol
	TransportGRPC = "grpc"
)

var validTransports = map[string]bool{
	TransportTCP:  true,
	TransportWS:   true,
	TransportGRPC: true,
}

// TrojanConfig represents the Trojan protocol configuration
// This is an immutable value object following DDD principles
type TrojanConfig struct {
	password          string
	transportProtocol string
	host              string
	path              string
	allowInsecure     bool
	sni               string
}

// NewTrojanConfig creates a new TrojanConfig with validation
func NewTrojanConfig(
	password string,
	transportProtocol string,
	host string,
	path string,
	allowInsecure bool,
	sni string,
) (TrojanConfig, error) {
	// Validate password
	if len(password) < 8 {
		return TrojanConfig{}, fmt.Errorf("password must be at least 8 characters long")
	}

	// Validate transport protocol
	if !isValidTransport(transportProtocol) {
		return TrojanConfig{}, fmt.Errorf("unsupported transport protocol: %s (must be tcp, ws, or grpc)", transportProtocol)
	}

	// Validate WebSocket-specific requirements
	if transportProtocol == TransportWS {
		if host == "" {
			return TrojanConfig{}, fmt.Errorf("host is required for WebSocket transport")
		}
		if path == "" {
			return TrojanConfig{}, fmt.Errorf("path is required for WebSocket transport")
		}
	}

	// Validate gRPC-specific requirements
	if transportProtocol == TransportGRPC {
		if host == "" {
			return TrojanConfig{}, fmt.Errorf("host is required for gRPC transport")
		}
	}

	return TrojanConfig{
		password:          password,
		transportProtocol: transportProtocol,
		host:              host,
		path:              path,
		allowInsecure:     allowInsecure,
		sni:               sni,
	}, nil
}

// Password returns the Trojan password
func (tc TrojanConfig) Password() string {
	return tc.password
}

// TransportProtocol returns the transport protocol
func (tc TrojanConfig) TransportProtocol() string {
	return tc.transportProtocol
}

// Host returns the host for WebSocket/gRPC
func (tc TrojanConfig) Host() string {
	return tc.host
}

// Path returns the path for WebSocket
func (tc TrojanConfig) Path() string {
	return tc.path
}

// AllowInsecure returns whether to allow insecure connections
func (tc TrojanConfig) AllowInsecure() bool {
	return tc.allowInsecure
}

// SNI returns the Server Name Indication
func (tc TrojanConfig) SNI() string {
	return tc.sni
}

// ToURI generates a Trojan URI string for subscription
// Format: trojan://password@host:port?parameters#remarks
func (tc TrojanConfig) ToURI(serverAddr string, serverPort uint16, remarks string) string {
	// Build base URI
	uri := fmt.Sprintf("trojan://%s@%s:%d", tc.password, serverAddr, serverPort)

	// Build query parameters
	params := url.Values{}

	// Add security parameter
	if tc.allowInsecure {
		params.Add("allowInsecure", "1")
	} else {
		params.Add("allowInsecure", "0")
	}

	// Add SNI if provided
	if tc.sni != "" {
		params.Add("sni", tc.sni)
	}

	// Add transport-specific parameters
	switch tc.transportProtocol {
	case TransportWS:
		params.Add("type", "ws")
		if tc.host != "" {
			params.Add("host", tc.host)
		}
		if tc.path != "" {
			params.Add("path", tc.path)
		}
	case TransportGRPC:
		params.Add("type", "grpc")
		if tc.host != "" {
			params.Add("serviceName", tc.host)
		}
	case TransportTCP:
		params.Add("type", "tcp")
	}

	// Append query parameters if any
	if len(params) > 0 {
		uri += "?" + params.Encode()
	}

	// Add remarks if provided
	if remarks != "" {
		uri += "#" + url.QueryEscape(remarks)
	}

	return uri
}

// String returns a string representation of the config
func (tc TrojanConfig) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("transport=%s", tc.transportProtocol))

	if tc.host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", tc.host))
	}

	if tc.path != "" {
		parts = append(parts, fmt.Sprintf("path=%s", tc.path))
	}

	if tc.sni != "" {
		parts = append(parts, fmt.Sprintf("sni=%s", tc.sni))
	}

	if tc.allowInsecure {
		parts = append(parts, "allowInsecure=true")
	}

	return strings.Join(parts, ", ")
}

// Equals checks if two TrojanConfig instances are equal
func (tc TrojanConfig) Equals(other TrojanConfig) bool {
	return tc.password == other.password &&
		tc.transportProtocol == other.transportProtocol &&
		tc.host == other.host &&
		tc.path == other.path &&
		tc.allowInsecure == other.allowInsecure &&
		tc.sni == other.sni
}

// isValidTransport validates the transport protocol
func isValidTransport(transport string) bool {
	return validTransports[transport]
}
