package session

// bare_literal_test.go — PF-063 cluster 4 player-level literal seam proofs:
// a game producer (World.ExecFill / World.ExecRecall) calls Player.SendLiteral,
// the manager's LiteralMessageSink resolves the controlling session and calls
// sendLiteralText, and the bytes render exactly as C wrote them with the
// following prompt — noncompact framed, compact not. Reverting any wired call
// site back to SendMessage appends a terminator and fails (mutation checks).
// Snoop and switched-descriptor routing are covered at the same seam.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestLiteralSeamFillMissEndToEnd — do_pour SCMD_FILL's miss is sent bare
// (src/act.item.c:1196); the producer is the game package itself.
func TestLiteralSeamFillMissEndToEnd(t *testing.T) {
	for _, compact := range []bool{false, true} {
		w, s := newBareSendItemHarness(t)
		setCompact(s, compact)
		// "bucket" is carried by nobody → the bare miss.
		w.ExecFill(s.player, "bucket fountain")
		assertBareWire(t, s, "You can't find it!")
	}
}

// TestLiteralSeamRecallFightingEndToEnd — do_recall's fighting gate is sent
// bare (src/act.other.c:1744); the harness player is level 1 (≤5 passes the
// experience gate) in a room with no BFR flag, fighting a foe.
func TestLiteralSeamRecallFightingEndToEnd(t *testing.T) {
	for _, compact := range []bool{false, true} {
		w, s := newBareSendItemHarness(t)
		setCompact(s, compact)
		foe := game.NewPlayer(2, "Foe", 1001)
		s.player.SetFightingBody(foe)
		w.ExecRecall(s.player, "")
		assertBareWire(t, s, "Your concentration is broken by your fighting!")
	}
}

// TestLiteralSeamSnoopSeesExactBytes — forwardSnoopText's literal path hands
// the snooping immortal the same bytes C's snooper read ("% …%%" framing,
// comm.c:1646-1651): nothing appended, nothing folded, on either side.
func TestLiteralSeamSnoopSeesExactBytes(t *testing.T) {
	w, s := newBareSendItemHarness(t)
	spy := makeTestSession(t, s.manager, "Snoopwiz", 1001, true)
	s.manager.mu.Lock()
	s.manager.sessions["snoopwiz"] = spy
	s.manager.mu.Unlock()
	s.snoopBy = spy
	spy.snooping = s

	w.ExecFill(s.player, "bucket fountain")

	if got := renderAllOutput(s); got != "You can't find it!" {
		t.Fatalf("target wire = %q, want the bare miss bytes", got)
	}
	if got, want := renderAllOutput(spy), "% You can't find it!%%"; got != want {
		t.Fatalf("snoop wire = %q, want %q", got, want)
	}
}

// TestLiteralSeamSwitchedDescriptorRoutes — a descriptor actively switched
// into another body receives that body's literal bytes on the controlling
// session (switchDescriptorByName), exactly as the ordinary sink routes.
func TestLiteralSeamSwitchedDescriptorRoutes(t *testing.T) {
	w, s := newBareSendItemHarness(t)
	m := s.manager

	original := s.player
	body := game.NewPlayer(2, "Borrowed", 1001)
	if err := w.AddPlayer(body); err != nil {
		t.Fatalf("AddPlayer body: %v", err)
	}
	m.mu.Lock()
	s.player = body
	s.isSwitched = true
	s.switchedOriginal = original
	s.switchedPlayer = body
	m.mu.Unlock()

	// The producer addresses the switched-into body by name; the
	// controlling session renders the bare bytes, and the body's own
	// (descriptorless) identity sees nothing.
	w.ExecFill(body, "bucket fountain")
	if got := renderAllOutput(s); got != "You can't find it!" {
		t.Fatalf("controlling session wire = %q, want the bare miss bytes", got)
	}
}
