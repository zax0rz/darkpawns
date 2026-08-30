package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func TestDoDisembowelDepthUsesCMessageAndDrawOrder(t *testing.T) {
	ch := NewPlayer(1, "Hero", 1001)
	ch.Level = 10
	ch.SetSkill(SkillDisembowel, 1)
	target := NewPlayer(2, "Victim", 1001)
	target.SetPosition(combat.PosSleeping)
	weapon := makeFidelityWeapon(9002, 11)
	if err := ch.Inventory.AddItem(weapon); err != nil {
		t.Fatalf("add piercing weapon: %v", err)
	}
	if err := ch.Equipment.Equip(weapon, ch.Inventory); err != nil {
		t.Fatalf("equip piercing weapon: %v", err)
	}

	const seed = 19
	dprng.ResetStream(seed)
	result := DoDisembowel(ch, target)
	if !result.Success {
		t.Fatal("sleeping target should reach the disembowel hit arm")
	}
	if result.SkillMsgType != SkillDisembowelNum || !result.StartCombat ||
		!result.SkillMsgAfterDamage || !result.SkillMsgInDamage || result.DamageSkill != SkillDisembowel {
		t.Errorf("result message/combat/path = (%d, %v, %v, %v, %q), want (%d, true, true, true, %q)",
			result.SkillMsgType, result.StartCombat, result.SkillMsgAfterDamage, result.SkillMsgInDamage,
			result.DamageSkill, SkillDisembowelNum, SkillDisembowel)
	}
	if result.Damage != ch.GetLevel()*2+ch.GetDamroll() {
		t.Errorf("damage = %d, want level*2+damroll = %d", result.Damage, ch.GetLevel()*2+ch.GetDamroll())
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillDisembowel {
		t.Errorf("deferred improvement = %#v, want [%q]", result.DeferredImprove, SkillDisembowel)
	}

	gotNext := dprng.Number(0, 999)
	dprng.ResetStream(seed)
	dprng.Number(1, 101) // do_disembowel percent
	dprng.Number(50, 100)
	dprng.Number(1, 20) // hit() to-hit roll
	dprng.Dice(1, 1)    // hit() weapon dice, replaced by disembowel damage
	wantNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("draw order/count mismatch: next=%d, want=%d", gotNext, wantNext)
	}
}

func TestDoDisembowel_SkillRollFailureUsesDamageMessagePath(t *testing.T) {
	ch := NewPlayer(1, "Hero", 1001)
	ch.Level = 10
	ch.SetSkill(SkillDisembowel, 1)
	target := NewPlayer(2, "Victim", 1001)
	target.SetPosition(combat.PosStanding)
	weapon := makeFidelityWeapon(9005, 11)
	if err := ch.Inventory.AddItem(weapon); err != nil {
		t.Fatalf("add piercing weapon: %v", err)
	}
	if err := ch.Equipment.Equip(weapon, ch.Inventory); err != nil {
		t.Fatalf("equip piercing weapon: %v", err)
	}

	var failure SkillResult
	for seed := uint32(1); seed <= 1000; seed++ {
		dprng.ResetStream(seed)
		failure = DoDisembowel(ch, target)
		if !failure.Success && failure.DeferredImprove == nil {
			break
		}
	}
	if failure.Success || failure.DeferredImprove != nil {
		t.Fatal("could not find a deterministic normal-player skill-roll failure")
	}
	if failure.SkillMsgType != SkillDisembowelNum || !failure.SkillMsgAfterDamage ||
		!failure.SkillMsgInDamage || failure.DamageSkill != SkillDisembowel || !failure.StartCombat {
		t.Fatalf("failure result = %#v, want numbered in-damage message path", failure)
	}
}

func TestDoDisembowel_NoWeapon(t *testing.T) {
	ch := NewPlayer(1, "Hero", 1001)
	ch.SetSkill(SkillDisembowel, 1)
	target := NewPlayer(2, "Victim", 1001)

	result := DoDisembowel(ch, target)
	if result.MessageToCh != "You need to wield a weapon to make it a success." {
		t.Fatalf("no-weapon message = %q", result.MessageToCh)
	}
}

func TestDoDisembowel_WrongWeapon(t *testing.T) {
	ch := NewPlayer(1, "Hero", 1001)
	ch.SetSkill(SkillDisembowel, 1)
	target := NewPlayer(2, "Victim", 1001)
	weapon := makeFidelityWeapon(9003, 3)
	if err := ch.Inventory.AddItem(weapon); err != nil {
		t.Fatalf("add weapon: %v", err)
	}
	if err := ch.Equipment.Equip(weapon, ch.Inventory); err != nil {
		t.Fatalf("equip weapon: %v", err)
	}

	result := DoDisembowel(ch, target)
	if result.MessageToCh != "Only piercing weapons can be used for disemboweling." {
		t.Fatalf("wrong-weapon message = %q", result.MessageToCh)
	}
}

func TestDoDisembowel_Mounted(t *testing.T) {
	ch := NewPlayer(1, "Hero", 1001)
	ch.SetSkill(SkillDisembowel, 1)
	ch.MountName = "a horse"
	target := NewPlayer(2, "Victim", 1001)
	weapon := makeFidelityWeapon(9004, 11)
	if err := ch.Inventory.AddItem(weapon); err != nil {
		t.Fatalf("add weapon: %v", err)
	}
	if err := ch.Equipment.Equip(weapon, ch.Inventory); err != nil {
		t.Fatalf("equip weapon: %v", err)
	}

	result := DoDisembowel(ch, target)
	if result.MessageToCh != "Dismount first!" {
		t.Fatalf("mounted message = %q", result.MessageToCh)
	}
}
