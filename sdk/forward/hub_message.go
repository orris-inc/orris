package forward

import (
	"fmt"
	"time"
)

// Send sends a message to the server.
func (hc *HubConn) Send(msg *HubMessage) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.closed {
		return ErrConnectionClosed
	}

	select {
	case hc.send <- msg:
		return nil
	default:
		return fmt.Errorf("send channel full")
	}
}

// SendStatus sends a status update to the server.
func (hc *HubConn) SendStatus(status *AgentStatus) error {
	msg := &HubMessage{
		Type:      MsgTypeStatus,
		Timestamp: time.Now().Unix(),
		Data:      status,
	}
	return hc.Send(msg)
}

// SendProbeResult sends a probe result to the server.
func (hc *HubConn) SendProbeResult(result *ProbeTaskResult) error {
	msg := &HubMessage{
		Type:      MsgTypeProbeResult,
		Timestamp: time.Now().Unix(),
		Data:      result,
	}
	return hc.Send(msg)
}

// SendConfigAck sends a configuration acknowledgment to the server.
func (hc *HubConn) SendConfigAck(ack *ConfigAckData) error {
	msg := &HubMessage{
		Type:      MsgTypeConfigAck,
		Timestamp: time.Now().Unix(),
		Data:      ack,
	}
	return hc.Send(msg)
}

// SendEvent sends an event to the server.
func (hc *HubConn) SendEvent(eventType, message string, extra any) error {
	msg := &HubMessage{
		Type:      MsgTypeEvent,
		Timestamp: time.Now().Unix(),
		Data: map[string]any{
			"event_type": eventType,
			"message":    message,
			"extra":      extra,
		},
	}
	return hc.Send(msg)
}
