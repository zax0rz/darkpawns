package session

import (
	"strings"
	"testing"
)

// Kick must close a live session (case-insensitively, like GetSession) with
// the notice queued before the close, and report false for offline names.
func TestManagerKick(t *testing.T) {
	m := makeTestManager(t)
	online := makeTestSession(t, m, "Kickme", 1001, true)
	m.mu.Lock()
	m.sessions["Kickme"] = online
	m.mu.Unlock()

	if m.Kick("Nobody", "bye") {
		t.Fatal("Kick for an offline name returned true")
	}

	if !m.Kick("kickme", "You have been disconnected by an administrator.") {
		t.Fatal("Kick returned false for a live session")
	}
	if !online.SendClosed() {
		t.Fatal("kicked session's send channel not closed")
	}

	// The notice was queued before the close: buffered frames are still
	// readable from the closed channel.
	msg, ok := drainSend(online)
	if !ok {
		t.Fatal("expected the kick notice on the send channel")
	}
	if !strings.Contains(string(msg), "disconnected by an administrator") {
		t.Fatalf("notice = %q, want the kick notice", msg)
	}
}
