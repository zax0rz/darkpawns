package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// renderedOutput drains a session's send channel through the shared terminal
// renderer, so assertions see the bytes a telnet client would.
func renderedOutput(s *Session) string {
	var out strings.Builder
	for {
		select {
		case msg, open := <-s.send:
			if !open {
				return out.String()
			}
			if f, ok := RenderTerminalFrame(msg); ok && f.Kind != FrameGMCP {
				out.WriteString(f.Text)
			}
		default:
			return out.String()
		}
	}
}

// dupeCheckFixture registers a playing "Returner" (the old session) and an
// observer in the same room, and builds the new login's session holding a
// freshly loaded copy of the character, as the password step leaves it.
func dupeCheckFixture(t *testing.T) (m *Manager, old, observer, fresh *Session) {
	t.Helper()
	m = makeTestManagerWithVoidRooms(t)
	old = makeTestSession(t, m, "Returner", 1001, true)
	old.transportDone = make(chan struct{})
	registerTestSession(t, m, old, "Returner")
	observer = makeTestSession(t, m, "Watcher", 1001, true)
	observer.player = game.NewPlayer(2, "Watcher", 1001)
	registerTestSession(t, m, observer, "Watcher")
	fresh = makeTestSession(t, m, "Returner", 1001, true)
	return m, old, observer, fresh
}

func assertTookOver(t *testing.T, m *Manager, old, fresh *Session) {
	t.Helper()
	if fresh.player != old.player {
		t.Fatal("the new login did not take over the live body")
	}
	if got, _ := m.GetSession("Returner"); got != fresh {
		t.Fatal("the name is not registered to the new session")
	}
	if !old.superseded.Load() {
		t.Fatal("the old session was not marked superseded")
	}
	if fresh.player.IsLinkless() {
		t.Fatal("the reconnected character is still linkless")
	}
}

// RECON (interpreter.c:1634-1641): a linkdead body is resumed in place.
func TestPerformDupeCheckReconnectsLinkdeadBody(t *testing.T) {
	m, old, observer, fresh := dupeCheckFixture(t)
	old.player.SetLinkless(true)
	old.DetachTransport()
	old.player.SetPosition(game.PosResting)

	if !fresh.performDupeCheck() {
		t.Fatal("performDupeCheck did not reconnect")
	}
	assertTookOver(t, m, old, fresh)
	if fresh.player.GetPosition() != game.PosResting {
		t.Fatal("the reconnected body lost its in-memory state")
	}
	if got := renderedOutput(fresh); !strings.HasPrefix(got, "\r\nReconnecting.\r\n") {
		t.Fatalf("player output = %q, want echo_on CRLF then Reconnecting.", got)
	}
	if got := renderedOutput(observer); !strings.Contains(got, "Returner has reconnected.") {
		t.Fatalf("room output = %q, want the reconnect line", got)
	}
}

// USURP (interpreter.c:1552-1560, 1642-1649): a body still in use is taken
// over, and the old connection is told and disconnected.
func TestPerformDupeCheckUsurpsBodyInUse(t *testing.T) {
	m, old, observer, fresh := dupeCheckFixture(t)

	if !fresh.performDupeCheck() {
		t.Fatal("performDupeCheck did not take over the body")
	}
	assertTookOver(t, m, old, fresh)
	if got := renderedOutput(old); !strings.Contains(got,
		"\r\nThis body has been usurped!\r\n\r\nMultiple login detected -- disconnecting.\r\n") {
		t.Fatalf("old session output = %q", got)
	}
	if got := renderedOutput(fresh); !strings.Contains(got, "You take over your own body, already in use!\r\n") {
		t.Fatalf("player output = %q", got)
	}
	if got := renderedOutput(observer); !strings.Contains(got, "body has been taken over by a new spirit!") {
		t.Fatalf("room output = %q", got)
	}

	// The old transport's teardown runs after the takeover. It must neither
	// leave the body linkdead nor unregister the new session (the bug in the
	// old takeover path, which removed whatever held the name).
	if m.HandleTransportDisconnect(old) {
		t.Fatal("a superseded session went linkdead")
	}
	m.UnregisterSession(old)
	if got, _ := m.GetSession("Returner"); got != fresh {
		t.Fatal("the old session's teardown unregistered the new session")
	}
	if _, ok := m.world.GetPlayer("Returner"); !ok {
		t.Fatal("the old session's teardown removed the character from the world")
	}
}

// An old session inside an OLC editor is not CON_PLAYING: C gives it only
// "Multiple login detected", then finds the body descriptor-less and
// reconnects (interpreter.c:1552-1571, 1590-1604). cleanup_olc runs with
// d->character already NULL, so the room hears no "stops using OLC".
func TestPerformDupeCheckFromEditorReconnects(t *testing.T) {
	m, old, observer, fresh := dupeCheckFixture(t)
	old.textEdit = &textEditState{} // a tedit buffer: CON_TEDIT

	if !fresh.performDupeCheck() {
		t.Fatal("performDupeCheck did not reconnect")
	}
	assertTookOver(t, m, old, fresh)
	if old.textEdit != nil {
		t.Fatal("the old session's editor was not freed")
	}
	oldOut := renderedOutput(old)
	if !strings.Contains(oldOut, "\r\nMultiple login detected -- disconnecting.\r\n") || strings.Contains(oldOut, "usurped") {
		t.Fatalf("old session output = %q, want only the multiple-login notice", oldOut)
	}
	if got := renderedOutput(fresh); !strings.Contains(got, "Reconnecting.\r\n") {
		t.Fatalf("player output = %q, want Reconnecting.", got)
	}
	room := renderedOutput(observer)
	if !strings.Contains(room, "Returner has reconnected.") || strings.Contains(room, "stops using OLC") {
		t.Fatalf("room output = %q, want only the reconnect line", room)
	}
}

// UNSWITCH (interpreter.c:1545-1556, 1650-1653): an immortal switched into
// another body gets their own body back, with no room message.
func TestPerformDupeCheckReturnsSwitchedImmortal(t *testing.T) {
	m, old, observer, fresh := dupeCheckFixture(t)
	old.isSwitched = true

	if !fresh.performDupeCheck() {
		t.Fatal("performDupeCheck did not reconnect the switched immortal")
	}
	assertTookOver(t, m, old, fresh)
	if got := renderedOutput(old); !strings.Contains(got, "\r\nMultiple login detected -- disconnecting.\r\n") ||
		strings.Contains(got, "usurped") {
		t.Fatalf("old session output = %q", got)
	}
	if got := renderedOutput(fresh); !strings.Contains(got, "Reconnecting to unswitched char.") {
		t.Fatalf("player output = %q, want the unswitch line", got)
	}
	if got := renderedOutput(observer); strings.Contains(got, "Returner") {
		t.Fatalf("room output = %q, want nothing", got)
	}
}

// With no copy of the character in the game, the login carries on to the
// MOTD and menu.
func TestPerformDupeCheckWithoutLiveCopy(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	fresh := makeTestSession(t, m, "Newcomer", 1001, true)
	if fresh.performDupeCheck() {
		t.Fatal("performDupeCheck reconnected with nothing to reconnect to")
	}
}
