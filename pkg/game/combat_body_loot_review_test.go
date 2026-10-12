package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCombatBodyLootNameCollision(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}, {VNum: 3001}}, Mobs: []parser.Mob{{VNum: 300, ShortDesc: "Collision", Keywords: "collision", Position: 8, DefaultPos: 8}}, Objs: []parser.Obj{{VNum: 200, Keywords: "bag", ShortDesc: "a bag", TypeFlag: ITEM_CONTAINER, WearFlags: [4]int{1}, Cost: 200}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	p := NewPlayer(1, "Collision", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	m, err := w.SpawnMobQuiet(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	m.SetMobFlag(MobFlagLoots)
	o, err := w.SpawnObject(200, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(o, p); err != nil {
		t.Fatal(err)
	}
	p.SetHP(-11)
	w.HandleDeath(p, m, combat.TYPE_HIT)
	if len(m.Inventory) != 1 || m.Inventory[0] != o {
		t.Fatalf("distinct same-name killer did not loot: %v", m.Inventory)
	}
}

// TestCombatBodyDeathCreditNameCollision proves the death-credit side of
// fight.c:1672-1690 for a mob killer that shares a name with an unrelated PC:
// the credit must not land on that PC. It is a credit assertion only — the
// fight.c:1672-1681 syslog producer (BRF, LVL_IMMORT, file TRUE, room name in
// the payload) is proven by pkg/game/pk_death_mudlog_test.go.
func TestCombatBodyDeathCreditNameCollision(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}, {VNum: 3001}}, Mobs: []parser.Mob{{VNum: 300, ShortDesc: "Collision", Keywords: "collision", Position: 8}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	observer := NewPlayer(1, "Collision", 1001)
	victim := NewPlayer(2, "Victim", 1001)
	for _, p := range []*Player{observer, victim} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
	}
	m, err := w.SpawnMobQuiet(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	victim.SetHP(-11)
	w.HandleDeath(victim, m, combat.TYPE_HIT)
	if observer.Kills != 0 || observer.PKs != 0 || observer.GetFlags()&(1<<uint(PlrOutlaw)) != 0 {
		t.Fatalf("NPC killer credited unrelated PC: kills=%d PKs=%d flags=%d", observer.Kills, observer.PKs, observer.GetFlags())
	}
}
