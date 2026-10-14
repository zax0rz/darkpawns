package game

import (
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// DamageBeforeMessage is shared by the damage() tails that live outside combat.
func (w *World) DamageBeforeMessage(ch, victim combat.Combatant, dam int) bool {
	return combat.DamageBeforeMessage(ch, victim, dam, w.damageBeforeMessageCallbacks())
}

func (w *World) damageBeforeMessageCallbacks() *combat.GameCallbacks {
	return &combat.GameCallbacks{
		DamageNewbieExp: func(ch combat.Combatant, amount int) {
			if p, ok := ch.(*Player); ok {
				w.GainExp(p, amount)
			}
		},
		StopFighting: func(body combat.Combatant) {
			if stopper, ok := w.combatEngine.(interface{ StopCombat(combat.Combatant) }); ok {
				stopper.StopCombat(body)
			} else {
				body.StopFighting()
				body.SetPosition(combat.GetPositionFromHP(body.GetHP(), combat.PosStanding))
			}
		},
		// R1a: C tests ROOM_FLAGGED(victim->in_room, ROOM_NEUTRAL) here
		// (src/fight.c:1493). During Crash_load in_room is NOWHERE, so C reads
		// world[-1].room_flags — an out-of-bounds read whose result decides
		// whether the neutral rescue ("You are saved by the gods!" and a
		// teleport to 8004) replaces the death. The port's lookup finds no room
		// for an unresolvable vnum, so no flags and the death path always runs;
		// recorded in the DP-1419 lethal-replay handoff.
		HasRoomFlag:   func(room int, flag string) bool { return flag == "ROOM_NEUTRAL" && w.RoomHasFlag(room, "neutral") },
		NeutralRescue: w.neutralRescue,
	}
}

// neutralRescue completes src/fight.c:1502-1520 after the shared combat teardown.
// It keeps actor identity and C act()'s visibility/awake recipient gates.
func (w *World) neutralRescue(ch, victim combat.Combatant) {
	p, ok := victim.(*Player)
	if !ok {
		return
	}
	if mob, ok := ch.(*MobInstance); ok {
		if mob.HasMobFlag(MobFlagMemory) {
			mob.Forget(p.GetName())
		}
		if mob.GetHunting() == p.GetName() {
			mob.ClearHunting()
		}
	}
	Act(w, true, ch, p, nil, nil, "$n is startled as $N is saved by the powers of the gods!", "", ToNotVict)
	Act(w, true, ch, p, nil, nil, "$N is saved by the powers of the gods!", "", ToChar)
	p.SendMessage("You are saved by the gods!\r\n")
	mount := w.riddenMount(p)
	w.clearMountedPair(p, mount)
	if mount != nil {
		mount.RemoveAffected(affMounted)
	}
	if err := w.transferBody(p, 8004, false); err != nil {
		slog.Error("neutral rescue transfer failed", "player", p.GetName(), "error", err)
		return
	}
	w.lookAtRoom(p, false)
}
