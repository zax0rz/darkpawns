package oraclediff

import (
	"errors"
	"fmt"
	"time"
)

// RestartStep is the probe step that stops the engine behind the actor's
// connection, starts that same engine again on the same disposable data
// directory and ports, and logs the character back in. The whole login
// transcript is the step's diffed block.
//
// Only the engine being probed is bounced. A probe is played twice — once
// against the C oracle, once against the Go port — and each pass drives one
// engine while the other engine's connection sits idle. Bouncing the peer
// engine from the actor's step would kill that idle connection during the other
// pass and desynchronise the block alignment, so each engine gets its own
// genuine stop/start, in its own pass, on its own disposable data directory.
//
// What the step therefore proves per engine, in C and in Go:
//
//   - the process really stops and really starts again (the port is rebound and
//     the boot path runs from scratch: area files plus zone resets);
//   - durable data in that engine's disposable directory survives (C's player,
//     rent, board, mail, house and clan files in the copied lib; the Go port's
//     world copy, runtime directory and database);
//   - transient world state does not (dropped objects, corpses, money, ash, mob
//     positions and HP, door bytes, recent gossip — RULEBOOK R4);
//   - the login transcript after the restart matches byte for byte.
//
// The step requires a quitting or linkdead-ended session first when the
// scenario means to prove a save survived: C saves on quit, and the C shutdown
// path itself saves nothing but the clan table, the whod file, the mud date and
// the closed player file (src/comm.c:288-293).
const RestartStep = "<RESTART>"

// Restarter is a Conn whose engine can be stopped and started again.
type Restarter interface {
	Conn
	Restart(quiescence time.Duration) (string, error)
}

// RestartConn is the actor connection of a scenario that uses RestartStep. It
// behaves like ReloginConn — forwarding to the current transport, replacing it
// on demand — and stops/restarts the engine in between.
type RestartConn struct {
	current Conn
	dial    func() (Conn, error)
	login   func(Conn) (string, error)
	// settle advances the engine on the old connection before it closes, the
	// same way ReloginConn does: C extracts a quitting character on the next
	// pulse, and in real play the socket is still open when that happens.
	settle func(Conn) (string, error)
	// restart stops the engine and starts it again on the same disposable data
	// directory and ports, returning only once it is serving again.
	restart func() error
}

// NewRestartConn wraps the actor's first connection. dial opens another
// connection to the same engine; login plays the scenario's [relogin:*] lines
// on it; restart bounces the engine behind the connection.
func NewRestartConn(first Conn, dial func() (Conn, error), login func(Conn) (string, error), settle func(Conn) (string, error), restart func() error) *RestartConn {
	return &RestartConn{current: first, dial: dial, login: login, settle: settle, restart: restart}
}

func (c *RestartConn) Send(line string) error { return c.current.Send(line) }

func (c *RestartConn) ReadUntilQuiescent(d time.Duration) (string, error) {
	return c.current.ReadUntilQuiescent(d)
}

func (c *RestartConn) Close() error { return c.current.Close() }

// Restart settles and closes the current transport, stops and starts the engine
// behind it, then dials and logs in again. Output still queued on the old
// transport (the tail of a quit, say) is kept at the head of the transcript, so
// nothing the old process sent is dropped from the comparison.
func (c *RestartConn) Restart(quiescence time.Duration) (string, error) {
	tail, _ := c.current.ReadUntilQuiescent(quiescence)
	if c.settle != nil {
		if settled, err := c.settle(c.current); err == nil {
			tail += settled
		}
	}
	if err := c.current.Close(); err != nil {
		return tail, fmt.Errorf("close before restart: %w", err)
	}
	// Give the engine a quiescence window to run its disconnect path (the save,
	// for a character still in the game) before the process is signalled.
	time.Sleep(quiescence)
	if err := c.restart(); err != nil {
		return tail, fmt.Errorf("restart engine: %w", err)
	}
	next, err := c.dial()
	if err != nil {
		return tail, fmt.Errorf("dial after restart: %w", err)
	}
	c.current = next
	transcript, err := c.login(next)
	if err != nil {
		return tail + transcript, fmt.Errorf("relogin after restart: %w", err)
	}
	return tail + transcript, nil
}

// errNoRestart is returned when a scenario uses RestartStep on a connection
// that cannot restart its engine.
var errNoRestart = errors.New("probe uses " + RestartStep + " but the actor connection cannot restart (missing [relogin:oracle]/[relogin:port])")
