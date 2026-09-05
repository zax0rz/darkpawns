package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newDisarmDepthPair(t *testing.T, targetFighting bool) (*Player, *Player, *ObjectInstance) {
	t.Helper()
	ch := NewPlayer(1, "Hero", 1001)
	ch.Level = 10
	ch.SetPosition(combat.PosFighting)
	ch.SetSkill(SkillDisarm, 100)

	target := NewPlayer(2, "Victim", 1001)
	target.Level = 11
	target.SetPosition(combat.PosFighting)
	ch.SetFighting(target.GetName())
	if targetFighting {
		target.SetFighting(ch.GetName())
	}

	weapon := NewObjectInstance(&parser.Obj{
		VNum:      9001,
		Keywords:  "test weapon",
		ShortDesc: "a test weapon",
		TypeFlag:  5,
		WearFlags: [4]int{1 << 13},
		Values:    [4]int{0, 1, 1, 3},
	}, -1)
	if err := target.Inventory.AddItem(weapon); err != nil {
		t.Fatalf("add test weapon: %v", err)
	}
	if err := target.Equipment.Equip(weapon, target.Inventory); err != nil {
		t.Fatalf("equip test weapon: %v", err)
	}
	return ch, target, weapon
}

func TestDoDisarmDepthPlayerSuccess(t *testing.T) {
	ch, target, weapon := newDisarmDepthPair(t, true)

	dprng.ResetStream(1)
	result := DoDisarm(ch, target, nil)
	if !result.Success {
		t.Fatal("seed 1 should pass disarm at skill 100")
	}
	if _, ok := target.Equipment.GetItemInSlot(SlotWield); ok {
		t.Fatal("successful disarm left the weapon equipped")
	}
	if weapon.Location != LocInventoryPlayer(target.GetName()) {
		t.Fatalf("weapon location = %#v, want victim inventory", weapon.Location)
	}

	if got, want := result.MessageToCh, "You disarm Victim and a test weapon goes flying!"; got != want {
		t.Errorf("actor message = %q, want %q", got, want)
	}
	if got, want := result.MessageToVict, "Hero deftly disarms you, knocking a test weapon from your hand!"; got != want {
		t.Errorf("victim message = %q, want %q", got, want)
	}
	if got, want := result.MessageToRoom, "Hero knocks a test weapon from Victim's hand!"; got != want {
		t.Errorf("room message = %q, want %q", got, want)
	}
	if result.RetaliateHit || !result.RetaliateHitAfterMessages {
		t.Errorf("retaliation flags = (%v, %v), want (false, true)", result.RetaliateHit, result.RetaliateHitAfterMessages)
	}
	if result.WaitCh != 2 {
		t.Errorf("wait rounds = %d, want 2", result.WaitCh)
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillDisarm {
		t.Errorf("deferred improvement = %#v, want [%q]", result.DeferredImprove, SkillDisarm)
	}
}

func TestDoDisarmDepthWeaponGatePrecedesCombatGate(t *testing.T) {
	ch := NewPlayer(1, "Hero", 1001)
	ch.SetSkill(SkillDisarm, 100)
	ch.SetFighting("Victim")
	target := NewPlayer(2, "Victim", 1001)

	result := DoDisarm(ch, target, nil)
	if result.MessageToCh != "he doesn't have anything to disarm." {
		t.Fatalf("no-weapon message = %q, want exact target-pronoun act", result.MessageToCh)
	}
}

func TestDoDisarmDepthFailureState(t *testing.T) {
	ch, target, _ := newDisarmDepthPair(t, true)
	ch.SetSkill(SkillDisarm, 1)

	dprng.ResetStream(1)
	result := DoDisarm(ch, target, nil)
	if result.Success {
		t.Fatal("seed 1 should fail disarm at skill 1")
	}
	if _, ok := target.Equipment.GetItemInSlot(SlotWield); !ok {
		t.Fatal("failed disarm removed the target weapon")
	}
	if !result.SelfStumble || result.RetaliateHit || !result.RetaliateHitAfterMessages {
		t.Errorf("failure flags = stumble %v, retaliate %v, after %v", result.SelfStumble, result.RetaliateHit, result.RetaliateHitAfterMessages)
	}
	if result.WaitCh != 2 {
		t.Errorf("failure wait rounds = %d, want 2", result.WaitCh)
	}
}

func TestDoDisarmDepthRetaliatesWhenVictimWasNotFighting(t *testing.T) {
	ch, target, _ := newDisarmDepthPair(t, false)

	dprng.ResetStream(1)
	result := DoDisarm(ch, target, nil)
	if !result.Success {
		t.Fatal("seed 1 should pass disarm at skill 100")
	}
	if !result.RetaliateHit || !result.RetaliateHitAfterMessages {
		t.Errorf("one-way combat retaliation flags = (%v, %v), want (true, true)", result.RetaliateHit, result.RetaliateHitAfterMessages)
	}
}
