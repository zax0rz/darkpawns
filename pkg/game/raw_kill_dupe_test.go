package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// countRefs counts every reference to obj reachable after a death: the room's
// top-level items, each corpse's contents (recursively), and whatever is still
// attached to the victim's equipment or inventory.
func countRefs(w *World, victim *Player, obj *ObjectInstance) int {
	var walk func(items []*ObjectInstance) int
	walk = func(items []*ObjectInstance) int {
		n := 0
		for _, item := range items {
			if item == nil {
				continue
			}
			if item == obj {
				n++
			}
			n += walk(item.Contains)
		}
		return n
	}
	n := walk(w.GetItemsInRoom(1001))
	if victim.Equipment != nil {
		for _, item := range victim.Equipment.GetEquippedItems() {
			if item == obj {
				n++
			}
			n += walk(item.Contains)
		}
	}
	if victim.Inventory != nil {
		n += walk(victim.Inventory.FindItems(""))
	}
	return n
}

// TestScroungeHandoverKeepsObjectUnique pins the hand-over discipline the
// counting test depends on: do_scrounge is C's read_object + obj_to_char
// (src/new_cmds2.c:115-125) — the found food joins ch->carrying and is in no
// room list. With the stale room location NewObjectFromProto defaults to, the
// corpse move cannot detach the object from the inventory and the extraction
// pass drops a second copy (src/fight.c:399-407 vs handler.c:1134-1142).
func TestScroungeHandoverKeepsObjectUnique(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Forest", Zone: 1, Sector: SECT_FOREST}},
		Objs:  []parser.Obj{{VNum: 28, Keywords: "game", ShortDesc: "some small game"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	old := combat.GetCallbacks()
	combat.SetCallbacks(w.WireCombatCallbacks())
	t.Cleanup(func() { combat.SetCallbacks(old) })

	victim := NewPlayer(41, "Scrounger", 1001)
	victim.SetPosition(combat.PosStanding)
	victim.SetSkill(SkillScrounge, 100)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	dprng.ResetStream(1)
	if result := DoScrounge(victim, w); !result.Success {
		t.Fatalf("scrounge failed: %+v", result)
	}
	food := victim.Inventory.FindItems("")
	if len(food) != 1 {
		t.Fatalf("carried items after scrounge = %d, want 1", len(food))
	}
	if food[0].Location.Kind != ObjInInventory {
		t.Fatalf("scrounged food location = %+v, want inventory (obj_to_char, src/new_cmds2.c:120)", food[0].Location)
	}

	combat.RawKill(victim, 0)
	w.ExtractPendingChars()
	if got := countRefs(w, victim, food[0]); got != 1 {
		t.Fatalf("scrounged food exists %d times after the raw kill, want exactly 1", got)
	}
}
