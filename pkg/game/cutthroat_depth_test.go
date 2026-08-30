package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

func cutthroatWeapon(t *testing.T, ch *Player, piercing bool) {
	t.Helper()
	weapon := makeSpikeWeapon("dagger")
	if piercing {
		weapon.Prototype.Values[3] = 11
	} else {
		weapon.Prototype.Values[3] = 3
	}
	equipWeapon(t, ch, weapon)
}

func TestDoCutthroatGateOrder(t *testing.T) {
	t.Run("weapon gates", func(t *testing.T) {
		w, ch := newCombatTestWorld(t)
		ch.SetSkill(SkillCutthroat, 100)
		mob := spawnTargetMob(t, w)

		result := DoCutthroat(ch, mob, w)
		if result.MessageToCh != "You need to wield a weapon to make it a success." {
			t.Fatalf("no-wield message = %q", result.MessageToCh)
		}

		cutthroatWeapon(t, ch, false)
		result = DoCutthroat(ch, mob, w)
		if result.MessageToCh != "Only daggers and such can be used for cutting a throat." {
			t.Fatalf("non-piercing message = %q", result.MessageToCh)
		}
	})

	t.Run("mounted precedes peaceful", func(t *testing.T) {
		w, ch := newCombatTestWorld(t)
		ch.SetSkill(SkillCutthroat, 100)
		mob := spawnTargetMob(t, w)
		cutthroatWeapon(t, ch, true)
		ch.SetAffect(affMounted, true)
		w.GetRoomInWorld(ch.GetRoom()).Flags = []string{"peaceful"}

		result := DoCutthroat(ch, mob, w)
		if result.MessageToCh != "Dismount first!" {
			t.Fatalf("mounted message = %q", result.MessageToCh)
		}

		ch.SetAffect(affMounted, false)
		result = DoCutthroat(ch, mob, w)
		if result.MessageToCh != "You feel too peaceful to slit a throat!" {
			t.Fatalf("peaceful message = %q", result.MessageToCh)
		}
	})

	t.Run("low-level and existing-affect gates", func(t *testing.T) {
		w, ch := newCombatTestWorld(t)
		ch.Level = LVL_IMMORT + 1
		ch.SetSkill(SkillCutthroat, 100)
		victim := NewPlayer(2, "Villager", ch.GetRoom())
		victim.Level = 10
		if err := w.AddPlayer(victim); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}
		cutthroatWeapon(t, ch, true)

		result := DoCutthroat(ch, victim, w)
		if result.MessageToCh != "Ancient forces protect Villager from your wrath!" {
			t.Fatalf("low-level message = %q", result.MessageToCh)
		}

		victim.Level = 11
		victim.SetAffect(affCutthroat, true)
		result = DoCutthroat(ch, victim, w)
		if result.MessageToCh != "Their throat is already slit!" {
			t.Fatalf("existing-affect message = %q", result.MessageToCh)
		}
	})

	t.Run("fighting gate", func(t *testing.T) {
		w, ch := newCombatTestWorld(t)
		ch.Level = LVL_IMMORT + 1
		ch.SetSkill(SkillCutthroat, 100)
		mob := spawnTargetMob(t, w)
		cutthroatWeapon(t, ch, true)
		ch.SetFighting("someone else")

		result := DoCutthroat(ch, mob, w)
		if result.MessageToCh != "You can't get close enough!" {
			t.Fatalf("fighting message = %q", result.MessageToCh)
		}
	})
}

func TestDoCutthroatSuccessAndFailureContracts(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	ch.Level = LVL_IMMORT + 1
	ch.SetSkill(SkillCutthroat, 100)
	victim := NewPlayer(2, "Victim", ch.GetRoom())
	victim.Level = 11
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	cutthroatWeapon(t, ch, true)

	result := DoCutthroat(ch, victim, w)
	if !result.Success || result.Damage != ch.GetLevel()/2 {
		t.Fatalf("success = %#v, want success and level/2 damage", result)
	}
	if result.SkillMsgType != SkillCutthroatNum || result.DamageSkill != SkillCutthroat {
		t.Fatalf("success message contract = set %d damage skill %q", result.SkillMsgType, result.DamageSkill)
	}
	if !result.SkillMsgAfterDamage || len(result.PreDamageImprove) != 1 || !result.StartCombat || result.WaitCh != 2 {
		t.Fatalf("success pipeline contract = %#v", result)
	}
	if !victim.IsAffected(affCutthroat) || len(victim.ActiveAffects) != 1 {
		t.Fatalf("success did not install one cutthroat affect: affected=%v count=%d", victim.IsAffected(affCutthroat), len(victim.ActiveAffects))
	}
	affect := victim.ActiveAffects[0]
	if affect.Duration != ch.GetLevel()*2 || affect.Location != engine.ApplyHitroll || affect.Magnitude != -2 || affect.Flags != engine.AFFCutthroat {
		t.Fatalf("cutthroat affect = %#v", affect)
	}

	ch.SetFighting("")
	victim.SetAffect(affCutthroat, false)
	victim.ActiveAffects = nil
	ch.Level = 20
	ch.SetSkill(SkillCutthroat, 1)
	dprng.ResetStream(1)
	result = DoCutthroat(ch, victim, w)
	if result.Success || !result.InitialAttack || result.MessageToCh != "Your slash at their throat barely misses!" {
		t.Fatalf("failure contract = %#v", result)
	}
	if !strings.Contains(result.MessageToVict, "makes a vicious lunge") || result.WaitCh != 2 {
		t.Fatalf("failure messages/wait = %#v", result)
	}
}

func TestDoCutthroatNoSkillDoesNotDraw(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	dprng.ResetStream(1)
	wantNext := dprng.Number(1, 101)
	dprng.ResetStream(1)
	result := DoCutthroat(ch, mob, w)
	if result.MessageToCh != "You're not trained in slitting throats!" {
		t.Fatalf("no-skill message = %q", result.MessageToCh)
	}
	if got := dprng.Number(1, 101); got != wantNext {
		t.Fatalf("no-skill path consumed RNG; next draw = %d want %d", got, wantNext)
	}
}

func TestDoCutthroatImmortalTargetOverridesCaster(t *testing.T) {
	w, ch := newCombatTestWorld(t)
	ch.Level = LVL_IMMORT + 1
	ch.SetSkill(SkillCutthroat, 100)
	victim := NewPlayer(2, "Immortal", ch.GetRoom())
	victim.Level = LVL_IMMORT + 1
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	cutthroatWeapon(t, ch, true)

	dprng.ResetStream(1)
	result := DoCutthroat(ch, victim, w)
	if result.Success || !result.InitialAttack {
		t.Fatalf("immortal-target result = %#v, want forced failure and ordinary hit", result)
	}
	if result.MessageToCh != "Your slash at their throat barely misses!" {
		t.Fatalf("immortal-target actor message = %q", result.MessageToCh)
	}
}
