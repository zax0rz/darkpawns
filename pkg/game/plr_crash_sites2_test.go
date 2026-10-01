package game

// plr_crash_sites2_test.go — R5h proofs for the fix-up-2 sites: each test
// drives the production function that holds the flag line and asserts the
// flag without ever setting it itself. Random entry points (steal) are
// controlled through dprng.ResetStream, the seam the existing steal depth
// tests use.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newFlagWorldWithMob is newFlagWorld plus a mob prototype (vnum 2001).
func newFlagWorldWithMob(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Vault"}},
		Mobs:  []parser.Mob{{VNum: 2001, Keywords: "mob twothousandone", ShortDesc: "the mob"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	return w
}

// stealDriveResult captures what a driven steal left behind.
type stealDriveResult struct {
	success      bool
	thiefFlagged bool
	victimFlag   bool
}

// driveCarriedSteal runs stealInventoryItem through DoSteal with the stream
// seeded so the selection roll succeeds, retrying seeds until the carried
// arm completes. The percent argument is fully determined by the level
// gates (an impl thief against a mortal victim gets percent = -1), so the
// steal itself always succeeds once the item is selected.
func driveCarriedSteal(t *testing.T, w *World, thief, victim *Player, itemName string) stealDriveResult {
	t.Helper()
	thief.Level = LVL_IMPL // percent = -1: the roll always succeeds
	thief.SetSkill(SkillSteal, 100)
	thief.SetPlrFlag(PlrOutlaw, true) // the command path reserves player-stealing for outlaws
	thief.Stats.Str = 15              // real strength so the strict carry gates pass
	thief.CopyBaseAttributes()
	for seed := uint32(1); seed < 5000; seed++ {
		dprng.ResetStream(seed)
		thief.SetPlrFlag(PlrCrash, false)
		victim.SetPlrFlag(PlrCrash, false)
		res := DoSteal(thief, victim, itemName, w)
		if res.Success {
			return stealDriveResult{success: true, thiefFlagged: thief.NeedsCrashSave(), victimFlag: victim.NeedsCrashSave()}
		}
	}
	t.Fatal("no seed made the carried steal succeed")
	return stealDriveResult{}
}

// TestStealCarriedFlagsVictim drives stealInventoryItem via DoSteal and
// asserts the victim's flag alone (C obj_from_char, src/act.other.c:471).
func TestStealCarriedFlagsVictim(t *testing.T) {
	w, thief := newFlagWorld(t)
	victim := NewPlayer(2, "Victimus", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	obj := newFlagItem(w, 7104, 1)
	if err := w.MoveObjectToPlayerInventory(obj, victim); err != nil {
		t.Fatalf("carry: %v", err)
	}

	got := driveCarriedSteal(t, w, thief, victim, "thing")

	if !got.success {
		t.Fatal("steal did not succeed")
	}
	if !got.victimFlag {
		t.Fatal("carried steal did not flag the victim (C obj_from_char, act.other.c:471)")
	}
}

// TestStealCarriedFlagsThief drives the same path and asserts the thief's
// flag alone (C obj_to_char, src/act.other.c:472).
func TestStealCarriedFlagsThief(t *testing.T) {
	w, thief := newFlagWorld(t)
	victim := NewPlayer(2, "Victimus", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	obj := newFlagItem(w, 7105, 1)
	if err := w.MoveObjectToPlayerInventory(obj, victim); err != nil {
		t.Fatalf("carry: %v", err)
	}

	got := driveCarriedSteal(t, w, thief, victim, "thing")

	if !got.success {
		t.Fatal("steal did not succeed")
	}
	if !got.thiefFlagged {
		t.Fatal("carried steal did not flag the thief (C obj_to_char, act.other.c:472)")
	}
}

// TestStealFullInventoryFlagsNobody: C checks the thief's capacity BEFORE
// obj_from_char (src/act.other.c:468-470); the port checks it after the
// removal, so the failure path flags nobody.
func TestStealFullInventoryFlagsNobody(t *testing.T) {
	w, thief := newFlagWorld(t)
	victim := NewPlayer(2, "Victimus", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	obj := newFlagItem(w, 7106, 1)
	if err := w.MoveObjectToPlayerInventory(obj, victim); err != nil {
		t.Fatalf("carry: %v", err)
	}
	// Fill the thief's inventory to capacity so the post-removal AddItem
	// fails and the port takes its capacity-refusal path. Capacity floors at
	// 5 + dex/2 + level/2; with dex 0 that is 5 slots.
	thief.Inventory.SetCapacity(15, 0, 0, 1)
	for i := 0; i < 5; i++ {
		filler := newFlagItem(w, 7199+i, 1)
		if err := thief.Inventory.AddItem(filler); err != nil {
			t.Fatalf("fill %d: %v", i, err)
		}
	}
	if got := thief.Inventory.GetItemCount(); got < thief.Inventory.Capacity {
		t.Fatalf("inventory not full: %d/%d", got, thief.Inventory.Capacity)
	}
	thief.Level = LVL_IMPL
	thief.SetSkill(SkillSteal, 100)
	thief.SetPlrFlag(PlrOutlaw, true)
	thief.Stats.Str = 15
	thief.CopyBaseAttributes()
	thief.Inventory.SetCapacity(15, 0, 0, 1) // preserve this capacity-refusal fixture
	thief.SetPlrFlag(PlrCrash, false)
	victim.SetPlrFlag(PlrCrash, false)

	res := DoSteal(thief, victim, "thing", w)
	if res.Success {
		t.Fatal("steal succeeded against a full inventory")
	}
	if thief.NeedsCrashSave() {
		t.Fatal("failed steal flagged the thief; C's capacity refusal flags nobody (act.other.c:468-470)")
	}
	if victim.NeedsCrashSave() {
		t.Fatal("failed steal flagged the victim; C's capacity refusal flags nobody (act.other.c:468-470)")
	}
	// The victim keeps the object (the port rolls the removal back).
	if _, ok := victim.Inventory.FindItem("thing"); !ok {
		t.Fatal("victim lost the object on a failed steal")
	}
}

// TestStealEquippedFlagsThiefOnly drives stealEquippedItem via DoSteal: C's
// equipped steal is obj_to_char(unequip_char(vict, eq_pos), ch)
// (src/act.other.c:429) — the thief only.
func TestStealEquippedFlagsThiefOnly(t *testing.T) {
	w, thief := newFlagWorld(t)
	victim := NewPlayer(2, "Victimus", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	ring := newFlagItem(w, 7107, (1<<0)|(1<<1)) // TAKE + FINGER
	if err := w.MoveObjectToPlayerInventory(ring, victim); err != nil {
		t.Fatalf("carry: %v", err)
	}
	if err := w.MoveObject(ring, LocEquippedPlayer(victim.Name, SlotFingerR)); err != nil {
		t.Fatalf("wear: %v", err)
	}
	victim.SetPosition(combat.PosSleeping) // stealEquippedItem refuses awake victims
	thief.Level = LVL_IMPL
	thief.SetSkill(SkillSteal, 100)
	thief.SetPlrFlag(PlrOutlaw, true)
	thief.Stats.Str = 15
	thief.CopyBaseAttributes()

	for seed := uint32(1); seed < 5000; seed++ {
		dprng.ResetStream(seed)
		thief.SetPlrFlag(PlrCrash, false)
		victim.SetPlrFlag(PlrCrash, false)
		res := DoSteal(thief, victim, "thing", w)
		if !res.Success {
			continue
		}
		if !thief.NeedsCrashSave() {
			t.Fatal("equipped steal did not flag the thief (C act.other.c:429)")
		}
		if victim.NeedsCrashSave() {
			t.Fatal("equipped steal flagged the victim; unequip_char never sets PLR_CRASH")
		}
		return
	}
	t.Fatal("no seed made the equipped steal succeed")
}

// TestInstakillPreservesCrashFlag mirrors TestDeathPreservesCrashFlag on the
// Instakill corpse path (death.go), both bands, inventory and equipment.
func TestInstakillPreservesCrashFlag(t *testing.T) {
	for _, tc := range []struct {
		name string
		was  bool
	}{
		{"clear-stays-clear", false},
		{"set-stays-set", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, p := newFlagWorld(t)
			carried := newFlagItem(w, 7112, 1)
			if err := w.MoveObjectToPlayerInventory(carried, p); err != nil {
				t.Fatalf("carry: %v", err)
			}
			worn := newFlagItem(w, 7113, (1<<0)|(1<<14))
			if err := w.MoveObjectToPlayerInventory(worn, p); err != nil {
				t.Fatalf("seat worn: %v", err)
			}
			if err := w.MoveObject(worn, LocEquippedPlayer(p.Name, SlotHold)); err != nil {
				t.Fatalf("wear: %v", err)
			}
			p.SetPlrFlag(PlrCrash, tc.was)
			p.SetHealth(-20)
			killer := NewPlayer(2, "Killer", 1001)
			if err := w.AddPlayer(killer); err != nil {
				t.Fatalf("AddPlayer killer: %v", err)
			}

			w.Instakill(p, killer, 0)

			if got := p.NeedsCrashSave(); got != tc.was {
				t.Fatalf("PLR_CRASH after instakill corpse = %v, want %v (C fight.c:399, 406-407)", got, tc.was)
			}
		})
	}
}

// TestFleshAlterUnwieldSetsFlag drives DoFleshAlter with a wielded weapon:
// C's do_flesh_alter returns the weapon with obj_to_char(unequip_char(ch,
// WEAR_WIELD), ch) (src/new_cmds.c:1929).
func TestFleshAlterUnwieldSetsFlag(t *testing.T) {
	w, p := newFlagWorld(t)
	weapon := newFlagItem(w, 7114, (1<<0)|(1<<13))
	if err := w.MoveObjectToPlayerInventory(weapon, p); err != nil {
		t.Fatalf("carry: %v", err)
	}
	if err := w.MoveObject(weapon, LocEquippedPlayer(p.Name, SlotWield)); err != nil {
		t.Fatalf("wield: %v", err)
	}
	p.SetSkill(SkillFleshAlter, 100)
	p.SetPlrFlag(PlrCrash, false)

	res := DoFleshAlter(p)
	if !res.Success {
		t.Fatalf("flesh alter failed: %+v", res)
	}
	if _, ok := p.Equipment.GetItemInSlot(SlotWield); ok {
		t.Fatal("weapon still wielded after flesh alter")
	}
	if !p.NeedsCrashSave() {
		t.Fatal("flesh alter's unwield did not set PLR_CRASH (C new_cmds.c:1929)")
	}
}

// TestMobDisarmSetsVictimFlag drives the paladin spec's mobDisarm directly:
// C's spec disarms through do_disarm (src/spec_procs.c:562), whose
// obj_to_char flags the victim (new_cmds2.c:236).
func TestMobDisarmSetsVictimFlag(t *testing.T) {
	w, victim := newFlagWorld(t)
	me := &MobInstance{VNum: 2001}
	me.Level = 30
	weapon := newFlagItem(w, 7115, (1<<0)|(1<<13))
	if err := w.MoveObjectToPlayerInventory(weapon, victim); err != nil {
		t.Fatalf("carry: %v", err)
	}
	if err := w.MoveObject(weapon, LocEquippedPlayer(victim.Name, SlotWield)); err != nil {
		t.Fatalf("wield: %v", err)
	}
	victim.SetPlrFlag(PlrCrash, false)
	me.Fighting = true
	me.FightingTarget = victim.Name
	victim.Fighting = me.GetName()

	// Each attempt must assert the flag BEFORE any re-wield: the re-wield
	// itself passes through the inventory detach arm, which would set the
	// flag and mask a missing production line.
	for seed := uint32(1); seed < 5000; seed++ {
		dprng.ResetStream(seed)
		victim.SetPlrFlag(PlrCrash, false)
		mobDisarm(w, me, victim)
		if _, stillWielded := victim.Equipment.GetItemInSlot(SlotWield); stillWielded {
			continue // the roll declined to disarm; try the next seed
		}
		if !victim.NeedsCrashSave() {
			t.Fatal("mob disarm removed the weapon without setting the victim's PLR_CRASH (C spec_procs.c:562 → new_cmds2.c:236)")
		}
		return
	}
	t.Fatal("no seed made mob disarm fire")
}

// TestEviltradeSetsFlag drives specEviltrade with a gold watch (vnum 13111)
// in inventory: C's spec takes the traded object with obj_from_char
// (src/spec_procs2.c:1168).
func TestEviltradeSetsFlag(t *testing.T) {
	w, p := newFlagWorld(t)
	watch := newFlagItem(w, 13111, 1)
	if err := w.MoveObjectToPlayerInventory(watch, p); err != nil {
		t.Fatalf("carry: %v", err)
	}
	me := &MobInstance{VNum: 2001}
	me.RoomVNum = 1001
	p.SetPlrFlag(PlrCrash, false)

	specEviltrade(w, p, me, "trade", "")

	if _, ok := p.Inventory.FindItem("thing"); ok {
		t.Fatal("the watch is still in inventory")
	}
	if !p.NeedsCrashSave() {
		t.Fatal("eviltrade did not set PLR_CRASH (C spec_procs2.c:1168 obj_from_char)")
	}
}

// TestMobCommandGiveSetsRecipientFlag drives executeMobCommand's give path:
// C's perform_give ends in obj_to_char(obj, vict) (src/act.item.c:696).
func TestMobCommandGiveSetsRecipientFlag(t *testing.T) {
	w := newFlagWorldWithMob(t)
	recipient := NewPlayer(1, "Flagtest", 1001)
	if err := w.AddPlayer(recipient); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	mob, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	obj := newFlagItem(w, 7116, 1)
	mob.AddToInventory(obj)
	recipient.SetPlrFlag(PlrCrash, false)

	w.executeMobCommand(2001, "give thing "+recipient.Name)

	if _, ok := recipient.Inventory.FindItem("thing"); !ok {
		t.Fatal("mob give did not deliver the object")
	}
	if !recipient.NeedsCrashSave() {
		t.Fatal("mob give did not set the recipient's PLR_CRASH (C act.item.c:696)")
	}
}

// TestScriptableGiveSetsFlag drives GiveItemToCharScriptable — the lua
// objto/oload "char" path, C's obj_to_char (handler.c:569-571).
func TestScriptableGiveSetsFlag(t *testing.T) {
	w, p := newFlagWorld(t)
	obj := newFlagItem(w, 7117, 1)
	p.SetPlrFlag(PlrCrash, false)

	if err := w.GiveItemToCharScriptable(p.Name, &scriptableObjInstanceWrapper{item: obj}); err != nil {
		t.Fatalf("GiveItemToCharScriptable: %v", err)
	}
	if _, ok := p.Inventory.FindItem("thing"); !ok {
		t.Fatal("scriptable give did not deliver the object")
	}
	if !p.NeedsCrashSave() {
		t.Fatal("scriptable give did not set PLR_CRASH (C obj_to_char, handler.c:569-571)")
	}
}

// compile-time: combat import used by mobDisarm signature references.
var _ combat.Combatant = (*Player)(nil)
