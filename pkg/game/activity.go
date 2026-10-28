package game

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/telemetry"
)

// ActivityForPlayer is an operator-only snapshot, never a gameplay decision.
// Match the name visibility used by entry/death mudlog, without copying hosts.
func ActivityForPlayer(p *Player) telemetry.Activity {
	return telemetry.Activity{Player: p.GetName(), Class: ClassNames[p.GetClass()], Level: p.GetLevel(), MinLevel: max(LVL_IMMORT, p.GetInvisLevel())}
}

func (w *World) recordPlayerDeath(p *Player, killer combat.Combatant, cause string) {
	e := ActivityForPlayer(p)
	if strings.HasPrefix(e.Player, "Guest_") {
		e.Player = "Guest"
	}
	e.RoomVNum = p.GetRoom()
	if room, ok := w.GetRoom(e.RoomVNum); ok {
		e.RoomName = room.Name
	}
	e.Cause = cause
	if killer != nil {
		switch {
		case killer.IsNPC():
			e.Cause = "mob"
			e.MobName = killer.GetName()
			if mob, ok := killer.(*MobInstance); ok && mob.Proto() != nil {
				e.MobVNum = mob.Proto().VNum
			}
		case killer != p:
			e.Cause = "player"
		default:
			e.Cause = "other"
		}
	}
	telemetry.RecordPlayerDeath(e)
}
