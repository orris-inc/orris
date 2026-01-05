// Package protocol provides protocol detection and sniffing functionality.
package protocol

import (
	"bytes"
	"net"
	"time"
)

// Protocol represents the detected protocol type.
type Protocol string

const (
	ProtocolUnknown     Protocol = "unknown"
	ProtocolHTTPConnect Protocol = "http_connect"
	ProtocolSOCKS4      Protocol = "socks4"
	ProtocolSOCKS5      Protocol = "socks5"
	ProtocolHTTP        Protocol = "http"
	ProtocolTLS         Protocol = "tls"
	ProtocolSSH         Protocol = "ssh"
	ProtocolFTP         Protocol = "ftp"
)

const (
	// defaultPeekSize is the number of bytes to peek for protocol detection.
	defaultPeekSize = 16

	// defaultSniffTimeout is the timeout for reading the first bytes.
	defaultSniffTimeout = 5 * time.Second
)

// httpMethods contains HTTP method prefixes for detection.
var httpMethods = [][]byte{
	[]byte("GET "),
	[]byte("POST "),
	[]byte("PUT "),
	[]byte("DELETE "),
	[]byte("HEAD "),
	[]byte("OPTIONS "),
	[]byte("PATCH "),
}

// ftpCommands contains FTP command prefixes for detection (uppercase).
// We use hasPrefixFold for case-insensitive matching without allocation.
var ftpCommands = [][]byte{
	[]byte("USER "), []byte("USER\r"),
	[]byte("PASS "), []byte("PASS\r"),
	[]byte("QUIT "), []byte("QUIT\r"),
	[]byte("PORT "), []byte("PORT\r"),
	[]byte("PASV "), []byte("PASV\r"),
	[]byte("LIST "), []byte("LIST\r"),
	[]byte("RETR "), []byte("RETR\r"),
}

// hasPrefixFold checks if data has the given prefix, case-insensitively.
// This avoids memory allocation compared to bytes.ToUpper + bytes.HasPrefix.
func hasPrefixFold(data, prefix []byte) bool {
	if len(data) < len(prefix) {
		return false
	}
	for i, b := range prefix {
		if toUpper(data[i]) != toUpper(b) {
			return false
		}
	}
	return true
}

// toUpper converts a byte to uppercase ASCII.
func toUpper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - ('a' - 'A')
	}
	return b
}

// ProtocolInfo contains information about the detected protocol.
type ProtocolInfo struct {
	Protocol Protocol
	Raw      []byte // first bytes that were peeked
}

// Sniffer detects the protocol of incoming connections.
type Sniffer struct {
	peekSize int
	timeout  time.Duration
}

// NewSniffer creates a new protocol sniffer with default settings.
// Default peek size is 16 bytes and timeout is 5 seconds.
func NewSniffer() *Sniffer {
	return &Sniffer{
		peekSize: defaultPeekSize,
		timeout:  defaultSniffTimeout,
	}
}

// Sniff reads the first bytes from the connection to detect the protocol.
// It returns the detected protocol info and a wrapped connection that includes
// the peeked bytes, allowing the caller to continue reading from the beginning.
func (s *Sniffer) Sniff(conn net.Conn) (*ProtocolInfo, *PeekedConn, error) {
	// Set read deadline for sniffing
	if err := conn.SetReadDeadline(time.Now().Add(s.timeout)); err != nil {
		return nil, nil, err
	}

	// Read first bytes
	// Note: Per io.Reader spec, Read may return n > 0 and err != nil simultaneously
	// (e.g., reading some data before EOF or timeout). We should process any data read.
	buf := make([]byte, s.peekSize)
	n, readErr := conn.Read(buf)

	// Clear read deadline regardless of read result
	_ = conn.SetReadDeadline(time.Time{})

	// If no data was read, return the error
	if n == 0 {
		if readErr != nil {
			return nil, nil, readErr
		}
		// n == 0 with no error is unusual but possible; treat as unknown protocol
	}

	data := buf[:n]
	info := s.detect(data)

	// Wrap connection with peeked data
	peekedConn := NewPeekedConn(conn, data)

	return info, peekedConn, nil
}

// detect analyzes the peeked data and returns the detected protocol.
func (s *Sniffer) detect(data []byte) *ProtocolInfo {
	info := &ProtocolInfo{
		Protocol: ProtocolUnknown,
		Raw:      data,
	}

	if len(data) == 0 {
		return info
	}

	// SOCKS5: first byte is 0x05
	if data[0] == 0x05 {
		info.Protocol = ProtocolSOCKS5
		return info
	}

	// SOCKS4: first byte is 0x04, second byte is 0x01 (CONNECT) or 0x02 (BIND)
	if len(data) >= 2 && data[0] == 0x04 && (data[1] == 0x01 || data[1] == 0x02) {
		info.Protocol = ProtocolSOCKS4
		return info
	}

	// TLS: first byte is 0x16 (handshake), second byte is 0x03 (SSL/TLS version)
	if len(data) >= 2 && data[0] == 0x16 && data[1] == 0x03 {
		info.Protocol = ProtocolTLS
		return info
	}

	// SSH: starts with "SSH-"
	if bytes.HasPrefix(data, []byte("SSH-")) {
		info.Protocol = ProtocolSSH
		return info
	}

	// HTTP CONNECT: starts with "CONNECT "
	if bytes.HasPrefix(data, []byte("CONNECT ")) {
		info.Protocol = ProtocolHTTPConnect
		return info
	}

	// HTTP: starts with common HTTP methods
	for _, method := range httpMethods {
		if bytes.HasPrefix(data, method) {
			info.Protocol = ProtocolHTTP
			return info
		}
	}

	// FTP: starts with FTP commands (case-insensitive, zero allocation)
	for _, prefix := range ftpCommands {
		if hasPrefixFold(data, prefix) {
			info.Protocol = ProtocolFTP
			return info
		}
	}

	return info
}
