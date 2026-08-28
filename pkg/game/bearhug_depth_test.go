package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// TestDoBearhug_ResultContract pins the C do_bearhug damage() contract for
// both branches: set 142 supplies the player-facing text, damage() starts
// combat even on a zero-damage miss, and the player waits two violence pulses.
func TestDoBearhug_ResultContract(t *testing.T) {
	w, ch := newBashTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.SetSkill(SkillBearhug, 100)

	var hit SkillResult
	for seed := uint32(1); seed < 100 && !hit.Success; seed++ {
		dprng.ResetStream(seed)
		hit = DoBearhug(ch, mob, w)
	}
	if !hit.Success {
		t.Fatal("no bearhug hit observed in 99 deterministic seeds")
	}
	if hit.Damage != ch.GetLevel()+ch.GetLevel()/2 {
		t.Fatalf("hit damage = %d, want level*1.5 = %d", hit.Damage, ch.GetLevel()+ch.GetLevel()/2)
	}
	if hit.SkillMsgType != SkillBearhugNum {
		t.Fatalf("hit SkillMsgType = %d, want %d", hit.SkillMsgType, SkillBearhugNum)
	}
	if !hit.StartCombat || hit.WaitCh != 2 {
		t.Fatalf("hit combat/wait contract = start %v, wait %d; want true, 2", hit.StartCombat, hit.WaitCh)
	}
	if len(hit.DeferredImprove) != 1 || hit.DeferredImprove[0] != SkillBearhug {
		t.Fatalf("hit DeferredImprove = %v, want [%q]", hit.DeferredImprove, SkillBearhug)
	}
	if hit.MessageToCh != "" || hit.MessageToVict != "" || hit.MessageToRoom != "" {
		t.Fatalf("hit must use skill_message set 142, got hardcoded messages: ch=%q victim=%q room=%q", hit.MessageToCh, hit.MessageToVict, hit.MessageToRoom)
	}

	ch.SetSkill(SkillBearhug, 1)
	dprng.ResetStream(1)
	miss := DoBearhug(ch, mob, w)
	if miss.Success || miss.Damage != 0 {
		t.Fatalf("low-skill bearhug = success %v damage %d, want failure/0", miss.Success, miss.Damage)
	}
	if miss.SkillMsgType != SkillBearhugNum || !miss.StartCombat || miss.WaitCh != 2 {
		t.Fatalf("miss contract = set %d, start %v, wait %d; want %d, true, 2", miss.SkillMsgType, miss.StartCombat, miss.WaitCh, SkillBearhugNum)
	}
	if len(miss.DeferredImprove) != 0 || miss.MessageToCh != "" {
		t.Fatalf("miss must not improve or invent text: improve=%v ch=%q", miss.DeferredImprove, miss.MessageToCh)
	}
}

func TestDoBearhug_CForcedFailureBranches(t *testing.T) {
	tests := []struct {
		name string
		set  func(*World, *Player, *MobInstance)
	}{
		{
			name: "sleeping target",
			set: func(_ *World, _ *Player, mob *MobInstance) {
				mob.SetPosition(combat.PosSleeping)
			},
		},
		{
			name: "immortal caster",
			set: func(_ *World, ch *Player, _ *MobInstance) {
				ch.SetLevel(LVL_IMMORT + 1)
			},
		},
		{
			name: "nobash target",
			set: func(_ *World, _ *Player, mob *MobInstance) {
				mob.SetMobFlag(MobFlagNobash)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, ch := newBashTestWorld(t)
			mob := spawnTargetMob(t, w)
			mob.SetPosition(combat.PosFighting)
			tc.set(w, ch, mob)
			ch.SetSkill(SkillBearhug, 100)

			result := DoBearhug(ch, mob, w)
			if result.Success || result.Damage != 0 {
				t.Fatalf("forced branch = success %v damage %d, want failure/0", result.Success, result.Damage)
			}
			if result.SkillMsgType != SkillBearhugNum {
				t.Fatalf("SkillMsgType = %d, want %d", result.SkillMsgType, SkillBearhugNum)
			}
		})
	}
}

func TestDoBearhug_CGateOrderAndEquipment(t *testing.T) {
	t.Run("immortal target precedes self and weapon", func(t *testing.T) {
		w, ch := newBashTestWorld(t)
		ch.SetSkill(SkillBearhug, 100)
		victim := NewPlayer(2, "Immortal", ch.GetRoom())
		victim.SetLevel(LVL_IMMORT)
		if err := w.AddPlayer(victim); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}
		weapon := makeSpikeWeapon("weapon")
		equipWeapon(t, ch, weapon)

		result := DoBearhug(ch, victim, w)
		if result.MessageToCh != "The gods reject your impunity.\r\n" {
			t.Fatalf("message = %q, want immortal-target gate", result.MessageToCh)
		}
	})

	t.Run("self precedes weapon", func(t *testing.T) {
		w, ch := newBashTestWorld(t)
		ch.SetSkill(SkillBearhug, 100)
		weapon := makeSpikeWeapon("weapon")
		equipWeapon(t, ch, weapon)

		result := DoBearhug(ch, ch, w)
		if result.MessageToCh != "Aren't we funny today...\r\n" {
			t.Fatalf("message = %q, want self-target gate", result.MessageToCh)
		}
	})

	t.Run("wielded weapon", func(t *testing.T) {
		w, ch := newBashTestWorld(t)
		ch.SetSkill(SkillBearhug, 100)
		mob := spawnTargetMob(t, w)
		weapon := makeSpikeWeapon("weapon")
		equipWeapon(t, ch, weapon)

		result := DoBearhug(ch, mob, w)
		if result.MessageToCh != "You need to be bare handed to get a good grip.\r\n" {
			t.Fatalf("message = %q, want wielded-weapon gate", result.MessageToCh)
		}
	})
}

func TestDoBearhug_NoMovementGate(t *testing.T) {
	w, ch := newBashTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.Move = 0
	ch.SetSkill(SkillBearhug, 100)

	dprng.ResetStream(1)
	result := DoBearhug(ch, mob, w)
	if result.MessageToCh != "" || result.SkillMsgType != SkillBearhugNum {
		t.Fatalf("zero-move bearhug was rejected before roll: message=%q set=%d", result.MessageToCh, result.SkillMsgType)
	}
}
