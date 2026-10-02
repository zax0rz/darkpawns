package oraclediff

import (
	"bufio"
	"errors"
	"io"
	"net"
	"time"
)

// Conn is the transport surface used by the scenario driver.
type Conn interface {
	Send(line string) error
	ReadUntilQuiescent(d time.Duration) (string, error)
	Close() error
}

// CloseReporter is implemented by transports that can report whether the
// server closed the connection. A scenario that declares the `compare-close`
// fixture renders CloseMarker into the block whose read observed the close, so
// a server-initiated disconnect is part of the compared transcript instead of
// an invisible end-of-data.
type CloseReporter interface {
	ObservedClose() bool
}

// CloseMarker is the canonical transcript line for a server-closed connection.
// It cannot be produced by either game engine's own output, so a block that
// carries it and a block that does not can never be a false match.
const CloseMarker = "<CLOSE>"

// connObservedClose reports whether a connection (or a wrapper around one) has
// observed the server closing its transport.
func connObservedClose(c Conn) bool {
	if reporter, ok := c.(CloseReporter); ok {
		return reporter.ObservedClose()
	}
	return false
}

// TCPConn drives either server over its telnet TCP listener.
type TCPConn struct {
	conn   net.Conn
	reader *bufio.Reader
	// closed records a server-side EOF seen by ReadUntilQuiescent. It is only
	// read after the read that set it, so the harness's single-threaded
	// per-connection driving needs no lock.
	closed        bool
	firstByteWait time.Duration
}

func NewTCPConn(conn net.Conn) *TCPConn {
	return &TCPConn{conn: conn, reader: bufio.NewReader(conn)}
}

// SetFirstByteWait separates response latency from trailing silence. A zero
// value preserves the legacy one-window reader for explicit timing controls.
func (c *TCPConn) SetFirstByteWait(d time.Duration) { c.firstByteWait = d }

// ObservedClose reports whether the server closed this transport.
func (c *TCPConn) ObservedClose() bool { return c.closed }

func (c *TCPConn) Send(line string) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	_, err := io.WriteString(c.conn, line+"\r\n")
	return err
}

// ReadUntilQuiescent returns after the server has emitted no bytes for d. Telnet
// negotiation is transport framing, so it is consumed here and never reaches
// the transcript normalizer.
func (c *TCPConn) ReadUntilQuiescent(d time.Duration) (string, error) {
	var out []byte
	firstDeadline := time.Now().Add(max(d, c.firstByteWait))
	for {
		deadline := firstDeadline
		if len(out) > 0 {
			deadline = time.Now().Add(d)
		}
		if err := c.conn.SetReadDeadline(deadline); err != nil {
			return string(out), err
		}
		b, err := c.reader.ReadByte()
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return string(out), nil
			}
			if errors.Is(err, io.EOF) {
				// The server closed this transport. Remember it so an opted-in
				// scenario can compare the disconnect itself (CloseMarker).
				c.closed = true
				if len(out) > 0 {
					return string(out), nil
				}
			}
			return string(out), err
		}
		if b == 255 { // IAC
			escaped, err := consumeIAC(c.reader)
			if err != nil {
				return string(out), err
			}
			if escaped {
				out = append(out, 255)
			}
			continue
		}
		out = append(out, b)
	}
}

func (c *TCPConn) Close() error {
	return c.conn.Close()
}

func consumeIAC(r *bufio.Reader) (escaped bool, err error) {
	cmd, err := r.ReadByte()
	if err != nil {
		return false, err
	}
	switch cmd {
	case 255: // IAC IAC - literal 0xFF byte
		return true, nil
	case 251, 252, 253, 254: // WILL, WONT, DO, DONT
		_, err = r.ReadByte()
		return false, err
	case 250: // SB ... IAC SE
		for {
			b, readErr := r.ReadByte()
			if readErr != nil {
				return false, readErr
			}
			if b != 255 {
				continue
			}
			next, readErr := r.ReadByte()
			if readErr != nil {
				return false, readErr
			}
			if next == 240 {
				return false, nil
			}
		}
	default:
		return false, nil
	}
}
