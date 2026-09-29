package oraclediff

import (
	"errors"
	"fmt"
	"time"
)

// CrashStep is the probe step that SIGKILLs the engine behind the actor's
// connection — immediately, with no settle, no quit, no transport close and
// no grace window first — waits for the process to exit, disposes of the
// dead transport, starts the same engine again on the same disposable data
// directory and ports, and logs the character back in. The whole relogin
// transcript is the step's diffed block.
//
// The ordering is the point. Everything RestartConn.Restart does before it
// signals the process — the settle pump, the transport close, the
// disconnect-processing window — is a chance for the engine to save the
// character, and each one masks what the step exists to prove: what the
// engine's DURABLE state held at the moment of the crash. A crash-recovery
// vehicle therefore must not reuse that pre-stop sequence (R5h: a step that
// cannot catch the unsaved state proves nothing about it).
//
// Like RestartStep, the crash applies only to the engine being probed, in
// its own pass, on its own disposable data directory; the other engine's
// idle connection is never touched from this pass.
//
// What the step proves per engine, on top of the restart contract:
//
//   - nothing the killed process had only in memory reaches the relogin:
//     characters are restored from their last durable save (C's player and
//     crash-rent files; the Go port's database), so a vehicle that changes
//     state and then crashes observes exactly what the engine persisted;
//   - the boot path runs from scratch, and the relogin transcript matches
//     byte for byte.
//
// The step requires the scenario's [relogin:*] lines like RestartStep does.
// C's own crash is not byte-faithful for character-record fields: save_char
// fwrite's without flushing (src/db.c:2404-2405), so gold or experience can
// lag the crash-rent objects that Crash_crashsave fclose'd. Vehicles compare
// objects; character-record parity is Go-test territory (R5f).
const CrashStep = "<CRASH>"

// Crasher is a Conn whose engine can be crashed (SIGKILL, then restarted).
type Crasher interface {
	Conn
	Crash(quiescence time.Duration) (string, error)
}

// CrashConn is the actor connection of a scenario that uses CrashStep. It
// forwards to the current transport and, on Crash, kills the engine behind
// it, replaces the transport, and logs in again. It deliberately has no
// settle hook and performs no read, close or sleep before the kill.
type CrashConn struct {
	current Conn
	dial    func() (Conn, error)
	login   func(Conn) (string, error)
	// kill stops the engine behind the connection with an immediate SIGKILL
	// and returns only once the process has exited.
	kill func() error
	// restart starts the engine again on the same disposable data directory
	// and ports, returning only once it is serving again.
	restart func() error
}

// NewCrashConn wraps the actor's first connection. dial opens another
// connection to the same engine; login plays the scenario's [relogin:*]
// lines on it; kill SIGKILLs the engine and waits for the exit; restart
// starts it again.
func NewCrashConn(first Conn, dial func() (Conn, error), login func(Conn) (string, error), kill func() error, restart func() error) *CrashConn {
	return &CrashConn{current: first, dial: dial, login: login, kill: kill, restart: restart}
}

func (c *CrashConn) Send(line string) error { return c.current.Send(line) }

func (c *CrashConn) ReadUntilQuiescent(d time.Duration) (string, error) {
	return c.current.ReadUntilQuiescent(d)
}

func (c *CrashConn) Close() error { return c.current.Close() }

// ObservedClose forwards the current transport's server-close observation.
func (c *CrashConn) ObservedClose() bool { return connObservedClose(c.current) }

// Crash SIGKILLs the engine behind the connection and waits for the process
// to exit, closes the dead transport, restarts the engine on the same
// disposable data directory and ports, then dials and logs in again.
//
// Nothing happens before the kill: no queued-output read, no settle, no
// close, no disconnect window — any of those could let the engine save and
// turn the crash into a graceful stop. The killed transport's buffered
// output is dropped with it, as it would be for a player whose server died.
func (c *CrashConn) Crash(quiescence time.Duration) (string, error) {
	if err := c.kill(); err != nil {
		return "", fmt.Errorf("crash engine: %w", err)
	}
	if err := c.current.Close(); err != nil {
		return "", fmt.Errorf("close crashed transport: %w", err)
	}
	if err := c.restart(); err != nil {
		return "", fmt.Errorf("restart engine after crash: %w", err)
	}
	next, err := c.dial()
	if err != nil {
		return "", fmt.Errorf("dial after crash: %w", err)
	}
	c.current = next
	transcript, err := c.login(next)
	if err != nil {
		return transcript, fmt.Errorf("relogin after crash: %w", err)
	}
	return transcript, nil
}

// errNoCrash is returned when a scenario uses CrashStep on a connection that
// cannot crash its engine.
var errNoCrash = errors.New("probe uses " + CrashStep + " but the actor connection cannot crash its engine (missing [relogin:oracle]/[relogin:port])")
