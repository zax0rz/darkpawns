package game

import (
	"fmt"
	"strings"
)

// CharacterReply ports do_reply for a mobile body (src/act.comm.c:934-973).
// C reads GET_LAST_TELL(ch) from ch->player_specials; every mobile shares the
// zeroed dummy_mob record (src/db.c:104,1281), so a switched NPC starts at 0,
// which is not NOBODY (-1). Bare reply therefore asks for text, and reply
// with text searches character_list for GET_IDNUM(tch) == 0. Every mobile
// carries idnum 0, so the first descriptor-bearing mobile is the target; in
// the single-switched-body witness that is the body itself.
func CharacterReply(ctx CharacterCommandContext, _ string, args []string) CharacterCommandResult {
	body, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	argument := ctx.RawArgs
	if argument == "" && len(args) > 0 {
		argument = strings.Join(args, " ")
	}
	ctx.World.doMobReply(body, argument)
	return CharacterCommandHandled
}

// doMobReply mirrors do_reply's mobile arm. C's skip_spaces strips leading
// spaces but preserves internal spacing in the tell text.
func (w *World) doMobReply(ch *MobInstance, argument string) {
	argument = strings.TrimLeft(argument, " \t\r\n\v\f")
	if w.npcDummyLastTell.Load() == 0 {
		if argument == "" {
			ch.SendMessage("What is your reply?\r\n")
			return
		}
		if target := w.firstDescriptorMob(); target != nil {
			w.mobPerformTell(ch, target, argument)
			return
		}
	}
	ch.SendMessage("They are no longer playing.\r\n")
}

// firstDescriptorMob returns the first switched mobile in descriptor order.
// C scans character_list; the session manager's descriptor snapshot preserves
// that order for the descriptor-bearing actors this arm can match.
func (w *World) firstDescriptorMob() *MobInstance {
	if w.PlayingDescriptorBodies == nil {
		return nil
	}
	for _, body := range w.PlayingDescriptorBodies() {
		if mob, ok := body.(*MobInstance); ok {
			return mob
		}
	}
	return nil
}

// mobPerformTell is perform_tell for a mobile source and victim
// (src/act.comm.c:873-895): victim line, optional AFK notice (mobile victims
// have no AFK preference), actor echo, then GET_LAST_TELL(vict) =
// GET_IDNUM(ch), which is 0 for every mobile.
func (w *World) mobPerformTell(ch, vict *MobInstance, message string) {
	message = deleteANSIControls(message)
	Act(w, false, ch, vict, nil, nil, fmt.Sprintf("$n tells you, '%s'", message), "", ToVict|ToSleep)
	Act(w, false, ch, vict, nil, nil, fmt.Sprintf("You tell $N, '%s'", message), "", ToChar|ToSleep)
	w.npcDummyLastTell.Store(0)
}
