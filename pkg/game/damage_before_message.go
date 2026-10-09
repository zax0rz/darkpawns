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
