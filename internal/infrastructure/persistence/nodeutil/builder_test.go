package nodeutil

import (
	"testing"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

func TestNormalizeProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		want     string
	}{
		{
			name:     "empty string defaults to shadowsocks",
			protocol: "",
			want:     "shadowsocks",
		},
		{
			name:     "shadowsocks protocol unchanged",
			protocol: "shadowsocks",
			want:     "shadowsocks",
		},
		{
			name:     "trojan protocol unchanged",
			protocol: "trojan",
			want:     "trojan",
		},
		{
			name:     "custom protocol unchanged",
			protocol: "vless",
			want:     "vless",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeProtocol(tt.protocol)
			if got != tt.want {
				t.Errorf("normalizeProtocol(%q) = %q, want %q", tt.protocol, got, tt.want)
			}
		})
	}
}

func TestNewProtocolConfigs(t *testing.T) {
	configs := NewProtocolConfigs()

	if configs.Trojan == nil {
		t.Error("Trojan map should be initialized")
	}
	if configs.Shadowsocks == nil {
		t.Error("Shadowsocks map should be initialized")
	}
	if len(configs.Trojan) != 0 {
		t.Error("Trojan map should be empty")
	}
	if len(configs.Shadowsocks) != 0 {
		t.Error("Shadowsocks map should be empty")
	}
}

func TestBuildNode(t *testing.T) {
	tests := []struct {
		name     string
		source   NodeSource
		configs  ProtocolConfigs
		wantNode *usecases.Node
	}{
		{
			name: "build shadowsocks node with config",
			source: NodeSource{
				ID:        1,
				Name:      "test-node",
				Address:   "1.2.3.4",
				Port:      8388,
				Protocol:  "shadowsocks",
				TokenHash: "hash123",
				SortOrder: 10,
			},
			configs: func() ProtocolConfigs {
				c := NewProtocolConfigs()
				plugin := "obfs-local"
				c.Shadowsocks[1] = &models.ShadowsocksConfigModel{
					NodeID:           1,
					EncryptionMethod: "aes-256-gcm",
					Plugin:           &plugin,
					PluginOpts:       []byte(`{"mode": "tls"}`),
				}
				return c
			}(),
			wantNode: &usecases.Node{
				ID:               1,
				Name:             "test-node",
				ServerAddress:    "1.2.3.4",
				SubscriptionPort: 8388,
				Protocol:         "shadowsocks",
				TokenHash:        "hash123",
				Password:         "",
				SortOrder:        10,
				EncryptionMethod: "aes-256-gcm",
				Plugin:           "obfs-local",
				PluginOpts:       map[string]string{"mode": "tls"},
			},
		},
		{
			name: "build trojan node with config",
			source: NodeSource{
				ID:        2,
				Name:      "trojan-node",
				Address:   "5.6.7.8",
				Port:      443,
				Protocol:  "trojan",
				TokenHash: "hash456",
				SortOrder: 20,
			},
			configs: func() ProtocolConfigs {
				c := NewProtocolConfigs()
				c.Trojan[2] = &models.TrojanConfigModel{
					NodeID:            2,
					TransportProtocol: "ws",
					Host:              "example.com",
					Path:              "/path",
					SNI:               "sni.example.com",
					AllowInsecure:     true,
				}
				return c
			}(),
			wantNode: &usecases.Node{
				ID:                2,
				Name:              "trojan-node",
				ServerAddress:     "5.6.7.8",
				SubscriptionPort:  443,
				Protocol:          "trojan",
				TokenHash:         "hash456",
				Password:          "",
				SortOrder:         20,
				TransportProtocol: "ws",
				Host:              "example.com",
				Path:              "/path",
				SNI:               "sni.example.com",
				AllowInsecure:     true,
			},
		},
		{
			name: "build node without config",
			source: NodeSource{
				ID:        3,
				Name:      "no-config-node",
				Address:   "9.10.11.12",
				Port:      1234,
				Protocol:  "shadowsocks",
				TokenHash: "hash789",
				SortOrder: 30,
			},
			configs: NewProtocolConfigs(),
			wantNode: &usecases.Node{
				ID:               3,
				Name:             "no-config-node",
				ServerAddress:    "9.10.11.12",
				SubscriptionPort: 1234,
				Protocol:         "shadowsocks",
				TokenHash:        "hash789",
				Password:         "",
				SortOrder:        30,
			},
		},
		{
			name: "build node with empty protocol defaults to shadowsocks",
			source: NodeSource{
				ID:        4,
				Name:      "default-protocol-node",
				Address:   "1.1.1.1",
				Port:      8080,
				Protocol:  "",
				TokenHash: "hashABC",
				SortOrder: 40,
			},
			configs: NewProtocolConfigs(),
			wantNode: &usecases.Node{
				ID:               4,
				Name:             "default-protocol-node",
				ServerAddress:    "1.1.1.1",
				SubscriptionPort: 8080,
				Protocol:         "shadowsocks",
				TokenHash:        "hashABC",
				Password:         "",
				SortOrder:        40,
			},
		},
		{
			name: "build trojan node without config defaults transport protocol",
			source: NodeSource{
				ID:        5,
				Name:      "trojan-no-config",
				Address:   "2.2.2.2",
				Port:      443,
				Protocol:  "trojan",
				TokenHash: "hashDEF",
				SortOrder: 50,
			},
			configs: NewProtocolConfigs(),
			wantNode: &usecases.Node{
				ID:                5,
				Name:              "trojan-no-config",
				ServerAddress:     "2.2.2.2",
				SubscriptionPort:  443,
				Protocol:          "trojan",
				TokenHash:         "hashDEF",
				Password:          "",
				SortOrder:         50,
				TransportProtocol: "tcp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildNode(tt.source, tt.configs)

			if got.ID != tt.wantNode.ID {
				t.Errorf("ID = %d, want %d", got.ID, tt.wantNode.ID)
			}
			if got.Name != tt.wantNode.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantNode.Name)
			}
			if got.ServerAddress != tt.wantNode.ServerAddress {
				t.Errorf("ServerAddress = %q, want %q", got.ServerAddress, tt.wantNode.ServerAddress)
			}
			if got.SubscriptionPort != tt.wantNode.SubscriptionPort {
				t.Errorf("SubscriptionPort = %d, want %d", got.SubscriptionPort, tt.wantNode.SubscriptionPort)
			}
			if got.Protocol != tt.wantNode.Protocol {
				t.Errorf("Protocol = %q, want %q", got.Protocol, tt.wantNode.Protocol)
			}
			if got.TokenHash != tt.wantNode.TokenHash {
				t.Errorf("TokenHash = %q, want %q", got.TokenHash, tt.wantNode.TokenHash)
			}
			if got.SortOrder != tt.wantNode.SortOrder {
				t.Errorf("SortOrder = %d, want %d", got.SortOrder, tt.wantNode.SortOrder)
			}
			if got.EncryptionMethod != tt.wantNode.EncryptionMethod {
				t.Errorf("EncryptionMethod = %q, want %q", got.EncryptionMethod, tt.wantNode.EncryptionMethod)
			}
			if got.Plugin != tt.wantNode.Plugin {
				t.Errorf("Plugin = %q, want %q", got.Plugin, tt.wantNode.Plugin)
			}
			if got.TransportProtocol != tt.wantNode.TransportProtocol {
				t.Errorf("TransportProtocol = %q, want %q", got.TransportProtocol, tt.wantNode.TransportProtocol)
			}
			if got.Host != tt.wantNode.Host {
				t.Errorf("Host = %q, want %q", got.Host, tt.wantNode.Host)
			}
			if got.Path != tt.wantNode.Path {
				t.Errorf("Path = %q, want %q", got.Path, tt.wantNode.Path)
			}
			if got.SNI != tt.wantNode.SNI {
				t.Errorf("SNI = %q, want %q", got.SNI, tt.wantNode.SNI)
			}
			if got.AllowInsecure != tt.wantNode.AllowInsecure {
				t.Errorf("AllowInsecure = %v, want %v", got.AllowInsecure, tt.wantNode.AllowInsecure)
			}
		})
	}
}

func TestApplyProtocolConfig(t *testing.T) {
	tests := []struct {
		name     string
		node     *usecases.Node
		protocol string
		nodeID   uint
		configs  ProtocolConfigs
		wantNode *usecases.Node
	}{
		{
			name:     "apply shadowsocks config with plugin",
			node:     &usecases.Node{ID: 1},
			protocol: "shadowsocks",
			nodeID:   1,
			configs: func() ProtocolConfigs {
				c := NewProtocolConfigs()
				plugin := "v2ray-plugin"
				c.Shadowsocks[1] = &models.ShadowsocksConfigModel{
					NodeID:           1,
					EncryptionMethod: "chacha20-ietf-poly1305",
					Plugin:           &plugin,
					PluginOpts:       []byte(`{"host": "cdn.example.com"}`),
				}
				return c
			}(),
			wantNode: &usecases.Node{
				ID:               1,
				EncryptionMethod: "chacha20-ietf-poly1305",
				Plugin:           "v2ray-plugin",
				PluginOpts:       map[string]string{"host": "cdn.example.com"},
			},
		},
		{
			name:     "apply shadowsocks config without plugin",
			node:     &usecases.Node{ID: 2},
			protocol: "shadowsocks",
			nodeID:   2,
			configs: func() ProtocolConfigs {
				c := NewProtocolConfigs()
				c.Shadowsocks[2] = &models.ShadowsocksConfigModel{
					NodeID:           2,
					EncryptionMethod: "aes-128-gcm",
					Plugin:           nil,
					PluginOpts:       nil,
				}
				return c
			}(),
			wantNode: &usecases.Node{
				ID:               2,
				EncryptionMethod: "aes-128-gcm",
				Plugin:           "",
				PluginOpts:       nil,
			},
		},
		{
			name:     "apply trojan config",
			node:     &usecases.Node{ID: 3},
			protocol: "trojan",
			nodeID:   3,
			configs: func() ProtocolConfigs {
				c := NewProtocolConfigs()
				c.Trojan[3] = &models.TrojanConfigModel{
					NodeID:            3,
					TransportProtocol: "grpc",
					Host:              "grpc.example.com",
					Path:              "/grpc",
					SNI:               "grpc-sni.example.com",
					AllowInsecure:     false,
				}
				return c
			}(),
			wantNode: &usecases.Node{
				ID:                3,
				TransportProtocol: "grpc",
				Host:              "grpc.example.com",
				Path:              "/grpc",
				SNI:               "grpc-sni.example.com",
				AllowInsecure:     false,
			},
		},
		{
			name:     "no config found for shadowsocks",
			node:     &usecases.Node{ID: 4},
			protocol: "shadowsocks",
			nodeID:   4,
			configs:  NewProtocolConfigs(),
			wantNode: &usecases.Node{
				ID: 4,
			},
		},
		{
			name:     "no config found for trojan defaults transport protocol",
			node:     &usecases.Node{ID: 5},
			protocol: "trojan",
			nodeID:   5,
			configs:  NewProtocolConfigs(),
			wantNode: &usecases.Node{
				ID:                5,
				TransportProtocol: "tcp",
			},
		},
		{
			name:     "empty protocol treated as shadowsocks",
			node:     &usecases.Node{ID: 6},
			protocol: "",
			nodeID:   6,
			configs: func() ProtocolConfigs {
				c := NewProtocolConfigs()
				c.Shadowsocks[6] = &models.ShadowsocksConfigModel{
					NodeID:           6,
					EncryptionMethod: "aes-256-gcm",
				}
				return c
			}(),
			wantNode: &usecases.Node{
				ID:               6,
				EncryptionMethod: "aes-256-gcm",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ApplyProtocolConfig(tt.node, tt.protocol, tt.nodeID, tt.configs)

			if tt.node.EncryptionMethod != tt.wantNode.EncryptionMethod {
				t.Errorf("EncryptionMethod = %q, want %q", tt.node.EncryptionMethod, tt.wantNode.EncryptionMethod)
			}
			if tt.node.Plugin != tt.wantNode.Plugin {
				t.Errorf("Plugin = %q, want %q", tt.node.Plugin, tt.wantNode.Plugin)
			}
			if tt.node.TransportProtocol != tt.wantNode.TransportProtocol {
				t.Errorf("TransportProtocol = %q, want %q", tt.node.TransportProtocol, tt.wantNode.TransportProtocol)
			}
			if tt.node.Host != tt.wantNode.Host {
				t.Errorf("Host = %q, want %q", tt.node.Host, tt.wantNode.Host)
			}
			if tt.node.Path != tt.wantNode.Path {
				t.Errorf("Path = %q, want %q", tt.node.Path, tt.wantNode.Path)
			}
			if tt.node.SNI != tt.wantNode.SNI {
				t.Errorf("SNI = %q, want %q", tt.node.SNI, tt.wantNode.SNI)
			}
			if tt.node.AllowInsecure != tt.wantNode.AllowInsecure {
				t.Errorf("AllowInsecure = %v, want %v", tt.node.AllowInsecure, tt.wantNode.AllowInsecure)
			}
		})
	}
}

func TestResolveServerAddress(t *testing.T) {
	tests := []struct {
		name           string
		configuredAddr string
		publicIPv4     *string
		publicIPv6     *string
		want           string
	}{
		{
			name:           "configured address takes priority",
			configuredAddr: "configured.example.com",
			publicIPv4:     strPtr("1.2.3.4"),
			publicIPv6:     strPtr("2001:db8::1"),
			want:           "configured.example.com",
		},
		{
			name:           "IPv4 fallback when no configured address",
			configuredAddr: "",
			publicIPv4:     strPtr("1.2.3.4"),
			publicIPv6:     strPtr("2001:db8::1"),
			want:           "1.2.3.4",
		},
		{
			name:           "IPv6 fallback when no configured address or IPv4",
			configuredAddr: "",
			publicIPv4:     nil,
			publicIPv6:     strPtr("2001:db8::1"),
			want:           "2001:db8::1",
		},
		{
			name:           "empty IPv4 falls back to IPv6",
			configuredAddr: "",
			publicIPv4:     strPtr(""),
			publicIPv6:     strPtr("2001:db8::1"),
			want:           "2001:db8::1",
		},
		{
			name:           "empty string when all addresses are empty",
			configuredAddr: "",
			publicIPv4:     nil,
			publicIPv6:     nil,
			want:           "",
		},
		{
			name:           "empty string when all addresses are empty strings",
			configuredAddr: "",
			publicIPv4:     strPtr(""),
			publicIPv6:     strPtr(""),
			want:           "",
		},
		{
			name:           "configured address even when empty string IPv4/IPv6",
			configuredAddr: "myserver.example.com",
			publicIPv4:     strPtr(""),
			publicIPv6:     strPtr(""),
			want:           "myserver.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveServerAddress(tt.configuredAddr, tt.publicIPv4, tt.publicIPv6)
			if got != tt.want {
				t.Errorf("ResolveServerAddress(%q, %v, %v) = %q, want %q",
					tt.configuredAddr, ptrStr(tt.publicIPv4), ptrStr(tt.publicIPv6), got, tt.want)
			}
		})
	}
}

func TestNodeModelToSource(t *testing.T) {
	tests := []struct {
		name string
		nm   *models.NodeModel
		want NodeSource
	}{
		{
			name: "convert with subscription port",
			nm: &models.NodeModel{
				ID:               1,
				Name:             "node1",
				ServerAddress:    "1.2.3.4",
				AgentPort:        8388,
				SubscriptionPort: uint16Ptr(443),
				Protocol:         "shadowsocks",
				TokenHash:        "hash1",
				SortOrder:        10,
				PublicIPv4:       strPtr("5.6.7.8"),
				PublicIPv6:       strPtr("2001:db8::1"),
			},
			want: NodeSource{
				ID:        1,
				Name:      "node1",
				Address:   "1.2.3.4",
				Port:      443,
				Protocol:  "shadowsocks",
				TokenHash: "hash1",
				SortOrder: 10,
			},
		},
		{
			name: "convert without subscription port uses agent port",
			nm: &models.NodeModel{
				ID:               2,
				Name:             "node2",
				ServerAddress:    "",
				AgentPort:        8388,
				SubscriptionPort: nil,
				Protocol:         "trojan",
				TokenHash:        "hash2",
				SortOrder:        20,
				PublicIPv4:       strPtr("10.0.0.1"),
				PublicIPv6:       nil,
			},
			want: NodeSource{
				ID:        2,
				Name:      "node2",
				Address:   "10.0.0.1",
				Port:      8388,
				Protocol:  "trojan",
				TokenHash: "hash2",
				SortOrder: 20,
			},
		},
		{
			name: "convert with server address configured",
			nm: &models.NodeModel{
				ID:               3,
				Name:             "node3",
				ServerAddress:    "configured.example.com",
				AgentPort:        1234,
				SubscriptionPort: nil,
				Protocol:         "shadowsocks",
				TokenHash:        "hash3",
				SortOrder:        30,
				PublicIPv4:       strPtr("9.9.9.9"),
				PublicIPv6:       strPtr("2001:db8::9"),
			},
			want: NodeSource{
				ID:        3,
				Name:      "node3",
				Address:   "configured.example.com",
				Port:      1234,
				Protocol:  "shadowsocks",
				TokenHash: "hash3",
				SortOrder: 30,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NodeModelToSource(tt.nm)

			if got.ID != tt.want.ID {
				t.Errorf("ID = %d, want %d", got.ID, tt.want.ID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.Address != tt.want.Address {
				t.Errorf("Address = %q, want %q", got.Address, tt.want.Address)
			}
			if got.Port != tt.want.Port {
				t.Errorf("Port = %d, want %d", got.Port, tt.want.Port)
			}
			if got.Protocol != tt.want.Protocol {
				t.Errorf("Protocol = %q, want %q", got.Protocol, tt.want.Protocol)
			}
			if got.TokenHash != tt.want.TokenHash {
				t.Errorf("TokenHash = %q, want %q", got.TokenHash, tt.want.TokenHash)
			}
			if got.SortOrder != tt.want.SortOrder {
				t.Errorf("SortOrder = %d, want %d", got.SortOrder, tt.want.SortOrder)
			}
		})
	}
}

func TestCopyProtocolFieldsFromNode(t *testing.T) {
	tests := []struct {
		name string
		src  *usecases.Node
		want *usecases.Node
	}{
		{
			name: "copy shadowsocks fields",
			src: &usecases.Node{
				ID:               1,
				EncryptionMethod: "aes-256-gcm",
				Plugin:           "obfs-local",
				PluginOpts:       map[string]string{"mode": "tls", "host": "example.com"},
			},
			want: &usecases.Node{
				EncryptionMethod: "aes-256-gcm",
				Plugin:           "obfs-local",
				PluginOpts:       map[string]string{"mode": "tls", "host": "example.com"},
			},
		},
		{
			name: "copy trojan fields",
			src: &usecases.Node{
				ID:                2,
				TransportProtocol: "ws",
				Host:              "ws.example.com",
				Path:              "/ws",
				SNI:               "sni.example.com",
				AllowInsecure:     true,
			},
			want: &usecases.Node{
				TransportProtocol: "ws",
				Host:              "ws.example.com",
				Path:              "/ws",
				SNI:               "sni.example.com",
				AllowInsecure:     true,
			},
		},
		{
			name: "copy all protocol fields",
			src: &usecases.Node{
				ID:                3,
				EncryptionMethod:  "chacha20-ietf-poly1305",
				Plugin:            "v2ray-plugin",
				PluginOpts:        map[string]string{"key": "value"},
				TransportProtocol: "grpc",
				Host:              "grpc.example.com",
				Path:              "/grpc",
				SNI:               "grpc-sni.example.com",
				AllowInsecure:     false,
			},
			want: &usecases.Node{
				EncryptionMethod:  "chacha20-ietf-poly1305",
				Plugin:            "v2ray-plugin",
				PluginOpts:        map[string]string{"key": "value"},
				TransportProtocol: "grpc",
				Host:              "grpc.example.com",
				Path:              "/grpc",
				SNI:               "grpc-sni.example.com",
				AllowInsecure:     false,
			},
		},
		{
			name: "copy empty fields",
			src:  &usecases.Node{ID: 4},
			want: &usecases.Node{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := &usecases.Node{}
			CopyProtocolFieldsFromNode(dst, tt.src)

			if dst.EncryptionMethod != tt.want.EncryptionMethod {
				t.Errorf("EncryptionMethod = %q, want %q", dst.EncryptionMethod, tt.want.EncryptionMethod)
			}
			if dst.Plugin != tt.want.Plugin {
				t.Errorf("Plugin = %q, want %q", dst.Plugin, tt.want.Plugin)
			}
			if dst.TransportProtocol != tt.want.TransportProtocol {
				t.Errorf("TransportProtocol = %q, want %q", dst.TransportProtocol, tt.want.TransportProtocol)
			}
			if dst.Host != tt.want.Host {
				t.Errorf("Host = %q, want %q", dst.Host, tt.want.Host)
			}
			if dst.Path != tt.want.Path {
				t.Errorf("Path = %q, want %q", dst.Path, tt.want.Path)
			}
			if dst.SNI != tt.want.SNI {
				t.Errorf("SNI = %q, want %q", dst.SNI, tt.want.SNI)
			}
			if dst.AllowInsecure != tt.want.AllowInsecure {
				t.Errorf("AllowInsecure = %v, want %v", dst.AllowInsecure, tt.want.AllowInsecure)
			}
			// Check PluginOpts map equality
			if len(dst.PluginOpts) != len(tt.want.PluginOpts) {
				t.Errorf("PluginOpts length = %d, want %d", len(dst.PluginOpts), len(tt.want.PluginOpts))
			}
			for k, v := range tt.want.PluginOpts {
				if dst.PluginOpts[k] != v {
					t.Errorf("PluginOpts[%q] = %q, want %q", k, dst.PluginOpts[k], v)
				}
			}
		})
	}
}

func TestParsePluginOpts(t *testing.T) {
	tests := []struct {
		name     string
		optsJSON []byte
		want     map[string]string
	}{
		{
			name:     "parse valid JSON",
			optsJSON: []byte(`{"mode": "tls", "host": "example.com"}`),
			want:     map[string]string{"mode": "tls", "host": "example.com"},
		},
		{
			name:     "parse empty JSON object",
			optsJSON: []byte(`{}`),
			want:     map[string]string{},
		},
		{
			name:     "parse invalid JSON returns empty map",
			optsJSON: []byte(`invalid`),
			want:     map[string]string{},
		},
		{
			name:     "parse nil returns empty map",
			optsJSON: nil,
			want:     map[string]string{},
		},
		{
			name:     "parse JSON with non-string values filters them out",
			optsJSON: []byte(`{"key1": "value1", "key2": 123, "key3": true}`),
			want:     map[string]string{"key1": "value1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePluginOpts(tt.optsJSON)

			if len(got) != len(tt.want) {
				t.Errorf("parsePluginOpts() length = %d, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parsePluginOpts()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

// Helper functions

func strPtr(s string) *string {
	return &s
}

func ptrStr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func uint16Ptr(v uint16) *uint16 {
	return &v
}
