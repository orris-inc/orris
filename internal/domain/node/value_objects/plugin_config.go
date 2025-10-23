package value_objects

import (
	"fmt"
	"strings"
)

type PluginConfig struct {
	plugin string
	opts   map[string]string
}

func NewObfsPlugin(mode string) (*PluginConfig, error) {
	if mode != "http" && mode != "tls" {
		return nil, fmt.Errorf("invalid obfs mode: %s (must be 'http' or 'tls')", mode)
	}

	return &PluginConfig{
		plugin: "obfs-local",
		opts: map[string]string{
			"obfs": mode,
		},
	}, nil
}

func NewV2RayPlugin(mode string, host string) (*PluginConfig, error) {
	if mode != "websocket" && mode != "quic" {
		return nil, fmt.Errorf("invalid v2ray-plugin mode: %s (must be 'websocket' or 'quic')", mode)
	}

	if host == "" {
		return nil, fmt.Errorf("host is required for v2ray-plugin")
	}

	return &PluginConfig{
		plugin: "v2ray-plugin",
		opts: map[string]string{
			"mode": mode,
			"host": host,
		},
	}, nil
}

func NewPluginConfig(plugin string, opts map[string]string) (*PluginConfig, error) {
	if plugin == "" {
		return nil, fmt.Errorf("plugin name cannot be empty")
	}

	if opts == nil {
		opts = make(map[string]string)
	}

	return &PluginConfig{
		plugin: plugin,
		opts:   opts,
	}, nil
}

func (pc *PluginConfig) Plugin() string {
	return pc.plugin
}

func (pc *PluginConfig) Opts() map[string]string {
	return pc.opts
}

func (pc *PluginConfig) ToPluginOpts() string {
	var parts []string
	for k, v := range pc.opts {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ";")
}

func (pc *PluginConfig) Equals(other *PluginConfig) bool {
	if pc == nil && other == nil {
		return true
	}
	if pc == nil || other == nil {
		return false
	}
	if pc.plugin != other.plugin {
		return false
	}
	if len(pc.opts) != len(other.opts) {
		return false
	}
	for k, v := range pc.opts {
		if other.opts[k] != v {
			return false
		}
	}
	return true
}
