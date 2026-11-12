package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1446 witness: tattooNoMagicRoom is C's plain act(..., TO_ROOM) with
// hide FALSE (spell_parser.c:424-432): observers who cannot see the caster
// hear "Someone's …", and the caster is excluded. The hand-rolled
// name-paste got the masking right but reimplemented the audience and the
// capitalisation; this pins the act() bytes instead.
func TestTattooNoMagicRoomMirrorsAct(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Nomagic Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }

	caster := NewPlayer(1, "Caster", 1001)
	if err := w.AddPlayer(caster); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	caster.SetLevel(30)
	if err := w.AddPlayer(NewPlayer(2, "Watcher", 1001)); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	caster.SetAffect(affInvisible, true)

	w.tattooNoMagicRoom(caster, "$n's magic fizzles out and dies.")

	got := out["Watcher"]
	if strings.Contains(got, "Caster") {
		t.Fatalf("watcher saw the invisible caster's name: %q", got)
	}
	if !strings.Contains(got, "Someone's magic fizzles out and dies.") {
		t.Fatalf("watcher nomagic line = %q, want PERS 'Someone' form", got)
	}
	if out["Caster"] != "" {
		t.Fatalf("caster should be excluded from their own act: %q", out["Caster"])
	}
}
