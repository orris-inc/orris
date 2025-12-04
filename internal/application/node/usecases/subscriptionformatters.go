package usecases

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

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

		if node.Protocol == "trojan" {
			link = node.ToTrojanURI(password)
		} else {
			// Shadowsocks URI format: ss://base64(method:password)@host:port#remarks
			auth := fmt.Sprintf("%s:%s", node.EncryptionMethod, password)
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

		links = append(links, link)
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
	Password       string            `yaml:"password"`
	UDP            bool              `yaml:"udp"`
	Plugin         string            `yaml:"plugin,omitempty"`
	PluginOpts     map[string]string `yaml:"plugin-opts,omitempty"`
	SNI            string            `yaml:"sni,omitempty"`
	SkipCertVerify bool              `yaml:"skip-cert-verify,omitempty"`
	Network        string            `yaml:"network,omitempty"`
	WSOpts         *clashWSOpts      `yaml:"ws-opts,omitempty"`
	GRPCOpts       *clashGRPCOpts    `yaml:"grpc-opts,omitempty"`
}

type clashWSOpts struct {
	Path    string            `yaml:"path,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

type clashGRPCOpts struct {
	GRPCServiceName string `yaml:"grpc-service-name,omitempty"`
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

		if node.Protocol == "trojan" {
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
		} else {
			proxy = clashProxy{
				Name:     node.Name,
				Type:     "ss",
				Server:   node.ServerAddress,
				Port:     node.SubscriptionPort,
				Cipher:   node.EncryptionMethod,
				Password: password,
				UDP:      true,
			}

			if node.Plugin != "" {
				proxy.Plugin = node.Plugin
				proxy.PluginOpts = node.PluginOpts
			}
		}

		config.Proxies = append(config.Proxies, proxy)
	}

	yamlBytes, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal clash config: %w", err)
	}

	return string(yamlBytes), nil
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

	for _, node := range nodes {
		// V2Ray format only supports Shadowsocks, skip Trojan nodes
		if node.Protocol == "trojan" {
			continue
		}

		v2rayNode := v2rayNode{
			Remarks:    node.Name,
			Server:     node.ServerAddress,
			ServerPort: node.SubscriptionPort,
			Password:   password,
			Method:     node.EncryptionMethod,
		}

		if node.Plugin != "" {
			v2rayNode.Plugin = node.Plugin
			v2rayNode.PluginOpts = formatPluginOpts(node.PluginOpts)
		}

		v2rayNodes = append(v2rayNodes, v2rayNode)
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

	for _, node := range nodes {
		// SIP008 format only supports Shadowsocks, skip Trojan nodes
		if node.Protocol == "trojan" {
			continue
		}

		server := sip008Server{
			ID:         fmt.Sprintf("node_%d", node.ID),
			Remarks:    node.Name,
			Server:     node.ServerAddress,
			ServerPort: node.SubscriptionPort,
			Password:   password,
			Method:     node.EncryptionMethod,
		}

		if node.Plugin != "" {
			server.Plugin = node.Plugin
			server.PluginOpts = formatPluginOpts(node.PluginOpts)
		}

		config.Servers = append(config.Servers, server)
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
		nodeName := strings.ReplaceAll(node.Name, " ", "_")
		var line string

		if node.Protocol == "trojan" {
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
		} else {
			// Shadowsocks format
			line = fmt.Sprintf("%s = ss, %s, %d, encrypt-method=%s, password=%s, udp-relay=true",
				nodeName,
				node.ServerAddress,
				node.SubscriptionPort,
				node.EncryptionMethod,
				password)

			if node.Plugin == "obfs-local" && len(node.PluginOpts) > 0 {
				if obfsMode, ok := node.PluginOpts["obfs"]; ok {
					line += fmt.Sprintf(", obfs=%s", obfsMode)
					if obfsHost, ok := node.PluginOpts["obfs-host"]; ok {
						line += fmt.Sprintf(", obfs-host=%s", obfsHost)
					}
				}
			}
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n"), nil
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
