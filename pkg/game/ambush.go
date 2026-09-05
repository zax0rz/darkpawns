package game

import (
	"context"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// ambushAttackType is the lib/misc/messages key used by C's SKILL_AMBUSH.
// It is deliberately kept separate from the Go skill name and from the
// interpreter command index.
const ambushAttackType = 191

// PlanAmbush implements do_ambush's delayed action setup. C stores the event
// in GET_ACTION(ch), leaves the player otherwise un-cooled, and resolves it
// after PULSE_VIOLENCE*2 game pulses.
func (w *World) PlanAmbush(ch *Player, target combat.Combatant) {
	ch.SendMessage("You crouch in the shadows and plan your ambush...\r\n")
	if w.EventQueue == nil {
		return
	}

	wasIn := ch.GetRoom()
	var actionID uint64
	actionID = w.EventQueue.Create(int64(engine.PULSE_VIOLENCE*2), ch.ID, 0, 0, 0, "", 0,
		func(_ context.Context, _, _, _, _ int, _ string, _ int) int64 {
			if ch.GetAmbushAction() != actionID {
				return 0
			}
			ch.ClearAmbushAction()
			w.resolveAmbush(ch, target, wasIn)
			return 0
		})
	ch.SetAmbushAction(actionID)
}

func (w *World) resolveAmbush(ch *Player, target combat.Combatant, wasIn int) {
	// EVENTFUNC(ambush_event): clear GET_ACTION first, then refuse a fighter.
	if ch.GetFighting() != "" {
		return
	}
	nowIn := ch.GetRoom()
	if wasIn != nowIn || target.GetRoom() != nowIn {
		ch.SendMessage("You seem to have lost your prey.\r\n")
		return
	}

	// C consumes this roll only at event time. MOB_AWARE forces the failure
	// branch after the draw, preserving the shared stream.
	percent := dprng.Number(1, 131)
	prob := ch.GetSkill(SkillAmbush)
	if mob, ok := target.(*MobInstance); ok && mob.HasFlag("AWARE") {
		percent = 200
	}

	if percent > prob {
		w.applyAmbushDamage(ch, target, 0)
	} else {
		dam := ch.GetStrToDam() + ch.GetDamroll()
		if wielded, ok := ch.Equipment.GetItemInSlot(SlotWield); ok && wielded != nil && wielded.Prototype != nil {
			dam += combat.RollDice(wielded.Prototype.Values[1], wielded.Prototype.Values[2])
		}
		dam += int(float64(ch.GetLevel()) * 2.6)
		if ch.IsAffected(affHide) {
			dam += int(float64(dam) * 0.10)
		}
		w.applyAmbushDamage(ch, target, dam)
		ImproveSkill(ch, SkillAmbush)
		setWaitState(target, 1)
	}
	ch.SetWaitState(1)
}

func (w *World) applyAmbushDamage(ch *Player, target combat.Combatant, dam int) {
	var onDeath func()
	if target.IsNPC() {
		onDeath = func() {
			// C's die_with_killer awards the mob XP before raw_kill emits the
			// death cry. HandleDeath owns the Go mob extraction/XP seam.
			w.HandleDeath(target, ch, ambushAttackType)
			combat.DeathCry(target)
		}
	}
	if onDeath != nil {
		combat.TakeDamageWithDeath(ch, target, dam, ambushAttackType, onDeath)
	} else {
		combat.TakeDamage(ch, target, dam, ambushAttackType)
	}

	// damage() enrolls both combatants even on a miss. The death callback has
	// already removed a lethal mob, so only surviving targets enter combat.
	if target.GetPosition() != combat.PosDead && w.combatEngine != nil && ch.GetFighting() == target.GetName() {
		if err := w.combatEngine.StartCombat(ch, target); err != nil {
			return
		}
	}
}

func setWaitState(target combat.Combatant, rounds int) {
	if waiter, ok := target.(interface{ SetWaitState(int) }); ok {
		waiter.SetWaitState(rounds)
	}
}
