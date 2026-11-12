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

func TestCombatEngineRoundWoundActHidesInvisibleVictim(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Lit Wound Room", Light: 1}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	attacker := NewPlayer(1, "Attacker", 1001)
	victim := NewPlayer(2, "Victim", 1001)
	watcher := NewPlayer(3, "Watcher", 1001)
	for _, player := range []*Player{attacker, victim, watcher} {
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer %s: %v", player.GetName(), err)
		}
	}

	// A natural 20 lands one bare-handed damage point: the victim enters the
	// stunned band at 0 HP. The lit room isolates invisibility as the reason
	// the ordinary peer cannot see the wound act.
	attacker.SetLevel(11)
	attacker.SetClass(ClassMageUser) // avoids the warrior extra-attack draw
	attacker.Stats.Str = 0
	attacker.CopyBaseAttributes()
	attacker.SetPosition(PosStanding)
	victim.SetLevel(11) // above the player-vs-player protection gate
	victim.SetHP(1)
	victim.SetAffect(affInvisible, true)
	if canSee(watcher, victim) {
		t.Fatal("precondition: ordinary watcher can see the invisible victim")
	}

	ce := combat.NewCombatEngine()
	oldCallbacks := combat.GetCallbacks()
	t.Cleanup(func() {
		ce.Stop()
		combat.SetCallbacks(oldCallbacks)
	})
	ce.SetCallbacks(w.WireCombatCallbacks())
	w.SetCombatEngine(ce)
	ce.MessageFunc = func(combat.Combatant, combat.Combatant, int, int) bool { return true }
	// Keep the engine's legacy fallback observable so restoring the old adapter
	// makes this test fail with the leaked room line.
	ce.SetBroadcastFunc(func(room int, message string, exclude []combat.Combatant) {
		for _, recipient := range w.GetPlayersInRoom(room) {
			skip := false
			for _, excluded := range exclude {
				if recipient == excluded {
					skip = true
					break
				}
			}
			if skip || recipient.GetPosition() <= combat.PosSleeping {
				continue
			}
			out[recipient.GetName()] += message
		}
	})
	if err := ce.StartCombat(attacker, victim); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}
	// The victim is not enrolled for a retaliation turn; the attacker remains the
	// active fighter for this single violence round.
	victim.SetFightingBody(nil)
	roller := combat.NewScriptedRoller([]int{1, 10000, 20, 0, 99, 99, 99, 99, 99})
	combat.WithRoller(roller, ce.PerformRound)

	if got := victim.GetPosition(); got != combat.PosStunned {
		target := attacker.GetFightingBody()
		targetName := "<nil>"
		if target != nil {
			targetName = target.GetName()
		}
		t.Fatalf("victim position = %d, want PosStunned (hp=%d, attacker position=%d, attacker target=%s, draws=%d)", got, victim.GetHP(), attacker.GetPosition(), targetName, roller.Index)
	}
	if got := victim.GetHP(); got != 0 {
		t.Fatalf("victim HP = %d, want 0", got)
	}
	if got := out[watcher.GetName()]; got != "" {
		t.Fatalf("ordinary peer received output for an invisible victim's wound: %q", got)
	}
}
