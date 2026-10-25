package game

// The combat-wire raw-kill path (combat.RawKill → cbMakeCorpse →
// makeRawKillBody) must leave each worn and carried object existing exactly
// once: C's make_corpse moves the whole carrying list into the corpse and
// obj_to_obj(unequip_char(ch, i))s every worn item after it
// (src/fight.c:399-407), clearing both lists, so the later extract_char_final
// drops nothing (src/handler.c:1134-1142).

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func dupeKillWorld(t *testing.T) (*World, *Player, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Arena", Zone: 1}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	old := combat.GetCallbacks()
	combat.SetCallbacks(w.WireCombatCallbacks())
	t.Cleanup(func() { combat.SetCallbacks(old) })

	victim := NewPlayer(31, "DupeVictim", 1001)
	victim.SetLevel(10)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	killer := NewPlayer(32, "DupeKiller", 1001)
	killer.SetLevel(15)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer killer: %v", err)
	}
	return w, victim, killer
}

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

// TestRawKillCorpseHoldsEachItemExactlyOnce kills an equipped victim through
// the combat wire and requires every worn and carried object to exist exactly
// once afterwards (src/fight.c:399-407; make_corpse clears both lists).
func TestRawKillCorpseHoldsEachItemExactlyOnce(t *testing.T) {
	w, victim, _ := dupeKillWorld(t)

	lantern := NewObjectInstance(&parser.Obj{
		VNum: 9201, ShortDesc: "a brass lantern", Keywords: "lantern brass-dupe-lantern",
		TypeFlag: itemLightTypeFlag, Values: [4]int{0, 0, 7, 0},
		WearFlags: [4]int{1<<0 | 1<<14}, // TAKE | HOLD
	}, -1)
	sword := NewObjectInstance(&parser.Obj{
		VNum: 9202, ShortDesc: "a dupe-marker sword", Keywords: "sword dupe-marker-sword",
		WearFlags: [4]int{1<<0 | 1<<16}, // TAKE | WIELD
	}, -1)
	shield := NewObjectInstance(&parser.Obj{
		VNum: 9203, ShortDesc: "a carried buckler", Keywords: "buckler dupe-carried-buckler",
	}, -1)

	// Inventory.AddItem links the object but does not set Location — every
	// production obj_to_char caller sets it (e.g. skill_stealth's
	// restoreCarriedItem, and attachObjectLocked drives it from Location), and
	// the world's detach arms dispatch on it. Set it like production does.
	if err := victim.Inventory.AddItem(shield); err != nil {
		t.Fatalf("AddItem shield: %v", err)
	}
	shield.Location = LocInventoryPlayer(victim.Name)
	if err := w.EquipItem(victim, lantern, eqWearHold); err != nil {
		t.Fatalf("EquipItem lantern: %v", err)
	}
	if err := w.EquipItem(victim, sword, eqWearWield); err != nil {
		t.Fatalf("EquipItem sword: %v", err)
	}

	combat.RawKill(victim, 0)
	w.ExtractPendingChars()

	corpses := 0
	for _, item := range w.GetItemsInRoom(1001) {
		if item != nil && item.IsCorpse {
			corpses++
		}
	}
	if corpses != 1 {
		t.Fatalf("corpse count after raw kill = %d, want 1", corpses)
	}

	for _, obj := range []*ObjectInstance{lantern, sword, shield} {
		if got := countRefs(w, victim, obj); got != 1 {
			t.Fatalf("%s exists %d times after the raw kill, want exactly 1", obj.GetShortDesc(), got)
		}
		if obj.Location.Kind != ObjInContainer {
			t.Fatalf("%s ended at %+v, want inside the corpse (fight.c:399-407 puts worn and carried items in the corpse)", obj.GetShortDesc(), obj.Location)
		}
	}
}

// TestScroungeHandoverKeepsObjectUnique pins the hand-over discipline the
// counting test depends on: do_scrounge is C's read_object + obj_to_char
// (src/new_cmds2.c:110-119) — the found food joins ch->carrying and is in no
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
		t.Fatalf("scrounged food location = %+v, want inventory (obj_to_char, new_cmds2.c:119)", food[0].Location)
	}

	combat.RawKill(victim, 0)
	w.ExtractPendingChars()
	if got := countRefs(w, victim, food[0]); got != 1 {
		t.Fatalf("scrounged food exists %d times after the raw kill, want exactly 1", got)
	}
}
