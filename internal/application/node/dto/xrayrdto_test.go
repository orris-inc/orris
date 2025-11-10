package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestV2RaySocksResponse_SuccessResponse tests success response format
func TestV2RaySocksResponse_SuccessResponse(t *testing.T) {
	data := map[string]string{
		"message": "operation successful",
	}

	response := NewSuccessResponse(data)

	assert.NotNil(t, response)
	assert.Equal(t, 1, response.Ret)
	assert.Equal(t, "success", response.Msg)
	assert.Equal(t, data, response.Data)

	// Test JSON marshaling
	jsonData, err := json.MarshalIndent(response, "", "  ")
	assert.NoError(t, err)
	t.Logf("Success Response JSON:\n%s", string(jsonData))
}

// TestV2RaySocksResponse_ErrorResponse tests error response format
func TestV2RaySocksResponse_ErrorResponse(t *testing.T) {
	errMsg := "invalid node configuration"

	response := NewErrorResponse(errMsg)

	assert.NotNil(t, response)
	assert.Equal(t, 0, response.Ret)
	assert.Equal(t, errMsg, response.Msg)
	assert.Nil(t, response.Data)

	// Test JSON marshaling
	jsonData, err := json.MarshalIndent(response, "", "  ")
	assert.NoError(t, err)
	t.Logf("Error Response JSON:\n%s", string(jsonData))
}

// TestNodeConfigResponse_JSONSerialization tests NodeConfigResponse JSON format
func TestNodeConfigResponse_JSONSerialization(t *testing.T) {
	config := &NodeConfigResponse{
		NodeID:            1,
		NodeType:          "shadowsocks",
		ServerHost:        "node1.example.com",
		ServerPort:        8388,
		Method:            "aes-256-gcm",
		ServerKey:         "password123",
		TransportProtocol: "tcp",
		Host:              "",
		Path:              "",
		EnableVless:       false,
		EnableXTLS:        false,
		SpeedLimit:        100,
		DeviceLimit:       3,
		RuleListPath:      "",
	}

	response := NewSuccessResponse(config)

	jsonData, err := json.MarshalIndent(response, "", "  ")
	assert.NoError(t, err)
	t.Logf("Node Config Response JSON:\n%s", string(jsonData))

	// Verify JSON structure
	var decoded V2RaySocksResponse
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, 1, decoded.Ret)
}

// TestNodeConfigResponse_WebSocketTransport tests WS transport configuration
func TestNodeConfigResponse_WebSocketTransport(t *testing.T) {
	config := &NodeConfigResponse{
		NodeID:            2,
		NodeType:          "trojan",
		ServerHost:        "node2.example.com",
		ServerPort:        443,
		Method:            "",
		ServerKey:         "",
		TransportProtocol: "ws",
		Host:              "cdn.example.com",
		Path:              "/api/v2ray",
		EnableVless:       false,
		EnableXTLS:        false,
		SpeedLimit:        0,
		DeviceLimit:       0,
		RuleListPath:      "/etc/xrayr/rules.txt",
	}

	response := NewSuccessResponse(config)

	jsonData, err := json.MarshalIndent(response, "", "  ")
	assert.NoError(t, err)
	t.Logf("Node Config Response (WebSocket) JSON:\n%s", string(jsonData))
}

// TestNodeUsersResponse_JSONSerialization tests NodeUsersResponse JSON format
func TestNodeUsersResponse_JSONSerialization(t *testing.T) {
	users := &NodeUsersResponse{
		Users: []NodeUserInfo{
			{
				ID:          1001,
				UUID:        "550e8400-e29b-41d4-a716-446655440000",
				Email:       "user1@example.com",
				SpeedLimit:  10485760,  // 10 Mbps in bps
				DeviceLimit: 2,
				ExpireTime:  1735689600, // 2025-01-01 00:00:00 UTC
			},
			{
				ID:          1002,
				UUID:        "550e8400-e29b-41d4-a716-446655440001",
				Email:       "user2@example.com",
				SpeedLimit:  0,  // unlimited
				DeviceLimit: 0,  // unlimited
				ExpireTime:  1767225600, // 2026-01-01 00:00:00 UTC
			},
		},
	}

	response := NewSuccessResponse(users)

	jsonData, err := json.MarshalIndent(response, "", "  ")
	assert.NoError(t, err)
	t.Logf("Node Users Response JSON:\n%s", string(jsonData))

	// Verify JSON structure
	var decoded V2RaySocksResponse
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, 1, decoded.Ret)
}

// TestReportUserTrafficRequest_JSONSerialization tests traffic report JSON format
func TestReportUserTrafficRequest_JSONSerialization(t *testing.T) {
	trafficReport := &ReportUserTrafficRequest{
		Users: []UserTrafficItem{
			{
				UID:      1001,
				Upload:   1073741824,  // 1 GB
				Download: 5368709120,  // 5 GB
			},
			{
				UID:      1002,
				Upload:   536870912,   // 512 MB
				Download: 2147483648,  // 2 GB
			},
		},
	}

	jsonData, err := json.MarshalIndent(trafficReport, "", "  ")
	assert.NoError(t, err)
	t.Logf("Report User Traffic Request JSON:\n%s", string(jsonData))

	// Verify deserialization
	var decoded ReportUserTrafficRequest
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Len(t, decoded.Users, 2)
	assert.Equal(t, 1001, decoded.Users[0].UID)
}

// TestReportNodeStatusRequest_JSONSerialization tests node status report JSON format
func TestReportNodeStatusRequest_JSONSerialization(t *testing.T) {
	statusReport := &ReportNodeStatusRequest{
		CPU:    "45%",
		Mem:    "68%",
		Net:    "1024 MB",
		Disk:   "72%",
		Uptime: 86400,  // 1 day in seconds
	}

	jsonData, err := json.MarshalIndent(statusReport, "", "  ")
	assert.NoError(t, err)
	t.Logf("Report Node Status Request JSON:\n%s", string(jsonData))

	// Verify deserialization
	var decoded ReportNodeStatusRequest
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, "45%", decoded.CPU)
	assert.Equal(t, 86400, decoded.Uptime)
}

// TestReportOnlineUsersRequest_JSONSerialization tests online users report JSON format
func TestReportOnlineUsersRequest_JSONSerialization(t *testing.T) {
	onlineReport := &ReportOnlineUsersRequest{
		Users: []OnlineUserItem{
			{
				UID: 1001,
				IP:  "192.168.1.100",
			},
			{
				UID: 1002,
				IP:  "192.168.1.101",
			},
			{
				UID: 1003,
				IP:  "10.0.0.50",
			},
		},
	}

	jsonData, err := json.MarshalIndent(onlineReport, "", "  ")
	assert.NoError(t, err)
	t.Logf("Report Online Users Request JSON:\n%s", string(jsonData))

	// Verify deserialization
	var decoded ReportOnlineUsersRequest
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Len(t, decoded.Users, 3)
	assert.Equal(t, "192.168.1.100", decoded.Users[0].IP)
}

// TestIsSSMethod tests Shadowsocks method detection
func TestIsSSMethod(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{"aes-256-gcm", true},
		{"aes-128-gcm", true},
		{"chacha20-ietf-poly1305", true},
		{"xchacha20-ietf-poly1305", true},
		{"trojan", false},
		{"vmess", false},
		{"", false},
		{"unknown-method", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := isSSMethod(tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEmptyUsersResponse tests empty users response handling
func TestEmptyUsersResponse(t *testing.T) {
	response := ToNodeUsersResponse(nil)

	assert.NotNil(t, response)
	assert.Empty(t, response.Users)

	jsonData, err := json.MarshalIndent(response, "", "  ")
	assert.NoError(t, err)
	t.Logf("Empty Users Response JSON:\n%s", string(jsonData))
}
