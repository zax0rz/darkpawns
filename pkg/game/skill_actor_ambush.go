package game

import (
	"context"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// characterAmbush ports do_ambush's immediate gates and delayed event setup
// for a switched mobile body (src/act.offensive.c:1386-1506).
func characterAmbush(ctx CharacterCommandContext, args []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	victim := mobileCommandTarget(ctx, mob, args, false)
	if victim == nil {
		mobileSend(ctx, mob, "Ambush who?\r\n")
		return CharacterCommandHandled
	}
	if ctx.World.DummyMobSkill(skillNumAmbush) == 0 {
		mobileSend(ctx, mob, "You'd better not.\r\n")
		return CharacterCommandHandled
	}
	if mob.getAmbushAction() != 0 {
		mobileSend(ctx, mob, "You are a little busy for that right now!\r\n")
		return CharacterCommandHandled
	}
	if mob == victim {
		mobileSend(ctx, mob, "Ambush yourself? You idiot!\r\n")
		return CharacterCommandHandled
	}
	room := ctx.World.GetRoomInWorld(mob.GetRoom())
	if room == nil || (room.Sector != SECT_FOREST && room.Sector != SECT_HILLS && room.Sector != SECT_MOUNTAIN && room.Sector != SECT_CITY) {
		mobileSend(ctx, mob, "Ambush someone here? Impossible!\r\n")
		return CharacterCommandHandled
	}
	if victim.GetFightingBody() != nil {
		mobileSend(ctx, mob, "They're too alert for that, currently.\r\n")
		return CharacterCommandHandled
	}
	mobileSend(ctx, mob, "You crouch in the shadows and plan your ambush...\r\n")
	ctx.World.planCharacterAmbush(mob, victim)
	return CharacterCommandHandled
}

func (w *World) planCharacterAmbush(mob *MobInstance, victim combat.Combatant) {
	if w.EventQueue == nil {
		return
	}
	wasIn := mob.GetRoom()
	var actionID uint64
	actionID = w.EventQueue.Create(int64(engine.PULSE_VIOLENCE*2), mob.ID, 0, 0, 0, "", 0,
		func(_ context.Context, _, _, _, _ int, _ string, _ int) int64 {
			if mob.getAmbushAction() != actionID {
				return 0
			}
			mob.clearAmbushAction()
			w.resolveCharacterAmbush(mob, victim, wasIn)
			return 0
		})
	mob.setAmbushAction(actionID)
}

func (w *World) resolveCharacterAmbush(mob *MobInstance, victim combat.Combatant, wasIn int) {
	if mob.GetFightingBody() != nil {
		return
	}
	nowIn := mob.GetRoom()
	if wasIn != nowIn || victim.GetRoom() != nowIn {
		mob.SendMessage("You seem to have lost your prey.\r\n")
		return
	}
	percent := dprng.Number(1, 131)
	prob := w.DummyMobSkill(skillNumAmbush)
	if targetMob, ok := victim.(*MobInstance); ok && targetMob.HasMobFlag(MobFlagAware) {
		percent = 200
	}
	if percent > prob {
		w.mobileCommandSkillDamage(mob, victim, 0, SkillAmbushNum)
	} else {
		damage := combat.StrAppToDam(mob) + mob.GetDamroll()
		if weapon := mobileWielded(mob); weapon != nil {
			damage += dprng.Dice(weapon.GetValue(1), weapon.GetValue(2))
		}
		damage += int(float64(mob.GetLevel()) * 2.6)
		if mob.IsAffected(affHide) {
			damage += int(float64(damage) * 0.10)
		}
		w.mobileCommandSkillDamage(mob, victim, damage, SkillAmbushNum)
		setCombatWait(victim, engine.PULSE_VIOLENCE)
	}
	mob.SetWaitStatePulses(engine.PULSE_VIOLENCE)
}
