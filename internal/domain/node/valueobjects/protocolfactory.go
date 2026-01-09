package valueobjects

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
// The password parameter should be the subscription UUID
func (sc ShadowsocksProtocolConfig) ToSubscriptionURI(serverAddr string, serverPort uint16, password string, remarks string) string {
	// Format: ss://base64(method:password)@server:port#remarks
	auth := sc.encryption.ToShadowsocksURI(password)
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

// VLESSProtocolConfig wraps VLESSConfig for VLESS protocol
type VLESSProtocolConfig struct {
	config VLESSConfig
}

// NewVLESSProtocolConfig creates a new VLESS protocol configuration
func NewVLESSProtocolConfig(config VLESSConfig) VLESSProtocolConfig {
	return VLESSProtocolConfig{
		config: config,
	}
}

// Config returns the VLESS configuration
func (vc VLESSProtocolConfig) Config() VLESSConfig {
	return vc.config
}

// String returns a string representation of the VLESS config
func (vc VLESSProtocolConfig) String() string {
	return vc.config.String()
}

// ToSubscriptionURI generates a subscription URI for VLESS
func (vc VLESSProtocolConfig) ToSubscriptionURI(serverAddr string, serverPort uint16, uuid string, remarks string) string {
	return vc.config.ToURI(uuid, serverAddr, serverPort, remarks)
}

// VMessProtocolConfig wraps VMessConfig for VMess protocol
type VMessProtocolConfig struct {
	config VMessConfig
}

// NewVMessProtocolConfig creates a new VMess protocol configuration
func NewVMessProtocolConfig(config VMessConfig) VMessProtocolConfig {
	return VMessProtocolConfig{
		config: config,
	}
}

// Config returns the VMess configuration
func (vm VMessProtocolConfig) Config() VMessConfig {
	return vm.config
}

// String returns a string representation of the VMess config
func (vm VMessProtocolConfig) String() string {
	return vm.config.String()
}

// ToSubscriptionURI generates a subscription URI for VMess
func (vm VMessProtocolConfig) ToSubscriptionURI(serverAddr string, serverPort uint16, uuid string, remarks string) (string, error) {
	return vm.config.ToURI(serverAddr, serverPort, uuid, remarks)
}

// Hysteria2ProtocolConfig wraps Hysteria2Config for Hysteria2 protocol
type Hysteria2ProtocolConfig struct {
	config Hysteria2Config
}

// NewHysteria2ProtocolConfig creates a new Hysteria2 protocol configuration
func NewHysteria2ProtocolConfig(config Hysteria2Config) Hysteria2ProtocolConfig {
	return Hysteria2ProtocolConfig{
		config: config,
	}
}

// Config returns the Hysteria2 configuration
func (hc Hysteria2ProtocolConfig) Config() Hysteria2Config {
	return hc.config
}

// String returns a string representation of the Hysteria2 config
func (hc Hysteria2ProtocolConfig) String() string {
	return hc.config.String()
}

// ToSubscriptionURI generates a subscription URI for Hysteria2
// Note: password is already stored in the config, this method ignores the password parameter
func (hc Hysteria2ProtocolConfig) ToSubscriptionURI(serverAddr string, serverPort uint16, remarks string) string {
	return hc.config.ToURI(serverAddr, serverPort, remarks)
}

// TUICProtocolConfig wraps TUICConfig for TUIC protocol
type TUICProtocolConfig struct {
	config TUICConfig
}

// NewTUICProtocolConfig creates a new TUIC protocol configuration
func NewTUICProtocolConfig(config TUICConfig) TUICProtocolConfig {
	return TUICProtocolConfig{
		config: config,
	}
}

// Config returns the TUIC configuration
func (tc TUICProtocolConfig) Config() TUICConfig {
	return tc.config
}

// String returns a string representation of the TUIC config
func (tc TUICProtocolConfig) String() string {
	return tc.config.String()
}

// ToSubscriptionURI generates a subscription URI for TUIC
// Note: uuid and password are already stored in the config
func (tc TUICProtocolConfig) ToSubscriptionURI(serverAddr string, serverPort uint16, remarks string) string {
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
	plugin *PluginConfig,
) (ShadowsocksProtocolConfig, error) {
	encryption, err := NewEncryptionConfig(method)
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

// CreateVLESSConfig creates a VLESS protocol configuration
func (f *ProtocolConfigFactory) CreateVLESSConfig(
	transportType string,
	flow string,
	security string,
	sni string,
	fingerprint string,
	allowInsecure bool,
	host string,
	path string,
	serviceName string,
	realityPublicKey string,
	realityShortID string,
	realitySpiderX string,
) (VLESSProtocolConfig, error) {
	config, err := NewVLESSConfig(
		transportType, flow, security, sni, fingerprint, allowInsecure,
		host, path, serviceName,
		realityPublicKey, realityShortID, realitySpiderX,
	)
	if err != nil {
		return VLESSProtocolConfig{}, fmt.Errorf("failed to create VLESS config: %w", err)
	}
	return NewVLESSProtocolConfig(config), nil
}

// CreateVMessConfig creates a VMess protocol configuration
func (f *ProtocolConfigFactory) CreateVMessConfig(
	alterID int,
	security string,
	transportType string,
	host string,
	path string,
	serviceName string,
	tls bool,
	sni string,
	allowInsecure bool,
) (VMessProtocolConfig, error) {
	config, err := NewVMessConfig(
		alterID, security, transportType, host, path, serviceName, tls, sni, allowInsecure,
	)
	if err != nil {
		return VMessProtocolConfig{}, fmt.Errorf("failed to create VMess config: %w", err)
	}
	return NewVMessProtocolConfig(config), nil
}

// CreateHysteria2Config creates a Hysteria2 protocol configuration
// Note: password is passed as a placeholder since it's derived from subscription UUID
func (f *ProtocolConfigFactory) CreateHysteria2Config(
	password string,
	congestionControl string,
	obfs string,
	obfsPassword string,
	upMbps *int,
	downMbps *int,
	sni string,
	allowInsecure bool,
	fingerprint string,
) (Hysteria2ProtocolConfig, error) {
	config, err := NewHysteria2Config(
		password, congestionControl, obfs, obfsPassword, upMbps, downMbps, sni, allowInsecure, fingerprint,
	)
	if err != nil {
		return Hysteria2ProtocolConfig{}, fmt.Errorf("failed to create Hysteria2 config: %w", err)
	}
	return NewHysteria2ProtocolConfig(config), nil
}

// CreateTUICConfig creates a TUIC protocol configuration
// Note: uuid and password are passed as placeholders since they're derived from subscription UUID
func (f *ProtocolConfigFactory) CreateTUICConfig(
	uuid string,
	password string,
	congestionControl string,
	udpRelayMode string,
	alpn string,
	sni string,
	allowInsecure bool,
	disableSNI bool,
) (TUICProtocolConfig, error) {
	config, err := NewTUICConfig(
		uuid, password, congestionControl, udpRelayMode, alpn, sni, allowInsecure, disableSNI,
	)
	if err != nil {
		return TUICProtocolConfig{}, fmt.Errorf("failed to create TUIC config: %w", err)
	}
	return NewTUICProtocolConfig(config), nil
}

// GenerateSubscriptionURI generates a subscription URI based on protocol type
// The password parameter is used as the authentication credential (subscription UUID for Shadowsocks)
func (f *ProtocolConfigFactory) GenerateSubscriptionURI(
	protocol Protocol,
	config ProtocolConfig,
	serverAddr string,
	serverPort uint16,
	password string,
	remarks string,
) (string, error) {
	if !protocol.IsValid() {
		return "", fmt.Errorf("invalid protocol: %s", protocol)
	}

	switch protocol {
	case ProtocolShadowsocks:
		if ssConfig, ok := config.(ShadowsocksProtocolConfig); ok {
			return ssConfig.ToSubscriptionURI(serverAddr, serverPort, password, remarks), nil
		}
		return "", fmt.Errorf("invalid config type for Shadowsocks protocol")

	case ProtocolTrojan:
		if trojanConfig, ok := config.(TrojanProtocolConfig); ok {
			return trojanConfig.ToSubscriptionURI(serverAddr, serverPort, remarks), nil
		}
		return "", fmt.Errorf("invalid config type for Trojan protocol")

	case ProtocolVLESS:
		if vlessConfig, ok := config.(VLESSProtocolConfig); ok {
			return vlessConfig.ToSubscriptionURI(serverAddr, serverPort, password, remarks), nil
		}
		return "", fmt.Errorf("invalid config type for VLESS protocol")

	case ProtocolVMess:
		if vmessConfig, ok := config.(VMessProtocolConfig); ok {
			return vmessConfig.ToSubscriptionURI(serverAddr, serverPort, password, remarks)
		}
		return "", fmt.Errorf("invalid config type for VMess protocol")

	case ProtocolHysteria2:
		if hysteria2Config, ok := config.(Hysteria2ProtocolConfig); ok {
			return hysteria2Config.ToSubscriptionURI(serverAddr, serverPort, remarks), nil
		}
		return "", fmt.Errorf("invalid config type for Hysteria2 protocol")

	case ProtocolTUIC:
		if tuicConfig, ok := config.(TUICProtocolConfig); ok {
			return tuicConfig.ToSubscriptionURI(serverAddr, serverPort, remarks), nil
		}
		return "", fmt.Errorf("invalid config type for TUIC protocol")

	default:
		return "", fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
