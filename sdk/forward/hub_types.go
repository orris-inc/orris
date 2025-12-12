package forward

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	hubWriteWait  = 10 * time.Second
	hubPongWait   = 60 * time.Second
	hubPingPeriod = 30 * time.Second
)

// ReconnectConfig holds reconnection strategy parameters.
type ReconnectConfig struct {
	// InitialInterval is the first retry delay (default: 1s)
	InitialInterval time.Duration

	// MaxInterval is the maximum retry delay (default: 60s)
	MaxInterval time.Duration

	// MaxElapsedTime is the total time to keep retrying. 0 means never stop (default: 0)
	MaxElapsedTime time.Duration

	// Multiplier is the exponential backoff multiplier (default: 2.0)
	Multiplier float64

	// RandomizationFactor adds jitter to prevent thundering herd (default: 0.1)
	RandomizationFactor float64

	// OnConnected is called when successfully connected
	OnConnected func()

	// OnDisconnected is called when disconnected (with error)
	OnDisconnected func(err error)

	// OnReconnecting is called before each reconnection attempt
	OnReconnecting func(attempt uint64, delay time.Duration)

	// OnConfigSync is called when a config sync message is received from the server.
	// This is used to handle incremental or full configuration updates.
	OnConfigSync ConfigSyncHandler
}

// DefaultReconnectConfig returns the default reconnection configuration.
func DefaultReconnectConfig() *ReconnectConfig {
	return &ReconnectConfig{
		InitialInterval:     1 * time.Second,
		MaxInterval:         60 * time.Second,
		MaxElapsedTime:      0, // Never give up
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
	}
}

// HubConn represents a WebSocket connection to the AgentHub.
type HubConn struct {
	conn   *websocket.Conn
	send   chan *HubMessage
	done   chan struct{}  // Signal channel for graceful shutdown
	Events chan *HubEvent // Event channel for agent to receive events
	mu     sync.Mutex     // Protects closed and send channel access
	closed bool

	// Message handler callback
	onMessage func(msg *HubMessage)
}

// HubMessage is the unified WebSocket message envelope.
type HubMessage struct {
	Type      string `json:"type"`
	AgentID   string `json:"agent_id,omitempty"` // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	Timestamp int64  `json:"timestamp"`
	Data      any    `json:"data,omitempty"`
}

// Hub message type constants.
const (
	// Agent -> Server message types.
	MsgTypeStatus    = "status"
	MsgTypeHeartbeat = "heartbeat"
	MsgTypeEvent     = "event"

	// Server -> Agent message types.
	MsgTypeCommand = "command"

	// Probe message types.
	MsgTypeProbeTask   = "probe_task"   // Server -> Agent
	MsgTypeProbeResult = "probe_result" // Agent -> Server

	// Config sync message types.
	MsgTypeConfigSync = "config_sync" // Server -> Agent
	MsgTypeConfigAck  = "config_ack"  // Agent -> Server
)

// HubEventType represents the type of event received from hub.
type HubEventType string

const (
	HubEventConfigSync HubEventType = "config_sync"
	HubEventProbeTask  HubEventType = "probe_task"
)

// HubEvent is a unified event structure for agent to consume.
type HubEvent struct {
	Type       HubEventType
	ConfigSync *ConfigSyncData
	ProbeTask  *ProbeTask
}

// ConfigSyncData represents configuration synchronization data.
type ConfigSyncData struct {
	Version            uint64         `json:"version"`
	FullSync           bool           `json:"full_sync"`
	Added              []RuleSyncData `json:"added,omitempty"`
	Updated            []RuleSyncData `json:"updated,omitempty"`
	Removed            []string       `json:"removed,omitempty"`              // Rule IDs to remove (Stripe-style prefixed, e.g., "fr_xxx")
	ClientToken        string         `json:"client_token,omitempty"`         // Agent's token for tunnel handshake (full sync only)
	TokenSigningSecret string         `json:"token_signing_secret,omitempty"` // Secret for local agent token verification (full sync only)
}

// RuleSyncData represents rule synchronization data (aligned with server DTO).
type RuleSyncData struct {
	ID                     string   `json:"id"`       // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	ShortID                string   `json:"short_id"` // Deprecated: use ID instead
	RuleType               string   `json:"rule_type"`
	ListenPort             uint16   `json:"listen_port"`
	TargetAddress          string   `json:"target_address,omitempty"`
	TargetPort             uint16   `json:"target_port,omitempty"`
	BindIP                 string   `json:"bind_ip,omitempty"` // Bind IP address for outbound connections
	Protocol               string   `json:"protocol"`
	Role                   string   `json:"role,omitempty"`
	AgentID                string   `json:"agent_id,omitempty"` // Entry agent ID (for exit agents to verify handshake)
	NextHopAgentID         string   `json:"next_hop_agent_id,omitempty"`
	NextHopAddress         string   `json:"next_hop_address,omitempty"`
	NextHopWsPort          uint16   `json:"next_hop_ws_port,omitempty"`
	NextHopPort            uint16   `json:"next_hop_port,omitempty"`             // Next agent's listen port for direct_chain
	NextHopConnectionToken string   `json:"next_hop_connection_token,omitempty"` // Short-term token for next hop authentication
	ChainAgentIDs          []string `json:"chain_agent_ids,omitempty"`
	ChainPosition          int      `json:"chain_position,omitempty"`
	IsLastInChain          bool     `json:"is_last_in_chain,omitempty"`
}

// ConfigAckData represents configuration acknowledgment data.
type ConfigAckData struct {
	Version uint64 `json:"version"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// HubMessageHandler is a callback function for handling hub messages.
type HubMessageHandler func(msg *HubMessage)

// ProbeTaskHandler is a callback function for handling probe tasks.
// Returns the probe result to be sent back to the server.
type ProbeTaskHandler func(task *ProbeTask) *ProbeTaskResult

// ConfigSyncHandler is a callback function for handling config sync events.
// The handler receives the config sync data and should return an error if processing failed.
type ConfigSyncHandler func(data *ConfigSyncData) error
