package oraclediff

import (
	"bufio"
	"errors"
	"io"
	"net"
	"strings"
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
	closed           bool
	firstByteWait    time.Duration
	awaitingResponse bool
}

func NewTCPConn(conn net.Conn) *TCPConn {
	return &TCPConn{conn: conn, reader: bufio.NewReader(conn)}
}

// FirstByteWaiter is implemented by transports that separate first-response
// latency from trailing quiescence.
type FirstByteWaiter interface {
	FirstByteWait() time.Duration
	SetFirstByteWait(d time.Duration)
}

// PromptWaiter is implemented by transports that wait until a target prompt
// suffix has appeared at least minCount times (after stripping ANSI escapes)
// before switching from maxWait to the trailing quiescence window. Used for
// multi-pulse steps (such as complex aliases that enter and exit the string
// editor across multiple MUD pulses) whose inter-pulse gaps can exceed the
// global quiescence window under load (DP-1592).
type PromptWaiter interface {
	ReadUntilPrompt(quiescence, maxWait time.Duration, suffix string, minCount int) (string, error)
}

func promptSatisfied(raw, suffix string, minCount int) bool {
	if suffix == "" || minCount <= 0 {
		return true
	}
	clean := ansiEscape.ReplaceAllString(raw, "")
	return strings.HasSuffix(clean, suffix) && strings.Count(clean, suffix) >= minCount
}

func readConnUntilPrompt(c Conn, quiescence, maxWait time.Duration, suffix string, minCount int) (string, error) {
	if pw, ok := c.(PromptWaiter); ok {
		return pw.ReadUntilPrompt(quiescence, maxWait, suffix, minCount)
	}
	return c.ReadUntilQuiescent(quiescence)
}

// SetFirstByteWait separates response latency after Send from trailing silence.
// Reads without a preceding Send (passive peers and drains) keep plain quiescence. A zero
// value preserves the legacy one-window reader for explicit timing controls.
func (c *TCPConn) SetFirstByteWait(d time.Duration) { c.firstByteWait = d }

// FirstByteWait returns the configured first-response wait duration.
func (c *TCPConn) FirstByteWait() time.Duration { return c.firstByteWait }

// ObservedClose reports whether the server closed this transport.
func (c *TCPConn) ObservedClose() bool { return c.closed }

func (c *TCPConn) Send(line string) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	_, err := io.WriteString(c.conn, line+"\r\n")
	c.awaitingResponse = err == nil
	return err
}

// ReadUntilQuiescent returns after the server has emitted no bytes for d. Telnet
// negotiation is transport framing, so it is consumed here and never reaches
// the transcript normalizer.
func (c *TCPConn) ReadUntilQuiescent(d time.Duration) (string, error) {
	firstWait := d
	if c.awaitingResponse {
		firstWait = max(d, c.firstByteWait)
	}
	c.awaitingResponse = false
	return c.readLoop(d, firstWait, "", 0)
}

// ReadUntilPrompt waits up to maxWait for suffix to appear at least minCount
// times at the end of output (after stripping ANSI escapes), then switches to
// the trailing quiescence window d so any immediately trailing bytes are still
// captured verbatim.
func (c *TCPConn) ReadUntilPrompt(d, maxWait time.Duration, suffix string, minCount int) (string, error) {
	wait := max(d, maxWait)
	if c.awaitingResponse {
		wait = max(wait, c.firstByteWait)
	}
	c.awaitingResponse = false
	return c.readLoop(d, wait, suffix, minCount)
}

func (c *TCPConn) readLoop(quiescence, maxWait time.Duration, suffix string, minCount int) (string, error) {
	var out []byte
	waitDeadline := time.Now().Add(maxWait)
	for {
		deadline := waitDeadline
		if len(out) > 0 && promptSatisfied(string(out), suffix, minCount) {
			deadline = time.Now().Add(quiescence)
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
