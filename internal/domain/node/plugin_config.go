package node

import (
	"fmt"
	"strings"
)

const (
	PluginObfs      = "obfs-local"
	PluginV2Ray     = "v2ray-plugin"
	PluginObfsHTTP  = "http"
	PluginObfsTLS   = "tls"
	PluginV2RayWS   = "websocket"
	PluginV2RayQUIC = "quic"
)

var supportedPlugins = map[string]bool{
	PluginObfs:  true,
	PluginV2Ray: true,
}

type PluginConfig struct {
	plugin string
	opts   map[string]string
}

func NewObfsPlugin(mode string) (*PluginConfig, error) {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))

	if normalizedMode != PluginObfsHTTP && normalizedMode != PluginObfsTLS {
		return nil, fmt.Errorf("invalid obfs mode: %s (must be http or tls)", mode)
	}

	opts := map[string]string{
		"obfs": normalizedMode,
	}

	return &PluginConfig{
		plugin: PluginObfs,
		opts:   opts,
	}, nil
}

func NewObfsPluginWithHost(mode, host string) (*PluginConfig, error) {
	pc, err := NewObfsPlugin(mode)
	if err != nil {
		return nil, err
	}

	if host != "" {
		pc.opts["obfs-host"] = host
	}

	return pc, nil
}

func NewV2RayPlugin(mode string) (*PluginConfig, error) {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))

	if normalizedMode != PluginV2RayWS && normalizedMode != PluginV2RayQUIC {
		return nil, fmt.Errorf("invalid v2ray mode: %s (must be websocket or quic)", mode)
	}

	opts := map[string]string{
		"mode": normalizedMode,
	}

	return &PluginConfig{
		plugin: PluginV2Ray,
		opts:   opts,
	}, nil
}

func NewV2RayPluginWithHost(mode, host string) (*PluginConfig, error) {
	pc, err := NewV2RayPlugin(mode)
	if err != nil {
		return nil, err
	}

	if host != "" {
		pc.opts["host"] = host
	}

	return pc, nil
}

func NewPluginConfig(plugin string, opts map[string]string) (*PluginConfig, error) {
	normalizedPlugin := strings.ToLower(strings.TrimSpace(plugin))

	if !supportedPlugins[normalizedPlugin] {
		return nil, fmt.Errorf("unsupported plugin: %s", plugin)
	}

	if opts == nil {
		opts = make(map[string]string)
	}

	return &PluginConfig{
		plugin: normalizedPlugin,
		opts:   opts,
	}, nil
}

func (pc *PluginConfig) Plugin() string {
	return pc.plugin
}

func (pc *PluginConfig) Opts() map[string]string {
	optsCopy := make(map[string]string, len(pc.opts))
	for k, v := range pc.opts {
		optsCopy[k] = v
	}
	return optsCopy
}

func (pc *PluginConfig) ToPluginOpts() string {
	var parts []string
	for k, v := range pc.opts {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ";")
}

func (pc *PluginConfig) IsObfs() bool {
	return pc.plugin == PluginObfs
}

func (pc *PluginConfig) IsV2Ray() bool {
	return pc.plugin == PluginV2Ray
}

func (pc *PluginConfig) GetOpt(key string) (string, bool) {
	val, ok := pc.opts[key]
	return val, ok
}

func (pc *PluginConfig) Equals(other *PluginConfig) bool {
	if pc == nil || other == nil {
		return pc == other
	}

	if pc.plugin != other.plugin {
		return false
	}

	if len(pc.opts) != len(other.opts) {
		return false
	}

	for k, v := range pc.opts {
		if otherV, ok := other.opts[k]; !ok || v != otherV {
			return false
		}
	}

	return true
}

func GetSupportedPlugins() []string {
	plugins := make([]string, 0, len(supportedPlugins))
	for plugin := range supportedPlugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}
