package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDoGroinripDepthResultContract(t *testing.T) {
	ch := NewPlayer(1, "Hero", 1001)
	ch.Level = 20
	ch.SetSkill(SkillGroinrip, 75)
	target := NewPlayer(2, "Victim", 1001)
	target.SetPosition(combat.PosSleeping)

	const seed = 19
	dprng.ResetStream(seed)
	result := DoGroinrip(ch, target, nil)
	if !result.Success || result.Damage != ch.GetLevel() {
		t.Fatalf("sleeping-target result = success %v damage %d, want true/%d", result.Success, result.Damage, ch.GetLevel())
	}
	if result.SkillMsgType != SkillGroinripNum || !result.SkillMsgInDamage || result.DamageSkill != SkillGroinrip {
		t.Fatalf("message path = set %d in-damage %v skill %q, want set %d/true/%q", result.SkillMsgType, result.SkillMsgInDamage, result.DamageSkill, SkillGroinripNum, SkillGroinrip)
	}
	if !result.StartCombat || result.WaitCh != 2 {
		t.Fatalf("combat/wait contract = start %v wait %d, want true/2", result.StartCombat, result.WaitCh)
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillGroinrip || !result.DeferredImproveAfterRoom {
		t.Fatalf("improvement contract = %#v after-room %v, want [%q]/true", result.DeferredImprove, result.DeferredImproveAfterRoom, SkillGroinrip)
	}
	if !result.RoomIncludesActor || !result.SpawnPuke || !strings.Contains(result.MessageToRoom, "\r\neverywhere!") {
		t.Fatalf("post-damage contract = actor %v puke %v room %q, want actor/puke and C line break", result.RoomIncludesActor, result.SpawnPuke, result.MessageToRoom)
	}

	gotNext := dprng.Number(0, 999)
	dprng.ResetStream(seed)
	dprng.Number(1, 121) // do_groinrip percent, even when sleep forces percent=0
	wantNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("groinrip roll consumed the wrong draws: next=%d want=%d", gotNext, wantNext)
	}
}

func TestDoGroinripDepthGateOrder(t *testing.T) {
	t.Run("no skill precedes mounted", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		ch.MountName = "a horse"
		victim := NewPlayer(2, "Victim", 1001)
		result := DoGroinrip(ch, victim, nil)
		if result.MessageToCh != "You're not trained in martial arts!" {
			t.Fatalf("message = %q, want no-skill gate", result.MessageToCh)
		}
	})

	t.Run("mounted", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		ch.SetSkill(SkillGroinrip, 75)
		ch.MountName = "a horse"
		victim := NewPlayer(2, "Victim", 1001)
		result := DoGroinrip(ch, victim, nil)
		if result.MessageToCh != "Dismount first!" {
			t.Fatalf("message = %q, want mounted gate", result.MessageToCh)
		}
	})

	t.Run("self", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		ch.SetSkill(SkillGroinrip, 75)
		result := DoGroinrip(ch, ch, nil)
		if result.MessageToCh != "No masochism allowed!" {
			t.Fatalf("message = %q, want self gate", result.MessageToCh)
		}
	})

	t.Run("shopkeeper", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		ch.SetSkill(SkillGroinrip, 75)
		const vnum = 990174
		old, had := MobSpecAssign[vnum]
		MobSpecAssign[vnum] = "shop_keeper"
		defer func() {
			if had {
				MobSpecAssign[vnum] = old
			} else {
				delete(MobSpecAssign, vnum)
			}
		}()
		victim := NewMob(&parser.Mob{VNum: vnum, Keywords: "keeper", ShortDesc: "a keeper", Sex: 1}, 1001)
		result := DoGroinrip(ch, victim, nil)
		if result.MessageToCh != "Ha Ha. Don't think so." {
			t.Fatalf("message = %q, want shopkeeper gate", result.MessageToCh)
		}
	})

	t.Run("immortal victim", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		ch.SetSkill(SkillGroinrip, 75)
		victim := NewPlayer(2, "God", 1001)
		victim.SetLevel(LVL_IMMORT)
		result := DoGroinrip(ch, victim, nil)
		if result.MessageToCh != "How dare you try to touch a god!\r\nYou are thrown across the room..." || ch.GetPosition() != combat.PosSitting {
			t.Fatalf("result = message %q position %d, want god rejection and sitting", result.MessageToCh, ch.GetPosition())
		}
	})

	t.Run("nonmale", func(t *testing.T) {
		ch := NewPlayer(1, "Hero", 1001)
		ch.SetSkill(SkillGroinrip, 75)
		victim := NewPlayer(2, "Girl", 1001)
		victim.SetSex(SexFemale)
		result := DoGroinrip(ch, victim, nil)
		if result.MessageToCh != "Umm, they have nothing there to tug on!" {
			t.Fatalf("message = %q, want nonmale gate", result.MessageToCh)
		}
	})
}
