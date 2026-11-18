package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func TestMagAffectsDoesNotActToDeadChillTouchVictim(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Test Room"}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	var output strings.Builder
	w.MessageSink = func(_ string, message []byte) { output.Write(message) }
	victim := NewPlayer(1, "Victim", 1001)
	victim.SetPosition(combat.PosDead)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}

	spells.MagAffects(20, victim, victim, spells.SpellChillTouch, int(spells.SaveSpell), w)

	if got := output.String(); strings.Contains(got, "You feel your strength wither!") {
		t.Fatalf("dead victim received act() line: %q", got)
	}
	if !victim.HasSpellAffect(spells.SpellChillTouch) {
		t.Fatal("C still applies the affect record before act() suppresses the dead victim line")
	}
}
