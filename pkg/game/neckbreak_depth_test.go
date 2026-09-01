package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDoNeckbreakDepthGateOrder(t *testing.T) {
	t.Run("no skill", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		victim := NewPlayer(2, "Victim", 1001)
		result := DoNeckbreak(ch, victim, nil)
		if result.MessageToCh != "What's that, idiot-san?" {
			t.Fatalf("message = %q, want no-skill gate", result.MessageToCh)
		}
	})

	t.Run("wielded weapon", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		ch.SetSkill(SkillNeckbreak, 100)
		weapon := makeFidelityWeapon(990190, 3)
		if err := ch.Inventory.AddItem(weapon); err != nil {
			t.Fatalf("add weapon: %v", err)
		}
		if err := ch.Equipment.Equip(weapon, ch.Inventory); err != nil {
			t.Fatalf("equip weapon: %v", err)
		}
		result := DoNeckbreak(ch, NewPlayer(2, "Victim", 1001), nil)
		if result.MessageToCh != "You can't do this and wield a weapon at the same time!" {
			t.Fatalf("message = %q, want wielded-weapon gate", result.MessageToCh)
		}
	})

	t.Run("shopkeeper", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		ch.SetSkill(SkillNeckbreak, 100)
		const vnum = 990191
		old, had := MobSpecAssign[vnum]
		MobSpecAssign[vnum] = "shop_keeper"
		defer func() {
			if had {
				MobSpecAssign[vnum] = old
			} else {
				delete(MobSpecAssign, vnum)
			}
		}()
		victim := NewMob(&parser.Mob{VNum: vnum, Keywords: "keeper", ShortDesc: "a keeper"}, 1001)
		result := DoNeckbreak(ch, victim, nil)
		if result.MessageToCh != "Haha.. Don't think so." {
			t.Fatalf("message = %q, want shopkeeper gate", result.MessageToCh)
		}
	})

	t.Run("self", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		ch.SetSkill(SkillNeckbreak, 100)
		result := DoNeckbreak(ch, ch, nil)
		if result.MessageToCh != "Aren't we funny today..." {
			t.Fatalf("message = %q, want self gate", result.MessageToCh)
		}
	})

	t.Run("peaceful", func(t *testing.T) {
		world, err := NewWorld(&parser.World{Rooms: []parser.Room{{
			VNum: 1001, Name: "Peaceful Room", Zone: 1, Flags: []string{"peaceful"},
		}}})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		t.Cleanup(world.StopAITicker)
		ch := NewPlayer(1, "Hero", 1001)
		ch.SetSkill(SkillNeckbreak, 100)
		result := DoNeckbreak(ch, NewPlayer(2, "Victim", 1001), world)
		if result.MessageToCh != "You can't contemplate violence in such a place!" {
			t.Fatalf("message = %q, want peaceful gate", result.MessageToCh)
		}
	})

	t.Run("mounted", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		ch.SetSkill(SkillNeckbreak, 100)
		ch.MountName = "a horse"
		result := DoNeckbreak(ch, NewPlayer(2, "Victim", 1001), nil)
		if result.MessageToCh != "Dismount first!" {
			t.Fatalf("message = %q, want mounted gate", result.MessageToCh)
		}
	})

	t.Run("low move", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		ch.SetSkill(SkillNeckbreak, 100)
		ch.SetMove(50)
		result := DoNeckbreak(ch, NewPlayer(2, "Victim", 1001), nil)
		if result.MessageToCh != "You haven't the energy to do this!" {
			t.Fatalf("message = %q, want low-move gate", result.MessageToCh)
		}
		if got := ch.GetMove(); got != 50 {
			t.Fatalf("move = %d, want unchanged 50 after rejected attempt", got)
		}
	})
}

func TestDoNeckbreakDepthResultContractAndDrawOrder(t *testing.T) {
	const seed = 1
	ch := NewPlayer(1, "Hero", 1001)
	ch.Level = 10
	ch.SetSkill(SkillNeckbreak, 100)
	ch.SetMove(100)
	target := NewPlayer(2, "Victim", 1001)

	dprng.ResetStream(seed)
	result := DoNeckbreak(ch, target, nil)
	if !result.Success || result.Damage <= 0 {
		t.Fatalf("result = %#v, want successful damaging arm", result)
	}
	if result.SkillMsgType != SkillNeckbreakNum || !result.SkillMsgInDamage || result.DamageSkill != SkillNeckbreak {
		t.Fatalf("message path = set %d in-damage %v skill %q, want set %d/true/%q", result.SkillMsgType, result.SkillMsgInDamage, result.DamageSkill, SkillNeckbreakNum, SkillNeckbreak)
	}
	if !result.StartCombat || result.WaitCh != 3 {
		t.Fatalf("combat/wait contract = start %v wait %d, want true/3", result.StartCombat, result.WaitCh)
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillNeckbreak {
		t.Fatalf("deferred improvement = %#v, want [%q]", result.DeferredImprove, SkillNeckbreak)
	}

	gotNext := dprng.Number(0, 999)
	dprng.ResetStream(seed)
	dprng.Number(1, 101)          // do_neckbreak percent
	dprng.Dice(18, ch.GetLevel()) // neckbreak damage before damage()/skill_message
	wantNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("neckbreak consumed the wrong draws: next=%d want=%d", gotNext, wantNext)
	}
}

func TestDoNeckbreakDepthFailureRetaliationContract(t *testing.T) {
	ch := NewPlayer(1, "Hero", 1001)
	ch.SetSkill(SkillNeckbreak, 1)
	ch.SetMove(100)
	target := NewPlayer(2, "Victim", 1001)

	dprng.ResetStream(1)
	result := DoNeckbreak(ch, target, nil)
	if result.Success {
		t.Fatal("skill 1 should take the deterministic failure arm")
	}
	if result.WaitCh != 3 || !result.RetaliateHit || !result.RetaliateHitAfterMessages {
		t.Fatalf("failure contract = wait %d retaliate %v after %v, want 3/true/true", result.WaitCh, result.RetaliateHit, result.RetaliateHitAfterMessages)
	}
	if result.MessageToCh == "" || result.MessageToVict == "" || result.MessageToRoom == "" {
		t.Fatalf("failure messages = %#v, want all three C act branches", result)
	}
}

func TestDoNeckbreakDepthPeacefulUsesCanonicalRoomBit(t *testing.T) {
	world, err := NewWorld(&parser.World{Rooms: []parser.Room{{
		VNum: 1001, Name: "Peaceful Room", Zone: 1, Flags: []string{"16", "0", "0", "0"},
	}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)
	ch := NewPlayer(1, "Hero", 1001)
	ch.SetSkill(SkillNeckbreak, 100)
	ch.SetMove(100)
	result := DoNeckbreak(ch, NewPlayer(2, "Victim", 1001), world)
	if result.MessageToCh != "You can't contemplate violence in such a place!" {
		t.Fatalf("message = %q, want numeric peaceful gate", result.MessageToCh)
	}
}

var _ combat.Combatant = (*Player)(nil)
