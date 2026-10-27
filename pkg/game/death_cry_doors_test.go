package game

// death_cry_doors_test.go — C's death_cry repoints the dying character into
// adjacent rooms through CAN_GO (src/fight.c:520-527), whose third clause
// skips closed doors (src/utils.h:565-567): a shut door muffles the cry.
// Both Go cry sites — the extraction-path deathCry and the combat engine's
// DeathCry (through the world's GetAdjacentRoom callback) — must skip closed
// exits the same way.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func deathCryDoorsWorld(t *testing.T) (*World, map[string]*int) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1001, Name: "Gatehouse", Zone: 1,
				Exits: map[string]parser.Exit{
					"north": {ToRoom: 1002},
					"west":  {ToRoom: 1003, Keywords: "iron gate", ExitInfo: parser.ExitClosed},
				},
			},
			{VNum: 1002, Name: "Open Side", Zone: 1},
			{VNum: 1003, Name: "Behind the Gate", Zone: 1},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	openSide := NewPlayer(2, "Openside", 1002)
	behindGate := NewPlayer(3, "Behindgate", 1003)
	if err := w.AddPlayer(openSide); err != nil {
		t.Fatal(err)
	}
	if err := w.AddPlayer(behindGate); err != nil {
		t.Fatal(err)
	}

	heard := map[string]*int{"Openside": new(int), "Behindgate": new(int)}
	w.MessageSink = func(name string, msg []byte) {
		if strings.Contains(string(msg), "someone's death cry") {
			if counter, ok := heard[name]; ok {
				*counter++
			}
		}
	}
	return w, heard
}

// TestDeathCryMuffledBehindClosedDoors drives the extraction-path deathCry:
// the open side hears the cry, the room behind the closed gate does not.
func TestDeathCryMuffledBehindClosedDoors(t *testing.T) {
	w, heard := deathCryDoorsWorld(t)
	doomed := NewPlayer(1, "Doomed", 1001)
	if err := w.AddPlayer(doomed); err != nil {
		t.Fatal(err)
	}
	w.deathCry(doomed)

	if *heard["Openside"] == 0 {
		t.Fatal("the open adjacent room did not hear the death cry")
	}
	if *heard["Behindgate"] != 0 {
		t.Fatal("the cry crossed the closed gate (CAN_GO skips closed exits, fight.c:520-527)")
	}
}

// TestCombatDeathCryMuffledBehindClosedDoors drives the combat engine's
// DeathCry with the wired adjacency callback and a recording broadcast, so
// the assertion is the exact room list C's CAN_GO walk would produce: the
// death room and the open side, never the room behind the closed gate.
func TestCombatDeathCryMuffledBehindClosedDoors(t *testing.T) {
	w, _ := deathCryDoorsWorld(t)
	doomed := NewPlayer(1, "Doomed", 1001)
	if err := w.AddPlayer(doomed); err != nil {
		t.Fatal(err)
	}
	wired := w.WireCombatCallbacks()
	rooms := map[int]bool{}
	old := combat.GetCallbacks()
	combat.SetCallbacks(&combat.GameCallbacks{
		Broadcast:       func(room int, _ string, _ []combat.Combatant) { rooms[room] = true },
		GetAdjacentRoom: wired.GetAdjacentRoom,
	})
	t.Cleanup(func() { combat.SetCallbacks(old) })

	combat.DeathCry(doomed)

	if !rooms[1001] || !rooms[1002] {
		t.Fatalf("cry room list %v lacks the death room or the open side", rooms)
	}
	if rooms[1003] {
		t.Fatal("the combat cry crossed the closed gate (CAN_GO skips closed exits, fight.c:520-527)")
	}
}
