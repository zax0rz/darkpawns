package session

// bare_send_test.go — PF-063 cluster 4 proofs: C sends these gate/failure
// replies with NO terminator (the prompt frame supplies the only line
// break), so they ride the literal path; the text event's line-ending append
// printed a blank line C never sends. Each test drives the real handler and
// the following prompt pass, then asserts the exact rendered bytes —
// noncompact: message + "\r\n" frame + prompt; compact: message + prompt
// (PRF_COMPACT suppresses the frame, comm.c:1631). Reverting any
// sendLiteralText swap back to Send fails the noncompact expectation.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// renderAllOutput drains the send channel and renders every frame the way
// the telnet writer does, concatenating text and prompt frame text.
func renderAllOutput(s *Session) string {
	var b strings.Builder
	for len(s.send) > 0 {
		f, ok := RenderTerminalFrame(<-s.send)
		if !ok {
			continue
		}
		switch f.Kind {
		case FrameText, FramePrompt:
			b.WriteString(f.Text)
		}
	}
	return b.String()
}

// setCompact toggles the PRF_COMPACT preference bit on the session's player.
func setCompact(s *Session, compact bool) {
	if compact {
		s.player.Flags |= 1 << uint(game.PrfCompact)
	} else {
		s.player.Flags &^= 1 << uint(game.PrfCompact)
	}
}

// assertBareWire runs one prompt pass and asserts the exact rendered bytes:
// the bare C message, then the prompt — with the "\r\n" frame between them
// only when the player is not compact.
func assertBareWire(t *testing.T, s *Session, message string) {
	t.Helper()
	s.SendPrompt()
	want := message + "\r\n> "
	if s.player.Flags&(1<<uint(game.PrfCompact)) != 0 {
		want = message + "> "
	}
	if got := renderAllOutput(s); got != want {
		t.Fatalf("rendered wire = %q, want %q", got, want)
	}
}

// newBareSendItemHarness builds a world whose only object has TypeFlag 0 —
// neither ITEM_SCROLL nor ITEM_POTION — and a session carrying it, so both
// wrong-type gates fire. Modeled on newReciteHarness.
func newBareSendItemHarness(t *testing.T) (*game.World, *Session) {
	t.Helper()
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Study"}},
		Objs: []parser.Obj{{
			VNum: 7301, Keywords: "trinket", ShortDesc: "a trinket",
			TypeFlag: 0, WearFlags: [4]int{(1 << 0) | (1 << 14)},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	m := newTestManager(t, w, nil)
	s := makeTestSession(t, m, "Quaffer", 1001, true)
	if err := w.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	// The player-level seam resolves sessions through the manager, so the
	// harness session must be registered like a live connection.
	m.mu.Lock()
	m.sessions["quaffer"] = s
	m.mu.Unlock()
	obj, err := w.SpawnObject(7301, -1)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	if err := w.MoveObjectToPlayerInventory(obj, s.player); err != nil {
		t.Fatalf("carry: %v", err)
	}
	return w, s
}

// TestBareWiznetGateRendersWithoutPhantomBlank — do_wiznet's mortal gate
// sends bare "Huh?!?" (src/act.wizard.c:1924). Covers the
// command-surface-punctuation and mortal-batch13 movers.
func TestBareWiznetGateRendersWithoutPhantomBlank(t *testing.T) {
	for _, compact := range []bool{false, true} {
		s, _ := newStringTestSession(t, 1)
		setCompact(s, compact)
		if err := cmdWiznetText(s, "hello"); err != nil {
			t.Fatalf("cmdWiznetText: %v", err)
		}
		assertBareWire(t, s, "Huh?!?")
	}
}

// TestBareReciteRefusalRendersWithoutPhantomBlank — do_use's SCMD_RECITE
// type gate sends bare (src/act.other.c:959). Covers the recite-wrong-type
// mover.
func TestBareReciteRefusalRendersWithoutPhantomBlank(t *testing.T) {
	for _, compact := range []bool{false, true} {
		_, s := newBareSendItemHarness(t)
		setCompact(s, compact)
		if err := cmdRecite(s, []string{"trinket"}); err != nil {
			t.Fatalf("cmdRecite: %v", err)
		}
		assertBareWire(t, s, "You can only recite scrolls.")
	}
}

// TestBareQuaffRefusalRendersWithoutPhantomBlank — do_use's SCMD_QUAFF type
// gate sends bare (src/act.other.c:953); traced sibling of the recite gate.
func TestBareQuaffRefusalRendersWithoutPhantomBlank(t *testing.T) {
	for _, compact := range []bool{false, true} {
		_, s := newBareSendItemHarness(t)
		setCompact(s, compact)
		if err := cmdQuaff(s, []string{"trinket"}); err != nil {
			t.Fatalf("cmdQuaff: %v", err)
		}
		assertBareWire(t, s, "You can only quaff potions.")
	}
}

// TestBareStringUsageRendersWithoutPhantomBlank — do_string's usage line is
// sent bare (src/modify.c:608).
func TestBareStringUsageRendersWithoutPhantomBlank(t *testing.T) {
	for _, compact := range []bool{false, true} {
		s, _ := newStringTestSession(t, LVL_IMPL)
		setCompact(s, compact)
		runString(t, s, "banana shield name x")
		assertBareWire(t, s, stringUsage)
	}
}

// TestBareStringRefuseRendersWithoutPhantomBlank — do_string's field-1
// player gate sends bare (src/modify.c:628). Covers the string-depth mover.
func TestBareStringRefuseRendersWithoutPhantomBlank(t *testing.T) {
	for _, compact := range []bool{false, true} {
		s, m := newStringTestSession(t, LVL_IMPL-2)
		setCompact(s, compact)
		victim := game.NewPlayer(2, "Renamevictim", 1001)
		if err := m.world.AddPlayer(victim); err != nil {
			t.Fatalf("AddPlayer victim: %v", err)
		}
		runString(t, s, "mob Renamevictim name bob")
		assertBareWire(t, s, stringPlayerFieldRefused)
	}
}
