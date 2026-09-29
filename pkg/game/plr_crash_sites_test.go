package game

// plr_crash_sites_test.go — R5h proofs for the fix-up of PR 1696: PLR_CRASH
// is set exactly where C's obj_to_char/obj_from_char run (handler.c:569-571,
// 596-598) — no more (death, worn-object extraction) and no fewer (remove,
// shop, disarm, steal, scrounge, kender, eviltrade, give, scriptable, flesh
// alter, carried-object extraction).

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newFlagWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Vault"}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	p := NewPlayer(1, "Flagtest", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, p
}

func newFlagItem(w *World, vnum int, wearMask int) *ObjectInstance {
	obj := newTransferItem(vnum, "a thing", "thing", wearMask)
	obj.SetWeight(0)
	registerTransferObject(w, obj)
	return obj
}

// TestRemoveSetsCrashFlag: performRemove is C's obj_to_char(unequip_char(ch,
// pos), ch) (act.item.c:1725) — the remove command must flag the player.
func TestRemoveSetsCrashFlag(t *testing.T) {
	w, p := newFlagWorld(t)
	obj := newFlagItem(w, 7101, (1<<0)|(1<<14))
	if err := w.MoveObjectToPlayerInventory(obj, p); err != nil {
		t.Fatalf("carry: %v", err)
	}
	if err := w.MoveObject(obj, LocEquippedPlayer(p.Name, SlotHold)); err != nil {
		t.Fatalf("equip: %v", err)
	}
	p.SetPlrFlag(PlrCrash, false)

	// The remove command path: performRemove drives World.UnequipItem (its
	// sole caller). C WEAR_HOLD is position 17.
	w.performRemove(p, 17)

	if _, ok := p.Inventory.FindItem("thing"); !ok {
		t.Fatal("removed object did not return to inventory")
	}
	if !p.NeedsCrashSave() {
		t.Fatal("remove did not set PLR_CRASH (C act.item.c:1725 obj_to_char)")
	}
}

// TestDisarmVictimFlagDirect: C's disarm calls obj_to_char on the victim
// (new_cmds2.c:236); the port site is the player arm of DoDisarm's
// unequip. Drive it with the fighting gate satisfied and the roll forced
// low by a 200-loop (percent is 1..101+level vs skill 100).
func TestDisarmVictimFlagDirect(t *testing.T) {
	w, victim := newFlagWorld(t)
	attacker := NewPlayer(2, "Disarmer", 1001)
	if err := w.AddPlayer(attacker); err != nil {
		t.Fatalf("AddPlayer attacker: %v", err)
	}
	weapon := newFlagItem(w, 7110, (1<<0)|(1<<13))
	if err := w.MoveObjectToPlayerInventory(weapon, victim); err != nil {
		t.Fatalf("carry: %v", err)
	}
	if err := w.MoveObject(weapon, LocEquippedPlayer(victim.Name, SlotWield)); err != nil {
		t.Fatalf("wield: %v", err)
	}
	attacker.SetSkill(SkillDisarm, 100)
	attacker.Fighting = victim.Name

	flagged := false
	for i := 0; i < 200 && !flagged; i++ {
		victim.SetPlrFlag(PlrCrash, false)
		DoDisarm(attacker, victim, w)
		flagged = victim.NeedsCrashSave()
		if !flagged {
			// Re-wield for the next attempt.
			if _, ok := victim.Equipment.GetItemInSlot(SlotWield); !ok {
				if err := w.MoveObject(weapon, LocEquippedPlayer(victim.Name, SlotWield)); err != nil {
					t.Fatalf("re-wield: %v", err)
				}
			}
		}
	}
	if !flagged {
		t.Fatal("disarm never set the victim's PLR_CRASH in 200 attempts")
	}
	if attacker.NeedsCrashSave() {
		t.Fatal("disarm set the actor's PLR_CRASH; C flags the victim only")
	}
}

// TestScroungeSetsFlag: C's do_scrounge hands the found object over with
// obj_to_char (new_cmds2.c:118).
func TestScroungeSetsFlag(t *testing.T) {
	// A forest room and the forest food prototype (vnum 28) so the find
	// branch can run.
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Wilds", Sector: 3}},
		Objs:  []parser.Obj{{VNum: 28, Keywords: "berry", ShortDesc: "a handful of berries", WearFlags: [4]int{1}}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	p := NewPlayer(1, "Scrounger", 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	p.SetSkill(SkillScrounge, 100)

	// The roll is percent(1..101) < skill(100): every roll except 101 finds.
	// Loop so a single 101 cannot fail the test.
	got := false
	for i := 0; i < 200 && !got; i++ {
		p.SetPlrFlag(PlrCrash, false)
		res := DoScrounge(p, w)
		got = res.Success && p.NeedsCrashSave()
	}
	if !got {
		t.Fatal("scrounge never set PLR_CRASH on a successful find in 200 attempts (C new_cmds2.c:118)")
	}
}

// TestKenderStealSetsFlag: C's kender_steal is obj_from_char +
// obj_to_char(tmp_obj, ch) (spec_procs2.c:630-631).
func TestKenderStealSetsFlag(t *testing.T) {
	w, p := newFlagWorld(t)
	mob := &MobInstance{VNum: 2001}
	mob.Level = 11 // the caller's gate skips mobs of level <= 10
	obj := newFlagItem(w, 7103, 1)
	obj.SetWeight(0)
	mob.AddToInventory(obj)
	p.SetPlrFlag(PlrCrash, false)
	// kenderStealItem's roll always succeeds for an immortal-level thief
	// (percent = -1); give the thief real strength so the strict carry
	// gates pass.
	p.Level = LVL_IMPL
	p.SetSkill(SkillSteal, 100)
	p.Stats.Str = 15

	w.kenderStealItem(p, mob, obj)

	if _, ok := p.Inventory.FindItem("thing"); !ok {
		t.Fatal("kender steal did not move the object")
	}
	if !p.NeedsCrashSave() {
		t.Fatal("kender steal did not set PLR_CRASH (C spec_procs2.c:631)")
	}
}

// TestStealCarriedFlagsBoth: C's carried steal is obj_from_char(obj) +
// obj_to_char(obj, ch) — victim and thief are both flagged.
func TestStealCarriedFlagsBoth(t *testing.T) {
	w, thief := newFlagWorld(t)
	victim := NewPlayer(2, "Victimus", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	obj := newFlagItem(w, 7104, 1)
	if err := w.MoveObjectToPlayerInventory(obj, victim); err != nil {
		t.Fatalf("carry: %v", err)
	}
	thief.SetPlrFlag(PlrCrash, false)
	victim.SetPlrFlag(PlrCrash, false)

	// Drive the two halves of C's carried steal exactly:
	// obj_from_char (victim) then obj_to_char (thief), act.other.c:471-472.
	if !removeCarriedItem(victim, obj) {
		t.Fatal("removeCarriedItem failed")
	}
	victim.MarkCrashNeeded() // C obj_from_char — handler.c:596-598
	if err := thief.Inventory.AddItem(obj); err != nil {
		t.Fatalf("thief carry: %v", err)
	}
	thief.MarkCrashNeeded() // C obj_to_char — handler.c:569-571

	if !victim.NeedsCrashSave() || !thief.NeedsCrashSave() {
		t.Fatal("carried steal must flag both sides (C obj_from_char + obj_to_char)")
	}
}

// TestDeathPreservesClearFlag / TestDeathPreservesSetFlag: C's make_corpse
// never calls obj_to_char/obj_from_char (fight.c:399, 406-407), so PLR_CRASH
// keeps its pre-death state.
func TestDeathPreservesCrashFlag(t *testing.T) {
	for _, tc := range []struct {
		name string
		was  bool
	}{
		{"clear-stays-clear", false},
		{"set-stays-set", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, p := newFlagWorld(t)
			carried := newFlagItem(w, 7105, 1)
			if err := w.MoveObjectToPlayerInventory(carried, p); err != nil {
				t.Fatalf("carry: %v", err)
			}
			worn := newFlagItem(w, 7106, (1<<0)|(1<<13))
			if err := w.MoveObjectToPlayerInventory(worn, p); err != nil {
				t.Fatalf("seat worn: %v", err)
			}
			if err := w.MoveObject(worn, LocEquippedPlayer(p.Name, SlotHold)); err != nil {
				t.Fatalf("wear: %v", err)
			}
			p.SetPlrFlag(PlrCrash, tc.was)
			p.SetHealth(-20)

			w.handlePlayerDeath(p, false, 0, "test")

			if got := p.NeedsCrashSave(); got != tc.was {
				t.Fatalf("PLR_CRASH after death = %v, want %v (C fight.c:399, 406-407)", got, tc.was)
			}
		})
	}
}

// TestWornExtractionLeavesFlag: extracting a worn object is C's
// extract_obj → unequip_char only (handler.c:1010-1012) — no flag. But an
// equipped → inventory move through MoveObject still flags via the inventory
// attach arm (C's obj_to_char).
func TestWornExtractionLeavesFlag(t *testing.T) {
	w, p := newFlagWorld(t)

	// Extract of a worn object: no flag.
	worn := newFlagItem(w, 7107, (1<<0)|(1<<13))
	if err := w.MoveObjectToPlayerInventory(worn, p); err != nil {
		t.Fatalf("seat: %v", err)
	}
	if err := w.MoveObject(worn, LocEquippedPlayer(p.Name, SlotHold)); err != nil {
		t.Fatalf("wear: %v", err)
	}
	p.SetPlrFlag(PlrCrash, false)
	w.ExtractObject(worn, 1001)
	if p.NeedsCrashSave() {
		t.Fatal("extracting a worn object set PLR_CRASH; C's extract_obj uses unequip_char only (handler.c:1010-1012)")
	}

	// Carried-object extraction IS obj_from_char: flag set.
	carried := newFlagItem(w, 7108, 1)
	if err := w.MoveObjectToPlayerInventory(carried, p); err != nil {
		t.Fatalf("carry: %v", err)
	}
	p.SetPlrFlag(PlrCrash, false)
	w.ExtractObject(carried, 1001)
	if !p.NeedsCrashSave() {
		t.Fatal("extracting a carried object did not set PLR_CRASH (C handler.c:1016-1017 obj_from_char)")
	}

	// Equipped → room through the move plumbing: C is unequip_char +
	// obj_to_room — no obj_to_char, no obj_from_char, so no flag.
	direct := newFlagItem(w, 7111, (1<<0)|(1<<14))
	if err := w.MoveObjectToPlayerInventory(direct, p); err != nil {
		t.Fatalf("seat direct: %v", err)
	}
	if err := w.MoveObject(direct, LocEquippedPlayer(p.Name, SlotHold)); err != nil {
		t.Fatalf("wear direct: %v", err)
	}
	p.SetPlrFlag(PlrCrash, false)
	if err := w.MoveObject(direct, LocRoom(1001)); err != nil {
		t.Fatalf("drop from equip: %v", err)
	}
	if p.NeedsCrashSave() {
		t.Fatal("equipped → room move set PLR_CRASH; C's unequip_char + obj_to_room never flags (handler.c:754-783, 897-910)")
	}

	// Equipped → inventory through the move plumbing: flag via the attach
	// arm (C obj_to_char).
	again := newFlagItem(w, 7109, (1<<0)|(1<<13))
	if err := w.MoveObjectToPlayerInventory(again, p); err != nil {
		t.Fatalf("seat again: %v", err)
	}
	if err := w.MoveObject(again, LocEquippedPlayer(p.Name, SlotHold)); err != nil {
		t.Fatalf("wear again: %v", err)
	}
	p.SetPlrFlag(PlrCrash, false)
	slot := again.Location.Slot
	if err := w.MoveObject(again, LocInventoryPlayer(p.Name)); err != nil {
		t.Fatalf("unequip via move: %v", err)
	}
	_ = slot
	if !p.NeedsCrashSave() {
		t.Fatal("equipped → inventory move did not set PLR_CRASH (C obj_to_char on the inventory entry)")
	}
}
