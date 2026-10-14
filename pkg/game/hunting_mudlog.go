package game

import "fmt"

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
