package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func addGaleruAliveMob(t *testing.T, w *World, room int) *MobInstance {
	t.Helper()
	proto := &parser.Mob{
		VNum:       1315,
		Keywords:   "galeru master elements rainbow snake",
		ShortDesc:  "Galeru, the Master of the Elements",
		LongDesc:   "Galeru coils here.",
		Level:      37,
		Position:   combat.PosStanding,
		DefaultPos: combat.PosStanding,
		HP:         parser.DiceRoll{Num: 1, Sides: 8, Plus: 20},
	}
	w.mu.Lock()
	w.mobs[proto.VNum] = proto
	w.mu.Unlock()
	mob, err := w.spawnMobQuiet(proto.VNum, room)
	if err != nil {
		t.Fatalf("spawn Galeru: %v", err)
	}
	return mob
}

func TestSpecElementsGaleruAlive_EntryAndExactMobGate(t *testing.T) {
	w, actor, _, _, messages := newGaleruColumnTestWorld(t)

	if specElementsGaleruAlive(w, actor, nil, "", "") {
		t.Fatal("commandless room-special call should fall through")
	}
	if actor.GetRoomVNum() != 1372 {
		t.Fatalf("commandless call moved actor to %d", actor.GetRoomVNum())
	}
	for name, msg := range messages {
		if msg != "" {
			t.Fatalf("commandless call emitted output for %s: %q", name, msg)
		}
	}

	galeru := addGaleruAliveMob(t, w, 1372)
	if specElementsGaleruAlive(w, actor, nil, "say", "hello") {
		t.Fatal("live Galeru should make the special return false")
	}
	if actor.GetRoomVNum() != 1372 || galeru.GetRoom() != 1372 {
		t.Fatalf("live Galeru gate changed rooms: actor=%d Galeru=%d", actor.GetRoomVNum(), galeru.GetRoom())
	}
}

func TestSpecElementsGaleruAlive_UsesExactVNumAndMovesNPCs(t *testing.T) {
	w, actor, peer, npc, _ := newGaleruColumnTestWorld(t)
	npc.Prototype.Keywords = "galeru decoy"

	if !specElementsGaleruAlive(w, actor, nil, "say", "hello") {
		t.Fatal("non-1315 Galeru keyword should not block the dead branch")
	}
	for _, player := range []*Player{actor, peer} {
		if got := player.GetRoomVNum(); got != 1395 {
			t.Errorf("%s room = %d, want 1395", player.GetName(), got)
		}
	}
	if got := npc.GetRoom(); got != 1395 {
		t.Errorf("NPC room = %d, want 1395", got)
	}
}
