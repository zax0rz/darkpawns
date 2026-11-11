package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1296 witness: the wounded-band room lines go through Act with
// hide_invisible TRUE (src/fight.c:1566-1584), so an invisible victim's
// wound is unseen by observers who cannot see it — the old plain broadcaster
// sent the real name to everyone in the room, including in the dark.

func woundActTestWorld(t *testing.T, observe map[string]string) (*World, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Wound Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(name string, msg []byte) { observe[name] += string(msg) }

	victim := NewPlayer(1, "Victim", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	victim.SetLevel(30)
	victim.SetPosition(PosFighting)
	victim.SetFightingBody(NewPlayer(9, "Attacker", 1001))
	if err := w.AddPlayer(NewPlayer(2, "Watcher", 1001)); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, victim
}

func TestWoundActHidesInvisibleVictimFromRoom(t *testing.T) {
	out := map[string]string{}
	w, victim := woundActTestWorld(t, out)
	victim.SetAffect(affInvisible, true)

	victim.SetHP(-1)
	newPos := combat.UpdatePositionAfterDamage(victim, w.WoundAct, nil)

	if newPos != combat.PosStunned {
		t.Fatalf("newPos = %d, want PosStunned", newPos)
	}
	if strings.Contains(out["Watcher"], "stunned") || strings.Contains(out["Watcher"], "Victim is") || strings.Contains(out["Watcher"], "Someone is stunned") {
		t.Fatalf("watcher saw invisible victim's wound: %q", out["Watcher"])
	}
	if !strings.Contains(out["Victim"], "You're stunned") {
		t.Fatalf("victim private wound line missing: %q", out["Victim"])
	}
}

// TestWoundActVisibleVictimReachesRoom is the control: a seen victim's wound
// is broadcast with its real name through the same Act path.
func TestWoundActVisibleVictimReachesRoom(t *testing.T) {
	out := map[string]string{}
	w, victim := woundActTestWorld(t, out)

	victim.SetHP(-1)
	newPos := combat.UpdatePositionAfterDamage(victim, w.WoundAct, nil)
	if newPos != combat.PosStunned {
		t.Fatalf("newPos = %d, want PosStunned", newPos)
	}
	if got := out["Watcher"]; !strings.Contains(got, "Victim is stunned") {
		t.Fatalf("watcher room bytes = %q, want the wounded line with the victim's name", got)
	}
}
