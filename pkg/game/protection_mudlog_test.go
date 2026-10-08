package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/spells"
)

func protectionDiagnosticProof(t *testing.T, spell, align int, protection string) {
	t.Helper()
	w, ch := rawKillWorld(t)
	ch.SetAlignment(0)
	victim := NewPlayer(2, "Aligned", 1001)
	victim.SetAlignment(align)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatal(err)
	}
	var events []string
	w.MessageSink = func(_ string, msg []byte) { events = append(events, string(msg)) }
	s, file := diagnosticCapture(t, MudlogBrief)
	s.probe = func() {
		if ch.HasPLRFlag(PlrExtract) || len(w.GetItemsInRoom(1001)) != 0 {
			t.Fatal("protection diagnostic after raw kill")
		}
		if len(events) != 1 {
			t.Fatalf("caster message must precede diagnostic: %q", events)
		}
		if !w.mu.TryLock() {
			t.Fatal("world lock held at protection diagnostic")
		}
		w.mu.Unlock()
		if !ch.mu.TryLock() {
			t.Fatal("caster lock held at protection diagnostic")
		}
		ch.mu.Unlock()
	}
	spells.MagAffects(10, ch, victim, spell, 0, w)
	requireDiagnostic(t, s, file, "Victim killed by Protection from "+protection+".", true)
	if !ch.HasPLRFlag(PlrExtract) || victim.HasPLRFlag(PlrExtract) {
		t.Fatal("C kills caster, not aligned victim")
	}
	// Refusal checks the victim's alignment, while log naming uses the caster.
	if strings.Contains(file.String(), "Aligned killed") {
		t.Fatal("protection diagnostic named victim")
	}
	s.messages = nil
	s.probe = nil
	file.Reset()
	neutral := NewPlayer(3, "Neutral", 1001)
	safe := NewPlayer(4, "Safe", 1001)
	spells.MagAffects(10, safe, neutral, spell, 0, w)
	if len(s.messages) != 0 || file.Len() != 0 {
		t.Fatal("safe protection emits diagnostic")
	}
}

func TestGameDiagnosticProtectionEvil(t *testing.T) {
	protectionDiagnosticProof(t, spells.SpellProtFromEvil, -1000, "Evil")
}
