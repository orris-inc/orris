package value_objects_test

import (
	"testing"

	vo "orris/internal/domain/node/value_objects"
)

// TestTrojanConfigTCP tests creating a Trojan node with TCP transport
func TestTrojanConfigTCP(t *testing.T) {
	// Create a Trojan config with TCP transport
	config, err := vo.NewTrojanConfig(
		"mySecurePassword123",
		vo.TransportTCP,
		"",
		"",
		false,
		"example.com",
	)
	if err != nil {
		t.Fatalf("Failed to create Trojan config: %v", err)
	}

	// Generate subscription URI
	uri := config.ToURI("example.com", 443, "My Trojan Node")
	if uri == "" {
		t.Error("Expected non-empty URI")
	}
	t.Logf("Generated URI: %s", uri)
}

// TestTrojanConfigWebSocket tests creating a Trojan node with WebSocket transport
func TestTrojanConfigWebSocket(t *testing.T) {
	// Create a Trojan config with WebSocket transport
	config, err := vo.NewTrojanConfig(
		"mySecurePassword123",
		vo.TransportWS,
		"ws.example.com",
		"/trojan",
		false,
		"example.com",
	)
	if err != nil {
		t.Fatalf("Failed to create Trojan config: %v", err)
	}

	// Generate subscription URI
	uri := config.ToURI("example.com", 443, "My Trojan WS Node")
	if uri == "" {
		t.Error("Expected non-empty URI")
	}
	t.Logf("Generated URI: %s", uri)
}

// TestTrojanConfigGRPC tests creating a Trojan node with gRPC transport
func TestTrojanConfigGRPC(t *testing.T) {
	// Create a Trojan config with gRPC transport
	config, err := vo.NewTrojanConfig(
		"mySecurePassword123",
		vo.TransportGRPC,
		"TrojanService",
		"",
		false,
		"example.com",
	)
	if err != nil {
		t.Fatalf("Failed to create Trojan config: %v", err)
	}

	// Generate subscription URI
	uri := config.ToURI("example.com", 443, "My Trojan gRPC Node")
	if uri == "" {
		t.Error("Expected non-empty URI")
	}
	t.Logf("Generated URI: %s", uri)
}

// TestProtocolConfigFactory tests using ProtocolConfigFactory
func TestProtocolConfigFactory(t *testing.T) {
	factory := vo.NewProtocolConfigFactory()

	// Create a Trojan config using factory
	trojanConfig, err := factory.CreateTrojanConfig(
		"mySecurePassword123",
		vo.TransportWS,
		"ws.example.com",
		"/trojan",
		false,
		"example.com",
	)
	if err != nil {
		t.Fatalf("Failed to create Trojan config: %v", err)
	}

	// Generate subscription URI
	uri, err := factory.GenerateSubscriptionURI(
		vo.ProtocolTrojan,
		trojanConfig,
		"example.com",
		443,
		"My Trojan Node",
	)
	if err != nil {
		t.Fatalf("Failed to generate subscription URI: %v", err)
	}

	if uri == "" {
		t.Error("Expected non-empty URI")
	}
	t.Logf("Generated URI: %s", uri)
}

// TestCompareProtocols tests comparing Shadowsocks and Trojan
func TestCompareProtocols(t *testing.T) {
	factory := vo.NewProtocolConfigFactory()

	// Create Shadowsocks config
	ssConfig, err := factory.CreateShadowsocksConfig(
		vo.MethodAES256GCM,
		"myPassword123",
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to create Shadowsocks config: %v", err)
	}

	// Create Trojan config
	trojanConfig, err := factory.CreateTrojanConfig(
		"mySecurePassword123",
		vo.TransportTCP,
		"",
		"",
		false,
		"example.com",
	)
	if err != nil {
		t.Fatalf("Failed to create Trojan config: %v", err)
	}

	// Generate URIs for both
	ssURI, err := factory.GenerateSubscriptionURI(
		vo.ProtocolShadowsocks,
		ssConfig,
		"example.com",
		8388,
		"SS Node",
	)
	if err != nil {
		t.Fatalf("Failed to generate Shadowsocks URI: %v", err)
	}

	trojanURI, err := factory.GenerateSubscriptionURI(
		vo.ProtocolTrojan,
		trojanConfig,
		"example.com",
		443,
		"Trojan Node",
	)
	if err != nil {
		t.Fatalf("Failed to generate Trojan URI: %v", err)
	}

	t.Logf("Shadowsocks: %s", ssURI)
	t.Logf("Trojan: %s", trojanURI)
}

func TestTrojanConfigValidation(t *testing.T) {
	tests := []struct {
		name              string
		password          string
		transportProtocol string
		host              string
		path              string
		allowInsecure     bool
		sni               string
		wantErr           bool
		errMsg            string
	}{
		{
			name:              "Valid TCP config",
			password:          "password123",
			transportProtocol: vo.TransportTCP,
			host:              "",
			path:              "",
			allowInsecure:     false,
			sni:               "example.com",
			wantErr:           false,
		},
		{
			name:              "Valid WebSocket config",
			password:          "password123",
			transportProtocol: vo.TransportWS,
			host:              "ws.example.com",
			path:              "/trojan",
			allowInsecure:     false,
			sni:               "example.com",
			wantErr:           false,
		},
		{
			name:              "Valid gRPC config",
			password:          "password123",
			transportProtocol: vo.TransportGRPC,
			host:              "TrojanService",
			path:              "",
			allowInsecure:     false,
			sni:               "example.com",
			wantErr:           false,
		},
		{
			name:              "Invalid password - too short",
			password:          "pass",
			transportProtocol: vo.TransportTCP,
			host:              "",
			path:              "",
			allowInsecure:     false,
			sni:               "example.com",
			wantErr:           true,
			errMsg:            "password must be at least 8 characters long",
		},
		{
			name:              "Invalid transport protocol",
			password:          "password123",
			transportProtocol: "invalid",
			host:              "",
			path:              "",
			allowInsecure:     false,
			sni:               "",
			wantErr:           true,
			errMsg:            "unsupported transport protocol",
		},
		{
			name:              "WebSocket without host",
			password:          "password123",
			transportProtocol: vo.TransportWS,
			host:              "",
			path:              "/trojan",
			allowInsecure:     false,
			sni:               "",
			wantErr:           true,
			errMsg:            "host is required for WebSocket transport",
		},
		{
			name:              "WebSocket without path",
			password:          "password123",
			transportProtocol: vo.TransportWS,
			host:              "ws.example.com",
			path:              "",
			allowInsecure:     false,
			sni:               "",
			wantErr:           true,
			errMsg:            "path is required for WebSocket transport",
		},
		{
			name:              "gRPC without host",
			password:          "password123",
			transportProtocol: vo.TransportGRPC,
			host:              "",
			path:              "",
			allowInsecure:     false,
			sni:               "",
			wantErr:           true,
			errMsg:            "host is required for gRPC transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := vo.NewTrojanConfig(
				tt.password,
				tt.transportProtocol,
				tt.host,
				tt.path,
				tt.allowInsecure,
				tt.sni,
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.errMsg != "" && err != nil {
					// Check if error message contains expected substring
					if !contains(err.Error(), tt.errMsg) {
						t.Errorf("Expected error message to contain %q, got %q", tt.errMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
