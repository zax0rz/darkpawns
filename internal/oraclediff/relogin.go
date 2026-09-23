package oraclediff

import (
	"errors"
	"fmt"
	"time"
)

// ReloginStep is the probe step that ends the actor's connection and logs the
// same character in again on a new one. The whole login transcript is the
// step's diffed block.
//
// A dropped connection in C leaves the character in the world, linkdead, and
// the next login reattaches to that live body without reading the player
// file. A scenario that means to prove a save survives therefore quits first
// (C saves and extracts on quit) and then relogs, so both servers read the
// character back from their own persistence: C's player and rent files in the
// disposable lib copy, the Go port's database.
const ReloginStep = "<RELOGIN>"

// Relogger is a Conn that can replace its transport with a fresh login.
type Relogger interface {
	Conn
	Relogin(quiescence time.Duration) (string, error)
}

// ReloginConn is the actor connection of a scenario that uses ReloginStep. It
// forwards to the current transport and, on Relogin, closes it, dials the same
// server again, and plays the scenario's login lines for that server.
type ReloginConn struct {
	current Conn
	dial    func() (Conn, error)
	login   func(Conn) (string, error)
	// settle advances a DP_CLOCK-frozen server on the old connection before
	// it closes: C extracts a quitting character on the next pulse, and in
	// real play the socket is still open when that happens.
	settle func(Conn) (string, error)
}

// NewReloginConn wraps the actor's first connection. dial opens another
// connection to the same server; login plays the scenario's [relogin:*]
// lines on it and returns the transcript, greeting included.
func NewReloginConn(first Conn, dial func() (Conn, error), login func(Conn) (string, error), settle func(Conn) (string, error)) *ReloginConn {
	return &ReloginConn{current: first, dial: dial, login: login, settle: settle}
}

func (c *ReloginConn) Send(line string) error { return c.current.Send(line) }

func (c *ReloginConn) ReadUntilQuiescent(d time.Duration) (string, error) {
	return c.current.ReadUntilQuiescent(d)
}

func (c *ReloginConn) Close() error { return c.current.Close() }

// Relogin settles the server on the current transport, closes it, lets the
// server finish with it, and logs in again. Output still queued on the old
// transport (the tail of a quit, say) is kept at the head of the transcript,
// so nothing the server sent is dropped from the comparison.
func (c *ReloginConn) Relogin(quiescence time.Duration) (string, error) {
	tail, _ := c.current.ReadUntilQuiescent(quiescence)
	// Settle while the old connection is still open, keeping what the server
	// sends there (C's menu after a quit, for one). A server that already
	// closed it has nothing left to settle on that connection.
	if c.settle != nil {
		if settled, err := c.settle(c.current); err == nil {
			tail += settled
		}
	}
	if err := c.current.Close(); err != nil {
		return tail, fmt.Errorf("close before relogin: %w", err)
	}
	// Give the server a quiescence window to notice the close and run its
	// disconnect path (the save, for a character still in the game).
	time.Sleep(quiescence)
	next, err := c.dial()
	if err != nil {
		return tail, fmt.Errorf("dial for relogin: %w", err)
	}
	c.current = next
	transcript, err := c.login(next)
	if err != nil {
		return tail + transcript, fmt.Errorf("relogin: %w", err)
	}
	return tail + transcript, nil
}

// errNoRelogin is returned when a scenario uses ReloginStep on a connection
// that cannot reconnect.
var errNoRelogin = errors.New("probe uses " + ReloginStep + " but the actor connection cannot relogin (missing [relogin:oracle]/[relogin:port])")
