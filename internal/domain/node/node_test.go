package node

import (
	"testing"
	"time"

	vo "orris/internal/domain/node/value_objects"

	"github.com/stretchr/testify/assert"
)

func TestNewNode(t *testing.T) {
	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encConfig, _ := vo.NewEncryptionConfig(vo.MethodAES256GCM, "password123")
	pluginConfig, _ := vo.NewObfsPlugin("http")
	metadata := vo.NewNodeMetadata("US", "California", []string{"fast", "stable"}, "Test node")

	t.Run("should create node successfully", func(t *testing.T) {
		node, err := NewNode(
			"test-node",
			serverAddr,
			8388,
			encConfig,
			pluginConfig,
			metadata,
			100,
			10*1024*1024*1024,
			1,
		)

		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Equal(t, "test-node", node.Name())
		assert.Equal(t, uint16(8388), node.ServerPort())
		assert.Equal(t, vo.NodeStatusInactive, node.Status())
		assert.Equal(t, uint(100), node.MaxUsers())
		assert.Equal(t, uint64(10*1024*1024*1024), node.TrafficLimit())
		assert.Equal(t, uint64(0), node.TrafficUsed())
		assert.NotEmpty(t, node.GetAPIToken())
		assert.NotEmpty(t, node.TokenHash())
		assert.Len(t, node.GetEvents(), 1)
	})

	t.Run("should fail when name is empty", func(t *testing.T) {
		node, err := NewNode(
			"",
			serverAddr,
			8388,
			encConfig,
			pluginConfig,
			metadata,
			100,
			10*1024*1024*1024,
			1,
		)

		assert.Error(t, err)
		assert.Nil(t, node)
		assert.Contains(t, err.Error(), "node name is required")
	})

	t.Run("should fail when port is zero", func(t *testing.T) {
		node, err := NewNode(
			"test-node",
			serverAddr,
			0,
			encConfig,
			pluginConfig,
			metadata,
			100,
			10*1024*1024*1024,
			1,
		)

		assert.Error(t, err)
		assert.Nil(t, node)
		assert.Contains(t, err.Error(), "server port is required")
	})

	t.Run("should record NodeCreatedEvent", func(t *testing.T) {
		node, err := NewNode(
			"test-node",
			serverAddr,
			8388,
			encConfig,
			pluginConfig,
			metadata,
			100,
			10*1024*1024*1024,
			1,
		)

		assert.NoError(t, err)
		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeCreatedEvent)
		assert.True(t, ok)
		assert.Equal(t, "test-node", event.Name)
		assert.Equal(t, uint16(8388), event.ServerPort)
		assert.Equal(t, "inactive", event.Status)
	})
}

func TestReconstructNode(t *testing.T) {
	serverAddr, _ := vo.NewServerAddress("192.168.1.1")
	encConfig, _ := vo.NewEncryptionConfig(vo.MethodAES256GCM, "password123")
	pluginConfig, _ := vo.NewObfsPlugin("http")
	metadata := vo.NewNodeMetadata("US", "California", []string{"fast"}, "Test")
	tokenHash := "abcdef1234567890"
	now := time.Now()

	t.Run("should reconstruct node successfully", func(t *testing.T) {
		node, err := ReconstructNode(
			1,
			"test-node",
			serverAddr,
			8388,
			encConfig,
			pluginConfig,
			vo.NodeStatusActive,
			metadata,
			tokenHash,
			100,
			10*1024*1024*1024,
			1024*1024,
			now,
			1,
			nil,
			1,
			now,
			now,
		)

		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Equal(t, uint(1), node.ID())
		assert.Equal(t, "test-node", node.Name())
		assert.Equal(t, vo.NodeStatusActive, node.Status())
		assert.Equal(t, tokenHash, node.TokenHash())
	})

	t.Run("should fail when ID is zero", func(t *testing.T) {
		node, err := ReconstructNode(
			0,
			"test-node",
			serverAddr,
			8388,
			encConfig,
			pluginConfig,
			vo.NodeStatusActive,
			metadata,
			tokenHash,
			100,
			10*1024*1024*1024,
			0,
			now,
			1,
			nil,
			1,
			now,
			now,
		)

		assert.Error(t, err)
		assert.Nil(t, node)
		assert.Contains(t, err.Error(), "node ID cannot be zero")
	})

	t.Run("should fail when token hash is empty", func(t *testing.T) {
		node, err := ReconstructNode(
			1,
			"test-node",
			serverAddr,
			8388,
			encConfig,
			pluginConfig,
			vo.NodeStatusActive,
			metadata,
			"",
			100,
			10*1024*1024*1024,
			0,
			now,
			1,
			nil,
			1,
			now,
			now,
		)

		assert.Error(t, err)
		assert.Nil(t, node)
		assert.Contains(t, err.Error(), "token hash is required")
	})
}

func TestNodeActivate(t *testing.T) {
	node := createTestNode(t)

	t.Run("should activate node from inactive status", func(t *testing.T) {
		err := node.Activate()

		assert.NoError(t, err)
		assert.Equal(t, vo.NodeStatusActive, node.Status())
		assert.Greater(t, node.Version(), 1)

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeStatusChangedEvent)
		assert.True(t, ok)
		assert.Equal(t, "inactive", event.OldStatus)
		assert.Equal(t, "active", event.NewStatus)
	})

	t.Run("should be idempotent when already active", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.Activate()
		node.ClearEvents()
		currentVersion := node.Version()

		err := node.Activate()

		assert.NoError(t, err)
		assert.Equal(t, vo.NodeStatusActive, node.Status())
		assert.Equal(t, currentVersion, node.Version())
		assert.Len(t, node.GetEvents(), 0)
	})

	t.Run("should activate from maintenance status", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.Activate()
		_ = node.EnterMaintenance("routine maintenance")
		node.ClearEvents()

		err := node.Activate()

		assert.NoError(t, err)
		assert.Equal(t, vo.NodeStatusActive, node.Status())
	})
}

func TestNodeDeactivate(t *testing.T) {
	node := createTestNode(t)
	_ = node.Activate()

	t.Run("should deactivate node from active status", func(t *testing.T) {
		node.ClearEvents()
		err := node.Deactivate()

		assert.NoError(t, err)
		assert.Equal(t, vo.NodeStatusInactive, node.Status())

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeStatusChangedEvent)
		assert.True(t, ok)
		assert.Equal(t, "active", event.OldStatus)
		assert.Equal(t, "inactive", event.NewStatus)
	})

	t.Run("should be idempotent when already inactive", func(t *testing.T) {
		node := createTestNode(t)
		currentVersion := node.Version()
		node.ClearEvents()

		err := node.Deactivate()

		assert.NoError(t, err)
		assert.Equal(t, vo.NodeStatusInactive, node.Status())
		assert.Equal(t, currentVersion, node.Version())
		assert.Len(t, node.GetEvents(), 0)
	})
}

func TestNodeEnterMaintenance(t *testing.T) {
	node := createTestNode(t)
	_ = node.Activate()

	t.Run("should enter maintenance mode with reason", func(t *testing.T) {
		node.ClearEvents()
		err := node.EnterMaintenance("scheduled maintenance")

		assert.NoError(t, err)
		assert.Equal(t, vo.NodeStatusMaintenance, node.Status())
		assert.NotNil(t, node.MaintenanceReason())
		assert.Equal(t, "scheduled maintenance", *node.MaintenanceReason())

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeStatusChangedEvent)
		assert.True(t, ok)
		assert.Equal(t, "active", event.OldStatus)
		assert.Equal(t, "maintenance", event.NewStatus)
		assert.Equal(t, "scheduled maintenance", event.Reason)
	})

	t.Run("should fail when reason is empty", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.Activate()

		err := node.EnterMaintenance("")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maintenance reason is required")
	})

	t.Run("should be idempotent when already in maintenance", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.Activate()
		_ = node.EnterMaintenance("reason1")
		node.ClearEvents()
		currentVersion := node.Version()

		err := node.EnterMaintenance("reason2")

		assert.NoError(t, err)
		assert.Equal(t, vo.NodeStatusMaintenance, node.Status())
		assert.Equal(t, currentVersion, node.Version())
		assert.Len(t, node.GetEvents(), 0)
	})

	t.Run("should fail from inactive status", func(t *testing.T) {
		node := createTestNode(t)

		err := node.EnterMaintenance("test reason")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot enter maintenance mode")
	})
}

func TestNodeExitMaintenance(t *testing.T) {
	node := createTestNode(t)
	_ = node.Activate()
	_ = node.EnterMaintenance("test maintenance")

	t.Run("should exit maintenance mode successfully", func(t *testing.T) {
		node.ClearEvents()
		err := node.ExitMaintenance()

		assert.NoError(t, err)
		assert.Equal(t, vo.NodeStatusActive, node.Status())
		assert.Nil(t, node.MaintenanceReason())

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeStatusChangedEvent)
		assert.True(t, ok)
		assert.Equal(t, "maintenance", event.OldStatus)
		assert.Equal(t, "active", event.NewStatus)
	})

	t.Run("should fail when not in maintenance mode", func(t *testing.T) {
		node := createTestNode(t)

		err := node.ExitMaintenance()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node is not in maintenance mode")
	})
}

func TestNodeUpdateServerAddress(t *testing.T) {
	node := createTestNode(t)
	node.ClearEvents()

	t.Run("should update server address successfully", func(t *testing.T) {
		newAddr, _ := vo.NewServerAddress("10.0.0.1")
		err := node.UpdateServerAddress(newAddr)

		assert.NoError(t, err)
		assert.Equal(t, "10.0.0.1", node.ServerAddress().Value())

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeUpdatedEvent)
		assert.True(t, ok)
		assert.Contains(t, event.UpdatedFields, "server_address")
	})

	t.Run("should be idempotent when address is same", func(t *testing.T) {
		node := createTestNode(t)
		node.ClearEvents()
		currentVersion := node.Version()

		err := node.UpdateServerAddress(node.ServerAddress())

		assert.NoError(t, err)
		assert.Equal(t, currentVersion, node.Version())
		assert.Len(t, node.GetEvents(), 0)
	})
}

func TestNodeUpdateServerPort(t *testing.T) {
	node := createTestNode(t)
	node.ClearEvents()

	t.Run("should update server port successfully", func(t *testing.T) {
		err := node.UpdateServerPort(9999)

		assert.NoError(t, err)
		assert.Equal(t, uint16(9999), node.ServerPort())

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeUpdatedEvent)
		assert.True(t, ok)
		assert.Contains(t, event.UpdatedFields, "server_port")
	})

	t.Run("should fail when port is zero", func(t *testing.T) {
		node := createTestNode(t)

		err := node.UpdateServerPort(0)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server port cannot be zero")
	})

	t.Run("should be idempotent when port is same", func(t *testing.T) {
		node := createTestNode(t)
		node.ClearEvents()
		currentPort := node.ServerPort()
		currentVersion := node.Version()

		err := node.UpdateServerPort(currentPort)

		assert.NoError(t, err)
		assert.Equal(t, currentVersion, node.Version())
		assert.Len(t, node.GetEvents(), 0)
	})
}

func TestNodeUpdateEncryption(t *testing.T) {
	node := createTestNode(t)
	node.ClearEvents()

	t.Run("should update encryption config successfully", func(t *testing.T) {
		newConfig, _ := vo.NewEncryptionConfig(vo.MethodChacha20IETFPoly1305, "newpassword123")
		err := node.UpdateEncryption(newConfig)

		assert.NoError(t, err)
		assert.Equal(t, vo.MethodChacha20IETFPoly1305, node.EncryptionConfig().Method())
		assert.Equal(t, "newpassword123", node.EncryptionConfig().Password())

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeUpdatedEvent)
		assert.True(t, ok)
		assert.Contains(t, event.UpdatedFields, "encryption_config")
	})
}

func TestNodeUpdatePlugin(t *testing.T) {
	node := createTestNode(t)
	node.ClearEvents()

	t.Run("should update plugin config successfully", func(t *testing.T) {
		newPlugin, _ := vo.NewV2RayPlugin("websocket", "example.com")
		err := node.UpdatePlugin(newPlugin)

		assert.NoError(t, err)
		assert.Equal(t, "v2ray-plugin", node.PluginConfig().Plugin())

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeUpdatedEvent)
		assert.True(t, ok)
		assert.Contains(t, event.UpdatedFields, "plugin_config")
	})

	t.Run("should accept nil plugin config", func(t *testing.T) {
		node := createTestNode(t)
		node.ClearEvents()

		err := node.UpdatePlugin(nil)

		assert.NoError(t, err)
		assert.Nil(t, node.PluginConfig())
	})
}

func TestNodeUpdateMetadata(t *testing.T) {
	node := createTestNode(t)
	node.ClearEvents()

	t.Run("should update metadata successfully", func(t *testing.T) {
		newMetadata := vo.NewNodeMetadata("CN", "Shanghai", []string{"premium"}, "Updated node")
		err := node.UpdateMetadata(newMetadata)

		assert.NoError(t, err)
		assert.Equal(t, "CN", node.Metadata().Country())
		assert.Equal(t, "Shanghai", node.Metadata().Region())

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeUpdatedEvent)
		assert.True(t, ok)
		assert.Contains(t, event.UpdatedFields, "metadata")
	})
}

func TestNodeUpdateMaxUsers(t *testing.T) {
	node := createTestNode(t)

	t.Run("should update max users successfully", func(t *testing.T) {
		err := node.UpdateMaxUsers(200)

		assert.NoError(t, err)
		assert.Equal(t, uint(200), node.MaxUsers())
	})

	t.Run("should allow zero max users", func(t *testing.T) {
		node := createTestNode(t)

		err := node.UpdateMaxUsers(0)

		assert.NoError(t, err)
		assert.Equal(t, uint(0), node.MaxUsers())
	})
}

func TestNodeUpdateTrafficLimit(t *testing.T) {
	node := createTestNode(t)

	t.Run("should update traffic limit successfully", func(t *testing.T) {
		err := node.UpdateTrafficLimit(20 * 1024 * 1024 * 1024)

		assert.NoError(t, err)
		assert.Equal(t, uint64(20*1024*1024*1024), node.TrafficLimit())
	})

	t.Run("should allow zero traffic limit (unlimited)", func(t *testing.T) {
		node := createTestNode(t)

		err := node.UpdateTrafficLimit(0)

		assert.NoError(t, err)
		assert.Equal(t, uint64(0), node.TrafficLimit())
	})
}

func TestNodeUpdateSortOrder(t *testing.T) {
	node := createTestNode(t)

	t.Run("should update sort order successfully", func(t *testing.T) {
		err := node.UpdateSortOrder(10)

		assert.NoError(t, err)
		assert.Equal(t, 10, node.SortOrder())
	})

	t.Run("should allow negative sort order", func(t *testing.T) {
		node := createTestNode(t)

		err := node.UpdateSortOrder(-5)

		assert.NoError(t, err)
		assert.Equal(t, -5, node.SortOrder())
	})
}

func TestNodeUpdateName(t *testing.T) {
	node := createTestNode(t)
	node.ClearEvents()

	t.Run("should update name successfully", func(t *testing.T) {
		err := node.UpdateName("new-node-name")

		assert.NoError(t, err)
		assert.Equal(t, "new-node-name", node.Name())

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeUpdatedEvent)
		assert.True(t, ok)
		assert.Contains(t, event.UpdatedFields, "name")
	})

	t.Run("should fail when name is empty", func(t *testing.T) {
		node := createTestNode(t)

		err := node.UpdateName("")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node name cannot be empty")
	})

	t.Run("should be idempotent when name is same", func(t *testing.T) {
		node := createTestNode(t)
		node.ClearEvents()
		currentVersion := node.Version()

		err := node.UpdateName(node.Name())

		assert.NoError(t, err)
		assert.Equal(t, currentVersion, node.Version())
		assert.Len(t, node.GetEvents(), 0)
	})
}

func TestNodeGenerateAPIToken(t *testing.T) {
	node := createTestNode(t)
	originalToken := node.GetAPIToken()
	originalHash := node.TokenHash()
	node.ClearAPIToken()
	node.ClearEvents()

	t.Run("should generate new API token", func(t *testing.T) {
		newToken, err := node.GenerateAPIToken()

		assert.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, originalToken, newToken)
		assert.NotEqual(t, originalHash, node.TokenHash())
		assert.Contains(t, newToken, "node_")

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeUpdatedEvent)
		assert.True(t, ok)
		assert.Contains(t, event.UpdatedFields, "api_token")
	})

	t.Run("should generate unique tokens", func(t *testing.T) {
		node1 := createTestNode(t)
		node2 := createTestNode(t)

		token1 := node1.GetAPIToken()
		token2 := node2.GetAPIToken()

		assert.NotEqual(t, token1, token2)
		assert.NotEqual(t, node1.TokenHash(), node2.TokenHash())
	})
}

func TestNodeVerifyAPIToken(t *testing.T) {
	node := createTestNode(t)
	validToken := node.GetAPIToken()

	t.Run("should verify valid token", func(t *testing.T) {
		isValid := node.VerifyAPIToken(validToken)

		assert.True(t, isValid)
	})

	t.Run("should reject invalid token", func(t *testing.T) {
		isValid := node.VerifyAPIToken("invalid_token")

		assert.False(t, isValid)
	})

	t.Run("should reject empty token", func(t *testing.T) {
		isValid := node.VerifyAPIToken("")

		assert.False(t, isValid)
	})

	t.Run("should use constant time comparison", func(t *testing.T) {
		node := createTestNode(t)
		validToken := node.GetAPIToken()

		similarToken := validToken[:len(validToken)-1] + "X"
		isValid := node.VerifyAPIToken(similarToken)

		assert.False(t, isValid)
	})
}

func TestNodeRecordTraffic(t *testing.T) {
	node := createTestNode(t)
	node.ClearEvents()

	t.Run("should record traffic successfully", func(t *testing.T) {
		err := node.RecordTraffic(1024*1024, 2048*1024)

		assert.NoError(t, err)
		assert.Equal(t, uint64(1024*1024+2048*1024), node.TrafficUsed())
	})

	t.Run("should accumulate traffic", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.RecordTraffic(1024, 2048)
		_ = node.RecordTraffic(512, 1024)

		assert.Equal(t, uint64(1024+2048+512+1024), node.TrafficUsed())
	})

	t.Run("should do nothing when both upload and download are zero", func(t *testing.T) {
		node := createTestNode(t)
		initialUsed := node.TrafficUsed()

		err := node.RecordTraffic(0, 0)

		assert.NoError(t, err)
		assert.Equal(t, initialUsed, node.TrafficUsed())
	})

	t.Run("should record event when traffic exceeds limit", func(t *testing.T) {
		node := createTestNode(t)
		node.ClearEvents()

		err := node.RecordTraffic(11*1024*1024*1024, 0)

		assert.NoError(t, err)
		assert.True(t, node.IsTrafficExceeded())

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeTrafficExceededEvent)
		assert.True(t, ok)
		assert.Equal(t, node.ID(), event.NodeID)
		assert.Equal(t, node.TrafficLimit(), event.TrafficLimit)
		assert.Equal(t, node.TrafficUsed(), event.TrafficUsed)
	})
}

func TestNodeIsTrafficExceeded(t *testing.T) {
	t.Run("should return false when limit is zero (unlimited)", func(t *testing.T) {
		serverAddr, _ := vo.NewServerAddress("192.168.1.1")
		encConfig, _ := vo.NewEncryptionConfig(vo.MethodAES256GCM, "password123")
		metadata := vo.NewNodeMetadata("US", "CA", []string{}, "")

		node, _ := NewNode("test", serverAddr, 8388, encConfig, nil, metadata, 100, 0, 1)
		_ = node.RecordTraffic(9999999, 9999999)

		assert.False(t, node.IsTrafficExceeded())
	})

	t.Run("should return false when usage below limit", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.RecordTraffic(1024, 2048)

		assert.False(t, node.IsTrafficExceeded())
	})

	t.Run("should return true when usage equals limit", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.RecordTraffic(10*1024*1024*1024, 0)

		assert.True(t, node.IsTrafficExceeded())
	})

	t.Run("should return true when usage exceeds limit", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.RecordTraffic(11*1024*1024*1024, 0)

		assert.True(t, node.IsTrafficExceeded())
	})
}

func TestNodeResetTraffic(t *testing.T) {
	node := createTestNode(t)
	_ = node.RecordTraffic(5*1024*1024*1024, 3*1024*1024*1024)
	node.ClearEvents()

	t.Run("should reset traffic usage", func(t *testing.T) {
		oldResetAt := node.TrafficResetAt()
		time.Sleep(10 * time.Millisecond)

		err := node.ResetTraffic()

		assert.NoError(t, err)
		assert.Equal(t, uint64(0), node.TrafficUsed())
		assert.True(t, node.TrafficResetAt().After(oldResetAt))

		events := node.GetEvents()
		assert.Len(t, events, 1)

		event, ok := events[0].(NodeUpdatedEvent)
		assert.True(t, ok)
		assert.Contains(t, event.UpdatedFields, "traffic_used")
	})

	t.Run("should allow traffic recording after reset", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.RecordTraffic(5*1024*1024*1024, 0)
		_ = node.ResetTraffic()

		err := node.RecordTraffic(1024, 2048)

		assert.NoError(t, err)
		assert.Equal(t, uint64(1024+2048), node.TrafficUsed())
		assert.False(t, node.IsTrafficExceeded())
	})
}

func TestNodeIsAvailable(t *testing.T) {
	t.Run("should return false when status is inactive", func(t *testing.T) {
		node := createTestNode(t)

		assert.False(t, node.IsAvailable())
	})

	t.Run("should return false when status is maintenance", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.Activate()
		_ = node.EnterMaintenance("test")

		assert.False(t, node.IsAvailable())
	})

	t.Run("should return false when traffic exceeded", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.Activate()
		_ = node.RecordTraffic(11*1024*1024*1024, 0)

		assert.False(t, node.IsAvailable())
	})

	t.Run("should return true when active and traffic ok", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.Activate()
		_ = node.RecordTraffic(1024, 2048)

		assert.True(t, node.IsAvailable())
	})

	t.Run("should return true when active with unlimited traffic", func(t *testing.T) {
		serverAddr, _ := vo.NewServerAddress("192.168.1.1")
		encConfig, _ := vo.NewEncryptionConfig(vo.MethodAES256GCM, "password123")
		metadata := vo.NewNodeMetadata("US", "CA", []string{}, "")

		node, _ := NewNode("test", serverAddr, 8388, encConfig, nil, metadata, 100, 0, 1)
		_ = node.Activate()
		_ = node.RecordTraffic(9999999, 9999999)

		assert.True(t, node.IsAvailable())
	})
}

func TestNodeSetID(t *testing.T) {
	node := createTestNode(t)

	t.Run("should set ID successfully when not set", func(t *testing.T) {
		err := node.SetID(123)

		assert.NoError(t, err)
		assert.Equal(t, uint(123), node.ID())
	})

	t.Run("should fail when ID already set", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.SetID(123)

		err := node.SetID(456)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node ID is already set")
		assert.Equal(t, uint(123), node.ID())
	})

	t.Run("should fail when ID is zero", func(t *testing.T) {
		node := createTestNode(t)

		err := node.SetID(0)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node ID cannot be zero")
	})
}

func TestNodeAPITokenManagement(t *testing.T) {
	node := createTestNode(t)
	originalToken := node.GetAPIToken()

	t.Run("should get API token", func(t *testing.T) {
		assert.NotEmpty(t, originalToken)
		assert.Contains(t, originalToken, "node_")
	})

	t.Run("should clear API token", func(t *testing.T) {
		node.ClearAPIToken()

		assert.Empty(t, node.GetAPIToken())
		assert.NotEmpty(t, node.TokenHash())
	})

	t.Run("should still verify token after clearing plain text", func(t *testing.T) {
		node := createTestNode(t)
		token := node.GetAPIToken()
		node.ClearAPIToken()

		assert.True(t, node.VerifyAPIToken(token))
	})
}

func TestNodeEventManagement(t *testing.T) {
	t.Run("should get and clear events", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.Activate()

		events := node.GetEvents()

		assert.Len(t, events, 1)
		assert.Len(t, node.GetEvents(), 0)
	})

	t.Run("should clear events explicitly", func(t *testing.T) {
		node := createTestNode(t)

		node.ClearEvents()

		assert.Len(t, node.GetEvents(), 0)
	})

	t.Run("should accumulate events", func(t *testing.T) {
		node := createTestNode(t)
		node.ClearEvents()

		_ = node.Activate()
		_ = node.UpdateName("new-name")
		_ = node.RecordTraffic(1024, 2048)

		events := node.GetEvents()
		assert.Len(t, events, 2)
	})
}

func TestNodeValidate(t *testing.T) {
	t.Run("should validate successfully for valid node", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.SetID(1)

		err := node.Validate()

		assert.NoError(t, err)
	})

	t.Run("should fail when name is empty", func(t *testing.T) {
		serverAddr, _ := vo.NewServerAddress("192.168.1.1")
		encConfig, _ := vo.NewEncryptionConfig(vo.MethodAES256GCM, "password123")
		metadata := vo.NewNodeMetadata("US", "CA", []string{}, "")

		node, _ := ReconstructNode(
			1, "test", serverAddr, 8388, encConfig, nil,
			vo.NodeStatusActive, metadata, "hash123",
			100, 10*1024*1024*1024, 0, time.Now(),
			1, nil, 1, time.Now(), time.Now(),
		)

		node.name = ""
		err := node.Validate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node name is required")
	})

	t.Run("should fail when server port is zero", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.SetID(1)
		node.serverPort = 0

		err := node.Validate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server port is required")
	})

	t.Run("should fail when token hash is empty", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.SetID(1)
		node.tokenHash = ""

		err := node.Validate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token hash is required")
	})

	t.Run("should fail when maintenance reason missing in maintenance mode", func(t *testing.T) {
		serverAddr, _ := vo.NewServerAddress("192.168.1.1")
		encConfig, _ := vo.NewEncryptionConfig(vo.MethodAES256GCM, "password123")
		metadata := vo.NewNodeMetadata("US", "CA", []string{}, "")

		node, _ := ReconstructNode(
			1, "test", serverAddr, 8388, encConfig, nil,
			vo.NodeStatusMaintenance, metadata, "hash123",
			100, 10*1024*1024*1024, 0, time.Now(),
			1, nil, 1, time.Now(), time.Now(),
		)

		err := node.Validate()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maintenance reason is required")
	})
}

func TestNodeVersionIncrement(t *testing.T) {
	node := createTestNode(t)
	initialVersion := node.Version()

	t.Run("should increment version on activate", func(t *testing.T) {
		_ = node.Activate()

		assert.Equal(t, initialVersion+1, node.Version())
	})

	t.Run("should increment version on deactivate", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.Activate()
		versionBefore := node.Version()

		_ = node.Deactivate()

		assert.Equal(t, versionBefore+1, node.Version())
	})

	t.Run("should increment version on update operations", func(t *testing.T) {
		node := createTestNode(t)
		initialVersion := node.Version()

		_ = node.UpdateName("new-name")
		_ = node.UpdateServerPort(9999)
		_ = node.ResetTraffic()

		assert.Equal(t, initialVersion+3, node.Version())
	})

	t.Run("should not increment version on no-op operations", func(t *testing.T) {
		node := createTestNode(t)
		_ = node.Activate()
		versionBefore := node.Version()

		_ = node.Activate()

		assert.Equal(t, versionBefore, node.Version())
	})
}

func TestNodeTimestamps(t *testing.T) {
	t.Run("should set timestamps on creation", func(t *testing.T) {
		before := time.Now()
		node := createTestNode(t)
		after := time.Now()

		assert.True(t, node.CreatedAt().After(before) || node.CreatedAt().Equal(before))
		assert.True(t, node.CreatedAt().Before(after) || node.CreatedAt().Equal(after))
		assert.Equal(t, node.CreatedAt(), node.UpdatedAt())
	})

	t.Run("should update timestamp on modifications", func(t *testing.T) {
		node := createTestNode(t)
		originalUpdatedAt := node.UpdatedAt()
		time.Sleep(10 * time.Millisecond)

		_ = node.Activate()

		assert.True(t, node.UpdatedAt().After(originalUpdatedAt))
		assert.Equal(t, node.CreatedAt(), node.CreatedAt())
	})
}

func createTestNode(t *testing.T) *Node {
	serverAddr, err := vo.NewServerAddress("192.168.1.1")
	assert.NoError(t, err)

	encConfig, err := vo.NewEncryptionConfig(vo.MethodAES256GCM, "password123")
	assert.NoError(t, err)

	pluginConfig, err := vo.NewObfsPlugin("http")
	assert.NoError(t, err)

	metadata := vo.NewNodeMetadata("US", "California", []string{"fast", "stable"}, "Test node")

	node, err := NewNode(
		"test-node",
		serverAddr,
		8388,
		encConfig,
		pluginConfig,
		metadata,
		100,
		10*1024*1024*1024,
		1,
	)
	assert.NoError(t, err)
	node.ClearEvents()

	return node
}
