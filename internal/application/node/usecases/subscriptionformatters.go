package usecases

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"gopkg.in/yaml.v3"
)

// adjustPasswordForMethod adjusts password format based on encryption method.
// The input password is expected to be hex-encoded 32-byte HMAC key material.
// For SS2022 methods, it converts to base64 with proper key length and combines with serverKey.
// For traditional SS methods, it keeps the hex format.
func adjustPasswordForMethod(password string, method string, tokenHash string) string {
	if password == "" {
		return password
	}

	// Traditional SS: keep hex format
	if !vo.IsSS2022Method(method) {
		return password
	}

	// SS2022: convert hex to base64 with proper key length
	keyMaterial, err := hex.DecodeString(password)
	if err != nil {
		return password
	}

	keySize := vo.GetSS2022KeySize(method)
	if keySize == 0 || keySize > len(keyMaterial) {
		return password
	}

	// Generate user key (base64)
	userKey := base64.StdEncoding.EncodeToString(keyMaterial[:keySize])

	// Generate server key from tokenHash
	serverKey := vo.GenerateSS2022ServerKey(tokenHash, method)
	if serverKey == "" {
		// Fallback to user key only if no server key available
		return userKey
	}

	// SS2022 password format: serverKey:userKey
	return serverKey + ":" + userKey
}

type Base64Formatter struct{}

func NewBase64Formatter() *Base64Formatter {
	return &Base64Formatter{}
}

func (f *Base64Formatter) Format(nodes []*Node) (string, error) {
	return f.FormatWithPassword(nodes, "")
}

func (f *Base64Formatter) FormatWithPassword(nodes []*Node, password string) (string, error) {
	var links []string

	for _, node := range nodes {
		var link string

		switch vo.Protocol(node.Protocol) {
		case vo.ProtocolTrojan:
			link = node.ToTrojanURI(password)
		case vo.ProtocolVLESS:
			if node.VLESSConfig != nil {
				// Use password as UUID for VLESS
				link = node.VLESSConfig.ToURI(password, node.ServerAddress, node.SubscriptionPort, node.Name)
			}
		case vo.ProtocolVMess:
			if node.VMessConfig != nil {
				// Use password as UUID for VMess
				uri, err := node.VMessConfig.ToURI(node.ServerAddress, node.SubscriptionPort, password, node.Name)
				if err == nil {
					link = uri
				}
			}
		case vo.ProtocolHysteria2:
			if node.Hysteria2Config != nil {
				// Use password derived from subscription UUID
				link = node.Hysteria2Config.ToURI(node.ServerAddress, node.SubscriptionPort, node.Name, password)
			}
		case vo.ProtocolTUIC:
			if node.TUICConfig != nil {
				// For TUIC, use password as both uuid and password (derived from subscription)
				link = node.TUICConfig.ToURI(node.ServerAddress, node.SubscriptionPort, node.Name, password, password)
			}
		default:
			// Shadowsocks: adjust password for SS2022 methods
			nodePassword := adjustPasswordForMethod(password, node.EncryptionMethod, node.TokenHash)

			// Shadowsocks URI format: ss://base64(method:password)@host:port#remarks
			auth := fmt.Sprintf("%s:%s", node.EncryptionMethod, nodePassword)
			authEncoded := base64.StdEncoding.EncodeToString([]byte(auth))

			link = fmt.Sprintf("ss://%s@%s:%d",
				authEncoded,
				node.ServerAddress,
				node.SubscriptionPort)

			if node.Plugin != "" {
				pluginOpts := formatPluginOpts(node.PluginOpts)
				link += fmt.Sprintf("?plugin=%s;%s",
					url.QueryEscape(node.Plugin),
					url.QueryEscape(pluginOpts))
			}

			if node.Name != "" {
				link += "#" + url.QueryEscape(node.Name)
			}
		}

		if link != "" {
			links = append(links, link)
		}
	}

	content := strings.Join(links, "\n")
	return base64.StdEncoding.EncodeToString([]byte(content)), nil
}

func (f *Base64Formatter) ContentType() string {
	return "text/plain; charset=utf-8"
}

type ClashFormatter struct{}

func NewClashFormatter() *ClashFormatter {
	return &ClashFormatter{}
}

type clashProxy struct {
	Name           string            `yaml:"name"`
	Type           string            `yaml:"type"`
	Server         string            `yaml:"server"`
	Port           uint16            `yaml:"port"`
	Cipher         string            `yaml:"cipher,omitempty"`
	Password       string            `yaml:"password,omitempty"`
	UDP            bool              `yaml:"udp,omitempty"`
	Plugin         string            `yaml:"plugin,omitempty"`
	PluginOpts     map[string]string `yaml:"plugin-opts,omitempty"`
	SNI            string            `yaml:"sni,omitempty"`
	SkipCertVerify bool              `yaml:"skip-cert-verify,omitempty"`
	Network        string            `yaml:"network,omitempty"`
	WSOpts         *clashWSOpts      `yaml:"ws-opts,omitempty"`
	GRPCOpts       *clashGRPCOpts    `yaml:"grpc-opts,omitempty"`
	H2Opts         *clashH2Opts      `yaml:"h2-opts,omitempty"`
	// VLESS/VMess specific fields
	UUID        string            `yaml:"uuid,omitempty"`
	Flow        string            `yaml:"flow,omitempty"`
	TLS         bool              `yaml:"tls,omitempty"`
	Fingerprint string            `yaml:"client-fingerprint,omitempty"`
	AlterID     int               `yaml:"alterId,omitempty"`
	RealityOpts *clashRealityOpts `yaml:"reality-opts,omitempty"`
	// Hysteria2 specific fields
	Obfs         string `yaml:"obfs,omitempty"`
	ObfsPassword string `yaml:"obfs-password,omitempty"`
	Up           string `yaml:"up,omitempty"`
	Down         string `yaml:"down,omitempty"`
	// TUIC specific fields
	CongestionController string   `yaml:"congestion-controller,omitempty"`
	UDPRelayMode         string   `yaml:"udp-relay-mode,omitempty"`
	ALPN                 []string `yaml:"alpn,omitempty"`
	DisableSNI           bool     `yaml:"disable-sni,omitempty"`
}

type clashWSOpts struct {
	Path    string            `yaml:"path,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

type clashH2Opts struct {
	Host []string `yaml:"host,omitempty"`
	Path string   `yaml:"path,omitempty"`
}

type clashGRPCOpts struct {
	GRPCServiceName string `yaml:"grpc-service-name,omitempty"`
}

type clashRealityOpts struct {
	PublicKey string `yaml:"public-key,omitempty"`
	ShortID   string `yaml:"short-id,omitempty"`
}

type clashConfig struct {
	Proxies []clashProxy `yaml:"proxies"`
}

func (f *ClashFormatter) Format(nodes []*Node) (string, error) {
	return f.FormatWithPassword(nodes, "")
}

func (f *ClashFormatter) FormatWithPassword(nodes []*Node, password string) (string, error) {
	config := clashConfig{
		Proxies: make([]clashProxy, 0, len(nodes)),
	}

	for _, node := range nodes {
		var proxy clashProxy

		switch vo.Protocol(node.Protocol) {
		case vo.ProtocolTrojan:
			proxy = clashProxy{
				Name:           node.Name,
				Type:           "trojan",
				Server:         node.ServerAddress,
				Port:           node.SubscriptionPort,
				Password:       password,
				UDP:            true,
				SNI:            node.SNI,
				SkipCertVerify: node.AllowInsecure,
			}

			// Handle transport
			switch node.TransportProtocol {
			case "ws":
				proxy.Network = "ws"
				proxy.WSOpts = &clashWSOpts{
					Path: node.Path,
				}
				if node.Host != "" {
					proxy.WSOpts.Headers = map[string]string{
						"Host": node.Host,
					}
				}
			case "grpc":
				proxy.Network = "grpc"
				proxy.GRPCOpts = &clashGRPCOpts{
					GRPCServiceName: node.Host,
				}
			}

		case vo.ProtocolVLESS:
			if node.VLESSConfig != nil {
				proxy = f.buildVLESSProxy(node, password)
			}

		case vo.ProtocolVMess:
			if node.VMessConfig != nil {
				proxy = f.buildVMessProxy(node, password)
			}

		case vo.ProtocolHysteria2:
			if node.Hysteria2Config != nil {
				proxy = f.buildHysteria2Proxy(node)
			}

		case vo.ProtocolTUIC:
			if node.TUICConfig != nil {
				proxy = f.buildTUICProxy(node)
			}

		default:
			// Shadowsocks: adjust password for SS2022 methods
			nodePassword := adjustPasswordForMethod(password, node.EncryptionMethod, node.TokenHash)

			proxy = clashProxy{
				Name:     node.Name,
				Type:     "ss",
				Server:   node.ServerAddress,
				Port:     node.SubscriptionPort,
				Cipher:   node.EncryptionMethod,
				Password: nodePassword,
				UDP:      true,
			}

			if node.Plugin != "" {
				proxy.Plugin = node.Plugin
				proxy.PluginOpts = node.PluginOpts
			}
		}

		// Only append non-empty proxy
		if proxy.Type != "" {
			config.Proxies = append(config.Proxies, proxy)
		}
	}

	yamlBytes, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal clash config: %w", err)
	}

	return string(yamlBytes), nil
}

// buildVLESSProxy builds a Clash Meta VLESS proxy configuration
func (f *ClashFormatter) buildVLESSProxy(node *Node, uuid string) clashProxy {
	cfg := node.VLESSConfig
	proxy := clashProxy{
		Name:           node.Name,
		Type:           "vless",
		Server:         node.ServerAddress,
		Port:           node.SubscriptionPort,
		UUID:           uuid,
		UDP:            true,
		Network:        cfg.TransportType(),
		SkipCertVerify: cfg.AllowInsecure(),
	}

	// Set flow control
	if cfg.Flow() != "" {
		proxy.Flow = cfg.Flow()
	}

	// Set TLS/Reality security
	switch cfg.Security() {
	case vo.VLESSSecurityTLS:
		proxy.TLS = true
		if cfg.SNI() != "" {
			proxy.SNI = cfg.SNI()
		}
		if cfg.Fingerprint() != "" {
			proxy.Fingerprint = cfg.Fingerprint()
		}
	case vo.VLESSSecurityReality:
		proxy.TLS = true
		if cfg.SNI() != "" {
			proxy.SNI = cfg.SNI()
		}
		if cfg.Fingerprint() != "" {
			proxy.Fingerprint = cfg.Fingerprint()
		}
		proxy.RealityOpts = &clashRealityOpts{
			PublicKey: cfg.PublicKey(),
			ShortID:   cfg.ShortID(),
		}
	}

	// Set transport-specific options
	switch cfg.TransportType() {
	case vo.VLESSTransportWS:
		proxy.WSOpts = &clashWSOpts{
			Path: cfg.Path(),
		}
		if cfg.Host() != "" {
			proxy.WSOpts.Headers = map[string]string{
				"Host": cfg.Host(),
			}
		}
	case vo.VLESSTransportGRPC:
		proxy.GRPCOpts = &clashGRPCOpts{
			GRPCServiceName: cfg.ServiceName(),
		}
	case vo.VLESSTransportH2:
		proxy.H2Opts = &clashH2Opts{
			Path: cfg.Path(),
		}
		if cfg.Host() != "" {
			proxy.H2Opts.Host = []string{cfg.Host()}
		}
	}

	return proxy
}

// buildVMessProxy builds a Clash Meta VMess proxy configuration
func (f *ClashFormatter) buildVMessProxy(node *Node, uuid string) clashProxy {
	cfg := node.VMessConfig
	proxy := clashProxy{
		Name:           node.Name,
		Type:           "vmess",
		Server:         node.ServerAddress,
		Port:           node.SubscriptionPort,
		UUID:           uuid,
		AlterID:        cfg.AlterID(),
		Cipher:         cfg.Security(),
		UDP:            true,
		Network:        cfg.TransportType(),
		TLS:            cfg.TLS(),
		SkipCertVerify: cfg.AllowInsecure(),
	}

	if cfg.SNI() != "" {
		proxy.SNI = cfg.SNI()
	}

	// Set transport-specific options
	switch cfg.TransportType() {
	case vo.VMessTransportWS:
		proxy.WSOpts = &clashWSOpts{
			Path: cfg.Path(),
		}
		if cfg.Host() != "" {
			proxy.WSOpts.Headers = map[string]string{
				"Host": cfg.Host(),
			}
		}
	case vo.VMessTransportHTTP:
		proxy.H2Opts = &clashH2Opts{
			Path: cfg.Path(),
		}
		if cfg.Host() != "" {
			proxy.H2Opts.Host = []string{cfg.Host()}
		}
	case vo.VMessTransportGRPC:
		proxy.GRPCOpts = &clashGRPCOpts{
			GRPCServiceName: cfg.ServiceName(),
		}
	}

	return proxy
}

// buildHysteria2Proxy builds a Clash Meta Hysteria2 proxy configuration
func (f *ClashFormatter) buildHysteria2Proxy(node *Node) clashProxy {
	cfg := node.Hysteria2Config
	proxy := clashProxy{
		Name:           node.Name,
		Type:           "hysteria2",
		Server:         node.ServerAddress,
		Port:           node.SubscriptionPort,
		Password:       cfg.Password(),
		SkipCertVerify: cfg.AllowInsecure(),
	}

	if cfg.SNI() != "" {
		proxy.SNI = cfg.SNI()
	}

	if cfg.Obfs() != "" {
		proxy.Obfs = cfg.Obfs()
		if cfg.ObfsPassword() != "" {
			proxy.ObfsPassword = cfg.ObfsPassword()
		}
	}

	// Bandwidth limits (Clash Meta uses string format with unit)
	if cfg.UpMbps() != nil {
		proxy.Up = fmt.Sprintf("%d Mbps", *cfg.UpMbps())
	}
	if cfg.DownMbps() != nil {
		proxy.Down = fmt.Sprintf("%d Mbps", *cfg.DownMbps())
	}

	if cfg.Fingerprint() != "" {
		proxy.Fingerprint = cfg.Fingerprint()
	}

	return proxy
}

// buildTUICProxy builds a Clash Meta TUIC proxy configuration
func (f *ClashFormatter) buildTUICProxy(node *Node) clashProxy {
	cfg := node.TUICConfig
	proxy := clashProxy{
		Name:                 node.Name,
		Type:                 "tuic",
		Server:               node.ServerAddress,
		Port:                 node.SubscriptionPort,
		UUID:                 cfg.UUID(),
		Password:             cfg.Password(),
		CongestionController: cfg.CongestionControl(),
		UDPRelayMode:         cfg.UDPRelayMode(),
		SkipCertVerify:       cfg.AllowInsecure(),
		DisableSNI:           cfg.DisableSNI(),
	}

	if cfg.SNI() != "" {
		proxy.SNI = cfg.SNI()
	}

	if cfg.ALPN() != "" {
		proxy.ALPN = strings.Split(cfg.ALPN(), ",")
	}

	return proxy
}

func (f *ClashFormatter) ContentType() string {
	return "text/yaml; charset=utf-8"
}

type V2RayFormatter struct{}

func NewV2RayFormatter() *V2RayFormatter {
	return &V2RayFormatter{}
}

type v2rayNode struct {
	Remarks    string `json:"remarks"`
	Server     string `json:"server"`
	ServerPort uint16 `json:"server_port"`
	Password   string `json:"password"`
	Method     string `json:"method"`
	Plugin     string `json:"plugin,omitempty"`
	PluginOpts string `json:"plugin_opts,omitempty"`
}

func (f *V2RayFormatter) Format(nodes []*Node) (string, error) {
	return f.FormatWithPassword(nodes, "")
}

func (f *V2RayFormatter) FormatWithPassword(nodes []*Node, password string) (string, error) {
	v2rayNodes := make([]v2rayNode, 0, len(nodes))
	skippedCount := 0

	for _, node := range nodes {
		// V2Ray format only supports Shadowsocks, skip other protocol nodes
		if vo.Protocol(node.Protocol) != vo.ProtocolShadowsocks {
			skippedCount++
			continue
		}

		// Adjust password for SS2022 methods
		nodePassword := adjustPasswordForMethod(password, node.EncryptionMethod, node.TokenHash)

		v2rayNode := v2rayNode{
			Remarks:    node.Name,
			Server:     node.ServerAddress,
			ServerPort: node.SubscriptionPort,
			Password:   nodePassword,
			Method:     node.EncryptionMethod,
		}

		if node.Plugin != "" {
			v2rayNode.Plugin = node.Plugin
			v2rayNode.PluginOpts = formatPluginOpts(node.PluginOpts)
		}

		v2rayNodes = append(v2rayNodes, v2rayNode)
	}

	// Return error if all nodes were non-Shadowsocks (V2Ray format only supports Shadowsocks)
	if len(v2rayNodes) == 0 && skippedCount > 0 {
		return "", fmt.Errorf("v2ray format only supports Shadowsocks protocol, please use base64 or clash format instead")
	}

	jsonBytes, err := json.MarshalIndent(v2rayNodes, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal v2ray config: %w", err)
	}

	return string(jsonBytes), nil
}

func (f *V2RayFormatter) ContentType() string {
	return "application/json; charset=utf-8"
}

type SIP008Formatter struct{}

func NewSIP008Formatter() *SIP008Formatter {
	return &SIP008Formatter{}
}

type sip008Server struct {
	ID         string `json:"id"`
	Remarks    string `json:"remarks"`
	Server     string `json:"server"`
	ServerPort uint16 `json:"server_port"`
	Password   string `json:"password"`
	Method     string `json:"method"`
	Plugin     string `json:"plugin,omitempty"`
	PluginOpts string `json:"plugin_opts,omitempty"`
}

type sip008Config struct {
	Version int            `json:"version"`
	Servers []sip008Server `json:"servers"`
}

func (f *SIP008Formatter) Format(nodes []*Node) (string, error) {
	return f.FormatWithPassword(nodes, "")
}

func (f *SIP008Formatter) FormatWithPassword(nodes []*Node, password string) (string, error) {
	config := sip008Config{
		Version: 1,
		Servers: make([]sip008Server, 0, len(nodes)),
	}
	skippedCount := 0

	for _, node := range nodes {
		// SIP008 format only supports Shadowsocks, skip other protocol nodes
		if vo.Protocol(node.Protocol) != vo.ProtocolShadowsocks {
			skippedCount++
			continue
		}

		// Adjust password for SS2022 methods
		nodePassword := adjustPasswordForMethod(password, node.EncryptionMethod, node.TokenHash)

		server := sip008Server{
			ID:         fmt.Sprintf("node_%d", node.ID),
			Remarks:    node.Name,
			Server:     node.ServerAddress,
			ServerPort: node.SubscriptionPort,
			Password:   nodePassword,
			Method:     node.EncryptionMethod,
		}

		if node.Plugin != "" {
			server.Plugin = node.Plugin
			server.PluginOpts = formatPluginOpts(node.PluginOpts)
		}

		config.Servers = append(config.Servers, server)
	}

	// Return error if all nodes were non-Shadowsocks (SIP008 format only supports Shadowsocks)
	if len(config.Servers) == 0 && skippedCount > 0 {
		return "", fmt.Errorf("sip008 format only supports Shadowsocks protocol, please use base64 or clash format instead")
	}

	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal sip008 config: %w", err)
	}

	return string(jsonBytes), nil
}

func (f *SIP008Formatter) ContentType() string {
	return "application/json; charset=utf-8"
}

type SurgeFormatter struct{}

func NewSurgeFormatter() *SurgeFormatter {
	return &SurgeFormatter{}
}

func (f *SurgeFormatter) Format(nodes []*Node) (string, error) {
	return f.FormatWithPassword(nodes, "")
}

func (f *SurgeFormatter) FormatWithPassword(nodes []*Node, password string) (string, error) {
	var lines []string
	lines = append(lines, "[Proxy]")

	for _, node := range nodes {
		nodeName := node.Name
		var line string

		switch vo.Protocol(node.Protocol) {
		case vo.ProtocolTrojan:
			// Surge Trojan format
			line = fmt.Sprintf("%s = trojan, %s, %d, password=%s",
				nodeName,
				node.ServerAddress,
				node.SubscriptionPort,
				password)

			if node.SNI != "" {
				line += fmt.Sprintf(", sni=%s", node.SNI)
			}
			if node.AllowInsecure {
				line += ", skip-cert-verify=true"
			}

			// Handle transport
			switch node.TransportProtocol {
			case "ws":
				line += ", ws=true"
				if node.Path != "" {
					line += fmt.Sprintf(", ws-path=%s", node.Path)
				}
				if node.Host != "" {
					line += fmt.Sprintf(", ws-headers=Host:%s", node.Host)
				}
			}

		case vo.ProtocolVLESS:
			// Surge 5 doesn't support VLESS natively, skip
			continue

		case vo.ProtocolVMess:
			if node.VMessConfig != nil {
				line = f.buildVMessLine(node, password)
			}

		case vo.ProtocolHysteria2:
			if node.Hysteria2Config != nil {
				line = f.buildHysteria2Line(node)
			}

		case vo.ProtocolTUIC:
			if node.TUICConfig != nil {
				line = f.buildTUICLine(node)
			}

		default:
			// Shadowsocks: adjust password for SS2022 methods
			nodePassword := adjustPasswordForMethod(password, node.EncryptionMethod, node.TokenHash)

			// Shadowsocks format
			line = fmt.Sprintf("%s = ss, %s, %d, encrypt-method=%s, password=%s, udp-relay=true",
				nodeName,
				node.ServerAddress,
				node.SubscriptionPort,
				node.EncryptionMethod,
				nodePassword)

			if node.Plugin == "obfs-local" && len(node.PluginOpts) > 0 {
				if obfsMode, ok := node.PluginOpts["obfs"]; ok {
					line += fmt.Sprintf(", obfs=%s", obfsMode)
					if obfsHost, ok := node.PluginOpts["obfs-host"]; ok {
						line += fmt.Sprintf(", obfs-host=%s", obfsHost)
					}
				}
			}
		}

		if line != "" {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n"), nil
}

// buildVMessLine builds a Surge 5 VMess proxy line
func (f *SurgeFormatter) buildVMessLine(node *Node, uuid string) string {
	cfg := node.VMessConfig

	// Surge VMess format: name = vmess, server, port, username=uuid, encrypt-method=auto
	line := fmt.Sprintf("%s = vmess, %s, %d, username=%s, encrypt-method=%s",
		node.Name,
		node.ServerAddress,
		node.SubscriptionPort,
		uuid,
		cfg.Security())

	// Add TLS settings
	if cfg.TLS() {
		line += ", tls=true"
		if cfg.SNI() != "" {
			line += fmt.Sprintf(", sni=%s", cfg.SNI())
		}
		if cfg.AllowInsecure() {
			line += ", skip-cert-verify=true"
		}
	}

	// Handle transport
	switch cfg.TransportType() {
	case vo.VMessTransportWS:
		line += ", ws=true"
		if cfg.Path() != "" {
			line += fmt.Sprintf(", ws-path=%s", cfg.Path())
		}
		if cfg.Host() != "" {
			line += fmt.Sprintf(", ws-headers=Host:%s", cfg.Host())
		}
	}

	return line
}

// buildHysteria2Line builds a Surge 5 Hysteria2 proxy line
func (f *SurgeFormatter) buildHysteria2Line(node *Node) string {
	cfg := node.Hysteria2Config

	// Surge Hysteria2 format: name = hysteria2, server, port, password=xxx
	line := fmt.Sprintf("%s = hysteria2, %s, %d, password=%s",
		node.Name,
		node.ServerAddress,
		node.SubscriptionPort,
		cfg.Password())

	if cfg.SNI() != "" {
		line += fmt.Sprintf(", sni=%s", cfg.SNI())
	}

	if cfg.AllowInsecure() {
		line += ", skip-cert-verify=true"
	}

	// Bandwidth limits
	if cfg.DownMbps() != nil {
		line += fmt.Sprintf(", download-bandwidth=%d", *cfg.DownMbps())
	}

	return line
}

// buildTUICLine builds a Surge 5 TUIC proxy line
func (f *SurgeFormatter) buildTUICLine(node *Node) string {
	cfg := node.TUICConfig

	// Surge TUIC format: name = tuic, server, port, token=uuid, password=xxx
	line := fmt.Sprintf("%s = tuic, %s, %d, token=%s",
		node.Name,
		node.ServerAddress,
		node.SubscriptionPort,
		cfg.UUID())

	if cfg.Password() != "" {
		line += fmt.Sprintf(", password=%s", cfg.Password())
	}

	if cfg.SNI() != "" {
		line += fmt.Sprintf(", sni=%s", cfg.SNI())
	}

	if cfg.AllowInsecure() {
		line += ", skip-cert-verify=true"
	}

	if cfg.ALPN() != "" {
		line += fmt.Sprintf(", alpn=%s", cfg.ALPN())
	}

	return line
}

func (f *SurgeFormatter) ContentType() string {
	return "text/plain; charset=utf-8"
}

func formatPluginOpts(opts map[string]string) string {
	if len(opts) == 0 {
		return ""
	}

	var parts []string
	for k, v := range opts {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ";")
}

// TemplateClashFormatter wraps ClashFormatter with template support
// It will use custom template if available, otherwise fall back to default formatter
type TemplateClashFormatter struct {
	renderer         *TemplateRenderer
	defaultFormatter *ClashFormatter
}

// NewTemplateClashFormatter creates a new template-aware Clash formatter
func NewTemplateClashFormatter(renderer *TemplateRenderer) *TemplateClashFormatter {
	return &TemplateClashFormatter{
		renderer:         renderer,
		defaultFormatter: NewClashFormatter(),
	}
}

func (f *TemplateClashFormatter) Format(nodes []*Node) (string, error) {
	return f.FormatWithPassword(nodes, "")
}

func (f *TemplateClashFormatter) FormatWithPassword(nodes []*Node, password string) (string, error) {
	// Try template rendering first
	if f.renderer.HasTemplate("clash") {
		content, err := f.renderer.RenderClash(nodes, password)
		if err != nil {
			// Log error but fall back to default formatter
			// (error logging handled by caller)
			return f.defaultFormatter.FormatWithPassword(nodes, password)
		}
		return content, nil
	}

	// Fall back to default formatter
	return f.defaultFormatter.FormatWithPassword(nodes, password)
}

func (f *TemplateClashFormatter) ContentType() string {
	return f.defaultFormatter.ContentType()
}

// TemplateSurgeFormatter wraps SurgeFormatter with template support
// It will use custom template if available, otherwise fall back to default formatter
type TemplateSurgeFormatter struct {
	renderer         *TemplateRenderer
	defaultFormatter *SurgeFormatter
}

// NewTemplateSurgeFormatter creates a new template-aware Surge formatter
func NewTemplateSurgeFormatter(renderer *TemplateRenderer) *TemplateSurgeFormatter {
	return &TemplateSurgeFormatter{
		renderer:         renderer,
		defaultFormatter: NewSurgeFormatter(),
	}
}

func (f *TemplateSurgeFormatter) Format(nodes []*Node) (string, error) {
	return f.FormatWithPassword(nodes, "")
}

func (f *TemplateSurgeFormatter) FormatWithPassword(nodes []*Node, password string) (string, error) {
	// Try template rendering first
	if f.renderer.HasTemplate("surge") {
		content, err := f.renderer.RenderSurge(nodes, password)
		if err != nil {
			// Log error but fall back to default formatter
			// (error logging handled by caller)
			return f.defaultFormatter.FormatWithPassword(nodes, password)
		}
		return content, nil
	}

	// Fall back to default formatter
	return f.defaultFormatter.FormatWithPassword(nodes, password)
}

func (f *TemplateSurgeFormatter) ContentType() string {
	return f.defaultFormatter.ContentType()
}
