package game

import (
	"fmt"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func seedForFirstNumber(t *testing.T, low, high int, accept func(int) bool) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 10000; seed++ {
		dprng.ResetStream(seed)
		if n := dprng.Number(low, high); accept(n) {
			return seed
		}
	}
	t.Fatal("no suitable deterministic seed found")
	return 0
}

func newStealTestMob(gold int) *MobInstance {
	mob := NewMob(&parser.Mob{
		VNum:      2001,
		Keywords:  "mark target",
		ShortDesc: "a test mark",
		Level:     1,
		Str:       10,
		Dex:       10,
	}, 1001)
	mob.SetGold(gold)
	return mob
}

func newStealTestThief(skill int) *Player {
	thief := NewPlayer(1, "Thief", 1001)
	thief.Class = ClassThief
	thief.Stats = CharStats{Str: 25, Dex: 25, Int: 10, Wis: 10, Con: 10, Cha: 10}
	thief.SetSkill(SkillSteal, skill)
	return thief
}

func TestDoHideDexBonusToggleAndImprove(t *testing.T) {
	seed := seedForFirstNumber(t, 1, 101, func(n int) bool { return n > 5 && n <= 30 })

	ch := NewPlayer(1, "Thief", 1001)
	ch.Stats = CharStats{Dex: 25}
	ch.SetSkill(SkillHide, 5)
	dprng.ResetStream(seed)
	result := DoHide(ch)
	if !result.Success || !ch.IsAffected(affHide) {
		t.Fatalf("dex-25 hide should succeed with table bonus: result=%+v hidden=%v", result, ch.IsAffected(affHide))
	}
	if result.MessageToCh != "You attempt to hide yourself." {
		t.Errorf("hide message = %q", result.MessageToCh)
	}

	// The same roll fails without the dex bonus. Starting hidden verifies that C
	// clears the old bit and rerolls instead of treating hide as a toggle.
	ch.Stats.Dex = 15
	ch.SetAffect(affHide, true)
	dprng.ResetStream(seed)
	result = DoHide(ch)
	if result.Success || ch.IsAffected(affHide) {
		t.Fatalf("rerolled hide should fail and clear old bit: result=%+v hidden=%v", result, ch.IsAffected(affHide))
	}
	if result.MessageToCh != "You attempt to hide yourself." {
		t.Errorf("failed hide message = %q", result.MessageToCh)
	}

	// Guaranteed success: one hide roll followed by improveSkill's gate and
	// increment draws.
	ch.Stats.Int = 100
	ch.Stats.Wis = 100
	ch.SetSkill(SkillHide, 50)
	ch.Stats.Dex = 25
	dprng.ResetStream(1)
	dprng.Number(1, 101)
	dprng.Number(1, 200)
	inc := dprng.Number(1, 3)
	dprng.ResetStream(1)
	result = DoHide(ch)
	if got := ch.GetSkill(SkillHide); got != 50+inc {
		t.Fatalf("hide improveSkill increment = %d; want %d", got, 50+inc)
	}
	wantMessage := "You attempt to hide yourself."
	if inc == 3 {
		wantMessage += "\r\nYour skill in hide improves."
	}
	if result.MessageToCh != wantMessage {
		t.Errorf("hide/improve output order = %q; want %q", result.MessageToCh, wantMessage)
	}
}

func TestDoSneakTimedAffectAndReroll(t *testing.T) {
	ch := NewPlayer(1, "Thief", 1001)
	ch.SetLevel(7)
	ch.Stats.Dex = 25
	ch.SetSkill(SkillSneak, 1000)
	ch.SetAffect(affSneak, true)
	ch.AddAffect(engine.NewAffectDirect(skillNumStealth, engine.ApplyNone, 3, 0, engine.AFFSneak, SkillStealth))

	dprng.ResetStream(1)
	result := DoSneak(ch)
	if !result.Success || result.MessageToCh != "Okay, you'll try to move silently for a while." {
		t.Fatalf("DoSneak result = %+v", result)
	}
	if !ch.IsAffected(affSneak) {
		t.Fatal("successful sneak did not apply AFF_SNEAK")
	}
	if len(ch.ActiveAffects) != 1 {
		t.Fatalf("active sneak affects = %d; want exactly 1", len(ch.ActiveAffects))
	}
	affect := ch.ActiveAffects[0]
	if affect.SpellID != skillNumSneak || affect.Duration != 7 || affect.Flags != engine.AFFSneak {
		t.Errorf("sneak affect = %+v; want spell=%d duration=7 flags=%d", affect, skillNumSneak, engine.AFFSneak)
	}
}

func TestDoStealCoinsDrawOrderGoldAndImprove(t *testing.T) {
	thief := newStealTestThief(50)
	thief.Stats.Int = 100
	thief.Stats.Wis = 100
	mob := newStealTestMob(1000)

	// C draw order: initial percent, coin percentage, improve gate, increment.
	dprng.ResetStream(1)
	dprng.Number(1, 101)
	pct := dprng.Number(1, 10)
	dprng.Number(1, 200)
	inc := dprng.Number(1, 3)
	wantGold := 1000 * pct / 100

	dprng.ResetStream(1)
	result := DoSteal(thief, mob, "coins", nil)
	if !result.Success {
		t.Fatalf("coin steal failed: %+v", result)
	}
	wantMessage := fmt.Sprintf("Bingo!  You got %d gold coins.", wantGold)
	if inc == 3 {
		wantMessage += "\r\nYour skill in steal improves."
	}
	if result.MessageToCh != wantMessage {
		t.Errorf("coin message = %q; want %q", result.MessageToCh, wantMessage)
	}
	if got := thief.GetGold(); got != wantGold {
		t.Errorf("thief gold = %d; want %d", got, wantGold)
	}
	if got := mob.GetGold(); got != 1000-wantGold {
		t.Errorf("mob gold = %d; want %d", got, 1000-wantGold)
	}
	if got := thief.GetSkill(SkillSteal); got != 50+inc {
		t.Errorf("steal skill = %d; want %d", got, 50+inc)
	}
	if result.WaitCh != 1 {
		t.Errorf("steal wait = %d; want 1", result.WaitCh)
	}
}

func TestDoStealCoinsZeroGoldDoesNotImprove(t *testing.T) {
	thief := newStealTestThief(1000)
	mob := newStealTestMob(1)

	dprng.ResetStream(1)
	result := DoSteal(thief, mob, "gold", nil)
	if !result.Success || result.MessageToCh != "You couldn't get any gold..." {
		t.Fatalf("zero-gold result = %+v", result)
	}
	if got := thief.GetSkill(SkillSteal); got != 1000 {
		t.Errorf("zero-gold steal improved skill to %d", got)
	}
}

func TestDoStealFailureAggrosAwakeMob(t *testing.T) {
	thief := newStealTestThief(0)
	thief.Stats.Dex = 15
	mob := newStealTestMob(1000)

	dprng.ResetStream(1)
	result := DoSteal(thief, mob, "coins", nil)
	if result.Success || result.MessageToCh != "Oops.." {
		t.Fatalf("failed coin steal result = %+v", result)
	}
	if !result.StartCombat {
		t.Error("failed steal against awake NPC must initiate combat")
	}
	if result.MessageToVict != "You discover that Thief has his hands in your wallet." {
		t.Errorf("victim message = %q", result.MessageToVict)
	}
}

func TestDoStealInitialRollPrecedesMissingItemLookup(t *testing.T) {
	thief := newStealTestThief(1000)
	mob := newStealTestMob(0)

	dprng.ResetStream(1)
	dprng.Number(1, 101)
	wantNext := dprng.Number(1, 101)

	dprng.ResetStream(1)
	result := DoSteal(thief, mob, "missing", nil)
	if result.MessageToCh != "It hasn't got that item." {
		t.Fatalf("missing-item message = %q", result.MessageToCh)
	}
	if got := dprng.Number(1, 101); got != wantNext {
		t.Fatalf("missing-item draw count: next=%d want=%d", got, wantNext)
	}
}

func TestDoStealInventoryAndEquipmentBranches(t *testing.T) {
	t.Run("inventory", func(t *testing.T) {
		thief := newStealTestThief(1000)
		mob := newStealTestMob(0)
		item := NewObjectInstance(&parser.Obj{VNum: 3001, Keywords: "gem ruby", ShortDesc: "a ruby", Weight: 2}, 1001)
		mob.AddToInventory(item)

		dprng.ResetStream(1)
		result := DoSteal(thief, mob, "gem", nil)
		if !result.Success || result.MessageToCh != "Got it!" {
			t.Fatalf("inventory steal result = %+v", result)
		}
		if _, found := thief.Inventory.FindItem("gem"); !found {
			t.Fatal("stolen inventory item was not transferred")
		}
	})

	t.Run("sleeping equipment", func(t *testing.T) {
		thief := newStealTestThief(1000)
		mob := newStealTestMob(0)
		mob.SetPosition(combat.PosSleeping)
		item := NewObjectInstance(&parser.Obj{VNum: 3002, Keywords: "ring silver", ShortDesc: "a silver ring", Weight: 1}, 1001)
		mob.EquipItem(item, int(SlotFingerR))

		dprng.ResetStream(1)
		result := DoSteal(thief, mob, "ring", nil)
		if !result.Success || result.MessageToCh != "You unequip a silver ring and steal it." {
			t.Fatalf("equipment steal result = %+v", result)
		}
		if _, found := thief.Inventory.FindItem("ring"); !found {
			t.Fatal("stolen equipment was not transferred")
		}
	})

	t.Run("awake equipment", func(t *testing.T) {
		thief := newStealTestThief(1000)
		mob := newStealTestMob(0)
		item := NewObjectInstance(&parser.Obj{VNum: 3003, Keywords: "ring gold", ShortDesc: "a gold ring", Weight: 1}, 1001)
		mob.EquipItem(item, int(SlotFingerR))

		dprng.ResetStream(1)
		result := DoSteal(thief, mob, "ring", nil)
		if result.Success || result.MessageToCh != "Steal the equipment now?  Impossible!" {
			t.Fatalf("awake equipment result = %+v", result)
		}
	})

	t.Run("equipment bypasses full victim inventory", func(t *testing.T) {
		thief := newStealTestThief(1000)
		thief.SetPlrFlag(PlrOutlaw, true)
		victim := NewPlayer(2, "Victim", 1001)
		victim.SetPosition(combat.PosSleeping)
		victim.Inventory.Capacity = 1
		filler := NewObjectInstance(&parser.Obj{VNum: 3004, Keywords: "stone", ShortDesc: "a stone"}, 1001)
		if err := victim.Inventory.AddItem(filler); err != nil {
			t.Fatalf("fill victim inventory: %v", err)
		}
		item := NewObjectInstance(&parser.Obj{VNum: 3005, Keywords: "ring bronze", ShortDesc: "a bronze ring", Weight: 1}, 1001)
		if err := victim.Equipment.SetSlot(SlotFingerR, item); err != nil {
			t.Fatalf("equip victim: %v", err)
		}
		item.Location = LocEquippedPlayer(victim.Name, SlotFingerR)

		dprng.ResetStream(1)
		result := DoSteal(thief, victim, "ring", nil)
		if !result.Success {
			t.Fatalf("full victim inventory blocked equipment steal: %+v", result)
		}
		if _, found := thief.Inventory.FindItem("ring"); !found {
			t.Fatal("player equipment was not transferred")
		}
	})
}
