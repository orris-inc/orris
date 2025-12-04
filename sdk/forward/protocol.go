package forward

import (
	"encoding/binary"
	"errors"
	"io"
)

// MessageType represents the type of tunnel message.
type MessageType uint8

const (
	// MsgConnect requests a new connection to the target.
	MsgConnect MessageType = 1
	// MsgData carries data for an existing connection.
	MsgData MessageType = 2
	// MsgClose closes a connection.
	MsgClose MessageType = 3
	// MsgPing is a heartbeat message.
	MsgPing MessageType = 4
	// MsgPong is a heartbeat response.
	MsgPong MessageType = 5
)

// TunnelMessage represents a message in the tunnel protocol.
// Wire format: [type:1][connID:8][length:4][payload:length]
type TunnelMessage struct {
	Type    MessageType
	ConnID  uint64
	Payload []byte
}

const (
	// HeaderSize is the size of the message header.
	HeaderSize = 1 + 8 + 4 // type + connID + length
	// MaxPayloadSize is the maximum payload size.
	MaxPayloadSize = 64 * 1024 // 64KB
)

var (
	// ErrPayloadTooLarge is returned when payload exceeds MaxPayloadSize.
	ErrPayloadTooLarge = errors.New("payload too large")
	// ErrInvalidMessage is returned when message format is invalid.
	ErrInvalidMessage = errors.New("invalid message format")
)

// Encode encodes the message to bytes.
func (m *TunnelMessage) Encode() ([]byte, error) {
	if len(m.Payload) > MaxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	buf := make([]byte, HeaderSize+len(m.Payload))
	buf[0] = byte(m.Type)
	binary.BigEndian.PutUint64(buf[1:9], m.ConnID)
	binary.BigEndian.PutUint32(buf[9:13], uint32(len(m.Payload)))
	copy(buf[13:], m.Payload)

	return buf, nil
}

// DecodeMessage decodes a message from reader.
func DecodeMessage(r io.Reader) (*TunnelMessage, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	msgType := MessageType(header[0])
	connID := binary.BigEndian.Uint64(header[1:9])
	length := binary.BigEndian.Uint32(header[9:13])

	if length > MaxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}

	return &TunnelMessage{
		Type:    msgType,
		ConnID:  connID,
		Payload: payload,
	}, nil
}

// NewConnectMessage creates a connect message.
func NewConnectMessage(connID uint64) *TunnelMessage {
	return &TunnelMessage{
		Type:   MsgConnect,
		ConnID: connID,
	}
}

// NewDataMessage creates a data message.
func NewDataMessage(connID uint64, data []byte) *TunnelMessage {
	return &TunnelMessage{
		Type:    MsgData,
		ConnID:  connID,
		Payload: data,
	}
}

// NewCloseMessage creates a close message.
func NewCloseMessage(connID uint64) *TunnelMessage {
	return &TunnelMessage{
		Type:   MsgClose,
		ConnID: connID,
	}
}

// NewPingMessage creates a ping message.
func NewPingMessage() *TunnelMessage {
	return &TunnelMessage{
		Type: MsgPing,
	}
}

// NewPongMessage creates a pong message.
func NewPongMessage() *TunnelMessage {
	return &TunnelMessage{
		Type: MsgPong,
	}
}
