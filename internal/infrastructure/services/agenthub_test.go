package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dto "github.com/orris-inc/orris/internal/shared/hubprotocol/forward"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// nopLogger is a no-op logger for testing.
type nopLogger struct{}

func newNopLogger() logger.Interface { return &nopLogger{} }

func (l *nopLogger) Debug(msg string, args ...any)                   {}
func (l *nopLogger) Info(msg string, args ...any)                    {}
func (l *nopLogger) Warn(msg string, args ...any)                    {}
func (l *nopLogger) Error(msg string, args ...any)                   {}
func (l *nopLogger) Fatal(msg string, args ...any)                   {}
func (l *nopLogger) With(args ...any) logger.Interface               { return l }
func (l *nopLogger) Named(name string) logger.Interface              { return l }
func (l *nopLogger) Debugw(msg string, keysAndValues ...interface{}) {}
func (l *nopLogger) Infow(msg string, keysAndValues ...interface{})  {}
func (l *nopLogger) Warnw(msg string, keysAndValues ...interface{})  {}
func (l *nopLogger) Errorw(msg string, keysAndValues ...interface{}) {}
func (l *nopLogger) Fatalw(msg string, keysAndValues ...interface{}) {}

// newTestHub builds an AgentHub without the background timeout checker, so that
// connections seeded by tests are not reaped while the test runs.
func newTestHub() *AgentHub {
	return &AgentHub{
		agents:            make(map[uint]*AgentHubConn),
		nodes:             make(map[uint]*NodeHubConn),
		messageHandlers:   make([]MessageHandler, 0),
		nodeStatusTimeout: 5 * time.Second,
		done:              make(chan struct{}),
		logger:            newNopLogger(),
	}
}

func TestCanReplaceConn(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name            string
		existingAddress string
		newAddress      string
		lastSeen        time.Time
		want            bool
	}{
		{"same address reconnect", "1.1.1.1", "1.1.1.1", now, true},
		{"unknown existing address", "", "1.1.1.1", now, true},
		{"unknown new address", "1.1.1.1", "", now, true},
		{"different address while healthy", "1.1.1.1", "2.2.2.2", now, false},
		{"different address after grace", "1.1.1.1", "2.2.2.2", now.Add(-2 * duplicateIdentityGrace), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, canReplaceConn(tt.existingAddress, tt.newAddress, tt.lastSeen))
		})
	}
}

func TestRegisterNodeAgent_RejectsDuplicateIdentity(t *testing.T) {
	h := newTestHub()
	existing := &NodeHubConn{
		NodeID:          1,
		Send:            make(chan []byte, 1),
		ObservedAddress: "1.1.1.1",
		LastSeen:        time.Now().UTC(),
		ConnectedAt:     time.Now().UTC(),
	}
	h.nodes[1] = existing

	conn, err := h.RegisterNodeAgent(1, nil, "2.2.2.2")

	require.ErrorIs(t, err, ErrDuplicateIdentity)
	assert.Nil(t, conn)
	assert.Same(t, existing, h.nodes[1], "the established connection must stay registered")
}

func TestRegisterAgent_RejectsDuplicateIdentity(t *testing.T) {
	h := newTestHub()
	existing := &AgentHubConn{
		AgentID:         1,
		Send:            make(chan *dto.HubMessage, 1),
		ObservedAddress: "1.1.1.1",
		LastSeen:        time.Now().UTC(),
		ConnectedAt:     time.Now().UTC(),
	}
	h.agents[1] = existing

	conn, err := h.RegisterAgent(1, nil, "2.2.2.2")

	require.ErrorIs(t, err, ErrDuplicateIdentity)
	assert.Nil(t, conn)
	assert.Same(t, existing, h.agents[1], "the established connection must stay registered")
}

func TestUnregisterNodeAgent_KeepsSupersededConnection(t *testing.T) {
	h := newTestHub()
	old := &NodeHubConn{NodeID: 1, Send: make(chan []byte, 1), ObservedAddress: "1.1.1.1"}
	current := &NodeHubConn{NodeID: 1, Send: make(chan []byte, 1), ObservedAddress: "1.1.1.1"}
	h.nodes[1] = current

	// The read pump of the replaced connection must not drop its successor.
	h.UnregisterNodeAgent(1, old)
	assert.Same(t, current, h.nodes[1])

	h.UnregisterNodeAgent(1, current)
	_, ok := h.nodes[1]
	assert.False(t, ok, "the current connection must be removed by its own read pump")
}

func TestUnregisterAgent_KeepsSupersededConnection(t *testing.T) {
	h := newTestHub()
	old := &AgentHubConn{AgentID: 1, Send: make(chan *dto.HubMessage, 1), ObservedAddress: "1.1.1.1"}
	current := &AgentHubConn{AgentID: 1, Send: make(chan *dto.HubMessage, 1), ObservedAddress: "1.1.1.1"}
	h.agents[1] = current

	h.UnregisterAgent(1, old)
	assert.Same(t, current, h.agents[1])

	h.UnregisterAgent(1, current)
	_, ok := h.agents[1]
	assert.False(t, ok, "the current connection must be removed by its own read pump")
}
