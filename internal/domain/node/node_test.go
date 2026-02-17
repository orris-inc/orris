package node

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
)

// --- Test helpers ---

// fakeSIDGenerator returns a deterministic SID generator for testing.
func fakeSIDGenerator(sid string) func() (string, error) {
	return func() (string, error) {
		return sid, nil
	}
}

// newShadowsocksNode creates a minimal valid Shadowsocks node for testing.
func newShadowsocksNode(t *testing.T) *Node {
	t.Helper()
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)

	enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
	require.NoError(t, err)

	meta := vo.NewNodeMetadata("us-west", []string{"premium"}, "test node")

	n, err := NewNode(
		"test-ss-node",
		addr,
		8388,  // agentPort
		nil,   // subscriptionPort
		vo.ProtocolShadowsocks,
		enc,
		nil, // pluginConfig
		nil, // trojanConfig
		nil, // vlessConfig
		nil, // vmessConfig
		nil, // hysteria2Config
		nil, // tuicConfig
		nil, // anytlsConfig
		meta,
		0,   // sortOrder
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_test123"),
	)
	require.NoError(t, err)
	return n
}

// newTrojanNode creates a minimal valid Trojan node for testing.
func newTrojanNode(t *testing.T) *Node {
	t.Helper()
	addr, err := vo.NewServerAddress("example.com")
	require.NoError(t, err)

	trojanCfg, err := vo.NewTrojanConfig("supersecretpass", "tcp", "", "", false, "example.com")
	require.NoError(t, err)

	meta := vo.NewNodeMetadata("eu-west", nil, "trojan node")

	n, err := NewNode(
		"test-trojan-node",
		addr,
		443,
		nil,
		vo.ProtocolTrojan,
		vo.EncryptionConfig{}, // not needed for trojan
		nil,
		&trojanCfg,
		nil,
		nil,
		nil,
		nil,
		nil, // anytlsConfig
		meta,
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_trojan456"),
	)
	require.NoError(t, err)
	return n
}

// reconstructedNode creates a node via ReconstructNode for state transition tests.
func reconstructedNode(t *testing.T, status vo.NodeStatus) *Node {
	t.Helper()
	addr, err := vo.NewServerAddress("10.0.0.1")
	require.NoError(t, err)

	enc, err := vo.NewEncryptionConfig(vo.MethodChacha20IETFPoly1305)
	require.NoError(t, err)

	meta := vo.NewNodeMetadata("ap-east", []string{"fast"}, "reconstructed node")
	now := time.Now().UTC()

	var reason *string
	if status == vo.NodeStatusMaintenance {
		r := "scheduled maintenance"
		reason = &r
	}

	n, err := ReconstructNode(
		1,                // id
		"node_recon001",  // sid
		"recon-node",     // name
		addr,             // serverAddress
		8388,             // agentPort
		nil,              // subscriptionPort
		vo.ProtocolShadowsocks,
		enc,
		nil,    // pluginConfig
		nil,    // trojanConfig
		nil,    // vlessConfig
		nil,    // vmessConfig
		nil,    // hysteria2Config
		nil,    // tuicConfig
		nil,    // anytlsConfig
		status, // status
		meta,
		[]uint{1, 2},    // groupIDs
		nil,              // userID
		"abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234", // tokenHash (64 hex chars)
		"",     // apiToken (cleared)
		0,      // sortOrder
		false,  // muteNotification
		reason, // maintenanceReason
		nil,    // routeConfig
		nil,    // dnsConfig
		nil,    // lastSeenAt
		nil,    // publicIPv4
		nil,    // publicIPv6
		nil,    // agentVersion
		nil,    // platform
		nil,    // arch
		nil,    // expiresAt
		nil,    // costLabel
		1,      // version
		now,    // createdAt
		now,    // updatedAt
	)
	require.NoError(t, err)
	return n
}

// --- Constructor Tests ---

func TestNewNode_ValidInput_Shadowsocks(t *testing.T) {
	addr, err := vo.NewServerAddress("192.168.1.1")
	require.NoError(t, err)

	enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
	require.NoError(t, err)

	meta := vo.NewNodeMetadata("us-east", []string{"premium", "fast"}, "US East SS node")

	n, err := NewNode(
		"my-ss-node",
		addr,
		8388,
		nil,
		vo.ProtocolShadowsocks,
		enc,
		nil, nil, nil, nil, nil, nil, nil,
		meta,
		10,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_abc123"),
	)

	require.NoError(t, err)
	require.NotNil(t, n)

	assert.Equal(t, "node_abc123", n.SID())
	assert.Equal(t, "my-ss-node", n.Name())
	assert.Equal(t, "192.168.1.1", n.ServerAddress().Value())
	assert.Equal(t, uint16(8388), n.AgentPort())
	assert.Nil(t, n.SubscriptionPort())
	assert.Equal(t, uint16(8388), n.EffectiveSubscriptionPort())
	assert.Equal(t, vo.ProtocolShadowsocks, n.Protocol())
	assert.Equal(t, vo.MethodAES256GCM, n.EncryptionConfig().Method())
	assert.Equal(t, vo.NodeStatusInactive, n.Status())
	assert.Equal(t, "us-east", n.Metadata().Region())
	assert.Equal(t, 10, n.SortOrder())
	assert.Equal(t, 1, n.Version())
	assert.NotEmpty(t, n.GetAPIToken())
	assert.NotEmpty(t, n.TokenHash())
	assert.Nil(t, n.UserID())
	assert.False(t, n.IsUserOwned())
	assert.Nil(t, n.ExpiresAt())
	assert.Nil(t, n.CostLabel())
	assert.False(t, n.MuteNotification())
}

func TestNewNode_ValidInput_Trojan(t *testing.T) {
	addr, err := vo.NewServerAddress("trojan.example.com")
	require.NoError(t, err)

	trojanCfg, err := vo.NewTrojanConfig("longpassword123", "tcp", "", "", false, "trojan.example.com")
	require.NoError(t, err)

	meta := vo.NewNodeMetadata("eu-central", nil, "")

	subPort := uint16(443)
	n, err := NewNode(
		"trojan-node",
		addr,
		8443,
		&subPort,
		vo.ProtocolTrojan,
		vo.EncryptionConfig{},
		nil, &trojanCfg, nil, nil, nil, nil, nil,
		meta,
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_tro789"),
	)

	require.NoError(t, err)
	require.NotNil(t, n)

	assert.Equal(t, vo.ProtocolTrojan, n.Protocol())
	assert.NotNil(t, n.TrojanConfig())
	assert.Equal(t, uint16(8443), n.AgentPort())
	assert.Equal(t, uint16(443), n.EffectiveSubscriptionPort())
}

func TestNewNode_ValidInput_VLESS(t *testing.T) {
	addr, err := vo.NewServerAddress("vless.example.com")
	require.NoError(t, err)

	vlessCfg, err := vo.NewVLESSConfig(
		"tcp",              // transportType
		"xtls-rprx-vision", // flow
		"reality",          // security
		"example.com",      // sni
		"chrome",           // fingerprint
		false,              // allowInsecure
		"",                 // host
		"",                 // path
		"",                 // serviceName
		"privkey123",       // privateKey
		"pubkey456",        // publicKey
		"abcd1234",         // shortID
		"",                 // spiderX
	)
	require.NoError(t, err)

	meta := vo.NewNodeMetadata("ap-southeast", nil, "VLESS Reality node")

	n, err := NewNode(
		"vless-reality-node",
		addr,
		443,
		nil,
		vo.ProtocolVLESS,
		vo.EncryptionConfig{},
		nil, nil, &vlessCfg, nil, nil, nil, nil,
		meta,
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_vless001"),
	)

	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Equal(t, vo.ProtocolVLESS, n.Protocol())
	assert.NotNil(t, n.VLESSConfig())
}

func TestNewNode_ValidInput_VMess(t *testing.T) {
	addr, err := vo.NewServerAddress("vmess.example.com")
	require.NoError(t, err)

	vmessCfg, err := vo.NewVMessConfig(
		0,          // alterID
		"auto",     // security
		"ws",       // transportType
		"cdn.com",  // host
		"/vmess",   // path
		"",         // serviceName
		true,       // tls
		"cdn.com",  // sni
		false,      // allowInsecure
	)
	require.NoError(t, err)

	meta := vo.NewNodeMetadata("us-west", nil, "")

	n, err := NewNode(
		"vmess-ws-node",
		addr,
		443,
		nil,
		vo.ProtocolVMess,
		vo.EncryptionConfig{},
		nil, nil, nil, &vmessCfg, nil, nil, nil,
		meta,
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_vmess001"),
	)

	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Equal(t, vo.ProtocolVMess, n.Protocol())
	assert.NotNil(t, n.VMessConfig())
}

func TestNewNode_ValidInput_Hysteria2(t *testing.T) {
	addr, err := vo.NewServerAddress("hy2.example.com")
	require.NoError(t, err)

	hy2Cfg, err := vo.NewHysteria2Config(
		"securepass123",  // password
		"bbr",            // congestionControl
		"",               // obfs
		"",               // obfsPassword
		nil,              // upMbps
		nil,              // downMbps
		"hy2.example.com", // sni
		false,            // allowInsecure
		"chrome",         // fingerprint
	)
	require.NoError(t, err)

	meta := vo.NewNodeMetadata("jp-east", nil, "")

	n, err := NewNode(
		"hy2-node",
		addr,
		443,
		nil,
		vo.ProtocolHysteria2,
		vo.EncryptionConfig{},
		nil, nil, nil, nil, &hy2Cfg, nil, nil,
		meta,
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_hy2001"),
	)

	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Equal(t, vo.ProtocolHysteria2, n.Protocol())
	assert.NotNil(t, n.Hysteria2Config())
}

func TestNewNode_ValidInput_TUIC(t *testing.T) {
	addr, err := vo.NewServerAddress("tuic.example.com")
	require.NoError(t, err)

	tuicCfg, err := vo.NewTUICConfig(
		"some-uuid-value",    // uuid
		"some-password",      // password
		"bbr",                // congestionControl
		"native",             // udpRelayMode
		"h3",                 // alpn
		"tuic.example.com",   // sni
		false,                // allowInsecure
		false,                // disableSNI
	)
	require.NoError(t, err)

	meta := vo.NewNodeMetadata("sg", nil, "")

	n, err := NewNode(
		"tuic-node",
		addr,
		443,
		nil,
		vo.ProtocolTUIC,
		vo.EncryptionConfig{},
		nil, nil, nil, nil, nil, &tuicCfg, nil,
		meta,
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_tuic001"),
	)

	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Equal(t, vo.ProtocolTUIC, n.Protocol())
	assert.NotNil(t, n.TUICConfig())
}

func TestNewNode_InvalidProtocol(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)

	_, err = NewNode(
		"bad-protocol-node",
		addr,
		8388,
		nil,
		vo.Protocol("wireguard"), // not a valid protocol
		vo.EncryptionConfig{},
		nil, nil, nil, nil, nil, nil, nil,
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_bad"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid protocol")
}

func TestNewNode_MissingName(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)

	enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
	require.NoError(t, err)

	_, err = NewNode(
		"", // empty name
		addr,
		8388,
		nil,
		vo.ProtocolShadowsocks,
		enc,
		nil, nil, nil, nil, nil, nil, nil,
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_noname"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "node name is required")
}

func TestNewNode_MissingAgentPort(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)

	enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
	require.NoError(t, err)

	_, err = NewNode(
		"test-node",
		addr,
		0, // zero port
		nil,
		vo.ProtocolShadowsocks,
		enc,
		nil, nil, nil, nil, nil, nil, nil,
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_noport"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent port is required")
}

func TestNewNode_ShadowsocksMissingEncryption(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)

	_, err = NewNode(
		"ss-no-enc",
		addr,
		8388,
		nil,
		vo.ProtocolShadowsocks,
		vo.EncryptionConfig{}, // empty encryption config
		nil, nil, nil, nil, nil, nil, nil,
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_noenc"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "encryption config is required for Shadowsocks")
}

func TestNewNode_TrojanMissingConfig(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)

	_, err = NewNode(
		"trojan-no-cfg",
		addr,
		443,
		nil,
		vo.ProtocolTrojan,
		vo.EncryptionConfig{},
		nil, nil, nil, nil, nil, nil, nil, // trojanConfig = nil, anytlsConfig = nil
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_notrojan"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "trojan config is required for Trojan protocol")
}

func TestNewNode_VLESSMissingConfig(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)

	_, err = NewNode(
		"vless-no-cfg",
		addr,
		443,
		nil,
		vo.ProtocolVLESS,
		vo.EncryptionConfig{},
		nil, nil, nil, nil, nil, nil, nil, // vlessConfig = nil, anytlsConfig = nil
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_novless"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "vless config is required for VLESS protocol")
}

func TestNewNode_VMessMissingConfig(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)

	_, err = NewNode(
		"vmess-no-cfg",
		addr,
		443,
		nil,
		vo.ProtocolVMess,
		vo.EncryptionConfig{},
		nil, nil, nil, nil, nil, nil, nil,
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_novmess"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "vmess config is required for VMess protocol")
}

func TestNewNode_Hysteria2MissingConfig(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)

	_, err = NewNode(
		"hy2-no-cfg",
		addr,
		443,
		nil,
		vo.ProtocolHysteria2,
		vo.EncryptionConfig{},
		nil, nil, nil, nil, nil, nil, nil,
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_nohy2"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "hysteria2 config is required for Hysteria2 protocol")
}

func TestNewNode_TUICMissingConfig(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)

	_, err = NewNode(
		"tuic-no-cfg",
		addr,
		443,
		nil,
		vo.ProtocolTUIC,
		vo.EncryptionConfig{},
		nil, nil, nil, nil, nil, nil, nil,
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_notuic"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tuic config is required for TUIC protocol")
}

func TestNewNode_ValidInput_AnyTLS(t *testing.T) {
	addr, err := vo.NewServerAddress("anytls.example.com")
	require.NoError(t, err)

	anytlsCfg, err := vo.NewAnyTLSConfig("securepass123", "anytls.example.com", false, "chrome", "30s", "30s", 2)
	require.NoError(t, err)

	meta := vo.NewNodeMetadata("ap-east", nil, "anytls node")

	n, err := NewNode(
		"test-anytls-node",
		addr,
		443,
		nil,
		vo.ProtocolAnyTLS,
		vo.EncryptionConfig{},
		nil,          // pluginConfig
		nil,          // trojanConfig
		nil,          // vlessConfig
		nil,          // vmessConfig
		nil,          // hysteria2Config
		nil,          // tuicConfig
		&anytlsCfg,   // anytlsConfig
		meta,
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_anytls789"),
	)

	require.NoError(t, err)
	assert.Equal(t, "test-anytls-node", n.Name())
	assert.True(t, n.Protocol().IsAnyTLS())
	require.NotNil(t, n.AnyTLSConfig())
	assert.Equal(t, "anytls.example.com", n.AnyTLSConfig().SNI())
	assert.Equal(t, "chrome", n.AnyTLSConfig().Fingerprint())
	assert.Equal(t, 2, n.AnyTLSConfig().MinIdleSession())
}

func TestNewNode_AnyTLSMissingConfig(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)

	_, err = NewNode(
		"anytls-no-cfg",
		addr,
		443,
		nil,
		vo.ProtocolAnyTLS,
		vo.EncryptionConfig{},
		nil, nil, nil, nil, nil, nil, nil, // anytlsConfig = nil
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_noanytls"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "anytls config is required for AnyTLS protocol")
}

func TestNewNode_WithSubscriptionPort(t *testing.T) {
	n := newShadowsocksNode(t)
	// Default: no subscription port, effective = agent port
	assert.Nil(t, n.SubscriptionPort())
	assert.Equal(t, uint16(8388), n.EffectiveSubscriptionPort())
}

// --- ReconstructNode Tests ---

func TestReconstructNode_Valid(t *testing.T) {
	n := reconstructedNode(t, vo.NodeStatusActive)

	assert.Equal(t, uint(1), n.ID())
	assert.Equal(t, "node_recon001", n.SID())
	assert.Equal(t, "recon-node", n.Name())
	assert.Equal(t, vo.NodeStatusActive, n.Status())
	assert.Equal(t, 1, n.Version())
	assert.Equal(t, 1, n.OriginalVersion())
	assert.Equal(t, []uint{1, 2}, n.GroupIDs())
}

func TestReconstructNode_ZeroID(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)
	enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
	require.NoError(t, err)
	now := time.Now().UTC()

	_, err = ReconstructNode(
		0, "node_x", "name", addr, 8388, nil,
		vo.ProtocolShadowsocks, enc,
		nil, nil, nil, nil, nil, nil, nil,
		vo.NodeStatusActive,
		vo.NewNodeMetadata("", nil, ""),
		nil, nil,
		"abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
		"", 0, false, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 1, now, now,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "node ID cannot be zero")
}

func TestReconstructNode_EmptySID(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)
	enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
	require.NoError(t, err)
	now := time.Now().UTC()

	_, err = ReconstructNode(
		1, "", "name", addr, 8388, nil,
		vo.ProtocolShadowsocks, enc,
		nil, nil, nil, nil, nil, nil, nil,
		vo.NodeStatusActive,
		vo.NewNodeMetadata("", nil, ""),
		nil, nil,
		"abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
		"", 0, false, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 1, now, now,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "node SID is required")
}

func TestReconstructNode_EmptyTokenHash(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)
	enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
	require.NoError(t, err)
	now := time.Now().UTC()

	_, err = ReconstructNode(
		1, "node_x", "name", addr, 8388, nil,
		vo.ProtocolShadowsocks, enc,
		nil, nil, nil, nil, nil, nil, nil,
		vo.NodeStatusActive,
		vo.NewNodeMetadata("", nil, ""),
		nil, nil,
		"", // empty tokenHash
		"", 0, false, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 1, now, now,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token hash is required")
}

// --- State Transition Tests ---

func TestNode_Activate(t *testing.T) {
	t.Run("from inactive to active", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusInactive)
		initialVersion := n.Version()

		err := n.Activate()
		require.NoError(t, err)
		assert.Equal(t, vo.NodeStatusActive, n.Status())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("from maintenance to active", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusMaintenance)

		err := n.Activate()
		require.NoError(t, err)
		assert.Equal(t, vo.NodeStatusActive, n.Status())
	})

	t.Run("idempotent when already active", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusActive)
		initialVersion := n.Version()

		err := n.Activate()
		require.NoError(t, err)
		assert.Equal(t, vo.NodeStatusActive, n.Status())
		assert.Equal(t, initialVersion, n.Version(), "version should not change on no-op")
	})
}

func TestNode_Deactivate(t *testing.T) {
	t.Run("from active to inactive", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusActive)
		initialVersion := n.Version()

		err := n.Deactivate()
		require.NoError(t, err)
		assert.Equal(t, vo.NodeStatusInactive, n.Status())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("from maintenance to inactive", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusMaintenance)

		err := n.Deactivate()
		require.NoError(t, err)
		assert.Equal(t, vo.NodeStatusInactive, n.Status())
	})

	t.Run("idempotent when already inactive", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusInactive)
		initialVersion := n.Version()

		err := n.Deactivate()
		require.NoError(t, err)
		assert.Equal(t, vo.NodeStatusInactive, n.Status())
		assert.Equal(t, initialVersion, n.Version())
	})
}

func TestNode_EnterMaintenance(t *testing.T) {
	t.Run("from active with reason", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusActive)
		initialVersion := n.Version()

		err := n.EnterMaintenance("scheduled downtime")
		require.NoError(t, err)
		assert.Equal(t, vo.NodeStatusMaintenance, n.Status())
		require.NotNil(t, n.MaintenanceReason())
		assert.Equal(t, "scheduled downtime", *n.MaintenanceReason())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("empty reason rejected", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusActive)

		err := n.EnterMaintenance("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maintenance reason is required")
		assert.Equal(t, vo.NodeStatusActive, n.Status())
	})

	t.Run("idempotent when already in maintenance", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusMaintenance)
		initialVersion := n.Version()

		err := n.EnterMaintenance("another reason")
		require.NoError(t, err)
		assert.Equal(t, vo.NodeStatusMaintenance, n.Status())
		assert.Equal(t, initialVersion, n.Version())
	})

	t.Run("from inactive rejected", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusInactive)

		err := n.EnterMaintenance("some reason")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot enter maintenance mode from status inactive")
	})
}

func TestNode_ExitMaintenance(t *testing.T) {
	t.Run("from maintenance to active", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusMaintenance)
		initialVersion := n.Version()

		err := n.ExitMaintenance()
		require.NoError(t, err)
		assert.Equal(t, vo.NodeStatusActive, n.Status())
		assert.Nil(t, n.MaintenanceReason())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("error when not in maintenance", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusActive)

		err := n.ExitMaintenance()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "node is not in maintenance mode")
	})
}

// --- Business Logic Tests ---

func TestNode_UpdateServerAddress(t *testing.T) {
	t.Run("update to new address", func(t *testing.T) {
		n := newShadowsocksNode(t)
		initialVersion := n.Version()

		newAddr, err := vo.NewServerAddress("5.6.7.8")
		require.NoError(t, err)

		err = n.UpdateServerAddress(newAddr)
		require.NoError(t, err)
		assert.Equal(t, "5.6.7.8", n.ServerAddress().Value())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("no-op when same address", func(t *testing.T) {
		n := newShadowsocksNode(t)
		initialVersion := n.Version()

		sameAddr, err := vo.NewServerAddress("1.2.3.4")
		require.NoError(t, err)

		err = n.UpdateServerAddress(sameAddr)
		require.NoError(t, err)
		assert.Equal(t, initialVersion, n.Version())
	})

	t.Run("update to domain address", func(t *testing.T) {
		n := newShadowsocksNode(t)

		domainAddr, err := vo.NewServerAddress("proxy.example.com")
		require.NoError(t, err)

		err = n.UpdateServerAddress(domainAddr)
		require.NoError(t, err)
		assert.Equal(t, "proxy.example.com", n.ServerAddress().Value())
	})
}

func TestNode_UpdateAgentPort(t *testing.T) {
	t.Run("update to new port", func(t *testing.T) {
		n := newShadowsocksNode(t)
		initialVersion := n.Version()

		err := n.UpdateAgentPort(9999)
		require.NoError(t, err)
		assert.Equal(t, uint16(9999), n.AgentPort())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("reject zero port", func(t *testing.T) {
		n := newShadowsocksNode(t)

		err := n.UpdateAgentPort(0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent port cannot be zero")
	})

	t.Run("no-op when same port", func(t *testing.T) {
		n := newShadowsocksNode(t)
		initialVersion := n.Version()

		err := n.UpdateAgentPort(8388)
		require.NoError(t, err)
		assert.Equal(t, initialVersion, n.Version())
	})
}

func TestNode_UpdateSubscriptionPort(t *testing.T) {
	t.Run("set subscription port", func(t *testing.T) {
		n := newShadowsocksNode(t)
		initialVersion := n.Version()

		port := uint16(443)
		err := n.UpdateSubscriptionPort(&port)
		require.NoError(t, err)
		require.NotNil(t, n.SubscriptionPort())
		assert.Equal(t, uint16(443), *n.SubscriptionPort())
		assert.Equal(t, uint16(443), n.EffectiveSubscriptionPort())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("clear subscription port", func(t *testing.T) {
		n := newShadowsocksNode(t)

		port := uint16(443)
		err := n.UpdateSubscriptionPort(&port)
		require.NoError(t, err)

		initialVersion := n.Version()
		err = n.UpdateSubscriptionPort(nil)
		require.NoError(t, err)
		assert.Nil(t, n.SubscriptionPort())
		assert.Equal(t, n.AgentPort(), n.EffectiveSubscriptionPort())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("reject zero subscription port", func(t *testing.T) {
		n := newShadowsocksNode(t)

		zero := uint16(0)
		err := n.UpdateSubscriptionPort(&zero)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subscription port cannot be zero")
	})

	t.Run("no-op when both nil", func(t *testing.T) {
		n := newShadowsocksNode(t)
		initialVersion := n.Version()

		err := n.UpdateSubscriptionPort(nil)
		require.NoError(t, err)
		assert.Equal(t, initialVersion, n.Version())
	})
}

func TestNode_GroupIDManagement(t *testing.T) {
	t.Run("AddGroupID", func(t *testing.T) {
		n := newShadowsocksNode(t)

		added := n.AddGroupID(1)
		assert.True(t, added)
		assert.Equal(t, []uint{1}, n.GroupIDs())

		added = n.AddGroupID(2)
		assert.True(t, added)
		assert.Equal(t, []uint{1, 2}, n.GroupIDs())
	})

	t.Run("AddGroupID deduplication", func(t *testing.T) {
		n := newShadowsocksNode(t)

		n.AddGroupID(1)
		initialVersion := n.Version()

		added := n.AddGroupID(1) // duplicate
		assert.False(t, added)
		assert.Equal(t, []uint{1}, n.GroupIDs())
		assert.Equal(t, initialVersion, n.Version(), "version should not change on duplicate add")
	})

	t.Run("RemoveGroupID", func(t *testing.T) {
		n := newShadowsocksNode(t)
		n.AddGroupID(1)
		n.AddGroupID(2)
		n.AddGroupID(3)

		removed := n.RemoveGroupID(2)
		assert.True(t, removed)
		assert.Equal(t, []uint{1, 3}, n.GroupIDs())
	})

	t.Run("RemoveGroupID nonexistent", func(t *testing.T) {
		n := newShadowsocksNode(t)
		n.AddGroupID(1)
		initialVersion := n.Version()

		removed := n.RemoveGroupID(99)
		assert.False(t, removed)
		assert.Equal(t, initialVersion, n.Version())
	})

	t.Run("HasGroupID", func(t *testing.T) {
		n := newShadowsocksNode(t)
		n.AddGroupID(1)
		n.AddGroupID(5)

		assert.True(t, n.HasGroupID(1))
		assert.True(t, n.HasGroupID(5))
		assert.False(t, n.HasGroupID(2))
		assert.False(t, n.HasGroupID(0))
	})

	t.Run("SetGroupIDs", func(t *testing.T) {
		n := newShadowsocksNode(t)
		n.AddGroupID(1)
		n.AddGroupID(2)

		n.SetGroupIDs([]uint{10, 20, 30})
		assert.Equal(t, []uint{10, 20, 30}, n.GroupIDs())
		assert.False(t, n.HasGroupID(1))
		assert.True(t, n.HasGroupID(10))
	})

	t.Run("SetGroupIDs to nil", func(t *testing.T) {
		n := newShadowsocksNode(t)
		n.AddGroupID(1)

		n.SetGroupIDs(nil)
		assert.Nil(t, n.GroupIDs())
		assert.False(t, n.HasGroupID(1))
	})
}

func TestNode_SetUserID(t *testing.T) {
	t.Run("set user ID", func(t *testing.T) {
		n := newShadowsocksNode(t)
		assert.Nil(t, n.UserID())
		assert.False(t, n.IsUserOwned())

		userID := uint(42)
		n.SetUserID(&userID)

		require.NotNil(t, n.UserID())
		assert.Equal(t, uint(42), *n.UserID())
		assert.True(t, n.IsUserOwned())
		assert.True(t, n.IsOwnedBy(42))
		assert.False(t, n.IsOwnedBy(99))
	})

	t.Run("clear user ID", func(t *testing.T) {
		n := newShadowsocksNode(t)
		userID := uint(42)
		n.SetUserID(&userID)
		require.True(t, n.IsUserOwned())

		n.SetUserID(nil)
		assert.Nil(t, n.UserID())
		assert.False(t, n.IsUserOwned())
		assert.False(t, n.IsOwnedBy(42))
	})
}

func TestNode_SetCostLabel(t *testing.T) {
	t.Run("set cost label", func(t *testing.T) {
		n := newShadowsocksNode(t)
		assert.Nil(t, n.CostLabel())

		label := "35$/m"
		n.SetCostLabel(&label)

		require.NotNil(t, n.CostLabel())
		assert.Equal(t, "35$/m", *n.CostLabel())
	})

	t.Run("clear cost label", func(t *testing.T) {
		n := newShadowsocksNode(t)
		label := "35$/m"
		n.SetCostLabel(&label)

		n.SetCostLabel(nil)
		assert.Nil(t, n.CostLabel())
	})
}

func TestNode_IsExpired(t *testing.T) {
	t.Run("no expiration", func(t *testing.T) {
		n := newShadowsocksNode(t)
		assert.False(t, n.IsExpired())
	})

	t.Run("not expired yet", func(t *testing.T) {
		n := newShadowsocksNode(t)
		future := time.Now().UTC().Add(24 * time.Hour)
		n.SetExpiresAt(&future)
		assert.False(t, n.IsExpired())
	})

	t.Run("already expired", func(t *testing.T) {
		n := newShadowsocksNode(t)
		past := time.Now().UTC().Add(-1 * time.Hour)
		n.SetExpiresAt(&past)
		assert.True(t, n.IsExpired())
	})
}

func TestNode_IsExpiringSoon(t *testing.T) {
	t.Run("no expiration", func(t *testing.T) {
		n := newShadowsocksNode(t)
		assert.False(t, n.IsExpiringSoon(7))
	})

	t.Run("expiring within threshold", func(t *testing.T) {
		n := newShadowsocksNode(t)
		// Expires in 3 days, check for 7-day threshold
		expiresIn3Days := time.Now().UTC().Add(3 * 24 * time.Hour)
		n.SetExpiresAt(&expiresIn3Days)
		assert.True(t, n.IsExpiringSoon(7))
	})

	t.Run("not expiring soon", func(t *testing.T) {
		n := newShadowsocksNode(t)
		// Expires in 30 days, check for 7-day threshold
		expiresIn30Days := time.Now().UTC().Add(30 * 24 * time.Hour)
		n.SetExpiresAt(&expiresIn30Days)
		assert.False(t, n.IsExpiringSoon(7))
	})

	t.Run("already expired counts as expiring soon", func(t *testing.T) {
		n := newShadowsocksNode(t)
		past := time.Now().UTC().Add(-1 * time.Hour)
		n.SetExpiresAt(&past)
		// Already expired means expiresAt.Before(threshold) is true
		assert.True(t, n.IsExpiringSoon(7))
	})
}

func TestNode_IsOnline(t *testing.T) {
	t.Run("nil lastSeenAt means offline", func(t *testing.T) {
		n := newShadowsocksNode(t)
		assert.Nil(t, n.LastSeenAt())
		assert.False(t, n.IsOnline())
	})

	t.Run("recently seen means online", func(t *testing.T) {
		// Create node with recent lastSeenAt via ReconstructNode
		addr, err := vo.NewServerAddress("10.0.0.1")
		require.NoError(t, err)
		enc, err := vo.NewEncryptionConfig(vo.MethodChacha20IETFPoly1305)
		require.NoError(t, err)
		now := time.Now().UTC()
		recentTime := now.Add(-1 * time.Minute) // 1 minute ago

		n2, err := ReconstructNode(
			2, "node_online001", "online-node", addr, 8388, nil,
			vo.ProtocolShadowsocks, enc,
			nil, nil, nil, nil, nil, nil, nil,
			vo.NodeStatusActive,
			vo.NewNodeMetadata("", nil, ""),
			nil, nil,
			"abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			"", 0, false, nil, nil, nil,
			&recentTime, // lastSeenAt = 1 minute ago
			nil, nil, nil, nil, nil, nil, nil, 1, now, now,
		)
		require.NoError(t, err)
		assert.True(t, n2.IsOnline())
	})

	t.Run("stale lastSeenAt means offline", func(t *testing.T) {
		addr, err := vo.NewServerAddress("10.0.0.1")
		require.NoError(t, err)
		enc, err := vo.NewEncryptionConfig(vo.MethodChacha20IETFPoly1305)
		require.NoError(t, err)
		now := time.Now().UTC()
		staleTime := now.Add(-10 * time.Minute) // 10 minutes ago (> 5 min timeout)

		n, err := ReconstructNode(
			3, "node_stale001", "stale-node", addr, 8388, nil,
			vo.ProtocolShadowsocks, enc,
			nil, nil, nil, nil, nil, nil, nil,
			vo.NodeStatusActive,
			vo.NewNodeMetadata("", nil, ""),
			nil, nil,
			"abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			"", 0, false, nil, nil, nil,
			&staleTime, // lastSeenAt = 10 minutes ago
			nil, nil, nil, nil, nil, nil, nil, 1, now, now,
		)
		require.NoError(t, err)
		assert.False(t, n.IsOnline())
	})
}

// --- Update Name Tests ---

func TestNode_UpdateName(t *testing.T) {
	t.Run("update to new name", func(t *testing.T) {
		n := newShadowsocksNode(t)
		initialVersion := n.Version()

		err := n.UpdateName("new-node-name")
		require.NoError(t, err)
		assert.Equal(t, "new-node-name", n.Name())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("reject empty name", func(t *testing.T) {
		n := newShadowsocksNode(t)

		err := n.UpdateName("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "node name cannot be empty")
	})

	t.Run("no-op when same name", func(t *testing.T) {
		n := newShadowsocksNode(t)
		initialVersion := n.Version()

		err := n.UpdateName("test-ss-node")
		require.NoError(t, err)
		assert.Equal(t, initialVersion, n.Version())
	})
}

// --- SetID Tests ---

func TestNode_SetID(t *testing.T) {
	t.Run("set ID on new node", func(t *testing.T) {
		n := newShadowsocksNode(t)
		assert.Equal(t, uint(0), n.ID())

		err := n.SetID(42)
		require.NoError(t, err)
		assert.Equal(t, uint(42), n.ID())
	})

	t.Run("reject setting ID twice", func(t *testing.T) {
		n := newShadowsocksNode(t)
		err := n.SetID(1)
		require.NoError(t, err)

		err = n.SetID(2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "node ID is already set")
	})

	t.Run("reject zero ID", func(t *testing.T) {
		n := newShadowsocksNode(t)

		err := n.SetID(0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "node ID cannot be zero")
	})
}

// --- Availability Tests ---

func TestNode_IsAvailable(t *testing.T) {
	t.Run("active node is available", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusActive)
		assert.True(t, n.IsAvailable())
	})

	t.Run("inactive node is not available", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusInactive)
		assert.False(t, n.IsAvailable())
	})

	t.Run("maintenance node is not available", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusMaintenance)
		assert.False(t, n.IsAvailable())
	})
}

// --- MuteNotification Tests ---

func TestNode_SetMuteNotification(t *testing.T) {
	n := newShadowsocksNode(t)
	assert.False(t, n.MuteNotification())

	initialVersion := n.Version()
	n.SetMuteNotification(true)
	assert.True(t, n.MuteNotification())
	assert.Equal(t, initialVersion+1, n.Version())

	n.SetMuteNotification(false)
	assert.False(t, n.MuteNotification())
}

// --- API Token Tests ---

func TestNode_APIToken(t *testing.T) {
	t.Run("new node has plain token", func(t *testing.T) {
		n := newShadowsocksNode(t)
		assert.NotEmpty(t, n.GetAPIToken())
		assert.NotEmpty(t, n.TokenHash())
	})

	t.Run("clear token removes plain text", func(t *testing.T) {
		n := newShadowsocksNode(t)
		token := n.GetAPIToken()
		assert.NotEmpty(t, token)

		n.ClearAPIToken()
		assert.Empty(t, n.GetAPIToken())
		assert.NotEmpty(t, n.TokenHash(), "hash should remain")
	})

	t.Run("verify token succeeds with correct token", func(t *testing.T) {
		n := newShadowsocksNode(t)
		token := n.GetAPIToken()

		assert.True(t, n.VerifyAPIToken(token))
	})

	t.Run("verify token fails with wrong token", func(t *testing.T) {
		n := newShadowsocksNode(t)

		assert.False(t, n.VerifyAPIToken("node_wrongtoken"))
	})

	t.Run("generate new token", func(t *testing.T) {
		n := newShadowsocksNode(t)
		oldToken := n.GetAPIToken()
		oldHash := n.TokenHash()
		initialVersion := n.Version()

		newToken, err := n.GenerateAPIToken()
		require.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, oldToken, newToken)
		assert.NotEqual(t, oldHash, n.TokenHash())
		assert.Equal(t, initialVersion+1, n.Version())

		// New token should verify
		assert.True(t, n.VerifyAPIToken(newToken))
		// Old token should no longer verify
		assert.False(t, n.VerifyAPIToken(oldToken))
	})
}

// --- EffectiveServerAddress Tests ---

func TestNode_EffectiveServerAddress(t *testing.T) {
	t.Run("returns server address when set", func(t *testing.T) {
		n := newShadowsocksNode(t)
		assert.Equal(t, "1.2.3.4", n.EffectiveServerAddress())
	})

	t.Run("falls back to publicIPv4", func(t *testing.T) {
		addr, err := vo.NewServerAddress("") // empty address
		require.NoError(t, err)
		enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
		require.NoError(t, err)
		now := time.Now().UTC()
		ipv4 := "203.0.113.1"

		n, err := ReconstructNode(
			4, "node_fb001", "fallback-node", addr, 8388, nil,
			vo.ProtocolShadowsocks, enc,
			nil, nil, nil, nil, nil, nil, nil,
			vo.NodeStatusActive,
			vo.NewNodeMetadata("", nil, ""),
			nil, nil,
			"abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			"", 0, false, nil, nil, nil, nil,
			&ipv4, // publicIPv4
			nil, nil, nil, nil, nil, nil, 1, now, now,
		)
		require.NoError(t, err)
		assert.Equal(t, "203.0.113.1", n.EffectiveServerAddress())
	})

	t.Run("returns empty when nothing set", func(t *testing.T) {
		addr, err := vo.NewServerAddress("") // empty address
		require.NoError(t, err)
		enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
		require.NoError(t, err)
		now := time.Now().UTC()

		n, err := ReconstructNode(
			5, "node_empty001", "empty-addr-node", addr, 8388, nil,
			vo.ProtocolShadowsocks, enc,
			nil, nil, nil, nil, nil, nil, nil,
			vo.NodeStatusActive,
			vo.NewNodeMetadata("", nil, ""),
			nil, nil,
			"abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			"", 0, false, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 1, now, now,
		)
		require.NoError(t, err)
		assert.Equal(t, "", n.EffectiveServerAddress())
	})
}

// --- SortOrder Tests ---

func TestNode_UpdateSortOrder(t *testing.T) {
	n := newShadowsocksNode(t)
	initialVersion := n.Version()

	err := n.UpdateSortOrder(100)
	require.NoError(t, err)
	assert.Equal(t, 100, n.SortOrder())
	assert.Equal(t, initialVersion+1, n.Version())
}

// --- RouteConfig Tests ---

func TestNode_RouteConfig(t *testing.T) {
	t.Run("new node has no route config", func(t *testing.T) {
		n := newShadowsocksNode(t)
		assert.False(t, n.HasRouteConfig())
		assert.Nil(t, n.RouteConfig())
	})

	t.Run("update route config", func(t *testing.T) {
		n := newShadowsocksNode(t)
		initialVersion := n.Version()

		rc, err := vo.NewRouteConfig(vo.OutboundProxy)
		require.NoError(t, err)

		err = n.UpdateRouteConfig(rc)
		require.NoError(t, err)
		assert.True(t, n.HasRouteConfig())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("clear route config", func(t *testing.T) {
		n := newShadowsocksNode(t)
		rc, err := vo.NewRouteConfig(vo.OutboundProxy)
		require.NoError(t, err)
		err = n.UpdateRouteConfig(rc)
		require.NoError(t, err)

		initialVersion := n.Version()
		n.ClearRouteConfig()
		assert.False(t, n.HasRouteConfig())
		assert.Nil(t, n.RouteConfig())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("update route config to nil", func(t *testing.T) {
		n := newShadowsocksNode(t)
		initialVersion := n.Version()

		err := n.UpdateRouteConfig(nil)
		require.NoError(t, err)
		assert.Equal(t, initialVersion+1, n.Version())
	})
}

// --- Protocol-specific config update validation ---

func TestNode_UpdateTrojanConfig_ProtocolMismatch(t *testing.T) {
	n := newShadowsocksNode(t) // Shadowsocks node

	trojanCfg, err := vo.NewTrojanConfig("longpassword123", "tcp", "", "", false, "example.com")
	require.NoError(t, err)

	err = n.UpdateTrojanConfig(&trojanCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update trojan config for non-trojan protocol")
}

func TestNode_UpdateVLESSConfig_ProtocolMismatch(t *testing.T) {
	n := newShadowsocksNode(t)

	vlessCfg, err := vo.NewVLESSConfig("tcp", "", "none", "", "", false, "", "", "", "", "", "", "")
	require.NoError(t, err)

	err = n.UpdateVLESSConfig(&vlessCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update vless config for non-vless protocol")
}

func TestNode_UpdateVMessConfig_ProtocolMismatch(t *testing.T) {
	n := newShadowsocksNode(t)

	vmessCfg, err := vo.NewVMessConfig(0, "auto", "tcp", "", "", "", false, "", false)
	require.NoError(t, err)

	err = n.UpdateVMessConfig(&vmessCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update vmess config for non-vmess protocol")
}

func TestNode_UpdateHysteria2Config_ProtocolMismatch(t *testing.T) {
	n := newShadowsocksNode(t)

	hy2Cfg, err := vo.NewHysteria2Config("securepass123", "bbr", "", "", nil, nil, "", false, "")
	require.NoError(t, err)

	err = n.UpdateHysteria2Config(&hy2Cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update hysteria2 config for non-hysteria2 protocol")
}

func TestNode_UpdateTUICConfig_ProtocolMismatch(t *testing.T) {
	n := newShadowsocksNode(t)

	tuicCfg, err := vo.NewTUICConfig("uuid", "pass", "bbr", "native", "", "", false, false)
	require.NoError(t, err)

	err = n.UpdateTUICConfig(&tuicCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update tuic config for non-tuic protocol")
}

func TestNode_UpdateAnyTLSConfig_ProtocolMismatch(t *testing.T) {
	n := newShadowsocksNode(t) // Shadowsocks node

	anytlsCfg, err := vo.NewAnyTLSConfig("securepass123", "example.com", false, "chrome", "", "", 0)
	require.NoError(t, err)

	err = n.UpdateAnyTLSConfig(&anytlsCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update anytls config for non-anytls protocol")
}

func TestNode_UpdateAnyTLSConfig_CorrectProtocol(t *testing.T) {
	addr, err := vo.NewServerAddress("anytls.example.com")
	require.NoError(t, err)

	anytlsCfg, err := vo.NewAnyTLSConfig("securepass123", "anytls.example.com", false, "chrome", "", "", 0)
	require.NoError(t, err)

	n, err := NewNode(
		"test-anytls-update",
		addr,
		443,
		nil,
		vo.ProtocolAnyTLS,
		vo.EncryptionConfig{},
		nil, nil, nil, nil, nil, nil, &anytlsCfg,
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_anytlsupd"),
	)
	require.NoError(t, err)
	initialVersion := n.Version()

	newCfg, err := vo.NewAnyTLSConfig("newlongpassword1", "new.example.com", true, "firefox", "60s", "120s", 5)
	require.NoError(t, err)

	err = n.UpdateAnyTLSConfig(&newCfg)
	require.NoError(t, err)
	assert.Equal(t, initialVersion+1, n.Version())
	require.NotNil(t, n.AnyTLSConfig())
	assert.Equal(t, "new.example.com", n.AnyTLSConfig().SNI())
	assert.Equal(t, "firefox", n.AnyTLSConfig().Fingerprint())
	assert.Equal(t, 5, n.AnyTLSConfig().MinIdleSession())
	assert.True(t, n.AnyTLSConfig().AllowInsecure())
}

func TestNode_UpdateTrojanConfig_CorrectProtocol(t *testing.T) {
	n := newTrojanNode(t)
	initialVersion := n.Version()

	newCfg, err := vo.NewTrojanConfig("newlongpassword", "ws", "ws.example.com", "/ws", false, "ws.example.com")
	require.NoError(t, err)

	err = n.UpdateTrojanConfig(&newCfg)
	require.NoError(t, err)
	assert.Equal(t, initialVersion+1, n.Version())
	require.NotNil(t, n.TrojanConfig())
	assert.Equal(t, "ws", n.TrojanConfig().TransportProtocol())
}

// --- Validate Tests ---

func TestNode_Validate(t *testing.T) {
	t.Run("valid node passes validation", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusActive)
		err := n.Validate()
		assert.NoError(t, err)
	})

	t.Run("maintenance without reason fails validation", func(t *testing.T) {
		addr, err := vo.NewServerAddress("1.2.3.4")
		require.NoError(t, err)
		enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
		require.NoError(t, err)
		now := time.Now().UTC()

		// Force create a node in maintenance without a reason (via reconstruct)
		n, err := ReconstructNode(
			10, "node_val001", "val-node", addr, 8388, nil,
			vo.ProtocolShadowsocks, enc,
			nil, nil, nil, nil, nil, nil, nil,
			vo.NodeStatusMaintenance, // maintenance status
			vo.NewNodeMetadata("", nil, ""),
			nil, nil,
			"abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
			"", 0, false,
			nil, // maintenanceReason = nil (invalid!)
			nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 1, now, now,
		)
		require.NoError(t, err, "ReconstructNode does not validate maintenance reason")

		err = n.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maintenance reason is required when in maintenance mode")
	})
}

// --- SetExpiresAt Tests ---

func TestNode_SetExpiresAt(t *testing.T) {
	t.Run("set expiration", func(t *testing.T) {
		n := newShadowsocksNode(t)
		initialVersion := n.Version()

		future := time.Now().UTC().Add(30 * 24 * time.Hour)
		n.SetExpiresAt(&future)

		require.NotNil(t, n.ExpiresAt())
		assert.Equal(t, future, *n.ExpiresAt())
		assert.Equal(t, initialVersion+1, n.Version())
	})

	t.Run("clear expiration", func(t *testing.T) {
		n := newShadowsocksNode(t)
		future := time.Now().UTC().Add(30 * 24 * time.Hour)
		n.SetExpiresAt(&future)

		n.SetExpiresAt(nil)
		assert.Nil(t, n.ExpiresAt())
		assert.False(t, n.IsExpired())
	})
}

// --- Agent Info Tests ---

func TestNode_AgentInfo(t *testing.T) {
	addr, err := vo.NewServerAddress("10.0.0.1")
	require.NoError(t, err)
	enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
	require.NoError(t, err)
	now := time.Now().UTC()
	ipv4 := "203.0.113.1"
	ipv6 := "2001:db8::1"
	agentVersion := "1.5.0"
	platform := "linux"
	arch := "amd64"

	n, err := ReconstructNode(
		6, "node_agent001", "agent-info-node", addr, 8388, nil,
		vo.ProtocolShadowsocks, enc,
		nil, nil, nil, nil, nil, nil, nil,
		vo.NodeStatusActive,
		vo.NewNodeMetadata("", nil, ""),
		nil, nil,
		"abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
		"", 0, false, nil, nil, nil, nil,
		&ipv4, &ipv6, &agentVersion, &platform, &arch,
		nil, nil, 1, now, now,
	)
	require.NoError(t, err)

	require.NotNil(t, n.PublicIPv4())
	assert.Equal(t, "203.0.113.1", *n.PublicIPv4())

	require.NotNil(t, n.PublicIPv6())
	assert.Equal(t, "2001:db8::1", *n.PublicIPv6())

	require.NotNil(t, n.AgentVersion())
	assert.Equal(t, "1.5.0", *n.AgentVersion())

	require.NotNil(t, n.AgentPlatform())
	assert.Equal(t, "linux", *n.AgentPlatform())

	require.NotNil(t, n.AgentArch())
	assert.Equal(t, "amd64", *n.AgentArch())
}

// --- Version Tracking Tests ---

func TestNode_VersionTracking(t *testing.T) {
	t.Run("multiple mutations increment version", func(t *testing.T) {
		n := newShadowsocksNode(t)
		assert.Equal(t, 1, n.Version())

		n.AddGroupID(1) // version 2
		assert.Equal(t, 2, n.Version())

		n.AddGroupID(2) // version 3
		assert.Equal(t, 3, n.Version())

		err := n.Activate()
		require.NoError(t, err) // version 4
		assert.Equal(t, 4, n.Version())

		label := "10$/m"
		n.SetCostLabel(&label) // version 5
		assert.Equal(t, 5, n.Version())
	})

	t.Run("original version preserved for optimistic locking", func(t *testing.T) {
		n := reconstructedNode(t, vo.NodeStatusActive)
		assert.Equal(t, 1, n.OriginalVersion())

		// Mutate several times
		n.AddGroupID(10)
		n.AddGroupID(20)
		err := n.UpdateName("mutated-name")
		require.NoError(t, err)

		// Original version should remain unchanged
		assert.Equal(t, 1, n.OriginalVersion())
		assert.Greater(t, n.Version(), n.OriginalVersion())
	})
}

// --- Empty ServerAddress (fallback to publicIP) ---

func TestNewNode_EmptyServerAddress(t *testing.T) {
	addr, err := vo.NewServerAddress("") // empty is allowed
	require.NoError(t, err)

	enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
	require.NoError(t, err)

	n, err := NewNode(
		"empty-addr-node",
		addr,
		8388,
		nil,
		vo.ProtocolShadowsocks,
		enc,
		nil, nil, nil, nil, nil, nil, nil,
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil, // routeConfig
		nil, // dnsConfig
		fakeSIDGenerator("node_emptyaddr"),
	)

	require.NoError(t, err)
	assert.Equal(t, "", n.ServerAddress().Value())
	assert.Equal(t, "", n.EffectiveServerAddress())
}

// --- DnsConfig Tests ---

// validDnsConfig creates a valid DnsConfig for node-level testing.
func validDnsConfig(t *testing.T) *vo.DnsConfig {
	t.Helper()
	s1, err := vo.NewDnsServer("remote", "https://1.1.1.1/dns-query")
	require.NoError(t, err)
	s1.WithDetour("proxy")

	s2, err := vo.NewDnsServer("local", "223.5.5.5")
	require.NoError(t, err)
	s2.WithDetour("direct")

	config, err := vo.NewDnsConfig("remote")
	require.NoError(t, err)
	require.NoError(t, config.SetServers([]vo.DnsServer{*s1, *s2}))

	rule, err := vo.NewDnsRule("local")
	require.NoError(t, err)
	rule.WithGeosite("cn")
	require.NoError(t, config.SetRules([]vo.DnsRule{*rule}))

	require.NoError(t, config.Validate())
	return config
}

func TestNewNode_WithDnsConfig(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)
	enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
	require.NoError(t, err)

	dnsConfig := validDnsConfig(t)

	n, err := NewNode(
		"dns-node",
		addr,
		8388,
		nil,
		vo.ProtocolShadowsocks,
		enc,
		nil, nil, nil, nil, nil, nil, nil,
		vo.NewNodeMetadata("us-west", nil, ""),
		0,
		nil,       // routeConfig
		dnsConfig, // dnsConfig
		fakeSIDGenerator("node_dns001"),
	)
	require.NoError(t, err)
	require.NotNil(t, n)

	require.NotNil(t, n.DnsConfig())
	assert.Equal(t, "remote", n.DnsConfig().Final())
	assert.Len(t, n.DnsConfig().Servers(), 2)
	assert.Len(t, n.DnsConfig().Rules(), 1)
}

func TestNewNode_WithInvalidDnsConfig(t *testing.T) {
	addr, err := vo.NewServerAddress("1.2.3.4")
	require.NoError(t, err)
	enc, err := vo.NewEncryptionConfig(vo.MethodAES256GCM)
	require.NoError(t, err)

	// Create a DnsConfig with final referencing a non-existent server
	// Use Reconstruct to bypass normal validation
	invalidConfig := vo.ReconstructDnsConfig(
		nil,      // no servers
		nil,      // no rules
		"remote", // final references undefined server
		"",
		false, false, false, false,
	)

	_, err = NewNode(
		"dns-invalid-node",
		addr,
		8388,
		nil,
		vo.ProtocolShadowsocks,
		enc,
		nil, nil, nil, nil, nil, nil, nil,
		vo.NewNodeMetadata("", nil, ""),
		0,
		nil,
		invalidConfig,
		fakeSIDGenerator("node_dnsinvalid"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dns config")
}

func TestNode_UpdateDnsConfig(t *testing.T) {
	n := newShadowsocksNode(t)
	assert.Nil(t, n.DnsConfig())
	initialVersion := n.Version()

	dnsConfig := validDnsConfig(t)
	err := n.UpdateDnsConfig(dnsConfig)
	require.NoError(t, err)

	require.NotNil(t, n.DnsConfig())
	assert.Equal(t, "remote", n.DnsConfig().Final())
	assert.Equal(t, initialVersion+1, n.Version())
}

func TestNode_UpdateDnsConfig_Invalid(t *testing.T) {
	n := newShadowsocksNode(t)

	// Create an invalid DnsConfig (final references undefined server)
	invalidConfig := vo.ReconstructDnsConfig(
		nil,
		nil,
		"nonexistent",
		"",
		false, false, false, false,
	)

	err := n.UpdateDnsConfig(invalidConfig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dns config")
	assert.Nil(t, n.DnsConfig(), "dns config should remain nil after failed update")
}

func TestNode_UpdateDnsConfig_Nil(t *testing.T) {
	n := newShadowsocksNode(t)
	dnsConfig := validDnsConfig(t)
	err := n.UpdateDnsConfig(dnsConfig)
	require.NoError(t, err)
	require.NotNil(t, n.DnsConfig())

	initialVersion := n.Version()

	// Passing nil should clear the config
	err = n.UpdateDnsConfig(nil)
	require.NoError(t, err)
	assert.Nil(t, n.DnsConfig())
	assert.Equal(t, initialVersion+1, n.Version())
}

func TestNode_ClearDnsConfig(t *testing.T) {
	n := newShadowsocksNode(t)

	// Set DNS config first
	dnsConfig := validDnsConfig(t)
	err := n.UpdateDnsConfig(dnsConfig)
	require.NoError(t, err)
	require.NotNil(t, n.DnsConfig())

	initialVersion := n.Version()

	// Clear it
	n.ClearDnsConfig()
	assert.Nil(t, n.DnsConfig())
	assert.Equal(t, initialVersion+1, n.Version())
}

// --- Metadata Tests ---

func TestNode_UpdateMetadata(t *testing.T) {
	n := newShadowsocksNode(t)
	initialVersion := n.Version()

	newMeta := vo.NewNodeMetadata("eu-central", []string{"budget", "stable"}, "updated description")
	err := n.UpdateMetadata(newMeta)
	require.NoError(t, err)

	assert.Equal(t, "eu-central", n.Metadata().Region())
	assert.Equal(t, []string{"budget", "stable"}, n.Metadata().Tags())
	assert.Equal(t, "updated description", n.Metadata().Description())
	assert.Equal(t, initialVersion+1, n.Version())
}

// --- Encryption Config Update Tests ---

func TestNode_UpdateEncryption(t *testing.T) {
	n := newShadowsocksNode(t)
	initialVersion := n.Version()

	newEnc, err := vo.NewEncryptionConfig(vo.MethodChacha20IETFPoly1305)
	require.NoError(t, err)

	err = n.UpdateEncryption(newEnc)
	require.NoError(t, err)
	assert.Equal(t, vo.MethodChacha20IETFPoly1305, n.EncryptionConfig().Method())
	assert.Equal(t, initialVersion+1, n.Version())
}

// --- Plugin Config Update Tests ---

func TestNode_UpdatePlugin(t *testing.T) {
	n := newShadowsocksNode(t)
	initialVersion := n.Version()

	plugin, err := vo.NewObfsPlugin("http")
	require.NoError(t, err)

	err = n.UpdatePlugin(plugin)
	require.NoError(t, err)
	require.NotNil(t, n.PluginConfig())
	assert.Equal(t, "obfs-local", n.PluginConfig().Plugin())
	assert.Equal(t, initialVersion+1, n.Version())
}

// --- CreatedAt/UpdatedAt Tests ---

func TestNode_Timestamps(t *testing.T) {
	n := newShadowsocksNode(t)
	createdAt := n.CreatedAt()
	updatedAt := n.UpdatedAt()

	assert.False(t, createdAt.IsZero())
	assert.False(t, updatedAt.IsZero())

	// After a mutation, updatedAt should advance (or stay same within clock resolution)
	err := n.UpdateName("timestamp-test")
	require.NoError(t, err)
	assert.True(t, n.UpdatedAt().After(updatedAt) || n.UpdatedAt().Equal(updatedAt))
	assert.Equal(t, createdAt, n.CreatedAt(), "createdAt must not change")
}
