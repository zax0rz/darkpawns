package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// Each numbered seam and the plain damage seam must preserve damage()'s
// victim stop, while still registering the fresh pair (src/fight.c:1443,
// 1630-1632). Drive the real command tail and real engine, not a call recorder.
func TestSkillTailPreservesWoundedVictimStop(t *testing.T) {
	for _, skill := range []string{game.SkillDragonKick, game.SkillSmackheads, game.SkillDisembowel, game.SkillGroinrip, game.SkillNeckbreak, game.SkillTigerPunch, game.SkillStrike, game.SkillCutthroat, game.SkillKick} {
		for _, wounded := range []bool{false, true} {
			name := skill + "/fresh"
			hp := 500
			if wounded {
				name = skill + "/wounded"
				hp = 10
			}
			t.Run(name, func(t *testing.T) {
				ktw := newKillTestWorld(t, hp, 0, 0, 10, "target")
				ktw.world.StopAITicker()
				actor := ktw.addPlayer(t, 1, "Attacker", 10, game.ClassWarrior, false)
				rig := newDrawOrderRig(t, actor.Name)
				defer rig.teardown()
				session := &killPayoutSession{player: actor, world: ktw.world, combatEngine: rig.engine}
				result := game.SkillResult{StartCombat: true, Damage: 16, DamageSkill: skill, SkillMsgInDamage: skill != game.SkillKick && skill != game.SkillCutthroat}
				dprng.ResetStream(1)
				if err := sendSkillResult(session, actor, ktw.mob, result); err != nil {
					t.Fatal(err)
				}
				if actor.GetFighting() != ktw.mob.GetName() {
					t.Fatalf("attacker lost C enrollment: %q", actor.GetFighting())
				}
				if wounded {
					if ktw.mob.GetPosition() != combat.PosMortally || ktw.mob.GetFighting() != "" {
						t.Fatalf("wounded victim re-enrolled: position %d fighting %q", ktw.mob.GetPosition(), ktw.mob.GetFighting())
					}
					actor.SetWaitState(10)
					before := actor.GetHP()
					rig.engine.PerformRound()
					if actor.GetHP() != before || ktw.mob.GetPosition() != combat.PosMortally || ktw.mob.GetFighting() != "" {
						t.Fatal("unconscious victim received an extra combat round")
					}
				} else {
					if ktw.mob.GetFighting() != actor.Name {
						t.Fatalf("fresh victim not enrolled: %q", ktw.mob.GetFighting())
					}
					ktw.mob.SetPosition(combat.PosSitting)
					rig.engine.PerformRound()
					if ktw.mob.GetPosition() != combat.PosFighting {
						t.Fatal("fresh victim never got its engine round")
					}
					if ktw.mob.GetFighting() != actor.Name {
						t.Fatal("fresh victim lost its combat round")
					}
				}
			})
		}
	}
}

func TestSmackheadsSecondaryTailPreservesWoundedVictimStop(t *testing.T) {
	for _, wounded := range []bool{false, true} {
		name := "fresh"
		if wounded {
			name = "wounded"
		}
		t.Run(name, func(t *testing.T) {
			ktw := newKillTestWorld(t, 500, 0, 0, 10, "target")
			ktw.world.StopAITicker()
			secondary, err := ktw.world.SpawnMob(testMobVNum, testRoomVNum)
			if err != nil {
				t.Fatal(err)
			}
			proto := *secondary.Proto()
			proto.ShortDesc = "a second victim"
			secondary.SetProto(&proto)
			secondary.SetPosition(combat.PosStanding)
			if wounded {
				secondary.CurrentHP = 10
			}
			actor := ktw.addPlayer(t, 1, "Attacker", 10, game.ClassWarrior, false)
			rig := newDrawOrderRig(t, actor.Name)
			defer rig.teardown()
			session := &killPayoutSession{player: actor, world: ktw.world, combatEngine: rig.engine}
			result := game.SkillResult{StartCombat: true, Damage: 16, DamageSkill: game.SkillSmackheads, SkillMsgInDamage: true, Targets: []combat.Combatant{ktw.mob, secondary}}
			if err := sendSkillResult(session, actor, ktw.mob, result); err != nil {
				t.Fatal(err)
			}
			if wounded {
				if secondary.GetFighting() != "" || secondary.GetPosition() != combat.PosMortally {
					t.Fatalf("secondary victim re-enrolled: %q position %d", secondary.GetFighting(), secondary.GetPosition())
				}
				rig.engine.PerformRound()
				if secondary.GetPosition() != combat.PosMortally {
					t.Fatal("unconscious secondary received a stand/round")
				}
			} else {
				if secondary.GetFighting() != actor.Name {
					t.Fatalf("fresh secondary not enrolled: %q", secondary.GetFighting())
				}
				secondary.SetPosition(combat.PosSitting)
				rig.engine.PerformRound()
				if secondary.GetPosition() != combat.PosFighting {
					t.Fatal("fresh secondary never got its engine round")
				}
			}
		})
	}
}
