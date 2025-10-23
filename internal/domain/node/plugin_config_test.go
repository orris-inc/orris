package node

import (
	"strings"
	"testing"
)

func TestNewObfsPlugin_ValidModes(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{"HTTP mode", "http"},
		{"TLS mode", "tls"},
		{"uppercase HTTP", "HTTP"},
		{"uppercase TLS", "TLS"},
		{"mixed case", "Http"},
		{"with whitespace", "  http  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewObfsPlugin(tt.mode)
			if err != nil {
				t.Errorf("NewObfsPlugin(%q) error = %v, want nil", tt.mode, err)
				return
			}
			if config.Plugin() != PluginObfs {
				t.Errorf("Plugin() = %q, want %q", config.Plugin(), PluginObfs)
			}
			if !config.IsObfs() {
				t.Error("IsObfs() = false, want true")
			}
			if config.IsV2Ray() {
				t.Error("IsV2Ray() = true, want false")
			}
		})
	}
}

func TestNewObfsPlugin_InvalidModes(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{"invalid mode", "invalid"},
		{"empty mode", ""},
		{"whitespace only", "   "},
		{"websocket", "websocket"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewObfsPlugin(tt.mode)
			if err == nil {
				t.Errorf("NewObfsPlugin(%q) error = nil, want error", tt.mode)
			}
		})
	}
}

func TestNewObfsPluginWithHost(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		host         string
		expectHost   bool
		expectedHost string
	}{
		{"HTTP with host", "http", "example.com", true, "example.com"},
		{"TLS with host", "tls", "api.example.com", true, "api.example.com"},
		{"HTTP without host", "http", "", false, ""},
		{"TLS with empty host", "tls", "", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewObfsPluginWithHost(tt.mode, tt.host)
			if err != nil {
				t.Errorf("NewObfsPluginWithHost(%q, %q) error = %v", tt.mode, tt.host, err)
				return
			}

			if tt.expectHost {
				host, ok := config.GetOpt("obfs-host")
				if !ok {
					t.Error("GetOpt(\"obfs-host\") returned false, want true")
				}
				if host != tt.expectedHost {
					t.Errorf("GetOpt(\"obfs-host\") = %q, want %q", host, tt.expectedHost)
				}
			} else {
				_, ok := config.GetOpt("obfs-host")
				if ok {
					t.Error("GetOpt(\"obfs-host\") returned true, want false")
				}
			}

			obfsMode, ok := config.GetOpt("obfs")
			if !ok {
				t.Error("GetOpt(\"obfs\") returned false, want true")
			}
			if obfsMode != strings.ToLower(tt.mode) {
				t.Errorf("GetOpt(\"obfs\") = %q, want %q", obfsMode, strings.ToLower(tt.mode))
			}
		})
	}
}

func TestNewV2RayPlugin_ValidModes(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{"websocket mode", "websocket"},
		{"QUIC mode", "quic"},
		{"uppercase websocket", "WEBSOCKET"},
		{"uppercase QUIC", "QUIC"},
		{"mixed case", "WebSocket"},
		{"with whitespace", "  websocket  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewV2RayPlugin(tt.mode)
			if err != nil {
				t.Errorf("NewV2RayPlugin(%q) error = %v, want nil", tt.mode, err)
				return
			}
			if config.Plugin() != PluginV2Ray {
				t.Errorf("Plugin() = %q, want %q", config.Plugin(), PluginV2Ray)
			}
			if !config.IsV2Ray() {
				t.Error("IsV2Ray() = false, want true")
			}
			if config.IsObfs() {
				t.Error("IsObfs() = true, want false")
			}
		})
	}
}

func TestNewV2RayPlugin_InvalidModes(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{"invalid mode", "invalid"},
		{"empty mode", ""},
		{"whitespace only", "   "},
		{"http", "http"},
		{"tls", "tls"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewV2RayPlugin(tt.mode)
			if err == nil {
				t.Errorf("NewV2RayPlugin(%q) error = nil, want error", tt.mode)
			}
		})
	}
}

func TestNewV2RayPluginWithHost(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		host         string
		expectHost   bool
		expectedHost string
	}{
		{"websocket with host", "websocket", "example.com", true, "example.com"},
		{"QUIC with host", "quic", "api.example.com", true, "api.example.com"},
		{"websocket without host", "websocket", "", false, ""},
		{"QUIC with empty host", "quic", "", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewV2RayPluginWithHost(tt.mode, tt.host)
			if err != nil {
				t.Errorf("NewV2RayPluginWithHost(%q, %q) error = %v", tt.mode, tt.host, err)
				return
			}

			if tt.expectHost {
				host, ok := config.GetOpt("host")
				if !ok {
					t.Error("GetOpt(\"host\") returned false, want true")
				}
				if host != tt.expectedHost {
					t.Errorf("GetOpt(\"host\") = %q, want %q", host, tt.expectedHost)
				}
			} else {
				_, ok := config.GetOpt("host")
				if ok {
					t.Error("GetOpt(\"host\") returned true, want false")
				}
			}

			mode, ok := config.GetOpt("mode")
			if !ok {
				t.Error("GetOpt(\"mode\") returned false, want true")
			}
			if mode != strings.ToLower(tt.mode) {
				t.Errorf("GetOpt(\"mode\") = %q, want %q", mode, strings.ToLower(tt.mode))
			}
		})
	}
}

func TestNewPluginConfig(t *testing.T) {
	tests := []struct {
		name    string
		plugin  string
		opts    map[string]string
		wantErr bool
	}{
		{"obfs plugin", "obfs-local", map[string]string{"obfs": "http"}, false},
		{"v2ray plugin", "v2ray-plugin", map[string]string{"mode": "websocket"}, false},
		{"nil opts", "obfs-local", nil, false},
		{"unsupported plugin", "unsupported", map[string]string{}, true},
		{"empty plugin", "", map[string]string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewPluginConfig(tt.plugin, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPluginConfig(%q, _) error = %v, wantErr %v", tt.plugin, err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("NewPluginConfig() returned nil config without error")
			}
		})
	}
}

func TestPluginConfig_ToPluginOpts(t *testing.T) {
	tests := []struct {
		name     string
		opts     map[string]string
		contains []string
	}{
		{
			"single option",
			map[string]string{"obfs": "http"},
			[]string{"obfs=http"},
		},
		{
			"multiple options",
			map[string]string{"obfs": "http", "obfs-host": "example.com"},
			[]string{"obfs=http", "obfs-host=example.com"},
		},
		{
			"empty options",
			map[string]string{},
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewPluginConfig("obfs-local", tt.opts)
			if err != nil {
				t.Fatalf("NewPluginConfig() error = %v", err)
			}

			result := config.ToPluginOpts()

			if len(tt.contains) == 0 && result != "" {
				t.Errorf("ToPluginOpts() = %q, want empty string", result)
				return
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("ToPluginOpts() = %q, should contain %q", result, expected)
				}
			}
		})
	}
}

func TestPluginConfig_Opts(t *testing.T) {
	originalOpts := map[string]string{
		"obfs":      "http",
		"obfs-host": "example.com",
	}

	config, err := NewPluginConfig("obfs-local", originalOpts)
	if err != nil {
		t.Fatalf("NewPluginConfig() error = %v", err)
	}

	opts := config.Opts()

	if len(opts) != len(originalOpts) {
		t.Errorf("Opts() returned %d options, want %d", len(opts), len(originalOpts))
	}

	for k, v := range originalOpts {
		if opts[k] != v {
			t.Errorf("Opts()[%q] = %q, want %q", k, opts[k], v)
		}
	}

	opts["new-key"] = "new-value"
	if _, ok := config.GetOpt("new-key"); ok {
		t.Error("modifying returned Opts() affected internal state")
	}
}

func TestPluginConfig_Equals(t *testing.T) {
	config1, _ := NewPluginConfig("obfs-local", map[string]string{"obfs": "http"})
	config2, _ := NewPluginConfig("obfs-local", map[string]string{"obfs": "http"})
	config3, _ := NewPluginConfig("v2ray-plugin", map[string]string{"mode": "websocket"})
	config4, _ := NewPluginConfig("obfs-local", map[string]string{"obfs": "tls"})
	config5, _ := NewPluginConfig("obfs-local", map[string]string{"obfs": "http", "obfs-host": "example.com"})

	tests := []struct {
		name     string
		config1  *PluginConfig
		config2  *PluginConfig
		expected bool
	}{
		{"same configs", config1, config2, true},
		{"different plugins", config1, config3, false},
		{"different opts", config1, config4, false},
		{"different number of opts", config1, config5, false},
		{"nil equals nil", nil, nil, true},
		{"nil vs non-nil", nil, config1, false},
		{"non-nil vs nil", config1, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config1.Equals(tt.config2)
			if result != tt.expected {
				t.Errorf("Equals() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetSupportedPlugins(t *testing.T) {
	plugins := GetSupportedPlugins()

	if len(plugins) == 0 {
		t.Error("GetSupportedPlugins() returned empty slice")
	}

	expectedPlugins := map[string]bool{
		"obfs-local":   true,
		"v2ray-plugin": true,
	}

	for _, plugin := range plugins {
		if !expectedPlugins[plugin] {
			t.Errorf("GetSupportedPlugins() contains unexpected plugin: %s", plugin)
		}
	}

	for plugin := range expectedPlugins {
		found := false
		for _, p := range plugins {
			if p == plugin {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetSupportedPlugins() missing expected plugin: %s", plugin)
		}
	}
}

func TestPluginConfig_GetOpt(t *testing.T) {
	opts := map[string]string{
		"obfs":      "http",
		"obfs-host": "example.com",
	}

	config, err := NewPluginConfig("obfs-local", opts)
	if err != nil {
		t.Fatalf("NewPluginConfig() error = %v", err)
	}

	tests := []struct {
		name      string
		key       string
		wantValue string
		wantOk    bool
	}{
		{"existing key", "obfs", "http", true},
		{"another existing key", "obfs-host", "example.com", true},
		{"non-existing key", "non-existent", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok := config.GetOpt(tt.key)
			if ok != tt.wantOk {
				t.Errorf("GetOpt(%q) ok = %v, want %v", tt.key, ok, tt.wantOk)
			}
			if value != tt.wantValue {
				t.Errorf("GetOpt(%q) value = %q, want %q", tt.key, value, tt.wantValue)
			}
		})
	}
}
