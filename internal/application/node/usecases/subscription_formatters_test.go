package usecases

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func createTestNode() *Node {
	return &Node{
		ID:               1,
		Name:             "Test Node",
		ServerAddress:    "example.com",
		ServerPort:       8388,
		EncryptionMethod: "aes-256-gcm",
		Password:         "test-password",
	}
}

func createTestNodeWithPlugin() *Node {
	node := createTestNode()
	node.Plugin = "obfs-local"
	node.PluginOpts = map[string]string{
		"obfs":      "http",
		"obfs-host": "cloudflare.com",
	}
	return node
}

func TestBase64Formatter_Format_SingleNode(t *testing.T) {
	formatter := NewBase64Formatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)

	content := string(decoded)
	assert.Contains(t, content, "ss://")
	assert.Contains(t, content, node.ServerAddress)
	assert.Contains(t, content, url.QueryEscape(node.Name))

	expectedAuth := base64.StdEncoding.EncodeToString([]byte(node.EncryptionMethod + ":" + node.Password))
	assert.Contains(t, content, expectedAuth)
}

func TestBase64Formatter_Format_MultipleNodes(t *testing.T) {
	formatter := NewBase64Formatter()
	nodes := []*Node{
		createTestNode(),
		{
			ID:               2,
			Name:             "Test Node 2",
			ServerAddress:    "example2.com",
			ServerPort:       8389,
			EncryptionMethod: "chacha20-ietf-poly1305",
			Password:         "password2",
		},
	}

	result, err := formatter.Format(nodes)
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)

	content := string(decoded)
	lines := strings.Split(content, "\n")
	assert.Equal(t, 2, len(lines))

	for i, line := range lines {
		assert.Contains(t, line, "ss://")
		assert.Contains(t, line, nodes[i].ServerAddress)
	}
}

func TestBase64Formatter_Format_WithPlugin(t *testing.T) {
	formatter := NewBase64Formatter()
	node := createTestNodeWithPlugin()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)

	content := string(decoded)
	assert.Contains(t, content, "?plugin=")
	assert.Contains(t, content, url.QueryEscape(node.Plugin))

	unescaped, err := url.QueryUnescape(content)
	require.NoError(t, err)
	assert.Contains(t, unescaped, "obfs=http")
	assert.Contains(t, unescaped, "obfs-host=cloudflare.com")
}

func TestBase64Formatter_Format_Base64Encoding(t *testing.T) {
	formatter := NewBase64Formatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	_, err = base64.StdEncoding.DecodeString(result)
	assert.NoError(t, err, "Result should be valid base64")
}

func TestBase64Formatter_ContentType(t *testing.T) {
	formatter := NewBase64Formatter()
	assert.Equal(t, "text/plain; charset=utf-8", formatter.ContentType())
}

func TestClashFormatter_Format_YAMLStructure(t *testing.T) {
	formatter := NewClashFormatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	var config clashConfig
	err = yaml.Unmarshal([]byte(result), &config)
	require.NoError(t, err)

	assert.Equal(t, 1, len(config.Proxies))
}

func TestClashFormatter_Format_ProxyConfiguration(t *testing.T) {
	formatter := NewClashFormatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	var config clashConfig
	err = yaml.Unmarshal([]byte(result), &config)
	require.NoError(t, err)

	proxy := config.Proxies[0]
	assert.Equal(t, node.Name, proxy.Name)
	assert.Equal(t, "ss", proxy.Type)
	assert.Equal(t, node.ServerAddress, proxy.Server)
	assert.Equal(t, node.ServerPort, proxy.Port)
	assert.Equal(t, node.EncryptionMethod, proxy.Cipher)
	assert.Equal(t, node.Password, proxy.Password)
	assert.True(t, proxy.UDP)
}

func TestClashFormatter_Format_WithPluginOptions(t *testing.T) {
	formatter := NewClashFormatter()
	node := createTestNodeWithPlugin()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	var config clashConfig
	err = yaml.Unmarshal([]byte(result), &config)
	require.NoError(t, err)

	proxy := config.Proxies[0]
	assert.Equal(t, node.Plugin, proxy.Plugin)
	assert.Equal(t, node.PluginOpts, proxy.PluginOpts)
}

func TestClashFormatter_Format_MultipleProxies(t *testing.T) {
	formatter := NewClashFormatter()
	nodes := []*Node{
		createTestNode(),
		{
			ID:               2,
			Name:             "Node 2",
			ServerAddress:    "server2.com",
			ServerPort:       9999,
			EncryptionMethod: "chacha20",
			Password:         "pass2",
		},
	}

	result, err := formatter.Format(nodes)
	require.NoError(t, err)

	var config clashConfig
	err = yaml.Unmarshal([]byte(result), &config)
	require.NoError(t, err)

	assert.Equal(t, 2, len(config.Proxies))
}

func TestClashFormatter_ContentType(t *testing.T) {
	formatter := NewClashFormatter()
	assert.Equal(t, "text/yaml; charset=utf-8", formatter.ContentType())
}

func TestV2RayFormatter_Format_JSONStructure(t *testing.T) {
	formatter := NewV2RayFormatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	var nodes []v2rayNode
	err = json.Unmarshal([]byte(result), &nodes)
	require.NoError(t, err)

	assert.Equal(t, 1, len(nodes))
}

func TestV2RayFormatter_Format_NodeConfiguration(t *testing.T) {
	formatter := NewV2RayFormatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	var nodes []v2rayNode
	err = json.Unmarshal([]byte(result), &nodes)
	require.NoError(t, err)

	v2rayNode := nodes[0]
	assert.Equal(t, node.Name, v2rayNode.Remarks)
	assert.Equal(t, node.ServerAddress, v2rayNode.Server)
	assert.Equal(t, node.ServerPort, v2rayNode.ServerPort)
	assert.Equal(t, node.Password, v2rayNode.Password)
	assert.Equal(t, node.EncryptionMethod, v2rayNode.Method)
}

func TestV2RayFormatter_Format_WithPlugin(t *testing.T) {
	formatter := NewV2RayFormatter()
	node := createTestNodeWithPlugin()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	var nodes []v2rayNode
	err = json.Unmarshal([]byte(result), &nodes)
	require.NoError(t, err)

	v2rayNode := nodes[0]
	assert.Equal(t, node.Plugin, v2rayNode.Plugin)
	assert.Contains(t, v2rayNode.PluginOpts, "obfs=http")
	assert.Contains(t, v2rayNode.PluginOpts, "obfs-host=cloudflare.com")
}

func TestV2RayFormatter_Format_MultipleNodes(t *testing.T) {
	formatter := NewV2RayFormatter()
	nodes := []*Node{createTestNode(), createTestNode()}
	nodes[1].ID = 2
	nodes[1].Name = "Node 2"

	result, err := formatter.Format(nodes)
	require.NoError(t, err)

	var v2rayNodes []v2rayNode
	err = json.Unmarshal([]byte(result), &v2rayNodes)
	require.NoError(t, err)

	assert.Equal(t, 2, len(v2rayNodes))
}

func TestV2RayFormatter_ContentType(t *testing.T) {
	formatter := NewV2RayFormatter()
	assert.Equal(t, "application/json; charset=utf-8", formatter.ContentType())
}

func TestSIP008Formatter_Format_StandardFormat(t *testing.T) {
	formatter := NewSIP008Formatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	var config sip008Config
	err = json.Unmarshal([]byte(result), &config)
	require.NoError(t, err)

	assert.Equal(t, 1, config.Version)
	assert.Equal(t, 1, len(config.Servers))
}

func TestSIP008Formatter_Format_VersionField(t *testing.T) {
	formatter := NewSIP008Formatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	var config sip008Config
	err = json.Unmarshal([]byte(result), &config)
	require.NoError(t, err)

	assert.Equal(t, 1, config.Version)
}

func TestSIP008Formatter_Format_ServerList(t *testing.T) {
	formatter := NewSIP008Formatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	var config sip008Config
	err = json.Unmarshal([]byte(result), &config)
	require.NoError(t, err)

	server := config.Servers[0]
	assert.Equal(t, "node_1", server.ID)
	assert.Equal(t, node.Name, server.Remarks)
	assert.Equal(t, node.ServerAddress, server.Server)
	assert.Equal(t, node.ServerPort, server.ServerPort)
	assert.Equal(t, node.Password, server.Password)
	assert.Equal(t, node.EncryptionMethod, server.Method)
}

func TestSIP008Formatter_Format_WithPlugin(t *testing.T) {
	formatter := NewSIP008Formatter()
	node := createTestNodeWithPlugin()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	var config sip008Config
	err = json.Unmarshal([]byte(result), &config)
	require.NoError(t, err)

	server := config.Servers[0]
	assert.Equal(t, node.Plugin, server.Plugin)
	assert.Contains(t, server.PluginOpts, "obfs=http")
}

func TestSIP008Formatter_Format_MultipleServers(t *testing.T) {
	formatter := NewSIP008Formatter()
	nodes := []*Node{
		createTestNode(),
		{
			ID:               2,
			Name:             "Node 2",
			ServerAddress:    "server2.com",
			ServerPort:       8389,
			EncryptionMethod: "chacha20",
			Password:         "pass2",
		},
	}

	result, err := formatter.Format(nodes)
	require.NoError(t, err)

	var config sip008Config
	err = json.Unmarshal([]byte(result), &config)
	require.NoError(t, err)

	assert.Equal(t, 2, len(config.Servers))
	assert.Equal(t, "node_1", config.Servers[0].ID)
	assert.Equal(t, "node_2", config.Servers[1].ID)
}

func TestSIP008Formatter_ContentType(t *testing.T) {
	formatter := NewSIP008Formatter()
	assert.Equal(t, "application/json; charset=utf-8", formatter.ContentType())
}

func TestSurgeFormatter_Format_INIFormat(t *testing.T) {
	formatter := NewSurgeFormatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	lines := strings.Split(result, "\n")
	assert.Equal(t, "[Proxy]", lines[0])
	assert.Greater(t, len(lines), 1)
}

func TestSurgeFormatter_Format_ProxyConfiguration(t *testing.T) {
	formatter := NewSurgeFormatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	lines := strings.Split(result, "\n")
	proxyLine := lines[1]

	assert.Contains(t, proxyLine, "Test_Node")
	assert.Contains(t, proxyLine, "ss")
	assert.Contains(t, proxyLine, node.ServerAddress)
	assert.Contains(t, proxyLine, "8388")
	assert.Contains(t, proxyLine, "encrypt-method="+node.EncryptionMethod)
	assert.Contains(t, proxyLine, "password="+node.Password)
	assert.Contains(t, proxyLine, "udp-relay=true")
}

func TestSurgeFormatter_Format_SpecialCharacterHandling(t *testing.T) {
	formatter := NewSurgeFormatter()
	node := createTestNode()
	node.Name = "Test Node With Spaces"

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	lines := strings.Split(result, "\n")
	proxyLine := lines[1]

	assert.Contains(t, proxyLine, "Test_Node_With_Spaces")
	assert.NotContains(t, proxyLine, "Test Node With Spaces")
}

func TestSurgeFormatter_Format_WithObfsPlugin(t *testing.T) {
	formatter := NewSurgeFormatter()
	node := createTestNodeWithPlugin()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	lines := strings.Split(result, "\n")
	proxyLine := lines[1]

	assert.Contains(t, proxyLine, "obfs=http")
	assert.Contains(t, proxyLine, "obfs-host=cloudflare.com")
}

func TestSurgeFormatter_Format_WithoutPlugin(t *testing.T) {
	formatter := NewSurgeFormatter()
	node := createTestNode()

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	assert.NotContains(t, result, "obfs=")
}

func TestSurgeFormatter_Format_MultipleProxies(t *testing.T) {
	formatter := NewSurgeFormatter()
	nodes := []*Node{
		createTestNode(),
		{
			ID:               2,
			Name:             "Node 2",
			ServerAddress:    "server2.com",
			ServerPort:       9999,
			EncryptionMethod: "chacha20",
			Password:         "pass2",
		},
	}

	result, err := formatter.Format(nodes)
	require.NoError(t, err)

	lines := strings.Split(result, "\n")
	assert.Equal(t, 3, len(lines))
	assert.Equal(t, "[Proxy]", lines[0])
}

func TestSurgeFormatter_ContentType(t *testing.T) {
	formatter := NewSurgeFormatter()
	assert.Equal(t, "text/plain; charset=utf-8", formatter.ContentType())
}

func TestFormatPluginOpts_EmptyMap(t *testing.T) {
	result := formatPluginOpts(map[string]string{})
	assert.Equal(t, "", result)
}

func TestFormatPluginOpts_SingleOption(t *testing.T) {
	opts := map[string]string{"key": "value"}
	result := formatPluginOpts(opts)
	assert.Equal(t, "key=value", result)
}

func TestFormatPluginOpts_MultipleOptions(t *testing.T) {
	opts := map[string]string{
		"obfs":      "http",
		"obfs-host": "cloudflare.com",
	}
	result := formatPluginOpts(opts)

	assert.Contains(t, result, "obfs=http")
	assert.Contains(t, result, "obfs-host=cloudflare.com")
	assert.Contains(t, result, ";")
}

func TestBase64Formatter_Format_EmptyNodes(t *testing.T) {
	formatter := NewBase64Formatter()
	result, err := formatter.Format([]*Node{})
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)
	assert.Equal(t, "", string(decoded))
}

func TestClashFormatter_Format_EmptyNodes(t *testing.T) {
	formatter := NewClashFormatter()
	result, err := formatter.Format([]*Node{})
	require.NoError(t, err)

	var config clashConfig
	err = yaml.Unmarshal([]byte(result), &config)
	require.NoError(t, err)
	assert.Equal(t, 0, len(config.Proxies))
}

func TestV2RayFormatter_Format_EmptyNodes(t *testing.T) {
	formatter := NewV2RayFormatter()
	result, err := formatter.Format([]*Node{})
	require.NoError(t, err)

	var nodes []v2rayNode
	err = json.Unmarshal([]byte(result), &nodes)
	require.NoError(t, err)
	assert.Equal(t, 0, len(nodes))
}

func TestSIP008Formatter_Format_EmptyNodes(t *testing.T) {
	formatter := NewSIP008Formatter()
	result, err := formatter.Format([]*Node{})
	require.NoError(t, err)

	var config sip008Config
	err = json.Unmarshal([]byte(result), &config)
	require.NoError(t, err)
	assert.Equal(t, 1, config.Version)
	assert.Equal(t, 0, len(config.Servers))
}

func TestSurgeFormatter_Format_EmptyNodes(t *testing.T) {
	formatter := NewSurgeFormatter()
	result, err := formatter.Format([]*Node{})
	require.NoError(t, err)

	assert.Equal(t, "[Proxy]", result)
}

func TestBase64Formatter_Format_NodeWithoutName(t *testing.T) {
	formatter := NewBase64Formatter()
	node := createTestNode()
	node.Name = ""

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)

	content := string(decoded)
	assert.NotContains(t, content, "#")
}

func TestSurgeFormatter_Format_NonObfsPlugin(t *testing.T) {
	formatter := NewSurgeFormatter()
	node := createTestNode()
	node.Plugin = "v2ray-plugin"
	node.PluginOpts = map[string]string{
		"mode": "websocket",
	}

	result, err := formatter.Format([]*Node{node})
	require.NoError(t, err)

	assert.NotContains(t, result, "obfs=")
	assert.NotContains(t, result, "obfs-host=")
}

func TestAllFormatters_ContentTypeCorrectness(t *testing.T) {
	tests := []struct {
		name        string
		formatter   SubscriptionFormatter
		contentType string
	}{
		{"Base64", NewBase64Formatter(), "text/plain; charset=utf-8"},
		{"Clash", NewClashFormatter(), "text/yaml; charset=utf-8"},
		{"V2Ray", NewV2RayFormatter(), "application/json; charset=utf-8"},
		{"SIP008", NewSIP008Formatter(), "application/json; charset=utf-8"},
		{"Surge", NewSurgeFormatter(), "text/plain; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.contentType, tt.formatter.ContentType())
		})
	}
}

func TestAllFormatters_BasicFormatting(t *testing.T) {
	node := createTestNode()

	tests := []struct {
		name      string
		formatter SubscriptionFormatter
	}{
		{"Base64", NewBase64Formatter()},
		{"Clash", NewClashFormatter()},
		{"V2Ray", NewV2RayFormatter()},
		{"SIP008", NewSIP008Formatter()},
		{"Surge", NewSurgeFormatter()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.formatter.Format([]*Node{node})
			require.NoError(t, err)
			assert.NotEmpty(t, result)
		})
	}
}
