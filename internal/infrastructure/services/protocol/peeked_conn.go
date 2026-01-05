package protocol

import (
	"io"
	"net"
)

// PeekedConn wraps a net.Conn and prepends already-read bytes.
// This allows re-reading the peeked bytes as if they were never read.
//
// Note: PeekedConn is NOT safe for concurrent use. Read and WriteTo
// must not be called concurrently as they both modify the internal offset.
// This is consistent with how net.Conn is typically used with io.Copy.
type PeekedConn struct {
	net.Conn
	peeked []byte
	offset int
}

// NewPeekedConn creates a connection that replays peeked bytes first.
func NewPeekedConn(conn net.Conn, peeked []byte) *PeekedConn {
	return &PeekedConn{
		Conn:   conn,
		peeked: peeked,
		offset: 0,
	}
}

// Read implements io.Reader, first returning peeked bytes then underlying conn.
func (c *PeekedConn) Read(b []byte) (int, error) {
	// If there are still peeked bytes to return
	if c.offset < len(c.peeked) {
		n := copy(b, c.peeked[c.offset:])
		c.offset += n
		return n, nil
	}

	// All peeked bytes consumed, read from underlying connection
	return c.Conn.Read(b)
}

// WriteTo implements io.WriterTo for potential splice optimization.
func (c *PeekedConn) WriteTo(w io.Writer) (int64, error) {
	var total int64

	// First write remaining peeked bytes
	if c.offset < len(c.peeked) {
		n, err := w.Write(c.peeked[c.offset:])
		c.offset += n
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	// Then copy from underlying connection
	// Check if underlying connection implements WriterTo for splice optimization
	if wt, ok := c.Conn.(io.WriterTo); ok {
		n, err := wt.WriteTo(w)
		total += n
		return total, err
	}

	// Fallback to io.Copy
	n, err := io.Copy(w, c.Conn)
	total += n
	return total, err
}
