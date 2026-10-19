package game

import (
	"fmt"
	"strconv"
)

// LogHuntingStart emits set_hunting's producer (src/utils.c:715-724):
//
//	if (IS_MOB(vict))
//	  (ch)->char_specials.hunting = vict;
//	else if (GET_IDNUM(vict)>0)
//	{
//	  sprintf(buf, "%s started hunting %s", GET_NAME(ch), GET_NAME(vict));
//	  mudlog(buf, CMP, LVL_IMMORT, FALSE);
//	  (ch)->char_specials.hunting_id = GET_IDNUM(vict);
//	}
//
// C logs only in the second arm, so a mobile prey is silent (IS_MOB reads
// ch->mob, and a *Player is never one) and so is a prey with no id number — a
// guest body, which C has no equivalent of. The line is emitted where C emits
// it: after the old hunting state is cleared and before the hunting id is
// stored.
//
// Callers resolve the prey to a *Player because only they can tell a player
// from a mobile. C logs player prey regardless of whether the hunter is a
// player or a mobile; only mobile hunters retain a Go-side target because
// HUNTING(ch) returns NULL for players (src/utils.c:715-733).
func LogHuntingStart(hunter Actor, prey *Player) {
	if hunter == nil || prey == nil || prey.GetID() <= 0 {
		return
	}
	MudLog(fmt.Sprintf("%s started hunting %s", hunter.GetName(), prey.GetName()), MudlogComplete, LVL_IMMORT, false)
}

// SetHuntingTarget is set_hunting's typed counterpart. Player targets are
// retained by ID and resolved only through a playing descriptor; mobile
// targets are retained by body ID. C logs player prey before storing its ID.
func (w *World) SetHuntingTarget(hunter Actor, prey Actor) {
	if hunter == nil {
		return
	}
	mobHunter, isMobHunter := hunter.(*MobInstance)
	if isMobHunter {
		if w.mobAlreadyHunts(mobHunter, prey) {
			return
		}
		mobHunter.ClearHunting()
	}
	if prey == nil {
		return
	}

	switch target := prey.(type) {
	case *Player:
		targetName, targetID := target.GetName(), target.GetID()
		LogHuntingStart(hunter, target)
		if isMobHunter && targetID > 0 {
			mobHunter.mu.Lock()
			mobHunter.Hunting = targetName
			mobHunter.HuntingID = strconv.Itoa(targetID)
			mobHunter.HuntingMobID = 0
			mobHunter.mu.Unlock()
		}
	case *MobInstance:
		if isMobHunter {
			targetName, targetID := target.GetName(), target.GetID()
			mobHunter.mu.Lock()
			mobHunter.Hunting = targetName
			mobHunter.HuntingID = ""
			mobHunter.HuntingMobID = targetID
			mobHunter.mu.Unlock()
		}
	}
}

func (w *World) mobAlreadyHunts(hunter *MobInstance, prey Actor) bool {
	if hunter == nil || prey == nil {
		return false
	}
	hunter.mu.RLock()
	name, playerID, mobID := hunter.Hunting, hunter.HuntingID, hunter.HuntingMobID
	hunter.mu.RUnlock()
	switch target := prey.(type) {
	case *Player:
		return target.GetID() > 0 && playerID == strconv.Itoa(target.GetID()) && w.playerHasPlayingDescriptor(target)
	case *MobInstance:
		return mobID != 0 && mobID == target.GetID()
	default:
		return name != "" && name == prey.GetName()
	}
}

func parseHuntingPlayerID(id string) (int, error) {
	return strconv.Atoi(id)
}
