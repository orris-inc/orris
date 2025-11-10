package value_objects

import (
	"fmt"
)

// ProtocolConfig represents a generic protocol configuration interface
// This interface allows different protocol configs to be used polymorphically
type ProtocolConfig interface {
	// String returns a string representation of the configuration
	String() string
}

// ShadowsocksProtocolConfig wraps EncryptionConfig for Shadowsocks protocol
type ShadowsocksProtocolConfig struct {
	encryption EncryptionConfig
	plugin     *PluginConfig
}

// NewShadowsocksProtocolConfig creates a new Shadowsocks protocol configuration
func NewShadowsocksProtocolConfig(encryption EncryptionConfig, plugin *PluginConfig) ShadowsocksProtocolConfig {
	return ShadowsocksProtocolConfig{
		encryption: encryption,
		plugin:     plugin,
	}
}

// Encryption returns the encryption configuration
func (sc ShadowsocksProtocolConfig) Encryption() EncryptionConfig {
	return sc.encryption
}

// Plugin returns the plugin configuration
func (sc ShadowsocksProtocolConfig) Plugin() *PluginConfig {
	return sc.plugin
}

// String returns a string representation of the Shadowsocks config
func (sc ShadowsocksProtocolConfig) String() string {
	result := fmt.Sprintf("method=%s", sc.encryption.Method())
	if sc.plugin != nil {
		result += fmt.Sprintf(", plugin=%s", sc.plugin.Plugin())
	}
	return result
}

// ToSubscriptionURI generates a subscription URI for Shadowsocks
func (sc ShadowsocksProtocolConfig) ToSubscriptionURI(serverAddr string, serverPort uint16, remarks string) string {
	// Format: ss://base64(method:password)@server:port#remarks
	auth := sc.encryption.ToShadowsocksURI()
	uri := fmt.Sprintf("ss://%s@%s:%d", auth, serverAddr, serverPort)

	if sc.plugin != nil {
		uri += fmt.Sprintf("?plugin=%s;%s", sc.plugin.Plugin(), sc.plugin.ToPluginOpts())
	}

	if remarks != "" {
		uri += "#" + remarks
	}

	return uri
}

// TrojanProtocolConfig wraps TrojanConfig for Trojan protocol
type TrojanProtocolConfig struct {
	config TrojanConfig
}

// NewTrojanProtocolConfig creates a new Trojan protocol configuration
func NewTrojanProtocolConfig(config TrojanConfig) TrojanProtocolConfig {
	return TrojanProtocolConfig{
		config: config,
	}
}

// Config returns the Trojan configuration
func (tc TrojanProtocolConfig) Config() TrojanConfig {
	return tc.config
}

// String returns a string representation of the Trojan config
func (tc TrojanProtocolConfig) String() string {
	return tc.config.String()
}

// ToSubscriptionURI generates a subscription URI for Trojan
func (tc TrojanProtocolConfig) ToSubscriptionURI(serverAddr string, serverPort uint16, remarks string) string {
	return tc.config.ToURI(serverAddr, serverPort, remarks)
}

// ProtocolConfigFactory creates protocol configurations based on protocol type
type ProtocolConfigFactory struct{}

// NewProtocolConfigFactory creates a new protocol config factory
func NewProtocolConfigFactory() *ProtocolConfigFactory {
	return &ProtocolConfigFactory{}
}

// CreateShadowsocksConfig creates a Shadowsocks protocol configuration
func (f *ProtocolConfigFactory) CreateShadowsocksConfig(
	method string,
	password string,
	plugin *PluginConfig,
) (ShadowsocksProtocolConfig, error) {
	encryption, err := NewEncryptionConfig(method, password)
	if err != nil {
		return ShadowsocksProtocolConfig{}, fmt.Errorf("failed to create encryption config: %w", err)
	}

	return NewShadowsocksProtocolConfig(encryption, plugin), nil
}

// CreateTrojanConfig creates a Trojan protocol configuration
func (f *ProtocolConfigFactory) CreateTrojanConfig(
	password string,
	transportProtocol string,
	host string,
	path string,
	allowInsecure bool,
	sni string,
) (TrojanProtocolConfig, error) {
	config, err := NewTrojanConfig(password, transportProtocol, host, path, allowInsecure, sni)
	if err != nil {
		return TrojanProtocolConfig{}, fmt.Errorf("failed to create trojan config: %w", err)
	}

	return NewTrojanProtocolConfig(config), nil
}

// GenerateSubscriptionURI generates a subscription URI based on protocol type
func (f *ProtocolConfigFactory) GenerateSubscriptionURI(
	protocol Protocol,
	config ProtocolConfig,
	serverAddr string,
	serverPort uint16,
	remarks string,
) (string, error) {
	if !protocol.IsValid() {
		return "", fmt.Errorf("invalid protocol: %s", protocol)
	}

	switch protocol {
	case ProtocolShadowsocks:
		if ssConfig, ok := config.(ShadowsocksProtocolConfig); ok {
			return ssConfig.ToSubscriptionURI(serverAddr, serverPort, remarks), nil
		}
		return "", fmt.Errorf("invalid config type for Shadowsocks protocol")

	case ProtocolTrojan:
		if trojanConfig, ok := config.(TrojanProtocolConfig); ok {
			return trojanConfig.ToSubscriptionURI(serverAddr, serverPort, remarks), nil
		}
		return "", fmt.Errorf("invalid config type for Trojan protocol")

	default:
		return "", fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
