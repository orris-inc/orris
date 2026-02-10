package pubsub

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dto "github.com/orris-inc/orris/internal/shared/hubprotocol/forward"
	nodedto "github.com/orris-inc/orris/internal/shared/hubprotocol/node"
)

func TestAgentCommandEvent_MarshalRoundtrip(t *testing.T) {
	event := AgentCommandEvent{
		AgentID: 42,
		Command: &dto.CommandData{
			CommandID: "cmd-123",
			Action:    dto.CmdActionReloadConfig,
		},
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded AgentCommandEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.AgentID, decoded.AgentID)
	assert.Equal(t, event.Command.CommandID, decoded.Command.CommandID)
	assert.Equal(t, event.Command.Action, decoded.Command.Action)
}

func TestNodeCommandEvent_MarshalRoundtrip(t *testing.T) {
	event := NodeCommandEvent{
		NodeID: 7,
		Command: &nodedto.NodeCommandData{
			CommandID: "cmd-456",
			Action:    nodedto.NodeCmdActionReloadConfig,
		},
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded NodeCommandEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.NodeID, decoded.NodeID)
	assert.Equal(t, event.Command.CommandID, decoded.Command.CommandID)
	assert.Equal(t, event.Command.Action, decoded.Command.Action)
}

func TestHubStatusEvent_MarshalRoundtrip(t *testing.T) {
	event := HubStatusEvent{
		Type:       HubEventAgentOnline,
		AgentID:    10,
		AgentSID:   "fa_abc123",
		AgentName:  "test-agent",
		Timestamp:  1700000000,
		InstanceID: "instance-1",
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded HubStatusEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.Type, decoded.Type)
	assert.Equal(t, event.AgentID, decoded.AgentID)
	assert.Equal(t, event.AgentSID, decoded.AgentSID)
	assert.Equal(t, event.AgentName, decoded.AgentName)
	assert.Equal(t, event.Timestamp, decoded.Timestamp)
	assert.Equal(t, event.InstanceID, decoded.InstanceID)
}

func TestHubStatusEvent_NodeEvent(t *testing.T) {
	event := HubStatusEvent{
		Type:      HubEventNodeOffline,
		NodeID:    5,
		NodeSID:   "node_xyz789",
		NodeName:  "test-node",
		Timestamp: 1700000000,
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded HubStatusEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, HubEventNodeOffline, decoded.Type)
	assert.Equal(t, uint(5), decoded.NodeID)
	assert.Equal(t, "node_xyz789", decoded.NodeSID)
}
