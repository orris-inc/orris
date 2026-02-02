package nodeutil

import (
	"testing"
	"time"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
)

func TestNewForwardedNodeBuilder(t *testing.T) {
	agentMap := map[uint]*models.ForwardAgentModel{
		1: {ID: 1, PublicAddress: "1.2.3.4"},
	}
	configs := NewProtocolConfigs()

	builder := NewForwardedNodeBuilder(agentMap, configs)

	if builder == nil {
		t.Fatal("NewForwardedNodeBuilder returned nil")
	}
	if builder.agentMap == nil {
		t.Error("agentMap should be set")
	}
	if len(builder.agentMap) != 1 {
		t.Errorf("agentMap length = %d, want 1", len(builder.agentMap))
	}
}

func TestForwardedNodeBuilder_ResolveRuleServerAddress(t *testing.T) {
	agentMap := map[uint]*models.ForwardAgentModel{
		1: {ID: 1, PublicAddress: "agent1.example.com"},
		2: {ID: 2, PublicAddress: "agent2.example.com"},
		3: {ID: 3, PublicAddress: ""},
	}
	builder := NewForwardedNodeBuilder(agentMap, NewProtocolConfigs())

	tests := []struct {
		name string
		rule *forward.ForwardRule
		want string
	}{
		{
			name: "external rule returns server address",
			rule: mustCreateExternalRule(t, "external.example.com", 8080, "test-external", 1),
			want: "external.example.com",
		},
		{
			name: "internal rule returns agent public address",
			rule: mustCreateDirectRule(t, 1, "test-direct", 8080, "target.example.com", 443),
			want: "agent1.example.com",
		},
		{
			name: "internal rule with different agent",
			rule: mustCreateDirectRule(t, 2, "test-direct-2", 8081, "target2.example.com", 443),
			want: "agent2.example.com",
		},
		{
			name: "internal rule with agent not in map returns empty",
			rule: mustCreateDirectRule(t, 99, "test-direct-missing", 8082, "target3.example.com", 443),
			want: "",
		},
		{
			name: "internal rule with agent having empty public address returns empty",
			rule: mustCreateDirectRule(t, 3, "test-direct-empty", 8083, "target4.example.com", 443),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.ResolveRuleServerAddress(tt.rule)
			if got != tt.want {
				t.Errorf("ResolveRuleServerAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestForwardedNodeBuilder_BuildFromNodeModel(t *testing.T) {
	agentMap := map[uint]*models.ForwardAgentModel{
		1: {ID: 1, PublicAddress: "agent.example.com"},
	}

	tests := []struct {
		name       string
		configs    ProtocolConfigs
		rule       *forward.ForwardRule
		targetNode *models.NodeModel
		wantNil    bool
		wantNode   *usecases.Node
	}{
		{
			name:    "build node from model with shadowsocks config",
			configs: newShadowsocksConfigs(10, "aes-256-gcm"),
			rule:    mustCreateDirectRuleWithTargetNode(t, 1, "forward-node", 9000, 10),
			targetNode: &models.NodeModel{
				ID:        10,
				Name:      "original-node",
				Protocol:  "shadowsocks",
				TokenHash: "hash10",
			},
			wantNil: false,
			wantNode: &usecases.Node{
				ID:               10,
				Name:             "forward-node",
				ServerAddress:    "agent.example.com",
				SubscriptionPort: 9000,
				Protocol:         "shadowsocks",
				TokenHash:        "hash10",
				EncryptionMethod: "aes-256-gcm",
			},
		},
		{
			name:    "build node from model with trojan config",
			configs: newTrojanConfigs(20, "ws", "example.com", "/ws"),
			rule:    mustCreateDirectRuleWithTargetNode(t, 1, "trojan-forward", 9001, 20),
			targetNode: &models.NodeModel{
				ID:        20,
				Name:      "original-trojan",
				Protocol:  "trojan",
				TokenHash: "hash20",
			},
			wantNil: false,
			wantNode: &usecases.Node{
				ID:                20,
				Name:              "trojan-forward",
				ServerAddress:     "agent.example.com",
				SubscriptionPort:  9001,
				Protocol:          "trojan",
				TokenHash:         "hash20",
				TransportProtocol: "ws",
				Host:              "example.com",
				Path:              "/ws",
			},
		},
		{
			name:       "nil rule target node ID returns nil",
			configs:    NewProtocolConfigs(),
			rule:       mustCreateDirectRule(t, 1, "no-target", 9002, "direct.example.com", 443),
			targetNode: &models.NodeModel{ID: 30},
			wantNil:    true,
		},
		{
			name:       "nil target node returns nil",
			configs:    NewProtocolConfigs(),
			rule:       mustCreateDirectRuleWithTargetNode(t, 1, "nil-target", 9003, 40),
			targetNode: nil,
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewForwardedNodeBuilder(agentMap, tt.configs)
			got := builder.BuildFromNodeModel(tt.rule, tt.targetNode)

			if tt.wantNil {
				if got != nil {
					t.Errorf("BuildFromNodeModel() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("BuildFromNodeModel() returned nil, want non-nil")
			}

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
			if got.EncryptionMethod != tt.wantNode.EncryptionMethod {
				t.Errorf("EncryptionMethod = %q, want %q", got.EncryptionMethod, tt.wantNode.EncryptionMethod)
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
		})
	}
}

func TestForwardedNodeBuilder_BuildFromNodeModel_EmptyServerAddress(t *testing.T) {
	// Agent with empty public address
	agentMap := map[uint]*models.ForwardAgentModel{
		1: {ID: 1, PublicAddress: ""},
	}
	builder := NewForwardedNodeBuilder(agentMap, NewProtocolConfigs())

	rule := mustCreateDirectRuleWithTargetNode(t, 1, "test", 9000, 10)
	targetNode := &models.NodeModel{ID: 10, Protocol: "shadowsocks", TokenHash: "hash"}

	got := builder.BuildFromNodeModel(rule, targetNode)
	if got != nil {
		t.Error("BuildFromNodeModel() should return nil when server address cannot be resolved")
	}
}

func TestForwardedNodeBuilder_BuildFromUsecaseNode(t *testing.T) {
	agentMap := map[uint]*models.ForwardAgentModel{
		1: {ID: 1, PublicAddress: "agent.example.com"},
	}
	builder := NewForwardedNodeBuilder(agentMap, NewProtocolConfigs())

	tests := []struct {
		name         string
		rule         *forward.ForwardRule
		originalNode *usecases.Node
		wantNil      bool
		wantNode     *usecases.Node
	}{
		{
			name: "build from usecase node copies protocol fields",
			rule: mustCreateDirectRuleWithTargetNode(t, 1, "forwarded-name", 9000, 10),
			originalNode: &usecases.Node{
				ID:                10,
				Name:              "original-name",
				ServerAddress:     "original.example.com",
				SubscriptionPort:  443,
				Protocol:          "shadowsocks",
				TokenHash:         "original-hash",
				Password:          "original-password",
				EncryptionMethod:  "aes-256-gcm",
				Plugin:            "obfs-local",
				PluginOpts:        map[string]string{"mode": "tls"},
				TransportProtocol: "",
				SortOrder:         100,
			},
			wantNil: false,
			wantNode: &usecases.Node{
				ID:               10,
				Name:             "forwarded-name",
				ServerAddress:    "agent.example.com",
				SubscriptionPort: 9000,
				Protocol:         "shadowsocks",
				TokenHash:        "original-hash",
				Password:         "original-password",
				EncryptionMethod: "aes-256-gcm",
				Plugin:           "obfs-local",
				PluginOpts:       map[string]string{"mode": "tls"},
			},
		},
		{
			name: "build from usecase node with trojan protocol",
			rule: mustCreateDirectRuleWithTargetNode(t, 1, "trojan-forwarded", 9001, 20),
			originalNode: &usecases.Node{
				ID:                20,
				Name:              "original-trojan",
				ServerAddress:     "original-trojan.example.com",
				SubscriptionPort:  443,
				Protocol:          "trojan",
				TokenHash:         "trojan-hash",
				Password:          "trojan-password",
				TransportProtocol: "ws",
				Host:              "ws.example.com",
				Path:              "/ws",
				SNI:               "sni.example.com",
				AllowInsecure:     true,
				SortOrder:         200,
			},
			wantNil: false,
			wantNode: &usecases.Node{
				ID:                20,
				Name:              "trojan-forwarded",
				ServerAddress:     "agent.example.com",
				SubscriptionPort:  9001,
				Protocol:          "trojan",
				TokenHash:         "trojan-hash",
				Password:          "trojan-password",
				TransportProtocol: "ws",
				Host:              "ws.example.com",
				Path:              "/ws",
				SNI:               "sni.example.com",
				AllowInsecure:     true,
			},
		},
		{
			name:         "nil original node returns nil",
			rule:         mustCreateDirectRuleWithTargetNode(t, 1, "test", 9002, 30),
			originalNode: nil,
			wantNil:      true,
		},
		{
			name:         "rule without target node ID returns nil",
			rule:         mustCreateDirectRule(t, 1, "no-target", 9003, "target.example.com", 443),
			originalNode: &usecases.Node{ID: 40},
			wantNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.BuildFromUsecaseNode(tt.rule, tt.originalNode)

			if tt.wantNil {
				if got != nil {
					t.Errorf("BuildFromUsecaseNode() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("BuildFromUsecaseNode() returned nil, want non-nil")
			}

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
			if got.Password != tt.wantNode.Password {
				t.Errorf("Password = %q, want %q", got.Password, tt.wantNode.Password)
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

func TestForwardedNodeBuilder_BuildForwardedNodesFromModels(t *testing.T) {
	agentMap := map[uint]*models.ForwardAgentModel{
		1: {ID: 1, PublicAddress: "agent1.example.com"},
		2: {ID: 2, PublicAddress: "agent2.example.com"},
	}
	configs := newShadowsocksConfigs(10, "aes-256-gcm")
	configs.Shadowsocks[20] = &models.ShadowsocksConfigModel{
		NodeID:           20,
		EncryptionMethod: "chacha20-ietf-poly1305",
	}

	builder := NewForwardedNodeBuilder(agentMap, configs)

	nodeMap := map[uint]*models.NodeModel{
		10: {ID: 10, Name: "node10", Protocol: "shadowsocks", TokenHash: "hash10"},
		20: {ID: 20, Name: "node20", Protocol: "shadowsocks", TokenHash: "hash20"},
		30: {ID: 30, Name: "node30", Protocol: "shadowsocks", TokenHash: "hash30"},
	}

	rules := []*forward.ForwardRule{
		mustCreateDirectRuleWithTargetNode(t, 1, "rule1", 9001, 10),
		mustCreateDirectRuleWithTargetNode(t, 2, "rule2", 9002, 20),
		mustCreateDirectRule(t, 1, "rule3-no-target", 9003, "target.example.com", 443), // no target node ID
		mustCreateDirectRuleWithTargetNode(t, 1, "rule4-missing", 9004, 99),            // target not in map
	}

	nodes := builder.BuildForwardedNodesFromModels(rules, nodeMap)

	if len(nodes) != 2 {
		t.Fatalf("BuildForwardedNodesFromModels() returned %d nodes, want 2", len(nodes))
	}

	// Verify first node
	if nodes[0].Name != "rule1" {
		t.Errorf("nodes[0].Name = %q, want %q", nodes[0].Name, "rule1")
	}
	if nodes[0].ServerAddress != "agent1.example.com" {
		t.Errorf("nodes[0].ServerAddress = %q, want %q", nodes[0].ServerAddress, "agent1.example.com")
	}
	if nodes[0].SubscriptionPort != 9001 {
		t.Errorf("nodes[0].SubscriptionPort = %d, want %d", nodes[0].SubscriptionPort, 9001)
	}
	if nodes[0].EncryptionMethod != "aes-256-gcm" {
		t.Errorf("nodes[0].EncryptionMethod = %q, want %q", nodes[0].EncryptionMethod, "aes-256-gcm")
	}

	// Verify second node
	if nodes[1].Name != "rule2" {
		t.Errorf("nodes[1].Name = %q, want %q", nodes[1].Name, "rule2")
	}
	if nodes[1].ServerAddress != "agent2.example.com" {
		t.Errorf("nodes[1].ServerAddress = %q, want %q", nodes[1].ServerAddress, "agent2.example.com")
	}
	if nodes[1].SubscriptionPort != 9002 {
		t.Errorf("nodes[1].SubscriptionPort = %d, want %d", nodes[1].SubscriptionPort, 9002)
	}
	if nodes[1].EncryptionMethod != "chacha20-ietf-poly1305" {
		t.Errorf("nodes[1].EncryptionMethod = %q, want %q", nodes[1].EncryptionMethod, "chacha20-ietf-poly1305")
	}
}

func TestForwardedNodeBuilder_BuildForwardedNodesFromModels_EmptyInput(t *testing.T) {
	builder := NewForwardedNodeBuilder(nil, NewProtocolConfigs())

	// Empty rules
	nodes := builder.BuildForwardedNodesFromModels(nil, nil)
	if len(nodes) != 0 {
		t.Errorf("BuildForwardedNodesFromModels(nil, nil) returned %d nodes, want 0", len(nodes))
	}

	// Empty nodeMap
	rules := []*forward.ForwardRule{
		mustCreateDirectRuleWithTargetNode(t, 1, "rule1", 9001, 10),
	}
	nodes = builder.BuildForwardedNodesFromModels(rules, map[uint]*models.NodeModel{})
	if len(nodes) != 0 {
		t.Errorf("BuildForwardedNodesFromModels with empty nodeMap returned %d nodes, want 0", len(nodes))
	}
}

func TestForwardedNodeBuilder_BuildForwardedNodesFromUsecaseNodes(t *testing.T) {
	agentMap := map[uint]*models.ForwardAgentModel{
		1: {ID: 1, PublicAddress: "agent1.example.com"},
		2: {ID: 2, PublicAddress: "agent2.example.com"},
	}
	builder := NewForwardedNodeBuilder(agentMap, NewProtocolConfigs())

	nodeMap := map[uint]*usecases.Node{
		10: {
			ID:               10,
			Name:             "original10",
			Protocol:         "shadowsocks",
			TokenHash:        "hash10",
			Password:         "pass10",
			EncryptionMethod: "aes-256-gcm",
		},
		20: {
			ID:                20,
			Name:              "original20",
			Protocol:          "trojan",
			TokenHash:         "hash20",
			Password:          "pass20",
			TransportProtocol: "ws",
			Host:              "ws.example.com",
		},
	}

	rules := []*forward.ForwardRule{
		mustCreateDirectRuleWithTargetNode(t, 1, "forward1", 9001, 10),
		mustCreateDirectRuleWithTargetNode(t, 2, "forward2", 9002, 20),
		mustCreateDirectRuleWithTargetNode(t, 1, "forward3-missing", 9003, 99), // not in map
	}

	nodes := builder.BuildForwardedNodesFromUsecaseNodes(rules, nodeMap)

	if len(nodes) != 2 {
		t.Fatalf("BuildForwardedNodesFromUsecaseNodes() returned %d nodes, want 2", len(nodes))
	}

	// Verify first node
	if nodes[0].Name != "forward1" {
		t.Errorf("nodes[0].Name = %q, want %q", nodes[0].Name, "forward1")
	}
	if nodes[0].ServerAddress != "agent1.example.com" {
		t.Errorf("nodes[0].ServerAddress = %q, want %q", nodes[0].ServerAddress, "agent1.example.com")
	}
	if nodes[0].EncryptionMethod != "aes-256-gcm" {
		t.Errorf("nodes[0].EncryptionMethod = %q, want %q", nodes[0].EncryptionMethod, "aes-256-gcm")
	}
	if nodes[0].Password != "pass10" {
		t.Errorf("nodes[0].Password = %q, want %q", nodes[0].Password, "pass10")
	}

	// Verify second node
	if nodes[1].Name != "forward2" {
		t.Errorf("nodes[1].Name = %q, want %q", nodes[1].Name, "forward2")
	}
	if nodes[1].ServerAddress != "agent2.example.com" {
		t.Errorf("nodes[1].ServerAddress = %q, want %q", nodes[1].ServerAddress, "agent2.example.com")
	}
	if nodes[1].TransportProtocol != "ws" {
		t.Errorf("nodes[1].TransportProtocol = %q, want %q", nodes[1].TransportProtocol, "ws")
	}
	if nodes[1].Host != "ws.example.com" {
		t.Errorf("nodes[1].Host = %q, want %q", nodes[1].Host, "ws.example.com")
	}
}

func TestForwardedNodeBuilder_BuildForwardedNodesFromUsecaseNodes_EmptyInput(t *testing.T) {
	builder := NewForwardedNodeBuilder(nil, NewProtocolConfigs())

	// Empty rules
	nodes := builder.BuildForwardedNodesFromUsecaseNodes(nil, nil)
	if len(nodes) != 0 {
		t.Errorf("BuildForwardedNodesFromUsecaseNodes(nil, nil) returned %d nodes, want 0", len(nodes))
	}

	// Empty nodeMap
	rules := []*forward.ForwardRule{
		mustCreateDirectRuleWithTargetNode(t, 1, "rule1", 9001, 10),
	}
	nodes = builder.BuildForwardedNodesFromUsecaseNodes(rules, map[uint]*usecases.Node{})
	if len(nodes) != 0 {
		t.Errorf("BuildForwardedNodesFromUsecaseNodes with empty nodeMap returned %d nodes, want 0", len(nodes))
	}
}

// Helper functions for creating test forward rules

func mustCreateDirectRule(t *testing.T, agentID uint, name string, listenPort uint16, targetAddr string, targetPort uint16) *forward.ForwardRule {
	t.Helper()
	rule, err := forward.ReconstructForwardRule(
		1,                             // id
		"fr_test123",                  // sid
		agentID,                       // agentID
		nil,                           // userID
		nil,                           // subscriptionID
		vo.ForwardRuleTypeDirect,      // ruleType
		0,                             // exitAgentID
		nil,                           // exitAgents
		vo.DefaultLoadBalanceStrategy, // loadBalanceStrategy
		nil,                           // chainAgentIDs
		nil,                           // chainPortConfig
		nil,                           // tunnelHops
		vo.TunnelTypeWS,               // tunnelType
		name,                          // name
		listenPort,                    // listenPort
		targetAddr,                    // targetAddress
		targetPort,                    // targetPort
		nil,                           // targetNodeID
		"",                            // bindIP
		vo.IPVersionAuto,              // ipVersion
		vo.ForwardProtocolTCP,         // protocol
		vo.ForwardStatusEnabled,       // status
		"",                            // remark
		0,                             // uploadBytes
		0,                             // downloadBytes
		nil,                           // trafficMultiplier
		0,                             // sortOrder
		nil,                           // groupIDs
		"",                            // serverAddress
		"",                            // externalSource
		"",                            // externalRuleID
		time.Now(),                    // createdAt
		time.Now(),                    // updatedAt
	)
	if err != nil {
		t.Fatalf("failed to create direct rule: %v", err)
	}
	return rule
}

func mustCreateDirectRuleWithTargetNode(t *testing.T, agentID uint, name string, listenPort uint16, targetNodeID uint) *forward.ForwardRule {
	t.Helper()
	rule, err := forward.ReconstructForwardRule(
		1,                             // id
		"fr_test456",                  // sid
		agentID,                       // agentID
		nil,                           // userID
		nil,                           // subscriptionID
		vo.ForwardRuleTypeDirect,      // ruleType
		0,                             // exitAgentID
		nil,                           // exitAgents
		vo.DefaultLoadBalanceStrategy, // loadBalanceStrategy
		nil,                           // chainAgentIDs
		nil,                           // chainPortConfig
		nil,                           // tunnelHops
		vo.TunnelTypeWS,               // tunnelType
		name,                          // name
		listenPort,                    // listenPort
		"",                            // targetAddress
		0,                             // targetPort
		&targetNodeID,                 // targetNodeID
		"",                            // bindIP
		vo.IPVersionAuto,              // ipVersion
		vo.ForwardProtocolTCP,         // protocol
		vo.ForwardStatusEnabled,       // status
		"",                            // remark
		0,                             // uploadBytes
		0,                             // downloadBytes
		nil,                           // trafficMultiplier
		0,                             // sortOrder
		nil,                           // groupIDs
		"",                            // serverAddress
		"",                            // externalSource
		"",                            // externalRuleID
		time.Now(),                    // createdAt
		time.Now(),                    // updatedAt
	)
	if err != nil {
		t.Fatalf("failed to create direct rule with target node: %v", err)
	}
	return rule
}

func mustCreateExternalRule(t *testing.T, serverAddr string, listenPort uint16, externalSource string, targetNodeID uint) *forward.ForwardRule {
	t.Helper()
	rule, err := forward.ReconstructForwardRule(
		1,                             // id
		"fr_ext789",                   // sid
		0,                             // agentID (external rules don't have agents)
		nil,                           // userID
		nil,                           // subscriptionID
		vo.ForwardRuleTypeExternal,    // ruleType
		0,                             // exitAgentID
		nil,                           // exitAgents
		vo.DefaultLoadBalanceStrategy, // loadBalanceStrategy
		nil,                           // chainAgentIDs
		nil,                           // chainPortConfig
		nil,                           // tunnelHops
		vo.TunnelTypeWS,               // tunnelType
		"external-rule",               // name
		listenPort,                    // listenPort
		"",                            // targetAddress
		0,                             // targetPort
		&targetNodeID,                 // targetNodeID
		"",                            // bindIP
		vo.IPVersionAuto,              // ipVersion
		vo.ForwardProtocolTCP,         // protocol
		vo.ForwardStatusEnabled,       // status
		"",                            // remark
		0,                             // uploadBytes
		0,                             // downloadBytes
		nil,                           // trafficMultiplier
		0,                             // sortOrder
		nil,                           // groupIDs
		serverAddr,                    // serverAddress
		externalSource,                // externalSource
		"",                            // externalRuleID
		time.Now(),                    // createdAt
		time.Now(),                    // updatedAt
	)
	if err != nil {
		t.Fatalf("failed to create external rule: %v", err)
	}
	return rule
}

// Helper to create shadowsocks protocol configs for testing
func newShadowsocksConfigs(nodeID uint, encryptionMethod string) ProtocolConfigs {
	c := NewProtocolConfigs()
	c.Shadowsocks[nodeID] = &models.ShadowsocksConfigModel{
		NodeID:           nodeID,
		EncryptionMethod: encryptionMethod,
	}
	return c
}

// Helper to create trojan protocol configs for testing
func newTrojanConfigs(nodeID uint, transportProtocol, host, path string) ProtocolConfigs {
	c := NewProtocolConfigs()
	c.Trojan[nodeID] = &models.TrojanConfigModel{
		NodeID:            nodeID,
		TransportProtocol: transportProtocol,
		Host:              host,
		Path:              path,
	}
	return c
}
