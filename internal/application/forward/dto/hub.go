// Package dto provides data transfer objects for the forward domain.
package dto

import hubproto "github.com/orris-inc/orris/internal/shared/hubprotocol/forward"

// Hub message type constants (re-exported from shared hubprotocol).
const (
	// Agent -> Server message types.
	MsgTypeStatus    = hubproto.MsgTypeStatus
	MsgTypeHeartbeat = hubproto.MsgTypeHeartbeat
	MsgTypeEvent     = hubproto.MsgTypeEvent

	// Server -> Agent message types.
	MsgTypeCommand = hubproto.MsgTypeCommand

	// Probe message types (Forward domain specific, routed through AgentHub).
	MsgTypeProbeTask   = hubproto.MsgTypeProbeTask   // Server -> Agent
	MsgTypeProbeResult = hubproto.MsgTypeProbeResult // Agent -> Server

	// Config sync message types (Forward domain specific, routed through AgentHub).
	MsgTypeConfigSync = hubproto.MsgTypeConfigSync // Server -> Agent
	MsgTypeConfigAck  = hubproto.MsgTypeConfigAck  // Agent -> Server

	// Rule sync status message types (Forward domain specific, routed through AgentHub).
	MsgTypeRuleSyncStatus = hubproto.MsgTypeRuleSyncStatus // Agent -> Server

	// Tunnel health report message types (Agent reports exit agent health status).
	MsgTypeTunnelHealthReport = hubproto.MsgTypeTunnelHealthReport // Agent -> Server
)

// HubMessage is the unified WebSocket message envelope (type alias from shared hubprotocol).
type HubMessage = hubproto.HubMessage

// CommandData represents a command to be sent to agent (type alias from shared hubprotocol).
type CommandData = hubproto.CommandData

// Command action constants (re-exported from shared hubprotocol).
const (
	CmdActionReloadConfig   = hubproto.CmdActionReloadConfig
	CmdActionRestartRule    = hubproto.CmdActionRestartRule
	CmdActionStopRule       = hubproto.CmdActionStopRule
	CmdActionProbe          = hubproto.CmdActionProbe
	CmdActionUpdate         = hubproto.CmdActionUpdate
	CmdActionAPIURLChanged  = hubproto.CmdActionAPIURLChanged
	CmdActionConfigRelocate = hubproto.CmdActionConfigRelocate
)

// APIURLChangedPayload contains the new API URL for agent reconnection (type alias from shared hubprotocol).
type APIURLChangedPayload = hubproto.APIURLChangedPayload

// AgentEventData represents an agent event payload (type alias from shared hubprotocol).
type AgentEventData = hubproto.AgentEventData

// Agent event type constants (re-exported from shared hubprotocol).
const (
	EventTypeConnected    = hubproto.EventTypeConnected
	EventTypeDisconnected = hubproto.EventTypeDisconnected
	EventTypeError        = hubproto.EventTypeError
	EventTypeConfigChange = hubproto.EventTypeConfigChange
)

// ConfigSyncData represents incremental configuration sync data (type alias from shared hubprotocol).
type ConfigSyncData = hubproto.ConfigSyncData

// ExitAgentSyncData represents an exit agent with connection info for load balancing (type alias from shared hubprotocol).
type ExitAgentSyncData = hubproto.ExitAgentSyncData

// HealthCheckConfig represents health check configuration for load balancing failover (type alias from shared hubprotocol).
type HealthCheckConfig = hubproto.HealthCheckConfig

// RuleSyncData represents rule sync data for config sync (type alias from shared hubprotocol).
type RuleSyncData = hubproto.RuleSyncData

// ConfigAckData represents agent acknowledgment of config sync (type alias from shared hubprotocol).
type ConfigAckData = hubproto.ConfigAckData

// TunnelHealthReport represents a tunnel health status report from entry agent (type alias from shared hubprotocol).
type TunnelHealthReport = hubproto.TunnelHealthReport
