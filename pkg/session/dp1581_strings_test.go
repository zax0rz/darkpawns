package session

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestShutdownGracefullyNotifyUsesCBytes verifies that OS-signal shutdown
// notifications send C's exact shutdown bytes (src/act.wizard.c:1099,1105)
// rather than an invented Go banner.
func TestShutdownGracefullyNotifyUsesCBytes(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Shuttest", 1001, true)
	m.mu.Lock()
	m.sessions[s.player.Name] = s
	m.mu.Unlock()

	m.shutdownGracefully(time.Second, true)

	got := readSessionText(t, s)
	want := "Shutting down for maintenance.\r\n"
	if got != want {
		t.Fatalf("shutdownGracefully(true) sent %q, want C's %q (src/act.wizard.c:1099,1105)", got, want)
	}
}

// TestCmdAliasWriteFailureStillEmitsCOutput verifies that alias save failures
// log internally and still send C's exact output (src/interpreter.c:1013,1036)
// rather than an invented "Error saving aliases." message.
func TestCmdAliasWriteFailureStillEmitsCOutput(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "../bad", 1001, true)
	s.player.Aliases = []game.PlayerAlias{{Alias: "b", Replacement: " bounce"}}

	if err := cmdAlias(s, []string{"a", "wave"}); err != nil {
		t.Fatalf("cmdAlias add: %v", err)
	}
	if got, want := readSessionText(t, s), "Alias added."; got != want {
		t.Fatalf("cmdAlias add on write failure = %q, want C's %q (src/interpreter.c:1036)", got, want)
	}

	if err := cmdAlias(s, []string{"a"}); err != nil {
		t.Fatalf("cmdAlias delete: %v", err)
	}
	if got, want := readSessionText(t, s), "Alias deleted."; got != want {
		t.Fatalf("cmdAlias delete on write failure = %q, want C's %q (src/interpreter.c:1013)", got, want)
	}
}
