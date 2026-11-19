package session

import (
	"strings"
	"testing"
	"time"
)

// C exits its game loop before closing descriptors, so room acts queued by the
// final socket closes have no later output pass (src/comm.c:478-688,283-286).
func TestCommandShutdownDoesNotBroadcastSessionLeave(t *testing.T) {
	m := makeTestManager(t)
	god := makeTestSession(t, m, "ShutGod", 1001, true)
	peer := makeTestSession(t, m, "ShutPeer", 1001, true)
	registerTestSession(t, m, god, "ShutGod")
	registerTestSession(t, m, peer, "ShutPeer")

	m.ShutdownGracefullyWithoutNotice(time.Second)

	got := renderedOutput(god) + renderedOutput(peer)
	if strings.Contains(got, "has left the game.") {
		t.Fatalf("command shutdown emitted a per-session leave line: %q", got)
	}
}
