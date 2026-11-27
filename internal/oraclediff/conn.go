package oraclediff

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
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

// PulseDrainTimeout returns the maximum wait for a frozen-clock pulse barrier
// of n pulses on a live world.
func PulseDrainTimeout(pulses int) time.Duration {
	wait := time.Duration(pulses) * 25 * time.Millisecond
	if wait < 15*time.Second {
		wait = 15 * time.Second
	}
	if wait > 90*time.Second {
		wait = 90 * time.Second
	}
	return wait
}

// DrainPulseBarrier proves that an engine has drained every queued DP_CLOCK
// pulse by sending an empty line ("", recorded in scenarios as "<ENTER>") and
// waiting for the command prompt "> ". Both engines process the empty line
// only after all queued pulses have completed synchronously:
//   - C: game_loop drains dp_seam_take() before dp_command_phase pops t->input
//     (src/comm.c:1899-1920, 2233).
//   - Go: HandleClockControl runs GameLoop.PumpPulses(n) synchronously before
//     the telnet reader goroutine reads the next line.
//
// NOTE on C side effects: process_dpclock_control (src/comm.c:1899-1920) only
// accepts "~dpclock pulse <count>" (1 <= count <= DP_CLOCK_MAX_PENDING) and
// emits no reply. Every other input line, including "", goes through
// process_descriptor_command (src/comm.c:475-492), which resets
// d->character->char_specials.timer = 0, pulls an idle character back from the
// void with act("$n has returned."), and clears d->has_prompt (cf. check_idling
// at src/limits.c:419-427, IDLE_TO_VOID = 8 at src/utils.h:131). Any live
// lockstep session that sends this barrier MUST therefore record "<ENTER>" as
// a real probe step in its output scenario so census replay executes the exact
// same timeline.
func DrainPulseBarrier(c Conn, pulses int, quiescence time.Duration) (string, error) {
	if connObservedClose(c) {
		return "", nil
	}
	if err := c.Send(""); err != nil {
		return "", err
	}
	out, err := readConnUntilPrompt(c, quiescence, PulseDrainTimeout(pulses), "> ", 1)
	if err != nil && errors.Is(err, io.EOF) {
		return out, nil
	}
	return out, err
}

// RunPulseWithBarrier sends "~dpclock pulse N", waits long enough for any
// live-world pulse output to arrive on both engines (preventing late Go output
// from bleeding into the barrier read), and then sends an empty-line ("<ENTER>")
// barrier and waits for the prompt "> " so both engines are guaranteed to have
// drained all N queued pulses before returning. When the barrier is sent, it
// is returned as a second AudienceProbeBlock so callers can compare it and
// record "<ENTER>" in the scenario to keep lockstep and batch replay identical.
func RunPulseWithBarrier(c Conn, pulses int, quiescence time.Duration, quietMobs bool) (AudienceProbeBlock, *AudienceProbeBlock, error) {
	stepLine := pulseControl + strconv.Itoa(pulses)
	if err := c.Send(stepLine); err != nil {
		return AudienceProbeBlock{}, nil, fmt.Errorf("send %q: %w", stepLine, err)
	}
	if waiter, ok := c.(FirstByteWaiter); ok && !quietMobs && pulses >= 100 {
		prev := waiter.FirstByteWait()
		pulseWait := min(time.Duration(pulses)*12*time.Millisecond, 60*time.Second)
		if pulseWait > prev {
			waiter.SetFirstByteWait(pulseWait)
			defer waiter.SetFirstByteWait(prev)
		}
	}
	out, err := c.ReadUntilQuiescent(quiescence)
	if err != nil && !errors.Is(err, io.EOF) {
		return AudienceProbeBlock{}, nil, fmt.Errorf("read after %q: %w\noutput so far:\n%s", stepLine, err, out)
	}
	pulseBlk := AudienceProbeBlock{
		Command:  stepLine,
		Audience: "actor",
		Output:   out,
		Closed:   connObservedClose(c),
	}
	var barrierBlk *AudienceProbeBlock
	if !connObservedClose(c) {
		bOut, bErr := DrainPulseBarrier(c, pulses, quiescence)
		if bErr != nil {
			return AudienceProbeBlock{}, nil, fmt.Errorf("pulse barrier after %q: %w\noutput so far:\n%s", stepLine, bErr, out)
		}
		barrierBlk = &AudienceProbeBlock{
			Command:  "",
			Audience: "actor",
			Output:   bOut,
			Closed:   connObservedClose(c),
		}
	}
	return pulseBlk, barrierBlk, nil
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
