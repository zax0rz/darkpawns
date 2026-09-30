package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

// savedLoginWaitConn models the source gate relevant to this fixture:
// src/interpreter.c:1766 assigns wait=1 after loading the saved name, and
// src/comm.c's descriptor phase cannot process the password until a pulse.
type savedLoginWaitConn struct {
	named, loggedIn bool
	wait            int
	queuedPassword  bool
}

func (c *savedLoginWaitConn) Send(line string) error {
	switch line {
	case "Reconnector":
		c.named, c.wait = true, 1
	case "~dpclock pulse 1":
		if c.wait > 0 {
			c.wait--
		}
		if c.wait == 0 && c.queuedPassword {
			c.loggedIn = true
		}
	case "reconpass":
		if c.named && c.wait == 0 {
			c.loggedIn = true
		} else {
			c.queuedPassword = true
		}
	}
	return nil
}
func (c *savedLoginWaitConn) ReadUntilQuiescent(time.Duration) (string, error) { return "", nil }
func (c *savedLoginWaitConn) Close() error                                     { return nil }

func TestReconnectFixturePumpsSavedLoginWaitBeforePassword(t *testing.T) {
	data, err := scenarioFiles.ReadFile("scenarios/lifecycle-reconnect-linkdead.txt")
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := oraclediff.ParseScenario("lifecycle-reconnect-linkdead", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for engine, setup := range map[string][]string{"oracle": scenario.ReloginOracle, "port": scenario.ReloginPort} {
		t.Run(engine, func(t *testing.T) {
			conn := &savedLoginWaitConn{}
			if _, err := oraclediff.RunSetup(conn, setup, time.Millisecond); err != nil {
				t.Fatal(err)
			}
			if !conn.loggedIn {
				t.Fatal("saved character's password remains queued behind nanny wait=1")
			}
			if conn.queuedPassword {
				t.Fatal("password arrived before the fixture pumped the nanny wait")
			}
		})
	}
}
